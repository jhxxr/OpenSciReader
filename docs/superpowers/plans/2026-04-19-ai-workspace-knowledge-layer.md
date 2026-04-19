# AI Workspace Knowledge Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the file-based workspace `raw / wiki / schema` knowledge layer, wire workspace scans to generate structured memory and wiki pages, and replace the reader AI sidebar with a knowledge-grounded copilot that can promote candidate memories.

**Architecture:** Reuse the existing workspace wiki job pipeline as the orchestration shell, but move the actual source of truth to filesystem-backed workspace knowledge artifacts under each workspace directory. Keep scan jobs and wiki page metadata in the existing SQLite-backed config store where it helps the UI, and add a new query/promotion backend that reads aggregated schema, wiki pages, and raw extracts in that order.

**Tech Stack:** Go, Wails v2, SQLite for job/page metadata, filesystem JSON/Markdown artifacts, React 18, TypeScript, Vite, MarkItDown.

---

## File Map

### Backend files

- Create: `workspace_knowledge_types.go`
  Defines file-backed knowledge structs: source records, source refs, entities, claims, relations, tasks, by-source payloads, query results, promotion payloads.
- Create: `workspace_knowledge_files.go`
  Owns workspace knowledge paths plus JSON/Markdown read-write helpers for `raw`, `wiki`, and `schema`.
- Create: `workspace_knowledge_compile.go`
  Merges `schema/by-source/*.json` into aggregated schema files and regenerates wiki outputs.
- Create: `workspace_knowledge_query.go`
  Implements retrieval, answer context assembly, candidate promotion, and conversation log append.
- Create: `workspace_knowledge_prompts.go`
  Centralizes JSON distillation prompts, wiki prompts, and query prompts so scan/query logic stays small.
- Create: `workspace_knowledge_files_test.go`
  Verifies layout creation plus JSON round-trips.
- Create: `workspace_knowledge_compile_test.go`
  Verifies aggregate compile output and wiki generation.
- Create: `workspace_knowledge_scan_test.go`
  Verifies the scan runner writes raw extracts, by-source schema files, and compiled outputs using fake dependencies.
- Create: `workspace_knowledge_query_test.go`
  Verifies retrieval order, evidence selection, and promotion write-back.
- Create: `workspace_knowledge_e2e_test.go`
  Covers the end-to-end scan -> compile -> query -> promote flow with mocked LLM output.

### Modified backend files

- Modify: `config_types.go`
  Add new Wails-bound types for knowledge query input/result, evidence hits, candidate memories, and promotion requests.
- Modify: `app.go`
  Expose new query/list/promote methods for the frontend and initialize any new services.
- Modify: `gateway_service.go`
  Add JSON-returning helper methods for source distillation and grounded query answers.
- Modify: `workspace_wiki_service.go`
  Reuse the existing scan job orchestration, but make it write `raw`, `schema/by-source`, aggregated schema, and wiki outputs instead of wiki-only markdown generation.

### Frontend files

- Create: `frontend/src/types/workspaceKnowledge.ts`
  Defines TypeScript contracts for knowledge entities, claims, tasks, evidence hits, query results, and promotion payloads.
- Create: `frontend/src/api/workspaceKnowledge.ts`
  Frontend facade that wraps existing scan/page methods plus the new list/query/promote methods.
- Create: `frontend/src/components/KnowledgeCopilotPanel.tsx`
  New reader sidebar panel with ask/answer/evidence/promote workflow.

### Modified frontend files

- Modify: `frontend/src/App.tsx`
  Track workspace-level knowledge state and pass memory/query props into workspace and reader views.
- Modify: `frontend/src/components/WorkspaceTab.tsx`
  Add memory browsing sections for entities, claims, tasks, and scan controls backed by the new API.
- Modify: `frontend/src/components/ReaderTab.tsx`
  Replace `ReaderAIPanel` with `KnowledgeCopilotPanel` and route sidebar state into the new query flow.
- Modify: `frontend/src/App.css`
  Style memory browser panels, evidence chips, and copilot layout.
- Delete: `frontend/src/components/ReaderAIPanel.tsx`
  Retire the legacy prompt-workbench implementation once the new copilot is wired in.

### Generated binding files

- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.d.ts`
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.js`
- Modify via `wails generate module`: `frontend/wailsjs/go/models.ts`

## Task 1: Add File-Backed Knowledge Contracts And Helpers

**Files:**
- Create: `workspace_knowledge_types.go`
- Create: `workspace_knowledge_files.go`
- Create: `workspace_knowledge_files_test.go`
- Modify: `config_types.go`

- [ ] **Step 1: Write the failing filesystem round-trip test**

```go
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
		ID:          "source:paper-a",
		WorkspaceID: workspace.ID,
		Title:       "Paper A",
		Slug:        "paper-a",
		Kind:        "pdf",
		AbsolutePath:"C:/papers/paper-a.pdf",
		ContentHash: "hash-a",
		ExtractPath: files.ExtractPath("paper-a"),
		Status:      "ready",
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./... -run TestWorkspaceKnowledgeFilesRoundTrip -count=1`

Expected: FAIL with compile errors such as `undefined: newWorkspaceKnowledgeFiles` and `undefined: WorkspaceKnowledgeSource`

- [ ] **Step 3: Write the minimal knowledge types and file helpers**

```go
// workspace_knowledge_types.go
package main

type WorkspaceKnowledgeSource struct {
	ID           string `json:"id"`
	WorkspaceID  string `json:"workspaceId"`
	Title        string `json:"title"`
	Slug         string `json:"slug"`
	Kind         string `json:"kind"`
	AbsolutePath string `json:"absolutePath"`
	ContentHash  string `json:"contentHash"`
	ExtractPath  string `json:"extractPath"`
	DocumentID   string `json:"documentId"`
	Status       string `json:"status"`
	LastScanAt   string `json:"lastScanAt"`
	LastError    string `json:"lastError"`
}

type WorkspaceKnowledgeSourceRef struct {
	SourceID  string `json:"sourceId"`
	PageStart int    `json:"pageStart"`
	PageEnd   int    `json:"pageEnd"`
	Excerpt   string `json:"excerpt"`
}

type WorkspaceKnowledgeEntity struct {
	ID          string                        `json:"id"`
	WorkspaceID string                        `json:"workspaceId"`
	Title       string                        `json:"title"`
	Type        string                        `json:"type"`
	Summary     string                        `json:"summary"`
	Aliases     []string                      `json:"aliases"`
	SourceRefs  []WorkspaceKnowledgeSourceRef `json:"sourceRefs"`
	Origin      string                        `json:"origin"`
	Status      string                        `json:"status"`
	Confidence  float64                       `json:"confidence"`
	CreatedAt   string                        `json:"createdAt"`
	UpdatedAt   string                        `json:"updatedAt"`
}

type WorkspaceKnowledgeClaim struct {
	ID          string                        `json:"id"`
	WorkspaceID string                        `json:"workspaceId"`
	Title       string                        `json:"title"`
	Type        string                        `json:"type"`
	Summary     string                        `json:"summary"`
	EntityIDs   []string                      `json:"entityIds"`
	SourceRefs  []WorkspaceKnowledgeSourceRef `json:"sourceRefs"`
	Origin      string                        `json:"origin"`
	Status      string                        `json:"status"`
	Confidence  float64                       `json:"confidence"`
	CreatedAt   string                        `json:"createdAt"`
	UpdatedAt   string                        `json:"updatedAt"`
}

type WorkspaceKnowledgeRelation struct {
	ID          string                        `json:"id"`
	WorkspaceID string                        `json:"workspaceId"`
	Type        string                        `json:"type"`
	FromID      string                        `json:"fromId"`
	ToID        string                        `json:"toId"`
	Summary     string                        `json:"summary"`
	SourceRefs  []WorkspaceKnowledgeSourceRef `json:"sourceRefs"`
	Origin      string                        `json:"origin"`
	Status      string                        `json:"status"`
	Confidence  float64                       `json:"confidence"`
	CreatedAt   string                        `json:"createdAt"`
	UpdatedAt   string                        `json:"updatedAt"`
}

type WorkspaceKnowledgeTask struct {
	ID          string                        `json:"id"`
	WorkspaceID string                        `json:"workspaceId"`
	Title       string                        `json:"title"`
	Type        string                        `json:"type"`
	Summary     string                        `json:"summary"`
	Priority    string                        `json:"priority"`
	SourceRefs  []WorkspaceKnowledgeSourceRef `json:"sourceRefs"`
	Origin      string                        `json:"origin"`
	Status      string                        `json:"status"`
	Confidence  float64                       `json:"confidence"`
	CreatedAt   string                        `json:"createdAt"`
	UpdatedAt   string                        `json:"updatedAt"`
}

type WorkspaceKnowledgeBySourcePayload struct {
	Source    WorkspaceKnowledgeSource     `json:"source"`
	Entities  []WorkspaceKnowledgeEntity   `json:"entities"`
	Claims    []WorkspaceKnowledgeClaim    `json:"claims"`
	Relations []WorkspaceKnowledgeRelation `json:"relations"`
	Tasks     []WorkspaceKnowledgeTask     `json:"tasks"`
}

type WorkspaceKnowledgeScanRunRecord struct {
	JobID       string   `json:"jobId"`
	WorkspaceID string   `json:"workspaceId"`
	Status      string   `json:"status"`
	Sources     []string `json:"sources"`
	Errors      []string `json:"errors"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}
```

```go
// workspace_knowledge_files.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type workspaceKnowledgeFiles struct {
	workspaceID string
	workspaceDir string
	rawDir string
	extractsDir string
	wikiDir string
	docsDir string
	conceptsDir string
	schemaDir string
	bySourceDir string
	scanRunsDir string
}

