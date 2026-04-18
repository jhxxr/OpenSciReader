package main

import (
	"fmt"
	"strings"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) SelectPDFTranslateRuntimePackage() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用上下文不可用")
	}

	selectedPath, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "选择 PDF 翻译运行时安装包",
		Filters: []wruntime.FileFilter{
			{
				DisplayName: "ZIP 压缩包 (*.zip)",
				Pattern:     "*.zip",
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("打开运行时安装包选择窗口失败: %w", err)
	}

	return strings.TrimSpace(selectedPath), nil
}
