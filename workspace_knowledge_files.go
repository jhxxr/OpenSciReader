package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type workspaceKnowledgeFiles struct {
	paths       appPaths
	workspaceID string
}

func newWorkspaceKnowledgeFiles(paths appPaths, workspaceID string) workspaceKnowledgeFiles {
	return workspaceKnowledgeFiles{
		paths:       paths,
		workspaceID: strings.TrimSpace(workspaceID),
	}
}

func (f workspaceKnowledgeFiles) EnsureLayout() error {
	layoutDirs, err := f.layoutDirs()
	if err != nil {
		return err
	}

	for _, directory := range layoutDirs {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return fmt.Errorf("create workspace knowledge directory %s: %w", directory, err)
		}
	}
	return nil
}

func (f workspaceKnowledgeFiles) SourcesPath() (string, error) {
	return f.SourcesManifestPath()
}

func (f workspaceKnowledgeFiles) SourcesManifestPath() (string, error) {
	stateDir, err := f.stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "sources.json"), nil
}

func (f workspaceKnowledgeFiles) CompileSummaryPath() (string, error) {
	stateDir, err := f.stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "compile-summary.json"), nil
}

func (f workspaceKnowledgeFiles) ExtractPath(sourceSlug string) (string, error) {
	return f.MarkItDownPath(sourceSlug)
}

func (f workspaceKnowledgeFiles) MarkItDownPath(sourceSlug string) (string, error) {
	markItDownDir, err := f.markItDownDir()
	if err != nil {
		return "", err
	}
	validatedSourceSlug, err := validateWorkspaceKnowledgePathSegment("source slug", sourceSlug)
	if err != nil {
		return "", err
	}
	return filepath.Join(markItDownDir, validatedSourceSlug+".md"), nil
}

func (f workspaceKnowledgeFiles) BySourcePath(sourceSlug string) (string, error) {
	bySourceDir, err := f.bySourceDir()
	if err != nil {
		return "", err
	}
	validatedSourceSlug, err := validateWorkspaceKnowledgePathSegment("source slug", sourceSlug)
	if err != nil {
		return "", err
	}
	return filepath.Join(bySourceDir, validatedSourceSlug+".json"), nil
}

