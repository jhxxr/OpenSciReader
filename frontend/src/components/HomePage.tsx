import { useEffect, useMemo, useState } from 'react';
import {
  LoaderCircle,
  RefreshCw,
  Sparkles,
  SquareX,
  Trash2,
} from 'lucide-react';
import { pdfTranslateApi } from '../api/pdfTranslate';
import { workspaceWikiApi } from '../api/workspaceWiki';
import {
  formatRelativeTimestamp,
  getChunkCaption,
  getChunkStatusLabel,
  getJobLiveHint,
  getJobPrimaryStatus,
  getJobProgressPercent,
  getJobSecondaryStatus,
  getJobSummaryLine,
  shouldUseIndeterminateProgress,
} from '../lib/pdfTranslatePresentation';
import { Button } from './ui/Button';
import type { PDFTranslateJobSnapshot } from '../types/pdfTranslate';
import type { WorkspaceWikiScanJob } from '../types/workspaceWiki';
import type { CollectionTree } from '../types/zotero';
import type { ZoteroState } from '../store/zoteroStore';
import { useTabStore } from '../store/tabStore';
import type { WorkspaceState } from '../store/workspaceStore';

interface HomePageProps {
  zotero: ZoteroState;
  workspace: WorkspaceState;
  onImportFiles: () => Promise<void>;
  onImportZoteroItem: (item: { id: string; title: string; pdfPath: string; citeKey: string }) => Promise<void>;
  onOpenPdf: (item: { id: string; title: string; pdfPath: string | null; workspaceId?: string; documentId?: string; sourceKind?: 'workspace_document' | 'zotero_item'; itemType?: string; citeKey?: string }) => void;
  onEnterWorkspace: (workspaceId: string) => Promise<void>;
  onCreateWorkspace: () => Promise<void>;
}

