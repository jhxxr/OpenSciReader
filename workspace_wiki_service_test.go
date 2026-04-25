package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCollectSourcesSkipsInternalWorkspaceKnowledgeDirectories(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.appDB.Close()
		_ = store.ocrDB.Close()
	})

	ctx := context.Background()
	workspace, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{ID: "workspace-a", Name: "Workspace A"})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	files := newWorkspaceKnowledgeFiles(paths, workspace.ID)
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}

	workspaceRoot, err := files.workspaceRootDir()
	if err != nil {
		t.Fatalf("workspaceRootDir() error = %v", err)
	}

	userPDFPath := filepath.Join(workspaceRoot, "paper-a.pdf")
	if err := os.WriteFile(userPDFPath, []byte("user pdf"), 0o600); err != nil {
		t.Fatalf("WriteFile(userPDFPath) error = %v", err)
	}

	internalMarkdownPath := filepath.Join(workspaceRoot, "inputs", "markitdown", "paper-a.md")
	if err := os.WriteFile(internalMarkdownPath, []byte("generated markdown"), 0o600); err != nil {
		t.Fatalf("WriteFile(internalMarkdownPath) error = %v", err)
	}

	internalPDFPath := filepath.Join(workspaceRoot, "sources", "pdfs", "paper-a.pdf")
	if err := os.WriteFile(internalPDFPath, []byte("generated pdf"), 0o600); err != nil {
		t.Fatalf("WriteFile(internalPDFPath) error = %v", err)
	}

	runner := &workspaceWikiScanRunner{paths: paths, store: store}
	sources, err := runner.collectSources(ctx, workspace, "", files)
	if err != nil {
		t.Fatalf("collectSources() error = %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("collectSources() returned %d sources, want 1", len(sources))
	}
	if sources[0].AbsolutePath != userPDFPath {
		t.Fatalf("collectSources()[0].AbsolutePath = %q, want %q", sources[0].AbsolutePath, userPDFPath)
	}
	if sources[0].Kind != "pdf" {
		t.Fatalf("collectSources()[0].Kind = %q, want %q", sources[0].Kind, "pdf")
	}
}

func TestStartWorkspaceWikiScanPersistsSourceProcessingState(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.appDB.Close()
		_ = store.ocrDB.Close()
	})

	ctx := context.Background()
	workspace, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{ID: "workspace-a", Name: "Workspace A"})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	seedPath := filepath.Join(t.TempDir(), "seed.md")
	if err := os.WriteFile(seedPath, []byte("# Seed\n\nA markdown source for workspace wiki scans."), 0o600); err != nil {
		t.Fatalf("WriteFile(seedPath) error = %v", err)
	}

	importResult, err := store.ImportFiles(ctx, paths, ImportFilesInput{
		WorkspaceID: workspace.ID,
		FilePaths:   []string{seedPath},
		SourceType:  "manual",
	})
	if err != nil {
		t.Fatalf("ImportFiles() error = %v", err)
	}
	if len(importResult.Documents) != 1 {
		t.Fatalf("ImportFiles() documents = %d, want 1", len(importResult.Documents))
	}

	service := newWorkspaceWikiService(paths, store, panicWorkspaceKnowledgeExtractor{}, stubWorkspaceKnowledgeLLM{})
	job, err := service.StartScan(ctx, WorkspaceWikiScanStartInput{WorkspaceID: workspace.ID})
	if err != nil {
		t.Fatalf("StartScan() error = %v", err)
	}

	job = waitForWorkspaceWikiJobTerminal(t, store, job.JobID)
	if job.Status != WorkspaceWikiScanJobCompleted {
		t.Fatalf("job.Status = %q, want %q (error=%q)", job.Status, WorkspaceWikiScanJobCompleted, job.Error)
	}

	files := newWorkspaceKnowledgeFiles(paths, workspace.ID)
	sources, err := files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources() error = %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("ReadSources() len = %d, want 1", len(sources))
	}
	if sources[0].MarkItDownStatus != "ready" {
		t.Fatalf("sources[0].MarkItDownStatus = %q, want %q", sources[0].MarkItDownStatus, "ready")
	}
	if sources[0].ExtractStatus != "ready" {
		t.Fatalf("sources[0].ExtractStatus = %q, want %q", sources[0].ExtractStatus, "ready")
	}

	summary, err := files.ReadCompileSummary()
	if err != nil {
		t.Fatalf("ReadCompileSummary() error = %v", err)
	}
	indexPath, err := files.IndexPath()
	if err != nil {
		t.Fatalf("IndexPath() error = %v", err)
	}
	if !containsString(summary.UpdatedWikiPaths, indexPath) {
		t.Fatalf("compile summary UpdatedWikiPaths = %#v, want to contain %q", summary.UpdatedWikiPaths, indexPath)
	}
	overviewPath, err := files.OverviewPath()
	if err != nil {
		t.Fatalf("OverviewPath() error = %v", err)
	}
	if !containsString(summary.UpdatedWikiPaths, overviewPath) {
		t.Fatalf("compile summary UpdatedWikiPaths = %#v, want to contain %q", summary.UpdatedWikiPaths, overviewPath)
	}
	logPath, err := files.LogPath()
	if err != nil {
		t.Fatalf("LogPath() error = %v", err)
	}
	if !containsString(summary.UpdatedWikiPaths, logPath) {
		t.Fatalf("compile summary UpdatedWikiPaths = %#v, want to contain %q", summary.UpdatedWikiPaths, logPath)
	}
}

