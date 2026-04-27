package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

const workspaceAgentQueryHistoryLimit = 4

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
	if s == nil || s.store == nil {
		return WorkspaceAgentAskResult{}, fmt.Errorf("workspace agent store is required")
	}
	if s.query == nil {
		return WorkspaceAgentAskResult{}, fmt.Errorf("workspace agent query is required")
	}

	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID == "" {
		return WorkspaceAgentAskResult{}, fmt.Errorf("workspace id is required")
	}
	question := strings.TrimSpace(input.Question)
	if question == "" {
		return WorkspaceAgentAskResult{}, fmt.Errorf("question is required")
	}
	if _, err := s.store.GetWorkspace(ctx, workspaceID); err != nil {
		return WorkspaceAgentAskResult{}, err
	}
	askSurface, err := normalizeWorkspaceAgentSurface(input.Surface)
	if err != nil {
		return WorkspaceAgentAskResult{}, err
	}

	sessionID := strings.TrimSpace(input.SessionID)
	var (
		session          WorkspaceAgentSession
		recentMessages   []WorkspaceAgentMessage
		userMessage      WorkspaceAgentMessage
		assistantMessage WorkspaceAgentMessage
	)
	if sessionID == "" {
		session = WorkspaceAgentSession{
			WorkspaceID: workspaceID,
			Title:       defaultWorkspaceAgentSessionTitle(question),
			Surface:     askSurface,
			Status:      "active",
		}
	} else {
		session, err = s.store.getWorkspaceAgentSession(ctx, workspaceID, sessionID)
		if err != nil {
			return WorkspaceAgentAskResult{}, err
		}
		recentMessages, err = s.store.ListWorkspaceAgentMessages(ctx, sessionID)
		if err != nil {
			return WorkspaceAgentAskResult{}, err
		}
	}

	queryResult, err := s.query.Query(ctx, WorkspaceKnowledgeQueryInput{
		WorkspaceID: workspaceID,
		ProviderID:  input.ProviderID,
		ModelID:     input.ModelID,
		Question:    buildWorkspaceAgentQueryQuestion(question, input, recentMessages),
	})
	if err != nil {
		return WorkspaceAgentAskResult{}, err
	}

	err = withWorkspaceAgentTx(ctx, s.store.appDB, func(tx *sql.Tx) error {
		if sessionID == "" {
			createdSession, createErr := s.store.CreateWorkspaceAgentSessionTx(ctx, tx, WorkspaceAgentSessionCreateInput{
				WorkspaceID: workspaceID,
				Title:       session.Title,
				Surface:     session.Surface,
			})
			if createErr != nil {
				return createErr
			}
			session = createdSession
		}

		createdUserMessage, appendErr := s.store.AppendWorkspaceAgentMessageTx(ctx, tx, WorkspaceAgentMessageCreateInput{
			SessionID:   session.ID,
			WorkspaceID: workspaceID,
			Surface:     askSurface,
			Role:        string(WorkspaceAgentMessageRoleUser),
			Kind:        "question",
			Prompt:      question,
			Content:     question,
		})
		if appendErr != nil {
			return appendErr
		}
		userMessage = createdUserMessage

		createdAssistantMessage, appendErr := s.store.AppendWorkspaceAgentMessageTx(ctx, tx, WorkspaceAgentMessageCreateInput{
			SessionID:     session.ID,
			WorkspaceID:   workspaceID,
			Surface:       askSurface,
			Role:          string(WorkspaceAgentMessageRoleAssistant),
			Kind:          "answer",
			Prompt:        question,
			Content:       strings.TrimSpace(queryResult.Answer),
			SkillName:     "ask_with_evidence",
			EvidenceCount: len(queryResult.Evidence),
		})
		if appendErr != nil {
			return appendErr
		}
		assistantMessage = createdAssistantMessage
		return nil
	})
	if err != nil {
		return WorkspaceAgentAskResult{}, err
	}

	session, err = s.store.getWorkspaceAgentSession(ctx, workspaceID, session.ID)
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

func withWorkspaceAgentTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin workspace agent transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit workspace agent transaction: %w", err)
	}
	return nil
}

func buildWorkspaceAgentQueryQuestion(question string, input WorkspaceAgentAskInput, recentMessages []WorkspaceAgentMessage) string {
	historyBlock := buildWorkspaceAgentRecentConversationBlock(recentMessages)
	readerContextBlock := buildWorkspaceAgentReaderContextBlock(input)
	if historyBlock == "" && readerContextBlock == "" {
		return question
	}

	var builder strings.Builder
	builder.WriteString(question)
	if historyBlock != "" {
		builder.WriteString("\n\n")
		builder.WriteString(historyBlock)
	}
	if readerContextBlock != "" {
		builder.WriteString("\n\n")
		builder.WriteString(readerContextBlock)
	}
	return builder.String()
}

func buildWorkspaceAgentRecentConversationBlock(messages []WorkspaceAgentMessage) string {
	if len(messages) == 0 {
		return ""
	}
	start := len(messages) - workspaceAgentQueryHistoryLimit
	if start < 0 {
		start = 0
	}
	recent := messages[start:]

	var builder strings.Builder
	builder.WriteString("Recent conversation:")
	for _, message := range recent {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			content = strings.TrimSpace(message.Prompt)
		}
		if content == "" {
			continue
		}
		builder.WriteString("\n- ")
		builder.WriteString(workspaceAgentConversationRoleLabel(message.Role))
		builder.WriteString(": ")
		builder.WriteString(content)
	}
	if builder.String() == "Recent conversation:" {
		return ""
	}
	return builder.String()
}

func buildWorkspaceAgentReaderContextBlock(input WorkspaceAgentAskInput) string {
	documentID := strings.TrimSpace(input.DocumentID)
	selection := strings.TrimSpace(input.Selection)
	includeDocumentContext := input.IncludeDocumentContext && documentID != ""
	includeWorkspaceContext := input.IncludeWorkspaceContext
	if !includeDocumentContext && !includeWorkspaceContext && selection == "" && input.CurrentPage <= 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("Reader context:")
	if includeDocumentContext {
		builder.WriteString("\n- documentId: ")
		builder.WriteString(documentID)
	}
	if includeWorkspaceContext {
		builder.WriteString("\n- workspaceContext: enabled")
	}
	if input.CurrentPage > 0 {
		builder.WriteString("\n- currentPage: ")
		builder.WriteString(fmt.Sprintf("%d", input.CurrentPage))
	}
	if selection != "" {
		builder.WriteString("\n- selection: ")
		builder.WriteString(selection)
	}
	return builder.String()
}

func workspaceAgentConversationRoleLabel(role string) string {
	switch strings.TrimSpace(role) {
	case string(WorkspaceAgentMessageRoleAssistant):
		return "Assistant"
	default:
		return "User"
	}
}

func defaultWorkspaceAgentSessionTitle(question string) string {
	title := strings.Join(strings.Fields(strings.TrimSpace(question)), " ")
	if title == "" {
		return ""
	}
	const maxTitleLen = 80
	if len(title) <= maxTitleLen {
		return title
	}
	return strings.TrimSpace(title[:maxTitleLen])
}

func isValidWorkspaceAgentMessageRole(role string) bool {
	switch strings.TrimSpace(role) {
	case string(WorkspaceAgentMessageRoleUser), string(WorkspaceAgentMessageRoleAssistant):
		return true
	default:
		return false
	}
}

func normalizeWorkspaceAgentSurface(surface string) (string, error) {
	trimmed := strings.TrimSpace(surface)
	if trimmed == "" {
		return string(WorkspaceAgentSurfaceWorkspace), nil
	}
	switch WorkspaceAgentSurface(trimmed) {
	case WorkspaceAgentSurfaceWorkspace, WorkspaceAgentSurfaceReader:
		return trimmed, nil
	default:
		return "", fmt.Errorf("invalid workspace agent surface: %s", trimmed)
	}
}
