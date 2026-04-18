import {
  useEffect,
  useRef,
  useState,
  type Dispatch,
  type KeyboardEvent as ReactKeyboardEvent,
  type MutableRefObject,
  type MouseEvent as ReactMouseEvent,
  type SetStateAction,
  type WheelEvent as ReactWheelEvent,
} from "react";
import { Camera, ChevronDown, ChevronUp, Languages, ZoomIn, ZoomOut } from "lucide-react";
import html2canvas from "html2canvas";
import * as pdfjsLib from "pdfjs-dist";
import { Button } from "./ui/Button";
import type { ReaderOutlineItem } from "../types/pdf";
import type { PDFTranslatedPageRender } from "../types/pdfTranslate";
import { useReaderStore } from "../store/readerStore";
import {
  destroyPDFDocument,
  openPDFDocument,
  type LoadedPDFDocument,
} from "../lib/pdfDocument";
import { loadReaderOutline } from "../lib/pdfOutline";

interface ReaderProps {
  pdfPath: string | null;
  onOutlineChange: (items: ReaderOutlineItem[]) => void;
  onPageCountChange: (pageCount: number) => void;
  onOpenTranslatePanel: () => void;
  translatedPages?: Record<number, PDFTranslatedPageRender>;
  translatedPreviewEnabled?: boolean;
  translatedStatusText?: string;
}

interface DragRect {
  pageNumber: number;
  left: number;
  top: number;
  width: number;
  height: number;
}

type TranslatedPageState = "empty" | "loading" | "ready" | "error";
type PDFRenderTask = ReturnType<pdfjsLib.PDFPageProxy["render"]>;

const DEFAULT_SCALE = 1.1;
const MIN_SCALE = 0.6;
const MAX_SCALE = 2.2;
const SCALE_STEP = 0.1;