func (f workspaceKnowledgeFiles) BySourcePaths() ([]string, error) {
	bySourceDir, err := f.bySourceDir()
	if err != nil {
		return nil, err
	}
	legacyBySourceDir, err := f.legacyBySourceDir()
	if err != nil {
		return nil, err
	}

	paths, err := workspaceKnowledgeJSONPaths(bySourceDir)
	if err != nil {
		return nil, fmt.Errorf("read workspace knowledge by-source directory %s: %w", bySourceDir, err)
	}
	legacyPaths, err := workspaceKnowledgeJSONPaths(legacyBySourceDir)
	if err != nil {
		return nil, fmt.Errorf("read workspace knowledge legacy by-source directory %s: %w", legacyBySourceDir, err)
	}

	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		seen[filepath.Base(path)] = struct{}{}
	}
	for _, path := range legacyPaths {
		if _, ok := seen[filepath.Base(path)]; ok {
			continue
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func (f workspaceKnowledgeFiles) ScanRunPath(scanRunID string) (string, error) {
	return f.JobPath(scanRunID)
}

func (f workspaceKnowledgeFiles) JobPath(jobID string) (string, error) {
	jobsDir, err := f.jobsDir()
	if err != nil {
		return "", err
	}
	validatedJobID, err := validateWorkspaceKnowledgePathSegment("job id", jobID)
	if err != nil {
		return "", err
	}
	return filepath.Join(jobsDir, validatedJobID+".json"), nil
}

func (f workspaceKnowledgeFiles) WriteSources(sources []WorkspaceKnowledgeSource) error {
	sourcesPath, err := f.SourcesPath()
	if err != nil {
		return err
	}
	return writeWorkspaceKnowledgeJSON(sourcesPath, sources)
}

func (f workspaceKnowledgeFiles) ReadSources() ([]WorkspaceKnowledgeSource, error) {
	sourcesPath, err := f.SourcesPath()
	if err != nil {
		return nil, err
	}
	legacySourcesPath, err := f.legacySourcesPath()
	if err != nil {
		return nil, err
	}

	readPath, err := workspaceKnowledgeFirstExistingPath(sourcesPath, legacySourcesPath)
	if err != nil {
		return nil, fmt.Errorf("stat workspace knowledge sources: %w", err)
	}
	if readPath == "" {
		return []WorkspaceKnowledgeSource{}, nil
	}

	var sources []WorkspaceKnowledgeSource
	if err := readWorkspaceKnowledgeJSON(readPath, &sources); err != nil {
		return nil, err
	}
	if sources == nil {
		return []WorkspaceKnowledgeSource{}, nil
	}
	return sources, nil
}

func (f workspaceKnowledgeFiles) WriteCompileSummary(summary WorkspaceKnowledgeCompileSummary) error {
	compileSummaryPath, err := f.CompileSummaryPath()
	if err != nil {
		return err
	}
	return writeWorkspaceKnowledgeJSON(compileSummaryPath, summary)
}

func (f workspaceKnowledgeFiles) DeleteCompileSummary() error {
	compileSummaryPath, err := f.CompileSummaryPath()
	if err != nil {
		return err
	}
	if err := os.Remove(compileSummaryPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove workspace knowledge compile summary: %w", err)
	}
	return nil
}

func (f workspaceKnowledgeFiles) ReadCompileSummary() (WorkspaceKnowledgeCompileSummary, error) {
	compileSummaryPath, err := f.CompileSummaryPath()
	if err != nil {
		return WorkspaceKnowledgeCompileSummary{}, err
	}

	if _, err := os.Stat(compileSummaryPath); err != nil {
		if os.IsNotExist(err) {
			return WorkspaceKnowledgeCompileSummary{
				CompileDirty:      true,
				WikiDirty:         true,
				IncludedSourceIDs: []string{},
				FailedSourceIDs:   []string{},
				UpdatedWikiPaths:  []string{},
			}, nil
		}
		return WorkspaceKnowledgeCompileSummary{}, fmt.Errorf("stat workspace knowledge compile summary: %w", err)
	}

	var summary WorkspaceKnowledgeCompileSummary
	if err := readWorkspaceKnowledgeJSON(compileSummaryPath, &summary); err != nil {
		return WorkspaceKnowledgeCompileSummary{}, err
	}
	if summary.IncludedSourceIDs == nil {
		summary.IncludedSourceIDs = []string{}
	}
	if summary.FailedSourceIDs == nil {
		summary.FailedSourceIDs = []string{}
	}
	if summary.UpdatedWikiPaths == nil {
		summary.UpdatedWikiPaths = []string{}
	}
	if len(summary.FailedSourceIDs) > 0 {
		summary.CompileDirty = true
		summary.WikiDirty = true
	}
	return summary, nil
}

func (f workspaceKnowledgeFiles) WriteBySource(sourceSlug string, payload WorkspaceKnowledgeBySourcePayload) error {
	bySourcePath, err := f.BySourcePath(sourceSlug)
	if err != nil {
		return err
	}
	return writeWorkspaceKnowledgeJSON(bySourcePath, payload)
}

func (f workspaceKnowledgeFiles) ReadBySource(sourceSlug string) (WorkspaceKnowledgeBySourcePayload, error) {
	bySourcePath, err := f.BySourcePath(sourceSlug)
	if err != nil {
		return WorkspaceKnowledgeBySourcePayload{}, err
	}
	legacyBySourcePath, err := f.legacyBySourcePath(sourceSlug)
	if err != nil {
		return WorkspaceKnowledgeBySourcePayload{}, err
	}
	readPath, err := workspaceKnowledgeFirstExistingPath(bySourcePath, legacyBySourcePath)
	if err != nil {
		return WorkspaceKnowledgeBySourcePayload{}, fmt.Errorf("stat workspace knowledge by-source payload: %w", err)
	}
	if readPath == "" {
		return WorkspaceKnowledgeBySourcePayload{}, fmt.Errorf("read workspace knowledge json %s: %w", bySourcePath, os.ErrNotExist)
	}

	var payload WorkspaceKnowledgeBySourcePayload
	if err := readWorkspaceKnowledgeJSON(readPath, &payload); err != nil {
		return WorkspaceKnowledgeBySourcePayload{}, err
	}
	return payload, nil
}

func (f workspaceKnowledgeFiles) DeleteBySource(sourceSlug string) error {
	bySourcePath, err := f.BySourcePath(sourceSlug)
	if err != nil {
		return err
	}
	legacyBySourcePath, err := f.legacyBySourcePath(sourceSlug)
	if err != nil {
		return err
	}
	for _, path := range []string{bySourcePath, legacyBySourcePath} {
		if err := removeWorkspaceKnowledgeFile(path); err != nil {
			return fmt.Errorf("remove workspace knowledge by-source payload %s: %w", path, err)
		}
	}
	return nil
}

func (f workspaceKnowledgeFiles) DeleteMarkItDown(sourceSlug string) error {
	markItDownPath, err := f.MarkItDownPath(sourceSlug)
	if err != nil {
		return err
	}
	if err := removeWorkspaceKnowledgeFile(markItDownPath); err != nil {
		return fmt.Errorf("remove workspace knowledge markdown %s: %w", markItDownPath, err)
	}
	return nil
}

func (f workspaceKnowledgeFiles) DeleteCompiledArtifacts() error {
	paths := make([]string, 0, 13)

	entitiesPath, err := f.EntitiesPath()
	if err != nil {
		return err
	}
	paths = append(paths, entitiesPath)

	claimsPath, err := f.ClaimsPath()
	if err != nil {
		return err
	}
	paths = append(paths, claimsPath)

	relationsPath, err := f.RelationsPath()
	if err != nil {
		return err
	}
	paths = append(paths, relationsPath)

	tasksPath, err := f.TasksPath()
	if err != nil {
		return err
	}
	paths = append(paths, tasksPath)

	legacyEntitiesPath, err := f.legacyEntitiesPath()
	if err != nil {
		return err
	}
	paths = append(paths, legacyEntitiesPath)

	legacyClaimsPath, err := f.legacyClaimsPath()
	if err != nil {
		return err
	}
	paths = append(paths, legacyClaimsPath)

	legacyRelationsPath, err := f.legacyRelationsPath()
	if err != nil {
		return err
	}
	paths = append(paths, legacyRelationsPath)

	legacyTasksPath, err := f.legacyTasksPath()
	if err != nil {
		return err
	}
	paths = append(paths, legacyTasksPath)

	indexPath, err := f.IndexPath()
	if err != nil {
		return err
	}
	paths = append(paths, indexPath)

	overviewPath, err := f.OverviewPath()
	if err != nil {
		return err
	}
	paths = append(paths, overviewPath)

	openQuestionsPath, err := f.OpenQuestionsPath()
	if err != nil {
		return err
	}
	paths = append(paths, openQuestionsPath)

	logPath, err := f.LogPath()
	if err != nil {
		return err
	}
	paths = append(paths, logPath)

	for _, path := range paths {
		if err := removeWorkspaceKnowledgeFile(path); err != nil {
			return err
		}
	}

	docsDir, err := f.docsDir()
	if err != nil {
		return err
	}
	if err := removeWorkspaceKnowledgeArtifactsNotInSet(docsDir, ".md", map[string]struct{}{}); err != nil {
		return err
	}

	conceptsDir, err := f.conceptsDir()
	if err != nil {
		return err
	}
	if err := removeWorkspaceKnowledgeArtifactsNotInSet(conceptsDir, ".md", map[string]struct{}{}); err != nil {
		return err
	}

	return f.DeleteCompileSummary()
}

func (f workspaceKnowledgeFiles) WriteScanRun(record WorkspaceKnowledgeScanRunRecord) error {
	scanRunPath, err := f.ScanRunPath(record.ID)
	if err != nil {
		return err
	}
	return writeWorkspaceKnowledgeJSON(scanRunPath, record)
}

func (f workspaceKnowledgeFiles) EntitiesPath() (string, error) {
	stateDir, err := f.stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "entities.json"), nil
}

func (f workspaceKnowledgeFiles) ClaimsPath() (string, error) {
	stateDir, err := f.stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "claims.json"), nil
}

