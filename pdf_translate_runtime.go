package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"OpenSciReader/internal/translator"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	pdfTranslateRuntimeSettingKey = "pdf_translate_runtime"
	pdfTranslateRuntimeID         = "pdf2zh-next"
	pdfTranslateRuntimePlatform   = "windows-amd64"
)

type pdfTranslateRuntimeManifest struct {
	SchemaVersion string `json:"schemaVersion"`
	RuntimeID     string `json:"runtimeId"`
	Version       string `json:"version"`
	Platform      string `json:"platform"`
}

type resolvedPDFTranslateRuntime struct {
	Config PDFTranslateRuntimeConfig
}

func newPDFTranslateManagerOrPanic(paths appPaths, store *configStore, ctx context.Context) *translator.Manager {
	runtime := resolvePDFTranslateRuntime(paths, store)
	manager, err := translator.NewManager(translator.Options{
		DataRootDir:       paths.TranslateJobsDir,
		RuntimeDir:        runtime.Config.RuntimeDir,
		WorkerScriptPath:  resolvePDFTranslateWorkerScriptPath(),
		PythonSearchPaths: resolvePDFTranslatePythonCandidates(runtime.Config.RuntimeDir),
		EventSink: func(event translator.JobEvent) {
			if event.JobID == "" {
				return
			}
			wruntime.EventsEmit(ctx, pdfTranslateRuntimeEventName(event.JobID), event)
		},
	})
	if err != nil {
		panic(err)
	}
	return manager
}

func resolvePDFTranslateWorkerScriptPath() string {
	candidates := []string{
		filepath.Join(executableDir(), "python_worker", "worker.py"),
		filepath.Join(projectWorkingDir(), "python_worker", "worker.py"),
	}
	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}
	return candidates[0]
}

func pdfTranslateRuntimeEventName(jobID string) string {
	return "pdf-translate:event:" + jobID
}

func resolveBundledPDFTranslateRuntimeDir() string {
	candidates := []string{
		filepath.Join(executableDir(), "runtime", "pdf2zh-next", "windows-amd64", "_upstream_extract", "pdf2zh"),
		filepath.Join(projectWorkingDir(), "runtime", "pdf2zh-next", "windows-amd64", "_upstream_extract", "pdf2zh"),
		filepath.Join(executableDir(), "runtime", "pdf2zh-next", "windows-amd64"),
		filepath.Join(projectWorkingDir(), "runtime", "pdf2zh-next", "windows-amd64"),
		filepath.Join(executableDir(), "runtime", "pdf2zh", "windows-amd64"),
		filepath.Join(projectWorkingDir(), "runtime", "pdf2zh", "windows-amd64"),
	}
	for _, candidate := range candidates {
		if dirExists(candidate) {
			return candidate
		}
	}
	return candidates[0]
}

func resolvePDFTranslateRuntime(paths appPaths, store *configStore) resolvedPDFTranslateRuntime {
	if store != nil {
		config, err := store.GetPDFTranslateRuntimeConfig(context.Background())
		if err == nil {
			validated := validateStoredPDFTranslateRuntimeConfig(config)
			if validated.Installed && validated.Status == PDFTranslateRuntimeStatusValid {
				return resolvedPDFTranslateRuntime{Config: validated}
			}
		}
	}

	bundled := defaultMissingPDFTranslateRuntimeConfig()
	bundled.RuntimeDir = resolveBundledPDFTranslateRuntimeDir()
	bundled = validateStoredPDFTranslateRuntimeConfig(bundled)
	return resolvedPDFTranslateRuntime{Config: bundled}
}

func resolvePDFTranslatePythonCandidates(runtimeDir string) []string {
	if strings.TrimSpace(runtimeDir) == "" {
		return nil
	}
	return []string{
		filepath.Join(runtimeDir, "runtime", "python.exe"),
		filepath.Join(runtimeDir, "python", "python.exe"),
		filepath.Join(runtimeDir, "python.exe"),
	}
}