func TestStartWorkspaceWikiScanClearsStaleKnowledgeAfterRerunFailure(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.appDB.Close()
		_ = store.ocrDB.Close()
	})

	ctx := context.Background()
	workspace, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{ID: "workspace-a", Name: "Workspace A"})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	seedPath := filepath.Join(t.TempDir(), "seed.md")
	if err := os.WriteFile(seedPath, []byte("# Seed\n\nA markdown source for rerun failure coverage."), 0o600); err != nil {
		t.Fatalf("WriteFile(seedPath) error = %v", err)
	}

	importResult, err := store.ImportFiles(ctx, paths, ImportFilesInput{
		WorkspaceID: workspace.ID,
		FilePaths:   []string{seedPath},
		SourceType:  "manual",
	})
	if err != nil {
		t.Fatalf("ImportFiles() error = %v", err)
	}
	if len(importResult.Documents) != 1 {
		t.Fatalf("ImportFiles() documents = %d, want 1", len(importResult.Documents))
	}

	files := newWorkspaceKnowledgeFiles(paths, workspace.ID)
	service := newWorkspaceWikiService(paths, store, panicWorkspaceKnowledgeExtractor{}, stubWorkspaceKnowledgeLLM{
		payload: WorkspaceKnowledgeBySourcePayload{
			Entities: []WorkspaceKnowledgeEntity{{ID: "entity:seed", Title: "Seed Entity", Type: "concept", Summary: "seed summary"}},
		},
	})

	firstJob, err := service.StartScan(ctx, WorkspaceWikiScanStartInput{WorkspaceID: workspace.ID})
	if err != nil {
		t.Fatalf("first StartScan() error = %v", err)
	}
	firstJob = waitForWorkspaceWikiJobTerminal(t, store, firstJob.JobID)
	if firstJob.Status != WorkspaceWikiScanJobCompleted {
		t.Fatalf("first job.Status = %q, want %q (error=%q)", firstJob.Status, WorkspaceWikiScanJobCompleted, firstJob.Error)
	}

	firstSources, err := files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources() after first run error = %v", err)
	}
	if len(firstSources) != 1 {
		t.Fatalf("ReadSources() after first run len = %d, want 1", len(firstSources))
	}
	firstSuccessAt := firstSources[0].LastSuccessAt
	if firstSuccessAt == "" {
		t.Fatalf("first source LastSuccessAt = %q, want non-empty", firstSuccessAt)
	}
	bySourcePath, err := files.BySourcePath(firstSources[0].Slug)
	if err != nil {
		t.Fatalf("BySourcePath() error = %v", err)
	}
	if _, err := os.Stat(bySourcePath); err != nil {
		t.Fatalf("Stat(bySourcePath) after first run error = %v", err)
	}
	if err := os.WriteFile(importResult.Documents[0].PrimaryPDFPath, []byte("# Seed\n\nUpdated content that forces a rerun."), 0o600); err != nil {
		t.Fatalf("WriteFile(imported source) before second run error = %v", err)
	}

	failingService := newWorkspaceWikiService(paths, store, panicWorkspaceKnowledgeExtractor{}, stubWorkspaceKnowledgeLLM{
		bySourceErr: fmt.Errorf("llm exploded"),
	})
	secondJob, err := failingService.StartScan(ctx, WorkspaceWikiScanStartInput{WorkspaceID: workspace.ID})
	if err != nil {
		t.Fatalf("second StartScan() error = %v", err)
	}
	secondJob = waitForWorkspaceWikiJobTerminal(t, store, secondJob.JobID)
	if secondJob.Status != WorkspaceWikiScanJobCompleted {
		t.Fatalf("second job.Status = %q, want %q (error=%q)", secondJob.Status, WorkspaceWikiScanJobCompleted, secondJob.Error)
	}

	secondSources, err := files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources() after second run error = %v", err)
	}
	if len(secondSources) != 1 {
		t.Fatalf("ReadSources() after second run len = %d, want 1", len(secondSources))
	}
	if secondSources[0].LastSuccessAt != firstSuccessAt {
		t.Fatalf("second source LastSuccessAt = %q, want preserved %q", secondSources[0].LastSuccessAt, firstSuccessAt)
	}
	if secondSources[0].ExtractStatus != "failed" {
		t.Fatalf("second source ExtractStatus = %q, want %q", secondSources[0].ExtractStatus, "failed")
	}
	if secondSources[0].LastError == "" {
		t.Fatalf("second source LastError = %q, want non-empty", secondSources[0].LastError)
	}
	if _, err := os.Stat(bySourcePath); !os.IsNotExist(err) {
		t.Fatalf("Stat(bySourcePath) after second run error = %v, want not exist", err)
	}

	entitiesPath, err := files.EntitiesPath()
	if err != nil {
		t.Fatalf("EntitiesPath() error = %v", err)
	}
	var entities []WorkspaceKnowledgeEntity
	if err := readJSONFile(entitiesPath, &entities); err != nil {
		t.Fatalf("readJSONFile(entitiesPath) error = %v", err)
	}
	if len(entities) != 0 {
		t.Fatalf("entities after failed rerun = %d, want 0", len(entities))
	}

	summary, err := files.ReadCompileSummary()
	if err != nil {
		t.Fatalf("ReadCompileSummary() after second run error = %v", err)
	}
	if len(summary.IncludedSourceIDs) != 0 {
		t.Fatalf("compile summary IncludedSourceIDs = %#v, want empty", summary.IncludedSourceIDs)
	}
	if !containsString(summary.FailedSourceIDs, secondSources[0].ID) {
		t.Fatalf("compile summary FailedSourceIDs = %#v, want to contain %q", summary.FailedSourceIDs, secondSources[0].ID)
	}
}

