# Graph Report - .  (2026-04-18)

## Corpus Check
- 102 files ¡¤ ~97,416 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1006 nodes ¡¤ 1725 edges ¡¤ 71 communities detected
- Extraction: 80% EXTRACTED ¡¤ 19% INFERRED ¡¤ 0% AMBIGUOUS ¡¤ INFERRED: 332 edges (avg confidence: 0.8)
- Token cost: 0 input ¡¤ 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_PDF Translate Jobs|PDF Translate Jobs]]
- [[_COMMUNITY_Gateway Translation Service|Gateway Translation Service]]
- [[_COMMUNITY_Wails Model Bindings|Wails Model Bindings]]
- [[_COMMUNITY_App UI Orchestration|App UI Orchestration]]
- [[_COMMUNITY_Wails Runtime Bridge|Wails Runtime Bridge]]
- [[_COMMUNITY_Config Store Persistence|Config Store Persistence]]
- [[_COMMUNITY_PDF Runtime Management|PDF Runtime Management]]
- [[_COMMUNITY_Zotero Integration|Zotero Integration]]
- [[_COMMUNITY_Docs And Logo Graph|Docs And Logo Graph]]
- [[_COMMUNITY_Python Worker Pipeline|Python Worker Pipeline]]
- [[_COMMUNITY_PDF Figure Extraction|PDF Figure Extraction]]
- [[_COMMUNITY_Reader AI Workflows|Reader AI Workflows]]
- [[_COMMUNITY_Dual PDF Reader|Dual PDF Reader]]
- [[_COMMUNITY_Shared Config Types|Shared Config Types]]
- [[_COMMUNITY_PDF Loading Service|PDF Loading Service]]
- [[_COMMUNITY_AI Catalog And Config UI|AI Catalog And Config UI]]
- [[_COMMUNITY_Translate Job Types|Translate Job Types]]
- [[_COMMUNITY_Translation Status Presentation|Translation Status Presentation]]
- [[_COMMUNITY_OCR Gateway|OCR Gateway]]
- [[_COMMUNITY_Worker Process Types|Worker Process Types]]
- [[_COMMUNITY_Desktop App Entrypoints|Desktop App Entrypoints]]
- [[_COMMUNITY_PDF Translate Frontend API|PDF Translate Frontend API]]
- [[_COMMUNITY_Frontend Config Store|Frontend Config Store]]
- [[_COMMUNITY_Provider Model Discovery|Provider Model Discovery]]
- [[_COMMUNITY_Third Party Licenses|Third Party Licenses]]
- [[_COMMUNITY_Frontend Zotero API|Frontend Zotero API]]
- [[_COMMUNITY_Export Filename Logic|Export Filename Logic]]
- [[_COMMUNITY_Typst Runtime Docs|Typst Runtime Docs]]
- [[_COMMUNITY_Translation Performance Presets|Translation Performance Presets]]
- [[_COMMUNITY_PDF Translate State Types|PDF Translate State Types]]
- [[_COMMUNITY_Third Party Notices|Third Party Notices]]
- [[_COMMUNITY_Common JSON IO|Common JSON IO]]
- [[_COMMUNITY_PDF Document Types|PDF Document Types]]
- [[_COMMUNITY_Zotero Types|Zotero Types]]
- [[_COMMUNITY_Frontend Gateway API|Frontend Gateway API]]
- [[_COMMUNITY_History API|History API]]
- [[_COMMUNITY_Notes API|Notes API]]
- [[_COMMUNITY_PDF API|PDF API]]
- [[_COMMUNITY_Phase6 API|Phase6 API]]
- [[_COMMUNITY_Workspace API|Workspace API]]
- [[_COMMUNITY_Model Discovery Modal|Model Discovery Modal]]
- [[_COMMUNITY_Font License Notice|Font License Notice]]
- [[_COMMUNITY_Bundled Runtime Docs|Bundled Runtime Docs]]
- [[_COMMUNITY_Release Assets Docs|Release Assets Docs]]
- [[_COMMUNITY_Crop Icon Script|Crop Icon Script]]
- [[_COMMUNITY_Manual Icon Crop Script|Manual Icon Crop Script]]
- [[_COMMUNITY_Rounded Corner Script|Rounded Corner Script]]
- [[_COMMUNITY_Runtime Dialog Actions|Runtime Dialog Actions]]
- [[_COMMUNITY_UI Button|UI Button]]
- [[_COMMUNITY_UI Utils|UI Utils]]
- [[_COMMUNITY_Non Windows Worker Config|Non Windows Worker Config]]
- [[_COMMUNITY_Runtime Dialog|Runtime Dialog]]
- [[_COMMUNITY_PostCSS Config|PostCSS Config]]
- [[_COMMUNITY_Vite Config|Vite Config]]
- [[_COMMUNITY_Frontend Entry|Frontend Entry]]
- [[_COMMUNITY_Vite Env Types|Vite Env Types]]
- [[_COMMUNITY_Markdown Preview|Markdown Preview]]
- [[_COMMUNITY_Reader Store|Reader Store]]
- [[_COMMUNITY_Tab Store|Tab Store]]
- [[_COMMUNITY_Workspace Store|Workspace Store]]
- [[_COMMUNITY_Zotero Store|Zotero Store]]
- [[_COMMUNITY_Config Types TS|Config Types TS]]
- [[_COMMUNITY_Drawing Types TS|Drawing Types TS]]
- [[_COMMUNITY_Gateway Types TS|Gateway Types TS]]
- [[_COMMUNITY_History Types TS|History Types TS]]
- [[_COMMUNITY_Notes Types TS|Notes Types TS]]
- [[_COMMUNITY_PDF Types TS|PDF Types TS]]
- [[_COMMUNITY_Workspace Types TS|Workspace Types TS]]
- [[_COMMUNITY_Zotero Types TS|Zotero Types TS]]
- [[_COMMUNITY_App JS Bindings|App JS Bindings]]
- [[_COMMUNITY_Runtime JS Bindings|Runtime JS Bindings]]

