# Workspace PDF Wiki Architecture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the workspace knowledge pipeline from the mixed `raw / schema / wiki` layout to the clarified `sources / inputs / state / wiki` architecture while preserving the existing workspace scan, compile, and query flows.

**Architecture:** Keep the current Go filesystem-first pipeline and SQLite-backed job/page metadata, but rename the on-disk layers so PDF sources, MarkItDown caches, machine state, and human-readable wiki output have explicit boundaries. Make source processing status explicit, keep by-source machine memory as JSON, and update compile/query logic to prefer wiki pages before structured state and derived Markdown.

**Tech Stack:** Go, Wails v2, SQLite, filesystem JSON/Markdown artifacts, React 18, TypeScript, Vite, MarkItDown.

---

## File Map

### Backend files

- Modify: `workspace_knowledge_files.go`
  Rename layout helpers from `raw/schema` semantics to `sources/inputs/state`, add explicit helpers for MarkItDown cache, sources manifest, job summaries, and wiki index/log outputs.
- Modify: `workspace_knowledge_types.go`
  Expand `WorkspaceKnowledgeSource` to separate source path, MarkItDown path, and source-level processing statuses. Add compile summary types used by the runner and query service.
- Modify: `workspace_wiki_service.go`
  Keep the scan runner, but update it to write `sources`, `inputs`, and `state` artifacts, mark source statuses, debounce workspace compile, and persist `state/compile.json`.
- Modify: `workspace_knowledge_compile.go`
  Read from `state/by-source`, write aggregate state under `state/`, generate `wiki/index.md` and `wiki/log.md`, and stop referring to `schema` or `raw/extracts` in comments or output helpers.
- Modify: `workspace_knowledge_query.go`
  Update evidence retrieval to prefer `wiki` content, then `state`, then `inputs/markitdown`, and only fall back to PDF-backed paths if required.
- Modify: `workspace_knowledge_prompts.go`
  Update prompt wording to refer to sources, MarkItDown cache, and wiki/state terminology consistently.
- Modify: `config_types.go`
  Add Wails-bound types for source-level processing status and compile summary if they need to be returned to the frontend.
- Modify: `app.go`
  Expose list/read methods for workspace source status and compile summary if the frontend needs status visibility.

### Test files

- Create: `workspace_knowledge_files_test.go`
  Verifies the new layout and path helper contract.
- Create: `workspace_knowledge_compile_test.go`
  Verifies `state` aggregate outputs plus `wiki/index.md` and `wiki/log.md` generation.
- Create: `workspace_knowledge_query_test.go`
  Verifies retrieval precedence `wiki -> state -> inputs/markitdown`.
- Modify: `workspace_wiki_service_test.go`
  Verifies source status transitions and the new on-disk artifact layout.

### Frontend files

- Modify: `frontend/src/types/workspaceWiki.ts`
  Add any new source/build status fields returned by Wails job APIs.
- Modify: `frontend/src/types/workspaceKnowledge.ts`
  Add `WorkspaceKnowledgeSource` and `WorkspaceKnowledgeCompileSummary` if exposed to the UI.
- Modify: `frontend/src/api/workspaceWiki.ts`
  Normalize any new job stage strings or status payload fields.
- Modify: `frontend/src/api/workspaceKnowledge.ts`
  Add wrappers for any new source-status list endpoints.
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.d.ts`
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.js`
- Modify via `wails generate module`: `frontend/wailsjs/go/models.ts`

## Task 1: Rename The Filesystem Layout Without Rewriting The Pipeline

**Files:**
- Create: `workspace_knowledge_files_test.go`
- Modify: `workspace_knowledge_files.go`
- Modify: `workspace_knowledge_types.go`

- [ ] **Step 1: Write the failing layout helper test**

