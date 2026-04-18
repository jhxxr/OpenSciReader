package translator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Options struct {
	DataRootDir       string
	RuntimeDir        string
	WorkerScriptPath  string
	PythonSearchPaths []string
	EventSink         func(JobEvent)
}

type Manager struct {
	dataRootDir       string
	runtimeDir        string
	workerScriptPath  string
	pythonSearchPaths []string
	eventSink         func(JobEvent)
	httpClient        *http.Client

	mu   sync.RWMutex
	jobs map[string]*jobRuntime
}

type jobRuntime struct {
	mu          sync.RWMutex
	rootDir     string
	stateFile   string
	cancel      context.CancelFunc
	snapshot    JobSnapshot
	subscribers map[int]chan JobEvent
	nextSubID   int
	sequence    int64
	eventSink   func(JobEvent)
}

func NewManager(options Options) (*Manager, error) {
	rootDir := strings.TrimSpace(options.DataRootDir)
	if rootDir == "" {
		return nil, fmt.Errorf("translator data root directory is required")
	}
	if err := os.MkdirAll(rootDir, 0o700); err != nil {
		return nil, fmt.Errorf("create translator data root: %w", err)
	}

	return &Manager{
		dataRootDir:       rootDir,
		runtimeDir:        strings.TrimSpace(options.RuntimeDir),
		workerScriptPath:  strings.TrimSpace(options.WorkerScriptPath),
		pythonSearchPaths: append([]string{}, options.PythonSearchPaths...),
		eventSink:         options.EventSink,
		httpClient:        &http.Client{},
		jobs:              make(map[string]*jobRuntime),
	}, nil
}

func (m *Manager) Start(ctx context.Context, request StartRequest) (JobSnapshot, error) {
	normalized, chunks, localPDFPath, err := m.normalizeRequest(ctx, request)
	if err != nil {
		return JobSnapshot{}, err
	}
	jobID := uuid.NewString()
	jobDir := filepath.Join(m.dataRootDir, jobID)
	if err := os.MkdirAll(jobDir, 0o700); err != nil {
		return JobSnapshot{}, fmt.Errorf("create job directory: %w", err)
	}

	now := nowRFC3339()
	snapshot := JobSnapshot{
		JobID:              jobID,
		RetryOfJobID:       normalized.RetryOfJobID,
		Mode:               normalized.Mode,
		Status:             JobStatusQueued,
		ItemID:             normalized.ItemID,
		ItemTitle:          normalized.ItemTitle,
		PDFPath:            normalized.PDFPath,
		LocalPDFPath:       localPDFPath,
		PageCount:          normalized.PageCount,
		SourceLang:         normalized.SourceLang,
		TargetLang:         normalized.TargetLang,
		PreviewChunkPages:  normalized.PreviewChunkPages,
		MaxPagesPerPart:    normalized.MaxPagesPerPart,
		QPS:                normalized.QPS,
		PoolMaxWorkers:     normalized.PoolMaxWorkers,
		TermPoolMaxWorkers: normalized.TermPoolMaxWorkers,
		ProviderID:         normalized.Provider.ProviderID,
		ProviderName:       normalized.Provider.ProviderName,
		ModelID:            normalized.Provider.ModelID,
		CreatedAt:          now,
		UpdatedAt:          now,
		Chunks:             chunks,
	}
	runtimeCtx, cancel := context.WithCancel(context.Background())
	job := &jobRuntime{
		rootDir:     jobDir,
		stateFile:   filepath.Join(jobDir, "job.json"),
		cancel:      cancel,
		snapshot:    snapshot,
		subscribers: make(map[int]chan JobEvent),
		eventSink:   m.eventSink,
	}
	if err := job.persist(); err != nil {
		cancel()
		return JobSnapshot{}, err
	}

	m.mu.Lock()
	m.jobs[jobID] = job
	m.mu.Unlock()

	go m.runJob(runtimeCtx, job, normalized)

	return snapshot, nil
}

