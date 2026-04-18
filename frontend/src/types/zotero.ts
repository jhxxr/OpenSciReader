export interface CollectionTree {
  id: string;
  name: string;
  libraryId: number;
  library: string;
  parentId: string;
  path: string;
  children: CollectionTree[];
}

export interface ZoteroItem {
  id: string;
  key: string;
  citeKey: string;
  title: string;
  creators: string;
  year: string;
  itemType: string;
  libraryId: number;
  collectionIds: string[];
  attachmentCount: number;
  hasPdf: boolean;
  pdfPath: string;
  rawId: string;
}
