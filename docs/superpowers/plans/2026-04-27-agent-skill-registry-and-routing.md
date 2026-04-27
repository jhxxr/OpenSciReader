# Agent Skill Registry And Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a real shared skill registry and routing layer so workspace and reader asks can run explicit or auto-selected agent skills on the same session system.

**Architecture:** Extend the current `workspaceAgentService` from a thin ask wrapper into a small orchestration layer with backend-owned skill definitions, simple routing heuristics, and per-skill execution handlers. Reuse the existing knowledge query and workspace wiki services instead of inventing a second execution stack, then surface the same backend skill catalog in the workspace shell and reader panel so both manual skill selection and agent-selected routing stay consistent.

**Tech Stack:** Go, SQLite, Wails v2, React 18, TypeScript, Zustand.

---

## Scope

This plan covers the next missing product layer after the shared session foundation:

1. backend-owned skill definitions
2. user-triggered skill selection
3. basic agent-selected routing heuristics
4. shared skill metadata across workspace and reader surfaces

This plan intentionally does **not** implement the later memory-browser deepening work such as dedicated `Insights` and `Builds` sections in Knowledge mode. It also does **not** replace the current retrieval pipeline or redesign the workspace wiki build engine.

## File Map

### Backend files

- Modify: `config_types.go`
  Add skill identifiers, skill definitions, routed execution metadata, and richer ask/result contracts.
- Create: `workspace_agent_skills.go`
  Hold the shared backend skill catalog, routing heuristics, and small helpers for skill labels and defaults.
- Modify: `workspace_agent_service.go`
  Route asks through explicit or automatic skills, call the right handler, and persist executed skill metadata on messages.
- Modify: `workspace_agent_service_test.go`
  Cover explicit skill execution, automatic routing, reader-specific routing, and build-memory behavior.
- Modify: `app.go`
  Pass the wiki service into the agent service and expose `ListWorkspaceAgentSkills()` to Wails.

### Frontend files

- Modify: `frontend/src/types/workspaceAgent.ts`
  Add shared skill definition types and routed execution metadata.
- Modify: `frontend/src/api/workspaceAgent.ts`
  Add `listSkills()` and return richer ask results.
- Modify: `frontend/src/store/workspaceAgentStore.ts`
  Cache the skill catalog and include selected skill state in ask flows.
- Modify: `frontend/src/components/WorkspaceAgentShell.tsx`
  Add manual skill buttons, auto/manual mode affordances, and executed-skill badges in the thread.
- Modify: `frontend/src/components/ReaderAIPanel.tsx`
  Let reader asks explicitly request `Ask with evidence`, `Reading outputs`, or `Cross-source synthesis`, while still allowing auto-routing.
- Modify: `frontend/src/style/workspace.css`
  Add styles for skill pills, selected state, and routed skill metadata.
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.d.ts`
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.js`
- Modify via `wails generate module`: `frontend/wailsjs/go/models.ts`

## Task 1: Add Backend Skill Definitions, Routing, And Execution

**Files:**
- Modify: `config_types.go`
- Create: `workspace_agent_skills.go`
- Modify: `workspace_agent_service.go`
- Modify: `workspace_agent_service_test.go`
- Modify: `app.go`

- [ ] **Step 1: Write the failing backend tests for explicit skills and auto-routing**

