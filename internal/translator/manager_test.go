package translator

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCanReusePreviewForExport(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	pdfPath := filepath.Join(tempDir, "demo.pdf")
	chunkMonoOne := filepath.Join(tempDir, "chunk-001.mono.pdf")
	chunkDualOne := filepath.Join(tempDir, "chunk-001.dual.pdf")
	chunkMonoTwo := filepath.Join(tempDir, "chunk-002.mono.pdf")
	chunkDualTwo := filepath.Join(tempDir, "chunk-002.dual.pdf")
	request := StartRequest{
		PDFPath:    pdfPath,
		PageCount:  50,
		SourceLang: "en",
		TargetLang: "zh-CN",
		Mode:       ModeExport,
		Provider: ProviderConfig{
			ProviderID: 7,
			ModelID:    "gpt-5.4",
		},
	}
	preview := JobSnapshot{
		Mode:       ModePreview,
		Status:     JobStatusCompleted,
		PDFPath:    strings.ToUpper(pdfPath),
		PageCount:  50,
		SourceLang: "EN",
		TargetLang: "zh-CN",
		ProviderID: 7,
		ModelID:    "gpt-5.4",
		Chunks: []ChunkStatus{
			{
				Index:             1,
				Status:            JobStatusCompleted,
				TranslatedPDFPath: chunkMonoOne,
				DualPDFPath:       chunkDualOne,
			},
			{
				Index:             2,
				Status:            JobStatusCompleted,
				TranslatedPDFPath: chunkMonoTwo,
				DualPDFPath:       chunkDualTwo,
			},
		},
	}

	if !canReusePreviewForExport(preview, request) {
		t.Fatalf("expected completed preview job to be reusable for export")
	}
}

func TestCanReusePreviewForExportRejectsMissingDualChunk(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	pdfPath := filepath.Join(tempDir, "demo.pdf")
	request := StartRequest{
		PDFPath:    pdfPath,
		PageCount:  25,
		SourceLang: "en",
		TargetLang: "zh-CN",
		Mode:       ModeExport,
		Provider: ProviderConfig{
			ProviderID: 9,
			ModelID:    "gpt-4.1",
		},
	}
	preview := JobSnapshot{
		Mode:       ModePreview,
		Status:     JobStatusCompleted,
		PDFPath:    pdfPath,
		PageCount:  25,
		SourceLang: "en",
		TargetLang: "zh-CN",
		ProviderID: 9,
		ModelID:    "gpt-4.1",
		Chunks: []ChunkStatus{
			{
				Index:             1,
				Status:            JobStatusCompleted,
				TranslatedPDFPath: filepath.Join(tempDir, "chunk-001.mono.pdf"),
			},
		},
	}

	if canReusePreviewForExport(preview, request) {
		t.Fatalf("expected preview job without dual chunk output to be rejected")
	}
}

func TestSharedWorkerHomeDirStableForSameDocument(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	pdfPath := filepath.Join(tempDir, "demo.pdf")
	manager := &Manager{dataRootDir: filepath.Join(tempDir, "translator-data")}
	previewPath := manager.sharedWorkerHomeDir(StartRequest{
		PDFPath:    pdfPath,
		SourceLang: "en",
		TargetLang: "zh-CN",
		Mode:       ModePreview,
	})
	exportPath := manager.sharedWorkerHomeDir(StartRequest{
		PDFPath:    strings.ToUpper(pdfPath),
		SourceLang: "EN",
		TargetLang: "zh-CN",
		Mode:       ModeExport,
	})

	if previewPath != exportPath {
		t.Fatalf("shared worker home mismatch: preview=%q export=%q", previewPath, exportPath)
	}
}