func newWorkspaceKnowledgeFiles(paths appPaths, workspaceID string) workspaceKnowledgeFiles {
	workspaceDir := filepath.Join(paths.WorkspacesRootDir, workspaceID)
	return workspaceKnowledgeFiles{
		workspaceID:  workspaceID,
		workspaceDir: workspaceDir,
		rawDir:       filepath.Join(workspaceDir, "raw"),
		extractsDir:  filepath.Join(workspaceDir, "raw", "extracts"),
		wikiDir:      filepath.Join(workspaceDir, "wiki"),
		docsDir:      filepath.Join(workspaceDir, "wiki", "docs"),
		conceptsDir:  filepath.Join(workspaceDir, "wiki", "concepts"),
		schemaDir:    filepath.Join(workspaceDir, "schema"),
		bySourceDir:  filepath.Join(workspaceDir, "schema", "by-source"),
		scanRunsDir:  filepath.Join(workspaceDir, "schema", "scan-runs"),
	}
}

func (f workspaceKnowledgeFiles) EnsureLayout() error {
	for _, dir := range []string{
		f.rawDir,
		f.extractsDir,
		f.wikiDir,
		f.docsDir,
		f.conceptsDir,
		f.schemaDir,
		f.bySourceDir,
		f.scanRunsDir,
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create knowledge directory %s: %w", dir, err)
		}
	}
	return nil
}

func (f workspaceKnowledgeFiles) SourcesPath() string             { return filepath.Join(f.rawDir, "sources.json") }
func (f workspaceKnowledgeFiles) ExtractPath(slug string) string  { return filepath.Join(f.extractsDir, slug+".md") }
func (f workspaceKnowledgeFiles) BySourcePath(slug string) string { return filepath.Join(f.bySourceDir, slug+".json") }
func (f workspaceKnowledgeFiles) ScanRunPath(jobID string) string { return filepath.Join(f.scanRunsDir, jobID+".json") }

func (f workspaceKnowledgeFiles) WriteSources(sources []WorkspaceKnowledgeSource) error {
	return writeWorkspaceKnowledgeJSON(f.SourcesPath(), sources)
}

func (f workspaceKnowledgeFiles) ReadSources() ([]WorkspaceKnowledgeSource, error) {
	var sources []WorkspaceKnowledgeSource
	err := readWorkspaceKnowledgeJSON(f.SourcesPath(), &sources)
	return sources, err
}

func (f workspaceKnowledgeFiles) WriteBySource(slug string, payload WorkspaceKnowledgeBySourcePayload) error {
	return writeWorkspaceKnowledgeJSON(f.BySourcePath(slug), payload)
}

func (f workspaceKnowledgeFiles) ReadBySource(slug string) (WorkspaceKnowledgeBySourcePayload, error) {
	var payload WorkspaceKnowledgeBySourcePayload
	err := readWorkspaceKnowledgeJSON(f.BySourcePath(slug), &payload)
	return payload, err
}

func (f workspaceKnowledgeFiles) WriteScanRun(jobID string, record WorkspaceKnowledgeScanRunRecord) error {
	return writeWorkspaceKnowledgeJSON(f.ScanRunPath(jobID), record)
}

func writeWorkspaceKnowledgeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json %s: %w", path, err)
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func readWorkspaceKnowledgeJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read json %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode json %s: %w", path, err)
	}
	return nil
}
```

```go
// config_types.go
type WorkspaceKnowledgeEvidenceHit struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Kind        string `json:"kind"`
	SourceID    string `json:"sourceId"`
	PageStart   int    `json:"pageStart"`
	PageEnd     int    `json:"pageEnd"`
	Excerpt     string `json:"excerpt"`
	MarkdownPath string `json:"markdownPath"`
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./... -run TestWorkspaceKnowledgeFilesRoundTrip -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add config_types.go workspace_knowledge_types.go workspace_knowledge_files.go workspace_knowledge_files_test.go
git commit -m "feat: add workspace knowledge file contracts"
```

## Task 2: Compile Aggregated Schema And Wiki Outputs

**Files:**
- Create: `workspace_knowledge_compile.go`
- Create: `workspace_knowledge_compile_test.go`
- Modify: `workspace_knowledge_files.go`

- [ ] **Step 1: Write the failing compile test**

```go
func TestCompileWorkspaceKnowledgeBuildsAggregatesAndWiki(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	files := workspaceKnowledgeFiles{
		workspaceID:  "workspace-a",
		workspaceDir: filepath.Join(tempDir, "workspace-a"),
		rawDir:       filepath.Join(tempDir, "workspace-a", "raw"),
		extractsDir:  filepath.Join(tempDir, "workspace-a", "raw", "extracts"),
		wikiDir:      filepath.Join(tempDir, "workspace-a", "wiki"),
		docsDir:      filepath.Join(tempDir, "workspace-a", "wiki", "docs"),
		conceptsDir:  filepath.Join(tempDir, "workspace-a", "wiki", "concepts"),
		schemaDir:    filepath.Join(tempDir, "workspace-a", "schema"),
		bySourceDir:  filepath.Join(tempDir, "workspace-a", "schema", "by-source"),
		scanRunsDir:  filepath.Join(tempDir, "workspace-a", "schema", "scan-runs"),
	}
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	now := nowRFC3339()
	payloadA := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{ID: "source:paper-a", WorkspaceID: "workspace-a", Title: "Paper A", Slug: "paper-a", Kind: "pdf"},
		Entities: []WorkspaceKnowledgeEntity{{
			ID: "entity:method:contrastive-memory",
			WorkspaceID: "workspace-a",
			Title: "Contrastive Memory",
			Type: "method",
			Summary: "Shared method summary",
			Aliases: []string{"CM"},
			SourceRefs: []WorkspaceKnowledgeSourceRef{{SourceID: "source:paper-a", PageStart: 3, PageEnd: 3, Excerpt: "Contrastive Memory excerpt"}},
			Origin: "scan",
			Status: "confirmed",
			Confidence: 0.9,
			CreatedAt: now,
			UpdatedAt: now,
		}},
		Claims: []WorkspaceKnowledgeClaim{{
			ID: "claim:paper-a-result",
			WorkspaceID: "workspace-a",
			Title: "Paper A improves retrieval accuracy",
			Type: "result",
			Summary: "Accuracy improves on the benchmark",
			EntityIDs: []string{"entity:method:contrastive-memory"},
			SourceRefs: []WorkspaceKnowledgeSourceRef{{SourceID: "source:paper-a", PageStart: 5, PageEnd: 5, Excerpt: "Accuracy improves"}},
			Origin: "scan",
			Status: "confirmed",
			Confidence: 0.85,
			CreatedAt: now,
			UpdatedAt: now,
		}},
	}
	payloadB := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{ID: "source:paper-b", WorkspaceID: "workspace-a", Title: "Paper B", Slug: "paper-b", Kind: "markdown"},
		Tasks: []WorkspaceKnowledgeTask{{
			ID: "task:compare-paper-a-paper-b",
			WorkspaceID: "workspace-a",
			Title: "Compare Paper A and Paper B",
			Type: "open_question",
			Summary: "Need to verify transferability claims",
			Priority: "medium",
			SourceRefs: []WorkspaceKnowledgeSourceRef{{SourceID: "source:paper-b", PageStart: 2, PageEnd: 2, Excerpt: "Need a comparison"}},
			Origin: "scan",
			Status: "candidate",
			Confidence: 0.6,
			CreatedAt: now,
			UpdatedAt: now,
		}},
	}

	if err := files.WriteBySource("paper-a", payloadA); err != nil {
		t.Fatalf("WriteBySource paper-a error: %v", err)
	}
	if err := files.WriteBySource("paper-b", payloadB); err != nil {
		t.Fatalf("WriteBySource paper-b error: %v", err)
	}

	snapshot, err := CompileWorkspaceKnowledge(files, "Knowledge Workspace")
	if err != nil {
		t.Fatalf("CompileWorkspaceKnowledge error: %v", err)
	}

	if len(snapshot.Entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(snapshot.Entities))
	}
	if len(snapshot.Claims) != 1 {
		t.Fatalf("claims = %d, want 1", len(snapshot.Claims))
	}
	if len(snapshot.Tasks) != 1 {
		t.Fatalf("tasks = %d, want 1", len(snapshot.Tasks))
	}

	overview, err := os.ReadFile(filepath.Join(files.wikiDir, "overview.md"))
	if err != nil {
		t.Fatalf("read overview.md error: %v", err)
	}
	if !strings.Contains(string(overview), "Knowledge Workspace") {
		t.Fatalf("overview.md = %q, want workspace title", string(overview))
	}

	conceptPage, err := os.ReadFile(filepath.Join(files.conceptsDir, "contrastive-memory.md"))
	if err != nil {
		t.Fatalf("read concept page error: %v", err)
	}
	if !strings.Contains(string(conceptPage), "Contrastive Memory") {
		t.Fatalf("concept page = %q, want concept title", string(conceptPage))
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./... -run TestCompileWorkspaceKnowledgeBuildsAggregatesAndWiki -count=1`

Expected: FAIL with `undefined: CompileWorkspaceKnowledge`

- [ ] **Step 3: Write the minimal compiler**

```go
// workspace_knowledge_compile.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type WorkspaceKnowledgeSnapshot struct {
	Entities  []WorkspaceKnowledgeEntity   `json:"entities"`
	Claims    []WorkspaceKnowledgeClaim    `json:"claims"`
	Relations []WorkspaceKnowledgeRelation `json:"relations"`
	Tasks     []WorkspaceKnowledgeTask     `json:"tasks"`
}

