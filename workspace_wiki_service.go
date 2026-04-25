package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type workspaceKnowledgeExtractor interface {
	ExtractMarkdown(ctx context.Context, rawPath string) (PDFMarkdownPayload, error)
}

type workspaceKnowledgeLLM interface {
	GenerateWorkspaceKnowledgeBySource(ctx context.Context, providerID, modelID int64, prompt string) (WorkspaceKnowledgeBySourcePayload, error)
	GenerateWorkspaceKnowledgeMarkdown(ctx context.Context, providerID, modelID int64, prompt string) (string, error)
}

type workspaceWikiService struct {
	paths        appPaths
	store        *configStore
	pdf          workspaceKnowledgeExtractor
	knowledgeLLM workspaceKnowledgeLLM
	mu           sync.Mutex
	runningJobs  map[string]context.CancelFunc
}

func newWorkspaceWikiService(paths appPaths, store *configStore, pdf workspaceKnowledgeExtractor, knowledgeLLM workspaceKnowledgeLLM) *workspaceWikiService {
	return &workspaceWikiService{
		paths:        paths,
		store:        store,
		pdf:          pdf,
		knowledgeLLM: knowledgeLLM,
		runningJobs:  map[string]context.CancelFunc{},
	}
}

func (s *workspaceWikiService) Start(ctx context.Context, input WorkspaceWikiScanStartInput) (WorkspaceWikiScanJob, error) {
	return s.StartScan(ctx, input)
}

func (s *workspaceWikiService) Cancel(ctx context.Context, jobID string) (WorkspaceWikiScanJob, error) {
	return s.CancelScan(ctx, jobID)
}

func (s *workspaceWikiService) StartScan(ctx context.Context, input WorkspaceWikiScanStartInput) (WorkspaceWikiScanJob, error) {
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID == "" {
		return WorkspaceWikiScanJob{}, fmt.Errorf("workspace id is required")
	}
	if s.store == nil {
		return WorkspaceWikiScanJob{}, fmt.Errorf("config store is unavailable")
	}
	if s.pdf == nil {
		return WorkspaceWikiScanJob{}, fmt.Errorf("workspace knowledge extractor is unavailable")
	}
	if s.knowledgeLLM == nil {
		return WorkspaceWikiScanJob{}, fmt.Errorf("workspace knowledge llm is unavailable")
	}
	job, err := s.store.SaveWorkspaceWikiScanJob(ctx, WorkspaceWikiScanJob{
		WorkspaceID:  workspaceID,
		DocumentID:   strings.TrimSpace(input.DocumentID),
		Status:       WorkspaceWikiScanJobQueued,
		CurrentItem:  "",
		CurrentStage: "queued",
		Message:      "queued",
		Error:        "",
		ProviderID:   input.ProviderID,
		ModelID:      input.ModelID,
		StartedAt:    nowRFC3339(),
		UpdatedAt:    nowRFC3339(),
	})
	if err != nil {
		return WorkspaceWikiScanJob{}, err
	}

	runner := &workspaceWikiScanRunner{
		paths:        s.paths,
		store:        s.store,
		pdf:          s.pdf,
		knowledgeLLM: s.knowledgeLLM,
	}
	runCtx, cancel := context.WithCancel(context.Background())
	s.trackJob(job.JobID, cancel)
	go func() {
		defer s.untrackJob(job.JobID)
		runner.run(runCtx, job)
	}()
	return job, nil
}

func (s *workspaceWikiService) CancelScan(ctx context.Context, jobID string) (WorkspaceWikiScanJob, error) {
	trimmedJobID := strings.TrimSpace(jobID)
	if trimmedJobID == "" {
		return WorkspaceWikiScanJob{}, fmt.Errorf("job id is required")
	}
	if s.store == nil {
		return WorkspaceWikiScanJob{}, fmt.Errorf("config store is unavailable")
	}

	job, err := s.store.GetWorkspaceWikiScanJob(ctx, trimmedJobID)
	if err != nil {
		return WorkspaceWikiScanJob{}, err
	}
	if job.Status == WorkspaceWikiScanJobCompleted || job.Status == WorkspaceWikiScanJobFailed || job.Status == WorkspaceWikiScanJobCancelled {
		return job, nil
	}

	s.cancelTrackedJob(trimmedJobID)
	return s.store.UpdateWorkspaceWikiScanJob(ctx, trimmedJobID, workspaceWikiScanJobUpdate{
		Status:          WorkspaceWikiScanJobCancelled,
		CurrentItem:     "",
		CurrentStage:    "cancelled",
		Message:         "scan cancelled",
		OverallProgress: job.OverallProgress,
		Error:           "",
		Finished:        true,
	})
}

func (s *workspaceWikiService) trackJob(jobID string, cancel context.CancelFunc) {
	if strings.TrimSpace(jobID) == "" || cancel == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runningJobs[jobID] = cancel
}

