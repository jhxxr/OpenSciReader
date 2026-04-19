# Reader Sidebar Copilot Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Completely rewrite ReaderAIPanel to replace prompt-driven workbench with knowledge-grounded copilot using four-section layout (Ask/Answer/Evidence/Promote).

**Architecture:** Single-file React component rewrite with internal state management, API integration to workspace knowledge query service, and grouped/collapsible evidence display.

**Tech Stack:** React hooks, TypeScript, Lucide icons, existing MarkdownPreview component, workspaceKnowledgeApi

---

## File Structure

### Files to Modify
- `frontend/src/components/ReaderAIPanel.tsx` - Complete rewrite (1671 lines → ~400-500 lines)

### Files to Reference
- `frontend/src/api/workspaceKnowledge.ts` - Query and promote APIs
- `frontend/src/types/workspaceKnowledge.ts` - Type definitions
- `frontend/src/components/MarkdownPreview.tsx` - Answer rendering
- `frontend/src/store/readerStore.ts` - Reader state (selection, activePage, snapshot)
- `frontend/src/components/ui/Button.tsx` - Button component

### Files to Update (CSS)
- `frontend/src/App.css` - Add copilot-specific styles

---

## Task 1: Create frontend API method for workspace knowledge query

**Files:**
- Modify: `frontend/src/api/workspaceKnowledge.ts`

- [ ] **Step 1: Add query method signature**

```typescript
interface WailsWorkspaceKnowledgeApp {
  // ... existing methods
  QueryWorkspaceKnowledge: (
    workspaceId: string,
    providerId: number,
    modelId: number,
    question: string,
    scopeSelection: string,
    scopeCurrentPage: number,
    scopeDocumentId: string,
    scopeWorkspaceContext: boolean
  ) => Promise<WorkspaceKnowledgeQueryResult>;
}

interface WorkspaceKnowledgeQueryResult {
  answer: string;
  evidence: WorkspaceKnowledgeEvidenceHit[];
  candidates: WorkspaceKnowledgeCandidate[];
}
```

- [ ] **Step 2: Add evidence and candidate types**

```typescript
export interface WorkspaceKnowledgeEvidenceHit {
  kind: "entity" | "claim" | "task" | "wiki_page" | "raw_excerpt";
  id: string;
  title: string;
  summary: string;
  excerpt: string;
  sourceRefs: WorkspaceKnowledgeSourceRef[];
}

export interface WorkspaceKnowledgeCandidate {
  id: string;
  title: string;
  type: string;
  summary: string;
  aliases: string[];
  entityIds: string[];
  priority: string;
  sourceId: string;
  pageStart: number;
  pageEnd: number;
  excerpt: string;
  sourceRefs: WorkspaceKnowledgeSourceRef[];
}
```

- [ ] **Step 3: Add query method to API object**

```typescript
export const workspaceKnowledgeApi = {
  // ... existing methods

  async queryWorkspaceKnowledge(
    workspaceId: string,
    providerId: number | null,
    modelId: number | null,
    question: string,
    scope: {
      selection?: string;
      currentPage?: number;
      documentId?: string;
      workspaceContext?: boolean;
    }
  ): Promise<WorkspaceKnowledgeQueryResult | null> {
    const app = getApp();
    if (!app || workspaceId.trim() === '' || !question.trim()) {
      return null;
    }
    return app.QueryWorkspaceKnowledge(
      workspaceId,
      providerId ?? 0,
      modelId ?? 0,
      question,
      scope.selection ?? '',
      scope.currentPage ?? 0,
      scope.documentId ?? '',
      scope.workspaceContext ?? false
    );
  },
};
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/workspaceKnowledge.ts
git commit -m "feat: add queryWorkspaceKnowledge API method"
```

---

## Task 2: Add promote candidates API method

**Files:**
- Modify: `frontend/src/api/workspaceKnowledge.ts`

- [ ] **Step 1: Add promote method signature**

```typescript
interface WailsWorkspaceKnowledgeApp {
  // ... existing methods
  PromoteWorkspaceKnowledgeCandidates: (
    workspaceId: string,
    candidates: WorkspaceKnowledgeCandidate[]
  ) => Promise<void>;
}
```

- [ ] **Step 2: Add promote method to API object**

