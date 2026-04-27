package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubWorkspaceAgentQuery struct {
	inputs []WorkspaceKnowledgeQueryInput
	result WorkspaceKnowledgeQueryResult
	err    error
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
	if result.Session.Status != "active" {
		t.Fatalf("result.Session.Status = %q, want %q", result.Session.Status, "active")
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
	if messages[0].ID <= 0 {
		t.Fatalf("messages[0].ID = %d, want > 0", messages[0].ID)
	}
	if messages[1].ID <= messages[0].ID {
		t.Fatalf("message ids out of order: first=%d second=%d", messages[0].ID, messages[1].ID)
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
		SessionID:   older.ID,
		WorkspaceID: workspace.ID,
		Surface:     string(WorkspaceAgentSurfaceWorkspace),
		Role:        string(WorkspaceAgentMessageRoleUser),
		Kind:        "message",
		Content:     "older",
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
		SessionID:   newer.ID,
		WorkspaceID: workspace.ID,
		Surface:     string(WorkspaceAgentSurfaceWorkspace),
		Role:        string(WorkspaceAgentMessageRoleUser),
		Kind:        "message",
		Content:     "newer",
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
	if sessions[0].Status != "active" {
		t.Fatalf("sessions[0].Status = %q, want %q", sessions[0].Status, "active")
	}
}

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
		WorkspaceID:             workspace.ID,
		SessionID:               session.ID,
		Surface:                 string(WorkspaceAgentSurfaceReader),
		Question:                "Explain this selected paragraph.",
		DocumentID:              "doc_123",
		IncludeDocumentContext:  true,
		IncludeWorkspaceContext: true,
		Selection:               "Attention replaces recurrence.",
		CurrentPage:             5,
	})
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if result.Session.ID != session.ID {
		t.Fatalf("result.Session.ID = %q, want %q", result.Session.ID, session.ID)
	}
	if len(query.inputs) != 1 {
		t.Fatalf("len(query.inputs) = %d, want 1", len(query.inputs))
	}
	if query.inputs[0].WorkspaceID != workspace.ID {
		t.Fatalf("query workspace = %q, want %q", query.inputs[0].WorkspaceID, workspace.ID)
	}
	if !strings.Contains(query.inputs[0].Question, "Reader context:") {
		t.Fatalf("query question = %q, want reader context heading", query.inputs[0].Question)
	}
	if !strings.Contains(query.inputs[0].Question, "- documentId: doc_123") {
		t.Fatalf("query question = %q, want document marker", query.inputs[0].Question)
	}
	if !strings.Contains(query.inputs[0].Question, "- workspaceContext: enabled") {
		t.Fatalf("query question = %q, want workspace marker", query.inputs[0].Question)
	}
	if !strings.Contains(query.inputs[0].Question, "- currentPage: 5") {
		t.Fatalf("query question = %q, want page marker", query.inputs[0].Question)
	}
	if !strings.Contains(query.inputs[0].Question, "- selection: Attention replaces recurrence.") {
		t.Fatalf("query question = %q, want selection marker", query.inputs[0].Question)
	}
	sessions, err := store.ListWorkspaceAgentSessions(ctx, workspace.ID)
	if err != nil {
		t.Fatalf("ListWorkspaceAgentSessions() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(sessions))
	}
	messages, err := store.ListWorkspaceAgentMessages(ctx, session.ID)
	if err != nil {
		t.Fatalf("ListWorkspaceAgentMessages() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(messages))
	}
	if messages[0].Surface != string(WorkspaceAgentSurfaceReader) {
		t.Fatalf("messages[0].Surface = %q, want %q", messages[0].Surface, WorkspaceAgentSurfaceReader)
	}
	if messages[1].Surface != string(WorkspaceAgentSurfaceReader) {
		t.Fatalf("messages[1].Surface = %q, want %q", messages[1].Surface, WorkspaceAgentSurfaceReader)
	}
	if messages[0].Content != "Explain this selected paragraph." {
		t.Fatalf("messages[0].Content = %q, want raw question", messages[0].Content)
	}
}

func TestWorkspaceAgentServiceAskUsesRecentSessionHistoryInDelegatedQuery(t *testing.T) {
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
	session, err := store.CreateWorkspaceAgentSession(ctx, WorkspaceAgentSessionCreateInput{
		WorkspaceID: workspace.ID,
		Title:       "Shared session",
		Surface:     string(WorkspaceAgentSurfaceWorkspace),
	})
	if err != nil {
		t.Fatalf("CreateWorkspaceAgentSession() error = %v", err)
	}
	if _, err := store.AppendWorkspaceAgentMessage(ctx, WorkspaceAgentMessageCreateInput{
		SessionID:   session.ID,
		WorkspaceID: workspace.ID,
		Surface:     string(WorkspaceAgentSurfaceWorkspace),
		Role:        string(WorkspaceAgentMessageRoleUser),
		Kind:        "question",
		Prompt:      "Summarize the findings.",
		Content:     "Summarize the findings.",
	}); err != nil {
		t.Fatalf("AppendWorkspaceAgentMessage(user) error = %v", err)
	}
	if _, err := store.AppendWorkspaceAgentMessage(ctx, WorkspaceAgentMessageCreateInput{
		SessionID:   session.ID,
		WorkspaceID: workspace.ID,
		Surface:     string(WorkspaceAgentSurfaceWorkspace),
		Role:        string(WorkspaceAgentMessageRoleAssistant),
		Kind:        "answer",
		Prompt:      "Summarize the findings.",
		Content:     "The findings focus on efficiency gains.",
	}); err != nil {
		t.Fatalf("AppendWorkspaceAgentMessage(assistant) error = %v", err)
	}

	query := &stubWorkspaceAgentQuery{result: WorkspaceKnowledgeQueryResult{Answer: "Follow-up answer."}}
	service := newWorkspaceAgentService(store, query)
	_, err = service.Ask(ctx, WorkspaceAgentAskInput{
		WorkspaceID: workspace.ID,
		SessionID:   session.ID,
		Surface:     string(WorkspaceAgentSurfaceWorkspace),
		Question:    "How does that compare to baseline?",
	})
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if len(query.inputs) != 1 {
		t.Fatalf("len(query.inputs) = %d, want 1", len(query.inputs))
	}
	if !strings.Contains(query.inputs[0].Question, "Recent conversation:") {
		t.Fatalf("query question = %q, want recent conversation heading", query.inputs[0].Question)
	}
	if !strings.Contains(query.inputs[0].Question, "User: Summarize the findings.") {
		t.Fatalf("query question = %q, want user history", query.inputs[0].Question)
	}
	if !strings.Contains(query.inputs[0].Question, "Assistant: The findings focus on efficiency gains.") {
		t.Fatalf("query question = %q, want assistant history", query.inputs[0].Question)
	}
	if !strings.Contains(query.inputs[0].Question, "How does that compare to baseline?") {
		t.Fatalf("query question = %q, want raw current question", query.inputs[0].Question)
	}
}

func TestAppendWorkspaceAgentMessageRejectsSessionWorkspaceMismatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	workspaceA, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{Name: "Workspace A"})
	if err != nil {
		t.Fatalf("CreateWorkspace(A) error = %v", err)
	}
	workspaceB, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{Name: "Workspace B"})
	if err != nil {
		t.Fatalf("CreateWorkspace(B) error = %v", err)
	}
	session, err := store.CreateWorkspaceAgentSession(ctx, WorkspaceAgentSessionCreateInput{
		WorkspaceID: workspaceA.ID,
		Title:       "Session A",
	})
	if err != nil {
		t.Fatalf("CreateWorkspaceAgentSession() error = %v", err)
	}

	if _, err := store.AppendWorkspaceAgentMessage(ctx, WorkspaceAgentMessageCreateInput{
		SessionID:   session.ID,
		WorkspaceID: workspaceB.ID,
		Role:        string(WorkspaceAgentMessageRoleUser),
		Content:     "cross workspace",
	}); err == nil {
		t.Fatal("AppendWorkspaceAgentMessage() error = nil, want mismatch error")
	}
}

