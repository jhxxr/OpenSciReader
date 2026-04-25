package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const (
	workspaceKnowledgeInsufficientEvidenceAnswer = "Insufficient evidence in workspace knowledge to answer this question."
	maxWorkspaceKnowledgeQueryEvidence           = 6
)

type workspaceKnowledgeQueryLLM interface {
	GenerateWorkspaceKnowledgeQuery(ctx context.Context, providerID, modelID int64, prompt string) (WorkspaceKnowledgeQueryResult, error)
}

type workspaceKnowledgeQueryService struct {
	paths appPaths
	files workspaceKnowledgeFiles
	llm   workspaceKnowledgeQueryLLM
}

type workspaceKnowledgeConversationLogEntry struct {
	Type             string                        `json:"type"`
	Timestamp        string                        `json:"timestamp"`
	WorkspaceID      string                        `json:"workspaceId"`
	ProviderID       int64                         `json:"providerId,omitempty"`
	ModelID          int64                         `json:"modelId,omitempty"`
	Question         string                        `json:"question,omitempty"`
	EvidenceIDs      []string                      `json:"evidenceIds,omitempty"`
	Answer           string                        `json:"answer,omitempty"`
	Candidates       []WorkspaceKnowledgeCandidate `json:"candidates,omitempty"`
	PromotedClaimIDs []string                      `json:"promotedClaimIds,omitempty"`
}

type workspaceKnowledgeEvidenceCandidate struct {
	hit        WorkspaceKnowledgeEvidenceHit
	searchText string
}

func newWorkspaceKnowledgeQueryService(paths appPaths, llm workspaceKnowledgeQueryLLM) *workspaceKnowledgeQueryService {
	return &workspaceKnowledgeQueryService{
		paths: paths,
		llm:   llm,
	}
}

func (s *workspaceKnowledgeQueryService) Query(ctx context.Context, input WorkspaceKnowledgeQueryInput) (WorkspaceKnowledgeQueryResult, error) {
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID == "" {
		return WorkspaceKnowledgeQueryResult{}, fmt.Errorf("workspace id is required")
	}
	question := strings.TrimSpace(input.Question)
	if question == "" {
		return WorkspaceKnowledgeQueryResult{}, fmt.Errorf("question is required")
	}

	files := s.workspaceFiles(workspaceID)
	if err := files.EnsureLayout(); err != nil {
		return WorkspaceKnowledgeQueryResult{}, err
	}

	evidence, err := retrieveWorkspaceKnowledgeEvidence(files, question)
	if err != nil {
		return WorkspaceKnowledgeQueryResult{}, err
	}

	result := WorkspaceKnowledgeQueryResult{
		Answer:     workspaceKnowledgeInsufficientEvidenceAnswer,
		Evidence:   evidence,
		Candidates: []WorkspaceKnowledgeCandidate{},
	}
	if len(evidence) > 0 {
		if s.llm == nil {
			return WorkspaceKnowledgeQueryResult{}, fmt.Errorf("workspace knowledge query llm is unavailable")
		}
		payload, err := s.llm.GenerateWorkspaceKnowledgeQuery(ctx, input.ProviderID, input.ModelID, buildWorkspaceKnowledgeQueryPrompt(input, evidence))
		if err != nil {
			return WorkspaceKnowledgeQueryResult{}, err
		}
		answer := strings.TrimSpace(payload.Answer)
		if answer != "" {
			result.Answer = answer
		}
		result.Candidates = normalizeWorkspaceKnowledgeCandidates(payload.Candidates)
	}

	appendWorkspaceKnowledgeConversationLogBestEffort(files, workspaceKnowledgeConversationLogEntry{
		Type:        "query",
		Timestamp:   nowRFC3339(),
		WorkspaceID: workspaceID,
		ProviderID:  input.ProviderID,
		ModelID:     input.ModelID,
		Question:    question,
		EvidenceIDs: workspaceKnowledgeEvidenceIDs(result.Evidence),
		Answer:      result.Answer,
		Candidates:  result.Candidates,
	})

	return result, nil
}