func CompileWorkspaceKnowledge(files workspaceKnowledgeFiles, workspaceTitle string) (WorkspaceKnowledgeSnapshot, error) {
	entries, err := os.ReadDir(files.bySourceDir)
	if err != nil {
		return WorkspaceKnowledgeSnapshot{}, fmt.Errorf("read by-source dir: %w", err)
	}

	entityByID := map[string]WorkspaceKnowledgeEntity{}
	claimByID := map[string]WorkspaceKnowledgeClaim{}
	relationByID := map[string]WorkspaceKnowledgeRelation{}
	taskByID := map[string]WorkspaceKnowledgeTask{}
	docPages := map[string]string{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		payload, err := files.ReadBySource(slug)
		if err != nil {
			return WorkspaceKnowledgeSnapshot{}, err
		}
		for _, entity := range payload.Entities {
			entityByID[entity.ID] = entity
		}
		for _, claim := range payload.Claims {
			claimByID[claim.ID] = claim
		}
		for _, relation := range payload.Relations {
			relationByID[relation.ID] = relation
		}
		for _, task := range payload.Tasks {
			taskByID[task.ID] = task
		}
		docPages[payload.Source.Slug] = buildDocumentWikiPage(payload)
	}

	snapshot := WorkspaceKnowledgeSnapshot{
		Entities:  mapEntities(entityByID),
		Claims:    mapClaims(claimByID),
		Relations: mapRelations(relationByID),
		Tasks:     mapTasks(taskByID),
	}

	if err := writeWorkspaceKnowledgeJSON(filepath.Join(files.schemaDir, "entities.json"), snapshot.Entities); err != nil {
		return WorkspaceKnowledgeSnapshot{}, err
	}
	if err := writeWorkspaceKnowledgeJSON(filepath.Join(files.schemaDir, "claims.json"), snapshot.Claims); err != nil {
		return WorkspaceKnowledgeSnapshot{}, err
	}
	if err := writeWorkspaceKnowledgeJSON(filepath.Join(files.schemaDir, "relations.json"), snapshot.Relations); err != nil {
		return WorkspaceKnowledgeSnapshot{}, err
	}
	if err := writeWorkspaceKnowledgeJSON(filepath.Join(files.schemaDir, "tasks.json"), snapshot.Tasks); err != nil {
		return WorkspaceKnowledgeSnapshot{}, err
	}

	for slug, markdown := range docPages {
		if err := os.WriteFile(filepath.Join(files.docsDir, slug+".md"), []byte(strings.TrimSpace(markdown)+"\n"), 0o600); err != nil {
			return WorkspaceKnowledgeSnapshot{}, fmt.Errorf("write doc wiki page %s: %w", slug, err)
		}
	}

	for _, entity := range snapshot.Entities {
		conceptMarkdown := buildConceptWikiPage(entity, snapshot.Claims)
		if err := os.WriteFile(filepath.Join(files.conceptsDir, workspaceWikiSlug(entity.Title)+".md"), []byte(strings.TrimSpace(conceptMarkdown)+"\n"), 0o600); err != nil {
			return WorkspaceKnowledgeSnapshot{}, fmt.Errorf("write concept wiki page %s: %w", entity.Title, err)
		}
	}

	if err := os.WriteFile(filepath.Join(files.wikiDir, "overview.md"), []byte(strings.TrimSpace(buildOverviewWikiPage(workspaceTitle, snapshot))+"\n"), 0o600); err != nil {
		return WorkspaceKnowledgeSnapshot{}, fmt.Errorf("write overview.md: %w", err)
	}
	if err := os.WriteFile(filepath.Join(files.wikiDir, "open-questions.md"), []byte(strings.TrimSpace(buildOpenQuestionsPage(snapshot.Tasks))+"\n"), 0o600); err != nil {
		return WorkspaceKnowledgeSnapshot{}, fmt.Errorf("write open-questions.md: %w", err)
	}

	return snapshot, nil
}

