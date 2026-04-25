package main

import (
	"path/filepath"
	"testing"
)

func TestWorkspaceKnowledgeFilesUsesSourcesInputsStateLayout(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	paths := appPaths{
		RootDir:           rootDir,
		LibraryRootDir:    filepath.Join(rootDir, "library"),
		WorkspacesRootDir: filepath.Join(rootDir, "library", "workspaces"),
	}

	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}

	workspaceRoot := filepath.Join(paths.WorkspacesRootDir, "workspace-a")

	markItDownPath, err := files.MarkItDownPath("paper-a")
	if err != nil {
		t.Fatalf("MarkItDownPath() error = %v", err)
	}
	if want := filepath.Join(workspaceRoot, "inputs", "markitdown", "paper-a.md"); markItDownPath != want {
		t.Fatalf("MarkItDownPath() = %q, want %q", markItDownPath, want)
	}

	bySourcePath, err := files.BySourcePath("paper-a")
	if err != nil {
		t.Fatalf("BySourcePath() error = %v", err)
	}
	if want := filepath.Join(workspaceRoot, "state", "by-source", "paper-a.json"); bySourcePath != want {
		t.Fatalf("BySourcePath() = %q, want %q", bySourcePath, want)
	}

	jobPath, err := files.JobPath("job-1")
	if err != nil {
		t.Fatalf("JobPath() error = %v", err)
	}
	if want := filepath.Join(workspaceRoot, "state", "jobs", "job-1.json"); jobPath != want {
		t.Fatalf("JobPath() = %q, want %q", jobPath, want)
	}

	indexPath, err := files.IndexPath()
	if err != nil {
		t.Fatalf("IndexPath() error = %v", err)
	}
	if want := filepath.Join(workspaceRoot, "wiki", "index.md"); indexPath != want {
		t.Fatalf("IndexPath() = %q, want %q", indexPath, want)
	}

	logPath, err := files.LogPath()
	if err != nil {
		t.Fatalf("LogPath() error = %v", err)
	}
	if want := filepath.Join(workspaceRoot, "wiki", "log.md"); logPath != want {
		t.Fatalf("LogPath() = %q, want %q", logPath, want)
	}
}
