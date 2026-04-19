package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		Source: WorkspaceKnowledgeSource{ID: "source:paper-b", WorkspaceID: "workspace-a", Title: "Paper B", Slug: "", Kind: "markdown"},
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
	if len(snapshot.Relations) != 0 {
		t.Fatalf("relations = %d, want 0", len(snapshot.Relations))
	}

	workspaceDir := filepath.Join(paths.WorkspacesRootDir, "workspace-a")
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

	documentPageB, err := os.ReadFile(filepath.Join(workspaceDir, "wiki", "docs", "source-paper-b.md"))
	if err != nil {
		t.Fatalf("read source-paper-b document page error: %v", err)
	}
	if !strings.Contains(string(documentPageB), "Paper B") {
		t.Fatalf("source-paper-b document page = %q, want source title", string(documentPageB))
	}

	conceptPage, err := os.ReadFile(filepath.Join(workspaceDir, "wiki", "concepts", "contrastive-memory.md"))
	if err != nil {
		t.Fatalf("read concept page error: %v", err)
	}
	if !strings.Contains(string(conceptPage), "Contrastive Memory") {
		t.Fatalf("concept page = %q, want concept title", string(conceptPage))
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