func (s *workspaceKnowledgeQueryService) Promote(_ context.Context, input WorkspaceKnowledgePromotionInput) error {
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID == "" {
		return fmt.Errorf("workspace id is required")
	}

	files := s.workspaceFiles(workspaceID)
	if err := files.EnsureLayout(); err != nil {
		return err
	}

	existingClaims, err := readWorkspaceKnowledgeClaims(files)
	if err != nil {
		return err
	}

	claimsByID := make(map[string]WorkspaceKnowledgeClaim, len(existingClaims))
	for _, claim := range existingClaims {
		claimsByID[claim.ID] = claim
	}

	promotedClaimIDs := make([]string, 0, len(input.Candidates))
	claimsChanged := false
	now := nowRFC3339()
	for _, candidate := range normalizeWorkspaceKnowledgeCandidates(input.Candidates) {
		if strings.TrimSpace(candidate.Type) != "claim" {
			continue
		}

		canonicalID := canonicalWorkspaceKnowledgeClaimID(candidate)
		if canonicalID == "" {
			continue
		}

		existingClaim, existingKey, err := findWorkspaceKnowledgeClaimForCandidate(claimsByID, canonicalID, candidate)
		if err != nil {
			return err
		}
		if existingKey != "" && existingKey != canonicalID {
			delete(claimsByID, existingKey)
			claimsChanged = true
		}

		claimID := canonicalID
		if strings.TrimSpace(existingClaim.ID) != "" {
			claimID = existingClaim.ID
		}

		claim, changed := mergeWorkspaceKnowledgePromotedClaim(existingClaim, claimID, workspaceID, WorkspaceKnowledgeCandidate{
			ID:         claimID,
			Title:      firstNonEmptyText(strings.TrimSpace(candidate.Title), claimID),
			Type:       candidate.Type,
			Summary:    candidate.Summary,
			EntityIDs:  candidate.EntityIDs,
			SourceRefs: candidate.SourceRefs,
			SourceID:   candidate.SourceID,
			PageStart:  candidate.PageStart,
			PageEnd:    candidate.PageEnd,
			Excerpt:    candidate.Excerpt,
		}, now)
		claimsByID[claimID] = claim
		claimsChanged = claimsChanged || changed
		promotedClaimIDs = append(promotedClaimIDs, claimID)
	}

	claims := make([]WorkspaceKnowledgeClaim, 0, len(claimsByID))
	for _, claim := range claimsByID {
		claims = append(claims, claim)
	}
	sort.Slice(claims, func(i, j int) bool {
		return lessClaim(claims[i], claims[j])
	})

	if claimsChanged {
		claimsPath, err := files.ClaimsPath()
		if err != nil {
			return err
		}
		if err := writeWorkspaceKnowledgeJSON(claimsPath, claims); err != nil {
			return err
		}
	}

	appendWorkspaceKnowledgeConversationLogBestEffort(files, workspaceKnowledgeConversationLogEntry{
		Type:             "promotion",
		Timestamp:        now,
		WorkspaceID:      workspaceID,
		PromotedClaimIDs: promotedClaimIDs,
	})
	return nil
}

func (s *workspaceKnowledgeQueryService) ListEntities(_ context.Context, workspaceID string) ([]WorkspaceKnowledgeEntity, error) {
	files := s.workspaceFiles(strings.TrimSpace(workspaceID))
	if err := files.EnsureLayout(); err != nil {
		return nil, err
	}
	return readWorkspaceKnowledgeEntities(files)
}

func (s *workspaceKnowledgeQueryService) ListClaims(_ context.Context, workspaceID string) ([]WorkspaceKnowledgeClaim, error) {
	files := s.workspaceFiles(strings.TrimSpace(workspaceID))
	if err := files.EnsureLayout(); err != nil {
		return nil, err
	}
	return readWorkspaceKnowledgeClaims(files)
}

func (s *workspaceKnowledgeQueryService) ListTasks(_ context.Context, workspaceID string) ([]WorkspaceKnowledgeTask, error) {
	files := s.workspaceFiles(strings.TrimSpace(workspaceID))
	if err := files.EnsureLayout(); err != nil {
		return nil, err
	}
	return readWorkspaceKnowledgeTasks(files)
}

func (s *workspaceKnowledgeQueryService) workspaceFiles(workspaceID string) workspaceKnowledgeFiles {
	if strings.TrimSpace(s.files.workspaceID) != "" && (strings.TrimSpace(workspaceID) == "" || strings.TrimSpace(s.files.workspaceID) == strings.TrimSpace(workspaceID)) {
		return s.files
	}
	return newWorkspaceKnowledgeFiles(s.paths, workspaceID)
}

