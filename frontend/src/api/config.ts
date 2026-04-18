import {
  DEFAULT_AI_WORKSPACE_CONFIG,
  DEFAULT_PDF_TRANSLATE_RUNTIME_CONFIG,
  type AIWorkspaceConfig,
  type ConfigSnapshot,
  type DiscoveredModel,
  type DiscoveredModelsResponse,
  type ModelRecord,
  type ModelUpsertInput,
  type PDFTranslateRuntimeConfig,
  type ProviderConfig,
  type ProviderRecord,
  type ProviderType,
  type ProviderUpsertInput,
} from "../types/config";
import {
  findMatchingProviderTemplate,
  getRecommendedModelsForProvider,
} from "../lib/aiCatalog";

interface WailsApp {
  GetConfigSnapshot: () => Promise<ConfigSnapshot>;
  GetAIWorkspaceConfig: () => Promise<AIWorkspaceConfig>;
  SaveAIWorkspaceConfig: (
    input: AIWorkspaceConfig,
  ) => Promise<AIWorkspaceConfig>;
  GetPDFTranslateRuntimeStatus: () => Promise<PDFTranslateRuntimeConfig>;
  SelectPDFTranslateRuntimePackage: () => Promise<string>;
  ImportPDFTranslateRuntime: (packagePath: string) => Promise<{ runtime: PDFTranslateRuntimeConfig }>;
  RemovePDFTranslateRuntime: () => Promise<void>;
  SaveProvider: (input: ProviderUpsertInput) => Promise<ProviderRecord>;
  DeleteProvider: (id: number) => Promise<void>;
  SaveModel: (input: ModelUpsertInput) => Promise<ModelRecord>;
  FetchProviderModels: (
    providerId: number,
  ) => Promise<DiscoveredModelsResponse>;
  DeleteModel: (id: number) => Promise<void>;
}

