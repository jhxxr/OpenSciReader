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

type Workspace struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type WorkspaceUpsertInput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type WorkspaceWikiScanJobStatus string

const (
	WorkspaceWikiScanJobQueued    WorkspaceWikiScanJobStatus = "queued"
	WorkspaceWikiScanJobRunning   WorkspaceWikiScanJobStatus = "running"
	WorkspaceWikiScanJobCompleted WorkspaceWikiScanJobStatus = "completed"
	WorkspaceWikiScanJobFailed    WorkspaceWikiScanJobStatus = "failed"
)

type WorkspaceWikiScanStartInput struct {
	WorkspaceID string `json:"workspaceId"`
	DocumentID  string `json:"documentId"`
	ProviderID  int64  `json:"providerId"`
	ModelID     int64  `json:"modelId"`
}

type WorkspaceWikiScanJob struct {
	ID           int64                      `json:"id"`
	WorkspaceID  string                     `json:"workspaceId"`
	DocumentID   string                     `json:"documentId"`
	Status       WorkspaceWikiScanJobStatus `json:"status"`
	CurrentStage string                     `json:"currentStage"`
	Message      string                     `json:"message"`
	ProviderID   int64                      `json:"providerId"`
	ModelID      int64                      `json:"modelId"`
	StartedAt    string                     `json:"startedAt"`
	FinishedAt   string                     `json:"finishedAt"`
	UpdatedAt    string                     `json:"updatedAt"`
}

type DocumentRecord struct {
	ID               string `json:"id"`
	WorkspaceID      string `json:"workspaceId"`
	Title            string `json:"title"`
	DocumentType     string `json:"documentType"`
	SourceType       string `json:"sourceType"`
	DefaultAssetID   string `json:"defaultAssetId"`
	OriginalFileName string `json:"originalFileName"`
	PrimaryPDFPath   string `json:"primaryPdfPath"`
	ContentHash      string `json:"contentHash"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type DocumentAssetRecord struct {
	ID           string `json:"id"`
	DocumentID   string `json:"documentId"`
	WorkspaceID  string `json:"workspaceId"`
	Kind         string `json:"kind"`
	Role         string `json:"role"`
	FileName     string `json:"fileName"`
	RelativePath string `json:"relativePath"`
	AbsolutePath string `json:"absolutePath"`
	MimeType     string `json:"mimeType"`
	ByteSize     int64  `json:"byteSize"`
	ContentHash  string `json:"contentHash"`
	CreatedAt    string `json:"createdAt"`
}

type ImportRecord struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	DocumentID  string `json:"documentId"`
	SourceType  string `json:"sourceType"`
	SourceLabel string `json:"sourceLabel"`
	SourceRef   string `json:"sourceRef"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	CreatedAt   string `json:"createdAt"`
}

type DocumentExternalLink struct {
	ID          string `json:"id"`
	DocumentID  string `json:"documentId"`
	WorkspaceID string `json:"workspaceId"`
	Provider    string `json:"provider"`
	ExternalID  string `json:"externalId"`
	ExternalKey string `json:"externalKey"`
	CreatedAt   string `json:"createdAt"`
}

type ImportFilesInput struct {
	WorkspaceID string   `json:"workspaceId"`
	FilePaths   []string `json:"filePaths"`
	SourceType  string   `json:"sourceType"`
	SourceLabel string   `json:"sourceLabel"`
	SourceRef   string   `json:"sourceRef"`
	Title       string   `json:"title"`
}

type ImportFilesResult struct {
	Workspace Workspace        `json:"workspace"`
	Documents []DocumentRecord `json:"documents"`
	Imports   []ImportRecord   `json:"imports"`
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

type WorkspaceKnowledgeEvidenceHit struct {
	Kind       string                        `json:"kind"`
	ID         string                        `json:"id"`
	Title      string                        `json:"title"`
	Summary    string                        `json:"summary"`
	Excerpt    string                        `json:"excerpt"`
	SourceRefs []WorkspaceKnowledgeSourceRef `json:"sourceRefs"`
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
	ID          int64  `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	DocumentID  string `json:"documentId"`
	ItemID      string `json:"itemId"`
	ItemTitle   string `json:"itemTitle"`
	Page        int    `json:"page"`
	Kind        string `json:"kind"`
	Prompt      string `json:"prompt"`
	Response    string `json:"response"`
	CreatedAt   string `json:"createdAt"`
}

type ReaderNoteEntry struct {
	ID          int64  `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	DocumentID  string `json:"documentId"`
	ItemID      string `json:"itemId"`
	ItemTitle   string `json:"itemTitle"`
	Page        int    `json:"page"`
	AnchorText  string `json:"anchorText"`
	Content     string `json:"content"`
	CreatedAt   string `json:"createdAt"`
}