func (f workspaceKnowledgeFiles) RelationsPath() (string, error) {
	stateDir, err := f.stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "relations.json"), nil
}

func (f workspaceKnowledgeFiles) TasksPath() (string, error) {
	stateDir, err := f.stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "tasks.json"), nil
}

func (f workspaceKnowledgeFiles) OverviewPath() (string, error) {
	wikiDir, err := f.wikiDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(wikiDir, "overview.md"), nil
}

func (f workspaceKnowledgeFiles) IndexPath() (string, error) {
	wikiDir, err := f.wikiDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(wikiDir, "index.md"), nil
}

func (f workspaceKnowledgeFiles) OpenQuestionsPath() (string, error) {
	wikiDir, err := f.wikiDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(wikiDir, "open-questions.md"), nil
}

func (f workspaceKnowledgeFiles) LogPath() (string, error) {
	wikiDir, err := f.wikiDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(wikiDir, "log.md"), nil
}

func (f workspaceKnowledgeFiles) DocumentWikiPath(sourceSlug string) (string, error) {
	docsDir, err := f.docsDir()
	if err != nil {
		return "", err
	}
	validatedSourceSlug, err := validateWorkspaceKnowledgePathSegment("source slug", sourceSlug)
	if err != nil {
		return "", err
	}
	return filepath.Join(docsDir, validatedSourceSlug+".md"), nil
}

