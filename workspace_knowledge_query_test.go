package main

import (
	"context"
	"os"
	"testing"
)

func TestRetrieveWorkspaceKnowledgeEvidencePrefersWikiBeforeStateAndInputs(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}

	indexPath, err := files.IndexPath()
	if err != nil {
		t.Fatalf("IndexPath() error = %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("# Index\n\n- [[concepts/attention|Attention]]\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(indexPath) error = %v", err)
	}

	conceptPath, err := files.ConceptWikiPath("attention")
	if err != nil {
		t.Fatalf("ConceptWikiPath() error = %v", err)
	}
	if err := os.WriteFile(conceptPath, []byte("# Attention\n\nAttention is the mechanism that focuses computation on relevant context.\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(conceptPath) error = %v", err)
	}

	entitiesPath, err := files.EntitiesPath()
	if err != nil {
		t.Fatalf("EntitiesPath() error = %v", err)
	}
	if err := writeWorkspaceKnowledgeJSON(entitiesPath, []WorkspaceKnowledgeEntity{{
		ID:          "entity:attention",
		WorkspaceID: "workspace-a",
		Title:       "Attention",
		Summary:     "State says attention is a matching entity.",
	}}); err != nil {
		t.Fatalf("writeWorkspaceKnowledgeJSON(entitiesPath) error = %v", err)
	}

	markItDownPath, err := files.MarkItDownPath("paper-a")
	if err != nil {
		t.Fatalf("MarkItDownPath() error = %v", err)
	}
	if err := os.WriteFile(markItDownPath, []byte("# Attention\n\nInput markdown also mentions attention.\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(markItDownPath) error = %v", err)
	}

	evidence, err := retrieveWorkspaceKnowledgeEvidence(files, "What is attention?")
	if err != nil {
		t.Fatalf("retrieveWorkspaceKnowledgeEvidence() error = %v", err)
	}
	if len(evidence) == 0 {
		t.Fatal("retrieveWorkspaceKnowledgeEvidence() returned no evidence")
	}
	if evidence[0].Kind != "wiki_page" {
		t.Fatalf("evidence[0].Kind = %q, want %q", evidence[0].Kind, "wiki_page")
	}
}

func TestRetrieveWorkspaceKnowledgeEvidenceFallsBackToStateBeforeInputs(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}

	indexPath, err := files.IndexPath()
	if err != nil {
		t.Fatalf("IndexPath() error = %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("# Index\n\nNo matching wiki evidence here.\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(indexPath) error = %v", err)
	}

	entitiesPath, err := files.EntitiesPath()
	if err != nil {
		t.Fatalf("EntitiesPath() error = %v", err)
	}
	if err := writeWorkspaceKnowledgeJSON(entitiesPath, []WorkspaceKnowledgeEntity{{
		ID:          "entity:transformer",
		WorkspaceID: "workspace-a",
		Title:       "Transformer",
		Summary:     "Transformer is the state-backed match for this query.",
	}}); err != nil {
		t.Fatalf("writeWorkspaceKnowledgeJSON(entitiesPath) error = %v", err)
	}

	markItDownPath, err := files.MarkItDownPath("paper-a")
	if err != nil {
		t.Fatalf("MarkItDownPath() error = %v", err)
	}
	if err := os.WriteFile(markItDownPath, []byte("# Transformer\n\nInput markdown also mentions transformer.\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(markItDownPath) error = %v", err)
	}

	evidence, err := retrieveWorkspaceKnowledgeEvidence(files, "What is a transformer?")
	if err != nil {
		t.Fatalf("retrieveWorkspaceKnowledgeEvidence() error = %v", err)
	}
	if len(evidence) == 0 {
		t.Fatal("retrieveWorkspaceKnowledgeEvidence() returned no evidence")
	}
	if evidence[0].Kind != "entity" {
		t.Fatalf("evidence[0].Kind = %q, want %q", evidence[0].Kind, "entity")
	}
}

