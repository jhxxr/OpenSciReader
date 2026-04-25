package main

import (
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
