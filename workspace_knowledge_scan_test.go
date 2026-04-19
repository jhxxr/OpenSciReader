package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceWikiScanRunnerWritesKnowledgeArtifacts(t *testing.T) {
	t.Parallel()

	fixture := newWorkspaceKnowledgeScanFixture(t)

	runner := &workspaceWikiScanRunner{
		paths: fixture.paths,
		store: fixture.store,
		pdf: &stubWorkspaceKnowledgeExtractor{
			markdown: "# Paper A\n\nThis paper introduces Contrastive Memory.",
		},
		knowledgeLLM: &stubWorkspaceKnowledgeLLM{
			bySource: WorkspaceKnowledgeBySourcePayload{
				Entities: []WorkspaceKnowledgeEntity{{
					ID:          "entity:method:contrastive-memory",
					WorkspaceID: fixture.workspace.ID,
					Title:       "Contrastive Memory",
					Type:        "method",
					Summary:     "A memory-augmented retrieval method",
					Origin:      "scan",
					Status:      "confirmed",
					Confidence:  0.9,
					CreatedAt:   nowRFC3339(),
					UpdatedAt:   nowRFC3339(),
				}},
			},
		},
	}

	job := saveWorkspaceWikiScanJobForTest(t, fixture.store, fixture.workspace.ID, "", 1, 2)
	runner.run(context.Background(), job)

	extractPath, err := fixture.files.ExtractPath("paper")
	if err != nil {
		t.Fatalf("ExtractPath error: %v", err)
	}
	extractBytes, err := os.ReadFile(extractPath)
	if err != nil {
		t.Fatalf("read extract error: %v", err)
	}
	if got := string(extractBytes); !strings.Contains(got, "Contrastive Memory") {
		t.Fatalf("extract markdown = %q, want source text", got)
	}

	bySourcePath, err := fixture.files.BySourcePath("paper")
	if err != nil {
		t.Fatalf("BySourcePath error: %v", err)
	}
	if _, err := os.Stat(bySourcePath); err != nil {
		t.Fatalf("stat by-source error: %v", err)
	}
	bySourcePayload, err := fixture.files.ReadBySource("paper")
	if err != nil {
		t.Fatalf("ReadBySource error: %v", err)
	}
	if len(bySourcePayload.Entities) != 1 || bySourcePayload.Entities[0].Title != "Contrastive Memory" {
		t.Fatalf("by-source payload = %#v, want one entity", bySourcePayload)
	}

	sources, err := fixture.files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("sources len = %d, want 1", len(sources))
	}
	if sources[0].Status != "ready" {
		t.Fatalf("source status = %q, want ready", sources[0].Status)
	}
	if sources[0].DocumentID != fixture.document.ID {
		t.Fatalf("source document id = %q, want %q", sources[0].DocumentID, fixture.document.ID)
	}
	if sources[0].ExtractPath != extractPath {
		t.Fatalf("source extract path = %q, want %q", sources[0].ExtractPath, extractPath)
	}
	if strings.TrimSpace(sources[0].ContentHash) == "" {
		t.Fatal("source content hash should not be empty")
	}

	overviewPath, err := fixture.files.OverviewPath()
	if err != nil {
		t.Fatalf("OverviewPath error: %v", err)
	}
	if _, err := os.Stat(overviewPath); err != nil {
		t.Fatalf("stat overview error: %v", err)
	}

	runRecord := readWorkspaceKnowledgeScanRunRecord(t, fixture.files, job.ID)
	if runRecord.Status != string(WorkspaceWikiScanJobCompleted) {
		t.Fatalf("scan run status = %q, want %q", runRecord.Status, WorkspaceWikiScanJobCompleted)
	}
	if len(runRecord.SourceIDs) != 1 || runRecord.SourceIDs[0] != "source:paper" {
		t.Fatalf("scan run source ids = %#v, want source:paper", runRecord.SourceIDs)
	}
	if strings.TrimSpace(runRecord.Message) == "" {
		t.Fatal("scan run message should not be empty")
	}

	jobRecord := readWorkspaceWikiScanJobForTest(t, fixture.store, job.ID)
	if jobRecord.Status != WorkspaceWikiScanJobCompleted {
		t.Fatalf("job status = %q, want %q", jobRecord.Status, WorkspaceWikiScanJobCompleted)
	}

	entitiesPath, err := fixture.files.EntitiesPath()
	if err != nil {
		t.Fatalf("EntitiesPath error: %v", err)
	}
	assertWorkspaceKnowledgeJSONCount(t, entitiesPath, 1)
}