func TestRetrieveWorkspaceKnowledgeEvidenceFallsBackToInputsAfterWikiAndStateMiss(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}

	indexPath, err := files.IndexPath()
	if err != nil {
		t.Fatalf("IndexPath() error = %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("# Index\n\nNo matching wiki evidence here.\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(indexPath) error = %v", err)
	}

	entitiesPath, err := files.EntitiesPath()
	if err != nil {
		t.Fatalf("EntitiesPath() error = %v", err)
	}
	if err := writeWorkspaceKnowledgeJSON(entitiesPath, []WorkspaceKnowledgeEntity{{
		ID:          "entity:attention",
		WorkspaceID: "workspace-a",
		Title:       "Attention",
		Summary:     "State evidence here is unrelated to the query topic.",
	}}); err != nil {
		t.Fatalf("writeWorkspaceKnowledgeJSON(entitiesPath) error = %v", err)
	}

	markItDownPath, err := files.MarkItDownPath("paper-a")
	if err != nil {
		t.Fatalf("MarkItDownPath() error = %v", err)
	}
	if err := os.WriteFile(markItDownPath, []byte("# Convolution\n\nA convolution combines local receptive fields in the input evidence.\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(markItDownPath) error = %v", err)
	}

	evidence, err := retrieveWorkspaceKnowledgeEvidence(files, "What is convolution?")
	if err != nil {
		t.Fatalf("retrieveWorkspaceKnowledgeEvidence() error = %v", err)
	}
	if len(evidence) == 0 {
		t.Fatal("retrieveWorkspaceKnowledgeEvidence() returned no evidence")
	}
	if evidence[0].Kind != "raw_excerpt" {
		t.Fatalf("evidence[0].Kind = %q, want %q", evidence[0].Kind, "raw_excerpt")
	}
}

func TestPromoteMarksCompileSummaryDirtyWhenClaimsChange(t *testing.T) {
	t.Parallel()

	paths := newTestAppPaths(t)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}
	if err := files.WriteCompileSummary(WorkspaceKnowledgeCompileSummary{
		WorkspaceID:       "workspace-a",
		IncludedSourceIDs: []string{"source:paper-a"},
		FailedSourceIDs:   []string{},
		UpdatedWikiPaths:  []string{"wiki/overview.md"},
		CompileDirty:      false,
		WikiDirty:         false,
	}); err != nil {
		t.Fatalf("WriteCompileSummary() error = %v", err)
	}
	claimsPath, err := files.ClaimsPath()
	if err != nil {
		t.Fatalf("ClaimsPath() error = %v", err)
	}
	if err := writeWorkspaceKnowledgeJSON(claimsPath, []WorkspaceKnowledgeClaim{}); err != nil {
		t.Fatalf("writeWorkspaceKnowledgeJSON(claims) error = %v", err)
	}

	service := &workspaceKnowledgeQueryService{paths: paths, files: files}
	err = service.Promote(context.Background(), WorkspaceKnowledgePromotionInput{
		WorkspaceID: "workspace-a",
		Candidates: []WorkspaceKnowledgeCandidate{{
			ID:        "candidate:claim:attention",
			Title:     "Attention claim",
			Type:      "claim",
			Summary:   "Promoted from query output.",
			SourceID:  "source:paper-a",
			PageStart: 1,
			PageEnd:   1,
			Excerpt:   "Attention improves sequence modeling.",
		}},
	})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	summary, err := files.ReadCompileSummary()
	if err != nil {
		t.Fatalf("ReadCompileSummary() error = %v", err)
	}
	if !summary.CompileDirty {
		t.Fatalf("summary.CompileDirty = %v, want true", summary.CompileDirty)
	}
	if !summary.WikiDirty {
		t.Fatalf("summary.WikiDirty = %v, want true", summary.WikiDirty)
	}
}
