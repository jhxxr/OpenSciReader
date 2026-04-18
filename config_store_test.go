package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"
)

func TestPDFTranslateRuntimeConfigPersistence(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runtimeDir := tempDir + "/runtime"
	pythonDir := runtimeDir + "/runtime"
	if err := os.MkdirAll(pythonDir, 0o755); err != nil {
		t.Fatalf("create runtime python dir error: %v", err)
	}
	if err := os.MkdirAll(runtimeDir+"/site-packages", 0o755); err != nil {
		t.Fatalf("create site-packages dir error: %v", err)
	}
	if err := os.WriteFile(pythonDir+"/python.exe", []byte(""), 0o600); err != nil {
		t.Fatalf("write python exe error: %v", err)
	}

	store, err := newConfigStore(appPaths{
		RootDir:                  tempDir,
		AppConfigDBPath:          tempDir + "/app.sqlite",
		OCRCacheDBPath:           tempDir + "/ocr.sqlite",
		EncryptionKeyPath:        tempDir + "/config.key",
		TranslateRuntimeRootDir:  tempDir + "/runtime-root",
		TranslateRuntimeCacheDir: tempDir + "/runtime-cache",
	})
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	defaults, err := store.GetPDFTranslateRuntimeConfig(t.Context())
	if err != nil {
		t.Fatalf("GetPDFTranslateRuntimeConfig default error: %v", err)
	}
	if defaults.Status != PDFTranslateRuntimeStatusMissing {
		t.Fatalf("default runtime status = %q, want %q", defaults.Status, PDFTranslateRuntimeStatusMissing)
	}

	saved, err := store.SavePDFTranslateRuntimeConfig(t.Context(), PDFTranslateRuntimeConfig{
		Installed:      true,
		Status:         PDFTranslateRuntimeStatusInstalling,
		RuntimeID:      "pdf2zh-next",
		Version:        "v1.2.3",
		Platform:       "windows-amd64",
		RuntimeDir:     runtimeDir,
		SourceFileName: "runtime.zip",
	})
	if err != nil {
		t.Fatalf("SavePDFTranslateRuntimeConfig error: %v", err)
	}
	if saved.Status != PDFTranslateRuntimeStatusValid {
		t.Fatalf("saved runtime status = %q, want %q", saved.Status, PDFTranslateRuntimeStatusValid)
	}
	if saved.PythonPath == "" {
		t.Fatalf("saved python path should be populated")
	}

	reloaded, err := store.GetPDFTranslateRuntimeConfig(t.Context())
	if err != nil {
		t.Fatalf("GetPDFTranslateRuntimeConfig reload error: %v", err)
	}
	if reloaded.RuntimeDir != runtimeDir {
		t.Fatalf("reloaded runtime dir = %q, want %q", reloaded.RuntimeDir, runtimeDir)
	}
	if reloaded.Status != PDFTranslateRuntimeStatusValid {
		t.Fatalf("reloaded runtime status = %q, want %q", reloaded.Status, PDFTranslateRuntimeStatusValid)
	}

	snapshot, err := store.GetConfigSnapshot(t.Context())
	if err != nil {
		t.Fatalf("GetConfigSnapshot error: %v", err)
	}
	if snapshot.PDFTranslateRuntime.RuntimeDir != runtimeDir {
		t.Fatalf("snapshot runtime dir = %q, want %q", snapshot.PDFTranslateRuntime.RuntimeDir, runtimeDir)
	}

	if err := store.ClearPDFTranslateRuntimeConfig(t.Context()); err != nil {
		t.Fatalf("ClearPDFTranslateRuntimeConfig error: %v", err)
	}
	cleared, err := store.GetPDFTranslateRuntimeConfig(t.Context())
	if err != nil {
		t.Fatalf("GetPDFTranslateRuntimeConfig after clear error: %v", err)
	}
	if cleared.Status != PDFTranslateRuntimeStatusMissing {
		t.Fatalf("cleared runtime status = %q, want %q", cleared.Status, PDFTranslateRuntimeStatusMissing)
	}
}

