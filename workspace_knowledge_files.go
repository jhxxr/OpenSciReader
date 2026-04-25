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

	entries, err := os.ReadDir(bySourceDir)
	if err != nil {
		return nil, fmt.Errorf("read workspace knowledge by-source directory %s: %w", bySourceDir, err)
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		paths = append(paths, filepath.Join(bySourceDir, entry.Name()))
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

	if _, err := os.Stat(sourcesPath); err != nil {
		if os.IsNotExist(err) {
			return []WorkspaceKnowledgeSource{}, nil
		}
		return nil, fmt.Errorf("stat workspace knowledge sources: %w", err)
	}

	var sources []WorkspaceKnowledgeSource
	if err := readWorkspaceKnowledgeJSON(sourcesPath, &sources); err != nil {
		return nil, err
	}
	if sources == nil {
		return []WorkspaceKnowledgeSource{}, nil
	}
	return sources, nil
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

	var payload WorkspaceKnowledgeBySourcePayload
	if err := readWorkspaceKnowledgeJSON(bySourcePath, &payload); err != nil {
		return WorkspaceKnowledgeBySourcePayload{}, err
	}
	return payload, nil
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
