# Graph Report - E:\0JHX\Project\OpenSciReader  (2026-04-18)

## Corpus Check
- 76 files · ~67,183 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 839 nodes · 1431 edges · 58 communities detected
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 260 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 28|Community 28]]
- [[_COMMUNITY_Community 29|Community 29]]
- [[_COMMUNITY_Community 30|Community 30]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 38|Community 38]]
- [[_COMMUNITY_Community 39|Community 39]]
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]

## God Nodes (most connected - your core abstractions)
1. `configStore` - 32 edges
2. `App` - 31 edges
3. `Manager` - 21 edges
4. `jobRuntime` - 18 edges
5. `gatewayService` - 17 edges
6. `TestManualPreviewExportReuseForWang2017()` - 17 edges
7. `newConfigStore()` - 16 edges
8. `nowRFC3339()` - 16 edges
9. `cloneSnapshot()` - 13 edges
10. `registerPDFTranslateHandlers()` - 11 edges

## Surprising Connections (you probably didn't know these)
- `handleSaveLLMProvider()` --calls--> `SaveProvider()`  [INFERRED]
  E:\0JHX\Project\OpenSciReader\frontend\src\App.tsx → frontend\wailsjs\go\main\App.js
- `handleSaveDrawingProvider()` --calls--> `SaveProvider()`  [INFERRED]
  E:\0JHX\Project\OpenSciReader\frontend\src\App.tsx → frontend\wailsjs\go\main\App.js
- `OpenSciReader` --conceptually_related_to--> `OpenSciReader Logo`  [INFERRED]
  README.md → frontend/src/assets/images/logo-universal.png
- `normalizeLocalPDFRequestPath()` --calls--> `normalizeAttachmentPath()`  [INFERRED]
  asset_local_handler.go → zotero_service.go
- `newSecretManager()` --calls--> `newConfigStore()`  [INFERRED]
  config_crypto.go → E:\0JHX\Project\OpenSciReader\config_store.go

## Hyperedges (group relationships)
- **OpenSciReader Translation Stack** — readme_openscireader, readme_local_pdf_translation_pipeline, readme_pdf2zh_next_high_level_do_translate_async_stream [EXTRACTED 1.00]
- **Bundled Translation Runtime Packaging** — readme_windows_installer, readme_python_worker, runtime_readme_bundled_pdf_translation_runtime [EXTRACTED 1.00]
- **OpenSciReader Logo Composition** — logo_universal_openscireader_logo, logo_universal_open_book, logo_universal_magnifying_glass [INFERRED 0.74]

## Communities

### Community 0 - "Community 0"
Cohesion: 0.05
Nodes (41): NewApp(), newLocalAssetHandler(), normalizeLocalPDFRequestPath(), serveLocalPDFFile(), serveRemotePDFFile(), handleCancelJob(), handleDeleteJob(), handleRefreshJobs() (+33 more)

### Community 1 - "Community 1"
Cohesion: 0.06
Nodes (48): GetAIWorkspaceConfig(), GetConfigSnapshot(), SaveProvider(), newConfigStore(), openSQLite(), encryptLegacyStringForTest(), TestAIWorkspaceConfigPersistence(), TestBootstrapMigratesLegacyEncryptedProviderSecrets() (+40 more)

### Community 2 - "Community 2"
Cohesion: 0.03
Nodes (20): AIWorkspaceConfig, ChatHistoryEntry, ChunkStatus, CollectionTree, ConfigSnapshot, DiscoveredModel, DiscoveredModelsResponse, FigureGenerationResult (+12 more)

### Community 3 - "Community 3"
Cohesion: 0.03
Nodes (3): EventsOn(), EventsOnce(), EventsOnMultiple()

### Community 4 - "Community 4"
Cohesion: 0.04
Nodes (31): DeleteChatHistory(), DeleteModel(), DeleteProvider(), FetchProviderModels(), GenerateResearchFigure(), GetCollections(), GetItemsByCollection(), handleDeleteModel() (+23 more)

### Community 5 - "Community 5"
Cohesion: 0.09
Nodes (31): appendQuery(), buildCollectionChildren(), buildCollectionTree(), decodeFileURL(), defaultLibraryName(), extractYear(), fetchPaginatedJSON(), firstNonEmpty() (+23 more)

### Community 6 - "Community 6"
Cohesion: 0.08
Nodes (12): decryptLegacyString(), loadExistingKey(), maskAPIKey(), newSecretManager(), secretManager, boolToInt(), defaultAIWorkspaceConfig(), isValidProviderType() (+4 more)

### Community 7 - "Community 7"
Cohesion: 0.11
Nodes (32): handleImportRuntime(), handleRemoveRuntime(), NewManager(), pdfTranslateRuntimeManifest, resolvedPDFTranslateRuntime, cloneAppPathsForTest(), copyFileForTest(), firstActiveLLMProviderAndModel() (+24 more)

