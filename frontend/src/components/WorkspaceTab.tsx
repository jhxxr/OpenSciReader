import type { ReactNode } from 'react';
import { LoaderCircle, RefreshCw, Trash2 } from 'lucide-react';
import { Button } from './ui/Button';
import type { DocumentRecord, Workspace } from '../types/workspace';
import type {
  WorkspaceKnowledgeClaim,
  WorkspaceKnowledgeEntity,
  WorkspaceKnowledgeSourceRef,
  WorkspaceKnowledgeTask,
} from '../types/workspaceKnowledge';

interface WorkspaceTabProps {
  workspace: Workspace | null;
  documents: DocumentRecord[];
  isImporting: boolean;
  deletingDocumentId: string | null;
  knowledgeEntities: WorkspaceKnowledgeEntity[];
  knowledgeClaims: WorkspaceKnowledgeClaim[];
  knowledgeTasks: WorkspaceKnowledgeTask[];
  isKnowledgeLoading: boolean;
  knowledgeError: string | null;
  onImportFiles: () => Promise<void>;
  onRefreshKnowledge: () => Promise<void>;
  onOpenPdf: (item: {
    id: string;
    title: string;
    pdfPath: string | null;
    workspaceId?: string;
    documentId?: string;
    sourceKind?: 'workspace_document' | 'zotero_item';
    itemType?: string;
    citeKey?: string;
  }) => void;
  onDeleteDocument: (document: Pick<DocumentRecord, 'id' | 'workspaceId' | 'title' | 'originalFileName'>) => Promise<void>;
}

export function WorkspaceTab({
  workspace,
  documents,
  isImporting,
  deletingDocumentId,
  knowledgeEntities,
  knowledgeClaims,
  knowledgeTasks,
  isKnowledgeLoading,
  knowledgeError,
  onImportFiles,
  onRefreshKnowledge,
  onOpenPdf,
  onDeleteDocument,
}: WorkspaceTabProps) {
  return (
    <div className="home-workspace-grid">
      <div className="home-workspace-card">
        <div className="section-header">
          <h3>{workspace?.name ?? 'No Workspace Selected'}</h3>
          <span className="badge badge-count">{documents.length}</span>
        </div>
        <p className="home-workspace-description">
          {workspace?.description || 'Import papers into the workspace and organize them around the current research topic.'}
        </p>
        <div className="home-workspace-actions">
          <Button variant="secondary" size="sm" onClick={() => void onImportFiles()} disabled={isImporting || !workspace}>
            {isImporting ? 'Importing...' : 'Import Files'}
          </Button>
        </div>
      </div>

      <div className="home-workspace-card">
        <div className="section-header">
          <h3>Documents</h3>
          <span className="badge badge-count">{documents.length}</span>
        </div>
        {documents.length ? (
          <div className="library-item-list">
            {documents.map((document) => (
              <div key={document.id} className="item-button-card home-workspace-document-card">
                <button
                  type="button"
                  className="item-button home-workspace-document-button"
                  onDoubleClick={() =>
                    onOpenPdf({
                      id: document.id,
                      title: document.title,
                      pdfPath: document.primaryPdfPath ?? null,
                      workspaceId: document.workspaceId,
                      documentId: document.id,
                      sourceKind: 'workspace_document',
                    })
                  }
                >
                  <strong>{document.title}</strong>
                  <small>{document.originalFileName || 'Imported document'} / {document.sourceType}</small>
                </button>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="home-workspace-document-delete danger"
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
            ))}
          </div>
        ) : (
          <p className="empty-inline">No documents in this workspace yet. Import a PDF or Markdown file first.</p>
        )}
      </div>

      <div className="home-workspace-card workspace-memory-card">
        <div className="section-header">
          <div>
            <h3>Workspace Memory</h3>
            <p className="workspace-memory-subtitle">Concepts, claims, and open follow-up work</p>
          </div>
          <Button variant="secondary" size="sm" onClick={() => void onRefreshKnowledge()} disabled={!workspace || isKnowledgeLoading}>
            <RefreshCw size={14} className={isKnowledgeLoading ? 'spin-inline' : ''} />
            {isKnowledgeLoading ? 'Refreshing...' : 'Refresh'}
          </Button>
        </div>

        {knowledgeError ? <div className="reader-error">{knowledgeError}</div> : null}

        <div className="workspace-memory-sections">
          <WorkspaceMemorySection
            title="Entities / Concepts"
            count={knowledgeEntities.length}
            emptyMessage="No extracted entities or concepts yet."
          >
            {knowledgeEntities.map((entity) => (
              <article key={entity.id} className="workspace-memory-item">
                <div className="workspace-memory-item-head">
                  <strong>{entity.title}</strong>
                  <div className="workspace-memory-meta">
                    {entity.type ? <span className="badge">{entity.type}</span> : null}
                    {entity.status ? <span className="badge">{entity.status}</span> : null}
                    {renderConfidenceBadge(entity.confidence)}
                  </div>
                </div>
                <p>{entity.summary || `Aliases recorded: ${entity.aliases.length}`}</p>
                {entity.aliases.length > 0 ? (
                  <div className="workspace-memory-source-list">
                    {entity.aliases.slice(0, 3).map((alias) => (
                      <span key={alias} className="workspace-memory-source-pill">
                        {alias}
                      </span>
                    ))}
                    {entity.aliases.length > 3 ? (
                      <span className="workspace-memory-source-pill">+{entity.aliases.length - 3}</span>
                    ) : null}
                  </div>
                ) : null}
                <small className="workspace-memory-footnote">{formatSourceSummary(entity.sourceRefs)}</small>
              </article>
            ))}
          </WorkspaceMemorySection>

          <WorkspaceMemorySection
            title="Key Claims"
            count={knowledgeClaims.length}
            emptyMessage="No key claims have been compiled yet."
          >
            {knowledgeClaims.map((claim) => (
              <article key={claim.id} className="workspace-memory-item">
                <div className="workspace-memory-item-head">
                  <strong>{claim.title}</strong>
                  <div className="workspace-memory-meta">
                    {claim.type ? <span className="badge">{claim.type}</span> : null}
                    {claim.status ? <span className="badge">{claim.status}</span> : null}
                    {renderConfidenceBadge(claim.confidence)}
                  </div>
                </div>
                <p>{claim.summary || 'No summary available.'}</p>
                {claim.entityIds.length > 0 ? (
                  <small className="workspace-memory-footnote">Linked entities: {claim.entityIds.length}</small>
                ) : null}
                <small className="workspace-memory-footnote">{formatSourceSummary(claim.sourceRefs)}</small>
              </article>
            ))}
          </WorkspaceMemorySection>

          <WorkspaceMemorySection
            title="Open Questions / Tasks"
            count={knowledgeTasks.length}
            emptyMessage="No open questions or tasks have been captured yet."
          >
            {knowledgeTasks.map((task) => (
              <article key={task.id} className="workspace-memory-item">
                <div className="workspace-memory-item-head">
                  <strong>{task.title}</strong>
                  <div className="workspace-memory-meta">
                    {task.priority ? <span className="badge">{task.priority}</span> : null}
                    {task.status ? <span className="badge">{task.status}</span> : null}
                    {renderConfidenceBadge(task.confidence)}
                  </div>
                </div>
                <p>{task.summary || 'No summary available.'}</p>
                <small className="workspace-memory-footnote">{formatSourceSummary(task.sourceRefs)}</small>
              </article>
            ))}
          </WorkspaceMemorySection>
        </div>
      </div>
    </div>
  );
}

