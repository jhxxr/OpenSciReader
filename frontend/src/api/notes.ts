import type { ReaderNoteEntry } from '../types/notes';

interface WailsNotesApp {
  SaveReaderNote: (entry: Omit<ReaderNoteEntry, 'id' | 'createdAt'>) => Promise<ReaderNoteEntry>;
  ListReaderNotes: (workspaceId: string, documentId: string, itemId: string) => Promise<ReaderNoteEntry[]>;
}

function isWailsApp(value: unknown): value is { go: { main: { App: WailsNotesApp } } } {
  return typeof value === 'object' && value !== null && 'go' in value;
}

function getApp(): WailsNotesApp | null {
  if (typeof window !== 'undefined' && isWailsApp(window) && window.go?.main?.App) {
    return window.go.main.App;
  }
  return null;
}

export const notesApi = {
  async saveReaderNote(entry: Omit<ReaderNoteEntry, 'id' | 'createdAt'>): Promise<ReaderNoteEntry> {
    const app = getApp();
    if (!app) {
      return { ...entry, id: Date.now(), createdAt: new Date().toISOString() };
    }
    return app.SaveReaderNote(entry);
  },
  async listReaderNotes(workspaceId: string, documentId: string, itemId: string): Promise<ReaderNoteEntry[]> {
    const app = getApp();
    if (!app) {
      return [];
    }
    return app.ListReaderNotes(workspaceId, documentId, itemId);
  },
};