func defaultMissingPDFTranslateRuntimeConfig() PDFTranslateRuntimeConfig {
	return PDFTranslateRuntimeConfig{
		Installed: false,
		Status:    PDFTranslateRuntimeStatusMissing,
		RuntimeID: pdfTranslateRuntimeID,
		Platform:  pdfTranslateRuntimePlatform,
	}
}

func normalizePDFTranslateRuntimeConfig(input PDFTranslateRuntimeConfig) PDFTranslateRuntimeConfig {
	config := defaultMissingPDFTranslateRuntimeConfig()
	if input.Status != "" {
		config.Status = input.Status
	}
	if trimmed := strings.TrimSpace(input.RuntimeID); trimmed != "" {
		config.RuntimeID = trimmed
	}
	if trimmed := strings.TrimSpace(input.Version); trimmed != "" {
		config.Version = trimmed
	}
	if trimmed := strings.TrimSpace(input.Platform); trimmed != "" {
		config.Platform = trimmed
	}
	config.RuntimeDir = strings.TrimSpace(input.RuntimeDir)
	config.PythonPath = strings.TrimSpace(input.PythonPath)
	config.ManifestPath = strings.TrimSpace(input.ManifestPath)
	config.InstalledAt = strings.TrimSpace(input.InstalledAt)
	config.SourceFileName = strings.TrimSpace(input.SourceFileName)
	config.LastValidationError = strings.TrimSpace(input.LastValidationError)
	config.Installed = input.Installed
	return config
}

func validateStoredPDFTranslateRuntimeConfig(input PDFTranslateRuntimeConfig) PDFTranslateRuntimeConfig {
	config := normalizePDFTranslateRuntimeConfig(input)
	if config.RuntimeDir == "" {
		config.Installed = false
		config.Status = PDFTranslateRuntimeStatusMissing
		if config.LastValidationError == "" {
			config.LastValidationError = "PDF translation runtime is not installed"
		}
		return config
	}

	if !dirExists(config.RuntimeDir) {
		config.Installed = false
		config.Status = PDFTranslateRuntimeStatusMissing
		config.LastValidationError = fmt.Sprintf("runtime directory not found: %s", config.RuntimeDir)
		return config
	}
	if !dirExists(filepath.Join(config.RuntimeDir, "site-packages")) {
		config.Installed = false
		config.Status = PDFTranslateRuntimeStatusInvalid
		config.LastValidationError = "runtime is missing site-packages"
		return config
	}
	if manifestPath := strings.TrimSpace(config.ManifestPath); manifestPath != "" && !fileExists(manifestPath) {
		config.Installed = false
		config.Status = PDFTranslateRuntimeStatusInvalid
		config.LastValidationError = fmt.Sprintf("runtime manifest not found: %s", manifestPath)
		return config
	}
	for _, candidate := range resolvePDFTranslatePythonCandidates(config.RuntimeDir) {
		if fileExists(candidate) {
			config.PythonPath = candidate
			config.Installed = true
			config.Status = PDFTranslateRuntimeStatusValid
			config.LastValidationError = ""
			if config.ManifestPath == "" {
				manifestPath := filepath.Join(config.RuntimeDir, "manifest.json")
				if fileExists(manifestPath) {
					config.ManifestPath = manifestPath
				}
			}
			return config
		}
	}
	config.Installed = false
	config.Status = PDFTranslateRuntimeStatusInvalid
	config.LastValidationError = "runtime is missing an embedded python executable"
	return config
}

