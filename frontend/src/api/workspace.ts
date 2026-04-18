import type {
  DocumentRecord,
  ImportFilesInput,
  ImportFilesResult,
  Workspace,
  WorkspaceUpsertInput,
} from '../types/workspace';

interface WailsWorkspaceApp {
  ListWorkspaces: () => Promise<Workspace[]>;
  CreateWorkspace: (input: WorkspaceUpsertInput) => Promise<Workspace>;
  ImportFiles: (input: ImportFilesInput) => Promise<ImportFilesResult>;
  ImportZoteroItem: (workspaceId: string, itemId: string, pdfPath: string, title: string, citeKey: string) => Promise<ImportFilesResult>;
  ListDocuments: (workspaceId: string) => Promise<DocumentRecord[]>;
  DeleteDocument: (workspaceId: string, documentId: string) => Promise<void>;
  SelectImportFiles?: () => Promise<string[]>;
}

function isWailsApp(value: unknown): value is { go: { main: { App: WailsWorkspaceApp } } } {
  return typeof value === 'object' && value !== null && 'go' in value;
}

function getApp(): WailsWorkspaceApp | null {
  if (typeof window !== 'undefined' && isWailsApp(window) && window.go?.main?.App) {
    return window.go.main.App;
  }
  return null;
}

export const workspaceApi = {
  async listWorkspaces(): Promise<Workspace[]> {
    const app = getApp();
    if (!app) {
      return [];
    }
    return app.ListWorkspaces();
  },

  async createWorkspace(input: WorkspaceUpsertInput): Promise<Workspace> {
    const app = getApp();
    if (!app) {
      throw new Error('Workspace desktop bridge is unavailable');
    }
    return app.CreateWorkspace(input);
  },

  async importFiles(input: ImportFilesInput): Promise<ImportFilesResult> {
    const app = getApp();
    if (!app) {
      throw new Error('Workspace desktop bridge is unavailable');
    }
    return app.ImportFiles(input);
  },

  async importZoteroItem(workspaceId: string, itemId: string, pdfPath: string, title: string, citeKey: string): Promise<ImportFilesResult> {
    const app = getApp();
    if (!app) {
      throw new Error('Workspace desktop bridge is unavailable');
    }
    return app.ImportZoteroItem(workspaceId, itemId, pdfPath, title, citeKey);
  },

  async listDocuments(workspaceId: string): Promise<DocumentRecord[]> {
    const app = getApp();
    if (!app) {
      return [];
    }
    return app.ListDocuments(workspaceId);
  },

  async deleteDocument(workspaceId: string, documentId: string): Promise<void> {
    const app = getApp();
    if (!app) {
      throw new Error('Workspace desktop bridge is unavailable');
    }
    return app.DeleteDocument(workspaceId, documentId);
  },

  async selectLocalFiles(): Promise<string[]> {
    const app = getApp();
    if (!app?.SelectImportFiles) {
      throw new Error('Workspace desktop bridge is unavailable');
    }
    return app.SelectImportFiles();
  },
};
