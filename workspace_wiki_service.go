package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type workspaceKnowledgeExtractor interface {
	ExtractMarkdown(ctx context.Context, rawPath string) (PDFMarkdownPayload, error)
}

type workspaceKnowledgeLLM interface {
	GenerateWorkspaceKnowledgeBySource(ctx context.Context, providerID, modelID int64, prompt string) (WorkspaceKnowledgeBySourcePayload, error)
	GenerateWorkspaceKnowledgeMarkdown(ctx context.Context, providerID, modelID int64, prompt string) (string, error)
}

type workspaceWikiService struct {
	paths        appPaths
	store        *configStore
	pdf          workspaceKnowledgeExtractor
	knowledgeLLM workspaceKnowledgeLLM
}

func newWorkspaceWikiService(paths appPaths, store *configStore, pdf workspaceKnowledgeExtractor, knowledgeLLM workspaceKnowledgeLLM) *workspaceWikiService {
	return &workspaceWikiService{
		paths:        paths,
		store:        store,
		pdf:          pdf,
		knowledgeLLM: knowledgeLLM,
	}
}

func (s *workspaceWikiService) StartScan(ctx context.Context, input WorkspaceWikiScanStartInput) (WorkspaceWikiScanJob, error) {
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID == "" {
		return WorkspaceWikiScanJob{}, fmt.Errorf("workspace id is required")
	}
	if s.store == nil {
		return WorkspaceWikiScanJob{}, fmt.Errorf("config store is unavailable")
	}
	if s.pdf == nil {
		return WorkspaceWikiScanJob{}, fmt.Errorf("workspace knowledge extractor is unavailable")
	}
	if s.knowledgeLLM == nil {
		return WorkspaceWikiScanJob{}, fmt.Errorf("workspace knowledge llm is unavailable")
	}
	job, err := s.store.SaveWorkspaceWikiScanJob(ctx, WorkspaceWikiScanJob{
		WorkspaceID:  workspaceID,
		DocumentID:   strings.TrimSpace(input.DocumentID),
		Status:       WorkspaceWikiScanJobQueued,
		CurrentStage: "queued",
		Message:      "queued",
		ProviderID:   input.ProviderID,
		ModelID:      input.ModelID,
		StartedAt:    nowRFC3339(),
		UpdatedAt:    nowRFC3339(),
	})
	if err != nil {
		return WorkspaceWikiScanJob{}, err
	}

	runner := &workspaceWikiScanRunner{
		paths:        s.paths,
		store:        s.store,
		pdf:          s.pdf,
		knowledgeLLM: s.knowledgeLLM,
	}
	go runner.run(context.Background(), job)
	return job, nil
}

type workspaceWikiScanRunner struct {
	paths        appPaths
	store        *configStore
	pdf          workspaceKnowledgeExtractor
	knowledgeLLM workspaceKnowledgeLLM
}

type workspaceKnowledgeScanCandidate struct {
	absolutePath string
	title        string
	documentID   string
	kind         string
	contentHash  string
}

const (
	defaultWorkspaceKnowledgePromptSourceChars = 12000
	maxWorkspaceKnowledgePromptSourceChars     = 48000
	workspaceKnowledgePromptSafetyChars        = 128
)

func (r *workspaceWikiScanRunner) run(ctx context.Context, job WorkspaceWikiScanJob) {
	if ctx == nil {
		ctx = context.Background()
	}
	job = r.saveJob(ctx, job, WorkspaceWikiScanJobRunning, "scan", "scanning workspace", false)
	if err := r.runScan(ctx, job); err != nil {
		_ = r.writeScanRunFailure(ctx, job, err)
		_ = r.saveJob(ctx, job, WorkspaceWikiScanJobFailed, "failed", err.Error(), true)
		return
	}
	r.saveJob(ctx, job, WorkspaceWikiScanJobCompleted, "completed", "scan completed", true)
}

