package main

import (
	"os"
	"path/filepath"
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

func TestCompileWorkspaceKnowledgeFallsBackToLegacySchemaBySourceLayout(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}

	workspaceRoot, err := files.workspaceRootDir()
	if err != nil {
		t.Fatalf("workspaceRootDir() error = %v", err)
	}
	legacyBySourcePath := filepath.Join(workspaceRoot, "schema", "by-source", "paper-a.json")
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
	if err := writeWorkspaceKnowledgeJSON(legacyBySourcePath, payload); err != nil {
		t.Fatalf("writeWorkspaceKnowledgeJSON(legacyBySourcePath) error = %v", err)
	}

	snapshot, err := CompileWorkspaceKnowledge(files, "Attention Workspace")
	if err != nil {
		t.Fatalf("CompileWorkspaceKnowledge() error = %v", err)
	}
	if len(snapshot.Sources) != 1 {
		t.Fatalf("snapshot.Sources len = %d, want 1", len(snapshot.Sources))
	}
	if len(snapshot.Entities) != 1 {
		t.Fatalf("snapshot.Entities len = %d, want 1", len(snapshot.Entities))
	}
}
