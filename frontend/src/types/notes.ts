export interface ReaderNoteEntry {
  id: number;
  workspaceId: string;
  documentId: string;
  itemId: string;
  itemTitle: string;
  page: number;
  anchorText: string;
  content: string;
  createdAt: string;
}
