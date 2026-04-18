import { useEffect, useMemo, useState } from 'react';
import {
  LoaderCircle,
  RefreshCw,
  SquareX,
  Trash2,
} from 'lucide-react';
import { pdfTranslateApi } from '../api/pdfTranslate';
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
import type { CollectionTree } from '../types/zotero';
import type { ZoteroState } from '../store/zoteroStore';

interface HomePageProps {
  zotero: ZoteroState;
  onOpenPdf: (item: { id: string; title: string; pdfPath: string | null; itemType?: string; citeKey?: string }) => void;
}

export function HomePage({ zotero, onOpenPdf }: HomePageProps) {
  const [jobs, setJobs] = useState<PDFTranslateJobSnapshot[]>([]);
  const [jobError, setJobError] = useState<string | null>(null);
  const [isLoadingJobs, setIsLoadingJobs] = useState(false);
  const [cancellingJobId, setCancellingJobId] = useState<string | null>(null);
  const [deletingJobId, setDeletingJobId] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function loadJobs() {
      if (!cancelled) {
        setIsLoadingJobs(true);
      }
      try {
        const snapshots = await pdfTranslateApi.listJobs();
        if (!cancelled) {
          setJobs(snapshots);
          setJobError(null);
        }
      } catch (error) {
        if (!cancelled) {
          setJobError(error instanceof Error ? error.message : '加载翻译任务失败');
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
  const runningJobs = useMemo(
    () => jobs.filter((job) => job.status === 'running').length,
    [jobs],
  );

  async function handleRefreshJobs() {
    setIsLoadingJobs(true);
    try {
      const snapshots = await pdfTranslateApi.listJobs();
      setJobs(snapshots);
      setJobError(null);
    } catch (error) {
      setJobError(error instanceof Error ? error.message : '刷新翻译任务失败');
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

  return (
    <div className="home-page">
      <div className="home-body">
        <h2 className="home-section-title">文献库</h2>
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
                  <button
                    key={item.id}
                    type="button"
                    className="item-button item-button-card"
                    onDoubleClick={() => onOpenPdf({ id: item.id, title: item.title, pdfPath: item.pdfPath ?? null, itemType: item.itemType, citeKey: item.citeKey })}
                    onClick={() => zotero.selectItem(item)}
                  >
                    <strong>{item.title}</strong>
                    <small>{item.year || 'Unknown year'} · {item.citeKey} {item.hasPdf ? 'PDF' : ''}</small>
                  </button>
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
          {visibleJobs.length ? (
            <div className="home-task-list">
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
              <h3>暂无 PDF 翻译任务</h3>
              <p>在阅读页启动保留格式翻译或整本导出后，这里会统一显示任务进度、使用模型和当前状态。</p>
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