```go
package main

import (
	"path/filepath"
	"testing"
)

func TestWorkspaceKnowledgeFilesUsesSourcesInputsStateLayout(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := appPaths{
		RootDir:           tempDir,
		WorkspacesRootDir: filepath.Join(tempDir, "library", "workspaces"),
	}
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")

	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	markitdownPath, err := files.MarkItDownPath("paper-a")
	if err != nil {
		t.Fatalf("MarkItDownPath error: %v", err)
	}
	bySourcePath, err := files.BySourcePath("paper-a")
	if err != nil {
		t.Fatalf("BySourcePath error: %v", err)
	}
	jobPath, err := files.JobPath("job-1")
	if err != nil {
		t.Fatalf("JobPath error: %v", err)
	}

	if want := filepath.Join(paths.WorkspacesRootDir, "workspace-a", "inputs", "markitdown", "paper-a.md"); markitdownPath != want {
		t.Fatalf("markitdownPath = %q, want %q", markitdownPath, want)
	}
	if want := filepath.Join(paths.WorkspacesRootDir, "workspace-a", "state", "by-source", "paper-a.json"); bySourcePath != want {
		t.Fatalf("bySourcePath = %q, want %q", bySourcePath, want)
	}
	if want := filepath.Join(paths.WorkspacesRootDir, "workspace-a", "state", "jobs", "job-1.json"); jobPath != want {
		t.Fatalf("jobPath = %q, want %q", jobPath, want)
	}
	if _, err := files.IndexPath(); err != nil {
		t.Fatalf("IndexPath error: %v", err)
	}
	if _, err := files.LogPath(); err != nil {
		t.Fatalf("LogPath error: %v", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./... -run TestWorkspaceKnowledgeFilesUsesSourcesInputsStateLayout -count=1`

Expected: FAIL with compile errors such as `files.MarkItDownPath undefined`, `files.JobPath undefined`, or path assertions still pointing at `raw/schema`.

- [ ] **Step 3: Implement the minimal layout helper migration**

```go
// workspace_knowledge_files.go
func (f workspaceKnowledgeFiles) MarkItDownPath(sourceSlug string) (string, error) {
	markitdownDir, err := f.markitdownDir()
	if err != nil {
		return "", err
	}
	validatedSourceSlug, err := validateWorkspaceKnowledgePathSegment("source slug", sourceSlug)
	if err != nil {
		return "", err
	}
	return filepath.Join(markitdownDir, validatedSourceSlug+".md"), nil
}

func (f workspaceKnowledgeFiles) SourcesManifestPath() (string, error) {
	stateDir, err := f.stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "sources.json"), nil
}

func (f workspaceKnowledgeFiles) JobPath(jobID string) (string, error) {
	jobsDir, err := f.jobsDir()
	if err != nil {
		return "", err
	}
	validatedJobID, err := validateWorkspaceKnowledgePathSegment("job id", jobID)
	if err != nil {
		return "", err
	}
	return filepath.Join(jobsDir, validatedJobID+".json"), nil
}

func (f workspaceKnowledgeFiles) IndexPath() (string, error) {
	wikiDir, err := f.wikiDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(wikiDir, "index.md"), nil
}

func (f workspaceKnowledgeFiles) LogPath() (string, error) {
	wikiDir, err := f.wikiDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(wikiDir, "log.md"), nil
}
```

```go
// workspace_knowledge_files.go
func (f workspaceKnowledgeFiles) layoutDirs() ([]string, error) {
	workspaceRootDir, err := f.workspaceRootDir()
	if err != nil {
		return nil, err
	}

	sourcesDir := filepath.Join(workspaceRootDir, "sources")
	pdfsDir := filepath.Join(sourcesDir, "pdfs")
	inputsDir := filepath.Join(workspaceRootDir, "inputs")
	markitdownDir := filepath.Join(inputsDir, "markitdown")
	manifestsDir := filepath.Join(inputsDir, "manifests")
	stateDir := filepath.Join(workspaceRootDir, "state")
	bySourceDir := filepath.Join(stateDir, "by-source")
	jobsDir := filepath.Join(stateDir, "jobs")
	wikiDir := filepath.Join(workspaceRootDir, "wiki")
	wikiDocsDir := filepath.Join(wikiDir, "docs")
	wikiConceptsDir := filepath.Join(wikiDir, "concepts")

	return []string{
		workspaceRootDir,
		sourcesDir,
		pdfsDir,
		inputsDir,
		markitdownDir,
		manifestsDir,
		stateDir,
		bySourceDir,
		jobsDir,
		wikiDir,
		wikiDocsDir,
		wikiConceptsDir,
	}, nil
}
```

