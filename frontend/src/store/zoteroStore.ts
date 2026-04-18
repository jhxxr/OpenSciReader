import { create } from 'zustand';
import { getMockCollections, getMockItems, zoteroApi } from '../api/zotero';
import type { CollectionTree, ZoteroItem } from '../types/zotero';

export interface ZoteroState {
  source: 'http';
  collections: CollectionTree[];
  selectedCollectionId: string | null;
  items: ZoteroItem[];
  selectedItem: ZoteroItem | null;
  isLoading: boolean;
  error: string | null;
  demoMode: boolean;
  loadCollections: () => Promise<void>;
  selectCollection: (collectionId: string) => Promise<void>;
  selectItem: (item: ZoteroItem | null) => void;
  clearError: () => void;
}

export const useZoteroStore = create<ZoteroState>((set) => ({
  source: 'http',
  collections: [],
  selectedCollectionId: null,
  items: [],
  selectedItem: null,
  isLoading: false,
  error: null,
  demoMode: false,
  async loadCollections() {
    set({ isLoading: true, error: null });
    try {
      const collections = await zoteroApi.getCollections('http');
      set({ collections, isLoading: false, demoMode: false });
    } catch (error) {
      const collections = getMockCollections();
      set({
        collections,
        isLoading: false,
        demoMode: true,
        error: error instanceof Error
          ? `Zotero 本地 API 连接失败，已切换到演示库。请确认 Zotero 正在运行。原始错误: ${error.message}`
          : 'Zotero 本地 API 连接失败，已切换到演示库。',
      });
    }
  },
  async selectCollection(collectionId) {
    set({ isLoading: true, error: null, selectedCollectionId: collectionId, selectedItem: null });
    try {
      const items = await zoteroApi.getItemsByCollection(collectionId);
      set({ items, selectedItem: items[0] ?? null, isLoading: false, demoMode: false });
    } catch (error) {
      const items = getMockItems(collectionId);
      set({
        items,
        selectedItem: items[0] ?? null,
        isLoading: false,
        demoMode: true,
        error: error instanceof Error
          ? `文献加载失败，已切换到演示数据。原始错误: ${error.message}`
          : '文献加载失败，已切换到演示数据。',
      });
    }
  },
  selectItem(item) {
    set({ selectedItem: item });
  },
  clearError() {
    set({ error: null });
  },
}));
