package translator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type workerRequest struct {
	JobID                     string         `json:"jobId"`
	Mode                      Mode           `json:"mode"`
	InputPDFPath              string         `json:"inputPdfPath"`
	WorkingDir                string         `json:"workingDir"`
	OutputDir                 string         `json:"outputDir"`
	WorkerHomeDir             string         `json:"workerHomeDir,omitempty"`
	SourceLang                string         `json:"sourceLang"`
	TargetLang                string         `json:"targetLang"`
	Pages                     string         `json:"pages,omitempty"`
	OnlyIncludeTranslatedPage bool           `json:"onlyIncludeTranslatedPage"`
	NoDual                    bool           `json:"noDual"`
	NoMono                    bool           `json:"noMono"`
	MaxPagesPerPart           int            `json:"maxPagesPerPart"`
	ReportInterval            float64        `json:"reportInterval"`
	QPS                       int            `json:"qps"`
	PoolMaxWorkers            int            `json:"poolMaxWorkers"`
	TermPoolMaxWorkers        int            `json:"termPoolMaxWorkers"`
	MinTextLength             int            `json:"minTextLength"`
	WatermarkOutputMode       string         `json:"watermarkOutputMode"`
	MergeMonoPDFPaths         []string       `json:"mergeMonoPdfPaths,omitempty"`
	MergeDualPDFPaths         []string       `json:"mergeDualPdfPaths,omitempty"`
	Provider                  ProviderConfig `json:"provider"`
}

type workerTranslateResult struct {
	OriginalPDFPath           string  `json:"original_pdf_path"`
	MonoPDFPath               string  `json:"mono_pdf_path"`
	DualPDFPath               string  `json:"dual_pdf_path"`
	NoWatermarkMonoPDFPath    string  `json:"no_watermark_mono_pdf_path"`
	NoWatermarkDualPDFPath    string  `json:"no_watermark_dual_pdf_path"`
	AutoExtractedGlossaryPath string  `json:"auto_extracted_glossary_path"`
	TotalSeconds              float64 `json:"total_seconds"`
	PeakMemoryUsage           float64 `json:"peak_memory_usage"`
}

type workerEvent struct {
	Type            string                `json:"type"`
	Stage           string                `json:"stage,omitempty"`
	StageProgress   float64               `json:"stage_progress,omitempty"`
	OverallProgress float64               `json:"overall_progress,omitempty"`
	StageCurrent    int                   `json:"stage_current,omitempty"`
	StageTotal      int                   `json:"stage_total,omitempty"`
	PartIndex       int                   `json:"part_index,omitempty"`
	TotalParts      int                   `json:"total_parts,omitempty"`
	Stages          []StageSummaryItem    `json:"stages,omitempty"`
	Error           string                `json:"error,omitempty"`
	ErrorType       string                `json:"error_type,omitempty"`
	Details         string                `json:"details,omitempty"`
	TranslateResult workerTranslateResult `json:"translate_result,omitempty"`
}

type pythonCommand struct {
	Command string
	Args    []string
}

type workerProcessError struct {
	message string
}

func (e workerProcessError) Error() string {
	return e.message
}

func (m *Manager) resolvePythonCommand() (pythonCommand, error) {
	searchPaths := make([]string, 0, len(m.pythonSearchPaths)+2)
	searchPaths = append(searchPaths, m.pythonSearchPaths...)
	searchPaths = append(searchPaths,
		filepath.Join(m.runtimeDir, "runtime", "python.exe"),
		filepath.Join(m.runtimeDir, "python", "python.exe"),
		filepath.Join(m.runtimeDir, "python.exe"),
	)

	for _, candidate := range searchPaths {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return pythonCommand{Command: candidate}, nil
		}
	}

	if path, err := exec.LookPath("python"); err == nil {
		return pythonCommand{Command: path}, nil
	}
	if path, err := exec.LookPath("py"); err == nil {
		return pythonCommand{Command: path, Args: []string{"-3"}}, nil
	}

	return pythonCommand{}, fmt.Errorf("python runtime not found")
}