- [ ] **Step 4: Run the filesystem tests**

Run: `go test ./... -run 'TestWorkspaceKnowledgeFilesUsesSourcesInputsStateLayout|TestWorkspaceKnowledgeFilesRoundTrip' -count=1`

Expected: PASS

- [ ] **Step 5: Commit the layout migration**

```bash
git add workspace_knowledge_files.go workspace_knowledge_types.go workspace_knowledge_files_test.go
git commit -m "refactor: rename workspace knowledge layout layers"
```

## Task 2: Split Source Processing State From Workspace Build State

**Files:**
- Modify: `workspace_knowledge_types.go`
- Modify: `workspace_wiki_service.go`
- Modify: `workspace_wiki_service_test.go`
- Modify: `config_types.go`

- [ ] **Step 1: Write the failing source status test**

```go
func TestStartWorkspaceWikiScanPersistsSourceProcessingState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	workspace, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{Name: "Knowledge Workspace"})
	if err != nil {
		t.Fatalf("CreateWorkspace error: %v", err)
	}

	seedPath := filepath.Join(t.TempDir(), "attention.md")
	if err := os.WriteFile(seedPath, []byte("# Attention\n\nattention matters\n"), 0o600); err != nil {
		t.Fatalf("write seed markdown error: %v", err)
	}

	if _, err := store.ImportFiles(ctx, paths, ImportFilesInput{WorkspaceID: workspace.ID, FilePaths: []string{seedPath}, Title: "Attention"}); err != nil {
		t.Fatalf("ImportFiles error: %v", err)
	}

	llm := &stubWorkspaceKnowledgeLLM{}
	service := newWorkspaceWikiService(paths, store, panicWorkspaceKnowledgeExtractor{}, llm)
	job, err := service.Start(ctx, WorkspaceWikiScanStartInput{WorkspaceID: workspace.ID, ProviderID: 1, ModelID: 1})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	job = waitForWorkspaceWikiJobTerminal(t, ctx, store, job.JobID)

	files := newWorkspaceKnowledgeFiles(paths, workspace.ID)
	sources, err := files.ReadSources()
	if err != nil {
		t.Fatalf("ReadSources error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("source count = %d, want 1", len(sources))
	}
	if sources[0].MarkItDownStatus != "ready" {
		t.Fatalf("markitdownStatus = %q, want ready", sources[0].MarkItDownStatus)
	}
	if sources[0].ExtractStatus != "ready" {
		t.Fatalf("extractStatus = %q, want ready", sources[0].ExtractStatus)
	}

	compileSummary, err := files.ReadCompileSummary()
	if err != nil {
		t.Fatalf("ReadCompileSummary error: %v", err)
	}
	if !slices.Contains(compileSummary.UpdatedWikiPaths, filepath.Join(paths.WorkspacesRootDir, workspace.ID, "wiki", "overview.md")) {
		t.Fatalf("updated wiki paths = %#v, want overview.md", compileSummary.UpdatedWikiPaths)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./... -run TestStartWorkspaceWikiScanPersistsSourceProcessingState -count=1`

Expected: FAIL because `WorkspaceKnowledgeSource` does not yet have `MarkItDownStatus` / `ExtractStatus` and `ReadCompileSummary` does not exist.

- [ ] **Step 3: Add explicit source status and compile summary types**

