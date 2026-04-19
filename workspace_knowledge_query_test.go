package main

import (
	"context"
	"os"
	"path/filepath"
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

func TestWorkspaceKnowledgePromotePreservesExistingClaimOnDuplicateCandidate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	claimsPath, err := files.ClaimsPath()
	if err != nil {
		t.Fatalf("ClaimsPath error: %v", err)
	}
	existingClaims := []WorkspaceKnowledgeClaim{{
		ID:          "candidate:claim:contrastive-memory-core",
		WorkspaceID: "workspace-a",
		Title:       "Contrastive Memory is the main method",
		Type:        "claim",
		Summary:     "Existing summary should survive duplicate promotion",
		EntityIDs:   []string{"entity:method:contrastive-memory"},
		SourceRefs: []WorkspaceKnowledgeSourceRef{{
			SourceID:  "source:paper-a",
			PageStart: 7,
			PageEnd:   8,
			Excerpt:   "Existing excerpt should survive duplicate promotion",
		}},
		Origin:     "promotion",
		Status:     "promoted",
		Confidence: 1,
		CreatedAt:  "2026-01-01T00:00:00Z",
		UpdatedAt:  "2026-01-02T00:00:00Z",
	}}
	if err := writeWorkspaceKnowledgeJSON(claimsPath, existingClaims); err != nil {
		t.Fatalf("write claims error: %v", err)
	}

	service := workspaceKnowledgeQueryService{files: files}
	if err := service.Promote(context.Background(), WorkspaceKnowledgePromotionInput{
		WorkspaceID: "workspace-a",
		Candidates: []WorkspaceKnowledgeCandidate{{
			ID:    "candidate:claim:contrastive-memory-core",
			Title: "Contrastive Memory is the main method",
			Type:  "claim",
		}},
	}); err != nil {
		t.Fatalf("Promote error: %v", err)
	}

	var gotClaims []WorkspaceKnowledgeClaim
	if err := readWorkspaceKnowledgeJSON(claimsPath, &gotClaims); err != nil {
		t.Fatalf("read claims error: %v", err)
	}
	if len(gotClaims) != 1 {
		t.Fatalf("len(claims) = %d, want 1", len(gotClaims))
	}
	got := gotClaims[0]
	if got.Summary != existingClaims[0].Summary {
		t.Fatalf("summary = %q, want %q", got.Summary, existingClaims[0].Summary)
	}
	if got.UpdatedAt != existingClaims[0].UpdatedAt {
		t.Fatalf("updatedAt = %q, want %q", got.UpdatedAt, existingClaims[0].UpdatedAt)
	}
	if len(got.EntityIDs) != 1 || got.EntityIDs[0] != existingClaims[0].EntityIDs[0] {
		t.Fatalf("entityIds = %#v, want %#v", got.EntityIDs, existingClaims[0].EntityIDs)
	}
	if len(got.SourceRefs) != 1 || got.SourceRefs[0].Excerpt != existingClaims[0].SourceRefs[0].Excerpt {
		t.Fatalf("sourceRefs = %#v, want %#v", got.SourceRefs, existingClaims[0].SourceRefs)
	}
}

func TestWorkspaceKnowledgeQueryFallsBackToWikiWhenSchemaUnreadable(t *testing.T) {
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
	if err := os.WriteFile(entitiesPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("write invalid entities error: %v", err)
	}

	conceptPath, err := files.ConceptWikiPath("contrastive-memory")
	if err != nil {
		t.Fatalf("ConceptWikiPath error: %v", err)
	}
	if err := writeWorkspaceKnowledgeMarkdown(conceptPath, "# Contrastive Memory\n\nMemory-augmented retrieval method\n"); err != nil {
		t.Fatalf("write concept wiki error: %v", err)
	}

	service := workspaceKnowledgeQueryService{
		files: files,
		llm: &stubWorkspaceKnowledgeQueryLLM{
			result: WorkspaceKnowledgeQueryResult{
				Answer: "Wiki evidence answered the question.",
			},
		},
	}

	result, err := service.Query(context.Background(), WorkspaceKnowledgeQueryInput{
		WorkspaceID: "workspace-a",
		ProviderID:  1,
		ModelID:     2,
		Question:    "What is Contrastive Memory?",
	})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if len(result.Evidence) == 0 || result.Evidence[0].Kind != "wiki" {
		t.Fatalf("evidence = %#v, want wiki fallback evidence", result.Evidence)
	}
}

