import { create } from 'zustand';
import { configApi } from '../api/config';
import { getErrorMessage } from '../lib/errors';
import type {
  ConfigSnapshot,
  ModelRecord,
  ModelUpsertInput,
  PDFTranslateRuntimeConfig,
  ProviderConfig,
  ProviderRecord,
  ProviderType,
  ProviderUpsertInput,
} from '../types/config';

interface ConfigState {
  snapshot: ConfigSnapshot;
  selectedProviderId: number | null;
  isLoading: boolean;
  runtimeAction: "idle" | "importing" | "removing";
  error: string | null;
  loadConfig: () => Promise<void>;
  importPDFTranslateRuntime: (packagePath: string) => Promise<PDFTranslateRuntimeConfig>;
  removePDFTranslateRuntime: () => Promise<void>;
  saveProvider: (input: ProviderUpsertInput) => Promise<ProviderRecord>;
  deleteProvider: (id: number) => Promise<void>;
  saveModel: (input: ModelUpsertInput) => Promise<ModelRecord>;
  deleteModel: (id: number) => Promise<void>;
  selectProvider: (id: number | null) => void;
  clearError: () => void;
}

function normalizeProviderConfig(config: ProviderConfig): ProviderConfig {
  return {
    ...config,
    models: config.models ?? [],
  };
}

function normalizeSnapshot(snapshot: ConfigSnapshot): ConfigSnapshot {
  return {
    providers: snapshot.providers.map(normalizeProviderConfig),
    pdfTranslateRuntime: snapshot.pdfTranslateRuntime,
  };
}

function upsertProviderConfig(configs: ProviderConfig[], provider: ProviderRecord): ProviderConfig[] {
  const existing = configs.find((item) => item.provider.id === provider.id);
  if (!existing) {
    return [...configs, { provider, models: [] }];
  }

  return configs.map((item) => (item.provider.id === provider.id ? { ...normalizeProviderConfig(item), provider } : normalizeProviderConfig(item)));
}

function upsertModel(configs: ProviderConfig[], model: ModelRecord): ProviderConfig[] {
  return configs.map((item) => {
    const normalized = normalizeProviderConfig(item);
    if (item.provider.id !== model.providerId) {
      return normalized;
    }

    const existing = normalized.models.find((entry) => entry.id === model.id);
    if (!existing) {
      return { ...normalized, models: [...normalized.models, model] };
    }

    return {
      ...normalized,
      models: normalized.models.map((entry) => (entry.id === model.id ? model : entry)),
    };
  });
}

export const useConfigStore = create<ConfigState>((set, get) => ({
  snapshot: { providers: [], pdfTranslateRuntime: { installed: false, status: 'missing', runtimeId: 'pdf2zh-next', version: '', platform: 'windows-amd64', runtimeDir: '', pythonPath: '', manifestPath: '', installedAt: '', sourceFileName: '', lastValidationError: '' } },
  selectedProviderId: null,
  isLoading: false,
  runtimeAction: 'idle',
  error: null,
  async loadConfig() {
    set({ isLoading: true, error: null });
    try {
      const snapshot = normalizeSnapshot(await configApi.getConfigSnapshot());
      const firstProviderId = snapshot.providers[0]?.provider.id ?? null;
      const currentSelected = get().selectedProviderId;
      set({
        snapshot,
        selectedProviderId: currentSelected ?? firstProviderId,
        isLoading: false,
      });
    } catch (error) {
      set({
        isLoading: false,
        error: getErrorMessage(error, 'Failed to load configuration'),
      });
    }
  },
  async importPDFTranslateRuntime(packagePath) {
    set({ runtimeAction: 'importing', error: null });
    try {
      const result = await configApi.importPDFTranslateRuntime(packagePath);
      set({
        snapshot: { ...get().snapshot, pdfTranslateRuntime: result.runtime },
        runtimeAction: 'idle',
      });
      return result.runtime;
    } catch (error) {
      const message = getErrorMessage(error, 'Failed to import PDF translation runtime');
      set({ runtimeAction: 'idle', error: message });
      throw error;
    }
  },
  async removePDFTranslateRuntime() {
    set({ runtimeAction: 'removing', error: null });
    try {
      await configApi.removePDFTranslateRuntime();
      const runtime = await configApi.getPDFTranslateRuntimeStatus();
      set({
        snapshot: { ...get().snapshot, pdfTranslateRuntime: runtime },
        runtimeAction: 'idle',
      });
    } catch (error) {
      const message = getErrorMessage(error, 'Failed to remove PDF translation runtime');
      set({ runtimeAction: 'idle', error: message });
      throw error;
    }
  },
  async saveProvider(input) {
    set({ isLoading: true, error: null });
    try {
      const provider = await configApi.saveProvider(input);
      const providers = upsertProviderConfig(get().snapshot.providers, provider);
      set({
        snapshot: { ...get().snapshot, providers },
        selectedProviderId: provider.id,
        isLoading: false,
      });
      return provider;
    } catch (error) {
      const message = getErrorMessage(error, 'Failed to save provider');
      set({ isLoading: false, error: message });
      throw error;
    }
  },
  async deleteProvider(id) {
    set({ isLoading: true, error: null });
    try {
      await configApi.deleteProvider(id);
      const providers = get().snapshot.providers.filter((item) => item.provider.id !== id);
      const selectedProviderId = get().selectedProviderId === id ? providers[0]?.provider.id ?? null : get().selectedProviderId;
      set({ snapshot: { ...get().snapshot, providers }, selectedProviderId, isLoading: false });
    } catch (error) {
      set({
        isLoading: false,
        error: getErrorMessage(error, 'Failed to delete provider'),
      });
    }
  },
  async saveModel(input) {
    set({ isLoading: true, error: null });
    try {
      const model = await configApi.saveModel(input);
      set({
        snapshot: { ...get().snapshot, providers: upsertModel(get().snapshot.providers, model) },
        isLoading: false,
      });
      return model;
    } catch (error) {
      const message = getErrorMessage(error, 'Failed to save model');
      set({ isLoading: false, error: message });
      throw error;
    }
  },
  async deleteModel(id) {
    set({ isLoading: true, error: null });
    try {
      await configApi.deleteModel(id);
      set({
        snapshot: {
          ...get().snapshot,
          providers: get().snapshot.providers.map((item) => {
            const normalized = normalizeProviderConfig(item);
            return {
              ...normalized,
              models: normalized.models.filter((model) => model.id !== id),
            };
          }),
        },
        isLoading: false,
      });
    } catch (error) {
      const message = getErrorMessage(error, 'Failed to delete model');
      set({
        isLoading: false,
        error: message,
      });
      throw error;
    }
  },
  selectProvider(id) {
    set({ selectedProviderId: id });
  },
  clearError() {
    set({ error: null });
  },
}));

export const useProviderConfigs = () => useConfigStore((state) => state.snapshot.providers);
export const useSelectedProviderConfig = () =>
  useConfigStore((state) =>
    state.snapshot.providers.find((item) => item.provider.id === state.selectedProviderId) ?? null,
  );
export const useProvidersByType = (type: ProviderType) =>
  useConfigStore((state) => state.snapshot.providers.filter((item) => item.provider.type === type));
