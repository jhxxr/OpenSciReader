import type {
  WorkspaceAgentAskInput,
  WorkspaceAgentAskResult,
  WorkspaceAgentMessage,
  WorkspaceAgentSkillDefinition,
  WorkspaceAgentSession,
} from '../types/workspaceAgent';

interface WailsWorkspaceAgentApp {
  ListWorkspaceAgentSkills: () => Promise<WorkspaceAgentSkillDefinition[]>;
  ListWorkspaceAgentSessions: (workspaceId: string) => Promise<WorkspaceAgentSession[]>;
  CreateWorkspaceAgentSession: (input: {
    workspaceId: string;
    title: string;
    surface: string;
  }) => Promise<WorkspaceAgentSession>;
  ListWorkspaceAgentMessagesForWorkspace: (workspaceId: string, sessionId: string) => Promise<WorkspaceAgentMessage[]>;
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

const workspaceIdBySessionId = new Map<string, string>();

function rememberSessions(sessions: WorkspaceAgentSession[]): WorkspaceAgentSession[] {
  sessions.forEach((session) => {
    if (session.id && session.workspaceId) {
      workspaceIdBySessionId.set(session.id, session.workspaceId);
    }
  });
  return sessions;
}

function rememberSession(session: WorkspaceAgentSession): WorkspaceAgentSession {
  if (session.id && session.workspaceId) {
    workspaceIdBySessionId.set(session.id, session.workspaceId);
  }
  return session;
}

export const workspaceAgentApi = {
  async listSkills(): Promise<WorkspaceAgentSkillDefinition[]> {
    const app = getApp();
    if (!app) {
      throw new Error('workspace agent API is unavailable');
    }
    return app.ListWorkspaceAgentSkills();
  },

  async listSessions(workspaceId: string): Promise<WorkspaceAgentSession[]> {
    const trimmedWorkspaceId = workspaceId.trim();
    if (trimmedWorkspaceId === '') {
      return [];
    }
    const app = getApp();
    if (!app) {
      throw new Error('workspace agent API is unavailable');
    }
    return rememberSessions(await app.ListWorkspaceAgentSessions(trimmedWorkspaceId));
  },

  async createSession(workspaceId: string, title: string, surface: string): Promise<WorkspaceAgentSession> {
    const app = getApp();
    if (!app || workspaceId.trim() === '') {
      throw new Error('workspace agent API is unavailable');
    }
    return rememberSession(await app.CreateWorkspaceAgentSession({ workspaceId, title, surface }));
  },

  async listMessages(sessionId: string): Promise<WorkspaceAgentMessage[]> {
    const trimmedSessionId = sessionId.trim();
    if (trimmedSessionId === '') {
      return [];
    }
    const workspaceId = workspaceIdBySessionId.get(trimmedSessionId)?.trim() ?? '';
    if (workspaceId === '') {
      return [];
    }
    const app = getApp();
    if (!app) {
      throw new Error('workspace agent API is unavailable');
    }
    return app.ListWorkspaceAgentMessagesForWorkspace(workspaceId, trimmedSessionId);
  },

  async ask(input: WorkspaceAgentAskInput): Promise<WorkspaceAgentAskResult> {
    const app = getApp();
    if (!app) {
      throw new Error('workspace agent API is unavailable');
    }
    const result = await app.AskWorkspaceAgent(input);
    rememberSession(result.session);
    return result;
  },
};
