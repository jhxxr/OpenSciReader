import * as Tabs from "@radix-ui/react-tabs";
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type MutableRefObject,
  type PointerEvent as ReactPointerEvent,
} from "react";
import {
  CircleAlert,
  Image as ImageIcon,
  Languages,
  List,
  MessageSquareText,
  PanelRightClose,
  PanelRightOpen,
  Settings2,
  Sparkles,
} from "lucide-react";
import { phase6Api } from "../api/phase6";
import { notesApi } from "../api/notes";
import { pdfTranslateApi } from "../api/pdfTranslate";
import {
  coercePerformanceValue,
  getPDFTranslatePerformancePreset,
} from "../lib/pdfTranslatePerformance";
import {
  getChunkCaption,
  getChunkStatusLabel,
  getJobLiveHint,
  getJobPrimaryStatus,
  getJobProgressPercent,
  getJobSecondaryStatus,
  getJobSummaryLine,
  shouldUseIndeterminateProgress,
} from "../lib/pdfTranslatePresentation";
import { toReaderUrl } from "../lib/pdfUrl";
import { Button } from "./ui/Button";
import { DualPdfReader } from "./DualPdfReader";
import { ReaderAIPanel } from "./ReaderAIPanel";
import { extractFigureCandidates } from "../lib/pdfFigures";
import { useReaderStore } from "../store/readerStore";
import type { TabItem } from "../store/tabStore";
import type { ReaderOutlineItem } from "../types/pdf";
import type { ReaderNoteEntry } from "../types/notes";
import type { PDFTranslateRuntimeConfig } from "../types/config";
import {
  applyPDFTranslateEvent,
  buildPDFTranslatedPageMap,
  type PDFTranslateJobSnapshot,
} from "../types/pdfTranslate";
import { useProviderConfigs, useConfigStore } from "../store/configStore";

type ReaderSidebarTab = "outline" | "figures" | "translate" | "assistant" | "notes";
type TranslatePerformanceMode = "auto" | "manual";