```go
// workspace_knowledge_types.go
type WorkspaceKnowledgeSource struct {
	ID               string `json:"sourceId"`
	WorkspaceID      string `json:"workspaceId"`
	Title            string `json:"title"`
	Slug             string `json:"slug"`
	Kind             string `json:"kind"`
	SourcePath       string `json:"sourcePath"`
	MarkItDownPath   string `json:"markitdownPath"`
	ContentHash      string `json:"contentHash"`
	DocumentID       string `json:"documentId"`
	MarkItDownStatus string `json:"markitdownStatus"`
	ExtractStatus    string `json:"extractStatus"`
	LastIngestAt     string `json:"lastIngestAt"`
	LastSuccessAt    string `json:"lastSuccessAt"`
	LastError        string `json:"lastError"`
}

type WorkspaceKnowledgeCompileSummary struct {
	WorkspaceID      string   `json:"workspaceId"`
	StartedAt        string   `json:"startedAt"`
	FinishedAt       string   `json:"finishedAt"`
	IncludedSourceIDs []string `json:"includedSourceIds"`
	FailedSourceIDs  []string `json:"failedSourceIds"`
	UpdatedWikiPaths []string `json:"updatedWikiPaths"`
	CompileDirty     bool     `json:"compileDirty"`
	WikiDirty        bool     `json:"wikiDirty"`
}
```

```go
// workspace_wiki_service.go
func markSourceReady(source WorkspaceKnowledgeSource, markitdownPath string) WorkspaceKnowledgeSource {
	now := nowRFC3339()
	source.MarkItDownPath = markitdownPath
	source.MarkItDownStatus = "ready"
	source.ExtractStatus = "ready"
	source.LastSuccessAt = now
	source.LastError = ""
	return source
}
```

```go
// workspace_knowledge_files.go
func (f workspaceKnowledgeFiles) CompileSummaryPath() (string, error) {
	stateDir, err := f.stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "compile.json"), nil
}

func (f workspaceKnowledgeFiles) ReadCompileSummary() (WorkspaceKnowledgeCompileSummary, error) {
	path, err := f.CompileSummaryPath()
	if err != nil {
		return WorkspaceKnowledgeCompileSummary{}, err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return WorkspaceKnowledgeCompileSummary{}, nil
		}
		return WorkspaceKnowledgeCompileSummary{}, err
	}
	var summary WorkspaceKnowledgeCompileSummary
	if err := readWorkspaceKnowledgeJSON(path, &summary); err != nil {
		return WorkspaceKnowledgeCompileSummary{}, err
	}
	return summary, nil
}
```

- [ ] **Step 4: Wire the runner to persist the new state fields**

```go
// workspace_wiki_service.go
source.MarkItDownStatus = "running"
source.ExtractStatus = "pending"

if err := writeWorkspaceKnowledgeMarkdown(markitdownPath, markdown); err != nil {
	source.MarkItDownStatus = "failed"
	source.LastError = err.Error()
	return err
}

source.MarkItDownStatus = "ready"
source.ExtractStatus = "running"

if err := files.WriteBySource(source.Slug, payload); err != nil {
	source.ExtractStatus = "failed"
	source.LastError = err.Error()
	return err
}

source = markSourceReady(source, markitdownPath)
```

- [ ] **Step 5: Run the runner tests**

Run: `go test ./... -run 'TestStartWorkspaceWikiScanCreatesPageRecords|TestStartWorkspaceWikiScanPersistsSourceProcessingState' -count=1`

Expected: PASS

- [ ] **Step 6: Commit the source state split**

```bash
git add workspace_knowledge_types.go workspace_wiki_service.go workspace_wiki_service_test.go workspace_knowledge_files.go config_types.go
git commit -m "feat: track source processing state in workspace wiki scans"
```

## Task 3: Rebuild Compile And Query Around `state` And `wiki`

**Files:**
- Create: `workspace_knowledge_compile_test.go`
- Create: `workspace_knowledge_query_test.go`
- Modify: `workspace_knowledge_compile.go`
- Modify: `workspace_knowledge_query.go`
- Modify: `workspace_knowledge_prompts.go`

- [ ] **Step 1: Write the failing compile output test**

