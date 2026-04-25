import type {
  WorkspaceKnowledgeClaim,
  WorkspaceKnowledgeCompileSummary,
  WorkspaceKnowledgeEntity,
  WorkspaceKnowledgeSource,
  WorkspaceKnowledgeSourceRef,
  WorkspaceKnowledgeTask,
} from '../types/workspaceKnowledge';

interface WailsWorkspaceKnowledgeApp {
  ListWorkspaceKnowledgeSources: (workspaceId: string) => Promise<WorkspaceKnowledgeSource[]>;
  GetWorkspaceKnowledgeCompileSummary: (
    workspaceId: string
  ) => Promise<WorkspaceKnowledgeCompileSummary>;
  ListWorkspaceKnowledgeEntities: (workspaceId: string) => Promise<WorkspaceKnowledgeEntity[]>;
  ListWorkspaceKnowledgeClaims: (workspaceId: string) => Promise<WorkspaceKnowledgeClaim[]>;
  ListWorkspaceKnowledgeTasks: (workspaceId: string) => Promise<WorkspaceKnowledgeTask[]>;
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
  PromoteWorkspaceKnowledgeCandidates: (
    workspaceId: string,
    candidates: WorkspaceKnowledgeCandidate[]
  ) => Promise<void>;
}

export interface WorkspaceKnowledgeQueryResult {
  answer: string;
  evidence: WorkspaceKnowledgeEvidenceHit[];
  candidates: WorkspaceKnowledgeCandidate[];
}

export interface WorkspaceKnowledgeEvidenceHit {
  kind: "entity" | "claim" | "task" | "wiki_page" | "raw_excerpt";
  id: string;
  title: string;
  summary: string;
  excerpt: string;
  sourceRefs: WorkspaceKnowledgeSourceRef[];
  confidence?: number;
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
  confidence?: number;
}

function isWailsApp(value: unknown): value is { go: { main: { App: WailsWorkspaceKnowledgeApp } } } {
  return typeof value === 'object' && value !== null && 'go' in value;
}

function getApp(): WailsWorkspaceKnowledgeApp | null {
  if (typeof window !== 'undefined' && isWailsApp(window) && window.go?.main?.App) {
    return window.go.main.App;
  }
  return null;
}

export const workspaceKnowledgeApi = {
  async listSources(workspaceId: string): Promise<WorkspaceKnowledgeSource[]> {
    const app = getApp();
    if (!app || workspaceId.trim() === '') {
      return [];
    }
    return app.ListWorkspaceKnowledgeSources(workspaceId);
  },

  async getCompileSummary(workspaceId: string): Promise<WorkspaceKnowledgeCompileSummary | null> {
    const app = getApp();
    if (!app || workspaceId.trim() === '') {
      return null;
    }
    return app.GetWorkspaceKnowledgeCompileSummary(workspaceId);
  },

  async listEntities(workspaceId: string): Promise<WorkspaceKnowledgeEntity[]> {
    const app = getApp();
    if (!app || workspaceId.trim() === '') {
      return [];
    }
    return app.ListWorkspaceKnowledgeEntities(workspaceId);
  },

  async listClaims(workspaceId: string): Promise<WorkspaceKnowledgeClaim[]> {
    const app = getApp();
    if (!app || workspaceId.trim() === '') {
      return [];
    }
    return app.ListWorkspaceKnowledgeClaims(workspaceId);
  },

  async listTasks(workspaceId: string): Promise<WorkspaceKnowledgeTask[]> {
    const app = getApp();
    if (!app || workspaceId.trim() === '') {
      return [];
    }
    return app.ListWorkspaceKnowledgeTasks(workspaceId);
  },

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