func TestWorkspaceWikiScanRunnerPreservesPreviousBySourceOnTransientSourceFailure(t *testing.T) {
	t.Parallel()

	fixture := newWorkspaceKnowledgeScanFixture(t)

	firstRunner := &workspaceWikiScanRunner{
		paths: fixture.paths,
		store: fixture.store,
		pdf: &stubWorkspaceKnowledgeExtractor{
			markdown: "# Paper A\n\nInitial extract.",
		},
		knowledgeLLM: &stubWorkspaceKnowledgeLLM{
			bySource: WorkspaceKnowledgeBySourcePayload{
				Entities: []WorkspaceKnowledgeEntity{{
					ID:          "entity:method:contrastive-memory",
					WorkspaceID: fixture.workspace.ID,
					Title:       "Contrastive Memory",
					Type:        "method",
					Summary:     "Initial knowledge",
					Origin:      "scan",
					Status:      "confirmed",
					Confidence:  0.9,
					CreatedAt:   nowRFC3339(),
					UpdatedAt:   nowRFC3339(),
				}},
			},
		},
	}
	firstJob := saveWorkspaceWikiScanJobForTest(t, fixture.store, fixture.workspace.ID, "", 1, 2)
	firstRunner.run(context.Background(), firstJob)

	bySourcePath, err := fixture.files.BySourcePath("paper")
	if err != nil {
		t.Fatalf("BySourcePath error: %v", err)
	}
	previousBySource, err := os.ReadFile(bySourcePath)
	if err != nil {
		t.Fatalf("read previous by-source error: %v", err)
	}

	if err := os.WriteFile(fixture.document.PrimaryPDFPath, []byte("%PDF-1.4 changed"), 0o600); err != nil {
		t.Fatalf("rewrite imported pdf error: %v", err)
	}

	secondRunner := &workspaceWikiScanRunner{
		paths: fixture.paths,
		store: fixture.store,
		pdf: &stubWorkspaceKnowledgeExtractor{
			err: errors.New("temporary extractor outage"),
		},
		knowledgeLLM: &stubWorkspaceKnowledgeLLM{},
	}
	secondJob := saveWorkspaceWikiScanJobForTest(t, fixture.store, fixture.workspace.ID, "", 1, 2)
	secondRunner.run(context.Background(), secondJob)

	preservedBySource, err := os.ReadFile(bySourcePath)
	if err != nil {
		t.Fatalf("read preserved by-source error: %v", err)
	}
	if !bytes.Equal(preservedBySource, previousBySource) {
		t.Fatal("by-source artifact changed after transient failure")
	}

	sources, err := fixture.files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("sources len = %d, want 1", len(sources))
	}
	if sources[0].Status != "error" {
		t.Fatalf("source status = %q, want error", sources[0].Status)
	}
	if !strings.Contains(sources[0].LastError, "temporary extractor outage") {
		t.Fatalf("source last error = %q, want extractor outage", sources[0].LastError)
	}

	runRecord := readWorkspaceKnowledgeScanRunRecord(t, fixture.files, secondJob.ID)
	if runRecord.Status != string(WorkspaceWikiScanJobCompleted) {
		t.Fatalf("scan run status = %q, want completed", runRecord.Status)
	}
	if !strings.Contains(runRecord.Message, "failed") {
		t.Fatalf("scan run message = %q, want failed count", runRecord.Message)
	}

	entitiesPath, err := fixture.files.EntitiesPath()
	if err != nil {
		t.Fatalf("EntitiesPath error: %v", err)
	}
	assertWorkspaceKnowledgeJSONCount(t, entitiesPath, 1)
}

