import { useEffect, useMemo, useRef, useState } from "react";
import {
  Copy,
  Download,
  History,
  RefreshCw,
  Settings2,
  Sparkles,
  Trash2,
} from "lucide-react";
import { EventsOff, EventsOn } from "../../wailsjs/runtime/runtime";
import { configApi } from "../api/config";
import { gatewayApi } from "../api/gateway";
import { historyApi } from "../api/history";
import {
  loadPDFTextChunks,
  loadPDFTextContext,
  type PDFTextChunk,
} from "../lib/pdfContext";
import {
  buildPaperImageGenerationPrompt,
  buildPaperImageSummaryInstruction,
} from "../lib/paperFigurePrompt";
import { useReaderStore } from "../store/readerStore";
import type {
  AIWorkspaceConfig,
  ModelRecord,
  ProviderConfig,
} from "../types/config";
import { DEFAULT_AI_WORKSPACE_CONFIG } from "../types/config";
import type { GatewayStreamEvent } from "../types/gateway";
import type { ChatHistoryEntry } from "../types/history";
import type { TabItem } from "../store/tabStore";
import { MarkdownPreview } from "./MarkdownPreview";
import { Button } from "./ui/Button";

type PromptScope = "document" | "selection" | "conversation";

interface AIResult {
  id: string;
  title: string;
  kind: string;
  prompt: string;
  content: string;
  createdAt: string;
  status: "streaming" | "done" | "error";
  meta?: string;
  source: "live" | "history";
}

interface ConversationPair {
  prompt: string;
  response: string;
}

interface ReaderAIPanelProps {
  tab: TabItem;
  llmConfigs: ProviderConfig[];
  drawingConfigs: ProviderConfig[];
  activeLLMConfig: ProviderConfig | null;
  activeLLMModel: ModelRecord | null;
  llmProviderId: number | null;
  llmModelId: number | null;
  setLlmProviderId: (value: number | null) => void;
  setLlmModelId: (value: number | null) => void;
}

interface PresetCard {
  id: "summary" | "table" | "methods" | "selection";
  label: string;
  description: string;
  requiresSelection?: boolean;
}

const PRESET_CARDS: PresetCard[] = [
  {
    id: "summary",
    label: "结构化总结",
    description: "支持单轮或多轮分块阅读，再综合成最终总结。",
  },
  {
    id: "table",
    label: "表格摘要",
    description: "按你自定义的模板输出 Markdown 表格。",
  },
  {
    id: "methods",
    label: "方法拆解",
    description: "把方法流程拆成可复述、可实现的说明。",
  },
  {
    id: "selection",
    label: "解释选区",
    description: "针对当前划词做精读解释。",
    requiresSelection: true,
  },
];

const MULTI_ROUND_CHUNK_PROMPT = `请阅读下面给出的论文连续片段，生成这一段的阅读纪要。

要求：
1. 使用简体中文和 Markdown。
2. 只基于这段内容总结，不要脑补缺失部分。
3. 输出以下结构：
## 这一段在讲什么
## 方法或实验细节
## 关键信号
## 暂时还不能确定什么
4. 如有公式，保留 LaTeX 写法。`;

const MULTI_ROUND_FINAL_PROMPT = `下面是按顺序得到的论文分块纪要。请你把它们综合成一份适合侧边栏阅读的最终总结。

要求：
1. 使用简体中文和 Markdown。
2. 输出结构必须包含：
## 一句话结论
## 研究问题
## 核心方法
## 实验与结果
## 创新点
## 局限性
## 下一步阅读建议
3. 如果分块纪要里存在信息边界或互相矛盾的地方，请在最后增加“## 信息边界”。
4. 不要重复逐块复述，要做真正的综合。`;

