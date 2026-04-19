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
	for _, directory := range []string{
		f.workspaceRootDir(),
		f.rawDir(),
		f.extractsDir(),
		f.schemaDir(),
		f.bySourceDir(),
		f.scanRunsDir(),
		f.wikiDir(),
		f.wikiDocsDir(),
		f.wikiConceptsDir(),
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return fmt.Errorf("create workspace knowledge directory %s: %w", directory, err)
		}
	}
	return nil
}

func (f workspaceKnowledgeFiles) SourcesPath() string {
	return filepath.Join(f.rawDir(), "sources.json")
}

func (f workspaceKnowledgeFiles) ExtractPath(sourceSlug string) string {
	return filepath.Join(f.extractsDir(), strings.TrimSpace(sourceSlug)+".md")
}

func (f workspaceKnowledgeFiles) BySourcePath(sourceSlug string) string {
	return filepath.Join(f.bySourceDir(), strings.TrimSpace(sourceSlug)+".json")
}

func (f workspaceKnowledgeFiles) ScanRunPath(scanRunID string) string {
	return filepath.Join(f.scanRunsDir(), strings.TrimSpace(scanRunID)+".json")
}

func (f workspaceKnowledgeFiles) WriteSources(sources []WorkspaceKnowledgeSource) error {
	return writeWorkspaceKnowledgeJSON(f.SourcesPath(), sources)
}

func (f workspaceKnowledgeFiles) ReadSources() ([]WorkspaceKnowledgeSource, error) {
	if _, err := os.Stat(f.SourcesPath()); err != nil {
		if os.IsNotExist(err) {
			return []WorkspaceKnowledgeSource{}, nil
		}
		return nil, fmt.Errorf("stat workspace knowledge sources: %w", err)
	}

	var sources []WorkspaceKnowledgeSource
	if err := readWorkspaceKnowledgeJSON(f.SourcesPath(), &sources); err != nil {
		return nil, err
	}
	if sources == nil {
		return []WorkspaceKnowledgeSource{}, nil
	}
	return sources, nil
}

func (f workspaceKnowledgeFiles) WriteBySource(sourceSlug string, payload WorkspaceKnowledgeBySourcePayload) error {
	return writeWorkspaceKnowledgeJSON(f.BySourcePath(sourceSlug), payload)
}

func (f workspaceKnowledgeFiles) ReadBySource(sourceSlug string) (WorkspaceKnowledgeBySourcePayload, error) {
	var payload WorkspaceKnowledgeBySourcePayload
	if err := readWorkspaceKnowledgeJSON(f.BySourcePath(sourceSlug), &payload); err != nil {
		return WorkspaceKnowledgeBySourcePayload{}, err
	}
	return payload, nil
}

func (f workspaceKnowledgeFiles) WriteScanRun(record WorkspaceKnowledgeScanRunRecord) error {
	runID := strings.TrimSpace(record.ID)
	if runID == "" {
		return fmt.Errorf("scan run id is required")
	}
	return writeWorkspaceKnowledgeJSON(f.ScanRunPath(runID), record)
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

func (f workspaceKnowledgeFiles) workspaceRootDir() string {
	return filepath.Join(f.paths.WorkspacesRootDir, f.workspaceID)
}

func (f workspaceKnowledgeFiles) rawDir() string {
	return filepath.Join(f.workspaceRootDir(), "raw")
}

func (f workspaceKnowledgeFiles) extractsDir() string {
	return filepath.Join(f.rawDir(), "extracts")
}

func (f workspaceKnowledgeFiles) schemaDir() string {
	return filepath.Join(f.workspaceRootDir(), "schema")
}

func (f workspaceKnowledgeFiles) bySourceDir() string {
	return filepath.Join(f.schemaDir(), "by-source")
}

func (f workspaceKnowledgeFiles) scanRunsDir() string {
	return filepath.Join(f.schemaDir(), "scan-runs")
}

func (f workspaceKnowledgeFiles) wikiDir() string {
	return filepath.Join(f.workspaceRootDir(), "wiki")
}

func (f workspaceKnowledgeFiles) wikiDocsDir() string {
	return filepath.Join(f.wikiDir(), "docs")
}

func (f workspaceKnowledgeFiles) wikiConceptsDir() string {
	return filepath.Join(f.wikiDir(), "concepts")
}
