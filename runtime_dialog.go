package main

import (
	"fmt"
	"strings"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) SelectPDFTranslateRuntimePackage() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("application context is unavailable")
	}

	selectedPath, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Select PDF translation runtime package",
		Filters: []wruntime.FileFilter{
			{
				DisplayName: "ZIP Archives (*.zip)",
				Pattern:     "*.zip",
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("open runtime package dialog: %w", err)
	}

	return strings.TrimSpace(selectedPath), nil
}
