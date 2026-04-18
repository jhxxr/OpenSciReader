import type { ChatHistoryEntry } from '../types/history';

interface WailsHistoryApp {
  SaveChatHistory: (entry: Omit<ChatHistoryEntry, 'id' | 'createdAt'>) => Promise<ChatHistoryEntry>;
  ListChatHistory: (workspaceId: string, documentId: string, itemId: string) => Promise<ChatHistoryEntry[]>;
  DeleteChatHistory: (id: number) => Promise<void>;
}

function isWailsApp(value: unknown): value is { go: { main: { App: WailsHistoryApp } } } {
  return typeof value === 'object' && value !== null && 'go' in value;
}

function getApp(): WailsHistoryApp | null {
  if (typeof window !== 'undefined' && isWailsApp(window) && window.go?.main?.App) {
    return window.go.main.App;
  }
  return null;
}

export const historyApi = {
  async saveChatHistory(entry: Omit<ChatHistoryEntry, 'id' | 'createdAt'>): Promise<ChatHistoryEntry> {
    const app = getApp();
    if (!app) {
      return { ...entry, id: Date.now(), createdAt: new Date().toISOString() };
    }
    return app.SaveChatHistory(entry);
  },
  async listChatHistory(workspaceId: string, documentId: string, itemId: string): Promise<ChatHistoryEntry[]> {
    const app = getApp();
    if (!app) {
      return [];
    }
    return app.ListChatHistory(workspaceId, documentId, itemId);
  },
  async deleteChatHistory(id: number): Promise<void> {
    const app = getApp();
    if (!app) {
      return;
    }
    return app.DeleteChatHistory(id);
  },
};
