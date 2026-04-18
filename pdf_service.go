package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	pdfMarkdownExtractorName    = "markitdown"
	pdfMarkdownExtractorVersion = "v1"
)

type pdfService struct {
	client *http.Client
	store  *configStore
}

func newPDFService(store *configStore) *pdfService {
	return &pdfService{
		client: &http.Client{Timeout: 90 * time.Second},
		store:  store,
	}
}

func (s *pdfService) LoadDocument(ctx context.Context, rawPath string) (PDFDocumentPayload, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return PDFDocumentPayload{}, fmt.Errorf("pdf path is required")
	}

	normalized := normalizeAttachmentPath(path)
	switch {
	case strings.HasPrefix(strings.ToLower(normalized), "http://"), strings.HasPrefix(strings.ToLower(normalized), "https://"):
		return s.loadRemoteDocument(ctx, normalized)
	default:
		return s.loadLocalDocument(normalized)
	}
}

func (s *pdfService) ExtractMarkdown(ctx context.Context, rawPath string) (PDFMarkdownPayload, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return PDFMarkdownPayload{}, fmt.Errorf("pdf path is required")
	}

	normalized := normalizeAttachmentPath(path)
	if !filepath.IsAbs(normalized) {
		return PDFMarkdownPayload{}, fmt.Errorf("pdf path must be absolute: %s", normalized)
	}
	if !strings.EqualFold(filepath.Ext(normalized), ".pdf") {
		return PDFMarkdownPayload{}, fmt.Errorf("only pdf files are supported: %s", normalized)
	}

	data, err := os.ReadFile(normalized)
	if err != nil {
		return PDFMarkdownPayload{}, fmt.Errorf("read pdf file: %w", err)
	}

	pdfHash := sha256Hex(data)
	if s.store != nil {
		cached, found, err := s.store.GetPDFMarkdownCache(ctx, pdfHash, pdfMarkdownExtractorName, pdfMarkdownExtractorVersion)
		if err == nil && found {
			cached.PDFPath = normalized
			cached.Cached = true
			return cached, nil
		}
	}

	payload, err := extractPDFMarkdownWithPython(ctx, normalized)
	if err != nil {
		return PDFMarkdownPayload{}, err
	}
	payload.PDFPath = normalized
	payload.Source = pdfMarkdownExtractorName
	payload.TotalChars = len(payload.Markdown)
	if strings.TrimSpace(payload.GeneratedAt) == "" {
		payload.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}

	if s.store == nil {
		return payload, nil
	}
	saved, err := s.store.SavePDFMarkdownCache(ctx, pdfMarkdownCacheRecord{
		PDFHash:          pdfHash,
		Extractor:        pdfMarkdownExtractorName,
		ExtractorVersion: pdfMarkdownExtractorVersion,
		Payload:          payload,
	})
	if err != nil {
		return payload, nil
	}
	saved.PDFPath = normalized
	return saved, nil
}