func (r *workspaceWikiScanRunner) runScan(ctx context.Context, job WorkspaceWikiScanJob) error {
	if r.store == nil {
		return fmt.Errorf("config store is unavailable")
	}
	if r.pdf == nil {
		return fmt.Errorf("workspace knowledge extractor is unavailable")
	}
	if r.knowledgeLLM == nil {
		return fmt.Errorf("workspace knowledge llm is unavailable")
	}

	workspace, err := r.store.GetWorkspace(ctx, job.WorkspaceID)
	if err != nil {
		return err
	}

	files := newWorkspaceKnowledgeFiles(r.paths, workspace.ID)
	if err := files.EnsureLayout(); err != nil {
		return err
	}

	sources, err := r.collectSources(ctx, workspace, strings.TrimSpace(job.DocumentID), files)
	if err != nil {
		return err
	}
	if strings.TrimSpace(job.DocumentID) != "" && len(sources) == 0 {
		return fmt.Errorf("no scan sources found for document %s", strings.TrimSpace(job.DocumentID))
	}

	manifest, err := r.loadSourceManifest(files)
	if err != nil {
		return err
	}
	existingByID := workspaceKnowledgeSourcesByID(manifest)

	fullScan := strings.TrimSpace(job.DocumentID) == ""
	if fullScan {
		if err := r.removeStaleSourceArtifacts(files, sources); err != nil {
			return err
		}
	}

	failureCount := 0
	skippedCount := 0
	sourceIDs := make([]string, 0, len(sources))
	for index := range sources {
		sourceIDs = append(sourceIDs, sources[index].ID)
		existingSource, hasExistingSource := existingByID[sources[index].ID]
		if hasExistingSource {
			skip, err := r.shouldSkipSource(files, sources[index], existingSource)
			if err != nil {
				return err
			}
			if skip {
				skippedCount++
				sources[index] = mergeSkippedWorkspaceKnowledgeSource(existingSource, sources[index])
				continue
			}
		}
		if err := r.processSource(ctx, workspace, job, files, &sources[index]); err != nil {
			failureCount++
			sources[index].Status = "error"
			sources[index].LastError = err.Error()
			sources[index].LastScanAt = nowRFC3339()
		}
	}

	manifest = mergeWorkspaceKnowledgeSources(manifest, sources, fullScan)
	if err := files.WriteSources(manifest); err != nil {
		return err
	}

	if _, err := CompileWorkspaceKnowledge(files, workspace.Name); err != nil {
		return err
	}

	return files.WriteScanRun(WorkspaceKnowledgeScanRunRecord{
		ID:          fmt.Sprintf("%d", job.ID),
		WorkspaceID: workspace.ID,
		Status:      string(WorkspaceWikiScanJobCompleted),
		StartedAt:   firstNonEmptyText(strings.TrimSpace(job.StartedAt), nowRFC3339()),
		FinishedAt:  nowRFC3339(),
		SourceIDs:   sourceIDs,
		Message:     workspaceKnowledgeScanMessage(len(sources), failureCount, skippedCount),
	})
}

func (r *workspaceWikiScanRunner) collectSources(ctx context.Context, workspace Workspace, documentID string, files workspaceKnowledgeFiles) ([]WorkspaceKnowledgeSource, error) {
	workspaceRoot, err := files.workspaceRootDir()
	if err != nil {
		return nil, err
	}

	documents, err := r.store.ListDocumentsByWorkspace(ctx, workspace.ID)
	if err != nil {
		return nil, err
	}

	documentIDByPath := map[string]string{}
	documentTitleByPath := map[string]string{}
	for _, document := range documents {
		normalizedPath := normalizeWorkspaceKnowledgeAbsolutePath(document.PrimaryPDFPath)
		if normalizedPath == "" {
			continue
		}
		documentIDByPath[normalizedPath] = document.ID
		documentTitleByPath[normalizedPath] = strings.TrimSpace(document.Title)
	}

	candidates := make([]workspaceKnowledgeScanCandidate, 0)
	err = filepath.WalkDir(workspaceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch normalizeWorkspaceKnowledgeAbsolutePath(path) {
			case normalizeWorkspaceKnowledgeAbsolutePath(filepath.Join(workspaceRoot, "raw")),
				normalizeWorkspaceKnowledgeAbsolutePath(filepath.Join(workspaceRoot, "schema")),
				normalizeWorkspaceKnowledgeAbsolutePath(filepath.Join(workspaceRoot, "wiki")):
				return fs.SkipDir
			}
			return nil
		}

		kind := workspaceKnowledgeSourceKind(path)
		if kind == "" {
			return nil
		}

		normalizedPath := normalizeWorkspaceKnowledgeAbsolutePath(path)
		mappedDocumentID := documentIDByPath[normalizedPath]
		if documentID != "" && mappedDocumentID != documentID {
			return nil
		}

		contentHash, err := sha256File(path)
		if err != nil {
			return fmt.Errorf("hash workspace source %s: %w", path, err)
		}
		title := strings.TrimSpace(documentTitleByPath[normalizedPath])
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		candidates = append(candidates, workspaceKnowledgeScanCandidate{
			absolutePath: path,
			title:        title,
			documentID:   mappedDocumentID,
			kind:         kind,
			contentHash:  contentHash,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk workspace sources: %w", err)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return normalizeWorkspaceKnowledgeAbsolutePath(candidates[i].absolutePath) < normalizeWorkspaceKnowledgeAbsolutePath(candidates[j].absolutePath)
	})

	sources := make([]WorkspaceKnowledgeSource, 0, len(candidates))
	for _, candidate := range candidates {
		baseSlug := workspaceKnowledgeSlug(strings.TrimSuffix(filepath.Base(candidate.absolutePath), filepath.Ext(candidate.absolutePath)))
		sourceKey, err := workspaceKnowledgeStableSourceKey(workspaceRoot, candidate)
		if err != nil {
			return nil, err
		}
		slug := workspaceKnowledgeStableSourceSlug(baseSlug, sourceKey)
		extractPath, err := files.ExtractPath(slug)
		if err != nil {
			return nil, err
		}
		sources = append(sources, WorkspaceKnowledgeSource{
			ID:           "source:" + slug,
			WorkspaceID:  workspace.ID,
			Title:        candidate.title,
			Slug:         slug,
			Kind:         candidate.kind,
			AbsolutePath: candidate.absolutePath,
			ContentHash:  candidate.contentHash,
			ExtractPath:  extractPath,
			DocumentID:   candidate.documentID,
			Status:       "pending",
		})
	}
	return sources, nil
}

