import type {
  WorkspaceKnowledgeClaim,
  WorkspaceKnowledgeEntity,
  WorkspaceKnowledgeTask,
} from '../types/workspaceKnowledge';

interface WailsWorkspaceKnowledgeApp {
  ListWorkspaceKnowledgeEntities: (workspaceId: string) => Promise<WorkspaceKnowledgeEntity[]>;
  ListWorkspaceKnowledgeClaims: (workspaceId: string) => Promise<WorkspaceKnowledgeClaim[]>;
  ListWorkspaceKnowledgeTasks: (workspaceId: string) => Promise<WorkspaceKnowledgeTask[]>;
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
};
