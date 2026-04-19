import { useCallback, useEffect, useMemo, useState } from 'react';
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
  wikiScanProviderId: number;
  wikiScanModelId: number;
  onStartWikiScan: (providerId: number, modelId: number) => Promise<void>;
  onCancelWikiScan: () => Promise<void>;
  onRefreshWikiPages: () => Promise<void>;
  onSelectWikiPage: (pageId: string) => Promise<void>;
  onDeleteWikiPages: () => Promise<void>;
  onImportFiles: () => Promise<void>;
  onRefreshDocuments: () => Promise<void>;
  onOpenPdf: (document: DocumentRecord) => void;
  onDeleteDocument: (document: Pick<DocumentRecord, 'id' | 'workspaceId' | 'title' | 'originalFileName'>) => Promise<void>;
  onChangeWikiScanModel: (providerId: number, modelId: number) => void;
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
  wikiScanProviderId,
  wikiScanModelId,
  onStartWikiScan,
  onCancelWikiScan,
  onRefreshWikiPages,
  onSelectWikiPage,
  onDeleteWikiPages,
  onImportFiles,
  onRefreshDocuments,
  onOpenPdf,
  onDeleteDocument,
  onChangeWikiScanModel,
}: WorkspaceTabProps) {
  const [localWikiScanProviderId, setLocalWikiScanProviderId] = useState<number>(wikiScanProviderId);
  const [localWikiScanModelId, setLocalWikiScanModelId] = useState<number>(wikiScanModelId);

  useEffect(() => {
    setLocalWikiScanProviderId(wikiScanProviderId);
  }, [wikiScanProviderId]);

  useEffect(() => {
    setLocalWikiScanModelId(wikiScanModelId);
  }, [wikiScanModelId]);

  const llmProvidersWithModels = useMemo(
    () => llmProviderConfigs.filter((item) => item.provider.type === 'llm' && item.models.length > 0),
    [llmProviderConfigs]
  );

  const selectedProviderConfig = useMemo(
    () => llmProvidersWithModels.find((item) => item.provider.id === localWikiScanProviderId) ?? null,
    [llmProvidersWithModels, localWikiScanProviderId]
  );

  const selectedModelRecord = useMemo(
    () => selectedProviderConfig?.models.find((model) => model.id === localWikiScanModelId) ?? null,
    [selectedProviderConfig, localWikiScanModelId]
  );

  const handleProviderChange = useCallback((event: React.ChangeEvent<HTMLSelectElement>) => {
    const newProviderId = Number(event.target.value);
    setLocalWikiScanProviderId(newProviderId);
    const provider = llmProvidersWithModels.find((item) => item.provider.id === newProviderId);
    const firstModelId = provider?.models[0]?.id ?? 0;
    setLocalWikiScanModelId(firstModelId);
    onChangeWikiScanModel(newProviderId, firstModelId);
  }, [llmProvidersWithModels, onChangeWikiScanModel]);

  const handleModelChange = useCallback((event: React.ChangeEvent<HTMLSelectElement>) => {
    const newModelId = Number(event.target.value);
    setLocalWikiScanModelId(newModelId);
    onChangeWikiScanModel(localWikiScanProviderId, newModelId);
  }, [localWikiScanProviderId, onChangeWikiScanModel]);

  const handleStartScan = useCallback(() => {
    if (localWikiScanProviderId > 0 && localWikiScanModelId > 0) {
      void onStartWikiScan(localWikiScanProviderId, localWikiScanModelId);
    }
  }, [localWikiScanProviderId, localWikiScanModelId, onStartWikiScan]);

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
                <div className="wiki-model-selector">
                  <select
                    className="wiki-provider-select"
                    value={localWikiScanProviderId}
                    onChange={handleProviderChange}
                    disabled={llmProvidersWithModels.length === 0 || isStartingWikiScan || activeWikiJob?.status === 'running'}
                  >
                    {llmProvidersWithModels.length === 0 ? (
                      <option value={0}>No LLM provider</option>
                    ) : localWikiScanProviderId === 0 ? (
                      <option value={0}>Select provider</option>
                    ) : null}
                    {llmProvidersWithModels.map((item) => (
                      <option key={item.provider.id} value={item.provider.id}>
                        {item.provider.name}
                      </option>
                    ))}
                  </select>
                  <select
                    className="wiki-model-select"
                    value={localWikiScanModelId}
                    onChange={handleModelChange}
                    disabled={!selectedProviderConfig || isStartingWikiScan || activeWikiJob?.status === 'running'}
                  >
                    {!selectedProviderConfig ? (
                      <option value={0}>Select model</option>
                    ) : null}
                    {selectedProviderConfig?.models.map((model) => (
                      <option key={model.id} value={model.id}>
                        {model.modelId}
                      </option>
                    ))}
                  </select>
                </div>
                <Button variant="secondary" size="sm" onClick={handleStartScan} disabled={isStartingWikiScan || isCancellingWikiScan || !selectedProviderConfig || !selectedModelRecord}>
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

            {!selectedProviderConfig || !selectedModelRecord ? (
              <p className="empty-inline">
                {llmProvidersWithModels.length === 0
                  ? 'Configure at least one LLM provider with a model before starting a workspace wiki scan.'
                  : '请先为当前工作区选择扫描模型'}
              </p>
            ) : null}

            {activeWikiJob ? (
              <>
                <div className="home-task-stage">
                  <div className="home-task-stage-head">
                    <strong>{activeWikiJob.message || 'Scanning workspace content'}</strong>
                    <span>{activeProgress.toFixed(1)}%</span>
                  </div>
                  <p>{activeWikiJob.currentItem || activeWikiJob.currentStage || 'Preparing workspace wiki scan...'}</p>
                  {selectedProviderConfig && selectedModelRecord ? (
                    <small className="wiki-scan-model-info">
                      Using {selectedProviderConfig.provider.name} / {selectedModelRecord.modelId}
                    </small>
                  ) : null}
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