func (m *Manager) Cancel(jobID string) (JobSnapshot, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return JobSnapshot{}, fmt.Errorf("job id is required")
	}

	if job, err := m.lookupJob(jobID); err == nil {
		job.markCancelled()
		snapshot := job.cloneSnapshot()
		job.emit(JobEvent{
			Type:      "cancelled",
			Message:   "translation job cancelled by user",
			JobStatus: JobStatusCancelled,
			Status:    &snapshot,
		})
		job.cancel()
		return snapshot, nil
	}

	stateFile := filepath.Join(m.dataRootDir, jobID, "job.json")
	var snapshot JobSnapshot
	if err := readJSONFile(stateFile, &snapshot); err != nil {
		return JobSnapshot{}, fmt.Errorf("load job %s: %w", jobID, err)
	}
	if !isTerminalJobStatus(snapshot.Status) {
		now := nowRFC3339()
		snapshot.Status = JobStatusCancelled
		snapshot.UpdatedAt = now
		snapshot.FinishedAt = now
		if err := writeJSONFile(stateFile, snapshot); err != nil {
			return JobSnapshot{}, fmt.Errorf("persist cancelled job %s: %w", jobID, err)
		}
	}
	return snapshot, nil
}

func (m *Manager) GetStatus(jobID string) (JobSnapshot, error) {
	if job, err := m.lookupJob(jobID); err == nil {
		return job.cloneSnapshot(), nil
	}

	stateFile := filepath.Join(m.dataRootDir, jobID, "job.json")
	var snapshot JobSnapshot
	if err := readJSONFile(stateFile, &snapshot); err != nil {
		return JobSnapshot{}, fmt.Errorf("load job %s: %w", jobID, err)
	}
	return snapshot, nil
}

func (m *Manager) ListJobs() ([]JobSnapshot, error) {
	entries, err := os.ReadDir(m.dataRootDir)
	if err != nil {
		return nil, fmt.Errorf("list translator jobs: %w", err)
	}

	m.mu.RLock()
	liveJobs := make(map[string]*jobRuntime, len(m.jobs))
	for jobID, job := range m.jobs {
		liveJobs[jobID] = job
	}
	m.mu.RUnlock()

	snapshots := make([]JobSnapshot, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		jobID := strings.TrimSpace(entry.Name())
		if jobID == "" || strings.HasPrefix(jobID, "_") {
			continue
		}
		if liveJob, ok := liveJobs[jobID]; ok {
			snapshots = append(snapshots, liveJob.cloneSnapshot())
			continue
		}

		stateFile := filepath.Join(m.dataRootDir, jobID, "job.json")
		var snapshot JobSnapshot
		if err := readJSONFile(stateFile, &snapshot); err != nil {
			continue
		}
		snapshots = append(snapshots, snapshot)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshotSortKey(snapshots[i]).After(snapshotSortKey(snapshots[j]))
	})

	return snapshots, nil
}

func (m *Manager) DeleteJob(jobID string) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return fmt.Errorf("job id is required")
	}

	jobDir := filepath.Join(m.dataRootDir, jobID)
	if liveJob, err := m.lookupJob(jobID); err == nil {
		snapshot := liveJob.cloneSnapshot()
		if !isTerminalJobStatus(snapshot.Status) {
			return fmt.Errorf("cancel running job before deleting")
		}
		liveJob.closeSubscribers()
		m.mu.Lock()
		delete(m.jobs, jobID)
		m.mu.Unlock()
	}

	if _, err := os.Stat(jobDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("job %s not found", jobID)
		}
		return fmt.Errorf("stat job directory %s: %w", jobID, err)
	}

	if err := os.RemoveAll(jobDir); err != nil {
		return fmt.Errorf("delete job %s: %w", jobID, err)
	}
	return nil
}

func (m *Manager) Subscribe(jobID string) (JobSnapshot, <-chan JobEvent, func(), error) {
	job, err := m.lookupJob(jobID)
	if err != nil {
		return JobSnapshot{}, nil, nil, err
	}
	ch := make(chan JobEvent, 64)

	job.mu.Lock()
	subID := job.nextSubID
	job.nextSubID++
	job.subscribers[subID] = ch
	snapshot := cloneSnapshot(job.snapshot)
	job.mu.Unlock()

	cancel := func() {
		job.mu.Lock()
		if existing, ok := job.subscribers[subID]; ok {
			delete(job.subscribers, subID)
			close(existing)
		}
		job.mu.Unlock()
	}

	return snapshot, ch, cancel, nil
}