```typescript
export const workspaceKnowledgeApi = {
  // ... existing methods

  async promoteCandidates(
    workspaceId: string,
    candidates: WorkspaceKnowledgeCandidate[]
  ): Promise<void> {
    const app = getApp();
    if (!app || workspaceId.trim() === '' || candidates.length === 0) {
      return;
    }
    return app.PromoteWorkspaceKnowledgeCandidates(workspaceId, candidates);
  },
};
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/workspaceKnowledge.ts
git commit -m "feat: add promoteCandidates API method"
```

---

## Task 3: Add copilot CSS styles

**Files:**
- Modify: `frontend/src/App.css`

- [ ] **Step 1: Add ask section styles**

```css
.reader-ask-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border-bottom: 1px solid var(--osr-border-light);
}

.reader-ask-textarea {
  width: 100%;
  min-height: 80px;
  padding: 12px;
  border: 1px solid var(--osr-border-light);
  border-radius: var(--osr-radius-md);
  background: var(--osr-bg-sidebar);
  color: var(--osr-text-primary);
  font-size: 14px;
  resize: vertical;
}

.reader-ask-textarea:focus {
  outline: none;
  border-color: var(--osr-accent);
}

.scope-checkboxes {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.scope-checkbox {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--osr-text-secondary);
  cursor: pointer;
}

.scope-checkbox input[type="checkbox"] {
  accent-color: var(--osr-accent);
}
```

- [ ] **Step 2: Add answer section styles**

```css
.reader-answer-section {
  padding: 16px;
  border-bottom: 1px solid var(--osr-border-light);
  min-height: 100px;
}

.reader-answer-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  color: var(--osr-text-muted);
}

.reader-answer-empty {
  text-align: center;
  padding: 32px;
  color: var(--osr-text-muted);
  font-size: 13px;
}
```

- [ ] **Step 3: Add evidence section styles**

```css
.reader-evidence-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  border-bottom: 1px solid var(--osr-border-light);
}

.evidence-group {
  border: 1px solid var(--osr-border-light);
  border-radius: var(--osr-radius-md);
  overflow: hidden;
}

.evidence-group-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fbfa 100%);
  border-bottom: 1px solid var(--osr-border-light);
}

.evidence-group-header h4 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
}

.evidence-group-count {
  background: var(--osr-accent);
  color: white;
  padding: 2px 8px;
  border-radius: var(--osr-radius-full);
  font-size: 11px;
  font-weight: 500;
}

.evidence-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  max-height: 300px;
  overflow-y: auto;
}

.evidence-item {
  padding: 10px;
  border-radius: var(--osr-radius-md);
  background: var(--osr-bg-sidebar);
  border: 1px solid var(--osr-border-light);
}

.evidence-item-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--osr-text-primary);
  margin-bottom: 4px;
}

.evidence-item-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-bottom: 6px;
}

.evidence-item-summary {
  font-size: 12px;
  line-height: 1.5;
  color: var(--osr-text-secondary);
  margin-bottom: 6px;
}

.evidence-item-source {
  font-size: 11px;
  color: var(--osr-text-muted);
}
```

- [ ] **Step 4: Add promote section styles**

```css
.reader-promote-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
}

.candidate-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  border: 1px solid var(--osr-border-light);
  border-radius: var(--osr-radius-md);
  background: var(--osr-bg-sidebar);
}

.candidate-summary {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.candidate-summary-main {
  flex: 1;
  min-width: 0;
}

.candidate-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--osr-text-primary);
  margin-bottom: 4px;
}

.candidate-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.candidate-actions {
  display: flex;
  gap: 6px;
  margin-top: 4px;
}

.candidate-details {
  padding-top: 8px;
  border-top: 1px solid var(--osr-border-light);
}

.candidate-aliases {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}

.alias-pill {
  display: inline-flex;
  align-items: center;
  padding: 3px 8px;
  border-radius: var(--osr-radius-full);
  background: rgba(13, 155, 118, 0.08);
  color: var(--osr-accent);
  font-size: 11px;
  font-weight: 500;
}

.candidate-source-footnote {
  font-size: 11px;
  color: var(--osr-text-muted);
}
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/App.css
git commit -m "feat: add reader sidebar copilot CSS styles"
```

---

## Task 4: Implement ReaderAIPanel skeleton with imports and types