func TestWorkspaceWikiScanFailureInvalidatesCompileSummary(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.appDB.Close()
		_ = store.ocrDB.Close()
	})

	ctx := context.Background()
	workspace, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{ID: "workspace-a", Name: "Workspace A"})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	files := newWorkspaceKnowledgeFiles(paths, workspace.ID)
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}
	if err := files.WriteCompileSummary(WorkspaceKnowledgeCompileSummary{WorkspaceID: workspace.ID, IncludedSourceIDs: []string{"source:a"}}); err != nil {
		t.Fatalf("WriteCompileSummary() error = %v", err)
	}

	runner := &workspaceWikiScanRunner{paths: paths, store: store}
	if err := runner.writeScanRunFailure(ctx, WorkspaceWikiScanJob{WorkspaceID: workspace.ID, JobID: "job-a"}, fmt.Errorf("boom")); err != nil {
		t.Fatalf("writeScanRunFailure() error = %v", err)
	}

	compileSummaryPath, err := files.CompileSummaryPath()
	if err != nil {
		t.Fatalf("CompileSummaryPath() error = %v", err)
	}
	if _, err := os.Stat(compileSummaryPath); !os.IsNotExist(err) {
		t.Fatalf("Stat(compileSummaryPath) after failure error = %v, want not exist", err)
	}
}

