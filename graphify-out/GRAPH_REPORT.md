# Graph Report - .  (2026-04-18)

## Corpus Check
- 84 files ¡¤ ~67,105 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 900 nodes ¡¤ 1492 edges ¡¤ 63 communities detected
- Extraction: 81% EXTRACTED ¡¤ 19% INFERRED ¡¤ 0% AMBIGUOUS ¡¤ INFERRED: 277 edges (avg confidence: 0.8)
- Token cost: 0 input ¡¤ 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_PDF Translation Jobs|PDF Translation Jobs]]
- [[_COMMUNITY_AI Gateway Providers|AI Gateway Providers]]
- [[_COMMUNITY_Wails Data Models|Wails Data Models]]
- [[_COMMUNITY_Wails Runtime Bridge|Wails Runtime Bridge]]
- [[_COMMUNITY_Reader PDF Utilities|Reader PDF Utilities]]
- [[_COMMUNITY_Reader State Stores|Reader State Stores]]
- [[_COMMUNITY_Configuration Types|Configuration Types]]
- [[_COMMUNITY_PDF Service Backend|PDF Service Backend]]
- [[_COMMUNITY_Zotero Integration|Zotero Integration]]
- [[_COMMUNITY_Frontend PDF API|Frontend PDF API]]
- [[_COMMUNITY_Notes And History API|Notes And History API]]
- [[_COMMUNITY_Reader Tab UI|Reader Tab UI]]
- [[_COMMUNITY_Home Page UI|Home Page UI]]
- [[_COMMUNITY_AI Panel UI|AI Panel UI]]
- [[_COMMUNITY_Markdown And Figures|Markdown And Figures]]
- [[_COMMUNITY_Frontend Type Models|Frontend Type Models]]
- [[_COMMUNITY_Gateway Translation Tests|Gateway Translation Tests]]
- [[_COMMUNITY_Translator Runtime Tests|Translator Runtime Tests]]
- [[_COMMUNITY_OCR Gateway Flow|OCR Gateway Flow]]
- [[_COMMUNITY_Provider Model Discovery|Provider Model Discovery]]
- [[_COMMUNITY_PDF Runtime Paths|PDF Runtime Paths]]
- [[_COMMUNITY_Asset File Handling|Asset File Handling]]
- [[_COMMUNITY_Crypto And Secrets|Crypto And Secrets]]
- [[_COMMUNITY_Config Store Persistence|Config Store Persistence]]
- [[_COMMUNITY_App Entry Wiring|App Entry Wiring]]
- [[_COMMUNITY_Frontend App Shell|Frontend App Shell]]
- [[_COMMUNITY_PDF Source Handling|PDF Source Handling]]
- [[_COMMUNITY_PDF Document Parsing|PDF Document Parsing]]
- [[_COMMUNITY_PDF Text Extraction|PDF Text Extraction]]
- [[_COMMUNITY_PDF Outline Parsing|PDF Outline Parsing]]
- [[_COMMUNITY_PDF Figure Extraction|PDF Figure Extraction]]
- [[_COMMUNITY_Translation Presentation|Translation Presentation]]
- [[_COMMUNITY_Translation Performance|Translation Performance]]
- [[_COMMUNITY_Utility Helpers|Utility Helpers]]
- [[_COMMUNITY_Button Component|Button Component]]
- [[_COMMUNITY_Reader Layout Types|Reader Layout Types]]
- [[_COMMUNITY_Gateway Request Types|Gateway Request Types]]
- [[_COMMUNITY_Config API Types|Config API Types]]
- [[_COMMUNITY_History Types|History Types]]
- [[_COMMUNITY_Notes Types|Notes Types]]
- [[_COMMUNITY_PDF Types|PDF Types]]
- [[_COMMUNITY_Translation Types|Translation Types]]
- [[_COMMUNITY_Zotero Types|Zotero Types]]
- [[_COMMUNITY_AI Catalog|AI Catalog]]
- [[_COMMUNITY_Figure Prompting|Figure Prompting]]
- [[_COMMUNITY_Config Store Tests|Config Store Tests]]
- [[_COMMUNITY_Gateway Service Core|Gateway Service Core]]
- [[_COMMUNITY_Translator Manager Core|Translator Manager Core]]
- [[_COMMUNITY_Python Worker|Python Worker]]
- [[_COMMUNITY_Runtime Bundles|Runtime Bundles]]
- [[_COMMUNITY_Build Tooling|Build Tooling]]
- [[_COMMUNITY_Project Documentation|Project Documentation]]
- [[_COMMUNITY_Third Party Notices|Third Party Notices]]
- [[_COMMUNITY_Embedded Runtime Docs|Embedded Runtime Docs]]
- [[_COMMUNITY_Logo Asset|Logo Asset]]
- [[_COMMUNITY_Main App Service|Main App Service]]
- [[_COMMUNITY_PDF HTTP Handlers|PDF HTTP Handlers]]
- [[_COMMUNITY_Config Path Helpers|Config Path Helpers]]
- [[_COMMUNITY_Gateway Integration Tests|Gateway Integration Tests]]
- [[_COMMUNITY_Translator Storage|Translator Storage]]
- [[_COMMUNITY_Translator Types|Translator Types]]
- [[_COMMUNITY_Reader App Bootstrap|Reader App Bootstrap]]
- [[_COMMUNITY_OpenSciReader Report|OpenSciReader Report]]

