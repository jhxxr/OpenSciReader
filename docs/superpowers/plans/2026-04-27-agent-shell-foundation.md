# Agent Shell Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first working slice of the session-first agent shell by adding shared workspace sessions, a backend agent service that reuses the existing workspace knowledge query flow, a Codex-style workspace shell, and reader/workspace convergence on the same session system.

**Architecture:** Keep the existing workspace wiki build and query pipeline, but stop treating it as the product shell. Add a thin `workspaceAgentService` over the current knowledge query service, persist sessions and thread messages in SQLite, and replace the current workspace tab grid with a session-first shell that routes both workspace asks and reader asks through the same agent/session layer.

**Tech Stack:** Go, SQLite, Wails v2, React 18, TypeScript, Zustand.

---

## Scope Split

The approved spec spans several independent subsystems. This document is **Plan 1: shared agent shell foundation**.

It intentionally does **not** implement the entire spec in one pass. Follow-on plans should cover:

1. richer wiki memory maintenance and lint flows
2. an explicit skill registry with autonomous routing rules
3. deeper knowledge-mode browsing and promoted insight management

This foundation plan must still produce working, testable software on its own.

## File Map

### Backend files

- Modify: `config_types.go`
  Add workspace agent session and message types plus ask input/result contracts.
- Modify: `config_store.go`
  Add SQLite tables and CRUD methods for workspace agent sessions and messages.
- Create: `workspace_agent_service.go`
  Add the shared agent service that persists thread messages and delegates factual grounding to `workspaceKnowledgeQueryService`.
- Create: `workspace_agent_service_test.go`
  Test session creation, message persistence, and query delegation.
- Modify: `app.go`
  Instantiate the agent service, expose Wails methods, and keep the existing wiki service/query service available behind the new shell.

### Frontend files

- Create: `frontend/src/types/workspaceAgent.ts`
  TypeScript contracts for sessions, messages, and ask results.
- Create: `frontend/src/api/workspaceAgent.ts`
  Wails bridge for workspace agent session and ask APIs.
- Create: `frontend/src/store/workspaceAgentStore.ts`
  Zustand store keyed by workspace for sessions, selected session, messages, and ask state.
- Create: `frontend/src/components/WorkspaceAgentShell.tsx`
  Left rail + center thread + right context pane shell for workspace sessions.
- Create: `frontend/src/components/WorkspaceKnowledgeMode.tsx`
  Move the existing knowledge build/wiki browsing UI behind the new `Knowledge` mode.
- Modify: `frontend/src/components/WorkspaceTab.tsx`
  Turn the old page grid into a mode switcher that hosts `WorkspaceAgentShell` and `WorkspaceKnowledgeMode`.
- Modify: `frontend/src/components/ReaderAIPanel.tsx`
  Route asks through the shared agent/session API instead of the direct knowledge query API.
- Modify: `frontend/src/components/ReaderTab.tsx`
  Pass workspace/session information into the reader AI panel.
- Modify: `frontend/src/store/tabStore.ts`
  Add optional `agentSessionId` support for document tabs.
- Modify: `frontend/src/App.tsx`
  Load session state when workspace tabs open and thread it into workspace and reader surfaces.
- Modify: `frontend/src/style/workspace.css`
  Add shell layout and panel styling for the new workspace surface.
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.d.ts`
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.js`
- Modify via `wails generate module`: `frontend/wailsjs/go/models.ts`

## Task 1: Persist Shared Agent Sessions And Messages In The Backend

**Files:**
- Modify: `config_types.go`
- Modify: `config_store.go`
- Create: `workspace_agent_service.go`
- Create: `workspace_agent_service_test.go`

- [ ] **Step 1: Write the failing backend tests**

```go
package main

import (
	"context"
	"testing"
)

type stubWorkspaceAgentQuery struct {
	inputs []WorkspaceKnowledgeQueryInput
	result WorkspaceKnowledgeQueryResult
	err   error
}

func (s *stubWorkspaceAgentQuery) Query(_ context.Context, input WorkspaceKnowledgeQueryInput) (WorkspaceKnowledgeQueryResult, error) {
	s.inputs = append(s.inputs, input)
	return s.result, s.err
}

func (s *stubWorkspaceAgentQuery) Promote(_ context.Context, _ WorkspaceKnowledgePromotionInput) error {
	return nil
}

func TestWorkspaceAgentServiceAskCreatesSessionAndMessages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	workspace, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{Name: "Agent Workspace"})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	query := &stubWorkspaceAgentQuery{result: WorkspaceKnowledgeQueryResult{
		Answer: "Grounded answer.",
		Evidence: []WorkspaceKnowledgeEvidenceHit{{
			Kind:  "wiki_page",
			ID:    "wiki:overview",
			Title: "Overview",
		}},
	}}
	service := newWorkspaceAgentService(store, query)

	result, err := service.Ask(ctx, WorkspaceAgentAskInput{
		WorkspaceID: workspace.ID,
		Surface:     string(WorkspaceAgentSurfaceWorkspace),
		Question:    "What changed in this workspace?",
	})
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if result.Session.ID == "" {
		t.Fatal("Ask() returned empty session id")
	}
	if len(query.inputs) != 1 {
		t.Fatalf("len(query.inputs) = %d, want 1", len(query.inputs))
	}

	messages, err := store.ListWorkspaceAgentMessages(ctx, result.Session.ID)
	if err != nil {
		t.Fatalf("ListWorkspaceAgentMessages() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(messages))
	}
	if messages[0].Role != string(WorkspaceAgentMessageRoleUser) {
		t.Fatalf("messages[0].Role = %q, want %q", messages[0].Role, WorkspaceAgentMessageRoleUser)
	}
	if messages[1].Role != string(WorkspaceAgentMessageRoleAssistant) {
		t.Fatalf("messages[1].Role = %q, want %q", messages[1].Role, WorkspaceAgentMessageRoleAssistant)
	}
	if messages[1].EvidenceCount != 1 {
		t.Fatalf("messages[1].EvidenceCount = %d, want 1", messages[1].EvidenceCount)
	}
	if messages[1].Content != "Grounded answer." {
		t.Fatalf("messages[1].Content = %q, want %q", messages[1].Content, "Grounded answer.")
	}
}

func TestListWorkspaceAgentSessionsSortsNewestFirst(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	workspace, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{Name: "Agent Workspace"})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	older, err := store.CreateWorkspaceAgentSession(ctx, WorkspaceAgentSessionCreateInput{
		WorkspaceID: workspace.ID,
		Title:       "Older session",
		Surface:     string(WorkspaceAgentSurfaceWorkspace),
	})
	if err != nil {
		t.Fatalf("CreateWorkspaceAgentSession(older) error = %v", err)
	}
	if _, err := store.AppendWorkspaceAgentMessage(ctx, WorkspaceAgentMessageCreateInput{
		SessionID:  older.ID,
		WorkspaceID: workspace.ID,
		Surface:    string(WorkspaceAgentSurfaceWorkspace),
		Role:       string(WorkspaceAgentMessageRoleUser),
		Kind:       "message",
		Content:    "older",
	}); err != nil {
		t.Fatalf("AppendWorkspaceAgentMessage(older) error = %v", err)
	}

	newer, err := store.CreateWorkspaceAgentSession(ctx, WorkspaceAgentSessionCreateInput{
		WorkspaceID: workspace.ID,
		Title:       "Newer session",
		Surface:     string(WorkspaceAgentSurfaceWorkspace),
	})
	if err != nil {
		t.Fatalf("CreateWorkspaceAgentSession(newer) error = %v", err)
	}
	if _, err := store.AppendWorkspaceAgentMessage(ctx, WorkspaceAgentMessageCreateInput{
		SessionID:  newer.ID,
		WorkspaceID: workspace.ID,
		Surface:    string(WorkspaceAgentSurfaceWorkspace),
		Role:       string(WorkspaceAgentMessageRoleUser),
		Kind:       "message",
		Content:    "newer",
	}); err != nil {
		t.Fatalf("AppendWorkspaceAgentMessage(newer) error = %v", err)
	}

	sessions, err := store.ListWorkspaceAgentSessions(ctx, workspace.ID)
	if err != nil {
		t.Fatalf("ListWorkspaceAgentSessions() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(sessions))
	}
	if sessions[0].ID != newer.ID {
		t.Fatalf("sessions[0].ID = %q, want %q", sessions[0].ID, newer.ID)
	}
}
```

