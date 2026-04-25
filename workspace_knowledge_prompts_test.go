package main

import (
	"strings"
	"testing"
)

func TestBuildWorkspaceKnowledgeBySourcePromptUsesCurrentSourceContract(t *testing.T) {
	t.Parallel()

	prompt := buildWorkspaceKnowledgeBySourcePrompt(
		Workspace{ID: "workspace-a", Name: "Workspace A"},
		WorkspaceKnowledgeSource{
			ID:         "source:paper-a",
			Title:      "Paper A",
			Slug:       "paper-a",
			Kind:       "pdf",
			DocumentID: "doc-a",
		},
		"# Markdown",
	)

	for _, field := range []string{"sourcePath", "markItDownPath", "markItDownStatus", "extractStatus", "lastIngestAt", "lastSuccessAt", "lastError"} {
		if !strings.Contains(prompt, field) {
			t.Fatalf("prompt missing current field %q", field)
		}
	}

	for _, field := range []string{"absolutePath", "extractPath", "status", "lastScanAt"} {
		if strings.Contains(prompt, field) {
			t.Fatalf("prompt unexpectedly contains legacy field %q", field)
		}
	}
}