## God Nodes (most connected - your core abstractions)
1. `configStore` - 32 edges
2. `App` - 26 edges
3. `Manager` - 21 edges
4. `jobRuntime` - 18 edges
5. `gatewayService` - 17 edges
6. `newConfigStore()` - 16 edges
7. `TestManualPreviewExportReuseForWang2017()` - 16 edges
8. `nowRFC3339()` - 16 edges
9. `Local PDF Layout Translation Pipeline` - 15 edges
10. `cloneSnapshot()` - 13 edges

## Surprising Connections (you probably didn't know these)
- `OpenSciReader` --semantically_similar_to--> `readme_openscireader`  [INFERRED] [semantically similar]
  README.md ¡ú graphify-out/GRAPH_REPORT.md
- `handleSaveLLMProvider()` --calls--> `SaveProvider()`  [INFERRED]
  frontend\src\App.tsx ¡ú frontend\wailsjs\go\main\App.js
- `handleSaveDrawingProvider()` --calls--> `SaveProvider()`  [INFERRED]
  frontend\src\App.tsx ¡ú frontend\wailsjs\go\main\App.js
- `handleBulkImportFigures()` --calls--> `extractFigureCandidates()`  [INFERRED]
  E:\0JHX\Project\OpenSciReader\frontend\src\components\ReaderTab.tsx ¡ú frontend\src\lib\pdfFigures.ts
- `OpenSciReader Logo` --conceptually_related_to--> `OpenSciReader`  [INFERRED]
  frontend/src/assets/images/logo-universal.png ¡ú README.md

## Hyperedges (group relationships)
- **OpenSciReader Logo Composition** ¡ª logo_universal_openscireader_logo, logo_universal_open_book, logo_universal_magnifying_glass [INFERRED 0.74]
- **OpenSciReader Translation Stack** ¡ª readme_openscireader, readme_local_pdf_layout_translation_pipeline, readme_pdf2zh_next_high_level_do_translate_async_stream [EXTRACTED 1.00]
- **Bundled Translation Runtime Packaging** ¡ª readme_windows_installer, readme_python_worker, graph_report_runtime_readme_bundled_pdf_translation_runtime [EXTRACTED 1.00]

## Communities

### Community 0 - "PDF Translation Jobs"
Cohesion: 0.05
Nodes (44): StartPDFTranslate(), handleCancelJob(), handleDeleteJob(), handleRefreshJobs(), loadJobs(), buildChunks(), canReusePreviewForExport(), cloneSnapshot() (+36 more)

### Community 1 - "AI Gateway Providers"
Cohesion: 0.06
Nodes (48): GetAIWorkspaceConfig(), GetConfigSnapshot(), SaveProvider(), newConfigStore(), openSQLite(), encryptLegacyStringForTest(), TestAIWorkspaceConfigPersistence(), TestBootstrapMigratesLegacyEncryptedProviderSecrets() (+40 more)

### Community 2 - "Wails Data Models"
Cohesion: 0.03
Nodes (20): AIWorkspaceConfig, ChatHistoryEntry, ChunkStatus, CollectionTree, ConfigSnapshot, DiscoveredModel, DiscoveredModelsResponse, FigureGenerationResult (+12 more)

### Community 3 - "Wails Runtime Bridge"
Cohesion: 0.03
Nodes (3): EventsOn(), EventsOnce(), EventsOnMultiple()