func (m *Manager) lookupJob(jobID string) (*jobRuntime, error) {
	m.mu.RLock()
	job := m.jobs[jobID]
	m.mu.RUnlock()
	if job == nil {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	return job, nil
}

func (m *Manager) normalizeRequest(ctx context.Context, request StartRequest) (StartRequest, []ChunkStatus, string, error) {
	normalized := request
	normalized.PDFPath = strings.TrimSpace(normalized.PDFPath)
	normalized.SourceLang = normalizeLanguage(normalized.SourceLang, "en")
	normalized.TargetLang = normalizeLanguage(normalized.TargetLang, "zh-CN")
	normalized.ItemID = strings.TrimSpace(normalized.ItemID)
	normalized.ItemTitle = strings.TrimSpace(normalized.ItemTitle)
	normalized.RetryOfJobID = strings.TrimSpace(normalized.RetryOfJobID)
	normalized.ReusePreviewJobID = strings.TrimSpace(normalized.ReusePreviewJobID)
	normalized.Provider.ProviderName = strings.TrimSpace(normalized.Provider.ProviderName)
	normalized.Provider.BaseURL = strings.TrimSpace(normalized.Provider.BaseURL)
	normalized.Provider.ModelID = strings.TrimSpace(normalized.Provider.ModelID)
	normalized.Provider.APIKey = strings.TrimSpace(normalized.Provider.APIKey)

	if normalized.Mode != ModePreview && normalized.Mode != ModeExport {
		return StartRequest{}, nil, "", fmt.Errorf("invalid mode: %s", normalized.Mode)
	}
	if normalized.PDFPath == "" {
		return StartRequest{}, nil, "", fmt.Errorf("pdf path is required")
	}
	if normalized.PageCount <= 0 {
		return StartRequest{}, nil, "", fmt.Errorf("page count must be greater than 0")
	}
	if normalized.Provider.BaseURL == "" || normalized.Provider.ModelID == "" {
		return StartRequest{}, nil, "", fmt.Errorf("layout translation model is incomplete")
	}
	if normalized.Provider.APIKey == "" {
		return StartRequest{}, nil, "", fmt.Errorf("provider %s has no api key", normalized.Provider.ProviderName)
	}
	if normalized.Mode == ModePreview {
		if normalized.PreviewChunkPages <= 0 {
			normalized.PreviewChunkPages = 25
		}
	} else {
		if normalized.MaxPagesPerPart < 0 {
			normalized.MaxPagesPerPart = 0
		}
	}
	if normalized.QPS <= 0 {
		normalized.QPS = 4
	}
	if normalized.PoolMaxWorkers <= 0 {
		normalized.PoolMaxWorkers = normalized.QPS
	}
	if normalized.TermPoolMaxWorkers <= 0 {
		normalized.TermPoolMaxWorkers = normalized.PoolMaxWorkers
	}

	localPDFPath, err := m.prepareInputDocument(ctx, normalized.PDFPath)
	if err != nil {
		return StartRequest{}, nil, "", err
	}

	chunks := buildChunks(normalized.Mode, normalized.PageCount, normalized.PreviewChunkPages)
	return normalized, chunks, localPDFPath, nil
}

func (m *Manager) prepareInputDocument(ctx context.Context, rawPath string) (string, error) {
	lower := strings.ToLower(rawPath)
	switch {
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
		jobInputDir := filepath.Join(m.dataRootDir, "_remote_cache")
		if err := os.MkdirAll(jobInputDir, 0o700); err != nil {
			return "", fmt.Errorf("create remote cache directory: %w", err)
		}
		targetPath := filepath.Join(jobInputDir, uuid.NewString()+".pdf")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawPath, nil)
		if err != nil {
			return "", fmt.Errorf("build remote pdf request: %w", err)
		}
		req.Header.Set("User-Agent", "OpenSciReader/1.0")
		req.Header.Set("Accept", "application/pdf,*/*")
		resp, err := m.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("fetch remote pdf: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return "", fmt.Errorf("remote pdf http error: %s", resp.Status)
		}
		file, err := os.Create(targetPath)
		if err != nil {
			return "", fmt.Errorf("create remote pdf cache file: %w", err)
		}
		defer file.Close()
		if _, err := io.Copy(file, resp.Body); err != nil {
			return "", fmt.Errorf("write remote pdf cache file: %w", err)
		}
		return targetPath, nil
	default:
		if !filepath.IsAbs(rawPath) {
			return "", fmt.Errorf("pdf path must be absolute: %s", rawPath)
		}
		cleanPath := filepath.Clean(rawPath)
		info, err := os.Stat(cleanPath)
		if err != nil {
			return "", fmt.Errorf("stat pdf path: %w", err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("pdf path points to a directory: %s", cleanPath)
		}
		if !strings.EqualFold(filepath.Ext(cleanPath), ".pdf") {
			return "", fmt.Errorf("only pdf files are supported: %s", cleanPath)
		}
		return cleanPath, nil
	}
}

func (m *Manager) runJob(ctx context.Context, job *jobRuntime, request StartRequest) {
	job.markRunning()
	job.emit(JobEvent{
		Type:      "job_started",
		Message:   "translation job started",
		JobStatus: JobStatusRunning,
	})
	stopHeartbeat := m.startHeartbeat(ctx, job)
	defer stopHeartbeat()

	switch request.Mode {
	case ModePreview:
		if err := m.runPreviewJob(ctx, job, request); err != nil {
			m.finishWithError(job, err)
			return
		}
	case ModeExport:
		if err := m.runExportJob(ctx, job, request); err != nil {
			m.finishWithError(job, err)
			return
		}
	default:
		m.finishWithError(job, fmt.Errorf("unsupported mode: %s", request.Mode))
		return
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		job.markCancelled()
		job.emit(JobEvent{
			Type:      "cancelled",
			Message:   "translation job cancelled",
			JobStatus: JobStatusCancelled,
		})
		return
	}

	job.markCompleted()
	job.emit(JobEvent{
		Type:      "finish",
		Message:   "translation job finished",
		JobStatus: JobStatusCompleted,
		Output:    job.currentOutput(),
	})
}

func (m *Manager) startHeartbeat(ctx context.Context, job *jobRuntime) func() {
	stopCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				snapshot := job.touchHeartbeat()
				if snapshot.Status != JobStatusRunning {
					continue
				}
				job.emit(JobEvent{
					Type:      "heartbeat",
					Message:   "translation job is still running",
					JobStatus: snapshot.Status,
					Status:    &snapshot,
				})
			}
		}
	}()
	return func() {
		close(stopCh)
	}
}

