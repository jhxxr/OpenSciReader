import { FileText, LoaderCircle, RefreshCw, Sparkles, SquareX, Trash2 } from 'lucide-react';
import { MarkdownPreview } from './MarkdownPreview';
import { Button } from './ui/Button';
import type { ProviderConfig } from '../types/config';
import type { DocumentRecord, Workspace } from '../types/workspace';
import type {
  WorkspaceWikiPage,
  WorkspaceWikiPageContent,
  WorkspaceWikiScanJob,
} from '../types/workspaceWiki';

interface WorkspaceTabProps {
  workspace: Workspace | null;
  documents: DocumentRecord[];
  isImporting: boolean;
  isLoadingDocuments: boolean;
  deletingDocumentId: string | null;
  llmProviderConfigs: ProviderConfig[];
  wikiPages: WorkspaceWikiPage[];
  selectedWikiPageId: string | null;
  wikiPageContent: WorkspaceWikiPageContent | null;
  isLoadingWikiPages: boolean;
  isLoadingWikiPageContent: boolean;
  activeWikiJob: WorkspaceWikiScanJob | null;
  wikiError: string | null;
  isStartingWikiScan: boolean;
  isCancellingWikiScan: boolean;
  isDeletingWikiPages: boolean;
  onImportFiles: () => Promise<void>;
  onRefreshDocuments: () => Promise<void>;
  onOpenPdf: (document: DocumentRecord) => void;
  onDeleteDocument: (document: Pick<DocumentRecord, 'id' | 'workspaceId' | 'title' | 'originalFileName'>) => Promise<void>;
  onStartWikiScan: () => Promise<void>;
  onCancelWikiScan: () => Promise<void>;
  onRefreshWikiPages: () => Promise<void>;
  onSelectWikiPage: (pageId: string) => Promise<void>;
  onDeleteWikiPages: () => Promise<void>;
}