### Community 4 - "Reader PDF Utilities"
Cohesion: 0.03
Nodes (64): 1431 Edges, 58 Communities Detected, 82% Extracted 18% Inferred 0% Ambiguous, 839 Nodes, Bundled Translation Runtime Packaging, Community Hubs Navigation, God Nodes, Graph Report (+56 more)

### Community 5 - "Reader State Stores"
Cohesion: 0.04
Nodes (27): App, DeleteChatHistory(), DeleteModel(), DeleteProvider(), FetchProviderModels(), GenerateResearchFigure(), GetCollections(), GetItemsByCollection() (+19 more)

### Community 6 - "Configuration Types"
Cohesion: 0.09
Nodes (31): appendQuery(), buildCollectionChildren(), buildCollectionTree(), decodeFileURL(), defaultLibraryName(), extractYear(), fetchPaginatedJSON(), firstNonEmpty() (+23 more)

### Community 7 - "PDF Service Backend"
Cohesion: 0.08
Nodes (12): decryptLegacyString(), loadExistingKey(), maskAPIKey(), newSecretManager(), secretManager, boolToInt(), defaultAIWorkspaceConfig(), isValidProviderType() (+4 more)

### Community 8 - "Zotero Integration"
Cohesion: 0.12
Nodes (30): handleImportRuntime(), pdfTranslateRuntimeManifest, resolvedPDFTranslateRuntime, cloneAppPathsForTest(), copyFileForTest(), firstActiveLLMProviderAndModel(), manualIntegrationPDFPath(), TestManualPreviewExportReuseForWang2017() (+22 more)

### Community 9 - "Frontend PDF API"
Cohesion: 0.09
Nodes (28): createProviderFromTemplate(), findMatchingProviderTemplate(), findModelPreset(), getRecommendedModelsForProvider(), normalize(), suggestContextWindow(), applyDrawingTemplate(), applyLLMTemplate() (+20 more)

### Community 10 - "Notes And History API"
Cohesion: 0.12
Nodes (29): destroyPDFDocument(), normalizePDFError(), openPDFDocument(), buildCaptionSideRect(), buildTextLineItem(), buildTextLines(), clampRect(), collectImageStats() (+21 more)

### Community 11 - "Reader Tab UI"
Cohesion: 0.11
Nodes (23): handleViewerKeyDown(), handleViewerWheel(), renderOriginalPDF(), resetZoom(), scrollByViewport(), updateScale(), zoomBy(), buildReaderOutline() (+15 more)

### Community 12 - "Home Page UI"
Cohesion: 0.14
Nodes (20): buildPaperImageGenerationPrompt(), buildPaperImageSummaryInstruction(), loadAllPageTexts(), loadPDFTextChunks(), loadPDFTextContext(), buildPromptForScope(), finalizeResult(), generateVisualSummaryForFigure() (+12 more)

### Community 13 - "AI Panel UI"
Cohesion: 0.17
Nodes (26): build_nested_payload(), build_settings(), emit(), find_settings_loader(), find_settings_model(), get_field(), get_model_annotation(), is_model_class() (+18 more)

### Community 14 - "Markdown And Figures"
Cohesion: 0.11
Nodes (17): AIWorkspaceConfig, ChatHistoryEntry, ConfigSnapshot, DiscoveredModel, DiscoveredModelsResponse, ModelRecord, ModelUpsertInput, OCRPageResult (+9 more)

### Community 15 - "Frontend Type Models"
Cohesion: 0.13
Nodes (9): toReaderUrl(), closeSubscription(), defaultReaderUIState(), downloadTextFile(), handleBulkImportFigures(), handleCancelTranslation(), handleDownloadPDF(), handleExportNotes() (+1 more)

### Community 16 - "Gateway Translation Tests"
Cohesion: 0.24
Nodes (12): getChunkCaption(), formatRelativeTimestamp(), getChunkStatusLabel(), getJobLiveHint(), getJobPrimaryStatus(), getJobProgressPercent(), getJobSecondaryStatus(), getJobStageLabel() (+4 more)

### Community 17 - "Translator Runtime Tests"
Cohesion: 0.31
Nodes (7): extractDataURLPayload(), clampUnit(), decodeOCRImageSize(), gatewayService, glmOCREndpoint(), normalizeOCRFileInput(), parseGLMOCRBlocks()

### Community 18 - "OCR Gateway Flow"
Cohesion: 0.25
Nodes (6): NewApp(), newLocalAssetHandler(), normalizeLocalPDFRequestPath(), serveLocalPDFFile(), serveRemotePDFFile(), main()