func (m *Manager) runPreviewJob(ctx context.Context, job *jobRuntime, request StartRequest) error {
	chunks := job.chunkList()
	workerHomeDir := m.sharedWorkerHomeDir(request)
	for _, chunk := range chunks {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		snapshot := job.cloneSnapshot()
		pages := fmt.Sprintf("%d-%d", chunk.StartPage, chunk.EndPage)
		workDir := filepath.Join(job.rootDir, fmt.Sprintf("preview-chunk-%03d", chunk.Index))
		if err := os.MkdirAll(workDir, 0o700); err != nil {
			return fmt.Errorf("create preview work directory: %w", err)
		}
		job.markChunkRunning(chunk.Index)
		job.emit(JobEvent{
			Type:      "chunk_started",
			Message:   fmt.Sprintf("preview chunk %d started", chunk.Index),
			JobStatus: JobStatusRunning,
			Chunk:     job.chunkByIndex(chunk.Index),
		})

		var chunkOutput *JobOutputs
		err := m.runWorker(ctx, workerRequest{
			JobID:                     snapshot.JobID,
			Mode:                      ModePreview,
			InputPDFPath:              snapshot.LocalPDFPath,
			WorkingDir:                workDir,
			OutputDir:                 workDir,
			WorkerHomeDir:             workerHomeDir,
			SourceLang:                request.SourceLang,
			TargetLang:                request.TargetLang,
			Pages:                     pages,
			OnlyIncludeTranslatedPage: true,
			NoDual:                    false,
			NoMono:                    false,
			MaxPagesPerPart:           0,
			ReportInterval:            0.25,
			QPS:                       request.QPS,
			PoolMaxWorkers:            request.PoolMaxWorkers,
			TermPoolMaxWorkers:        request.TermPoolMaxWorkers,
			MinTextLength:             5,
			WatermarkOutputMode:       "no_watermark",
			Provider:                  request.Provider,
		}, func(event workerEvent) error {
			chunkOutput = m.applyWorkerEvent(job, chunk.Index, event)
			return nil
		})
		if err != nil {
			job.markChunkFailed(chunk.Index, err.Error())
			return err
		}
		job.markChunkCompleted(chunk.Index, chunkOutput)
		job.emit(JobEvent{
			Type:      "chunk_finished",
			Message:   fmt.Sprintf("preview chunk %d finished", chunk.Index),
			JobStatus: JobStatusRunning,
			Chunk:     job.chunkByIndex(chunk.Index),
			Output:    chunkOutput,
		})
	}
	return nil
}