func TestReadCompileSummaryMissingMarksDirty(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}

	summary, err := files.ReadCompileSummary()
	if err != nil {
		t.Fatalf("ReadCompileSummary() error = %v", err)
	}
	if !summary.CompileDirty {
		t.Fatalf("summary.CompileDirty = %v, want true", summary.CompileDirty)
	}
	if !summary.WikiDirty {
		t.Fatalf("summary.WikiDirty = %v, want true", summary.WikiDirty)
	}
	if len(summary.IncludedSourceIDs) != 0 {
		t.Fatalf("summary.IncludedSourceIDs = %#v, want empty", summary.IncludedSourceIDs)
	}
	if len(summary.FailedSourceIDs) != 0 {
		t.Fatalf("summary.FailedSourceIDs = %#v, want empty", summary.FailedSourceIDs)
	}
	if len(summary.UpdatedWikiPaths) != 0 {
		t.Fatalf("summary.UpdatedWikiPaths = %#v, want empty", summary.UpdatedWikiPaths)
	}
}

func TestBuildWorkspaceKnowledgeCompileSummaryIncludesDeletedWikiPaths(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}

	deletedDocPath, err := files.DocumentWikiPath("old-doc")
	if err != nil {
		t.Fatalf("DocumentWikiPath() error = %v", err)
	}
	deletedConceptPath, err := files.ConceptWikiPath("old-concept")
	if err != nil {
		t.Fatalf("ConceptWikiPath() error = %v", err)
	}
	for _, path := range []string{deletedDocPath, deletedConceptPath} {
		if err := writeWorkspaceKnowledgeMarkdown(path, "stale"); err != nil {
			t.Fatalf("writeWorkspaceKnowledgeMarkdown(%q) error = %v", path, err)
		}
	}

	previousWikiPaths, err := workspaceKnowledgeCurrentWikiPaths(files)
	if err != nil {
		t.Fatalf("workspaceKnowledgeCurrentWikiPaths() error = %v", err)
	}

	summary, err := buildWorkspaceKnowledgeCompileSummary(files, "workspace-a", "started", WorkspaceKnowledgeSnapshot{}, nil, previousWikiPaths)
	if err != nil {
		t.Fatalf("buildWorkspaceKnowledgeCompileSummary() error = %v", err)
	}
	if !containsString(summary.UpdatedWikiPaths, deletedDocPath) {
		t.Fatalf("summary.UpdatedWikiPaths = %#v, want to contain deleted doc path %q", summary.UpdatedWikiPaths, deletedDocPath)
	}
	if !containsString(summary.UpdatedWikiPaths, deletedConceptPath) {
		t.Fatalf("summary.UpdatedWikiPaths = %#v, want to contain deleted concept path %q", summary.UpdatedWikiPaths, deletedConceptPath)
	}
	overviewPath, err := files.OverviewPath()
	if err != nil {
		t.Fatalf("OverviewPath() error = %v", err)
	}
	if !containsString(summary.UpdatedWikiPaths, overviewPath) {
		t.Fatalf("summary.UpdatedWikiPaths = %#v, want to contain current overview path %q", summary.UpdatedWikiPaths, overviewPath)
	}
}