export function ReaderAIPanel({
  tab,
  llmConfigs,
  drawingConfigs,
  activeLLMConfig,
  activeLLMModel,
  llmProviderId,
  llmModelId,
  setLlmProviderId,
  setLlmModelId,
}: ReaderAIPanelProps) {
  const selection = useReaderStore((state) => state.selection);
  const snapshot = useReaderStore((state) => state.snapshot);
  const activePage = useReaderStore((state) => state.activePage);
  const figureCaptures = useReaderStore((state) => state.figureCaptures);
  const setSnapshot = useReaderStore((state) => state.setSnapshot);

  const [workspaceConfig, setWorkspaceConfig] = useState<AIWorkspaceConfig>(
    DEFAULT_AI_WORKSPACE_CONFIG,
  );
  const [configLoaded, setConfigLoaded] = useState(false);
  const [panelError, setPanelError] = useState<string | null>(null);
  const [results, setResults] = useState<AIResult[]>([]);
  const [chatHistory, setChatHistory] = useState<ChatHistoryEntry[]>([]);
  const [deletingHistoryIDs, setDeletingHistoryIDs] = useState<number[]>([]);
  const [conversationPairs, setConversationPairs] = useState<
    ConversationPair[]
  >([]);
  const [isRunning, setIsRunning] = useState(false);
  const [isPreparingContext, setIsPreparingContext] = useState(false);
  const [generatedFigure, setGeneratedFigure] = useState("");
  const [drawingError, setDrawingError] = useState<string | null>(null);
  const [isGeneratingFigure, setIsGeneratingFigure] = useState(false);
  const activeEventNameRef = useRef<string | null>(null);
  const hydratedHistoryKeyRef = useRef("");

  const currentFigureCapture = useMemo(
    () =>
      figureCaptures.find(
        (capture) =>
          capture.itemId === (tab.pdfPath ?? "") && capture.page === activePage,
      ) ??
      figureCaptures.find(
        (capture) => capture.itemId === (tab.pdfPath ?? ""),
      ) ??
      null,
    [activePage, figureCaptures, tab.pdfPath],
  );

  const availableHistory = useMemo(
    () =>
      chatHistory.filter((entry) => entry.kind !== "translation").slice(0, 12),
    [chatHistory],
  );
  const drawingProviderConfig = useMemo(
    () =>
      drawingConfigs.find(
        (item) => item.provider.id === workspaceConfig.drawingProviderId,
      ) ?? null,
    [drawingConfigs, workspaceConfig.drawingProviderId],
  );
  const drawingProviderName = drawingProviderConfig?.provider.name ?? "";

  const workspaceID = tab.workspaceId ?? "";

  useEffect(() => {
    let cancelled = false;

    if (!workspaceID) {
      setWorkspaceConfig(DEFAULT_AI_WORKSPACE_CONFIG);
      setConfigLoaded(true);
      return () => {
        cancelled = true;
      };
    }

    setConfigLoaded(false);
    void configApi
      .getAIWorkspaceConfig(workspaceID)
      .then((config) => {
        if (!cancelled) {
          setWorkspaceConfig(config);
          setConfigLoaded(true);
        }
      })
      .catch((error) => {
        if (!cancelled) {
          setPanelError(
            error instanceof Error ? error.message : "加载 AI 配置失败",
          );
          setConfigLoaded(true);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [workspaceID]);

  useEffect(() => {
    if (!configLoaded || !workspaceID) {
      return undefined;
    }

    const timer = window.setTimeout(() => {
      void configApi.saveAIWorkspaceConfig(workspaceID, workspaceConfig).catch((error) => {
        setPanelError(
          error instanceof Error ? error.message : "保存 AI 配置失败",
        );
      });
    }, 450);

    return () => {
      window.clearTimeout(timer);
    };
  }, [configLoaded, workspaceConfig, workspaceID]);

  useEffect(() => {
    if (!drawingConfigs.length) {
      return;
    }

    if (
      drawingConfigs.some(
        (item) => item.provider.id === workspaceConfig.drawingProviderId,
      )
    ) {
      return;
    }

    updateWorkspaceConfig({
      drawingProviderId: drawingConfigs[0]?.provider.id ?? 0,
    });
  }, [drawingConfigs, workspaceConfig.drawingProviderId]);

  useEffect(() => {
    let cancelled = false;
    hydratedHistoryKeyRef.current = "";
    setPanelError(null);
    setResults([]);
    setConversationPairs([]);
    setGeneratedFigure("");

    if (!tab.id) {
      setChatHistory([]);
      return () => {
        cancelled = true;
      };
    }

    void historyApi
      .listChatHistory(tab.workspaceId ?? "", tab.documentId ?? "", tab.id)
      .then((entries) => {
        if (!cancelled) {
          setChatHistory(entries);
        }
      })
      .catch((error) => {
        if (!cancelled) {
          setPanelError(
            error instanceof Error ? error.message : "加载 AI 历史失败",
          );
        }
      });

    return () => {
      cancelled = true;
    };
  }, [tab.id]);

  useEffect(() => {
    if (!tab.id || !chatHistory.length) {
      return;
    }

    const hydrationKey = `${tab.workspaceId ?? ""}:${tab.documentId ?? tab.id}:${workspaceConfig.autoRestoreCount}`;
    if (hydratedHistoryKeyRef.current === hydrationKey) {
      return;
    }

    const restoredEntries = chatHistory
      .filter(
        (entry) =>
          entry.kind !== "translation" && entry.kind !== "summary_chunk",
      )
      .slice(0, workspaceConfig.autoRestoreCount);

    const restoredResults = restoredEntries.map(historyEntryToResult);
    const restoredPairs = restoredEntries
      .slice()
      .reverse()
      .map((entry) => ({
        prompt: entry.prompt,
        response: entry.response,
      }));

    setResults((current) =>
      current.some((entry) => entry.source === "live")
        ? current
        : restoredResults,
    );
    setConversationPairs((current) =>
      current.length ? current : restoredPairs,
    );
    hydratedHistoryKeyRef.current = hydrationKey;
  }, [chatHistory, tab.workspaceId, tab.documentId, tab.id, workspaceConfig.autoRestoreCount]);

  useEffect(
    () => () => {
      if (activeEventNameRef.current) {
        EventsOff(activeEventNameRef.current);
      }
    },
    [],
  );

  const providerHint = activeLLMModel
    ? `${activeLLMConfig?.provider.name ?? "LLM"} / ${activeLLMModel.modelId}`
    : "请选择可用的 LLM Provider 和 Model";

  function updateWorkspaceConfig(patch: Partial<AIWorkspaceConfig>) {
    setWorkspaceConfig((current) => ({ ...current, ...patch }));
  }

  function prependResult(result: AIResult) {
    setResults((current) => [
      result,
      ...current.filter((entry) => entry.id !== result.id),
    ]);
  }

  function patchResult(resultID: string, patch: Partial<AIResult>) {
    setResults((current) =>
      current.map((entry) =>
        entry.id === resultID ? { ...entry, ...patch } : entry,
      ),
    );
  }

  function finalizeResult(
    resultID: string,
    patch: Partial<AIResult>,
  ): AIResult | null {
    let finalResult: AIResult | null = null;
    setResults((current) =>
      current.map((entry) => {
        if (entry.id !== resultID) {
          return entry;
        }
        finalResult = { ...entry, ...patch };
        return finalResult;
      }),
    );
    return finalResult;
  }

  async function handlePresetRun(preset: PresetCard) {
    if (preset.requiresSelection && !selection.cleaned.trim()) {
      setPanelError("请先在 PDF 中选择一段文本。");
      return;
    }

    switch (preset.id) {
      case "summary":
        await runSummaryWorkflow();
        return;
      case "table":
        await runPrompt({
          title: "表格摘要",
          kind: "table",
          displayPrompt: "表格摘要",
          userInstruction: buildTableInstruction(workspaceConfig),
          scope: "document",
        });
        return;
      case "methods":
        await runPrompt({
          title: "方法拆解",
          kind: "analysis",
          displayPrompt: "方法拆解",
          userInstruction: buildMethodsInstruction(),
          scope: "document",
        });
        return;
      case "selection":
        await runPrompt({
          title: "解释选区",
          kind: "chat",
          displayPrompt: "解释当前选区",
          userInstruction: buildSelectionInstruction(),
          scope: "selection",
        });
        return;
      default:
        return;
    }
  }

  async function runSummaryWorkflow() {
    if (!tab.pdfPath || workspaceConfig.summaryMode === "single") {
      await runPrompt({
        title: "结构化总结",
        kind: "summary",
        displayPrompt: "结构化总结",
        userInstruction: buildSingleSummaryInstruction(),
        scope: "document",
      });
      return;
    }

    setIsPreparingContext(true);
    try {
      const chunks = await loadPDFTextChunks(
        tab.pdfPath,
        workspaceConfig.summaryChunkPages,
        workspaceConfig.summaryChunkMaxChars,
      );
      if (workspaceConfig.summaryMode === "auto" && chunks.length <= 1) {
        await runPrompt({
          title: "结构化总结",
          kind: "summary",
          displayPrompt: "结构化总结",
          userInstruction: buildSingleSummaryInstruction(),
          scope: "document",
        });
        return;
      }

      await runMultiRoundSummary(chunks);
    } catch (error) {
      setPanelError(
        error instanceof Error ? error.message : "准备多轮总结失败",
      );
    } finally {
      setIsPreparingContext(false);
    }
  }

  async function runCustomPrompt() {
    const question = workspaceConfig.customPromptDraft.trim();
    if (!question) {
      return;
    }

    await runPrompt({
      title: "自定义提问",
      kind: "chat",
      displayPrompt: question,
      userInstruction: `请基于当前论文内容回答我的问题。请使用简体中文和 Markdown 输出。\n\n问题：${question}`,
      scope: tab.pdfPath ? "document" : "selection",
    });
  }

  async function runFollowUpPrompt() {
    const question = workspaceConfig.followUpPromptDraft.trim();
    if (!question) {
      return;
    }

    if (!conversationPairs.length) {
      setPanelError("先生成一次总结、表格或自定义回答，再继续追问。");
      return;
    }

    const transcript = conversationPairs
      .slice(-3)
      .map(
        (pair, index) =>
          `第 ${index + 1} 轮问题：${pair.prompt}\n第 ${index + 1} 轮回答：\n${pair.response}`,
      )
      .join("\n\n");

    await runPrompt({
      title: "继续追问",
      kind: "followup",
      displayPrompt: question,
      userInstruction: `下面是之前围绕同一篇论文的对话，请基于这些上下文继续回答，不要忽略之前已经提炼出的结论。

历史对话：
${transcript}

新的追问：
${question}

要求：
1. 使用简体中文和 Markdown。
2. 如果需要修正之前的结论，明确指出修正点。
3. 优先结合当前页、当前选区和截图上下文。`,
      scope: "conversation",
    });
  }

  async function runPrompt({
    title,
    kind,
    displayPrompt,
    userInstruction,
    scope,
  }: {
    title: string;
    kind: string;
    displayPrompt: string;
    userInstruction: string;
    scope: PromptScope;
  }) {
    if (!llmProviderId || !llmModelId) {
      setPanelError("请先选择 LLM Provider 和 Model。");
      return;
    }

    setPanelError(null);
    setIsRunning(true);
    const resultID = createResultId();
    const resultMeta: string[] = [];

    prependResult({
      id: resultID,
      title,
      kind,
      prompt: displayPrompt,
      content: "",
      createdAt: new Date().toISOString(),
      status: "streaming",
      source: "live",
    });

    try {
      const requestPrompt = await buildPromptForScope(
        scope,
        userInstruction,
        resultMeta,
      );
      patchResult(resultID, { meta: resultMeta.join(" · ") });

      let response = "";
      response = await streamLLMText(requestPrompt, (chunk) => {
        response += chunk;
        patchResult(resultID, { content: response });
      });

      const finalResult = finalizeResult(resultID, {
        status: "done",
        content: response,
        meta: resultMeta.join(" · "),
      });
      if (!finalResult) {
        return;
      }

      await saveResultToHistory(finalResult, displayPrompt);
      setConversationPairs((current) => [
        ...current,
        { prompt: displayPrompt, response },
      ]);
    } catch (error) {
      const message = error instanceof Error ? error.message : "AI 请求失败";
      setPanelError(message);
      finalizeResult(resultID, {
        status: "error",
        content: message,
      });
    } finally {
      setIsRunning(false);
      setIsPreparingContext(false);
    }
  }

  async function runMultiRoundSummary(chunks: PDFTextChunk[]) {
    if (!llmProviderId || !llmModelId) {
      setPanelError("请先选择 LLM Provider 和 Model。");
      return;
    }

    setPanelError(null);
    setIsRunning(true);
    const resultID = createResultId();

    prependResult({
      id: resultID,
      title: "多轮分块总结",
      kind: "summary_multi",
      prompt: "多轮分块总结",
      content: renderMultiRoundProgress([], null, "", chunks.length),
      createdAt: new Date().toISOString(),
      status: "streaming",
      meta: `多轮总结 ${chunks.length} 段 · 每段最多 ${workspaceConfig.summaryChunkPages} 页 / ${workspaceConfig.summaryChunkMaxChars.toLocaleString()} 字符`,
      source: "live",
    });

    try {
      const chunkSummaries: Array<{ chunk: PDFTextChunk; summary: string }> =
        [];

      for (const chunk of chunks) {
        let liveChunk = "";
        patchResult(resultID, {
          content: renderMultiRoundProgress(
            chunkSummaries,
            { chunk, content: liveChunk },
            "",
            chunks.length,
          ),
        });

        const prompt = `${MULTI_ROUND_CHUNK_PROMPT}

当前片段页码：P${chunk.startPage}-${chunk.endPage}

论文片段：
${chunk.text}`;

        const summary = await streamLLMText(prompt, (piece) => {
          liveChunk += piece;
          patchResult(resultID, {
            content: renderMultiRoundProgress(
              chunkSummaries,
              { chunk, content: liveChunk },
              "",
              chunks.length,
            ),
          });
        });

        chunkSummaries.push({ chunk, summary });
        patchResult(resultID, {
          content: renderMultiRoundProgress(
            chunkSummaries,
            null,
            "",
            chunks.length,
          ),
        });
      }

      let finalDraft = "";
      const finalPrompt = `${MULTI_ROUND_FINAL_PROMPT}

以下是分块纪要：
${chunkSummaries
  .map(
    (
      item,
      index,
    ) => `### Chunk ${index + 1} · P${item.chunk.startPage}-${item.chunk.endPage}
${item.summary}`,
  )
  .join("\n\n")}`;

      const finalResponse = await streamLLMText(finalPrompt, (piece) => {
        finalDraft += piece;
        patchResult(resultID, {
          content: renderMultiRoundProgress(
            chunkSummaries,
            null,
            finalDraft,
            chunks.length,
          ),
        });
      });

      const finalContent = `${finalResponse}

---
## 分块阅读轨迹
${chunkSummaries
  .map(
    (
      item,
      index,
    ) => `### Chunk ${index + 1} · P${item.chunk.startPage}-${item.chunk.endPage}
${item.summary}`,
  )
  .join("\n\n")}`;

      const finalResult = finalizeResult(resultID, {
        status: "done",
        content: finalContent,
      });
      if (!finalResult) {
        return;
      }

      await saveResultToHistory(finalResult, "多轮分块总结");
      setConversationPairs((current) => [
        ...current,
        { prompt: "多轮分块总结", response: finalContent },
      ]);
    } catch (error) {
      const message = error instanceof Error ? error.message : "多轮总结失败";
      setPanelError(message);
      finalizeResult(resultID, {
        status: "error",
        content: message,
      });
    } finally {
      setIsRunning(false);
      setIsPreparingContext(false);
    }
  }

  async function buildPromptForScope(
    scope: PromptScope,
    userInstruction: string,
    resultMeta: string[],
  ) {
    const promptParts = [
      "你是 OpenSciReader 的学术阅读助手。请使用简体中文回答，并尽量保持结构清晰。",
    ];

    if (scope === "document" && tab.pdfPath) {
      setIsPreparingContext(true);
      try {
        const contextWindow = activeLLMModel?.contextWindow ?? 0;
        const context = await loadPDFTextContext(
          tab.pdfPath,
          estimateTextBudget(contextWindow),
        );
        const pageSummary = context.truncated
          ? `${context.sourceLabel} · 已注入 ${context.includedPages.length}/${context.totalPages} 段正文，约 ${context.text.length.toLocaleString()} 字符（已截断）`
          : `${context.sourceLabel} · 已注入全文 ${context.totalPages} 段，约 ${context.totalCharacters.toLocaleString()} 字符`;

        promptParts.push(
          context.source === "markitdown"
            ? `下面是从 PDF 中提取并结构化整理的论文 Markdown 内容，请优先基于其章节结构、列表和表格语义回答：\n\n${context.text}`
            : `下面是从 PDF 中按页提取的论文正文，请优先基于这部分信息回答：\n\n${context.text}`,
        );
        resultMeta.push(pageSummary);
      } catch (error) {
        resultMeta.push("全文提取失败，已退回当前页上下文");
        promptParts.push(
          "未能成功提取全文，请优先结合当前页、当前选区和截图上下文回答。",
        );
        if (error instanceof Error) {
          setPanelError(error.message);
        }
      } finally {
        setIsPreparingContext(false);
      }
    }

    if (scope === "selection" && selection.cleaned.trim()) {
      promptParts.push(`当前选中的论文片段如下：\n${selection.cleaned.trim()}`);
      resultMeta.push(`基于当前选区 ${selection.cleaned.trim().length} 字`);
    }

    if (scope === "conversation") {
      resultMeta.push(
        `延续最近 ${Math.min(conversationPairs.length, 3)} 轮对话`,
      );
    }

    promptParts.push(userInstruction);
    return promptParts.join("\n\n");
  }

  async function streamLLMText(
    prompt: string,
    onChunk: (chunk: string) => void,
  ): Promise<string> {
    if (!llmProviderId || !llmModelId) {
      throw new Error("请先选择 LLM Provider 和 Model。");
    }

    const requestID = await gatewayApi.streamLLMChat(
      llmProviderId,
      llmModelId,
      prompt,
      {
        selection: selection.cleaned,
        snapshot: snapshot?.dataUrl ?? "",
        page: activePage,
        itemTitle: tab.title ?? "",
        citeKey: tab.citeKey ?? "",
      },
    );

    const eventName = `gateway:chat:${requestID}`;
    activeEventNameRef.current = eventName;
    EventsOff(eventName);

    return new Promise<string>((resolve, reject) => {
      let response = "";

      EventsOn(eventName, (payload: GatewayStreamEvent) => {
        if (payload.type === "chunk") {
          const piece = payload.content ?? "";
          response += piece;
          onChunk(piece);
          return;
        }

        if (payload.type === "error") {
          EventsOff(eventName);
          if (activeEventNameRef.current === eventName) {
            activeEventNameRef.current = null;
          }
          reject(new Error(payload.error ?? "未知网关错误"));
          return;
        }

        if (payload.type === "done") {
          EventsOff(eventName);
          if (activeEventNameRef.current === eventName) {
            activeEventNameRef.current = null;
          }
          resolve(response);
        }
      });
    });
  }

  async function generateVisualSummaryForFigure(): Promise<string> {
    const scope: PromptScope = tab.pdfPath
      ? "document"
      : selection.cleaned.trim()
        ? "selection"
        : "document";

    const prompt = await buildPromptForScope(
      scope,
      `${buildPaperImageSummaryInstruction(
        "中文",
      )}\n\n请同时参考当前页截图、图表区域截图和当前选区（如果有），输出可直接用于生图的视觉摘要。`,
      [],
    );

    return streamLLMText(prompt, () => {});
  }

  async function handleGenerateFigure() {
    if (
      !llmProviderId ||
      !llmModelId ||
      !workspaceConfig.drawingProviderId ||
      !workspaceConfig.drawingModel.trim()
    ) {
      setDrawingError(
        "请先选择当前阅读 LLM，以及独立的 Drawing Provider / Model。",
      );
      return;
    }

    setIsGeneratingFigure(true);
    setDrawingError(null);

    try {
      const visualSummary = await generateVisualSummaryForFigure();
      const result = await gatewayApi.generateResearchFigure(
        0,
        0,
        buildPaperImageGenerationPrompt(
          visualSummary,
          tab.title ?? "",
          "中文",
          workspaceConfig.drawingPromptDraft,
        ),
        {
          selection: selection.cleaned,
          snapshot: snapshot?.dataUrl ?? currentFigureCapture?.dataUrl ?? "",
          page: activePage,
          itemTitle: tab.title ?? "",
          citeKey: tab.citeKey ?? "",
        },
        workspaceID,
      );

      setGeneratedFigure(result.dataUrl);
      setSnapshot({ dataUrl: result.dataUrl, width: 1024, height: 1024 });
    } catch (error) {
      setDrawingError(
        error instanceof Error ? error.message : "论文总结绘图失败",
      );
    } finally {
      setIsGeneratingFigure(false);
    }
  }

  async function saveResultToHistory(result: AIResult, prompt: string) {
    if (!tab.id) {
      return;
    }

    const entry = await historyApi.saveChatHistory({
      workspaceId: tab.workspaceId ?? "",
      documentId: tab.documentId ?? "",
      itemId: tab.id,
      itemTitle: tab.title,
      page: activePage,
      kind: result.kind,
      prompt,
      response: result.content,
    });
    setChatHistory((current) => [entry, ...current]);
  }

  function loadHistoryEntry(entry: ChatHistoryEntry) {
    const hydratedResult = historyEntryToResult(entry);
    setResults((current) => [
      hydratedResult,
      ...current.filter((item) => item.id !== hydratedResult.id),
    ]);
    setConversationPairs([{ prompt: entry.prompt, response: entry.response }]);
  }

  async function handleDeleteHistoryEntry(entry: ChatHistoryEntry) {
    const shouldDelete = window.confirm("确定删除这条 AI 历史记录吗？");
    if (!shouldDelete) {
      return;
    }

    setPanelError(null);
    setDeletingHistoryIDs((current) => [...current, entry.id]);
    try {
      await historyApi.deleteChatHistory(entry.id);
      setChatHistory((current) => current.filter((item) => item.id !== entry.id));
      setResults((current) =>
        current.filter((item) => item.id !== `history-${entry.id}`),
      );
      setConversationPairs((current) => {
        if (
          current.length === 1 &&
          current[0]?.prompt === entry.prompt &&
          current[0]?.response === entry.response
        ) {
          return [];
        }
        return current;
      });
    } catch (error) {
      setPanelError(error instanceof Error ? error.message : "删除历史失败");
    } finally {
      setDeletingHistoryIDs((current) =>
        current.filter((historyID) => historyID !== entry.id),
      );
    }
  }

  function resetWorkspaceConfig() {
    setWorkspaceConfig(DEFAULT_AI_WORKSPACE_CONFIG);
  }

  return (
    <div className="ai-sidebar">
      <div className="context-card ai-context-card">
        <div className="ai-context-header">
          <strong>AI 阅读工作区</strong>
          <span className="badge badge-accent">P{activePage}</span>
        </div>
        <p>{providerHint}</p>
        <div className="ai-context-chips">
          <span className="ai-chip">
            {selection.cleaned
              ? `已划词 ${selection.cleaned.length} 字`
              : "未划词"}
          </span>
          <span className="ai-chip">
            {snapshot?.dataUrl ? "已捕获截图" : "无截图上下文"}
          </span>
          <span className="ai-chip">
            {tab.pdfPath ? "支持全文总结" : "仅当前页上下文"}
          </span>
        </div>
      </div>

      <div className="field-grid">
        <label className="field">
          <span>LLM Provider</span>
          <select
            value={llmProviderId ?? ""}
            onChange={(event) =>
              setLlmProviderId(Number(event.target.value) || null)
            }
          >
            <option value="">请选择</option>
            {llmConfigs.map((item) => (
              <option key={item.provider.id} value={item.provider.id}>
                {item.provider.name}
              </option>
            ))}
          </select>
        </label>
        <label className="field">
          <span>LLM Model</span>
          <select
            value={llmModelId ?? ""}
            onChange={(event) =>
              setLlmModelId(Number(event.target.value) || null)
            }
          >
            <option value="">请选择</option>
            {(activeLLMConfig?.models ?? []).map((model) => (
              <option key={model.id} value={model.id}>
                {model.modelId}
              </option>
            ))}
          </select>
        </label>
      </div>

      <details className="ai-settings-panel">
        <summary>
          <Settings2 size={14} /> 总结与表格配置
        </summary>
        <div className="ai-settings-body">
          <div className="field-grid">
            <label className="field">
              <span>总结模式</span>
              <select
                value={workspaceConfig.summaryMode}
                onChange={(event) =>
                  updateWorkspaceConfig({
                    summaryMode: event.target
                      .value as AIWorkspaceConfig["summaryMode"],
                  })
                }
              >
                <option value="auto">自动</option>
                <option value="single">单轮</option>
                <option value="multi">多轮分块</option>
              </select>
            </label>
            <label className="field">
              <span>自动恢复结果数</span>
              <input
                type="number"
                min={1}
                max={12}
                value={workspaceConfig.autoRestoreCount}
                onChange={(event) =>
                  updateWorkspaceConfig({
                    autoRestoreCount: Number(event.target.value) || 1,
                  })
                }
              />
            </label>
          </div>

          <div className="field-grid">
            <label className="field">
              <span>每轮页数</span>
              <input
                type="number"
                min={1}
                max={30}
                value={workspaceConfig.summaryChunkPages}
                onChange={(event) =>
                  updateWorkspaceConfig({
                    summaryChunkPages: Number(event.target.value) || 1,
                  })
                }
              />
            </label>
            <label className="field">
              <span>每轮最大字符</span>
              <input
                type="number"
                min={4000}
                max={120000}
                step={1000}
                value={workspaceConfig.summaryChunkMaxChars}
                onChange={(event) =>
                  updateWorkspaceConfig({
                    summaryChunkMaxChars: Number(event.target.value) || 4000,
                  })
                }
              />
            </label>
          </div>

          <label className="field">
            <span>表格模板</span>
            <textarea
              className="prompt-input ai-template-input"
              value={workspaceConfig.tableTemplate}
              onChange={(event) =>
                updateWorkspaceConfig({ tableTemplate: event.target.value })
              }
            />
          </label>

          <label className="field">
            <span>填表提示词</span>
            <textarea
              className="prompt-input"
              value={workspaceConfig.tablePrompt}
              onChange={(event) =>
                updateWorkspaceConfig({ tablePrompt: event.target.value })
              }
            />
          </label>

          <div className="prompt-actions">
            <Button
              variant="secondary"
              size="sm"
              onClick={resetWorkspaceConfig}
            >
              恢复默认
            </Button>
            <span className="field-hint">
              这些配置会自动保存，下次打开仍会保留。
            </span>
          </div>
        </div>
      </details>

      <div className="ai-section">
        <div className="section-header">
          <strong>快捷任务</strong>
          <span className="badge badge-count">{PRESET_CARDS.length}</span>
        </div>
        <div className="ai-preset-grid">
          {PRESET_CARDS.map((preset) => (
            <button
              key={preset.id}
              type="button"
              className="ai-preset-card"
              disabled={
                isRunning ||
                isPreparingContext ||
                (preset.requiresSelection && !selection.cleaned.trim())
              }
              onClick={() => void handlePresetRun(preset)}
            >
              <span>{preset.label}</span>
              <small>{preset.description}</small>
            </button>
          ))}
        </div>
      </div>

      <div className="ai-section">
        <div className="section-header">
          <strong>自定义提问</strong>
          <span className="badge">
            {tab.pdfPath ? "默认携带全文" : "当前页上下文"}
          </span>
        </div>
        <label className="field">
          <span>Prompt</span>
          <textarea
            className="prompt-input"
            placeholder="例如：请比较作者的方法与常见 baseline 的差异，并指出最值得复现的部分。"
            value={workspaceConfig.customPromptDraft}
            onChange={(event) =>
              updateWorkspaceConfig({ customPromptDraft: event.target.value })
            }
          />
        </label>
        <div className="prompt-actions">
          <Button
            onClick={() => void runCustomPrompt()}
            disabled={
              isRunning ||
              isPreparingContext ||
              !workspaceConfig.customPromptDraft.trim()
            }
          >
            {isPreparingContext
              ? "提取全文中..."
              : isRunning
                ? "生成中..."
                : "发送到 AI"}
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() =>
              updateWorkspaceConfig({ customPromptDraft: selection.cleaned })
            }
            disabled={!selection.cleaned}
          >
            用当前划词提问
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() =>
              updateWorkspaceConfig({
                customPromptDraft:
                  "请把这篇论文最值得在组会上讲的 5 个点列出来。",
              })
            }
          >
            填入组会模板
          </Button>
        </div>
      </div>

      <div className="ai-section">
        <div className="section-header">
          <strong>继续追问</strong>
          <span className="badge">
            {conversationPairs.length
              ? `最近 ${Math.min(conversationPairs.length, 3)} 轮`
              : "暂无上下文"}
          </span>
        </div>
        <label className="field">
          <span>Follow-up</span>
          <textarea
            className="prompt-input"
            placeholder="例如：这个方法最可能失败在哪种数据分布上？"
            value={workspaceConfig.followUpPromptDraft}
            onChange={(event) =>
              updateWorkspaceConfig({ followUpPromptDraft: event.target.value })
            }
          />
        </label>
        <div className="prompt-actions">
          <Button
            variant="secondary"
            onClick={() => void runFollowUpPrompt()}
            disabled={
              isRunning ||
              !conversationPairs.length ||
              !workspaceConfig.followUpPromptDraft.trim()
            }
          >
            继续追问
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setConversationPairs([])}
            disabled={!conversationPairs.length}
          >
            清空追问上下文
          </Button>
        </div>
      </div>

      <div className="ai-section">
        <div className="section-header">
          <strong>科研绘图</strong>
          <span className="badge">
            {snapshot?.dataUrl || currentFigureCapture?.dataUrl
              ? "有视觉上下文"
              : "纯文本提示"}
          </span>
        </div>
        <div className="field-grid">
          <label className="field">
            <span>Drawing Provider</span>
            <select
              value={workspaceConfig.drawingProviderId || ""}
              onChange={(event) =>
                updateWorkspaceConfig({
                  drawingProviderId: Number(event.target.value) || 0,
                })
              }
            >
              <option value="">请选择</option>
              {drawingConfigs.map((item) => (
                <option key={item.provider.id} value={item.provider.id}>
                  {item.provider.name}
                </option>
              ))}
            </select>
          </label>
          <label className="field">
            <span>Drawing Model</span>
            <input
              value={workspaceConfig.drawingModel}
              onChange={(event) =>
                updateWorkspaceConfig({ drawingModel: event.target.value })
              }
              placeholder="gemini-3-pro-image-preview"
            />
          </label>
        </div>
        <div className="provider-summary">
          <strong>{drawingProviderName || "未配置 Drawing Provider"}</strong>
          <p>论文视觉摘要使用当前 LLM 提炼，最终出图固定走这里的 Drawing Model。</p>
          <small className="mono-inline">
            {workspaceConfig.drawingModel || "gemini-3-pro-image-preview"}
          </small>
        </div>
        <label className="field">
          <span>Extra Figure Instructions</span>
          <textarea
            className="prompt-input"
            value={workspaceConfig.drawingPromptDraft}
            onChange={(event) =>
              updateWorkspaceConfig({ drawingPromptDraft: event.target.value })
            }
          />
        </label>
        <Button
          variant="secondary"
          onClick={() => void handleGenerateFigure()}
          disabled={
            isGeneratingFigure ||
            !llmProviderId ||
            !llmModelId ||
            !workspaceConfig.drawingProviderId ||
            !workspaceConfig.drawingModel.trim()
          }
        >
          {isGeneratingFigure ? "生成中..." : "生成论文总结图"}
        </Button>
        {drawingError ? (
          <div className="reader-error">{drawingError}</div>
        ) : null}
        {generatedFigure ? (
          <img
            className="generated-figure"
            src={generatedFigure}
            alt="Generated scientific figure"
          />
        ) : null}
      </div>

      {panelError ? <div className="reader-error">{panelError}</div> : null}

      <div className="ai-section">
        <div className="section-header">
          <strong>当前结果</strong>
          <span className="badge badge-count">{results.length}</span>
        </div>
        <div className="ai-results">
          {results.length ? (
            results.map((result) => (
              <article
                key={result.id}
                className={`note-card ai-result-card ${result.status === "error" ? "ai-result-card-error" : ""}`}
              >
                <div className="ai-result-header">
                  <div>
                    <strong>{result.title}</strong>
                    <div className="ai-result-meta">
                      <span className="history-kind">
                        {historyKindLabel(result.kind)}
                      </span>
                      <small>
                        {new Date(result.createdAt).toLocaleString()}
                      </small>
                    </div>
                  </div>
                  <div className="ai-result-actions">
                    <button
                      type="button"
                      className="icon-button"
                      onClick={() => void copyText(result.content)}
                      title="复制结果"
                    >
                      <Copy size={14} />
                    </button>
                    <button
                      type="button"
                      className="icon-button"
                      onClick={() =>
                        downloadMarkdown(
                          `${sanitizeFileName(tab.title)}-${result.kind}.md`,
                          result.content,
                        )
                      }
                      title="导出 Markdown"
                    >
                      <Download size={14} />
                    </button>
                  </div>
                </div>
                <small className="ai-result-prompt">{result.prompt}</small>
                {result.meta ? (
                  <small className="mono-inline">{result.meta}</small>
                ) : null}
                <MarkdownPreview
                  content={result.content}
                  placeholder={
                    result.status === "streaming"
                      ? "AI 正在生成结果..."
                      : "暂无结果。"
                  }
                />
              </article>
            ))
          ) : (
            <p className="empty-inline">
              会自动恢复最近的 AI 结果；也可以先点一个快捷任务。
            </p>
          )}
        </div>
      </div>

      <div className="ai-section">
        <div className="section-header">
          <strong>
            <History size={14} /> 历史记录
          </strong>
          <div className="prompt-actions">
            <Button
              variant="secondary"
              size="sm"
              onClick={() =>
                void refreshHistory(
                  tab.workspaceId ?? "",
                  tab.documentId ?? "",
                  tab.id,
                  setChatHistory,
                  setPanelError,
                )
              }
              disabled={!tab.id}
            >
              <RefreshCw size={14} />
              刷新
            </Button>
          </div>
        </div>
        <div className="note-list">
          {availableHistory.length ? (
            availableHistory.map((entry) => {
              const isDeleting = deletingHistoryIDs.includes(entry.id);
              return (
                <article key={entry.id} className="note-card ai-history-card">
                  <div className="note-meta">
                    <span className="history-kind">
                      {historyKindLabel(entry.kind)}
                    </span>
                    <small>{new Date(entry.createdAt).toLocaleString()}</small>
                  </div>
                  <strong>{entry.prompt}</strong>
                  <p>
                    {entry.response.slice(0, 140)}
                    {entry.response.length > 140 ? "..." : ""}
                  </p>
                  <div className="prompt-actions">
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => loadHistoryEntry(entry)}
                      disabled={isDeleting}
                    >
                      <Sparkles size={14} />
                      载入工作区
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => void handleDeleteHistoryEntry(entry)}
                      disabled={isDeleting}
                    >
                      <Trash2 size={14} />
                      {isDeleting ? "删除中..." : "删除"}
                    </Button>
                  </div>
                </article>
              );
            })
          ) : (
            <p className="empty-inline">还没有 AI 历史记录。</p>
          )}
        </div>
      </div>
    </div>
  );
}