func TestListWorkspaceAgentMessagesForWorkspaceRejectsWorkspaceMismatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	workspaceA, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{Name: "Workspace A"})
	if err != nil {
		t.Fatalf("CreateWorkspace(A) error = %v", err)
	}
	workspaceB, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{Name: "Workspace B"})
	if err != nil {
		t.Fatalf("CreateWorkspace(B) error = %v", err)
	}
	session, err := store.CreateWorkspaceAgentSession(ctx, WorkspaceAgentSessionCreateInput{
		WorkspaceID: workspaceA.ID,
		Title:       "Session A",
	})
	if err != nil {
		t.Fatalf("CreateWorkspaceAgentSession() error = %v", err)
	}
	if _, err := store.AppendWorkspaceAgentMessage(ctx, WorkspaceAgentMessageCreateInput{
		SessionID:   session.ID,
		WorkspaceID: workspaceA.ID,
		Role:        string(WorkspaceAgentMessageRoleUser),
		Content:     "hello",
	}); err != nil {
		t.Fatalf("AppendWorkspaceAgentMessage() error = %v", err)
	}

	if _, err := store.ListWorkspaceAgentMessagesForWorkspace(ctx, workspaceB.ID, session.ID); err == nil {
		t.Fatal("ListWorkspaceAgentMessagesForWorkspace() error = nil, want mismatch error")
	}
}

func TestAppendWorkspaceAgentMessageRejectsInvalidRole(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	paths := newTestAppPaths(t)
	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	workspace, err := store.CreateWorkspace(ctx, WorkspaceUpsertInput{Name: "Workspace A"})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}
	session, err := store.CreateWorkspaceAgentSession(ctx, WorkspaceAgentSessionCreateInput{WorkspaceID: workspace.ID})
	if err != nil {
		t.Fatalf("CreateWorkspaceAgentSession() error = %v", err)
	}

	if _, err := store.AppendWorkspaceAgentMessage(ctx, WorkspaceAgentMessageCreateInput{
		SessionID:   session.ID,
		WorkspaceID: workspace.ID,
		Role:        "system",
		Content:     "invalid role",
	}); err == nil {
		t.Fatal("AppendWorkspaceAgentMessage() error = nil, want invalid role error")
	}
}