func importPDFTranslateRuntimePackage(paths appPaths, packagePath string, progress *runtimeImportProgressEmitter) (PDFTranslateRuntimeConfig, error) {
	packagePath = strings.TrimSpace(packagePath)
	if packagePath == "" {
		return PDFTranslateRuntimeConfig{}, fmt.Errorf("runtime package path is required")
	}
	if !fileExists(packagePath) {
		return PDFTranslateRuntimeConfig{}, fmt.Errorf("runtime package not found: %s", packagePath)
	}

	stagingDir, err := os.MkdirTemp(paths.TranslateRuntimeCacheDir, "runtime-import-")
	if err != nil {
		return PDFTranslateRuntimeConfig{}, fmt.Errorf("create runtime staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	if err := extractZip(packagePath, stagingDir, progress); err != nil {
		return PDFTranslateRuntimeConfig{}, err
	}

	progress.Emit("validating", "正在校验运行时安装包", 0.88, 0, 0)
	runtimeRoot, manifest, err := detectImportedPDFTranslateRuntime(stagingDir)
	if err != nil {
		return PDFTranslateRuntimeConfig{}, err
	}
	if manifest.RuntimeID != "" && manifest.RuntimeID != pdfTranslateRuntimeID {
		return PDFTranslateRuntimeConfig{}, fmt.Errorf("unexpected runtime id: %s", manifest.RuntimeID)
	}
	if manifest.Platform != "" && manifest.Platform != pdfTranslateRuntimePlatform {
		return PDFTranslateRuntimeConfig{}, fmt.Errorf("runtime platform %s is not supported by this build", manifest.Platform)
	}

	progress.Emit("activating", "正在启用导入的运行时", 0.93, 0, 0)
	targetDir := filepath.Join(paths.TranslateRuntimeRootDir, sanitizeRuntimeVersion(manifest.Version))
	if err := os.RemoveAll(targetDir); err != nil {
		return PDFTranslateRuntimeConfig{}, fmt.Errorf("remove existing runtime directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o700); err != nil {
		return PDFTranslateRuntimeConfig{}, fmt.Errorf("prepare runtime root directory: %w", err)
	}
	if err := os.Rename(runtimeRoot, targetDir); err != nil {
		return PDFTranslateRuntimeConfig{}, fmt.Errorf("activate imported runtime: %w", err)
	}

	progress.Emit("validating", "正在校验导入后的运行时", 0.95, 0, 0)
	config := validateStoredPDFTranslateRuntimeConfig(PDFTranslateRuntimeConfig{
		Installed:      true,
		Status:         PDFTranslateRuntimeStatusInstalling,
		RuntimeID:      firstNonEmptyString(manifest.RuntimeID, pdfTranslateRuntimeID),
		Version:        firstNonEmptyString(manifest.Version, "unknown"),
		Platform:       firstNonEmptyString(manifest.Platform, pdfTranslateRuntimePlatform),
		RuntimeDir:     targetDir,
		ManifestPath:   filepath.Join(targetDir, "manifest.json"),
		InstalledAt:    nowRFC3339(),
		SourceFileName: filepath.Base(packagePath),
	})
	if config.Status != PDFTranslateRuntimeStatusValid {
		return PDFTranslateRuntimeConfig{}, fmt.Errorf("%s", config.LastValidationError)
	}
	return config, nil
}

func removeImportedPDFTranslateRuntime(config PDFTranslateRuntimeConfig) error {
	if runtimeDir := strings.TrimSpace(config.RuntimeDir); runtimeDir != "" && dirExists(runtimeDir) {
		if err := os.RemoveAll(runtimeDir); err != nil {
			return fmt.Errorf("remove runtime directory: %w", err)
		}
	}
	return nil
}

func detectImportedPDFTranslateRuntime(root string) (string, pdfTranslateRuntimeManifest, error) {
	manifestPath, err := findFileByName(root, "manifest.json")
	if err != nil {
		return "", pdfTranslateRuntimeManifest{}, fmt.Errorf("locate runtime manifest: %w", err)
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", pdfTranslateRuntimeManifest{}, fmt.Errorf("read runtime manifest: %w", err)
	}
	var manifest pdfTranslateRuntimeManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", pdfTranslateRuntimeManifest{}, fmt.Errorf("decode runtime manifest: %w", err)
	}
	runtimeRoot := filepath.Dir(manifestPath)
	if !dirExists(filepath.Join(runtimeRoot, "site-packages")) {
		return "", pdfTranslateRuntimeManifest{}, fmt.Errorf("runtime package missing site-packages directory")
	}
	return runtimeRoot, manifest, nil
}

func extractZip(zipPath, destDir string, progress *runtimeImportProgressEmitter) error {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open runtime package: %w", err)
	}
	defer archive.Close()

	totalBytes := int64(0)
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		totalBytes += zipEntryUncompressedSize(file)
	}

	completedBytes := int64(0)
	lastProgressEmit := time.Time{}
	emitExtractProgress := func(force bool) {
		if progress == nil {
			return
		}
		now := time.Now()
		if !force && !lastProgressEmit.IsZero() && now.Sub(lastProgressEmit) < 120*time.Millisecond {
			return
		}
		lastProgressEmit = now
		progress.Emit(
			"extracting",
			"正在解压运行时文件",
			runtimeImportExtractionProgress(completedBytes, totalBytes),
			completedBytes,
			totalBytes,
		)
	}

	emitExtractProgress(true)
	for _, file := range archive.File {
		targetPath := filepath.Join(destDir, file.Name)
		cleanTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleanTarget, filepath.Clean(destDir)+string(os.PathSeparator)) && cleanTarget != filepath.Clean(destDir) {
			return fmt.Errorf("runtime package contains invalid path: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0o700); err != nil {
				return fmt.Errorf("create runtime directory %s: %w", cleanTarget, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o700); err != nil {
			return fmt.Errorf("create parent directory for %s: %w", cleanTarget, err)
		}
		src, err := file.Open()
		if err != nil {
			return fmt.Errorf("open archived file %s: %w", file.Name, err)
		}
		dst, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			src.Close()
			return fmt.Errorf("create extracted file %s: %w", cleanTarget, err)
		}

		buffer := make([]byte, 32*1024)
		for {
			bytesRead, readErr := src.Read(buffer)
			if bytesRead > 0 {
				if _, err := dst.Write(buffer[:bytesRead]); err != nil {
					dst.Close()
					src.Close()
					return fmt.Errorf("extract file %s: %w", cleanTarget, err)
				}
				completedBytes += int64(bytesRead)
				emitExtractProgress(false)
			}
			if readErr == nil {
				continue
			}
			if readErr != io.EOF {
				dst.Close()
				src.Close()
				return fmt.Errorf("extract file %s: %w", cleanTarget, readErr)
			}
			break
		}
		dst.Close()
		src.Close()
		emitExtractProgress(true)
	}
	return nil
}

func runtimeImportExtractionProgress(completedBytes, totalBytes int64) float64 {
	const (
		extractStart = 0.08
		extractEnd   = 0.82
	)

	if totalBytes <= 0 {
		return extractStart
	}
	ratio := float64(completedBytes) / float64(totalBytes)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return extractStart + (extractEnd-extractStart)*ratio
}

func zipEntryUncompressedSize(file *zip.File) int64 {
	if file == nil {
		return 0
	}
	if file.UncompressedSize64 > uint64(^uint64(0)>>1) {
		return int64(^uint64(0) >> 1)
	}
	return int64(file.UncompressedSize64)
}

func findFileByName(root, name string) (string, error) {
	var match string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(d.Name(), name) {
			match = path
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return "", err
	}
	if strings.TrimSpace(match) == "" {
		return "", fmt.Errorf("%s not found", name)
	}
	return match, nil
}

func sanitizeRuntimeVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	return replacer.Replace(trimmed)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func executableDir() string {
	path, err := os.Executable()
	if err != nil {
		return projectWorkingDir()
	}
	return filepath.Dir(path)
}

func projectWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
