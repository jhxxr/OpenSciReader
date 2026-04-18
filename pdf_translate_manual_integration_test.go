package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"OpenSciReader/internal/translator"
)

func TestManualPreviewExportReuseForWang2017(t *testing.T) {
	pdfPath := manualIntegrationPDFPath(t)
	if _, err := os.Stat(pdfPath); err != nil {
		t.Skipf("manual integration PDF not available at %q: %v", pdfPath, err)
	}

	paths, err := resolveAppPaths()
	if err != nil {
		t.Fatalf("resolve app paths: %v", err)
	}
	paths = cloneAppPathsForTest(t, paths)

	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("new config store: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	providerID, modelRecord, err := firstActiveLLMProviderAndModel(t.Context(), store)
	if err != nil {
		t.Fatalf("load llm provider/model: %v", err)
	}
	provider, err := store.GetProviderSecret(t.Context(), providerID)
	if err != nil {
		t.Fatalf("load provider secret: %v", err)
	}

	jobRoot := t.TempDir()
	jobRoot = workspaceTempDir(t, "manual-jobs-")
	runtime := resolvePDFTranslateRuntime(paths, store)
	manager, err := translator.NewManager(translator.Options{
		DataRootDir:       jobRoot,
		RuntimeDir:        runtime.Config.RuntimeDir,
		WorkerScriptPath:  resolvePDFTranslateWorkerScriptPath(),
		PythonSearchPaths: resolvePDFTranslatePythonCandidates(runtime.Config.RuntimeDir),
	})
	if err != nil {
		t.Fatalf("new translator manager: %v", err)
	}

	request := translator.StartRequest{
		PDFPath:            pdfPath,
		PageCount:          7,
		ItemID:             "manual-wang2017",
		ItemTitle:          "wang2017",
		SourceLang:         "en",
		TargetLang:         "zh-CN",
		Mode:               translator.ModePreview,
		PreviewChunkPages:  25,
		QPS:                1,
		PoolMaxWorkers:     1,
		TermPoolMaxWorkers: 1,
		Provider: translator.ProviderConfig{
			ProviderID:   provider.ID,
			ProviderName: provider.Name,
			BaseURL:      provider.BaseURL,
			APIKey:       provider.APIKey,
			ModelID:      modelRecord.ModelID,
		},
	}

	previewJob, err := manager.Start(context.Background(), request)
	if err != nil {
		t.Fatalf("start preview: %v", err)
	}

	previewFinal := waitForTerminalJob(t, manager, previewJob.JobID, 10*time.Minute)
	if previewFinal.Status != translator.JobStatusCompleted {
		t.Fatalf("preview status = %s, error = %s", previewFinal.Status, previewFinal.Error)
	}
	if len(previewFinal.Chunks) != 1 {
		t.Fatalf("preview chunk count = %d, want 1", len(previewFinal.Chunks))
	}
	if strings.TrimSpace(previewFinal.Chunks[0].TranslatedPDFPath) == "" {
		t.Fatalf("preview translated pdf path is empty")
	}
	if strings.TrimSpace(previewFinal.Chunks[0].DualPDFPath) == "" {
		t.Fatalf("preview dual pdf path is empty")
	}

	exportRequest := request
	exportRequest.Mode = translator.ModeExport
	exportRequest.MaxPagesPerPart = 120
	exportRequest.ReusePreviewJobID = previewFinal.JobID

	exportJob, err := manager.Start(context.Background(), exportRequest)
	if err != nil {
		t.Fatalf("start export: %v", err)
	}

	_, events, unsubscribe, err := manager.Subscribe(exportJob.JobID)
	if err != nil {
		t.Fatalf("subscribe export job: %v", err)
	}
	defer unsubscribe()

	sawMergeStage := false
	done := make(chan struct{})
	go func() {
		defer close(done)
		for event := range events {
			stage := strings.TrimSpace(event.Stage)
			if strings.EqualFold(stage, "merge preview outputs") {
				sawMergeStage = true
			}
		}
	}()

	exportFinal := waitForTerminalJob(t, manager, exportJob.JobID, 3*time.Minute)
	unsubscribe()
	<-done

	if exportFinal.Status != translator.JobStatusCompleted {
		t.Fatalf("export status = %s, error = %s", exportFinal.Status, exportFinal.Error)
	}
	if !sawMergeStage {
		t.Fatalf("export did not emit merge preview outputs stage")
	}
	if strings.TrimSpace(exportFinal.Outputs.NoWatermarkMonoPDFPath) == "" &&
		strings.TrimSpace(exportFinal.Outputs.MonoPDFPath) == "" {
		t.Fatalf("export mono output path is empty")
	}
	if strings.TrimSpace(exportFinal.Outputs.NoWatermarkMixedPDFPath) == "" &&
		strings.TrimSpace(exportFinal.Outputs.MixedPDFPath) == "" {
		t.Fatalf("export mixed output path is empty")
	}
	if strings.TrimSpace(exportFinal.Outputs.NoWatermarkDualPDFPath) == "" &&
		strings.TrimSpace(exportFinal.Outputs.DualPDFPath) == "" {
		t.Fatalf("export dual output path is empty")
	}
}

func waitForTerminalJob(
	t *testing.T,
	manager *translator.Manager,
	jobID string,
	timeout time.Duration,
) translator.JobSnapshot {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		snapshot, err := manager.GetStatus(jobID)
		if err != nil {
			t.Fatalf("get status %s: %v", jobID, err)
		}
		switch snapshot.Status {
		case translator.JobStatusCompleted, translator.JobStatusFailed, translator.JobStatusCancelled:
			return snapshot
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatalf("timed out waiting for job %s", jobID)
	return translator.JobSnapshot{}
}

func firstActiveLLMProviderAndModel(
	ctx context.Context,
	store *configStore,
) (int64, ModelRecord, error) {
	snapshot, err := store.GetConfigSnapshot(ctx)
	if err != nil {
		return 0, ModelRecord{}, err
	}
	for _, providerConfig := range snapshot.Providers {
		if providerConfig.Provider.Type != ProviderTypeLLM || !providerConfig.Provider.IsActive {
			continue
		}
		if len(providerConfig.Models) == 0 {
			continue
		}
		return providerConfig.Provider.ID, providerConfig.Models[0], nil
	}
	return 0, ModelRecord{}, os.ErrNotExist
}

func cloneAppPathsForTest(t *testing.T, source appPaths) appPaths {
	t.Helper()

	rootDir := workspaceTempDir(t, "manual-config-")
	cloned := appPaths{
		RootDir:                  rootDir,
		AppConfigDBPath:          filepath.Join(rootDir, "app_config.sqlite"),
		OCRCacheDBPath:           filepath.Join(rootDir, "ocr_cache.sqlite"),
		EncryptionKeyPath:        filepath.Join(rootDir, "config.key"),
		TranslateRootDir:         filepath.Join(rootDir, "reader_translate"),
		TranslateJobsDir:         filepath.Join(rootDir, "reader_translate", "jobs"),
		TranslateRuntimeRootDir:  filepath.Join(rootDir, "reader_translate", "runtime"),
		TranslateRuntimeCacheDir: filepath.Join(rootDir, "reader_translate", "runtime-cache"),
	}
	for _, directory := range []string{cloned.RootDir, cloned.TranslateJobsDir, cloned.TranslateRuntimeRootDir, cloned.TranslateRuntimeCacheDir} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatalf("mkdir %s: %v", directory, err)
		}
	}
	copyFileForTest(t, source.AppConfigDBPath, cloned.AppConfigDBPath)
	copyFileForTest(t, source.OCRCacheDBPath, cloned.OCRCacheDBPath)
	copyFileForTest(t, source.EncryptionKeyPath, cloned.EncryptionKeyPath)
	return cloned
}

func copyFileForTest(t *testing.T, src, dst string) {
	t.Helper()

	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open %s: %v", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("create %s: %v", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy %s -> %s: %v", src, dst, err)
	}
}

func workspaceTempDir(t *testing.T, pattern string) string {
	t.Helper()

	root := filepath.Join(".", ".tmp-manual")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", root, err)
	}
	path, err := os.MkdirTemp(root, pattern)
	if err != nil {
		t.Fatalf("mktemp %s: %v", root, err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(path)
	})
	return path
}

func manualIntegrationPDFPath(t *testing.T) string {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping manual integration test in short mode")
	}

	if os.Getenv("OPENSCIREADER_RUN_MANUAL_INTEGRATION_TESTS") != "1" {
		t.Skip("set OPENSCIREADER_RUN_MANUAL_INTEGRATION_TESTS=1 to run manual PDF integration tests")
	}

	pdfPath := strings.TrimSpace(os.Getenv("OPENSCIREADER_MANUAL_TEST_PDF"))
	if pdfPath == "" {
		t.Skip("set OPENSCIREADER_MANUAL_TEST_PDF to a local sample PDF path")
	}

	return pdfPath
}
