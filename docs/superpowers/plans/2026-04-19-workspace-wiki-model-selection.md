# Workspace Wiki Model Selection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add provider/model selection UI to Workspace Wiki panel with per-workspace persistence.

**Architecture:** Extend AIWorkspaceConfig with WikiScanProviderID/WikiScanModelID fields; JSON storage means no DB schema change. Frontend adds two dropdowns in WorkspaceTab; selection auto-saves with debounce.

**Tech Stack:** Go backend with JSON-config storage; React frontend with existing config API.

---

## File Structure

| File | Responsibility |
|------|----------------|
| `config_types.go` | Define WikiScanProviderID/WikiScanModelID in AIWorkspaceConfig struct |
| `config_store.go` | Initialize defaults and normalize new fields |
| `frontend/src/types/config.ts` | TypeScript type for AIWorkspaceConfig with new fields |
| `frontend/src/components/WorkspaceTab.tsx` | Provider/model dropdowns, state, selection logic |
| `frontend/src/App.tsx` | Pass selection to onStartWikiScan callback |

---

### Task 1: Backend - Add Config Fields

**Files:**
- Modify: `config_types.go:167-179`
- Modify: `config_store.go:1612-1635, 1669-1676`

- [ ] **Step 1: Add WikiScan fields to AIWorkspaceConfig struct**

In `config_types.go`, modify the `AIWorkspaceConfig` struct (line 167):

```go
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
```

- [ ] **Step 2: Add defaults in defaultAIWorkspaceConfig**

In `config_store.go`, modify `defaultAIWorkspaceConfig` (line 1612), add two new fields after `DrawingModel`:

```go
func defaultAIWorkspaceConfig() AIWorkspaceConfig {
	return AIWorkspaceConfig{
		SummaryMode:          "auto",
		SummaryChunkPages:    6,
		SummaryChunkMaxChars: 18000,
		AutoRestoreCount:     3,
		TableTemplate: `| 维度 | 内容 |
| --- | --- |
| 论文标题 | |
| 研究问题 | |
| 核心方法 | |
| 数据/实验设置 | |
| 关键结果 | |
| 创新点 | |
| 局限性 | |
| 我能直接借鉴什么 | |`,
		TablePrompt:         "请仔细阅读当前论文，并严格按照给定的 Markdown 表格模板填写。要求：1. 只输出填好的表格。2. 所有单元格用中文填写。3. 若原文未明确提及，填写"未明确说明"。4. 内容应简洁但能支持快速比较论文。",
		CustomPromptDraft:   "",
		FollowUpPromptDraft: "",
		DrawingPromptDraft:  "根据当前论文内容，生成一张适合组会汇报的科研概念图，突出问题、方法流程、关键结果和应用价值。",
		DrawingProviderID:   0,
		DrawingModel:        "gemini-3-pro-image-preview",
		WikiScanProviderID:  0,
		WikiScanModelID:     0,
	}
}
```

- [ ] **Step 3: Add normalization in normalizeAIWorkspaceConfig**

In `config_store.go`, modify `normalizeAIWorkspaceConfig` (around line 1669), add after DrawingModel normalization:

```go
	if input.DrawingProviderID > 0 {
		config.DrawingProviderID = input.DrawingProviderID
	}
	if trimmed := strings.TrimSpace(input.DrawingModel); trimmed != "" {
		config.DrawingModel = trimmed
	}
	if input.WikiScanProviderID > 0 {
		config.WikiScanProviderID = input.WikiScanProviderID
	}
	if input.WikiScanModelID > 0 {
		config.WikiScanModelID = input.WikiScanModelID
	}

	return config
```

- [ ] **Step 4: Run build to verify**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 5: Commit backend changes**

```bash
git add config_types.go config_store.go
git commit -m "feat: add WikiScanProviderID/WikiScanModelID to AIWorkspaceConfig"
```

---

### Task 2: Frontend - Update TypeScript Types

**Files:**
- Modify: `frontend/src/types/config.ts:107-130`
- Modify: `frontend/wailsjs/go/models.ts` (auto-generated, verify after wails dev)

- [ ] **Step 1: Add WikiScan fields to AIWorkspaceConfig type**

In `frontend/src/types/config.ts`, modify `AIWorkspaceConfig` interface (line 93):

