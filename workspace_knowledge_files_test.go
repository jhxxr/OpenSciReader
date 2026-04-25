package main

import (
	"os"
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
	requiredDirs := []string{
		filepath.Join(workspaceRoot, "sources"),
		filepath.Join(workspaceRoot, "sources", "pdfs"),
		filepath.Join(workspaceRoot, "inputs"),
		filepath.Join(workspaceRoot, "inputs", "markitdown"),
		filepath.Join(workspaceRoot, "inputs", "manifests"),
		filepath.Join(workspaceRoot, "state"),
		filepath.Join(workspaceRoot, "state", "by-source"),
		filepath.Join(workspaceRoot, "state", "jobs"),
		filepath.Join(workspaceRoot, "wiki"),
		filepath.Join(workspaceRoot, "wiki", "docs"),
		filepath.Join(workspaceRoot, "wiki", "concepts"),
	}
	for _, dir := range requiredDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("expected directory %q: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to be a directory", dir)
		}
	}

	sourcesManifestPath, err := files.SourcesManifestPath()
	if err != nil {
		t.Fatalf("SourcesManifestPath() error = %v", err)
	}
	if want := filepath.Join(workspaceRoot, "state", "sources.json"); sourcesManifestPath != want {
		t.Fatalf("SourcesManifestPath() = %q, want %q", sourcesManifestPath, want)
	}

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

	overviewPath, err := files.OverviewPath()
	if err != nil {
		t.Fatalf("OverviewPath() error = %v", err)
	}
	if want := filepath.Join(workspaceRoot, "wiki", "overview.md"); overviewPath != want {
		t.Fatalf("OverviewPath() = %q, want %q", overviewPath, want)
	}

	openQuestionsPath, err := files.OpenQuestionsPath()
	if err != nil {
		t.Fatalf("OpenQuestionsPath() error = %v", err)
	}
	if want := filepath.Join(workspaceRoot, "wiki", "open-questions.md"); openQuestionsPath != want {
		t.Fatalf("OpenQuestionsPath() = %q, want %q", openQuestionsPath, want)
	}
}

func TestWorkspaceKnowledgeFilesReadSourcesFallsBackToLegacyRawLayout(t *testing.T) {
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
	legacyPath := filepath.Join(workspaceRoot, "raw", "sources.json")
	if err := writeWorkspaceKnowledgeJSON(legacyPath, []WorkspaceKnowledgeSource{{
		ID:          "source:paper-a",
		WorkspaceID: "workspace-a",
		Title:       "Paper A",
		Slug:        "paper-a",
		Kind:        "pdf",
	}}); err != nil {
		t.Fatalf("writeWorkspaceKnowledgeJSON(legacyPath) error = %v", err)
	}

	sources, err := files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources() error = %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("ReadSources() len = %d, want 1", len(sources))
	}
	if sources[0].ID != "source:paper-a" {
		t.Fatalf("sources[0].ID = %q, want %q", sources[0].ID, "source:paper-a")
	}
}

func TestWorkspaceKnowledgeFilesReadStateFallsBackToLegacyAggregateLayout(t *testing.T) {
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
	legacySchemaDir := filepath.Join(workspaceRoot, "schema")
	if err := writeWorkspaceKnowledgeJSON(filepath.Join(legacySchemaDir, "entities.json"), []WorkspaceKnowledgeEntity{{ID: "entity:attention", Title: "Attention"}}); err != nil {
		t.Fatalf("writeWorkspaceKnowledgeJSON(entities) error = %v", err)
	}
	if err := writeWorkspaceKnowledgeJSON(filepath.Join(legacySchemaDir, "claims.json"), []WorkspaceKnowledgeClaim{{ID: "claim:attention", Title: "Attention claim"}}); err != nil {
		t.Fatalf("writeWorkspaceKnowledgeJSON(claims) error = %v", err)
	}
	if err := writeWorkspaceKnowledgeJSON(filepath.Join(legacySchemaDir, "tasks.json"), []WorkspaceKnowledgeTask{{ID: "task:attention", Title: "Attention task"}}); err != nil {
		t.Fatalf("writeWorkspaceKnowledgeJSON(tasks) error = %v", err)
	}
	if err := writeWorkspaceKnowledgeJSON(filepath.Join(legacySchemaDir, "relations.json"), []WorkspaceKnowledgeRelation{{ID: "relation:attention", Type: "supports"}}); err != nil {
		t.Fatalf("writeWorkspaceKnowledgeJSON(relations) error = %v", err)
	}

	entities, err := readWorkspaceKnowledgeEntities(files)
	if err != nil {
		t.Fatalf("readWorkspaceKnowledgeEntities() error = %v", err)
	}
	if len(entities) != 1 || entities[0].ID != "entity:attention" {
		t.Fatalf("entities = %#v, want legacy entity", entities)
	}

	claims, err := readWorkspaceKnowledgeClaims(files)
	if err != nil {
		t.Fatalf("readWorkspaceKnowledgeClaims() error = %v", err)
	}
	if len(claims) != 1 || claims[0].ID != "claim:attention" {
		t.Fatalf("claims = %#v, want legacy claim", claims)
	}

	tasks, err := readWorkspaceKnowledgeTasks(files)
	if err != nil {
		t.Fatalf("readWorkspaceKnowledgeTasks() error = %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "task:attention" {
		t.Fatalf("tasks = %#v, want legacy task", tasks)
	}

	relations, err := readWorkspaceKnowledgeRelations(files)
	if err != nil {
		t.Fatalf("readWorkspaceKnowledgeRelations() error = %v", err)
	}
	if len(relations) != 1 || relations[0].ID != "relation:attention" {
		t.Fatalf("relations = %#v, want legacy relation", relations)
	}
}