func (s *workspaceWikiService) untrackJob(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.runningJobs, strings.TrimSpace(jobID))
}

func (s *workspaceWikiService) cancelTrackedJob(jobID string) {
	s.mu.Lock()
	cancel := s.runningJobs[strings.TrimSpace(jobID)]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

type workspaceWikiScanRunner struct {
	paths        appPaths
	store        *configStore
	pdf          workspaceKnowledgeExtractor
	knowledgeLLM workspaceKnowledgeLLM
}

type workspaceKnowledgeScanCandidate struct {
	absolutePath string
	title        string
	documentID   string
	kind         string
	contentHash  string
}

const (
	defaultWorkspaceKnowledgePromptSourceChars = 12000
	maxWorkspaceKnowledgePromptSourceChars     = 48000
	workspaceKnowledgePromptSafetyChars        = 128
)

func (r *workspaceWikiScanRunner) run(ctx context.Context, job WorkspaceWikiScanJob) {
	if ctx == nil {
		ctx = context.Background()
	}
	job = r.saveJob(ctx, job, WorkspaceWikiScanJobRunning, "scan", "scanning workspace", false)
	if err := r.runScan(ctx, job); err != nil {
		if errors.Is(err, context.Canceled) {
			_ = r.writeScanRunCancelled(job)
			_ = r.saveJob(ctx, job, WorkspaceWikiScanJobCancelled, "cancelled", "scan cancelled", true)
			return
		}
		_ = r.writeScanRunFailure(ctx, job, err)
		_ = r.saveJob(ctx, job, WorkspaceWikiScanJobFailed, "failed", err.Error(), true)
		return
	}
	r.saveJob(ctx, job, WorkspaceWikiScanJobCompleted, "completed", "scan completed", true)
}

func (r *workspaceWikiScanRunner) runScan(ctx context.Context, job WorkspaceWikiScanJob) error {
	if r.store == nil {
		return fmt.Errorf("config store is unavailable")
	}
	if r.pdf == nil {
		return fmt.Errorf("workspace knowledge extractor is unavailable")
	}
	if r.knowledgeLLM == nil {
		return fmt.Errorf("workspace knowledge llm is unavailable")
	}

	workspace, err := r.store.GetWorkspace(ctx, job.WorkspaceID)
	if err != nil {
		return err
	}

	files := newWorkspaceKnowledgeFiles(r.paths, workspace.ID)
	if err := files.EnsureLayout(); err != nil {
		return err
	}

	sources, err := r.collectSources(ctx, workspace, strings.TrimSpace(job.DocumentID), files)
	if err != nil {
		return err
	}
	job.TotalItems = len(sources)
	job.ProcessedItems = 0
	job.FailedItems = 0
	job.OverallProgress = 0
	job.CurrentItem = ""
	job = r.persistJob(ctx, job)
	if strings.TrimSpace(job.DocumentID) != "" && len(sources) == 0 {
		return fmt.Errorf("no scan sources found for document %s", strings.TrimSpace(job.DocumentID))
	}

	manifest, err := r.loadSourceManifest(files)
	if err != nil {
		return err
	}
	existingByID := workspaceKnowledgeSourcesByID(manifest)

	fullScan := strings.TrimSpace(job.DocumentID) == ""
	if fullScan {
		if err := r.removeStaleSourceArtifacts(files, sources); err != nil {
			return err
		}
	}

	manifest = mergeWorkspaceKnowledgeSources(manifest, sources, fullScan)
	if err := files.WriteSources(manifest); err != nil {
		return err
	}
	manifestByID := workspaceKnowledgeSourcesByID(manifest)

	failureCount := 0
	skippedCount := 0
	sourceIDs := make([]string, 0, len(sources))
	failedSourceIDs := make([]string, 0)
	for index := range sources {
		if err := ctx.Err(); err != nil {
			return err
		}
		sourceIDs = append(sourceIDs, sources[index].ID)
		job.CurrentItem = sources[index].Title
		job.CurrentStage = "scan"
		job.Message = fmt.Sprintf("scanning %s", sources[index].Title)
		job = r.persistJob(ctx, job)
		existingSource, hasExistingSource := existingByID[sources[index].ID]
		if hasExistingSource {
			skip, err := r.shouldSkipSource(files, sources[index], existingSource)
			if err != nil {
				return err
			}
			if skip {
				skippedCount++
				sources[index] = mergeSkippedWorkspaceKnowledgeSource(existingSource, sources[index])
				if err := persistWorkspaceKnowledgeSource(files, manifestByID, sources[index]); err != nil {
					return err
				}
				job.ProcessedItems++
				job.OverallProgress = workspaceWikiScanProgress(job.ProcessedItems, job.TotalItems)
				job.Message = fmt.Sprintf("skipped %s", sources[index].Title)
				job = r.persistJob(ctx, job)
				continue
			}
		}
		if err := r.processSource(ctx, workspace, job, files, &sources[index]); err != nil {
			failureCount++
			failedSourceIDs = append(failedSourceIDs, sources[index].ID)
			if err := persistWorkspaceKnowledgeSource(files, manifestByID, sources[index]); err != nil {
				return err
			}
			job.FailedItems = failureCount
			job.Error = err.Error()
			job.Message = err.Error()
		} else {
			if err := persistWorkspaceKnowledgeSource(files, manifestByID, sources[index]); err != nil {
				return err
			}
			job.Error = ""
		}
		job.ProcessedItems++
		job.OverallProgress = workspaceWikiScanProgress(job.ProcessedItems, job.TotalItems)
		job = r.persistJob(ctx, job)
	}

	manifest = mergeWorkspaceKnowledgeSources(manifest, sources, fullScan)
	if err := files.WriteSources(manifest); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	snapshot, err := CompileWorkspaceKnowledge(files, workspace.Name)
	if err != nil {
		return err
	}
	compileSummary, err := buildWorkspaceKnowledgeCompileSummary(files, workspace.ID, strings.TrimSpace(job.StartedAt), snapshot, failedSourceIDs)
	if err != nil {
		return err
	}
	if err := files.WriteCompileSummary(compileSummary); err != nil {
		return err
	}

	return files.WriteScanRun(WorkspaceKnowledgeScanRunRecord{
		ID:          strings.TrimSpace(job.JobID),
		WorkspaceID: workspace.ID,
		Status:      string(WorkspaceWikiScanJobCompleted),
		StartedAt:   firstNonEmptyText(strings.TrimSpace(job.StartedAt), nowRFC3339()),
		FinishedAt:  nowRFC3339(),
		SourceIDs:   sourceIDs,
		Message:     workspaceKnowledgeScanMessage(len(sources), failureCount, skippedCount),
	})
}

func (r *workspaceWikiScanRunner) collectSources(ctx context.Context, workspace Workspace, documentID string, files workspaceKnowledgeFiles) ([]WorkspaceKnowledgeSource, error) {
	workspaceRoot, err := files.workspaceRootDir()
	if err != nil {
		return nil, err
	}

	documents, err := r.store.ListDocumentsByWorkspace(ctx, workspace.ID)
	if err != nil {
		return nil, err
	}

	documentIDByPath := map[string]string{}
	documentTitleByPath := map[string]string{}
	for _, document := range documents {
		normalizedPath := normalizeWorkspaceKnowledgeAbsolutePath(document.PrimaryPDFPath)
		if normalizedPath == "" {
			continue
		}
		documentIDByPath[normalizedPath] = document.ID
		documentTitleByPath[normalizedPath] = strings.TrimSpace(document.Title)
	}

	candidates := make([]workspaceKnowledgeScanCandidate, 0)
	err = filepath.WalkDir(workspaceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if isInternalWorkspaceKnowledgeDir(workspaceRoot, path) {
				return fs.SkipDir
			}
			return nil
		}

		kind := workspaceKnowledgeSourceKind(path)
		if kind == "" {
			return nil
		}

		normalizedPath := normalizeWorkspaceKnowledgeAbsolutePath(path)
		mappedDocumentID := documentIDByPath[normalizedPath]
		if documentID != "" && mappedDocumentID != documentID {
			return nil
		}

		contentHash, err := sha256File(path)
		if err != nil {
			return fmt.Errorf("hash workspace source %s: %w", path, err)
		}
		title := strings.TrimSpace(documentTitleByPath[normalizedPath])
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		candidates = append(candidates, workspaceKnowledgeScanCandidate{
			absolutePath: path,
			title:        title,
			documentID:   mappedDocumentID,
			kind:         kind,
			contentHash:  contentHash,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk workspace sources: %w", err)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return normalizeWorkspaceKnowledgeAbsolutePath(candidates[i].absolutePath) < normalizeWorkspaceKnowledgeAbsolutePath(candidates[j].absolutePath)
	})

	sources := make([]WorkspaceKnowledgeSource, 0, len(candidates))
	for _, candidate := range candidates {
		baseSlug := workspaceKnowledgeSlug(strings.TrimSuffix(filepath.Base(candidate.absolutePath), filepath.Ext(candidate.absolutePath)))
		sourceKey, err := workspaceKnowledgeStableSourceKey(workspaceRoot, candidate)
		if err != nil {
			return nil, err
		}
		slug := workspaceKnowledgeStableSourceSlug(baseSlug, sourceKey)
		extractPath, err := files.ExtractPath(slug)
		if err != nil {
			return nil, err
		}
		sources = append(sources, WorkspaceKnowledgeSource{
			ID:               "source:" + slug,
			WorkspaceID:      workspace.ID,
			Title:            candidate.title,
			Slug:             slug,
			Kind:             candidate.kind,
			SourcePath:       candidate.absolutePath,
			MarkItDownPath:   extractPath,
			MarkItDownStatus: string(WorkspaceKnowledgeProcessingPending),
			ExtractStatus:    string(WorkspaceKnowledgeProcessingPending),
			AbsolutePath:     candidate.absolutePath,
			ContentHash:      candidate.contentHash,
			ExtractPath:      extractPath,
			DocumentID:       candidate.documentID,
		})
	}
	return sources, nil
}

func (r *workspaceWikiScanRunner) processSource(ctx context.Context, workspace Workspace, job WorkspaceWikiScanJob, files workspaceKnowledgeFiles, source *WorkspaceKnowledgeSource) error {
	source.LastIngestAt = nowRFC3339()
	source.LastError = ""
	source.MarkItDownStatus = string(WorkspaceKnowledgeProcessingRunning)
	source.ExtractStatus = string(WorkspaceKnowledgeProcessingPending)
	if err := updateWorkspaceKnowledgeSourceManifest(files, *source); err != nil {
		return err
	}

	markdown, err := r.extractSourceMarkdown(ctx, *source)
	if err != nil {
		source.MarkItDownStatus = string(WorkspaceKnowledgeProcessingFailed)
		source.ExtractStatus = string(WorkspaceKnowledgeProcessingPending)
		source.LastError = err.Error()
		return err
	}
	markItDownPath := firstNonEmptyText(strings.TrimSpace(source.MarkItDownPath), strings.TrimSpace(source.ExtractPath))
	source.MarkItDownPath = markItDownPath
	source.ExtractPath = markItDownPath
	if err := writeWorkspaceKnowledgeMarkdown(markItDownPath, markdown); err != nil {
		source.MarkItDownStatus = string(WorkspaceKnowledgeProcessingFailed)
		source.ExtractStatus = string(WorkspaceKnowledgeProcessingPending)
		source.LastError = err.Error()
		return err
	}
	source.MarkItDownStatus = string(WorkspaceKnowledgeProcessingReady)
	source.ExtractStatus = string(WorkspaceKnowledgeProcessingRunning)
	source.LastError = ""
	if err := updateWorkspaceKnowledgeSourceManifest(files, *source); err != nil {
		return err
	}

	prompt, err := buildWorkspaceKnowledgeBySourcePromptWithinBudget(workspace, *source, markdown, r.workspaceKnowledgePromptCharBudget(ctx, job.ModelID))
	if err != nil {
		source.ExtractStatus = string(WorkspaceKnowledgeProcessingFailed)
		source.LastError = err.Error()
		return err
	}
	payload, err := r.knowledgeLLM.GenerateWorkspaceKnowledgeBySource(ctx, job.ProviderID, job.ModelID, prompt)
	if err != nil {
		source.ExtractStatus = string(WorkspaceKnowledgeProcessingFailed)
		source.LastError = fmt.Sprintf("generate by-source knowledge for %s: %v", source.Title, err)
		return fmt.Errorf("generate by-source knowledge for %s: %w", source.Title, err)
	}

	payload = normalizeWorkspaceKnowledgeBySourcePayload(*source, payload)
	if err := files.WriteBySource(source.Slug, payload); err != nil {
		source.ExtractStatus = string(WorkspaceKnowledgeProcessingFailed)
		source.LastError = err.Error()
		return err
	}

	source.MarkItDownStatus = string(WorkspaceKnowledgeProcessingReady)
	source.ExtractStatus = string(WorkspaceKnowledgeProcessingReady)
	source.LastError = ""
	source.LastSuccessAt = nowRFC3339()
	return nil
}

func (r *workspaceWikiScanRunner) extractSourceMarkdown(ctx context.Context, source WorkspaceKnowledgeSource) (string, error) {
	switch strings.ToLower(strings.TrimSpace(source.Kind)) {
	case "pdf":
		payload, err := r.pdf.ExtractMarkdown(ctx, source.AbsolutePath)
		if err != nil {
			return "", fmt.Errorf("extract markdown from %s: %w", source.AbsolutePath, err)
		}
		markdown := strings.TrimSpace(payload.Markdown)
		if markdown == "" {
			return "", fmt.Errorf("extract markdown from %s: empty markdown", source.AbsolutePath)
		}
		return markdown, nil
	case "markdown":
		content, err := os.ReadFile(source.AbsolutePath)
		if err != nil {
			return "", fmt.Errorf("read markdown source %s: %w", source.AbsolutePath, err)
		}
		markdown := strings.TrimSpace(string(content))
		if markdown == "" {
			return "", fmt.Errorf("markdown source %s is empty", source.AbsolutePath)
		}
		return markdown, nil
	default:
		return "", fmt.Errorf("unsupported workspace knowledge source kind %s", source.Kind)
	}
}

func (r *workspaceWikiScanRunner) loadSourceManifest(files workspaceKnowledgeFiles) ([]WorkspaceKnowledgeSource, error) {
	return files.ReadSources()
}

func (r *workspaceWikiScanRunner) removeStaleSourceArtifacts(files workspaceKnowledgeFiles, currentSources []WorkspaceKnowledgeSource) error {
	currentSlugs := make(map[string]struct{}, len(currentSources))
	for _, source := range currentSources {
		currentSlugs[source.Slug] = struct{}{}
	}

	extractsDir, err := files.extractsDir()
	if err != nil {
		return err
	}
	if err := removeWorkspaceKnowledgeArtifactsNotInSet(extractsDir, ".md", currentSlugs); err != nil {
		return err
	}

	bySourceDir, err := files.bySourceDir()
	if err != nil {
		return err
	}
	return removeWorkspaceKnowledgeArtifactsNotInSet(bySourceDir, ".json", currentSlugs)
}

func (r *workspaceWikiScanRunner) saveJob(ctx context.Context, job WorkspaceWikiScanJob, status WorkspaceWikiScanJobStatus, stage, message string, finished bool) WorkspaceWikiScanJob {
	if r.store == nil {
		return job
	}
	job.Status = status
	job.CurrentStage = strings.TrimSpace(stage)
	job.Message = strings.TrimSpace(message)
	job.UpdatedAt = nowRFC3339()
	if strings.TrimSpace(job.StartedAt) == "" {
		job.StartedAt = job.UpdatedAt
	}
	if finished {
		job.FinishedAt = job.UpdatedAt
	}
	if status == WorkspaceWikiScanJobFailed {
		job.Error = strings.TrimSpace(message)
	} else {
		job.Error = ""
	}
	saved, err := r.store.SaveWorkspaceWikiScanJob(ctx, job)
	if err != nil {
		return job
	}
	return saved
}

func (r *workspaceWikiScanRunner) persistJob(ctx context.Context, job WorkspaceWikiScanJob) WorkspaceWikiScanJob {
	if r.store == nil {
		return job
	}
	job.UpdatedAt = nowRFC3339()
	saved, err := r.store.SaveWorkspaceWikiScanJob(ctx, job)
	if err != nil {
		return job
	}
	return saved
}

func (r *workspaceWikiScanRunner) shouldSkipSource(files workspaceKnowledgeFiles, currentSource, existingSource WorkspaceKnowledgeSource) (bool, error) {
	if existingSource.ID == "" {
		return false, nil
	}
	if existingSource.ContentHash != currentSource.ContentHash {
		return false, nil
	}
	if existingSource.MarkItDownStatus != string(WorkspaceKnowledgeProcessingReady) || existingSource.ExtractStatus != string(WorkspaceKnowledgeProcessingReady) {
		return false, nil
	}
	if !workspaceKnowledgeFileExists(firstNonEmptyText(currentSource.MarkItDownPath, existingSource.MarkItDownPath, currentSource.ExtractPath, existingSource.ExtractPath)) {
		return false, nil
	}

	bySourcePath, err := files.BySourcePath(currentSource.Slug)
	if err != nil {
		return false, err
	}
	if !workspaceKnowledgeFileExists(bySourcePath) {
		return false, nil
	}
	return true, nil
}

func (r *workspaceWikiScanRunner) workspaceKnowledgePromptCharBudget(ctx context.Context, modelID int64) int {
	budget := defaultWorkspaceKnowledgePromptSourceChars
	if r.store == nil || modelID <= 0 {
		return budget
	}

	model, err := r.store.GetModel(ctx, modelID)
	if err != nil || model.ContextWindow <= 0 {
		return budget
	}

	budget = model.ContextWindow*5 - workspaceKnowledgePromptSafetyChars
	if budget < 0 {
		budget = 0
	}
	if budget > maxWorkspaceKnowledgePromptSourceChars {
		return maxWorkspaceKnowledgePromptSourceChars
	}
	return budget
}

func (r *workspaceWikiScanRunner) writeScanRunFailure(ctx context.Context, job WorkspaceWikiScanJob, runErr error) error {
	workspaceID := strings.TrimSpace(job.WorkspaceID)
	if workspaceID == "" {
		return nil
	}
	files := newWorkspaceKnowledgeFiles(r.paths, workspaceID)
	if err := files.EnsureLayout(); err != nil {
		return err
	}
	return files.WriteScanRun(WorkspaceKnowledgeScanRunRecord{
		ID:          strings.TrimSpace(job.JobID),
		WorkspaceID: workspaceID,
		Status:      string(WorkspaceWikiScanJobFailed),
		StartedAt:   firstNonEmptyText(strings.TrimSpace(job.StartedAt), nowRFC3339()),
		FinishedAt:  nowRFC3339(),
		Message:     strings.TrimSpace(runErr.Error()),
	})
}

func (r *workspaceWikiScanRunner) writeScanRunCancelled(job WorkspaceWikiScanJob) error {
	workspaceID := strings.TrimSpace(job.WorkspaceID)
	if workspaceID == "" {
		return nil
	}
	files := newWorkspaceKnowledgeFiles(r.paths, workspaceID)
	if err := files.EnsureLayout(); err != nil {
		return err
	}
	return files.WriteScanRun(WorkspaceKnowledgeScanRunRecord{
		ID:          strings.TrimSpace(job.JobID),
		WorkspaceID: workspaceID,
		Status:      string(WorkspaceWikiScanJobCancelled),
		StartedAt:   firstNonEmptyText(strings.TrimSpace(job.StartedAt), nowRFC3339()),
		FinishedAt:  nowRFC3339(),
		Message:     "scan cancelled",
	})
}

func workspaceWikiScanProgress(processedItems, totalItems int) float64 {
	if totalItems <= 0 {
		return 0
	}
	if processedItems >= totalItems {
		return 1
	}
	if processedItems <= 0 {
		return 0
	}
	return float64(processedItems) / float64(totalItems)
}

func mergeWorkspaceKnowledgeSources(existing, updated []WorkspaceKnowledgeSource, fullScan bool) []WorkspaceKnowledgeSource {
	if fullScan {
		sort.Slice(updated, func(i, j int) bool {
			return lessSource(updated[i], updated[j])
		})
		return updated
	}

	mergedByID := map[string]WorkspaceKnowledgeSource{}
	for _, source := range existing {
		mergedByID[source.ID] = source
	}
	for _, source := range updated {
		mergedByID[source.ID] = source
	}

	merged := make([]WorkspaceKnowledgeSource, 0, len(mergedByID))
	for _, source := range mergedByID {
		merged = append(merged, source)
	}
	sort.Slice(merged, func(i, j int) bool {
		return lessSource(merged[i], merged[j])
	})
	return merged
}

func workspaceKnowledgeSourcesByID(sources []WorkspaceKnowledgeSource) map[string]WorkspaceKnowledgeSource {
	byID := make(map[string]WorkspaceKnowledgeSource, len(sources))
	for _, source := range sources {
		byID[source.ID] = source
	}
	return byID
}

func mergeSkippedWorkspaceKnowledgeSource(existingSource, currentSource WorkspaceKnowledgeSource) WorkspaceKnowledgeSource {
	existingSource.WorkspaceID = currentSource.WorkspaceID
	existingSource.Title = currentSource.Title
	existingSource.Slug = currentSource.Slug
	existingSource.Kind = currentSource.Kind
	existingSource.SourcePath = currentSource.SourcePath
	existingSource.MarkItDownPath = firstNonEmptyText(currentSource.MarkItDownPath, existingSource.MarkItDownPath)
	existingSource.AbsolutePath = currentSource.AbsolutePath
	existingSource.ContentHash = currentSource.ContentHash
	existingSource.ExtractPath = currentSource.ExtractPath
	existingSource.DocumentID = currentSource.DocumentID
	return existingSource
}

func persistWorkspaceKnowledgeSource(files workspaceKnowledgeFiles, manifestByID map[string]WorkspaceKnowledgeSource, source WorkspaceKnowledgeSource) error {
	manifestByID[source.ID] = source
	manifest := make([]WorkspaceKnowledgeSource, 0, len(manifestByID))
	for _, item := range manifestByID {
		manifest = append(manifest, item)
	}
	sort.Slice(manifest, func(i, j int) bool {
		return lessSource(manifest[i], manifest[j])
	})
	return files.WriteSources(manifest)
}

func updateWorkspaceKnowledgeSourceManifest(files workspaceKnowledgeFiles, source WorkspaceKnowledgeSource) error {
	sources, err := files.ReadSources()
	if err != nil {
		return err
	}
	return files.WriteSources(mergeWorkspaceKnowledgeSources(sources, []WorkspaceKnowledgeSource{source}, false))
}

func buildWorkspaceKnowledgeCompileSummary(files workspaceKnowledgeFiles, workspaceID, startedAt string, snapshot WorkspaceKnowledgeSnapshot, failedSourceIDs []string) (WorkspaceKnowledgeCompileSummary, error) {
	updatedWikiPaths, err := workspaceKnowledgeUpdatedWikiPaths(files, snapshot)
	if err != nil {
		return WorkspaceKnowledgeCompileSummary{}, err
	}
	includedSourceIDs := make([]string, 0, len(snapshot.Sources))
	for _, source := range snapshot.Sources {
		includedSourceIDs = append(includedSourceIDs, source.ID)
	}

	return WorkspaceKnowledgeCompileSummary{
		WorkspaceID:       strings.TrimSpace(workspaceID),
		StartedAt:         firstNonEmptyText(strings.TrimSpace(startedAt), nowRFC3339()),
		FinishedAt:        nowRFC3339(),
		IncludedSourceIDs: includedSourceIDs,
		FailedSourceIDs:   append([]string(nil), failedSourceIDs...),
		UpdatedWikiPaths:  updatedWikiPaths,
		CompileDirty:      false,
		WikiDirty:         false,
	}, nil
}

func workspaceKnowledgeUpdatedWikiPaths(files workspaceKnowledgeFiles, snapshot WorkspaceKnowledgeSnapshot) ([]string, error) {
	paths := make([]string, 0, len(snapshot.Sources)+len(snapshot.Entities)+2)

	overviewPath, err := files.OverviewPath()
	if err != nil {
		return nil, err
	}
	paths = append(paths, overviewPath)

	openQuestionsPath, err := files.OpenQuestionsPath()
	if err != nil {
		return nil, err
	}
	paths = append(paths, openQuestionsPath)

	for _, source := range snapshot.Sources {
		documentPath, err := files.DocumentWikiPath(source.Slug)
		if err != nil {
			return nil, err
		}
		paths = append(paths, documentPath)
	}

	for _, conceptSlug := range buildConceptSlugs(snapshot.Entities) {
		conceptPath, err := files.ConceptWikiPath(conceptSlug)
		if err != nil {
			return nil, err
		}
		paths = append(paths, conceptPath)
	}

	sort.Strings(paths)
	return paths, nil
}

func normalizeWorkspaceKnowledgeBySourcePayload(source WorkspaceKnowledgeSource, payload WorkspaceKnowledgeBySourcePayload) WorkspaceKnowledgeBySourcePayload {
	now := nowRFC3339()
	payload.Source = source
	if payload.Entities == nil {
		payload.Entities = []WorkspaceKnowledgeEntity{}
	}
	if payload.Claims == nil {
		payload.Claims = []WorkspaceKnowledgeClaim{}
	}
	if payload.Relations == nil {
		payload.Relations = []WorkspaceKnowledgeRelation{}
	}
	if payload.Tasks == nil {
		payload.Tasks = []WorkspaceKnowledgeTask{}
	}

	for index := range payload.Entities {
		if strings.TrimSpace(payload.Entities[index].WorkspaceID) == "" {
			payload.Entities[index].WorkspaceID = source.WorkspaceID
		}
		if strings.TrimSpace(payload.Entities[index].Origin) == "" {
			payload.Entities[index].Origin = "scan"
		}
		if strings.TrimSpace(payload.Entities[index].CreatedAt) == "" {
			payload.Entities[index].CreatedAt = now
		}
		if strings.TrimSpace(payload.Entities[index].UpdatedAt) == "" {
			payload.Entities[index].UpdatedAt = now
		}
	}
	for index := range payload.Claims {
		if strings.TrimSpace(payload.Claims[index].WorkspaceID) == "" {
			payload.Claims[index].WorkspaceID = source.WorkspaceID
		}
		if strings.TrimSpace(payload.Claims[index].Origin) == "" {
			payload.Claims[index].Origin = "scan"
		}
		if strings.TrimSpace(payload.Claims[index].CreatedAt) == "" {
			payload.Claims[index].CreatedAt = now
		}
		if strings.TrimSpace(payload.Claims[index].UpdatedAt) == "" {
			payload.Claims[index].UpdatedAt = now
		}
	}
	for index := range payload.Relations {
		if strings.TrimSpace(payload.Relations[index].WorkspaceID) == "" {
			payload.Relations[index].WorkspaceID = source.WorkspaceID
		}
		if strings.TrimSpace(payload.Relations[index].Origin) == "" {
			payload.Relations[index].Origin = "scan"
		}
		if strings.TrimSpace(payload.Relations[index].CreatedAt) == "" {
			payload.Relations[index].CreatedAt = now
		}
		if strings.TrimSpace(payload.Relations[index].UpdatedAt) == "" {
			payload.Relations[index].UpdatedAt = now
		}
	}
	for index := range payload.Tasks {
		if strings.TrimSpace(payload.Tasks[index].WorkspaceID) == "" {
			payload.Tasks[index].WorkspaceID = source.WorkspaceID
		}
		if strings.TrimSpace(payload.Tasks[index].Origin) == "" {
			payload.Tasks[index].Origin = "scan"
		}
		if strings.TrimSpace(payload.Tasks[index].CreatedAt) == "" {
			payload.Tasks[index].CreatedAt = now
		}
		if strings.TrimSpace(payload.Tasks[index].UpdatedAt) == "" {
			payload.Tasks[index].UpdatedAt = now
		}
	}
	return payload
}

func workspaceKnowledgeSourceKind(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		return "pdf"
	case ".md", ".markdown":
		return "markdown"
	default:
		return ""
	}
}