function WorkspaceMemorySection({
  title,
  count,
  emptyMessage,
  children,
}: {
  title: string;
  count: number;
  emptyMessage: string;
  children: ReactNode;
}) {
  return (
    <section className="workspace-memory-section">
      <div className="workspace-memory-section-header">
        <h4>{title}</h4>
        <span className="badge badge-count">{count}</span>
      </div>
      {count > 0 ? <div className="workspace-memory-list">{children}</div> : <p className="empty-inline">{emptyMessage}</p>}
    </section>
  );
}

function renderConfidenceBadge(confidence: number) {
  if (!Number.isFinite(confidence) || confidence <= 0) {
    return null;
  }
  return <span className="badge badge-accent">{Math.round(confidence * 100)}%</span>;
}

function formatSourceSummary(sourceRefs: WorkspaceKnowledgeSourceRef[]) {
  if (sourceRefs.length === 0) {
    return 'No source anchors';
  }

  const labels = sourceRefs.slice(0, 2).map(formatSourceRefLabel).filter(Boolean);
  const suffix = sourceRefs.length > 2 ? ` +${sourceRefs.length - 2}` : '';
  return `Sources: ${sourceRefs.length}${labels.length > 0 ? ` (${labels.join(' / ')}${suffix})` : ''}`;
}

function formatSourceRefLabel(sourceRef: WorkspaceKnowledgeSourceRef) {
  if (sourceRef.pageStart > 0 && sourceRef.pageEnd > 0) {
    return sourceRef.pageStart === sourceRef.pageEnd
      ? `p.${sourceRef.pageStart}`
      : `pp.${sourceRef.pageStart}-${sourceRef.pageEnd}`;
  }
  if (sourceRef.pageStart > 0) {
    return `p.${sourceRef.pageStart}`;
  }
  return sourceRef.sourceId;
}