func (f workspaceKnowledgeFiles) ConceptWikiPath(conceptSlug string) (string, error) {
	conceptsDir, err := f.conceptsDir()
	if err != nil {
		return "", err
	}
	validatedConceptSlug, err := validateWorkspaceKnowledgePathSegment("concept slug", conceptSlug)
	if err != nil {
		return "", err
	}
	return filepath.Join(conceptsDir, validatedConceptSlug+".md"), nil
}

func writeWorkspaceKnowledgeJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create workspace knowledge parent directory %s: %w", filepath.Dir(path), err)
	}
	return writeJSONFile(path, payload)
}

func writeWorkspaceKnowledgeMarkdown(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create workspace knowledge parent directory %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write workspace knowledge markdown %s: %w", path, err)
	}
	return nil
}

func readWorkspaceKnowledgeJSON(path string, target any) error {
	return readJSONFile(path, target)
}

func (f workspaceKnowledgeFiles) layoutDirs() ([]string, error) {
	workspaceRootDir, err := f.workspaceRootDir()
	if err != nil {
		return nil, err
	}

	sourcesDir := filepath.Join(workspaceRootDir, "sources")
	sourcePDFsDir := filepath.Join(sourcesDir, "pdfs")
	inputsDir := filepath.Join(workspaceRootDir, "inputs")
	markItDownDir := filepath.Join(inputsDir, "markitdown")
	inputManifestsDir := filepath.Join(inputsDir, "manifests")
	stateDir := filepath.Join(workspaceRootDir, "state")
	bySourceDir := filepath.Join(stateDir, "by-source")
	jobsDir := filepath.Join(stateDir, "jobs")
	wikiDir := filepath.Join(workspaceRootDir, "wiki")
	wikiDocsDir := filepath.Join(wikiDir, "docs")
	wikiConceptsDir := filepath.Join(wikiDir, "concepts")

	return []string{
		workspaceRootDir,
		sourcesDir,
		sourcePDFsDir,
		inputsDir,
		markItDownDir,
		inputManifestsDir,
		stateDir,
		bySourceDir,
		jobsDir,
		wikiDir,
		wikiDocsDir,
		wikiConceptsDir,
	}, nil
}

func (f workspaceKnowledgeFiles) workspaceRootDir() (string, error) {
	validatedWorkspaceID, err := validateWorkspaceKnowledgePathSegment("workspace id", f.workspaceID)
	if err != nil {
		return "", err
	}
	return filepath.Join(f.paths.WorkspacesRootDir, validatedWorkspaceID), nil
}

func (f workspaceKnowledgeFiles) rawDir() (string, error) {
	return f.sourcesDir()
}

func (f workspaceKnowledgeFiles) legacyRawDir() (string, error) {
	workspaceRootDir, err := f.workspaceRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(workspaceRootDir, "raw"), nil
}

func (f workspaceKnowledgeFiles) sourcesDir() (string, error) {
	workspaceRootDir, err := f.workspaceRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(workspaceRootDir, "sources"), nil
}

