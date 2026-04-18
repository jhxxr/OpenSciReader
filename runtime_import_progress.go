package main

import (
	"context"
	"sync"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const pdfTranslateRuntimeImportProgressEvent = "pdf-translate-runtime:import-progress"

type PDFTranslateRuntimeImportProgress struct {
	Stage          string  `json:"stage"`
	Message        string  `json:"message"`
	Progress       float64 `json:"progress"`
	BytesCompleted int64   `json:"bytesCompleted"`
	BytesTotal     int64   `json:"bytesTotal"`
}

type runtimeImportProgressEmitter struct {
	ctx          context.Context
	mu           sync.Mutex
	lastProgress float64
}

func newRuntimeImportProgressEmitter(ctx context.Context) *runtimeImportProgressEmitter {
	return &runtimeImportProgressEmitter{ctx: ctx}
}

func (e *runtimeImportProgressEmitter) Emit(stage, message string, progress float64, bytesCompleted, bytesTotal int64) {
	if e == nil || e.ctx == nil {
		return
	}

	progress = clampRuntimeImportProgress(progress)
	if bytesCompleted < 0 {
		bytesCompleted = 0
	}
	if bytesTotal < 0 {
		bytesTotal = 0
	}

	e.mu.Lock()
	e.lastProgress = progress
	e.mu.Unlock()

	wruntime.EventsEmit(e.ctx, pdfTranslateRuntimeImportProgressEvent, PDFTranslateRuntimeImportProgress{
		Stage:          stage,
		Message:        message,
		Progress:       progress,
		BytesCompleted: bytesCompleted,
		BytesTotal:     bytesTotal,
	})
}

func (e *runtimeImportProgressEmitter) Fail(message string) {
	if e == nil {
		return
	}

	e.mu.Lock()
	progress := e.lastProgress
	e.mu.Unlock()

	e.Emit("failed", message, progress, 0, 0)
}

func clampRuntimeImportProgress(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}