```go
func TestCompileWorkspaceKnowledgeWritesStateAndWikiIndex(t *testing.T) {
	t.Parallel()

	paths := appPaths{WorkspacesRootDir: t.TempDir()}
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	payload := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{
			ID:             "source:paper-a",
			WorkspaceID:    "workspace-a",
			Title:          "Paper A",
			Slug:           "paper-a",
			Kind:           "markdown",
			SourcePath:     "sources/pdfs/paper-a.pdf",
			MarkItDownPath: "inputs/markitdown/paper-a.md",
		},
		Entities: []WorkspaceKnowledgeEntity{{
			ID: "entity:attention",
			WorkspaceID: "workspace-a",
			Title: "Attention",
			Type: "concept",
			Summary: "Attention is the main concept.",
		}},
	}
	if err := files.WriteBySource("paper-a", payload); err != nil {
		t.Fatalf("WriteBySource error: %v", err)
	}

	_, err := CompileWorkspaceKnowledge(files, "Attention Workspace")
	if err != nil {
		t.Fatalf("CompileWorkspaceKnowledge error: %v", err)
	}

	indexPath, _ := files.IndexPath()
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("ReadFile(index) error: %v", err)
	}
	if !strings.Contains(string(indexData), "[[docs/paper-a|Paper A]]") {
		t.Fatalf("index markdown = %q, want source doc link", string(indexData))
	}

	logPath, _ := files.LogPath()
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("stat log path error: %v", err)
	}
}
```

- [ ] **Step 2: Write the failing retrieval-order test**

```go
func TestRetrieveWorkspaceKnowledgeEvidencePrefersWikiBeforeStateAndInputs(t *testing.T) {
	t.Parallel()

	paths := appPaths{WorkspacesRootDir: t.TempDir()}
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	indexPath, _ := files.IndexPath()
	if err := writeWorkspaceKnowledgeMarkdown(indexPath, "# Index\n\n## Concepts\n- [[concepts/attention|Attention]]\n"); err != nil {
		t.Fatalf("write index error: %v", err)
	}
	conceptPath, _ := files.ConceptWikiPath("attention")
	if err := writeWorkspaceKnowledgeMarkdown(conceptPath, "# Attention\n\nAttention is the highest-level summary.\n"); err != nil {
		t.Fatalf("write concept error: %v", err)
	}

	hits, err := retrieveWorkspaceKnowledgeEvidence(files, "What is attention?")
	if err != nil {
		t.Fatalf("retrieveWorkspaceKnowledgeEvidence error: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("hits should not be empty")
	}
	if hits[0].Kind != "wiki_page" {
		t.Fatalf("first hit kind = %q, want wiki_page", hits[0].Kind)
	}
}
```

- [ ] **Step 3: Run the compile and query tests to verify they fail**

Run: `go test ./... -run 'TestCompileWorkspaceKnowledgeWritesStateAndWikiIndex|TestRetrieveWorkspaceKnowledgeEvidencePrefersWikiBeforeStateAndInputs' -count=1`

Expected: FAIL because `wiki/index.md` and `wiki/log.md` are not written yet and retrieval still reads aggregate state before wiki.

- [ ] **Step 4: Implement `state` aggregate writes and wiki index/log generation**

```go
// workspace_knowledge_compile.go
func writeWorkspaceKnowledgeAggregates(files workspaceKnowledgeFiles, snapshot WorkspaceKnowledgeSnapshot) error {
	entitiesPath, err := files.EntitiesPath()
	if err != nil {
		return err
	}
	if err := writeWorkspaceKnowledgeJSON(entitiesPath, snapshot.Entities); err != nil {
		return err
	}
	claimsPath, err := files.ClaimsPath()
	if err != nil {
		return err
	}
	if err := writeWorkspaceKnowledgeJSON(claimsPath, snapshot.Claims); err != nil {
		return err
	}
	relationsPath, err := files.RelationsPath()
	if err != nil {
		return err
	}
	if err := writeWorkspaceKnowledgeJSON(relationsPath, snapshot.Relations); err != nil {
		return err
	}
	tasksPath, err := files.TasksPath()
	if err != nil {
		return err
	}
	return writeWorkspaceKnowledgeJSON(tasksPath, snapshot.Tasks)
}

func buildIndexWikiPage(snapshot WorkspaceKnowledgeSnapshot, sourceDocSlugs map[string]string, conceptSlugs map[string]string) string {
	var builder strings.Builder
	builder.WriteString("# Workspace Index\n\n")
	builder.WriteString("## Documents\n")
	for _, source := range snapshot.Sources {
		slug := sourceDocSlugs[source.ID]
		builder.WriteString(fmt.Sprintf("- [[docs/%s|%s]]\n", slug, source.Title))
	}
	builder.WriteString("\n## Concepts\n")
	for _, entity := range snapshot.Entities {
		slug := conceptSlugs[entity.ID]
		builder.WriteString(fmt.Sprintf("- [[concepts/%s|%s]]\n", slug, entity.Title))
	}
	return builder.String()
}
```

