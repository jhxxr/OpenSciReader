package translator

type Mode string

const (
	ModePreview Mode = "preview"
	ModeExport  Mode = "export"
)

type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

type ProviderConfig struct {
	ProviderID   int64  `json:"providerId"`
	ProviderName string `json:"providerName"`
	BaseURL      string `json:"baseUrl"`
	APIKey       string `json:"apiKey"`
	ModelID      string `json:"modelId"`
}

type StartRequest struct {
	PDFPath            string         `json:"pdfPath"`
	PageCount          int            `json:"pageCount"`
	ItemID             string         `json:"itemId"`
	ItemTitle          string         `json:"itemTitle"`
	SourceLang         string         `json:"sourceLang"`
	TargetLang         string         `json:"targetLang"`
	Mode               Mode           `json:"mode"`
	PreviewChunkPages  int            `json:"previewChunkPages"`
	MaxPagesPerPart    int            `json:"maxPagesPerPart"`
	QPS                int            `json:"qps"`
	PoolMaxWorkers     int            `json:"poolMaxWorkers"`
	TermPoolMaxWorkers int            `json:"termPoolMaxWorkers"`
	RetryOfJobID       string         `json:"retryJobId"`
	ReusePreviewJobID  string         `json:"reusePreviewJobId"`
	Provider           ProviderConfig `json:"provider"`
}

type JobOutputs struct {
	OriginalPDFPath           string  `json:"originalPdfPath,omitempty"`
	MonoPDFPath               string  `json:"monoPdfPath,omitempty"`
	DualPDFPath               string  `json:"dualPdfPath,omitempty"`
	NoWatermarkMonoPDFPath    string  `json:"noWatermarkMonoPdfPath,omitempty"`
	NoWatermarkDualPDFPath    string  `json:"noWatermarkDualPdfPath,omitempty"`
	AutoExtractedGlossaryPath string  `json:"autoExtractedGlossaryPath,omitempty"`
	TotalSeconds              float64 `json:"totalSeconds,omitempty"`
	PeakMemoryUsage           float64 `json:"peakMemoryUsage,omitempty"`
}

type ChunkStatus struct {
	Index                int       `json:"index"`
	StartPage            int       `json:"startPage"`
	EndPage              int       `json:"endPage"`
	Status               JobStatus `json:"status"`
	TranslatedPDFPath    string    `json:"translatedPdfPath,omitempty"`
	DualPDFPath          string    `json:"dualPdfPath,omitempty"`
	TranslatedPageOffset int       `json:"translatedPageOffset,omitempty"`
	StartedAt            string    `json:"startedAt,omitempty"`
	FinishedAt           string    `json:"finishedAt,omitempty"`
	TotalSeconds         float64   `json:"totalSeconds,omitempty"`
	Error                string    `json:"error,omitempty"`
}

type StageSummaryItem struct {
	Name    string  `json:"name"`
	Percent float64 `json:"percent"`
}

type JobSnapshot struct {
	JobID              string        `json:"jobId"`
	RetryOfJobID       string        `json:"retryOfJobId,omitempty"`
	Mode               Mode          `json:"mode"`
	Status             JobStatus     `json:"status"`
	ItemID             string        `json:"itemId,omitempty"`
	ItemTitle          string        `json:"itemTitle,omitempty"`
	PDFPath            string        `json:"pdfPath"`
	LocalPDFPath       string        `json:"localPdfPath"`
	PageCount          int           `json:"pageCount"`
	SourceLang         string        `json:"sourceLang"`
	TargetLang         string        `json:"targetLang"`
	PreviewChunkPages  int           `json:"previewChunkPages"`
	MaxPagesPerPart    int           `json:"maxPagesPerPart"`
	QPS                int           `json:"qps"`
	PoolMaxWorkers     int           `json:"poolMaxWorkers"`
	TermPoolMaxWorkers int           `json:"termPoolMaxWorkers"`
	ProviderID         int64         `json:"providerId"`
	ProviderName       string        `json:"providerName"`
	ModelID            string        `json:"modelId"`
	CreatedAt          string        `json:"createdAt"`
	UpdatedAt          string        `json:"updatedAt"`
	StartedAt          string        `json:"startedAt,omitempty"`
	FinishedAt         string        `json:"finishedAt,omitempty"`
	CurrentStage       string        `json:"currentStage,omitempty"`
	OverallProgress    float64       `json:"overallProgress,omitempty"`
	StageProgress      float64       `json:"stageProgress,omitempty"`
	StageCurrent       int           `json:"stageCurrent,omitempty"`
	StageTotal         int           `json:"stageTotal,omitempty"`
	PartIndex          int           `json:"partIndex,omitempty"`
	TotalParts         int           `json:"totalParts,omitempty"`
	Error              string        `json:"error,omitempty"`
	Outputs            JobOutputs    `json:"outputs"`
	Chunks             []ChunkStatus `json:"chunks"`
}

type JobEvent struct {
	Sequence        int64              `json:"sequence"`
	JobID           string             `json:"jobId"`
	Mode            Mode               `json:"mode"`
	Type            string             `json:"type"`
	Timestamp       string             `json:"timestamp"`
	JobStatus       JobStatus          `json:"jobStatus"`
	Message         string             `json:"message,omitempty"`
	Error           string             `json:"error,omitempty"`
	ErrorType       string             `json:"errorType,omitempty"`
	Details         string             `json:"details,omitempty"`
	Stage           string             `json:"stage,omitempty"`
	StageProgress   float64            `json:"stageProgress,omitempty"`
	OverallProgress float64            `json:"overallProgress,omitempty"`
	StageCurrent    int                `json:"stageCurrent,omitempty"`
	StageTotal      int                `json:"stageTotal,omitempty"`
	PartIndex       int                `json:"partIndex,omitempty"`
	TotalParts      int                `json:"totalParts,omitempty"`
	Stages          []StageSummaryItem `json:"stages,omitempty"`
	Chunk           *ChunkStatus       `json:"chunk,omitempty"`
	Output          *JobOutputs        `json:"output,omitempty"`
	Status          *JobSnapshot       `json:"status,omitempty"`
}