**Files:**
- Modify: `frontend/src/components/ReaderAIPanel.tsx`

- [ ] **Step 1: Update imports**

```typescript
import { useEffect, useMemo, useRef, useState } from "react";
import {
  Copy,
  Download,
  History,
  RefreshCw,
  Settings2,
  Sparkles,
  Trash2,
} from "lucide-react";
import { EventsOff, EventsOn } from "../../wailsjs/runtime/runtime";
import { configApi } from "../api/config";
import { gatewayApi } from "../api/gateway";
import { workspaceKnowledgeApi } from "../api/workspaceKnowledge";
import { historyApi } from "../api/history";
import {
  loadPDFTextChunks,
  loadPDFTextContext,
  type PDFTextChunk,
} from "../lib/pdfContext";
import {
  buildPaperImageGenerationPrompt,
  buildPaperImageSummaryInstruction,
} from "../lib/paperFigurePrompt";
import { useReaderStore } from "../store/readerStore";
import type {
  AIWorkspaceConfig,
  ModelRecord,
  ProviderConfig,
} from "../types/config";
import { DEFAULT_AI_WORKSPACE_CONFIG } from "../types/config";
import type { GatewayStreamEvent } from "../types/gateway";
import type { ChatHistoryEntry } from "../types/history";
import type { TabItem } from "../store/tabStore";
import type {
  WorkspaceKnowledgeQueryResult,
  WorkspaceKnowledgeEvidenceHit,
  WorkspaceKnowledgeCandidate,
} from "../types/workspaceKnowledge";
import { MarkdownPreview } from "./MarkdownPreview";
import { Button } from "./ui/Button";
```

- [ ] **Step 2: Add component state interfaces**

```typescript
interface CopilotState {
  question: string;
  scope: {
    selection: boolean;
    page: boolean;
    document: boolean;
    workspace: boolean;
  };
  isAsking: boolean;
  answer: string | null;
  answerError: string | null;
  evidence: {
    entities: WorkspaceKnowledgeEvidenceHit[];
    claims: WorkspaceKnowledgeEvidenceHit[];
    tasks: WorkspaceKnowledgeEvidenceHit[];
    sources: WorkspaceKnowledgeEvidenceHit[];
  };
  expandedGroups: {
    entities: boolean;
    claims: boolean;
    tasks: boolean;
    sources: boolean;
  };
  candidates: WorkspaceKnowledgeCandidate[];
  expandedCandidates: Set<string>;
  promotingIds: Set<string>;
  promoteError: string | null;
}
```

- [ ] **Step 3: Add helper function signatures**

```typescript
function getErrorMessage(error: unknown, context: string): string;
function renderConfidenceBadge(confidence: number): React.ReactNode | null;
function formatSourceSummary(sourceRefs: any[]): string;
function formatSourceRefLabel(sourceRef: any): string;
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/ReaderAIPanel.tsx
git commit -m "feat: add ReaderAIPanel skeleton with imports and types"
```

---

## Task 5: Implement helper functions for formatting and error handling

**Files:**
- Modify: `frontend/src/components/ReaderAIPanel.tsx`

- [ ] **Step 1: Implement getErrorMessage**

```typescript
function getErrorMessage(error: unknown, context: string): string {
  if (error instanceof Error) {
    return `${context}: ${error.message}`;
  }
  if (typeof error === 'string') {
    return `${context}: ${error}`;
  }
  return context;
}
```

- [ ] **Step 2: Implement renderConfidenceBadge**

```typescript
function renderConfidenceBadge(confidence: number): React.ReactNode | null {
  if (!Number.isFinite(confidence) || confidence <= 0) {
    return null;
  }
  return <span className="badge badge-accent">{Math.round(confidence * 100)}%</span>;
}
```

- [ ] **Step 3: Implement formatSourceSummary**

```typescript
function formatSourceSummary(sourceRefs: any[]): string {
  if (!sourceRefs || sourceRefs.length === 0) {
    return 'No source anchors';
  }

  const labels = sourceRefs.slice(0, 2).map(formatSourceRefLabel).filter(Boolean);
  const suffix = sourceRefs.length > 2 ? ` +${sourceRefs.length - 2}` : '';
  return `Sources: ${sourceRefs.length}${labels.length > 0 ? ` (${labels.join(' / ')}${suffix})` : ''}`;
}
```