## God Nodes (most connected - your core abstractions)
1. `configStore` - 44 edges
2. `App` - 33 edges
3. `Manager` - 21 edges
4. `newConfigStore()` - 19 edges
5. `gatewayService` - 18 edges
6. `jobRuntime` - 18 edges
7. `nowRFC3339()` - 17 edges
8. `TestManualPreviewExportReuseForWang2017()` - 16 edges
9. `cloneSnapshot()` - 13 edges
10. `SaveProvider()` - 12 edges

## Surprising Connections (you probably didn't know these)
- `OpenSciReader Logo` --semantically_similar_to--> `OpenSciReader`  [INFERRED] [semantically similar]
  assets/logo.png ¡ú README.md
- `OpenSciReader Logo` --semantically_similar_to--> `OpenSciReader Logo`  [INFERRED] [semantically similar]
  graphify-out/GRAPH_REPORT.md ¡ú assets/logo.png
- `handleSaveLLMProvider()` --calls--> `SaveProvider()`  [INFERRED]
  frontend\src\App.tsx ¡ú frontend\wailsjs\go\main\App.js
- `handleSaveDrawingProvider()` --calls--> `SaveProvider()`  [INFERRED]
  frontend\src\App.tsx ¡ú frontend\wailsjs\go\main\App.js
- `OpenSciReader Logo` --conceptually_related_to--> `OpenSciReader`  [INFERRED]
  frontend/src/assets/images/logo-universal.png ¡ú README.md

## Hyperedges (group relationships)
- **OpenSciReader Logo Composition** ¡ª logo_universal_openscireader_logo, logo_universal_open_book, logo_universal_magnifying_glass [INFERRED 0.74]
- **OpenSciReader Translation Stack** ¡ª readme_openscireader, readme_local_pdf_layout_translation_pipeline, readme_private_python_worker_contract [EXTRACTED 1.00]
- **Bundled Translation Runtime Packaging** ¡ª readme_windows_installer, readme_python_worker, readme_standalone_runtime_package [EXTRACTED 1.00]
- **OpenSciReader Logo Composition** ¡ª logo_openscireader_logo, logo_open_book, logo_magnifying_glass [INFERRED 0.74]