func (s *pdfService) loadLocalDocument(path string) (PDFDocumentPayload, error) {
	if !filepath.IsAbs(path) {
		return PDFDocumentPayload{}, fmt.Errorf("pdf path must be absolute: %s", path)
	}
	if !strings.EqualFold(filepath.Ext(path), ".pdf") {
		return PDFDocumentPayload{}, fmt.Errorf("only pdf files are supported: %s", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return PDFDocumentPayload{}, fmt.Errorf("stat pdf file: %w", err)
	}
	if info.IsDir() {
		return PDFDocumentPayload{}, fmt.Errorf("path points to a directory: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return PDFDocumentPayload{}, fmt.Errorf("read pdf file: %w", err)
	}

	return PDFDocumentPayload{
		Path:       path,
		DataBase64: base64.StdEncoding.EncodeToString(data),
	}, nil
}

func (s *pdfService) loadRemoteDocument(ctx context.Context, remoteURL string) (PDFDocumentPayload, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return PDFDocumentPayload{}, fmt.Errorf("build remote pdf request: %w", err)
	}
	req.Header.Set("User-Agent", "OpenSciReader/1.0")
	req.Header.Set("Accept", "application/pdf,*/*")

	resp, err := s.client.Do(req)
	if err != nil {
		return PDFDocumentPayload{}, fmt.Errorf("fetch remote pdf: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return PDFDocumentPayload{}, fmt.Errorf("remote pdf http error: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return PDFDocumentPayload{}, fmt.Errorf("read remote pdf: %w", err)
	}

	return PDFDocumentPayload{
		Path:       remoteURL,
		DataBase64: base64.StdEncoding.EncodeToString(data),
	}, nil
}

type markItDownScriptResult struct {
	Markdown string `json:"markdown"`
}

func extractPDFMarkdownWithPython(ctx context.Context, pdfPath string) (PDFMarkdownPayload, error) {
	pythonCommand, pythonArgs, err := resolvePythonForPDFMarkdown()
	if err != nil {
		return PDFMarkdownPayload{}, err
	}

	script := strings.TrimSpace(`
import json
import sys
from markitdown import MarkItDown

pdf_path = sys.argv[1]
result = MarkItDown().convert(pdf_path)
text = getattr(result, "text_content", "") or ""
print(json.dumps({"markdown": text}, ensure_ascii=False))
`)
	args := append([]string{}, pythonArgs...)
	args = append(args, "-c", script, pdfPath)

	cmd := exec.CommandContext(ctx, pythonCommand, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return PDFMarkdownPayload{}, fmt.Errorf("run markitdown: %s", message)
	}

	var result markItDownScriptResult
	if err := json.Unmarshal(output, &result); err != nil {
		return PDFMarkdownPayload{}, fmt.Errorf("decode markitdown output: %w", err)
	}
	markdown := strings.TrimSpace(result.Markdown)
	if markdown == "" {
		return PDFMarkdownPayload{}, fmt.Errorf("markitdown returned empty markdown")
	}

	sections := splitMarkdownSections(markdown)
	return PDFMarkdownPayload{
		Source:      pdfMarkdownExtractorName,
		Markdown:    markdown,
		Sections:    sections,
		TotalChars:  len(markdown),
		Cached:      false,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func resolvePythonForPDFMarkdown() (string, []string, error) {
	if path, err := exec.LookPath("python"); err == nil {
		return path, nil, nil
	}
	if path, err := exec.LookPath("py"); err == nil {
		return path, []string{"-3"}, nil
	}
	return "", nil, fmt.Errorf("python runtime not found")
}

func splitMarkdownSections(markdown string) []PDFMarkdownSection {
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		return nil
	}

	lines := strings.Split(trimmed, "\n")
	sections := make([]PDFMarkdownSection, 0)
	currentTitle := "文档开始"
	currentLevel := 0
	currentLines := make([]string, 0)
	index := 0
	flush := func() {
		text := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if text == "" {
			currentLines = currentLines[:0]
			return
		}
		index += 1
		sections = append(sections, PDFMarkdownSection{
			Index:      index,
			Title:      currentTitle,
			Level:      currentLevel,
			StartPage:  0,
			EndPage:    0,
			Text:       text,
			Characters: len(text),
		})
		currentLines = currentLines[:0]
	}

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		level, title, ok := parseMarkdownHeading(trimmedLine)
		if ok {
			flush()
			currentTitle = title
			currentLevel = level
			currentLines = append(currentLines, trimmedLine)
			continue
		}
		currentLines = append(currentLines, line)
	}
	flush()
	return sections
}

func parseMarkdownHeading(line string) (int, string, bool) {
	if !strings.HasPrefix(line, "#") {
		return 0, "", false
	}
	level := 0
	for level < len(line) && line[level] == '#' {
		level += 1
	}
	if level == 0 || level > 6 || len(line) <= level || line[level] != ' ' {
		return 0, "", false
	}
	title := strings.TrimSpace(line[level:])
	if title == "" {
		return 0, "", false
	}
	return level, title, true
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