```go
func TestWorkspaceAgentServiceAskUsesExplicitTaskPlanningSkill(t *testing.T) {
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

	query := &stubWorkspaceAgentQuery{result: WorkspaceKnowledgeQueryResult{Answer: "Planned response."}}
	service := newWorkspaceAgentService(store, query, nil)

	result, err := service.Ask(ctx, WorkspaceAgentAskInput{
		WorkspaceID: workspace.ID,
		Surface:     string(WorkspaceAgentSurfaceWorkspace),
		SkillName:   string(WorkspaceAgentSkillTaskPlanning),
		Question:    "Plan the next experiments for this topic.",
	})
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if result.ExecutedSkill.Name != string(WorkspaceAgentSkillTaskPlanning) {
		t.Fatalf("ExecutedSkill.Name = %q, want %q", result.ExecutedSkill.Name, WorkspaceAgentSkillTaskPlanning)
	}
	if len(query.inputs) != 1 {
		t.Fatalf("len(query.inputs) = %d, want 1", len(query.inputs))
	}
	if !strings.Contains(query.inputs[0].Question, "Task planning mode") {
		t.Fatalf("query.inputs[0].Question = %q, want task-planning prompt prefix", query.inputs[0].Question)
	}
	if result.AssistantMessage.SkillName != string(WorkspaceAgentSkillTaskPlanning) {
		t.Fatalf("AssistantMessage.SkillName = %q, want %q", result.AssistantMessage.SkillName, WorkspaceAgentSkillTaskPlanning)
	}
}

func TestWorkspaceAgentServiceAskAutoRoutesReaderSummariesToReadingOutputs(t *testing.T) {
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

	query := &stubWorkspaceAgentQuery{result: WorkspaceKnowledgeQueryResult{Answer: "Reading note."}}
	service := newWorkspaceAgentService(store, query, nil)

	result, err := service.Ask(ctx, WorkspaceAgentAskInput{
		WorkspaceID:            workspace.ID,
		Surface:                string(WorkspaceAgentSurfaceReader),
		IncludeDocumentContext: true,
		DocumentID:             "doc-1",
		Question:               "Summarize this page into reading notes.",
	})
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if result.ExecutedSkill.Name != string(WorkspaceAgentSkillReadingOutputs) {
		t.Fatalf("ExecutedSkill.Name = %q, want %q", result.ExecutedSkill.Name, WorkspaceAgentSkillReadingOutputs)
	}
	if result.ExecutedSkill.RoutedBy != "auto" {
		t.Fatalf("ExecutedSkill.RoutedBy = %q, want auto", result.ExecutedSkill.RoutedBy)
	}
}

func TestWorkspaceAgentServiceAskUsesBuildMemorySkill(t *testing.T) {
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

	wiki := &stubWorkspaceAgentWiki{}
	service := newWorkspaceAgentService(store, &stubWorkspaceAgentQuery{}, wiki)

	result, err := service.Ask(ctx, WorkspaceAgentAskInput{
		WorkspaceID: workspace.ID,
		Surface:     string(WorkspaceAgentSurfaceWorkspace),
		SkillName:   string(WorkspaceAgentSkillBuildMemory),
		ProviderID:  7,
		ModelID:     11,
		Question:    "Build workspace memory now.",
	})
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if len(wiki.starts) != 1 {
		t.Fatalf("len(wiki.starts) = %d, want 1", len(wiki.starts))
	}
	if result.Query.Answer == "" {
		t.Fatal("Query.Answer is empty, want build status text")
	}
}
```

- [ ] **Step 2: Run the backend tests to verify they fail**

Run: `go test ./... -run "TestWorkspaceAgentServiceAskUsesExplicitTaskPlanningSkill|TestWorkspaceAgentServiceAskAutoRoutesReaderSummariesToReadingOutputs|TestWorkspaceAgentServiceAskUsesBuildMemorySkill" -count=1`

Expected: FAIL with compile errors such as `undefined: WorkspaceAgentSkillTaskPlanning`, missing `ExecutedSkill`, or a `newWorkspaceAgentService` signature mismatch.

- [ ] **Step 3: Add shared backend skill contracts**

```go
type WorkspaceAgentSkillName string

const (
	WorkspaceAgentSkillAskWithEvidence   WorkspaceAgentSkillName = "ask_with_evidence"
	WorkspaceAgentSkillReadingOutputs    WorkspaceAgentSkillName = "reading_outputs"
	WorkspaceAgentSkillTaskPlanning      WorkspaceAgentSkillName = "task_planning"
	WorkspaceAgentSkillBuildMemory       WorkspaceAgentSkillName = "build_memory"
	WorkspaceAgentSkillCrossSource       WorkspaceAgentSkillName = "cross_source_synthesis"
	WorkspaceAgentSkillPromoteToWiki     WorkspaceAgentSkillName = "promote_to_wiki"
	WorkspaceAgentSkillToolExecution     WorkspaceAgentSkillName = "tool_execution"
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
```

