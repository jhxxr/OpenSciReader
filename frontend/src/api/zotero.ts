import type { CollectionTree, ZoteroItem } from '../types/zotero';

interface WailsZoteroApp {
  GetCollections: (source: string) => Promise<CollectionTree[]>;
  GetItemsByCollection: (collectionId: string) => Promise<ZoteroItem[]>;
  ResolvePDFPath: (itemId: string) => Promise<string>;
}

function isWailsApp(value: unknown): value is { go: { main: { App: WailsZoteroApp } } } {
  return typeof value === 'object' && value !== null && 'go' in value;
}

const mockCollections: CollectionTree[] = [
  {
    id: 'library:1',
    name: 'My Library',
    libraryId: 1,
    library: 'My Library',
    parentId: '',
    path: '/My Library',
    children: [
      {
        id: 'col-ai',
        name: 'AI Reading',
        libraryId: 1,
        library: 'My Library',
        parentId: 'library:1',
        path: '/My Library/AI Reading',
        children: [],
      },
    ],
  },
];

const mockItems: Record<string, ZoteroItem[]> = {
  'col-ai': [
    {
      id: 'smith2024rag',
      key: 'ABCDEFG1',
      citeKey: 'smith2024rag',
      title: 'Retrieval-Augmented Generation for Scientific Reading',
      creators: 'Smith, Jane',
      year: '2024',
      itemType: 'journalArticle',
      libraryId: 1,
      collectionIds: ['col-ai'],
      attachmentCount: 1,
      hasPdf: true,
      pdfPath: 'C:/Zotero/storage/ABCDEFG1/paper.pdf',
      rawId: 'http://zotero.org/users/local/items/ABCDEFG1',
    },
  ],
};

function createMockApp(): WailsZoteroApp {
  return {
    async GetCollections() {
      return mockCollections;
    },
    async GetItemsByCollection(collectionId) {
      return mockItems[collectionId] ?? [];
    },
    async ResolvePDFPath(itemId) {
      return Object.values(mockItems).flat().find((item) => item.id === itemId)?.pdfPath ?? '';
    },
  };
}

function getApp(): WailsZoteroApp {
  if (typeof window !== 'undefined' && isWailsApp(window) && window.go?.main?.App) {
    return window.go.main.App;
  }
  return createMockApp();
}

const app = getApp();

export const zoteroApi = {
  getCollections(source = 'http') {
    return app.GetCollections(source);
  },
  getItemsByCollection(collectionId: string) {
    return app.GetItemsByCollection(collectionId);
  },
  resolvePDFPath(itemId: string) {
    return app.ResolvePDFPath(itemId);
  },
};

export function getMockCollections(): CollectionTree[] {
  return mockCollections;
}

export function getMockItems(collectionId: string): ZoteroItem[] {
  return mockItems[collectionId] ?? [];
}
