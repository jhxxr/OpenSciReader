package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceWikiScanRunnerWritesKnowledgeArtifacts(t *testing.T) {
	t.Parallel()

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
	defer func() { _ = store.Close() }()

	workspace, err := store.CreateWorkspace(t.Context(), WorkspaceUpsertInput{Name: "Scan Workspace", Description: "", Color: ""})
	if err != nil {
		t.Fatalf("CreateWorkspace error: %v", err)
	}

	pdfPath := filepath.Join(tempDir, "paper.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4 test"), 0o600); err != nil {
		t.Fatalf("write pdf error: %v", err)
	}
	if _, err := store.ImportFiles(t.Context(), paths, ImportFilesInput{
		WorkspaceID: workspace.ID,
		FilePaths:   []string{pdfPath},
		SourceType:  "manual",
	}); err != nil {
		t.Fatalf("ImportFiles error: %v", err)
	}

	runner := &workspaceWikiScanRunner{
		paths: paths,
		store: store,
		pdf: &stubWorkspaceKnowledgeExtractor{
			markdown: "# Paper A\n\nThis paper introduces Contrastive Memory.",
		},
		knowledgeLLM: &stubWorkspaceKnowledgeLLM{
			bySource: WorkspaceKnowledgeBySourcePayload{
				Entities: []WorkspaceKnowledgeEntity{{
					ID:          "entity:method:contrastive-memory",
					WorkspaceID: workspace.ID,
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
			docWiki:      "# Paper A\n\n## Summary\nA memory-augmented retrieval paper.",
			overviewWiki: "# Scan Workspace Overview\n\n## Themes\n- Contrastive Memory",
		},
	}

	job, err := store.SaveWorkspaceWikiScanJob(t.Context(), WorkspaceWikiScanJob{
		WorkspaceID:  workspace.ID,
		Status:       WorkspaceWikiScanJobQueued,
		CurrentStage: "queued",
		Message:      "queued",
		ProviderID:   1,
		ModelID:      2,
		StartedAt:    nowRFC3339(),
		UpdatedAt:    nowRFC3339(),
	})
	if err != nil {
		t.Fatalf("SaveWorkspaceWikiScanJob error: %v", err)
	}

	runner.run(context.Background(), job)

	files := newWorkspaceKnowledgeFiles(paths, workspace.ID)
	extractPath, err := files.ExtractPath("paper")
	if err != nil {
		t.Fatalf("ExtractPath error: %v", err)
	}
	if _, err := os.Stat(extractPath); err != nil {
		t.Fatalf("stat extract error: %v", err)
	}
	bySourcePath, err := files.BySourcePath("paper")
	if err != nil {
		t.Fatalf("BySourcePath error: %v", err)
	}
	if _, err := os.Stat(bySourcePath); err != nil {
		t.Fatalf("stat by-source error: %v", err)
	}
	overviewPath, err := files.OverviewPath()
	if err != nil {
		t.Fatalf("OverviewPath error: %v", err)
	}
	if _, err := os.Stat(overviewPath); err != nil {
		t.Fatalf("stat overview error: %v", err)
	}
}

type stubWorkspaceKnowledgeExtractor struct {
	markdown string
	err      error
}

var _ workspaceKnowledgeExtractor = (*stubWorkspaceKnowledgeExtractor)(nil)

func (s *stubWorkspaceKnowledgeExtractor) ExtractMarkdown(_ context.Context, _ string) (PDFMarkdownPayload, error) {
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
	bySource          WorkspaceKnowledgeBySourcePayload
	docWiki           string
	overviewWiki      string
	bySourceErr       error
	markdownErr       error
	markdownCallCount int
}

var _ workspaceKnowledgeLLM = (*stubWorkspaceKnowledgeLLM)(nil)

func (s *stubWorkspaceKnowledgeLLM) GenerateWorkspaceKnowledgeBySource(_ context.Context, _ int64, _ int64, _ string) (WorkspaceKnowledgeBySourcePayload, error) {
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