```typescript
export interface AIWorkspaceConfig {
  summaryMode: AISummaryMode;
  summaryChunkPages: number;
  summaryChunkMaxChars: number;
  autoRestoreCount: number;
  tableTemplate: string;
  tablePrompt: string;
  customPromptDraft: string;
  followUpPromptDraft: string;
  drawingPromptDraft: string;
  drawingProviderId: number;
  drawingModel: string;
  wikiScanProviderId: number;
  wikiScanModelId: number;
}
```

- [ ] **Step 2: Update DEFAULT_AI_WORKSPACE_CONFIG**

In `frontend/src/types/config.ts`, modify `DEFAULT_AI_WORKSPACE_CONFIG` (line 107):

```typescript
export const DEFAULT_AI_WORKSPACE_CONFIG: AIWorkspaceConfig = {
  summaryMode: "auto",
  summaryChunkPages: 6,
  summaryChunkMaxChars: 18000,
  autoRestoreCount: 3,
  tableTemplate: `| 维度 | 内容 |
| --- | --- |
| 论文标题 | |
| 研究问题 | |
| 核心方法 | |
| 数据/实验设置 | |
| 关键结果 | |
| 创新点 | |
| 局限性 | |
| 我能直接借鉴什么 | |`,
  tablePrompt:
    "请仔细阅读当前论文，并严格按照给定的 Markdown 表格模板填写。要求：1. 只输出填好的表格。2. 所有单元格用中文填写。3. 若原文未明确提及，填写"未明确说明"。4. 内容应简洁但能支持快速比较论文。",
  customPromptDraft: "",
  followUpPromptDraft: "",
  drawingPromptDraft:
    "额外要求：图中文字尽量使用简体中文，整体像一页适合组会汇报的科研海报。",
  drawingProviderId: 0,
  drawingModel: "gemini-3-pro-image-preview",
  wikiScanProviderId: 0,
  wikiScanModelId: 0,
};
```

- [ ] **Step 3: Commit frontend type changes**

```bash
git add frontend/src/types/config.ts
git commit -m "feat: add wikiScanProviderId/wikiScanModelId to AIWorkspaceConfig type"
```

---

### Task 3: Frontend - Add Model Selection UI to WorkspaceTab

**Files:**
- Modify: `frontend/src/components/WorkspaceTab.tsx`

- [ ] **Step 1: Add interface for new props**

In `frontend/src/components/WorkspaceTab.tsx`, modify the `WorkspaceTabProps` interface (line 12), add after `isDeletingWikiPages`:

```typescript
interface WorkspaceTabProps {
  workspace: Workspace | null;
  documents: DocumentRecord[];
  isImporting: boolean;
  isLoadingDocuments: boolean;
  deletingDocumentId: string | null;
  llmProviderConfigs: ProviderConfig[];
  wikiPages: WorkspaceWikiPage[];
  selectedWikiPageId: string | null;
  wikiPageContent: WorkspaceWikiPageContent | null;
  isLoadingWikiPages: boolean;
  isLoadingWikiPageContent: boolean;
  activeWikiJob: WorkspaceWikiScanJob | null;
  wikiError: string | null;
  isStartingWikiScan: boolean;
  isCancellingWikiScan: boolean;
  isDeletingWikiPages: boolean;
  wikiScanProviderId: number;
  wikiScanModelId: number;
  onImportFiles: () => Promise<void>;
  onRefreshDocuments: () => Promise<void>;
  onOpenPdf: (document: DocumentRecord) => void;
  onDeleteDocument: (document: Pick<DocumentRecord, 'id' | 'workspaceId' | 'title' | 'originalFileName'>) => Promise<void>;
  onStartWikiScan: (providerId: number, modelId: number) => Promise<void>;
  onCancelWikiScan: () => Promise<void>;
  onRefreshWikiPages: () => Promise<void>;
  onSelectWikiPage: (pageId: string) => Promise<void>;
  onDeleteWikiPages: () => Promise<void>;
  onChangeWikiScanModel: (providerId: number, modelId: number) => void;
}
```

- [ ] **Step 2: Update function signature and add internal state**

Modify the function signature (line 40) and add provider selection state:

```typescript
export function WorkspaceTab({
  workspace,
  documents,
  isImporting,
  isLoadingDocuments,
  deletingDocumentId,
  llmProviderConfigs,
  wikiPages,
  selectedWikiPageId,
  wikiPageContent,
  isLoadingWikiPages,
  isLoadingWikiPageContent,
  activeWikiJob,
  wikiError,
  isStartingWikiScan,
  isCancellingWikiScan,
  isDeletingWikiPages,
  wikiScanProviderId,
  wikiScanModelId,
  onImportFiles,
  onRefreshDocuments,
  onOpenPdf,
  onDeleteDocument,
  onStartWikiScan,
  onCancelWikiScan,
  onRefreshWikiPages,
  onSelectWikiPage,
  onDeleteWikiPages,
  onChangeWikiScanModel,
}: WorkspaceTabProps) {
  const hasRunnableModel = llmProviderConfigs.some((item) => item.models.length > 0);
  
  const [localWikiScanProviderId, setLocalWikiScanProviderId] = useState<number>(wikiScanProviderId);
  const [localWikiScanModelId, setLocalWikiScanModelId] = useState<number>(wikiScanModelId);
  
  const selectedPage =
    wikiPageContent?.page?.id === selectedWikiPageId
      ? wikiPageContent.page
      : wikiPages.find((page) => page.id === selectedWikiPageId) ?? wikiPageContent?.page ?? null;
  const activeProgress = Math.max(0, Math.min(100, (activeWikiJob?.overallProgress ?? 0) * 100));
```

- [ ] **Step 3: Add useEffect to sync incoming props to local state**

Add after the selectedPage computation:

```typescript
  useEffect(() => {
    setLocalWikiScanProviderId(wikiScanProviderId);
  }, [wikiScanProviderId]);

  useEffect(() => {
    setLocalWikiScanModelId(wikiScanModelId);
  }, [wikiScanModelId]);
```

- [ ] **Step 4: Add provider/model selection helper logic**

Add helper functions before the early return for null workspace:

```typescript
  const llmProvidersWithModels = useMemo(
    () => llmProviderConfigs.filter((item) => item.provider.type === 'llm' && item.models.length > 0),
    [llmProviderConfigs]
  );

  const selectedProviderConfig = useMemo(
    () => llmProvidersWithModels.find((item) => item.provider.id === localWikiScanProviderId) ?? null,
    [llmProvidersWithModels, localWikiScanProviderId]
  );

  const selectedModelRecord = useMemo(
    () => selectedProviderConfig?.models.find((model) => model.id === localWikiScanModelId) ?? null,
    [selectedProviderConfig, localWikiScanModelId]
  );

  const handleProviderChange = useCallback((event: React.ChangeEvent<HTMLSelectElement>) => {
    const newProviderId = Number(event.target.value);
    setLocalWikiScanProviderId(newProviderId);
    const provider = llmProvidersWithModels.find((item) => item.provider.id === newProviderId);
    const firstModelId = provider?.models[0]?.id ?? 0;
    setLocalWikiScanModelId(firstModelId);
    onChangeWikiScanModel(newProviderId, firstModelId);
  }, [llmProvidersWithModels, onChangeWikiScanModel]);

  const handleModelChange = useCallback((event: React.ChangeEvent<HTMLSelectElement>) => {
    const newModelId = Number(event.target.value);
    setLocalWikiScanModelId(newModelId);
    onChangeWikiScanModel(localWikiScanProviderId, newModelId);
  }, [localWikiScanProviderId, onChangeWikiScanModel]);

  const handleStartScan = useCallback(() => {
    if (localWikiScanProviderId > 0 && localWikiScanModelId > 0) {
      void onStartWikiScan(localWikiScanProviderId, localWikiScanModelId);
    }
  }, [localWikiScanProviderId, localWikiScanModelId, onStartWikiScan]);
```

- [ ] **Step 5: Add imports**

Add at the top of the file (line 1):

```typescript
import { useCallback, useEffect, useMemo, useState } from 'react';
import { FileText, LoaderCircle, RefreshCw, Sparkles, SquareX, Trash2 } from 'lucide-react';
```

- [ ] **Step 6: Replace Scan button onClick and disabled logic**

Find the Scan button (around line 163) and modify:

```typescript
                <Button 
                  variant="secondary" 
                  size="sm" 
                  onClick={handleStartScan} 
                  disabled={isStartingWikiScan || isCancellingWikiScan || !selectedProviderConfig || !selectedModelRecord}
                >
                  <Sparkles size={14} />
                  {isStartingWikiScan ? 'Starting...' : 'Scan'}
                </Button>
```

- [ ] **Step 7: Add provider/model dropdowns UI**