func (m *Manager) runExportJob(ctx context.Context, job *jobRuntime, request StartRequest) error {
	chunkIndex := 1
	snapshot := job.cloneSnapshot()
	workDir := filepath.Join(job.rootDir, "export")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return fmt.Errorf("create export work directory: %w", err)
	}
	workerHomeDir := m.sharedWorkerHomeDir(request)

	if previewSnapshot, ok, err := m.loadReusablePreviewSnapshot(request); err != nil {
		return err
	} else if ok {
		job.markChunkRunning(chunkIndex)
		job.emit(JobEvent{
			Type:      "chunk_started",
			Message:   "export merge started",
			JobStatus: JobStatusRunning,
			Chunk:     job.chunkByIndex(chunkIndex),
		})

		var output *JobOutputs
		err = m.runWorker(ctx, workerRequest{
			JobID:             snapshot.JobID,
			Mode:              ModeExport,
			InputPDFPath:      snapshot.LocalPDFPath,
			WorkingDir:        workDir,
			OutputDir:         workDir,
			WorkerHomeDir:     workerHomeDir,
			MergeMonoPDFPaths: previewChunkPaths(previewSnapshot.Chunks, func(chunk ChunkStatus) string { return chunk.TranslatedPDFPath }),
			MergeDualPDFPaths: previewChunkPaths(previewSnapshot.Chunks, func(chunk ChunkStatus) string { return chunk.DualPDFPath }),
		}, func(event workerEvent) error {
			output = m.applyWorkerEvent(job, chunkIndex, event)
			return nil
		})
		if err != nil {
			job.markChunkFailed(chunkIndex, err.Error())
			return err
		}
		job.markChunkCompleted(chunkIndex, output)
		job.emit(JobEvent{
			Type:      "chunk_finished",
			Message:   "export merge finished",
			JobStatus: JobStatusRunning,
			Chunk:     job.chunkByIndex(chunkIndex),
			Output:    output,
		})
		return nil
	}

	job.markChunkRunning(chunkIndex)
	job.emit(JobEvent{
		Type:      "chunk_started",
		Message:   "export translation started",
		JobStatus: JobStatusRunning,
		Chunk:     job.chunkByIndex(chunkIndex),
	})

	var output *JobOutputs
	err := m.runWorker(ctx, workerRequest{
		JobID:               snapshot.JobID,
		Mode:                ModeExport,
		InputPDFPath:        snapshot.LocalPDFPath,
		WorkingDir:          workDir,
		OutputDir:           workDir,
		WorkerHomeDir:       workerHomeDir,
		SourceLang:          request.SourceLang,
		TargetLang:          request.TargetLang,
		NoDual:              false,
		NoMono:              false,
		MaxPagesPerPart:     request.MaxPagesPerPart,
		ReportInterval:      0.25,
		QPS:                 request.QPS,
		PoolMaxWorkers:      request.PoolMaxWorkers,
		TermPoolMaxWorkers:  request.TermPoolMaxWorkers,
		MinTextLength:       5,
		WatermarkOutputMode: "no_watermark",
		Provider:            request.Provider,
	}, func(event workerEvent) error {
		output = m.applyWorkerEvent(job, chunkIndex, event)
		return nil
	})
	if err != nil {
		job.markChunkFailed(chunkIndex, err.Error())
		return err
	}
	job.markChunkCompleted(chunkIndex, output)
	job.emit(JobEvent{
		Type:      "chunk_finished",
		Message:   "export translation finished",
		JobStatus: JobStatusRunning,
		Chunk:     job.chunkByIndex(chunkIndex),
		Output:    output,
	})
	return nil
}

func (m *Manager) applyWorkerEvent(job *jobRuntime, chunkIndex int, event workerEvent) *JobOutputs {
	switch event.Type {
	case "stage_summary":
		job.updateProgress(event)
		job.emit(JobEvent{
			Type:       "stage_summary",
			JobStatus:  job.cloneSnapshot().Status,
			Stages:     event.Stages,
			PartIndex:  event.PartIndex,
			TotalParts: event.TotalParts,
		})
	case "progress_start", "progress_update", "progress_end":
		job.updateProgress(event)
		job.emit(JobEvent{
			Type:            event.Type,
			JobStatus:       job.cloneSnapshot().Status,
			Stage:           event.Stage,
			StageProgress:   event.StageProgress,
			OverallProgress: event.OverallProgress,
			StageCurrent:    event.StageCurrent,
			StageTotal:      event.StageTotal,
			PartIndex:       event.PartIndex,
			TotalParts:      event.TotalParts,
			Chunk:           job.chunkByIndex(chunkIndex),
		})
	case "finish":
		output := toJobOutputs(event.TranslateResult)
		job.setOutputs(output)
		return &output
	case "error":
		job.emit(JobEvent{
			Type:      "error",
			JobStatus: JobStatusFailed,
			Error:     compactWorkerErrorMessage(event.Error),
			ErrorType: event.ErrorType,
			Details:   event.Details,
			Chunk:     job.chunkByIndex(chunkIndex),
		})
	}
	return nil
}