func buildWorkspaceKnowledgeQueryPrompt(input WorkspaceKnowledgeQueryInput, evidence []WorkspaceKnowledgeEvidenceHit) string {
	var builder strings.Builder
	builder.WriteString("Answer the workspace knowledge question using only the supplied evidence.\n")
	builder.WriteString("Return valid JSON only with this shape:\n")
	builder.WriteString("{\n")
	builder.WriteString("  \"answer\": \"...\",\n")
	builder.WriteString("  \"candidates\": [\n")
	builder.WriteString("    {\n")
	builder.WriteString("      \"id\": \"candidate:claim:...\",\n")
	builder.WriteString("      \"title\": \"...\",\n")
	builder.WriteString("      \"type\": \"claim\",\n")
	builder.WriteString("      \"summary\": \"...\",\n")
	builder.WriteString("      \"aliases\": [],\n")
	builder.WriteString("      \"entityIds\": [],\n")
	builder.WriteString("      \"priority\": \"\",\n")
	builder.WriteString("      \"sourceId\": \"\",\n")
	builder.WriteString("      \"pageStart\": 0,\n")
	builder.WriteString("      \"pageEnd\": 0,\n")
	builder.WriteString("      \"excerpt\": \"\"\n")
	builder.WriteString("    }\n")
	builder.WriteString("  ]\n")
	builder.WriteString("}\n")
	builder.WriteString("Rules:\n")
	builder.WriteString("- Ground the answer in the evidence only.\n")
	builder.WriteString("- If the evidence is insufficient, say so plainly.\n")
	builder.WriteString("- Use an empty candidates array when there is nothing worth promoting.\n")
	builder.WriteString("- Prefer concise claim candidates with stable-looking ids.\n\n")
	builder.WriteString("Workspace:\n")
	builder.WriteString("- workspaceId: ")
	builder.WriteString(strings.TrimSpace(input.WorkspaceID))
	builder.WriteString("\n")
	builder.WriteString("- question: ")
	builder.WriteString(strings.TrimSpace(input.Question))
	builder.WriteString("\n\nEvidence:\n")
	for _, hit := range evidence {
		builder.WriteString("- kind: ")
		builder.WriteString(hit.Kind)
		builder.WriteString("\n")
		builder.WriteString("  id: ")
		builder.WriteString(hit.ID)
		builder.WriteString("\n")
		builder.WriteString("  title: ")
		builder.WriteString(strings.TrimSpace(hit.Title))
		builder.WriteString("\n")
		builder.WriteString("  summary: ")
		builder.WriteString(strings.TrimSpace(hit.Summary))
		builder.WriteString("\n")
		builder.WriteString("  excerpt: ")
		builder.WriteString(strings.TrimSpace(hit.Excerpt))
		builder.WriteString("\n")
		builder.WriteString("  sourceId: ")
		builder.WriteString(firstSourceID(hit.SourceRefs))
		builder.WriteString("\n")
		builder.WriteString("  pageStart: ")
		builder.WriteString(fmt.Sprintf("%d", firstPageStart(hit.SourceRefs)))
		builder.WriteString("\n")
		builder.WriteString("  pageEnd: ")
		builder.WriteString(fmt.Sprintf("%d", firstPageEnd(hit.SourceRefs)))
		builder.WriteString("\n")
		builder.WriteString("  sourceExcerpt: ")
		builder.WriteString(firstNonEmptyText(firstExcerpt(hit.SourceRefs), strings.TrimSpace(hit.Excerpt)))
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func firstSourceID(sourceRefs []WorkspaceKnowledgeSourceRef) string {
	if len(sourceRefs) == 0 {
		return ""
	}
	return strings.TrimSpace(sourceRefs[0].SourceID)
}

func firstPageStart(sourceRefs []WorkspaceKnowledgeSourceRef) int {
	if len(sourceRefs) == 0 {
		return 0
	}
	return sourceRefs[0].PageStart
}

func firstPageEnd(sourceRefs []WorkspaceKnowledgeSourceRef) int {
	if len(sourceRefs) == 0 {
		return 0
	}
	return sourceRefs[0].PageEnd
}

func firstExcerpt(sourceRefs []WorkspaceKnowledgeSourceRef) string {
	if len(sourceRefs) == 0 {
		return ""
	}
	return strings.TrimSpace(sourceRefs[0].Excerpt)
}

func readWorkspaceKnowledgeEntities(files workspaceKnowledgeFiles) ([]WorkspaceKnowledgeEntity, error) {
	entitiesPath, err := files.EntitiesPath()
	if err != nil {
		return nil, err
	}

	var entities []WorkspaceKnowledgeEntity
	if err := readWorkspaceKnowledgeOptionalJSON(entitiesPath, &entities); err != nil {
		return nil, err
	}
	if entities == nil {
		return []WorkspaceKnowledgeEntity{}, nil
	}
	sort.Slice(entities, func(i, j int) bool {
		return lessEntity(entities[i], entities[j])
	})
	return entities, nil
}

func readWorkspaceKnowledgeClaims(files workspaceKnowledgeFiles) ([]WorkspaceKnowledgeClaim, error) {
	claimsPath, err := files.ClaimsPath()
	if err != nil {
		return nil, err
	}

	var claims []WorkspaceKnowledgeClaim
	if err := readWorkspaceKnowledgeOptionalJSON(claimsPath, &claims); err != nil {
		return nil, err
	}
	if claims == nil {
		return []WorkspaceKnowledgeClaim{}, nil
	}
	sort.Slice(claims, func(i, j int) bool {
		return lessClaim(claims[i], claims[j])
	})
	return claims, nil
}

func readWorkspaceKnowledgeTasks(files workspaceKnowledgeFiles) ([]WorkspaceKnowledgeTask, error) {
	tasksPath, err := files.TasksPath()
	if err != nil {
		return nil, err
	}

	var tasks []WorkspaceKnowledgeTask
	if err := readWorkspaceKnowledgeOptionalJSON(tasksPath, &tasks); err != nil {
		return nil, err
	}
	if tasks == nil {
		return []WorkspaceKnowledgeTask{}, nil
	}
	sort.Slice(tasks, func(i, j int) bool {
		return lessTask(tasks[i], tasks[j])
	})
	return tasks, nil
}

func readWorkspaceKnowledgeOptionalJSON(path string, target any) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat workspace knowledge file %s: %w", path, err)
	}
	return readWorkspaceKnowledgeJSON(path, target)
}

func retrieveWorkspaceKnowledgeEvidence(files workspaceKnowledgeFiles, question string) ([]WorkspaceKnowledgeEvidenceHit, error) {
	wikiEvidence, err := retrieveWorkspaceKnowledgeWikiEvidence(files, question)
	if err == nil && len(wikiEvidence) > 0 {
		return wikiEvidence, nil
	}
	wikiErr := err

	stateEvidence, err := retrieveWorkspaceKnowledgeStateEvidence(files, question)
	if err == nil && len(stateEvidence) > 0 {
		return stateEvidence, nil
	}
	stateErr := err

	inputEvidence, inputErr := retrieveWorkspaceKnowledgeInputEvidence(files, question)
	if inputErr == nil && len(inputEvidence) > 0 {
		return inputEvidence, nil
	}
	if inputErr != nil {
		return nil, inputErr
	}
	if wikiErr != nil {
		return nil, wikiErr
	}
	if stateErr != nil {
		return nil, stateErr
	}
	return inputEvidence, nil
}

