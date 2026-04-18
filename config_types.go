package main

type ProviderType string

const (
	ProviderTypeLLM       ProviderType = "llm"
	ProviderTypeOCR       ProviderType = "ocr"
	ProviderTypeDrawing   ProviderType = "drawing"
	ProviderTypeTranslate ProviderType = "translate"
)

type PDFTranslateRuntimeStatus string

const (
	PDFTranslateRuntimeStatusMissing    PDFTranslateRuntimeStatus = "missing"
	PDFTranslateRuntimeStatusValid      PDFTranslateRuntimeStatus = "valid"
	PDFTranslateRuntimeStatusInvalid    PDFTranslateRuntimeStatus = "invalid"
	PDFTranslateRuntimeStatusInstalling PDFTranslateRuntimeStatus = "installing"
)

type ProviderRecord struct {
	ID           int64        `json:"id"`
	Name         string       `json:"name"`
	Type         ProviderType `json:"type"`
	BaseURL      string       `json:"baseUrl"`
	Region       string       `json:"region"`
	HasAPIKey    bool         `json:"hasApiKey"`
	APIKeyMasked string       `json:"apiKeyMasked"`
	IsActive     bool         `json:"isActive"`
}

type ProviderUpsertInput struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Type        ProviderType `json:"type"`
	BaseURL     string       `json:"baseUrl"`
	Region      string       `json:"region"`
	APIKey      string       `json:"apiKey"`
	ClearAPIKey bool         `json:"clearApiKey"`
	IsActive    bool         `json:"isActive"`
}

type ModelRecord struct {
	ID            int64  `json:"id"`
	ProviderID    int64  `json:"providerId"`
	ModelID       string `json:"modelId"`
	ContextWindow int    `json:"contextWindow"`
}

type ModelUpsertInput struct {
	ID            int64  `json:"id"`
	ProviderID    int64  `json:"providerId"`
	ModelID       string `json:"modelId"`
	ContextWindow int    `json:"contextWindow"`
}

type ProviderConfig struct {
	Provider ProviderRecord `json:"provider"`
	Models   []ModelRecord  `json:"models"`
}

type PDFTranslateRuntimeConfig struct {
	Installed           bool                      `json:"installed"`
	Status              PDFTranslateRuntimeStatus `json:"status"`
	RuntimeID           string                    `json:"runtimeId"`
	Version             string                    `json:"version"`
	Platform            string                    `json:"platform"`
	RuntimeDir          string                    `json:"runtimeDir"`
	PythonPath          string                    `json:"pythonPath"`
	ManifestPath        string                    `json:"manifestPath"`
	InstalledAt         string                    `json:"installedAt"`
	SourceFileName      string                    `json:"sourceFileName"`
	LastValidationError string                    `json:"lastValidationError"`
}

type PDFTranslateRuntimeImportResult struct {
	Runtime PDFTranslateRuntimeConfig `json:"runtime"`
}

type ConfigSnapshot struct {
	Providers           []ProviderConfig          `json:"providers"`
	PDFTranslateRuntime PDFTranslateRuntimeConfig `json:"pdfTranslateRuntime"`
}

type AIWorkspaceConfig struct {
	SummaryMode          string `json:"summaryMode"`
	SummaryChunkPages    int    `json:"summaryChunkPages"`
	SummaryChunkMaxChars int    `json:"summaryChunkMaxChars"`
	AutoRestoreCount     int    `json:"autoRestoreCount"`
	TableTemplate        string `json:"tableTemplate"`
	TablePrompt          string `json:"tablePrompt"`
	CustomPromptDraft    string `json:"customPromptDraft"`
	FollowUpPromptDraft  string `json:"followUpPromptDraft"`
	DrawingPromptDraft   string `json:"drawingPromptDraft"`
	DrawingProviderID    int64  `json:"drawingProviderId"`
	DrawingModel         string `json:"drawingModel"`
}

type DiscoveredModel struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	OwnedBy string `json:"ownedBy"`
}

type DiscoveredModelsResponse struct {
	Models []DiscoveredModel `json:"models"`
	Total  int               `json:"total"`
}

type OCRTextBlock struct {
	Text   string  `json:"text"`
	Left   float64 `json:"left"`
	Top    float64 `json:"top"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type OCRPageResult struct {
	ID         int64          `json:"id"`
	PDFHash    string         `json:"pdfHash"`
	PageNumber int            `json:"pageNumber"`
	Resolution int            `json:"resolution"`
	Blocks     []OCRTextBlock `json:"blocks"`
	CreatedAt  string         `json:"createdAt"`
}

type ChatHistoryEntry struct {
	ID        int64  `json:"id"`
	ItemID    string `json:"itemId"`
	ItemTitle string `json:"itemTitle"`
	Page      int    `json:"page"`
	Kind      string `json:"kind"`
	Prompt    string `json:"prompt"`
	Response  string `json:"response"`
	CreatedAt string `json:"createdAt"`
}

type ReaderNoteEntry struct {
	ID         int64  `json:"id"`
	ItemID     string `json:"itemId"`
	ItemTitle  string `json:"itemTitle"`
	Page       int    `json:"page"`
	AnchorText string `json:"anchorText"`
	Content    string `json:"content"`
	CreatedAt  string `json:"createdAt"`
}