### Community 8 - "Community 8"
Cohesion: 0.09
Nodes (27): createProviderFromTemplate(), findMatchingProviderTemplate(), findModelPreset(), getRecommendedModelsForProvider(), normalize(), suggestContextWindow(), applyDrawingTemplate(), applyLLMTemplate() (+19 more)

### Community 9 - "Community 9"
Cohesion: 0.11
Nodes (30): destroyPDFDocument(), normalizePDFError(), openPDFDocument(), buildCaptionSideRect(), buildTextLineItem(), buildTextLines(), clampRect(), collectImageStats() (+22 more)

### Community 10 - "Community 10"
Cohesion: 0.11
Nodes (23): handleViewerKeyDown(), handleViewerWheel(), renderOriginalPDF(), resetZoom(), scrollByViewport(), updateScale(), zoomBy(), buildReaderOutline() (+15 more)

### Community 11 - "Community 11"
Cohesion: 0.14
Nodes (20): buildPaperImageGenerationPrompt(), buildPaperImageSummaryInstruction(), loadAllPageTexts(), loadPDFTextChunks(), loadPDFTextContext(), buildPromptForScope(), finalizeResult(), generateVisualSummaryForFigure() (+12 more)

### Community 12 - "Community 12"
Cohesion: 0.17
Nodes (26): build_nested_payload(), build_settings(), emit(), find_settings_loader(), find_settings_model(), get_field(), get_model_annotation(), is_model_class() (+18 more)

### Community 13 - "Community 13"
Cohesion: 0.14
Nodes (13): toReaderUrl(), bindPDFTranslateEvents(), closeSubscription(), defaultReaderUIState(), downloadTextFile(), getPDFTranslateRuntimeBlockedMessage(), handleCancelTranslation(), handleDownloadPDF() (+5 more)

### Community 14 - "Community 14"
Cohesion: 0.11
Nodes (17): AIWorkspaceConfig, ChatHistoryEntry, ConfigSnapshot, DiscoveredModel, DiscoveredModelsResponse, ModelRecord, ModelUpsertInput, OCRPageResult (+9 more)

### Community 15 - "Community 15"
Cohesion: 0.13
Nodes (15): Magnifying Glass, Open Book, OpenSciReader Logo, PDF Document, BabelDOC Offline Assets Package, Local PDF Layout Translation Pipeline, OpenSciReader, pdf2zh_next.high_level.do_translate_async_stream (+7 more)

### Community 16 - "Community 16"
Cohesion: 0.24
Nodes (12): getChunkCaption(), formatRelativeTimestamp(), getChunkStatusLabel(), getJobLiveHint(), getJobPrimaryStatus(), getJobProgressPercent(), getJobSecondaryStatus(), getJobStageLabel() (+4 more)

### Community 17 - "Community 17"
Cohesion: 0.18
Nodes (6): resolveAppPaths(), newGatewayService(), appPaths, newPDFService(), pdfService, newZoteroService()

### Community 18 - "Community 18"
Cohesion: 0.31
Nodes (7): extractDataURLPayload(), clampUnit(), decodeOCRImageSize(), gatewayService, glmOCREndpoint(), normalizeOCRFileInput(), parseGLMOCRBlocks()

### Community 19 - "Community 19"
Cohesion: 0.32
Nodes (3): getWailsApp(), isWailsApp(), isWailsDesktop()

### Community 20 - "Community 20"
Cohesion: 0.25
Nodes (0): 

### Community 21 - "Community 21"
Cohesion: 0.48
Nodes (6): buildProviderModelsEndpoint(), dedupeAndSortDiscoveredModels(), isGoogleModelDiscoveryProvider(), parseDiscoveredModels(), parseGoogleModels(), parseOpenAICompatibleModels()

### Community 22 - "Community 22"
Cohesion: 0.29
Nodes (7): Apache License 2.0, bzip2, Guido van Rossum, libffi, Microsoft Distributable Code, Python, Python Software Foundation License Version 2

### Community 23 - "Community 23"
Cohesion: 0.47
Nodes (3): createMockApp(), getApp(), isWailsApp()

### Community 24 - "Community 24"
Cohesion: 0.33
Nodes (6): Typst Design Principles Rationale, Incremental Compilation, LaTeX, NLnet, Typst, Typst CLI

### Community 25 - "Community 25"
Cohesion: 0.6
Nodes (3): getPDFTranslatePerformancePreset(), includesAny(), normalizeText()

### Community 26 - "Community 26"
Cohesion: 0.7
Nodes (3): applyPDFTranslateEvent(), normalizePDFTranslateSnapshot(), normalizeProgressValue()

### Community 27 - "Community 27"
Cohesion: 0.4
Nodes (5): AGPL-3.0 License, document.v1.json, MIT License, Release Compliance Rationale, Third-Party Notices

### Community 28 - "Community 28"
Cohesion: 0.5
Nodes (0): 

### Community 29 - "Community 29"
Cohesion: 0.67
Nodes (2): CollectionTree, ZoteroItem

