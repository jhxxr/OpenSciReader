package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func newWorkspaceKnowledgeTestPaths(rootDir string) appPaths {
	return appPaths{
		RootDir:           rootDir,
		AppConfigDBPath:   filepath.Join(rootDir, "app.sqlite"),
		OCRCacheDBPath:    filepath.Join(rootDir, "ocr.sqlite"),
		EncryptionKeyPath: filepath.Join(rootDir, "config.key"),
		LibraryRootDir:    filepath.Join(rootDir, "library"),
		WorkspacesRootDir: filepath.Join(rootDir, "library", "workspaces"),
	}
}

func TestWorkspaceKnowledgeFilesRoundTrip(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)

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

	extractPath, err := files.ExtractPath("paper-a")
	if err != nil {
		t.Fatalf("ExtractPath error: %v", err)
	}

	source := WorkspaceKnowledgeSource{
		ID:           "source:paper-a",
		WorkspaceID:  workspace.ID,
		Title:        "Paper A",
		Slug:         "paper-a",
		Kind:         "pdf",
		AbsolutePath: "C:/papers/paper-a.pdf",
		ContentHash:  "hash-a",
		ExtractPath:  extractPath,
		Status:       "ready",
	}

	if err := files.WriteSources([]WorkspaceKnowledgeSource{source}); err != nil {
		t.Fatalf("WriteSources error: %v", err)
	}

	sourcesPath, err := files.SourcesPath()
	if err != nil {
		t.Fatalf("SourcesPath error: %v", err)
	}

	rawSourcesJSON, err := os.ReadFile(sourcesPath)
	if err != nil {
		t.Fatalf("ReadFile sources manifest error: %v", err)
	}

	var sourceRecords []map[string]any
	if err := json.Unmarshal(rawSourcesJSON, &sourceRecords); err != nil {
		t.Fatalf("Unmarshal sources manifest error: %v", err)
	}
	if len(sourceRecords) != 1 {
		t.Fatalf("sources manifest len = %d, want 1", len(sourceRecords))
	}
	if _, ok := sourceRecords[0]["sourceId"]; !ok {
		t.Fatalf("sources manifest keys = %#v, want sourceId", sourceRecords[0])
	}
	if _, ok := sourceRecords[0]["id"]; ok {
		t.Fatalf("sources manifest keys = %#v, did not expect id", sourceRecords[0])
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

func TestWorkspaceKnowledgeFilesRejectsInvalidWorkspaceID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		workspaceID string
	}{
		{name: "empty", workspaceID: ""},
		{name: "whitespace", workspaceID: "   "},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			paths := newWorkspaceKnowledgeTestPaths(tempDir)
			files := newWorkspaceKnowledgeFiles(paths, tc.workspaceID)

			if _, err := files.SourcesPath(); err == nil {
				t.Fatalf("SourcesPath expected error for workspaceID %q", tc.workspaceID)
			}
			if err := files.EnsureLayout(); err == nil {
				t.Fatalf("EnsureLayout expected error for workspaceID %q", tc.workspaceID)
			}
			if err := files.WriteSources(nil); err == nil {
				t.Fatalf("WriteSources expected error for workspaceID %q", tc.workspaceID)
			}

			info, err := os.Stat(paths.WorkspacesRootDir)
			if os.IsNotExist(err) {
				return
			}
			if err != nil {
				t.Fatalf("Stat workspaces root error: %v", err)
			}
			if !info.IsDir() {
				t.Fatalf("workspaces root %q should be a directory", paths.WorkspacesRootDir)
			}

			entries, err := os.ReadDir(paths.WorkspacesRootDir)
			if err != nil {
				t.Fatalf("ReadDir workspaces root error: %v", err)
			}
			if len(entries) != 0 {
				t.Fatalf("workspaces root entries = %d, want 0", len(entries))
			}
		})
	}
}

