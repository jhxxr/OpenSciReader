import { create } from 'zustand';
import { workspaceApi } from '../api/workspace';
import type { DocumentRecord, Workspace } from '../types/workspace';
import type { ZoteroItem } from '../types/zotero';

export interface WorkspaceState {
  workspaces: Workspace[];
  activeWorkspaceId: string | null;
  documents: DocumentRecord[];
  isLoading: boolean;
  isImporting: boolean;
  error: string | null;
  loadWorkspaces: () => Promise<void>;
  selectWorkspace: (workspaceId: string) => Promise<void>;
  createWorkspace: (input: { name: string; description: string; color: string }) => Promise<Workspace>;
  importFiles: (filePaths: string[]) => Promise<void>;
  importZoteroItem: (item: ZoteroItem) => Promise<void>;
  clearError: () => void;
}

export const useWorkspaceStore = create<WorkspaceState>((set, get) => ({
  workspaces: [],
  activeWorkspaceId: null,
  documents: [],
  isLoading: false,
  isImporting: false,
  error: null,

  async loadWorkspaces() {
    set({ isLoading: true, error: null });
    try {
      let workspaces = await workspaceApi.listWorkspaces();
      if (workspaces.length === 0) {
        const defaultWorkspace = await workspaceApi.createWorkspace({
          name: 'Default Workspace',
          description: 'OpenSciReader 默认工作区',
          color: '#6366f1',
        });
        workspaces = [defaultWorkspace];
      }
      const activeWorkspaceId = get().activeWorkspaceId ?? workspaces[0]?.id ?? null;
      set({ workspaces, activeWorkspaceId, isLoading: false, error: null });
      if (activeWorkspaceId) {
        const documents = await workspaceApi.listDocuments(activeWorkspaceId);
        set({ documents });
      } else {
        set({ documents: [] });
      }
    } catch (error) {
      set({
        isLoading: false,
        error: error instanceof Error ? error.message : '加载工作区失败',
      });
    }
  },

  async selectWorkspace(workspaceId) {
    set({ activeWorkspaceId: workspaceId, isLoading: true, error: null });
    try {
      const documents = await workspaceApi.listDocuments(workspaceId);
      set({ documents, isLoading: false });
    } catch (error) {
      set({
        isLoading: false,
        error: error instanceof Error ? error.message : '加载工作区文档失败',
      });
    }
  },

  async createWorkspace(input) {
    const workspace = await workspaceApi.createWorkspace(input);
    const workspaces = [workspace, ...get().workspaces];
    set({ workspaces, activeWorkspaceId: workspace.id, documents: [], error: null });
    return workspace;
  },

  async importFiles(filePaths) {
    const workspaceId = get().activeWorkspaceId;
    if (!workspaceId) {
      throw new Error('请先创建或选择一个工作区');
    }
    if (filePaths.length === 0) {
      return;
    }

    set({ isImporting: true, error: null });
    try {
      await workspaceApi.importFiles({ workspaceId, filePaths });
      const documents = await workspaceApi.listDocuments(workspaceId);
      set({ documents, isImporting: false, error: null });
    } catch (error) {
      set({
        isImporting: false,
        error: error instanceof Error ? error.message : '导入文件失败',
      });
      throw error;
    }
  },

  async importZoteroItem(item) {
    const workspaceId = get().activeWorkspaceId;
    if (!workspaceId) {
      throw new Error('请先创建或选择一个工作区');
    }
    if (!item.pdfPath) {
      throw new Error('当前 Zotero 条目没有可导入的 PDF');
    }

    set({ isImporting: true, error: null });
    try {
      await workspaceApi.importZoteroItem(workspaceId, item.id, item.pdfPath, item.title, item.citeKey);
      const documents = await workspaceApi.listDocuments(workspaceId);
      set({ documents, isImporting: false, error: null });
    } catch (error) {
      set({
        isImporting: false,
        error: error instanceof Error ? error.message : '导入 Zotero 文献失败',
      });
      throw error;
    }
  },

  clearError() {
    set({ error: null });
  },
}));
