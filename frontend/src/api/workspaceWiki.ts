import type {
  WorkspaceWikiJobEvent,
  WorkspaceWikiPage,
  WorkspaceWikiPageContent,
  WorkspaceWikiScanJob,
  WorkspaceWikiScanStartInput,
} from '../types/workspaceWiki';
import {
  isWorkspaceWikiJobTerminal,
  normalizeWorkspaceWikiJob,
} from '../types/workspaceWiki';

interface WailsWorkspaceWikiApp {
  StartWorkspaceWikiScan: (input: WorkspaceWikiScanStartInput) => Promise<WorkspaceWikiScanJob>;
  CancelWorkspaceWikiScan: (jobId: string) => Promise<WorkspaceWikiScanJob>;
  GetWorkspaceWikiScanJob: (jobId: string) => Promise<WorkspaceWikiScanJob>;
  ListWorkspaceWikiScanJobs: () => Promise<WorkspaceWikiScanJob[]>;
  DeleteWorkspaceWikiScanJob: (jobId: string) => Promise<void>;
  ListWorkspaceWikiPages: (workspaceId: string) => Promise<WorkspaceWikiPage[]>;
  GetWorkspaceWikiPage: (pageId: string) => Promise<WorkspaceWikiPageContent>;
  DeleteWorkspaceWikiPages: (workspaceId: string) => Promise<void>;
}

function isWailsApp(value: unknown): value is { go: { main: { App: WailsWorkspaceWikiApp } } } {
  return typeof value === 'object' && value !== null && 'go' in value;
}

function getWailsApp(): WailsWorkspaceWikiApp | null {
  if (typeof window !== 'undefined' && isWailsApp(window) && window.go?.main?.App) {
    return window.go.main.App;
  }
  return null;
}

function normalizeWorkspaceWikiPage(page: WorkspaceWikiPage): WorkspaceWikiPage {
  return {
    ...page,
    sourceDocumentId: page.sourceDocumentId ?? '',
    summary: page.summary ?? '',
    markdownPath: page.markdownPath ?? '',
  };
}

export const workspaceWikiApi = {
  async start(input: WorkspaceWikiScanStartInput): Promise<WorkspaceWikiScanJob> {
    const app = getWailsApp();
    if (!app) {
      throw new Error('workspace wiki API is unavailable');
    }
    return normalizeWorkspaceWikiJob(await app.StartWorkspaceWikiScan(input));
  },

  async cancel(jobId: string): Promise<WorkspaceWikiScanJob> {
    const app = getWailsApp();
    if (!app) {
      throw new Error('workspace wiki API is unavailable');
    }
    return normalizeWorkspaceWikiJob(await app.CancelWorkspaceWikiScan(jobId));
  },

  async getJob(jobId: string): Promise<WorkspaceWikiScanJob> {
    const app = getWailsApp();
    if (!app) {
      throw new Error('workspace wiki API is unavailable');
    }
    return normalizeWorkspaceWikiJob(await app.GetWorkspaceWikiScanJob(jobId));
  },

  async listJobs(): Promise<WorkspaceWikiScanJob[]> {
    const app = getWailsApp();
    if (!app) {
      return [];
    }
    return (await app.ListWorkspaceWikiScanJobs()).map(normalizeWorkspaceWikiJob);
  },

  async deleteJob(jobId: string): Promise<void> {
    const app = getWailsApp();
    if (!app) {
      throw new Error('workspace wiki API is unavailable');
    }
    await app.DeleteWorkspaceWikiScanJob(jobId);
  },

  async listPages(workspaceId: string): Promise<WorkspaceWikiPage[]> {
    const app = getWailsApp();
    if (!app || workspaceId.trim() === '') {
      return [];
    }
    return (await app.ListWorkspaceWikiPages(workspaceId)).map(normalizeWorkspaceWikiPage);
  },

  async getPage(pageId: string): Promise<WorkspaceWikiPageContent> {
    const app = getWailsApp();
    if (!app || pageId.trim() === '') {
      throw new Error('workspace wiki page is unavailable');
    }
    const content = await app.GetWorkspaceWikiPage(pageId);
    return {
      ...content,
      page: normalizeWorkspaceWikiPage(content.page),
      markdown: content.markdown ?? '',
    };
  },

  async deletePages(workspaceId: string): Promise<void> {
    const app = getWailsApp();
    if (!app || workspaceId.trim() === '') {
      throw new Error('workspace wiki API is unavailable');
    }
    await app.DeleteWorkspaceWikiPages(workspaceId);
  },

  subscribe(
    jobId: string,
    onEvent: (event: WorkspaceWikiJobEvent) => void,
    onError?: (error: Error) => void,
  ): () => void {
    if (typeof window === 'undefined' || jobId.trim() === '') {
      return () => {};
    }

    let stopped = false;
    let timerId: number | null = null;

    const stop = () => {
      stopped = true;
      if (timerId !== null) {
        window.clearInterval(timerId);
        timerId = null;
      }
    };

    const poll = async () => {
      try {
        const status = await workspaceWikiApi.getJob(jobId);
        if (stopped) {
          return;
        }
        onEvent({
          jobId: status.jobId,
          type: status.status,
          status,
          message: status.message,
          error: status.error,
        });
        if (isWorkspaceWikiJobTerminal(status.status)) {
          stop();
        }
      } catch (error) {
        if (stopped) {
          return;
        }
        onError?.(error instanceof Error ? error : new Error('轮询 workspace wiki 任务失败'));
        stop();
      }
    };

    void poll();
    timerId = window.setInterval(() => {
      void poll();
    }, 1200);
    return stop;
  },
};
