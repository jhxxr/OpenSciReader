package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"OpenSciReader/internal/translator"

	"github.com/gorilla/websocket"
)

var pdfTranslateWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool { return true },
}

func registerPDFTranslateHandlers(mux *http.ServeMux, app *App) {
	mux.HandleFunc("/api/pdf-translate/start", func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			rw.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if app == nil || app.translator == nil || app.store == nil {
			writeJSONError(rw, http.StatusServiceUnavailable, "pdf translate service is unavailable")
			return
		}

		var payload PDFTranslateStartInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeJSONError(rw, http.StatusBadRequest, fmt.Sprintf("decode start payload: %v", err))
			return
		}

		snapshot, err := app.startPDFTranslate(req.Context(), payload)
		if err != nil {
			writeJSONError(rw, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(rw, http.StatusOK, snapshot)
	})
	mux.HandleFunc("/api/pdf-translate/jobs", func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			rw.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if app == nil || app.translator == nil {
			writeJSONError(rw, http.StatusServiceUnavailable, "pdf translate service is unavailable")
			return
		}
		jobs, err := app.translator.ListJobs()
		if err != nil {
			writeJSONError(rw, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(rw, http.StatusOK, jobs)
	})

	mux.HandleFunc("/api/pdf-translate/", func(rw http.ResponseWriter, req *http.Request) {
		if app == nil || app.translator == nil {
			writeJSONError(rw, http.StatusServiceUnavailable, "pdf translate service is unavailable")
			return
		}

		trimmed := strings.TrimPrefix(req.URL.Path, "/api/pdf-translate/")
		parts := strings.Split(trimmed, "/")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			http.NotFound(rw, req)
			return
		}

		jobID := strings.TrimSpace(parts[0])
		action := strings.TrimSpace(parts[1])
		switch action {
		case "cancel":
			if req.Method != http.MethodPost {
				rw.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			snapshot, err := app.translator.Cancel(jobID)
			if err != nil {
				writeJSONError(rw, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(rw, http.StatusOK, snapshot)
		case "delete":
			if req.Method != http.MethodPost && req.Method != http.MethodDelete {
				rw.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if err := app.translator.DeleteJob(jobID); err != nil {
				statusCode := http.StatusBadRequest
				if strings.Contains(err.Error(), "not found") {
					statusCode = http.StatusNotFound
				}
				writeJSONError(rw, statusCode, err.Error())
				return
			}
			writeJSON(rw, http.StatusOK, map[string]string{"jobId": jobID, "status": "deleted"})
		case "status":
			if req.Method != http.MethodGet {
				rw.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			snapshot, err := app.translator.GetStatus(jobID)
			if err != nil {
				writeJSONError(rw, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(rw, http.StatusOK, snapshot)
		case "events":
			handlePDFTranslateEventsWS(rw, req, app, jobID)
		default:
			http.NotFound(rw, req)
		}
	})
}

func handlePDFTranslateEventsWS(rw http.ResponseWriter, req *http.Request, app *App, jobID string) {
	if req.Method != http.MethodGet {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot, events, unsubscribe, err := app.translator.Subscribe(jobID)
	if err != nil {
		writeJSONError(rw, http.StatusNotFound, err.Error())
		return
	}

	conn, err := pdfTranslateWSUpgrader.Upgrade(rw, req, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	defer unsubscribe()

	if err := conn.WriteJSON(translator.JobEvent{
		Type:      "status_snapshot",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		JobID:     snapshot.JobID,
		Mode:      snapshot.Mode,
		JobStatus: snapshot.Status,
		Status:    &snapshot,
	}); err != nil {
		return
	}

	for event := range events {
		if err := conn.WriteJSON(event); err != nil {
			return
		}
	}
}

func writeJSON(rw http.ResponseWriter, statusCode int, payload any) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(statusCode)
	_ = json.NewEncoder(rw).Encode(payload)
}

func writeJSONError(rw http.ResponseWriter, statusCode int, message string) {
	writeJSON(rw, statusCode, map[string]string{"error": message})
}
