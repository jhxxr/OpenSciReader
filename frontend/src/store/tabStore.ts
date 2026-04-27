import { create } from 'zustand';

export interface TabItem {
  id: string;
  title: string;
  pdfPath: string | null;
  type?: 'document' | 'workspace';
  workspaceId?: string;
  documentId?: string;
  agentSessionId?: string;
  sourceKind?: 'workspace_document' | 'zotero_item';
  itemType?: string;
  citeKey?: string;
}

interface TabState {
  tabs: TabItem[];
  activeTabId: string; // 'home' or a TabItem ID

  openTab: (item: TabItem) => void;
  closeTab: (id: string) => void;
  setActiveTab: (id: string) => void;
  setTabAgentSessionId: (id: string, sessionId: string | null) => void;
}

function getTabKey(item: TabItem): string {
  if (item.type === 'workspace') {
    return `workspace:${item.workspaceId ?? item.id}`;
  }
  return `document:${item.workspaceId ?? ''}:${item.documentId ?? item.id}`;
}

export const useTabStore = create<TabState>((set) => ({
  tabs: [],
  activeTabId: 'home',

  openTab: (item) =>
    set((state) => {
      const key = getTabKey(item);
      const existingIndex = state.tabs.findIndex((tab) => getTabKey(tab) === key);
      if (existingIndex >= 0) {
        const existing = state.tabs[existingIndex];
        const tabs = [...state.tabs];
        tabs[existingIndex] = {
          ...existing,
          ...item,
          id: existing.id,
        };
        return {
          tabs,
          activeTabId: existing.id,
        };
      }
      return {
        tabs: [...state.tabs, item],
        activeTabId: item.id,
      };
    }),

  closeTab: (id) =>
    set((state) => {
      const newTabs = state.tabs.filter((t) => t.id !== id);
      let newActiveId = state.activeTabId;
      if (state.activeTabId === id) {
        const index = state.tabs.findIndex((t) => t.id === id);
        if (newTabs.length > 0) {
          const nextIndex = index === 0 ? 0 : index - 1;
          newActiveId = newTabs[nextIndex].id;
        } else {
          newActiveId = 'home';
        }
      }
      return {
        tabs: newTabs,
        activeTabId: newActiveId,
      };
    }),

  setActiveTab: (id) => set({ activeTabId: id }),

  setTabAgentSessionId: (id, sessionId) =>
    set((state) => ({
      tabs: state.tabs.map((tab) =>
        tab.id === id
          ? {
              ...tab,
              agentSessionId: sessionId ?? undefined,
            }
          : tab,
      ),
    })),
}));