- [ ] **Step 4: Add the shared skill catalog and routing helpers**

```go
var workspaceAgentSkillCatalog = []WorkspaceAgentSkillDefinition{
	{Name: string(WorkspaceAgentSkillAskWithEvidence), Label: "Ask with evidence", Description: "Ground an answer in workspace memory and evidence.", ManualEnabled: true, AutoEnabled: true, ReaderEnabled: true},
	{Name: string(WorkspaceAgentSkillReadingOutputs), Label: "Reading outputs", Description: "Generate structured reading notes and summaries.", ManualEnabled: true, AutoEnabled: true, ReaderEnabled: true},
	{Name: string(WorkspaceAgentSkillTaskPlanning), Label: "Task planning", Description: "Turn a research goal into concrete next steps.", ManualEnabled: true, AutoEnabled: true, ReaderEnabled: false},
	{Name: string(WorkspaceAgentSkillBuildMemory), Label: "Build memory", Description: "Run the workspace memory build pipeline.", ManualEnabled: true, AutoEnabled: true, ReaderEnabled: false, WorkspaceOnly: true},
	{Name: string(WorkspaceAgentSkillCrossSource), Label: "Cross-source synthesis", Description: "Compare and synthesize across sources.", ManualEnabled: true, AutoEnabled: true, ReaderEnabled: true},
	{Name: string(WorkspaceAgentSkillPromoteToWiki), Label: "Promote to wiki", Description: "File useful output back into workspace memory.", ManualEnabled: false, AutoEnabled: false, ReaderEnabled: true},
	{Name: string(WorkspaceAgentSkillToolExecution), Label: "Tool execution", Description: "Run non-LLM tools when needed.", ManualEnabled: false, AutoEnabled: false, ReaderEnabled: false},
}

func listWorkspaceAgentSkills() []WorkspaceAgentSkillDefinition {
	result := make([]WorkspaceAgentSkillDefinition, len(workspaceAgentSkillCatalog))
	copy(result, workspaceAgentSkillCatalog)
	return result
}

func routeWorkspaceAgentSkill(input WorkspaceAgentAskInput) WorkspaceAgentExecutedSkill {
	explicit := strings.TrimSpace(input.SkillName)
	if explicit != "" {
		definition := getWorkspaceAgentSkillDefinition(explicit)
		return WorkspaceAgentExecutedSkill{Name: definition.Name, Label: definition.Label, RoutedBy: "manual", Reason: "user_selected", DisplayText: definition.Label}
	}

	question := strings.ToLower(strings.TrimSpace(input.Question))
	if strings.Contains(question, "build") && strings.Contains(question, "memory") {
		return newAutoWorkspaceAgentSkill(WorkspaceAgentSkillBuildMemory, "workspace_build_request")
	}
	if input.Surface == string(WorkspaceAgentSurfaceReader) && (strings.Contains(question, "summary") || strings.Contains(question, "notes")) {
		return newAutoWorkspaceAgentSkill(WorkspaceAgentSkillReadingOutputs, "reader_summary_request")
	}
	if strings.Contains(question, "plan") || strings.Contains(question, "next step") {
		return newAutoWorkspaceAgentSkill(WorkspaceAgentSkillTaskPlanning, "planning_language")
	}
	if strings.Contains(question, "compare") || strings.Contains(question, "across") || strings.Contains(question, "synthesis") {
		return newAutoWorkspaceAgentSkill(WorkspaceAgentSkillCrossSource, "cross_source_language")
	}
	return newAutoWorkspaceAgentSkill(WorkspaceAgentSkillAskWithEvidence, "default_grounded_answer")
}
```

- [ ] **Step 5: Route `workspaceAgentService` through skill handlers**

