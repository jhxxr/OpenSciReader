import type {
  ProviderRecord,
  ProviderType,
  ProviderUpsertInput,
} from "../types/config";

export interface ModelPreset {
  id: string;
  label: string;
  contextWindow: number;
  description: string;
  tags: string[];
  providerTemplateIds: string[];
}

export interface ProviderTemplate {
  id: string;
  type: ProviderType;
  name: string;
  baseUrl: string;
  region?: string;
  description: string;
  discoveryMode: "openai_compat" | "manual" | "none";
  recommendedModelIds: string[];
  matchers: {
    nameIncludes?: string[];
    baseUrlIncludes?: string[];
  };
}

const MODEL_PRESETS: ModelPreset[] = [
  {
    id: "gpt-4.1",
    label: "GPT-4.1",
    contextWindow: 1047576,
    description: "通用研究问答和图像理解。",
    tags: ["OpenAI", "通用", "图像"],
    providerTemplateIds: ["openai-official"],
  },
  {
    id: "gpt-4.1-mini",
    label: "GPT-4.1 mini",
    contextWindow: 1047576,
    description: "成本更低，适合日常阅读助手。",
    tags: ["OpenAI", "轻量", "图像"],
    providerTemplateIds: ["openai-official"],
  },
  {
    id: "gpt-4.1-nano",
    label: "GPT-4.1 nano",
    contextWindow: 1047576,
    description: "超轻量快速补全。",
    tags: ["OpenAI", "极速"],
    providerTemplateIds: ["openai-official"],
  },
  {
    id: "deepseek-chat",
    label: "DeepSeek Chat",
    contextWindow: 128000,
    description: "通用对话与翻译。",
    tags: ["DeepSeek", "通用"],
    providerTemplateIds: ["deepseek"],
  },
  {
    id: "deepseek-reasoner",
    label: "DeepSeek Reasoner",
    contextWindow: 128000,
    description: "更适合长链路推理和总结。",
    tags: ["DeepSeek", "推理"],
    providerTemplateIds: ["deepseek"],
  },
  {
    id: "glm-5.1",
    label: "GLM 5.1",
    contextWindow: 204800,
    description: "较新的 GLM 通用模型。",
    tags: ["GLM", "通用"],
    providerTemplateIds: ["zhipu-glm"],
  },
  {
    id: "glm-5",
    label: "GLM 5",
    contextWindow: 204800,
    description: "适合中文学术场景。",
    tags: ["GLM", "中文"],
    providerTemplateIds: ["zhipu-glm"],
  },
  {
    id: "glm-5-turbo",
    label: "GLM 5 Turbo",
    contextWindow: 204800,
    description: "更偏速度和成本。",
    tags: ["GLM", "轻量"],
    providerTemplateIds: ["zhipu-glm"],
  },
  {
    id: "glm-4.7",
    label: "GLM 4.7",
    contextWindow: 204800,
    description: "兼容旧项目配置。",
    tags: ["GLM", "兼容"],
    providerTemplateIds: ["zhipu-glm"],
  },
  {
    id: "qwen3.5-plus",
    label: "Qwen3.5 Plus",
    contextWindow: 1000000,
    description: "适合长上下文阅读和问答。",
    tags: ["Qwen", "长上下文"],
    providerTemplateIds: ["dashscope-compatible"],
  },
];