func (r *workspaceWikiScanRunner) processSource(ctx context.Context, workspace Workspace, job WorkspaceWikiScanJob, files workspaceKnowledgeFiles, source *WorkspaceKnowledgeSource) error {
	markdown, err := r.extractSourceMarkdown(ctx, *source)
	if err != nil {
		return err
	}
	if err := writeWorkspaceKnowledgeMarkdown(source.ExtractPath, markdown); err != nil {
		return err
	}

	prompt := buildWorkspaceKnowledgeBySourcePromptWithinBudget(workspace, *source, markdown, r.workspaceKnowledgePromptCharBudget(ctx, job.ModelID))
	payload, err := r.knowledgeLLM.GenerateWorkspaceKnowledgeBySource(ctx, job.ProviderID, job.ModelID, prompt)
	if err != nil {
		return fmt.Errorf("generate by-source knowledge for %s: %w", source.Title, err)
	}

	payload = normalizeWorkspaceKnowledgeBySourcePayload(*source, payload)
	if err := files.WriteBySource(source.Slug, payload); err != nil {
		return err
	}

	source.Status = "ready"
	source.LastError = ""
	source.LastScanAt = nowRFC3339()
	return nil
}

func (r *workspaceWikiScanRunner) extractSourceMarkdown(ctx context.Context, source WorkspaceKnowledgeSource) (string, error) {
	switch strings.ToLower(strings.TrimSpace(source.Kind)) {
	case "pdf":
		payload, err := r.pdf.ExtractMarkdown(ctx, source.AbsolutePath)
		if err != nil {
			return "", fmt.Errorf("extract markdown from %s: %w", source.AbsolutePath, err)
		}
		markdown := strings.TrimSpace(payload.Markdown)
		if markdown == "" {
			return "", fmt.Errorf("extract markdown from %s: empty markdown", source.AbsolutePath)
		}
		return markdown, nil
	case "markdown":
		content, err := os.ReadFile(source.AbsolutePath)
		if err != nil {
			return "", fmt.Errorf("read markdown source %s: %w", source.AbsolutePath, err)
		}
		markdown := strings.TrimSpace(string(content))
		if markdown == "" {
			return "", fmt.Errorf("markdown source %s is empty", source.AbsolutePath)
		}
		return markdown, nil
	default:
		return "", fmt.Errorf("unsupported workspace knowledge source kind %s", source.Kind)
	}
}

func (r *workspaceWikiScanRunner) loadSourceManifest(files workspaceKnowledgeFiles) ([]WorkspaceKnowledgeSource, error) {
	return files.ReadSources()
}