### Community 30 - "Community 30"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 31 - "Community 31"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 32 - "Community 32"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 33 - "Community 33"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 34 - "Community 34"
Cohesion: 1.0
Nodes (2): getApp(), isWailsApp()

### Community 35 - "Community 35"
Cohesion: 0.67
Nodes (0): 

### Community 36 - "Community 36"
Cohesion: 0.67
Nodes (3): Font Software, The Nunito Project Authors, SIL Open Font License 1.1

### Community 37 - "Community 37"
Cohesion: 1.0
Nodes (1): PDFDocumentPayload

### Community 38 - "Community 38"
Cohesion: 1.0
Nodes (1): PDFTranslateStartInput

### Community 39 - "Community 39"
Cohesion: 1.0
Nodes (0): 

### Community 40 - "Community 40"
Cohesion: 1.0
Nodes (0): 

### Community 41 - "Community 41"
Cohesion: 1.0
Nodes (0): 

### Community 42 - "Community 42"
Cohesion: 1.0
Nodes (0): 

### Community 43 - "Community 43"
Cohesion: 1.0
Nodes (0): 

### Community 44 - "Community 44"
Cohesion: 1.0
Nodes (0): 

### Community 45 - "Community 45"
Cohesion: 1.0
Nodes (0): 

### Community 46 - "Community 46"
Cohesion: 1.0
Nodes (0): 

### Community 47 - "Community 47"
Cohesion: 1.0
Nodes (0): 

### Community 48 - "Community 48"
Cohesion: 1.0
Nodes (0): 

### Community 49 - "Community 49"
Cohesion: 1.0
Nodes (0): 

### Community 50 - "Community 50"
Cohesion: 1.0
Nodes (0): 

### Community 51 - "Community 51"
Cohesion: 1.0
Nodes (0): 

### Community 52 - "Community 52"
Cohesion: 1.0
Nodes (0): 

### Community 53 - "Community 53"
Cohesion: 1.0
Nodes (0): 

### Community 54 - "Community 54"
Cohesion: 1.0
Nodes (0): 

### Community 55 - "Community 55"
Cohesion: 1.0
Nodes (0): 

### Community 56 - "Community 56"
Cohesion: 1.0
Nodes (0): 

### Community 57 - "Community 57"
Cohesion: 1.0
Nodes (0): 

## Ambiguous Edges - Review These
- `OpenSciReader Logo` → `Open Book`  [AMBIGUOUS]
  frontend/src/assets/images/logo-universal.png · relation: form
- `OpenSciReader Logo` → `Magnifying Glass`  [AMBIGUOUS]
  frontend/src/assets/images/logo-universal.png · relation: form
- `OpenSciReader Logo` → `PDF Document`  [AMBIGUOUS]
  frontend/src/assets/images/logo-universal.png · relation: form

## Knowledge Gaps
- **79 isolated node(s):** `appPaths`, `providerSecretRecord`, `PDFTranslateRuntimeStatus`, `ProviderRecord`, `ProviderUpsertInput` (+74 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 37`** (2 nodes): `pdf_document_types.go`, `PDFDocumentPayload`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 38`** (2 nodes): `pdf_translate_service.go`, `PDFTranslateStartInput`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (2 nodes): `cn()`, `Button.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (2 nodes): `utils.ts`, `cn()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 41`** (1 nodes): `postcss.config.js`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 42`** (1 nodes): `vite.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (1 nodes): `main.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 44`** (1 nodes): `vite-env.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 45`** (1 nodes): `MarkdownPreview.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 46`** (1 nodes): `readerStore.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 47`** (1 nodes): `tabStore.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 48`** (1 nodes): `zoteroStore.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 49`** (1 nodes): `config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 50`** (1 nodes): `drawing.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 51`** (1 nodes): `gateway.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 52`** (1 nodes): `history.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 53`** (1 nodes): `notes.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 54`** (1 nodes): `pdf.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (1 nodes): `zotero.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (1 nodes): `App.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 57`** (1 nodes): `runtime.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `OpenSciReader Logo` and `Open Book`?**
  _Edge tagged AMBIGUOUS (relation: form) - confidence is low._
- **What is the exact relationship between `OpenSciReader Logo` and `Magnifying Glass`?**
  _Edge tagged AMBIGUOUS (relation: form) - confidence is low._
- **What is the exact relationship between `OpenSciReader Logo` and `PDF Document`?**
  _Edge tagged AMBIGUOUS (relation: form) - confidence is low._
- **Why does `handleBulkImportFigures()` connect `Community 9` to `Community 13`?**
  _High betweenness centrality (0.094) - this node is a cross-community bridge._
- **Why does `EventsEmit()` connect `Community 1` to `Community 3`, `Community 7`?**
  _High betweenness centrality (0.092) - this node is a cross-community bridge._
- **What connects `appPaths`, `providerSecretRecord`, `PDFTranslateRuntimeStatus` to the rest of the system?**
  _79 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.05 - nodes in this community are weakly interconnected._