- [ ] **Step 4: Implement formatSourceRefLabel**

```typescript
function formatSourceRefLabel(sourceRef: any): string {
  if (sourceRef.pageStart > 0 && sourceRef.pageEnd > 0) {
    return sourceRef.pageStart === sourceRef.pageEnd
      ? `p.${sourceRef.pageStart}`
      : `pp.${sourceRef.pageStart}-${sourceRef.pageEnd}`;
  }
  if (sourceRef.pageStart > 0) {
    return `p.${sourceRef.pageStart}`;
  }
  return sourceRef.sourceId || '';
}
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/ReaderAIPanel.tsx
git commit -m "feat: add helper functions for copilot formatting"
```

---

## Task 6: Implement Ask section component

**Files:**
- Modify: `frontend/src/components/ReaderAIPanel.tsx`

- [ ] **Step 1: Add Ask state initialization**

```typescript
export function ReaderAIPanel({ tab, llmConfigs, drawingConfigs, activeLLMConfig, activeLLMModel, llmProviderId, llmModelId, setLlmProviderId, setLlmModelId }: ReaderAIPanelProps) {
  const selection = useReaderStore((state) => state.selection);
  const activePage = useReaderStore((state) => state.activePage);
  const [copilotState, setCopilotState] = useState<CopilotState>({
    question: '',
    scope: { selection: false, page: false, document: false, workspace: false },
    isAsking: false,
    answer: null,
    answerError: null,
    evidence: { entities: [], claims: [], tasks: [], sources: [] },
    expandedGroups: { entities: false, claims: false, tasks: false, sources: false },
    candidates: [],
    expandedCandidates: new Set(),
    promotingIds: new Set(),
    promoteError: null,
  });

  const workspaceId = useMemo(() => tab.workspaceId || '', [tab.workspaceId]);
  const documentId = useMemo(() => tab.documentId || '', [tab.documentId]);
```

- [ ] **Step 2: Add Ask section JSX**

```typescript
const askSection = (
  <div className="reader-ask-section">
    <textarea
      className="reader-ask-textarea"
      value={copilotState.question}
      onChange={(e) => setCopilotState(prev => ({ ...prev, question: e.target.value }))}
      placeholder="向工作区知识提问..."
      disabled={copilotState.isAsking}
    />

    <div className="scope-checkboxes">
      <label className="scope-checkbox">
        <input
          type="checkbox"
          checked={copilotState.scope.selection}
          disabled={!selection || copilotState.isAsking}
          onChange={(e) => setCopilotState(prev => ({
            ...prev,
            scope: { ...prev.scope, selection: e.target.checked }
          }))}
        />
        当前选区
      </label>
      <label className="scope-checkbox">
        <input
          type="checkbox"
          checked={copilotState.scope.page}
          disabled={copilotState.isAsking}
          onChange={(e) => setCopilotState(prev => ({
            ...prev,
            scope: { ...prev.scope, page: e.target.checked }
          }))}
        />
        当前页面
      </label>
      <label className="scope-checkbox">
        <input
          type="checkbox"
          checked={copilotState.scope.document}
          disabled={copilotState.isAsking}
          onChange={(e) => setCopilotState(prev => ({
            ...prev,
            scope: { ...prev.scope, document: e.target.checked }
          }))}
        />
        当前文档
      </label>
      <label className="scope-checkbox">
        <input
          type="checkbox"
          checked={copilotState.scope.workspace}
          disabled={copilotState.isAsking}
          onChange={(e) => setCopilotState(prev => ({
            ...prev,
            scope: { ...prev.scope, workspace: e.target.checked }
          }))}
        />
        工作区上下文
      </label>
    </div>

    <Button
      variant="secondary"
      size="sm"
      onClick={() => handleAsk()}
      disabled={copilotState.isAsking || !copilotState.question.trim()}
    >
      {copilotState.isAsking ? '提问中...' : 'Ask'}
    </Button>
  </div>
);
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/ReaderAIPanel.tsx
git commit -m "feat: implement Ask section component"
```

---

## Task 7: Implement handleAsk function for query execution

**Files:**
- Modify: `frontend/src/components/ReaderAIPanel.tsx`

