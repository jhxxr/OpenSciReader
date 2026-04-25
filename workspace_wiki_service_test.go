package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