func TestWorkspaceWikiScanRunnerSkipsUnchangedSources(t *testing.T) {
	t.Parallel()

	fixture := newWorkspaceKnowledgeScanFixture(t)

	firstRunner := &workspaceWikiScanRunner{
		paths: fixture.paths,
		store: fixture.store,
		pdf: &stubWorkspaceKnowledgeExtractor{
			markdown: "# Paper A\n\nInitial extract.",
		},
		knowledgeLLM: &stubWorkspaceKnowledgeLLM{
			bySource: WorkspaceKnowledgeBySourcePayload{
				Entities: []WorkspaceKnowledgeEntity{{
					ID:          "entity:method:contrastive-memory",
					WorkspaceID: fixture.workspace.ID,
					Title:       "Contrastive Memory",
					Type:        "method",
					Summary:     "Initial knowledge",
					Origin:      "scan",
					Status:      "confirmed",
					Confidence:  0.9,
					CreatedAt:   nowRFC3339(),
					UpdatedAt:   nowRFC3339(),
				}},
			},
		},
	}
	firstJob := saveWorkspaceWikiScanJobForTest(t, fixture.store, fixture.workspace.ID, "", 1, 2)
	firstRunner.run(context.Background(), firstJob)

	originalSources, err := fixture.files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources first run error: %v", err)
	}
	if len(originalSources) != 1 {
		t.Fatalf("first run sources len = %d, want 1", len(originalSources))
	}

	bySourcePath, err := fixture.files.BySourcePath("paper")
	if err != nil {
		t.Fatalf("BySourcePath error: %v", err)
	}
	originalBySource, err := os.ReadFile(bySourcePath)
	if err != nil {
		t.Fatalf("read first by-source error: %v", err)
	}

	secondExtractor := &stubWorkspaceKnowledgeExtractor{
		markdown: "# Paper A\n\nShould not be used.",
	}
	secondLLM := &stubWorkspaceKnowledgeLLM{
		bySource: WorkspaceKnowledgeBySourcePayload{
			Entities: []WorkspaceKnowledgeEntity{{
				ID:          "entity:method:should-not-change",
				WorkspaceID: fixture.workspace.ID,
				Title:       "Should Not Change",
				Type:        "method",
				Summary:     "Should not be written",
				Origin:      "scan",
				Status:      "confirmed",
				Confidence:  0.9,
				CreatedAt:   nowRFC3339(),
				UpdatedAt:   nowRFC3339(),
			}},
		},
	}
	secondRunner := &workspaceWikiScanRunner{
		paths:        fixture.paths,
		store:        fixture.store,
		pdf:          secondExtractor,
		knowledgeLLM: secondLLM,
	}
	secondJob := saveWorkspaceWikiScanJobForTest(t, fixture.store, fixture.workspace.ID, "", 1, 2)
	secondRunner.run(context.Background(), secondJob)

	if secondExtractor.callCount != 0 {
		t.Fatalf("extractor call count = %d, want 0", secondExtractor.callCount)
	}
	if secondLLM.bySourceCallCount != 0 {
		t.Fatalf("by-source llm call count = %d, want 0", secondLLM.bySourceCallCount)
	}

	skippedSources, err := fixture.files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources second run error: %v", err)
	}
	if len(skippedSources) != 1 {
		t.Fatalf("second run sources len = %d, want 1", len(skippedSources))
	}
	if skippedSources[0].LastScanAt != originalSources[0].LastScanAt {
		t.Fatalf("last scan at = %q, want preserved %q", skippedSources[0].LastScanAt, originalSources[0].LastScanAt)
	}
	if skippedSources[0].Status != "ready" {
		t.Fatalf("source status = %q, want ready", skippedSources[0].Status)
	}

	skippedBySource, err := os.ReadFile(bySourcePath)
	if err != nil {
		t.Fatalf("read skipped by-source error: %v", err)
	}
	if !bytes.Equal(skippedBySource, originalBySource) {
		t.Fatal("by-source artifact changed for unchanged source")
	}

	runRecord := readWorkspaceKnowledgeScanRunRecord(t, fixture.files, secondJob.ID)
	if !strings.Contains(runRecord.Message, "skipped") {
		t.Fatalf("scan run message = %q, want skipped count", runRecord.Message)
	}
}