func TestAIWorkspaceConfigPersistence(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store, err := newConfigStore(appPaths{
		RootDir:           tempDir,
		AppConfigDBPath:   tempDir + "/app.sqlite",
		OCRCacheDBPath:    tempDir + "/ocr.sqlite",
		EncryptionKeyPath: tempDir + "/config.key",
	})
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	defaults, err := store.GetAIWorkspaceConfig(t.Context())
	if err != nil {
		t.Fatalf("GetAIWorkspaceConfig default error: %v", err)
	}
	if defaults.SummaryMode != "auto" {
		t.Fatalf("default SummaryMode = %q, want auto", defaults.SummaryMode)
	}

	saved, err := store.SaveAIWorkspaceConfig(t.Context(), AIWorkspaceConfig{
		SummaryMode:          "multi",
		SummaryChunkPages:    4,
		SummaryChunkMaxChars: 15000,
		AutoRestoreCount:     5,
		TableTemplate:        "| A | B |\n| --- | --- |\n| x | y |",
		TablePrompt:          "fill table",
		CustomPromptDraft:    "draft question",
		FollowUpPromptDraft:  "follow up",
		DrawingPromptDraft:   "draw chart",
		DrawingProviderID:    7,
		DrawingModel:         "gemini-3-pro-image-preview",
	})
	if err != nil {
		t.Fatalf("SaveAIWorkspaceConfig error: %v", err)
	}
	if saved.SummaryMode != "multi" {
		t.Fatalf("saved SummaryMode = %q, want multi", saved.SummaryMode)
	}

	reloaded, err := store.GetAIWorkspaceConfig(t.Context())
	if err != nil {
		t.Fatalf("GetAIWorkspaceConfig reload error: %v", err)
	}
	if reloaded.SummaryChunkPages != 4 {
		t.Fatalf("reloaded SummaryChunkPages = %d, want 4", reloaded.SummaryChunkPages)
	}
	if reloaded.TablePrompt != "fill table" {
		t.Fatalf("reloaded TablePrompt = %q, want fill table", reloaded.TablePrompt)
	}
	if reloaded.DrawingPromptDraft != "draw chart" {
		t.Fatalf("reloaded DrawingPromptDraft = %q, want draw chart", reloaded.DrawingPromptDraft)
	}
	if reloaded.DrawingProviderID != 7 {
		t.Fatalf("reloaded DrawingProviderID = %d, want 7", reloaded.DrawingProviderID)
	}
	if reloaded.DrawingModel != "gemini-3-pro-image-preview" {
		t.Fatalf("reloaded DrawingModel = %q, want gemini-3-pro-image-preview", reloaded.DrawingModel)
	}
}

