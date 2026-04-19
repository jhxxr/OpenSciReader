package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCompileWorkspaceKnowledgeBuildsAggregatesAndWiki(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	workspaceDir := filepath.Join(paths.WorkspacesRootDir, "workspace-a")
	staleDocPath := filepath.Join(workspaceDir, "wiki", "docs", "stale.md")
	staleConceptPath := filepath.Join(workspaceDir, "wiki", "concepts", "stale.md")
	if err := os.WriteFile(staleDocPath, []byte("stale doc"), 0o600); err != nil {
		t.Fatalf("write stale doc error: %v", err)
	}
	if err := os.WriteFile(staleConceptPath, []byte("stale concept"), 0o600); err != nil {
		t.Fatalf("write stale concept error: %v", err)
	}

	now := nowRFC3339()
	payloadA := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{ID: "source:paper-a", WorkspaceID: "workspace-a", Title: "Paper A", Slug: "paper-a", Kind: "pdf"},
		Entities: []WorkspaceKnowledgeEntity{{
			ID:          "entity:method:contrastive-memory",
			WorkspaceID: "workspace-a",
			Title:       "Contrastive Memory",
			Type:        "method",
			Summary:     "Shared method summary",
			Aliases:     []string{"CM"},
			SourceRefs: []WorkspaceKnowledgeSourceRef{{
				SourceID:  "source:paper-a",
				PageStart: 3,
				PageEnd:   3,
				Excerpt:   "Contrastive Memory excerpt",
			}},
			Origin:     "scan",
			Status:     "confirmed",
			Confidence: 0.9,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
		Claims: []WorkspaceKnowledgeClaim{{
			ID:          "claim:paper-a-result",
			WorkspaceID: "workspace-a",
			Title:       "Paper A improves retrieval accuracy",
			Type:        "result",
			Summary:     "Accuracy improves on the benchmark",
			EntityIDs:   []string{"entity:method:contrastive-memory"},
			SourceRefs: []WorkspaceKnowledgeSourceRef{{
				SourceID:  "source:paper-a",
				PageStart: 5,
				PageEnd:   5,
				Excerpt:   "Accuracy improves",
			}},
			Origin:     "scan",
			Status:     "confirmed",
			Confidence: 0.85,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
	}
	payloadB := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{ID: "source:paper-b", WorkspaceID: "workspace-a", Title: "Paper B", Slug: "paper-b-metadata", Kind: "markdown"},
		Tasks: []WorkspaceKnowledgeTask{{
			ID:          "task:compare-paper-a-paper-b",
			WorkspaceID: "workspace-a",
			Title:       "Compare Paper A and Paper B",
			Type:        "open_question",
			Summary:     "Need to verify transferability claims",
			Priority:    "medium",
			SourceRefs: []WorkspaceKnowledgeSourceRef{{
				SourceID:  "source:paper-b",
				PageStart: 2,
				PageEnd:   2,
				Excerpt:   "Need a comparison",
			}},
			Origin:     "scan",
			Status:     "candidate",
			Confidence: 0.6,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
	}

	if err := files.WriteBySource("paper-a", payloadA); err != nil {
		t.Fatalf("WriteBySource paper-a error: %v", err)
	}
	if err := files.WriteBySource("paper-b-canonical", payloadB); err != nil {
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
	if len(snapshot.Relations) != 0 {
		t.Fatalf("relations = %d, want 0", len(snapshot.Relations))
	}

	assertWorkspaceKnowledgeJSONCount(t, filepath.Join(workspaceDir, "schema", "entities.json"), 1)
	assertWorkspaceKnowledgeJSONCount(t, filepath.Join(workspaceDir, "schema", "claims.json"), 1)
	assertWorkspaceKnowledgeJSONCount(t, filepath.Join(workspaceDir, "schema", "relations.json"), 0)
	assertWorkspaceKnowledgeJSONCount(t, filepath.Join(workspaceDir, "schema", "tasks.json"), 1)

	overview, err := os.ReadFile(filepath.Join(workspaceDir, "wiki", "overview.md"))
	if err != nil {
		t.Fatalf("read overview.md error: %v", err)
	}
	if !strings.Contains(string(overview), "Knowledge Workspace") {
		t.Fatalf("overview.md = %q, want workspace title", string(overview))
	}
	if !strings.Contains(string(overview), "docs/paper-b-canonical.md") {
		t.Fatalf("overview.md = %q, want canonical doc link", string(overview))
	}

	openQuestions, err := os.ReadFile(filepath.Join(workspaceDir, "wiki", "open-questions.md"))
	if err != nil {
		t.Fatalf("read open-questions.md error: %v", err)
	}
	if !strings.Contains(string(openQuestions), "Compare Paper A and Paper B") {
		t.Fatalf("open-questions.md = %q, want task title", string(openQuestions))
	}

	documentPageA, err := os.ReadFile(filepath.Join(workspaceDir, "wiki", "docs", "paper-a.md"))
	if err != nil {
		t.Fatalf("read paper-a document page error: %v", err)
	}
	if !strings.Contains(string(documentPageA), "Paper A") {
		t.Fatalf("paper-a document page = %q, want source title", string(documentPageA))
	}

	documentPageB, err := os.ReadFile(filepath.Join(workspaceDir, "wiki", "docs", "paper-b-canonical.md"))
	if err != nil {
		t.Fatalf("read paper-b-canonical document page error: %v", err)
	}
	if !strings.Contains(string(documentPageB), "Paper B") {
		t.Fatalf("paper-b-canonical document page = %q, want source title", string(documentPageB))
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, "wiki", "docs", "paper-b-metadata.md")); !os.IsNotExist(err) {
		t.Fatalf("paper-b-metadata.md stat err = %v, want not exist", err)
	}
	if _, err := os.Stat(staleDocPath); !os.IsNotExist(err) {
		t.Fatalf("stale doc stat err = %v, want not exist", err)
	}
	if _, err := os.Stat(staleConceptPath); !os.IsNotExist(err) {
		t.Fatalf("stale concept stat err = %v, want not exist", err)
	}

	conceptPage, err := os.ReadFile(filepath.Join(workspaceDir, "wiki", "concepts", "contrastive-memory.md"))
	if err != nil {
		t.Fatalf("read concept page error: %v", err)
	}
	if !strings.Contains(string(conceptPage), "Contrastive Memory") {
		t.Fatalf("concept page = %q, want concept title", string(conceptPage))
	}
}

func TestCompileWorkspaceKnowledgePreservesExistingWikiOnTargetValidationError(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	now := nowRFC3339()
	payload := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{
			ID:          "source:paper-a",
			WorkspaceID: "workspace-a",
			Title:       "Paper A",
			Slug:        "paper-a",
			Kind:        "pdf",
		},
		Entities: []WorkspaceKnowledgeEntity{{
			ID:          "entity:method:contrastive-memory",
			WorkspaceID: "workspace-a",
			Title:       "Contrastive Memory",
			Type:        "method",
			Summary:     "Shared method summary",
			SourceRefs: []WorkspaceKnowledgeSourceRef{{
				SourceID:  "source:paper-a",
				PageStart: 1,
				PageEnd:   1,
				Excerpt:   "Excerpt",
			}},
			Origin:     "scan",
			Status:     "confirmed",
			Confidence: 0.9,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
	}

	if err := files.WriteBySource("paper-a", payload); err != nil {
		t.Fatalf("WriteBySource error: %v", err)
	}

	workspaceDir := filepath.Join(paths.WorkspacesRootDir, "workspace-a")
	staleDocPath := filepath.Join(workspaceDir, "wiki", "docs", "stale.md")
	staleConceptPath := filepath.Join(workspaceDir, "wiki", "concepts", "stale.md")
	if err := os.WriteFile(staleDocPath, []byte("stale doc"), 0o600); err != nil {
		t.Fatalf("write stale doc error: %v", err)
	}
	if err := os.WriteFile(staleConceptPath, []byte("stale concept"), 0o600); err != nil {
		t.Fatalf("write stale concept error: %v", err)
	}

	blockingDocPath := filepath.Join(workspaceDir, "wiki", "docs", "paper-a.md")
	if err := os.Mkdir(blockingDocPath, 0o700); err != nil {
		t.Fatalf("create blocking doc directory error: %v", err)
	}

	if _, err := CompileWorkspaceKnowledge(files, "Knowledge Workspace"); err == nil {
		t.Fatal("CompileWorkspaceKnowledge expected error")
	}

	if _, err := os.Stat(staleDocPath); err != nil {
		t.Fatalf("stale doc stat error: %v", err)
	}
	if _, err := os.Stat(staleConceptPath); err != nil {
		t.Fatalf("stale concept stat error: %v", err)
	}
}

func TestCompileWorkspaceKnowledgePreservesSourceProvenanceForDuplicateTitles(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	now := nowRFC3339()
	payloadA := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{
			ID:          "source:alpha",
			WorkspaceID: "workspace-a",
			Title:       "Shared Title",
			Slug:        "alpha",
			Kind:        "pdf",
		},
		Entities: []WorkspaceKnowledgeEntity{{
			ID:          "entity:concept:shared",
			WorkspaceID: "workspace-a",
			Title:       "Shared Concept",
			Type:        "concept",
			Summary:     "Summary",
			SourceRefs: []WorkspaceKnowledgeSourceRef{
				{SourceID: "source:beta", PageStart: 2, PageEnd: 2, Excerpt: "beta excerpt"},
				{SourceID: "source:alpha", PageStart: 1, PageEnd: 1, Excerpt: "alpha excerpt"},
			},
			Origin:     "scan",
			Status:     "confirmed",
			Confidence: 0.9,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
		Tasks: []WorkspaceKnowledgeTask{{
			ID:          "task:compare-sources",
			WorkspaceID: "workspace-a",
			Title:       "Compare duplicate titles",
			Type:        "open_question",
			Summary:     "Check provenance labels",
			Priority:    "high",
			SourceRefs: []WorkspaceKnowledgeSourceRef{
				{SourceID: "source:beta", PageStart: 4, PageEnd: 4, Excerpt: "beta task"},
				{SourceID: "source:alpha", PageStart: 3, PageEnd: 3, Excerpt: "alpha task"},
			},
			Origin:     "scan",
			Status:     "candidate",
			Confidence: 0.7,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
	}
	payloadB := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{
			ID:          "source:beta",
			WorkspaceID: "workspace-a",
			Title:       "Shared Title",
			Slug:        "beta",
			Kind:        "markdown",
		},
	}

	if err := files.WriteBySource("beta", payloadB); err != nil {
		t.Fatalf("WriteBySource beta error: %v", err)
	}
	if err := files.WriteBySource("alpha", payloadA); err != nil {
		t.Fatalf("WriteBySource alpha error: %v", err)
	}

	if _, err := CompileWorkspaceKnowledge(files, "Knowledge Workspace"); err != nil {
		t.Fatalf("CompileWorkspaceKnowledge error: %v", err)
	}

	workspaceDir := filepath.Join(paths.WorkspacesRootDir, "workspace-a")
	expectedSources := "Shared Title (`source:alpha`), Shared Title (`source:beta`)"

	conceptPage, err := os.ReadFile(filepath.Join(workspaceDir, "wiki", "concepts", "shared-concept.md"))
	if err != nil {
		t.Fatalf("read concept page error: %v", err)
	}
	if !strings.Contains(string(conceptPage), expectedSources) {
		t.Fatalf("concept page = %q, want sources %q", string(conceptPage), expectedSources)
	}

	openQuestions, err := os.ReadFile(filepath.Join(workspaceDir, "wiki", "open-questions.md"))
	if err != nil {
		t.Fatalf("read open-questions.md error: %v", err)
	}
	if !strings.Contains(string(openQuestions), expectedSources) {
		t.Fatalf("open-questions.md = %q, want sources %q", string(openQuestions), expectedSources)
	}
}

func TestCompileWorkspaceKnowledgeNormalizesAggregateOrdering(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	now := nowRFC3339()
	payloadAlpha := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{
			ID:          "source:alpha",
			WorkspaceID: "workspace-a",
			Title:       "Alpha Source",
			Slug:        "alpha",
			Kind:        "pdf",
		},
		Entities: []WorkspaceKnowledgeEntity{
			{
				ID:          "entity:zeta",
				WorkspaceID: "workspace-a",
				Title:       "Zeta Concept",
				Type:        "concept",
				Summary:     "Zeta summary",
				Aliases:     []string{"Zulu", "Alpha"},
				SourceRefs: []WorkspaceKnowledgeSourceRef{
					{SourceID: "source:beta", PageStart: 4, PageEnd: 4, Excerpt: "beta excerpt"},
					{SourceID: "source:alpha", PageStart: 2, PageEnd: 2, Excerpt: "alpha excerpt"},
				},
				Origin:     "scan",
				Status:     "confirmed",
				Confidence: 0.9,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
			{
				ID:          "entity:alpha",
				WorkspaceID: "workspace-a",
				Title:       "Alpha Concept",
				Type:        "concept",
				Summary:     "Alpha summary",
				Aliases:     []string{"Gamma", "Beta"},
				SourceRefs: []WorkspaceKnowledgeSourceRef{
					{SourceID: "source:beta", PageStart: 6, PageEnd: 6, Excerpt: "beta excerpt"},
					{SourceID: "source:alpha", PageStart: 1, PageEnd: 1, Excerpt: "alpha excerpt"},
				},
				Origin:     "scan",
				Status:     "confirmed",
				Confidence: 0.95,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
		Claims: []WorkspaceKnowledgeClaim{{
			ID:          "claim:ordering",
			WorkspaceID: "workspace-a",
			Title:       "Ordering claim",
			Type:        "result",
			Summary:     "Ordering summary",
			EntityIDs:   []string{"entity:zeta", "entity:alpha"},
			SourceRefs: []WorkspaceKnowledgeSourceRef{
				{SourceID: "source:beta", PageStart: 8, PageEnd: 8, Excerpt: "beta claim"},
				{SourceID: "source:alpha", PageStart: 3, PageEnd: 3, Excerpt: "alpha claim"},
			},
			Origin:     "scan",
			Status:     "confirmed",
			Confidence: 0.8,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
		Relations: []WorkspaceKnowledgeRelation{{
			ID:          "relation:ordering",
			WorkspaceID: "workspace-a",
			Type:        "supports",
			FromID:      "entity:zeta",
			ToID:        "entity:alpha",
			Summary:     "Ordering relation",
			SourceRefs: []WorkspaceKnowledgeSourceRef{
				{SourceID: "source:beta", PageStart: 10, PageEnd: 10, Excerpt: "beta relation"},
				{SourceID: "source:alpha", PageStart: 5, PageEnd: 5, Excerpt: "alpha relation"},
			},
			Origin:     "scan",
			Status:     "confirmed",
			Confidence: 0.7,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
		Tasks: []WorkspaceKnowledgeTask{{
			ID:          "task:ordering",
			WorkspaceID: "workspace-a",
			Title:       "Ordering task",
			Type:        "open_question",
			Summary:     "Ordering task summary",
			Priority:    "low",
			SourceRefs: []WorkspaceKnowledgeSourceRef{
				{SourceID: "source:beta", PageStart: 12, PageEnd: 12, Excerpt: "beta task"},
				{SourceID: "source:alpha", PageStart: 7, PageEnd: 7, Excerpt: "alpha task"},
			},
			Origin:     "scan",
			Status:     "candidate",
			Confidence: 0.6,
			CreatedAt:  now,
			UpdatedAt:  now,
		}},
	}
	payloadBeta := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{
			ID:          "source:beta",
			WorkspaceID: "workspace-a",
			Title:       "Beta Source",
			Slug:        "beta",
			Kind:        "markdown",
		},
	}

	if err := files.WriteBySource("beta", payloadBeta); err != nil {
		t.Fatalf("WriteBySource beta error: %v", err)
	}
	if err := files.WriteBySource("alpha", payloadAlpha); err != nil {
		t.Fatalf("WriteBySource alpha error: %v", err)
	}

	if _, err := CompileWorkspaceKnowledge(files, "Knowledge Workspace"); err != nil {
		t.Fatalf("CompileWorkspaceKnowledge error: %v", err)
	}

	workspaceDir := filepath.Join(paths.WorkspacesRootDir, "workspace-a")

	var entities []WorkspaceKnowledgeEntity
	if err := readWorkspaceKnowledgeJSON(filepath.Join(workspaceDir, "schema", "entities.json"), &entities); err != nil {
		t.Fatalf("read entities.json error: %v", err)
	}
	if got := []string{entities[0].ID, entities[1].ID}; !reflect.DeepEqual(got, []string{"entity:alpha", "entity:zeta"}) {
		t.Fatalf("entity order = %#v, want [entity:alpha entity:zeta]", got)
	}
	if !reflect.DeepEqual(entities[0].Aliases, []string{"Beta", "Gamma"}) {
		t.Fatalf("entity alpha aliases = %#v, want [Beta Gamma]", entities[0].Aliases)
	}
	if !reflect.DeepEqual(entities[1].Aliases, []string{"Alpha", "Zulu"}) {
		t.Fatalf("entity zeta aliases = %#v, want [Alpha Zulu]", entities[1].Aliases)
	}
	assertWorkspaceKnowledgeSourceRefs(t, entities[0].SourceRefs, []WorkspaceKnowledgeSourceRef{
		{SourceID: "source:alpha", PageStart: 1, PageEnd: 1, Excerpt: "alpha excerpt"},
		{SourceID: "source:beta", PageStart: 6, PageEnd: 6, Excerpt: "beta excerpt"},
	})
	assertWorkspaceKnowledgeSourceRefs(t, entities[1].SourceRefs, []WorkspaceKnowledgeSourceRef{
		{SourceID: "source:alpha", PageStart: 2, PageEnd: 2, Excerpt: "alpha excerpt"},
		{SourceID: "source:beta", PageStart: 4, PageEnd: 4, Excerpt: "beta excerpt"},
	})

	var claims []WorkspaceKnowledgeClaim
	if err := readWorkspaceKnowledgeJSON(filepath.Join(workspaceDir, "schema", "claims.json"), &claims); err != nil {
		t.Fatalf("read claims.json error: %v", err)
	}
	if !reflect.DeepEqual(claims[0].EntityIDs, []string{"entity:alpha", "entity:zeta"}) {
		t.Fatalf("claim entity IDs = %#v, want [entity:alpha entity:zeta]", claims[0].EntityIDs)
	}
	assertWorkspaceKnowledgeSourceRefs(t, claims[0].SourceRefs, []WorkspaceKnowledgeSourceRef{
		{SourceID: "source:alpha", PageStart: 3, PageEnd: 3, Excerpt: "alpha claim"},
		{SourceID: "source:beta", PageStart: 8, PageEnd: 8, Excerpt: "beta claim"},
	})

	var relations []WorkspaceKnowledgeRelation
	if err := readWorkspaceKnowledgeJSON(filepath.Join(workspaceDir, "schema", "relations.json"), &relations); err != nil {
		t.Fatalf("read relations.json error: %v", err)
	}
	assertWorkspaceKnowledgeSourceRefs(t, relations[0].SourceRefs, []WorkspaceKnowledgeSourceRef{
		{SourceID: "source:alpha", PageStart: 5, PageEnd: 5, Excerpt: "alpha relation"},
		{SourceID: "source:beta", PageStart: 10, PageEnd: 10, Excerpt: "beta relation"},
	})

	var tasks []WorkspaceKnowledgeTask
	if err := readWorkspaceKnowledgeJSON(filepath.Join(workspaceDir, "schema", "tasks.json"), &tasks); err != nil {
		t.Fatalf("read tasks.json error: %v", err)
	}
	assertWorkspaceKnowledgeSourceRefs(t, tasks[0].SourceRefs, []WorkspaceKnowledgeSourceRef{
		{SourceID: "source:alpha", PageStart: 7, PageEnd: 7, Excerpt: "alpha task"},
		{SourceID: "source:beta", PageStart: 12, PageEnd: 12, Excerpt: "beta task"},
	})
}

func TestCompileWorkspaceKnowledgeFailsOnDuplicateSourceID(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	payloadA := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{
			ID:          "source:duplicate",
			WorkspaceID: "workspace-a",
			Title:       "First",
			Slug:        "first",
			Kind:        "pdf",
		},
	}
	payloadB := WorkspaceKnowledgeBySourcePayload{
		Source: WorkspaceKnowledgeSource{
			ID:          "source:duplicate",
			WorkspaceID: "workspace-a",
			Title:       "Second",
			Slug:        "second",
			Kind:        "markdown",
		},
	}

	if err := files.WriteBySource("beta", payloadB); err != nil {
		t.Fatalf("WriteBySource beta error: %v", err)
	}
	if err := files.WriteBySource("alpha", payloadA); err != nil {
		t.Fatalf("WriteBySource alpha error: %v", err)
	}

	_, err := CompileWorkspaceKnowledge(files, "Knowledge Workspace")
	if err == nil {
		t.Fatal("CompileWorkspaceKnowledge expected error")
	}
	want := `duplicate workspace knowledge source id "source:duplicate" in by-source files "alpha" and "beta"`
	if err.Error() != want {
		t.Fatalf("CompileWorkspaceKnowledge error = %q, want %q", err.Error(), want)
	}
}

func assertWorkspaceKnowledgeJSONCount(t *testing.T, path string, want int) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s error: %v", filepath.Base(path), err)
	}

	var records []map[string]any
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatalf("unmarshal %s error: %v", filepath.Base(path), err)
	}
	if len(records) != want {
		t.Fatalf("%s records = %d, want %d", filepath.Base(path), len(records), want)
	}
}

func assertWorkspaceKnowledgeSourceRefs(t *testing.T, got, want []WorkspaceKnowledgeSourceRef) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("source refs = %#v, want %#v", got, want)
	}
}