func TestWorkspaceKnowledgeQueryFallsBackToRawWhenWikiUnreadable(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	overviewPath, err := files.OverviewPath()
	if err != nil {
		t.Fatalf("OverviewPath error: %v", err)
	}
	if err := os.MkdirAll(overviewPath, 0o700); err != nil {
		t.Fatalf("make overview directory error: %v", err)
	}

	extractPath, err := files.ExtractPath("paper-a")
	if err != nil {
		t.Fatalf("ExtractPath error: %v", err)
	}
	if err := writeWorkspaceKnowledgeMarkdown(extractPath, "Contrastive Memory is described directly in the raw extract."); err != nil {
		t.Fatalf("write extract error: %v", err)
	}
	if err := files.WriteSources([]WorkspaceKnowledgeSource{{
		ID:           "source:paper-a",
		WorkspaceID:  "workspace-a",
		Title:        "Paper A",
		Slug:         "paper-a",
		Kind:         "pdf",
		AbsolutePath: filepath.Join(tempDir, "paper-a.pdf"),
		ExtractPath:  extractPath,
		Status:       "ready",
	}}); err != nil {
		t.Fatalf("WriteSources error: %v", err)
	}

	service := workspaceKnowledgeQueryService{
		files: files,
		llm: &stubWorkspaceKnowledgeQueryLLM{
			result: WorkspaceKnowledgeQueryResult{
				Answer: "Raw evidence answered the question.",
			},
		},
	}

	result, err := service.Query(context.Background(), WorkspaceKnowledgeQueryInput{
		WorkspaceID: "workspace-a",
		ProviderID:  1,
		ModelID:     2,
		Question:    "What is Contrastive Memory?",
	})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if len(result.Evidence) == 0 || result.Evidence[0].Kind != "extract" {
		t.Fatalf("evidence = %#v, want raw extract fallback evidence", result.Evidence)
	}
}

func TestWorkspaceKnowledgePromoteCanonicalizesSemanticallyIdenticalClaimIDs(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	service := workspaceKnowledgeQueryService{files: files}
	firstCandidate := WorkspaceKnowledgeCandidate{
		ID:      "candidate:claim:first-session-id",
		Title:   "Contrastive Memory is the main method",
		Type:    "claim",
		Summary: "The workspace centers on Contrastive Memory",
		SourceRefs: []WorkspaceKnowledgeSourceRef{{
			SourceID:  "source:paper-a",
			PageStart: 3,
			PageEnd:   3,
			Excerpt:   "Contrastive Memory excerpt",
		}},
	}
	if err := service.Promote(context.Background(), WorkspaceKnowledgePromotionInput{
		WorkspaceID: "workspace-a",
		Candidates:  []WorkspaceKnowledgeCandidate{firstCandidate},
	}); err != nil {
		t.Fatalf("first Promote error: %v", err)
	}

	claimsPath, err := files.ClaimsPath()
	if err != nil {
		t.Fatalf("ClaimsPath error: %v", err)
	}
	var firstClaims []WorkspaceKnowledgeClaim
	if err := readWorkspaceKnowledgeJSON(claimsPath, &firstClaims); err != nil {
		t.Fatalf("read first claims error: %v", err)
	}
	if len(firstClaims) != 1 {
		t.Fatalf("len(firstClaims) = %d, want 1", len(firstClaims))
	}
	firstID := firstClaims[0].ID
	if firstID == firstCandidate.ID {
		t.Fatalf("claim id = %q, want canonical backend id instead of model id", firstID)
	}

	secondCandidate := firstCandidate
	secondCandidate.ID = "candidate:claim:second-session-id"
	if err := service.Promote(context.Background(), WorkspaceKnowledgePromotionInput{
		WorkspaceID: "workspace-a",
		Candidates:  []WorkspaceKnowledgeCandidate{secondCandidate},
	}); err != nil {
		t.Fatalf("second Promote error: %v", err)
	}

	var secondClaims []WorkspaceKnowledgeClaim
	if err := readWorkspaceKnowledgeJSON(claimsPath, &secondClaims); err != nil {
		t.Fatalf("read second claims error: %v", err)
	}
	if len(secondClaims) != 1 {
		t.Fatalf("len(secondClaims) = %d, want 1", len(secondClaims))
	}
	if secondClaims[0].ID != firstID {
		t.Fatalf("claim id = %q, want stable canonical id %q", secondClaims[0].ID, firstID)
	}
}