func (m *Manager) runWorker(ctx context.Context, request workerRequest, onEvent func(workerEvent) error) error {
	pythonCmd, err := m.resolvePythonCommand()
	if err != nil {
		return err
	}
	workerScript := m.workerScriptPath
	if workerScript == "" {
		return fmt.Errorf("worker script path is empty")
	}
	info, err := os.Stat(workerScript)
	if err != nil {
		return fmt.Errorf("stat worker script: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("worker script path points to a directory: %s", workerScript)
	}

	args := append([]string{}, pythonCmd.Args...)
	args = append(args, workerScript)

	cmd := exec.CommandContext(ctx, pythonCmd.Command, args...)
	configureWorkerProcess(cmd)
	cmd.Dir = request.WorkingDir
	workerHomeDir := strings.TrimSpace(request.WorkerHomeDir)
	if workerHomeDir == "" {
		workerHomeDir = filepath.Join(request.WorkingDir, "_worker_home")
	}
	cacheDirs := []string{
		workerHomeDir,
		filepath.Join(workerHomeDir, ".cache"),
		filepath.Join(workerHomeDir, ".cache", "babeldoc"),
		filepath.Join(workerHomeDir, "AppData", "Roaming"),
		filepath.Join(workerHomeDir, "AppData", "Local"),
	}
	for _, directory := range cacheDirs {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return fmt.Errorf("prepare worker cache directory %s: %w", directory, err)
		}
	}
	cmd.Env = append(os.Environ(),
		"PYTHONUTF8=1",
		fmt.Sprintf("PYTHONPATH=%s", filepath.Join(m.runtimeDir, "site-packages")),
		fmt.Sprintf("OPENSCIREADER_PDF2ZH_RUNTIME_DIR=%s", m.runtimeDir),
		fmt.Sprintf("HOME=%s", workerHomeDir),
		fmt.Sprintf("USERPROFILE=%s", workerHomeDir),
		fmt.Sprintf("APPDATA=%s", filepath.Join(workerHomeDir, "AppData", "Roaming")),
		fmt.Sprintf("LOCALAPPDATA=%s", filepath.Join(workerHomeDir, "AppData", "Local")),
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open worker stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open worker stdout: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start worker: %w", err)
	}

	go func() {
		defer stdin.Close()
		encoder := json.NewEncoder(stdin)
		_ = encoder.Encode(request)
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var workerErr error
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var event workerEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return fmt.Errorf("decode worker event: %w; line=%s", err, line)
		}
		if event.Type == "error" && workerErr == nil {
			workerErr = workerProcessError{message: formatWorkerEventError(event)}
		}
		if err := onEvent(event); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("read worker stdout: %w", err)
	}

	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if workerErr != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return fmt.Errorf("%w: %s", workerErr, stderrText)
		}
		return workerErr
	}
	if waitErr != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return fmt.Errorf("worker exit error: %w: %s", waitErr, stderrText)
		}
		return fmt.Errorf("worker exit error: %w", waitErr)
	}
	return nil
}

func formatWorkerEventError(event workerEvent) string {
	parts := make([]string, 0, 2)
	if event.ErrorType != "" {
		parts = append(parts, event.ErrorType)
	}
	if message := compactWorkerErrorMessage(event.Error); message != "" {
		parts = append(parts, message)
	}
	if len(parts) > 0 {
		return strings.Join(parts, ": ")
	}
	return "worker returned an unknown error"
}

func compactWorkerErrorMessage(raw string) string {
	text := strings.TrimSpace(strings.ReplaceAll(raw, "\r\n", "\n"))
	if text == "" {
		return ""
	}

	line := text
	for _, marker := range []string{
		"\n",
		"Traceback:",
		"Subprocess traceback:",
		"Received error from subprocess:",
	} {
		if idx := strings.Index(line, marker); idx >= 0 {
			line = line[:idx]
		}
	}

	const inputFilesWarning = "settings.basic.input_files is for cli"
	if idx := strings.Index(line, inputFilesWarning); idx >= 0 {
		line = line[:idx]
	}

	return strings.TrimSpace(strings.TrimRight(line, " :"))
}
