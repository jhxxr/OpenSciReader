import { useCallback, useEffect, useMemo, useState, type Dispatch, type ReactNode, type SetStateAction } from 'react';
import { BookOpen, FileText, Home, Plus, Settings2, Sparkles, Trash2, Wand2, X } from 'lucide-react';
import './App.css';
import { configApi } from './api/config';
import { workspaceApi } from './api/workspace';
import { workspaceKnowledgeApi } from './api/workspaceKnowledge';
import { workspaceWikiApi } from './api/workspaceWiki';
import { ModelDiscoveryModal } from './components/ModelDiscoveryModal';
import { HomePage } from './components/HomePage';
import { ReaderTab } from './components/ReaderTab';
import { WorkspaceTab } from './components/WorkspaceTab';
import { Button } from './components/ui/Button';
import { getErrorMessage } from './lib/errors';
import {
  createProviderFromTemplate,
  findMatchingProviderTemplate,
  findModelPreset,
  getProviderTemplatesByType,
  getRecommendedModelsForProvider,
  suggestContextWindow,
  type ModelPreset,
  type ProviderTemplate,
} from './lib/aiCatalog';
import { useConfigStore, useProviderConfigs } from './store/configStore';
import { useTabStore } from './store/tabStore';
import { useWorkspaceStore } from './store/workspaceStore';
import { useZoteroStore } from './store/zoteroStore';
import {
  PROVIDER_BASE_URL_HINTS,
  DEFAULT_AI_WORKSPACE_CONFIG,
  type DiscoveredModel,
  type ModelRecord,
  type ModelUpsertInput,
  type PDFTranslateRuntimeConfig,
  type PDFTranslateRuntimeImportProgress,
  type ProviderConfig,
  type ProviderRecord,
  type ProviderUpsertInput,
} from './types/config';
import type { WorkspaceWikiPage, WorkspaceWikiPageContent, WorkspaceWikiScanJob } from './types/workspaceWiki';
import type { WorkspaceKnowledgeCompileSummary, WorkspaceKnowledgeSource } from './types/workspaceKnowledge';

const EMPTY_LLM_FORM: ProviderUpsertInput = {
  name: '',
  type: 'llm',
  baseUrl: '',
  region: '',
  apiKey: '',
  clearApiKey: false,
  isActive: true,
};

const EMPTY_DRAWING_FORM: ProviderUpsertInput = {
  name: '',
  type: 'drawing',
  baseUrl: PROVIDER_BASE_URL_HINTS.drawing,
  region: '',
  apiKey: '',
  clearApiKey: false,
  isActive: true,
};

const EMPTY_TRANSLATION_FORM: ProviderUpsertInput = {
  name: '',
  type: 'translate',
  baseUrl: PROVIDER_BASE_URL_HINTS.translate,
  region: '',
  apiKey: '',
  clearApiKey: false,
  isActive: true,
};

const EMPTY_MODEL_FORM: ModelUpsertInput = {
  providerId: 0,
  modelId: '',
  contextWindow: 0,
};

function normalizeModelKey(value: string): string {
  return value.trim().toLowerCase();
}

function currentSelectedPageId(
  workspaceId: string,
  pages: WorkspaceWikiPage[],
  state: Record<string, { selectedPageId: string | null }>,
): string | null {
  const existing = state[workspaceId]?.selectedPageId;
  if (existing && pages.some((page) => page.id === existing)) {
    return existing;
  }
  return pages[0]?.id ?? null;
}

function isDeepLXTranslateEndpoint(value: string): boolean {
  try {
    const parsedURL = new URL(value.trim());
    const normalizedPath = parsedURL.pathname.replace(/\/+$/, '').toLowerCase();
    return normalizedPath.endsWith('/translate');
  } catch {
    return false;
  }
}