## Communities

### Community 0 - "PDF Translate Jobs"
Cohesion: 0.05
Nodes (47): sanitizeFileName(), handleCancelJob(), handleDeleteJob(), handleRefreshJobs(), loadJobs(), buildChunks(), canReusePreviewForExport(), cloneSnapshot() (+39 more)

### Community 1 - "Gateway Translation Service"
Cohesion: 0.06
Nodes (55): GetConfigSnapshot(), SaveProvider(), serveRemotePDFFile(), newConfigStore(), encryptLegacyStringForTest(), TestAIWorkspaceConfigPersistence(), TestBootstrapMigratesLegacyEncryptedProviderSecrets(), TestBootstrapPurgesDeprecatedOCRCache() (+47 more)

### Community 2 - "Wails Model Bindings"
Cohesion: 0.03
Nodes (24): AIWorkspaceConfig, ChatHistoryEntry, ChunkStatus, CollectionTree, ConfigSnapshot, DiscoveredModel, DiscoveredModelsResponse, FigureGenerationResult (+16 more)

### Community 3 - "App UI Orchestration"
Cohesion: 0.04
Nodes (43): createProviderFromTemplate(), App, applyDrawingTemplate(), applyLLMTemplate(), applyTranslationTemplate(), DeleteChatHistory(), DeleteModel(), DeleteProvider() (+35 more)

### Community 4 - "Wails Runtime Bridge"
Cohesion: 0.03
Nodes (3): EventsOn(), EventsOnce(), EventsOnMultiple()

### Community 5 - "Config Store Persistence"
Cohesion: 0.06
Nodes (22): async(), decryptLegacyString(), loadExistingKey(), maskAPIKey(), newSecretManager(), secretManager, boolToInt(), configStore (+14 more)

### Community 6 - "PDF Runtime Management"
Cohesion: 0.08
Nodes (38): emitGatewayEvent(), NewManager(), cloneAppPathsForTest(), copyFileForTest(), firstActiveLLMProviderAndModel(), manualIntegrationPDFPath(), TestManualPreviewExportReuseForWang2017(), waitForTerminalJob() (+30 more)

### Community 7 - "Zotero Integration"
Cohesion: 0.09
Nodes (31): appendQuery(), buildCollectionChildren(), buildCollectionTree(), decodeFileURL(), defaultLibraryName(), extractYear(), fetchPaginatedJSON(), firstNonEmpty() (+23 more)

### Community 8 - "Docs And Logo Graph"
Cohesion: 0.06
Nodes (43): Ambiguous Edges Review, Communities, Community Hubs Navigation, Logo Asset Community, Project Documentation Community, God Nodes, Graph Report, Hyperedges (+35 more)

### Community 9 - "Python Worker Pipeline"
Cohesion: 0.13
Nodes (35): build_nested_payload(), build_settings(), emit(), finalize_export_translate_result(), find_settings_loader(), find_settings_model(), get_field(), get_model_annotation() (+27 more)

### Community 10 - "PDF Figure Extraction"
Cohesion: 0.11
Nodes (30): destroyPDFDocument(), normalizePDFError(), openPDFDocument(), buildCaptionSideRect(), buildTextLineItem(), buildTextLines(), clampRect(), collectImageStats() (+22 more)

### Community 11 - "Reader AI Workflows"
Cohesion: 0.11
Nodes (25): ExtractPDFMarkdown(), SaveChatHistory(), buildPaperImageGenerationPrompt(), buildPaperImageSummaryInstruction(), buildMarkdownChunks(), buildMarkdownContext(), loadAllPageTexts(), loadPDFMarkdown() (+17 more)

### Community 12 - "Dual PDF Reader"
Cohesion: 0.11
Nodes (23): handleViewerKeyDown(), handleViewerWheel(), renderOriginalPDF(), resetZoom(), scrollByViewport(), updateScale(), zoomBy(), buildReaderOutline() (+15 more)

