package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"OpenSciReader/internal/translator"
	"github.com/wailsapp/wails/v2/pkg/runtime"
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
	wiki       *workspaceWikiService
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
	a.pdf = newPDFService(store)
	a.translator = newPDFTranslateManagerOrPanic(paths, store, a.ctx)
	a.wiki = newWorkspaceWikiService(paths, store, a.pdf, a.gateway)
}

func (a *App) shutdown(_ context.Context) {
	if a.store != nil {
		_ = a.store.Close()
	}
}

func (a *App) GetConfigSnapshot() (ConfigSnapshot, error) {
	return a.store.GetConfigSnapshot(a.ctx)
}

func (a *App) GetAIWorkspaceConfig(workspaceID string) (AIWorkspaceConfig, error) {
	return a.store.GetAIWorkspaceConfig(a.ctx, workspaceID)
}

func (a *App) SaveAIWorkspaceConfig(workspaceID string, input AIWorkspaceConfig) (AIWorkspaceConfig, error) {
	return a.store.SaveAIWorkspaceConfig(a.ctx, workspaceID, input)
}

func (a *App) GetPDFTranslateRuntimeStatus() (PDFTranslateRuntimeConfig, error) {
	return resolvePDFTranslateRuntime(a.paths, a.store).Config, nil
}

func (a *App) ImportPDFTranslateRuntime(packagePath string) (PDFTranslateRuntimeImportResult, error) {
	progress := newRuntimeImportProgressEmitter(a.ctx)
	if a.store == nil {
		progress.Fail("配置存储不可用")
		return PDFTranslateRuntimeImportResult{}, fmt.Errorf("配置存储不可用")
	}
	progress.Emit("preparing", "正在准备运行时安装包", 0.02, 0, 0)
	config, err := importPDFTranslateRuntimePackage(a.paths, packagePath, progress)
	if err != nil {
		progress.Fail(err.Error())
		return PDFTranslateRuntimeImportResult{}, err
	}
	progress.Emit("saving", "正在保存运行时配置", 0.96, 0, 0)
	saved, err := a.store.SavePDFTranslateRuntimeConfig(a.ctx, config)
	if err != nil {
		progress.Fail(err.Error())
		return PDFTranslateRuntimeImportResult{}, err
	}
	progress.Emit("finalizing", "正在重新加载翻译运行时", 0.99, 0, 0)
	a.translator = newPDFTranslateManagerOrPanic(a.paths, a.store, a.ctx)
	progress.Emit("completed", "运行时导入完成", 1, 0, 0)
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

func (a *App) ListWorkspaces() ([]Workspace, error) {
	return a.store.ListWorkspaces(a.ctx)
}

func (a *App) CreateWorkspace(input WorkspaceUpsertInput) (Workspace, error) {
	return a.store.CreateWorkspace(a.ctx, input)
}

func (a *App) ImportFiles(input ImportFilesInput) (ImportFilesResult, error) {
	return a.store.ImportFiles(a.ctx, a.paths, input)
}

func (a *App) ImportZoteroItem(workspaceID, itemID, pdfPath, title, citeKey string) (ImportFilesResult, error) {
	return a.store.ImportFiles(a.ctx, a.paths, ImportFilesInput{
		WorkspaceID: workspaceID,
		FilePaths:   []string{pdfPath},
		SourceType:  "zotero",
		SourceLabel: firstNonEmpty(citeKey, title),
		SourceRef:   itemID,
		Title:       strings.TrimSpace(title),
	})
}

func (a *App) SelectImportFiles() ([]string, error) {
	return runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择要导入的文件",
		Filters: []runtime.FileFilter{{
			DisplayName: "Documents",
			Pattern:     "*.pdf;*.md;*.markdown",
		}},
	})
}

func (a *App) ListDocuments(workspaceID string) ([]DocumentRecord, error) {
	return a.store.ListDocumentsByWorkspace(a.ctx, workspaceID)
}

func (a *App) DeleteDocument(workspaceID, documentID string) error {
	return a.store.DeleteDocument(a.ctx, a.paths, workspaceID, documentID)
}

func (a *App) StartWorkspaceWikiScan(input WorkspaceWikiScanStartInput) (WorkspaceWikiScanJob, error) {
	if a.wiki == nil {
		return WorkspaceWikiScanJob{}, fmt.Errorf("workspace wiki service is unavailable")
	}
	return a.wiki.Start(a.ctx, input)
}

func (a *App) GetWorkspaceWikiScanJob(jobID string) (WorkspaceWikiScanJob, error) {
	return a.store.GetWorkspaceWikiScanJob(a.ctx, jobID)
}

func (a *App) ListWorkspaceWikiScanJobs() ([]WorkspaceWikiScanJob, error) {
	return a.store.ListWorkspaceWikiScanJobs(a.ctx)
}

func (a *App) CancelWorkspaceWikiScan(jobID string) (WorkspaceWikiScanJob, error) {
	if a.wiki == nil {
		return WorkspaceWikiScanJob{}, fmt.Errorf("workspace wiki service is unavailable")
	}
	return a.wiki.Cancel(a.ctx, jobID)
}

