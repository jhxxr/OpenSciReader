import { EventsOn } from "../../wailsjs/runtime/runtime";
import type {
  PDFTranslateEvent,
  PDFTranslateJobSnapshot,
  PDFTranslateStartPayload,
} from "../types/pdfTranslate";
import { normalizePDFTranslateSnapshot } from "../types/pdfTranslate";

interface WailsPDFTranslateApp {
  StartPDFTranslate: (payload: PDFTranslateStartPayload) => Promise<PDFTranslateJobSnapshot>;
  CancelPDFTranslate: (jobId: string) => Promise<PDFTranslateJobSnapshot>;
  GetPDFTranslateStatus: (jobId: string) => Promise<PDFTranslateJobSnapshot>;
  ListPDFTranslateJobs: () => Promise<PDFTranslateJobSnapshot[]>;
  DeletePDFTranslateJob: (jobId: string) => Promise<void>;
}

function isWailsApp(value: unknown): value is { go: { main: { App: WailsPDFTranslateApp } } } {
  return typeof value === "object" && value !== null && "go" in value;
}

function isWailsDesktop() {
  return typeof window !== "undefined" && isWailsApp(window) && Boolean(window.go?.main?.App);
}

function getWailsApp(): WailsPDFTranslateApp | null {
  if (isWailsDesktop()) {
    return (window as typeof window & { go: { main: { App: WailsPDFTranslateApp } } }).go.main.App;
  }
  return null;
}

async function readJSONResponse<T>(response: Response): Promise<T> {
  const text = await response.text();
  const payload = text ? (JSON.parse(text) as T | { error?: string }) : {};
  if (!response.ok) {
    const message =
      typeof payload === "object" &&
      payload !== null &&
      "error" in payload &&
      typeof payload.error === "string"
        ? payload.error
        : `HTTP ${response.status}`;
    throw new Error(message);
  }
  return payload as T;
}

function jobURL(path: string) {
  return `/api/pdf-translate${path}`;
}

function websocketOrigin() {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const host = window.location.host || "wails.localhost";
  return `${protocol}//${host}`;
}

function pdfTranslateRuntimeEventName(jobId: string) {
  return `pdf-translate:event:${jobId}`;
}

export const pdfTranslateApi = {
  async start(payload: PDFTranslateStartPayload): Promise<PDFTranslateJobSnapshot> {
    const app = getWailsApp();
    if (app) {
      return normalizePDFTranslateSnapshot(await app.StartPDFTranslate(payload));
    }
    const response = await fetch(jobURL("/start"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    return normalizePDFTranslateSnapshot(
      await readJSONResponse<PDFTranslateJobSnapshot>(response),
    );
  },
  async cancel(jobId: string): Promise<PDFTranslateJobSnapshot> {
    const app = getWailsApp();
    if (app) {
      return normalizePDFTranslateSnapshot(await app.CancelPDFTranslate(jobId));
    }
    const response = await fetch(jobURL(`/${encodeURIComponent(jobId)}/cancel`), {
      method: "POST",
    });
    return normalizePDFTranslateSnapshot(
      await readJSONResponse<PDFTranslateJobSnapshot>(response),
    );
  },
  async getStatus(jobId: string): Promise<PDFTranslateJobSnapshot> {
    const app = getWailsApp();
    if (app) {
      return normalizePDFTranslateSnapshot(await app.GetPDFTranslateStatus(jobId));
    }
    const response = await fetch(jobURL(`/${encodeURIComponent(jobId)}/status`));
    return normalizePDFTranslateSnapshot(
      await readJSONResponse<PDFTranslateJobSnapshot>(response),
    );
  },
  async listJobs(): Promise<PDFTranslateJobSnapshot[]> {
    const app = getWailsApp();
    if (app) {
      return (await app.ListPDFTranslateJobs()).map(normalizePDFTranslateSnapshot);
    }
    const response = await fetch(jobURL("/jobs"));
    return (await readJSONResponse<PDFTranslateJobSnapshot[]>(response)).map(
      normalizePDFTranslateSnapshot,
    );
  },
  async deleteJob(jobId: string): Promise<void> {
    const app = getWailsApp();
    if (app) {
      await app.DeletePDFTranslateJob(jobId);
      return;
    }
    const response = await fetch(jobURL(`/${encodeURIComponent(jobId)}/delete`), {
      method: 'POST',
    });
    await readJSONResponse<{ jobId: string; status: string }>(response);
  },
  subscribe(
    jobId: string,
    onEvent: (event: PDFTranslateEvent) => void,
    onError?: (error: Error) => void,
  ): () => void {
    if (isWailsDesktop()) {
      return EventsOn(pdfTranslateRuntimeEventName(jobId), (payload: PDFTranslateEvent) => {
        onEvent(payload);
      });
    }

    const socket = new WebSocket(
      `${websocketOrigin()}/api/pdf-translate/${encodeURIComponent(jobId)}/events`,
    );
    socket.onmessage = (raw) => {
      try {
        onEvent(JSON.parse(raw.data) as PDFTranslateEvent);
      } catch (error) {
        onError?.(error instanceof Error ? error : new Error("解析翻译事件失败"));
      }
    };
    socket.onerror = () => {
      onError?.(new Error("翻译事件连接异常"));
    };
    return () => {
      socket.close();
    };
  },
};