export function ReaderTab({
  tab,
  providerConfigs,
}: {
  tab: TabItem;
  providerConfigs: ReturnType<typeof useProviderConfigs>;
}) {
  const selection = useReaderStore((state) => state.selection);
  const snapshot = useReaderStore((state) => state.snapshot);
  const runtimeConfig = useConfigStore((state) => state.snapshot.pdfTranslateRuntime);
  const activePage = useReaderStore((state) => state.activePage);
  const jumpToPage = useReaderStore((state) => state.jumpToPage);
  const navigateToAnchor = useReaderStore((state) => state.navigateToAnchor);
  const figureCaptures = useReaderStore((state) => state.figureCaptures);
  const addFigureCaptures = useReaderStore((state) => state.addFigureCaptures);
  const setSnapshot = useReaderStore((state) => state.setSnapshot);

  const [outline, setOutline] = useState<ReaderOutlineItem[]>([]);
  const [sidebarOpen, setSidebarOpen] = useState(() => readReaderUIState().sidebarOpen);
  const [activeSidebarTab, setActiveSidebarTab] = useState<ReaderSidebarTab>(
    () => readReaderUIState().activeSidebarTab,
  );
  const [translateProviderId, setTranslateProviderId] = useState<number | null>(
    () => readReaderUIState().translateProviderId,
  );
  const [llmProviderId, setLlmProviderId] = useState<number | null>(
    () => readReaderUIState().llmProviderId,
  );
  const [llmModelId, setLlmModelId] = useState<number | null>(
    () => readReaderUIState().llmModelId,
  );
  const [readerPageCount, setReaderPageCount] = useState(0);
  const [sourceLang, setSourceLang] = useState("EN");
  const [targetLang, setTargetLang] = useState("ZH");
  const [translation, setTranslation] = useState("");
  const [translationError, setTranslationError] = useState<string | null>(null);
  const [isTranslating, setIsTranslating] = useState(false);
  const [figureImportError, setFigureImportError] = useState<string | null>(null);
  const [isImportingFigures, setIsImportingFigures] = useState(false);
  const [noteDraft, setNoteDraft] = useState("");
  const [notes, setNotes] = useState<ReaderNoteEntry[]>([]);
  const [previewJob, setPreviewJob] = useState<PDFTranslateJobSnapshot | null>(null);
  const [exportJob, setExportJob] = useState<PDFTranslateJobSnapshot | null>(null);
  const [pdfTranslateError, setPDFTranslateError] = useState<string | null>(null);
  const [isStartingPreview, setIsStartingPreview] = useState(false);
  const [isStartingExport, setIsStartingExport] = useState(false);
  const [exportMaxPagesPerPart, setExportMaxPagesPerPart] = useState(120);
  const [translatePerformanceMode, setTranslatePerformanceMode] =
    useState<TranslatePerformanceMode>(
      () => readReaderUIState().translatePerformanceMode,
    );
  const [translateQPS, setTranslateQPS] = useState(
    () => readReaderUIState().translateQPS,
  );
  const [poolMaxWorkers, setPoolMaxWorkers] = useState(
    () => readReaderUIState().poolMaxWorkers,
  );
  const [termPoolMaxWorkers, setTermPoolMaxWorkers] = useState(
    () => readReaderUIState().termPoolMaxWorkers,
  );
  const [showPerformanceHelp, setShowPerformanceHelp] = useState(false);

  const previewSubscriptionRef = useRef<(() => void) | null>(null);
  const exportSubscriptionRef = useRef<(() => void) | null>(null);

  const groupedProviders = useMemo(
    () => ({
      llm: providerConfigs.filter((item) => item.provider.type === "llm"),
      drawing: providerConfigs.filter((item) => item.provider.type === "drawing"),
      translate: providerConfigs.filter((item) => item.provider.type === "translate"),
    }),
    [providerConfigs],
  );

  const activeLLMConfig = useMemo(
    () =>
      groupedProviders.llm.find((item) => item.provider.id === llmProviderId) ??
      null,
    [groupedProviders.llm, llmProviderId],
  );

  const activeLLMModel = useMemo(
    () =>
      activeLLMConfig?.models.find((model) => model.id === llmModelId) ?? null,
    [activeLLMConfig, llmModelId],
  );

  const translatePerformancePreset = useMemo(
    () =>
      getPDFTranslatePerformancePreset(
        activeLLMConfig?.provider ?? null,
        activeLLMModel,
      ),
    [activeLLMConfig, activeLLMModel],
  );

  const effectiveTranslatePerformance = useMemo(
    () => ({
      qps:
        translatePerformanceMode === "manual"
          ? coercePerformanceValue(
              translateQPS,
              translatePerformancePreset.qps,
            )
          : translatePerformancePreset.qps,
      poolMaxWorkers:
        translatePerformanceMode === "manual"
          ? coercePerformanceValue(
              poolMaxWorkers,
              translatePerformancePreset.poolMaxWorkers,
            )
          : translatePerformancePreset.poolMaxWorkers,
      termPoolMaxWorkers:
        translatePerformanceMode === "manual"
          ? coercePerformanceValue(
              termPoolMaxWorkers,
              translatePerformancePreset.termPoolMaxWorkers,
            )
          : translatePerformancePreset.termPoolMaxWorkers,
    }),
    [
      poolMaxWorkers,
      termPoolMaxWorkers,
      translatePerformanceMode,
      translatePerformancePreset,
      translateQPS,
    ],
  );

  useEffect(() => {
    setTranslateProviderId(
      (value) => value ?? groupedProviders.translate[0]?.provider.id ?? null,
    );
  }, [groupedProviders.translate]);

  useEffect(() => {
    setLlmProviderId(
      (value) => value ?? groupedProviders.llm[0]?.provider.id ?? null,
    );
  }, [groupedProviders.llm]);

  useEffect(() => {
    setLlmModelId((value) =>
      activeLLMConfig?.models.some((model) => model.id === value)
        ? value
        : (activeLLMConfig?.models[0]?.id ?? null),
    );
  }, [activeLLMConfig]);

  useEffect(() => {
    setFigureImportError(null);
    setReaderPageCount(0);
    setPreviewJob(null);
    setExportJob(null);
    setPDFTranslateError(null);
    closeSubscription(previewSubscriptionRef);
    closeSubscription(exportSubscriptionRef);
  }, [tab.id, tab.pdfPath]);

  useEffect(() => {
    writeReaderUIState({
      sidebarOpen,
      activeSidebarTab,
      translateProviderId,
      llmProviderId,
      llmModelId,
      translatePerformanceMode,
      translateQPS,
      poolMaxWorkers,
      termPoolMaxWorkers,
    });
  }, [
    sidebarOpen,
    activeSidebarTab,
    translateProviderId,
    llmProviderId,
    llmModelId,
    translatePerformanceMode,
    translateQPS,
    poolMaxWorkers,
    termPoolMaxWorkers,
  ]);

  useEffect(() => {
    let cancelled = false;

    if (!tab.id) {
      setNotes([]);
      return () => {
        cancelled = true;
      };
    }

    void notesApi.listReaderNotes(tab.id).then((entries) => {
      if (!cancelled) {
        setNotes(entries);
      }
    });

    return () => {
      cancelled = true;
    };
  }, [tab.id]);

  useEffect(() => {
    setTranslation("");
    setTranslationError(null);
  }, [selection.cleaned, sourceLang, targetLang, translateProviderId]);

  useEffect(() => {
    let cancelled = false;

    if (!tab.pdfPath) {
      return () => {
        cancelled = true;
      };
    }

    void pdfTranslateApi
      .listJobs()
      .then((jobs) => {
        if (cancelled) {
          return;
        }
        const relatedJobs = jobs.filter(
          (job) =>
            (tab.id && job.itemId === tab.id) ||
            (tab.pdfPath && job.pdfPath === tab.pdfPath),
        );
        const sortedJobs = relatedJobs.sort(compareTranslateJobsByTime);
        const latestPreviewJob =
          sortedJobs.find((job) => job.mode === "preview") ?? null;
        const latestExportJob =
          sortedJobs.find((job) => job.mode === "export") ?? null;

        setPreviewJob(latestPreviewJob);
        setExportJob(latestExportJob);

        if (latestPreviewJob?.status === "running") {
          bindPDFTranslateEvents("preview", latestPreviewJob.jobId);
        }
        if (latestExportJob?.status === "running") {
          bindPDFTranslateEvents("export", latestExportJob.jobId);
        }
      })
      .catch((error) => {
        if (!cancelled) {
          setPDFTranslateError(
            error instanceof Error ? error.message : "恢复历史翻译任务失败",
          );
        }
      });

    return () => {
      cancelled = true;
    };
  }, [tab.id, tab.pdfPath]);

  useEffect(
    () => () => {
      closeSubscription(previewSubscriptionRef);
      closeSubscription(exportSubscriptionRef);
    },
    [],
  );

  useEffect(() => {
    if (!previewJob || previewJob.status !== "running") {
      return;
    }
    const timer = window.setInterval(() => {
      void pdfTranslateApi
        .getStatus(previewJob.jobId)
        .then((snapshot) => setPreviewJob(snapshot))
        .catch((error) => {
          setPDFTranslateError(
            error instanceof Error ? error.message : "获取预览任务状态失败",
          );
        });
    }, 5000);
    return () => window.clearInterval(timer);
  }, [previewJob?.jobId, previewJob?.status]);

  useEffect(() => {
    if (!exportJob || exportJob.status !== "running") {
      return;
    }
    const timer = window.setInterval(() => {
      void pdfTranslateApi
        .getStatus(exportJob.jobId)
        .then((snapshot) => setExportJob(snapshot))
        .catch((error) => {
          setPDFTranslateError(
            error instanceof Error ? error.message : "获取导出任务状态失败",
          );
        });
    }, 5000);
    return () => window.clearInterval(timer);
  }, [exportJob?.jobId, exportJob?.status]);

  const currentFigureCaptures = figureCaptures.filter(
    (capture) => capture.itemId === (tab.pdfPath ?? ""),
  );
  const selectedFigureCapture = useMemo(
    () =>
      currentFigureCaptures.find(
        (capture) => capture.dataUrl === snapshot?.dataUrl,
      ) ??
      currentFigureCaptures[0] ??
      null,
    [currentFigureCaptures, snapshot?.dataUrl],
  );
  const currentItemNotes = notes.filter((note) => note.itemId === tab.id);
  const translatedPageMap = useMemo(
    () => buildPDFTranslatedPageMap(previewJob),
    [previewJob],
  );
  const translatedPreviewEnabled = Boolean(
    previewJob &&
      (previewJob.status === "running" ||
        previewJob.chunks.some((chunk) => chunk.status === "completed")),
  );
  const previewStatusText = useMemo(
    () => buildPreviewStatusText(previewJob),
    [previewJob],
  );

  async function handleTranslate() {
    if (!translateProviderId || !selection.cleaned) {
      return;
    }

    setIsTranslating(true);
    setTranslationError(null);

    try {
      const translated = await phase6Api.proxyTranslation(
        translateProviderId,
        0,
        selection.cleaned,
        sourceLang,
        targetLang,
      );
      setTranslation(translated);
    } catch (error) {
      setTranslationError(error instanceof Error ? error.message : "翻译失败");
    } finally {
      setIsTranslating(false);
    }
  }

  function handleSaveNote() {
    if (!tab.id || noteDraft.trim() === "") {
      return;
    }

    void notesApi
      .saveReaderNote({
        itemId: tab.id,
        itemTitle: tab.title,
        page: activePage,
        anchorText: selection.cleaned,
        content: noteDraft.trim(),
      })
      .then((entry) => {
        setNotes((current) => [entry, ...current]);
      });

    setNoteDraft("");
  }

  function handleExportNotes() {
    if (!currentItemNotes.length || !tab.id) {
      return;
    }

    const content = [
      `# ${tab.title}`,
      "",
      ...currentItemNotes.flatMap((note) => [
        `## Page ${note.page}`,
        note.anchorText ? `Anchor: ${note.anchorText}` : "",
        note.content,
        "",
      ]),
    ].join("\n");

    downloadTextFile(`${sanitizeFileName(tab.title)}-notes.md`, content);
  }

  async function handleBulkImportFigures() {
    if (!tab.pdfPath || isImportingFigures) {
      return;
    }

    setIsImportingFigures(true);
    setFigureImportError(null);

    try {
      const candidates = await extractFigureCandidates(tab.pdfPath);
      if (!candidates.length) {
        setFigureImportError("未识别到高置信度图表候选，请切换到对应页后使用区域截图。");
        return;
      }

      const existingKeys = new Set(
        currentFigureCaptures.map((capture) => `${capture.page}:${capture.title}`),
      );
      const freshCaptures = candidates
        .filter((candidate) => {
          const key = `${candidate.page}:${candidate.title}`;
          if (existingKeys.has(key)) {
            return false;
          }
          existingKeys.add(key);
          return true;
        })
        .map((candidate, index) => ({
          id: `${Date.now()}-${index}`,
          itemId: tab.pdfPath ?? "unknown",
          page: candidate.page,
          title: candidate.title,
          dataUrl: candidate.dataUrl,
          width: candidate.width,
          height: candidate.height,
          createdAt: new Date().toISOString(),
        }));

      if (!freshCaptures.length) {
        setActiveSidebarTab("figures");
        if (currentFigureCaptures[0]) {
          setSnapshot(currentFigureCaptures[0]);
          jumpToPage(currentFigureCaptures[0].page);
        }
        setFigureImportError("本次识别结果与现有图表库重复，没有新增截图。");
        return;
      }

      addFigureCaptures(freshCaptures);
      setActiveSidebarTab("figures");
      setSnapshot(freshCaptures[0]);
      jumpToPage(freshCaptures[0].page);
    } catch (error) {
      setFigureImportError(error instanceof Error ? error.message : "自动提取图表失败");
    } finally {
      setIsImportingFigures(false);
    }
  }

  function bindPDFTranslateEvents(kind: "preview" | "export", jobId: string) {
    const subscriptionRef =
      kind === "preview" ? previewSubscriptionRef : exportSubscriptionRef;
    const setJob = kind === "preview" ? setPreviewJob : setExportJob;
    closeSubscription(subscriptionRef);

    subscriptionRef.current = pdfTranslateApi.subscribe(
      jobId,
      (event) => {
        setJob((current) => applyPDFTranslateEvent(current, event));
        if (event.status) {
          return;
        }
        if (event.error) {
          setPDFTranslateError(event.error);
        }
        if (
          event.type === "chunk_finished" ||
          event.type === "finish" ||
          event.type === "cancelled" ||
          event.type === "error"
        ) {
          void pdfTranslateApi
            .getStatus(jobId)
            .then((snapshot) => setJob(snapshot))
            .catch((error) => {
              setPDFTranslateError(
                error instanceof Error ? error.message : "获取翻译状态失败",
              );
            });
        }
      },
      (error) => {
        setPDFTranslateError(error.message);
      },
    );

    void pdfTranslateApi
      .getStatus(jobId)
      .then((snapshot) => setJob(snapshot))
      .catch((error) => {
        setPDFTranslateError(
          error instanceof Error ? error.message : "获取翻译状态失败",
        );
      });
  }

  function resetTranslatePerformanceToPreset() {
    setTranslateQPS(translatePerformancePreset.qps);
    setPoolMaxWorkers(translatePerformancePreset.poolMaxWorkers);
    setTermPoolMaxWorkers(translatePerformancePreset.termPoolMaxWorkers);
  }

  async function handleStartPreviewTranslation(retryJobId = "") {
    if (runtimeConfig.status !== "valid") {
      setPDFTranslateError(getPDFTranslateRuntimeBlockedMessage(runtimeConfig));
      setSidebarOpen(true);
      return;
    }
    if (!tab.pdfPath || !readerPageCount || !llmProviderId || !llmModelId) {
      setPDFTranslateError("请先打开 PDF，并选择可用的 LLM Provider / Model。");
      return;
    }

    setIsStartingPreview(true);
    setPDFTranslateError(null);
    try {
      const nextPreviewJob = await pdfTranslateApi.start({
        pdfPath: tab.pdfPath,
        pageCount: readerPageCount,
        itemId: tab.id,
        itemTitle: tab.title,
        sourceLang: normalizeLayoutTranslateLanguage(sourceLang),
        targetLang: normalizeLayoutTranslateLanguage(targetLang),
        mode: "preview",
        previewChunkPages: 25,
        maxPagesPerPart: 0,
        qps: effectiveTranslatePerformance.qps,
        poolMaxWorkers: effectiveTranslatePerformance.poolMaxWorkers,
        termPoolMaxWorkers: effectiveTranslatePerformance.termPoolMaxWorkers,
        retryJobId,
        llmProviderId,
        llmModelId,
      });
      setPreviewJob(nextPreviewJob);
      bindPDFTranslateEvents("preview", nextPreviewJob.jobId);
    } catch (error) {
      setPDFTranslateError(
        error instanceof Error ? error.message : "启动保留格式翻译预览失败",
      );
    } finally {
      setIsStartingPreview(false);
    }
  }

  async function handleStartExportTranslation() {
    if (runtimeConfig.status !== "valid") {
      setPDFTranslateError(getPDFTranslateRuntimeBlockedMessage(runtimeConfig));
      setSidebarOpen(true);
      return;
    }
    if (!tab.pdfPath || !readerPageCount || !llmProviderId || !llmModelId) {
      setPDFTranslateError("请先打开 PDF，并选择可用的 LLM Provider / Model。");
      return;
    }

    setIsStartingExport(true);
    setPDFTranslateError(null);
    try {
      const nextExportJob = await pdfTranslateApi.start({
        pdfPath: tab.pdfPath,
        pageCount: readerPageCount,
        itemId: tab.id,
        itemTitle: tab.title,
        sourceLang: normalizeLayoutTranslateLanguage(sourceLang),
        targetLang: normalizeLayoutTranslateLanguage(targetLang),
        mode: "export",
        previewChunkPages: 25,
        maxPagesPerPart: Math.max(0, exportMaxPagesPerPart || 0),
        qps: effectiveTranslatePerformance.qps,
        poolMaxWorkers: effectiveTranslatePerformance.poolMaxWorkers,
        termPoolMaxWorkers: effectiveTranslatePerformance.termPoolMaxWorkers,
        reusePreviewJobId: previewJob?.status === "completed" ? previewJob.jobId : "",
        llmProviderId,
        llmModelId,
      });
      setExportJob(nextExportJob);
      bindPDFTranslateEvents("export", nextExportJob.jobId);
    } catch (error) {
      setPDFTranslateError(error instanceof Error ? error.message : "启动导出任务失败");
    } finally {
      setIsStartingExport(false);
    }
  }

  async function handleCancelTranslation(
    job: PDFTranslateJobSnapshot | null,
    kind: "preview" | "export",
  ) {
    if (!job) {
      return;
    }

    try {
      const cancelledJob = await pdfTranslateApi.cancel(job.jobId);
      if (kind === "preview") {
        setPreviewJob(cancelledJob);
      } else {
        setExportJob(cancelledJob);
      }
    } catch (error) {
      setPDFTranslateError(error instanceof Error ? error.message : "取消翻译任务失败");
    }
  }

  function handleDownloadPDF(path: string | undefined) {
    if (!path) {
      return;
    }
    const link = document.createElement("a");
    link.href = toReaderUrl(path);
    link.download = "";
    link.click();
  }

  function openTranslatePanel() {
    setSidebarOpen(true);
    setActiveSidebarTab("translate");
  }

  return (
    <div className="reader-view">
      <div className="reader-main">
        <div className="reader-placeholder">
          <div className="reader-frame">
            <DualPdfReader
              pdfPath={tab.pdfPath || null}
              onOutlineChange={setOutline}
              onPageCountChange={setReaderPageCount}
              onOpenTranslatePanel={openTranslatePanel}
              translatedPages={translatedPageMap}
              translatedPreviewEnabled={translatedPreviewEnabled}
              translatedStatusText={previewStatusText}
            />
          </div>
        </div>
      </div>

      {!sidebarOpen ? (
        <button
          type="button"
          className="sidebar-toggle-btn"
          style={{ position: "fixed", right: 0, zIndex: 20 }}
          onClick={() => setSidebarOpen(true)}
        >
          <PanelRightOpen size={16} />
        </button>
      ) : null}

      <aside
        className={`reader-sidebar ${sidebarOpen ? "" : "reader-sidebar-collapsed"}`}
      >
        <div className="reader-sidebar-header">
          <h3>{tab.title}</h3>
          <button
            type="button"
            className="icon-button"
            onClick={() => setSidebarOpen(false)}
          >
            <PanelRightClose size={16} />
          </button>
        </div>

        <Tabs.Root
          className="tabs-root"
          value={activeSidebarTab}
          onValueChange={(value) => setActiveSidebarTab(value as ReaderSidebarTab)}
        >
          <Tabs.List className="tabs-list" aria-label="Sidebar Tabs">
            <Tabs.Trigger className="tabs-trigger" value="outline" title="目录">
              <List size={20} />
              <span>目录</span>
            </Tabs.Trigger>
            <Tabs.Trigger className="tabs-trigger" value="figures" title="图表">
              <ImageIcon size={20} />
              <span>图表</span>
            </Tabs.Trigger>
            <Tabs.Trigger className="tabs-trigger" value="translate" title="翻译">
              <Languages size={20} />
              <span>翻译</span>
            </Tabs.Trigger>
            <Tabs.Trigger className="tabs-trigger" value="assistant" title="AI 助手">
              <Sparkles size={20} />
              <span>AI</span>
            </Tabs.Trigger>
            <Tabs.Trigger className="tabs-trigger" value="notes" title="笔记">
              <MessageSquareText size={20} />
              <span>笔记</span>
            </Tabs.Trigger>
          </Tabs.List>

          <Tabs.Content className="tabs-content" value="outline" forceMount>
            <div className="section-header">
              <h3>文档目录</h3>
              <span className="badge badge-count">{countOutlineItems(outline)}</span>
            </div>
            <div className="tree-list" style={{ overflowY: "auto", flex: 1, minHeight: 0 }}>
              {outline.length ? (
                outline.map((item, index) => (
                  <OutlineNode
                    key={`${item.title}-${index}`}
                    node={item}
                    onJump={jumpToPage}
                  />
                ))
              ) : (
                <p className="empty-inline">打开 PDF 后显示目录。</p>
              )}
            </div>
          </Tabs.Content>

          <Tabs.Content className="tabs-content" value="figures" forceMount>
            <div className="section-header">
              <h3>候选图表</h3>
              <span className="badge badge-count">{currentFigureCaptures.length}</span>
            </div>
            <div className="tree-list" style={{ overflowY: "auto", flex: 1, minHeight: 0 }}>
              {selectedFigureCapture ? (
                <div className="figure-preview-card">
                  <img
                    className="figure-preview-image"
                    src={selectedFigureCapture.dataUrl}
                    alt={selectedFigureCapture.title}
                    loading="lazy"
                  />
                  <div className="figure-preview-meta">
                    <strong>{selectedFigureCapture.title}</strong>
                    <small>
                      P{selectedFigureCapture.page} | {selectedFigureCapture.width}x{selectedFigureCapture.height}
                    </small>
                  </div>
                </div>
              ) : null}
              {currentFigureCaptures.length ? (
                <div className="figure-capture-list">
                  {currentFigureCaptures.map((capture) => (
                    <button
                      key={capture.id}
                      type="button"
                      className={`item-button item-button-card figure-capture-item ${selectedFigureCapture?.id === capture.id ? "item-button-active" : ""}`}
                      onClick={() => {
                        setSnapshot(capture);
                        jumpToPage(capture.page);
                      }}
                    >
                      <img
                        className="figure-capture-thumb"
                        src={capture.dataUrl}
                        alt={capture.title}
                        loading="lazy"
                      />
                      <strong>{capture.title}</strong>
                      <small>
                        P{capture.page} | {capture.width}x{capture.height}
                      </small>
                    </button>
                  ))}
                </div>
              ) : (
                <p className="empty-inline">暂时还没有图表截图。</p>
              )}
              {figureImportError ? <div className="reader-error">{figureImportError}</div> : null}
              {tab.pdfPath ? (
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => void handleBulkImportFigures()}
                  disabled={isImportingFigures}
                >
                  {isImportingFigures ? "提取中..." : "自动提取图表候选"}
                </Button>
              ) : null}
            </div>
          </Tabs.Content>

          <Tabs.Content className="tabs-content" value="translate" forceMount>
            <div className="context-card">
              <strong>保留格式翻译</strong>
              <p>
                预览模式按 25 页一块翻译。某个 chunk 完成后，阅读区右栏只替换对应页位，不维护一个实时增长的整本预览 PDF。
              </p>
            </div>
            {runtimeConfig.status !== "valid" ? (
              <div className="runtime-blocked-card">
                <div>
                  <strong>PDF 翻译运行时未就绪</strong>
                  <p>{getPDFTranslateRuntimeBlockedMessage(runtimeConfig)}</p>
                </div>
                <Button variant="secondary" size="sm" onClick={() => useConfigStore.setState({ error: '请点击右上角设置，先导入 PDF 翻译运行时。' })}>
                  <Settings2 size={14} />
                  前往设置导入
                </Button>
              </div>
            ) : null}
            <div className="context-card">
              <span className="badge badge-accent">{readerPageCount || "--"} 页</span>
              <strong>
                {activeLLMModel
                  ? `${activeLLMConfig?.provider.name ?? "LLM"} / ${activeLLMModel.modelId}`
                  : "未选择排版翻译模型"}
              </strong>
              <p>排版翻译统一走现有 LLM Provider / Model 配置，不复用普通划词翻译渠道。</p>
            </div>
            <div className="lang-row">
              <label className="field">
                <span>源语言</span>
                <select value={sourceLang} onChange={(event) => setSourceLang(event.target.value)}>
                  <option value="EN">EN</option>
                  <option value="DE">DE</option>
                  <option value="FR">FR</option>
                  <option value="JA">JA</option>
                  <option value="AUTO">AUTO</option>
                </select>
              </label>
              <label className="field">
                <span>目标语言</span>
                <select value={targetLang} onChange={(event) => setTargetLang(event.target.value)}>
                  <option value="ZH">ZH</option>
                  <option value="EN">EN</option>
                  <option value="JA">JA</option>
                </select>
              </label>
            </div>
            <label className="field">
              <span>排版翻译 LLM Provider</span>
              <select
                value={llmProviderId ?? ""}
                onChange={(event) => setLlmProviderId(Number(event.target.value) || null)}
              >
                <option value="">请选择</option>
                {groupedProviders.llm.map((item) => (
                  <option key={item.provider.id} value={item.provider.id}>
                    {item.provider.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="field">
              <span>排版翻译模型</span>
              <select
                value={llmModelId ?? ""}
                onChange={(event) => setLlmModelId(Number(event.target.value) || null)}
                disabled={!activeLLMConfig}
              >
                <option value="">请选择</option>
                {(activeLLMConfig?.models ?? []).map((model) => (
                  <option key={model.id} value={model.id}>
                    {model.modelId}
                  </option>
                ))}
              </select>
            </label>
            <div className="context-card">
              <div className="performance-panel-head">
                <div className="performance-panel-copy">
                  <strong>并发设置</strong>
                  <p>
                    自动预设会根据当前 Provider / Model 给一个起始值。接口稳定就逐步上调，遇到 429
                    或超时就下调。
                  </p>
                </div>
                <button
                  type="button"
                  className="field-help-trigger"
                  aria-label="说明并发设置"
                  aria-expanded={showPerformanceHelp}
                  onClick={() => setShowPerformanceHelp((value) => !value)}
                >
                  <CircleAlert size={16} />
                </button>
              </div>
              {showPerformanceHelp ? (
                <div className="field-help-card">
                  <strong>这些设置分别控制什么</strong>
                  <p>`QPS`：每秒最多发多少次模型请求。</p>
                  <p>`pool_max_workers`：正式翻译阶段，同时并行处理多少段内容。</p>
                  <p>`term_pool_max_workers`：自动术语提取阶段的并发数。</p>
                </div>
              ) : null}
              <label className="field">
                <span>模式</span>
                <select
                  value={translatePerformanceMode}
                  onChange={(event) =>
                    setTranslatePerformanceMode(
                      event.target.value as TranslatePerformanceMode,
                    )
                  }
                >
                  <option value="auto">自动预设</option>
                  <option value="manual">手动调整</option>
                </select>
              </label>
              <p className="field-hint">
                当前预设：{translatePerformancePreset.label} · QPS{" "}
                {translatePerformancePreset.qps} · Pool{" "}
                {translatePerformancePreset.poolMaxWorkers} · Term Pool{" "}
                {translatePerformancePreset.termPoolMaxWorkers}
              </p>
              <p className="field-hint">{translatePerformancePreset.reason}</p>
              <div className="performance-grid">
                <label className="field">
                  <span>QPS</span>
                  <input
                    type="number"
                    min="1"
                    step="1"
                    value={
                      translatePerformanceMode === "manual"
                        ? translateQPS
                        : translatePerformancePreset.qps
                    }
                    disabled={translatePerformanceMode !== "manual"}
                    onChange={(event) =>
                      setTranslateQPS(Number(event.target.value) || 0)
                    }
                  />
                </label>
                <label className="field">
                  <span>pool_max_workers</span>
                  <input
                    type="number"
                    min="1"
                    step="1"
                    value={
                      translatePerformanceMode === "manual"
                        ? poolMaxWorkers
                        : translatePerformancePreset.poolMaxWorkers
                    }
                    disabled={translatePerformanceMode !== "manual"}
                    onChange={(event) =>
                      setPoolMaxWorkers(Number(event.target.value) || 0)
                    }
                  />
                </label>
                <label className="field">
                  <span>term_pool_max_workers</span>
                  <input
                    type="number"
                    min="1"
                    step="1"
                    value={
                      translatePerformanceMode === "manual"
                        ? termPoolMaxWorkers
                        : translatePerformancePreset.termPoolMaxWorkers
                    }
                    disabled={translatePerformanceMode !== "manual"}
                    onChange={(event) =>
                      setTermPoolMaxWorkers(Number(event.target.value) || 0)
                    }
                  />
                </label>
              </div>
              <div className="row-actions">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={resetTranslatePerformanceToPreset}
                  disabled={translatePerformanceMode !== "manual"}
                >
                  恢复当前模型预设
                </Button>
                <span className="field-hint">
                  本次生效：QPS {effectiveTranslatePerformance.qps} · Pool{" "}
                  {effectiveTranslatePerformance.poolMaxWorkers} · Term Pool{" "}
                  {effectiveTranslatePerformance.termPoolMaxWorkers}
                </span>
              </div>
            </div>
            <div className="row-actions">
              <Button
                onClick={() => void handleStartPreviewTranslation()}
                disabled={isStartingPreview || !tab.pdfPath || !readerPageCount || !llmProviderId || !llmModelId}
              >
                {isStartingPreview ? "启动中..." : "开始保留格式翻译预览"}
              </Button>
              <Button
                variant="secondary"
                onClick={() => void handleCancelTranslation(previewJob, "preview")}
                disabled={!previewJob || previewJob.status !== "running"}
              >
                取消预览任务
              </Button>
              <Button
                variant="secondary"
                onClick={() => void handleStartPreviewTranslation(previewJob?.jobId ?? "")}
                disabled={!previewJob}
              >
                重试预览任务
              </Button>
            </div>
            {previewJob ? (
              <div className="context-card">
                <strong>预览任务</strong>
                <p>
                  Job {previewJob.jobId} · {previewJob.status} · 已完成{" "}
                  {previewJob.chunks.filter((chunk) => chunk.status === "completed").length}/
                  {previewJob.chunks.length} 个 chunk
                </p>
                <div className="task-status-panel">
                  <div className="task-status-panel-head">
                    <strong>{getJobPrimaryStatus(previewJob)}</strong>
                    <span>{getJobProgressPercent(previewJob).toFixed(1)}%</span>
                  </div>
                  <p>{getJobSecondaryStatus(previewJob) || "等待新的翻译事件..."}</p>
                </div>
                <div className="translate-progress">
                  <div className="translate-progress-bar">
                    <div
                      className={`translate-progress-fill ${shouldUseIndeterminateProgress(previewJob) ? "translate-progress-fill-indeterminate" : ""}`}
                      style={{
                        width: `${Math.max(
                          shouldUseIndeterminateProgress(previewJob) ? 28 : 4,
                          Math.min(100, getJobProgressPercent(previewJob)),
                        )}%`,
                      }}
                    />
                  </div>
                  <div className="translate-progress-meta">
                    <span>{getJobSummaryLine(previewJob)}</span>
                  </div>
                </div>
                <div className="task-chunk-list">
                  {previewJob.chunks.map((chunk) => (
                    <span
                      key={`${previewJob.jobId}-${chunk.index}`}
                      className={`task-chunk-pill task-chunk-pill-${chunk.status}`}
                    >
                      {getChunkCaption(chunk)} · {getChunkStatusLabel(chunk.status)}
                    </span>
                  ))}
                </div>
                <p className="empty-inline">{getJobLiveHint(previewJob)}</p>
                {previewJob.error ? <p className="empty-inline">{previewJob.error}</p> : null}
              </div>
            ) : null}
            <div className="context-card">
              <strong>整本导出</strong>
              <p>
                导出模式会整本重新执行一次，产出 mono + dual PDF。大文件可通过 max-pages-per-part 自动切分。
              </p>
            </div>
            <label className="field">
              <span>max-pages-per-part</span>
              <input
                type="number"
                min="0"
                value={exportMaxPagesPerPart}
                onChange={(event) => setExportMaxPagesPerPart(Number(event.target.value) || 0)}
              />
            </label>
            <div className="row-actions">
              <Button
                onClick={() => void handleStartExportTranslation()}
                disabled={isStartingExport || !tab.pdfPath || !readerPageCount || !llmProviderId || !llmModelId}
              >
                {isStartingExport ? "导出启动中..." : "开始导出 mono + dual"}
              </Button>
              <Button
                variant="secondary"
                onClick={() => void handleCancelTranslation(exportJob, "export")}
                disabled={!exportJob || exportJob.status !== "running"}
              >
                取消导出任务
              </Button>
            </div>
            {exportJob ? (
              <div className="context-card">
                <strong>导出任务</strong>
                <p>
                  Job {exportJob.jobId} · {exportJob.status}
                  {exportJob.outputs.totalSeconds
                    ? ` · ${exportJob.outputs.totalSeconds.toFixed(1)}s`
                    : ""}
                </p>
                <div className="task-status-panel">
                  <div className="task-status-panel-head">
                    <strong>{getJobPrimaryStatus(exportJob)}</strong>
                    <span>{getJobProgressPercent(exportJob).toFixed(1)}%</span>
                  </div>
                  <p>{getJobSecondaryStatus(exportJob) || "等待新的翻译事件..."}</p>
                </div>
                <div className="translate-progress">
                  <div className="translate-progress-bar">
                    <div
                      className={`translate-progress-fill ${shouldUseIndeterminateProgress(exportJob) ? "translate-progress-fill-indeterminate" : ""}`}
                      style={{
                        width: `${Math.max(
                          shouldUseIndeterminateProgress(exportJob) ? 28 : 4,
                          Math.min(100, getJobProgressPercent(exportJob)),
                        )}%`,
                      }}
                    />
                  </div>
                  <div className="translate-progress-meta">
                    <span>{getJobSummaryLine(exportJob)}</span>
                  </div>
                </div>
                <div className="task-chunk-list">
                  {exportJob.chunks.map((chunk) => (
                    <span
                      key={`${exportJob.jobId}-${chunk.index}`}
                      className={`task-chunk-pill task-chunk-pill-${chunk.status}`}
                    >
                      {getChunkCaption(chunk)} · {getChunkStatusLabel(chunk.status)}
                    </span>
                  ))}
                </div>
                <div className="row-actions">
                  <Button
                    variant="secondary"
                    onClick={() =>
                      handleDownloadPDF(
                        exportJob.outputs.noWatermarkMonoPdfPath || exportJob.outputs.monoPdfPath,
                      )
                    }
                    disabled={!exportJob.outputs.noWatermarkMonoPdfPath && !exportJob.outputs.monoPdfPath}
                  >
                    下载译文 PDF
                  </Button>
                  <Button
                    variant="secondary"
                    onClick={() =>
                      handleDownloadPDF(
                        exportJob.outputs.noWatermarkDualPdfPath || exportJob.outputs.dualPdfPath,
                      )
                    }
                    disabled={!exportJob.outputs.noWatermarkDualPdfPath && !exportJob.outputs.dualPdfPath}
                  >
                    下载对照 PDF
                  </Button>
                </div>
                <p className="empty-inline">{getJobLiveHint(exportJob)}</p>
                {exportJob.error ? <p className="empty-inline">{exportJob.error}</p> : null}
              </div>
            ) : null}
            {pdfTranslateError ? <div className="reader-error">{pdfTranslateError}</div> : null}

            <div className="context-card">
              <strong>划词直译</strong>
              <p>下面的快捷翻译仍然保留，适合对当前选区做即时直译。</p>
            </div>
            <div className="context-card">
              <span className="badge badge-accent">P{activePage}</span>
              <strong>当前选区直译</strong>
              <p>
                {selection.cleaned
                  ? "下面的直译结果只针对当前选区。"
                  : "在 PDF 上划词后，这里会显示当前选中的文本。"}
              </p>
            </div>
            <ReadOnlyOutput
              value={selection.cleaned}
              placeholder="当前划词会显示在这里。"
              defaultHeight={96}
              maxHeightRatio={0.35}
            />
            <label className="field">
              <span>选区直译渠道</span>
              <select
                value={translateProviderId ?? ""}
                onChange={(event) => setTranslateProviderId(Number(event.target.value) || null)}
              >
                <option value="">请选择</option>
                {groupedProviders.translate.map((item) => (
                  <option key={item.provider.id} value={item.provider.id}>
                    {item.provider.name}
                  </option>
                ))}
              </select>
            </label>
            <div className="lang-row">
              <label className="field">
                <span>源语言</span>
                <select value={sourceLang} onChange={(event) => setSourceLang(event.target.value)}>
                  <option value="EN">EN</option>
                  <option value="DE">DE</option>
                  <option value="FR">FR</option>
                  <option value="JA">JA</option>
                  <option value="AUTO">AUTO</option>
                </select>
              </label>
              <label className="field">
                <span>目标语言</span>
                <select value={targetLang} onChange={(event) => setTargetLang(event.target.value)}>
                  <option value="ZH">ZH</option>
                  <option value="EN">EN</option>
                  <option value="JA">JA</option>
                </select>
              </label>
            </div>
            <Button
              onClick={() => void handleTranslate()}
              disabled={isTranslating || !translateProviderId || !selection.cleaned}
            >
              {isTranslating ? "翻译中..." : "翻译当前划词"}
            </Button>
            {translationError ? <div className="reader-error">{translationError}</div> : null}
            <ReadOnlyOutput
              value={translation}
              placeholder="当前划词的直译结果会显示在这里。"
            />
          </Tabs.Content>

          <Tabs.Content className="tabs-content" value="assistant" forceMount>
            <ReaderAIPanel
              tab={tab}
              llmConfigs={groupedProviders.llm}
              drawingConfigs={groupedProviders.drawing}
              activeLLMConfig={activeLLMConfig}
              activeLLMModel={activeLLMModel}
              llmProviderId={llmProviderId}
              llmModelId={llmModelId}
              setLlmProviderId={setLlmProviderId}
              setLlmModelId={setLlmModelId}
            />
          </Tabs.Content>

          <Tabs.Content className="tabs-content" value="notes" forceMount>
            {selection.cleaned ? (
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setNoteDraft((current) => current || selection.cleaned)}
              >
                插入当前划词
              </Button>
            ) : null}
            <label className="field">
              <span>Markdown Note</span>
              <textarea
                className="prompt-input"
                value={noteDraft}
                onChange={(event) => setNoteDraft(event.target.value)}
              />
            </label>
            <Button onClick={handleSaveNote} disabled={!tab.id || noteDraft.trim() === ""}>
              保存笔记
            </Button>
            <Button
              variant="secondary"
              onClick={handleExportNotes}
              disabled={!currentItemNotes.length}
            >
              导出笔记
            </Button>
            <div className="note-list">
              {currentItemNotes.map((note) => (
                <article key={note.id} className="note-card">
                  <div className="note-meta">
                    <span className="badge badge-accent">P{note.page}</span>
                    <small>{new Date(note.createdAt).toLocaleString()}</small>
                  </div>
                  {note.anchorText ? (
                    <button
                      type="button"
                      className="text-button"
                      onClick={() => navigateToAnchor(note.page, note.anchorText)}
                    >
                      跳转到锚点
                    </button>
                  ) : null}
                  <p>{note.content}</p>
                </article>
              ))}
            </div>
          </Tabs.Content>
        </Tabs.Root>
      </aside>
    </div>
  );
}

function OutlineNode({
  node,
  onJump,
}: {
  node: ReaderOutlineItem;
  onJump: (page: number) => void;
}) {
  return (
    <div className="tree-node">
      {node.pageNumber ? (
        <button
          type="button"
          className="outline-row outline-row-button"
          onClick={() => onJump(node.pageNumber ?? 1)}
        >
          <span>{node.title}</span>
          <small>P{node.pageNumber}</small>
        </button>
      ) : (
        <div className="outline-row">
          <span>{node.title}</span>
          <small />
        </div>
      )}
      {node.items.length ? (
        <div className="tree-children">
          {node.items.map((child, index) => (
            <OutlineNode
              key={`${child.title}-${index}`}
              node={child}
              onJump={onJump}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function ReadOnlyOutput({
  value,
  placeholder,
  className = "",
  defaultHeight = 120,
  maxHeightRatio = 0.6,
}: {
  value: string;
  placeholder: string;
  className?: string;
  defaultHeight?: number;
  maxHeightRatio?: number;
}) {
  const outputClassName = className ? `chat-output ${className}` : "chat-output";
  const [height, setHeight] = useState(defaultHeight);
  const stopResizeRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    setHeight(defaultHeight);
  }, [defaultHeight, value === ""]);

  useEffect(() => () => stopResizeRef.current?.(), []);

  const handleResizeStart = (event: ReactPointerEvent<HTMLButtonElement>) => {
    event.preventDefault();

    const startY = event.clientY;
    const startHeight = height;
    const maxHeight =
      typeof window === "undefined"
        ? Math.round(defaultHeight / maxHeightRatio)
        : Math.round(window.innerHeight * maxHeightRatio);

    const cleanup = () => {
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerup", handlePointerUp);
      window.removeEventListener("pointercancel", handlePointerUp);
      document.body.style.removeProperty("user-select");
      document.body.style.removeProperty("cursor");
      stopResizeRef.current = null;
    };

    const handlePointerMove = (moveEvent: PointerEvent) => {
      const nextHeight = Math.min(
        maxHeight,
        Math.max(defaultHeight, startHeight + moveEvent.clientY - startY),
      );
      setHeight(nextHeight);
    };

    const handlePointerUp = () => {
      cleanup();
    };

    stopResizeRef.current?.();
    stopResizeRef.current = cleanup;
    document.body.style.setProperty("user-select", "none");
    document.body.style.setProperty("cursor", "ns-resize");
    window.addEventListener("pointermove", handlePointerMove);
    window.addEventListener("pointerup", handlePointerUp);
    window.addEventListener("pointercancel", handlePointerUp);
  };

  return (
    <div className="chat-output-wrapper">
      <textarea
        className={outputClassName}
        value={value}
        placeholder={placeholder}
        readOnly
        spellCheck={false}
        style={{ height }}
      />
      <button
        type="button"
        className="chat-output-resizer"
        aria-label="调整结果框高度"
        onPointerDown={handleResizeStart}
      >
        <span />
      </button>
    </div>
  );
}

function countOutlineItems(items: ReaderOutlineItem[]): number {
  return items.reduce(
    (total, item) => total + 1 + countOutlineItems(item.items),
    0,
  );
}

function sanitizeFileName(input: string) {
  return input.replace(/[\\/:*?"<>|]/g, "_").slice(0, 80);
}

function downloadTextFile(filename: string, content: string) {
  const blob = new Blob([content], { type: "text/markdown;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

function getPDFTranslateRuntimeBlockedMessage(runtimeConfig: PDFTranslateRuntimeConfig) {
  if (runtimeConfig.lastValidationError.trim()) {
    return `当前运行时不可用：${runtimeConfig.lastValidationError}`;
  }
  if (runtimeConfig.status === "installing") {
    return "PDF 翻译运行时仍在安装中，请稍后再试。";
  }
  if (runtimeConfig.status === "invalid") {
    return "PDF 翻译运行时校验失败，请重新导入正确的运行时包。";
  }
  return "PDF 翻译运行时尚未安装，请先在设置中导入独立运行时包。";
}

function closeSubscription(subscriptionRef: MutableRefObject<(() => void) | null>) {
  if (!subscriptionRef.current) {
    return;
  }
  subscriptionRef.current();
  subscriptionRef.current = null;
}

function buildPreviewStatusText(job: PDFTranslateJobSnapshot | null) {
  if (!job) {
    return "等待开始保留格式翻译预览。";
  }
  const completedChunks = job.chunks.filter((chunk) => chunk.status === "completed").length;
  const totalChunks = job.chunks.length;
  if (job.status === "running") {
    return `${job.currentStage || "翻译中"} · 已完成 ${completedChunks}/${totalChunks} 个 chunk`;
  }
  if (job.status === "completed") {
    return `预览完成 · 共 ${totalChunks} 个 chunk`;
  }
  if (job.status === "cancelled") {
    return `预览已取消 · 已完成 ${completedChunks}/${totalChunks} 个 chunk`;
  }
  if (job.status === "failed") {
    return `预览失败 · ${job.error || "请重试当前任务"}`;
  }
  return "等待 worker 事件...";
}

function compareTranslateJobsByTime(a: PDFTranslateJobSnapshot, b: PDFTranslateJobSnapshot) {
  const getTime = (job: PDFTranslateJobSnapshot) =>
    Date.parse(job.updatedAt || job.finishedAt || job.startedAt || job.createdAt || "") || 0;
  return getTime(b) - getTime(a);
}

function normalizeLayoutTranslateLanguage(value: string) {
  switch (value.trim().toUpperCase()) {
    case "ZH":
      return "zh-CN";
    case "EN":
      return "en";
    case "JA":
      return "ja";
    case "DE":
      return "de";
    case "FR":
      return "fr";
    case "AUTO":
      return "auto";
    default:
      return value.trim() || "auto";
  }
}

interface ReaderUIState {
  sidebarOpen: boolean;
  activeSidebarTab: ReaderSidebarTab;
  translateProviderId: number | null;
  llmProviderId: number | null;
  llmModelId: number | null;
  translatePerformanceMode: TranslatePerformanceMode;
  translateQPS: number;
  poolMaxWorkers: number;
  termPoolMaxWorkers: number;
}

const READER_UI_STORAGE_KEY = "osr:reader-ui-state";

function defaultReaderUIState(): ReaderUIState {
  return {
    sidebarOpen: true,
    activeSidebarTab: "outline",
    translateProviderId: null,
    llmProviderId: null,
    llmModelId: null,
    translatePerformanceMode: "auto",
    translateQPS: 6,
    poolMaxWorkers: 6,
    termPoolMaxWorkers: 3,
  };
}

function readReaderUIState(): ReaderUIState {
  if (typeof window === "undefined") {
    return defaultReaderUIState();
  }

  try {
    const raw = window.localStorage.getItem(READER_UI_STORAGE_KEY);
    if (!raw) {
      return defaultReaderUIState();
    }

    const parsed = JSON.parse(raw) as Partial<ReaderUIState> & {
      activeSidebarTab?: string;
    };
    const allowedTabs: ReaderSidebarTab[] = [
      "outline",
      "figures",
      "translate",
      "assistant",
      "notes",
    ];
    const activeSidebarTab = allowedTabs.includes(
      parsed.activeSidebarTab as ReaderSidebarTab,
    )
      ? (parsed.activeSidebarTab as ReaderSidebarTab)
      : "translate";
    const translatePerformanceMode =
      parsed.translatePerformanceMode === "manual" ? "manual" : "auto";
    return {
      ...defaultReaderUIState(),
      ...parsed,
      activeSidebarTab,
      translatePerformanceMode,
    };
  } catch {
    return defaultReaderUIState();
  }
}

function writeReaderUIState(state: ReaderUIState) {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(READER_UI_STORAGE_KEY, JSON.stringify(state));
}