- [ ] **Step 2: Run the backend tests to verify they fail**

Run: `go test ./... -run 'TestWorkspaceAgentServiceAskCreatesSessionAndMessages|TestListWorkspaceAgentSessionsSortsNewestFirst' -count=1`

Expected: FAIL with compile errors such as `undefined: WorkspaceAgentAskInput`, `undefined: newWorkspaceAgentService`, or missing store methods.

- [ ] **Step 3: Add agent session and message contracts**

```go
// config_types.go
type WorkspaceAgentSurface string

const (
	WorkspaceAgentSurfaceWorkspace WorkspaceAgentSurface = "workspace"
	WorkspaceAgentSurfaceReader    WorkspaceAgentSurface = "reader"
)

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
	ID            int64  `json:"id"`
	SessionID     string `json:"sessionId"`
	WorkspaceID   string `json:"workspaceId"`
	Surface       string `json:"surface"`
	Role          string `json:"role"`
	Kind          string `json:"kind"`
	Prompt        string `json:"prompt"`
	Content       string `json:"content"`
	SkillName     string `json:"skillName"`
	EvidenceCount int    `json:"evidenceCount"`
	CreatedAt     string `json:"createdAt"`
}

type WorkspaceAgentSessionCreateInput struct {
	WorkspaceID string `json:"workspaceId"`
	Title       string `json:"title"`
	Surface     string `json:"surface"`
}

type WorkspaceAgentMessageCreateInput struct {
	SessionID     string `json:"sessionId"`
	WorkspaceID   string `json:"workspaceId"`
	Surface       string `json:"surface"`
	Role          string `json:"role"`
	Kind          string `json:"kind"`
	Prompt        string `json:"prompt"`
	Content       string `json:"content"`
	SkillName     string `json:"skillName"`
	EvidenceCount int    `json:"evidenceCount"`
}

type WorkspaceAgentAskInput struct {
	WorkspaceID string `json:"workspaceId"`
	SessionID   string `json:"sessionId"`
	Surface     string `json:"surface"`
	Question    string `json:"question"`
	DocumentID  string `json:"documentId"`
	Selection   string `json:"selection"`
	CurrentPage int    `json:"currentPage"`
	ProviderID  int64  `json:"providerId"`
	ModelID     int64  `json:"modelId"`
}

type WorkspaceAgentAskResult struct {
	Session          WorkspaceAgentSession        `json:"session"`
	UserMessage      WorkspaceAgentMessage        `json:"userMessage"`
	AssistantMessage WorkspaceAgentMessage        `json:"assistantMessage"`
	Query            WorkspaceKnowledgeQueryResult `json:"query"`
}
```

- [ ] **Step 4: Add SQLite tables and store methods**

```go
// config_store.go bootstrap()
	`CREATE TABLE IF NOT EXISTS workspace_agent_sessions (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		title TEXT NOT NULL,
		surface TEXT NOT NULL DEFAULT 'workspace',
		status TEXT NOT NULL DEFAULT 'active',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
	);`,
	`CREATE TABLE IF NOT EXISTS workspace_agent_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		workspace_id TEXT NOT NULL,
		surface TEXT NOT NULL DEFAULT 'workspace',
		role TEXT NOT NULL,
		kind TEXT NOT NULL DEFAULT 'message',
		prompt TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL,
		skill_name TEXT NOT NULL DEFAULT '',
		evidence_count INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		FOREIGN KEY(session_id) REFERENCES workspace_agent_sessions(id) ON DELETE CASCADE,
		FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
	);`,
```

