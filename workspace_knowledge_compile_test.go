package main

import (
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
		Source: WorkspaceKnowledgeSource{ID: "source:paper-b", WorkspaceID: "workspace-a", Title: "Paper B", Slug: "paper-b", Kind: "markdown"},
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

	workspaceDir := filepath.Join(paths.WorkspacesRootDir, "workspace-a")
	overview, err := os.ReadFile(filepath.Join(workspaceDir, "wiki", "overview.md"))
	if err != nil {
		t.Fatalf("read overview.md error: %v", err)
	}
	if !strings.Contains(string(overview), "Knowledge Workspace") {
		t.Fatalf("overview.md = %q, want workspace title", string(overview))
	}

	conceptPage, err := os.ReadFile(filepath.Join(workspaceDir, "wiki", "concepts", "contrastive-memory.md"))
	if err != nil {
		t.Fatalf("read concept page error: %v", err)
	}
	if !strings.Contains(string(conceptPage), "Contrastive Memory") {
		t.Fatalf("concept page = %q, want concept title", string(conceptPage))
	}
}
