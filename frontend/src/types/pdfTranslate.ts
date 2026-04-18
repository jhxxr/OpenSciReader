export type PDFTranslateMode = "preview" | "export";
export type PDFTranslateJobStatus =
  | "queued"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

export interface PDFTranslateJobOutputs {
  originalPdfPath?: string;
  monoPdfPath?: string;
  dualPdfPath?: string;
  noWatermarkMonoPdfPath?: string;
  noWatermarkDualPdfPath?: string;
  autoExtractedGlossaryPath?: string;
  totalSeconds?: number;
  peakMemoryUsage?: number;
}

export interface PDFTranslateChunkStatus {
  index: number;
  startPage: number;
  endPage: number;
  status: PDFTranslateJobStatus;
  translatedPdfPath?: string;
  dualPdfPath?: string;
  translatedPageOffset?: number;
  startedAt?: string;
  finishedAt?: string;
  totalSeconds?: number;
  error?: string;
}

export interface PDFTranslateJobSnapshot {
  jobId: string;
  retryOfJobId?: string;
  mode: PDFTranslateMode;
  status: PDFTranslateJobStatus;
  itemId?: string;
  itemTitle?: string;
  pdfPath: string;
  localPdfPath: string;
  pageCount: number;
  sourceLang: string;
  targetLang: string;
  previewChunkPages: number;
  maxPagesPerPart: number;
  qps: number;
  poolMaxWorkers: number;
  termPoolMaxWorkers: number;
  providerId: number;
  providerName: string;
  modelId: string;
  createdAt: string;
  updatedAt: string;
  startedAt?: string;
  finishedAt?: string;
  currentStage?: string;
  overallProgress?: number;
  stageProgress?: number;
  stageCurrent?: number;
  stageTotal?: number;
  partIndex?: number;
  totalParts?: number;
  error?: string;
  outputs: PDFTranslateJobOutputs;
  chunks: PDFTranslateChunkStatus[];
}

export interface PDFTranslateEvent {
  sequence: number;
  jobId: string;
  mode: PDFTranslateMode;
  type: string;
  timestamp: string;
  jobStatus: PDFTranslateJobStatus;
  message?: string;
  error?: string;
  errorType?: string;
  details?: string;
  stage?: string;
  stageProgress?: number;
  overallProgress?: number;
  stageCurrent?: number;
  stageTotal?: number;
  partIndex?: number;
  totalParts?: number;
  chunk?: PDFTranslateChunkStatus;
  output?: PDFTranslateJobOutputs;
  status?: PDFTranslateJobSnapshot;
}

export interface PDFTranslateStartPayload {
  pdfPath: string;
  pageCount: number;
  itemId: string;
  itemTitle: string;
  sourceLang: string;
  targetLang: string;
  mode: PDFTranslateMode;
  previewChunkPages: number;
  maxPagesPerPart: number;
  qps: number;
  poolMaxWorkers: number;
  termPoolMaxWorkers: number;
  retryJobId?: string;
  reusePreviewJobId?: string;
  llmProviderId: number;
  llmModelId: number;
}

export interface PDFTranslatedPageRender {
  pdfPath: string;
  translatedPageNumber: number;
  chunkIndex: number;
}

function normalizeProgressValue(value: number | undefined): number | undefined {
  if (value === undefined || value === null || Number.isNaN(value)) {
    return undefined;
  }
  if (value > 1) {
    return Math.max(0, Math.min(1, value / 100));
  }
  return Math.max(0, Math.min(1, value));
}

export function normalizePDFTranslateSnapshot(
  snapshot: PDFTranslateJobSnapshot,
): PDFTranslateJobSnapshot {
  return {
    ...snapshot,
    overallProgress: normalizeProgressValue(snapshot.overallProgress),
    stageProgress: normalizeProgressValue(snapshot.stageProgress),
  };
}

export function applyPDFTranslateEvent(
  current: PDFTranslateJobSnapshot | null,
  event: PDFTranslateEvent,
): PDFTranslateJobSnapshot | null {
  if (event.status) {
    return normalizePDFTranslateSnapshot(event.status);
  }
  if (!current || current.jobId !== event.jobId) {
    return current;
  }

  const next: PDFTranslateJobSnapshot = {
    ...current,
    status: event.jobStatus ?? current.status,
    updatedAt: event.timestamp || current.updatedAt,
    currentStage: event.stage ?? current.currentStage,
    overallProgress:
      normalizeProgressValue(event.overallProgress) ?? current.overallProgress,
    stageProgress:
      normalizeProgressValue(event.stageProgress) ?? current.stageProgress,
    stageCurrent: event.stageCurrent ?? current.stageCurrent,
    stageTotal: event.stageTotal ?? current.stageTotal,
    partIndex: event.partIndex ?? current.partIndex,
    totalParts: event.totalParts ?? current.totalParts,
    error: event.error ?? current.error,
    outputs: event.output ? { ...current.outputs, ...event.output } : current.outputs,
    chunks: current.chunks.map((chunk) =>
      !event.chunk || chunk.index !== event.chunk.index
        ? chunk
        : { ...chunk, ...event.chunk },
    ),
  };

  if (event.type === "finish") {
    next.finishedAt = event.timestamp || next.finishedAt;
  }
  if (event.type === "cancelled" || event.type === "error") {
    next.finishedAt = event.timestamp || next.finishedAt;
  }
  return next;
}

export function buildPDFTranslatedPageMap(
  snapshot: PDFTranslateJobSnapshot | null,
): Record<number, PDFTranslatedPageRender> {
  if (!snapshot || snapshot.mode !== "preview") {
    return {};
  }

  const pageMap: Record<number, PDFTranslatedPageRender> = {};
  for (const chunk of snapshot.chunks) {
    if (!chunk.translatedPdfPath || chunk.status !== "completed") {
      continue;
    }
    for (let sourcePage = chunk.startPage; sourcePage <= chunk.endPage; sourcePage += 1) {
      pageMap[sourcePage] = {
        pdfPath: chunk.translatedPdfPath,
        translatedPageNumber: sourcePage - chunk.startPage + 1,
        chunkIndex: chunk.index,
      };
    }
  }
  return pageMap;
}