function isWailsApp(
  value: unknown,
): value is { go: { main: { App: WailsApp } } } {
  return typeof value === "object" && value !== null && "go" in value;
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function createMockSnapshot(): ConfigSnapshot {
  const providers: ProviderConfig[] = [
    {
      provider: {
        id: 1,
        name: "OpenAI Official",
        type: "llm",
        baseUrl: "https://api.openai.com/v1",
        region: "",
        hasApiKey: true,
        apiKeyMasked: "已配置",
        isActive: true,
      },
      models: [
        { id: 1, providerId: 1, modelId: "gpt-4.1", contextWindow: 1047576 },
        {
          id: 2,
          providerId: 1,
          modelId: "gpt-4.1-mini",
          contextWindow: 1047576,
        },
      ],
    },
    {
      provider: {
        id: 2,
        name: "Gemini Image",
        type: "drawing",
        baseUrl: "https://generativelanguage.googleapis.com/v1beta",
        region: "",
        hasApiKey: true,
        apiKeyMasked: "已配置",
        isActive: true,
      },
      models: [],
    },
    {
      provider: {
        id: 3,
        name: "DeepL",
        type: "translate",
        baseUrl: "https://api-free.deepl.com/v2",
        region: "",
        hasApiKey: false,
        apiKeyMasked: "",
        isActive: true,
      },
      models: [],
    },
  ];

  return { providers, pdfTranslateRuntime: { ...DEFAULT_PDF_TRANSLATE_RUNTIME_CONFIG } };
}

function createMockAIWorkspaceConfig(): AIWorkspaceConfig {
  return {
    ...DEFAULT_AI_WORKSPACE_CONFIG,
    drawingProviderId: 2,
  };
}

function buildMockDiscoveredModels(provider: ProviderRecord): DiscoveredModel[] {
  const recommended = getRecommendedModelsForProvider(provider).map((item) => ({
    id: item.id,
    name: item.label,
    ownedBy: provider.name,
  }));

  if (recommended.length > 0) {
    return recommended;
  }

  const matchedTemplate = findMatchingProviderTemplate(provider);
  if (matchedTemplate?.id === "ollama") {
    return [
      { id: "qwen2.5:14b", name: "Qwen2.5 14B", ownedBy: "ollama" },
      { id: "llama3.1:8b", name: "Llama 3.1 8B", ownedBy: "ollama" },
      {
        id: "granite3.2-vision",
        name: "Granite 3.2 Vision",
        ownedBy: "ollama",
      },
    ];
  }

  if (matchedTemplate?.id === "openrouter") {
    return [
      { id: "openai/gpt-4.1", name: "OpenAI / GPT-4.1", ownedBy: "openrouter" },
      {
        id: "google/gemini-2.5-pro",
        name: "Google / Gemini 2.5 Pro",
        ownedBy: "openrouter",
      },
      {
        id: "deepseek/deepseek-r1",
        name: "DeepSeek / R1",
        ownedBy: "openrouter",
      },
    ];
  }

  return [
    { id: "gpt-4.1", name: "GPT-4.1", ownedBy: provider.name },
    { id: "gpt-4.1-mini", name: "GPT-4.1 mini", ownedBy: provider.name },
  ];
}

function createMockApp(): WailsApp {
  let snapshot = createMockSnapshot();
  let aiWorkspaceConfig = createMockAIWorkspaceConfig();
  let nextProviderId = 4;
  let nextModelId = 3;

  return {
    GetConfigSnapshot: async () => {
      await delay(120);
      return snapshot;
    },
    GetAIWorkspaceConfig: async () => {
      await delay(80);
      return aiWorkspaceConfig;
    },
    SaveAIWorkspaceConfig: async (input) => {
      await delay(80);
      aiWorkspaceConfig = { ...input };
      return aiWorkspaceConfig;
    },
    GetPDFTranslateRuntimeStatus: async () => {
      await delay(80);
      return snapshot.pdfTranslateRuntime;
    },
    SelectPDFTranslateRuntimePackage: async () => {
      await delay(80);
      return "C:/Users/demo/Downloads/OpenSciReader-pdf-runtime-windows-amd64-mock.zip";
    },
    ImportPDFTranslateRuntime: async (packagePath) => {
      await delay(180);
      const trimmed = packagePath.trim();
      if (!trimmed) {
        throw new Error("请选择运行时安装包");
      }
      const fileName = trimmed.split(/[\\/]/).pop() || trimmed;
      const runtime: PDFTranslateRuntimeConfig = {
        installed: true,
        status: "valid",
        runtimeId: "pdf2zh-next",
        version: "mock-runtime",
        platform: "windows-amd64",
        runtimeDir: `C:/Users/demo/.openscireader/reader_translate/runtime/${fileName.replace(/\.[^.]+$/, "")}`,
        pythonPath: "C:/Users/demo/.openscireader/reader_translate/runtime/python.exe",
        manifestPath: "C:/Users/demo/.openscireader/reader_translate/runtime/manifest.json",
        installedAt: new Date().toISOString(),
        sourceFileName: fileName,
        lastValidationError: "",
      };
      snapshot = {
        ...snapshot,
        pdfTranslateRuntime: runtime,
      };
      return { runtime };
    },
    RemovePDFTranslateRuntime: async () => {
      await delay(120);
      snapshot = {
        ...snapshot,
        pdfTranslateRuntime: { ...DEFAULT_PDF_TRANSLATE_RUNTIME_CONFIG },
      };
    },
    SaveProvider: async (input) => {
      await delay(120);
      const provider: ProviderRecord = {
        id: input.id ?? nextProviderId++,
        name: input.name,
        type: input.type,
        baseUrl: input.baseUrl,
        region: input.region,
        hasApiKey: input.clearApiKey
          ? false
          : input.apiKey.length > 0 || Boolean(input.id),
        apiKeyMasked: input.clearApiKey
          ? ""
          : input.apiKey || input.id
            ? "已配置"
            : "",
        isActive: input.isActive,
      };

      const existingIndex = snapshot.providers.findIndex(
        (item) => item.provider.id === provider.id,
      );
      if (existingIndex >= 0) {
        snapshot = {
          ...snapshot,
          providers: snapshot.providers.map((item, index) =>
            index === existingIndex ? { ...item, provider } : item,
          ),
        };
      } else {
        snapshot = {
          ...snapshot,
          providers: [...snapshot.providers, { provider, models: [] }],
        };
      }

      return provider;
    },
    DeleteProvider: async (id) => {
      await delay(80);
      snapshot = {
        ...snapshot,
        providers: snapshot.providers.filter((item) => item.provider.id !== id),
      };
    },
    SaveModel: async (input) => {
      await delay(120);
      const normalizedModelID = input.modelId.trim().toLowerCase();
      const providerConfig = snapshot.providers.find(
        (item) => item.provider.id === input.providerId,
      );
      const duplicate = providerConfig?.models.find(
        (entry) => entry.modelId.trim().toLowerCase() === normalizedModelID,
      );
      if (duplicate && duplicate.id !== input.id) {
        throw new Error(
          `model ${input.modelId} already exists for provider ${input.providerId}`,
        );
      }

      const model: ModelRecord = {
        id: input.id ?? nextModelId++,
        providerId: input.providerId,
        modelId: input.modelId,
        contextWindow: input.contextWindow,
      };

      snapshot = {
        ...snapshot,
        providers: snapshot.providers.map((item) => {
          if (item.provider.id !== input.providerId) {
            return item;
          }

          const existingIndex = item.models.findIndex(
            (entry) => entry.id === model.id,
          );
          if (existingIndex >= 0) {
            return {
              ...item,
              models: item.models.map((entry, index) =>
                index === existingIndex ? model : entry,
              ),
            };
          }

          return { ...item, models: [...item.models, model] };
        }),
      };

      return model;
    },
    FetchProviderModels: async (providerId) => {
      await delay(180);
      const provider = snapshot.providers.find(
        (item) => item.provider.id === providerId,
      )?.provider;
      if (!provider) {
        throw new Error(`provider ${providerId} not found`);
      }
      const models = buildMockDiscoveredModels(provider);
      return { models, total: models.length };
    },
    DeleteModel: async (id) => {
      await delay(80);
      snapshot = {
        ...snapshot,
        providers: snapshot.providers.map((item) => ({
          ...item,
          models: item.models.filter((model) => model.id !== id),
        })),
      };
    },
  };
}

function getApp(): WailsApp {
  if (
    typeof window !== "undefined" &&
    isWailsApp(window) &&
    window.go?.main?.App
  ) {
    return window.go.main.App;
  }

  return createMockApp();
}

const app = getApp();

export const configApi = {
  getConfigSnapshot(): Promise<ConfigSnapshot> {
    return app.GetConfigSnapshot();
  },
  getAIWorkspaceConfig(): Promise<AIWorkspaceConfig> {
    return app.GetAIWorkspaceConfig();
  },
  saveAIWorkspaceConfig(input: AIWorkspaceConfig): Promise<AIWorkspaceConfig> {
    return app.SaveAIWorkspaceConfig(input);
  },
  getPDFTranslateRuntimeStatus(): Promise<PDFTranslateRuntimeConfig> {
    return app.GetPDFTranslateRuntimeStatus();
  },
  selectPDFTranslateRuntimePackage(): Promise<string> {
    return app.SelectPDFTranslateRuntimePackage();
  },
  importPDFTranslateRuntime(packagePath: string): Promise<{ runtime: PDFTranslateRuntimeConfig }> {
    return app.ImportPDFTranslateRuntime(packagePath);
  },
  removePDFTranslateRuntime(): Promise<void> {
    return app.RemovePDFTranslateRuntime();
  },
  saveProvider(input: ProviderUpsertInput): Promise<ProviderRecord> {
    return app.SaveProvider(input);
  },
  deleteProvider(id: number): Promise<void> {
    return app.DeleteProvider(id);
  },
  saveModel(input: ModelUpsertInput): Promise<ModelRecord> {
    return app.SaveModel(input);
  },
  fetchProviderModels(providerId: number): Promise<DiscoveredModelsResponse> {
    return app.FetchProviderModels(providerId);
  },
  deleteModel(id: number): Promise<void> {
    return app.DeleteModel(id);
  },
};

export function createSuggestedProvider(type: ProviderType): ProviderUpsertInput {
  return {
    name: "",
    type,
    baseUrl: "",
    region: "",
    apiKey: "",
    clearApiKey: false,
    isActive: true,
  };
}