func (r *workspaceWikiScanRunner) removeStaleSourceArtifacts(files workspaceKnowledgeFiles, currentSources []WorkspaceKnowledgeSource) error {
	currentSlugs := make(map[string]struct{}, len(currentSources))
	for _, source := range currentSources {
		currentSlugs[source.Slug] = struct{}{}
	}

	extractsDir, err := files.extractsDir()
	if err != nil {
		return err
	}
	if err := removeWorkspaceKnowledgeArtifactsNotInSet(extractsDir, ".md", currentSlugs); err != nil {
		return err
	}

	bySourceDir, err := files.bySourceDir()
	if err != nil {
		return err
	}
	return removeWorkspaceKnowledgeArtifactsNotInSet(bySourceDir, ".json", currentSlugs)
}

func (r *workspaceWikiScanRunner) saveJob(ctx context.Context, job WorkspaceWikiScanJob, status WorkspaceWikiScanJobStatus, stage, message string, finished bool) WorkspaceWikiScanJob {
	if r.store == nil {
		return job
	}
	job.Status = status
	job.CurrentStage = strings.TrimSpace(stage)
	job.Message = strings.TrimSpace(message)
	job.UpdatedAt = nowRFC3339()
	if strings.TrimSpace(job.StartedAt) == "" {
		job.StartedAt = job.UpdatedAt
	}
	if finished {
		job.FinishedAt = job.UpdatedAt
	}
	saved, err := r.store.SaveWorkspaceWikiScanJob(ctx, job)
	if err != nil {
		return job
	}
	return saved
}

func (r *workspaceWikiScanRunner) shouldSkipSource(files workspaceKnowledgeFiles, currentSource, existingSource WorkspaceKnowledgeSource) (bool, error) {
	if existingSource.ID == "" {
		return false, nil
	}
	if existingSource.ContentHash != currentSource.ContentHash {
		return false, nil
	}
	if existingSource.Status != "ready" {
		return false, nil
	}
	if !workspaceKnowledgeFileExists(firstNonEmptyText(currentSource.ExtractPath, existingSource.ExtractPath)) {
		return false, nil
	}

	bySourcePath, err := files.BySourcePath(currentSource.Slug)
	if err != nil {
		return false, err
	}
	if !workspaceKnowledgeFileExists(bySourcePath) {
		return false, nil
	}
	return true, nil
}

func (r *workspaceWikiScanRunner) workspaceKnowledgePromptCharBudget(ctx context.Context, modelID int64) int {
	budget := defaultWorkspaceKnowledgePromptSourceChars
	if r.store == nil || modelID <= 0 {
		return budget
	}

	model, err := r.store.GetModel(ctx, modelID)
	if err != nil || model.ContextWindow <= 0 {
		return budget
	}

	budget = model.ContextWindow*5 - workspaceKnowledgePromptSafetyChars
	if budget < 0 {
		budget = 0
	}
	if budget > maxWorkspaceKnowledgePromptSourceChars {
		return maxWorkspaceKnowledgePromptSourceChars
	}
	return budget
}

func (r *workspaceWikiScanRunner) writeScanRunFailure(ctx context.Context, job WorkspaceWikiScanJob, runErr error) error {
	workspaceID := strings.TrimSpace(job.WorkspaceID)
	if workspaceID == "" {
		return nil
	}
	files := newWorkspaceKnowledgeFiles(r.paths, workspaceID)
	if err := files.EnsureLayout(); err != nil {
		return err
	}
	return files.WriteScanRun(WorkspaceKnowledgeScanRunRecord{
		ID:          fmt.Sprintf("%d", job.ID),
		WorkspaceID: workspaceID,
		Status:      string(WorkspaceWikiScanJobFailed),
		StartedAt:   firstNonEmptyText(strings.TrimSpace(job.StartedAt), nowRFC3339()),
		FinishedAt:  nowRFC3339(),
		Message:     strings.TrimSpace(runErr.Error()),
	})
}

func mergeWorkspaceKnowledgeSources(existing, updated []WorkspaceKnowledgeSource, fullScan bool) []WorkspaceKnowledgeSource {
	if fullScan {
		sort.Slice(updated, func(i, j int) bool {
			return lessSource(updated[i], updated[j])
		})
		return updated
	}

	mergedByID := map[string]WorkspaceKnowledgeSource{}
	for _, source := range existing {
		mergedByID[source.ID] = source
	}
	for _, source := range updated {
		mergedByID[source.ID] = source
	}

	merged := make([]WorkspaceKnowledgeSource, 0, len(mergedByID))
	for _, source := range mergedByID {
		merged = append(merged, source)
	}
	sort.Slice(merged, func(i, j int) bool {
		return lessSource(merged[i], merged[j])
	})
	return merged
}

