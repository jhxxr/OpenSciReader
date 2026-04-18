package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchProviderModelsAddsBrowserLikeUserAgent(t *testing.T) {
	t.Parallel()

	store := newGatewayTestConfigStore(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("User-Agent"); got != browserLikeUserAgent {
			t.Fatalf("user-agent = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("authorization header = %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gemma4:31b","name":"Gemma 4 31B","owned_by":"custom"}]}`))
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

	service := &gatewayService{store: store, client: server.Client()}
	response, err := service.FetchProviderModels(context.Background(), provider.ID)
	if err != nil {
		t.Fatalf("FetchProviderModels returned error: %v", err)
	}
	if response.Total != 1 || len(response.Models) != 1 || response.Models[0].ID != "gemma4:31b" {
		t.Fatalf("unexpected models response: %+v", response)
	}
}

func TestProxyLLMTranslationAddsBrowserLikeUserAgent(t *testing.T) {
	t.Parallel()

	store := newGatewayTestConfigStore(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("User-Agent"); got != browserLikeUserAgent {
			t.Fatalf("user-agent = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("authorization header = %q", got)
		}

		var payload openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.Model != "gemma4:31b" {
			t.Fatalf("model = %q", payload.Model)
		}
		if len(payload.Messages) != 1 {
			t.Fatalf("messages = %+v", payload.Messages)
		}
		content, _ := payload.Messages[0]["content"].(string)
		if !strings.Contains(content, "Translate the following text from en to zh-CN") {
			t.Fatalf("unexpected prompt: %q", content)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"你好"}}]}`))
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
	translated, err := service.proxyLLMTranslation(context.Background(), provider.ID, model.ID, "hello", "en", "zh-CN")
	if err != nil {
		t.Fatalf("proxyLLMTranslation returned error: %v", err)
	}
	if translated != "你好" {
		t.Fatalf("translated text = %q", translated)
	}
}

func newGatewayTestConfigStore(t *testing.T) *configStore {
	t.Helper()

	tempDir := t.TempDir()
	store, err := newConfigStore(appPaths{
		RootDir:           tempDir,
		AppConfigDBPath:   tempDir + "/app.sqlite",
		OCRCacheDBPath:    tempDir + "/ocr.sqlite",
		EncryptionKeyPath: tempDir + "/config.key",
	})
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close config store: %v", err)
		}
	})
	return store
}