### Community 13 - "Shared Config Types"
Cohesion: 0.08
Nodes (24): AIWorkspaceConfig, ChatHistoryEntry, ConfigSnapshot, DiscoveredModel, DiscoveredModelsResponse, DocumentAssetRecord, DocumentExternalLink, DocumentRecord (+16 more)

### Community 14 - "PDF Loading Service"
Cohesion: 0.13
Nodes (12): appPaths, resolveAppPaths(), newGatewayService(), extractPDFMarkdownWithPython(), markItDownScriptResult, newPDFService(), parseMarkdownHeading(), pdfService (+4 more)

### Community 15 - "AI Catalog And Config UI"
Cohesion: 0.18
Nodes (13): findMatchingProviderTemplate(), findModelPreset(), getRecommendedModelsForProvider(), normalize(), suggestContextWindow(), handleModelIDChange(), buildMockDiscoveredModels(), createMockAIWorkspaceConfig() (+5 more)

### Community 16 - "Translate Job Types"
Cohesion: 0.13
Nodes (12): StartPDFTranslate(), normalizeOpenAICompatibleBaseURL(), PDFTranslateStartInput, ChunkStatus, JobEvent, JobOutputs, JobSnapshot, JobStatus (+4 more)

### Community 17 - "Translation Status Presentation"
Cohesion: 0.24
Nodes (12): getChunkCaption(), formatRelativeTimestamp(), getChunkStatusLabel(), getJobLiveHint(), getJobPrimaryStatus(), getJobProgressPercent(), getJobSecondaryStatus(), getJobStageLabel() (+4 more)

### Community 18 - "OCR Gateway"
Cohesion: 0.31
Nodes (7): extractDataURLPayload(), clampUnit(), decodeOCRImageSize(), gatewayService, glmOCREndpoint(), normalizeOCRFileInput(), parseGLMOCRBlocks()

### Community 19 - "Worker Process Types"
Cohesion: 0.22
Nodes (8): pythonCommand, workerEvent, workerProcessError, workerRequest, workerTranslateResult, compactWorkerErrorMessage(), formatWorkerEventError(), TestCompactWorkerErrorMessage()

### Community 20 - "Desktop App Entrypoints"
Cohesion: 0.29
Nodes (5): NewApp(), newLocalAssetHandler(), normalizeLocalPDFRequestPath(), serveLocalPDFFile(), main()

### Community 21 - "PDF Translate Frontend API"
Cohesion: 0.32
Nodes (3): getWailsApp(), isWailsApp(), isWailsDesktop()

### Community 22 - "Frontend Config Store"
Cohesion: 0.25
Nodes (0): 

### Community 23 - "Provider Model Discovery"
Cohesion: 0.48
Nodes (6): buildProviderModelsEndpoint(), dedupeAndSortDiscoveredModels(), isGoogleModelDiscoveryProvider(), parseDiscoveredModels(), parseGoogleModels(), parseOpenAICompatibleModels()

### Community 24 - "Third Party Licenses"
Cohesion: 0.29
Nodes (7): Apache License 2.0, bzip2, Guido van Rossum, libffi, Microsoft Distributable Code, Python, Python Software Foundation License Version 2

### Community 25 - "Frontend Zotero API"
Cohesion: 0.47
Nodes (3): createMockApp(), getApp(), isWailsApp()

### Community 26 - "Export Filename Logic"
Cohesion: 0.47
Nodes (4): exportOutputFileStem(), sanitizeExportOutputStem(), TestExportOutputFileStemFallsBackToPDFBaseName(), TestExportOutputFileStemUsesItemTitleAndLanguages()

### Community 27 - "Typst Runtime Docs"
Cohesion: 0.33
Nodes (6): Typst Design Principles Rationale, Incremental Compilation, LaTeX, NLnet, Typst, Typst CLI

### Community 28 - "Translation Performance Presets"
Cohesion: 0.6
Nodes (3): getPDFTranslatePerformancePreset(), includesAny(), normalizeText()