In the Workspace Wiki panel section (around line 156-183), add dropdowns before the Scan button:

```typescript
        <div className="workspace-side-panels">
          <div className="workspace-panel workspace-wiki-status-panel">
            <div className="section-header workspace-panel-header">
              <div>
                <h3>Workspace Wiki</h3>
                <p className="workspace-panel-description">Run a workspace scan to build overview and per-document wiki pages.</p>
              </div>
              <div className="workspace-panel-header-actions home-workspace-actions home-workspace-actions-stacked">
                <div className="wiki-model-selector">
                  <select
                    className="wiki-provider-select"
                    value={localWikiScanProviderId}
                    onChange={handleProviderChange}
                    disabled={llmProvidersWithModels.length === 0 || isStartingWikiScan || activeWikiJob?.status === 'running'}
                  >
                    {llmProvidersWithModels.length === 0 ? (
                      <option value={0}>No LLM provider</option>
                    ) : localWikiScanProviderId === 0 ? (
                      <option value={0}>Select provider</option>
                    ) : null}
                    {llmProvidersWithModels.map((item) => (
                      <option key={item.provider.id} value={item.provider.id}>
                        {item.provider.name}
                      </option>
                    ))}
                  </select>
                  <select
                    className="wiki-model-select"
                    value={localWikiScanModelId}
                    onChange={handleModelChange}
                    disabled={!selectedProviderConfig || isStartingWikiScan || activeWikiJob?.status === 'running'}
                  >
                    {!selectedProviderConfig ? (
                      <option value={0}>Select model</option>
                    ) : null}
                    {selectedProviderConfig?.models.map((model) => (
                      <option key={model.id} value={model.id}>
                        {model.modelId}
                      </option>
                    ))}
                  </select>
                </div>
                <Button variant="secondary" size="sm" onClick={handleStartScan} disabled={isStartingWikiScan || isCancellingWikiScan || !selectedProviderConfig || !selectedModelRecord}>
                  <Sparkles size={14} />
                  {isStartingWikiScan ? 'Starting...' : 'Scan'}
                </Button>
```

- [ ] **Step 8: Update the error message**

Replace the "Configure at least one LLM provider" message (around line 182):

```typescript
            {!selectedProviderConfig || !selectedModelRecord ? (
              <p className="empty-inline">
                {llmProvidersWithModels.length === 0
                  ? 'Configure at least one LLM provider with a model before starting a workspace wiki scan.'
                  : '请先为当前工作区选择扫描模型'}
              </p>
            ) : null}
```

- [ ] **Step 9: Add model info to running status display**

In the activeWikiJob status section (around line 186-204), add model info to the display:

```typescript
            {activeWikiJob ? (
              <>
                <div className="home-task-stage">
                  <div className="home-task-stage-head">
                    <strong>{activeWikiJob.message || 'Scanning workspace content'}</strong>
                    <span>{activeProgress.toFixed(1)}%</span>
                  </div>
                  <p>{activeWikiJob.currentItem || activeWikiJob.currentStage || 'Preparing workspace wiki scan...'}</p>
                  {selectedProviderConfig && selectedModelRecord ? (
                    <small className="wiki-scan-model-info">
                      Using {selectedProviderConfig.provider.name} / {selectedModelRecord.modelId}
                    </small>
                  ) : null}
                </div>
```

- [ ] **Step 10: Run frontend build to verify**

Run: `cd frontend && npm run build`
Expected: No TypeScript errors

- [ ] **Step 11: Commit WorkspaceTab changes**

```bash
git add frontend/src/components/WorkspaceTab.tsx
git commit -m "feat: add provider/model selection UI to WorkspaceTab"
```

---

### Task 4: Frontend - Update App.tsx Integration

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Add state for wiki scan model selection**

In App.tsx, find the workspaceTabWiki state definition and add wikiScanProviderId/wikiScanModelId fields. Search for the `workspaceTabWiki` state setter pattern and add new fields:

```typescript
  const [workspaceTabWiki, setWorkspaceTabWiki] = useState<Record<string, {
    pages: WorkspaceWikiPage[];
    selectedPageId: string | null;
    pageContent: WorkspaceWikiPageContent | null;
    isLoadingPages: boolean;
    isLoadingPageContent: boolean;
    activeJob: WorkspaceWikiScanJob | null;
    wikiError: string | null;
    isStarting: boolean;
    isCancelling: boolean;
    isDeleting: boolean;
    unsubscribe: (() => void) | null;
    wikiScanProviderId: number;
    wikiScanModelId: number;
  }>>({});
```

