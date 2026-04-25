export interface WorkspaceKnowledgeSourceRef {
  sourceId: string;
  pageStart: number;
  pageEnd: number;
  excerpt: string;
}

export interface WorkspaceKnowledgeSource {
  sourceId: string;
  workspaceId: string;
  title: string;
  slug: string;
  kind: string;
  sourcePath: string;
  markItDownPath: string;
  markItDownStatus: string;
  extractStatus: string;
  lastIngestAt: string;
  lastSuccessAt: string;
  lastError: string;
  absolutePath?: string;
  contentHash: string;
  extractPath?: string;
  documentId: string;
}

export interface WorkspaceKnowledgeCompileSummary {
  workspaceId: string;
  startedAt: string;
  finishedAt: string;
  includedSourceIds: string[];
  failedSourceIds: string[];
  updatedWikiPaths: string[];
  compileDirty: boolean;
  wikiDirty: boolean;
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