```go
// workspace_knowledge_compile.go
type workspaceKnowledgeWikiWritePlan struct {
	Index         workspaceKnowledgeOutputFile
	Overview      workspaceKnowledgeOutputFile
	OpenQuestions workspaceKnowledgeOutputFile
	Log           workspaceKnowledgeOutputFile
	Documents     []workspaceKnowledgeOutputFile
	Concepts      []workspaceKnowledgeOutputFile
}

func buildWorkspaceKnowledgeLogPage(snapshot WorkspaceKnowledgeSnapshot) string {
	return fmt.Sprintf(
		"# Workspace Log\n\n## Latest Compile\n- sources: %d\n- entities: %d\n- claims: %d\n- tasks: %d\n",
		len(snapshot.Sources),
		len(snapshot.Entities),
		len(snapshot.Claims),
		len(snapshot.Tasks),
	)
}

indexPath, _ := files.IndexPath()
logPath, _ := files.LogPath()
plan.Index = workspaceKnowledgeOutputFile{Path: indexPath, Content: buildIndexWikiPage(snapshot, sourceDocSlugs, conceptSlugs)}
plan.Log = workspaceKnowledgeOutputFile{Path: logPath, Content: buildWorkspaceKnowledgeLogPage(snapshot)}
```

- [ ] **Step 5: Implement wiki-first evidence retrieval**

```go
// workspace_knowledge_query.go
func retrieveWorkspaceKnowledgeEvidence(files workspaceKnowledgeFiles, question string) ([]WorkspaceKnowledgeEvidenceHit, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return []WorkspaceKnowledgeEvidenceHit{}, nil
	}

	wikiHits, err := retrieveWorkspaceWikiEvidence(files, question)
	if err != nil {
		return nil, err
	}
	stateHits, err := retrieveWorkspaceStateEvidence(files, question)
	if err != nil {
		return nil, err
	}
	inputHits, err := retrieveWorkspaceMarkItDownEvidence(files, question)
	if err != nil {
		return nil, err
	}

	hits := append([]WorkspaceKnowledgeEvidenceHit{}, wikiHits...)
	hits = append(hits, stateHits...)
	hits = append(hits, inputHits...)
	if len(hits) > maxWorkspaceKnowledgeQueryEvidence {
		hits = hits[:maxWorkspaceKnowledgeQueryEvidence]
	}
	return hits, nil
}
```

- [ ] **Step 6: Run the compile and query tests**

Run: `go test ./... -run 'TestCompileWorkspaceKnowledgeWritesStateAndWikiIndex|TestRetrieveWorkspaceKnowledgeEvidencePrefersWikiBeforeStateAndInputs' -count=1`

Expected: PASS

- [ ] **Step 7: Commit the compile/query migration**

```bash
git add workspace_knowledge_compile.go workspace_knowledge_query.go workspace_knowledge_prompts.go workspace_knowledge_compile_test.go workspace_knowledge_query_test.go
git commit -m "feat: prefer wiki and state in workspace knowledge pipeline"
```

## Task 4: Expose Source And Build Status Through Wails And Frontend Types

**Files:**
- Modify: `config_types.go`
- Modify: `app.go`
- Modify: `frontend/src/types/workspaceKnowledge.ts`
- Modify: `frontend/src/api/workspaceKnowledge.ts`
- Modify: `frontend/src/types/workspaceWiki.ts`
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.d.ts`
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.js`
- Modify via `wails generate module`: `frontend/wailsjs/go/models.ts`

- [ ] **Step 1: Write the failing frontend-contract test at the TypeScript layer**