func isInternalWorkspaceKnowledgeDir(workspaceRoot, path string) bool {
	normalizedPath := normalizeWorkspaceKnowledgeAbsolutePath(path)
	for _, dirName := range []string{"raw", "schema", "sources", "inputs", "state", "wiki"} {
		if normalizedPath == normalizeWorkspaceKnowledgeAbsolutePath(filepath.Join(workspaceRoot, dirName)) {
			return true
		}
	}
	return false
}

func workspaceKnowledgeStableSourceKey(workspaceRoot string, candidate workspaceKnowledgeScanCandidate) (string, error) {
	if trimmedDocumentID := strings.TrimSpace(candidate.documentID); trimmedDocumentID != "" {
		return "document:" + trimmedDocumentID, nil
	}
	relativePath, err := filepath.Rel(workspaceRoot, candidate.absolutePath)
	if err != nil {
		return "", fmt.Errorf("resolve workspace knowledge relative path for %s: %w", candidate.absolutePath, err)
	}
	return "path:" + normalizeWorkspaceKnowledgeRelativePath(relativePath), nil
}

func workspaceKnowledgeStableSourceSlug(baseSlug, sourceKey string) string {
	normalizedBaseSlug := strings.TrimSpace(baseSlug)
	if normalizedBaseSlug == "" {
		normalizedBaseSlug = "item"
	}
	keyHash := sha256Hex([]byte(strings.TrimSpace(sourceKey)))
	if len(keyHash) > 12 {
		keyHash = keyHash[:12]
	}
	if keyHash == "" {
		return normalizedBaseSlug
	}
	return normalizedBaseSlug + "-" + keyHash
}

