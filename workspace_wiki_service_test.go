package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type stubWorkspaceKnowledgeLLM struct {
	callCount int
	prompts   []string
}

func (s *stubWorkspaceKnowledgeLLM) GenerateWorkspaceKnowledgeBySource(_ context.Context, _ int64, _ int64, prompt string) (WorkspaceKnowledgeBySourcePayload, error) {
	s.callCount++
	s.prompts = append(s.prompts, prompt)
	return WorkspaceKnowledgeBySourcePayload{
		Entities: []WorkspaceKnowledgeEntity{
			{
				Title:   "Attention",
				Type:    "concept",
				Summary: "Attention is the main concept extracted from this note.",
			},
		},
		Claims: []WorkspaceKnowledgeClaim{
			{
				Title:   "Attention is central",
				Type:    "finding",
				Summary: "Attention is the core topic in this note.",
			},
		},
	}, nil
}

func (s *stubWorkspaceKnowledgeLLM) GenerateWorkspaceKnowledgeMarkdown(_ context.Context, _ int64, _ int64, _ string) (string, error) {
	return "", nil
}

type panicWorkspaceKnowledgeExtractor struct{}

func (panicWorkspaceKnowledgeExtractor) ExtractMarkdown(_ context.Context, rawPath string) (PDFMarkdownPayload, error) {
	return PDFMarkdownPayload{}, fmt.Errorf("unexpected pdf extraction for %s", rawPath)
}

func TestStartWorkspaceWikiScanCreatesPageRecords(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("store.Close error: %v", closeErr)
		}
	})

	workspace, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{Name: "Knowledge Workspace"})
	if err != nil {
		t.Fatalf("CreateWorkspace error: %v", err)
	}

	seedPath := filepath.Join(t.TempDir(), "attention.md")
	seedMarkdown := strings.TrimSpace(`# Attention Notes

Attention is a central mechanism in the imported document.

## Findings

- Transformers rely on attention layers.`) + "\n"
	if err := os.WriteFile(seedPath, []byte(seedMarkdown), 0o600); err != nil {
		t.Fatalf("write seed markdown error: %v", err)
	}

	importResult, err := store.ImportFiles(ctx, paths, ImportFilesInput{
		WorkspaceID: workspace.ID,
		FilePaths:   []string{seedPath},
		Title:       "Attention Notes",
	})
	if err != nil {
		t.Fatalf("ImportFiles error: %v", err)
	}
	if len(importResult.Documents) != 1 {
		t.Fatalf("ImportFiles documents = %d, want 1", len(importResult.Documents))
	}

	llm := &stubWorkspaceKnowledgeLLM{}
	service := newWorkspaceWikiService(paths, store, panicWorkspaceKnowledgeExtractor{}, llm)
	job, err := service.Start(ctx, WorkspaceWikiScanStartInput{
		WorkspaceID: workspace.ID,
		ProviderID:  1,
		ModelID:     1,
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	job = waitForWorkspaceWikiJobTerminal(t, ctx, store, job.JobID)
	if job.Status != WorkspaceWikiScanJobCompleted {
		t.Fatalf("job status = %s, want %s (message=%q error=%q)", job.Status, WorkspaceWikiScanJobCompleted, job.Message, job.Error)
	}
	if llm.callCount != 1 {
		t.Fatalf("llm call count = %d, want 1", llm.callCount)
	}

	pages, err := store.ListWorkspaceWikiPages(ctx, workspace.ID)
	if err != nil {
		t.Fatalf("ListWorkspaceWikiPages error: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("wiki page count = %d, want 2", len(pages))
	}

	var overviewPage *WorkspaceWikiPage
	var documentPage *WorkspaceWikiPage
	for index := range pages {
		switch pages[index].Kind {
		case WorkspaceWikiPageOverview:
			overviewPage = &pages[index]
		case WorkspaceWikiPageDocument:
			documentPage = &pages[index]
		}
	}
	if overviewPage == nil {
		t.Fatalf("overview page missing: %+v", pages)
	}
	if documentPage == nil {
		t.Fatalf("document page missing: %+v", pages)
	}
	if documentPage.SourceDocumentID != importResult.Documents[0].ID {
		t.Fatalf("document page sourceDocumentId = %q, want %q", documentPage.SourceDocumentID, importResult.Documents[0].ID)
	}
	if strings.TrimSpace(documentPage.Summary) == "" {
		t.Fatalf("document page summary should not be empty")
	}
	if _, err := os.Stat(overviewPage.MarkdownPath); err != nil {
		t.Fatalf("stat overview markdown error: %v", err)
	}
	if _, err := os.Stat(documentPage.MarkdownPath); err != nil {
		t.Fatalf("stat document markdown error: %v", err)
	}
}

func waitForWorkspaceWikiJobTerminal(t *testing.T, ctx context.Context, store *configStore, jobID string) WorkspaceWikiScanJob {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, err := store.GetWorkspaceWikiScanJob(ctx, jobID)
		if err != nil {
			t.Fatalf("GetWorkspaceWikiScanJob error: %v", err)
		}
		if job.Status == WorkspaceWikiScanJobCompleted || job.Status == WorkspaceWikiScanJobFailed || job.Status == WorkspaceWikiScanJobCancelled {
			return job
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("workspace wiki job %s did not reach a terminal state", jobID)
	return WorkspaceWikiScanJob{}
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

	for _, dir := range []string{
		paths.RootDir,
		paths.TranslateJobsDir,
		paths.WikiJobsDir,
		paths.TranslateRuntimeRootDir,
		paths.TranslateRuntimeCacheDir,
		paths.LibraryRootDir,
		paths.WorkspacesRootDir,
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("MkdirAll(%s) error: %v", dir, err)
		}
	}

	return paths
}