func TestWorkspaceWikiScanInterruptedRerunPreservesUntouchedSourceState(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.appDB.Close()
		_ = store.ocrDB.Close()
	})

	ctx := context.Background()
	workspace, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{ID: "workspace-a", Name: "Workspace A"})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	seedDir := t.TempDir()
	firstSeedPath := filepath.Join(seedDir, "a-first.md")
	secondSeedPath := filepath.Join(seedDir, "z-second.md")
	if err := os.WriteFile(firstSeedPath, []byte("# First\n\nready once."), 0o600); err != nil {
		t.Fatalf("WriteFile(firstSeedPath) error = %v", err)
	}
	if err := os.WriteFile(secondSeedPath, []byte("# Second\n\nready once."), 0o600); err != nil {
		t.Fatalf("WriteFile(secondSeedPath) error = %v", err)
	}

	importResult, err := store.ImportFiles(ctx, paths, ImportFilesInput{
		WorkspaceID: workspace.ID,
		FilePaths:   []string{firstSeedPath, secondSeedPath},
		SourceType:  "manual",
	})
	if err != nil {
		t.Fatalf("ImportFiles() error = %v", err)
	}
	if len(importResult.Documents) != 2 {
		t.Fatalf("ImportFiles() documents = %d, want 2", len(importResult.Documents))
	}

	runner := &workspaceWikiScanRunner{
		paths: paths,
		store: store,
		pdf:   panicWorkspaceKnowledgeExtractor{},
		knowledgeLLM: stubWorkspaceKnowledgeLLM{
			payload: WorkspaceKnowledgeBySourcePayload{},
		},
	}
	firstRunJob := newSavedWorkspaceWikiScanJob(t, store, workspace.ID)
	runner.run(ctx, firstRunJob)

	files := newWorkspaceKnowledgeFiles(paths, workspace.ID)
	sourcesAfterFirstRun, err := files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources() after first run error = %v", err)
	}
	if len(sourcesAfterFirstRun) != 2 {
		t.Fatalf("ReadSources() after first run len = %d, want 2", len(sourcesAfterFirstRun))
	}
	secondSource := sourcesAfterFirstRun[1]
	if secondSource.MarkItDownStatus != "ready" {
		t.Fatalf("second source MarkItDownStatus after first run = %q, want ready", secondSource.MarkItDownStatus)
	}
	if secondSource.ExtractStatus != "ready" {
		t.Fatalf("second source ExtractStatus after first run = %q, want ready", secondSource.ExtractStatus)
	}
	if secondSource.LastSuccessAt == "" {
		t.Fatalf("second source LastSuccessAt after first run = %q, want non-empty", secondSource.LastSuccessAt)
	}

	if err := os.WriteFile(importResult.Documents[0].PrimaryPDFPath, []byte("# First\n\nchanged before interrupted rerun."), 0o600); err != nil {
		t.Fatalf("WriteFile(first imported source) error = %v", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	runner.knowledgeLLM = stubWorkspaceKnowledgeLLM{
		bySourceErr: fmt.Errorf("stop after first failure"),
		onGenerateBySource: func() {
			cancel()
		},
	}
	secondRunJob := newSavedWorkspaceWikiScanJob(t, store, workspace.ID)
	runner.run(runCtx, secondRunJob)

	sourcesAfterInterruptedRun, err := files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources() after interrupted rerun error = %v", err)
	}
	if len(sourcesAfterInterruptedRun) != 2 {
		t.Fatalf("ReadSources() after interrupted rerun len = %d, want 2", len(sourcesAfterInterruptedRun))
	}
	untouchedSource := sourcesAfterInterruptedRun[1]
	if untouchedSource.MarkItDownStatus != "ready" {
		t.Fatalf("untouched source MarkItDownStatus after interrupted rerun = %q, want ready", untouchedSource.MarkItDownStatus)
	}
	if untouchedSource.ExtractStatus != "ready" {
		t.Fatalf("untouched source ExtractStatus after interrupted rerun = %q, want ready", untouchedSource.ExtractStatus)
	}
	if untouchedSource.LastError != "" {
		t.Fatalf("untouched source LastError after interrupted rerun = %q, want empty", untouchedSource.LastError)
	}
	if untouchedSource.LastSuccessAt != secondSource.LastSuccessAt {
		t.Fatalf("untouched source LastSuccessAt after interrupted rerun = %q, want preserved %q", untouchedSource.LastSuccessAt, secondSource.LastSuccessAt)
	}
}