export default function App() {
  const tabs = useTabStore((state) => state.tabs);
  const activeTabId = useTabStore((state) => state.activeTabId);
  const openTab = useTabStore((state) => state.openTab);
  const closeTab = useTabStore((state) => state.closeTab);
  const setActiveTab = useTabStore((state) => state.setActiveTab);

  const zotero = useZoteroStore();
  const workspace = useWorkspaceStore();
  const providerConfigs = useProviderConfigs();
  const llmProviderConfigs = useMemo(
    () => providerConfigs.filter((item) => item.provider.type === 'llm'),
    [providerConfigs],
  );
  const drawingProviderConfigs = useMemo(
    () => providerConfigs.filter((item) => item.provider.type === 'drawing'),
    [providerConfigs],
  );
  const translationProviderConfigs = useMemo(
    () => providerConfigs.filter((item) => item.provider.type === 'translate'),
    [providerConfigs],
  );

  const { loadConfig, importPDFTranslateRuntime, removePDFTranslateRuntime, saveProvider, deleteProvider, saveModel, deleteModel, isLoading, runtimeAction, error, clearError, snapshot } =
    useConfigStore();

  const [configOpen, setConfigOpen] = useState(false);
  const [llmForm, setLlmForm] = useState<ProviderUpsertInput>(EMPTY_LLM_FORM);
  const [drawingForm, setDrawingForm] = useState<ProviderUpsertInput>(EMPTY_DRAWING_FORM);
  const [translationForm, setTranslationForm] = useState<ProviderUpsertInput>(EMPTY_TRANSLATION_FORM);
  const [modelForm, setModelForm] = useState<ModelUpsertInput>(EMPTY_MODEL_FORM);
  const [discoveryProviderId, setDiscoveryProviderId] = useState<number | null>(null);
  const [discoveredModels, setDiscoveredModels] = useState<DiscoveredModel[]>([]);
  const [isDiscoveringModels, setIsDiscoveringModels] = useState(false);
  const [discoveryError, setDiscoveryError] = useState<string | null>(null);
  const [deletingModelId, setDeletingModelId] = useState<number | null>(null);
  const [runtimeImportPath, setRuntimeImportPath] = useState('');
  const [runtimeImportProgress, setRuntimeImportProgress] = useState<PDFTranslateRuntimeImportProgress | null>(null);
  const [workspaceTabDocuments, setWorkspaceTabDocuments] = useState<Record<string, { documents: import('./types/workspace').DocumentRecord[]; isLoading: boolean; deletingDocumentId: string | null }>>({});
const [workspaceTabWiki, setWorkspaceTabWiki] = useState<Record<string, {
  pages: WorkspaceWikiPage[];
  selectedPageId: string | null;
  pageContent: WorkspaceWikiPageContent | null;
  isLoadingPages: boolean;
  isLoadingPageContent: boolean;
  activeJob: WorkspaceWikiScanJob | null;
  wikiError: string | null;
  isStarting: boolean;
  isCancelling: boolean;
  isDeleting: boolean;
  unsubscribe?: (() => void) | null;
  wikiScanProviderId: number;
  wikiScanModelId: number;
  sources: WorkspaceKnowledgeSource[];
  compileSummary: WorkspaceKnowledgeCompileSummary | null;
}>>({});

  useEffect(() => {
    void loadConfig();
  }, [loadConfig]);

  useEffect(() => {
    void zotero.loadCollections();
  }, [zotero.loadCollections]);

  useEffect(() => {
    void workspace.loadWorkspaces();
  }, [workspace.loadWorkspaces]);

  useEffect(() => configApi.subscribePDFTranslateRuntimeImportProgress(setRuntimeImportProgress), []);

  useEffect(() => {
    if (runtimeAction !== 'importing') {
      setRuntimeImportProgress(null);
    }
  }, [runtimeAction]);

  useEffect(() => {
    setModelForm((current) => {
      if (llmProviderConfigs.some((item) => item.provider.id === current.providerId)) {
        return current;
      }
      return { ...current, providerId: llmProviderConfigs[0]?.provider.id ?? 0 };
    });
  }, [llmProviderConfigs]);

  const llmTemplates = useMemo(() => getProviderTemplatesByType('llm'), []);
  const drawingTemplates = useMemo(() => getProviderTemplatesByType('drawing'), []);
  const translationTemplates = useMemo(() => getProviderTemplatesByType('translate'), []);

  const selectedModelProviderConfig = useMemo(
    () => llmProviderConfigs.find((item) => item.provider.id === modelForm.providerId) ?? null,
    [llmProviderConfigs, modelForm.providerId],
  );
  const selectedModelProvider = selectedModelProviderConfig?.provider ?? null;
  const selectedModelProviderId = selectedModelProvider?.id ?? 0;
  const selectedModelProviderModels = selectedModelProviderConfig?.models ?? [];
  const selectedModelProviderTemplate = useMemo(
    () => (selectedModelProvider ? findMatchingProviderTemplate(selectedModelProvider) : null),
    [selectedModelProvider],
  );
  const selectedTranslationTemplate = useMemo(
    () =>
      findMatchingProviderTemplate({
        name: translationForm.name,
        baseUrl: translationForm.baseUrl,
        type: translationForm.type,
      }),
    [translationForm.baseUrl, translationForm.name, translationForm.type],
  );
  const selectedModelProviderRecommendedModels = useMemo(
    () => (selectedModelProvider ? getRecommendedModelsForProvider(selectedModelProvider) : []),
    [selectedModelProvider],
  );
  const selectedModelProviderExistingIds = useMemo(
    () => new Set(selectedModelProviderModels.map((item) => normalizeModelKey(item.modelId))),
    [selectedModelProviderModels],
  );

  const discoveryProviderConfig = useMemo(
    () => llmProviderConfigs.find((item) => item.provider.id === discoveryProviderId) ?? null,
    [llmProviderConfigs, discoveryProviderId],
  );

  const ensureWorkspaceTabDocuments = useCallback(async (workspaceId: string) => {
    setWorkspaceTabDocuments((current) => ({
      ...current,
      [workspaceId]: {
        documents: current[workspaceId]?.documents ?? [],
        isLoading: true,
        deletingDocumentId: current[workspaceId]?.deletingDocumentId ?? null,
      },
    }));
    try {
      const documents = await workspaceApi.listDocuments(workspaceId);
      setWorkspaceTabDocuments((current) => ({
        ...current,
        [workspaceId]: {
          documents,
          isLoading: false,
          deletingDocumentId: current[workspaceId]?.deletingDocumentId ?? null,
        },
      }));
    } catch {
      setWorkspaceTabDocuments((current) => ({
        ...current,
        [workspaceId]: {
          documents: current[workspaceId]?.documents ?? [],
          isLoading: false,
          deletingDocumentId: current[workspaceId]?.deletingDocumentId ?? null,
        },
      }));
    }
  }, []);

  const ensureWorkspaceTabWiki = useCallback(async (workspaceId: string) => {
    setWorkspaceTabWiki((current) => ({
      ...current,
      [workspaceId]: {
        pages: current[workspaceId]?.pages ?? [],
        selectedPageId: current[workspaceId]?.selectedPageId ?? null,
        pageContent: current[workspaceId]?.pageContent ?? null,
        isLoadingPages: true,
        isLoadingPageContent: current[workspaceId]?.isLoadingPageContent ?? false,
        activeJob: current[workspaceId]?.activeJob ?? null,
        wikiError: null,
        isStarting: current[workspaceId]?.isStarting ?? false,
        isCancelling: current[workspaceId]?.isCancelling ?? false,
        isDeleting: current[workspaceId]?.isDeleting ?? false,
        unsubscribe: current[workspaceId]?.unsubscribe ?? null,
        wikiScanProviderId: current[workspaceId]?.wikiScanProviderId ?? 0,
        wikiScanModelId: current[workspaceId]?.wikiScanModelId ?? 0,
        sources: current[workspaceId]?.sources ?? [],
        compileSummary: current[workspaceId]?.compileSummary ?? null,
      },
    }));
    try {
      const [pages, jobs, sources, compileSummary] = await Promise.all([
        workspaceWikiApi.listPages(workspaceId),
        workspaceWikiApi.listJobs(),
        workspaceKnowledgeApi.listSources(workspaceId),
        workspaceKnowledgeApi.getCompileSummary(workspaceId),
      ]);
      const activeJob = jobs.find((job) => job.workspaceId === workspaceId && (job.status === 'queued' || job.status === 'running')) ?? null;
      const selectedPageId = currentSelectedPageId(workspaceId, pages, workspaceTabWiki);
      
      let wikiScanProviderId = 0;
      let wikiScanModelId = 0;
      try {
        const config = await configApi.getAIWorkspaceConfig(workspaceId);
        wikiScanProviderId = config.wikiScanProviderId ?? 0;
        wikiScanModelId = config.wikiScanModelId ?? 0;
      } catch { /* use defaults */ }
      
      setWorkspaceTabWiki((current) => ({
        ...current,
        [workspaceId]: {
          pages,
          selectedPageId,
          pageContent: selectedPageId && current[workspaceId]?.pageContent?.page.id === selectedPageId ? current[workspaceId]?.pageContent ?? null : null,
          isLoadingPages: false,
          isLoadingPageContent: false,
          activeJob,
          wikiError: null,
          isStarting: current[workspaceId]?.isStarting ?? false,
          isCancelling: current[workspaceId]?.isCancelling ?? false,
          isDeleting: current[workspaceId]?.isDeleting ?? false,
          unsubscribe: current[workspaceId]?.unsubscribe ?? null,
          wikiScanProviderId: current[workspaceId]?.wikiScanProviderId ?? wikiScanProviderId,
          wikiScanModelId: current[workspaceId]?.wikiScanModelId ?? wikiScanModelId,
          sources,
          compileSummary,
        },
      }));
      if (selectedPageId) {
        const content = await workspaceWikiApi.getPage(selectedPageId);
        setWorkspaceTabWiki((current) => ({
          ...current,
          [workspaceId]: {
            pages: current[workspaceId]?.pages ?? pages,
            selectedPageId,
            pageContent: content,
            isLoadingPages: false,
            isLoadingPageContent: false,
          activeJob: current[workspaceId]?.activeJob ?? activeJob,
          wikiError: null,
          isStarting: current[workspaceId]?.isStarting ?? false,
          isCancelling: current[workspaceId]?.isCancelling ?? false,
           isDeleting: current[workspaceId]?.isDeleting ?? false,
           unsubscribe: current[workspaceId]?.unsubscribe ?? null,
           wikiScanProviderId: current[workspaceId]?.wikiScanProviderId ?? 0,
           wikiScanModelId: current[workspaceId]?.wikiScanModelId ?? 0,
           sources: current[workspaceId]?.sources ?? sources,
           compileSummary: current[workspaceId]?.compileSummary ?? compileSummary,
         },
       }));
       }
    } catch (error) {
      setWorkspaceTabWiki((current) => ({
        ...current,
        [workspaceId]: {
          pages: current[workspaceId]?.pages ?? [],
          selectedPageId: current[workspaceId]?.selectedPageId ?? null,
          pageContent: current[workspaceId]?.pageContent ?? null,
          isLoadingPages: false,
          isLoadingPageContent: false,
          activeJob: current[workspaceId]?.activeJob ?? null,
          wikiError: error instanceof Error ? error.message : '加载 wiki 失败',
          isStarting: current[workspaceId]?.isStarting ?? false,
          isCancelling: current[workspaceId]?.isCancelling ?? false,
          isDeleting: current[workspaceId]?.isDeleting ?? false,
          unsubscribe: current[workspaceId]?.unsubscribe ?? null,
          wikiScanProviderId: current[workspaceId]?.wikiScanProviderId ?? 0,
          wikiScanModelId: current[workspaceId]?.wikiScanModelId ?? 0,
          sources: current[workspaceId]?.sources ?? [],
          compileSummary: current[workspaceId]?.compileSummary ?? null,
        },
      }));
    }
  }, [workspaceTabWiki]);

  const handleChangeWikiScanModel = useCallback((workspaceId: string, providerId: number, modelId: number) => {
    setWorkspaceTabWiki((current) => ({
      ...current,
      [workspaceId]: {
        ...current[workspaceId],
        wikiScanProviderId: providerId,
        wikiScanModelId: modelId,
      },
    }));
    void configApi.saveAIWorkspaceConfig(workspaceId, {
      ...DEFAULT_AI_WORKSPACE_CONFIG,
      wikiScanProviderId: providerId,
      wikiScanModelId: modelId,
    }).catch((error) => {
      console.error('Failed to save wiki scan model config:', error);
    });
  }, []);

  const handleOpenPdfTab = useCallback(
    (item: { id: string; title: string; pdfPath: string | null; workspaceId?: string; documentId?: string; sourceKind?: 'workspace_document' | 'zotero_item'; itemType?: string; citeKey?: string }) => {
      openTab({
        id: item.id,
        title: item.title,
        pdfPath: item.pdfPath,
        type: 'document',
        workspaceId: item.workspaceId,
        documentId: item.documentId,
        sourceKind: item.sourceKind,
        itemType: item.itemType,
        citeKey: item.citeKey,
      });
    },
    [openTab],
  );

  const handleOpenWorkspaceTab = useCallback(
    async (workspaceId: string) => {
      const item = workspace.workspaces.find((entry) => entry.id === workspaceId);
      if (!item) {
        return;
      }
      await Promise.all([
        ensureWorkspaceTabDocuments(item.id),
        ensureWorkspaceTabWiki(item.id),
      ]);
      openTab({
        id: `workspace:${item.id}`,
        title: item.name,
        pdfPath: null,
        type: 'workspace',
        workspaceId: item.id,
      });
    },
    [ensureWorkspaceTabDocuments, ensureWorkspaceTabWiki, openTab, workspace.workspaces],
  );

  const handleImportZoteroItem = useCallback(async (item: { id: string; title: string; pdfPath: string; citeKey: string }) => {
    await workspace.importZoteroItem({
      id: item.id,
      key: item.citeKey,
      citeKey: item.citeKey,
      title: item.title,
      creators: '',
      year: '',
      itemType: 'journalArticle',
      libraryId: 0,
      collectionIds: [],
      attachmentCount: 1,
      hasPdf: true,
      pdfPath: item.pdfPath,
      rawId: item.id,
    });
  }, [workspace]);

  const applyLLMTemplate = (template: ProviderTemplate) => setLlmForm(createProviderFromTemplate(template));
  const applyDrawingTemplate = (template: ProviderTemplate) => setDrawingForm(createProviderFromTemplate(template));
  const applyTranslationTemplate = (template: ProviderTemplate) =>
    setTranslationForm(createProviderFromTemplate(template));

  const handleSaveLLMProvider = async () => {
    try {
      const savedProvider = await saveProvider(llmForm);
      setLlmForm(EMPTY_LLM_FORM);
      setModelForm((current) => ({ ...current, providerId: current.providerId || savedProvider.id }));
    } catch {}
  };

  const handleSaveDrawingProvider = async () => {
    try {
      await saveProvider(drawingForm);
      setDrawingForm(EMPTY_DRAWING_FORM);
    } catch {}
  };

  const handleSaveTranslationProvider = async () => {
    if (selectedTranslationTemplate?.id === 'deeplx' && !isDeepLXTranslateEndpoint(translationForm.baseUrl)) {
      useConfigStore.setState({
        error: 'DeepLX 需要填写完整 URL，并包含最后的 /translate，例如 https://your-host/v1/translate',
      });
      return;
    }

    try {
      await saveProvider(translationForm);
      setTranslationForm(EMPTY_TRANSLATION_FORM);
    } catch {}
  };

  const handleModelIDChange = (value: string) => {
    const matchedPreset = findModelPreset(value);
    setModelForm((current) => ({
      ...current,
      modelId: value,
      contextWindow: current.contextWindow > 0 || !matchedPreset ? current.contextWindow : matchedPreset.contextWindow,
    }));
  };

  const handleSaveModel = async () => {
    try {
      await saveModel(modelForm);
      setModelForm((current) => ({ ...current, modelId: '', contextWindow: 0 }));
    } catch {}
  };

  const handleQuickAddModelPreset = async (preset: ModelPreset) => {
    if (!selectedModelProvider || selectedModelProviderExistingIds.has(normalizeModelKey(preset.id))) {
      return;
    }
    try {
      await saveModel({ providerId: selectedModelProvider.id, modelId: preset.id, contextWindow: preset.contextWindow });
    } catch {}
  };

  const fetchProviderModels = useCallback(async (providerId: number) => {
    setIsDiscoveringModels(true);
    setDiscoveryError(null);
    try {
      const response = await configApi.fetchProviderModels(providerId);
      setDiscoveredModels(response.models);
    } catch (fetchError) {
      setDiscoveryError(fetchError instanceof Error ? fetchError.message : '拉取模型失败');
    } finally {
      setIsDiscoveringModels(false);
    }
  }, []);

  const handleOpenModelDiscovery = (providerId: number) => {
    setDiscoveryProviderId(providerId);
    setDiscoveredModels([]);
    setDiscoveryError(null);
    setModelForm((current) => ({ ...current, providerId }));
    void fetchProviderModels(providerId);
  };

  const handleCloseModelDiscovery = () => {
    setDiscoveryProviderId(null);
    setDiscoveredModels([]);
    setDiscoveryError(null);
    setIsDiscoveringModels(false);
  };

  const handleCloseConfig = () => {
    setConfigOpen(false);
    handleCloseModelDiscovery();
  };

  const handleApplyDiscoveredModels = async (modelIds: string[]) => {
    if (!discoveryProviderConfig || modelIds.length === 0) {
      return;
    }
    const existingModelIdSet = new Set(
      discoveryProviderConfig.models.map((item) => normalizeModelKey(item.modelId)),
    );
    try {
      for (const modelId of modelIds) {
        if (existingModelIdSet.has(normalizeModelKey(modelId))) {
          continue;
        }
        await saveModel({
          providerId: discoveryProviderConfig.provider.id,
          modelId,
          contextWindow: suggestContextWindow(modelId),
        });
      }
      handleCloseModelDiscovery();
    } catch {}
  };

  const handleDeleteModel = async (model: ModelRecord) => {
    if (deletingModelId === model.id) {
      return;
    }

    const shouldDelete =
      typeof window === 'undefined' ? true : window.confirm(`确认删除模型“${model.modelId}”？`);
    if (!shouldDelete) {
      return;
    }

    setDeletingModelId(model.id);
    try {
      await deleteModel(model.id);
      setModelForm((current) => {
        if (current.providerId !== model.providerId) {
          return current;
        }
        if (normalizeModelKey(current.modelId) !== normalizeModelKey(model.modelId)) {
          return current;
        }
        return { ...current, modelId: '', contextWindow: 0 };
      });
    } catch {
    } finally {
      setDeletingModelId((current) => (current === model.id ? null : current));
    }
  };

  const handleSelectRuntimePackage = async () => {
    try {
      const selectedPath = await configApi.selectPDFTranslateRuntimePackage();
      if (selectedPath.trim()) {
        setRuntimeImportPath(selectedPath.trim());
      }
    } catch (error) {
      useConfigStore.setState({
        error: getErrorMessage(error, '选择 PDF 翻译运行时安装包失败'),
      });
    }
  };

  const handleImportRuntime = async () => {
    if (!runtimeImportPath.trim()) {
      useConfigStore.setState({ error: '请先选择 PDF 翻译运行时安装包。' });
      return;
    }
    setRuntimeImportProgress({
      stage: 'preparing',
      message: '正在准备运行时安装包',
      progress: 0.02,
      bytesCompleted: 0,
      bytesTotal: 0,
    });
    try {
      await importPDFTranslateRuntime(runtimeImportPath.trim());
      setRuntimeImportPath('');
    } catch {}
  };

  const handleRemoveRuntime = async () => {
    const shouldRemove =
      typeof window === 'undefined' ? true : window.confirm('确认移除当前 PDF 翻译运行时？');
    if (!shouldRemove) {
      return;
    }
    try {
      await removePDFTranslateRuntime();
    } catch {}
  };

  return (
    <div className="app-shell">
      <div className="app-titlebar">
        <div className="app-titlebar-brand">
          <BookOpen size={18} />
          <span>OpenSciReader</span>
        </div>
        <div className="app-titlebar-actions">
          <Button variant="ghost" size="icon-sm" onClick={() => setConfigOpen(true)}>
            <Settings2 size={16} />
          </Button>
        </div>
      </div>

      <div className="tab-bar">
        <button
          type="button"
          className={`tab-item tab-home ${activeTabId === 'home' ? 'tab-item-active' : ''}`}
          onClick={() => setActiveTab('home')}
        >
          <Home size={14} className="tab-item-icon" />
          <span className="tab-item-title">首页</span>
        </button>
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            className={`tab-item ${activeTabId === tab.id ? 'tab-item-active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            <FileText size={14} className="tab-item-icon" />
            <span className="tab-item-title">{tab.title}</span>
            <span
              className="tab-item-close"
              role="button"
              tabIndex={0}
              onClick={(event) => {
                event.stopPropagation();
                closeTab(tab.id);
              }}
            >
              <X size={12} />
            </span>
          </button>
        ))}
      </div>

      {error || zotero.error || workspace.error ? (
        <div className="banner-error">
          <span>{error ?? workspace.error ?? zotero.error}</span>
          <button
            type="button"
            onClick={() => {
              clearError();
              workspace.clearError();
              zotero.clearError();
            }}
          >
            关闭
          </button>
        </div>
      ) : null}

      <div className="app-content">
        {activeTabId === 'home' ? (
          <HomePage
            zotero={zotero}
            workspace={workspace}
            onImportFiles={async () => {
              const selected = await workspaceApi.selectLocalFiles?.();
              if (selected?.length) {
                await workspace.importFiles(selected);
              }
            }}
            onImportZoteroItem={handleImportZoteroItem}
            onOpenPdf={handleOpenPdfTab}
            onEnterWorkspace={async (workspaceId) => {
              await workspace.selectWorkspace(workspaceId);
              await handleOpenWorkspaceTab(workspaceId);
            }}
            onCreateWorkspace={async () => {
              const name = typeof window === 'undefined' ? '' : window.prompt('请输入工作区名称', 'New Workspace') ?? '';
              if (!name.trim()) {
                return;
              }
              try {
                const createdWorkspace = await workspace.createWorkspace({
                  name: name.trim(),
                  description: '',
                  color: '#6366f1',
                });
                await handleOpenWorkspaceTab(createdWorkspace.id);
              } catch {}
            }}
          />
        ) : (
          tabs
            .filter((tab) => tab.id === activeTabId)
            .map((tab) =>
              tab.type === 'workspace' ? (
                <WorkspaceTab
                  key={tab.id}
                  workspace={workspace.workspaces.find((item) => item.id === tab.workspaceId) ?? null}
                  documents={tab.workspaceId ? workspaceTabDocuments[tab.workspaceId]?.documents ?? [] : []}
                  isImporting={workspace.isImporting && tab.workspaceId === workspace.activeWorkspaceId}
                  isLoadingDocuments={tab.workspaceId ? workspaceTabDocuments[tab.workspaceId]?.isLoading ?? false : false}
                  deletingDocumentId={tab.workspaceId ? workspaceTabDocuments[tab.workspaceId]?.deletingDocumentId ?? null : null}
                  llmProviderConfigs={llmProviderConfigs}
                  wikiPages={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.pages ?? [] : []}
                  selectedWikiPageId={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.selectedPageId ?? null : null}
                  wikiPageContent={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.pageContent ?? null : null}
                  isLoadingWikiPages={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.isLoadingPages ?? false : false}
                  isLoadingWikiPageContent={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.isLoadingPageContent ?? false : false}
                  activeWikiJob={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.activeJob ?? null : null}
                  wikiError={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.wikiError ?? null : null}
                  isStartingWikiScan={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.isStarting ?? false : false}
                  isCancellingWikiScan={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.isCancelling ?? false : false}
                  isDeletingWikiPages={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.isDeleting ?? false : false}
                  wikiScanProviderId={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.wikiScanProviderId ?? 0 : 0}
                  wikiScanModelId={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.wikiScanModelId ?? 0 : 0}
                  wikiSources={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.sources ?? [] : []}
                  compileSummary={tab.workspaceId ? workspaceTabWiki[tab.workspaceId]?.compileSummary ?? null : null}
                  onImportFiles={async () => {
                    if (!tab.workspaceId) {
                      return;
                    }
                    await workspace.selectWorkspace(tab.workspaceId);
                    const selected = await workspaceApi.selectLocalFiles?.();
                    if (selected?.length) {
                      await workspace.importFiles(selected);
                      await ensureWorkspaceTabDocuments(tab.workspaceId);
                    }
                  }}
                  onRefreshDocuments={async () => {
                    if (!tab.workspaceId) {
                      return;
                    }
                    await ensureWorkspaceTabDocuments(tab.workspaceId);
                  }}
                  onOpenPdf={(document) => handleOpenPdfTab({
                    id: document.id,
                    title: document.title,
                    pdfPath: document.primaryPdfPath ?? null,
                    workspaceId: document.workspaceId,
                    documentId: document.id,
                    sourceKind: 'workspace_document',
                  })}
                  onDeleteDocument={async (document) => {
                    setWorkspaceTabDocuments((current) => ({
                      ...current,
                      [document.workspaceId]: {
                        documents: current[document.workspaceId]?.documents ?? [],
                        isLoading: current[document.workspaceId]?.isLoading ?? false,
                        deletingDocumentId: document.id,
                      },
                    }));
                    await workspace.selectWorkspace(document.workspaceId);
                    await workspace.deleteDocument(document.id);
                    await Promise.all([
                      ensureWorkspaceTabDocuments(document.workspaceId),
                      ensureWorkspaceTabWiki(document.workspaceId),
                    ]);
                    setWorkspaceTabDocuments((current) => ({
                      ...current,
                      [document.workspaceId]: {
                        documents: current[document.workspaceId]?.documents ?? [],
                        isLoading: current[document.workspaceId]?.isLoading ?? false,
                        deletingDocumentId: null,
                      },
                    }));
                  }}
                  onStartWikiScan={async (providerId: number, modelId: number) => {
                    if (!tab.workspaceId) {
                      return;
                    }
                    if (!providerId || !modelId) {
                      setWorkspaceTabWiki((current) => ({
                        ...current,
                        [tab.workspaceId!]: {
                          pages: current[tab.workspaceId!]?.pages ?? [],
                          selectedPageId: current[tab.workspaceId!]?.selectedPageId ?? null,
                          pageContent: current[tab.workspaceId!]?.pageContent ?? null,
                          isLoadingPages: false,
                          isLoadingPageContent: false,
                          activeJob: current[tab.workspaceId!]?.activeJob ?? null,
                          wikiError: '请先为当前工作区选择扫描模型',
                          isStarting: false,
                          isCancelling: false,
                          isDeleting: false,
                           unsubscribe: current[tab.workspaceId!]?.unsubscribe ?? null,
                           wikiScanProviderId: current[tab.workspaceId!]?.wikiScanProviderId ?? 0,
                           wikiScanModelId: current[tab.workspaceId!]?.wikiScanModelId ?? 0,
                           sources: current[tab.workspaceId!]?.sources ?? [],
                           compileSummary: current[tab.workspaceId!]?.compileSummary ?? null,
                         },
                       }));
                      return;
                    }
                    setWorkspaceTabWiki((current) => ({
                      ...current,
                      [tab.workspaceId!]: {
                        pages: current[tab.workspaceId!]?.pages ?? [],
                        selectedPageId: current[tab.workspaceId!]?.selectedPageId ?? null,
                        pageContent: current[tab.workspaceId!]?.pageContent ?? null,
                        isLoadingPages: false,
                        isLoadingPageContent: false,
                        activeJob: current[tab.workspaceId!]?.activeJob ?? null,
                        wikiError: null,
                        isStarting: true,
                        isCancelling: current[tab.workspaceId!]?.isCancelling ?? false,
                        isDeleting: current[tab.workspaceId!]?.isDeleting ?? false,
                        unsubscribe: current[tab.workspaceId!]?.unsubscribe ?? null,
                        wikiScanProviderId: current[tab.workspaceId!]?.wikiScanProviderId ?? 0,
                        wikiScanModelId: current[tab.workspaceId!]?.wikiScanModelId ?? 0,
                        sources: current[tab.workspaceId!]?.sources ?? [],
                        compileSummary: current[tab.workspaceId!]?.compileSummary ?? null,
                      },
                    }));
                    const job = await workspaceWikiApi.start({ workspaceId: tab.workspaceId, providerId, modelId });
                    const unsubscribe = workspaceWikiApi.subscribe(job.jobId, (event) => {
                      setWorkspaceTabWiki((current) => ({
                        ...current,
                        [tab.workspaceId!]: {
                          pages: current[tab.workspaceId!]?.pages ?? [],
                          selectedPageId: current[tab.workspaceId!]?.selectedPageId ?? null,
                          pageContent: current[tab.workspaceId!]?.pageContent ?? null,
                          isLoadingPages: false,
                          isLoadingPageContent: false,
                          activeJob: event.status,
                          wikiError: event.error ?? null,
                          isStarting: false,
                          isCancelling: false,
                          isDeleting: current[tab.workspaceId!]?.isDeleting ?? false,
                           unsubscribe: current[tab.workspaceId!]?.unsubscribe ?? unsubscribe,
                           wikiScanProviderId: current[tab.workspaceId!]?.wikiScanProviderId ?? 0,
                           wikiScanModelId: current[tab.workspaceId!]?.wikiScanModelId ?? 0,
                           sources: current[tab.workspaceId!]?.sources ?? [],
                           compileSummary: current[tab.workspaceId!]?.compileSummary ?? null,
                         },
                       }));
                      if (event.status.status === 'completed' || event.status.status === 'failed' || event.status.status === 'cancelled') {
                        unsubscribe();
                        void ensureWorkspaceTabWiki(tab.workspaceId!);
                      }
                    });
                    setWorkspaceTabWiki((current) => ({
                      ...current,
                      [tab.workspaceId!]: {
                        pages: current[tab.workspaceId!]?.pages ?? [],
                        selectedPageId: current[tab.workspaceId!]?.selectedPageId ?? null,
                        pageContent: current[tab.workspaceId!]?.pageContent ?? null,
                        isLoadingPages: false,
                        isLoadingPageContent: false,
                        activeJob: job,
                        wikiError: null,
                        isStarting: false,
                        isCancelling: false,
                        isDeleting: current[tab.workspaceId!]?.isDeleting ?? false,
                        unsubscribe,
                        wikiScanProviderId: current[tab.workspaceId!]?.wikiScanProviderId ?? 0,
                        wikiScanModelId: current[tab.workspaceId!]?.wikiScanModelId ?? 0,
                        sources: current[tab.workspaceId!]?.sources ?? [],
                        compileSummary: current[tab.workspaceId!]?.compileSummary ?? null,
                      },
                    }));
                  }}
                  onCancelWikiScan={async () => {
                    if (!tab.workspaceId) {
                      return;
                    }
                    const job = workspaceTabWiki[tab.workspaceId]?.activeJob;
                    if (!job) {
                      return;
                    }
                    setWorkspaceTabWiki((current) => ({
                      ...current,
                      [tab.workspaceId!]: {
                        ...current[tab.workspaceId!],
                        isCancelling: true,
                      },
                    }));
                    await workspaceWikiApi.cancel(job.jobId);
                    await ensureWorkspaceTabWiki(tab.workspaceId);
                  }}
                  onRefreshWikiPages={async () => {
                    if (!tab.workspaceId) {
                      return;
                    }
                    await ensureWorkspaceTabWiki(tab.workspaceId);
                  }}
                  onSelectWikiPage={async (pageId) => {
                    if (!tab.workspaceId) {
                      return;
                    }
                    setWorkspaceTabWiki((current) => ({
                      ...current,
                      [tab.workspaceId!]: {
                        ...current[tab.workspaceId!],
                        selectedPageId: pageId,
                        isLoadingPageContent: true,
                        wikiError: null,
                      },
                    }));
                    try {
                      const content = await workspaceWikiApi.getPage(pageId);
                      setWorkspaceTabWiki((current) => ({
                        ...current,
                        [tab.workspaceId!]: {
                          ...current[tab.workspaceId!],
                          selectedPageId: pageId,
                          pageContent: content,
                          isLoadingPageContent: false,
                        },
                      }));
                    } catch (error) {
                      setWorkspaceTabWiki((current) => ({
                        ...current,
                        [tab.workspaceId!]: {
                          ...current[tab.workspaceId!],
                          isLoadingPageContent: false,
                          wikiError: error instanceof Error ? error.message : '加载 wiki 页面失败',
                        },
                      }));
                    }
                  }}
                  onDeleteWikiPages={async () => {
                    if (!tab.workspaceId) {
                      return;
                    }
                    setWorkspaceTabWiki((current) => ({
                      ...current,
                      [tab.workspaceId!]: {
                        ...current[tab.workspaceId!],
                        isDeleting: true,
                        wikiError: null,
                      },
                    }));
                    await workspaceWikiApi.deletePages(tab.workspaceId);
                    await ensureWorkspaceTabWiki(tab.workspaceId);
                    setWorkspaceTabWiki((current) => ({
                      ...current,
                      [tab.workspaceId!]: {
                        ...current[tab.workspaceId!],
                        isDeleting: false,
                      },
                    }));
                  }}
                  onChangeWikiScanModel={(providerId: number, modelId: number) => {
                    if (tab.workspaceId) {
                      handleChangeWikiScanModel(tab.workspaceId, providerId, modelId);
                    }
                  }}
                />
              ) : (
                <ReaderTab key={tab.id} tab={tab} providerConfigs={providerConfigs} />
              ),
            )
        )}
      </div>

      {configOpen ? (
        <div className="modal-backdrop" role="presentation" onClick={handleCloseConfig}>
          <section className="modal" role="dialog" aria-modal="true" onClick={(event) => event.stopPropagation()}>
            <div className="modal-header">
              <div>
                <p className="panel-kicker">Configuration</p>
                <h2>多渠道配置</h2>
              </div>
              <Button variant="ghost" onClick={handleCloseConfig}>
                关闭
              </Button>
            </div>

            <div className="modal-layout">
              <LLMConfigCard
                isLoading={isLoading}
                llmForm={llmForm}
                llmTemplates={llmTemplates}
                llmProviderConfigs={llmProviderConfigs}
                modelForm={modelForm}
                selectedModelProvider={selectedModelProvider}
                selectedModelProviderId={selectedModelProviderId}
                selectedModelProviderModels={selectedModelProviderModels}
                selectedModelProviderTemplate={selectedModelProviderTemplate}
                selectedModelProviderRecommendedModels={selectedModelProviderRecommendedModels}
                selectedModelProviderExistingIds={selectedModelProviderExistingIds}
                deletingModelId={deletingModelId}
                onApplyTemplate={applyLLMTemplate}
                onChangeLLMForm={setLlmForm}
                onSaveLLMProvider={handleSaveLLMProvider}
                onDeleteProvider={deleteProvider}
                onOpenModelDiscovery={handleOpenModelDiscovery}
                onChangeModelForm={setModelForm}
                onChangeModelId={handleModelIDChange}
                onSaveModel={handleSaveModel}
                onQuickAddPreset={handleQuickAddModelPreset}
                onDeleteModel={handleDeleteModel}
              />

              <div className="card-stack">
                <PDFTranslateRuntimeCard
                  runtime={snapshot.pdfTranslateRuntime}
                  runtimeImportPath={runtimeImportPath}
                  runtimeImportProgress={runtimeImportProgress}
                  runtimeAction={runtimeAction}
                  onSelectRuntimePackage={handleSelectRuntimePackage}
                  onImportRuntime={handleImportRuntime}
                  onRemoveRuntime={handleRemoveRuntime}
                />
                {/*
                <ApiConfigCard
                  title="OCR API"
                  description={
                    <>
                      OCR 设置像翻译 API 一样独立管理，不参与 LLM 模型配置。GLM OCR 默认使用官方
                      <code> layout_parsing </code>
                      端点，模型固定为
                      <code> glm-ocr </code>。
                    </>
                  }
                  templates={ocrTemplates}
                  form={ocrForm}
                  providers={ocrProviderConfigs}
                  saveLabel="保存 OCR API"
                  baseUrlPlaceholder={PROVIDER_BASE_URL_HINTS.ocr}
                  onApplyTemplate={applyOCRTemplate}
                  onChangeForm={setOcrForm}
                  onSaveProvider={handleSaveOCRProvider}
                  onDeleteProvider={deleteProvider}
                />
                */}
                <ApiConfigCard
                  title="Drawing API"
                  description="科研绘图单独走这里的接口配置，不复用上面的 LLM Provider。当前推荐直接配置 Gemini Image。"
                  templates={drawingTemplates}
                  form={drawingForm}
                  providers={drawingProviderConfigs}
                  saveLabel="保存 Drawing API"
                  baseUrlPlaceholder={PROVIDER_BASE_URL_HINTS.drawing}
                  onApplyTemplate={applyDrawingTemplate}
                  onChangeForm={setDrawingForm}
                  onSaveProvider={handleSaveDrawingProvider}
                  onDeleteProvider={deleteProvider}
                />

                <ApiConfigCard
                  title="翻译渠道"
                  description="普通翻译渠道只配置接口地址与密钥，不配置模型。排版翻译仍然使用上面的 LLM Provider / Model。"
                  templates={translationTemplates}
                  form={translationForm}
                  providers={translationProviderConfigs}
                  saveLabel="保存翻译渠道"
                  baseUrlPlaceholder={
                    selectedTranslationTemplate?.id === 'deeplx'
                      ? 'https://your-deeplx.example.com/v1/translate'
                      : PROVIDER_BASE_URL_HINTS.translate
                  }
                  baseUrlHelpText={
                    selectedTranslationTemplate?.id === 'deeplx'
                      ? 'DeepLX 必须填写完整 endpoint，包含最后的 /translate，例如 .../translate、.../v1/translate 或 .../v2/translate。'
                      : undefined
                  }
                  regionPlaceholder="如 Microsoft Translator 需要填写"
                  showRegion
                  onApplyTemplate={applyTranslationTemplate}
                  onChangeForm={setTranslationForm}
                  onSaveProvider={handleSaveTranslationProvider}
                  onDeleteProvider={deleteProvider}
                />
              </div>
            </div>
          </section>
        </div>
      ) : null}

      <ModelDiscoveryModal
        open={Boolean(discoveryProviderConfig)}
        providerName={discoveryProviderConfig?.provider.name ?? ''}
        existingModelIds={discoveryProviderConfig?.models.map((item) => item.modelId) ?? []}
        models={discoveredModels}
        isLoading={isDiscoveringModels}
        error={discoveryError}
        onFetch={() => discoveryProviderId && void fetchProviderModels(discoveryProviderId)}
        onClose={handleCloseModelDiscovery}
        onApply={(modelIds) => void handleApplyDiscoveredModels(modelIds)}
      />
    </div>
  );
}

interface PDFTranslateRuntimeCardProps {
  runtime: PDFTranslateRuntimeConfig;
  runtimeImportPath: string;
  runtimeImportProgress: PDFTranslateRuntimeImportProgress | null;
  runtimeAction: 'idle' | 'importing' | 'removing';
  onSelectRuntimePackage: () => void;
  onImportRuntime: () => void;
  onRemoveRuntime: () => void;
}

function PDFTranslateRuntimeCard({
  runtime,
  runtimeImportPath,
  runtimeImportProgress,
  runtimeAction,
  onSelectRuntimePackage,
  onImportRuntime,
  onRemoveRuntime,
}: PDFTranslateRuntimeCardProps) {
  const statusMeta = getPDFTranslateRuntimeStatusMeta(runtime.status);
  const importProgressPercent = getRuntimeImportPercent(runtimeImportProgress);
  const showImportProgress = runtimeAction === 'importing' && runtimeImportProgress !== null;

  return (
    <section className="card">
      <div className="section-header">
        <h3>PDF 翻译运行时</h3>
        <span className={`inline-badge ${statusMeta.tone === 'danger' ? 'runtime-inline-badge-danger' : statusMeta.tone === 'success' ? 'runtime-inline-badge-success' : 'inline-badge-muted'}`}>
          {statusMeta.label}
        </span>
      </div>
      <p className="empty-inline">
        主安装包不再内置 BabelDOC / pdf2zh 运行时。先导入单独下载的运行时压缩包，再启用保留格式翻译。
      </p>

      <div className="runtime-summary-grid">
        <div className="context-card">
          <strong>{runtime.version || '未安装'}</strong>
          <p>版本</p>
        </div>
        <div className="context-card">
          <strong>{runtime.platform || '--'}</strong>
          <p>平台</p>
        </div>
      </div>

      {runtime.lastValidationError ? (
        <div className="reader-error">{runtime.lastValidationError}</div>
      ) : null}

      <label className="field">
        <span>运行时安装包</span>
        <Button variant="outline" onClick={() => void onSelectRuntimePackage()} disabled={runtimeAction !== 'idle'}>
          选择 ZIP 安装包
        </Button>
      </label>
      <p className="field-hint">{runtimeImportPath || '请选择从 release 下载的运行时 zip 包。Wails 桌面环境会把文件路径传给后端。'}</p>

      {showImportProgress ? (
        <div className="translate-progress">
          <div className="translate-progress-bar">
            <div
              className={`translate-progress-fill ${importProgressPercent <= 0 ? 'translate-progress-fill-indeterminate' : ''}`}
              style={{ width: `${Math.max(importProgressPercent <= 0 ? 28 : 4, Math.min(100, importProgressPercent))}%` }}
            />
          </div>
          <div className="translate-progress-meta">
            <span>{runtimeImportProgress!.message}</span>
            <span>{importProgressPercent.toFixed(1)}%</span>
            {runtimeImportProgress!.bytesTotal > 0 ? (
              <span>{`${formatByteCount(runtimeImportProgress!.bytesCompleted)} / ${formatByteCount(runtimeImportProgress!.bytesTotal)}`}</span>
            ) : null}
          </div>
        </div>
      ) : null}
      <div className="row-actions">
        <Button onClick={() => void onImportRuntime()} disabled={runtimeAction !== 'idle' || !runtimeImportPath.trim()}>
          {runtimeAction === 'importing' ? '导入中...' : '导入运行时'}
        </Button>
        <Button
          variant="secondary"
          onClick={() => void onRemoveRuntime()}
          disabled={runtimeAction !== 'idle' || !runtime.installed}
        >
          {runtimeAction === 'removing' ? '移除中...' : '移除运行时'}
        </Button>
      </div>

      <div className="provider-summary">
        <strong>当前状态</strong>
        <p>{statusMeta.description}</p>
        <small className="mono-inline">{runtime.runtimeDir || '尚未安装运行时目录'}</small>
        {runtime.sourceFileName ? <small>来源：{runtime.sourceFileName}</small> : null}
        {runtime.installedAt ? <small>安装时间：{new Date(runtime.installedAt).toLocaleString()}</small> : null}
      </div>
    </section>
  );
}

function getPDFTranslateRuntimeStatusMeta(status: PDFTranslateRuntimeConfig['status']) {
  switch (status) {
    case 'valid':
      return {
        label: '可用',
        tone: 'success' as const,
        description: '运行时已安装，可以直接启动保留格式翻译。',
      };
    case 'invalid':
      return {
        label: '异常',
        tone: 'danger' as const,
        description: '运行时目录存在，但校验失败。请重新导入正确的安装包。',
      };
    case 'installing':
      return {
        label: '安装中',
        tone: 'neutral' as const,
        description: '正在处理运行时包，完成后会自动切换到可用状态。',
      };
    default:
      return {
        label: '未安装',
        tone: 'neutral' as const,
        description: '当前还没有导入 PDF 翻译运行时，保留格式翻译会被禁用。',
      };
  }
}

function getRuntimeImportPercent(progress: PDFTranslateRuntimeImportProgress | null) {
  if (!progress) {
    return 0;
  }
  return Math.max(0, Math.min(100, Math.round(progress.progress * 1000) / 10));
}

function formatByteCount(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return '0 B';
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = value;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }

  const digits = size >= 100 || unitIndex === 0 ? 0 : size >= 10 ? 1 : 2;
  return `${size.toFixed(digits)} ${units[unitIndex]}`;
}

interface LLMConfigCardProps {
  isLoading: boolean;
  llmForm: ProviderUpsertInput;
  llmTemplates: ProviderTemplate[];
  llmProviderConfigs: ProviderConfig[];
  modelForm: ModelUpsertInput;
  selectedModelProvider: ProviderRecord | null;
  selectedModelProviderId: number;
  selectedModelProviderModels: ProviderConfig['models'];
  selectedModelProviderTemplate: ProviderTemplate | null;
  selectedModelProviderRecommendedModels: ModelPreset[];
  selectedModelProviderExistingIds: ReadonlySet<string>;
  deletingModelId: number | null;
  onApplyTemplate: (template: ProviderTemplate) => void;
  onChangeLLMForm: Dispatch<SetStateAction<ProviderUpsertInput>>;
  onSaveLLMProvider: () => void;
  onDeleteProvider: (id: number) => Promise<void>;
  onOpenModelDiscovery: (providerId: number) => void;
  onChangeModelForm: Dispatch<SetStateAction<ModelUpsertInput>>;
  onChangeModelId: (value: string) => void;
  onSaveModel: () => void;
  onQuickAddPreset: (preset: ModelPreset) => void;
  onDeleteModel: (model: ModelRecord) => Promise<void>;
}

function LLMConfigCard({
  isLoading,
  llmForm,
  llmTemplates,
  llmProviderConfigs,
  modelForm,
  selectedModelProvider,
  selectedModelProviderId,
  selectedModelProviderModels,
  selectedModelProviderTemplate,
  selectedModelProviderRecommendedModels,
  selectedModelProviderExistingIds,
  deletingModelId,
  onApplyTemplate,
  onChangeLLMForm,
  onSaveLLMProvider,
  onDeleteProvider,
  onOpenModelDiscovery,
  onChangeModelForm,
  onChangeModelId,
  onSaveModel,
  onQuickAddPreset,
  onDeleteModel,
}: LLMConfigCardProps) {
  return (
    <section className="card">
      <div className="section-header">
        <h3>LLM 渠道</h3>
      </div>
      <p className="empty-inline">
        参考 ai-toolbox 的流程，先保存 LLM provider，再补推荐模型或直接从 provider 的
        <code> /models </code>
        拉取模型列表。
      </p>

      <div className="template-group">
        <strong>LLM 模板</strong>
        <div className="preset-grid">
          {llmTemplates.map((template) => (
            <button
              key={template.id}
              type="button"
              className="preset-button"
              onClick={() => onApplyTemplate(template)}
              title={template.description}
            >
              {template.name}
            </button>
          ))}
        </div>
      </div>

      <label className="field">
        <span>名称</span>
        <input value={llmForm.name} onChange={(event) => onChangeLLMForm((current) => ({ ...current, name: event.target.value }))} />
      </label>
      <label className="field">
        <span>接口地址</span>
        <input
          value={llmForm.baseUrl}
          placeholder={PROVIDER_BASE_URL_HINTS.llm}
          onChange={(event) => onChangeLLMForm((current) => ({ ...current, baseUrl: event.target.value }))}
        />
      </label>
      <label className="field">
        <span>API 密钥</span>
        <input
          type="password"
          value={llmForm.apiKey}
          onChange={(event) =>
            onChangeLLMForm((current) => ({ ...current, apiKey: event.target.value, clearApiKey: false }))
          }
        />
      </label>
      <Button onClick={() => void onSaveLLMProvider()} disabled={isLoading || llmForm.name.trim() === ''}>
        <Plus size={16} />
        保存 LLM 渠道
      </Button>

      <div className="provider-list">
        {llmProviderConfigs.map((item) => {
          const matchedTemplate = findMatchingProviderTemplate(item.provider);
          return (
            <article key={item.provider.id} className="provider-card">
              <div className="provider-card-header">
                <div>
                  <h4>{item.provider.name}</h4>
                  <p>{matchedTemplate?.description ?? '自定义 OpenAI-compatible LLM provider。'}</p>
                </div>
                <div className="row-actions">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onOpenModelDiscovery(item.provider.id)}
                    disabled={item.provider.baseUrl.trim() === ''}
                  >
                    <Wand2 size={14} />
                    发现模型
                  </Button>
                  <button
                    type="button"
                    className="text-button danger"
                    onClick={() => void onDeleteProvider(item.provider.id)}
                  >
                    <Trash2 size={12} />
                  </button>
                </div>
              </div>
              <div className="meta-row">
                <span className="badge badge-count">{item.models.length} models</span>
                <small className="mono-inline">{item.provider.baseUrl}</small>
              </div>
              {item.models.length > 0 ? (
                <ul className="model-list">
                  {item.models.map((model) => (
                    <li key={model.id}>
                      <span>{model.modelId}</span>
                      <button
                        type="button"
                        className="text-button danger"
                        disabled={isLoading || deletingModelId === model.id}
                        onClick={() => void onDeleteModel(model)}
                      >
                        删除
                      </button>
                      <small>{model.contextWindow ? model.contextWindow.toLocaleString() : '未设置'}</small>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="empty-inline">当前还没有配置模型。</p>
              )}
            </article>
          );
        })}
      </div>

      <div className="model-form">
        <h4>模型配置</h4>
        <label className="field">
          <span>渠道</span>
          <select
            value={modelForm.providerId || ''}
            onChange={(event) =>
              onChangeModelForm((current) => ({ ...current, providerId: Number(event.target.value) }))
            }
          >
            <option value="">请选择</option>
            {llmProviderConfigs.map((item) => (
              <option key={item.provider.id} value={item.provider.id}>
                {item.provider.name}
              </option>
            ))}
          </select>
        </label>

        {selectedModelProvider ? (
          <div className="provider-summary">
            <strong>{selectedModelProvider.name}</strong>
            <p>{selectedModelProviderTemplate?.description ?? '可手动填写模型，也可在线发现。'}</p>
            <small className="mono-inline">{selectedModelProvider.baseUrl}</small>
          </div>
        ) : (
          <p className="empty-inline">先选择一个已保存的 LLM provider。</p>
        )}

        {selectedModelProviderRecommendedModels.length > 0 ? (
          <div className="model-preset-grid">
            {selectedModelProviderRecommendedModels.map((preset) => {
              const alreadyAdded = selectedModelProviderExistingIds.has(normalizeModelKey(preset.id));
              return (
                <article key={preset.id} className="model-preset-card">
                  <div className="model-preset-head">
                    <strong>{preset.label}</strong>
                    {alreadyAdded ? <span className="inline-badge inline-badge-muted">已添加</span> : null}
                  </div>
                  <p>{preset.description}</p>
                  <div className="meta-row">
                    <small>{preset.id}</small>
                    <small>上下文 {preset.contextWindow.toLocaleString()}</small>
                  </div>
                  <div className="chip-row">
                    {preset.tags.map((tag) => (
                      <span key={tag} className="inline-badge">
                        {tag}
                      </span>
                    ))}
                  </div>
                  <div className="row-actions">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() =>
                        onChangeModelForm((current) => ({
                          ...current,
                          providerId: selectedModelProviderId,
                          modelId: preset.id,
                          contextWindow: preset.contextWindow,
                        }))
                      }
                    >
                      填入表单
                    </Button>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => void onQuickAddPreset(preset)}
                      disabled={isLoading || alreadyAdded}
                    >
                      <Plus size={14} />
                      直接添加
                    </Button>
                  </div>
                </article>
              );
            })}
          </div>
        ) : null}

        <div className="model-toolbar">
          <p className="empty-inline">
            {selectedModelProvider
              ? '保存 provider 后可直接从 provider 的 /models 拉取模型列表。'
              : '选择 provider 后，可用在线发现代替手填 modelId。'}
          </p>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => selectedModelProvider && onOpenModelDiscovery(selectedModelProvider.id)}
            disabled={!selectedModelProvider}
          >
            <Sparkles size={14} />
            在线发现模型
          </Button>
        </div>

        <label className="field">
          <span>模型 ID</span>
          <input value={modelForm.modelId} onChange={(event) => onChangeModelId(event.target.value)} />
        </label>
        <label className="field">
          <span>上下文窗口</span>
          <input
            type="number"
            min="0"
            value={modelForm.contextWindow}
            onChange={(event) =>
              onChangeModelForm((current) => ({ ...current, contextWindow: Number(event.target.value) || 0 }))
            }
          />
        </label>
        <Button
          variant="secondary"
          onClick={() => void onSaveModel()}
          disabled={isLoading || !modelForm.providerId || modelForm.modelId.trim() === ''}
        >
          保存模型
        </Button>

        {selectedModelProviderModels.length > 0 ? (
          <ul className="model-list">
            {selectedModelProviderModels.map((model) => (
              <li key={model.id}>
                <span>{model.modelId}</span>
                <div className="row-actions">
                  <small>{model.contextWindow ? model.contextWindow.toLocaleString() : '未设置'}</small>
                  <button
                    type="button"
                    className="text-button danger"
                    disabled={isLoading || deletingModelId === model.id}
                    onClick={() => void onDeleteModel(model)}
                  >
                    删除
                  </button>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <p className="empty-inline">当前 provider 还没有配置模型。</p>
        )}
      </div>
    </section>
  );
}

interface ApiConfigCardProps {
  title: string;
  description: ReactNode;
  templates: ProviderTemplate[];
  form: ProviderUpsertInput;
  providers: ProviderConfig[];
  saveLabel: string;
  baseUrlPlaceholder: string;
  baseUrlHelpText?: ReactNode;
  regionPlaceholder?: string;
  showRegion?: boolean;
  onApplyTemplate: (template: ProviderTemplate) => void;
  onChangeForm: Dispatch<SetStateAction<ProviderUpsertInput>>;
  onSaveProvider: () => void;
  onDeleteProvider: (id: number) => Promise<void>;
}

function ApiConfigCard({
  title,
  description,
  templates,
  form,
  providers,
  saveLabel,
  baseUrlPlaceholder,
  baseUrlHelpText,
  regionPlaceholder,
  showRegion = false,
  onApplyTemplate,
  onChangeForm,
  onSaveProvider,
  onDeleteProvider,
}: ApiConfigCardProps) {
  const defaultDescription = title === 'OCR API' ? 'GLM OCR layout_parsing endpoint.' : `自定义${title}。`;

  return (
    <section className="card">
      <div className="section-header">
        <h3>{title}</h3>
      </div>
      <p className="empty-inline">{description}</p>

      <div className="template-group">
        <strong>{title} 模板</strong>
        <div className="preset-grid">
          {templates.map((template) => (
            <button
              key={template.id}
              type="button"
              className="preset-button"
              onClick={() => onApplyTemplate(template)}
              title={title === 'OCR API' ? 'GLM OCR layout_parsing endpoint.' : template.description}
            >
              {template.name}
            </button>
          ))}
        </div>
      </div>

      <label className="field">
        <span>名称</span>
        <input value={form.name} onChange={(event) => onChangeForm((current) => ({ ...current, name: event.target.value }))} />
      </label>
      <label className="field">
        <span>接口地址</span>
        <input
          value={form.baseUrl}
          placeholder={baseUrlPlaceholder}
          onChange={(event) => onChangeForm((current) => ({ ...current, baseUrl: event.target.value }))}
        />
      </label>
      {baseUrlHelpText ? <p className="field-hint">{baseUrlHelpText}</p> : null}
      {showRegion ? (
        <label className="field">
          <span>Region</span>
          <input
            value={form.region}
            placeholder={regionPlaceholder}
            onChange={(event) => onChangeForm((current) => ({ ...current, region: event.target.value }))}
          />
        </label>
      ) : null}
      <label className="field">
        <span>API 密钥</span>
        <input
          type="password"
          value={form.apiKey}
          onChange={(event) =>
            onChangeForm((current) => ({ ...current, apiKey: event.target.value, clearApiKey: false }))
          }
        />
      </label>
      <Button onClick={() => void onSaveProvider()} disabled={form.name.trim() === ''}>
        <Plus size={16} />
        {saveLabel}
      </Button>

      <div className="provider-list">
        {providers.map((item) => (
          <article key={item.provider.id} className="provider-card">
            <div className="provider-card-header">
              <div>
                <h4>{item.provider.name}</h4>
                <p>{title === 'OCR API' ? defaultDescription : findMatchingProviderTemplate(item.provider)?.description ?? defaultDescription}</p>
              </div>
              <button
                type="button"
                className="text-button danger"
                onClick={() => void onDeleteProvider(item.provider.id)}
              >
                <Trash2 size={12} />
              </button>
            </div>
            <div className="meta-row">
              <small className="mono-inline">{item.provider.baseUrl}</small>
              {showRegion && item.provider.region ? <small>region {item.provider.region}</small> : null}
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}