- [ ] **Step 1: Implement handleAsk function**

```typescript
async function handleAsk() {
  if (!workspaceId || !activeLLMModel || copilotState.isAsking) {
    return;
  }

  setCopilotState(prev => ({ ...prev, isAsking: true, answerError: null }));

  try {
    const result = await workspaceKnowledgeApi.queryWorkspaceKnowledge(
      workspaceId,
      llmProviderId,
      llmModelId,
      copilotState.question,
      {
        selection: copilotState.scope.selection ? selection : undefined,
        currentPage: copilotState.scope.page ? activePage : undefined,
        documentId: copilotState.scope.document ? documentId : undefined,
        workspaceContext: copilotState.scope.workspace,
      }
    );

    if (!result) {
      setCopilotState(prev => ({
        ...prev,
        isAsking: false,
        answer: null,
        answerError: '无法连接到知识服务',
      }));
      return;
    }

    // Group evidence by kind
    const evidence = {
      entities: result.evidence.filter(e => e.kind === 'entity'),
      claims: result.evidence.filter(e => e.kind === 'claim'),
      tasks: result.evidence.filter(e => e.kind === 'task'),
      sources: result.evidence.filter(e => e.kind === 'wiki_page' || e.kind === 'raw_excerpt'),
    };

    setCopilotState(prev => ({
      ...prev,
      isAsking: false,
      answer: result.answer,
      evidence,
      candidates: result.candidates,
    }));
  } catch (error) {
    setCopilotState(prev => ({
      ...prev,
      isAsking: false,
      answer: null,
      answerError: getErrorMessage(error, '查询失败'),
    }));
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/ReaderAIPanel.tsx
git commit -m "feat: implement handleAsk query execution"
```

---

## Task 8: Implement Answer section component

**Files:**
- Modify: `frontend/src/components/ReaderAIPanel.tsx`

- [ ] **Step 1: Add Answer section JSX**

```typescript
const answerSection = (
  <div className="reader-answer-section">
    {copilotState.isAsking && (
      <div className="reader-answer-loading">
        <RefreshCw size={20} className="spin-inline" />
        <span style={{ marginLeft: 8 }}>正在查询知识库...</span>
      </div>
    )}

    {!copilotState.isAsking && copilotState.answerError && (
      <div className="reader-error">{copilotState.answerError}</div>
    )}

    {!copilotState.isAsking && !copilotState.answerError && !copilotState.answer && (
      <div className="reader-answer-empty">
        <Sparkles size={24} />
        <p>向工作区知识提问获取答案</p>
      </div>
    )}

    {!copilotState.isAsking && !copilotState.answerError && copilotState.answer && (
      <MarkdownPreview content={copilotState.answer} />
    )}
  </div>
);
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/ReaderAIPanel.tsx
git commit -m "feat: implement Answer section component"
```

---

## Task 9: Implement Evidence section component with grouped layout

**Files:**
- Modify: `frontend/src/components/ReaderAIPanel.tsx`

- [ ] **Step 1: Add Evidence section JSX**

```typescript
const evidenceSection = (
  <div className="reader-evidence-section">
    {['entities', 'claims', 'tasks', 'sources'].map((group) => (
      <div key={group} className="evidence-group">
        <div className="evidence-group-header">
          <h4>
            {group === 'entities' && 'Entities / Concepts'}
            {group === 'claims' && 'Key Claims'}
            {group === 'tasks' && 'Open Questions / Tasks'}
            {group === 'sources' && 'Sources'}
          </h4>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span className="evidence-group-count">{copilotState.evidence[group as keyof typeof copilotState.evidence].length}</span>
            <Button variant="ghost" size="sm" onClick={() => toggleGroup(group as keyof typeof copilotState.expandedGroups)}>
              {copilotState.expandedGroups[group as keyof typeof copilotState.expandedGroups] ? '收起' : '展开'}
            </Button>
          </div>
        </div>

        {copilotState.expandedGroups[group as keyof typeof copilotState.expandedGroups] && (
          <div className="evidence-list">
            {copilotState.evidence[group as keyof typeof copilotState.evidence].map((item) => (
              <div key={item.id} className="evidence-item">
                <div className="evidence-item-title">{item.title}</div>
                <div className="evidence-item-meta">
                  {renderConfidenceBadge(item.confidence || 0)}
                  <span className="badge">{item.kind}</span>
                </div>
                <div className="evidence-item-summary">{item.summary || item.excerpt}</div>
                <div className="evidence-item-source">{formatSourceSummary(item.sourceRefs)}</div>
              </div>
            ))}
          </div>
        )}

        {!copilotState.expandedGroups[group as keyof typeof copilotState.expandedGroups] && copilotState.evidence[group as keyof typeof copilotState.evidence].length === 0 && (
          <div style={{ padding: '12px', color: 'var(--osr-text-muted)', fontSize: '12px' }}>
            {group === 'entities' && '暂无实体或概念'}
            {group === 'claims' && '暂无关键主张'}
            {group === 'tasks' && '暂无开放问题或任务'}
            {group === 'sources' && '暂无来源'}
          </div>
        )}
      </div>
    ))}
  </div>
);
```