```ts
export interface WorkspaceKnowledgeSource {
  sourceId: string;
  workspaceId: string;
  title: string;
  slug: string;
  kind: string;
  sourcePath: string;
  markitdownPath: string;
  markitdownStatus: 'pending' | 'running' | 'ready' | 'failed';
  extractStatus: 'pending' | 'running' | 'ready' | 'failed' | 'stale';
  lastIngestAt: string;
  lastSuccessAt: string;
  lastError: string;
}
```

Use this snippet to update `frontend/src/types/workspaceKnowledge.ts`; the compile should fail until the Wails bindings and API wrappers are updated.

- [ ] **Step 2: Run the frontend build to verify it fails**

Run: `npm run build`

Workdir: `frontend`

Expected: FAIL with TypeScript errors because the new source types and Wails methods are not yet wired.

- [ ] **Step 3: Expose backend list methods and frontend wrappers**

```go
// app.go
func (a *App) ListWorkspaceKnowledgeSources(workspaceID string) ([]WorkspaceKnowledgeSource, error) {
	files := newWorkspaceKnowledgeFiles(a.paths, workspaceID)
	if err := files.EnsureLayout(); err != nil {
		return nil, err
	}
	return files.ReadSources()
}

func (a *App) GetWorkspaceKnowledgeCompileSummary(workspaceID string) (WorkspaceKnowledgeCompileSummary, error) {
	files := newWorkspaceKnowledgeFiles(a.paths, workspaceID)
	if err := files.EnsureLayout(); err != nil {
		return WorkspaceKnowledgeCompileSummary{}, err
	}
	return files.ReadCompileSummary()
}
```

```ts
// frontend/src/api/workspaceKnowledge.ts
interface WailsWorkspaceKnowledgeApp {
  ListWorkspaceKnowledgeSources: (workspaceId: string) => Promise<WorkspaceKnowledgeSource[]>;
  GetWorkspaceKnowledgeCompileSummary: (workspaceId: string) => Promise<WorkspaceKnowledgeCompileSummary>;
}

async listSources(workspaceId: string): Promise<WorkspaceKnowledgeSource[]> {
  const app = getApp();
  if (!app || workspaceId.trim() === '') {
    return [];
  }
  return app.ListWorkspaceKnowledgeSources(workspaceId);
},

async getCompileSummary(workspaceId: string): Promise<WorkspaceKnowledgeCompileSummary | null> {
  const app = getApp();
  if (!app || workspaceId.trim() === '') {
    return null;
  }
  return app.GetWorkspaceKnowledgeCompileSummary(workspaceId);
},
```

- [ ] **Step 4: Regenerate bindings and run verification**

Run: `wails generate module`

Expected: Wails updates `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/main/App.js`, and `frontend/wailsjs/go/models.ts` with the new methods and types.

Run: `npm run build`

Workdir: `frontend`

Expected: PASS

Run: `go test ./...`

Expected: PASS

- [ ] **Step 5: Commit the Wails/frontend contract update**

```bash
git add app.go config_types.go frontend/src/types/workspaceKnowledge.ts frontend/src/api/workspaceKnowledge.ts frontend/src/types/workspaceWiki.ts frontend/wailsjs/go/main/App.d.ts frontend/wailsjs/go/main/App.js frontend/wailsjs/go/models.ts
git commit -m "feat: expose workspace knowledge source status"
```

## Self-Review

### Spec coverage

- `sources / inputs / state / wiki` filesystem split is implemented by Task 1.
- explicit source processing state and compile summary are implemented by Task 2.
- wiki-first query precedence and `wiki/index.md` / `wiki/log.md` generation are implemented by Task 3.
- source/build status visibility through Wails and TypeScript contracts is implemented by Task 4.

### Placeholder scan

- No `TODO`, `TBD`, or unspecified “add tests later” steps remain.
- Every code-writing step includes concrete paths, signatures, and representative code.
- Every verification step includes an exact command and expected result.

### Type consistency

- Source fields are consistently named `sourcePath`, `markitdownPath`, `markitdownStatus`, and `extractStatus` across Go and TypeScript.
- Compile summary is consistently named `WorkspaceKnowledgeCompileSummary` in Go and TypeScript.
- On-disk paths consistently use `inputs/markitdown`, `state/by-source`, and `state/jobs`.
