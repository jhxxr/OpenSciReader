package main

import (
	"net/http"
	"strings"
)

// Some third-party OpenAI-compatible providers reject SDK or empty user agents
// behind WAF rules, while allowing normal browser-like traffic.
const browserLikeUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"

func applyProviderRequestHeaders(req *http.Request, bearerToken string) {
	if req == nil {
		return
	}
	req.Header.Set("User-Agent", browserLikeUserAgent)
	if token := strings.TrimSpace(bearerToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}