function buildSingleSummaryInstruction() {
  return `请阅读当前论文，并输出适合侧边栏阅读的 Markdown 总结。

要求：
1. 使用简体中文。
2. 按照以下标题输出：
## 一句话结论
## 研究问题
## 核心方法
## 实验与结果
## 创新点
## 局限性
## 下一步阅读建议
3. 每一节尽量短句化，突出信息密度。
4. 如果提供的论文文本可能不完整，请在最后增加“## 信息边界”说明。`;
}

function buildTableInstruction(config: AIWorkspaceConfig) {
  return `${config.tablePrompt}

表格模板：
${config.tableTemplate}`;
}

function buildMethodsInstruction() {
  return `请把当前论文的方法部分拆解成工程师可复述的说明。

要求：
1. 使用简体中文和 Markdown。
2. 依次输出：
## 方法主线
## 输入与输出
## 关键模块
## 训练/推理流程
## 公式或指标解释
## 最容易忽略的实现细节
3. 如果有公式，保留 LaTeX 写法。`;
}

function buildSelectionInstruction() {
  return `请围绕当前选中的论文片段进行精读讲解。

要求：
1. 使用简体中文。
2. 先解释原文在说什么，再解释它为什么重要。
3. 如果里面有术语、公式或缩写，请补充白话解释。
4. 最后给出一个“我应该带着什么问题继续往下读”的小结。`;
}