func (f workspaceKnowledgeFiles) extractsDir() (string, error) {
	return f.markItDownDir()
}

func (f workspaceKnowledgeFiles) inputsDir() (string, error) {
	workspaceRootDir, err := f.workspaceRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(workspaceRootDir, "inputs"), nil
}

func (f workspaceKnowledgeFiles) markItDownDir() (string, error) {
	inputsDir, err := f.inputsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(inputsDir, "markitdown"), nil
}

func (f workspaceKnowledgeFiles) schemaDir() (string, error) {
	return f.stateDir()
}

func (f workspaceKnowledgeFiles) legacySchemaDir() (string, error) {
	workspaceRootDir, err := f.workspaceRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(workspaceRootDir, "schema"), nil
}

func (f workspaceKnowledgeFiles) stateDir() (string, error) {
	workspaceRootDir, err := f.workspaceRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(workspaceRootDir, "state"), nil
}

func (f workspaceKnowledgeFiles) bySourceDir() (string, error) {
	stateDir, err := f.stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "by-source"), nil
}

func (f workspaceKnowledgeFiles) legacyBySourceDir() (string, error) {
	schemaDir, err := f.legacySchemaDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(schemaDir, "by-source"), nil
}

func (f workspaceKnowledgeFiles) legacySourcesPath() (string, error) {
	legacyRawDir, err := f.legacyRawDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(legacyRawDir, "sources.json"), nil
}

func (f workspaceKnowledgeFiles) legacyBySourcePath(sourceSlug string) (string, error) {
	legacyBySourceDir, err := f.legacyBySourceDir()
	if err != nil {
		return "", err
	}
	validatedSourceSlug, err := validateWorkspaceKnowledgePathSegment("source slug", sourceSlug)
	if err != nil {
		return "", err
	}
	return filepath.Join(legacyBySourceDir, validatedSourceSlug+".json"), nil
}

func (f workspaceKnowledgeFiles) legacyEntitiesPath() (string, error) {
	legacySchemaDir, err := f.legacySchemaDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(legacySchemaDir, "entities.json"), nil
}

func (f workspaceKnowledgeFiles) legacyClaimsPath() (string, error) {
	legacySchemaDir, err := f.legacySchemaDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(legacySchemaDir, "claims.json"), nil
}

func (f workspaceKnowledgeFiles) legacyRelationsPath() (string, error) {
	legacySchemaDir, err := f.legacySchemaDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(legacySchemaDir, "relations.json"), nil
}

func (f workspaceKnowledgeFiles) legacyTasksPath() (string, error) {
	legacySchemaDir, err := f.legacySchemaDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(legacySchemaDir, "tasks.json"), nil
}

func (f workspaceKnowledgeFiles) scanRunsDir() (string, error) {
	return f.jobsDir()
}

func (f workspaceKnowledgeFiles) jobsDir() (string, error) {
	stateDir, err := f.stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "jobs"), nil
}

func (f workspaceKnowledgeFiles) wikiDir() (string, error) {
	workspaceRootDir, err := f.workspaceRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(workspaceRootDir, "wiki"), nil
}

func (f workspaceKnowledgeFiles) docsDir() (string, error) {
	wikiDir, err := f.wikiDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(wikiDir, "docs"), nil
}

func (f workspaceKnowledgeFiles) conceptsDir() (string, error) {
	wikiDir, err := f.wikiDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(wikiDir, "concepts"), nil
}

func validateWorkspaceKnowledgePathSegment(name, value string) (string, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	if trimmedValue == "." || trimmedValue == ".." {
		return "", fmt.Errorf("%s must not contain traversal segments", name)
	}
	if filepath.IsAbs(trimmedValue) {
		return "", fmt.Errorf("%s must not be absolute", name)
	}
	if strings.ContainsAny(trimmedValue, `<>:"/\|?*`) {
		return "", fmt.Errorf("%s must not contain Windows-invalid filename characters", name)
	}
	return trimmedValue, nil
}

func workspaceKnowledgeJSONPaths(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}
	return paths, nil
}

func workspaceKnowledgeFirstExistingPath(paths ...string) (string, error) {
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", nil
}
