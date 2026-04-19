package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const appDirName = ".openscireader"

type appPaths struct {
	RootDir                  string
	AppConfigDBPath          string
	OCRCacheDBPath           string
	EncryptionKeyPath        string
	TranslateRootDir         string
	TranslateJobsDir         string
	WikiRootDir              string
	WikiJobsDir              string
	TranslateRuntimeRootDir  string
	TranslateRuntimeCacheDir string
	LibraryRootDir           string
	WorkspacesRootDir        string
}

func resolveAppPaths() (appPaths, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return appPaths{}, fmt.Errorf("resolve user config directory: %w", err)
	}

	rootDir := filepath.Join(configDir, appDirName)
	paths := appPaths{
		RootDir:                  rootDir,
		AppConfigDBPath:          filepath.Join(rootDir, "app_config.sqlite"),
		OCRCacheDBPath:           filepath.Join(rootDir, "ocr_cache.sqlite"),
		EncryptionKeyPath:        filepath.Join(rootDir, "config.key"),
		TranslateRootDir:         filepath.Join(rootDir, "reader_translate"),
		TranslateJobsDir:         filepath.Join(rootDir, "reader_translate", "jobs"),
		WikiRootDir:              filepath.Join(rootDir, "workspace_wiki"),
		WikiJobsDir:              filepath.Join(rootDir, "workspace_wiki", "jobs"),
		TranslateRuntimeRootDir:  filepath.Join(rootDir, "reader_translate", "runtime"),
		TranslateRuntimeCacheDir: filepath.Join(rootDir, "reader_translate", "runtime-cache"),
		LibraryRootDir:           filepath.Join(rootDir, "library"),
		WorkspacesRootDir:        filepath.Join(rootDir, "library", "workspaces"),
	}

	for _, directory := range []string{paths.RootDir, paths.TranslateJobsDir, paths.WikiJobsDir, paths.TranslateRuntimeRootDir, paths.TranslateRuntimeCacheDir, paths.LibraryRootDir, paths.WorkspacesRootDir} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return appPaths{}, fmt.Errorf("create app data directory %s: %w", directory, err)
		}
	}

	return paths, nil
}
