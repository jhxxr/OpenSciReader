package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func newLocalAssetHandler(app *App) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/osr/file", serveLocalPDFFile)
	mux.HandleFunc("/osr/remote-pdf", serveRemotePDFFile)
	registerPDFTranslateHandlers(mux, app)
	return mux
}

func serveLocalPDFFile(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	rawPath := strings.TrimSpace(req.URL.Query().Get("path"))
	if rawPath == "" {
		http.Error(rw, "missing path", http.StatusBadRequest)
		return
	}

	cleanPath := normalizeLocalPDFRequestPath(rawPath)
	if cleanPath == "" {
		http.Error(rw, "missing path", http.StatusBadRequest)
		return
	}
	if !filepath.IsAbs(cleanPath) {
		http.Error(rw, "path must be absolute", http.StatusBadRequest)
		return
	}
	if strings.ToLower(filepath.Ext(cleanPath)) != ".pdf" {
		http.Error(rw, "only pdf files are supported", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(rw, "file not found", http.StatusNotFound)
			return
		}
		http.Error(rw, fmt.Sprintf("stat file: %v", err), http.StatusInternalServerError)
		return
	}
	if info.IsDir() {
		http.Error(rw, "path points to a directory", http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/pdf")
	http.ServeFile(rw, req, cleanPath)
}

func serveRemotePDFFile(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	remoteURL := strings.TrimSpace(req.URL.Query().Get("url"))
	if remoteURL == "" {
		http.Error(rw, "missing url", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(strings.ToLower(remoteURL), "http://") && !strings.HasPrefix(strings.ToLower(remoteURL), "https://") {
		http.Error(rw, "only http/https urls are supported", http.StatusBadRequest)
		return
	}

	proxyReq, err := http.NewRequestWithContext(req.Context(), http.MethodGet, remoteURL, nil)
	if err != nil {
		http.Error(rw, fmt.Sprintf("build remote request: %v", err), http.StatusBadRequest)
		return
	}
	proxyReq.Header.Set("User-Agent", "OpenSciReader/1.0")
	proxyReq.Header.Set("Accept", "application/pdf,*/*")

	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		http.Error(rw, fmt.Sprintf("fetch remote pdf: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		http.Error(rw, fmt.Sprintf("remote pdf http error: %s", resp.Status), http.StatusBadGateway)
		return
	}

	rw.Header().Set("Content-Type", "application/pdf")
	if _, err := io.Copy(rw, resp.Body); err != nil {
		http.Error(rw, fmt.Sprintf("stream remote pdf: %v", err), http.StatusBadGateway)
	}
}

func normalizeLocalPDFRequestPath(rawPath string) string {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return ""
	}
	return normalizeAttachmentPath(rawPath)
}