func (m *Manager) finishWithError(job *jobRuntime, err error) {
	if errors.Is(err, context.Canceled) {
		job.markCancelled()
		job.emit(JobEvent{
			Type:      "cancelled",
			Message:   "translation job cancelled",
			JobStatus: JobStatusCancelled,
		})
		return
	}
	job.markFailed(err.Error())
	job.emit(JobEvent{
		Type:      "error",
		JobStatus: JobStatusFailed,
		Error:     err.Error(),
	})
}

func buildChunks(mode Mode, pageCount, previewChunkPages int) []ChunkStatus {
	if pageCount <= 0 {
		return []ChunkStatus{}
	}
	if mode == ModeExport {
		return []ChunkStatus{{
			Index:     1,
			StartPage: 1,
			EndPage:   pageCount,
			Status:    JobStatusQueued,
		}}
	}
	if previewChunkPages <= 0 {
		previewChunkPages = 25
	}
	chunks := make([]ChunkStatus, 0, (pageCount+previewChunkPages-1)/previewChunkPages)
	index := 1
	for startPage := 1; startPage <= pageCount; startPage += previewChunkPages {
		endPage := startPage + previewChunkPages - 1
		if endPage > pageCount {
			endPage = pageCount
		}
		chunks = append(chunks, ChunkStatus{
			Index:     index,
			StartPage: startPage,
			EndPage:   endPage,
			Status:    JobStatusQueued,
		})
		index++
	}
	return chunks
}

func normalizeLanguage(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func (m *Manager) sharedWorkerHomeDir(request StartRequest) string {
	key := strings.Join([]string{
		normalizePDFSourceKey(request.PDFPath),
		strings.ToLower(strings.TrimSpace(request.SourceLang)),
		strings.ToLower(strings.TrimSpace(request.TargetLang)),
	}, "\n")
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(m.dataRootDir, "_worker_cache", hex.EncodeToString(sum[:]))
}

func (m *Manager) loadJobSnapshot(jobID string) (JobSnapshot, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return JobSnapshot{}, fmt.Errorf("job id is required")
	}
	if job, err := m.lookupJob(jobID); err == nil {
		return job.cloneSnapshot(), nil
	}
	stateFile := filepath.Join(m.dataRootDir, jobID, "job.json")
	var snapshot JobSnapshot
	if err := readJSONFile(stateFile, &snapshot); err != nil {
		return JobSnapshot{}, fmt.Errorf("load job %s: %w", jobID, err)
	}
	return snapshot, nil
}

func (m *Manager) loadReusablePreviewSnapshot(request StartRequest) (JobSnapshot, bool, error) {
	if request.Mode != ModeExport || request.ReusePreviewJobID == "" {
		return JobSnapshot{}, false, nil
	}
	previewSnapshot, err := m.loadJobSnapshot(request.ReusePreviewJobID)
	if err != nil {
		return JobSnapshot{}, false, nil
	}
	if !canReusePreviewForExport(previewSnapshot, request) {
		return JobSnapshot{}, false, nil
	}
	return previewSnapshot, true, nil
}

func canReusePreviewForExport(previewSnapshot JobSnapshot, request StartRequest) bool {
	if previewSnapshot.Mode != ModePreview || previewSnapshot.Status != JobStatusCompleted {
		return false
	}
	if previewSnapshot.PageCount != request.PageCount {
		return false
	}
	if !samePDFSource(previewSnapshot.PDFPath, request.PDFPath) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(previewSnapshot.SourceLang), strings.TrimSpace(request.SourceLang)) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(previewSnapshot.TargetLang), strings.TrimSpace(request.TargetLang)) {
		return false
	}
	if previewSnapshot.ProviderID != request.Provider.ProviderID {
		return false
	}
	if strings.TrimSpace(previewSnapshot.ModelID) != strings.TrimSpace(request.Provider.ModelID) {
		return false
	}
	if len(previewSnapshot.Chunks) == 0 {
		return false
	}
	for _, chunk := range previewSnapshot.Chunks {
		if chunk.Status != JobStatusCompleted {
			return false
		}
		if strings.TrimSpace(chunk.TranslatedPDFPath) == "" || strings.TrimSpace(chunk.DualPDFPath) == "" {
			return false
		}
	}
	return true
}