```go
type workspaceAgentWikiRunner interface {
	Start(ctx context.Context, input WorkspaceWikiScanStartInput) (WorkspaceWikiScanJob, error)
}

type workspaceAgentService struct {
	store *configStore
	query workspaceAgentQuery
	wiki  workspaceAgentWikiRunner
}

func newWorkspaceAgentService(store *configStore, query workspaceAgentQuery, wiki workspaceAgentWikiRunner) *workspaceAgentService {
	return &workspaceAgentService{store: store, query: query, wiki: wiki}
}

func (s *workspaceAgentService) ListSkills() []WorkspaceAgentSkillDefinition {
	return listWorkspaceAgentSkills()
}

func (s *workspaceAgentService) Ask(ctx context.Context, input WorkspaceAgentAskInput) (WorkspaceAgentAskResult, error) {
	...
	executedSkill := routeWorkspaceAgentSkill(input)
	queryResult, err := s.executeSkill(ctx, executedSkill, input, recentMessages)
	...
	createdAssistantMessage, appendErr := s.store.AppendWorkspaceAgentMessageTx(ctx, tx, WorkspaceAgentMessageCreateInput{
		SessionID:     session.ID,
		WorkspaceID:   workspaceID,
		Surface:       askSurface,
		Role:          string(WorkspaceAgentMessageRoleAssistant),
		Kind:          "answer",
		Prompt:        question,
		Content:       strings.TrimSpace(queryResult.Answer),
		SkillName:     executedSkill.Name,
		EvidenceCount: len(queryResult.Evidence),
	})
	...
	return WorkspaceAgentAskResult{
		Session:          session,
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
		ExecutedSkill:    executedSkill,
		Query:            queryResult,
	}, nil
}
```

```go
func (s *workspaceAgentService) executeSkill(ctx context.Context, skill WorkspaceAgentExecutedSkill, input WorkspaceAgentAskInput, recentMessages []WorkspaceAgentMessage) (WorkspaceKnowledgeQueryResult, error) {
	switch skill.Name {
	case string(WorkspaceAgentSkillBuildMemory):
		if s.wiki == nil {
			return WorkspaceKnowledgeQueryResult{}, fmt.Errorf("workspace wiki service is unavailable")
		}
		job, err := s.wiki.Start(ctx, WorkspaceWikiScanStartInput{
			WorkspaceID: input.WorkspaceID,
			ProviderID:  input.ProviderID,
			ModelID:     input.ModelID,
		})
		if err != nil {
			return WorkspaceKnowledgeQueryResult{}, err
		}
		return WorkspaceKnowledgeQueryResult{Answer: fmt.Sprintf("Started workspace memory build %s.", job.JobID)}, nil
	case string(WorkspaceAgentSkillTaskPlanning):
		return s.query.Query(ctx, WorkspaceKnowledgeQueryInput{
			WorkspaceID: input.WorkspaceID,
			ProviderID:  input.ProviderID,
			ModelID:     input.ModelID,
			Question:    buildWorkspaceAgentSkillPrompt("Task planning mode", input, recentMessages),
		})
	case string(WorkspaceAgentSkillReadingOutputs):
		return s.query.Query(ctx, WorkspaceKnowledgeQueryInput{
			WorkspaceID: input.WorkspaceID,
			ProviderID:  input.ProviderID,
			ModelID:     input.ModelID,
			Question:    buildWorkspaceAgentSkillPrompt("Reading outputs mode", input, recentMessages),
		})
	case string(WorkspaceAgentSkillCrossSource):
		return s.query.Query(ctx, WorkspaceKnowledgeQueryInput{
			WorkspaceID: input.WorkspaceID,
			ProviderID:  input.ProviderID,
			ModelID:     input.ModelID,
			Question:    buildWorkspaceAgentSkillPrompt("Cross-source synthesis mode", input, recentMessages),
		})
	default:
		return s.query.Query(ctx, WorkspaceKnowledgeQueryInput{
			WorkspaceID: input.WorkspaceID,
			ProviderID:  input.ProviderID,
			ModelID:     input.ModelID,
			Question:    buildWorkspaceAgentQueryQuestion(strings.TrimSpace(input.Question), input, recentMessages),
		})
	}
}
```

- [ ] **Step 6: Expose the backend skill list through Wails**

```go
func (a *App) startup(ctx context.Context) {
	...
	a.agent = newWorkspaceAgentService(store, a.query, a.wiki)
}

func (a *App) ListWorkspaceAgentSkills() []WorkspaceAgentSkillDefinition {
	if a.agent == nil {
		return []WorkspaceAgentSkillDefinition{}
	}
	return a.agent.ListSkills()
}
```