### Community 29 - "PDF Translate State Types"
Cohesion: 0.7
Nodes (3): applyPDFTranslateEvent(), normalizePDFTranslateSnapshot(), normalizeProgressValue()

### Community 30 - "Third Party Notices"
Cohesion: 0.4
Nodes (5): AGPL-3.0 License, document.v1.json, MIT License, Release Compliance Rationale, Third-Party Notices

### Community 31 - "Common JSON IO"
Cohesion: 0.5
Nodes (0): 

### Community 32 - "PDF Document Types"
Cohesion: 0.5
Nodes (3): PDFDocumentPayload, PDFMarkdownPayload, PDFMarkdownSection

### Community 33 - "Zotero Types"
Cohesion: 0.67
Nodes (2): CollectionTree, ZoteroItem

### Community 34 - "Frontend Gateway API"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 35 - "History API"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 36 - "Notes API"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 37 - "PDF API"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 38 - "Phase6 API"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 39 - "Workspace API"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 40 - "Model Discovery Modal"
Cohesion: 0.67
Nodes (0): 

### Community 41 - "Font License Notice"
Cohesion: 0.67
Nodes (3): Font Software, The Nunito Project Authors, SIL Open Font License 1.1

### Community 42 - "Bundled Runtime Docs"
Cohesion: 0.67
Nodes (3): BabelDOC, Bundled PDF Translation Runtime, pdf2zh_next

### Community 43 - "Release Assets Docs"
Cohesion: 0.67
Nodes (3): GitHub Actions Release Packaging, Slim Installer Asset, Standalone Runtime Asset

### Community 44 - "Crop Icon Script"
Cohesion: 1.0
Nodes (0): 

### Community 45 - "Manual Icon Crop Script"
Cohesion: 1.0
Nodes (0): 

### Community 46 - "Rounded Corner Script"
Cohesion: 1.0
Nodes (0): 

### Community 47 - "Runtime Dialog Actions"
Cohesion: 1.0
Nodes (1): App

### Community 48 - "UI Button"
Cohesion: 1.0
Nodes (0): 

### Community 49 - "UI Utils"
Cohesion: 1.0
Nodes (0): 

### Community 50 - "Non Windows Worker Config"
Cohesion: 1.0
Nodes (0): 

### Community 51 - "Runtime Dialog"
Cohesion: 1.0
Nodes (0): 

### Community 52 - "PostCSS Config"
Cohesion: 1.0
Nodes (0): 

### Community 53 - "Vite Config"
Cohesion: 1.0
Nodes (0): 

### Community 54 - "Frontend Entry"
Cohesion: 1.0
Nodes (0): 

### Community 55 - "Vite Env Types"
Cohesion: 1.0
Nodes (0): 

### Community 56 - "Markdown Preview"
Cohesion: 1.0
Nodes (0): 

### Community 57 - "Reader Store"
Cohesion: 1.0
Nodes (0): 

### Community 58 - "Tab Store"
Cohesion: 1.0
Nodes (0): 

### Community 59 - "Workspace Store"
Cohesion: 1.0
Nodes (0): 

### Community 60 - "Zotero Store"
Cohesion: 1.0
Nodes (0): 

### Community 61 - "Config Types TS"
Cohesion: 1.0
Nodes (0): 

### Community 62 - "Drawing Types TS"
Cohesion: 1.0
Nodes (0): 

### Community 63 - "Gateway Types TS"
Cohesion: 1.0
Nodes (0): 

### Community 64 - "History Types TS"
Cohesion: 1.0
Nodes (0): 

### Community 65 - "Notes Types TS"
Cohesion: 1.0
Nodes (0): 

### Community 66 - "PDF Types TS"
Cohesion: 1.0
Nodes (0): 

### Community 67 - "Workspace Types TS"
Cohesion: 1.0
Nodes (0): 

### Community 68 - "Zotero Types TS"
Cohesion: 1.0
Nodes (0): 

### Community 69 - "App JS Bindings"
Cohesion: 1.0
Nodes (0): 

### Community 70 - "Runtime JS Bindings"
Cohesion: 1.0
Nodes (0): 

