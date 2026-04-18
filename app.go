package main

import (
	"context"
	"fmt"

	"OpenSciReader/internal/translator"
)

// App struct
type App struct {
	ctx        context.Context
	store      *configStore
	paths      appPaths
	zotero     *zoteroService
	gateway    *gatewayService
	pdf        *pdfService
	translator *translator.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	paths, err := resolveAppPaths()
	if err != nil {
		panic(err)
	}

	store, err := newConfigStore(paths)
	if err != nil {
		panic(err)
	}

	a.paths = paths
	a.store = store
	a.zotero = newZoteroService()
	a.gateway = newGatewayService(store)
	a.pdf = newPDFService()
	a.translator = newPDFTranslateManagerOrPanic(paths, store, a.ctx)
}

func (a *App) shutdown(_ context.Context) {
	if a.store != nil {
		_ = a.store.Close()
	}
}

func (a *App) GetConfigSnapshot() (ConfigSnapshot, error) {
	return a.store.GetConfigSnapshot(a.ctx)
}

func (a *App) GetAIWorkspaceConfig() (AIWorkspaceConfig, error) {
	return a.store.GetAIWorkspaceConfig(a.ctx)
}

func (a *App) SaveAIWorkspaceConfig(input AIWorkspaceConfig) (AIWorkspaceConfig, error) {
	return a.store.SaveAIWorkspaceConfig(a.ctx, input)
}

func (a *App) GetPDFTranslateRuntimeStatus() (PDFTranslateRuntimeConfig, error) {
	return resolvePDFTranslateRuntime(a.paths, a.store).Config, nil
}

func (a *App) ImportPDFTranslateRuntime(packagePath string) (PDFTranslateRuntimeImportResult, error) {
	if a.store == nil {
		return PDFTranslateRuntimeImportResult{}, fmt.Errorf("config store is unavailable")
	}
	config, err := importPDFTranslateRuntimePackage(a.paths, packagePath)
	if err != nil {
		return PDFTranslateRuntimeImportResult{}, err
	}
	saved, err := a.store.SavePDFTranslateRuntimeConfig(a.ctx, config)
	if err != nil {
		return PDFTranslateRuntimeImportResult{}, err
	}
	a.translator = newPDFTranslateManagerOrPanic(a.paths, a.store, a.ctx)
	return PDFTranslateRuntimeImportResult{Runtime: saved}, nil
}

func (a *App) RemovePDFTranslateRuntime() error {
	if a.store == nil {
		return fmt.Errorf("config store is unavailable")
	}
	config, err := a.store.GetPDFTranslateRuntimeConfig(a.ctx)
	if err != nil {
		return err
	}
	if err := removeImportedPDFTranslateRuntime(config); err != nil {
		return err
	}
	if err := a.store.ClearPDFTranslateRuntimeConfig(a.ctx); err != nil {
		return err
	}
	a.translator = newPDFTranslateManagerOrPanic(a.paths, a.store, a.ctx)
	return nil
}

func (a *App) SaveProvider(input ProviderUpsertInput) (ProviderRecord, error) {
	return a.store.SaveProvider(a.ctx, input)
}

func (a *App) DeleteProvider(id int64) error {
	return a.store.DeleteProvider(a.ctx, id)
}

func (a *App) SaveModel(input ModelUpsertInput) (ModelRecord, error) {
	return a.store.SaveModel(a.ctx, input)
}

func (a *App) FetchProviderModels(providerID int64) (DiscoveredModelsResponse, error) {
	return a.gateway.FetchProviderModels(a.ctx, providerID)
}

func (a *App) DeleteModel(id int64) error {
	return a.store.DeleteModel(a.ctx, id)
}

func (a *App) GetCollections(source string) ([]CollectionTree, error) {
	return a.zotero.GetCollections(a.ctx, source)
}

func (a *App) GetItemsByCollection(collectionID string) ([]ZoteroItem, error) {
	return a.zotero.GetItemsByCollection(a.ctx, collectionID)
}

func (a *App) ResolvePDFPath(itemID string) (string, error) {
	return a.zotero.ResolvePDFPath(a.ctx, itemID)
}

func (a *App) LoadPDFDocument(pdfPath string) (PDFDocumentPayload, error) {
	return a.pdf.LoadDocument(a.ctx, pdfPath)
}

func (a *App) StreamLLMChat(providerID, modelID int64, prompt string, contextData GatewayContextData) (string, error) {
	return a.gateway.StreamLLMChat(a.ctx, a.ctx, providerID, modelID, prompt, contextData)
}

func (a *App) ProxyTranslation(providerID, modelID int64, text, sourceLang, targetLang string) (string, error) {
	return a.gateway.ProxyTranslation(a.ctx, providerID, modelID, text, sourceLang, targetLang)
}

func (a *App) GenerateResearchFigure(providerID, modelID int64, prompt string, contextData GatewayContextData) (FigureGenerationResult, error) {
	return a.gateway.GenerateResearchFigure(a.ctx, providerID, modelID, prompt, contextData)
}

func (a *App) SaveChatHistory(entry ChatHistoryEntry) (ChatHistoryEntry, error) {
	return a.store.SaveChatHistory(a.ctx, entry)
}

func (a *App) ListChatHistory(itemID string) ([]ChatHistoryEntry, error) {
	return a.store.ListChatHistory(a.ctx, itemID)
}

func (a *App) DeleteChatHistory(id int64) error {
	return a.store.DeleteChatHistory(a.ctx, id)
}

func (a *App) SaveReaderNote(entry ReaderNoteEntry) (ReaderNoteEntry, error) {
	return a.store.SaveReaderNote(a.ctx, entry)
}

func (a *App) ListReaderNotes(itemID string) ([]ReaderNoteEntry, error) {
	return a.store.ListReaderNotes(a.ctx, itemID)
}