export function WorkspaceTab({
  workspace,
  documents,
  isImporting,
  isLoadingDocuments,
  deletingDocumentId,
  llmProviderConfigs,
  wikiPages,
  selectedWikiPageId,
  wikiPageContent,
  isLoadingWikiPages,
  isLoadingWikiPageContent,
  activeWikiJob,
  wikiError,
  isStartingWikiScan,
  isCancellingWikiScan,
  isDeletingWikiPages,
  onImportFiles,
  onRefreshDocuments,
  onOpenPdf,
  onDeleteDocument,
  onStartWikiScan,
  onCancelWikiScan,
  onRefreshWikiPages,
  onSelectWikiPage,
  onDeleteWikiPages,
}: WorkspaceTabProps) {
  const hasRunnableModel = llmProviderConfigs.some((item) => item.models.length > 0);
  const selectedPage =
    wikiPageContent?.page?.id === selectedWikiPageId
      ? wikiPageContent.page
      : wikiPages.find((page) => page.id === selectedWikiPageId) ?? wikiPageContent?.page ?? null;
  const activeProgress = Math.max(0, Math.min(100, (activeWikiJob?.overallProgress ?? 0) * 100));

  if (!workspace) {
    return (
      <section className="workspace-tab workspace-tab-empty">
        <div className="workspace-panel">
          <h2>No Workspace Selected</h2>
          <p className="workspace-panel-description">Choose a workspace from the home page to browse documents and workspace wiki pages.</p>
        </div>
      </section>
    );
  }

  return (
    <section className="workspace-tab">
      <div className="workspace-tab-hero">
        <div>
          <p className="panel-kicker">Workspace</p>
          <h2>{workspace.name}</h2>
          <p>{workspace.description || 'Import documents, run wiki scans, and browse the generated workspace notes here.'}</p>
        </div>
        <div className="workspace-tab-hero-actions">
          <Button variant="secondary" onClick={() => void onImportFiles()} disabled={isImporting}>
            {isImporting ? 'Importing...' : 'Import Files'}
          </Button>
        </div>
      </div>

      <div className="workspace-tab-grid">
        <div className="workspace-panel">
          <div className="section-header workspace-panel-header">
            <div>
              <h3>Documents</h3>
              <p className="workspace-panel-description">Open imported PDFs from here or remove outdated workspace sources.</p>
            </div>
            <div className="workspace-panel-header-actions">
              <Button variant="secondary" size="sm" onClick={() => void onRefreshDocuments()} disabled={isLoadingDocuments}>
                <RefreshCw size={14} className={isLoadingDocuments ? 'spin-inline' : ''} />
                {isLoadingDocuments ? 'Refreshing...' : 'Refresh'}
              </Button>
            </div>
          </div>

          {documents.length > 0 ? (
            <div className="workspace-document-list">
              {documents.map((document) => (
                <article key={document.id} className="workspace-document-card">
                  <div className="workspace-document-copy">
                    <strong>{document.title}</strong>
                    <small>{document.originalFileName || 'Imported document'} / {document.sourceType}</small>
                  </div>
                  <div className="workspace-document-actions">
                    <Button variant="secondary" size="sm" onClick={() => onOpenPdf(document)}>
                      <FileText size={14} />
                      Open PDF
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="danger"
                      onClick={() =>
                        void onDeleteDocument({
                          id: document.id,
                          workspaceId: document.workspaceId,
                          title: document.title,
                          originalFileName: document.originalFileName,
                        })
                      }
                      disabled={deletingDocumentId === document.id}
                      aria-label={`Delete ${document.title}`}
                      title={`Delete ${document.title}`}
                    >
                      {deletingDocumentId === document.id ? <LoaderCircle size={14} className="spin-inline" /> : <Trash2 size={14} />}
                    </Button>
                  </div>
                </article>
              ))}
            </div>
          ) : (
            <p className="empty-inline">No documents in this workspace yet. Import a PDF or Markdown file to begin.</p>
          )}
        </div>

        <div className="workspace-side-panels">
          <div className="workspace-panel workspace-wiki-status-panel">
            <div className="section-header workspace-panel-header">
              <div>
                <h3>Workspace Wiki</h3>
                <p className="workspace-panel-description">Run a workspace scan to build overview and per-document wiki pages.</p>
              </div>
              <div className="workspace-panel-header-actions home-workspace-actions home-workspace-actions-stacked">
                <Button variant="secondary" size="sm" onClick={() => void onStartWikiScan()} disabled={isStartingWikiScan || isCancellingWikiScan || !hasRunnableModel}>
                  <Sparkles size={14} />
                  {isStartingWikiScan ? 'Starting...' : 'Scan'}
                </Button>
                <Button variant="secondary" size="sm" onClick={() => void onCancelWikiScan()} disabled={!activeWikiJob || activeWikiJob.status !== 'running' || isCancellingWikiScan}>
                  <SquareX size={14} />
                  {isCancellingWikiScan ? 'Cancelling...' : 'Cancel'}
                </Button>
                <Button variant="secondary" size="sm" onClick={() => void onRefreshWikiPages()} disabled={isLoadingWikiPages || isStartingWikiScan}>
                  <RefreshCw size={14} className={isLoadingWikiPages ? 'spin-inline' : ''} />
                  Refresh
                </Button>
                <Button variant="ghost" size="sm" className="danger" onClick={() => void onDeleteWikiPages()} disabled={isDeletingWikiPages || wikiPages.length === 0}>
                  <Trash2 size={14} />
                  {isDeletingWikiPages ? 'Deleting...' : 'Delete Pages'}
                </Button>
              </div>
            </div>

            {!hasRunnableModel ? (
              <p className="empty-inline">Configure at least one LLM provider with a model before starting a workspace wiki scan.</p>
            ) : null}

            {activeWikiJob ? (
              <>
                <div className="home-task-stage">
                  <div className="home-task-stage-head">
                    <strong>{activeWikiJob.message || 'Scanning workspace content'}</strong>
                    <span>{activeProgress.toFixed(1)}%</span>
                  </div>
                  <p>{activeWikiJob.currentItem || activeWikiJob.currentStage || 'Preparing workspace wiki scan...'}</p>
                </div>
                <div className="translate-progress">
                  <div className="translate-progress-bar">
                    <div className="translate-progress-fill" style={{ width: `${Math.max(4, activeProgress)}%` }} />
                  </div>
                  <div className="translate-progress-meta">
                    <span>{activeWikiJob.processedItems}/{activeWikiJob.totalItems || 0} processed</span>
                    <span>{activeWikiJob.failedItems} failed</span>
                  </div>
                </div>
              </>
            ) : (
              <p className="empty-inline">No active wiki scan. The latest generated pages will appear below.</p>
            )}

            {wikiError ? <div className="reader-error">{wikiError}</div> : null}
          </div>

          <div className="workspace-panel workspace-wiki-content-panel">
            <div className="section-header workspace-panel-header">
              <div>
                <h3>Wiki Pages</h3>
                <p className="workspace-panel-description">Browse the generated overview and document pages for this workspace.</p>
              </div>
              <span className="badge badge-count">{wikiPages.length}</span>
            </div>

            {wikiPages.length > 0 ? (
              <div className="workspace-wiki-viewer">
                <div className="workspace-wiki-page-list">
                  {wikiPages.map((page) => (
                    <button
                      key={page.id}
                      type="button"
                      className={`workspace-wiki-page-button ${selectedWikiPageId === page.id ? 'workspace-wiki-page-button-active' : ''}`}
                      onClick={() => void onSelectWikiPage(page.id)}
                    >
                      <strong>{page.title}</strong>
                      <small>{page.kind === 'overview' ? 'Overview' : 'Document Page'}</small>
                      <span>{page.summary || 'No summary available yet.'}</span>
                    </button>
                  ))}
                </div>

                <div>
                  <div className="section-header">
                    <div>
                      <h3>{selectedPage?.title || 'Select a wiki page'}</h3>
                      <p className="workspace-panel-description">{selectedPage?.summary || 'Choose a page to preview its markdown content.'}</p>
                    </div>
                  </div>
                  {isLoadingWikiPageContent ? (
                    <div className="empty-inline">Loading page content...</div>
                  ) : (
                    <MarkdownPreview content={wikiPageContent?.markdown ?? ''} placeholder="No wiki page selected yet." />
                  )}
                </div>
              </div>
            ) : (
              <p className="empty-inline">No wiki pages have been generated for this workspace yet.</p>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