func TestWorkspaceWikiScanRunnerFailsTargetedScanWhenDocumentResolvesToZeroSources(t *testing.T) {
	t.Parallel()

	fixture := newWorkspaceKnowledgeScanFixture(t)

	extractor := &stubWorkspaceKnowledgeExtractor{
		markdown: "# Paper A\n\nShould not run.",
	}
	llm := &stubWorkspaceKnowledgeLLM{}
	runner := &workspaceWikiScanRunner{
		paths:        fixture.paths,
		store:        fixture.store,
		pdf:          extractor,
		knowledgeLLM: llm,
	}

	job := saveWorkspaceWikiScanJobForTest(t, fixture.store, fixture.workspace.ID, "doc-does-not-exist", 1, 2)
	runner.run(context.Background(), job)

	if extractor.callCount != 0 {
		t.Fatalf("extractor call count = %d, want 0", extractor.callCount)
	}
	if llm.bySourceCallCount != 0 {
		t.Fatalf("by-source llm call count = %d, want 0", llm.bySourceCallCount)
	}

	runRecord := readWorkspaceKnowledgeScanRunRecord(t, fixture.files, job.ID)
	if runRecord.Status != string(WorkspaceWikiScanJobFailed) {
		t.Fatalf("scan run status = %q, want %q", runRecord.Status, WorkspaceWikiScanJobFailed)
	}
	if !strings.Contains(runRecord.Message, "no scan sources") {
		t.Fatalf("scan run message = %q, want no scan sources error", runRecord.Message)
	}

	jobRecord := readWorkspaceWikiScanJobForTest(t, fixture.store, job.ID)
	if jobRecord.Status != WorkspaceWikiScanJobFailed {
		t.Fatalf("job status = %q, want %q", jobRecord.Status, WorkspaceWikiScanJobFailed)
	}
	if !strings.Contains(jobRecord.Message, "no scan sources") {
		t.Fatalf("job message = %q, want no scan sources error", jobRecord.Message)
	}
}

func TestWorkspaceWikiScanRunnerTrimsLargeMarkdownBeforeDistillation(t *testing.T) {
	t.Parallel()

	fixture := newWorkspaceKnowledgeScanFixture(t)
	provider, model := saveWorkspaceKnowledgeLLMModelForTest(t, fixture.store, 2048)

	largeMarkdown := "# Paper A\n\n" +
		strings.Repeat("A", 15000) +
		"\n\nMIDDLE-MARKER\n\n" +
		strings.Repeat("B", 15000) +
		"\n\nEND-MARKER"

	llm := &stubWorkspaceKnowledgeLLM{
		bySource: WorkspaceKnowledgeBySourcePayload{
			Entities: []WorkspaceKnowledgeEntity{{
				ID:          "entity:method:contrastive-memory",
				WorkspaceID: fixture.workspace.ID,
				Title:       "Contrastive Memory",
				Type:        "method",
				Summary:     "Trimmed knowledge",
				Origin:      "scan",
				Status:      "confirmed",
				Confidence:  0.9,
				CreatedAt:   nowRFC3339(),
				UpdatedAt:   nowRFC3339(),
			}},
		},
	}
	runner := &workspaceWikiScanRunner{
		paths: fixture.paths,
		store: fixture.store,
		pdf: &stubWorkspaceKnowledgeExtractor{
			markdown: largeMarkdown,
		},
		knowledgeLLM: llm,
	}

	job := saveWorkspaceWikiScanJobForTest(t, fixture.store, fixture.workspace.ID, "", provider.ID, model.ID)
	runner.run(context.Background(), job)

	if llm.bySourceCallCount != 1 {
		t.Fatalf("by-source llm call count = %d, want 1", llm.bySourceCallCount)
	}

	sources, err := fixture.files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("sources len = %d, want 1", len(sources))
	}

	fullPrompt := buildWorkspaceKnowledgeBySourcePrompt(fixture.workspace, sources[0], largeMarkdown)
	if len(llm.lastBySourcePrompt) >= len(fullPrompt) {
		t.Fatalf("trimmed prompt len = %d, want less than full prompt len %d", len(llm.lastBySourcePrompt), len(fullPrompt))
	}
	if !strings.Contains(llm.lastBySourcePrompt, "trimmed") {
		t.Fatalf("prompt = %q, want trim marker", llm.lastBySourcePrompt)
	}
	if strings.Contains(llm.lastBySourcePrompt, "MIDDLE-MARKER") {
		t.Fatalf("prompt should omit middle marker, got %q", llm.lastBySourcePrompt)
	}
	if !strings.Contains(llm.lastBySourcePrompt, "END-MARKER") {
		t.Fatalf("prompt should retain end marker, got %q", llm.lastBySourcePrompt)
	}
}

