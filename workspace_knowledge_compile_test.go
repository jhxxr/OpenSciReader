package main

import (
	"os"
	"strings"
	"testing"
)

func TestCompileWorkspaceKnowledgeWritesStateAndWikiIndex(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}

	payload := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{
			ID:          "source:paper-a",
			WorkspaceID: "workspace-a",
			Title:       "Paper A",
			Slug:        "paper-a",
			Kind:        "pdf",
		},
		Entities: []WorkspaceKnowledgeEntity{{
			ID:          "entity:attention",
			WorkspaceID: "workspace-a",
			Title:       "Attention",
			Summary:     "Attention is the focus mechanism discussed in Paper A.",
		}},
	}
	if err := files.WriteBySource("paper-a", payload); err != nil {
		t.Fatalf("WriteBySource() error = %v", err)
	}

	if _, err := CompileWorkspaceKnowledge(files, "Attention Workspace"); err != nil {
		t.Fatalf("CompileWorkspaceKnowledge() error = %v", err)
	}

	indexPath, err := files.IndexPath()
	if err != nil {
		t.Fatalf("IndexPath() error = %v", err)
	}
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("ReadFile(indexPath) error = %v", err)
	}
	if !strings.Contains(string(indexContent), "[[docs/paper-a|Paper A]]") {
		t.Fatalf("index.md = %q, want doc wikilink", string(indexContent))
	}

	logPath, err := files.LogPath()
	if err != nil {
		t.Fatalf("LogPath() error = %v", err)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("Stat(logPath) error = %v", err)
	}
}