func mapEntities(input map[string]WorkspaceKnowledgeEntity) []WorkspaceKnowledgeEntity {
	out := make([]WorkspaceKnowledgeEntity, 0, len(input))
	for _, entity := range input {
		out = append(out, entity)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

func mapClaims(input map[string]WorkspaceKnowledgeClaim) []WorkspaceKnowledgeClaim {
	out := make([]WorkspaceKnowledgeClaim, 0, len(input))
	for _, claim := range input {
		out = append(out, claim)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

func mapRelations(input map[string]WorkspaceKnowledgeRelation) []WorkspaceKnowledgeRelation {
	out := make([]WorkspaceKnowledgeRelation, 0, len(input))
	for _, relation := range input {
		out = append(out, relation)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func mapTasks(input map[string]WorkspaceKnowledgeTask) []WorkspaceKnowledgeTask {
	out := make([]WorkspaceKnowledgeTask, 0, len(input))
	for _, task := range input {
		out = append(out, task)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}
```

```go
func buildDocumentWikiPage(payload WorkspaceKnowledgeBySourcePayload) string {
	return fmt.Sprintf(`# %s

## Summary
%s

## Claims
%s
`, payload.Source.Title, joinClaimSummaries(payload.Claims), joinClaimTitles(payload.Claims))
}

func buildConceptWikiPage(entity WorkspaceKnowledgeEntity, claims []WorkspaceKnowledgeClaim) string {
	return fmt.Sprintf(`# %s

## Definition
%s

## Related Claims
%s
`, entity.Title, entity.Summary, renderConceptClaims(entity.ID, claims))
}

func buildOverviewWikiPage(workspaceTitle string, snapshot WorkspaceKnowledgeSnapshot) string {
	return fmt.Sprintf(`# %s Overview

## Themes
- Entities: %d
- Claims: %d
- Tasks: %d
`, workspaceTitle, len(snapshot.Entities), len(snapshot.Claims), len(snapshot.Tasks))
}

func buildOpenQuestionsPage(tasks []WorkspaceKnowledgeTask) string {
	if len(tasks) == 0 {
		return "# Open Questions\n\nNo open questions."
	}
	lines := []string{"# Open Questions", ""}
	for _, task := range tasks {
		lines = append(lines, "- "+task.Title+": "+task.Summary)
	}
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./... -run TestCompileWorkspaceKnowledgeBuildsAggregatesAndWiki -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add workspace_knowledge_compile.go workspace_knowledge_compile_test.go workspace_knowledge_files.go
git commit -m "feat: compile workspace knowledge into wiki outputs"
```

## Task 3: Extend The Scan Runner To Produce Raw, By-Source, And Compiled Knowledge

**Files:**
- Create: `workspace_knowledge_prompts.go`
- Create: `workspace_knowledge_scan_test.go`
- Modify: `config_types.go`
- Modify: `workspace_wiki_service.go`
- Modify: `gateway_service.go`
- Modify: `app.go`

- [ ] **Step 1: Write the failing scan runner test**

```go
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
					ID: "entity:method:contrastive-memory",
					WorkspaceID: workspace.ID,
					Title: "Contrastive Memory",
					Type: "method",
					Summary: "A memory-augmented retrieval method",
					Origin: "scan",
					Status: "confirmed",
					Confidence: 0.9,
					CreatedAt: nowRFC3339(),
					UpdatedAt: nowRFC3339(),
				}},
			},
			docWiki: "# Paper A\n\n## Summary\nA memory-augmented retrieval paper.",
			overviewWiki: "# Scan Workspace Overview\n\n## Themes\n- Contrastive Memory",
		},
	}

	job, err := store.SaveWorkspaceWikiScanJob(t.Context(), WorkspaceWikiScanJob{
		WorkspaceID:     workspace.ID,
		Status:          WorkspaceWikiScanJobQueued,
		CurrentStage:    "queued",
		Message:         "queued",
		ProviderID:      1,
		ModelID:         2,
		StartedAt:       nowRFC3339(),
		UpdatedAt:       nowRFC3339(),
	})
	if err != nil {
		t.Fatalf("SaveWorkspaceWikiScanJob error: %v", err)
	}

	runner.run(context.Background(), job)

	files := newWorkspaceKnowledgeFiles(paths, workspace.ID)
	if _, err := os.Stat(files.ExtractPath("paper")); err != nil {
		t.Fatalf("stat extract error: %v", err)
	}
	if _, err := os.Stat(files.BySourcePath("paper")); err != nil {
		t.Fatalf("stat by-source error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(files.wikiDir, "overview.md")); err != nil {
		t.Fatalf("stat overview error: %v", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./... -run TestWorkspaceWikiScanRunnerWritesKnowledgeArtifacts -count=1`

Expected: FAIL with compile errors for `stubWorkspaceKnowledgeExtractor`, `stubWorkspaceKnowledgeLLM`, or missing runner fields

- [ ] **Step 3: Refactor the scan runner around extract -> distill -> compile**

```go
// workspace_wiki_service.go
type WorkspaceWikiScanStartInput struct {
	WorkspaceID string `json:"workspaceId"`
	DocumentID  string `json:"documentId"`
	ProviderID  int64  `json:"providerId"`
	ModelID     int64  `json:"modelId"`
}

type workspaceKnowledgeExtractor interface {
	ExtractMarkdown(ctx context.Context, path string) (PDFMarkdownPayload, error)
}

type workspaceKnowledgeLLM interface {
	GenerateWorkspaceKnowledgeBySource(ctx context.Context, providerID, modelID int64, prompt string) (WorkspaceKnowledgeBySourcePayload, error)
	GenerateWorkspaceKnowledgeMarkdown(ctx context.Context, providerID, modelID int64, prompt string) (string, error)
}

type workspaceWikiScanRunner struct {
	app          *App
	paths        appPaths
	store        *configStore
	pdf          workspaceKnowledgeExtractor
	knowledgeLLM workspaceKnowledgeLLM
	mu           sync.Mutex
	jobs         map[string]context.CancelFunc
}

func newWorkspaceWikiScanRunner(app *App) *workspaceWikiScanRunner {
	return &workspaceWikiScanRunner{
		app:          app,
		paths:        app.paths,
		store:        app.store,
		pdf:          app.pdf,
		knowledgeLLM: app.gateway,
		jobs:         make(map[string]context.CancelFunc),
	}
}

func (r *workspaceWikiScanRunner) Start(ctx context.Context, input WorkspaceWikiScanStartInput) (WorkspaceWikiScanJob, error) {
	if strings.TrimSpace(input.WorkspaceID) == "" {
		return WorkspaceWikiScanJob{}, fmt.Errorf("workspace id is required")
	}
	job, err := r.store.SaveWorkspaceWikiScanJob(ctx, WorkspaceWikiScanJob{
		WorkspaceID:     input.WorkspaceID,
		Status:          WorkspaceWikiScanJobQueued,
		CurrentStage:    "queued",
		Message:         "Preparing workspace knowledge scan",
		ProviderID:      input.ProviderID,
		ModelID:         input.ModelID,
		StartedAt:       nowRFC3339(),
		UpdatedAt:       nowRFC3339(),
	})
	if err != nil {
		return WorkspaceWikiScanJob{}, err
	}
	go func() {
		workspace, _ := r.store.GetWorkspace(context.Background(), input.WorkspaceID)
		files := newWorkspaceKnowledgeFiles(r.paths, input.WorkspaceID)
		_ = files.EnsureLayout()
		sources, _ := r.collectSources(context.Background(), input.WorkspaceID)
		sources = filterWorkspaceSourcesByDocument(sources, input.DocumentID)
		for _, source := range sources {
			_ = r.processSource(context.Background(), files, workspace, job, source)
		}
		_ = r.finalizeKnowledgeRun(context.Background(), files, workspace, job)
	}()
	return job, nil
}

func filterWorkspaceSourcesByDocument(sources []WorkspaceWikiScanSource, documentID string) []WorkspaceWikiScanSource {
	if strings.TrimSpace(documentID) == "" {
		return sources
	}
	filtered := make([]WorkspaceWikiScanSource, 0, len(sources))
	for _, source := range sources {
		if source.DocumentID == documentID {
			filtered = append(filtered, source)
		}
	}
	return filtered
}

func (r *workspaceWikiScanRunner) processSource(ctx context.Context, files workspaceKnowledgeFiles, workspace Workspace, job WorkspaceWikiScanJob, source WorkspaceWikiScanSource) error {
	text, err := r.extractSourceText(ctx, source)
	if err != nil {
		return err
	}

	extractPath := files.ExtractPath(workspaceWikiSlug(source.Title))
	if err := os.WriteFile(extractPath, []byte(strings.TrimSpace(text)+"\n"), 0o600); err != nil {
		return fmt.Errorf("write extract %s: %w", extractPath, err)
	}

	bySource, err := r.knowledgeLLM.GenerateWorkspaceKnowledgeBySource(ctx, job.ProviderID, job.ModelID, buildBySourceKnowledgePrompt(workspace, source, text))
	if err != nil {
		return err
	}
	bySource.Source = WorkspaceKnowledgeSource{
		ID:           buildWorkspaceKnowledgeSourceID(workspace.ID, source.Title),
		WorkspaceID:  workspace.ID,
		Title:        source.Title,
		Slug:         workspaceWikiSlug(source.Title),
		Kind:         source.SourceType,
		AbsolutePath: source.AbsolutePath,
		ContentHash:  source.Key,
		ExtractPath:  extractPath,
		DocumentID:   source.DocumentID,
		Status:       "ready",
		LastScanAt:   nowRFC3339(),
	}
	if err := files.WriteBySource(bySource.Source.Slug, bySource); err != nil {
		return err
	}
	sources, _ := files.ReadSources()
	sources = append(sources, bySource.Source)
	if err := files.WriteSources(sources); err != nil {
		return err
	}
	if err := files.WriteScanRun(job.JobID, WorkspaceKnowledgeScanRunRecord{
		JobID:       job.JobID,
		WorkspaceID: workspace.ID,
		Status:      "running",
		Sources:     []string{bySource.Source.ID},
		Errors:      []string{},
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}); err != nil {
		return err
	}

	docMarkdown, err := r.knowledgeLLM.GenerateWorkspaceKnowledgeMarkdown(ctx, job.ProviderID, job.ModelID, buildDocumentWikiPrompt(workspace, source, text))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(files.docsDir, bySource.Source.Slug+".md"), []byte(strings.TrimSpace(docMarkdown)+"\n"), 0o600)
}

func (r *workspaceWikiScanRunner) finalizeKnowledgeRun(ctx context.Context, files workspaceKnowledgeFiles, workspace Workspace, job WorkspaceWikiScanJob) error {
	snapshot, err := CompileWorkspaceKnowledge(files, workspace.Name)
	if err != nil {
		return err
	}
	overviewMarkdown, err := r.knowledgeLLM.GenerateWorkspaceKnowledgeMarkdown(ctx, job.ProviderID, job.ModelID, buildOverviewWikiPrompt(workspace, buildKnowledgeSummaries(snapshot)))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(files.wikiDir, "overview.md"), []byte(strings.TrimSpace(overviewMarkdown)+"\n"), 0o600); err != nil {
		return err
	}
	return files.WriteScanRun(job.JobID, WorkspaceKnowledgeScanRunRecord{
		JobID:       job.JobID,
		WorkspaceID: workspace.ID,
		Status:      "completed",
		Sources:     []string{},
		Errors:      []string{},
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	})
}

func buildKnowledgeSummaries(snapshot WorkspaceKnowledgeSnapshot) []string {
	summaries := make([]string, 0, len(snapshot.Claims))
	for _, claim := range snapshot.Claims {
		summaries = append(summaries, fmt.Sprintf("## %s\n\n%s", claim.Title, claim.Summary))
	}
	return summaries
}

type stubWorkspaceKnowledgeExtractor struct {
	markdown string
}

func (s *stubWorkspaceKnowledgeExtractor) ExtractMarkdown(_ context.Context, _ string) (PDFMarkdownPayload, error) {
	return PDFMarkdownPayload{
		Markdown:    s.markdown,
		Source:      "stub",
		TotalChars:  len(s.markdown),
		GeneratedAt: nowRFC3339(),
	}, nil
}

type stubWorkspaceKnowledgeLLM struct {
	bySource    WorkspaceKnowledgeBySourcePayload
	docWiki     string
	overviewWiki string
	query       WorkspaceKnowledgeQueryResult
}

func (s *stubWorkspaceKnowledgeLLM) GenerateWorkspaceKnowledgeBySource(_ context.Context, _ int64, _ int64, _ string) (WorkspaceKnowledgeBySourcePayload, error) {
	return s.bySource, nil
}

func (s *stubWorkspaceKnowledgeLLM) GenerateWorkspaceKnowledgeMarkdown(_ context.Context, _ int64, _ int64, prompt string) (string, error) {
	if strings.Contains(prompt, "overview") {
		return s.overviewWiki, nil
	}
	return s.docWiki, nil
}

func (s *stubWorkspaceKnowledgeLLM) GenerateWorkspaceKnowledgeQuery(_ context.Context, _ int64, _ int64, _ string) (WorkspaceKnowledgeQueryResult, error) {
	return s.query, nil
}
```

```go
// workspace_knowledge_prompts.go
package main

import "fmt"

func buildBySourceKnowledgePrompt(workspace Workspace, source WorkspaceWikiScanSource, text string) string {
	return fmt.Sprintf(`Return JSON only.
Schema:
{
  "entities": [],
  "claims": [],
  "relations": [],
  "tasks": []
}

Workspace: %s
Source title: %s
Source type: %s
Source path: %s

Source content:
%s`, workspace.Name, source.Title, source.SourceType, source.AbsolutePath, trimWikiSourceText(text))
}
```

```go
// gateway_service.go
func (g *gatewayService) generateOpenAICompatibleText(ctx context.Context, providerID, modelID int64, systemPrompt, userPrompt string) (string, error) {
	provider, err := g.store.GetProviderSecret(ctx, providerID)
	if err != nil {
		return "", err
	}
	model, err := g.store.GetModel(ctx, modelID)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(openAIChatRequest{
		Model: model.ModelID,
		Messages: []map[string]any{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		Stream: false,
	})
	if err != nil {
		return "", err
	}

	resp, err := g.doRequestWith429Retry(ctx, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(provider.BaseURL, "/")+"/chat/completions", bytes.NewReader(payload))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		applyProviderRequestHeaders(req, provider.APIKey)
		return req, nil
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("gateway http error: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty llm response")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		content = strings.TrimSpace(parsed.Choices[0].Text)
	}
	if content == "" {
		return "", fmt.Errorf("empty llm content")
	}
	return content, nil
}

func (g *gatewayService) GenerateWorkspaceKnowledgeBySource(ctx context.Context, providerID, modelID int64, prompt string) (WorkspaceKnowledgeBySourcePayload, error) {
	content, err := g.generateOpenAICompatibleText(ctx, providerID, modelID, "You extract structured workspace knowledge. Return JSON only.", prompt)
	if err != nil {
		return WorkspaceKnowledgeBySourcePayload{}, err
	}
	var payload WorkspaceKnowledgeBySourcePayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return WorkspaceKnowledgeBySourcePayload{}, fmt.Errorf("decode workspace knowledge json: %w", err)
	}
	return payload, nil
}

func (g *gatewayService) GenerateWorkspaceKnowledgeMarkdown(ctx context.Context, providerID, modelID int64, prompt string) (string, error) {
	return g.generateOpenAICompatibleText(ctx, providerID, modelID, "You are OpenSciReader, a precise academic workspace wiki writer. Return markdown only.", prompt)
}
```

- [ ] **Step 4: Run the scan runner tests**

Run: `go test ./... -run "TestWorkspaceWikiScanRunnerWritesKnowledgeArtifacts|TestCompileWorkspaceKnowledgeBuildsAggregatesAndWiki" -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app.go gateway_service.go workspace_wiki_service.go workspace_knowledge_prompts.go workspace_knowledge_scan_test.go
git commit -m "feat: wire workspace scan into knowledge artifacts"
```

## Task 4: Add Query, Evidence Retrieval, Promotion, And Wails Bindings

**Files:**
- Create: `workspace_knowledge_query.go`
- Create: `workspace_knowledge_query_test.go`
- Modify: `config_types.go`
- Modify: `app.go`
- Modify: `gateway_service.go`

- [ ] **Step 1: Write the failing query and promotion test**

```go
func TestWorkspaceKnowledgeQueryPrefersSchemaAndPromotesCandidates(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	files := workspaceKnowledgeFiles{
		workspaceID:  "workspace-a",
		workspaceDir: filepath.Join(tempDir, "workspace-a"),
		rawDir:       filepath.Join(tempDir, "workspace-a", "raw"),
		extractsDir:  filepath.Join(tempDir, "workspace-a", "raw", "extracts"),
		wikiDir:      filepath.Join(tempDir, "workspace-a", "wiki"),
		docsDir:      filepath.Join(tempDir, "workspace-a", "wiki", "docs"),
		conceptsDir:  filepath.Join(tempDir, "workspace-a", "wiki", "concepts"),
		schemaDir:    filepath.Join(tempDir, "workspace-a", "schema"),
		bySourceDir:  filepath.Join(tempDir, "workspace-a", "schema", "by-source"),
		scanRunsDir:  filepath.Join(tempDir, "workspace-a", "schema", "scan-runs"),
	}
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	entities := []WorkspaceKnowledgeEntity{{
		ID: "entity:method:contrastive-memory",
		WorkspaceID: "workspace-a",
		Title: "Contrastive Memory",
		Type: "method",
		Summary: "Memory-augmented retrieval method",
		SourceRefs: []WorkspaceKnowledgeSourceRef{{SourceID: "source:paper-a", PageStart: 3, PageEnd: 3, Excerpt: "Contrastive Memory excerpt"}},
		Origin: "scan",
		Status: "confirmed",
		Confidence: 0.9,
		CreatedAt: nowRFC3339(),
		UpdatedAt: nowRFC3339(),
	}}
	if err := writeWorkspaceKnowledgeJSON(filepath.Join(files.schemaDir, "entities.json"), entities); err != nil {
		t.Fatalf("write entities error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(files.conceptsDir, "contrastive-memory.md"), []byte("# Contrastive Memory\n\n## Definition\nMemory-augmented retrieval method\n"), 0o600); err != nil {
		t.Fatalf("write concept page error: %v", err)
	}

	service := workspaceKnowledgeQueryService{
		files: files,
		llm: &stubWorkspaceKnowledgeLLM{
			query: WorkspaceKnowledgeQueryResult{
				Answer: "Contrastive Memory is the main method in the workspace.",
				Candidates: []WorkspaceKnowledgeCandidate{{
					ID: "candidate:claim:contrastive-memory-core",
					Title: "Contrastive Memory is the main method",
					Type: "claim",
					Summary: "The workspace centers on Contrastive Memory",
				}},
			},
		},
	}

	result, err := service.Query(context.Background(), WorkspaceKnowledgeQueryInput{
		WorkspaceID: "workspace-a",
		ProviderID:  1,
		ModelID:     2,
		Question:    "What is the main method?",
	})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if len(result.Evidence) == 0 || result.Evidence[0].ID != "entity:method:contrastive-memory" {
		t.Fatalf("evidence = %#v, want schema entity first", result.Evidence)
	}

	if err := service.Promote(context.Background(), WorkspaceKnowledgePromotionInput{
		WorkspaceID: "workspace-a",
		Candidates:  result.Candidates,
	}); err != nil {
		t.Fatalf("Promote error: %v", err)
	}

	claimsPath := filepath.Join(files.schemaDir, "claims.json")
	data, err := os.ReadFile(claimsPath)
	if err != nil {
		t.Fatalf("read claims.json error: %v", err)
	}
	if !strings.Contains(string(data), "Contrastive Memory is the main method") {
		t.Fatalf("claims.json = %q, want promoted claim", string(data))
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./... -run TestWorkspaceKnowledgeQueryPrefersSchemaAndPromotesCandidates -count=1`

Expected: FAIL with `undefined: workspaceKnowledgeQueryService` and missing query/promotion types

- [ ] **Step 3: Add backend query and promotion support**

```go
// config_types.go
type WorkspaceKnowledgeQueryInput struct {
	WorkspaceID string `json:"workspaceId"`
	DocumentID  string `json:"documentId"`
	ProviderID  int64  `json:"providerId"`
	ModelID     int64  `json:"modelId"`
	Page        int    `json:"page"`
	Selection   string `json:"selection"`
	Scope       string `json:"scope"`
	Question    string `json:"question"`
}

type WorkspaceKnowledgeCandidate struct {
	ID         string                        `json:"id"`
	Title      string                        `json:"title"`
	Type       string                        `json:"type"`
	Summary    string                        `json:"summary"`
	EntityIDs  []string                      `json:"entityIds"`
	SourceRefs []WorkspaceKnowledgeSourceRef `json:"sourceRefs"`
}

type WorkspaceKnowledgeQueryResult struct {
	Answer     string                          `json:"answer"`
	Evidence   []WorkspaceKnowledgeEvidenceHit `json:"evidence"`
	Candidates []WorkspaceKnowledgeCandidate   `json:"candidates"`
}

type WorkspaceKnowledgePromotionInput struct {
	WorkspaceID string                       `json:"workspaceId"`
	Candidates  []WorkspaceKnowledgeCandidate `json:"candidates"`
}
```

```go
// workspace_knowledge_query.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type workspaceKnowledgeQueryLLM interface {
	GenerateWorkspaceKnowledgeQuery(ctx context.Context, providerID, modelID int64, prompt string) (WorkspaceKnowledgeQueryResult, error)
}

type workspaceKnowledgeQueryService struct {
	files workspaceKnowledgeFiles
	llm   workspaceKnowledgeQueryLLM
}

func (s workspaceKnowledgeQueryService) Query(ctx context.Context, input WorkspaceKnowledgeQueryInput) (WorkspaceKnowledgeQueryResult, error) {
	entities, _ := readWorkspaceKnowledgeEntities(filepath.Join(s.files.schemaDir, "entities.json"))
	evidence := make([]WorkspaceKnowledgeEvidenceHit, 0, len(entities))
	for _, entity := range entities {
		evidence = append(evidence, WorkspaceKnowledgeEvidenceHit{
			ID:        entity.ID,
			Title:     entity.Title,
			Kind:      "entity",
			SourceID:  firstSourceID(entity.SourceRefs),
			PageStart: firstPageStart(entity.SourceRefs),
			PageEnd:   firstPageEnd(entity.SourceRefs),
			Excerpt:   firstExcerpt(entity.SourceRefs),
		})
	}
	if len(evidence) == 0 {
		wikiBytes, wikiErr := os.ReadFile(filepath.Join(s.files.conceptsDir, "contrastive-memory.md"))
		if wikiErr == nil {
			evidence = append(evidence, WorkspaceKnowledgeEvidenceHit{
				ID:          "wiki:contrastive-memory",
				Title:       "Contrastive Memory",
				Kind:        "wiki",
				SourceID:    "",
				PageStart:   0,
				PageEnd:     0,
				Excerpt:     string(wikiBytes),
				MarkdownPath: filepath.Join(s.files.conceptsDir, "contrastive-memory.md"),
			})
		}
	}
	if len(evidence) == 0 {
		rawBytes, rawErr := os.ReadFile(filepath.Join(s.files.extractsDir, "paper-a.md"))
		if rawErr == nil {
			evidence = append(evidence, WorkspaceKnowledgeEvidenceHit{
				ID:          "raw:paper-a",
				Title:       "paper-a",
				Kind:        "raw",
				SourceID:    "source:paper-a",
				PageStart:   0,
				PageEnd:     0,
				Excerpt:     string(rawBytes),
				MarkdownPath: filepath.Join(s.files.extractsDir, "paper-a.md"),
			})
		}
	}
	if len(evidence) == 0 {
		return WorkspaceKnowledgeQueryResult{
			Answer:     "Current workspace evidence is insufficient for a grounded answer.",
			Evidence:   []WorkspaceKnowledgeEvidenceHit{},
			Candidates: []WorkspaceKnowledgeCandidate{},
		}, nil
	}

	result, err := s.llm.GenerateWorkspaceKnowledgeQuery(ctx, input.ProviderID, input.ModelID, buildWorkspaceKnowledgeQueryPrompt(input, evidence))
	if err != nil {
		return WorkspaceKnowledgeQueryResult{}, err
	}
	result.Evidence = evidence
	if err := s.appendConversationEvent(input, result); err != nil {
		return WorkspaceKnowledgeQueryResult{}, err
	}
	return result, nil
}

func (s workspaceKnowledgeQueryService) Promote(ctx context.Context, input WorkspaceKnowledgePromotionInput) error {
	claimsPath := filepath.Join(s.files.schemaDir, "claims.json")
	existing, _ := readWorkspaceKnowledgeClaims(claimsPath)
	now := nowRFC3339()
	for _, candidate := range input.Candidates {
		if candidate.Type != "claim" {
			continue
		}
		existing = append(existing, WorkspaceKnowledgeClaim{
			ID:          candidate.ID,
			WorkspaceID: input.WorkspaceID,
			Title:       candidate.Title,
			Type:        candidate.Type,
			Summary:     candidate.Summary,
			EntityIDs:   candidate.EntityIDs,
			SourceRefs:  candidate.SourceRefs,
			Origin:      "chat",
			Status:      "confirmed",
			Confidence:  0.7,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	return writeWorkspaceKnowledgeJSON(claimsPath, existing)
}

func (s workspaceKnowledgeQueryService) appendConversationEvent(input WorkspaceKnowledgeQueryInput, result WorkspaceKnowledgeQueryResult) error {
	path := filepath.Join(s.files.schemaDir, "conversation-log.jsonl")
	event := map[string]any{
		"workspaceId": input.WorkspaceID,
		"question":    input.Question,
		"scope":       input.Scope,
		"evidence":    result.Evidence,
		"candidates":  result.Candidates,
		"answer":      result.Answer,
		"createdAt":   nowRFC3339(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal conversation event: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open conversation log: %w", err)
	}
	defer file.Close()
	_, err = file.WriteString(string(data) + "\n")
	return err
}

func buildWorkspaceKnowledgeQueryPrompt(input WorkspaceKnowledgeQueryInput, evidence []WorkspaceKnowledgeEvidenceHit) string {
	lines := []string{
		"Return JSON only.",
		fmt.Sprintf("Question: %s", input.Question),
		fmt.Sprintf("Scope: %s", input.Scope),
		"Evidence:",
	}
	for _, item := range evidence {
		lines = append(lines, fmt.Sprintf("- [%s] %s :: %s", item.Kind, item.Title, item.Excerpt))
	}
	return strings.Join(lines, "\n")
}

func firstSourceID(refs []WorkspaceKnowledgeSourceRef) string {
	if len(refs) == 0 {
		return ""
	}
	return refs[0].SourceID
}

func firstPageStart(refs []WorkspaceKnowledgeSourceRef) int {
	if len(refs) == 0 {
		return 0
	}
	return refs[0].PageStart
}

func firstPageEnd(refs []WorkspaceKnowledgeSourceRef) int {
	if len(refs) == 0 {
		return 0
	}
	return refs[0].PageEnd
}

func firstExcerpt(refs []WorkspaceKnowledgeSourceRef) string {
	if len(refs) == 0 {
		return ""
	}
	return refs[0].Excerpt
}
```

```go
// app.go
func (a *App) QueryWorkspaceKnowledge(input WorkspaceKnowledgeQueryInput) (WorkspaceKnowledgeQueryResult, error) {
	files := newWorkspaceKnowledgeFiles(a.paths, input.WorkspaceID)
	service := workspaceKnowledgeQueryService{files: files, llm: a.gateway}
	return service.Query(a.ctx, input)
}

func (a *App) PromoteWorkspaceKnowledge(input WorkspaceKnowledgePromotionInput) error {
	files := newWorkspaceKnowledgeFiles(a.paths, input.WorkspaceID)
	service := workspaceKnowledgeQueryService{files: files, llm: a.gateway}
	return service.Promote(a.ctx, input)
}

func (a *App) ListWorkspaceKnowledgeEntities(workspaceID string) ([]WorkspaceKnowledgeEntity, error) {
	files := newWorkspaceKnowledgeFiles(a.paths, workspaceID)
	return readWorkspaceKnowledgeEntities(filepath.Join(files.schemaDir, "entities.json"))
}

func (a *App) ListWorkspaceKnowledgeClaims(workspaceID string) ([]WorkspaceKnowledgeClaim, error) {
	files := newWorkspaceKnowledgeFiles(a.paths, workspaceID)
	return readWorkspaceKnowledgeClaims(filepath.Join(files.schemaDir, "claims.json"))
}

func (a *App) ListWorkspaceKnowledgeTasks(workspaceID string) ([]WorkspaceKnowledgeTask, error) {
	files := newWorkspaceKnowledgeFiles(a.paths, workspaceID)
	return readWorkspaceKnowledgeTasks(filepath.Join(files.schemaDir, "tasks.json"))
}
```

```go
// gateway_service.go
func (g *gatewayService) GenerateWorkspaceKnowledgeQuery(ctx context.Context, providerID, modelID int64, prompt string) (WorkspaceKnowledgeQueryResult, error) {
	content, err := g.generateOpenAICompatibleText(ctx, providerID, modelID, "You answer workspace questions using supplied evidence. Return JSON only with answer and candidates.", prompt)
	if err != nil {
		return WorkspaceKnowledgeQueryResult{}, err
	}
	var result WorkspaceKnowledgeQueryResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return WorkspaceKnowledgeQueryResult{}, fmt.Errorf("decode workspace knowledge query json: %w", err)
	}
	return result, nil
}
```

- [ ] **Step 4: Run the query tests**

Run: `go test ./... -run TestWorkspaceKnowledgeQueryPrefersSchemaAndPromotesCandidates -count=1`

Expected: PASS

- [ ] **Step 5: Regenerate the Wails bindings**

Run: `wails generate module`

Expected: `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/main/App.js`, and `frontend/wailsjs/go/models.ts` update with the new knowledge query/list/promote methods and types

- [ ] **Step 6: Commit**

```bash
git add app.go config_types.go gateway_service.go workspace_knowledge_query.go workspace_knowledge_query_test.go frontend/wailsjs/go/main/App.d.ts frontend/wailsjs/go/main/App.js frontend/wailsjs/go/models.ts
git commit -m "feat: add workspace knowledge query and promotion api"
```

## Task 5: Add Workspace Knowledge API And Memory Browser UI

**Files:**
- Create: `frontend/src/types/workspaceKnowledge.ts`
- Create: `frontend/src/api/workspaceKnowledge.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/WorkspaceTab.tsx`
- Modify: `frontend/src/App.css`

- [ ] **Step 1: Write the failing frontend contract change**

```tsx
// frontend/src/components/WorkspaceTab.tsx
interface WorkspaceTabProps {
  workspace: Workspace | null;
  documents: DocumentRecord[];
  isImporting: boolean;
  isLoadingDocuments: boolean;
  deletingDocumentId: string | null;
  llmProviderConfigs: ProviderConfig[];
  wikiPages: WorkspaceWikiPage[];
  selectedWikiPageId: string | null;
  wikiPageContent: WorkspaceWikiPageContent | null;
  isLoadingWikiPages: boolean;
  isLoadingWikiPageContent: boolean;
  activeWikiJob: WorkspaceWikiScanJob | null;
  wikiError: string | null;
  isStartingWikiScan: boolean;
  isCancellingWikiScan: boolean;
  isDeletingWikiPages: boolean;
  knowledgeEntities: WorkspaceKnowledgeEntity[];
  knowledgeClaims: WorkspaceKnowledgeClaim[];
  knowledgeTasks: WorkspaceKnowledgeTask[];
  isLoadingKnowledge: boolean;
  onRefreshKnowledge: () => Promise<void>;
}

// frontend/src/App.tsx
const [workspaceTabKnowledge, setWorkspaceTabKnowledge] = useState<Record<string, {
  entities: WorkspaceKnowledgeEntity[];
  claims: WorkspaceKnowledgeClaim[];
  tasks: WorkspaceKnowledgeTask[];
  isLoading: boolean;
}>>({});

const ensureWorkspaceTabKnowledge = useCallback(async (workspaceId: string) => {
  setWorkspaceTabKnowledge((current) => ({
    ...current,
    [workspaceId]: {
      entities: current[workspaceId]?.entities ?? [],
      claims: current[workspaceId]?.claims ?? [],
      tasks: current[workspaceId]?.tasks ?? [],
      isLoading: true,
    },
  }));

  const [entities, claims, tasks] = await Promise.all([
    workspaceKnowledgeApi.listEntities(workspaceId),
    workspaceKnowledgeApi.listClaims(workspaceId),
    workspaceKnowledgeApi.listTasks(workspaceId),
  ]);

  setWorkspaceTabKnowledge((current) => ({
    ...current,
    [workspaceId]: { entities, claims, tasks, isLoading: false },
  }));
}, []);
```

- [ ] **Step 2: Run the frontend build to verify it fails**

Run: `npm run build`

Workdir: `frontend`

Expected: FAIL with TypeScript errors for missing `workspaceKnowledgeApi` import, missing `WorkspaceKnowledgeEntity` types, and missing `WorkspaceTab` props

- [ ] **Step 3: Implement the frontend knowledge API and browser panels**

```ts
// frontend/src/types/workspaceKnowledge.ts
export interface WorkspaceKnowledgeSourceRef {
  sourceId: string;
  pageStart: number;
  pageEnd: number;
  excerpt: string;
}

export interface WorkspaceKnowledgeEntity {
  id: string;
  workspaceId: string;
  title: string;
  type: string;
  summary: string;
  aliases: string[];
  sourceRefs: WorkspaceKnowledgeSourceRef[];
  origin: string;
  status: string;
  confidence: number;
  createdAt: string;
  updatedAt: string;
}

export interface WorkspaceKnowledgeClaim {
  id: string;
  workspaceId: string;
  title: string;
  type: string;
  summary: string;
  entityIds: string[];
  sourceRefs: WorkspaceKnowledgeSourceRef[];
  origin: string;
  status: string;
  confidence: number;
  createdAt: string;
  updatedAt: string;
}

export interface WorkspaceKnowledgeTask {
  id: string;
  workspaceId: string;
  title: string;
  type: string;
  summary: string;
  priority: string;
  sourceRefs: WorkspaceKnowledgeSourceRef[];
  origin: string;
  status: string;
  confidence: number;
  createdAt: string;
  updatedAt: string;
}
```

```ts
// frontend/src/api/workspaceKnowledge.ts
import type {
  WorkspaceKnowledgeClaim,
  WorkspaceKnowledgeEntity,
  WorkspaceKnowledgeTask,
} from "../types/workspaceKnowledge";

interface WailsWorkspaceKnowledgeApp {
  ListWorkspaceKnowledgeEntities: (workspaceId: string) => Promise<WorkspaceKnowledgeEntity[]>;
  ListWorkspaceKnowledgeClaims: (workspaceId: string) => Promise<WorkspaceKnowledgeClaim[]>;
  ListWorkspaceKnowledgeTasks: (workspaceId: string) => Promise<WorkspaceKnowledgeTask[]>;
}

function getApp(): WailsWorkspaceKnowledgeApp | null {
  if (typeof window !== "undefined" && "go" in window && window.go?.main?.App) {
    return window.go.main.App as WailsWorkspaceKnowledgeApp;
  }
  return null;
}

export const workspaceKnowledgeApi = {
  async listEntities(workspaceId: string): Promise<WorkspaceKnowledgeEntity[]> {
    return (await getApp()?.ListWorkspaceKnowledgeEntities(workspaceId)) ?? [];
  },
  async listClaims(workspaceId: string): Promise<WorkspaceKnowledgeClaim[]> {
    return (await getApp()?.ListWorkspaceKnowledgeClaims(workspaceId)) ?? [];
  },
  async listTasks(workspaceId: string): Promise<WorkspaceKnowledgeTask[]> {
    return (await getApp()?.ListWorkspaceKnowledgeTasks(workspaceId)) ?? [];
  },
};
```

```tsx
// frontend/src/components/WorkspaceTab.tsx
<article className="workspace-panel">
  <div className="section-header workspace-panel-header">
    <h3>
      <Brain size={16} />
      Workspace Memory
    </h3>
    <Button variant="ghost" size="sm" onClick={() => void onRefreshKnowledge()} disabled={isLoadingKnowledge}>
      {isLoadingKnowledge ? "Refreshing..." : "Refresh"}
    </Button>
  </div>
  <div className="workspace-memory-grid">
    <div>
      <strong>Entities</strong>
      {knowledgeEntities.map((entity) => (
        <button key={entity.id} type="button" className="workspace-memory-card">
          <span>{entity.title}</span>
          <small>{entity.type}</small>
          <p>{entity.summary}</p>
        </button>
      ))}
    </div>
    <div>
      <strong>Key Claims</strong>
      {knowledgeClaims.map((claim) => (
        <div key={claim.id} className="workspace-memory-card">
          <span>{claim.title}</span>
          <small>{claim.type}</small>
          <p>{claim.summary}</p>
        </div>
      ))}
    </div>
    <div>
      <strong>Open Questions</strong>
      {knowledgeTasks.map((task) => (
        <div key={task.id} className="workspace-memory-card">
          <span>{task.title}</span>
          <small>{task.priority}</small>
          <p>{task.summary}</p>
        </div>
      ))}
    </div>
  </div>
</article>
```

- [ ] **Step 4: Run the frontend build to verify it passes**

Run: `npm run build`

Workdir: `frontend`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/types/workspaceKnowledge.ts frontend/src/api/workspaceKnowledge.ts frontend/src/App.tsx frontend/src/components/WorkspaceTab.tsx frontend/src/App.css
git commit -m "feat: add workspace memory browser ui"
```

## Task 6: Replace The Reader Sidebar With Knowledge Copilot

**Files:**
- Create: `frontend/src/components/KnowledgeCopilotPanel.tsx`
- Modify: `frontend/src/components/ReaderTab.tsx`
- Modify: `frontend/src/App.css`
- Delete: `frontend/src/components/ReaderAIPanel.tsx`

- [ ] **Step 1: Write the failing sidebar swap**

```tsx
// frontend/src/components/ReaderTab.tsx
import { KnowledgeCopilotPanel } from "./KnowledgeCopilotPanel";

// inside the assistant sidebar tab:
<KnowledgeCopilotPanel
  tab={tab}
  llmConfigs={groupedProviders.llm}
  activeLLMConfig={activeLLMConfig}
  activeLLMModel={activeLLMModel}
  llmProviderId={llmProviderId}
  llmModelId={llmModelId}
  setLlmProviderId={setLlmProviderId}
  setLlmModelId={setLlmModelId}
/>
```

- [ ] **Step 2: Run the frontend build to verify it fails**

Run: `npm run build`

Workdir: `frontend`

Expected: FAIL with `Cannot find module './KnowledgeCopilotPanel'`

- [ ] **Step 3: Implement the new copilot panel and retire the legacy panel**

```tsx
// frontend/src/components/KnowledgeCopilotPanel.tsx
import { useMemo, useState } from "react";
import { Sparkles, FileText, BookOpen, Plus } from "lucide-react";
import { workspaceKnowledgeApi } from "../api/workspaceKnowledge";
import type { ProviderConfig, ModelRecord } from "../types/config";
import type { TabItem } from "../store/tabStore";
import type {
  WorkspaceKnowledgeCandidate,
  WorkspaceKnowledgeEvidenceHit,
} from "../types/workspaceKnowledge";
import { MarkdownPreview } from "./MarkdownPreview";
import { Button } from "./ui/Button";

type KnowledgeScope = "selection" | "document" | "workspace";

interface KnowledgeCopilotPanelProps {
  tab: TabItem;
  llmConfigs: ProviderConfig[];
  activeLLMConfig: ProviderConfig | null;
  activeLLMModel: ModelRecord | null;
  llmProviderId: number | null;
  llmModelId: number | null;
  setLlmProviderId: (value: number | null) => void;
  setLlmModelId: (value: number | null) => void;
}

export function KnowledgeCopilotPanel({
  tab,
  llmConfigs,
  activeLLMConfig,
  activeLLMModel,
  llmProviderId,
  llmModelId,
  setLlmProviderId,
  setLlmModelId,
}: KnowledgeCopilotPanelProps) {
  const [scope, setScope] = useState<KnowledgeScope>("document");
  const [question, setQuestion] = useState("");
  const [answer, setAnswer] = useState("");
  const [evidence, setEvidence] = useState<WorkspaceKnowledgeEvidenceHit[]>([]);
  const [candidates, setCandidates] = useState<WorkspaceKnowledgeCandidate[]>([]);
  const [isRunning, setIsRunning] = useState(false);
  const [panelError, setPanelError] = useState<string | null>(null);

  const providerHint = useMemo(
    () => activeLLMModel ? `${activeLLMConfig?.provider.name ?? "LLM"} / ${activeLLMModel.modelId}` : "Select provider and model",
    [activeLLMConfig, activeLLMModel],
  );

  async function handleAsk() {
    if (!tab.workspaceId || !llmProviderId || !llmModelId || !question.trim()) {
      return;
    }
    setIsRunning(true);
    setPanelError(null);
    try {
      const result = await workspaceKnowledgeApi.query({
        workspaceId: tab.workspaceId,
        documentId: tab.documentId ?? "",
        providerId: llmProviderId,
        modelId: llmModelId,
        page: 0,
        selection: "",
        scope,
        question,
      });
      setAnswer(result.answer);
      setEvidence(result.evidence);
      setCandidates(result.candidates);
    } catch (error) {
      setPanelError(error instanceof Error ? error.message : "Knowledge query failed");
    } finally {
      setIsRunning(false);
    }
  }

  async function handlePromote() {
    if (!tab.workspaceId || candidates.length === 0) {
      return;
    }
    await workspaceKnowledgeApi.promote({
      workspaceId: tab.workspaceId,
      candidates,
    });
  }

  return (
    <div className="knowledge-copilot">
      <div className="section-header">
        <strong><Sparkles size={14} /> Knowledge Copilot</strong>
        <span className="badge">{providerHint}</span>
      </div>

      <label className="field">
        <span>Scope</span>
        <select value={scope} onChange={(event) => setScope(event.target.value as KnowledgeScope)}>
          <option value="selection">Selection</option>
          <option value="document">Document</option>
          <option value="workspace">Workspace</option>
        </select>
      </label>

      <label className="field">
        <span>Question</span>
        <textarea className="prompt-input" value={question} onChange={(event) => setQuestion(event.target.value)} />
      </label>

      <div className="prompt-actions">
        <Button onClick={() => void handleAsk()} disabled={isRunning || !question.trim()}>
          {isRunning ? "Answering..." : "Ask"}
        </Button>
        <Button variant="secondary" onClick={() => void handlePromote()} disabled={candidates.length === 0}>
          <Plus size={14} />
          Promote Memory
        </Button>
        <Button
          variant="secondary"
          onClick={() => void workspaceKnowledgeApi.startScan(tab.workspaceId ?? "", llmProviderId ?? 0, llmModelId ?? 0, tab.documentId ?? "")}
          disabled={!tab.workspaceId || !tab.documentId || !llmProviderId || !llmModelId}
        >
          Rescan Document
        </Button>
        <Button
          variant="secondary"
          onClick={() => void workspaceKnowledgeApi.startScan(tab.workspaceId ?? "", llmProviderId ?? 0, llmModelId ?? 0, "")}
          disabled={!tab.workspaceId || !llmProviderId || !llmModelId}
        >
          Rescan Workspace
        </Button>
      </div>

      {panelError ? <div className="reader-error">{panelError}</div> : null}

      <div className="knowledge-answer-panel">
        <div className="section-header">
          <strong><FileText size={14} /> Answer</strong>
        </div>
        <MarkdownPreview content={answer} placeholder="Ask a question to see a grounded answer." />
      </div>

      <div className="knowledge-evidence-panel">
        <div className="section-header">
          <strong><BookOpen size={14} /> Evidence</strong>
        </div>
        {evidence.map((item) => (
          <div key={item.id} className="workspace-memory-card">
            <span>{item.title}</span>
            <small>{item.kind}</small>
            <p>{item.excerpt}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
```

```ts
// frontend/src/api/workspaceKnowledge.ts
import type {
  WorkspaceKnowledgeClaim,
  WorkspaceKnowledgeEntity,
  WorkspaceKnowledgeEvidenceHit,
  WorkspaceKnowledgeQueryResult,
  WorkspaceKnowledgeSourceRef,
  WorkspaceKnowledgeTask,
} from "../types/workspaceKnowledge";
import type { WorkspaceWikiScanJob } from "../types/workspaceWiki";

export interface WorkspaceKnowledgeQueryInput {
  workspaceId: string;
  documentId: string;
  providerId: number;
  modelId: number;
  page: number;
  selection: string;
  scope: string;
  question: string;
}

export interface WorkspaceKnowledgeCandidate {
  id: string;
  title: string;
  type: string;
  summary: string;
  entityIds: string[];
  sourceRefs: WorkspaceKnowledgeSourceRef[];
}

export interface WorkspaceKnowledgeEvidenceHit {
  id: string;
  title: string;
  kind: string;
  sourceId: string;
  pageStart: number;
  pageEnd: number;
  excerpt: string;
  markdownPath: string;
}

export interface WorkspaceKnowledgeQueryResult {
  answer: string;
  evidence: WorkspaceKnowledgeEvidenceHit[];
  candidates: WorkspaceKnowledgeCandidate[];
}

export interface WorkspaceKnowledgePromotionInput {
  workspaceId: string;
  candidates: WorkspaceKnowledgeCandidate[];
}

interface WailsWorkspaceKnowledgeApp {
  ListWorkspaceKnowledgeEntities: (workspaceId: string) => Promise<WorkspaceKnowledgeEntity[]>;
  ListWorkspaceKnowledgeClaims: (workspaceId: string) => Promise<WorkspaceKnowledgeClaim[]>;
  ListWorkspaceKnowledgeTasks: (workspaceId: string) => Promise<WorkspaceKnowledgeTask[]>;
  StartWorkspaceWikiScan: (input: { workspaceId: string; documentId: string; providerId: number; modelId: number }) => Promise<WorkspaceWikiScanJob>;
  QueryWorkspaceKnowledge: (input: WorkspaceKnowledgeQueryInput) => Promise<WorkspaceKnowledgeQueryResult>;
  PromoteWorkspaceKnowledge: (input: WorkspaceKnowledgePromotionInput) => Promise<void>;
}

function getApp(): WailsWorkspaceKnowledgeApp | null {
  if (typeof window !== "undefined" && "go" in window && window.go?.main?.App) {
    return window.go.main.App as WailsWorkspaceKnowledgeApp;
  }
  return null;
}

export const workspaceKnowledgeApi = {
  async listEntities(workspaceId: string): Promise<WorkspaceKnowledgeEntity[]> {
    const app = getApp() as WailsWorkspaceKnowledgeApp | null;
    return app ? app.ListWorkspaceKnowledgeEntities(workspaceId) : [];
  },
  async listClaims(workspaceId: string): Promise<WorkspaceKnowledgeClaim[]> {
    const app = getApp() as WailsWorkspaceKnowledgeApp | null;
    return app ? app.ListWorkspaceKnowledgeClaims(workspaceId) : [];
  },
  async listTasks(workspaceId: string): Promise<WorkspaceKnowledgeTask[]> {
    const app = getApp() as WailsWorkspaceKnowledgeApp | null;
    return app ? app.ListWorkspaceKnowledgeTasks(workspaceId) : [];
  },
  async startScan(workspaceId: string, providerId: number, modelId: number, documentId: string): Promise<WorkspaceWikiScanJob | null> {
    const app = getApp() as WailsWorkspaceKnowledgeApp | null;
    if (!app) {
      return null;
    }
    return app.StartWorkspaceWikiScan({ workspaceId, documentId, providerId, modelId });
  },
  async query(input: WorkspaceKnowledgeQueryInput): Promise<WorkspaceKnowledgeQueryResult> {
    const app = getApp() as WailsWorkspaceKnowledgeApp | null;
    if (!app) {
      return { answer: "", evidence: [], candidates: [] };
    }
    return app.QueryWorkspaceKnowledge(input);
  },
  async promote(input: WorkspaceKnowledgePromotionInput): Promise<void> {
    const app = getApp() as WailsWorkspaceKnowledgeApp | null;
    if (!app) {
      return;
    }
    await app.PromoteWorkspaceKnowledge(input);
  },
};
```

- [ ] **Step 4: Run the frontend build to verify it passes**

Run: `npm run build`

Workdir: `frontend`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/KnowledgeCopilotPanel.tsx frontend/src/components/ReaderTab.tsx frontend/src/api/workspaceKnowledge.ts frontend/src/App.css
git rm frontend/src/components/ReaderAIPanel.tsx
git commit -m "feat: replace legacy reader ai with knowledge copilot"
```

## Task 7: Add End-To-End Regression Coverage And Final Verification

**Files:**
- Create: `workspace_knowledge_e2e_test.go`
- Modify: `workspace_knowledge_query.go`
- Modify: `workspace_wiki_service.go`

- [ ] **Step 1: Write the failing end-to-end integration test**

```go
func TestWorkspaceKnowledgeEndToEndScanQueryPromote(t *testing.T) {
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

	workspace, err := store.CreateWorkspace(t.Context(), WorkspaceUpsertInput{Name: "End To End", Description: "", Color: ""})
	if err != nil {
		t.Fatalf("CreateWorkspace error: %v", err)
	}

	sourcePath := filepath.Join(tempDir, "paper.md")
	if err := os.WriteFile(sourcePath, []byte("# Paper\n\nContrastive Memory improves retrieval."), 0o600); err != nil {
		t.Fatalf("write markdown error: %v", err)
	}
	if _, err := store.ImportFiles(t.Context(), paths, ImportFilesInput{
		WorkspaceID: workspace.ID,
		FilePaths:   []string{sourcePath},
		SourceType:  "manual",
	}); err != nil {
		t.Fatalf("ImportFiles error: %v", err)
	}

	runner := &workspaceWikiScanRunner{
		paths: paths,
		store: store,
		pdf: &stubWorkspaceKnowledgeExtractor{
			markdown: "# Paper\n\nContrastive Memory improves retrieval.",
		},
		knowledgeLLM: &stubWorkspaceKnowledgeLLM{
			bySource: WorkspaceKnowledgeBySourcePayload{
				Entities: []WorkspaceKnowledgeEntity{{
					ID: "entity:method:contrastive-memory",
					WorkspaceID: workspace.ID,
					Title: "Contrastive Memory",
					Type: "method",
					Summary: "Memory-augmented retrieval method",
					Origin: "scan",
					Status: "confirmed",
					Confidence: 0.9,
					CreatedAt: nowRFC3339(),
					UpdatedAt: nowRFC3339(),
				}},
			},
			docWiki: "# Paper\n\n## Summary\nContrastive Memory improves retrieval.",
			overviewWiki: "# End To End Overview\n\n## Themes\n- Contrastive Memory",
			query: WorkspaceKnowledgeQueryResult{
				Answer: "Contrastive Memory is the key method in the workspace.",
				Candidates: []WorkspaceKnowledgeCandidate{{
					ID: "candidate:claim:contrastive-memory-key",
					Title: "Contrastive Memory is the key method",
					Type: "claim",
					Summary: "The workspace centers on Contrastive Memory",
				}},
			},
		},
	}

	job, err := store.SaveWorkspaceWikiScanJob(t.Context(), WorkspaceWikiScanJob{
		WorkspaceID:     workspace.ID,
		Status:          WorkspaceWikiScanJobQueued,
		CurrentStage:    "queued",
		Message:         "queued",
		ProviderID:      1,
		ModelID:         2,
		StartedAt:       nowRFC3339(),
		UpdatedAt:       nowRFC3339(),
	})
	if err != nil {
		t.Fatalf("SaveWorkspaceWikiScanJob error: %v", err)
	}
	runner.run(context.Background(), job)

	files := newWorkspaceKnowledgeFiles(paths, workspace.ID)
	queryService := workspaceKnowledgeQueryService{files: files, llm: runner.knowledgeLLM}
	result, err := queryService.Query(context.Background(), WorkspaceKnowledgeQueryInput{
		WorkspaceID: workspace.ID,
		ProviderID:  1,
		ModelID:     2,
		Scope:       "workspace",
		Question:    "What is the key method?",
	})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if result.Answer == "" || len(result.Candidates) != 1 {
		t.Fatalf("query result = %#v, want answer and one candidate", result)
	}

	if err := queryService.Promote(context.Background(), WorkspaceKnowledgePromotionInput{
		WorkspaceID: workspace.ID,
		Candidates:  result.Candidates,
	}); err != nil {
		t.Fatalf("Promote error: %v", err)
	}

	claims, err := readWorkspaceKnowledgeClaims(filepath.Join(files.schemaDir, "claims.json"))
	if err != nil {
		t.Fatalf("read claims error: %v", err)
	}
	if len(claims) == 0 {
		t.Fatalf("claims = %#v, want promoted claim", claims)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./... -run TestWorkspaceKnowledgeEndToEndScanQueryPromote -count=1`

Expected: FAIL until all scan, query, and promote glue paths are complete

- [ ] **Step 3: Fill the last integration gaps**

```go
// workspace_knowledge_query.go
func readWorkspaceKnowledgeEntities(path string) ([]WorkspaceKnowledgeEntity, error) {
	var entities []WorkspaceKnowledgeEntity
	err := readWorkspaceKnowledgeJSON(path, &entities)
	return entities, err
}

func readWorkspaceKnowledgeClaims(path string) ([]WorkspaceKnowledgeClaim, error) {
	var claims []WorkspaceKnowledgeClaim
	err := readWorkspaceKnowledgeJSON(path, &claims)
	return claims, err
}

func readWorkspaceKnowledgeTasks(path string) ([]WorkspaceKnowledgeTask, error) {
	var tasks []WorkspaceKnowledgeTask
	err := readWorkspaceKnowledgeJSON(path, &tasks)
	return tasks, err
}
```

```go
// workspace_wiki_service.go
func buildWorkspaceKnowledgeSourceID(workspaceID, title string) string {
	return "source:" + workspaceID + ":" + workspaceWikiSlug(title)
}
```

- [ ] **Step 4: Run the final verification suite**

Run: `go test ./...`

Expected: PASS

Run: `npm run build`

Workdir: `frontend`

Expected: PASS

Run: `wails generate module`

Expected: no binding generation errors and no unexpected drift beyond the new knowledge APIs

- [ ] **Step 5: Commit**

```bash
git add workspace_knowledge_e2e_test.go workspace_knowledge_query.go workspace_wiki_service.go frontend/wailsjs/go/main/App.d.ts frontend/wailsjs/go/main/App.js frontend/wailsjs/go/models.ts
git commit -m "test: cover end-to-end workspace knowledge flow"
```