- [ ] **Step 2: Add toggleGroup helper**

```typescript
function toggleGroup(group: keyof CopilotState['expandedGroups']) {
  setCopilotState(prev => ({
    ...prev,
    expandedGroups: {
      ...prev.expandedGroups,
      [group]: !prev.expandedGroups[group],
    },
  }));
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/ReaderAIPanel.tsx
git commit -m "feat: implement Evidence section with grouped layout"
```

---

## Task 10: Implement Promote section component with expandable candidates

**Files:**
- Modify: `frontend/src/components/ReaderAIPanel.tsx`

- [ ] **Step 1: Add Promote section JSX**

```typescript
const promoteSection = (
  <div className="reader-promote-section">
    <h3>Candidate Memories</h3>
    {copilotState.promoteError && <div className="reader-error">{copilotState.promoteError}</div>}

    {copilotState.candidates.length === 0 && !copilotState.promoteError && (
      <p className="empty-inline">No candidate memories extracted.</p>
    )}

    {copilotState.candidates.map((candidate) => (
      <div key={candidate.id} className="candidate-item">
        <div className="candidate-summary">
          <div className="candidate-summary-main">
            <div className="candidate-title">{candidate.title}</div>
            <div className="candidate-meta">
              <span className="badge">{candidate.type}</span>
              {renderConfidenceBadge(candidate.confidence || 0)}
            </div>
          </div>
        </div>

        {copilotState.expandedCandidates.has(candidate.id) && (
          <div className="candidate-details">
            <p>{candidate.summary}</p>
            {candidate.aliases && candidate.aliases.length > 0 && (
              <div className="candidate-aliases">
                {candidate.aliases.slice(0, 3).map((alias) => (
                  <span key={alias} className="alias-pill">{alias}</span>
                ))}
                {candidate.aliases.length > 3 && (
                  <span className="alias-pill">+{candidate.aliases.length - 3}</span>
                )}
              </div>
            )}
            {candidate.entityIds && candidate.entityIds.length > 0 && (
              <small>Linked entities: {candidate.entityIds.length}</small>
            )}
            <small className="candidate-source-footnote">{formatSourceSummary(candidate.sourceRefs)}</small>
          </div>
        )}

        <div className="candidate-actions">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => handlePromote(candidate.id)}
            disabled={copilotState.promotingIds.has(candidate.id)}
          >
            {copilotState.promotingIds.has(candidate.id) ? 'Promoting...' : 'Promote'}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => toggleCandidateExpand(candidate.id)}
          >
            {copilotState.expandedCandidates.has(candidate.id) ? '收起' : '展开'}
          </Button>
        </div>
      </div>
    ))}
  </div>
);
```

- [ ] **Step 2: Add toggleCandidateExpand helper**

```typescript
function toggleCandidateExpand(candidateId: string) {
  setCopilotState(prev => {
    const newExpanded = new Set(prev.expandedCandidates);
    if (newExpanded.has(candidateId)) {
      newExpanded.delete(candidateId);
    } else {
      newExpanded.add(candidateId);
    }
    return { ...prev, expandedCandidates: newExpanded };
  });
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/ReaderAIPanel.tsx
git commit -m "feat: implement Promote section with expandable candidates"
```

---

## Task 11: Implement handlePromote function

**Files:**
- Modify: `frontend/src/components/ReaderAIPanel.tsx`