func workspaceKnowledgeSourcesByID(sources []WorkspaceKnowledgeSource) map[string]WorkspaceKnowledgeSource {
	byID := make(map[string]WorkspaceKnowledgeSource, len(sources))
	for _, source := range sources {
		byID[source.ID] = source
	}
	return byID
}

func mergeSkippedWorkspaceKnowledgeSource(existingSource, currentSource WorkspaceKnowledgeSource) WorkspaceKnowledgeSource {
	existingSource.WorkspaceID = currentSource.WorkspaceID
	existingSource.Title = currentSource.Title
	existingSource.Slug = currentSource.Slug
	existingSource.Kind = currentSource.Kind
	existingSource.AbsolutePath = currentSource.AbsolutePath
	existingSource.ContentHash = currentSource.ContentHash
	existingSource.ExtractPath = currentSource.ExtractPath
	existingSource.DocumentID = currentSource.DocumentID
	return existingSource
}

func normalizeWorkspaceKnowledgeBySourcePayload(source WorkspaceKnowledgeSource, payload WorkspaceKnowledgeBySourcePayload) WorkspaceKnowledgeBySourcePayload {
	now := nowRFC3339()
	payload.Source = source
	if payload.Entities == nil {
		payload.Entities = []WorkspaceKnowledgeEntity{}
	}
	if payload.Claims == nil {
		payload.Claims = []WorkspaceKnowledgeClaim{}
	}
	if payload.Relations == nil {
		payload.Relations = []WorkspaceKnowledgeRelation{}
	}
	if payload.Tasks == nil {
		payload.Tasks = []WorkspaceKnowledgeTask{}
	}

	for index := range payload.Entities {
		if strings.TrimSpace(payload.Entities[index].WorkspaceID) == "" {
			payload.Entities[index].WorkspaceID = source.WorkspaceID
		}
		if strings.TrimSpace(payload.Entities[index].Origin) == "" {
			payload.Entities[index].Origin = "scan"
		}
		if strings.TrimSpace(payload.Entities[index].CreatedAt) == "" {
			payload.Entities[index].CreatedAt = now
		}
		if strings.TrimSpace(payload.Entities[index].UpdatedAt) == "" {
			payload.Entities[index].UpdatedAt = now
		}
	}
	for index := range payload.Claims {
		if strings.TrimSpace(payload.Claims[index].WorkspaceID) == "" {
			payload.Claims[index].WorkspaceID = source.WorkspaceID
		}
		if strings.TrimSpace(payload.Claims[index].Origin) == "" {
			payload.Claims[index].Origin = "scan"
		}
		if strings.TrimSpace(payload.Claims[index].CreatedAt) == "" {
			payload.Claims[index].CreatedAt = now
		}
		if strings.TrimSpace(payload.Claims[index].UpdatedAt) == "" {
			payload.Claims[index].UpdatedAt = now
		}
	}
	for index := range payload.Relations {
		if strings.TrimSpace(payload.Relations[index].WorkspaceID) == "" {
			payload.Relations[index].WorkspaceID = source.WorkspaceID
		}
		if strings.TrimSpace(payload.Relations[index].Origin) == "" {
			payload.Relations[index].Origin = "scan"
		}
		if strings.TrimSpace(payload.Relations[index].CreatedAt) == "" {
			payload.Relations[index].CreatedAt = now
		}
		if strings.TrimSpace(payload.Relations[index].UpdatedAt) == "" {
			payload.Relations[index].UpdatedAt = now
		}
	}
	for index := range payload.Tasks {
		if strings.TrimSpace(payload.Tasks[index].WorkspaceID) == "" {
			payload.Tasks[index].WorkspaceID = source.WorkspaceID
		}
		if strings.TrimSpace(payload.Tasks[index].Origin) == "" {
			payload.Tasks[index].Origin = "scan"
		}
		if strings.TrimSpace(payload.Tasks[index].CreatedAt) == "" {
			payload.Tasks[index].CreatedAt = now
		}
		if strings.TrimSpace(payload.Tasks[index].UpdatedAt) == "" {
			payload.Tasks[index].UpdatedAt = now
		}
	}
	return payload
}

func workspaceKnowledgeSourceKind(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		return "pdf"
	case ".md", ".markdown":
		return "markdown"
	default:
		return ""
	}
}

