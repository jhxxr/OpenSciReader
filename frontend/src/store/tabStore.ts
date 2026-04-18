import { create } from 'zustand';

export interface TabItem {
  id: string;
  title: string;
  pdfPath: string | null;
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

export const useTabStore = create<TabState>((set) => ({
  tabs: [],
  activeTabId: 'home',

  openTab: (item) =>
    set((state) => {
      const key = `${item.workspaceId ?? ''}:${item.documentId ?? item.id}`;
      const exists = state.tabs.find(
        (t) => `${t.workspaceId ?? ''}:${t.documentId ?? t.id}` === key,
      );
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
          // Select previous tab or next tab
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