- [ ] **Step 1: Implement handlePromote function**

```typescript
async function handlePromote(candidateId: string) {
  if (!workspaceId || copilotState.promotingIds.has(candidateId)) {
    return;
  }

  setCopilotState(prev => {
    const newPromoting = new Set(prev.promotingIds);
    newPromoting.add(candidateId);
    return { ...prev, promotingIds: newPromoting, promoteError: null };
  });

  try {
    const candidate = copilotState.candidates.find(c => c.id === candidateId);
    if (!candidate) {
      throw new Error('Candidate not found');
    }

    await workspaceKnowledgeApi.promoteCandidates(workspaceId, [candidate]);

    // Remove promoted candidate from list
    setCopilotState(prev => ({
      ...prev,
      candidates: prev.candidates.filter(c => c.id !== candidateId),
      promotingIds: (() => {
        const next = new Set(prev.promotingIds);
        next.delete(candidateId);
        return next;
      })(),
    }));
  } catch (error) {
    setCopilotState(prev => {
      const nextPromoting = new Set(prev.promotingIds);
      nextPromoting.delete(candidateId);
      return {
        ...prev,
        promotingIds: nextPromoting,
        promoteError: getErrorMessage(error, 'Promote failed'),
      };
    });
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/ReaderAIPanel.tsx
git commit -m "feat: implement handlePromote function"
```

---

## Task 12: Assemble full component with conditional rendering

**Files:**
- Modify: `frontend/src/components/ReaderAIPanel.tsx`

- [ ] **Step 1: Add conditional render logic**

```typescript
export function ReaderAIPanel({ tab, llmConfigs, drawingConfigs, activeLLMConfig, activeLLMModel, llmProviderId, llmModelId, setLlmProviderId, setLlmModelId }: ReaderAIPanelProps) {
  // ... existing state and helper functions

  // Check if we should show copilot mode
  const shouldShowCopilot = useMemo(() => {
    return workspaceId && activeLLMModel;
  }, [workspaceId, activeLLMModel]);

  // Legacy mode fallback (not implemented in this version)
  if (!shouldShowCopilot) {
    return (
      <div className="reader-error">
        请先选择工作区并配置 AI 模型以使用知识助手。
      </div>
    );
  }

  return (
    <div className="reader-ai-panel">
      {askSection}
      {answerSection}
      {evidenceSection}
      {promoteSection}
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/ReaderAIPanel.tsx
git commit -m "feat: assemble full copilot component with conditional rendering"
```

---

## Task 13: Verify build and run tests

**Files:**
- Test: `frontend/`

- [ ] **Step 1: Run TypeScript compilation**

```bash
cd frontend
npm run build
```

Expected: PASS with no TypeScript errors

- [ ] **Step 2: Verify no console errors**

Check browser console for runtime errors when loading ReaderAIPanel.

- [ ] **Step 3: Manual test the copilot flow**

1. Open a document in a workspace
2. Type a question and select scopes
3. Click Ask
4. Verify Answer displays correctly
5. Verify Evidence groups are collapsible
6. Promote a candidate memory
7. Verify promoted memory is removed from list

- [ ] **Step 4: Commit final version**

```bash
git add frontend/src/components/ReaderAIPanel.tsx
git commit -m "feat: complete reader sidebar copilot implementation"
```

---

## Testing Strategy

### Manual Testing Checklist

- [ ] Ask with no scopes selected → should still query with workspace context
- [ ] Ask with selection scope + no text selected → selection checkbox disabled
- [ ] Ask with page scope → passes page number to API
- [ ] Ask with document scope → passes document ID to API
- [ ] Ask during active query → Ask button disabled
- [ ] Empty result → displays appropriate empty state
- [ ] API error → displays error message
- [ ] Expand/collapse Evidence groups → state persists correctly
- [ ] Expand/collapse candidate → details show/hide
- [ ] Promote single candidate → candidate removed from list
- [ ] Promote during error → candidate kept for retry
- [ ] Multiple concurrent promotes → each handled independently
- [ ] Markdown rendering → answer displays with proper formatting

### Integration Points

- ReaderTab passes correct props (tab, llmConfigs, etc.)
- ReaderStore provides selection and activePage
- workspaceKnowledgeApi calls backend correctly
- Errors are formatted and displayed to user