## Ambiguous Edges - Review These
- `OpenSciReader Logo` ¡ú `Open Book`  [AMBIGUOUS]
  frontend/src/assets/images/logo-universal.png ¡¤ relation: form
- `OpenSciReader Logo` ¡ú `Magnifying Glass`  [AMBIGUOUS]
  frontend/src/assets/images/logo-universal.png ¡¤ relation: form
- `OpenSciReader Logo` ¡ú `PDF Document`  [AMBIGUOUS]
  frontend/src/assets/images/logo-universal.png ¡¤ relation: form
- `OpenSciReader Logo` ¡ú `Open Book`  [AMBIGUOUS]
  assets/logo.png ¡¤ relation: conceptually_related_to
- `OpenSciReader Logo` ¡ú `Magnifying Glass`  [AMBIGUOUS]
  assets/logo.png ¡¤ relation: conceptually_related_to
- `OpenSciReader Logo` ¡ú `PDF Document`  [AMBIGUOUS]
  assets/logo.png ¡¤ relation: conceptually_related_to

## Knowledge Gaps
- **110 isolated node(s):** `appPaths`, `providerSecretRecord`, `pdfMarkdownCacheRecord`, `PDFTranslateRuntimeStatus`, `ProviderRecord` (+105 more)
  These have ¡Ü1 connection - possible missing edges or undocumented components.
- **Thin community `Crop Icon Script`** (2 nodes): `crop_margins()`, `crop_icon.py`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Manual Icon Crop Script`** (2 nodes): `manual_center_crop()`, `crop_icon2.py`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Rounded Corner Script`** (2 nodes): `add_rounded_corners()`, `rounded_corners.py`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Runtime Dialog Actions`** (2 nodes): `App`, `.SelectPDFTranslateRuntimePackage()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `UI Button`** (2 nodes): `cn()`, `Button.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `UI Utils`** (2 nodes): `utils.ts`, `cn()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Non Windows Worker Config`** (2 nodes): `worker_nonwindows.go`, `configureWorkerProcess()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Runtime Dialog`** (1 nodes): `runtime_dialog.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `PostCSS Config`** (1 nodes): `postcss.config.js`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Vite Config`** (1 nodes): `vite.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Frontend Entry`** (1 nodes): `main.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Vite Env Types`** (1 nodes): `vite-env.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Markdown Preview`** (1 nodes): `MarkdownPreview.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Reader Store`** (1 nodes): `readerStore.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Tab Store`** (1 nodes): `tabStore.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Workspace Store`** (1 nodes): `workspaceStore.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Zotero Store`** (1 nodes): `zoteroStore.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Config Types TS`** (1 nodes): `config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Drawing Types TS`** (1 nodes): `drawing.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Gateway Types TS`** (1 nodes): `gateway.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `History Types TS`** (1 nodes): `history.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Notes Types TS`** (1 nodes): `notes.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `PDF Types TS`** (1 nodes): `pdf.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Workspace Types TS`** (1 nodes): `workspace.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Zotero Types TS`** (1 nodes): `zotero.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `App JS Bindings`** (1 nodes): `App.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Runtime JS Bindings`** (1 nodes): `runtime.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `OpenSciReader Logo` and `Open Book`?**
  _Edge tagged AMBIGUOUS (relation: form) - confidence is low._
- **What is the exact relationship between `OpenSciReader Logo` and `Magnifying Glass`?**
  _Edge tagged AMBIGUOUS (relation: form) - confidence is low._
- **What is the exact relationship between `OpenSciReader Logo` and `PDF Document`?**
  _Edge tagged AMBIGUOUS (relation: form) - confidence is low._
- **What is the exact relationship between `OpenSciReader Logo` and `Open Book`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `OpenSciReader Logo` and `Magnifying Glass`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `OpenSciReader Logo` and `PDF Document`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `EventsEmit()` connect `PDF Runtime Management` to `Wails Runtime Bridge`?**
  _High betweenness centrality (0.081) - this node is a cross-community bridge._