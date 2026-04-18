package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type pdfService struct {
	client *http.Client
}

func newPDFService() *pdfService {
	return &pdfService{
		client: &http.Client{Timeout: 90 * time.Second},
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