func retrieveWorkspaceKnowledgeStateEvidence(files workspaceKnowledgeFiles, question string) ([]WorkspaceKnowledgeEvidenceHit, error) {
	hits := make([]WorkspaceKnowledgeEvidenceHit, 0)
	var firstErr error

	entities, err := readWorkspaceKnowledgeEntities(files)
	if err != nil {
		firstErr = err
	}
	for _, entity := range entities {
		hits = append(hits, WorkspaceKnowledgeEvidenceHit{
			Kind:       "entity",
			ID:         entity.ID,
			Title:      entity.Title,
			Summary:    entity.Summary,
			Excerpt:    firstExcerpt(entity.SourceRefs),
			SourceRefs: append([]WorkspaceKnowledgeSourceRef(nil), entity.SourceRefs...),
		})
	}

	claims, err := readWorkspaceKnowledgeClaims(files)
	if err != nil && firstErr == nil {
		firstErr = err
	}
	for _, claim := range claims {
		hits = append(hits, WorkspaceKnowledgeEvidenceHit{
			Kind:       "claim",
			ID:         claim.ID,
			Title:      claim.Title,
			Summary:    claim.Summary,
			Excerpt:    firstExcerpt(claim.SourceRefs),
			SourceRefs: append([]WorkspaceKnowledgeSourceRef(nil), claim.SourceRefs...),
		})
	}

	tasks, err := readWorkspaceKnowledgeTasks(files)
	if err != nil && firstErr == nil {
		firstErr = err
	}
	for _, task := range tasks {
		hits = append(hits, WorkspaceKnowledgeEvidenceHit{
			Kind:       "task",
			ID:         task.ID,
			Title:      task.Title,
			Summary:    task.Summary,
			Excerpt:    firstExcerpt(task.SourceRefs),
			SourceRefs: append([]WorkspaceKnowledgeSourceRef(nil), task.SourceRefs...),
		})
	}

	selected := selectRelevantWorkspaceKnowledgeHits(question, hits)
	if len(selected) > 0 {
		return selected, nil
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return []WorkspaceKnowledgeEvidenceHit{}, nil
}

func retrieveWorkspaceKnowledgeWikiEvidence(files workspaceKnowledgeFiles, question string) ([]WorkspaceKnowledgeEvidenceHit, error) {
	wikiDir, err := files.wikiDir()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(wikiDir); err != nil {
		if os.IsNotExist(err) {
			return []WorkspaceKnowledgeEvidenceHit{}, nil
		}
		return nil, fmt.Errorf("stat workspace knowledge wiki directory %s: %w", wikiDir, err)
	}

	type wikiCandidate struct {
		path string
		id   string
	}

	candidates := []wikiCandidate{}
	var firstErr error
	indexPath, err := files.IndexPath()
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, wikiCandidate{path: indexPath, id: "wiki:index"})
	overviewPath, err := files.OverviewPath()
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, wikiCandidate{path: overviewPath, id: "wiki:overview"})
	openQuestionsPath, err := files.OpenQuestionsPath()
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, wikiCandidate{path: openQuestionsPath, id: "wiki:open-questions"})
	logPath, err := files.LogPath()
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, wikiCandidate{path: logPath, id: "wiki:log"})

	for _, subdir := range []func() (string, error){files.docsDir, files.conceptsDir} {
		dir, err := subdir()
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("read workspace knowledge wiki directory %s: %w", dir, err)
			}
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
				continue
			}
			filename := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			relativeDir := filepath.Base(dir)
			candidates = append(candidates, wikiCandidate{
				path: filepath.Join(dir, entry.Name()),
				id:   "wiki:" + filepath.ToSlash(filepath.Join(relativeDir, filename)),
			})
		}
	}

	hits := make([]WorkspaceKnowledgeEvidenceHit, 0, len(candidates))
	for _, candidate := range candidates {
		content, err := os.ReadFile(candidate.path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("read workspace knowledge wiki page %s: %w", candidate.path, err)
			}
			continue
		}
		trimmed := strings.TrimSpace(string(content))
		if trimmed == "" {
			continue
		}
		title := workspaceKnowledgeReadableTitle(candidate.id)
		hits = append(hits, WorkspaceKnowledgeEvidenceHit{
			Kind:    "wiki_page",
			ID:      candidate.id,
			Title:   title,
			Summary: firstWorkspaceKnowledgeMarkdownLine(trimmed),
			Excerpt: workspaceKnowledgeTrimmedExcerpt(trimmed),
		})
	}
	selected := selectRelevantWorkspaceKnowledgeHits(question, hits)
	if len(selected) > 0 {
		return selected, nil
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return []WorkspaceKnowledgeEvidenceHit{}, nil
}

