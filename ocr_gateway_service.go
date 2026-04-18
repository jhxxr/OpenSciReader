package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (g *gatewayService) runGLMOCRLayoutParsing(ctx context.Context, provider providerSecretRecord, fileInput string) ([]OCRTextBlock, error) {
	endpoint, err := glmOCREndpoint(provider.BaseURL)
	if err != nil {
		return nil, err
	}

	filePayload := normalizeOCRFileInput(fileInput)
	if filePayload == "" {
		return nil, fmt.Errorf("ocr image payload is empty")
	}

	payload, err := json.Marshal(map[string]any{
		"model": "glm-ocr",
		"file":  filePayload,
	})
	if err != nil {
		return nil, err
	}

	body, err := g.doGLMOCRRequest(ctx, endpoint, provider.APIKey, payload)
	if err != nil {
		return nil, err
	}

	imageWidth, imageHeight, err := decodeOCRImageSize(filePayload)
	if err != nil {
		return nil, err
	}

	return parseGLMOCRBlocks(body, imageWidth, imageHeight)
}

func glmOCREndpoint(rawBaseURL string) (string, error) {
	baseURL := strings.TrimSpace(rawBaseURL)
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4/layout_parsing"
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid ocr base url: %w", err)
	}

	pathLower := strings.ToLower(strings.TrimRight(parsedURL.Path, "/"))
	if !strings.HasSuffix(pathLower, "/layout_parsing") {
		parsedURL.Path = strings.TrimRight(parsedURL.Path, "/") + "/layout_parsing"
	}
	return parsedURL.String(), nil
}

func normalizeOCRFileInput(input string) string {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "data:image/") {
		return extractDataURLPayload(trimmed)
	}
	return trimmed
}

func (g *gatewayService) doGLMOCRRequest(ctx context.Context, endpoint, apiKey string, payload []byte) ([]byte, error) {
	token := strings.TrimSpace(apiKey)
	if token == "" {
		return nil, fmt.Errorf("ocr api key is empty")
	}

	authCandidates := []string{token}
	if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
		authCandidates = append(authCandidates, "Bearer "+token)
	}

	var lastErr error
	for index, authValue := range authCandidates {
		resp, err := g.doRequestWith429Retry(ctx, func() (*http.Request, error) {
			req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
			if reqErr != nil {
				return nil, reqErr
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", authValue)
			return req, nil
		})
		if err != nil {
			return nil, err
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if resp.StatusCode < 300 {
			return body, nil
		}

		lastErr = fmt.Errorf("ocr gateway http error: %s %s", resp.Status, strings.TrimSpace(string(body)))
		if resp.StatusCode != http.StatusUnauthorized || index == len(authCandidates)-1 {
			break
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("ocr request failed")
	}
	return nil, lastErr
}

func decodeOCRImageSize(base64Payload string) (float64, float64, error) {
	decoded, err := base64.StdEncoding.DecodeString(base64Payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(base64Payload)
		if err != nil {
			return 0, 0, fmt.Errorf("decode ocr image: %w", err)
		}
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(decoded))
	if err != nil {
		return 0, 0, fmt.Errorf("read ocr image size: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 {
		return 0, 0, fmt.Errorf("invalid ocr image size")
	}
	return float64(config.Width), float64(config.Height), nil
}

func parseGLMOCRBlocks(body []byte, imageWidth, imageHeight float64) ([]OCRTextBlock, error) {
	type layoutDetail struct {
		Content string    `json:"content"`
		BBox2D  []float64 `json:"bbox_2d"`
	}
	type response struct {
		Data struct {
			LayoutDetails []layoutDetail `json:"layout_details"`
		} `json:"data"`
	}

	var parsed response
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode glm ocr response: %w", err)
	}
	if len(parsed.Data.LayoutDetails) == 0 {
		return nil, fmt.Errorf("empty glm ocr response")
	}

	blocks := make([]OCRTextBlock, 0, len(parsed.Data.LayoutDetails))
	for _, detail := range parsed.Data.LayoutDetails {
		text := strings.TrimSpace(detail.Content)
		if text == "" || len(detail.BBox2D) < 4 {
			continue
		}

		left := clampUnit(detail.BBox2D[0] / imageWidth)
		top := clampUnit(detail.BBox2D[1] / imageHeight)
		right := clampUnit(detail.BBox2D[2] / imageWidth)
		bottom := clampUnit(detail.BBox2D[3] / imageHeight)

		if right <= left || bottom <= top {
			continue
		}

		blocks = append(blocks, OCRTextBlock{
			Text:   text,
			Left:   left,
			Top:    top,
			Width:  right - left,
			Height: bottom - top,
		})
	}

	if len(blocks) == 0 {
		return nil, fmt.Errorf("glm ocr returned no positioned text blocks")
	}
	return blocks, nil
}

func clampUnit(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}
