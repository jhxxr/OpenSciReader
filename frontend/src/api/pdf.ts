import type { PDFDocumentPayload } from '../types/pdf';

interface WailsPDFApp {
  LoadPDFDocument: (pdfPath: string) => Promise<PDFDocumentPayload>;
}

function isWailsApp(value: unknown): value is { go: { main: { App: WailsPDFApp } } } {
  return typeof value === 'object' && value !== null && 'go' in value;
}

function getApp(): WailsPDFApp | null {
  if (typeof window !== 'undefined' && isWailsApp(window) && window.go?.main?.App) {
    return window.go.main.App;
  }
  return null;
}

export const pdfApi = {
  async loadPDFDocument(pdfPath: string): Promise<PDFDocumentPayload> {
    const app = getApp();
    if (!app) {
      throw new Error('PDF desktop bridge is unavailable');
    }
    return app.LoadPDFDocument(pdfPath);
  },
};