func normalizeWorkspaceKnowledgeAbsolutePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(filepath.Clean(trimmed))
}

func normalizeWorkspaceKnowledgeRelativePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(filepath.ToSlash(filepath.Clean(trimmed)))
}

func buildWorkspaceKnowledgeBySourcePromptWithinBudget(workspace Workspace, source WorkspaceKnowledgeSource, markdown string, totalPromptBudget int) (string, error) {
	if totalPromptBudget <= 0 {
		return "", fmt.Errorf("prompt budget too small for workspace knowledge extraction")
	}

	emptyPrompt := buildWorkspaceKnowledgeBySourcePrompt(workspace, source, "")
	if len(emptyPrompt) >= totalPromptBudget {
		return "", fmt.Errorf("prompt budget too small for workspace knowledge extraction")
	}

	maxSourceChars := totalPromptBudget - len(emptyPrompt)
	bestPrompt := emptyPrompt
	low := 0
	high := maxSourceChars
	for low <= high {
		mid := (low + high) / 2
		candidateMarkdown := trimWorkspaceKnowledgePromptMarkdown(markdown, mid)
		candidatePrompt := buildWorkspaceKnowledgeBySourcePrompt(workspace, source, candidateMarkdown)
		if len(candidatePrompt) <= totalPromptBudget {
			bestPrompt = candidatePrompt
			low = mid + 1
			continue
		}
		high = mid - 1
	}
	return bestPrompt, nil
}