- [ ] **Step 7: Run the backend tests to verify the skill layer passes**

Run: `go test ./... -run "TestWorkspaceAgentServiceAskUsesExplicitTaskPlanningSkill|TestWorkspaceAgentServiceAskAutoRoutesReaderSummariesToReadingOutputs|TestWorkspaceAgentServiceAskUsesBuildMemorySkill|TestWorkspaceAgentServiceAskCreatesSessionAndMessages|TestWorkspaceAgentServiceAskUsesExistingSession|TestWorkspaceAgentServiceAskUsesRecentSessionHistoryInDelegatedQuery" -count=1`

Expected: PASS.

## Task 2: Surface The Shared Skill Catalog In The Workspace Shell

**Files:**
- Modify: `frontend/src/types/workspaceAgent.ts`
- Modify: `frontend/src/api/workspaceAgent.ts`
- Modify: `frontend/src/store/workspaceAgentStore.ts`
- Modify: `frontend/src/components/WorkspaceAgentShell.tsx`
- Modify: `frontend/src/style/workspace.css`
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.d.ts`
- Modify via `wails generate module`: `frontend/wailsjs/go/main/App.js`
- Modify via `wails generate module`: `frontend/wailsjs/go/models.ts`

- [ ] **Step 1: Write the failing frontend test or type-level assertions for skill metadata flow**

```ts
export interface WorkspaceAgentSkillDefinition {
  name: string;
  label: string;
  description: string;
  manualEnabled: boolean;
  autoEnabled: boolean;
  readerEnabled: boolean;
  workspaceOnly: boolean;
}

export interface WorkspaceAgentExecutedSkill {
  name: string;
  label: string;
  routedBy: 'manual' | 'auto';
  reason: string;
  displayText: string;
}
```

The failure here should come from TypeScript build errors until the API/store/component usage is updated to include `skills`, `selectedSkillName`, and `executedSkill`.

- [ ] **Step 2: Run the frontend build to verify it fails before implementation**

Run: `npm run build`

Expected: FAIL with TypeScript errors around missing `WorkspaceAgentSkillDefinition`, missing `ListWorkspaceAgentSkills`, or `result.executedSkill` not existing.

- [ ] **Step 3: Extend the frontend workspace-agent contracts and API**

```ts
interface WailsWorkspaceAgentApp {
  ListWorkspaceAgentSkills: () => Promise<WorkspaceAgentSkillDefinition[]>;
  ListWorkspaceAgentSessions: (workspaceId: string) => Promise<WorkspaceAgentSession[]>;
  ...
}

export const workspaceAgentApi = {
  async listSkills(): Promise<WorkspaceAgentSkillDefinition[]> {
    const app = getApp();
    if (!app) {
      return [];
    }
    return app.ListWorkspaceAgentSkills();
  },
  ...
};
```

- [ ] **Step 4: Cache the skill list and selected skill in the store**

```ts
interface WorkspaceAgentPaneState {
  sessions: WorkspaceAgentSession[];
  activeSessionId: string | null;
  messages: Record<string, WorkspaceAgentMessage[]>;
  skills: WorkspaceAgentSkillDefinition[];
  selectedSkillName: string | null;
  ...
}

interface WorkspaceAgentState {
  panes: Record<string, WorkspaceAgentPaneState>;
  ensureWorkspace: (workspaceId: string) => Promise<void>;
  setSelectedSkill: (workspaceId: string, skillName: string | null) => void;
  ...
}
```

- [ ] **Step 5: Add manual skill selection and executed-skill badges in the workspace shell**

```tsx
const manualSkills = (pane?.skills ?? []).filter((skill) => skill.manualEnabled);