func TestBootstrapPurgesDeprecatedOCRProviders(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := appPaths{
		RootDir:           tempDir,
		AppConfigDBPath:   tempDir + "/app.sqlite",
		OCRCacheDBPath:    tempDir + "/ocr.sqlite",
		EncryptionKeyPath: tempDir + "/config.key",
	}

	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}

	llmProvider, err := store.SaveProvider(t.Context(), ProviderUpsertInput{
		Name:     "LLM",
		Type:     ProviderTypeLLM,
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "test-key",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("SaveProvider llm error: %v", err)
	}

	result, err := store.appDB.Exec(
		`INSERT INTO providers (name, type, base_url, region, api_key, is_active) VALUES (?, ?, ?, ?, ?, ?);`,
		"Legacy OCR", string(ProviderTypeOCR), "https://legacy.example/ocr", "", "", 1,
	)
	if err != nil {
		t.Fatalf("insert legacy ocr provider error: %v", err)
	}
	ocrProviderID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read ocr provider id error: %v", err)
	}
	if _, err := store.appDB.Exec(
		`INSERT INTO models (provider_id, model_id, context_window) VALUES (?, ?, ?);`,
		ocrProviderID, "legacy-ocr-model", 0,
	); err != nil {
		t.Fatalf("insert legacy ocr model error: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close store error: %v", err)
	}

	reopened, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("reopen configStore error: %v", err)
	}
	defer func() {
		_ = reopened.Close()
	}()

	snapshot, err := reopened.GetConfigSnapshot(t.Context())
	if err != nil {
		t.Fatalf("GetConfigSnapshot error: %v", err)
	}
	if len(snapshot.Providers) != 1 {
		t.Fatalf("provider count = %d, want 1", len(snapshot.Providers))
	}
	if snapshot.Providers[0].Provider.ID != llmProvider.ID {
		t.Fatalf("remaining provider id = %d, want %d", snapshot.Providers[0].Provider.ID, llmProvider.ID)
	}

	var providerCount int
	if err := reopened.appDB.QueryRow(`SELECT COUNT(*) FROM providers WHERE type = ?;`, string(ProviderTypeOCR)).Scan(&providerCount); err != nil {
		t.Fatalf("count ocr providers error: %v", err)
	}
	if providerCount != 0 {
		t.Fatalf("ocr provider count = %d, want 0", providerCount)
	}

	var modelCount int
	if err := reopened.appDB.QueryRow(`SELECT COUNT(*) FROM models WHERE provider_id = ?;`, ocrProviderID).Scan(&modelCount); err != nil {
		t.Fatalf("count ocr models error: %v", err)
	}
	if modelCount != 0 {
		t.Fatalf("ocr model count = %d, want 0", modelCount)
	}
}

func TestSaveProviderRejectsOCRType(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store, err := newConfigStore(appPaths{
		RootDir:           tempDir,
		AppConfigDBPath:   tempDir + "/app.sqlite",
		OCRCacheDBPath:    tempDir + "/ocr.sqlite",
		EncryptionKeyPath: tempDir + "/config.key",
	})
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	_, err = store.SaveProvider(t.Context(), ProviderUpsertInput{
		Name:     "Deprecated OCR",
		Type:     ProviderTypeOCR,
		BaseURL:  "https://example.com/ocr",
		APIKey:   "test-key",
		IsActive: true,
	})
	if err == nil {
		t.Fatalf("SaveProvider expected error for OCR provider type")
	}
}