func TestWorkspaceKnowledgePromoteIsIdempotentForIdenticalSourceRefs(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	service := workspaceKnowledgeQueryService{files: files}
	candidate := WorkspaceKnowledgeCandidate{
		ID:      "candidate:claim:first-session-id",
		Title:   "Contrastive Memory is the main method",
		Type:    "claim",
		Summary: "The workspace centers on Contrastive Memory",
		SourceRefs: []WorkspaceKnowledgeSourceRef{{
			SourceID:  "source:paper-a",
			PageStart: 3,
			PageEnd:   3,
			Excerpt:   "Contrastive Memory excerpt",
		}},
	}
	if err := service.Promote(context.Background(), WorkspaceKnowledgePromotionInput{
		WorkspaceID: "workspace-a",
		Candidates:  []WorkspaceKnowledgeCandidate{candidate},
	}); err != nil {
		t.Fatalf("first Promote error: %v", err)
	}

	claimsPath, err := files.ClaimsPath()
	if err != nil {
		t.Fatalf("ClaimsPath error: %v", err)
	}
	var firstClaims []WorkspaceKnowledgeClaim
	if err := readWorkspaceKnowledgeJSON(claimsPath, &firstClaims); err != nil {
		t.Fatalf("read first claims error: %v", err)
	}
	if len(firstClaims) != 1 {
		t.Fatalf("len(firstClaims) = %d, want 1", len(firstClaims))
	}
	first := firstClaims[0]

	candidate.ID = "candidate:claim:second-session-id"
	if err := service.Promote(context.Background(), WorkspaceKnowledgePromotionInput{
		WorkspaceID: "workspace-a",
		Candidates:  []WorkspaceKnowledgeCandidate{candidate},
	}); err != nil {
		t.Fatalf("second Promote error: %v", err)
	}

	var secondClaims []WorkspaceKnowledgeClaim
	if err := readWorkspaceKnowledgeJSON(claimsPath, &secondClaims); err != nil {
		t.Fatalf("read second claims error: %v", err)
	}
	if len(secondClaims) != 1 {
		t.Fatalf("len(secondClaims) = %d, want 1", len(secondClaims))
	}
	second := secondClaims[0]
	if second.UpdatedAt != first.UpdatedAt {
		t.Fatalf("updatedAt = %q, want unchanged %q", second.UpdatedAt, first.UpdatedAt)
	}
	if len(second.SourceRefs) != 1 {
		t.Fatalf("len(sourceRefs) = %d, want 1", len(second.SourceRefs))
	}
	if second.SourceRefs[0] != candidate.SourceRefs[0] {
		t.Fatalf("sourceRef = %#v, want %#v", second.SourceRefs[0], candidate.SourceRefs[0])
	}
}