<div className="workspace-agent-skill-strip">
  <button
    type="button"
    className={`workspace-agent-skill-pill ${!pane?.selectedSkillName ? 'workspace-agent-skill-pill-active' : ''}`}
    onClick={() => setSelectedSkill(workspace.id, null)}
  >
    Auto
  </button>
  {manualSkills.map((skill) => (
    <button
      key={skill.name}
      type="button"
      className={`workspace-agent-skill-pill ${pane?.selectedSkillName === skill.name ? 'workspace-agent-skill-pill-active' : ''}`}
      onClick={() => setSelectedSkill(workspace.id, skill.name)}
      title={skill.description}
    >
      {skill.label}
    </button>
  ))}
</div>
```

```tsx
await ask({
  workspaceId: workspace.id,
  sessionId: activeSessionId ?? '',
  surface: 'workspace',
  skillName: pane?.selectedSkillName ?? undefined,
  providerId: defaultProviderConfig.provider.id,
  modelId: defaultModel.id,
  question,
});
```

```tsx
{message.skillName ? <small>{formatSkillLabel(message.skillName)}</small> : null}
```

- [ ] **Step 6: Add CSS for skill pills and routed metadata**

```css
.workspace-agent-skill-strip {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.workspace-agent-skill-pill {
  border: 1px solid var(--border-color);
  background: var(--input-bg);
  color: var(--text-muted);
  border-radius: 999px;
  padding: 6px 10px;
}

.workspace-agent-skill-pill-active {
  color: var(--text-color);
  border-color: var(--accent-color);
}
```

- [ ] **Step 7: Regenerate Wails bindings and run the frontend build**

Run: `wails generate module && npm run build`

Expected: PASS, with only the existing non-blocking asset warnings if they still apply.

## Task 3: Extend Reader Asks To Use The Shared Skill System

**Files:**
- Modify: `frontend/src/components/ReaderAIPanel.tsx`
- Modify: `frontend/src/types/workspaceAgent.ts`
- Modify: `frontend/src/api/workspaceAgent.ts`

- [ ] **Step 1: Write the failing reader-facing type and behavior checks**

```ts
type ReaderAskScope = {
  selection: boolean;
  page: boolean;
  document: boolean;
  workspace: boolean;
};

type ReaderSkillName = 'ask_with_evidence' | 'reading_outputs' | 'cross_source_synthesis';
```

The failure should come from the component trying to pass `skillName` before the backend contract and local UI state support it.

- [ ] **Step 2: Run the frontend build to verify it fails before implementation**

Run: `npm run build`

Expected: FAIL with missing `skillName` state or incompatible ask payload types.

- [ ] **Step 3: Add reader skill mode UI and pass it into the shared ask payload**

```tsx
const [selectedSkillName, setSelectedSkillName] = useState<ReaderSkillName | null>(null);

const readerSkillOptions: Array<{ name: ReaderSkillName | null; label: string }> = [
  { name: null, label: 'Auto' },
  { name: 'ask_with_evidence', label: 'Ask' },
  { name: 'reading_outputs', label: 'Reading Output' },
  { name: 'cross_source_synthesis', label: 'Synthesis' },
];
```

```tsx
const result = await workspaceAgentApi.ask(buildReaderAskInput({
  workspaceId,
  sessionId,
  documentId,
  selection: selection.cleaned,
  activePage,
  llmProviderId,
  llmModelId,
  question: copilotState.question,
  skillName: selectedSkillName,
  scope: {
    selection: copilotState.scope.selection,
    page: copilotState.scope.page,
    document: copilotState.scope.document,
    workspace: copilotState.scope.workspace,
  },
}));
```

- [ ] **Step 4: Show the executed skill on returned answers**

```tsx
setCopilotState(prev => ({
  ...prev,
  isAsking: false,
  answer: result.query.answer,
  answerSkillLabel: result.executedSkill.label,
  evidence,
  candidates: result.query.candidates,
}));
```

- [ ] **Step 5: Run the full verification set**

Run: `go test ./... && npm run build`

Expected: PASS.

## Self-Review

- Spec coverage: this plan implements the missing `skill registry and routing layer` priority from the approved design, while deliberately leaving `Insights/Builds` browser deepening to a later plan.
- Placeholder scan: all tasks name exact files, commands, and concrete code shapes.
- Type consistency: backend `SkillName`, frontend `skillName`, and returned `executedSkill` stay aligned across the plan.