func TestSaveProviderStoresAPIKeyInPlaintext(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store, err := newConfigStore(appPaths{
		RootDir:           tempDir,
		AppConfigDBPath:   tempDir + "/app.sqlite",
		OCRCacheDBPath:    tempDir + "/ocr.sqlite",
		EncryptionKeyPath: tempDir + "/config.key",
	})
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	provider, err := store.SaveProvider(t.Context(), ProviderUpsertInput{
		Name:     "Plaintext Provider",
		Type:     ProviderTypeLLM,
		BaseURL:  "https://api.example.com/v1",
		APIKey:   "plain-local-key",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("SaveProvider error: %v", err)
	}

	var storedKey string
	if err := store.appDB.QueryRow(`SELECT api_key FROM providers WHERE id = ?;`, provider.ID).Scan(&storedKey); err != nil {
		t.Fatalf("query stored api_key error: %v", err)
	}
	if storedKey != "plain-local-key" {
		t.Fatalf("stored api_key = %q, want %q", storedKey, "plain-local-key")
	}

	secret, err := store.GetProviderSecret(t.Context(), provider.ID)
	if err != nil {
		t.Fatalf("GetProviderSecret error: %v", err)
	}
	if secret.APIKey != "plain-local-key" {
		t.Fatalf("decrypted api_key = %q, want %q", secret.APIKey, "plain-local-key")
	}
}

func TestBootstrapMigratesLegacyEncryptedProviderSecrets(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := appPaths{
		RootDir:           tempDir,
		AppConfigDBPath:   tempDir + "/app.sqlite",
		OCRCacheDBPath:    tempDir + "/ocr.sqlite",
		EncryptionKeyPath: tempDir + "/config.key",
	}

	legacyKey := make([]byte, 32)
	if _, err := rand.Read(legacyKey); err != nil {
		t.Fatalf("rand.Read legacy key error: %v", err)
	}
	if err := os.WriteFile(paths.EncryptionKeyPath, []byte(base64.StdEncoding.EncodeToString(legacyKey)), 0o600); err != nil {
		t.Fatalf("write legacy key file error: %v", err)
	}
	legacyCipherText, err := encryptLegacyStringForTest(legacyKey, "legacy-secret-key")
	if err != nil {
		t.Fatalf("encrypt legacy string error: %v", err)
	}

	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}
	if _, err := store.appDB.Exec(
		`INSERT INTO providers (name, type, base_url, region, api_key, is_active) VALUES (?, ?, ?, ?, ?, ?);`,
		"Legacy Encrypted", string(ProviderTypeLLM), "https://api.example.com/v1", "", legacyCipherText, 1,
	); err != nil {
		t.Fatalf("insert legacy encrypted provider error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store error: %v", err)
	}

	reopened, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("reopen configStore error: %v", err)
	}
	defer func() {
		_ = reopened.Close()
	}()

	var (
		providerID int64
		storedKey  string
	)
	if err := reopened.appDB.QueryRow(`SELECT id, api_key FROM providers WHERE name = ?;`, "Legacy Encrypted").Scan(&providerID, &storedKey); err != nil {
		t.Fatalf("query migrated provider error: %v", err)
	}
	if storedKey != "legacy-secret-key" {
		t.Fatalf("migrated stored api_key = %q, want %q", storedKey, "legacy-secret-key")
	}

	secret, err := reopened.GetProviderSecret(t.Context(), providerID)
	if err != nil {
		t.Fatalf("GetProviderSecret migrated error: %v", err)
	}
	if secret.APIKey != "legacy-secret-key" {
		t.Fatalf("migrated APIKey = %q, want %q", secret.APIKey, "legacy-secret-key")
	}
}

