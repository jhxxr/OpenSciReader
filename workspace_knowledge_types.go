package main

type WorkspaceKnowledgeSource struct {
	ID           string `json:"sourceId"`
	WorkspaceID  string `json:"workspaceId"`
	Title        string `json:"title"`
	Slug         string `json:"slug"`
	Kind         string `json:"kind"`
	AbsolutePath string `json:"absolutePath"`
	ContentHash  string `json:"contentHash"`
	ExtractPath  string `json:"extractPath"`
	DocumentID   string `json:"documentId"`
	Status       string `json:"status"`
	LastScanAt   string `json:"lastScanAt"`
	LastError    string `json:"lastError"`
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
