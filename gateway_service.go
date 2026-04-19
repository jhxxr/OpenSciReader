package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type GatewayContextData struct {
	Selection string `json:"selection"`
	Snapshot  string `json:"snapshot"`
	Page      int    `json:"page"`
	ItemTitle string `json:"itemTitle"`
	CiteKey   string `json:"citeKey"`
}

type gatewayService struct {
	store  *configStore
	client *http.Client
}

type gatewayStreamEvent struct {
	RequestID string `json:"requestId"`
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
}

type FigureGenerationResult struct {
	MimeType string `json:"mimeType"`
	DataURL  string `json:"dataUrl"`
	Prompt   string `json:"prompt"`
}

type openAIChatRequest struct {
	Model    string           `json:"model"`
	Messages []map[string]any `json:"messages"`
	Stream   bool             `json:"stream"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Text string `json:"text"`
	} `json:"choices"`
}

type deepLXEndpointMode string

const (
	deepLXEndpointFree          deepLXEndpointMode = "free"
	deepLXEndpointPro           deepLXEndpointMode = "pro"
	deepLXEndpointOfficial      deepLXEndpointMode = "official"
	gatewayHTTP429MaxRetryDelay                    = 5 * time.Second
	gatewayHTTP429RetryCount                       = 2
)

type deepLXEndpoint struct {
	URL  string
	Mode deepLXEndpointMode
}

var gatewayHTTP429RetryDelays = []time.Duration{
	1 * time.Second,
	2 * time.Second,
}

var gatewaySleepWithContext = func(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func newGatewayService(store *configStore) *gatewayService {
	return &gatewayService{
		store: store,
		client: &http.Client{
			Timeout: 0,
		},
	}
}

func (g *gatewayService) doRequestWith429Retry(ctx context.Context, buildRequest func() (*http.Request, error)) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		req, err := buildRequest()
		if err != nil {
			return nil, err
		}

		resp, err := g.client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests || attempt >= gatewayHTTP429RetryCount {
			return resp, nil
		}

		delay := gatewayHTTP429RetryDelay(resp, attempt)
		drainAndCloseResponse(resp)
		if err := gatewaySleepWithContext(ctx, delay); err != nil {
			return nil, err
		}
	}
}

func gatewayHTTP429RetryDelay(resp *http.Response, retryIndex int) time.Duration {
	if resp != nil {
		if delay, ok := parseRetryAfterDelay(resp.Header.Get("Retry-After")); ok && delay > 0 && delay <= gatewayHTTP429MaxRetryDelay {
			return delay
		}
	}
	if retryIndex >= 0 && retryIndex < len(gatewayHTTP429RetryDelays) {
		return gatewayHTTP429RetryDelays[retryIndex]
	}
	return time.Duration(retryIndex+1) * time.Second
}

func parseRetryAfterDelay(value string) (time.Duration, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(trimmed); err == nil {
		return time.Duration(seconds) * time.Second, true
	}
	retryAt, err := http.ParseTime(trimmed)
	if err != nil {
		return 0, false
	}
	delay := time.Until(retryAt)
	if delay <= 0 {
		return 0, false
	}
	return delay, true
}

func drainAndCloseResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()
}

func (g *gatewayService) StreamLLMChat(ctx context.Context, appCtx context.Context, providerID, modelID int64, prompt string, contextData GatewayContextData) (string, error) {
	provider, err := g.store.GetProviderSecret(ctx, providerID)
	if err != nil {
		return "", err
	}
	model, err := g.store.GetModel(ctx, modelID)
	if err != nil {
		return "", err
	}
	if model.ProviderID != provider.ID {
		return "", fmt.Errorf("model %d does not belong to provider %d", modelID, providerID)
	}
	if !provider.IsActive {
		return "", fmt.Errorf("provider %s is inactive", provider.Name)
	}
	if provider.APIKey == "" {
		return "", fmt.Errorf("provider %s has no api key", provider.Name)
	}

	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
	go g.streamOpenAICompatible(appCtx, gatewayEventName(requestID), provider, model, strings.TrimSpace(prompt), contextData, requestID)
	return requestID, nil
}

func (g *gatewayService) streamOpenAICompatible(appCtx context.Context, eventName string, provider providerSecretRecord, model ModelRecord, prompt string, contextData GatewayContextData, requestID string) {
	payload, err := json.Marshal(openAIChatRequest{
		Model:    model.ModelID,
		Messages: buildChatMessages(prompt, contextData),
		Stream:   true,
	})
	if err != nil {
		emitGatewayEvent(appCtx, eventName, gatewayStreamEvent{RequestID: requestID, Type: "error", Error: err.Error()})
		return
	}

	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		emitGatewayEvent(appCtx, eventName, gatewayStreamEvent{RequestID: requestID, Type: "error", Error: "provider base URL is empty"})
		return
	}

	resp, err := g.doRequestWith429Retry(appCtx, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(appCtx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(payload))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		applyProviderRequestHeaders(req, provider.APIKey)
		return req, nil
	})
	if err != nil {
		emitGatewayEvent(appCtx, eventName, gatewayStreamEvent{RequestID: requestID, Type: "error", Error: err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		emitGatewayEvent(appCtx, eventName, gatewayStreamEvent{RequestID: requestID, Type: "error", Error: fmt.Sprintf("gateway http error: %s %s", resp.Status, strings.TrimSpace(string(body)))})
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			emitGatewayEvent(appCtx, eventName, gatewayStreamEvent{RequestID: requestID, Type: "done"})
			return
		}

		chunk := extractStreamChunk(data)
		if chunk != "" {
			emitGatewayEvent(appCtx, eventName, gatewayStreamEvent{RequestID: requestID, Type: "chunk", Content: chunk})
		}
	}

	if err := scanner.Err(); err != nil {
		emitGatewayEvent(appCtx, eventName, gatewayStreamEvent{RequestID: requestID, Type: "error", Error: err.Error()})
		return
	}

	emitGatewayEvent(appCtx, eventName, gatewayStreamEvent{RequestID: requestID, Type: "done"})
}

func (g *gatewayService) GenerateWorkspaceKnowledgeBySource(ctx context.Context, providerID, modelID int64, prompt string) (WorkspaceKnowledgeBySourcePayload, error) {
	text, err := g.generateOpenAICompatibleText(ctx, providerID, modelID, []map[string]any{
		{
			"role":    "system",
			"content": "Return only valid JSON for the requested workspace knowledge payload.",
		},
		{
			"role":    "user",
			"content": strings.TrimSpace(prompt),
		},
	})
	if err != nil {
		return WorkspaceKnowledgeBySourcePayload{}, err
	}

	text = stripMarkdownCodeFence(text)
	var payload WorkspaceKnowledgeBySourcePayload
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return WorkspaceKnowledgeBySourcePayload{}, fmt.Errorf("decode workspace knowledge by-source payload: %w", err)
	}
	return payload, nil
}

func (g *gatewayService) GenerateWorkspaceKnowledgeMarkdown(ctx context.Context, providerID, modelID int64, prompt string) (string, error) {
	return g.generateOpenAICompatibleText(ctx, providerID, modelID, []map[string]any{
		{
			"role":    "system",
			"content": "Return only markdown.",
		},
		{
			"role":    "user",
			"content": strings.TrimSpace(prompt),
		},
	})
}

func (g *gatewayService) GenerateWorkspaceKnowledgeQuery(ctx context.Context, providerID, modelID int64, prompt string) (WorkspaceKnowledgeQueryResult, error) {
	text, err := g.generateOpenAICompatibleText(ctx, providerID, modelID, []map[string]any{
		{
			"role":    "system",
			"content": "Return only valid JSON for the requested workspace knowledge query payload.",
		},
		{
			"role":    "user",
			"content": strings.TrimSpace(prompt),
		},
	})
	if err != nil {
		return WorkspaceKnowledgeQueryResult{}, err
	}

	text = stripMarkdownCodeFence(text)
	var payload WorkspaceKnowledgeQueryResult
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return WorkspaceKnowledgeQueryResult{}, fmt.Errorf("decode workspace knowledge query payload: %w", err)
	}
	if payload.Candidates == nil {
		payload.Candidates = []WorkspaceKnowledgeCandidate{}
	}
	return payload, nil
}

func (g *gatewayService) generateOpenAICompatibleText(ctx context.Context, providerID, modelID int64, messages []map[string]any) (string, error) {
	provider, model, err := g.loadOpenAICompatibleProviderModel(ctx, providerID, modelID)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(openAIChatRequest{
		Model:    model.ModelID,
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return "", err
	}

	resp, err := g.doRequestWith429Retry(ctx, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(provider.BaseURL, "/")+"/chat/completions", bytes.NewReader(payload))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		applyProviderRequestHeaders(req, provider.APIKey)
		return req, nil
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("gateway http error: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return parseOpenAICompatibleResponseText(body)
}

func (g *gatewayService) loadOpenAICompatibleProviderModel(ctx context.Context, providerID, modelID int64) (providerSecretRecord, ModelRecord, error) {
	provider, err := g.store.GetProviderSecret(ctx, providerID)
	if err != nil {
		return providerSecretRecord{}, ModelRecord{}, err
	}
	model, err := g.store.GetModel(ctx, modelID)
	if err != nil {
		return providerSecretRecord{}, ModelRecord{}, err
	}
	if model.ProviderID != provider.ID {
		return providerSecretRecord{}, ModelRecord{}, fmt.Errorf("model %d does not belong to provider %d", modelID, providerID)
	}
	if !provider.IsActive {
		return providerSecretRecord{}, ModelRecord{}, fmt.Errorf("provider %s is inactive", provider.Name)
	}
	if provider.APIKey == "" {
		return providerSecretRecord{}, ModelRecord{}, fmt.Errorf("provider %s has no api key", provider.Name)
	}
	return provider, model, nil
}

func buildChatMessages(prompt string, contextData GatewayContextData) []map[string]any {
	parts := make([]string, 0, 5)
	if contextData.ItemTitle != "" {
		parts = append(parts, "Document: "+contextData.ItemTitle)
	}
	if contextData.CiteKey != "" {
		parts = append(parts, "CiteKey: "+contextData.CiteKey)
	}
	if contextData.Page > 0 {
		parts = append(parts, fmt.Sprintf("Page: %d", contextData.Page))
	}
	if contextData.Selection != "" {
		parts = append(parts, "Selected text:\n"+contextData.Selection)
	}

	content := []map[string]any{{
		"type": "text",
		"text": strings.TrimSpace(strings.Join([]string{
			"Use the supplied academic context when available.",
			strings.Join(parts, "\n\n"),
			"User request:\n" + prompt,
		}, "\n\n")),
	}}

	if strings.HasPrefix(contextData.Snapshot, "data:image/") {
		content = append(content, map[string]any{
			"type":      "image_url",
			"image_url": map[string]any{"url": contextData.Snapshot},
		})
	}

	return []map[string]any{
		{"role": "system", "content": "You are OpenSciReader, a precise academic reading assistant."},
		{"role": "user", "content": content},
	}
}

func extractStreamChunk(data string) string {
	type choice struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Text string `json:"text"`
	}
	type payload struct {
		Choices []choice `json:"choices"`
	}

	var parsed payload
	if err := json.Unmarshal([]byte(data), &parsed); err != nil || len(parsed.Choices) == 0 {
		return ""
	}
	if parsed.Choices[0].Delta.Content != "" {
		return parsed.Choices[0].Delta.Content
	}
	if parsed.Choices[0].Message.Content != "" {
		return parsed.Choices[0].Message.Content
	}
	return parsed.Choices[0].Text
}

func parseOpenAICompatibleResponseText(body []byte) (string, error) {
	type response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Text string `json:"text"`
		} `json:"choices"`
	}

	var parsed response
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content != "" {
		return content, nil
	}

	content = strings.TrimSpace(parsed.Choices[0].Text)
	if content != "" {
		return content, nil
	}
	return "", fmt.Errorf("empty response")
}

func stripMarkdownCodeFence(content string) string {
	trimmed := strings.TrimSpace(content)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	return strings.TrimSpace(trimmed)
}

func gatewayEventName(requestID string) string {
	return "gateway:chat:" + requestID
}

func emitGatewayEvent(ctx context.Context, eventName string, payload gatewayStreamEvent) {
	wruntime.EventsEmit(ctx, eventName, payload)
}

func (g *gatewayService) GenerateWorkspaceWiki(ctx context.Context, providerID, modelID int64, prompt string) (string, error) {
	provider, err := g.store.GetProviderSecret(ctx, providerID)
	if err != nil {
		return "", err
	}
	model, err := g.store.GetModel(ctx, modelID)
	if err != nil {
		return "", err
	}
	if model.ProviderID != provider.ID {
		return "", fmt.Errorf("model %d does not belong to provider %d", modelID, providerID)
	}
	if provider.Type != ProviderTypeLLM {
		return "", fmt.Errorf("provider %s is not an llm provider", provider.Name)
	}
	if !provider.IsActive {
		return "", fmt.Errorf("provider %s is inactive", provider.Name)
	}
	if provider.APIKey == "" {
		return "", fmt.Errorf("provider %s has no api key", provider.Name)
	}

	payload, err := json.Marshal(openAIChatRequest{
		Model: model.ModelID,
		Messages: []map[string]any{
			{"role": "system", "content": "You are OpenSciReader, a precise academic workspace wiki writer. Return markdown only."},
			{"role": "user", "content": strings.TrimSpace(prompt)},
		},
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("marshal workspace wiki request: %w", err)
	}

	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		return "", fmt.Errorf("provider base URL is empty")
	}

	resp, err := g.doRequestWith429Retry(ctx, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(payload))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		applyProviderRequestHeaders(req, provider.APIKey)
		return req, nil
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("gateway http error: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return "", fmt.Errorf("read workspace wiki response: %w", err)
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("decode workspace wiki response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("workspace wiki response has no choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		content = strings.TrimSpace(parsed.Choices[0].Text)
	}
	if content == "" {
		return "", fmt.Errorf("workspace wiki response is empty")
	}
	return content, nil
}

func (g *gatewayService) GenerateResearchFigure(ctx context.Context, _, _ int64, prompt string, contextData GatewayContextData, workspaceID string) (FigureGenerationResult, error) {
	workspaceConfig, err := g.store.GetAIWorkspaceConfig(ctx, workspaceID)
	if err != nil {
		return FigureGenerationResult{}, err
	}
	if workspaceConfig.DrawingProviderID == 0 {
		return FigureGenerationResult{}, fmt.Errorf("drawing provider is not configured")
	}
	modelID := strings.TrimSpace(workspaceConfig.DrawingModel)
	if modelID == "" {
		return FigureGenerationResult{}, fmt.Errorf("drawing model is not configured")
	}

	provider, err := g.store.GetProviderSecret(ctx, workspaceConfig.DrawingProviderID)
	if err != nil {
		return FigureGenerationResult{}, err
	}
	if provider.Type != ProviderTypeDrawing {
		return FigureGenerationResult{}, fmt.Errorf("provider %s is not a drawing provider", provider.Name)
	}
	if !provider.IsActive {
		return FigureGenerationResult{}, fmt.Errorf("provider %s is inactive", provider.Name)
	}
	if provider.APIKey == "" {
		return FigureGenerationResult{}, fmt.Errorf("provider %s has no api key", provider.Name)
	}

	model := ModelRecord{
		ProviderID: provider.ID,
		ModelID:    modelID,
	}

	if isGeminiProvider(provider) {
		return g.generateGeminiFigure(ctx, provider, model, prompt, contextData)
	}

	return FigureGenerationResult{}, fmt.Errorf("provider %s does not support research drawing yet", provider.Name)
}

func (g *gatewayService) ProxyTranslation(ctx context.Context, providerID, modelID int64, text, sourceLang, targetLang string) (string, error) {
	provider, err := g.store.GetProviderSecret(ctx, providerID)
	if err != nil {
		return "", err
	}
	if !provider.IsActive {
		return "", fmt.Errorf("provider %s is inactive", provider.Name)
	}

	if isDeepLXProvider(provider) {
		return g.proxyDeepLX(ctx, provider, text, sourceLang, targetLang)
	}
	if isGoogleTranslateProvider(provider) {
		return g.proxyGoogleTranslation(ctx, provider, text, sourceLang, targetLang)
	}
	if isMicrosoftTranslateProvider(provider) {
		return g.proxyMicrosoftTranslation(ctx, provider, text, sourceLang, targetLang)
	}

	if strings.Contains(strings.ToLower(provider.Name), "deepl") || strings.Contains(strings.ToLower(provider.BaseURL), "deepl") {
		return g.proxyDeepL(ctx, provider, text, sourceLang, targetLang)
	}

	if modelID == 0 {
		modelID, err = g.firstModelIDByProvider(ctx, providerID)
		if err != nil {
			return "", err
		}
	}
	return g.proxyLLMTranslation(ctx, providerID, modelID, text, sourceLang, targetLang)
}

func (g *gatewayService) ProxyTranslationWithContext(
	ctx context.Context,
	providerID, modelID int64,
	text, sourceLang, targetLang, previousText, nextText string,
) (string, error) {
	provider, err := g.store.GetProviderSecret(ctx, providerID)
	if err != nil {
		return "", err
	}
	if !provider.IsActive {
		return "", fmt.Errorf("provider %s is inactive", provider.Name)
	}

	if provider.Type != ProviderTypeLLM {
		return g.ProxyTranslation(ctx, providerID, modelID, text, sourceLang, targetLang)
	}
	if modelID == 0 {
		return g.ProxyTranslation(ctx, providerID, modelID, text, sourceLang, targetLang)
	}
	return g.proxyLLMTranslationWithContext(ctx, providerID, modelID, text, sourceLang, targetLang, previousText, nextText)
}

func (g *gatewayService) RunAndCacheGLMOCR(ctx context.Context, providerID int64, pdfHash string, pageNum, resolution int, base64Img string) (OCRPageResult, error) {
	if cached, err := g.store.GetOCRResult(ctx, pdfHash, pageNum); err == nil {
		return cached, nil
	}

	provider, err := g.store.GetProviderSecret(ctx, providerID)
	if err != nil {
		return OCRPageResult{}, err
	}
	blocks, err := g.runGLMOCRLayoutParsing(ctx, provider, base64Img)
	if err != nil {
		return OCRPageResult{}, err
	}

	return g.store.SaveOCRResult(ctx, OCRPageResult{
		PDFHash:    pdfHash,
		PageNumber: pageNum,
		Resolution: resolution,
		Blocks:     blocks,
	})
}

func (g *gatewayService) proxyDeepL(ctx context.Context, provider providerSecretRecord, text, sourceLang, targetLang string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("text", text)
	_ = writer.WriteField("target_lang", strings.ToUpper(targetLang))
	if sourceLang != "" {
		_ = writer.WriteField("source_lang", strings.ToUpper(sourceLang))
	}
	_ = writer.Close()
	bodyBytes := append([]byte(nil), body.Bytes()...)
	contentType := writer.FormDataContentType()

	resp, err := g.doRequestWith429Retry(ctx, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(provider.BaseURL, "/")+"/translate", bytes.NewReader(bodyBytes))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "DeepL-Auth-Key "+provider.APIKey)
		return req, nil
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("translation http error: %s %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	type deepLResponse struct {
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}
	var parsed deepLResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Translations) == 0 {
		return "", fmt.Errorf("empty translation response")
	}
	return parsed.Translations[0].Text, nil
}

func (g *gatewayService) proxyDeepLX(ctx context.Context, provider providerSecretRecord, text, sourceLang, targetLang string) (string, error) {
	endpoint, err := deepLXTranslateEndpoint(provider.BaseURL)
	if err != nil {
		return "", err
	}

	token := strings.TrimSpace(provider.APIKey)
	if token == "" {
		return "", fmt.Errorf("provider %s has no api key", provider.Name)
	}

	if endpoint.Mode == deepLXEndpointOfficial {
		return g.proxyDeepLXOfficial(ctx, endpoint.URL, token, text, sourceLang, targetLang)
	}

	return g.proxyDeepLXBearer(ctx, endpoint.URL, token, text, sourceLang, targetLang)
}

func (g *gatewayService) proxyDeepLXBearer(ctx context.Context, endpointURL, token, text, sourceLang, targetLang string) (string, error) {
	payload := map[string]string{
		"text":        text,
		"target_lang": strings.ToUpper(strings.TrimSpace(targetLang)),
	}
	source := strings.ToUpper(strings.TrimSpace(sourceLang))
	if source != "" && source != "AUTO" {
		payload["source_lang"] = source
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	doRequest := func(useQueryToken bool) (*http.Response, []byte, error) {
		requestURL := endpointURL
		authToken := token
		if useQueryToken {
			parsedURL, parseErr := url.Parse(requestURL)
			if parseErr != nil {
				return nil, nil, parseErr
			}
			query := parsedURL.Query()
			query.Set("token", authToken)
			parsedURL.RawQuery = query.Encode()
			requestURL = parsedURL.String()
			authToken = ""
		}

		resp, respErr := g.doRequestWith429Retry(ctx, func() (*http.Request, error) {
			req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(requestBody))
			if reqErr != nil {
				return nil, reqErr
			}
			req.Header.Set("Content-Type", "application/json")
			if authToken != "" {
				req.Header.Set("Authorization", "Bearer "+authToken)
			}
			return req, nil
		})
		if respErr != nil {
			return nil, nil, respErr
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return resp, body, readErr
	}

	resp, body, err := doRequest(false)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp, body, err = doRequest(true)
		if err != nil {
			return "", err
		}
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("translation http error: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return parseDeepLXTranslationResponse(body)
}

func (g *gatewayService) proxyDeepLXOfficial(ctx context.Context, endpointURL, token, text, sourceLang, targetLang string) (string, error) {
	payload := map[string]any{
		"text":        []string{text},
		"target_lang": strings.ToUpper(strings.TrimSpace(targetLang)),
	}
	source := strings.ToUpper(strings.TrimSpace(sourceLang))
	if source != "" && source != "AUTO" {
		payload["source_lang"] = source
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := g.doRequestWith429Retry(ctx, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, bytes.NewReader(requestBody))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "DeepL-Auth-Key "+token)
		return req, nil
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("translation http error: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return parseDeepLXTranslationResponse(body)
}

func parseDeepLXTranslationResponse(body []byte) (string, error) {
	type deepLXResponse struct {
		Code         int      `json:"code"`
		Data         string   `json:"data"`
		Message      string   `json:"message"`
		Alternatives []string `json:"alternatives"`
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}

	var parsed deepLXResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.Data) != "" {
		return strings.TrimSpace(parsed.Data), nil
	}
	if len(parsed.Translations) > 0 && strings.TrimSpace(parsed.Translations[0].Text) != "" {
		return strings.TrimSpace(parsed.Translations[0].Text), nil
	}
	if len(parsed.Alternatives) > 0 && strings.TrimSpace(parsed.Alternatives[0]) != "" {
		return strings.TrimSpace(parsed.Alternatives[0]), nil
	}
	if strings.TrimSpace(parsed.Message) != "" {
		return "", fmt.Errorf("%s", parsed.Message)
	}
	return "", fmt.Errorf("empty translation response")
}

func (g *gatewayService) generateGeminiFigure(ctx context.Context, provider providerSecretRecord, model ModelRecord, prompt string, contextData GatewayContextData) (FigureGenerationResult, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	body := map[string]any{
		"contents": []map[string]any{{
			"role":  "user",
			"parts": buildGeminiFigureParts(prompt, contextData),
		}},
		"generationConfig": map[string]any{
			"responseModalities": []string{"TEXT", "IMAGE"},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return FigureGenerationResult{}, err
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", baseURL, model.ModelID, provider.APIKey)
	resp, err := g.doRequestWith429Retry(ctx, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return FigureGenerationResult{}, err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return FigureGenerationResult{}, err
	}
	if resp.StatusCode >= 300 {
		return FigureGenerationResult{}, fmt.Errorf("gemini draw http error: %s %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	return parseGeminiFigureResponse(responseBody, prompt)
}

func (g *gatewayService) proxyLLMTranslation(ctx context.Context, providerID, modelID int64, text, sourceLang, targetLang string) (string, error) {
	provider, err := g.store.GetProviderSecret(ctx, providerID)
	if err != nil {
		return "", err
	}
	model, err := g.store.GetModel(ctx, modelID)
	if err != nil {
		return "", err
	}
	prompt := fmt.Sprintf("Translate the following text from %s to %s. Return only the translated text.\n\n%s", sourceLang, targetLang, text)
	payload, err := json.Marshal(openAIChatRequest{Model: model.ModelID, Stream: false, Messages: []map[string]any{{"role": "user", "content": prompt}}})
	if err != nil {
		return "", err
	}
	resp, err := g.doRequestWith429Retry(ctx, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(provider.BaseURL, "/")+"/chat/completions", bytes.NewReader(payload))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		applyProviderRequestHeaders(req, provider.APIKey)
		return req, nil
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("translation gateway http error: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	type response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	var parsed response
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty translation response")
	}
	return parsed.Choices[0].Message.Content, nil
}

func (g *gatewayService) proxyLLMTranslationWithContext(
	ctx context.Context,
	providerID, modelID int64,
	text, sourceLang, targetLang, previousText, nextText string,
) (string, error) {
	provider, err := g.store.GetProviderSecret(ctx, providerID)
	if err != nil {
		return "", err
	}
	model, err := g.store.GetModel(ctx, modelID)
	if err != nil {
		return "", err
	}

	prompt := buildContextualTranslationPrompt(text, sourceLang, targetLang, previousText, nextText)
	payload, err := json.Marshal(openAIChatRequest{
		Model:  model.ModelID,
		Stream: false,
		Messages: []map[string]any{
			{
				"role":    "system",
				"content": "You are a precise academic translation engine. Translate only the target segment. Use surrounding context only to resolve terminology and continuity. Return only the translated target segment.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
	})
	if err != nil {
		return "", err
	}

	resp, err := g.doRequestWith429Retry(ctx, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(provider.BaseURL, "/")+"/chat/completions", bytes.NewReader(payload))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		applyProviderRequestHeaders(req, provider.APIKey)
		return req, nil
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("translation gateway http error: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	type response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	var parsed response
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty translation response")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func buildContextualTranslationPrompt(text, sourceLang, targetLang, previousText, nextText string) string {
	sections := []string{
		fmt.Sprintf("Translate the target segment from %s to %s.", sourceLang, targetLang),
		"Only output the translation for the target segment.",
		"Do not translate the context segments themselves.",
	}
	if strings.TrimSpace(previousText) != "" {
		sections = append(sections, "Previous context:\n"+strings.TrimSpace(previousText))
	}
	sections = append(sections, "Target segment:\n"+strings.TrimSpace(text))
	if strings.TrimSpace(nextText) != "" {
		sections = append(sections, "Next context:\n"+strings.TrimSpace(nextText))
	}
	return strings.Join(sections, "\n\n")
}

func isDeepLXProvider(provider providerSecretRecord) bool {
	name := strings.ToLower(strings.TrimSpace(provider.Name))
	baseURL := strings.ToLower(strings.TrimSpace(provider.BaseURL))
	if strings.Contains(name, "deeplx") || strings.Contains(baseURL, "deeplx") {
		return true
	}

	endpoint, err := deepLXTranslateEndpoint(provider.BaseURL)
	if err != nil {
		return false
	}
	return endpoint.Mode == deepLXEndpointPro || endpoint.Mode == deepLXEndpointOfficial
}

func isGoogleTranslateProvider(provider providerSecretRecord) bool {
	name := strings.ToLower(strings.TrimSpace(provider.Name))
	baseURL := strings.ToLower(strings.TrimSpace(provider.BaseURL))
	return strings.Contains(name, "google") || strings.Contains(baseURL, "translation.googleapis.com")
}

func isMicrosoftTranslateProvider(provider providerSecretRecord) bool {
	name := strings.ToLower(strings.TrimSpace(provider.Name))
	baseURL := strings.ToLower(strings.TrimSpace(provider.BaseURL))
	return strings.Contains(name, "microsoft") ||
		strings.Contains(name, "azure") ||
		strings.Contains(baseURL, "cognitive.microsofttranslator.com") ||
		strings.Contains(baseURL, "cognitiveservices.azure.com")
}

func deepLXTranslateEndpoint(rawBaseURL string) (deepLXEndpoint, error) {
	baseURL := strings.TrimSpace(rawBaseURL)
	if baseURL == "" {
		return deepLXEndpoint{}, fmt.Errorf("provider base URL is empty")
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return deepLXEndpoint{}, fmt.Errorf("invalid provider base URL: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return deepLXEndpoint{}, fmt.Errorf("deeplx endpoint must be a full url ending with /translate")
	}

	normalizedPath := strings.TrimRight(parsedURL.Path, "/")
	if normalizedPath == "" {
		return deepLXEndpoint{}, fmt.Errorf("deeplx endpoint must include the full /translate path")
	}
	lowerPath := strings.ToLower(normalizedPath)

	mode := deepLXEndpointFree
	switch {
	case strings.HasSuffix(lowerPath, "/v1/translate"):
		mode = deepLXEndpointPro
	case strings.HasSuffix(lowerPath, "/v2/translate"):
		mode = deepLXEndpointOfficial
	case strings.HasSuffix(lowerPath, "/translate"):
		mode = deepLXEndpointFree
	default:
		return deepLXEndpoint{}, fmt.Errorf("deeplx endpoint must end with /translate, /v1/translate, or /v2/translate")
	}

	parsedURL.Path = normalizedPath
	return deepLXEndpoint{
		URL:  parsedURL.String(),
		Mode: mode,
	}, nil
}

func (g *gatewayService) proxyGoogleTranslation(ctx context.Context, provider providerSecretRecord, text, sourceLang, targetLang string) (string, error) {
	endpoint := strings.TrimSpace(provider.BaseURL)
	if endpoint == "" {
		endpoint = "https://translation.googleapis.com/language/translate/v2"
	}
	if provider.APIKey == "" {
		return "", fmt.Errorf("provider %s has no api key", provider.Name)
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid provider base URL: %w", err)
	}
	query := parsedURL.Query()
	query.Set("key", provider.APIKey)
	parsedURL.RawQuery = query.Encode()

	form := url.Values{}
	form.Set("q", text)
	form.Set("target", normalizeGoogleTargetLang(targetLang))
	if source := normalizeGoogleSourceLang(sourceLang); source != "" {
		form.Set("source", source)
	}
	form.Set("format", "text")

	formBody := form.Encode()
	resp, err := g.doRequestWith429Retry(ctx, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, parsedURL.String(), strings.NewReader(formBody))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req, nil
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("translation http error: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	type googleResponse struct {
		Data struct {
			Translations []struct {
				TranslatedText string `json:"translatedText"`
			} `json:"translations"`
		} `json:"data"`
		Error any `json:"error"`
	}
	var parsed googleResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Data.Translations) == 0 {
		return "", fmt.Errorf("empty translation response")
	}
	return html.UnescapeString(strings.TrimSpace(parsed.Data.Translations[0].TranslatedText)), nil
}

func (g *gatewayService) proxyMicrosoftTranslation(ctx context.Context, provider providerSecretRecord, text, sourceLang, targetLang string) (string, error) {
	endpoint, region, err := microsoftTranslateEndpoint(provider, sourceLang, targetLang)
	if err != nil {
		return "", err
	}
	if provider.APIKey == "" {
		return "", fmt.Errorf("provider %s has no api key", provider.Name)
	}

	requestBody, err := json.Marshal([]map[string]string{{"Text": text}})
	if err != nil {
		return "", err
	}
	resp, err := g.doRequestWith429Retry(ctx, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Ocp-Apim-Subscription-Key", provider.APIKey)
		if region != "" {
			req.Header.Set("Ocp-Apim-Subscription-Region", region)
		}
		return req, nil
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("translation http error: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	type microsoftResponse []struct {
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}
	var parsed microsoftResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed) == 0 || len(parsed[0].Translations) == 0 {
		return "", fmt.Errorf("empty translation response")
	}
	return strings.TrimSpace(parsed[0].Translations[0].Text), nil
}

func microsoftTranslateEndpoint(provider providerSecretRecord, sourceLang, targetLang string) (string, string, error) {
	baseURL := strings.TrimSpace(provider.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.cognitive.microsofttranslator.com/translate"
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid provider base URL: %w", err)
	}
	pathLower := strings.ToLower(strings.TrimRight(parsedURL.Path, "/"))
	switch {
	case strings.HasSuffix(pathLower, "/translator/text/v3.0/translate"), strings.HasSuffix(pathLower, "/translate"):
	case strings.HasSuffix(pathLower, "/translator/text/v3.0"):
		parsedURL.Path = strings.TrimRight(parsedURL.Path, "/") + "/translate"
	default:
		parsedURL.Path = strings.TrimRight(parsedURL.Path, "/") + "/translate"
	}

	query := parsedURL.Query()
	query.Set("api-version", "3.0")
	query.Set("to", normalizeMicrosoftTargetLang(targetLang))
	if source := normalizeMicrosoftSourceLang(sourceLang); source != "" {
		query.Set("from", source)
	}
	region := strings.TrimSpace(provider.Region)
	if region == "" {
		region = strings.TrimSpace(query.Get("region"))
	}
	query.Del("region")
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), region, nil
}

func normalizeGoogleTargetLang(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ZH":
		return "zh-CN"
	case "EN":
		return "en"
	case "JA":
		return "ja"
	case "DE":
		return "de"
	case "FR":
		return "fr"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeGoogleSourceLang(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "AUTO") {
		return ""
	}
	return normalizeGoogleTargetLang(value)
}

func normalizeMicrosoftTargetLang(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ZH":
		return "zh-Hans"
	case "EN":
		return "en"
	case "JA":
		return "ja"
	case "DE":
		return "de"
	case "FR":
		return "fr"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeMicrosoftSourceLang(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "AUTO") {
		return ""
	}
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ZH":
		return "zh-Hans"
	default:
		return normalizeGoogleTargetLang(value)
	}
}

func (g *gatewayService) firstModelIDByProvider(ctx context.Context, providerID int64) (int64, error) {
	rows, err := g.store.appDB.QueryContext(ctx, `SELECT id FROM models WHERE provider_id = ? ORDER BY id LIMIT 1;`, providerID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, fmt.Errorf("provider %d has no models", providerID)
	}
	var id int64
	if err := rows.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func parseOCRBlocksFromChatResponse(body []byte) ([]OCRTextBlock, error) {
	type response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	var parsed response
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("empty ocr response")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	var blocks []OCRTextBlock
	if err := json.Unmarshal([]byte(content), &blocks); err != nil {
		return nil, fmt.Errorf("decode ocr json: %w", err)
	}
	return blocks, nil
}

func isGeminiProvider(provider providerSecretRecord) bool {
	name := strings.ToLower(provider.Name)
	baseURL := strings.ToLower(provider.BaseURL)
	return strings.Contains(name, "gemini") || strings.Contains(name, "google") || strings.Contains(baseURL, "generativelanguage.googleapis.com")
}

func buildGeminiFigureParts(prompt string, contextData GatewayContextData) []map[string]any {
	text := strings.TrimSpace(strings.Join([]string{
		"Generate a clean scientific figure for academic communication.",
		"Prefer white background, publication-ready layout, clear labels, and restrained colors.",
		"If the prompt implies a workflow, draw an annotated pipeline diagram.",
		"If the prompt implies experimental results, draw a polished chart with meaningful labels.",
		"Context:\nDocument: " + contextData.ItemTitle + "\nCiteKey: " + contextData.CiteKey + fmt.Sprintf("\nPage: %d", contextData.Page),
		"Selected text:\n" + contextData.Selection,
		"User drawing request:\n" + prompt,
	}, "\n\n"))

	parts := []map[string]any{{
		"text": text,
	}}
	if strings.HasPrefix(contextData.Snapshot, "data:image/") {
		parts = append(parts, map[string]any{
			"inline_data": map[string]any{
				"mime_type": detectDataURLMimeType(contextData.Snapshot),
				"data":      extractDataURLPayload(contextData.Snapshot),
			},
		})
	}
	return parts
}

func parseGeminiFigureResponse(body []byte, prompt string) (FigureGenerationResult, error) {
	type geminiResponse struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text       string `json:"text"`
					InlineData *struct {
						MimeType string `json:"mime_type"`
						Data     string `json:"data"`
					} `json:"inline_data"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	var parsed geminiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return FigureGenerationResult{}, err
	}
	for _, candidate := range parsed.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil && part.InlineData.Data != "" {
				mime := part.InlineData.MimeType
				if mime == "" {
					mime = "image/png"
				}
				return FigureGenerationResult{
					MimeType: mime,
					DataURL:  fmt.Sprintf("data:%s;base64,%s", mime, part.InlineData.Data),
					Prompt:   prompt,
				}, nil
			}
		}
	}
	return FigureGenerationResult{}, fmt.Errorf("gemini response did not include inline image data")
}

func detectDataURLMimeType(input string) string {
	if strings.HasPrefix(input, "data:image/jpeg") {
		return "image/jpeg"
	}
	if strings.HasPrefix(input, "data:image/webp") {
		return "image/webp"
	}
	return "image/png"
}

func extractDataURLPayload(input string) string {
	parts := strings.SplitN(input, ",", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}