type workspaceKnowledgeScanFixture struct {
	paths     appPaths
	store     *configStore
	workspace Workspace
	document  DocumentRecord
	files     workspaceKnowledgeFiles
}

func newWorkspaceKnowledgeScanFixture(t *testing.T) workspaceKnowledgeScanFixture {
	t.Helper()

	tempDir := t.TempDir()
	paths := appPaths{
		RootDir:           tempDir,
		AppConfigDBPath:   filepath.Join(tempDir, "app.sqlite"),
		OCRCacheDBPath:    filepath.Join(tempDir, "ocr.sqlite"),
		EncryptionKeyPath: filepath.Join(tempDir, "config.key"),
		LibraryRootDir:    filepath.Join(tempDir, "library"),
		WorkspacesRootDir: filepath.Join(tempDir, "library", "workspaces"),
	}

	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store error: %v", err)
		}
	})

	workspace, err := store.CreateWorkspace(t.Context(), WorkspaceUpsertInput{Name: "Scan Workspace", Description: "", Color: ""})
	if err != nil {
		t.Fatalf("CreateWorkspace error: %v", err)
	}

	pdfPath := filepath.Join(tempDir, "paper.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4 test"), 0o600); err != nil {
		t.Fatalf("write pdf error: %v", err)
	}
	importResult, err := store.ImportFiles(t.Context(), paths, ImportFilesInput{
		WorkspaceID: workspace.ID,
		FilePaths:   []string{pdfPath},
		SourceType:  "manual",
	})
	if err != nil {
		t.Fatalf("ImportFiles error: %v", err)
	}
	if len(importResult.Documents) != 1 {
		t.Fatalf("imported documents len = %d, want 1", len(importResult.Documents))
	}

	return workspaceKnowledgeScanFixture{
		paths:     paths,
		store:     store,
		workspace: workspace,
		document:  importResult.Documents[0],
		files:     newWorkspaceKnowledgeFiles(paths, workspace.ID),
	}
}

func saveWorkspaceWikiScanJobForTest(t *testing.T, store *configStore, workspaceID, documentID string, providerID, modelID int64) WorkspaceWikiScanJob {
	t.Helper()

	job, err := store.SaveWorkspaceWikiScanJob(t.Context(), WorkspaceWikiScanJob{
		WorkspaceID:  workspaceID,
		DocumentID:   documentID,
		Status:       WorkspaceWikiScanJobQueued,
		CurrentStage: "queued",
		Message:      "queued",
		ProviderID:   providerID,
		ModelID:      modelID,
		StartedAt:    nowRFC3339(),
		UpdatedAt:    nowRFC3339(),
	})
	if err != nil {
		t.Fatalf("SaveWorkspaceWikiScanJob error: %v", err)
	}
	return job
}

