export interface WorkspaceKnowledgeSourceRef {
  sourceId: string;
  pageStart: number;
  pageEnd: number;
  excerpt: string;
}

export interface WorkspaceKnowledgeEntity {
  id: string;
  workspaceId: string;
  title: string;
  type: string;
  summary: string;
  aliases: string[];
  sourceRefs: WorkspaceKnowledgeSourceRef[];
  origin: string;
  status: string;
  confidence: number;
  createdAt: string;
  updatedAt: string;
}

export interface WorkspaceKnowledgeClaim {
  id: string;
  workspaceId: string;
  title: string;
  type: string;
  summary: string;
  entityIds: string[];
  sourceRefs: WorkspaceKnowledgeSourceRef[];
  origin: string;
  status: string;
  confidence: number;
  createdAt: string;
  updatedAt: string;
}

export interface WorkspaceKnowledgeTask {
  id: string;
  workspaceId: string;
  title: string;
  type: string;
  summary: string;
  priority: string;
  sourceRefs: WorkspaceKnowledgeSourceRef[];
  origin: string;
  status: string;
  confidence: number;
  createdAt: string;
  updatedAt: string;
}