function renderMultiRoundProgress(
  finishedChunks: Array<{ chunk: PDFTextChunk; summary: string }>,
  activeChunk: { chunk: PDFTextChunk; content: string } | null,
  finalDraft: string,
  totalChunks: number,
) {
  const sections: string[] = [
    "# 多轮分块总结",
    "",
    `- 已完成 ${finishedChunks.length}/${totalChunks} 段`,
  ];

  if (finishedChunks.length) {
    sections.push("", "## 已完成的分块纪要");
    finishedChunks.forEach((item, index) => {
      sections.push(
        "",
        `### Chunk ${index + 1} · P${item.chunk.startPage}-${item.chunk.endPage}`,
        item.summary,
      );
    });
  }

  if (activeChunk) {
    sections.push(
      "",
      `## 正在处理 Chunk · P${activeChunk.chunk.startPage}-${activeChunk.chunk.endPage}`,
      activeChunk.content || "AI 正在读取这一段...",
    );
  }

  if (finalDraft) {
    sections.push("", "## 正在综合最终总结", finalDraft);
  }

  return sections.join("\n");
}

function historyEntryToResult(entry: ChatHistoryEntry): AIResult {
  return {
    id: `history-${entry.id}`,
    title: historyKindTitle(entry.kind),
    kind: entry.kind,
    prompt: entry.prompt,
    content: entry.response,
    createdAt: entry.createdAt,
    status: "done",
    source: "history",
    meta: `来自历史记录 · P${entry.page}`,
  };
}