func retrieveWorkspaceKnowledgeInputEvidence(files workspaceKnowledgeFiles, question string) ([]WorkspaceKnowledgeEvidenceHit, error) {
	extractsDir, err := files.extractsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(extractsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []WorkspaceKnowledgeEvidenceHit{}, nil
		}
		return nil, fmt.Errorf("read workspace knowledge extracts directory %s: %w", extractsDir, err)
	}

	sourceRefsBySlug, err := workspaceKnowledgeSourceRefsBySlug(files)
	if err != nil {
		return nil, err
	}

	candidates := make([]workspaceKnowledgeEvidenceCandidate, 0, len(entries))
	var firstErr error
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		path := filepath.Join(extractsDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("read workspace knowledge extract %s: %w", path, err)
			}
			continue
		}
		trimmed := strings.TrimSpace(string(content))
		if trimmed == "" {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		candidates = append(candidates, workspaceKnowledgeEvidenceCandidate{
			hit: WorkspaceKnowledgeEvidenceHit{
				Kind:       "raw_excerpt",
				ID:         "extract:" + slug,
				Title:      workspaceKnowledgeReadableTitle(slug),
				Summary:    firstWorkspaceKnowledgeMarkdownLine(trimmed),
				Excerpt:    workspaceKnowledgeTrimmedExcerpt(trimmed),
				SourceRefs: sourceRefsBySlug[slug],
			},
			searchText: trimmed,
		})
	}
	selected := selectRelevantWorkspaceKnowledgeEvidenceCandidates(question, candidates)
	if len(selected) > 0 {
		return selected, nil
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return []WorkspaceKnowledgeEvidenceHit{}, nil
}

func workspaceKnowledgeSourceRefsBySlug(files workspaceKnowledgeFiles) (map[string][]WorkspaceKnowledgeSourceRef, error) {
	sources, err := files.ReadSources()
	if err != nil {
		return nil, err
	}
	refs := make(map[string][]WorkspaceKnowledgeSourceRef, len(sources))
	for _, source := range sources {
		if source.Slug == "" || source.ID == "" {
			continue
		}
		refs[source.Slug] = []WorkspaceKnowledgeSourceRef{{
			SourceID: source.ID,
		}}
	}
	return refs, nil
}

func selectRelevantWorkspaceKnowledgeHits(question string, hits []WorkspaceKnowledgeEvidenceHit) []WorkspaceKnowledgeEvidenceHit {
	candidates := make([]workspaceKnowledgeEvidenceCandidate, 0, len(hits))
	for _, hit := range hits {
		candidates = append(candidates, workspaceKnowledgeEvidenceCandidate{hit: hit})
	}
	return selectRelevantWorkspaceKnowledgeEvidenceCandidates(question, candidates)
}

func selectRelevantWorkspaceKnowledgeEvidenceCandidates(question string, candidates []workspaceKnowledgeEvidenceCandidate) []WorkspaceKnowledgeEvidenceHit {
	if len(candidates) == 0 {
		return []WorkspaceKnowledgeEvidenceHit{}
	}

	terms := workspaceKnowledgeQueryTerms(question)
	type scoredHit struct {
		hit   workspaceKnowledgeEvidenceCandidate
		score int
	}

	scored := make([]scoredHit, 0, len(candidates))
	for _, candidate := range candidates {
		score := workspaceKnowledgeHitScore(candidate.hit, terms, candidate.searchText)
		if score == 0 && len(terms) > 0 {
			continue
		}
		scored = append(scored, scoredHit{hit: candidate, score: score})
	}
	if len(scored) == 0 {
		return []WorkspaceKnowledgeEvidenceHit{}
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].hit.hit.Kind != scored[j].hit.hit.Kind {
			return scored[i].hit.hit.Kind < scored[j].hit.hit.Kind
		}
		if scored[i].hit.hit.Title != scored[j].hit.hit.Title {
			return scored[i].hit.hit.Title < scored[j].hit.hit.Title
		}
		return scored[i].hit.hit.ID < scored[j].hit.hit.ID
	})

	limit := len(scored)
	if limit > maxWorkspaceKnowledgeQueryEvidence {
		limit = maxWorkspaceKnowledgeQueryEvidence
	}
	selected := make([]WorkspaceKnowledgeEvidenceHit, 0, limit)
	for _, candidate := range scored[:limit] {
		selected = append(selected, candidate.hit.hit)
	}
	return selected
}

func workspaceKnowledgeHitScore(hit WorkspaceKnowledgeEvidenceHit, terms []string, searchText string) int {
	if len(terms) == 0 {
		return 1
	}

	title := strings.ToLower(strings.TrimSpace(hit.Title))
	summary := strings.ToLower(strings.TrimSpace(hit.Summary))
	excerpt := strings.ToLower(strings.TrimSpace(hit.Excerpt))
	fullText := strings.ToLower(strings.TrimSpace(searchText))
	score := 0
	for _, term := range terms {
		if term == "" {
			continue
		}
		if strings.Contains(title, term) {
			score += 5
		}
		if strings.Contains(summary, term) {
			score += 3
		}
		if strings.Contains(excerpt, term) {
			score++
		}
		if fullText != "" && !strings.Contains(title, term) && !strings.Contains(summary, term) && !strings.Contains(excerpt, term) && strings.Contains(fullText, term) {
			score++
		}
	}
	return score
}

func workspaceKnowledgeQueryTerms(question string) []string {
	normalized := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, question)
	fields := strings.Fields(normalized)
	seen := map[string]struct{}{}
	terms := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) < 3 {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		terms = append(terms, field)
	}
	return terms
}