const PROVIDER_TEMPLATES: ProviderTemplate[] = [
  {
    id: "openai-official",
    type: "llm",
    name: "OpenAI Official",
    baseUrl: "https://api.openai.com/v1",
    description: "官方 OpenAI 兼容入口，可直接配推荐模型。",
    discoveryMode: "openai_compat",
    recommendedModelIds: ["gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano"],
    matchers: {
      nameIncludes: ["openai"],
      baseUrlIncludes: ["api.openai.com/v1"],
    },
  },
  {
    id: "deepseek",
    type: "llm",
    name: "DeepSeek",
    baseUrl: "https://api.deepseek.com/v1",
    description: "官方 OpenAI-compatible 接口。",
    discoveryMode: "openai_compat",
    recommendedModelIds: ["deepseek-chat", "deepseek-reasoner"],
    matchers: {
      nameIncludes: ["deepseek"],
      baseUrlIncludes: ["api.deepseek.com/v1"],
    },
  },
  {
    id: "zhipu-glm",
    type: "llm",
    name: "Zhipu GLM",
    baseUrl: "https://open.bigmodel.cn/api/paas/v4",
    description: "智谱 OpenAI-compatible 接口。",
    discoveryMode: "openai_compat",
    recommendedModelIds: ["glm-5.1", "glm-5", "glm-5-turbo", "glm-4.7"],
    matchers: {
      nameIncludes: ["zhipu", "glm"],
      baseUrlIncludes: [
        "open.bigmodel.cn/api/paas/v4/layout_parsing",
        "open.bigmodel.cn/api/paas/v4",
      ],
    },
  },
  {
    id: "dashscope-compatible",
    type: "llm",
    name: "DashScope Compatible",
    baseUrl: "https://dashscope.aliyuncs.com/compatible-mode/v1",
    description: "阿里云百炼兼容模式。",
    discoveryMode: "openai_compat",
    recommendedModelIds: ["qwen3.5-plus"],
    matchers: {
      nameIncludes: ["dashscope", "qwen"],
      baseUrlIncludes: ["dashscope.aliyuncs.com/compatible-mode/v1"],
    },
  },
  {
    id: "openrouter",
    type: "llm",
    name: "OpenRouter",
    baseUrl: "https://openrouter.ai/api/v1",
    description: "模型很多，建议保存后直接在线发现。",
    discoveryMode: "openai_compat",
    recommendedModelIds: [],
    matchers: {
      nameIncludes: ["openrouter"],
      baseUrlIncludes: ["openrouter.ai/api/v1"],
    },
  },
  {
    id: "ollama",
    type: "llm",
    name: "Ollama",
    baseUrl: "http://localhost:11434/v1",
    description: "本地模型网关，建议直接在线发现本机模型。",
    discoveryMode: "openai_compat",
    recommendedModelIds: [],
    matchers: {
      nameIncludes: ["ollama"],
      baseUrlIncludes: ["localhost:11434/v1", "127.0.0.1:11434/v1"],
    },
  },
  {
    id: "glm-ocr",
    type: "ocr",
    name: "GLM OCR",
    baseUrl: "https://open.bigmodel.cn/api/paas/v4/layout_parsing",
    description: "GLM OCR 的 layout_parsing 接口。",
    discoveryMode: "none",
    recommendedModelIds: [],
    matchers: {
      nameIncludes: ["glm", "ocr"],
      baseUrlIncludes: [
        "open.bigmodel.cn/api/paas/v4/layout_parsing",
        "open.bigmodel.cn/api/paas/v4",
      ],
    },
  },
  {
    id: "gemini-image",
    type: "drawing",
    name: "Gemini Image",
    baseUrl: "https://generativelanguage.googleapis.com/v1beta",
    description:
      "Gemini 图像生成接口，适合 gemini-3-pro-image-preview 和 gemini-2.5-flash-image。",
    discoveryMode: "none",
    recommendedModelIds: [],
    matchers: {
      nameIncludes: ["gemini", "google"],
      baseUrlIncludes: ["generativelanguage.googleapis.com/v1beta"],
    },
  },
  {
    id: "deepl",
    type: "translate",
    name: "DeepL",
    baseUrl: "https://api-free.deepl.com/v2",
    description: "官方 DeepL 翻译接口。",
    discoveryMode: "none",
    recommendedModelIds: [],
    matchers: {
      nameIncludes: ["deepl"],
      baseUrlIncludes: ["api-free.deepl.com/v2", "api.deepl.com/v2"],
    },
  },
  {
    id: "deeplx",
    type: "translate",
    name: "DeepLX",
    baseUrl: "https://api.deeplx.org/translate",
    description:
      "自部署 DeepLX，URL 需要填写完整 endpoint，并包含最后的 /translate。",
    discoveryMode: "none",
    recommendedModelIds: [],
    matchers: {
      nameIncludes: ["deeplx"],
      baseUrlIncludes: ["deeplx", "/v1/translate", "/v2/translate"],
    },
  },
  {
    id: "google-translate",
    type: "translate",
    name: "Google Cloud Translation",
    baseUrl: "https://translation.googleapis.com/language/translate/v2",
    description: "Google 翻译 API。",
    discoveryMode: "none",
    recommendedModelIds: [],
    matchers: {
      nameIncludes: ["google"],
      baseUrlIncludes: [
        "translation.googleapis.com/language/translate/v2",
      ],
    },
  },
  {
    id: "microsoft-translate",
    type: "translate",
    name: "Microsoft Translator",
    baseUrl: "https://api.cognitive.microsofttranslator.com/translate",
    region: "eastasia",
    description: "微软翻译接口，通常还需要 region。",
    discoveryMode: "none",
    recommendedModelIds: [],
    matchers: {
      nameIncludes: ["microsoft", "azure"],
      baseUrlIncludes: ["cognitive.microsofttranslator.com/translate"],
    },
  },
];

function normalize(value: string): string {
  return value.trim().toLowerCase();
}

export function getProviderTemplatesByType(type: ProviderType): ProviderTemplate[] {
  return PROVIDER_TEMPLATES.filter((item) => item.type === type);
}

export function createProviderFromTemplate(
  template: ProviderTemplate,
): ProviderUpsertInput {
  return {
    name: template.name,
    type: template.type,
    baseUrl: template.baseUrl,
    region: template.region ?? "",
    apiKey: "",
    clearApiKey: false,
    isActive: true,
  };
}

export function findModelPreset(modelId: string): ModelPreset | undefined {
  const normalizedModelID = normalize(modelId);
  if (!normalizedModelID) {
    return undefined;
  }

  return MODEL_PRESETS.find(
    (item) => normalize(item.id) === normalizedModelID,
  );
}

export function suggestContextWindow(modelId: string): number {
  return findModelPreset(modelId)?.contextWindow ?? 0;
}

export function findMatchingProviderTemplate(
  provider: Pick<ProviderRecord, "name" | "baseUrl" | "type">,
): ProviderTemplate | null {
  const providerName = normalize(provider.name);
  const providerBaseURL = normalize(provider.baseUrl);

  for (const template of PROVIDER_TEMPLATES) {
    if (template.type !== provider.type) {
      continue;
    }

    const nameMatched = (template.matchers.nameIncludes ?? []).some((item) =>
      providerName.includes(normalize(item)),
    );
    const baseURLMatched = (template.matchers.baseUrlIncludes ?? []).some(
      (item) => providerBaseURL.includes(normalize(item)),
    );

    if (nameMatched || baseURLMatched) {
      return template;
    }
  }

  return null;
}

export function getRecommendedModelsForProvider(
  provider: Pick<ProviderRecord, "name" | "baseUrl" | "type">,
): ModelPreset[] {
  const matchedTemplate = findMatchingProviderTemplate(provider);
  if (!matchedTemplate) {
    return [];
  }

  return matchedTemplate.recommendedModelIds
    .map((modelId) => findModelPreset(modelId))
    .filter((item): item is ModelPreset => Boolean(item));
}