func samePDFSource(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" {
		return false
	}
	if looksLikeURL(left) || looksLikeURL(right) {
		return normalizePDFSourceKey(left) == normalizePDFSourceKey(right)
	}
	return normalizePDFSourceKey(left) == normalizePDFSourceKey(right)
}

func looksLikeURL(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func normalizePDFSourceKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if looksLikeURL(value) {
		return value
	}
	return strings.ToLower(filepath.Clean(value))
}

func previewChunkPaths(chunks []ChunkStatus, selectPath func(ChunkStatus) string) []string {
	paths := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		path := strings.TrimSpace(selectPath(chunk))
		if path == "" {
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

func isTerminalJobStatus(status JobStatus) bool {
	switch status {
	case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		return true
	default:
		return false
	}
}

func snapshotSortKey(snapshot JobSnapshot) time.Time {
	for _, candidate := range []string{snapshot.UpdatedAt, snapshot.FinishedAt, snapshot.StartedAt, snapshot.CreatedAt} {
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(candidate)); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func toJobOutputs(result workerTranslateResult) JobOutputs {
	return JobOutputs{
		OriginalPDFPath:           result.OriginalPDFPath,
		MonoPDFPath:               result.MonoPDFPath,
		DualPDFPath:               result.DualPDFPath,
		NoWatermarkMonoPDFPath:    result.NoWatermarkMonoPDFPath,
		NoWatermarkDualPDFPath:    result.NoWatermarkDualPDFPath,
		AutoExtractedGlossaryPath: result.AutoExtractedGlossaryPath,
		TotalSeconds:              result.TotalSeconds,
		PeakMemoryUsage:           result.PeakMemoryUsage,
	}
}

func (j *jobRuntime) cloneSnapshot() JobSnapshot {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return cloneSnapshot(j.snapshot)
}

func (j *jobRuntime) persist() error {
	j.mu.RLock()
	snapshot := cloneSnapshot(j.snapshot)
	j.mu.RUnlock()
	return writeJSONFile(j.stateFile, snapshot)
}

func (j *jobRuntime) emit(event JobEvent) {
	j.mu.Lock()
	j.sequence++
	event.Sequence = j.sequence
	event.JobID = j.snapshot.JobID
	event.Mode = j.snapshot.Mode
	event.Timestamp = nowRFC3339()
	event.JobStatus = j.snapshot.Status
	subscribers := make([]chan JobEvent, 0, len(j.subscribers))
	for _, subscriber := range j.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	j.mu.Unlock()

	for _, subscriber := range subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
	if j.eventSink != nil {
		j.eventSink(event)
	}
}

func (j *jobRuntime) closeSubscribers() {
	j.mu.Lock()
	defer j.mu.Unlock()
	for subID, subscriber := range j.subscribers {
		close(subscriber)
		delete(j.subscribers, subID)
	}
}

func (j *jobRuntime) markRunning() {
	j.mu.Lock()
	now := nowRFC3339()
	j.snapshot.Status = JobStatusRunning
	j.snapshot.StartedAt = now
	j.snapshot.UpdatedAt = now
	j.mu.Unlock()
	_ = j.persist()
}

func (j *jobRuntime) touchHeartbeat() JobSnapshot {
	j.mu.Lock()
	j.snapshot.UpdatedAt = nowRFC3339()
	snapshot := cloneSnapshot(j.snapshot)
	j.mu.Unlock()
	return snapshot
}

func (j *jobRuntime) markCompleted() {
	j.mu.Lock()
	now := nowRFC3339()
	j.snapshot.Status = JobStatusCompleted
	j.snapshot.FinishedAt = now
	j.snapshot.UpdatedAt = now
	j.mu.Unlock()
	_ = j.persist()
}

func (j *jobRuntime) markFailed(message string) {
	j.mu.Lock()
	now := nowRFC3339()
	j.snapshot.Status = JobStatusFailed
	j.snapshot.Error = strings.TrimSpace(message)
	j.snapshot.FinishedAt = now
	j.snapshot.UpdatedAt = now
	j.mu.Unlock()
	_ = j.persist()
}

func (j *jobRuntime) markCancelled() {
	j.mu.Lock()
	now := nowRFC3339()
	j.snapshot.Status = JobStatusCancelled
	j.snapshot.FinishedAt = now
	j.snapshot.UpdatedAt = now
	j.mu.Unlock()
	_ = j.persist()
}

func (j *jobRuntime) updateProgress(event workerEvent) {
	j.mu.Lock()
	j.snapshot.CurrentStage = event.Stage
	j.snapshot.StageProgress = event.StageProgress
	j.snapshot.OverallProgress = event.OverallProgress
	j.snapshot.StageCurrent = event.StageCurrent
	j.snapshot.StageTotal = event.StageTotal
	j.snapshot.PartIndex = event.PartIndex
	j.snapshot.TotalParts = event.TotalParts
	j.snapshot.UpdatedAt = nowRFC3339()
	j.mu.Unlock()
}

func (j *jobRuntime) markChunkRunning(index int) {
	j.mu.Lock()
	now := nowRFC3339()
	for chunkIndex := range j.snapshot.Chunks {
		if j.snapshot.Chunks[chunkIndex].Index == index {
			j.snapshot.Chunks[chunkIndex].Status = JobStatusRunning
			j.snapshot.Chunks[chunkIndex].StartedAt = now
			j.snapshot.Chunks[chunkIndex].Error = ""
			break
		}
	}
	j.snapshot.UpdatedAt = now
	j.mu.Unlock()
	_ = j.persist()
}

func (j *jobRuntime) markChunkCompleted(index int, output *JobOutputs) {
	j.mu.Lock()
	now := nowRFC3339()
	for chunkIndex := range j.snapshot.Chunks {
		if j.snapshot.Chunks[chunkIndex].Index != index {
			continue
		}
		j.snapshot.Chunks[chunkIndex].Status = JobStatusCompleted
		j.snapshot.Chunks[chunkIndex].FinishedAt = now
		if output != nil {
			j.snapshot.Chunks[chunkIndex].TranslatedPDFPath = firstNonEmpty(output.NoWatermarkMonoPDFPath, output.MonoPDFPath)
			j.snapshot.Chunks[chunkIndex].DualPDFPath = firstNonEmpty(output.NoWatermarkDualPDFPath, output.DualPDFPath)
			j.snapshot.Chunks[chunkIndex].TranslatedPageOffset = j.snapshot.Chunks[chunkIndex].StartPage - 1
			j.snapshot.Chunks[chunkIndex].TotalSeconds = output.TotalSeconds
		}
		break
	}
	j.snapshot.UpdatedAt = now
	j.mu.Unlock()
	_ = j.persist()
}

func (j *jobRuntime) markChunkFailed(index int, message string) {
	j.mu.Lock()
	now := nowRFC3339()
	for chunkIndex := range j.snapshot.Chunks {
		if j.snapshot.Chunks[chunkIndex].Index == index {
			j.snapshot.Chunks[chunkIndex].Status = JobStatusFailed
			j.snapshot.Chunks[chunkIndex].FinishedAt = now
			j.snapshot.Chunks[chunkIndex].Error = strings.TrimSpace(message)
			break
		}
	}
	j.snapshot.Error = strings.TrimSpace(message)
	j.snapshot.UpdatedAt = now
	j.mu.Unlock()
	_ = j.persist()
}

func (j *jobRuntime) setOutputs(output JobOutputs) {
	j.mu.Lock()
	j.snapshot.Outputs = output
	j.snapshot.UpdatedAt = nowRFC3339()
	j.mu.Unlock()
	_ = j.persist()
}

func (j *jobRuntime) currentOutput() *JobOutputs {
	j.mu.RLock()
	defer j.mu.RUnlock()
	output := j.snapshot.Outputs
	return &output
}

func (j *jobRuntime) chunkList() []ChunkStatus {
	j.mu.RLock()
	defer j.mu.RUnlock()
	chunks := make([]ChunkStatus, len(j.snapshot.Chunks))
	copy(chunks, j.snapshot.Chunks)
	return chunks
}

func (j *jobRuntime) chunkByIndex(index int) *ChunkStatus {
	j.mu.RLock()
	defer j.mu.RUnlock()
	for _, chunk := range j.snapshot.Chunks {
		if chunk.Index == index {
			copied := chunk
			return &copied
		}
	}
	return nil
}

func cloneSnapshot(snapshot JobSnapshot) JobSnapshot {
	copied := snapshot
	copied.Chunks = append([]ChunkStatus(nil), snapshot.Chunks...)
	return copied
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