func workspaceKnowledgeEvidenceIDs(hits []WorkspaceKnowledgeEvidenceHit) []string {
	ids := make([]string, 0, len(hits))
	for _, hit := range hits {
		if id := strings.TrimSpace(hit.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func appendWorkspaceKnowledgeConversationLog(files workspaceKnowledgeFiles, entry workspaceKnowledgeConversationLogEntry) error {
	schemaDir, err := files.schemaDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(schemaDir, 0o700); err != nil {
		return fmt.Errorf("create workspace knowledge schema directory %s: %w", schemaDir, err)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal workspace knowledge conversation log entry: %w", err)
	}

	logPath := filepath.Join(schemaDir, "conversation-log.jsonl")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open workspace knowledge conversation log %s: %w", logPath, err)
	}
	defer file.Close()

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append workspace knowledge conversation log %s: %w", logPath, err)
	}
	return nil
}

func appendWorkspaceKnowledgeConversationLogBestEffort(files workspaceKnowledgeFiles, entry workspaceKnowledgeConversationLogEntry) {
	_ = appendWorkspaceKnowledgeConversationLog(files, entry)
}

func candidateSourceRefs(candidate WorkspaceKnowledgeCandidate) []WorkspaceKnowledgeSourceRef {
	if len(candidate.SourceRefs) > 0 {
		return append([]WorkspaceKnowledgeSourceRef(nil), candidate.SourceRefs...)
	}
	if strings.TrimSpace(candidate.SourceID) == "" {
		return []WorkspaceKnowledgeSourceRef{}
	}
	return []WorkspaceKnowledgeSourceRef{{
		SourceID:  strings.TrimSpace(candidate.SourceID),
		PageStart: candidate.PageStart,
		PageEnd:   candidate.PageEnd,
		Excerpt:   strings.TrimSpace(candidate.Excerpt),
	}}
}

func canonicalWorkspaceKnowledgeClaimID(candidate WorkspaceKnowledgeCandidate) string {
	title := strings.TrimSpace(candidate.Title)
	summary := strings.TrimSpace(candidate.Summary)
	entityIDs := normalizeWorkspaceKnowledgeStringSlice(candidate.EntityIDs)
	sourceRefs := uniqueWorkspaceKnowledgeSourceRefs(candidateSourceRefs(candidate))

	base := workspaceKnowledgeSlug(firstNonEmptyText(title, summary, firstExcerpt(sourceRefs), "claim"))
	if base == "" {
		return ""
	}

	anchorParts := make([]string, 0, len(sourceRefs))
	for _, sourceRef := range sourceRefs {
		anchorParts = append(anchorParts, fmt.Sprintf("%s:%d:%d", strings.TrimSpace(sourceRef.SourceID), sourceRef.PageStart, sourceRef.PageEnd))
	}

	key := strings.Join([]string{
		workspaceKnowledgeStableKeyText(title),
		workspaceKnowledgeStableKeyText(summary),
		strings.Join(entityIDs, "|"),
		strings.Join(anchorParts, "|"),
	}, "\n")
	keyHash := sha256Hex([]byte(strings.TrimSpace(key)))
	if len(keyHash) > 12 {
		keyHash = keyHash[:12]
	}
	if keyHash == "" {
		return "claim:" + base
	}
	return "claim:" + base + "-" + keyHash
}

func findWorkspaceKnowledgeClaimForCandidate(claimsByID map[string]WorkspaceKnowledgeClaim, canonicalID string, candidate WorkspaceKnowledgeCandidate) (WorkspaceKnowledgeClaim, string, error) {
	if claim, ok := claimsByID[canonicalID]; ok {
		return claim, canonicalID, nil
	}

	canonicalMatchIDs := make([]string, 0)
	for existingID, claim := range claimsByID {
		if canonicalWorkspaceKnowledgeClaimID(WorkspaceKnowledgeCandidate{
			Title:      claim.Title,
			Type:       claim.Type,
			Summary:    claim.Summary,
			EntityIDs:  claim.EntityIDs,
			SourceRefs: claim.SourceRefs,
		}) == canonicalID {
			canonicalMatchIDs = append(canonicalMatchIDs, existingID)
		}
	}
	if len(canonicalMatchIDs) > 1 {
		return WorkspaceKnowledgeClaim{}, "", ambiguousWorkspaceKnowledgeClaimMatchError(canonicalMatchIDs)
	}
	if len(canonicalMatchIDs) == 1 {
		existingID := canonicalMatchIDs[0]
		return claimsByID[existingID], existingID, nil
	}

	semanticMatchIDs := make([]string, 0)
	for existingID, claim := range claimsByID {
		if workspaceKnowledgeClaimMatchesCandidate(claim, candidate) {
			semanticMatchIDs = append(semanticMatchIDs, existingID)
		}
	}
	if len(semanticMatchIDs) > 1 {
		return WorkspaceKnowledgeClaim{}, "", ambiguousWorkspaceKnowledgeClaimMatchError(semanticMatchIDs)
	}
	if len(semanticMatchIDs) == 1 {
		existingID := semanticMatchIDs[0]
		return claimsByID[existingID], existingID, nil
	}
	return WorkspaceKnowledgeClaim{}, "", nil
}

func mergeWorkspaceKnowledgePromotedClaim(existing WorkspaceKnowledgeClaim, canonicalID, workspaceID string, candidate WorkspaceKnowledgeCandidate, now string) (WorkspaceKnowledgeClaim, bool) {
	candidateSourceRefs := uniqueWorkspaceKnowledgeSourceRefs(candidateSourceRefs(candidate))
	candidateEntityIDs := normalizeWorkspaceKnowledgeStringSlice(candidate.EntityIDs)

	if strings.TrimSpace(existing.ID) == "" {
		createdAt := firstNonEmptyText(existing.CreatedAt, now)
		updatedAt := firstNonEmptyText(existing.UpdatedAt, now)
		return WorkspaceKnowledgeClaim{
			ID:          strings.TrimSpace(canonicalID),
			WorkspaceID: strings.TrimSpace(workspaceID),
			Title:       firstNonEmptyText(strings.TrimSpace(candidate.Title), strings.TrimSpace(canonicalID)),
			Type:        firstNonEmptyText(strings.TrimSpace(existing.Type), "claim"),
			Summary:     strings.TrimSpace(candidate.Summary),
			EntityIDs:   candidateEntityIDs,
			SourceRefs:  candidateSourceRefs,
			Origin:      firstNonEmptyText(strings.TrimSpace(existing.Origin), "promotion"),
			Status:      firstNonEmptyText(strings.TrimSpace(existing.Status), "promoted"),
			Confidence:  firstNonZeroFloat(existing.Confidence, 1),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}, true
	}

	merged := existing
	merged.ID = strings.TrimSpace(canonicalID)
	if merged.WorkspaceID == "" {
		merged.WorkspaceID = strings.TrimSpace(workspaceID)
	}
	if merged.Title == "" {
		merged.Title = firstNonEmptyText(strings.TrimSpace(candidate.Title), merged.ID)
	}
	if merged.Type == "" {
		merged.Type = "claim"
	}
	if merged.Summary == "" {
		merged.Summary = strings.TrimSpace(candidate.Summary)
	}
	merged.EntityIDs = mergeWorkspaceKnowledgeStringValues(merged.EntityIDs, candidateEntityIDs)
	merged.SourceRefs = mergeWorkspaceKnowledgeSourceRefValues(merged.SourceRefs, candidateSourceRefs)
	if merged.Origin == "" {
		merged.Origin = "promotion"
	}
	if merged.Status == "" {
		merged.Status = "promoted"
	}
	if merged.Confidence == 0 {
		merged.Confidence = 1
	}
	if merged.CreatedAt == "" {
		merged.CreatedAt = firstNonEmptyText(existing.CreatedAt, now)
	}

	timestampChanged := !workspaceKnowledgeClaimsEqualIgnoringIDAndUpdatedAt(existing, merged)
	if timestampChanged {
		merged.UpdatedAt = firstNonEmptyText(now, existing.UpdatedAt, merged.CreatedAt)
	} else {
		merged.UpdatedAt = existing.UpdatedAt
	}
	if merged.UpdatedAt == "" {
		merged.UpdatedAt = firstNonEmptyText(merged.CreatedAt, now)
	}
	changed := !workspaceKnowledgeClaimsEqualIgnoringUpdatedAt(existing, merged)
	return merged, changed
}

func workspaceKnowledgeClaimMatchesCandidate(claim WorkspaceKnowledgeClaim, candidate WorkspaceKnowledgeCandidate) bool {
	candidateTitle := workspaceKnowledgeStableKeyText(candidate.Title)
	candidateSummary := workspaceKnowledgeStableKeyText(candidate.Summary)
	candidateEntityIDs := normalizeWorkspaceKnowledgeStringSlice(candidate.EntityIDs)
	candidateSourceRefs := uniqueWorkspaceKnowledgeSourceRefs(candidateSourceRefs(candidate))
	claimTitle := workspaceKnowledgeStableKeyText(claim.Title)
	claimSummary := workspaceKnowledgeStableKeyText(claim.Summary)
	claimEntityIDs := normalizeWorkspaceKnowledgeStringSlice(claim.EntityIDs)
	claimSourceRefs := uniqueWorkspaceKnowledgeSourceRefs(claim.SourceRefs)

	if candidateTitle == "" && candidateSummary == "" && len(candidateEntityIDs) == 0 && len(candidateSourceRefs) == 0 {
		return false
	}
	if !workspaceKnowledgeStableTextCompatible(claimTitle, candidateTitle) {
		return false
	}
	if !workspaceKnowledgeStableTextCompatible(claimSummary, candidateSummary) {
		return false
	}
	if !workspaceKnowledgeStableStringSlicesCompatible(claimEntityIDs, candidateEntityIDs) {
		return false
	}
	if !workspaceKnowledgeStableSourceRefAnchorsCompatible(claimSourceRefs, candidateSourceRefs) {
		return false
	}
	return true
}

func normalizeWorkspaceKnowledgeCandidates(candidates []WorkspaceKnowledgeCandidate) []WorkspaceKnowledgeCandidate {
	if len(candidates) == 0 {
		return []WorkspaceKnowledgeCandidate{}
	}
	normalized := make([]WorkspaceKnowledgeCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		normalized = append(normalized, WorkspaceKnowledgeCandidate{
			ID:         strings.TrimSpace(candidate.ID),
			Title:      strings.TrimSpace(candidate.Title),
			Type:       strings.ToLower(strings.TrimSpace(candidate.Type)),
			Summary:    strings.TrimSpace(candidate.Summary),
			Aliases:    normalizeWorkspaceKnowledgeStringSlice(candidate.Aliases),
			EntityIDs:  normalizeWorkspaceKnowledgeStringSlice(candidate.EntityIDs),
			Priority:   strings.TrimSpace(candidate.Priority),
			SourceID:   strings.TrimSpace(candidate.SourceID),
			PageStart:  candidate.PageStart,
			PageEnd:    candidate.PageEnd,
			Excerpt:    strings.TrimSpace(candidate.Excerpt),
			SourceRefs: uniqueWorkspaceKnowledgeSourceRefs(candidateSourceRefs(candidate)),
		})
	}
	return normalized
}

func mergeWorkspaceKnowledgeStringValues(existing, candidate []string) []string {
	merged := append([]string(nil), existing...)
	merged = append(merged, candidate...)
	return normalizeWorkspaceKnowledgeStringSlice(merged)
}

func mergeWorkspaceKnowledgeSourceRefValues(existing, candidate []WorkspaceKnowledgeSourceRef) []WorkspaceKnowledgeSourceRef {
	merged := append([]WorkspaceKnowledgeSourceRef(nil), existing...)
	merged = append(merged, candidate...)
	return uniqueWorkspaceKnowledgeSourceRefs(merged)
}

func workspaceKnowledgeClaimsEqualIgnoringUpdatedAt(left, right WorkspaceKnowledgeClaim) bool {
	return left.ID == right.ID &&
		left.WorkspaceID == right.WorkspaceID &&
		left.Title == right.Title &&
		left.Type == right.Type &&
		left.Summary == right.Summary &&
		left.Origin == right.Origin &&
		left.Status == right.Status &&
		left.Confidence == right.Confidence &&
		left.CreatedAt == right.CreatedAt &&
		equalWorkspaceKnowledgeStringSlices(left.EntityIDs, right.EntityIDs) &&
		equalWorkspaceKnowledgeSourceRefs(left.SourceRefs, right.SourceRefs)
}

func workspaceKnowledgeClaimsEqualIgnoringIDAndUpdatedAt(left, right WorkspaceKnowledgeClaim) bool {
	return left.WorkspaceID == right.WorkspaceID &&
		left.Title == right.Title &&
		left.Type == right.Type &&
		left.Summary == right.Summary &&
		left.Origin == right.Origin &&
		left.Status == right.Status &&
		left.Confidence == right.Confidence &&
		left.CreatedAt == right.CreatedAt &&
		equalWorkspaceKnowledgeStringSlices(left.EntityIDs, right.EntityIDs) &&
		equalWorkspaceKnowledgeSourceRefs(left.SourceRefs, right.SourceRefs)
}

func equalWorkspaceKnowledgeStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalWorkspaceKnowledgeSourceRefs(left, right []WorkspaceKnowledgeSourceRef) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalWorkspaceKnowledgeSourceRefAnchors(left, right []WorkspaceKnowledgeSourceRef) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].SourceID != right[index].SourceID || left[index].PageStart != right[index].PageStart || left[index].PageEnd != right[index].PageEnd {
			return false
		}
	}
	return true
}

