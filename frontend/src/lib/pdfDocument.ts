import * as pdfjsLib from 'pdfjs-dist';
import pdfWorkerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url';
import { loadPDFBytes } from './pdfSource';

pdfjsLib.GlobalWorkerOptions.workerSrc = pdfWorkerUrl;

let disableWorkerGlobally = false;

export interface LoadedPDFDocument {
  loadingTask: pdfjsLib.PDFDocumentLoadingTask;
  pdf: pdfjsLib.PDFDocumentProxy;
}

export async function openPDFDocument(pdfPath: string): Promise<LoadedPDFDocument> {
  const pdfBytes = await loadPDFBytes(pdfPath);
  const attemptModes = disableWorkerGlobally ? [true] : [false, true];
  let lastError: unknown = null;

  for (const disableWorker of attemptModes) {
    const documentInit = {
      data: pdfBytes.slice(),
      disableWorker,
    };
    const loadingTask = pdfjsLib.getDocument(documentInit);
    try {
      const pdf = await loadingTask.promise;
      if (disableWorker) {
        disableWorkerGlobally = true;
      }
      return { loadingTask, pdf };
    } catch (error) {
      lastError = error;
      await destroyPDFDocument(loadingTask);
      if (!disableWorker) {
        disableWorkerGlobally = true;
      }
    }
  }

  throw normalizePDFError(lastError);
}

export async function destroyPDFDocument(loadingTask: pdfjsLib.PDFDocumentLoadingTask | null | undefined) {
  if (!loadingTask) {
    return;
  }

  try {
    await loadingTask.destroy();
  } catch {
    // Ignore destroy races during rapid page or zoom changes.
  }
}

function normalizePDFError(error: unknown): Error {
  if (error instanceof Error) {
    return error;
  }
  return new Error('Failed to open PDF document');
}