func TestWorkspaceAgentServiceAskQueryFailureDoesNotPersistSession(t *testing.T) {
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

	service := newWorkspaceAgentService(store, &stubWorkspaceAgentQuery{err: errors.New("query failed")})
	if _, err := service.Ask(ctx, WorkspaceAgentAskInput{
		WorkspaceID: workspace.ID,
		Question:    "What changed?",
	}); err == nil {
		t.Fatal("Ask() error = nil, want query failure")
	}

	sessions, err := store.ListWorkspaceAgentSessions(ctx, workspace.ID)
	if err != nil {
		t.Fatalf("ListWorkspaceAgentSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("len(sessions) = %d, want 0", len(sessions))
	}

	var messageCount int
	if err := store.appDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_agent_messages;`).Scan(&messageCount); err != nil {
		t.Fatalf("count workspace_agent_messages error = %v", err)
	}
	if messageCount != 0 {
		t.Fatalf("message count = %d, want 0", messageCount)
	}
}

func TestWorkspaceAgentServiceAskRejectsInvalidSurface(t *testing.T) {
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

	service := newWorkspaceAgentService(store, &stubWorkspaceAgentQuery{result: WorkspaceKnowledgeQueryResult{Answer: "ok"}})
	if _, err := service.Ask(ctx, WorkspaceAgentAskInput{
		WorkspaceID: workspace.ID,
		Surface:     "invalid-surface",
		Question:    "What changed?",
	}); err == nil {
		t.Fatal("Ask() error = nil, want invalid surface error")
	}
}

func TestWorkspaceAgentServiceAskAcceptsReaderContext(t *testing.T) {
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

	query := &stubWorkspaceAgentQuery{result: WorkspaceKnowledgeQueryResult{Answer: "ok"}}
	service := newWorkspaceAgentService(store, query)
	result, err := service.Ask(ctx, WorkspaceAgentAskInput{
		WorkspaceID:            workspace.ID,
		Surface:                string(WorkspaceAgentSurfaceReader),
		DocumentID:             "doc_1",
		IncludeDocumentContext: true,
		Selection:              "selected text",
		CurrentPage:            3,
		Question:               "What changed?",
	})
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if result.Session.Surface != string(WorkspaceAgentSurfaceReader) {
		t.Fatalf("result.Session.Surface = %q, want %q", result.Session.Surface, WorkspaceAgentSurfaceReader)
	}
	if len(query.inputs) != 1 {
		t.Fatalf("len(query.inputs) = %d, want 1", len(query.inputs))
	}
	if !strings.Contains(query.inputs[0].Question, "Reader context:") {
		t.Fatalf("query question = %q, want reader context heading", query.inputs[0].Question)
	}
	if !strings.Contains(query.inputs[0].Question, "- documentId: doc_1") {
		t.Fatalf("query question = %q, want document marker", query.inputs[0].Question)
	}
	if !strings.Contains(query.inputs[0].Question, "- currentPage: 3") {
		t.Fatalf("query question = %q, want page marker", query.inputs[0].Question)
	}
	if !strings.Contains(query.inputs[0].Question, "- selection: selected text") {
		t.Fatalf("query question = %q, want selection marker", query.inputs[0].Question)
	}
	if strings.Contains(query.inputs[0].Question, "- workspaceContext: enabled") {
		t.Fatalf("query question = %q, want no workspace marker by default", query.inputs[0].Question)
	}
	messages, err := store.ListWorkspaceAgentMessages(ctx, result.Session.ID)
	if err != nil {
		t.Fatalf("ListWorkspaceAgentMessages() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(messages))
	}
	if messages[0].Content != "What changed?" {
		t.Fatalf("messages[0].Content = %q, want raw question", messages[0].Content)
	}
}

func TestWorkspaceAgentServiceAskOmitsDisabledReaderContextMarkers(t *testing.T) {
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

	query := &stubWorkspaceAgentQuery{result: WorkspaceKnowledgeQueryResult{Answer: "ok"}}
	service := newWorkspaceAgentService(store, query)
	_, err = service.Ask(ctx, WorkspaceAgentAskInput{
		WorkspaceID: workspace.ID,
		Surface:     string(WorkspaceAgentSurfaceReader),
		DocumentID:  "doc_1",
		Selection:   "selected text",
		CurrentPage: 3,
		Question:    "What changed?",
	})
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if len(query.inputs) != 1 {
		t.Fatalf("len(query.inputs) = %d, want 1", len(query.inputs))
	}
	if query.inputs[0].Question != "What changed?\n\nReader context:\n- currentPage: 3\n- selection: selected text" {
		t.Fatalf("query question = %q, want only enabled reader markers", query.inputs[0].Question)
	}
}
