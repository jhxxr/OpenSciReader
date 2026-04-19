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
	rawDir, err := f.rawDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(rawDir, "sources.json"), nil
}

func (f workspaceKnowledgeFiles) ExtractPath(sourceSlug string) (string, error) {
	extractsDir, err := f.extractsDir()
	if err != nil {
		return "", err
	}
	validatedSourceSlug, err := validateWorkspaceKnowledgePathSegment("source slug", sourceSlug)
	if err != nil {
		return "", err
	}
	return filepath.Join(extractsDir, validatedSourceSlug+".md"), nil
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

func (f workspaceKnowledgeFiles) ScanRunPath(scanRunID string) (string, error) {
	scanRunsDir, err := f.scanRunsDir()
	if err != nil {
		return "", err
	}
	validatedScanRunID, err := validateWorkspaceKnowledgePathSegment("scan run id", scanRunID)
	if err != nil {
		return "", err
	}
	return filepath.Join(scanRunsDir, validatedScanRunID+".json"), nil
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

func writeWorkspaceKnowledgeJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create workspace knowledge parent directory %s: %w", filepath.Dir(path), err)
	}
	return writeJSONFile(path, payload)
}

func readWorkspaceKnowledgeJSON(path string, target any) error {
	return readJSONFile(path, target)
}

func (f workspaceKnowledgeFiles) layoutDirs() ([]string, error) {
	workspaceRootDir, err := f.workspaceRootDir()
	if err != nil {
		return nil, err
	}

	rawDir := filepath.Join(workspaceRootDir, "raw")
	extractsDir := filepath.Join(rawDir, "extracts")
	schemaDir := filepath.Join(workspaceRootDir, "schema")
	bySourceDir := filepath.Join(schemaDir, "by-source")
	scanRunsDir := filepath.Join(schemaDir, "scan-runs")
	wikiDir := filepath.Join(workspaceRootDir, "wiki")
	wikiDocsDir := filepath.Join(wikiDir, "docs")
	wikiConceptsDir := filepath.Join(wikiDir, "concepts")

	return []string{
		workspaceRootDir,
		rawDir,
		extractsDir,
		schemaDir,
		bySourceDir,
		scanRunsDir,
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
	workspaceRootDir, err := f.workspaceRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(workspaceRootDir, "raw"), nil
}

func (f workspaceKnowledgeFiles) extractsDir() (string, error) {
	rawDir, err := f.rawDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(rawDir, "extracts"), nil
}

func (f workspaceKnowledgeFiles) schemaDir() (string, error) {
	workspaceRootDir, err := f.workspaceRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(workspaceRootDir, "schema"), nil
}

func (f workspaceKnowledgeFiles) bySourceDir() (string, error) {
	schemaDir, err := f.schemaDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(schemaDir, "by-source"), nil
}

func (f workspaceKnowledgeFiles) scanRunsDir() (string, error) {
	schemaDir, err := f.schemaDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(schemaDir, "scan-runs"), nil
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
	if strings.ContainsAny(trimmedValue, `/\`) {
		return "", fmt.Errorf("%s must not contain path separators", name)
	}
	return trimmedValue, nil
}
