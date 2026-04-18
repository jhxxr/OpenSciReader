export interface Workspace {
  id: string;
  name: string;
  description: string;
  color: string;
  createdAt: string;
  updatedAt: string;
}

export interface WorkspaceUpsertInput {
  id?: string;
  name: string;
  description: string;
  color: string;
}

export interface DocumentRecord {
  id: string;
  workspaceId: string;
  title: string;
  documentType: string;
  sourceType: string;
  defaultAssetId: string;
  originalFileName: string;
  primaryPdfPath: string;
  contentHash: string;
  createdAt: string;
  updatedAt: string;
}

export interface ImportRecord {
  id: string;
  workspaceId: string;
  documentId: string;
  sourceType: string;
  sourceLabel: string;
  sourceRef: string;
  status: string;
  message: string;
  createdAt: string;
}

export interface ImportFilesInput {
  workspaceId: string;
  filePaths: string[];
  sourceType?: string;
  sourceLabel?: string;
  sourceRef?: string;
  title?: string;
}

export interface ImportFilesResult {
  workspace: Workspace;
  documents: DocumentRecord[];
  imports: ImportRecord[];
}
