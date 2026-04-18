export interface ChatHistoryEntry {
  id: number;
  itemId: string;
  itemTitle: string;
  page: number;
  kind: string;
  prompt: string;
  response: string;
  createdAt: string;
}