func trimWorkspaceKnowledgePromptMarkdown(markdown string, maxChars int) string {
	trimmedMarkdown := strings.TrimSpace(markdown)
	if trimmedMarkdown == "" {
		return ""
	}
	if maxChars <= 0 {
		maxChars = defaultWorkspaceKnowledgePromptSourceChars
	}

	runes := []rune(trimmedMarkdown)
	if len(runes) <= maxChars {
		return trimmedMarkdown
	}

	headCount := (maxChars * 3) / 4
	if headCount <= 0 {
		headCount = maxChars / 2
	}
	if headCount <= 0 {
		headCount = 1
	}
	tailCount := maxChars - headCount
	if tailCount <= 0 {
		tailCount = 1
		if headCount > 1 {
			headCount = maxChars - tailCount
		}
	}

	omittedCount := len(runes) - (headCount + tailCount)
	if omittedCount <= 0 {
		return string(runes[:maxChars])
	}

	return string(runes[:headCount]) +
		fmt.Sprintf("\n\n[... trimmed %d characters for scan prompt ...]\n\n", omittedCount) +
		string(runes[len(runes)-tailCount:])
}

func workspaceKnowledgeScanMessage(totalSources, failedSources, skippedSources int) string {
	message := fmt.Sprintf("processed %d sources", totalSources)
	details := make([]string, 0, 2)
	if failedSources > 0 {
		details = append(details, fmt.Sprintf("%d failed", failedSources))
	}
	if skippedSources > 0 {
		details = append(details, fmt.Sprintf("%d skipped", skippedSources))
	}
	if len(details) == 0 {
		return message
	}
	return fmt.Sprintf("%s (%s)", message, strings.Join(details, ", "))
}

func removeWorkspaceKnowledgeArtifactsNotInSet(dir, extension string, keep map[string]struct{}) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read workspace knowledge directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), extension) {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if _, ok := keep[slug]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return fmt.Errorf("remove workspace knowledge artifact %s: %w", filepath.Join(dir, entry.Name()), err)
		}
	}
	return nil
}

func removeWorkspaceKnowledgeFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove workspace knowledge file %s: %w", path, err)
	}
	return nil
}

func workspaceKnowledgeFileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
