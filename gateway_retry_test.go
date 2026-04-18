package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyLLMTranslationRetriesOn429(t *testing.T) {
	store := newGatewayTestConfigStore(t)
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if requestCount < 3 {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"slow down"}`, http.StatusTooManyRequests)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"浣犲ソ"}}]}`))
	}))
	defer server.Close()

	provider, err := store.SaveProvider(context.Background(), ProviderUpsertInput{
		Name:     "OpenAI Compat",
		Type:     ProviderTypeLLM,
		BaseURL:  server.URL,
		APIKey:   "sk-test",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("save provider: %v", err)
	}
	model, err := store.SaveModel(context.Background(), ModelUpsertInput{
		ProviderID: provider.ID,
		ModelID:    "gemma4:31b",
	})
	if err != nil {
		t.Fatalf("save model: %v", err)
	}

	service := &gatewayService{store: store, client: server.Client()}
	translated, err := service.proxyLLMTranslation(context.Background(), provider.ID, model.ID, "hello", "en", "zh")
	if err != nil {
		t.Fatalf("proxyLLMTranslation returned error: %v", err)
	}
	if translated != "浣犲ソ" {
		t.Fatalf("translated text = %q, want %q", translated, "浣犲ソ")
	}
	if requestCount != 3 {
		t.Fatalf("request count = %d, want 3", requestCount)
	}
}

func TestGenerateGeminiFigureRetriesOn429(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"slow down"}`, http.StatusTooManyRequests)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"inline_data":{"mime_type":"image/png","data":"QUJD"}}]}}]}`))
	}))
	defer server.Close()

	service := &gatewayService{client: server.Client()}
	result, err := service.generateGeminiFigure(context.Background(), providerSecretRecord{
		ProviderRecord: ProviderRecord{
			Name:    "Gemini",
			BaseURL: server.URL,
		},
		APIKey: "sk-test",
	}, ModelRecord{ModelID: "gemini-test"}, "draw", GatewayContextData{})
	if err != nil {
		t.Fatalf("generateGeminiFigure returned error: %v", err)
	}
	if result.DataURL != "data:image/png;base64,QUJD" {
		t.Fatalf("data url = %q", result.DataURL)
	}
	if requestCount != 3 {
		t.Fatalf("request count = %d, want 3", requestCount)
	}
}
