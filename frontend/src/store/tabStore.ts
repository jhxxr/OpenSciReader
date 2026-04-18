import { create } from 'zustand';

export interface TabItem {
  id: string; // The zotero item ID or random
  title: string;
  pdfPath: string | null;
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
      const exists = state.tabs.find((t) => t.id === item.id);
      if (exists) {
        return { activeTabId: item.id };
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
