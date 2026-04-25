package main

type WorkspaceKnowledgeSource struct {
	ID               string `json:"sourceId"`
	WorkspaceID      string `json:"workspaceId"`
	Title            string `json:"title"`
	Slug             string `json:"slug"`
	Kind             string `json:"kind"`
	SourcePath       string `json:"sourcePath"`
	MarkItDownPath   string `json:"markItDownPath"`
	MarkItDownStatus string `json:"markItDownStatus"`
	ExtractStatus    string `json:"extractStatus"`
	LastIngestAt     string `json:"lastIngestAt"`
	LastSuccessAt    string `json:"lastSuccessAt"`
	LastError        string `json:"lastError"`
	AbsolutePath     string `json:"absolutePath,omitempty"`
	ContentHash      string `json:"contentHash"`
	ExtractPath      string `json:"extractPath,omitempty"`
	DocumentID       string `json:"documentId"`
}

type WorkspaceKnowledgeCompileSummary struct {
	WorkspaceID       string   `json:"workspaceId"`
	StartedAt         string   `json:"startedAt"`
	FinishedAt        string   `json:"finishedAt"`
	IncludedSourceIDs []string `json:"includedSourceIds"`
	FailedSourceIDs   []string `json:"failedSourceIds"`
	UpdatedWikiPaths  []string `json:"updatedWikiPaths"`
	CompileDirty      bool     `json:"compileDirty"`
	WikiDirty         bool     `json:"wikiDirty"`
}

type WorkspaceKnowledgeSourceRef struct {
	SourceID  string `json:"sourceId"`
	PageStart int    `json:"pageStart"`
	PageEnd   int    `json:"pageEnd"`
	Excerpt   string `json:"excerpt"`
}

type WorkspaceKnowledgeEntity struct {
	ID          string                        `json:"id"`
	WorkspaceID string                        `json:"workspaceId"`
	Title       string                        `json:"title"`
	Type        string                        `json:"type"`
	Summary     string                        `json:"summary"`
	Aliases     []string                      `json:"aliases"`
	SourceRefs  []WorkspaceKnowledgeSourceRef `json:"sourceRefs"`
	Origin      string                        `json:"origin"`
	Status      string                        `json:"status"`
	Confidence  float64                       `json:"confidence"`
	CreatedAt   string                        `json:"createdAt"`
	UpdatedAt   string                        `json:"updatedAt"`
}

type WorkspaceKnowledgeClaim struct {
	ID          string                        `json:"id"`
	WorkspaceID string                        `json:"workspaceId"`
	Title       string                        `json:"title"`
	Type        string                        `json:"type"`
	Summary     string                        `json:"summary"`
	EntityIDs   []string                      `json:"entityIds"`
	SourceRefs  []WorkspaceKnowledgeSourceRef `json:"sourceRefs"`
	Origin      string                        `json:"origin"`
	Status      string                        `json:"status"`
	Confidence  float64                       `json:"confidence"`
	CreatedAt   string                        `json:"createdAt"`
	UpdatedAt   string                        `json:"updatedAt"`
}

type WorkspaceKnowledgeRelation struct {
	ID          string                        `json:"id"`
	WorkspaceID string                        `json:"workspaceId"`
	Type        string                        `json:"type"`
	FromID      string                        `json:"fromId"`
	ToID        string                        `json:"toId"`
	Summary     string                        `json:"summary"`
	SourceRefs  []WorkspaceKnowledgeSourceRef `json:"sourceRefs"`
	Origin      string                        `json:"origin"`
	Status      string                        `json:"status"`
	Confidence  float64                       `json:"confidence"`
	CreatedAt   string                        `json:"createdAt"`
	UpdatedAt   string                        `json:"updatedAt"`
}

type WorkspaceKnowledgeTask struct {
	ID          string                        `json:"id"`
	WorkspaceID string                        `json:"workspaceId"`
	Title       string                        `json:"title"`
	Type        string                        `json:"type"`
	Summary     string                        `json:"summary"`
	Priority    string                        `json:"priority"`
	SourceRefs  []WorkspaceKnowledgeSourceRef `json:"sourceRefs"`
	Origin      string                        `json:"origin"`
	Status      string                        `json:"status"`
	Confidence  float64                       `json:"confidence"`
	CreatedAt   string                        `json:"createdAt"`
	UpdatedAt   string                        `json:"updatedAt"`
}

type WorkspaceKnowledgeBySourcePayload struct {
	Source    WorkspaceKnowledgeSource     `json:"source"`
	Entities  []WorkspaceKnowledgeEntity   `json:"entities"`
	Claims    []WorkspaceKnowledgeClaim    `json:"claims"`
	Relations []WorkspaceKnowledgeRelation `json:"relations"`
	Tasks     []WorkspaceKnowledgeTask     `json:"tasks"`
}

type WorkspaceKnowledgeScanRunRecord struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspaceId"`
	Status      string   `json:"status"`
	StartedAt   string   `json:"startedAt"`
	FinishedAt  string   `json:"finishedAt"`
	SourceIDs   []string `json:"sourceIds"`
	Message     string   `json:"message"`
}
