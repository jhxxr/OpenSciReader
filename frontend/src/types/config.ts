export type ProviderType = "llm" | "ocr" | "drawing" | "translate";

export type PDFTranslateRuntimeStatus =
  | "missing"
  | "valid"
  | "invalid"
  | "installing";

export interface PDFTranslateRuntimeConfig {
  installed: boolean;
  status: PDFTranslateRuntimeStatus;
  runtimeId: string;
  version: string;
  platform: string;
  runtimeDir: string;
  pythonPath: string;
  manifestPath: string;
  installedAt: string;
  sourceFileName: string;
  lastValidationError: string;
}

export interface PDFTranslateRuntimeImportProgress {
  stage: string;
  message: string;
  progress: number;
  bytesCompleted: number;
  bytesTotal: number;
}

export const DEFAULT_PDF_TRANSLATE_RUNTIME_CONFIG: PDFTranslateRuntimeConfig = {
  installed: false,
  status: "missing",
  runtimeId: "pdf2zh-next",
  version: "",
  platform: "windows-amd64",
  runtimeDir: "",
  pythonPath: "",
  manifestPath: "",
  installedAt: "",
  sourceFileName: "",
  lastValidationError: "",
};

export interface ProviderRecord {
  id: number;
  name: string;
  type: ProviderType;
  baseUrl: string;
  region: string;
  hasApiKey: boolean;
  apiKeyMasked: string;
  isActive: boolean;
}

export interface ProviderUpsertInput {
  id?: number;
  name: string;
  type: ProviderType;
  baseUrl: string;
  region: string;
  apiKey: string;
  clearApiKey: boolean;
  isActive: boolean;
}

export interface ModelRecord {
  id: number;
  providerId: number;
  modelId: string;
  contextWindow: number;
}

export interface ModelUpsertInput {
  id?: number;
  providerId: number;
  modelId: string;
  contextWindow: number;
}

export interface ProviderConfig {
  provider: ProviderRecord;
  models: ModelRecord[];
}

export interface ConfigSnapshot {
  providers: ProviderConfig[];
  pdfTranslateRuntime: PDFTranslateRuntimeConfig;
}

export type AISummaryMode = "auto" | "single" | "multi";

export interface AIWorkspaceConfig {
  summaryMode: AISummaryMode;
  summaryChunkPages: number;
  summaryChunkMaxChars: number;
  autoRestoreCount: number;
  tableTemplate: string;
  tablePrompt: string;
  customPromptDraft: string;
  followUpPromptDraft: string;
  drawingPromptDraft: string;
  drawingProviderId: number;
  drawingModel: string;
  wikiScanProviderId: number;
  wikiScanModelId: number;
}

export const DEFAULT_AI_WORKSPACE_CONFIG: AIWorkspaceConfig = {
  summaryMode: "auto",
  summaryChunkPages: 6,
  summaryChunkMaxChars: 18000,
  autoRestoreCount: 3,
  tableTemplate: `| 维度 | 内容 |
| --- | --- |
| 论文标题 | |
| 研究问题 | |
| 核心方法 | |
| 数据/实验设置 | |
| 关键结果 | |
| 创新点 | |
| 局限性 | |
| 我能直接借鉴什么 | |`,
  tablePrompt:
    "请仔细阅读当前论文，并严格按照给定的 Markdown 表格模板填写。要求：1. 只输出填好的表格。2. 所有单元格用中文填写。3. 若原文未明确提及，填写“未明确说明”。4. 内容应简洁但能支持快速比较论文。",
  customPromptDraft: "",
  followUpPromptDraft: "",
  drawingPromptDraft:
    "额外要求：图中文字尽量使用简体中文，整体像一页适合组会汇报的科研海报。",
  drawingProviderId: 0,
  drawingModel: "gemini-3-pro-image-preview",
  wikiScanProviderId: 0,
  wikiScanModelId: 0,
};

export interface DiscoveredModel {
  id: string;
  name: string;
  ownedBy: string;
}

export interface DiscoveredModelsResponse {
  models: DiscoveredModel[];
  total: number;
}

export const PROVIDER_TYPE_LABELS: Record<ProviderType, string> = {
  llm: "LLM",
  ocr: "OCR",
  drawing: "Drawing",
  translate: "Translate",
};

export const PROVIDER_BASE_URL_HINTS: Record<ProviderType, string> = {
  llm: "https://api.openai.com/v1",
  ocr: "https://open.bigmodel.cn/api/paas/v4/layout_parsing",
  drawing: "https://generativelanguage.googleapis.com/v1beta",
  translate: "https://api-free.deepl.com/v2",
};
