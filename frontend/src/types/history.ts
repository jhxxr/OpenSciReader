export interface ChatHistoryEntry {
  id: number;
  workspaceId: string;
  documentId: string;
  itemId: string;
  itemTitle: string;
  page: number;
  kind: string;
  prompt: string;
  response: string;
  createdAt: string;
}