async function refreshHistory(
  workspaceID: string,
  documentID: string,
  itemID: string,
  setChatHistory: (value: ChatHistoryEntry[]) => void,
  setPanelError: (value: string | null) => void,
) {
  if (!itemID && !documentID) {
    return;
  }

  try {
    setPanelError(null);
    setChatHistory(await historyApi.listChatHistory(workspaceID, documentID, itemID));
  } catch (error) {
    setPanelError(error instanceof Error ? error.message : "刷新历史失败");
  }
}

function historyKindLabel(kind: string) {
  switch (kind) {
    case "summary":
      return "SUMMARY";
    case "summary_multi":
      return "MULTI";
    case "table":
      return "TABLE";
    case "analysis":
      return "ANALYSIS";
    case "followup":
      return "FOLLOWUP";
    case "translation":
      return "TRANSLATION";
    default:
      return "CHAT";
  }
}

function historyKindTitle(kind: string) {
  switch (kind) {
    case "summary":
      return "结构化总结";
    case "summary_multi":
      return "多轮分块总结";
    case "table":
      return "表格摘要";
    case "analysis":
      return "方法拆解";
    case "followup":
      return "继续追问";
    default:
      return "历史回答";
  }
}

function estimateTextBudget(contextWindow: number) {
  if (!contextWindow || contextWindow <= 0) {
    return 24000;
  }
  return Math.max(12000, Math.min(90000, Math.floor(contextWindow * 1.8)));
}

function createResultId() {
  return `${Date.now()}-${Math.random().toString(16).slice(2, 8)}`;
}

async function copyText(value: string) {
  if (!value.trim()) {
    return;
  }
  await navigator.clipboard.writeText(value);
}

function downloadMarkdown(filename: string, content: string) {
  const blob = new Blob([content], { type: "text/markdown;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

function sanitizeFileName(input: string) {
  return input.replace(/[\\/:*?"<>|]/g, "_").slice(0, 80) || "paper";
}