func workspaceKnowledgeStableTextCompatible(left, right string) bool {
	if left == "" || right == "" {
		return true
	}
	return left == right
}

func workspaceKnowledgeStableStringSlicesCompatible(left, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return true
	}
	return equalWorkspaceKnowledgeStringSlices(left, right)
}

func workspaceKnowledgeStableSourceRefAnchorsCompatible(left, right []WorkspaceKnowledgeSourceRef) bool {
	if len(left) == 0 || len(right) == 0 {
		return true
	}
	return equalWorkspaceKnowledgeSourceRefAnchors(left, right)
}

func ambiguousWorkspaceKnowledgeClaimMatchError(ids []string) error {
	sortedIDs := append([]string(nil), ids...)
	sort.Strings(sortedIDs)
	return fmt.Errorf("ambiguous workspace knowledge claim matches: %s", strings.Join(sortedIDs, ", "))
}

func firstNonZeroFloat(values ...float64) float64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func uniqueWorkspaceKnowledgeSourceRefs(sourceRefs []WorkspaceKnowledgeSourceRef) []WorkspaceKnowledgeSourceRef {
	normalized := normalizeWorkspaceKnowledgeSourceRefs(sourceRefs)
	if len(normalized) == 0 {
		return []WorkspaceKnowledgeSourceRef{}
	}
	unique := make([]WorkspaceKnowledgeSourceRef, 0, len(normalized))
	for _, sourceRef := range normalized {
		if len(unique) > 0 && unique[len(unique)-1] == sourceRef {
			continue
		}
		unique = append(unique, sourceRef)
	}
	return unique
}

func workspaceKnowledgeStableKeyText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func normalizeWorkspaceKnowledgeStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func firstWorkspaceKnowledgeMarkdownLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(strings.TrimLeft(line, "#*- "))
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func workspaceKnowledgeTrimmedExcerpt(content string) string {
	trimmed := strings.TrimSpace(content)
	const maxChars = 600
	if len(trimmed) <= maxChars {
		return trimmed
	}
	return strings.TrimSpace(trimmed[:maxChars]) + "..."
}

func workspaceKnowledgeReadableTitle(value string) string {
	replaced := strings.NewReplacer("wiki:", "", "extract:", "", "/", " ", "-", " ", "_", " ").Replace(value)
	parts := strings.Fields(strings.TrimSpace(replaced))
	for index, part := range parts {
		runes := []rune(strings.ToLower(part))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[index] = string(runes)
	}
	return strings.Join(parts, " ")
}