func newSavedWorkspaceWikiScanJob(t *testing.T, store *configStore, workspaceID string) WorkspaceWikiScanJob {
	t.Helper()

	job, err := store.SaveWorkspaceWikiScanJob(context.Background(), WorkspaceWikiScanJob{
		WorkspaceID:  workspaceID,
		Status:       WorkspaceWikiScanJobQueued,
		CurrentStage: "queued",
		Message:      "queued",
		StartedAt:    nowRFC3339(),
		UpdatedAt:    nowRFC3339(),
	})
	if err != nil {
		t.Fatalf("SaveWorkspaceWikiScanJob() error = %v", err)
	}
	return job
}

type panicWorkspaceKnowledgeExtractor struct{}

func (panicWorkspaceKnowledgeExtractor) ExtractMarkdown(ctx context.Context, rawPath string) (PDFMarkdownPayload, error) {
	panic("unexpected ExtractMarkdown call")
}

type stubWorkspaceKnowledgeLLM struct {
	payload            WorkspaceKnowledgeBySourcePayload
	bySourceErr        error
	onGenerateBySource func()
}

func (s stubWorkspaceKnowledgeLLM) GenerateWorkspaceKnowledgeBySource(ctx context.Context, providerID, modelID int64, prompt string) (WorkspaceKnowledgeBySourcePayload, error) {
	if s.onGenerateBySource != nil {
		s.onGenerateBySource()
	}
	if s.bySourceErr != nil {
		return WorkspaceKnowledgeBySourcePayload{}, s.bySourceErr
	}
	return s.payload, nil
}

func (s stubWorkspaceKnowledgeLLM) GenerateWorkspaceKnowledgeMarkdown(ctx context.Context, providerID, modelID int64, prompt string) (string, error) {
	return "", nil
}

func waitForWorkspaceWikiJobTerminal(t *testing.T, store *configStore, jobID string) WorkspaceWikiScanJob {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, err := store.GetWorkspaceWikiScanJob(context.Background(), jobID)
		if err != nil {
			t.Fatalf("GetWorkspaceWikiScanJob() error = %v", err)
		}
		if job.Status == WorkspaceWikiScanJobCompleted || job.Status == WorkspaceWikiScanJobFailed || job.Status == WorkspaceWikiScanJobCancelled {
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("workspace wiki job %q did not reach terminal state", jobID)
	return WorkspaceWikiScanJob{}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func newTestAppPaths(t *testing.T) appPaths {
	t.Helper()

	rootDir := t.TempDir()
	paths := appPaths{
		RootDir:                  rootDir,
		AppConfigDBPath:          filepath.Join(rootDir, "app_config.sqlite"),
		OCRCacheDBPath:           filepath.Join(rootDir, "ocr_cache.sqlite"),
		EncryptionKeyPath:        filepath.Join(rootDir, "config.key"),
		TranslateRootDir:         filepath.Join(rootDir, "reader_translate"),
		TranslateJobsDir:         filepath.Join(rootDir, "reader_translate", "jobs"),
		WikiRootDir:              filepath.Join(rootDir, "workspace_wiki"),
		WikiJobsDir:              filepath.Join(rootDir, "workspace_wiki", "jobs"),
		TranslateRuntimeRootDir:  filepath.Join(rootDir, "reader_translate", "runtime"),
		TranslateRuntimeCacheDir: filepath.Join(rootDir, "reader_translate", "runtime-cache"),
		LibraryRootDir:           filepath.Join(rootDir, "library"),
		WorkspacesRootDir:        filepath.Join(rootDir, "library", "workspaces"),
	}

	for _, directory := range []string{
		paths.RootDir,
		paths.TranslateJobsDir,
		paths.WikiJobsDir,
		paths.TranslateRuntimeRootDir,
		paths.TranslateRuntimeCacheDir,
		paths.LibraryRootDir,
		paths.WorkspacesRootDir,
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", directory, err)
		}
	}

	return paths
}