```go
// config_store.go
func (s *configStore) CreateWorkspaceAgentSession(ctx context.Context, input WorkspaceAgentSessionCreateInput) (WorkspaceAgentSession, error) {
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID == "" {
		return WorkspaceAgentSession{}, fmt.Errorf("workspace id is required")
	}
	if _, err := s.GetWorkspace(ctx, workspaceID); err != nil {
		return WorkspaceAgentSession{}, err
	}
	now := nowRFC3339()
	session := WorkspaceAgentSession{
		ID:          newEntityID("agent_session"),
		WorkspaceID: workspaceID,
		Title:       firstNonEmptyText(strings.TrimSpace(input.Title), "New Research Session"),
		Surface:     firstNonEmptyText(strings.TrimSpace(input.Surface), string(WorkspaceAgentSurfaceWorkspace)),
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := s.appDB.ExecContext(ctx, `
		INSERT INTO workspace_agent_sessions (id, workspace_id, title, surface, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`, session.ID, session.WorkspaceID, session.Title, session.Surface, session.Status, session.CreatedAt, session.UpdatedAt); err != nil {
		return WorkspaceAgentSession{}, fmt.Errorf("create workspace agent session: %w", err)
	}
	return session, nil
}

func (s *configStore) AppendWorkspaceAgentMessage(ctx context.Context, input WorkspaceAgentMessageCreateInput) (WorkspaceAgentMessage, error) {
	createdAt := nowRFC3339()
	result, err := s.appDB.ExecContext(ctx, `
		INSERT INTO workspace_agent_messages (session_id, workspace_id, surface, role, kind, prompt, content, skill_name, evidence_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, strings.TrimSpace(input.SessionID), strings.TrimSpace(input.WorkspaceID), firstNonEmptyText(strings.TrimSpace(input.Surface), string(WorkspaceAgentSurfaceWorkspace)), strings.TrimSpace(input.Role), firstNonEmptyText(strings.TrimSpace(input.Kind), "message"), strings.TrimSpace(input.Prompt), strings.TrimSpace(input.Content), strings.TrimSpace(input.SkillName), input.EvidenceCount, createdAt)
	if err != nil {
		return WorkspaceAgentMessage{}, fmt.Errorf("append workspace agent message: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return WorkspaceAgentMessage{}, fmt.Errorf("read workspace agent message id: %w", err)
	}
	if _, err := s.appDB.ExecContext(ctx, `
		UPDATE workspace_agent_sessions
		SET updated_at = ?
		WHERE id = ?;
	`, createdAt, strings.TrimSpace(input.SessionID)); err != nil {
		return WorkspaceAgentMessage{}, fmt.Errorf("touch workspace agent session: %w", err)
	}
	return WorkspaceAgentMessage{
		ID:            id,
		SessionID:     strings.TrimSpace(input.SessionID),
		WorkspaceID:   strings.TrimSpace(input.WorkspaceID),
		Surface:       firstNonEmptyText(strings.TrimSpace(input.Surface), string(WorkspaceAgentSurfaceWorkspace)),
		Role:          strings.TrimSpace(input.Role),
		Kind:          firstNonEmptyText(strings.TrimSpace(input.Kind), "message"),
		Prompt:        strings.TrimSpace(input.Prompt),
		Content:       strings.TrimSpace(input.Content),
		SkillName:     strings.TrimSpace(input.SkillName),
		EvidenceCount: input.EvidenceCount,
		CreatedAt:     createdAt,
	}, nil
}

func (s *configStore) ListWorkspaceAgentSessions(ctx context.Context, workspaceID string) ([]WorkspaceAgentSession, error) {
	rows, err := s.appDB.QueryContext(ctx, `
		SELECT id, workspace_id, title, surface, status, created_at, updated_at
		FROM workspace_agent_sessions
		WHERE workspace_id = ?
		ORDER BY updated_at DESC, created_at DESC, id DESC;
	`, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("list workspace agent sessions: %w", err)
	}
	defer rows.Close()
	sessions := []WorkspaceAgentSession{}
	for rows.Next() {
		var session WorkspaceAgentSession
		if err := rows.Scan(&session.ID, &session.WorkspaceID, &session.Title, &session.Surface, &session.Status, &session.CreatedAt, &session.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace agent session: %w", err)
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (s *configStore) ListWorkspaceAgentMessages(ctx context.Context, sessionID string) ([]WorkspaceAgentMessage, error) {
	rows, err := s.appDB.QueryContext(ctx, `
		SELECT id, session_id, workspace_id, surface, role, kind, prompt, content, skill_name, evidence_count, created_at
		FROM workspace_agent_messages
		WHERE session_id = ?
		ORDER BY id ASC;
	`, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, fmt.Errorf("list workspace agent messages: %w", err)
	}
	defer rows.Close()
	messages := []WorkspaceAgentMessage{}
	for rows.Next() {
		var message WorkspaceAgentMessage
		if err := rows.Scan(&message.ID, &message.SessionID, &message.WorkspaceID, &message.Surface, &message.Role, &message.Kind, &message.Prompt, &message.Content, &message.SkillName, &message.EvidenceCount, &message.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace agent message: %w", err)
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}
```

- [ ] **Step 5: Create the shared workspace agent service**

```go
// workspace_agent_service.go
package main

import (
	"context"
	"fmt"
	"strings"
)

type workspaceAgentQuery interface {
	Query(ctx context.Context, input WorkspaceKnowledgeQueryInput) (WorkspaceKnowledgeQueryResult, error)
	Promote(ctx context.Context, input WorkspaceKnowledgePromotionInput) error
}

type workspaceAgentService struct {
	store *configStore
	query workspaceAgentQuery
}

func newWorkspaceAgentService(store *configStore, query workspaceAgentQuery) *workspaceAgentService {
	return &workspaceAgentService{store: store, query: query}
}

func (s *workspaceAgentService) Ask(ctx context.Context, input WorkspaceAgentAskInput) (WorkspaceAgentAskResult, error) {
	if s.store == nil {
		return WorkspaceAgentAskResult{}, fmt.Errorf("config store is unavailable")
	}
	if s.query == nil {
		return WorkspaceAgentAskResult{}, fmt.Errorf("workspace agent query service is unavailable")
	}
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	question := strings.TrimSpace(input.Question)
	if workspaceID == "" {
		return WorkspaceAgentAskResult{}, fmt.Errorf("workspace id is required")
	}
	if question == "" {
		return WorkspaceAgentAskResult{}, fmt.Errorf("question is required")
	}

	sessionID := strings.TrimSpace(input.SessionID)
	var session WorkspaceAgentSession
	var err error
	if sessionID == "" {
		session, err = s.store.CreateWorkspaceAgentSession(ctx, WorkspaceAgentSessionCreateInput{
			WorkspaceID: workspaceID,
			Title:       question,
			Surface:     firstNonEmptyText(strings.TrimSpace(input.Surface), string(WorkspaceAgentSurfaceWorkspace)),
		})
		if err != nil {
			return WorkspaceAgentAskResult{}, err
		}
	} else {
		sessions, err := s.store.ListWorkspaceAgentSessions(ctx, workspaceID)
		if err != nil {
			return WorkspaceAgentAskResult{}, err
		}
		for _, candidate := range sessions {
			if candidate.ID == sessionID {
				session = candidate
				break
			}
		}
		if session.ID == "" {
			return WorkspaceAgentAskResult{}, fmt.Errorf("workspace agent session %s not found", sessionID)
		}
	}

	userMessage, err := s.store.AppendWorkspaceAgentMessage(ctx, WorkspaceAgentMessageCreateInput{
		SessionID:   session.ID,
		WorkspaceID: workspaceID,
		Surface:     firstNonEmptyText(strings.TrimSpace(input.Surface), session.Surface),
		Role:        string(WorkspaceAgentMessageRoleUser),
		Kind:        "question",
		Prompt:      question,
		Content:     question,
	})
	if err != nil {
		return WorkspaceAgentAskResult{}, err
	}

	queryResult, err := s.query.Query(ctx, WorkspaceKnowledgeQueryInput{
		WorkspaceID: workspaceID,
		ProviderID:  input.ProviderID,
		ModelID:     input.ModelID,
		Question:    question,
	})
	if err != nil {
		return WorkspaceAgentAskResult{}, err
	}

	assistantMessage, err := s.store.AppendWorkspaceAgentMessage(ctx, WorkspaceAgentMessageCreateInput{
		SessionID:     session.ID,
		WorkspaceID:   workspaceID,
		Surface:       firstNonEmptyText(strings.TrimSpace(input.Surface), session.Surface),
		Role:          string(WorkspaceAgentMessageRoleAssistant),
		Kind:          "answer",
		Prompt:        question,
		Content:       queryResult.Answer,
		SkillName:     "ask_with_evidence",
		EvidenceCount: len(queryResult.Evidence),
	})
	if err != nil {
		return WorkspaceAgentAskResult{}, err
	}

	return WorkspaceAgentAskResult{
		Session:          session,
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
		Query:            queryResult,
	}, nil
}
```

- [ ] **Step 6: Run the backend tests to verify they pass**

Run: `go test ./... -run 'TestWorkspaceAgentServiceAskCreatesSessionAndMessages|TestListWorkspaceAgentSessionsSortsNewestFirst' -count=1`

Expected: PASS

- [ ] **Step 7: Commit the backend session foundation**

```bash
git add config_types.go config_store.go workspace_agent_service.go workspace_agent_service_test.go
git commit -m "feat: add workspace agent session persistence"
```

## Task 2: Replace The Workspace Grid With A Session-First Agent Shell

**Files:**
- Modify: `app.go`
- Create: `frontend/src/types/workspaceAgent.ts`
- Create: `frontend/src/api/workspaceAgent.ts`
- Create: `frontend/src/store/workspaceAgentStore.ts`
- Create: `frontend/src/components/WorkspaceAgentShell.tsx`
- Create: `frontend/src/components/WorkspaceKnowledgeMode.tsx`
- Modify: `frontend/src/components/WorkspaceTab.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/style/workspace.css`
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.d.ts`
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.js`
- Modify via `wails generate module`: `frontend/wailsjs/go/models.ts`

- [ ] **Step 1: Write the failing workspace-shell build**

```tsx
// frontend/src/components/WorkspaceTab.tsx
import { WorkspaceAgentShell } from './WorkspaceAgentShell';
import { WorkspaceKnowledgeMode } from './WorkspaceKnowledgeMode';

const [mode, setMode] = useState<'sessions' | 'knowledge'>('sessions');

return mode === 'sessions' ? (
  <WorkspaceAgentShell
    workspace={workspace}
    sessions={agentSessions}
    activeSessionId={activeSessionId}
    messages={activeMessages}
    isLoading={isLoadingAgent}
    isAsking={isAskingAgent}
    error={agentError}
    onCreateSession={onCreateSession}
    onSelectSession={onSelectSession}
    onAsk={onAskAgent}
    onSwitchMode={() => setMode('knowledge')}
  />
) : (
  <WorkspaceKnowledgeMode
    workspace={workspace}
    documents={documents}
    wikiPages={wikiPages}
    selectedWikiPageId={selectedWikiPageId}
    wikiPageContent={wikiPageContent}
    activeWikiJob={activeWikiJob}
    wikiSources={wikiSources}
    compileSummary={compileSummary}
    onSwitchMode={() => setMode('sessions')}
    onStartWikiScan={onStartWikiScan}
    onCancelWikiScan={onCancelWikiScan}
    onRefreshWikiPages={onRefreshWikiPages}
    onSelectWikiPage={onSelectWikiPage}
    onDeleteWikiPages={onDeleteWikiPages}
    onImportFiles={onImportFiles}
    onRefreshDocuments={onRefreshDocuments}
    onOpenPdf={onOpenPdf}
    onDeleteDocument={onDeleteDocument}
    onChangeWikiScanModel={onChangeWikiScanModel}
  />
);
```

- [ ] **Step 2: Run the frontend build to verify it fails**

Run: `npm run build`

Workdir: `frontend`

Expected: FAIL with module resolution or missing prop errors for `WorkspaceAgentShell`, `WorkspaceKnowledgeMode`, and the new agent state props.

- [ ] **Step 3: Expose workspace agent methods through Wails and create the frontend contracts**

```go
// app.go
type App struct {
	ctx        context.Context
	store      *configStore
	paths      appPaths
	zotero     *zoteroService
	gateway    *gatewayService
	pdf        *pdfService
	translator *translator.Manager
	wiki       *workspaceWikiService
	query      *workspaceKnowledgeQueryService
	agent      *workspaceAgentService
}

func (a *App) startup(ctx context.Context) {
	...
	a.query = newWorkspaceKnowledgeQueryService(paths, a.gateway)
	a.agent = newWorkspaceAgentService(store, a.query)
}

func (a *App) ListWorkspaceAgentSessions(workspaceID string) ([]WorkspaceAgentSession, error) {
	return a.store.ListWorkspaceAgentSessions(a.ctx, workspaceID)
}

func (a *App) CreateWorkspaceAgentSession(input WorkspaceAgentSessionCreateInput) (WorkspaceAgentSession, error) {
	return a.store.CreateWorkspaceAgentSession(a.ctx, input)
}

func (a *App) ListWorkspaceAgentMessages(sessionID string) ([]WorkspaceAgentMessage, error) {
	return a.store.ListWorkspaceAgentMessages(a.ctx, sessionID)
}

func (a *App) AskWorkspaceAgent(input WorkspaceAgentAskInput) (WorkspaceAgentAskResult, error) {
	if a.agent == nil {
		return WorkspaceAgentAskResult{}, fmt.Errorf("workspace agent service is unavailable")
	}
	return a.agent.Ask(a.ctx, input)
}
```

```ts
// frontend/src/types/workspaceAgent.ts
import type { WorkspaceKnowledgeQueryResult } from '../api/workspaceKnowledge';

export interface WorkspaceAgentSession {
  id: string;
  workspaceId: string;
  title: string;
  surface: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface WorkspaceAgentMessage {
  id: number;
  sessionId: string;
  workspaceId: string;
  surface: string;
  role: 'user' | 'assistant';
  kind: string;
  prompt: string;
  content: string;
  skillName: string;
  evidenceCount: number;
  createdAt: string;
}

export interface WorkspaceAgentAskInput {
  workspaceId: string;
  sessionId?: string;
  surface: 'workspace' | 'reader';
  question: string;
  documentId?: string;
  selection?: string;
  currentPage?: number;
  providerId: number;
  modelId: number;
}

export interface WorkspaceAgentAskResult {
  session: WorkspaceAgentSession;
  userMessage: WorkspaceAgentMessage;
  assistantMessage: WorkspaceAgentMessage;
  query: WorkspaceKnowledgeQueryResult;
}
```

```ts
// frontend/src/api/workspaceAgent.ts
import type {
  WorkspaceAgentAskInput,
  WorkspaceAgentAskResult,
  WorkspaceAgentMessage,
  WorkspaceAgentSession,
} from '../types/workspaceAgent';

interface WailsWorkspaceAgentApp {
  ListWorkspaceAgentSessions: (workspaceId: string) => Promise<WorkspaceAgentSession[]>;
  CreateWorkspaceAgentSession: (input: { workspaceId: string; title: string; surface: string }) => Promise<WorkspaceAgentSession>;
  ListWorkspaceAgentMessages: (sessionId: string) => Promise<WorkspaceAgentMessage[]>;
  AskWorkspaceAgent: (input: WorkspaceAgentAskInput) => Promise<WorkspaceAgentAskResult>;
}

function isWailsApp(value: unknown): value is { go: { main: { App: WailsWorkspaceAgentApp } } } {
  return typeof value === 'object' && value !== null && 'go' in value;
}

function getApp(): WailsWorkspaceAgentApp | null {
  if (typeof window !== 'undefined' && isWailsApp(window) && window.go?.main?.App) {
    return window.go.main.App;
  }
  return null;
}

export const workspaceAgentApi = {
  async listSessions(workspaceId: string): Promise<WorkspaceAgentSession[]> {
    const app = getApp();
    if (!app || workspaceId.trim() === '') {
      return [];
    }
    return app.ListWorkspaceAgentSessions(workspaceId);
  },

  async createSession(workspaceId: string, title: string, surface: 'workspace' | 'reader'): Promise<WorkspaceAgentSession> {
    const app = getApp();
    if (!app) {
      throw new Error('Workspace agent desktop bridge is unavailable');
    }
    return app.CreateWorkspaceAgentSession({ workspaceId, title, surface });
  },

  async listMessages(sessionId: string): Promise<WorkspaceAgentMessage[]> {
    const app = getApp();
    if (!app || sessionId.trim() === '') {
      return [];
    }
    return app.ListWorkspaceAgentMessages(sessionId);
  },

  async ask(input: WorkspaceAgentAskInput): Promise<WorkspaceAgentAskResult> {
    const app = getApp();
    if (!app) {
      throw new Error('Workspace agent desktop bridge is unavailable');
    }
    return app.AskWorkspaceAgent(input);
  },
};
```

```ts
// frontend/src/store/workspaceAgentStore.ts
import { create } from 'zustand';
import { workspaceAgentApi } from '../api/workspaceAgent';
import type { WorkspaceAgentAskInput, WorkspaceAgentAskResult, WorkspaceAgentMessage, WorkspaceAgentSession } from '../types/workspaceAgent';

interface WorkspaceAgentPaneState {
  sessions: WorkspaceAgentSession[];
  activeSessionId: string | null;
  messagesBySession: Record<string, WorkspaceAgentMessage[]>;
  isLoading: boolean;
  isAsking: boolean;
  error: string | null;
}

interface WorkspaceAgentStore {
  panes: Record<string, WorkspaceAgentPaneState>;
  ensureWorkspace: (workspaceId: string) => Promise<void>;
  createSession: (workspaceId: string, title: string, surface: 'workspace' | 'reader') => Promise<WorkspaceAgentSession>;
  loadMessages: (workspaceId: string, sessionId: string) => Promise<void>;
  ask: (input: WorkspaceAgentAskInput) => Promise<WorkspaceAgentAskResult>;
}

export const useWorkspaceAgentStore = create<WorkspaceAgentStore>((set, get) => ({
  panes: {},

  async ensureWorkspace(workspaceId) {
    set((state) => ({
      panes: {
        ...state.panes,
        [workspaceId]: {
          sessions: state.panes[workspaceId]?.sessions ?? [],
          activeSessionId: state.panes[workspaceId]?.activeSessionId ?? null,
          messagesBySession: state.panes[workspaceId]?.messagesBySession ?? {},
          isLoading: true,
          isAsking: state.panes[workspaceId]?.isAsking ?? false,
          error: null,
        },
      },
    }));
    try {
      const sessions = await workspaceAgentApi.listSessions(workspaceId);
      const activeSessionId = sessions[0]?.id ?? null;
      const messages = activeSessionId ? await workspaceAgentApi.listMessages(activeSessionId) : [];
      set((state) => ({
        panes: {
          ...state.panes,
          [workspaceId]: {
            sessions,
            activeSessionId,
            messagesBySession: activeSessionId ? { [activeSessionId]: messages } : {},
            isLoading: false,
            isAsking: false,
            error: null,
          },
        },
      }));
    } catch (error) {
      set((state) => ({
        panes: {
          ...state.panes,
          [workspaceId]: {
            sessions: state.panes[workspaceId]?.sessions ?? [],
            activeSessionId: state.panes[workspaceId]?.activeSessionId ?? null,
            messagesBySession: state.panes[workspaceId]?.messagesBySession ?? {},
            isLoading: false,
            isAsking: false,
            error: error instanceof Error ? error.message : '加载 agent sessions 失败',
          },
        },
      }));
    }
  },

  async createSession(workspaceId, title, surface) {
    const session = await workspaceAgentApi.createSession(workspaceId, title, surface);
    set((state) => ({
      panes: {
        ...state.panes,
        [workspaceId]: {
          sessions: [session, ...(state.panes[workspaceId]?.sessions ?? [])],
          activeSessionId: session.id,
          messagesBySession: state.panes[workspaceId]?.messagesBySession ?? {},
          isLoading: false,
          isAsking: false,
          error: null,
        },
      },
    }));
    return session;
  },

  async loadMessages(workspaceId, sessionId) {
    const messages = await workspaceAgentApi.listMessages(sessionId);
    set((state) => ({
      panes: {
        ...state.panes,
        [workspaceId]: {
          sessions: state.panes[workspaceId]?.sessions ?? [],
          activeSessionId: sessionId,
          messagesBySession: {
            ...(state.panes[workspaceId]?.messagesBySession ?? {}),
            [sessionId]: messages,
          },
          isLoading: false,
          isAsking: false,
          error: null,
        },
      },
    }));
  },

  async ask(input) {
    set((state) => ({
      panes: {
        ...state.panes,
        [input.workspaceId]: {
          sessions: state.panes[input.workspaceId]?.sessions ?? [],
          activeSessionId: state.panes[input.workspaceId]?.activeSessionId ?? input.sessionId ?? null,
          messagesBySession: state.panes[input.workspaceId]?.messagesBySession ?? {},
          isLoading: false,
          isAsking: true,
          error: null,
        },
      },
    }));
    try {
      const result = await workspaceAgentApi.ask(input);
      set((state) => {
        const existingPane = state.panes[input.workspaceId];
        const existingMessages = existingPane?.messagesBySession[result.session.id] ?? [];
        const nextSessions = [result.session, ...(existingPane?.sessions ?? []).filter((session) => session.id !== result.session.id)];
        return {
          panes: {
            ...state.panes,
            [input.workspaceId]: {
              sessions: nextSessions,
              activeSessionId: result.session.id,
              messagesBySession: {
                ...(existingPane?.messagesBySession ?? {}),
                [result.session.id]: [...existingMessages, result.userMessage, result.assistantMessage],
              },
              isLoading: false,
              isAsking: false,
              error: null,
            },
          },
        };
      });
      return result;
    } catch (error) {
      set((state) => ({
        panes: {
          ...state.panes,
          [input.workspaceId]: {
            sessions: state.panes[input.workspaceId]?.sessions ?? [],
            activeSessionId: state.panes[input.workspaceId]?.activeSessionId ?? input.sessionId ?? null,
            messagesBySession: state.panes[input.workspaceId]?.messagesBySession ?? {},
            isLoading: false,
            isAsking: false,
            error: error instanceof Error ? error.message : 'Agent ask 失败',
          },
        },
      }));
      throw error;
    }
  },
}));
```

- [ ] **Step 4: Create the session-first shell and move the old wiki UI into Knowledge mode**

```tsx
// frontend/src/components/WorkspaceAgentShell.tsx
import { Button } from './ui/Button';
import type { Workspace } from '../types/workspace';
import type { WorkspaceAgentMessage, WorkspaceAgentSession } from '../types/workspaceAgent';

export function WorkspaceAgentShell({
  workspace,
  sessions,
  activeSessionId,
  messages,
  isLoading,
  isAsking,
  error,
  onCreateSession,
  onSelectSession,
  onAsk,
  onSwitchMode,
}: {
  workspace: Workspace | null;
  sessions: WorkspaceAgentSession[];
  activeSessionId: string | null;
  messages: WorkspaceAgentMessage[];
  isLoading: boolean;
  isAsking: boolean;
  error: string | null;
  onCreateSession: () => Promise<void>;
  onSelectSession: (sessionId: string) => Promise<void>;
  onAsk: (question: string) => Promise<void>;
  onSwitchMode: () => void;
}) {
  return (
    <section className="workspace-agent-shell">
      <aside className="workspace-agent-left-rail">
        <div className="workspace-agent-left-header">
          <h2>{workspace?.name ?? 'Workspace'}</h2>
          <Button size="sm" onClick={() => void onCreateSession()}>+ New</Button>
        </div>
        <div className="workspace-agent-mode-strip">
          <button type="button" className="workspace-agent-mode-active">Sessions</button>
          <button type="button" onClick={onSwitchMode}>Knowledge</button>
        </div>
        <div className="workspace-agent-session-list">
          {sessions.map((session) => (
            <button
              key={session.id}
              type="button"
              className={session.id === activeSessionId ? 'workspace-agent-session-item workspace-agent-session-item-active' : 'workspace-agent-session-item'}
              onClick={() => void onSelectSession(session.id)}
            >
              <strong>{session.title}</strong>
              <small>{session.updatedAt}</small>
            </button>
          ))}
        </div>
      </aside>
      <div className="workspace-agent-center-pane">
        <div className="workspace-agent-thread">
          {messages.map((message) => (
            <article key={message.id} className={`workspace-agent-message workspace-agent-message-${message.role}`}>
              <small>{message.role}</small>
              <div>{message.content}</div>
            </article>
          ))}
        </div>
        <WorkspaceAgentComposer disabled={isAsking || !workspace} onAsk={onAsk} />
        {isLoading ? <div className="empty-inline">Loading sessions...</div> : null}
        {error ? <div className="reader-error">{error}</div> : null}
      </div>
      <aside className="workspace-agent-context-pane">
        <div className="workspace-agent-context-card">
          <strong>Suggested Skills</strong>
          <span>Ask with evidence</span>
          <span>Cross-source synthesis</span>
          <span>Promote to wiki</span>
        </div>
      </aside>
    </section>
  );
}
```

```tsx
// frontend/src/components/WorkspaceKnowledgeMode.tsx
import type { ProviderConfig } from '../types/config';
import type { DocumentRecord, Workspace } from '../types/workspace';
import type { WorkspaceKnowledgeCompileSummary, WorkspaceKnowledgeSource } from '../types/workspaceKnowledge';
import type { WorkspaceWikiPage, WorkspaceWikiPageContent, WorkspaceWikiScanJob } from '../types/workspaceWiki';

export function WorkspaceKnowledgeMode({
  workspace,
  documents,
  wikiPages,
  selectedWikiPageId,
  wikiPageContent,
  activeWikiJob,
  wikiSources,
  compileSummary,
  llmProviderConfigs,
  wikiScanProviderId,
  wikiScanModelId,
  isLoadingWikiPages,
  isLoadingWikiPageContent,
  isStartingWikiScan,
  isCancellingWikiScan,
  isDeletingWikiPages,
  onSwitchMode,
  onStartWikiScan,
  onCancelWikiScan,
  onRefreshWikiPages,
  onSelectWikiPage,
  onDeleteWikiPages,
  onImportFiles,
  onRefreshDocuments,
  onOpenPdf,
  onDeleteDocument,
  onChangeWikiScanModel,
}: {
  workspace: Workspace | null;
  documents: DocumentRecord[];
  wikiPages: WorkspaceWikiPage[];
  selectedWikiPageId: string | null;
  wikiPageContent: WorkspaceWikiPageContent | null;
  activeWikiJob: WorkspaceWikiScanJob | null;
  wikiSources: WorkspaceKnowledgeSource[];
  compileSummary: WorkspaceKnowledgeCompileSummary | null;
  llmProviderConfigs: ProviderConfig[];
  wikiScanProviderId: number;
  wikiScanModelId: number;
  isLoadingWikiPages: boolean;
  isLoadingWikiPageContent: boolean;
  isStartingWikiScan: boolean;
  isCancellingWikiScan: boolean;
  isDeletingWikiPages: boolean;
  onSwitchMode: () => void;
  onStartWikiScan: (providerId: number, modelId: number) => Promise<void>;
  onCancelWikiScan: () => Promise<void>;
  onRefreshWikiPages: () => Promise<void>;
  onSelectWikiPage: (pageId: string) => Promise<void>;
  onDeleteWikiPages: () => Promise<void>;
  onImportFiles: () => Promise<void>;
  onRefreshDocuments: () => Promise<void>;
  onOpenPdf: (document: DocumentRecord) => void;
  onDeleteDocument: (document: Pick<DocumentRecord, 'id' | 'workspaceId' | 'title' | 'originalFileName'>) => Promise<void>;
  onChangeWikiScanModel: (providerId: number, modelId: number) => void;
}) {
  return (
    <section className="workspace-knowledge-mode">
      <div className="workspace-agent-mode-strip workspace-agent-mode-strip-inline">
        <button type="button" onClick={onSwitchMode}>Sessions</button>
        <button type="button" className="workspace-agent-mode-active">Knowledge</button>
      </div>
      <WorkspaceKnowledgePanels
        workspace={workspace}
        documents={documents}
        wikiPages={wikiPages}
        selectedWikiPageId={selectedWikiPageId}
        wikiPageContent={wikiPageContent}
        activeWikiJob={activeWikiJob}
        wikiSources={wikiSources}
        compileSummary={compileSummary}
        llmProviderConfigs={llmProviderConfigs}
        wikiScanProviderId={wikiScanProviderId}
        wikiScanModelId={wikiScanModelId}
        isLoadingWikiPages={isLoadingWikiPages}
        isLoadingWikiPageContent={isLoadingWikiPageContent}
        isStartingWikiScan={isStartingWikiScan}
        isCancellingWikiScan={isCancellingWikiScan}
        isDeletingWikiPages={isDeletingWikiPages}
        onStartWikiScan={onStartWikiScan}
        onCancelWikiScan={onCancelWikiScan}
        onRefreshWikiPages={onRefreshWikiPages}
        onSelectWikiPage={onSelectWikiPage}
        onDeleteWikiPages={onDeleteWikiPages}
        onImportFiles={onImportFiles}
        onRefreshDocuments={onRefreshDocuments}
        onOpenPdf={onOpenPdf}
        onDeleteDocument={onDeleteDocument}
        onChangeWikiScanModel={onChangeWikiScanModel}
      />
    </section>
  );
}
```

```tsx
// frontend/src/App.tsx (new workspace agent pane state)
const [workspaceTabAgent, setWorkspaceTabAgent] = useState<Record<string, {
  sessions: WorkspaceAgentSession[];
  activeSessionId: string | null;
  messagesBySession: Record<string, WorkspaceAgentMessage[]>;
  isLoading: boolean;
  isAsking: boolean;
  error: string | null;
}>>({});

const ensureWorkspaceTabAgent = useCallback(async (workspaceId: string) => {
  setWorkspaceTabAgent((current) => ({
    ...current,
    [workspaceId]: {
      sessions: current[workspaceId]?.sessions ?? [],
      activeSessionId: current[workspaceId]?.activeSessionId ?? null,
      messagesBySession: current[workspaceId]?.messagesBySession ?? {},
      isLoading: true,
      isAsking: current[workspaceId]?.isAsking ?? false,
      error: null,
    },
  }));
  const sessions = await workspaceAgentApi.listSessions(workspaceId);
  const activeSessionId = sessions[0]?.id ?? null;
  const messages = activeSessionId ? await workspaceAgentApi.listMessages(activeSessionId) : [];
  setWorkspaceTabAgent((current) => ({
    ...current,
    [workspaceId]: {
      sessions,
      activeSessionId,
      messagesBySession: activeSessionId ? { [activeSessionId]: messages } : {},
      isLoading: false,
      isAsking: false,
      error: null,
    },
  }));
}, []);
```

```css
/* frontend/src/style/workspace.css */
.workspace-agent-shell {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr) 320px;
  min-height: calc(100vh - 110px);
}

.workspace-agent-left-rail {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px;
  border-right: 1px solid var(--color-border-default);
  background: color-mix(in srgb, var(--color-bg-secondary) 88%, #f4a259 12%);
}

.workspace-agent-center-pane {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.workspace-agent-thread {
  flex: 1;
  overflow: auto;
  padding: 20px 24px;
}

.workspace-agent-context-pane {
  border-left: 1px solid var(--color-border-default);
  padding: 20px;
  background: var(--color-bg-secondary);
}
```

- [ ] **Step 5: Regenerate Wails bindings and run the frontend build**

Run: `wails generate module`

Expected: updates `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/main/App.js`, and `frontend/wailsjs/go/models.ts` with the new workspace agent methods and types.

Run: `npm run build`

Workdir: `frontend`

Expected: PASS

- [ ] **Step 6: Commit the workspace shell foundation**

```bash
git add app.go frontend/src/types/workspaceAgent.ts frontend/src/api/workspaceAgent.ts frontend/src/store/workspaceAgentStore.ts frontend/src/components/WorkspaceAgentShell.tsx frontend/src/components/WorkspaceKnowledgeMode.tsx frontend/src/components/WorkspaceTab.tsx frontend/src/App.tsx frontend/src/style/workspace.css frontend/wailsjs/go/main/App.d.ts frontend/wailsjs/go/main/App.js frontend/wailsjs/go/models.ts
git commit -m "feat: add session-first workspace agent shell"
```

## Task 3: Route Reader Asks Through The Shared Agent Session System

**Files:**
- Modify: `frontend/src/store/tabStore.ts`
- Modify: `frontend/src/components/ReaderTab.tsx`
- Modify: `frontend/src/components/ReaderAIPanel.tsx`
- Modify: `workspace_agent_service.go`
- Modify: `workspace_agent_service_test.go`

- [ ] **Step 1: Write the failing shared-session backend test**

```go
func TestWorkspaceAgentServiceAskUsesExistingSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	workspace, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{Name: "Reader Workspace"})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}
	session, err := store.CreateWorkspaceAgentSession(ctx, WorkspaceAgentSessionCreateInput{
		WorkspaceID: workspace.ID,
		Title:       "Shared session",
		Surface:     string(WorkspaceAgentSurfaceWorkspace),
	})
	if err != nil {
		t.Fatalf("CreateWorkspaceAgentSession() error = %v", err)
	}

	query := &stubWorkspaceAgentQuery{result: WorkspaceKnowledgeQueryResult{Answer: "Reader-grounded answer."}}
	service := newWorkspaceAgentService(store, query)

	result, err := service.Ask(ctx, WorkspaceAgentAskInput{
		WorkspaceID: workspace.ID,
		SessionID:   session.ID,
		Surface:     string(WorkspaceAgentSurfaceReader),
		Question:    "Explain this selected paragraph.",
		DocumentID:  "doc_123",
		Selection:   "Attention replaces recurrence.",
		CurrentPage: 5,
	})
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if result.Session.ID != session.ID {
		t.Fatalf("result.Session.ID = %q, want %q", result.Session.ID, session.ID)
	}
}
```

- [ ] **Step 2: Run the targeted backend test to verify it fails if session reuse is broken**

Run: `go test ./... -run TestWorkspaceAgentServiceAskUsesExistingSession -count=1`

Expected: FAIL until `Ask()` reuses the supplied session and the new reader wiring is complete.

- [ ] **Step 3: Add optional agent session binding to tabs and pass it into the reader**

```ts
// frontend/src/store/tabStore.ts
export interface TabItem {
  id: string;
  title: string;
  pdfPath: string | null;
  type?: 'document' | 'workspace';
  workspaceId?: string;
  documentId?: string;
  sourceKind?: 'workspace_document' | 'zotero_item';
  itemType?: string;
  citeKey?: string;
  agentSessionId?: string;
}
```

```tsx
// frontend/src/components/ReaderTab.tsx
<ReaderAIPanel
  tab={tab}
  llmConfigs={groupedProviders.llm}
  drawingConfigs={groupedProviders.drawing}
  activeLLMConfig={activeLLMConfig}
  activeLLMModel={activeLLMModel}
  llmProviderId={llmProviderId}
  llmModelId={llmModelId}
  setLlmProviderId={setLlmProviderId}
  setLlmModelId={setLlmModelId}
  initialSessionId={tab.agentSessionId ?? null}
/>
```

- [ ] **Step 4: Replace direct knowledge-query calls in the reader with shared agent asks**

```tsx
// frontend/src/components/ReaderAIPanel.tsx
import { workspaceAgentApi } from '../api/workspaceAgent';

interface ReaderAIPanelProps {
  tab: TabItem;
  llmConfigs: ProviderConfig[];
  drawingConfigs: ProviderConfig[];
  activeLLMConfig: ProviderConfig | null;
  activeLLMModel: ModelRecord | null;
  llmProviderId: number | null;
  llmModelId: number | null;
  setLlmProviderId: (value: number | null) => void;
  setLlmModelId: (value: number | null) => void;
  initialSessionId: string | null;
}

const [agentSessionId, setAgentSessionId] = useState<string | null>(initialSessionId);

async function handleAsk() {
  if (!workspaceId || !activeLLMModel || copilotState.isAsking) {
    return;
  }
  setCopilotState((prev) => ({ ...prev, isAsking: true, answerError: null }));
  try {
    const result = await workspaceAgentApi.ask({
      workspaceId,
      sessionId: agentSessionId ?? undefined,
      surface: 'reader',
      question: copilotState.question,
      documentId,
      selection: copilotState.scope.selection ? selection.cleaned : undefined,
      currentPage: copilotState.scope.page ? activePage : undefined,
      providerId: llmProviderId ?? 0,
      modelId: llmModelId ?? 0,
    });
    setAgentSessionId(result.session.id);
    const evidence = {
      entities: result.query.evidence.filter((e) => e.kind === 'entity'),
      claims: result.query.evidence.filter((e) => e.kind === 'claim'),
      tasks: result.query.evidence.filter((e) => e.kind === 'task'),
      sources: result.query.evidence.filter((e) => e.kind === 'wiki_page' || e.kind === 'raw_excerpt'),
    };
    setCopilotState((prev) => ({
      ...prev,
      isAsking: false,
      answer: result.query.answer,
      evidence,
      candidates: result.query.candidates,
    }));
  } catch (error) {
    setCopilotState((prev) => ({
      ...prev,
      isAsking: false,
      answer: null,
      answerError: getErrorMessage(error, '查询失败'),
    }));
  }
}
```

- [ ] **Step 5: Run verification for reader/workspace convergence**

Run: `go test ./... -run 'TestWorkspaceAgentServiceAskCreatesSessionAndMessages|TestListWorkspaceAgentSessionsSortsNewestFirst|TestWorkspaceAgentServiceAskUsesExistingSession' -count=1`

Expected: PASS

Run: `npm run build`

Workdir: `frontend`

Expected: PASS

Run: `go test ./...`

Expected: PASS

- [ ] **Step 6: Commit the shared reader/workspace agent flow**

```bash
git add frontend/src/store/tabStore.ts frontend/src/components/ReaderTab.tsx frontend/src/components/ReaderAIPanel.tsx workspace_agent_service.go workspace_agent_service_test.go
git commit -m "feat: share agent sessions between workspace and reader"
```

## Self-Review

### Spec coverage

- The workspace becomes an agent-first shell in Task 2.
- The workspace and reader share one agent/session model in Tasks 1 and 3.
- The wiki remains the backing memory layer because `workspaceAgentService` delegates grounding to the existing workspace knowledge query flow in Tasks 1 and 3.
- Manual memory build and knowledge browsing remain available through `WorkspaceKnowledgeMode` in Task 2.
- Promote remains compatible with future work because the shared ask flow still returns promote candidates in Task 3.

### Placeholder scan

- No `TODO`, `TBD`, or “implement later” placeholders remain.
- Each task includes exact files, code snippets, and verification commands.
- Frontend verification uses `npm run build` because this repo does not currently ship a browser-side test runner.

### Type consistency

- Backend and frontend both use `WorkspaceAgentSession`, `WorkspaceAgentMessage`, `WorkspaceAgentAskInput`, and `WorkspaceAgentAskResult` consistently.
- `agentSessionId` is the shared tab/session handoff field across workspace and reader surfaces.
- The ask skill remains grounded in `WorkspaceKnowledgeQueryResult`, so the evidence and candidate shape stays consistent across the old knowledge query layer and the new agent shell.