func TestWorkspaceKnowledgePromotePreservesCanonicalIDFromRichToSparseCandidate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	service := workspaceKnowledgeQueryService{files: files}
	richCandidate := WorkspaceKnowledgeCandidate{
		ID:        "candidate:claim:rich-session-id",
		Title:     "Contrastive Memory is the main method",
		Type:      "claim",
		Summary:   "The workspace centers on Contrastive Memory",
		EntityIDs: []string{"entity:method:contrastive-memory"},
		SourceRefs: []WorkspaceKnowledgeSourceRef{{
			SourceID:  "source:paper-a",
			PageStart: 3,
			PageEnd:   3,
			Excerpt:   "Contrastive Memory excerpt",
		}},
	}
	if err := service.Promote(context.Background(), WorkspaceKnowledgePromotionInput{
		WorkspaceID: "workspace-a",
		Candidates:  []WorkspaceKnowledgeCandidate{richCandidate},
	}); err != nil {
		t.Fatalf("rich Promote error: %v", err)
	}

	claimsPath, err := files.ClaimsPath()
	if err != nil {
		t.Fatalf("ClaimsPath error: %v", err)
	}
	var firstClaims []WorkspaceKnowledgeClaim
	if err := readWorkspaceKnowledgeJSON(claimsPath, &firstClaims); err != nil {
		t.Fatalf("read first claims error: %v", err)
	}
	if len(firstClaims) != 1 {
		t.Fatalf("len(firstClaims) = %d, want 1", len(firstClaims))
	}
	first := firstClaims[0]

	sparseCandidate := WorkspaceKnowledgeCandidate{
		ID:      "candidate:claim:sparse-session-id",
		Title:   richCandidate.Title,
		Type:    "claim",
		Summary: richCandidate.Summary,
	}
	if err := service.Promote(context.Background(), WorkspaceKnowledgePromotionInput{
		WorkspaceID: "workspace-a",
		Candidates:  []WorkspaceKnowledgeCandidate{sparseCandidate},
	}); err != nil {
		t.Fatalf("sparse Promote error: %v", err)
	}

	var secondClaims []WorkspaceKnowledgeClaim
	if err := readWorkspaceKnowledgeJSON(claimsPath, &secondClaims); err != nil {
		t.Fatalf("read second claims error: %v", err)
	}
	if len(secondClaims) != 1 {
		t.Fatalf("len(secondClaims) = %d, want 1", len(secondClaims))
	}
	if secondClaims[0].ID != first.ID {
		t.Fatalf("claim id = %q, want preserved canonical id %q", secondClaims[0].ID, first.ID)
	}
}

func TestWorkspaceKnowledgePromotePreservesCanonicalIDFromSparseToRichCandidate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	service := workspaceKnowledgeQueryService{files: files}
	sparseCandidate := WorkspaceKnowledgeCandidate{
		ID:      "candidate:claim:sparse-session-id",
		Title:   "Contrastive Memory is the main method",
		Type:    "claim",
		Summary: "The workspace centers on Contrastive Memory",
	}
	if err := service.Promote(context.Background(), WorkspaceKnowledgePromotionInput{
		WorkspaceID: "workspace-a",
		Candidates:  []WorkspaceKnowledgeCandidate{sparseCandidate},
	}); err != nil {
		t.Fatalf("sparse Promote error: %v", err)
	}

	claimsPath, err := files.ClaimsPath()
	if err != nil {
		t.Fatalf("ClaimsPath error: %v", err)
	}
	var firstClaims []WorkspaceKnowledgeClaim
	if err := readWorkspaceKnowledgeJSON(claimsPath, &firstClaims); err != nil {
		t.Fatalf("read first claims error: %v", err)
	}
	if len(firstClaims) != 1 {
		t.Fatalf("len(firstClaims) = %d, want 1", len(firstClaims))
	}
	first := firstClaims[0]

	richCandidate := WorkspaceKnowledgeCandidate{
		ID:        "candidate:claim:rich-session-id",
		Title:     sparseCandidate.Title,
		Type:      "claim",
		Summary:   sparseCandidate.Summary,
		EntityIDs: []string{"entity:method:contrastive-memory"},
		SourceRefs: []WorkspaceKnowledgeSourceRef{{
			SourceID:  "source:paper-a",
			PageStart: 3,
			PageEnd:   3,
			Excerpt:   "Contrastive Memory excerpt",
		}},
	}
	if err := service.Promote(context.Background(), WorkspaceKnowledgePromotionInput{
		WorkspaceID: "workspace-a",
		Candidates:  []WorkspaceKnowledgeCandidate{richCandidate},
	}); err != nil {
		t.Fatalf("rich Promote error: %v", err)
	}

	var secondClaims []WorkspaceKnowledgeClaim
	if err := readWorkspaceKnowledgeJSON(claimsPath, &secondClaims); err != nil {
		t.Fatalf("read second claims error: %v", err)
	}
	if len(secondClaims) != 1 {
		t.Fatalf("len(secondClaims) = %d, want 1", len(secondClaims))
	}
	if secondClaims[0].ID != first.ID {
		t.Fatalf("claim id = %q, want preserved canonical id %q", secondClaims[0].ID, first.ID)
	}
}

