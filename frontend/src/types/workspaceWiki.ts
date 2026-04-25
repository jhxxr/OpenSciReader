export type WorkspaceWikiScanJobStatus =
  | 'queued'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled';

export type WorkspaceWikiPageKind = 'index' | 'overview' | 'open_questions' | 'log' | 'document' | 'concept';

export function workspaceWikiPageKindLabel(kind: WorkspaceWikiPageKind): string {
  switch (kind) {
    case 'index':
      return 'Index';
    case 'overview':
      return 'Overview';
    case 'open_questions':
      return 'Open Questions';
    case 'log':
      return 'Log';
    case 'document':
      return 'Document';
    case 'concept':
      return 'Concept';
    default:
      return kind;
  }
}

export interface WorkspaceWikiScanStartInput {
  workspaceId: string;
  documentId?: string;
  providerId: number;
  modelId: number;
}

export interface WorkspaceWikiScanJob {
  jobId: string;
  workspaceId: string;
  documentId: string;
  status: WorkspaceWikiScanJobStatus;
  totalItems: number;
  processedItems: number;
  failedItems: number;
  currentItem: string;
  currentStage: string;
  message: string;
  overallProgress: number;
  providerId: number;
  modelId: number;
  error?: string;
  startedAt: string;
  updatedAt: string;
  finishedAt?: string;
}

export interface WorkspaceWikiPage {
  id: string;
  workspaceId: string;
  sourceDocumentId: string;
  title: string;
  slug: string;
  kind: WorkspaceWikiPageKind;
  markdownPath: string;
  summary: string;
  createdAt: string;
  updatedAt: string;
}

export interface WorkspaceWikiPageContent {
  page: WorkspaceWikiPage;
  markdown: string;
}

export interface WorkspaceWikiJobEvent {
  jobId: string;
  type: string;
  status: WorkspaceWikiScanJob;
  message?: string;
  error?: string;
}

function clampProgress(value: number | undefined): number {
  if (value === undefined || value === null || Number.isNaN(value)) {
    return 0;
  }
  if (value > 1) {
    return Math.max(0, Math.min(1, value / 100));
  }
  return Math.max(0, Math.min(1, value));
}

export function normalizeWorkspaceWikiJob(job: WorkspaceWikiScanJob): WorkspaceWikiScanJob {
  return {
    ...job,
    documentId: job.documentId ?? '',
    currentItem: job.currentItem ?? '',
    currentStage: job.currentStage ?? '',
    message: job.message ?? '',
    overallProgress: clampProgress(job.overallProgress),
    error: job.error ?? '',
    finishedAt: job.finishedAt ?? '',
  };
}

export function isWorkspaceWikiJobTerminal(status: WorkspaceWikiScanJobStatus): boolean {
  return status === 'completed' || status === 'failed' || status === 'cancelled';
}
