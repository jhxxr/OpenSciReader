import { create } from 'zustand';

export interface TabItem {
  id: string;
  title: string;
  pdfPath: string | null;
  type?: 'document' | 'workspace';
  workspaceId?: string;
  documentId?: string;
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
      const exists = state.tabs.find((tab) => getTabKey(tab) === key);
      if (exists) {
        return { activeTabId: exists.id };
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
}));