func workspaceKnowledgeStableSourceKey(workspaceRoot string, candidate workspaceKnowledgeScanCandidate) (string, error) {
	if trimmedDocumentID := strings.TrimSpace(candidate.documentID); trimmedDocumentID != "" {
		return "document:" + trimmedDocumentID, nil
	}
	relativePath, err := filepath.Rel(workspaceRoot, candidate.absolutePath)
	if err != nil {
		return "", fmt.Errorf("resolve workspace knowledge relative path for %s: %w", candidate.absolutePath, err)
	}
	return "path:" + normalizeWorkspaceKnowledgeRelativePath(relativePath), nil
}

func workspaceKnowledgeStableSourceSlug(baseSlug, sourceKey string) string {
	normalizedBaseSlug := strings.TrimSpace(baseSlug)
	if normalizedBaseSlug == "" {
		normalizedBaseSlug = "item"
	}
	keyHash := sha256Hex([]byte(strings.TrimSpace(sourceKey)))
	if len(keyHash) > 12 {
		keyHash = keyHash[:12]
	}
	if keyHash == "" {
		return normalizedBaseSlug
	}
	return normalizedBaseSlug + "-" + keyHash
}

func normalizeWorkspaceKnowledgeAbsolutePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(filepath.Clean(trimmed))
}

func normalizeWorkspaceKnowledgeRelativePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(filepath.ToSlash(filepath.Clean(trimmed)))
}

func buildWorkspaceKnowledgeBySourcePromptWithinBudget(workspace Workspace, source WorkspaceKnowledgeSource, markdown string, totalPromptBudget int) string {
	if totalPromptBudget <= 0 {
		return buildWorkspaceKnowledgeBySourcePrompt(workspace, source, "")
	}

	emptyPrompt := buildWorkspaceKnowledgeBySourcePrompt(workspace, source, "")
	if len(emptyPrompt) >= totalPromptBudget {
		return emptyPrompt
	}

	maxSourceChars := totalPromptBudget - len(emptyPrompt)
	bestPrompt := emptyPrompt
	low := 0
	high := maxSourceChars
	for low <= high {
		mid := (low + high) / 2
		candidateMarkdown := trimWorkspaceKnowledgePromptMarkdown(markdown, mid)
		candidatePrompt := buildWorkspaceKnowledgeBySourcePrompt(workspace, source, candidateMarkdown)
		if len(candidatePrompt) <= totalPromptBudget {
			bestPrompt = candidatePrompt
			low = mid + 1
			continue
		}
		high = mid - 1
	}
	return bestPrompt
}

func trimWorkspaceKnowledgePromptMarkdown(markdown string, maxChars int) string {
	trimmedMarkdown := strings.TrimSpace(markdown)
	if trimmedMarkdown == "" {
		return ""
	}
	if maxChars <= 0 {
		maxChars = defaultWorkspaceKnowledgePromptSourceChars
	}

	runes := []rune(trimmedMarkdown)
	if len(runes) <= maxChars {
		return trimmedMarkdown
	}

	headCount := (maxChars * 3) / 4
	if headCount <= 0 {
		headCount = maxChars / 2
	}
	if headCount <= 0 {
		headCount = 1
	}
	tailCount := maxChars - headCount
	if tailCount <= 0 {
		tailCount = 1
		if headCount > 1 {
			headCount = maxChars - tailCount
		}
	}

	omittedCount := len(runes) - (headCount + tailCount)
	if omittedCount <= 0 {
		return string(runes[:maxChars])
	}

	return string(runes[:headCount]) +
		fmt.Sprintf("\n\n[... trimmed %d characters for scan prompt ...]\n\n", omittedCount) +
		string(runes[len(runes)-tailCount:])
}

func workspaceKnowledgeScanMessage(totalSources, failedSources, skippedSources int) string {
	message := fmt.Sprintf("processed %d sources", totalSources)
	details := make([]string, 0, 2)
	if failedSources > 0 {
		details = append(details, fmt.Sprintf("%d failed", failedSources))
	}
	if skippedSources > 0 {
		details = append(details, fmt.Sprintf("%d skipped", skippedSources))
	}
	if len(details) == 0 {
		return message
	}
	return fmt.Sprintf("%s (%s)", message, strings.Join(details, ", "))
}

func removeWorkspaceKnowledgeArtifactsNotInSet(dir, extension string, keep map[string]struct{}) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read workspace knowledge directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), extension) {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if _, ok := keep[slug]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return fmt.Errorf("remove workspace knowledge artifact %s: %w", filepath.Join(dir, entry.Name()), err)
		}
	}
	return nil
}

func removeWorkspaceKnowledgeFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove workspace knowledge file %s: %w", path, err)
	}
	return nil
}

func workspaceKnowledgeFileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