- [ ] **Step 2: Update ensureWorkspaceTabWiki helper**

Find the `ensureWorkspaceTabWiki` function and add the new fields with defaults:

```typescript
  async function ensureWorkspaceTabWiki(workspaceId: string) {
    const config = await configApi.getAIWorkspaceConfig(workspaceId);
    setWorkspaceTabWiki((current) => ({
      ...current,
      [workspaceId]: {
        pages: current[workspaceId]?.pages ?? [],
        selectedPageId: current[workspaceId]?.selectedPageId ?? null,
        pageContent: current[workspaceId]?.pageContent ?? null,
        isLoadingPages: false,
        isLoadingPageContent: false,
        activeJob: current[workspaceId]?.activeJob ?? null,
        wikiError: null,
        isStarting: false,
        isCancelling: false,
        isDeleting: false,
        unsubscribe: current[workspaceId]?.unsubscribe ?? null,
        wikiScanProviderId: config.wikiScanProviderId ?? 0,
        wikiScanModelId: config.wikiScanModelId ?? 0,
      },
    }));
  }
```

- [ ] **Step 3: Add onChangeWikiScanModel handler**

Add a new handler function after the existing wiki-related handlers:

```typescript
  function handleChangeWikiScanModel(workspaceId: string, providerId: number, modelId: number) {
    setWorkspaceTabWiki((current) => ({
      ...current,
      [workspaceId]: {
        ...current[workspaceId],
        wikiScanProviderId: providerId,
        wikiScanModelId: modelId,
      },
    }));
    
    void configApi.saveAIWorkspaceConfig(workspaceId, {
      ...DEFAULT_AI_WORKSPACE_CONFIG,
      wikiScanProviderId: providerId,
      wikiScanModelId: modelId,
    }).catch((error) => {
      console.error('Failed to save wiki scan model config:', error);
    });
  }
```

- [ ] **Step 4: Update onStartWikiScan callback in ReaderTab render**

Find where ReaderTab passes `onStartWikiScan` and modify to accept providerId/modelId parameters (around line 740):

```typescript
                  onStartWikiScan={async (providerId: number, modelId: number) => {
                    if (!tab.workspaceId) {
                      return;
                    }
                    if (!providerId || !modelId) {
                      setWorkspaceTabWiki((current) => ({
                        ...current,
                        [tab.workspaceId!]: {
                          ...current[tab.workspaceId!],
                          wikiError: '请先为当前工作区选择扫描模型',
                        },
                      }));
                      return;
                    }
                    setWorkspaceTabWiki((current) => ({
                      ...current,
                      [tab.workspaceId!]: {
                        ...current[tab.workspaceId!],
                        wikiError: null,
                        isStarting: true,
                      },
                    }));
                    const job = await workspaceWikiApi.start({ workspaceId: tab.workspaceId, providerId, modelId });
```

- [ ] **Step 5: Pass new props to WorkspaceTab**

Find where WorkspaceTab is rendered and add the new props:

