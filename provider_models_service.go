package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

func (g *gatewayService) FetchProviderModels(ctx context.Context, providerID int64) (DiscoveredModelsResponse, error) {
	provider, err := g.store.GetProviderSecret(ctx, providerID)
	if err != nil {
		return DiscoveredModelsResponse{}, err
	}

	endpoint, useBearerAuth, err := buildProviderModelsEndpoint(provider)
	if err != nil {
		return DiscoveredModelsResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return DiscoveredModelsResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	if useBearerAuth && strings.TrimSpace(provider.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(provider.APIKey))
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return DiscoveredModelsResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return DiscoveredModelsResponse{}, err
	}
	if resp.StatusCode >= 300 {
		return DiscoveredModelsResponse{}, fmt.Errorf("model discovery http error: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	models, err := parseDiscoveredModels(body)
	if err != nil {
		return DiscoveredModelsResponse{}, err
	}

	return DiscoveredModelsResponse{
		Models: models,
		Total:  len(models),
	}, nil
}

func buildProviderModelsEndpoint(provider providerSecretRecord) (string, bool, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		return "", false, fmt.Errorf("provider %s has empty base url", provider.Name)
	}

	if isGoogleModelDiscoveryProvider(provider) {
		parsedURL, err := url.Parse(baseURL)
		if err != nil {
			return "", false, fmt.Errorf("invalid provider base url: %w", err)
		}
		if !strings.HasSuffix(strings.TrimRight(parsedURL.Path, "/"), "/models") {
			parsedURL.Path = strings.TrimRight(parsedURL.Path, "/") + "/models"
		}
		query := parsedURL.Query()
		if apiKey := strings.TrimSpace(provider.APIKey); apiKey != "" {
			query.Set("key", apiKey)
		}
		parsedURL.RawQuery = query.Encode()
		return parsedURL.String(), false, nil
	}

	if strings.HasSuffix(strings.ToLower(baseURL), "/models") {
		return baseURL, true, nil
	}
	return baseURL + "/models", true, nil
}

func isGoogleModelDiscoveryProvider(provider providerSecretRecord) bool {
	name := strings.ToLower(strings.TrimSpace(provider.Name))
	baseURL := strings.ToLower(strings.TrimSpace(provider.BaseURL))
	return strings.Contains(name, "gemini") ||
		strings.Contains(name, "google") ||
		strings.Contains(baseURL, "generativelanguage.googleapis.com")
}

func parseDiscoveredModels(body []byte) ([]DiscoveredModel, error) {
	if models, ok := parseOpenAICompatibleModels(body); ok {
		return dedupeAndSortDiscoveredModels(models), nil
	}
	if models, ok := parseGoogleModels(body); ok {
		return dedupeAndSortDiscoveredModels(models), nil
	}
	return nil, fmt.Errorf("unsupported model discovery response: %s", strings.TrimSpace(string(body)))
}

func parseOpenAICompatibleModels(body []byte) ([]DiscoveredModel, bool) {
	type openAIModel struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		OwnedBy string `json:"owned_by"`
	}
	type openAIResponse struct {
		Data []openAIModel `json:"data"`
	}

	var response openAIResponse
	if err := json.Unmarshal(body, &response); err != nil || len(response.Data) == 0 {
		return nil, false
	}

	models := make([]DiscoveredModel, 0, len(response.Data))
	for _, item := range response.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = id
		}
		models = append(models, DiscoveredModel{
			ID:      id,
			Name:    name,
			OwnedBy: strings.TrimSpace(item.OwnedBy),
		})
	}
	return models, len(models) > 0
}

func parseGoogleModels(body []byte) ([]DiscoveredModel, bool) {
	type googleModel struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	}
	type googleResponse struct {
		Models []googleModel `json:"models"`
	}

	var response googleResponse
	if err := json.Unmarshal(body, &response); err != nil || len(response.Models) == 0 {
		return nil, false
	}

	models := make([]DiscoveredModel, 0, len(response.Models))
	for _, item := range response.Models {
		id := strings.TrimSpace(strings.TrimPrefix(item.Name, "models/"))
		if id == "" {
			continue
		}
		name := strings.TrimSpace(item.DisplayName)
		if name == "" {
			name = id
		}
		models = append(models, DiscoveredModel{
			ID:      id,
			Name:    name,
			OwnedBy: "google",
		})
	}
	return models, len(models) > 0
}

func dedupeAndSortDiscoveredModels(models []DiscoveredModel) []DiscoveredModel {
	unique := make(map[string]DiscoveredModel, len(models))
	order := make([]string, 0, len(models))
	for _, item := range models {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		if _, exists := unique[id]; exists {
			continue
		}
		unique[id] = item
		order = append(order, id)
	}

	sort.SliceStable(order, func(i, j int) bool {
		left := strings.ToLower(unique[order[i]].Name)
		right := strings.ToLower(unique[order[j]].Name)
		if left == right {
			return strings.ToLower(order[i]) < strings.ToLower(order[j])
		}
		return left < right
	})

	result := make([]DiscoveredModel, 0, len(order))
	for _, id := range order {
		result = append(result, unique[id])
	}
	return result
}
