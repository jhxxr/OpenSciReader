export namespace main {
	
	export class AIWorkspaceConfig {
	    summaryMode: string;
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
	
	    static createFrom(source: any = {}) {
	        return new AIWorkspaceConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summaryMode = source["summaryMode"];
	        this.summaryChunkPages = source["summaryChunkPages"];
	        this.summaryChunkMaxChars = source["summaryChunkMaxChars"];
	        this.autoRestoreCount = source["autoRestoreCount"];
	        this.tableTemplate = source["tableTemplate"];
	        this.tablePrompt = source["tablePrompt"];
	        this.customPromptDraft = source["customPromptDraft"];
	        this.followUpPromptDraft = source["followUpPromptDraft"];
	        this.drawingPromptDraft = source["drawingPromptDraft"];
	        this.drawingProviderId = source["drawingProviderId"];
	        this.drawingModel = source["drawingModel"];
	    }
	}
	export class ChatHistoryEntry {
	    id: number;
	    itemId: string;
	    itemTitle: string;
	    page: number;
	    kind: string;
	    prompt: string;
	    response: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatHistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.itemId = source["itemId"];
	        this.itemTitle = source["itemTitle"];
	        this.page = source["page"];
	        this.kind = source["kind"];
	        this.prompt = source["prompt"];
	        this.response = source["response"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class CollectionTree {
	    id: string;
	    name: string;
	    libraryId: number;
	    library: string;
	    parentId: string;
	    path: string;
	    children: CollectionTree[];
	
	    static createFrom(source: any = {}) {
	        return new CollectionTree(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.libraryId = source["libraryId"];
	        this.library = source["library"];
	        this.parentId = source["parentId"];
	        this.path = source["path"];
	        this.children = this.convertValues(source["children"], CollectionTree);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PDFTranslateRuntimeConfig {
	    installed: boolean;
	    status: string;
	    runtimeId: string;
	    version: string;
	    platform: string;
	    runtimeDir: string;
	    pythonPath: string;
	    manifestPath: string;
	    installedAt: string;
	    sourceFileName: string;
	    lastValidationError: string;
	
	    static createFrom(source: any = {}) {
	        return new PDFTranslateRuntimeConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.status = source["status"];
	        this.runtimeId = source["runtimeId"];
	        this.version = source["version"];
	        this.platform = source["platform"];
	        this.runtimeDir = source["runtimeDir"];
	        this.pythonPath = source["pythonPath"];
	        this.manifestPath = source["manifestPath"];
	        this.installedAt = source["installedAt"];
	        this.sourceFileName = source["sourceFileName"];
	        this.lastValidationError = source["lastValidationError"];
	    }
	}
	export class ModelRecord {
	    id: number;
	    providerId: number;
	    modelId: string;
	    contextWindow: number;
	
	    static createFrom(source: any = {}) {
	        return new ModelRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.providerId = source["providerId"];
	        this.modelId = source["modelId"];
	        this.contextWindow = source["contextWindow"];
	    }
	}
	export class ProviderRecord {
	    id: number;
	    name: string;
	    type: string;
	    baseUrl: string;
	    region: string;
	    hasApiKey: boolean;
	    apiKeyMasked: string;
	    isActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.baseUrl = source["baseUrl"];
	        this.region = source["region"];
	        this.hasApiKey = source["hasApiKey"];
	        this.apiKeyMasked = source["apiKeyMasked"];
	        this.isActive = source["isActive"];
	    }
	}
	export class ProviderConfig {
	    provider: ProviderRecord;
	    models: ModelRecord[];
	
	    static createFrom(source: any = {}) {
	        return new ProviderConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = this.convertValues(source["provider"], ProviderRecord);
	        this.models = this.convertValues(source["models"], ModelRecord);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConfigSnapshot {
	    providers: ProviderConfig[];
	    pdfTranslateRuntime: PDFTranslateRuntimeConfig;
	
	    static createFrom(source: any = {}) {
	        return new ConfigSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providers = this.convertValues(source["providers"], ProviderConfig);
	        this.pdfTranslateRuntime = this.convertValues(source["pdfTranslateRuntime"], PDFTranslateRuntimeConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DiscoveredModel {
	    id: string;
	    name: string;
	    ownedBy: string;
	
	    static createFrom(source: any = {}) {
	        return new DiscoveredModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.ownedBy = source["ownedBy"];
	    }
	}
	export class DiscoveredModelsResponse {
	    models: DiscoveredModel[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new DiscoveredModelsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.models = this.convertValues(source["models"], DiscoveredModel);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FigureGenerationResult {
	    mimeType: string;
	    dataUrl: string;
	    prompt: string;
	
	    static createFrom(source: any = {}) {
	        return new FigureGenerationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mimeType = source["mimeType"];
	        this.dataUrl = source["dataUrl"];
	        this.prompt = source["prompt"];
	    }
	}
	export class GatewayContextData {
	    selection: string;
	    snapshot: string;
	    page: number;
	    itemTitle: string;
	    citeKey: string;
	
	    static createFrom(source: any = {}) {
	        return new GatewayContextData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.selection = source["selection"];
	        this.snapshot = source["snapshot"];
	        this.page = source["page"];
	        this.itemTitle = source["itemTitle"];
	        this.citeKey = source["citeKey"];
	    }
	}
	
	export class ModelUpsertInput {
	    id: number;
	    providerId: number;
	    modelId: string;
	    contextWindow: number;
	
	    static createFrom(source: any = {}) {
	        return new ModelUpsertInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.providerId = source["providerId"];
	        this.modelId = source["modelId"];
	        this.contextWindow = source["contextWindow"];
	    }
	}
	export class PDFDocumentPayload {
	    path: string;
	    dataBase64: string;
	
	    static createFrom(source: any = {}) {
	        return new PDFDocumentPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.dataBase64 = source["dataBase64"];
	    }
	}
	export class PDFMarkdownSection {
	    index: number;
	    title: string;
	    level: number;
	    startPage: number;
	    endPage: number;
	    text: string;
	    characters: number;
	
	    static createFrom(source: any = {}) {
	        return new PDFMarkdownSection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.title = source["title"];
	        this.level = source["level"];
	        this.startPage = source["startPage"];
	        this.endPage = source["endPage"];
	        this.text = source["text"];
	        this.characters = source["characters"];
	    }
	}
	export class PDFMarkdownPayload {
	    pdfPath: string;
	    source: string;
	    markdown: string;
	    sections: PDFMarkdownSection[];
	    totalChars: number;
	    cached: boolean;
	    generatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new PDFMarkdownPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pdfPath = source["pdfPath"];
	        this.source = source["source"];
	        this.markdown = source["markdown"];
	        this.sections = this.convertValues(source["sections"], PDFMarkdownSection);
	        this.totalChars = source["totalChars"];
	        this.cached = source["cached"];
	        this.generatedAt = source["generatedAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class PDFTranslateRuntimeImportResult {
	    runtime: PDFTranslateRuntimeConfig;
	
	    static createFrom(source: any = {}) {
	        return new PDFTranslateRuntimeImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runtime = this.convertValues(source["runtime"], PDFTranslateRuntimeConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PDFTranslateStartInput {
	    pdfPath: string;
	    pageCount: number;
	    itemId: string;
	    itemTitle: string;
	    sourceLang: string;
	    targetLang: string;
	    mode: string;
	    previewChunkPages: number;
	    maxPagesPerPart: number;
	    qps: number;
	    poolMaxWorkers: number;
	    termPoolMaxWorkers: number;
	    retryJobId: string;
	    reusePreviewJobId: string;
	    llmProviderId: number;
	    llmModelId: number;
	
	    static createFrom(source: any = {}) {
	        return new PDFTranslateStartInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pdfPath = source["pdfPath"];
	        this.pageCount = source["pageCount"];
	        this.itemId = source["itemId"];
	        this.itemTitle = source["itemTitle"];
	        this.sourceLang = source["sourceLang"];
	        this.targetLang = source["targetLang"];
	        this.mode = source["mode"];
	        this.previewChunkPages = source["previewChunkPages"];
	        this.maxPagesPerPart = source["maxPagesPerPart"];
	        this.qps = source["qps"];
	        this.poolMaxWorkers = source["poolMaxWorkers"];
	        this.termPoolMaxWorkers = source["termPoolMaxWorkers"];
	        this.retryJobId = source["retryJobId"];
	        this.reusePreviewJobId = source["reusePreviewJobId"];
	        this.llmProviderId = source["llmProviderId"];
	        this.llmModelId = source["llmModelId"];
	    }
	}
	
	
	export class ProviderUpsertInput {
	    id: number;
	    name: string;
	    type: string;
	    baseUrl: string;
	    region: string;
	    apiKey: string;
	    clearApiKey: boolean;
	    isActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderUpsertInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.baseUrl = source["baseUrl"];
	        this.region = source["region"];
	        this.apiKey = source["apiKey"];
	        this.clearApiKey = source["clearApiKey"];
	        this.isActive = source["isActive"];
	    }
	}
	export class ReaderNoteEntry {
	    id: number;
	    itemId: string;
	    itemTitle: string;
	    page: number;
	    anchorText: string;
	    content: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ReaderNoteEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.itemId = source["itemId"];
	        this.itemTitle = source["itemTitle"];
	        this.page = source["page"];
	        this.anchorText = source["anchorText"];
	        this.content = source["content"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class ZoteroItem {
	    id: string;
	    key: string;
	    citeKey: string;
	    title: string;
	    creators: string;
	    year: string;
	    itemType: string;
	    libraryId: number;
	    collectionIds: string[];
	    attachmentCount: number;
	    hasPdf: boolean;
	    pdfPath: string;
	    rawId: string;
	
	    static createFrom(source: any = {}) {
	        return new ZoteroItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.key = source["key"];
	        this.citeKey = source["citeKey"];
	        this.title = source["title"];
	        this.creators = source["creators"];
	        this.year = source["year"];
	        this.itemType = source["itemType"];
	        this.libraryId = source["libraryId"];
	        this.collectionIds = source["collectionIds"];
	        this.attachmentCount = source["attachmentCount"];
	        this.hasPdf = source["hasPdf"];
	        this.pdfPath = source["pdfPath"];
	        this.rawId = source["rawId"];
	    }
	}

}

export namespace translator {
	
	export class ChunkStatus {
	    index: number;
	    startPage: number;
	    endPage: number;
	    status: string;
	    translatedPdfPath?: string;
	    dualPdfPath?: string;
	    translatedPageOffset?: number;
	    startedAt?: string;
	    finishedAt?: string;
	    totalSeconds?: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ChunkStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.startPage = source["startPage"];
	        this.endPage = source["endPage"];
	        this.status = source["status"];
	        this.translatedPdfPath = source["translatedPdfPath"];
	        this.dualPdfPath = source["dualPdfPath"];
	        this.translatedPageOffset = source["translatedPageOffset"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	        this.totalSeconds = source["totalSeconds"];
	        this.error = source["error"];
	    }
	}
	export class JobOutputs {
	    originalPdfPath?: string;
	    monoPdfPath?: string;
	    mixedPdfPath?: string;
	    dualPdfPath?: string;
	    noWatermarkMonoPdfPath?: string;
	    noWatermarkMixedPdfPath?: string;
	    noWatermarkDualPdfPath?: string;
	    autoExtractedGlossaryPath?: string;
	    totalSeconds?: number;
	    peakMemoryUsage?: number;
	
	    static createFrom(source: any = {}) {
	        return new JobOutputs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalPdfPath = source["originalPdfPath"];
	        this.monoPdfPath = source["monoPdfPath"];
	        this.mixedPdfPath = source["mixedPdfPath"];
	        this.dualPdfPath = source["dualPdfPath"];
	        this.noWatermarkMonoPdfPath = source["noWatermarkMonoPdfPath"];
	        this.noWatermarkMixedPdfPath = source["noWatermarkMixedPdfPath"];
	        this.noWatermarkDualPdfPath = source["noWatermarkDualPdfPath"];
	        this.autoExtractedGlossaryPath = source["autoExtractedGlossaryPath"];
	        this.totalSeconds = source["totalSeconds"];
	        this.peakMemoryUsage = source["peakMemoryUsage"];
	    }
	}
	export class JobSnapshot {
	    jobId: string;
	    retryOfJobId?: string;
	    mode: string;
	    status: string;
	    itemId?: string;
	    itemTitle?: string;
	    pdfPath: string;
	    localPdfPath: string;
	    pageCount: number;
	    sourceLang: string;
	    targetLang: string;
	    previewChunkPages: number;
	    maxPagesPerPart: number;
	    qps: number;
	    poolMaxWorkers: number;
	    termPoolMaxWorkers: number;
	    providerId: number;
	    providerName: string;
	    modelId: string;
	    createdAt: string;
	    updatedAt: string;
	    startedAt?: string;
	    finishedAt?: string;
	    currentStage?: string;
	    overallProgress?: number;
	    stageProgress?: number;
	    stageCurrent?: number;
	    stageTotal?: number;
	    partIndex?: number;
	    totalParts?: number;
	    error?: string;
	    outputs: JobOutputs;
	    chunks: ChunkStatus[];
	
	    static createFrom(source: any = {}) {
	        return new JobSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	        this.retryOfJobId = source["retryOfJobId"];
	        this.mode = source["mode"];
	        this.status = source["status"];
	        this.itemId = source["itemId"];
	        this.itemTitle = source["itemTitle"];
	        this.pdfPath = source["pdfPath"];
	        this.localPdfPath = source["localPdfPath"];
	        this.pageCount = source["pageCount"];
	        this.sourceLang = source["sourceLang"];
	        this.targetLang = source["targetLang"];
	        this.previewChunkPages = source["previewChunkPages"];
	        this.maxPagesPerPart = source["maxPagesPerPart"];
	        this.qps = source["qps"];
	        this.poolMaxWorkers = source["poolMaxWorkers"];
	        this.termPoolMaxWorkers = source["termPoolMaxWorkers"];
	        this.providerId = source["providerId"];
	        this.providerName = source["providerName"];
	        this.modelId = source["modelId"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	        this.currentStage = source["currentStage"];
	        this.overallProgress = source["overallProgress"];
	        this.stageProgress = source["stageProgress"];
	        this.stageCurrent = source["stageCurrent"];
	        this.stageTotal = source["stageTotal"];
	        this.partIndex = source["partIndex"];
	        this.totalParts = source["totalParts"];
	        this.error = source["error"];
	        this.outputs = this.convertValues(source["outputs"], JobOutputs);
	        this.chunks = this.convertValues(source["chunks"], ChunkStatus);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

