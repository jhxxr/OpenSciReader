import { create } from 'zustand';
import { workspaceAgentApi } from '../api/workspaceAgent';
import { getErrorMessage } from '../lib/errors';
import type {
  WorkspaceAgentAskInput,
  WorkspaceAgentAskResult,
  WorkspaceAgentExecutedSkill,
  WorkspaceAgentMessage,
  WorkspaceAgentSkillDefinition,
  WorkspaceAgentSession,
} from '../types/workspaceAgent';

interface WorkspaceAgentPaneState {
  sessions: WorkspaceAgentSession[];
  activeSessionId: string | null;
  messages: Record<string, WorkspaceAgentMessage[]>;
  executedSkills: Record<string, Record<number, WorkspaceAgentExecutedSkill>>;
  skills: WorkspaceAgentSkillDefinition[];
  skillsLoaded: boolean;
  selectedSkillName: string | null;
  loading: boolean;
  asking: boolean;
  error: string | null;
  sessionRequestToken: number;
  messageRequestToken: number;
}

interface WorkspaceAgentState {
  panes: Record<string, WorkspaceAgentPaneState>;
  ensureWorkspace: (workspaceId: string) => Promise<void>;
  setSelectedSkill: (workspaceId: string, skillName: string | null) => void;
  createSession: (workspaceId: string, title: string, surface: string) => Promise<WorkspaceAgentSession>;
  loadMessages: (workspaceId: string, sessionId: string) => Promise<void>;
  ask: (input: WorkspaceAgentAskInput) => Promise<WorkspaceAgentAskResult>;
}

const EMPTY_PANE: WorkspaceAgentPaneState = {
  sessions: [],
  activeSessionId: null,
  messages: {},
  executedSkills: {},
  skills: [],
  skillsLoaded: false,
  selectedSkillName: null,
  loading: false,
  asking: false,
  error: null,
  sessionRequestToken: 0,
  messageRequestToken: 0,
};

function executedSkillsFromMessages(messages: WorkspaceAgentMessage[]): Record<number, WorkspaceAgentExecutedSkill> {
  const executedSkills: Record<number, WorkspaceAgentExecutedSkill> = {};
  messages.forEach((message) => {
    if (message.executedSkill) {
      executedSkills[message.id] = message.executedSkill;
    }
  });
  return executedSkills;
}

function getPane(state: WorkspaceAgentState, workspaceId: string): WorkspaceAgentPaneState {
  return state.panes[workspaceId] ?? EMPTY_PANE;
}

function setPane(
  panes: Record<string, WorkspaceAgentPaneState>,
  workspaceId: string,
  next: Partial<WorkspaceAgentPaneState>,
): Record<string, WorkspaceAgentPaneState> {
  const current = panes[workspaceId] ?? EMPTY_PANE;
  return {
    ...panes,
    [workspaceId]: {
      ...current,
      ...next,
    },
  };
}