func (a *App) DeleteWorkspaceWikiScanJob(jobID string) error {
	return a.store.DeleteWorkspaceWikiScanJob(a.ctx, jobID)
}

func (a *App) ListWorkspaceWikiPages(workspaceID string) ([]WorkspaceWikiPage, error) {
	return a.store.ListWorkspaceWikiPages(a.ctx, workspaceID)
}

func (a *App) GetWorkspaceWikiPage(pageID string) (WorkspaceWikiPageContent, error) {
	page, err := a.store.GetWorkspaceWikiPage(a.ctx, pageID)
	if err != nil {
		return WorkspaceWikiPageContent{}, err
	}
	data, err := os.ReadFile(page.MarkdownPath)
	if err != nil {
		return WorkspaceWikiPageContent{}, fmt.Errorf("read workspace wiki markdown: %w", err)
	}
	return WorkspaceWikiPageContent{Page: page, Markdown: string(data)}, nil
}

func (a *App) DeleteWorkspaceWikiPages(workspaceID string) error {
	files := newWorkspaceKnowledgeFiles(a.paths, workspaceID)
	if err := files.EnsureLayout(); err != nil {
		return err
	}

	indexPath, err := files.IndexPath()
	if err != nil {
		return err
	}
	overviewPath, err := files.OverviewPath()
	if err != nil {
		return err
	}
	openQuestionsPath, err := files.OpenQuestionsPath()
	if err != nil {
		return err
	}
	logPath, err := files.LogPath()
	if err != nil {
		return err
	}
	for _, path := range []string{indexPath, overviewPath, openQuestionsPath, logPath} {
		if err := removeWorkspaceKnowledgeFile(path); err != nil {
			return err
		}
	}

	docsDir, err := files.docsDir()
	if err != nil {
		return err
	}
	if err := removeWorkspaceKnowledgeArtifactsNotInSet(docsDir, ".md", map[string]struct{}{}); err != nil {
		return err
	}
	conceptsDir, err := files.conceptsDir()
	if err != nil {
		return err
	}
	if err := removeWorkspaceKnowledgeArtifactsNotInSet(conceptsDir, ".md", map[string]struct{}{}); err != nil {
		return err
	}

	pages, err := a.store.ListWorkspaceWikiPages(a.ctx, workspaceID)
	if err != nil {
		return err
	}
	for _, page := range pages {
		if strings.TrimSpace(page.MarkdownPath) == "" {
			continue
		}
		if err := os.Remove(page.MarkdownPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove workspace wiki page file: %w", err)
		}
	}
	return a.store.DeleteWorkspaceWikiPagesByWorkspace(a.ctx, workspaceID)
}

func (a *App) ListWorkspaceKnowledgeSources(workspaceID string) ([]WorkspaceKnowledgeSource, error) {
	files := newWorkspaceKnowledgeFiles(a.paths, workspaceID)
	if err := files.EnsureLayout(); err != nil {
		return nil, err
	}
	return files.ReadSources()
}

func (a *App) GetWorkspaceKnowledgeCompileSummary(workspaceID string) (WorkspaceKnowledgeCompileSummary, error) {
	files := newWorkspaceKnowledgeFiles(a.paths, workspaceID)
	if err := files.EnsureLayout(); err != nil {
		return WorkspaceKnowledgeCompileSummary{}, err
	}
	return files.ReadCompileSummary()
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

func (a *App) ExtractPDFMarkdown(pdfPath string) (PDFMarkdownPayload, error) {
	return a.pdf.ExtractMarkdown(a.ctx, pdfPath)
}

func (a *App) StreamLLMChat(providerID, modelID int64, prompt string, contextData GatewayContextData) (string, error) {
	return a.gateway.StreamLLMChat(a.ctx, a.ctx, providerID, modelID, prompt, contextData)
}

func (a *App) ProxyTranslation(providerID, modelID int64, text, sourceLang, targetLang string) (string, error) {
	return a.gateway.ProxyTranslation(a.ctx, providerID, modelID, text, sourceLang, targetLang)
}

func (a *App) GenerateResearchFigure(providerID, modelID int64, prompt string, contextData GatewayContextData, workspaceID string) (FigureGenerationResult, error) {
	return a.gateway.GenerateResearchFigure(a.ctx, providerID, modelID, prompt, contextData, workspaceID)
}

func (a *App) SaveChatHistory(entry ChatHistoryEntry) (ChatHistoryEntry, error) {
	return a.store.SaveChatHistory(a.ctx, entry)
}

func (a *App) ListChatHistory(workspaceID, documentID, itemID string) ([]ChatHistoryEntry, error) {
	return a.store.ListChatHistory(a.ctx, workspaceID, documentID, itemID)
}

func (a *App) DeleteChatHistory(id int64) error {
	return a.store.DeleteChatHistory(a.ctx, id)
}

func (a *App) SaveReaderNote(entry ReaderNoteEntry) (ReaderNoteEntry, error) {
	return a.store.SaveReaderNote(a.ctx, entry)
}

func (a *App) ListReaderNotes(workspaceID, documentID, itemID string) ([]ReaderNoteEntry, error) {
	return a.store.ListReaderNotes(a.ctx, workspaceID, documentID, itemID)
}