```typescript
                <WorkspaceTab
                  workspace={workspace.workspaces.find((w) => w.id === workspace.activeWorkspaceId) ?? null}
                  documents={workspaceDocuments[workspace.activeWorkspaceId ?? ''] ?? []}
                  isImporting={workspace.isImporting}
                  isLoadingDocuments={workspace.isLoadingDocuments}
                  deletingDocumentId={null}
                  llmProviderConfigs={llmProviderConfigs}
                  wikiPages={workspaceTabWiki[workspace.activeWorkspaceId ?? '']?.pages ?? []}
                  selectedWikiPageId={workspaceTabWiki[workspace.activeWorkspaceId ?? '']?.selectedPageId ?? null}
                  wikiPageContent={workspaceTabWiki[workspace.activeWorkspaceId ?? '']?.pageContent ?? null}
                  isLoadingWikiPages={workspaceTabWiki[workspace.activeWorkspaceId ?? '']?.isLoadingPages ?? false}
                  isLoadingWikiPageContent={workspaceTabWiki[workspace.activeWorkspaceId ?? '']?.isLoadingPageContent ?? false}
                  activeWikiJob={workspaceTabWiki[workspace.activeWorkspaceId ?? '']?.activeJob ?? null}
                  wikiError={workspaceTabWiki[workspace.activeWorkspaceId ?? '']?.wikiError ?? null}
                  isStartingWikiScan={workspaceTabWiki[workspace.activeWorkspaceId ?? '']?.isStarting ?? false}
                  isCancellingWikiScan={workspaceTabWiki[workspace.activeWorkspaceId ?? '']?.isCancelling ?? false}
                  isDeletingWikiPages={workspaceTabWiki[workspace.activeWorkspaceId ?? '']?.isDeleting ?? false}
                  wikiScanProviderId={workspaceTabWiki[workspace.activeWorkspaceId ?? '']?.wikiScanProviderId ?? 0}
                  wikiScanModelId={workspaceTabWiki[workspace.activeWorkspaceId ?? '']?.wikiScanModelId ?? 0}
                  onImportFiles={() => workspace.importFiles(workspace.activeWorkspaceId ?? '')}
                  onRefreshDocuments={() => workspace.refreshDocuments(workspace.activeWorkspaceId ?? '')}
                  onOpenPdf={(doc) => handleOpenDocument(doc)}
                  onDeleteDocument={(doc) => handleDeleteWorkspaceDocument(doc)}
                  onStartWikiScan={(providerId, modelId) => handleStartWikiScan(workspace.activeWorkspaceId ?? '', providerId, modelId)}
                  onCancelWikiScan={() => handleCancelWikiScan(workspace.activeWorkspaceId ?? '')}
                  onRefreshWikiPages={() => ensureWorkspaceTabWiki(workspace.activeWorkspaceId ?? '')}
                  onSelectWikiPage={(pageId) => handleSelectWikiPage(workspace.activeWorkspaceId ?? '', pageId)}
                  onDeleteWikiPages={() => handleDeleteWikiPages(workspace.activeWorkspaceId ?? '')}
                  onChangeWikiScanModel={(providerId, modelId) => handleChangeWikiScanModel(workspace.activeWorkspaceId ?? '', providerId, modelId)}
                />
```

- [ ] **Step 6: Run frontend build**

Run: `cd frontend && npm run build`
Expected: No TypeScript errors

- [ ] **Step 7: Commit App.tsx changes**

```bash
git add frontend/src/App.tsx
git commit -m "feat: integrate wiki scan model selection in App.tsx"
```

---

### Task 5: CSS Styling for Model Selector

**Files:**
- Modify: `frontend/src/style/workspace.css` (or appropriate stylesheet)

- [ ] **Step 1: Add CSS for wiki model selector**

Add styles for the new dropdown containers:

```css
.wiki-model-selector {
  display: flex;
  gap: 8px;
  align-items: center;
}

.wiki-provider-select,
.wiki-model-select {
  padding: 4px 8px;
  border-radius: 4px;
  border: 1px solid var(--border-color);
  background: var(--input-bg);
  color: var(--text-color);
  font-size: 12px;
  min-width: 120px;
}

.wiki-provider-select:focus,
.wiki-model-select:focus {
  outline: none;
  border-color: var(--accent-color);
}

.wiki-scan-model-info {
  color: var(--text-muted);
  margin-top: 4px;
}
```

- [ ] **Step 2: Commit styling changes**

```bash
git add frontend/src/style/workspace.css
git commit -m "feat: add CSS for wiki model selector dropdowns"
```

---

### Task 6: Integration Test

- [ ] **Step 1: Start development server**

Run: `wails dev`

- [ ] **Step 2: Manual verification**

1. Open the app
2. Go to Workspace tab
3. Verify provider/model dropdowns appear in Wiki panel
4. Select a provider - model dropdown should filter to that provider's models
5. Click Scan - verify it starts with selected model
6. Switch to another workspace - verify selection persists
7. Return to first workspace - verify selection restored

- [ ] **Step 3: Verify error handling**

1. Delete the selected model from settings
2. Return to workspace - verify selection auto-resets to first available

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat: workspace wiki model selection complete"
```

---

## Summary

This plan adds provider/model selection to the Workspace Wiki scan panel:
1. Backend: Extend AIWorkspaceConfig struct and defaults
2. Frontend types: Add TypeScript interface fields
3. WorkspaceTab: Provider/model dropdowns with local state
4. App.tsx: Integration and persistence handlers
5. CSS: Styling for new dropdowns
6. Manual testing for functionality and edge cases

No database schema changes required - config stored as JSON blob.