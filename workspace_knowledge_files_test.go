package main

import (
	"path/filepath"
	"testing"
)

func TestWorkspaceKnowledgeFilesRoundTrip(t *testing.T) {
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

	workspace, err := store.CreateWorkspace(t.Context(), WorkspaceUpsertInput{
		Name:        "Knowledge Workspace",
		Description: "",
		Color:       "",
	})
	if err != nil {
		t.Fatalf("CreateWorkspace error: %v", err)
	}

	files := newWorkspaceKnowledgeFiles(paths, workspace.ID)
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	source := WorkspaceKnowledgeSource{
		ID:           "source:paper-a",
		WorkspaceID:  workspace.ID,
		Title:        "Paper A",
		Slug:         "paper-a",
		Kind:         "pdf",
		AbsolutePath: "C:/papers/paper-a.pdf",
		ContentHash:  "hash-a",
		ExtractPath:  files.ExtractPath("paper-a"),
		Status:       "ready",
	}

	if err := files.WriteSources([]WorkspaceKnowledgeSource{source}); err != nil {
		t.Fatalf("WriteSources error: %v", err)
	}

	gotSources, err := files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources error: %v", err)
	}
	if len(gotSources) != 1 || gotSources[0].ID != "source:paper-a" {
		t.Fatalf("got sources = %#v, want source:paper-a", gotSources)
	}

	entity := WorkspaceKnowledgeEntity{
		ID:          "entity:method:paper-a",
		WorkspaceID: workspace.ID,
		Title:       "Paper A Method",
		Type:        "method",
		Summary:     "A simple method",
		Aliases:     []string{"PAM"},
		SourceRefs: []WorkspaceKnowledgeSourceRef{{
			SourceID:  "source:paper-a",
			PageStart: 1,
			PageEnd:   1,
			Excerpt:   "Method excerpt",
		}},
		Origin:     "scan",
		Status:     "confirmed",
		Confidence: 0.8,
		CreatedAt:  nowRFC3339(),
		UpdatedAt:  nowRFC3339(),
	}

	payload := WorkspaceKnowledgeBySourcePayload{
		Source:   source,
		Entities: []WorkspaceKnowledgeEntity{entity},
	}

	if err := files.WriteBySource("paper-a", payload); err != nil {
		t.Fatalf("WriteBySource error: %v", err)
	}

	gotPayload, err := files.ReadBySource("paper-a")
	if err != nil {
		t.Fatalf("ReadBySource error: %v", err)
	}
	if len(gotPayload.Entities) != 1 || gotPayload.Entities[0].ID != entity.ID {
		t.Fatalf("got payload = %#v, want entity %q", gotPayload, entity.ID)
	}
}
