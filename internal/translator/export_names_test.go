package translator

import (
	"path/filepath"
	"testing"
)

func TestExportOutputFileStemUsesItemTitleAndLanguages(t *testing.T) {
	t.Parallel()

	stem := exportOutputFileStem(StartRequest{
		ItemTitle:  "Adaptive BCI Review",
		PDFPath:    filepath.Join("papers", "demo.pdf"),
		SourceLang: "en",
		TargetLang: "zh-CN",
	})

	if stem != "Adaptive-BCI-Review.en-to-zh-CN" {
		t.Fatalf("export output stem = %q", stem)
	}
}

func TestExportOutputFileStemFallsBackToPDFBaseName(t *testing.T) {
	t.Parallel()

	stem := exportOutputFileStem(StartRequest{
		PDFPath:    filepath.Join("papers", "adaptive review?.pdf"),
		SourceLang: "EN",
		TargetLang: "ZH/CN",
	})

	if stem != "adaptive-review.EN-to-ZH-CN" {
		t.Fatalf("export output stem = %q", stem)
	}
}