func TestWorkspaceKnowledgeFilesRejectsTraversalPathSegments(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")

	slugCases := []struct {
		name  string
		value string
	}{
		{name: "forward-slash", value: "../paper-a"},
		{name: "backslash", value: `..\paper-a`},
	}
	for _, tc := range slugCases {
		tc := tc
		t.Run("slug-"+tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := files.ExtractPath(tc.value); err == nil {
				t.Fatalf("ExtractPath expected error for slug %q", tc.value)
			}
			if _, err := files.BySourcePath(tc.value); err == nil {
				t.Fatalf("BySourcePath expected error for slug %q", tc.value)
			}
			if err := files.WriteBySource(tc.value, WorkspaceKnowledgeBySourcePayload{}); err == nil {
				t.Fatalf("WriteBySource expected error for slug %q", tc.value)
			}
		})
	}

	scanRunCases := []struct {
		name  string
		value string
	}{
		{name: "forward-slash", value: "../scan-run-1"},
		{name: "backslash", value: `..\scan-run-1`},
	}
	for _, tc := range scanRunCases {
		tc := tc
		t.Run("scan-run-"+tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := files.ScanRunPath(tc.value); err == nil {
				t.Fatalf("ScanRunPath expected error for scan run ID %q", tc.value)
			}
			if err := files.WriteScanRun(WorkspaceKnowledgeScanRunRecord{ID: tc.value}); err == nil {
				t.Fatalf("WriteScanRun expected error for scan run ID %q", tc.value)
			}
		})
	}
}

func TestWorkspaceKnowledgeFilesRejectsWindowsInvalidPathCharacters(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")

	invalidSlug := "paper:a"
	if _, err := files.ExtractPath(invalidSlug); err == nil {
		t.Fatalf("ExtractPath expected error for slug %q", invalidSlug)
	}
	if _, err := files.BySourcePath(invalidSlug); err == nil {
		t.Fatalf("BySourcePath expected error for slug %q", invalidSlug)
	}
	if err := files.WriteBySource(invalidSlug, WorkspaceKnowledgeBySourcePayload{}); err == nil {
		t.Fatalf("WriteBySource expected error for slug %q", invalidSlug)
	}

	invalidScanRunID := "scan-run:1"
	if _, err := files.ScanRunPath(invalidScanRunID); err == nil {
		t.Fatalf("ScanRunPath expected error for scan run ID %q", invalidScanRunID)
	}
	if err := files.WriteScanRun(WorkspaceKnowledgeScanRunRecord{ID: invalidScanRunID}); err == nil {
		t.Fatalf("WriteScanRun expected error for scan run ID %q", invalidScanRunID)
	}
}

func TestWorkspaceKnowledgeFilesWriteScanRun(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")

	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	record := WorkspaceKnowledgeScanRunRecord{
		ID:          "scan-run-1",
		WorkspaceID: "workspace-a",
		Status:      "completed",
		StartedAt:   nowRFC3339(),
		FinishedAt:  nowRFC3339(),
		SourceIDs:   []string{"source:paper-a"},
		Message:     "ok",
	}
	if err := files.WriteScanRun(record); err != nil {
		t.Fatalf("WriteScanRun error: %v", err)
	}

	scanRunPath, err := files.ScanRunPath(record.ID)
	if err != nil {
		t.Fatalf("ScanRunPath error: %v", err)
	}

	data, err := os.ReadFile(scanRunPath)
	if err != nil {
		t.Fatalf("ReadFile scan run error: %v", err)
	}

	var got WorkspaceKnowledgeScanRunRecord
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal scan run error: %v", err)
	}
	if got.ID != record.ID {
		t.Fatalf("scan run ID = %q, want %q", got.ID, record.ID)
	}
	if got.Status != record.Status {
		t.Fatalf("scan run status = %q, want %q", got.Status, record.Status)
	}
	if len(got.SourceIDs) != 1 || got.SourceIDs[0] != "source:paper-a" {
		t.Fatalf("scan run source IDs = %#v, want source:paper-a", got.SourceIDs)
	}
}

func TestWorkspaceKnowledgeFilesEnsureLayoutCreatesExpectedDirectoryTree(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")

	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	expectedDirs := []string{
		filepath.Join(paths.WorkspacesRootDir, "workspace-a"),
		filepath.Join(paths.WorkspacesRootDir, "workspace-a", "raw"),
		filepath.Join(paths.WorkspacesRootDir, "workspace-a", "raw", "extracts"),
		filepath.Join(paths.WorkspacesRootDir, "workspace-a", "schema"),
		filepath.Join(paths.WorkspacesRootDir, "workspace-a", "schema", "by-source"),
		filepath.Join(paths.WorkspacesRootDir, "workspace-a", "schema", "scan-runs"),
		filepath.Join(paths.WorkspacesRootDir, "workspace-a", "wiki"),
		filepath.Join(paths.WorkspacesRootDir, "workspace-a", "wiki", "docs"),
		filepath.Join(paths.WorkspacesRootDir, "workspace-a", "wiki", "concepts"),
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Stat directory %q error: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q should be a directory", dir)
		}
	}
}