func TestWorkspaceKnowledgeQueryIgnoresConversationLogAppendFailure(t *testing.T) {
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
	if err := writeWorkspaceKnowledgeJSON(entitiesPath, []WorkspaceKnowledgeEntity{{
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
	}}); err != nil {
		t.Fatalf("write entities error: %v", err)
	}

	schemaDir, err := files.schemaDir()
	if err != nil {
		t.Fatalf("schemaDir error: %v", err)
	}
	logPath := filepath.Join(schemaDir, "conversation-log.jsonl")
	if err := os.MkdirAll(logPath, 0o700); err != nil {
		t.Fatalf("make log directory error: %v", err)
	}

	service := workspaceKnowledgeQueryService{
		files: files,
		llm: &stubWorkspaceKnowledgeQueryLLM{
			result: WorkspaceKnowledgeQueryResult{
				Answer: "Contrastive Memory is the main method in the workspace.",
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
	if got := result.Answer; got != "Contrastive Memory is the main method in the workspace." {
		t.Fatalf("answer = %q", got)
	}
}

func TestWorkspaceKnowledgeQueryReturnsSchemaErrorWhenNoFallbackEvidence(t *testing.T) {
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
	if err := os.WriteFile(entitiesPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("write invalid entities error: %v", err)
	}

	service := workspaceKnowledgeQueryService{
		files: files,
		llm:   &stubWorkspaceKnowledgeQueryLLM{},
	}

	_, err = service.Query(context.Background(), WorkspaceKnowledgeQueryInput{
		WorkspaceID: "workspace-a",
		ProviderID:  1,
		ModelID:     2,
		Question:    "What is Contrastive Memory?",
	})
	if err == nil || !strings.Contains(err.Error(), "entities.json") {
		t.Fatalf("Query error = %v, want schema read error", err)
	}
}

func TestWorkspaceKnowledgeQueryReturnsWikiErrorWhenNoRawFallbackEvidence(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	overviewPath, err := files.OverviewPath()
	if err != nil {
		t.Fatalf("OverviewPath error: %v", err)
	}
	if err := os.MkdirAll(overviewPath, 0o700); err != nil {
		t.Fatalf("make overview directory error: %v", err)
	}

	service := workspaceKnowledgeQueryService{
		files: files,
		llm:   &stubWorkspaceKnowledgeQueryLLM{},
	}

	_, err = service.Query(context.Background(), WorkspaceKnowledgeQueryInput{
		WorkspaceID: "workspace-a",
		ProviderID:  1,
		ModelID:     2,
		Question:    "What is Contrastive Memory?",
	})
	if err == nil || !strings.Contains(err.Error(), "wiki") {
		t.Fatalf("Query error = %v, want wiki read error", err)
	}
}

func TestWorkspaceKnowledgeQueryFindsRawEvidenceBeyondExcerptWindow(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := newWorkspaceKnowledgeTestPaths(tempDir)
	files := newWorkspaceKnowledgeFiles(paths, "workspace-a")
	if err := files.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout error: %v", err)
	}

	extractPath, err := files.ExtractPath("paper-a")
	if err != nil {
		t.Fatalf("ExtractPath error: %v", err)
	}
	longExtract := "# Paper A\n\n" + strings.Repeat("filler ", 130) + "\nlate-anchor evidence appears here\n"
	if err := writeWorkspaceKnowledgeMarkdown(extractPath, longExtract); err != nil {
		t.Fatalf("write extract error: %v", err)
	}
	if err := files.WriteSources([]WorkspaceKnowledgeSource{{
		ID:           "source:paper-a",
		WorkspaceID:  "workspace-a",
		Title:        "Paper A",
		Slug:         "paper-a",
		Kind:         "pdf",
		AbsolutePath: filepath.Join(tempDir, "paper-a.pdf"),
		ExtractPath:  extractPath,
		Status:       "ready",
	}}); err != nil {
		t.Fatalf("WriteSources error: %v", err)
	}

	service := workspaceKnowledgeQueryService{
		files: files,
		llm: &stubWorkspaceKnowledgeQueryLLM{
			result: WorkspaceKnowledgeQueryResult{
				Answer: "The late anchor was found in the raw extract.",
			},
		},
	}

	result, err := service.Query(context.Background(), WorkspaceKnowledgeQueryInput{
		WorkspaceID: "workspace-a",
		ProviderID:  1,
		ModelID:     2,
		Question:    "Where is the late anchor evidence?",
	})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if len(result.Evidence) == 0 || result.Evidence[0].Kind != "extract" {
		t.Fatalf("evidence = %#v, want raw extract evidence", result.Evidence)
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
