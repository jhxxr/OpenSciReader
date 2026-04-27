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
	WikiScanProviderID   int64  `json:"wikiScanProviderId"`
	WikiScanModelID      int64  `json:"wikiScanModelId"`
}

type WorkspaceKnowledgeEvidenceHit struct {
	Kind       string                        `json:"kind"`
	ID         string                        `json:"id"`
	Title      string                        `json:"title"`
	Summary    string                        `json:"summary"`
	Excerpt    string                        `json:"excerpt"`
	SourceRefs []WorkspaceKnowledgeSourceRef `json:"sourceRefs"`
}

type WorkspaceKnowledgeProcessingStatus string

const (
	WorkspaceKnowledgeProcessingPending WorkspaceKnowledgeProcessingStatus = "pending"
	WorkspaceKnowledgeProcessingRunning WorkspaceKnowledgeProcessingStatus = "running"
	WorkspaceKnowledgeProcessingReady   WorkspaceKnowledgeProcessingStatus = "ready"
	WorkspaceKnowledgeProcessingFailed  WorkspaceKnowledgeProcessingStatus = "failed"
)

type WorkspaceKnowledgeQueryInput struct {
	WorkspaceID string `json:"workspaceId"`
	ProviderID  int64  `json:"providerId"`
	ModelID     int64  `json:"modelId"`
	Question    string `json:"question"`
}

type WorkspaceKnowledgeCandidate struct {
	ID         string                        `json:"id"`
	Title      string                        `json:"title"`
	Type       string                        `json:"type"`
	Summary    string                        `json:"summary"`
	Aliases    []string                      `json:"aliases"`
	EntityIDs  []string                      `json:"entityIds"`
	Priority   string                        `json:"priority"`
	SourceID   string                        `json:"sourceId"`
	PageStart  int                           `json:"pageStart"`
	PageEnd    int                           `json:"pageEnd"`
	Excerpt    string                        `json:"excerpt"`
	SourceRefs []WorkspaceKnowledgeSourceRef `json:"sourceRefs"`
}

type WorkspaceKnowledgeQueryResult struct {
	Answer     string                          `json:"answer"`
	Evidence   []WorkspaceKnowledgeEvidenceHit `json:"evidence"`
	Candidates []WorkspaceKnowledgeCandidate   `json:"candidates"`
}

type WorkspaceKnowledgePromotionInput struct {
	WorkspaceID string                        `json:"workspaceId"`
	Candidates  []WorkspaceKnowledgeCandidate `json:"candidates"`
}

type WorkspaceAgentSurface string

const (
	WorkspaceAgentSurfaceWorkspace WorkspaceAgentSurface = "workspace"
	WorkspaceAgentSurfaceReader    WorkspaceAgentSurface = "reader"
)

type WorkspaceAgentSkillName string

const (
	WorkspaceAgentSkillAskWithEvidence WorkspaceAgentSkillName = "ask_with_evidence"
	WorkspaceAgentSkillReadingOutputs  WorkspaceAgentSkillName = "reading_outputs"
	WorkspaceAgentSkillTaskPlanning    WorkspaceAgentSkillName = "task_planning"
	WorkspaceAgentSkillBuildMemory     WorkspaceAgentSkillName = "build_memory"
	WorkspaceAgentSkillCrossSource     WorkspaceAgentSkillName = "cross_source_synthesis"
	WorkspaceAgentSkillPromoteToWiki   WorkspaceAgentSkillName = "promote_to_wiki"
	WorkspaceAgentSkillToolExecution   WorkspaceAgentSkillName = "tool_execution"
)

type WorkspaceAgentSkillDefinition struct {
	Name          string `json:"name"`
	Label         string `json:"label"`
	Description   string `json:"description"`
	ManualEnabled bool   `json:"manualEnabled"`
	AutoEnabled   bool   `json:"autoEnabled"`
	ReaderEnabled bool   `json:"readerEnabled"`
	WorkspaceOnly bool   `json:"workspaceOnly"`
}

type WorkspaceAgentExecutedSkill struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	RoutedBy    string `json:"routedBy"`
	Reason      string `json:"reason"`
	DisplayText string `json:"displayText"`
}

type WorkspaceAgentMessageRole string

const (
	WorkspaceAgentMessageRoleUser      WorkspaceAgentMessageRole = "user"
	WorkspaceAgentMessageRoleAssistant WorkspaceAgentMessageRole = "assistant"
)