func TestDeleteModelRemovesOnlyTargetModel(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store, err := newConfigStore(appPaths{
		RootDir:           tempDir,
		AppConfigDBPath:   tempDir + "/app.sqlite",
		OCRCacheDBPath:    tempDir + "/ocr.sqlite",
		EncryptionKeyPath: tempDir + "/config.key",
	})
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	provider, err := store.SaveProvider(t.Context(), ProviderUpsertInput{
		Name:     "LLM",
		Type:     ProviderTypeLLM,
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "test-key",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("SaveProvider error: %v", err)
	}

	firstModel, err := store.SaveModel(t.Context(), ModelUpsertInput{
		ProviderID:    provider.ID,
		ModelID:       "gpt-4.1",
		ContextWindow: 1047576,
	})
	if err != nil {
		t.Fatalf("SaveModel first error: %v", err)
	}
	secondModel, err := store.SaveModel(t.Context(), ModelUpsertInput{
		ProviderID:    provider.ID,
		ModelID:       "gpt-4.1-mini",
		ContextWindow: 1047576,
	})
	if err != nil {
		t.Fatalf("SaveModel second error: %v", err)
	}

	if err := store.DeleteModel(t.Context(), firstModel.ID); err != nil {
		t.Fatalf("DeleteModel error: %v", err)
	}

	snapshot, err := store.GetConfigSnapshot(t.Context())
	if err != nil {
		t.Fatalf("GetConfigSnapshot error: %v", err)
	}
	if len(snapshot.Providers) != 1 {
		t.Fatalf("provider count = %d, want 1", len(snapshot.Providers))
	}
	if len(snapshot.Providers[0].Models) != 1 {
		t.Fatalf("model count = %d, want 1", len(snapshot.Providers[0].Models))
	}
	if snapshot.Providers[0].Models[0].ID != secondModel.ID {
		t.Fatalf("remaining model id = %d, want %d", snapshot.Providers[0].Models[0].ID, secondModel.ID)
	}

	if _, err := store.GetModel(t.Context(), firstModel.ID); err == nil {
		t.Fatalf("GetModel deleted model expected error")
	}
}

func encryptLegacyStringForTest(key []byte, plainText string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func TestBootstrapPurgesDeprecatedOCRCache(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := appPaths{
		RootDir:           tempDir,
		AppConfigDBPath:   tempDir + "/app.sqlite",
		OCRCacheDBPath:    tempDir + "/ocr.sqlite",
		EncryptionKeyPath: tempDir + "/config.key",
	}

	store, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}
	if _, err := store.ocrDB.Exec(
		`INSERT INTO page_ocr_results (pdf_hash, page_number, resolution, json_data, created_at) VALUES (?, ?, ?, ?, ?);`,
		"hash", 1, 2, "[]", nowRFC3339(),
	); err != nil {
		t.Fatalf("insert ocr cache row error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store error: %v", err)
	}

	reopened, err := newConfigStore(paths)
	if err != nil {
		t.Fatalf("reopen configStore error: %v", err)
	}
	defer func() {
		_ = reopened.Close()
	}()

	var count int
	if err := reopened.ocrDB.QueryRow(`SELECT COUNT(*) FROM page_ocr_results;`).Scan(&count); err != nil {
		t.Fatalf("count ocr cache rows error: %v", err)
	}
	if count != 0 {
		t.Fatalf("ocr cache row count = %d, want 0", count)
	}
}

func TestDeleteChatHistoryRemovesOnlyTargetEntry(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store, err := newConfigStore(appPaths{
		RootDir:           tempDir,
		AppConfigDBPath:   tempDir + "/app.sqlite",
		OCRCacheDBPath:    tempDir + "/ocr.sqlite",
		EncryptionKeyPath: tempDir + "/config.key",
	})
	if err != nil {
		t.Fatalf("newConfigStore error: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	first, err := store.SaveChatHistory(t.Context(), ChatHistoryEntry{
		ItemID:    "item-a",
		ItemTitle: "Paper A",
		Page:      1,
		Kind:      "summary",
		Prompt:    "first prompt",
		Response:  "first response",
	})
	if err != nil {
		t.Fatalf("SaveChatHistory first error: %v", err)
	}
	second, err := store.SaveChatHistory(t.Context(), ChatHistoryEntry{
		ItemID:    "item-a",
		ItemTitle: "Paper A",
		Page:      2,
		Kind:      "chat",
		Prompt:    "second prompt",
		Response:  "second response",
	})
	if err != nil {
		t.Fatalf("SaveChatHistory second error: %v", err)
	}
	otherItemEntry, err := store.SaveChatHistory(t.Context(), ChatHistoryEntry{
		ItemID:    "item-b",
		ItemTitle: "Paper B",
		Page:      3,
		Kind:      "chat",
		Prompt:    "other prompt",
		Response:  "other response",
	})
	if err != nil {
		t.Fatalf("SaveChatHistory other item error: %v", err)
	}

	if err := store.DeleteChatHistory(t.Context(), first.ID); err != nil {
		t.Fatalf("DeleteChatHistory error: %v", err)
	}

	itemAHistory, err := store.ListChatHistory(t.Context(), "item-a")
	if err != nil {
		t.Fatalf("ListChatHistory item-a error: %v", err)
	}
	if len(itemAHistory) != 1 {
		t.Fatalf("item-a history count = %d, want 1", len(itemAHistory))
	}
	if itemAHistory[0].ID != second.ID {
		t.Fatalf("remaining item-a history id = %d, want %d", itemAHistory[0].ID, second.ID)
	}

	itemBHistory, err := store.ListChatHistory(t.Context(), "item-b")
	if err != nil {
		t.Fatalf("ListChatHistory item-b error: %v", err)
	}
	if len(itemBHistory) != 1 {
		t.Fatalf("item-b history count = %d, want 1", len(itemBHistory))
	}
	if itemBHistory[0].ID != otherItemEntry.ID {
		t.Fatalf("remaining item-b history id = %d, want %d", itemBHistory[0].ID, otherItemEntry.ID)
	}
}