export const useWorkspaceAgentStore = create<WorkspaceAgentState>((set, get) => ({
  panes: {},

  async ensureWorkspace(workspaceId) {
    const trimmedWorkspaceId = workspaceId.trim();
    if (!trimmedWorkspaceId) {
      return;
    }
    const existingPane = getPane(get(), trimmedWorkspaceId);
    const shouldLoadSkills = !existingPane.skillsLoaded;
    set((state) => ({
      panes: setPane(state.panes, trimmedWorkspaceId, {
        loading: true,
        error: null,
        sessionRequestToken: (state.panes[trimmedWorkspaceId]?.sessionRequestToken ?? 0) + 1,
      }),
    }));
    const requestToken = getPane(get(), trimmedWorkspaceId).sessionRequestToken;
    try {
      const sessions = await workspaceAgentApi.listSessions(trimmedWorkspaceId);
      let skills = existingPane.skills;
      let skillsLoaded = existingPane.skillsLoaded;
      if (shouldLoadSkills) {
        try {
          skills = await workspaceAgentApi.listSkills();
          skillsLoaded = true;
        } catch {
          skillsLoaded = false;
        }
      }
      let nextActiveSessionId: string | null = null;
      let shouldLoadMessages = false;
      set((state) => {
        const pane = state.panes[trimmedWorkspaceId] ?? EMPTY_PANE;
        if (pane.sessionRequestToken != requestToken) {
          return state;
        }
        nextActiveSessionId =
          pane.activeSessionId && sessions.some((session) => session.id === pane.activeSessionId)
            ? pane.activeSessionId
            : sessions[0]?.id ?? null;
        shouldLoadMessages = Boolean(nextActiveSessionId && !pane.messages[nextActiveSessionId]);
        return {
          panes: setPane(state.panes, trimmedWorkspaceId, {
            sessions,
            skills,
            skillsLoaded,
            activeSessionId: nextActiveSessionId,
            loading: false,
            error: null,
          }),
        };
      });
      if (nextActiveSessionId) {
        if (shouldLoadMessages) {
          await get().loadMessages(trimmedWorkspaceId, nextActiveSessionId);
        }
      }
    } catch (error) {
      set((state) => ({
        panes:
          (state.panes[trimmedWorkspaceId]?.sessionRequestToken ?? 0) === requestToken
            ? setPane(state.panes, trimmedWorkspaceId, {
                loading: false,
                error: getErrorMessage(error, 'Failed to load workspace sessions'),
              })
            : state.panes,
      }));
    }
  },

  setSelectedSkill(workspaceId, skillName) {
    const trimmedWorkspaceId = workspaceId.trim();
    if (!trimmedWorkspaceId) {
      return;
    }
    const trimmedSkillName = skillName?.trim() ?? '';
    set((state) => ({
      panes: setPane(state.panes, trimmedWorkspaceId, { selectedSkillName: trimmedSkillName || null }),
    }));
  },

  async createSession(workspaceId, title, surface) {
    const trimmedWorkspaceId = workspaceId.trim();
    const trimmedTitle = title.trim();
    set((state) => ({
      panes: setPane(state.panes, trimmedWorkspaceId, { loading: true, error: null }),
    }));
    try {
      const session = await workspaceAgentApi.createSession(trimmedWorkspaceId, trimmedTitle, surface);
      set((state) => {
        const pane = state.panes[trimmedWorkspaceId] ?? EMPTY_PANE;
        return {
          panes: setPane(state.panes, trimmedWorkspaceId, {
            sessions: [session, ...pane.sessions.filter((item) => item.id !== session.id)],
            activeSessionId: session.id,
            messages: { ...pane.messages, [session.id]: [] },
            executedSkills: { ...pane.executedSkills, [session.id]: {} },
            loading: false,
            error: null,
          }),
        };
      });
      return session;
    } catch (error) {
      set((state) => ({
        panes: setPane(state.panes, trimmedWorkspaceId, {
          loading: false,
          error: getErrorMessage(error, 'Failed to create workspace session'),
        }),
      }));
      throw error;
    }
  },

  async loadMessages(workspaceId, sessionId) {
    const trimmedWorkspaceId = workspaceId.trim();
    const trimmedSessionId = sessionId.trim();
    if (!trimmedWorkspaceId || !trimmedSessionId) {
      return;
    }
    set((state) => ({
      panes: setPane(state.panes, trimmedWorkspaceId, {
        activeSessionId: trimmedSessionId,
        loading: true,
        error: null,
        messageRequestToken: (state.panes[trimmedWorkspaceId]?.messageRequestToken ?? 0) + 1,
      }),
    }));
    const requestToken = getPane(get(), trimmedWorkspaceId).messageRequestToken;
    try {
      const messages = await workspaceAgentApi.listMessages(trimmedSessionId);
      set((state) => {
        const pane = state.panes[trimmedWorkspaceId] ?? EMPTY_PANE;
        const loadedExecutedSkills = executedSkillsFromMessages(messages);
        if (pane.messageRequestToken !== requestToken) {
          return state;
        }
        return {
          panes: setPane(state.panes, trimmedWorkspaceId, {
            activeSessionId: trimmedSessionId,
            messages: { ...pane.messages, [trimmedSessionId]: messages },
            executedSkills: { ...pane.executedSkills, [trimmedSessionId]: loadedExecutedSkills },
            loading: false,
            error: null,
          }),
        };
      });
    } catch (error) {
      set((state) => ({
        panes:
          (state.panes[trimmedWorkspaceId]?.messageRequestToken ?? 0) === requestToken
            ? setPane(state.panes, trimmedWorkspaceId, {
                loading: false,
                error: getErrorMessage(error, 'Failed to load workspace messages'),
              })
            : state.panes,
      }));
    }
  },

  async ask(input) {
    const workspaceId = input.workspaceId.trim();
    if (!workspaceId) {
      throw new Error('workspace id is required');
    }
    set((state) => ({
      panes: setPane(state.panes, workspaceId, { asking: true, error: null }),
    }));
    try {
      const result = await workspaceAgentApi.ask(input);
      set((state) => {
        const pane = state.panes[workspaceId] ?? EMPTY_PANE;
        const existingMessages = pane.messages[result.session.id] ?? [];
        const nextMessages = existingMessages.filter(
          (message) => message.id !== result.userMessage.id && message.id !== result.assistantMessage.id,
        );
        return {
          panes: setPane(state.panes, workspaceId, {
            sessions: [result.session, ...pane.sessions.filter((session) => session.id !== result.session.id)],
            activeSessionId: result.session.id,
            messages: {
              ...pane.messages,
              [result.session.id]: [...nextMessages, result.userMessage, result.assistantMessage],
            },
            executedSkills: {
              ...pane.executedSkills,
              [result.session.id]: {
                ...(pane.executedSkills[result.session.id] ?? {}),
                [result.assistantMessage.id]: result.executedSkill,
              },
            },
            messageRequestToken: pane.messageRequestToken + 1,
            loading: false,
            asking: false,
            error: null,
          }),
        };
      });
      return result;
    } catch (error) {
      set((state) => ({
        panes: setPane(state.panes, workspaceId, {
          asking: false,
          error: getErrorMessage(error, 'Failed to ask workspace agent'),
        }),
      }));
      throw error;
    }
  },
}));
