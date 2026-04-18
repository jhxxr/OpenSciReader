export interface GatewayContextData {
  selection: string;
  snapshot: string;
  page: number;
  itemTitle: string;
  citeKey: string;
}

export interface GatewayStreamEvent {
  requestId: string;
  type: 'chunk' | 'done' | 'error';
  content?: string;
  error?: string;
}