type WorkspaceAgentSession struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Title       string `json:"title"`
	Surface     string `json:"surface"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type WorkspaceAgentMessage struct {
	ID            int64                        `json:"id"`
	SessionID     string                       `json:"sessionId"`
	WorkspaceID   string                       `json:"workspaceId"`
	Surface       string                       `json:"surface"`
	Role          string                       `json:"role"`
	Kind          string                       `json:"kind"`
	Prompt        string                       `json:"prompt"`
	Content       string                       `json:"content"`
	SkillName     string                       `json:"skillName"`
	ExecutedSkill *WorkspaceAgentExecutedSkill `json:"executedSkill,omitempty"`
	EvidenceCount int                          `json:"evidenceCount"`
	CreatedAt     string                       `json:"createdAt"`
}

type WorkspaceAgentSessionCreateInput struct {
	WorkspaceID string `json:"workspaceId"`
	Title       string `json:"title"`
	Surface     string `json:"surface"`
}

type WorkspaceAgentMessageCreateInput struct {
	SessionID     string                       `json:"sessionId"`
	WorkspaceID   string                       `json:"workspaceId"`
	Surface       string                       `json:"surface"`
	Role          string                       `json:"role"`
	Kind          string                       `json:"kind"`
	Prompt        string                       `json:"prompt"`
	Content       string                       `json:"content"`
	SkillName     string                       `json:"skillName"`
	ExecutedSkill *WorkspaceAgentExecutedSkill `json:"executedSkill,omitempty"`
	EvidenceCount int                          `json:"evidenceCount"`
}

type WorkspaceAgentAskInput struct {
	WorkspaceID             string `json:"workspaceId"`
	DocumentID              string `json:"documentId"`
	SessionID               string `json:"sessionId"`
	Surface                 string `json:"surface"`
	SkillName               string `json:"skillName"`
	IncludeDocumentContext  bool   `json:"includeDocumentContext"`
	IncludeWorkspaceContext bool   `json:"includeWorkspaceContext"`
	Selection               string `json:"selection"`
	CurrentPage             int    `json:"currentPage"`
	ProviderID              int64  `json:"providerId"`
	ModelID                 int64  `json:"modelId"`
	Question                string `json:"question"`
}

type WorkspaceAgentAskResult struct {
	Session          WorkspaceAgentSession         `json:"session"`
	UserMessage      WorkspaceAgentMessage         `json:"userMessage"`
	AssistantMessage WorkspaceAgentMessage         `json:"assistantMessage"`
	ExecutedSkill    WorkspaceAgentExecutedSkill   `json:"executedSkill"`
	Query            WorkspaceKnowledgeQueryResult `json:"query"`
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

type WorkspaceWikiScanJobStatus string

const (
	WorkspaceWikiScanJobQueued    WorkspaceWikiScanJobStatus = "queued"
	WorkspaceWikiScanJobRunning   WorkspaceWikiScanJobStatus = "running"
	WorkspaceWikiScanJobCompleted WorkspaceWikiScanJobStatus = "completed"
	WorkspaceWikiScanJobFailed    WorkspaceWikiScanJobStatus = "failed"
	WorkspaceWikiScanJobCancelled WorkspaceWikiScanJobStatus = "cancelled"
)

type WorkspaceWikiPageKind string

const (
	WorkspaceWikiPageIndex         WorkspaceWikiPageKind = "index"
	WorkspaceWikiPageOverview      WorkspaceWikiPageKind = "overview"
	WorkspaceWikiPageOpenQuestions WorkspaceWikiPageKind = "open_questions"
	WorkspaceWikiPageLog           WorkspaceWikiPageKind = "log"
	WorkspaceWikiPageDocument      WorkspaceWikiPageKind = "document"
	WorkspaceWikiPageConcept       WorkspaceWikiPageKind = "concept"
)

type WorkspaceWikiScanStartInput struct {
	WorkspaceID string `json:"workspaceId"`
	DocumentID  string `json:"documentId"`
	ProviderID  int64  `json:"providerId"`
	ModelID     int64  `json:"modelId"`
}

type WorkspaceWikiScanJob struct {
	JobID           string                     `json:"jobId"`
	WorkspaceID     string                     `json:"workspaceId"`
	DocumentID      string                     `json:"documentId"`
	Status          WorkspaceWikiScanJobStatus `json:"status"`
	TotalItems      int                        `json:"totalItems"`
	ProcessedItems  int                        `json:"processedItems"`
	FailedItems     int                        `json:"failedItems"`
	CurrentItem     string                     `json:"currentItem"`
	CurrentStage    string                     `json:"currentStage"`
	Message         string                     `json:"message"`
	OverallProgress float64                    `json:"overallProgress"`
	ProviderID      int64                      `json:"providerId"`
	ModelID         int64                      `json:"modelId"`
	Error           string                     `json:"error,omitempty"`
	StartedAt       string                     `json:"startedAt"`
	UpdatedAt       string                     `json:"updatedAt"`
	FinishedAt      string                     `json:"finishedAt,omitempty"`
}

type WorkspaceWikiPage struct {
	ID               string                `json:"id"`
	WorkspaceID      string                `json:"workspaceId"`
	SourceDocumentID string                `json:"sourceDocumentId"`
	Title            string                `json:"title"`
	Slug             string                `json:"slug"`
	Kind             WorkspaceWikiPageKind `json:"kind"`
	MarkdownPath     string                `json:"markdownPath"`
	Summary          string                `json:"summary"`
	CreatedAt        string                `json:"createdAt"`
	UpdatedAt        string                `json:"updatedAt"`
}

type WorkspaceWikiPageContent struct {
	Page     WorkspaceWikiPage `json:"page"`
	Markdown string            `json:"markdown"`
}

type WorkspaceWikiJobEvent struct {
	JobID   string               `json:"jobId"`
	Type    string               `json:"type"`
	Status  WorkspaceWikiScanJob `json:"status"`
	Message string               `json:"message,omitempty"`
	Error   string               `json:"error,omitempty"`
}

type WorkspaceWikiScanSource struct {
	Key          string
	Title        string
	SourceType   string
	DocumentID   string
	AbsolutePath string
	Kind         string
}

type workspaceWikiScanRunResult struct {
	Processed int
	Failed    int
}

type workspaceWikiScanJobUpdate struct {
	Status          WorkspaceWikiScanJobStatus
	ProcessedItems  int
	FailedItems     int
	CurrentItem     string
	CurrentStage    string
	Message         string
	OverallProgress float64
	Error           string
	Finished        bool
}

type workspaceWikiPageRecord struct {
	Page     WorkspaceWikiPage
	Markdown string
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