export function DualPdfReader({
  pdfPath,
  onOutlineChange,
  onPageCountChange,
  onOpenTranslatePanel,
  translatedPages = {},
  translatedPreviewEnabled = false,
  translatedStatusText = "",
}: ReaderProps) {
  const viewerRef = useRef<HTMLDivElement | null>(null);
  const rowRefs = useRef<Record<number, HTMLDivElement | null>>({});
  const surfaceRefs = useRef<Record<number, HTMLDivElement | null>>({});
  const canvasRefs = useRef<Record<number, HTMLCanvasElement | null>>({});
  const translatedSurfaceRefs = useRef<Record<number, HTMLDivElement | null>>({});
  const translatedCanvasRefs = useRef<Record<number, HTMLCanvasElement | null>>({});
  const textLayerRefs = useRef<Record<number, HTMLDivElement | null>>({});
  const textLayerTasksRef = useRef<Map<number, pdfjsLib.TextLayer>>(new Map());
  const originalRenderTasksRef = useRef<Map<number, PDFRenderTask>>(new Map());
  const translatedRenderTasksRef = useRef<Map<number, PDFRenderTask>>(new Map());
  const pdfRef = useRef<pdfjsLib.PDFDocumentProxy | null>(null);
  const translatedDocsRef = useRef<Map<string, LoadedPDFDocument>>(new Map());
  const translatedDocPromisesRef = useRef<Map<string, Promise<LoadedPDFDocument>>>(new Map());
  const translatedRenderKeysRef = useRef<Record<number, string>>({});
  const dragStartRef = useRef<{ pageNumber: number; x: number; y: number } | null>(null);

  const selection = useReaderStore((state) => state.selection);
  const setSelection = useReaderStore((state) => state.setSelection);
  const clearSelection = useReaderStore((state) => state.clearSelection);
  const setSnapshot = useReaderStore((state) => state.setSnapshot);
  const activePage = useReaderStore((state) => state.activePage);
  const setActivePage = useReaderStore((state) => state.setActivePage);
  const desiredPage = useReaderStore((state) => state.desiredPage);
  const clearAnchor = useReaderStore((state) => state.clearAnchor);
  const anchorText = useReaderStore((state) => state.anchorText);
  const addFigureCapture = useReaderStore((state) => state.addFigureCapture);

  const [pageCount, setPageCount] = useState(0);
  const [scale, setScale] = useState(DEFAULT_SCALE);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [translatedError, setTranslatedError] = useState<string | null>(null);
  const [translatedPageStates, setTranslatedPageStates] = useState<
    Record<number, TranslatedPageState>
  >({});
  const [captureMode, setCaptureMode] = useState(false);
  const [dragRect, setDragRect] = useState<DragRect | null>(null);

  useEffect(() => {
    textLayerTasksRef.current.forEach((task) => task.cancel());
    textLayerTasksRef.current.clear();
    cancelRenderTasks(originalRenderTasksRef.current);
    cancelRenderTasks(translatedRenderTasksRef.current);
    rowRefs.current = {};
    surfaceRefs.current = {};
    canvasRefs.current = {};
    translatedSurfaceRefs.current = {};
    translatedCanvasRefs.current = {};
    textLayerRefs.current = {};
    translatedRenderKeysRef.current = {};
    pdfRef.current = null;
    setPageCount(0);
    setError(null);
    setTranslatedError(null);
    setTranslatedPageStates({});
    setCaptureMode(false);
    setDragRect(null);
    dragStartRef.current = null;
    setActivePage(1);
    onPageCountChange(0);
    if (!pdfPath) {
      clearSelection();
      setSnapshot(null);
    }
  }, [clearSelection, onPageCountChange, pdfPath, setActivePage, setSnapshot]);

  useEffect(() => {
    let disposed = false;
    void destroyLoadedDocuments(translatedDocsRef.current).then(() => {
      if (disposed) {
        return;
      }
      translatedDocsRef.current.clear();
      translatedDocPromisesRef.current.clear();
    });
    return () => {
      disposed = true;
    };
  }, [pdfPath]);

  useEffect(() => {
    let cancelled = false;
    let loadingTask: pdfjsLib.PDFDocumentLoadingTask | null = null;

    if (!pdfPath) {
      onOutlineChange([]);
      return () => {
        cancelled = true;
      };
    }

    const currentPdfPath = pdfPath;

    async function renderOriginalPDF() {
      setIsLoading(true);
      setError(null);
      onOutlineChange([]);
      textLayerTasksRef.current.forEach((task) => task.cancel());
      textLayerTasksRef.current.clear();
      cancelRenderTasks(originalRenderTasksRef.current);

      try {
        const loaded = await openPDFDocument(currentPdfPath);
        loadingTask = loaded.loadingTask;
        const pdf = loaded.pdf;
        pdfRef.current = pdf;

        if (cancelled) {
          return;
        }

        setPageCount(pdf.numPages);
        onPageCountChange(pdf.numPages);
        onOutlineChange(await loadReaderOutline(currentPdfPath, pdf));

        await waitForPageRefs(pdf.numPages, rowRefs);
        if (cancelled) {
          return;
        }

        for (let pageNumber = 1; pageNumber <= pdf.numPages; pageNumber += 1) {
          if (cancelled) {
            return;
          }
          const surface = surfaceRefs.current[pageNumber];
          const canvas = canvasRefs.current[pageNumber];
          const textLayer = textLayerRefs.current[pageNumber];
          if (!surface || !canvas || !textLayer) {
            continue;
          }

          textLayer.replaceChildren();
          const page = await pdf.getPage(pageNumber);
          await renderPageToCanvas(
            pageNumber,
            page,
            surface,
            canvas,
            scale,
            originalRenderTasksRef,
          );

          const nextTextLayer = new pdfjsLib.TextLayer({
            textContentSource: page.streamTextContent(),
            container: textLayer,
            viewport: page.getViewport({ scale }),
          });
          textLayerTasksRef.current.set(pageNumber, nextTextLayer);
          await nextTextLayer.render();
        }

        if (!cancelled) {
          syncActivePageFromScroll(viewerRef, rowRefs, pdf.numPages, setActivePage);
        }
      } catch (readerError) {
        if (!cancelled) {
          setError(readerError instanceof Error ? readerError.message : "Failed to load PDF");
          onOutlineChange([]);
          onPageCountChange(0);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void renderOriginalPDF();

    return () => {
      cancelled = true;
      pdfRef.current = null;
      cancelRenderTasks(originalRenderTasksRef.current);
      textLayerTasksRef.current.forEach((task) => task.cancel());
      textLayerTasksRef.current.clear();
      void destroyPDFDocument(loadingTask);
    };
  }, [onOutlineChange, onPageCountChange, pdfPath, scale, setActivePage]);

  useEffect(() => {
    if (!desiredPage || !pageCount) {
      return;
    }
    const target = rowRefs.current[desiredPage];
    if (!target) {
      return;
    }
    target.scrollIntoView({ behavior: "smooth", block: "start" });
    const timer = window.setTimeout(() => clearAnchor(), anchorText ? 1200 : 250);
    return () => window.clearTimeout(timer);
  }, [anchorText, clearAnchor, desiredPage, pageCount]);

  useEffect(() => {
    const layers = Object.values(textLayerRefs.current);
    const anchorProbe = anchorText.trim().slice(0, Math.min(anchorText.trim().length, 18));
    for (const layer of layers) {
      if (!layer) {
        continue;
      }
      const spans = Array.from(layer.querySelectorAll("span"));
      for (const span of spans) {
        const matches = Boolean(anchorProbe) && (span.textContent ?? "").includes(anchorProbe);
        span.classList.toggle("text-layer-item-anchor", matches);
      }
    }
  }, [anchorText, pageCount]);

  useEffect(() => {
    let cancelled = false;

    async function renderTranslatedPages() {
      if (!pageCount) {
        return;
      }

      if (!translatedPreviewEnabled) {
        cancelRenderTasks(translatedRenderTasksRef.current);
        translatedRenderKeysRef.current = {};
        setTranslatedError(null);
        for (let pageNumber = 1; pageNumber <= pageCount; pageNumber += 1) {
          clearRenderedSurface(
            translatedSurfaceRefs.current[pageNumber],
            translatedCanvasRefs.current[pageNumber],
          );
        }
        setTranslatedPageStates({});
        return;
      }

      setTranslatedError(null);
      for (let pageNumber = 1; pageNumber <= pageCount; pageNumber += 1) {
        if (cancelled) {
          return;
        }

        const mapping = translatedPages[pageNumber];
        const surface = translatedSurfaceRefs.current[pageNumber];
        const canvas = translatedCanvasRefs.current[pageNumber];
        if (!surface || !canvas) {
          continue;
        }

        const renderKey = mapping
          ? `${mapping.pdfPath}#${mapping.translatedPageNumber}@${scale}`
          : `empty@${scale}`;
        if (translatedRenderKeysRef.current[pageNumber] === renderKey) {
          continue;
        }
        translatedRenderKeysRef.current[pageNumber] = renderKey;

        if (!mapping) {
          setTranslatedPageState(setTranslatedPageStates, pageNumber, "empty");
          clearRenderedSurface(surface, canvas);
          continue;
        }

        setTranslatedPageState(setTranslatedPageStates, pageNumber, "loading");
        try {
          const loaded = await getCachedTranslatedDocument(
            translatedDocsRef,
            translatedDocPromisesRef,
            mapping.pdfPath,
          );
          if (cancelled) {
            return;
          }
          const translatedPage = await loaded.pdf.getPage(mapping.translatedPageNumber);
          if (cancelled) {
            return;
          }
          await renderPageToCanvas(
            pageNumber,
            translatedPage,
            surface,
            canvas,
            scale,
            translatedRenderTasksRef,
          );
          setTranslatedPageState(setTranslatedPageStates, pageNumber, "ready");
        } catch (renderError) {
          if (!cancelled) {
            clearRenderedSurface(surface, canvas);
            setTranslatedPageState(setTranslatedPageStates, pageNumber, "error");
            setTranslatedError(
              renderError instanceof Error
                ? renderError.message
                : "Failed to render translated PDF page",
            );
          }
        }
      }
    }

    void renderTranslatedPages();

    return () => {
      cancelled = true;
      cancelRenderTasks(translatedRenderTasksRef.current);
    };
  }, [pageCount, scale, translatedPages, translatedPreviewEnabled]);

  function handleViewerScroll() {
    syncActivePageFromScroll(viewerRef, rowRefs, pageCount, setActivePage);
  }

  function handleMouseUp() {
    if (captureMode) {
      return;
    }
    window.setTimeout(() => {
      const nextSelection = window.getSelection();
      const raw = nextSelection?.toString() ?? "";
      setSelection({ raw, cleaned: cleanAcademicSelection(raw) });
    }, 0);
  }

  function handleCaptureStart(pageNumber: number, event: ReactMouseEvent<HTMLDivElement>) {
    if (!captureMode) {
      return;
    }
    const bounds = event.currentTarget.getBoundingClientRect();
    const x = event.clientX - bounds.left;
    const y = event.clientY - bounds.top;
    dragStartRef.current = { pageNumber, x, y };
    setDragRect({ pageNumber, left: x, top: y, width: 0, height: 0 });
  }

  function handleCaptureMove(pageNumber: number, event: ReactMouseEvent<HTMLDivElement>) {
    if (!captureMode || !dragStartRef.current || dragStartRef.current.pageNumber !== pageNumber) {
      return;
    }
    const bounds = event.currentTarget.getBoundingClientRect();
    const currentX = event.clientX - bounds.left;
    const currentY = event.clientY - bounds.top;
    const start = dragStartRef.current;
    setDragRect({
      pageNumber,
      left: Math.min(start.x, currentX),
      top: Math.min(start.y, currentY),
      width: Math.abs(currentX - start.x),
      height: Math.abs(currentY - start.y),
    });
  }

  async function handleCaptureEnd(pageNumber: number) {
    if (!captureMode || !dragRect || dragRect.pageNumber !== pageNumber) {
      dragStartRef.current = null;
      return;
    }

    dragStartRef.current = null;
    if (dragRect.width < 12 || dragRect.height < 12) {
      setDragRect(null);
      return;
    }

    const target = surfaceRefs.current[pageNumber];
    if (!target) {
      setDragRect(null);
      return;
    }

    const fullCanvas = await html2canvas(target, { backgroundColor: null, scale: 2 });
    const cropCanvas = document.createElement("canvas");
    cropCanvas.width = Math.round(dragRect.width * 2);
    cropCanvas.height = Math.round(dragRect.height * 2);
    const cropContext = cropCanvas.getContext("2d");
    if (!cropContext) {
      setDragRect(null);
      return;
    }

    cropContext.drawImage(
      fullCanvas,
      Math.round(dragRect.left * 2),
      Math.round(dragRect.top * 2),
      cropCanvas.width,
      cropCanvas.height,
      0,
      0,
      cropCanvas.width,
      cropCanvas.height,
    );
    const dataUrl = cropCanvas.toDataURL("image/png");
    setSnapshot({ dataUrl, width: cropCanvas.width, height: cropCanvas.height });
    addFigureCapture({
      id: `${Date.now()}-${pageNumber}`,
      itemId: pdfPath ?? "unknown",
      page: pageNumber,
      title: `Page ${pageNumber} Capture`,
      dataUrl,
      width: cropCanvas.width,
      height: cropCanvas.height,
      createdAt: new Date().toISOString(),
    });
    setDragRect(null);
    setCaptureMode(false);
  }

  function handleCaptureToggle() {
    setCaptureMode((current) => {
      const next = !current;
      if (!next) {
        setDragRect(null);
      }
      return next;
    });
  }

  function scrollByViewport(direction: 1 | -1) {
    const viewer = viewerRef.current;
    if (!viewer) {
      return;
    }
    const offset = Math.max(160, Math.round(viewer.clientHeight * 0.85)) * direction;
    viewer.scrollBy({ top: offset, behavior: "smooth" });
  }

  function updateScale(nextScale: number | ((current: number) => number)) {
    setScale((current) => {
      const resolved = typeof nextScale === "function" ? nextScale(current) : nextScale;
      return clampScale(resolved);
    });
  }

  function zoomBy(direction: 1 | -1) {
    updateScale((current) => current + direction * SCALE_STEP);
  }

  function resetZoom() {
    updateScale(DEFAULT_SCALE);
  }

  function handleViewerWheel(event: ReactWheelEvent<HTMLDivElement>) {
    if (!(event.ctrlKey || event.metaKey) || captureMode) {
      return;
    }
    event.preventDefault();
    if (event.deltaY < 0) {
      zoomBy(1);
      return;
    }
    if (event.deltaY > 0) {
      zoomBy(-1);
    }
  }

  function handleViewerKeyDown(event: ReactKeyboardEvent<HTMLDivElement>) {
    if (captureMode) {
      return;
    }
    if (event.defaultPrevented || event.altKey) {
      return;
    }
    const target = event.target;
    if (
      target instanceof HTMLInputElement ||
      target instanceof HTMLTextAreaElement ||
      target instanceof HTMLSelectElement ||
      target instanceof HTMLButtonElement ||
      (target instanceof HTMLElement && target.isContentEditable)
    ) {
      return;
    }
    if (event.ctrlKey || event.metaKey) {
      switch (event.key) {
        case "=":
        case "+":
          event.preventDefault();
          zoomBy(1);
          break;
        case "-":
        case "_":
          event.preventDefault();
          zoomBy(-1);
          break;
        case "0":
          event.preventDefault();
          resetZoom();
          break;
        default:
          break;
      }
      return;
    }
    switch (event.key) {
      case "ArrowDown":
      case "PageDown":
        event.preventDefault();
        scrollByViewport(1);
        break;
      case "ArrowUp":
      case "PageUp":
        event.preventDefault();
        scrollByViewport(-1);
        break;
      case " ":
        event.preventDefault();
        scrollByViewport(event.shiftKey ? -1 : 1);
        break;
      default:
        break;
    }
  }

  if (!pdfPath) {
    return (
      <div className="pdf-reader-empty-state">
        <div className="pdf-empty-card">
          <span className="reader-pill">PDF Workspace</span>
          <h3>选择一篇文献后开始阅读</h3>
          <p>当前支持原文 PDF 阅读、划词翻译、区域截图和保留格式翻译预览。</p>
        </div>
      </div>
    );
  }

  return (
    <div className="pdf-reader">
      <div className="pdf-toolbar">
        <div className="pdf-toolbar-group">
          <span className="reader-pill">PDF</span>
          <span className="reader-pill reader-pill-muted">
            Page {pageCount ? `${activePage} / ${pageCount}` : "-- / --"}
          </span>
          <span className="reader-pill reader-pill-muted">Zoom {Math.round(scale * 100)}%</span>
          <span className="reader-pill reader-pill-muted">
            <Languages size={14} />
            {translatedPreviewEnabled ? "Original + Translation" : "Original"}
          </span>
        </div>
        <div className="pdf-toolbar-group">
          <Button variant="secondary" size="icon-sm" onClick={() => scrollByViewport(-1)} disabled={isLoading || pageCount === 0}>
            <ChevronUp size={16} />
          </Button>
          <Button variant="secondary" size="icon-sm" onClick={() => scrollByViewport(1)} disabled={isLoading || pageCount === 0}>
            <ChevronDown size={16} />
          </Button>
          <Button variant={captureMode ? "default" : "secondary"} size="sm" onClick={handleCaptureToggle} disabled={isLoading}>
            <Camera size={16} />
            {captureMode ? "取消截图" : "区域截图"}
          </Button>
          <Button variant="secondary" size="icon-sm" onClick={() => zoomBy(-1)} disabled={isLoading}>
            <ZoomOut size={16} />
          </Button>
          <Button variant="secondary" size="icon-sm" onClick={() => zoomBy(1)} disabled={isLoading}>
            <ZoomIn size={16} />
          </Button>
        </div>
      </div>

      <div className="pdf-reader-status">
        <div className="pdf-reader-status-copy">
          <strong>
            {captureMode
              ? "截图模式已开启"
              : translatedPreviewEnabled
                ? "左侧显示原文，右侧在 chunk 完成后替换为对应译文页"
                : "当前仅保留原文阅读与划词翻译"}
          </strong>
          <span>
            {captureMode
              ? "在原文页上拖拽一个矩形区域，松开后会保存到图表库。"
              : selection.cleaned
                ? `已选中 ${selection.cleaned.length} 个字符，可以直接翻译、提问或保存笔记。`
                : translatedPreviewEnabled
                  ? translatedStatusText || "保留格式翻译预览按 25 页一块完成，完成的页会直接替换右侧对应页位。"
                  : "在 PDF 上划词后，可以直接打开右侧面板进行翻译或提问。"}
          </span>
        </div>
        <div className="pdf-reader-status-metrics">
          {selection.cleaned ? (
            <Button size="sm" variant="secondary" onClick={onOpenTranslatePanel}>
              翻译划词
            </Button>
          ) : null}
        </div>
      </div>

      {error ? <div className="reader-error">{error}</div> : null}
      {translatedError ? <div className="reader-error">{translatedError}</div> : null}
      {isLoading ? <div className="reader-loading">PDF 正在渲染...</div> : null}

      <div
        ref={viewerRef}
        className={`pdf-canvas-wrap ${captureMode ? "capture-mode" : ""}`}
        onScroll={handleViewerScroll}
        onWheel={handleViewerWheel}
        onMouseUpCapture={handleMouseUp}
        onKeyDown={handleViewerKeyDown}
        tabIndex={0}
      >
        <div className={`pdf-stage ${translatedPreviewEnabled ? "dual-pdf-stage" : ""}`}>
          {Array.from({ length: pageCount }, (_, index) => {
            const pageNumber = index + 1;
            const translatedMapping = translatedPages[pageNumber];
            const translatedState = translatedPageStates[pageNumber] ?? "empty";
            return (
              <div
                key={pageNumber}
                ref={(node) => {
                  rowRefs.current[pageNumber] = node;
                }}
                className={`dual-pdf-row ${translatedPreviewEnabled ? "" : "pdf-single-row"} ${pageNumber === activePage ? "dual-pdf-row-active" : ""}`}
              >
                <div className={`dual-pdf-panel ${translatedPreviewEnabled ? "" : "dual-pdf-panel-single"}`}>
                  <div className="dual-pdf-panel-label">Original</div>
                  <div
                    ref={(node) => {
                      surfaceRefs.current[pageNumber] = node;
                    }}
                    className={`pdf-page-surface ${pageNumber === activePage ? "pdf-page-surface-active" : ""}`}
                    data-page-number={pageNumber}
                    onMouseDown={(event) => handleCaptureStart(pageNumber, event)}
                    onMouseMove={(event) => handleCaptureMove(pageNumber, event)}
                    onMouseUp={() => void handleCaptureEnd(pageNumber)}
                    onMouseLeave={() => void handleCaptureEnd(pageNumber)}
                  >
                    <canvas
                      ref={(node) => {
                        canvasRefs.current[pageNumber] = node;
                      }}
                      className="pdf-canvas"
                    />
                    <div
                      ref={(node) => {
                        textLayerRefs.current[pageNumber] = node;
                      }}
                      className="text-layer"
                      aria-hidden="false"
                    />
                    {dragRect && dragRect.pageNumber === pageNumber ? (
                      <div
                        className="capture-rect"
                        style={{
                          left: dragRect.left,
                          top: dragRect.top,
                          width: dragRect.width,
                          height: dragRect.height,
                        }}
                      />
                    ) : null}
                  </div>
                </div>

                {translatedPreviewEnabled ? (
                  <div className="dual-pdf-panel">
                    <div className="dual-pdf-panel-label">Translation</div>
                    <div
                      ref={(node) => {
                        translatedSurfaceRefs.current[pageNumber] = node;
                      }}
                      className={`pdf-page-surface translated-page-surface ${pageNumber === activePage ? "pdf-page-surface-active" : ""}`}
                      data-page-number={pageNumber}
                    >
                      <canvas
                        ref={(node) => {
                          translatedCanvasRefs.current[pageNumber] = node;
                        }}
                        className="pdf-canvas"
                      />
                      {translatedState !== "ready" ? (
                        <div
                          className={`translated-page-placeholder ${translatedState === "error" ? "translated-page-placeholder-error" : ""}`}
                        >
                          {translatedState === "loading"
                            ? `第 ${pageNumber} 页译文载入中...`
                            : translatedState === "error"
                              ? `第 ${pageNumber} 页译文渲染失败`
                              : translatedMapping
                                ? `第 ${pageNumber} 页所属 chunk 已完成，正在替换...`
                                : translatedStatusText ||
                                  "等待当前页所属的 25 页 chunk 翻译完成后替换这里的译文页。"}
                        </div>
                      ) : null}
                    </div>
                  </div>
                ) : null}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

async function waitForPageRefs(
  pageCount: number,
  rowRefs: MutableRefObject<Record<number, HTMLDivElement | null>>,
) {
  let attempts = 0;
  while (attempts < 20) {
    let ready = true;
    for (let pageNumber = 1; pageNumber <= pageCount; pageNumber += 1) {
      if (!rowRefs.current[pageNumber]) {
        ready = false;
        break;
      }
    }
    if (ready) {
      return;
    }
    await new Promise<void>((resolve) => window.requestAnimationFrame(() => resolve()));
    attempts += 1;
  }
}

function syncActivePageFromScroll(
  viewerRef: MutableRefObject<HTMLDivElement | null>,
  rowRefs: MutableRefObject<Record<number, HTMLDivElement | null>>,
  pageCount: number,
  setActivePage: (page: number) => void,
) {
  const viewer = viewerRef.current;
  if (!viewer || pageCount === 0) {
    return;
  }
  const viewportCenter = viewer.scrollTop + viewer.clientHeight / 2;
  let closestPage = 1;
  let closestDistance = Number.POSITIVE_INFINITY;
  for (let pageNumber = 1; pageNumber <= pageCount; pageNumber += 1) {
    const row = rowRefs.current[pageNumber];
    if (!row) {
      continue;
    }
    const rowCenter = row.offsetTop + row.offsetHeight / 2;
    const distance = Math.abs(rowCenter - viewportCenter);
    if (distance < closestDistance) {
      closestDistance = distance;
      closestPage = pageNumber;
    }
  }
  setActivePage(closestPage);
}

function cleanAcademicSelection(input: string): string {
  return input.replace(/\u00ad/g, "").replace(/-\n/g, "").replace(/\s+/g, " ").trim();
}

function clampScale(value: number): number {
  return Math.min(MAX_SCALE, Math.max(MIN_SCALE, Number(value.toFixed(1))));
}

function setTranslatedPageState(
  setTranslatedPageStates: Dispatch<SetStateAction<Record<number, TranslatedPageState>>>,
  pageNumber: number,
  state: TranslatedPageState,
) {
  setTranslatedPageStates((current) =>
    current[pageNumber] === state ? current : { ...current, [pageNumber]: state },
  );
}

async function getCachedTranslatedDocument(
  translatedDocsRef: MutableRefObject<Map<string, LoadedPDFDocument>>,
  translatedDocPromisesRef: MutableRefObject<Map<string, Promise<LoadedPDFDocument>>>,
  pdfPath: string,
) {
  const cached = translatedDocsRef.current.get(pdfPath);
  if (cached) {
    return cached;
  }

  const pending = translatedDocPromisesRef.current.get(pdfPath);
  if (pending) {
    return pending;
  }

  const nextPending = openPDFDocument(pdfPath)
    .then((loaded) => {
      translatedDocsRef.current.set(pdfPath, loaded);
      return loaded;
    })
    .catch((error) => {
      translatedDocPromisesRef.current.delete(pdfPath);
      throw error;
    });
  translatedDocPromisesRef.current.set(pdfPath, nextPending);
  return nextPending;
}

async function destroyLoadedDocuments(documents: Map<string, LoadedPDFDocument>) {
  await Promise.all(
    Array.from(documents.values()).map((loaded) => destroyPDFDocument(loaded.loadingTask)),
  );
}

function clearRenderedSurface(
  surface: HTMLDivElement | null | undefined,
  canvas: HTMLCanvasElement | null | undefined,
) {
  if (surface) {
    surface.style.width = "";
    surface.style.height = "";
  }
  if (!canvas) {
    return;
  }
  const context = canvas.getContext("2d");
  if (context) {
    context.clearRect(0, 0, canvas.width, canvas.height);
  }
  canvas.width = 0;
  canvas.height = 0;
  canvas.style.width = "";
  canvas.style.height = "";
}

async function renderPageToCanvas(
  pageNumber: number,
  page: pdfjsLib.PDFPageProxy,
  surface: HTMLDivElement,
  canvas: HTMLCanvasElement,
  scale: number,
  renderTasksRef: MutableRefObject<Map<number, PDFRenderTask>>,
) {
  const viewport = page.getViewport({ scale });
  const outputScale = new pdfjsLib.OutputScale();
  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Canvas 2D context unavailable");
  }

  surface.style.setProperty("--scale-factor", `${viewport.scale}`);
  surface.style.setProperty("--user-unit", `${page.userUnit}`);
  surface.style.width = `${viewport.width}px`;
  surface.style.height = `${viewport.height}px`;

  canvas.width = Math.max(1, Math.floor(viewport.width * outputScale.sx));
  canvas.height = Math.max(1, Math.floor(viewport.height * outputScale.sy));
  canvas.style.width = `${viewport.width}px`;
  canvas.style.height = `${viewport.height}px`;

  context.setTransform(1, 0, 0, 1, 0, 0);
  const renderTask = page.render({
    canvas,
    canvasContext: context,
    viewport,
    transform: outputScale.scaled
      ? [outputScale.sx, 0, 0, outputScale.sy, 0, 0]
      : undefined,
  });
  renderTasksRef.current.set(pageNumber, renderTask);

  try {
    await renderTask.promise;
  } finally {
    if (renderTasksRef.current.get(pageNumber) === renderTask) {
      renderTasksRef.current.delete(pageNumber);
    }
  }
}

function cancelRenderTasks(tasks: Map<number, PDFRenderTask>) {
  tasks.forEach((task) => {
    try {
      task.cancel();
    } catch {
      // Ignore render cancellation races during rapid zoom or chunk swaps.
    }
  });
  tasks.clear();
}