### Community 19 - "Provider Model Discovery"
Cohesion: 0.22
Nodes (8): ChunkStatus, JobEvent, JobOutputs, JobSnapshot, JobStatus, ProviderConfig, StageSummaryItem, StartRequest

### Community 20 - "PDF Runtime Paths"
Cohesion: 0.32
Nodes (3): getWailsApp(), isWailsApp(), isWailsDesktop()

### Community 21 - "Asset File Handling"
Cohesion: 0.25
Nodes (0): 

### Community 22 - "Crypto And Secrets"
Cohesion: 0.48
Nodes (6): buildProviderModelsEndpoint(), dedupeAndSortDiscoveredModels(), isGoogleModelDiscoveryProvider(), parseDiscoveredModels(), parseGoogleModels(), parseOpenAICompatibleModels()

### Community 23 - "Config Store Persistence"
Cohesion: 0.29
Nodes (7): Apache License 2.0, bzip2, Guido van Rossum, libffi, Microsoft Distributable Code, Python, Python Software Foundation License Version 2

### Community 24 - "App Entry Wiring"
Cohesion: 0.47
Nodes (3): createMockApp(), getApp(), isWailsApp()

### Community 25 - "Frontend App Shell"
Cohesion: 0.33
Nodes (6): Typst Design Principles Rationale, Incremental Compilation, LaTeX, NLnet, Typst, Typst CLI

### Community 26 - "PDF Source Handling"
Cohesion: 0.6
Nodes (3): getPDFTranslatePerformancePreset(), includesAny(), normalizeText()

### Community 27 - "PDF Document Parsing"
Cohesion: 0.7
Nodes (3): applyPDFTranslateEvent(), normalizePDFTranslateSnapshot(), normalizeProgressValue()

### Community 28 - "PDF Text Extraction"
Cohesion: 0.4
Nodes (5): AGPL-3.0 License, document.v1.json, MIT License, Release Compliance Rationale, Third-Party Notices

### Community 29 - "PDF Outline Parsing"
Cohesion: 0.5
Nodes (0): 

### Community 30 - "PDF Figure Extraction"
Cohesion: 0.5
Nodes (4): GitHub Actions Release Packaging, Slim Installer Asset, Standalone Runtime Asset, Windows Release Workflow

### Community 31 - "Translation Presentation"
Cohesion: 0.67
Nodes (2): CollectionTree, ZoteroItem

### Community 32 - "Translation Performance"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 33 - "Utility Helpers"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 34 - "Button Component"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 35 - "Reader Layout Types"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 36 - "Gateway Request Types"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 37 - "Config API Types"
Cohesion: 0.67
Nodes (0): 

### Community 38 - "History Types"
Cohesion: 0.67
Nodes (3): Font Software, The Nunito Project Authors, SIL Open Font License 1.1

### Community 39 - "Notes Types"
Cohesion: 0.67
Nodes (3): BabelDOC, Bundled PDF Translation Runtime, pdf2zh_next

### Community 40 - "PDF Types"
Cohesion: 0.67
Nodes (3): python_worker, runtime/webview2/windows-amd64, Windows Installer

### Community 41 - "Translation Types"
Cohesion: 1.0
Nodes (1): PDFDocumentPayload

### Community 42 - "Zotero Types"
Cohesion: 1.0
Nodes (1): PDFTranslateStartInput

### Community 43 - "AI Catalog"
Cohesion: 1.0
Nodes (0): 

### Community 44 - "Figure Prompting"
Cohesion: 1.0
Nodes (0): 

### Community 45 - "Config Store Tests"
Cohesion: 1.0
Nodes (0): 

### Community 46 - "Gateway Service Core"
Cohesion: 1.0
Nodes (0): 

### Community 47 - "Translator Manager Core"
Cohesion: 1.0
Nodes (0): 

### Community 48 - "Python Worker"
Cohesion: 1.0
Nodes (0): 

### Community 49 - "Runtime Bundles"
Cohesion: 1.0
Nodes (0): 

### Community 50 - "Build Tooling"
Cohesion: 1.0
Nodes (0): 

### Community 51 - "Project Documentation"
Cohesion: 1.0
Nodes (0): 

### Community 52 - "Third Party Notices"
Cohesion: 1.0
Nodes (0): 

### Community 53 - "Embedded Runtime Docs"
Cohesion: 1.0
Nodes (0): 

