package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestWorkspaceKnowledgeQueryPrefersSchemaAndPromotesCandidates(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	entitiesPath, err := files.EntitiesPath()
	if err != nil {
		t.Fatalf("EntitiesPath error: %v", err)
	}
	entities := []WorkspaceKnowledgeEntity{{
		ID:          "entity:method:contrastive-memory",
		WorkspaceID: "workspace-a",
		Title:       "Contrastive Memory",
		Type:        "method",
		Summary:     "Memory-augmented retrieval method",
		SourceRefs: []WorkspaceKnowledgeSourceRef{{
			SourceID:  "source:paper-a",
			PageStart: 3,
			PageEnd:   3,
			Excerpt:   "Contrastive Memory excerpt",
		}},
		Origin:     "scan",
		Status:     "confirmed",
		Confidence: 0.9,
		CreatedAt:  nowRFC3339(),
		UpdatedAt:  nowRFC3339(),
	}}
	if err := writeWorkspaceKnowledgeJSON(entitiesPath, entities); err != nil {
		t.Fatalf("write entities error: %v", err)
	}

	conceptPath, err := files.ConceptWikiPath("contrastive-memory")
	if err != nil {
		t.Fatalf("ConceptWikiPath error: %v", err)
	}
	if err := writeWorkspaceKnowledgeMarkdown(conceptPath, "# Contrastive Memory\n\n## Definition\nMemory-augmented retrieval method\n"); err != nil {
		t.Fatalf("write concept page error: %v", err)
	}

	service := workspaceKnowledgeQueryService{
		files: files,
		llm: &stubWorkspaceKnowledgeQueryLLM{
			result: WorkspaceKnowledgeQueryResult{
				Answer: "Contrastive Memory is the main method in the workspace.",
				Candidates: []WorkspaceKnowledgeCandidate{{
					ID:      "candidate:claim:contrastive-memory-core",
					Title:   "Contrastive Memory is the main method",
					Type:    "claim",
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

	claimsPath, err := files.ClaimsPath()
	if err != nil {
		t.Fatalf("ClaimsPath error: %v", err)
	}
	data, err := os.ReadFile(claimsPath)
	if err != nil {
		t.Fatalf("read claims.json error: %v", err)
	}
	if !strings.Contains(string(data), "Contrastive Memory is the main method") {
		t.Fatalf("claims.json = %q, want promoted claim", string(data))
	}
}

func TestWorkspaceKnowledgeQueryReturnsInsufficientEvidenceWithoutSources(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	llm := &stubWorkspaceKnowledgeQueryLLM{
		result: WorkspaceKnowledgeQueryResult{
			Answer: "This should not be returned.",
		},
	}
	service := workspaceKnowledgeQueryService{
		files: files,
		llm:   llm,
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
	if got := result.Answer; !strings.Contains(strings.ToLower(got), "insufficient evidence") {
		t.Fatalf("answer = %q, want insufficient evidence response", got)
	}
	if llm.prompt != "" {
		t.Fatalf("llm prompt = %q, want no llm call without evidence", llm.prompt)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("candidates = %#v, want no candidates without evidence", result.Candidates)
	}
}

type stubWorkspaceKnowledgeQueryLLM struct {
	result WorkspaceKnowledgeQueryResult
	err    error
	prompt string
}

func (s *stubWorkspaceKnowledgeQueryLLM) GenerateWorkspaceKnowledgeQuery(_ context.Context, _ int64, _ int64, prompt string) (WorkspaceKnowledgeQueryResult, error) {
	s.prompt = prompt
	if s.err != nil {
		return WorkspaceKnowledgeQueryResult{}, s.err
	}
	return s.result, nil
}