func readWorkspaceKnowledgeScanRunRecord(t *testing.T, files workspaceKnowledgeFiles, jobID int64) WorkspaceKnowledgeScanRunRecord {
	t.Helper()

	scanRunPath, err := files.ScanRunPath(fmt.Sprintf("%d", jobID))
	if err != nil {
		t.Fatalf("ScanRunPath error: %v", err)
	}
	var record WorkspaceKnowledgeScanRunRecord
	if err := readWorkspaceKnowledgeJSON(scanRunPath, &record); err != nil {
		t.Fatalf("read scan run record error: %v", err)
	}
	return record
}

func readWorkspaceWikiScanJobForTest(t *testing.T, store *configStore, jobID int64) WorkspaceWikiScanJob {
	t.Helper()

	row := store.appDB.QueryRowContext(t.Context(), `
		SELECT id, workspace_id, document_id, status, current_stage, message, provider_id, model_id, started_at, finished_at, updated_at
		FROM workspace_wiki_scan_jobs
		WHERE id = ?;
	`, jobID)

	var job WorkspaceWikiScanJob
	var status string
	if err := row.Scan(&job.ID, &job.WorkspaceID, &job.DocumentID, &status, &job.CurrentStage, &job.Message, &job.ProviderID, &job.ModelID, &job.StartedAt, &job.FinishedAt, &job.UpdatedAt); err != nil {
		t.Fatalf("read workspace wiki scan job error: %v", err)
	}
	job.Status = WorkspaceWikiScanJobStatus(status)
	return job
}

func saveWorkspaceKnowledgeLLMModelForTest(t *testing.T, store *configStore, contextWindow int) (ProviderRecord, ModelRecord) {
	t.Helper()

	provider, err := store.SaveProvider(t.Context(), ProviderUpsertInput{
		Name:     "Scan Test Provider",
		Type:     ProviderTypeLLM,
		BaseURL:  "http://localhost",
		APIKey:   "sk-test",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("SaveProvider error: %v", err)
	}
	model, err := store.SaveModel(t.Context(), ModelUpsertInput{
		ProviderID:    provider.ID,
		ModelID:       "scan-test-model",
		ContextWindow: contextWindow,
	})
	if err != nil {
		t.Fatalf("SaveModel error: %v", err)
	}
	return provider, model
}

type stubWorkspaceKnowledgeExtractor struct {
	markdown  string
	err       error
	callCount int
}

var _ workspaceKnowledgeExtractor = (*stubWorkspaceKnowledgeExtractor)(nil)

func (s *stubWorkspaceKnowledgeExtractor) ExtractMarkdown(_ context.Context, _ string) (PDFMarkdownPayload, error) {
	s.callCount++
	if s.err != nil {
		return PDFMarkdownPayload{}, s.err
	}
	return PDFMarkdownPayload{
		Markdown:    s.markdown,
		TotalChars:  len(s.markdown),
		GeneratedAt: nowRFC3339(),
	}, nil
}

type stubWorkspaceKnowledgeLLM struct {
	bySource           WorkspaceKnowledgeBySourcePayload
	docWiki            string
	overviewWiki       string
	bySourceErr        error
	markdownErr        error
	bySourceCallCount  int
	markdownCallCount  int
	lastBySourcePrompt string
}

var _ workspaceKnowledgeLLM = (*stubWorkspaceKnowledgeLLM)(nil)

func (s *stubWorkspaceKnowledgeLLM) GenerateWorkspaceKnowledgeBySource(_ context.Context, _ int64, _ int64, prompt string) (WorkspaceKnowledgeBySourcePayload, error) {
	s.bySourceCallCount++
	s.lastBySourcePrompt = prompt
	if s.bySourceErr != nil {
		return WorkspaceKnowledgeBySourcePayload{}, s.bySourceErr
	}
	return s.bySource, nil
}

func (s *stubWorkspaceKnowledgeLLM) GenerateWorkspaceKnowledgeMarkdown(_ context.Context, _ int64, _ int64, _ string) (string, error) {
	if s.markdownErr != nil {
		return "", s.markdownErr
	}
	s.markdownCallCount++
	if s.markdownCallCount == 1 {
		return s.docWiki, nil
	}
	return s.overviewWiki, nil
}