### Community 54 - "Logo Asset"
Cohesion: 1.0
Nodes (0): 

### Community 55 - "Main App Service"
Cohesion: 1.0
Nodes (0): 

### Community 56 - "PDF HTTP Handlers"
Cohesion: 1.0
Nodes (0): 

### Community 57 - "Config Path Helpers"
Cohesion: 1.0
Nodes (0): 

### Community 58 - "Gateway Integration Tests"
Cohesion: 1.0
Nodes (0): 

### Community 59 - "Translator Storage"
Cohesion: 1.0
Nodes (0): 

### Community 60 - "Translator Types"
Cohesion: 1.0
Nodes (0): 

### Community 61 - "Reader App Bootstrap"
Cohesion: 1.0
Nodes (0): 

### Community 62 - "OpenSciReader Report"
Cohesion: 1.0
Nodes (1): runtime/README Bundled PDF Translation Runtime

## Ambiguous Edges - Review These
- `OpenSciReader Logo` ¡ú `Open Book`  [AMBIGUOUS]
  frontend/src/assets/images/logo-universal.png ¡¤ relation: form
- `OpenSciReader Logo` ¡ú `Magnifying Glass`  [AMBIGUOUS]
  frontend/src/assets/images/logo-universal.png ¡¤ relation: form
- `OpenSciReader Logo` ¡ú `PDF Document`  [AMBIGUOUS]
  frontend/src/assets/images/logo-universal.png ¡¤ relation: form

## Knowledge Gaps
- **119 isolated node(s):** `appPaths`, `providerSecretRecord`, `PDFTranslateRuntimeStatus`, `ProviderRecord`, `ProviderUpsertInput` (+114 more)
  These have ¡Ü1 connection - possible missing edges or undocumented components.
- **Thin community `Translation Types`** (2 nodes): `pdf_document_types.go`, `PDFDocumentPayload`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Zotero Types`** (2 nodes): `pdf_translate_service.go`, `PDFTranslateStartInput`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `AI Catalog`** (2 nodes): `cn()`, `Button.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Figure Prompting`** (2 nodes): `utils.ts`, `cn()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Config Store Tests`** (1 nodes): `postcss.config.js`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Gateway Service Core`** (1 nodes): `vite.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Translator Manager Core`** (1 nodes): `main.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Python Worker`** (1 nodes): `vite-env.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Runtime Bundles`** (1 nodes): `MarkdownPreview.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Build Tooling`** (1 nodes): `readerStore.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Project Documentation`** (1 nodes): `tabStore.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Third Party Notices`** (1 nodes): `zoteroStore.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Embedded Runtime Docs`** (1 nodes): `config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Logo Asset`** (1 nodes): `drawing.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Main App Service`** (1 nodes): `gateway.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `PDF HTTP Handlers`** (1 nodes): `history.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Config Path Helpers`** (1 nodes): `notes.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Gateway Integration Tests`** (1 nodes): `pdf.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Translator Storage`** (1 nodes): `zotero.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Translator Types`** (1 nodes): `App.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Reader App Bootstrap`** (1 nodes): `runtime.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `OpenSciReader Report`** (1 nodes): `runtime/README Bundled PDF Translation Runtime`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `OpenSciReader Logo` and `Open Book`?**
  _Edge tagged AMBIGUOUS (relation: form) - confidence is low._
- **What is the exact relationship between `OpenSciReader Logo` and `Magnifying Glass`?**
  _Edge tagged AMBIGUOUS (relation: form) - confidence is low._
- **What is the exact relationship between `OpenSciReader Logo` and `PDF Document`?**
  _Edge tagged AMBIGUOUS (relation: form) - confidence is low._
- **Why does `extractFigureCandidates()` connect `Notes And History API` to `Frontend Type Models`?**
  _High betweenness centrality (0.086) - this node is a cross-community bridge._
- **Why does `handleBulkImportFigures()` connect `Frontend Type Models` to `Notes And History API`?**
  _High betweenness centrality (0.082) - this node is a cross-community bridge._
- **Why does `EventsEmit()` connect `AI Gateway Providers` to `Zotero Integration`, `Wails Runtime Bridge`?**
  _High betweenness centrality (0.080) - this node is a cross-community bridge._
- **What connects `appPaths`, `providerSecretRecord`, `PDFTranslateRuntimeStatus` to the rest of the system?**
  _119 weakly-connected nodes found - possible documentation gaps or missing edges._