export function HomePage({ zotero, workspace, onImportFiles, onImportZoteroItem, onOpenPdf, onEnterWorkspace, onCreateWorkspace }: HomePageProps) {
  const [jobs, setJobs] = useState<PDFTranslateJobSnapshot[]>([]);
  const [wikiJobs, setWikiJobs] = useState<WorkspaceWikiScanJob[]>([]);
  const [jobError, setJobError] = useState<string | null>(null);
  const [isLoadingJobs, setIsLoadingJobs] = useState(false);
  const [cancellingJobId, setCancellingJobId] = useState<string | null>(null);
  const [deletingJobId, setDeletingJobId] = useState<string | null>(null);
  const [deletingDocumentId, setDeletingDocumentId] = useState<string | null>(null);
  const tabs = useTabStore((state) => state.tabs);
  const closeTab = useTabStore((state) => state.closeTab);

  useEffect(() => {
    let cancelled = false;

    async function loadJobs() {
      if (!cancelled) {
        setIsLoadingJobs(true);
      }
      try {
        const [snapshots, wikiSnapshots] = await Promise.all([
          pdfTranslateApi.listJobs(),
          workspaceWikiApi.listJobs(),
        ]);
        if (!cancelled) {
          setJobs(filterHomeVisibleJobs(snapshots));
          setWikiJobs(filterHomeVisibleWikiJobs(wikiSnapshots));
          setJobError(null);
        }
      } catch (error) {
        if (!cancelled) {
          setJobError(error instanceof Error ? error.message : '加载任务失败');
        }
      } finally {
        if (!cancelled) {
          setIsLoadingJobs(false);
        }
      }
    }

    void loadJobs();
    const timer = window.setInterval(() => {
      void loadJobs();
    }, 5000);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, []);

  const visibleJobs = useMemo(() => jobs.slice(0, 10), [jobs]);
  const visibleWikiJobs = useMemo(() => wikiJobs.slice(0, 10), [wikiJobs]);
  const runningJobs = useMemo(
    () => jobs.filter((job) => job.status === 'running').length + wikiJobs.filter((job) => job.status === 'running').length,
    [jobs, wikiJobs],
  );

  async function handleRefreshJobs() {
    setIsLoadingJobs(true);
    try {
      const [snapshots, wikiSnapshots] = await Promise.all([
        pdfTranslateApi.listJobs(),
        workspaceWikiApi.listJobs(),
      ]);
      setJobs(filterHomeVisibleJobs(snapshots));
      setWikiJobs(filterHomeVisibleWikiJobs(wikiSnapshots));
      setJobError(null);
    } catch (error) {
      setJobError(error instanceof Error ? error.message : '刷新任务失败');
    } finally {
      setIsLoadingJobs(false);
    }
  }

  async function handleCancelJob(job: PDFTranslateJobSnapshot) {
    setCancellingJobId(job.jobId);
    try {
      const snapshot = await pdfTranslateApi.cancel(job.jobId);
      setJobs((current) =>
        [snapshot, ...current.filter((item) => item.jobId !== snapshot.jobId)].sort(compareJobsByTime),
      );
      setJobError(null);
    } catch (error) {
      setJobError(error instanceof Error ? error.message : '取消翻译任务失败');
    } finally {
      setCancellingJobId(null);
    }
  }

  async function handleDeleteJob(job: PDFTranslateJobSnapshot) {
    const confirmed =
      typeof window === 'undefined'
        ? true
        : window.confirm(`确认删除任务“${job.itemTitle || job.jobId}”？此操作会移除任务记录和本地产物目录。`);
    if (!confirmed) {
      return;
    }

    setDeletingJobId(job.jobId);
    try {
      await pdfTranslateApi.deleteJob(job.jobId);
      setJobs((current) => current.filter((item) => item.jobId !== job.jobId));
      setJobError(null);
    } catch (error) {
      setJobError(error instanceof Error ? error.message : '删除翻译任务失败');
    } finally {
      setDeletingJobId(null);
    }
  }

  function handleOpenTaskPdf(job: PDFTranslateJobSnapshot) {
    onOpenPdf({
      id: job.itemId || job.jobId,
      title: job.itemTitle || 'PDF Translate Task',
      pdfPath: job.pdfPath || null,
    });
  }

  async function handleDeleteWorkspaceDocument(document: { id: string; workspaceId: string; title: string; originalFileName: string }) {
    const label = document.title || document.originalFileName || document.id;
    const confirmed =
      typeof window === 'undefined'
        ? true
        : window.confirm('\u786e\u8ba4\u4ece\u5f53\u524d\u5de5\u4f5c\u533a\u5220\u9664\u6587\u6863 "' + label + '" \u5417\uff1f\u8fd9\u4f1a\u79fb\u9664\u672c\u5730\u526f\u672c\u3001\u7b14\u8bb0\u548c\u804a\u5929\u8bb0\u5f55\u3002');
    if (!confirmed) {
      return;
    }

    setDeletingDocumentId(document.id);
    try {
      await workspace.deleteDocument(document.id);
      const matchingTab = tabs.find(
        (tab) => tab.workspaceId === document.workspaceId && tab.documentId === document.id,
      );
      if (matchingTab) {
        closeTab(matchingTab.id);
      }
    } catch {
    } finally {
      setDeletingDocumentId(null);
    }
  }

  return (
    <div className="home-page">
      <div className="home-body">
        <h2 className="home-section-title">工作区</h2>
        <div className="home-workspace-grid">
          <div className="home-workspace-card home-workspace-primary-card">
            <div className="section-header">
              <div>
                <h3>{workspace.workspaces.find((item: { id: string }) => item.id === workspace.activeWorkspaceId)?.name ?? '未选择工作区'}</h3>
                <p className="home-workspace-subtitle">AI 工作台，可为不同研究任务分别建立上下文、文档集与后续 wiki。</p>
              </div>
              <span className="badge badge-count">{workspace.workspaces.length}</span>
            </div>
            <select
              className="workspace-switcher home-workspace-switcher"
              value={workspace.activeWorkspaceId ?? ''}
              onChange={(event) => void workspace.selectWorkspace(event.target.value)}
              disabled={workspace.isLoading || workspace.workspaces.length === 0}
            >
              {workspace.workspaces.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
            <p className="home-workspace-description">
              {workspace.workspaces.find((item: { id: string }) => item.id === workspace.activeWorkspaceId)?.description || '把论文导入到 OpenSciReader 自有数据目录，并围绕当前研究主题组织文档。后续这里会承载 AI 记忆与工作区级 wiki 扫描能力。'}
            </p>
            <div className="home-workspace-actions home-workspace-actions-stacked">
              <Button variant="default" size="sm" onClick={() => void onEnterWorkspace(workspace.activeWorkspaceId ?? '')} disabled={!workspace.activeWorkspaceId}>
                进入工作区
              </Button>
              <Button variant="secondary" size="sm" onClick={() => void onImportFiles()} disabled={workspace.isImporting || !workspace.activeWorkspaceId}>
                {workspace.isImporting ? '导入中...' : '导入文件'}
              </Button>
              <Button variant="secondary" size="sm" onClick={() => void onCreateWorkspace()}>
                新建工作区
              </Button>
            </div>
          </div>
          <div className="home-workspace-card">
            <div className="section-header">
              <h3>当前文档</h3>
              <span className="badge badge-count">{workspace.documents.length}</span>
            </div>
            {workspace.documents.length ? (
              <div className="library-item-list">
                {workspace.documents.map((document: { id: string; workspaceId: string; title: string; primaryPdfPath: string; originalFileName: string; sourceType: string }) => (
                  <div key={document.id} className="item-button-card home-workspace-document-card">
                    <button
                      type="button"
                      className="item-button home-workspace-document-button"
                      onDoubleClick={() => onOpenPdf({ id: document.id, title: document.title, pdfPath: document.primaryPdfPath ?? null, workspaceId: document.workspaceId, documentId: document.id, sourceKind: 'workspace_document' })}
                    >
                    <strong>{document.title}</strong>
                    <small>{document.originalFileName || '已导入文档'} · {document.sourceType}</small>
                    </button>
                    <div className="home-workspace-document-actions">
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => onOpenPdf({ id: document.id, title: document.title, pdfPath: document.primaryPdfPath ?? null, workspaceId: document.workspaceId, documentId: document.id, sourceKind: 'workspace_document' })}
                      >
                        打开 PDF
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="home-workspace-document-delete danger"
                        onClick={() => void handleDeleteWorkspaceDocument(document)}
                        disabled={deletingDocumentId === document.id}
                        aria-label={`删除 ${document.title}`}
                        title={`删除 ${document.title}`}
                      >
                        {deletingDocumentId === document.id ? <LoaderCircle size={14} className="spin-inline" /> : <Trash2 size={14} />}
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            ) : <p className="empty-inline">当前工作区还没有文档，先创建或进入一个工作区，再导入 PDF 或 Markdown。</p>}
          </div>
        </div>

        <h2 className="home-section-title">导入源</h2>
        <div className="home-library">
          <div className="home-library-tree">
            <div className="section-header"><h3>文献目录</h3><span className="badge badge-count">{zotero.collections.length}</span></div>
            <p style={{ margin: '0 0 8px', fontSize: 12, color: 'var(--osr-text-muted)' }}>
              {zotero.demoMode ? '演示模式：请确认 Zotero 正在运行。' : '已连接 Zotero 本地 API'}
            </p>
            <div className="tree-list">
              {zotero.collections.map((collection) => (
                <CollectionNode key={collection.id} node={collection} selectedCollectionId={zotero.selectedCollectionId} onSelect={(id) => void zotero.selectCollection(id)} />
              ))}
            </div>
          </div>

          <div className="home-library-items">
            <div className="section-header"><h3>当前文献</h3><span className="badge badge-count">{zotero.items.length}</span></div>
            {zotero.items.length ? (
              <div className="library-item-list">
                {zotero.items.map((item) => (
                  <div key={item.id} className="item-button item-button-card">
                    <button
                      type="button"
                      className="item-button item-button-card"
                      onDoubleClick={() => onOpenPdf({ id: item.id, title: item.title, pdfPath: item.pdfPath ?? null, sourceKind: 'zotero_item', itemType: item.itemType, citeKey: item.citeKey })}
                      onClick={() => zotero.selectItem(item)}
                    >
                      <strong>{item.title}</strong>
                      <small>{item.year || 'Unknown year'} · {item.citeKey} {item.hasPdf ? 'PDF' : ''}</small>
                    </button>
                    <div className="home-workspace-actions">
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => void onImportZoteroItem({ id: item.id, title: item.title, pdfPath: item.pdfPath, citeKey: item.citeKey })}
                        disabled={!item.pdfPath || workspace.isImporting || !workspace.activeWorkspaceId}
                      >
                        导入到当前工作区
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            ) : <p className="empty-inline">从左侧目录选择后，这里会列出当前集合的文献条目。</p>}
          </div>
        </div>

        <div className="section-header home-task-header">
          <h2 className="home-section-title">
            任务中心
            {runningJobs > 0 ? <span className="badge badge-accent">{runningJobs} 运行中</span> : null}
          </h2>
          <Button variant="secondary" size="sm" onClick={() => void handleRefreshJobs()} disabled={isLoadingJobs}>
            <RefreshCw size={14} className={isLoadingJobs ? 'spin-inline' : ''} />
            {isLoadingJobs ? '刷新中...' : '刷新任务'}
          </Button>
        </div>
        <div className="home-task-center">
          {visibleJobs.length || visibleWikiJobs.length ? (
            <div className="home-task-list">
              {visibleWikiJobs.map((job) => {
                const progress = Math.max(0, Math.min(100, (job.overallProgress ?? 0) * 100));
                return (
                  <article key={job.jobId} className="home-task-card">
                    <div className="home-task-card-head">
                      <div>
                        <div className="home-task-card-meta">
                          <span className={`badge badge-accent badge-job-${job.status}`}>{job.status}</span>
                          <span className="badge badge-count">Wiki 扫描</span>
                          <span className="badge badge-count">{job.processedItems}/{job.totalItems || 0}</span>
                        </div>
                        <h3>{workspace.workspaces.find((item) => item.id === job.workspaceId)?.name || '工作区 Wiki 任务'}</h3>
                        <p>{job.currentItem || '扫描工作区文档与目录'}</p>
                      </div>
                    </div>

                    <div className="home-task-stage">
                      <div className="home-task-stage-head">
                        <strong>{job.message || '正在扫描工作区内容'}</strong>
                        <span>{progress.toFixed(1)}%</span>
                      </div>
                      <p>{job.currentStage || '等待扫描事件...'}</p>
                    </div>

                    <div className="translate-progress">
                      <div className="translate-progress-bar">
                        <div className="translate-progress-fill" style={{ width: `${Math.max(4, progress)}%` }} />
                      </div>
                      <div className="translate-progress-meta">
                        <span>失败 {job.failedItems} 项</span>
                        <span>最近更新：{formatRelativeTimestamp(job.updatedAt || job.startedAt)}</span>
                      </div>
                    </div>

                    {job.error ? <div className="reader-error">{job.error}</div> : null}
                  </article>
                );
              })}
              {visibleJobs.map((job) => {
                const progress = getJobProgressPercent(job);
                const indeterminate = shouldUseIndeterminateProgress(job);
                return (
                  <article key={job.jobId} className="home-task-card">
                    <div className="home-task-card-head">
                      <div>
                        <div className="home-task-card-meta">
                          <span className={`badge badge-accent badge-job-${job.status}`}>{job.status}</span>
                          <span className="badge badge-count">{job.mode === 'preview' ? '预览' : '导出'}</span>
                          <span className="badge badge-count">{job.pageCount} 页</span>
                        </div>
                        <h3>{job.itemTitle || '未命名 PDF 任务'}</h3>
                        <p>{job.providerName} / {job.modelId}</p>
                      </div>
                      <div className="row-actions">
                        <Button variant="secondary" size="sm" onClick={() => handleOpenTaskPdf(job)}>
                          打开 PDF
                        </Button>
                        <Button
                          variant="secondary"
                          size="sm"
                          onClick={() => void handleCancelJob(job)}
                          disabled={job.status !== 'running' || cancellingJobId === job.jobId}
                        >
                          <SquareX size={14} />
                          {cancellingJobId === job.jobId ? '取消中...' : '取消任务'}
                        </Button>
                        <Button
                          variant="secondary"
                          size="sm"
                          onClick={() => void handleDeleteJob(job)}
                          disabled={job.status === 'running' || deletingJobId === job.jobId}
                        >
                          <Trash2 size={14} />
                          {deletingJobId === job.jobId ? '删除中...' : '删除任务'}
                        </Button>
                      </div>
                    </div>

                    <div className="home-task-stage">
                      <div className="home-task-stage-head">
                        <strong>{getJobPrimaryStatus(job)}</strong>
                        <span>{progress.toFixed(1)}%</span>
                      </div>
                      <p>{getJobSecondaryStatus(job) || '等待新的翻译事件...'}</p>
                    </div>

                    <div className="translate-progress">
                      <div className="translate-progress-bar">
                        <div
                          className={`translate-progress-fill ${indeterminate ? 'translate-progress-fill-indeterminate' : ''}`}
                          style={{ width: `${Math.max(indeterminate ? 28 : 4, Math.min(100, progress))}%` }}
                        />
                      </div>
                      <div className="translate-progress-meta">
                        <span>{getJobSummaryLine(job)}</span>
                        <span>最近更新：{formatRelativeTimestamp(job.updatedAt || job.startedAt || job.createdAt)}</span>
                      </div>
                    </div>

                    <div className="home-task-chunks">
                      {job.chunks.map((chunk) => (
                        <span
                          key={`${job.jobId}-${chunk.index}`}
                          className={`home-task-chunk home-task-chunk-${chunk.status}`}
                        >
                          {getChunkCaption(chunk)} · {getChunkStatusLabel(chunk.status)}
                        </span>
                      ))}
                    </div>

                    <p className="home-task-live-hint">{getJobLiveHint(job)}</p>

                    {job.error ? <div className="reader-error">{job.error}</div> : null}
                  </article>
                );
              })}
            </div>
          ) : (
            <div className="home-task-empty">
              <div className="feature-card-icon"><LoaderCircle size={22} /></div>
              <h3>暂无任务</h3>
              <p>在阅读页启动 PDF 翻译或在工作区中发起 wiki 扫描后，这里会统一显示任务进度与状态。</p>
            </div>
          )}
          {jobError ? <div className="reader-error">{jobError}</div> : null}
        </div>
      </div>
    </div>
  );
}

function CollectionNode({
  node,
  selectedCollectionId,
  onSelect,
}: {
  node: CollectionTree;
  selectedCollectionId: string | null;
  onSelect: (id: string) => void;
}) {
  return (
    <div className="tree-node">
      <button
        type="button"
        className={`tree-button ${selectedCollectionId === node.id ? 'tree-button-active' : ''}`}
        onClick={() => onSelect(node.id)}
      >
        {node.name}
      </button>
      {node.children.length ? (
        <div className="tree-children">
          {node.children.map((child) => (
            <CollectionNode key={child.id} node={child} selectedCollectionId={selectedCollectionId} onSelect={onSelect} />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function compareJobsByTime(a: PDFTranslateJobSnapshot, b: PDFTranslateJobSnapshot) {
  const parseSnapshotTime = (job: PDFTranslateJobSnapshot) =>
    Date.parse(job.updatedAt || job.finishedAt || job.startedAt || job.createdAt || '') || 0;
  return parseSnapshotTime(b) - parseSnapshotTime(a);
}

function filterHomeVisibleJobs(jobs: PDFTranslateJobSnapshot[]) {
  return jobs.filter((job) => job.mode === 'preview').sort(compareJobsByTime);
}

function filterHomeVisibleWikiJobs(jobs: WorkspaceWikiScanJob[]) {
  return [...jobs].sort((a, b) => {
    const left = Date.parse(a.updatedAt || a.finishedAt || a.startedAt || '') || 0;
    const right = Date.parse(b.updatedAt || b.finishedAt || b.startedAt || '') || 0;
    return right - left;
  });
}
