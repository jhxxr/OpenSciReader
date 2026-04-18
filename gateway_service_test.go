package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeepLXTranslateEndpoint(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		rawURL   string
		wantURL  string
		wantMode deepLXEndpointMode
		wantErr  bool
	}{
		{
			name:     "free endpoint",
			rawURL:   "https://example.com/translate",
			wantURL:  "https://example.com/translate",
			wantMode: deepLXEndpointFree,
		},
		{
			name:     "pro endpoint trims trailing slash",
			rawURL:   "https://example.com/v1/translate/",
			wantURL:  "https://example.com/v1/translate",
			wantMode: deepLXEndpointPro,
		},
		{
			name:     "official endpoint keeps query",
			rawURL:   "https://example.com/api/v2/translate?foo=bar",
			wantURL:  "https://example.com/api/v2/translate?foo=bar",
			wantMode: deepLXEndpointOfficial,
		},
		{
			name:    "rejects root url",
			rawURL:  "https://example.com",
			wantErr: true,
		},
		{
			name:    "rejects missing scheme",
			rawURL:  "example.com/v1/translate",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			endpoint, err := deepLXTranslateEndpoint(tc.rawURL)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.rawURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("deepLXTranslateEndpoint(%q) returned error: %v", tc.rawURL, err)
			}
			if endpoint.URL != tc.wantURL {
				t.Fatalf("endpoint URL = %q, want %q", endpoint.URL, tc.wantURL)
			}
			if endpoint.Mode != tc.wantMode {
				t.Fatalf("endpoint mode = %q, want %q", endpoint.Mode, tc.wantMode)
			}
		})
	}
}

func TestIsDeepLXProviderRecognizesVersionedEndpoints(t *testing.T) {
	t.Parallel()

	provider := providerSecretRecord{
		ProviderRecord: ProviderRecord{
			Name:    "Custom Translator",
			BaseURL: "https://example.com/v2/translate",
		},
	}

	if !isDeepLXProvider(provider) {
		t.Fatalf("expected versioned DeeplX endpoint to be recognized")
	}
}

func TestProxyDeepLXBearerEndpoint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/translate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("authorization header = %q", got)
		}

		var payload struct {
			Text       string `json:"text"`
			SourceLang string `json:"source_lang"`
			TargetLang string `json:"target_lang"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.Text != "hello" || payload.SourceLang != "EN" || payload.TargetLang != "ZH" {
			t.Fatalf("unexpected payload: %+v", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":"你好"}`))
	}))
	defer server.Close()

	gateway := &gatewayService{client: server.Client()}
	translated, err := gateway.proxyDeepLX(context.Background(), providerSecretRecord{
		ProviderRecord: ProviderRecord{
			Name:    "DeepLX",
			BaseURL: server.URL + "/v1/translate",
		},
		APIKey: "sk-test",
	}, "hello", "en", "zh")
	if err != nil {
		t.Fatalf("proxyDeepLX returned error: %v", err)
	}
	if translated != "你好" {
		t.Fatalf("translated text = %q, want %q", translated, "你好")
	}
}

func TestProxyDeepLXFallsBackToQueryToken(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path != "/translate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		if requestCount == 1 {
			if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
				t.Fatalf("authorization header = %q", got)
			}
			http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if got := r.URL.Query().Get("token"); got != "sk-test" {
			t.Fatalf("token query = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("authorization header should be empty on query-token retry, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":"你好"}`))
	}))
	defer server.Close()

	gateway := &gatewayService{client: server.Client()}
	translated, err := gateway.proxyDeepLX(context.Background(), providerSecretRecord{
		ProviderRecord: ProviderRecord{
			Name:    "DeepLX",
			BaseURL: server.URL + "/translate",
		},
		APIKey: "sk-test",
	}, "hello", "en", "zh")
	if err != nil {
		t.Fatalf("proxyDeepLX returned error: %v", err)
	}
	if translated != "你好" {
		t.Fatalf("translated text = %q, want %q", translated, "你好")
	}
}

func TestProxyDeepLXOfficialEndpoint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/translate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "DeepL-Auth-Key sk-test" {
			t.Fatalf("authorization header = %q", got)
		}

		var payload struct {
			Text       []string `json:"text"`
			SourceLang string   `json:"source_lang"`
			TargetLang string   `json:"target_lang"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if len(payload.Text) != 1 || payload.Text[0] != "hello" {
			t.Fatalf("unexpected text payload: %+v", payload.Text)
		}
		if payload.SourceLang != "EN" || payload.TargetLang != "ZH" {
			t.Fatalf("unexpected payload: %+v", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"translations":[{"text":"你好"}]}`))
	}))
	defer server.Close()

	gateway := &gatewayService{client: server.Client()}
	translated, err := gateway.proxyDeepLX(context.Background(), providerSecretRecord{
		ProviderRecord: ProviderRecord{
			Name:    "Custom Translator",
			BaseURL: server.URL + "/v2/translate",
		},
		APIKey: "sk-test",
	}, "hello", "en", "zh")
	if err != nil {
		t.Fatalf("proxyDeepLX returned error: %v", err)
	}
	if translated != "你好" {
		t.Fatalf("translated text = %q, want %q", translated, "你好")
	}
}
