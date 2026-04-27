package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type configStore struct {
	ocrDB   *sql.DB
	appDB   *sql.DB
	secrets *secretManager
}

type providerSecretRecord struct {
	ProviderRecord
	APIKey string
}

func newConfigStore(paths appPaths) (*configStore, error) {
	secrets, err := newSecretManager(paths.EncryptionKeyPath)
	if err != nil {
		return nil, err
	}

	appDB, err := openSQLite(paths.AppConfigDBPath)
	if err != nil {
		return nil, err
	}

	ocrDB, err := openSQLite(paths.OCRCacheDBPath)
	if err != nil {
		_ = appDB.Close()
		return nil, err
	}

	store := &configStore{ocrDB: ocrDB, appDB: appDB, secrets: secrets}
	if err := store.bootstrap(); err != nil {
		_ = appDB.Close()
		_ = ocrDB.Close()
		return nil, err
	}

	return store, nil
}

func openSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database %s: %w", path, err)
	}

	// Keep each test database on a single SQLite connection.
	// The workspace wiki tests poll job state while a background goroutine updates it.
	// With multiple pooled connections, modernc sqlite can surface transient SQLITE_BUSY
	// errors against the same file on Windows. Serializing access through one connection
	// keeps the behavior stable without changing higher-level logic.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys for %s: %w", path, err)
	}

	return db, nil
}

func (s *configStore) bootstrap() error {
	appSchema := []string{
		`CREATE TABLE IF NOT EXISTS providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			base_url TEXT NOT NULL DEFAULT '',
			region TEXT NOT NULL DEFAULT '',
			api_key TEXT NOT NULL DEFAULT '',
			is_active INTEGER NOT NULL DEFAULT 1
		);`,
		`CREATE TABLE IF NOT EXISTS models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER NOT NULL,
			model_id TEXT NOT NULL,
			context_window INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY(provider_id) REFERENCES providers(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS workspaces (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			color TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			title TEXT NOT NULL,
			document_type TEXT NOT NULL DEFAULT 'paper',
			source_type TEXT NOT NULL DEFAULT 'manual',
			default_asset_id TEXT NOT NULL DEFAULT '',
			original_file_name TEXT NOT NULL DEFAULT '',
			primary_pdf_path TEXT NOT NULL DEFAULT '',
			content_hash TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS document_assets (
			id TEXT PRIMARY KEY,
			document_id TEXT NOT NULL,
			workspace_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT '',
			file_name TEXT NOT NULL,
			relative_path TEXT NOT NULL,
			absolute_path TEXT NOT NULL,
			mime_type TEXT NOT NULL DEFAULT '',
			byte_size INTEGER NOT NULL DEFAULT 0,
			content_hash TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY(document_id) REFERENCES documents(id) ON DELETE CASCADE,
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS import_records (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			document_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			source_label TEXT NOT NULL DEFAULT '',
			source_ref TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'completed',
			message TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
			FOREIGN KEY(document_id) REFERENCES documents(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS document_external_links (
			id TEXT PRIMARY KEY,
			document_id TEXT NOT NULL,
			workspace_id TEXT NOT NULL,
			provider TEXT NOT NULL,
			external_id TEXT NOT NULL DEFAULT '',
			external_key TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY(document_id) REFERENCES documents(id) ON DELETE CASCADE,
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS workspace_ai_configs (
			workspace_id TEXT PRIMARY KEY,
			config_json TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS workspace_agent_sessions (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			title TEXT NOT NULL,
			surface TEXT NOT NULL DEFAULT 'workspace',
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS workspace_agent_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			workspace_id TEXT NOT NULL,
			surface TEXT NOT NULL DEFAULT 'workspace',
			role TEXT NOT NULL,
			kind TEXT NOT NULL DEFAULT 'message',
			prompt TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			skill_name TEXT NOT NULL DEFAULT '',
			evidence_count INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			FOREIGN KEY(session_id) REFERENCES workspace_agent_sessions(id) ON DELETE CASCADE,
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS ai_chat_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id TEXT NOT NULL DEFAULT '',
			document_id TEXT NOT NULL DEFAULT '',
			item_id TEXT NOT NULL,
			item_title TEXT NOT NULL,
			page INTEGER NOT NULL DEFAULT 1,
			kind TEXT NOT NULL DEFAULT 'chat',
			prompt TEXT NOT NULL,
			response TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS reader_notes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id TEXT NOT NULL DEFAULT '',
			document_id TEXT NOT NULL DEFAULT '',
			item_id TEXT NOT NULL,
			item_title TEXT NOT NULL,
			page INTEGER NOT NULL DEFAULT 1,
			anchor_text TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS app_settings (
			setting_key TEXT PRIMARY KEY,
			setting_value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS pdf_markdown_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pdf_hash TEXT NOT NULL,
			extractor TEXT NOT NULL,
			extractor_version TEXT NOT NULL,
			json_data TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(pdf_hash, extractor, extractor_version)
		);`,
		`CREATE TABLE IF NOT EXISTS workspace_wiki_scan_jobs (
			job_id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			document_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			total_items INTEGER NOT NULL DEFAULT 0,
			processed_items INTEGER NOT NULL DEFAULT 0,
			failed_items INTEGER NOT NULL DEFAULT 0,
			current_item TEXT NOT NULL DEFAULT '',
			current_stage TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			overall_progress REAL NOT NULL DEFAULT 0,
			provider_id INTEGER NOT NULL DEFAULT 0,
			model_id INTEGER NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '',
			started_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			finished_at TEXT NOT NULL DEFAULT '',
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS workspace_wiki_pages (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			source_document_id TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL,
			slug TEXT NOT NULL,
			kind TEXT NOT NULL,
			markdown_path TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);`,
	}

	for _, stmt := range appSchema {
		if _, err := s.appDB.Exec(stmt); err != nil {
			return fmt.Errorf("bootstrap app config schema: %w", err)
		}
	}
	if err := s.ensureProvidersRegionColumn(); err != nil {
		return err
	}
	if err := s.ensureHistoryDocumentColumns(); err != nil {
		return err
	}
	if err := s.migrateLegacyEncryptedProviderSecrets(); err != nil {
		return err
	}
	if err := s.purgeDeprecatedOCRProviders(); err != nil {
		return err
	}
	if err := s.ensureWorkspaceWikiScanJobsSchema(); err != nil {
		return err
	}

	ocrSchema := `CREATE TABLE IF NOT EXISTS page_ocr_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pdf_hash TEXT NOT NULL,
		page_number INTEGER NOT NULL,
		resolution INTEGER NOT NULL,
		json_data TEXT NOT NULL,
		created_at TEXT NOT NULL
	);`

	if _, err := s.ocrDB.Exec(ocrSchema); err != nil {
		return fmt.Errorf("bootstrap ocr cache schema: %w", err)
	}
	if err := s.purgeDeprecatedOCRCache(); err != nil {
		return err
	}

	return nil
}

func (s *configStore) migrateLegacyEncryptedProviderSecrets() error {
	if s.secrets == nil {
		return nil
	}

	rows, err := s.appDB.Query(`
		SELECT id, api_key
		FROM providers
		WHERE api_key <> '';
	`)
	if err != nil {
		return fmt.Errorf("list provider secrets for migration: %w", err)
	}
	defer rows.Close()

	type migratedSecret struct {
		id    int64
		value string
	}
	var pending []migratedSecret
	for rows.Next() {
		var (
			id    int64
			value string
		)
		if err := rows.Scan(&id, &value); err != nil {
			return fmt.Errorf("scan provider secret for migration: %w", err)
		}
		normalized, changed, err := s.secrets.NormalizeStoredSecret(value)
		if err != nil {
			return fmt.Errorf("normalize provider secret for migration: %w", err)
		}
		if changed {
			pending = append(pending, migratedSecret{id: id, value: normalized})
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate provider secrets for migration: %w", err)
	}

	for _, item := range pending {
		if _, err := s.appDB.Exec(`
			UPDATE providers
			SET api_key = ?
			WHERE id = ?;
		`, item.value, item.id); err != nil {
			return fmt.Errorf("migrate provider secret %d: %w", item.id, err)
		}
	}
	return nil
}

func (s *configStore) Close() error {
	var errs []error
	if s.appDB != nil {
		err := s.appDB.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if s.ocrDB != nil {
		err := s.ocrDB.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *configStore) GetConfigSnapshot(ctx context.Context) (ConfigSnapshot, error) {
	providerRows, err := s.appDB.QueryContext(ctx, `
		SELECT id, name, type, base_url, region, api_key, is_active
		FROM providers
		ORDER BY type, name, id;
	`)
	if err != nil {
		return ConfigSnapshot{}, fmt.Errorf("list providers: %w", err)
	}
	defer providerRows.Close()

	configs := make([]ProviderConfig, 0)
	modelsByProvider := make(map[int64][]ModelRecord)

	modelRows, err := s.appDB.QueryContext(ctx, `
		SELECT id, provider_id, model_id, context_window
		FROM models
		ORDER BY provider_id, model_id, id;
	`)
	if err != nil {
		return ConfigSnapshot{}, fmt.Errorf("list models: %w", err)
	}
	defer modelRows.Close()

	for modelRows.Next() {
		var model ModelRecord
		if err := modelRows.Scan(&model.ID, &model.ProviderID, &model.ModelID, &model.ContextWindow); err != nil {
			return ConfigSnapshot{}, fmt.Errorf("scan model: %w", err)
		}
		modelsByProvider[model.ProviderID] = append(modelsByProvider[model.ProviderID], model)
	}
	if err := modelRows.Err(); err != nil {
		return ConfigSnapshot{}, fmt.Errorf("iterate models: %w", err)
	}

	for providerRows.Next() {
		var (
			provider      ProviderRecord
			providerType  string
			region        string
			encryptedKey  string
			isActiveValue int
		)

		if err := providerRows.Scan(&provider.ID, &provider.Name, &providerType, &provider.BaseURL, &region, &encryptedKey, &isActiveValue); err != nil {
			return ConfigSnapshot{}, fmt.Errorf("scan provider: %w", err)
		}

		provider.Type = ProviderType(providerType)
		provider.Region = region
		provider.HasAPIKey = encryptedKey != ""
		provider.APIKeyMasked = maskAPIKey(provider.HasAPIKey)
		provider.IsActive = isActiveValue == 1
		if !isValidProviderType(provider.Type) {
			continue
		}
		models := modelsByProvider[provider.ID]
		if models == nil {
			models = []ModelRecord{}
		}

		configs = append(configs, ProviderConfig{
			Provider: provider,
			Models:   models,
		})
	}

	if err := providerRows.Err(); err != nil {
		return ConfigSnapshot{}, fmt.Errorf("iterate providers: %w", err)
	}

	runtimeConfig, err := s.GetPDFTranslateRuntimeConfig(ctx)
	if err != nil {
		return ConfigSnapshot{}, fmt.Errorf("load pdf translate runtime config: %w", err)
	}

	return ConfigSnapshot{Providers: configs, PDFTranslateRuntime: runtimeConfig}, nil
}

func (s *configStore) SaveProvider(ctx context.Context, input ProviderUpsertInput) (ProviderRecord, error) {
	name := strings.TrimSpace(input.Name)
	baseURL := strings.TrimSpace(input.BaseURL)
	region := strings.TrimSpace(input.Region)
	if name == "" {
		return ProviderRecord{}, fmt.Errorf("provider name is required")
	}
	if !isValidProviderType(input.Type) {
		return ProviderRecord{}, fmt.Errorf("invalid provider type: %s", input.Type)
	}

	if input.ID == 0 {
		return s.createProvider(ctx, ProviderUpsertInput{
			Name:        name,
			Type:        input.Type,
			BaseURL:     baseURL,
			Region:      region,
			APIKey:      strings.TrimSpace(input.APIKey),
			ClearAPIKey: input.ClearAPIKey,
			IsActive:    input.IsActive,
		})
	}

	return s.updateProvider(ctx, ProviderUpsertInput{
		ID:          input.ID,
		Name:        name,
		Type:        input.Type,
		BaseURL:     baseURL,
		Region:      region,
		APIKey:      strings.TrimSpace(input.APIKey),
		ClearAPIKey: input.ClearAPIKey,
		IsActive:    input.IsActive,
	})
}

func (s *configStore) createProvider(ctx context.Context, input ProviderUpsertInput) (ProviderRecord, error) {
	encryptedKey, err := s.secrets.EncryptString(input.APIKey)
	if err != nil {
		return ProviderRecord{}, err
	}

	result, err := s.appDB.ExecContext(ctx, `
		INSERT INTO providers (name, type, base_url, region, api_key, is_active)
		VALUES (?, ?, ?, ?, ?, ?);
	`, input.Name, string(input.Type), input.BaseURL, input.Region, encryptedKey, boolToInt(input.IsActive))
	if err != nil {
		return ProviderRecord{}, fmt.Errorf("create provider: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return ProviderRecord{}, fmt.Errorf("read provider id: %w", err)
	}

	return ProviderRecord{
		ID:           id,
		Name:         input.Name,
		Type:         input.Type,
		BaseURL:      input.BaseURL,
		Region:       input.Region,
		HasAPIKey:    encryptedKey != "",
		APIKeyMasked: maskAPIKey(encryptedKey != ""),
		IsActive:     input.IsActive,
	}, nil
}

func (s *configStore) updateProvider(ctx context.Context, input ProviderUpsertInput) (ProviderRecord, error) {
	var existingKey string
	row := s.appDB.QueryRowContext(ctx, `SELECT api_key FROM providers WHERE id = ?;`, input.ID)
	if err := row.Scan(&existingKey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProviderRecord{}, fmt.Errorf("provider %d not found", input.ID)
		}
		return ProviderRecord{}, fmt.Errorf("load provider before update: %w", err)
	}

	finalKey := existingKey
	if input.ClearAPIKey {
		finalKey = ""
	} else if input.APIKey != "" {
		encryptedKey, err := s.secrets.EncryptString(input.APIKey)
		if err != nil {
			return ProviderRecord{}, err
		}
		finalKey = encryptedKey
	}

	result, err := s.appDB.ExecContext(ctx, `
		UPDATE providers
		SET name = ?, type = ?, base_url = ?, region = ?, api_key = ?, is_active = ?
		WHERE id = ?;
	`, input.Name, string(input.Type), input.BaseURL, input.Region, finalKey, boolToInt(input.IsActive), input.ID)
	if err != nil {
		return ProviderRecord{}, fmt.Errorf("update provider: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ProviderRecord{}, fmt.Errorf("check provider update result: %w", err)
	}
	if rowsAffected == 0 {
		return ProviderRecord{}, fmt.Errorf("provider %d not found", input.ID)
	}

	return ProviderRecord{
		ID:           input.ID,
		Name:         input.Name,
		Type:         input.Type,
		BaseURL:      input.BaseURL,
		Region:       input.Region,
		HasAPIKey:    finalKey != "",
		APIKeyMasked: maskAPIKey(finalKey != ""),
		IsActive:     input.IsActive,
	}, nil
}

func (s *configStore) DeleteProvider(ctx context.Context, id int64) error {
	result, err := s.appDB.ExecContext(ctx, `DELETE FROM providers WHERE id = ?;`, id)
	if err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check provider delete result: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("provider %d not found", id)
	}
	return nil
}

func (s *configStore) SaveModel(ctx context.Context, input ModelUpsertInput) (ModelRecord, error) {
	modelID := strings.TrimSpace(input.ModelID)
	if input.ProviderID == 0 {
		return ModelRecord{}, fmt.Errorf("provider id is required")
	}
	if modelID == "" {
		return ModelRecord{}, fmt.Errorf("model id is required")
	}
	if input.ContextWindow < 0 {
		return ModelRecord{}, fmt.Errorf("context window must be non-negative")
	}

	if err := s.ensureProviderExists(ctx, input.ProviderID); err != nil {
		return ModelRecord{}, err
	}

	existingID, err := s.lookupModelID(ctx, input.ProviderID, modelID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ModelRecord{}, err
	}
	if err == nil && existingID != input.ID {
		return ModelRecord{}, fmt.Errorf("model %s already exists for provider %d", modelID, input.ProviderID)
	}

	if input.ID == 0 {
		result, err := s.appDB.ExecContext(ctx, `
			INSERT INTO models (provider_id, model_id, context_window)
			VALUES (?, ?, ?);
		`, input.ProviderID, modelID, input.ContextWindow)
		if err != nil {
			return ModelRecord{}, fmt.Errorf("create model: %w", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return ModelRecord{}, fmt.Errorf("read model id: %w", err)
		}
		return ModelRecord{ID: id, ProviderID: input.ProviderID, ModelID: modelID, ContextWindow: input.ContextWindow}, nil
	}

	result, err := s.appDB.ExecContext(ctx, `
		UPDATE models
		SET provider_id = ?, model_id = ?, context_window = ?
		WHERE id = ?;
	`, input.ProviderID, modelID, input.ContextWindow, input.ID)
	if err != nil {
		return ModelRecord{}, fmt.Errorf("update model: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ModelRecord{}, fmt.Errorf("check model update result: %w", err)
	}
	if rowsAffected == 0 {
		return ModelRecord{}, fmt.Errorf("model %d not found", input.ID)
	}

	return ModelRecord{ID: input.ID, ProviderID: input.ProviderID, ModelID: modelID, ContextWindow: input.ContextWindow}, nil
}

func (s *configStore) DeleteModel(ctx context.Context, id int64) error {
	result, err := s.appDB.ExecContext(ctx, `DELETE FROM models WHERE id = ?;`, id)
	if err != nil {
		return fmt.Errorf("delete model: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check model delete result: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("model %d not found", id)
	}
	return nil
}

func (s *configStore) lookupModelID(ctx context.Context, providerID int64, modelID string) (int64, error) {
	var id int64
	err := s.appDB.QueryRowContext(ctx, `
		SELECT id
		FROM models
		WHERE provider_id = ? AND model_id = ?
		ORDER BY id
		LIMIT 1;
	`, providerID, modelID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, sql.ErrNoRows
		}
		return 0, fmt.Errorf("lookup model %s for provider %d: %w", modelID, providerID, err)
	}
	return id, nil
}

func (s *configStore) ensureProviderExists(ctx context.Context, id int64) error {
	var exists int
	err := s.appDB.QueryRowContext(ctx, `SELECT 1 FROM providers WHERE id = ?;`, id).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("provider %d not found", id)
		}
		return fmt.Errorf("check provider existence: %w", err)
	}
	return nil
}

func (s *configStore) GetProviderSecret(ctx context.Context, id int64) (providerSecretRecord, error) {
	row := s.appDB.QueryRowContext(ctx, `
		SELECT id, name, type, base_url, region, api_key, is_active
		FROM providers
		WHERE id = ?;
	`, id)

	var (
		record       providerSecretRecord
		providerType string
		region       string
		encryptedKey string
		isActiveInt  int
	)

	if err := row.Scan(&record.ID, &record.Name, &providerType, &record.BaseURL, &region, &encryptedKey, &isActiveInt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return providerSecretRecord{}, fmt.Errorf("provider %d not found", id)
		}
		return providerSecretRecord{}, fmt.Errorf("load provider: %w", err)
	}

	record.Type = ProviderType(providerType)
	record.Region = region
	record.HasAPIKey = encryptedKey != ""
	record.APIKeyMasked = maskAPIKey(record.HasAPIKey)
	record.IsActive = isActiveInt == 1

	decrypted, err := s.secrets.DecryptString(encryptedKey)
	if err != nil {
		return providerSecretRecord{}, fmt.Errorf("decrypt provider api key: %w", err)
	}
	record.APIKey = decrypted

	return record, nil
}

func (s *configStore) GetModel(ctx context.Context, id int64) (ModelRecord, error) {
	row := s.appDB.QueryRowContext(ctx, `
		SELECT id, provider_id, model_id, context_window
		FROM models
		WHERE id = ?;
	`, id)

	var record ModelRecord
	if err := row.Scan(&record.ID, &record.ProviderID, &record.ModelID, &record.ContextWindow); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ModelRecord{}, fmt.Errorf("model %d not found", id)
		}
		return ModelRecord{}, fmt.Errorf("load model: %w", err)
	}

	return record, nil
}

func (s *configStore) GetOCRResult(ctx context.Context, pdfHash string, pageNumber int) (OCRPageResult, error) {
	row := s.ocrDB.QueryRowContext(ctx, `
		SELECT id, pdf_hash, page_number, resolution, json_data, created_at
		FROM page_ocr_results
		WHERE pdf_hash = ? AND page_number = ?
		ORDER BY id DESC
		LIMIT 1;
	`, strings.TrimSpace(pdfHash), pageNumber)

	var (
		result   OCRPageResult
		jsonData string
	)
	if err := row.Scan(&result.ID, &result.PDFHash, &result.PageNumber, &result.Resolution, &jsonData, &result.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OCRPageResult{}, fmt.Errorf("ocr cache miss")
		}
		return OCRPageResult{}, fmt.Errorf("load ocr cache: %w", err)
	}
	if err := json.Unmarshal([]byte(jsonData), &result.Blocks); err != nil {
		return OCRPageResult{}, fmt.Errorf("decode ocr cache: %w", err)
	}
	return result, nil
}

func (s *configStore) SaveOCRResult(ctx context.Context, result OCRPageResult) (OCRPageResult, error) {
	encoded, err := json.Marshal(result.Blocks)
	if err != nil {
		return OCRPageResult{}, fmt.Errorf("encode ocr cache: %w", err)
	}
	createdAt := time.Now().UTC().Format(time.RFC3339)
	insertResult, err := s.ocrDB.ExecContext(ctx, `
		INSERT INTO page_ocr_results (pdf_hash, page_number, resolution, json_data, created_at)
		VALUES (?, ?, ?, ?, ?);
	`, strings.TrimSpace(result.PDFHash), result.PageNumber, result.Resolution, string(encoded), createdAt)
	if err != nil {
		return OCRPageResult{}, fmt.Errorf("save ocr cache: %w", err)
	}
	id, err := insertResult.LastInsertId()
	if err != nil {
		return OCRPageResult{}, fmt.Errorf("read ocr cache id: %w", err)
	}
	result.ID = id
	result.CreatedAt = createdAt
	return result, nil
}

type pdfMarkdownCacheRecord struct {
	PDFHash          string
	Extractor        string
	ExtractorVersion string
	Payload          PDFMarkdownPayload
}

func (s *configStore) GetPDFMarkdownCache(ctx context.Context, pdfHash, extractor, extractorVersion string) (PDFMarkdownPayload, bool, error) {
	row := s.appDB.QueryRowContext(ctx, `
		SELECT json_data
		FROM pdf_markdown_cache
		WHERE pdf_hash = ? AND extractor = ? AND extractor_version = ?
		LIMIT 1;
	`, strings.TrimSpace(pdfHash), strings.TrimSpace(extractor), strings.TrimSpace(extractorVersion))

	var raw string
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PDFMarkdownPayload{}, false, nil
		}
		return PDFMarkdownPayload{}, false, fmt.Errorf("query pdf markdown cache: %w", err)
	}

	var payload PDFMarkdownPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return PDFMarkdownPayload{}, false, fmt.Errorf("decode pdf markdown cache: %w", err)
	}
	payload.Cached = true
	return payload, true, nil
}

func (s *configStore) SavePDFMarkdownCache(ctx context.Context, record pdfMarkdownCacheRecord) (PDFMarkdownPayload, error) {
	payload := record.Payload
	payload.Cached = false
	if strings.TrimSpace(payload.GeneratedAt) == "" {
		payload.GeneratedAt = nowRFC3339()
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return PDFMarkdownPayload{}, fmt.Errorf("encode pdf markdown cache: %w", err)
	}

	if _, err := s.appDB.ExecContext(ctx, `
		INSERT INTO pdf_markdown_cache (pdf_hash, extractor, extractor_version, json_data, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(pdf_hash, extractor, extractor_version) DO UPDATE SET
			json_data = excluded.json_data,
			created_at = excluded.created_at;
	`, strings.TrimSpace(record.PDFHash), strings.TrimSpace(record.Extractor), strings.TrimSpace(record.ExtractorVersion), string(encoded), payload.GeneratedAt); err != nil {
		return PDFMarkdownPayload{}, fmt.Errorf("save pdf markdown cache: %w", err)
	}

	return payload, nil
}

func (s *configStore) CreateWorkspace(ctx context.Context, input WorkspaceUpsertInput) (Workspace, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Workspace{}, fmt.Errorf("workspace name is required")
	}

	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = newEntityID("ws")
	}
	createdAt := nowRFC3339()
	workspace := Workspace{
		ID:          id,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Color:       strings.TrimSpace(input.Color),
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}

	if _, err := s.appDB.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, description, color, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?);
	`, workspace.ID, workspace.Name, workspace.Description, workspace.Color, workspace.CreatedAt, workspace.UpdatedAt); err != nil {
		return Workspace{}, fmt.Errorf("create workspace: %w", err)
	}

	return workspace, nil
}

func (s *configStore) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	rows, err := s.appDB.QueryContext(ctx, `
		SELECT id, name, description, color, created_at, updated_at
		FROM workspaces
		ORDER BY updated_at DESC, created_at DESC, id DESC;
	`)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	workspaces := []Workspace{}
	for rows.Next() {
		var workspace Workspace
		if err := rows.Scan(&workspace.ID, &workspace.Name, &workspace.Description, &workspace.Color, &workspace.CreatedAt, &workspace.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace: %w", err)
		}
		workspaces = append(workspaces, workspace)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspaces: %w", err)
	}
	return workspaces, nil
}

func (s *configStore) GetWorkspace(ctx context.Context, workspaceID string) (Workspace, error) {
	row := s.appDB.QueryRowContext(ctx, `
		SELECT id, name, description, color, created_at, updated_at
		FROM workspaces
		WHERE id = ?;
	`, strings.TrimSpace(workspaceID))

	var workspace Workspace
	if err := row.Scan(&workspace.ID, &workspace.Name, &workspace.Description, &workspace.Color, &workspace.CreatedAt, &workspace.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Workspace{}, fmt.Errorf("workspace %s not found", strings.TrimSpace(workspaceID))
		}
		return Workspace{}, fmt.Errorf("load workspace: %w", err)
	}
	return workspace, nil
}

func (s *configStore) CreateWorkspaceAgentSession(ctx context.Context, input WorkspaceAgentSessionCreateInput) (WorkspaceAgentSession, error) {
	session, err := newWorkspaceAgentSessionRecord(ctx, func(ctx context.Context, workspaceID string) error {
		_, err := s.GetWorkspace(ctx, workspaceID)
		return err
	}, input)
	if err != nil {
		return WorkspaceAgentSession{}, err
	}
	if _, err := s.appDB.ExecContext(ctx, `
		INSERT INTO workspace_agent_sessions (id, workspace_id, title, surface, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`, session.ID, session.WorkspaceID, session.Title, session.Surface, session.Status, session.CreatedAt, session.UpdatedAt); err != nil {
		return WorkspaceAgentSession{}, fmt.Errorf("create workspace agent session: %w", err)
	}

	return session, nil
}

func (s *configStore) AppendWorkspaceAgentMessage(ctx context.Context, input WorkspaceAgentMessageCreateInput) (WorkspaceAgentMessage, error) {
	message, err := newWorkspaceAgentMessageRecord(ctx, s.getWorkspaceAgentSession, input)
	if err != nil {
		return WorkspaceAgentMessage{}, err
	}

	result, err := s.appDB.ExecContext(ctx, `
		INSERT INTO workspace_agent_messages (session_id, workspace_id, surface, role, kind, prompt, content, skill_name, evidence_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, message.SessionID, message.WorkspaceID, message.Surface, message.Role, message.Kind, message.Prompt, message.Content, message.SkillName, message.EvidenceCount, message.CreatedAt)
	if err != nil {
		return WorkspaceAgentMessage{}, fmt.Errorf("append workspace agent message: %w", err)
	}
	messageID, err := result.LastInsertId()
	if err != nil {
		return WorkspaceAgentMessage{}, fmt.Errorf("read workspace agent message id: %w", err)
	}
	message.ID = messageID
	if _, err := s.appDB.ExecContext(ctx, `
		UPDATE workspace_agent_sessions
		SET updated_at = ?
		WHERE id = ? AND workspace_id = ?;
	`, message.CreatedAt, message.SessionID, message.WorkspaceID); err != nil {
		return WorkspaceAgentMessage{}, fmt.Errorf("touch workspace agent session: %w", err)
	}

	return message, nil
}

func (s *configStore) CreateWorkspaceAgentSessionTx(ctx context.Context, tx *sql.Tx, input WorkspaceAgentSessionCreateInput) (WorkspaceAgentSession, error) {
	session, err := newWorkspaceAgentSessionRecord(ctx, func(ctx context.Context, workspaceID string) error {
		_, err := getWorkspaceRecordTx(ctx, tx, workspaceID)
		return err
	}, input)
	if err != nil {
		return WorkspaceAgentSession{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO workspace_agent_sessions (id, workspace_id, title, surface, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`, session.ID, session.WorkspaceID, session.Title, session.Surface, session.Status, session.CreatedAt, session.UpdatedAt); err != nil {
		return WorkspaceAgentSession{}, fmt.Errorf("create workspace agent session: %w", err)
	}
	return session, nil
}

func (s *configStore) AppendWorkspaceAgentMessageTx(ctx context.Context, tx *sql.Tx, input WorkspaceAgentMessageCreateInput) (WorkspaceAgentMessage, error) {
	message, err := newWorkspaceAgentMessageRecord(ctx, func(ctx context.Context, workspaceID, sessionID string) (WorkspaceAgentSession, error) {
		return getWorkspaceAgentSessionTx(ctx, tx, workspaceID, sessionID)
	}, input)
	if err != nil {
		return WorkspaceAgentMessage{}, err
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO workspace_agent_messages (session_id, workspace_id, surface, role, kind, prompt, content, skill_name, evidence_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, message.SessionID, message.WorkspaceID, message.Surface, message.Role, message.Kind, message.Prompt, message.Content, message.SkillName, message.EvidenceCount, message.CreatedAt)
	if err != nil {
		return WorkspaceAgentMessage{}, fmt.Errorf("append workspace agent message: %w", err)
	}
	messageID, err := result.LastInsertId()
	if err != nil {
		return WorkspaceAgentMessage{}, fmt.Errorf("read workspace agent message id: %w", err)
	}
	message.ID = messageID
	if _, err := tx.ExecContext(ctx, `
		UPDATE workspace_agent_sessions
		SET updated_at = ?
		WHERE id = ? AND workspace_id = ?;
	`, message.CreatedAt, message.SessionID, message.WorkspaceID); err != nil {
		return WorkspaceAgentMessage{}, fmt.Errorf("touch workspace agent session: %w", err)
	}
	return message, nil
}

func newWorkspaceAgentSessionRecord(ctx context.Context, ensureWorkspace func(context.Context, string) error, input WorkspaceAgentSessionCreateInput) (WorkspaceAgentSession, error) {
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID == "" {
		return WorkspaceAgentSession{}, fmt.Errorf("workspace id is required")
	}
	if err := ensureWorkspace(ctx, workspaceID); err != nil {
		return WorkspaceAgentSession{}, err
	}

	title := strings.TrimSpace(input.Title)
	surface, err := normalizeWorkspaceAgentSurface(input.Surface)
	if err != nil {
		return WorkspaceAgentSession{}, err
	}

	timestamp := nowRFC3339()
	session := WorkspaceAgentSession{
		ID:          newEntityID("was"),
		WorkspaceID: workspaceID,
		Title:       title,
		Surface:     surface,
		Status:      "active",
		CreatedAt:   timestamp,
		UpdatedAt:   timestamp,
	}
	return session, nil
}

func getWorkspaceRecordTx(ctx context.Context, tx *sql.Tx, workspaceID string) (Workspace, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, name, description, color, created_at, updated_at
		FROM workspaces
		WHERE id = ?;
	`, strings.TrimSpace(workspaceID))

	var workspace Workspace
	if err := row.Scan(&workspace.ID, &workspace.Name, &workspace.Description, &workspace.Color, &workspace.CreatedAt, &workspace.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Workspace{}, fmt.Errorf("workspace %s not found", strings.TrimSpace(workspaceID))
		}
		return Workspace{}, fmt.Errorf("load workspace: %w", err)
	}
	return workspace, nil
}

func newWorkspaceAgentMessageRecord(ctx context.Context, getSession func(context.Context, string, string) (WorkspaceAgentSession, error), input WorkspaceAgentMessageCreateInput) (WorkspaceAgentMessage, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return WorkspaceAgentMessage{}, fmt.Errorf("session id is required")
	}
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID == "" {
		return WorkspaceAgentMessage{}, fmt.Errorf("workspace id is required")
	}
	session, err := getSession(ctx, workspaceID, sessionID)
	if err != nil {
		return WorkspaceAgentMessage{}, err
	}
	rawSurface := strings.TrimSpace(input.Surface)
	surface := session.Surface
	if rawSurface != "" {
		surface, err = normalizeWorkspaceAgentSurface(rawSurface)
		if err != nil {
			return WorkspaceAgentMessage{}, err
		}
	}
	if surface == "" {
		surface = session.Surface
	}
	if surface == "" {
		surface = string(WorkspaceAgentSurfaceWorkspace)
	}
	role := strings.TrimSpace(input.Role)
	if !isValidWorkspaceAgentMessageRole(role) {
		return WorkspaceAgentMessage{}, fmt.Errorf("invalid workspace agent role: %s", role)
	}
	kind := strings.TrimSpace(input.Kind)
	if kind == "" {
		kind = "message"
	}
	content := strings.TrimSpace(input.Content)

	message := WorkspaceAgentMessage{
		SessionID:     sessionID,
		WorkspaceID:   workspaceID,
		Surface:       surface,
		Role:          role,
		Kind:          kind,
		Prompt:        strings.TrimSpace(input.Prompt),
		Content:       content,
		SkillName:     strings.TrimSpace(input.SkillName),
		EvidenceCount: input.EvidenceCount,
		CreatedAt:     nowRFC3339(),
	}
	return message, nil
}

func getWorkspaceAgentSessionTx(ctx context.Context, tx *sql.Tx, workspaceID, sessionID string) (WorkspaceAgentSession, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, workspace_id, title, surface, status, created_at, updated_at
		FROM workspace_agent_sessions
		WHERE id = ? AND workspace_id = ?;
	`, strings.TrimSpace(sessionID), strings.TrimSpace(workspaceID))

	var session WorkspaceAgentSession
	if err := row.Scan(&session.ID, &session.WorkspaceID, &session.Title, &session.Surface, &session.Status, &session.CreatedAt, &session.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WorkspaceAgentSession{}, fmt.Errorf("workspace agent session %s not found in workspace %s", strings.TrimSpace(sessionID), strings.TrimSpace(workspaceID))
		}
		return WorkspaceAgentSession{}, fmt.Errorf("load workspace agent session: %w", err)
	}
	return session, nil
}

func (s *configStore) ListWorkspaceAgentSessions(ctx context.Context, workspaceID string) ([]WorkspaceAgentSession, error) {
	rows, err := s.appDB.QueryContext(ctx, `
		SELECT s.id, s.workspace_id, s.title, s.surface, s.status, s.created_at, s.updated_at, COALESCE(MAX(m.id), 0) AS last_message_order, s.rowid
		FROM workspace_agent_sessions s
		LEFT JOIN workspace_agent_messages m ON m.session_id = s.id
		WHERE s.workspace_id = ?
		GROUP BY s.id, s.workspace_id, s.title, s.surface, s.status, s.created_at, s.updated_at, s.rowid
		ORDER BY last_message_order DESC, s.updated_at DESC, s.created_at DESC, s.rowid DESC;
	`, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("list workspace agent sessions: %w", err)
	}
	defer rows.Close()

	sessions := []WorkspaceAgentSession{}
	for rows.Next() {
		var session WorkspaceAgentSession
		var (
			lastMessageOrder int64
			sessionRowID     int64
		)
		if err := rows.Scan(&session.ID, &session.WorkspaceID, &session.Title, &session.Surface, &session.Status, &session.CreatedAt, &session.UpdatedAt, &lastMessageOrder, &sessionRowID); err != nil {
			return nil, fmt.Errorf("scan workspace agent session: %w", err)
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace agent sessions: %w", err)
	}
	return sessions, nil
}

func (s *configStore) ListWorkspaceAgentMessages(ctx context.Context, sessionID string) ([]WorkspaceAgentMessage, error) {
	rows, err := s.appDB.QueryContext(ctx, `
		SELECT id, session_id, workspace_id, surface, role, kind, prompt, content, skill_name, evidence_count, created_at
		FROM workspace_agent_messages
		WHERE session_id = ?
		ORDER BY id ASC;
	`, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, fmt.Errorf("list workspace agent messages: %w", err)
	}
	defer rows.Close()

	messages := []WorkspaceAgentMessage{}
	for rows.Next() {
		var message WorkspaceAgentMessage
		if err := rows.Scan(&message.ID, &message.SessionID, &message.WorkspaceID, &message.Surface, &message.Role, &message.Kind, &message.Prompt, &message.Content, &message.SkillName, &message.EvidenceCount, &message.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace agent message: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace agent messages: %w", err)
	}
	return messages, nil
}

func (s *configStore) ListWorkspaceAgentMessagesForWorkspace(ctx context.Context, workspaceID, sessionID string) ([]WorkspaceAgentMessage, error) {
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	trimmedSessionID := strings.TrimSpace(sessionID)
	if _, err := s.getWorkspaceAgentSession(ctx, trimmedWorkspaceID, trimmedSessionID); err != nil {
		return nil, err
	}

	rows, err := s.appDB.QueryContext(ctx, `
		SELECT id, session_id, workspace_id, surface, role, kind, prompt, content, skill_name, evidence_count, created_at
		FROM workspace_agent_messages
		WHERE session_id = ? AND workspace_id = ?
		ORDER BY id ASC;
	`, trimmedSessionID, trimmedWorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace agent messages: %w", err)
	}
	defer rows.Close()

	messages := []WorkspaceAgentMessage{}
	for rows.Next() {
		var message WorkspaceAgentMessage
		if err := rows.Scan(&message.ID, &message.SessionID, &message.WorkspaceID, &message.Surface, &message.Role, &message.Kind, &message.Prompt, &message.Content, &message.SkillName, &message.EvidenceCount, &message.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace agent message: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace agent messages: %w", err)
	}
	return messages, nil
}

func (s *configStore) getWorkspaceAgentSession(ctx context.Context, workspaceID, sessionID string) (WorkspaceAgentSession, error) {
	row := s.appDB.QueryRowContext(ctx, `
		SELECT id, workspace_id, title, surface, status, created_at, updated_at
		FROM workspace_agent_sessions
		WHERE id = ? AND workspace_id = ?;
	`, strings.TrimSpace(sessionID), strings.TrimSpace(workspaceID))

	var session WorkspaceAgentSession
	if err := row.Scan(&session.ID, &session.WorkspaceID, &session.Title, &session.Surface, &session.Status, &session.CreatedAt, &session.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WorkspaceAgentSession{}, fmt.Errorf("workspace agent session %s not found in workspace %s", strings.TrimSpace(sessionID), strings.TrimSpace(workspaceID))
		}
		return WorkspaceAgentSession{}, fmt.Errorf("load workspace agent session: %w", err)
	}
	return session, nil
}

func (s *configStore) ListDocumentsByWorkspace(ctx context.Context, workspaceID string) ([]DocumentRecord, error) {
	rows, err := s.appDB.QueryContext(ctx, `
		SELECT id, workspace_id, title, document_type, source_type, default_asset_id, original_file_name, primary_pdf_path, content_hash, created_at, updated_at
		FROM documents
		WHERE workspace_id = ?
		ORDER BY updated_at DESC, created_at DESC, id DESC;
	`, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	documents := []DocumentRecord{}
	for rows.Next() {
		var document DocumentRecord
		if err := rows.Scan(
			&document.ID,
			&document.WorkspaceID,
			&document.Title,
			&document.DocumentType,
			&document.SourceType,
			&document.DefaultAssetID,
			&document.OriginalFileName,
			&document.PrimaryPDFPath,
			&document.ContentHash,
			&document.CreatedAt,
			&document.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		documents = append(documents, document)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate documents: %w", err)
	}
	return documents, nil
}

func (s *configStore) DeleteDocument(ctx context.Context, paths appPaths, workspaceID, documentID string) error {
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	trimmedDocumentID := strings.TrimSpace(documentID)
	if trimmedWorkspaceID == "" {
		return fmt.Errorf("workspace id is required")
	}
	if trimmedDocumentID == "" {
		return fmt.Errorf("document id is required")
	}

	if _, err := s.GetWorkspace(ctx, trimmedWorkspaceID); err != nil {
		return err
	}

	var existingDocumentID string
	var primaryPDFPath string
	if err := s.appDB.QueryRowContext(ctx, `
		SELECT id, primary_pdf_path
		FROM documents
		WHERE id = ? AND workspace_id = ?
		LIMIT 1;
	`, trimmedDocumentID, trimmedWorkspaceID).Scan(&existingDocumentID, &primaryPDFPath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("document %s not found in workspace %s", trimmedDocumentID, trimmedWorkspaceID)
		}
		return fmt.Errorf("load document before delete: %w", err)
	}

	deletedDocumentKnownPaths, err := s.workspaceKnowledgeKnownPathsForDocument(ctx, paths, trimmedWorkspaceID, trimmedDocumentID, primaryPDFPath)
	if err != nil {
		return err
	}

	files := newWorkspaceKnowledgeFiles(paths, trimmedWorkspaceID)
	manifest, err := files.ReadSources()
	if err != nil {
		return fmt.Errorf("read workspace knowledge sources before delete: %w", err)
	}
	remainingSources := make([]WorkspaceKnowledgeSource, 0, len(manifest))
	deletedSources := make([]WorkspaceKnowledgeSource, 0)
	for _, source := range manifest {
		if strings.TrimSpace(source.DocumentID) == trimmedDocumentID || workspaceKnowledgeSourceMatchesDeletedDocumentPaths(source, deletedDocumentKnownPaths) {
			deletedSources = append(deletedSources, source)
			continue
		}
		remainingSources = append(remainingSources, source)
	}

	documentRoot := filepath.Join(paths.WorkspacesRootDir, trimmedWorkspaceID, "documents", trimmedDocumentID)
	if err := os.RemoveAll(documentRoot); err != nil {
		return fmt.Errorf("remove document files: %w", err)
	}

	tx, err := s.appDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin document delete transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `
		DELETE FROM ai_chat_history
		WHERE workspace_id = ? AND document_id = ?;
	`, trimmedWorkspaceID, trimmedDocumentID); err != nil {
		return fmt.Errorf("delete document chat history: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `
		DELETE FROM reader_notes
		WHERE workspace_id = ? AND document_id = ?;
	`, trimmedWorkspaceID, trimmedDocumentID); err != nil {
		return fmt.Errorf("delete document reader notes: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `
		DELETE FROM workspace_wiki_pages
		WHERE workspace_id = ?;
	`, trimmedWorkspaceID); err != nil {
		return fmt.Errorf("delete document wiki pages: %w", err)
	}

	deleteResult, err := tx.ExecContext(ctx, `
		DELETE FROM documents
		WHERE id = ? AND workspace_id = ?;
	`, trimmedDocumentID, trimmedWorkspaceID)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
	}

	rowsAffected, err := deleteResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("check document delete result: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("document %s not found in workspace %s", trimmedDocumentID, trimmedWorkspaceID)
	}

	if _, err = tx.ExecContext(ctx, `
		UPDATE workspaces
		SET updated_at = ?
		WHERE id = ?;
	`, nowRFC3339(), trimmedWorkspaceID); err != nil {
		return fmt.Errorf("touch workspace after document delete: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit document delete transaction: %w", err)
	}

	if err := files.WriteSources(remainingSources); err != nil {
		return fmt.Errorf("update workspace knowledge sources after delete: %w", err)
	}
	for _, source := range deletedSources {
		if err := files.DeleteMarkItDown(source.Slug); err != nil {
			return fmt.Errorf("delete workspace knowledge markdown for %s: %w", source.Slug, err)
		}
		if err := files.DeleteBySource(source.Slug); err != nil {
			return fmt.Errorf("delete workspace knowledge by-source payload for %s: %w", source.Slug, err)
		}
	}
	if err := files.DeleteCompiledArtifacts(); err != nil {
		return fmt.Errorf("invalidate workspace compiled knowledge after document delete: %w", err)
	}
	return nil
}

func (s *configStore) workspaceKnowledgeKnownPathsForDocument(ctx context.Context, paths appPaths, workspaceID, documentID, primaryPDFPath string) (map[string]struct{}, error) {
	knownPaths := make(map[string]struct{})
	addWorkspaceKnowledgeKnownPath(knownPaths, primaryPDFPath)

	rows, err := s.appDB.QueryContext(ctx, `
		SELECT absolute_path, relative_path
		FROM document_assets
		WHERE workspace_id = ? AND document_id = ?;
	`, strings.TrimSpace(workspaceID), strings.TrimSpace(documentID))
	if err != nil {
		return nil, fmt.Errorf("list document assets before delete: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var absolutePath string
		var relativePath string
		if err := rows.Scan(&absolutePath, &relativePath); err != nil {
			return nil, fmt.Errorf("scan document asset before delete: %w", err)
		}
		addWorkspaceKnowledgeKnownPath(knownPaths, absolutePath)
		trimmedRelativePath := strings.TrimSpace(relativePath)
		if trimmedRelativePath != "" {
			assetPath := filepath.FromSlash(trimmedRelativePath)
			if !filepath.IsAbs(assetPath) {
				assetPath = filepath.Join(paths.RootDir, assetPath)
			}
			addWorkspaceKnowledgeKnownPath(knownPaths, assetPath)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate document assets before delete: %w", err)
	}
	return knownPaths, nil
}

func workspaceKnowledgeSourceMatchesDeletedDocumentPaths(source WorkspaceKnowledgeSource, knownPaths map[string]struct{}) bool {
	for _, candidate := range []string{source.SourcePath, source.AbsolutePath} {
		normalizedPath := normalizeWorkspaceKnowledgeAbsolutePath(candidate)
		if normalizedPath == "" {
			continue
		}
		if _, ok := knownPaths[normalizedPath]; ok {
			return true
		}
	}
	return false
}

func addWorkspaceKnowledgeKnownPath(knownPaths map[string]struct{}, path string) {
	normalizedPath := normalizeWorkspaceKnowledgeAbsolutePath(path)
	if normalizedPath == "" {
		return
	}
	knownPaths[normalizedPath] = struct{}{}
}

func (s *configStore) ImportFiles(ctx context.Context, paths appPaths, input ImportFilesInput) (ImportFilesResult, error) {
	workspace, err := s.GetWorkspace(ctx, input.WorkspaceID)
	if err != nil {
		return ImportFilesResult{}, err
	}
	if len(input.FilePaths) == 0 {
		return ImportFilesResult{}, fmt.Errorf("at least one file path is required")
	}

	workspaceRoot := filepath.Join(paths.WorkspacesRootDir, workspace.ID)
	if err := os.MkdirAll(workspaceRoot, 0o700); err != nil {
		return ImportFilesResult{}, fmt.Errorf("create workspace root: %w", err)
	}

	result := ImportFilesResult{Workspace: workspace}
	sourceType := strings.TrimSpace(input.SourceType)
	if sourceType == "" {
		sourceType = "manual"
	}
	sourceLabel := strings.TrimSpace(input.SourceLabel)
	sourceRef := strings.TrimSpace(input.SourceRef)
	for _, rawPath := range input.FilePaths {
		importedDocument, importRecord, err := s.importSingleFile(ctx, paths, workspace, rawPath, sourceType, sourceLabel, sourceRef, input.Title)
		if err != nil {
			return ImportFilesResult{}, err
		}
		result.Documents = append(result.Documents, importedDocument)
		result.Imports = append(result.Imports, importRecord)
	}

	return result, nil
}

func (s *configStore) importSingleFile(ctx context.Context, paths appPaths, workspace Workspace, rawPath, sourceType, sourceLabel, sourceRef, preferredTitle string) (DocumentRecord, ImportRecord, error) {
	filePath := filepath.Clean(strings.TrimSpace(rawPath))
	if filePath == "" {
		return DocumentRecord{}, ImportRecord{}, fmt.Errorf("file path is required")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return DocumentRecord{}, ImportRecord{}, fmt.Errorf("stat import file %s: %w", filePath, err)
	}
	if info.IsDir() {
		return DocumentRecord{}, ImportRecord{}, fmt.Errorf("import file %s is a directory", filePath)
	}

	contentHash, err := sha256File(filePath)
	if err != nil {
		return DocumentRecord{}, ImportRecord{}, fmt.Errorf("hash import file %s: %w", filePath, err)
	}

	title := strings.TrimSpace(preferredTitle)
	if title == "" {
		title = strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
	}
	documentID := newEntityID("doc")
	assetID := newEntityID("asset")
	importID := newEntityID("import")
	createdAt := nowRFC3339()
	assetFileName := sanitizeFileName(info.Name())
	assetRelativePath := filepath.Join("library", "workspaces", workspace.ID, "documents", documentID, "assets", "original", assetFileName)
	assetAbsolutePath := filepath.Join(paths.RootDir, assetRelativePath)
	assetDir := filepath.Dir(assetAbsolutePath)
	if err := os.MkdirAll(assetDir, 0o700); err != nil {
		return DocumentRecord{}, ImportRecord{}, fmt.Errorf("create asset directory: %w", err)
	}
	if err := copyFile(filePath, assetAbsolutePath); err != nil {
		return DocumentRecord{}, ImportRecord{}, fmt.Errorf("copy import file: %w", err)
	}

	document := DocumentRecord{
		ID:               documentID,
		WorkspaceID:      workspace.ID,
		Title:            title,
		DocumentType:     "paper",
		SourceType:       sourceType,
		DefaultAssetID:   assetID,
		OriginalFileName: info.Name(),
		PrimaryPDFPath:   assetAbsolutePath,
		ContentHash:      contentHash,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}
	asset := DocumentAssetRecord{
		ID:           assetID,
		DocumentID:   documentID,
		WorkspaceID:  workspace.ID,
		Kind:         detectAssetKind(info.Name()),
		Role:         "original",
		FileName:     assetFileName,
		RelativePath: filepath.ToSlash(assetRelativePath),
		AbsolutePath: assetAbsolutePath,
		MimeType:     detectMimeType(info.Name()),
		ByteSize:     info.Size(),
		ContentHash:  contentHash,
		CreatedAt:    createdAt,
	}
	importRecord := ImportRecord{
		ID:          importID,
		WorkspaceID: workspace.ID,
		DocumentID:  documentID,
		SourceType:  sourceType,
		SourceLabel: firstNonEmpty(sourceLabel, sourceType),
		SourceRef:   firstNonEmpty(sourceRef, filePath),
		Status:      "completed",
		Message:     "",
		CreatedAt:   createdAt,
	}

	if err := s.saveImportedDocument(ctx, workspace.ID, document, asset, importRecord); err != nil {
		return DocumentRecord{}, ImportRecord{}, err
	}
	if sourceType == "zotero" {
		link := DocumentExternalLink{
			ID:          newEntityID("link"),
			DocumentID:  document.ID,
			WorkspaceID: workspace.ID,
			Provider:    "zotero",
			ExternalID:  strings.TrimSpace(sourceRef),
			ExternalKey: strings.TrimSpace(sourceLabel),
			CreatedAt:   createdAt,
		}
		if err := s.saveDocumentExternalLink(ctx, link); err != nil {
			return DocumentRecord{}, ImportRecord{}, err
		}
	}
	if err := s.touchWorkspace(ctx, workspace.ID, createdAt); err != nil {
		return DocumentRecord{}, ImportRecord{}, err
	}

	return document, importRecord, nil
}

func (s *configStore) saveImportedDocument(ctx context.Context, workspaceID string, document DocumentRecord, asset DocumentAssetRecord, importRecord ImportRecord) error {
	tx, err := s.appDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin import transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO documents (id, workspace_id, title, document_type, source_type, default_asset_id, original_file_name, primary_pdf_path, content_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, document.ID, document.WorkspaceID, document.Title, document.DocumentType, document.SourceType, document.DefaultAssetID, document.OriginalFileName, document.PrimaryPDFPath, document.ContentHash, document.CreatedAt, document.UpdatedAt); err != nil {
		return fmt.Errorf("save document: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO document_assets (id, document_id, workspace_id, kind, role, file_name, relative_path, absolute_path, mime_type, byte_size, content_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, asset.ID, asset.DocumentID, asset.WorkspaceID, asset.Kind, asset.Role, asset.FileName, asset.RelativePath, asset.AbsolutePath, asset.MimeType, asset.ByteSize, asset.ContentHash, asset.CreatedAt); err != nil {
		return fmt.Errorf("save document asset: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO import_records (id, workspace_id, document_id, source_type, source_label, source_ref, status, message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, importRecord.ID, importRecord.WorkspaceID, importRecord.DocumentID, importRecord.SourceType, importRecord.SourceLabel, importRecord.SourceRef, importRecord.Status, importRecord.Message, importRecord.CreatedAt); err != nil {
		return fmt.Errorf("save import record: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `
		UPDATE workspaces
		SET updated_at = ?
		WHERE id = ?;
	`, document.UpdatedAt, workspaceID); err != nil {
		return fmt.Errorf("touch workspace: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit import transaction: %w", err)
	}
	return nil
}

func (s *configStore) saveDocumentExternalLink(ctx context.Context, link DocumentExternalLink) error {
	if strings.TrimSpace(link.DocumentID) == "" || strings.TrimSpace(link.WorkspaceID) == "" || strings.TrimSpace(link.Provider) == "" {
		return fmt.Errorf("document external link is incomplete")
	}
	if _, err := s.appDB.ExecContext(ctx, `
		INSERT INTO document_external_links (id, document_id, workspace_id, provider, external_id, external_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`, link.ID, link.DocumentID, link.WorkspaceID, link.Provider, link.ExternalID, link.ExternalKey, link.CreatedAt); err != nil {
		return fmt.Errorf("save document external link: %w", err)
	}
	return nil
}

func (s *configStore) touchWorkspace(ctx context.Context, workspaceID, updatedAt string) error {
	if _, err := s.appDB.ExecContext(ctx, `
		UPDATE workspaces
		SET updated_at = ?
		WHERE id = ?;
	`, updatedAt, strings.TrimSpace(workspaceID)); err != nil {
		return fmt.Errorf("touch workspace: %w", err)
	}
	return nil
}

func newEntityID(prefix string) string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(bytes))
}

func sha256File(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func copyFile(srcPath, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return dstFile.Close()
}

func sanitizeFileName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "document.pdf"
	}
	replacer := strings.NewReplacer("<", "_", ">", "_", ":", "_", `"`, "_", "/", "_", `\\`, "_", "|", "_", "?", "_", "*", "_")
	return replacer.Replace(trimmed)
}

func detectAssetKind(fileName string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName)))
	switch ext {
	case ".pdf":
		return "pdf"
	case ".md", ".markdown":
		return "markdown"
	default:
		return "file"
	}
}

func detectMimeType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName)))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".md", ".markdown":
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}

func (s *configStore) ListDocumentAssetsByWorkspace(ctx context.Context, workspaceID string) ([]DocumentAssetRecord, error) {
	rows, err := s.appDB.QueryContext(ctx, `
		SELECT id, document_id, workspace_id, kind, role, file_name, relative_path, absolute_path, mime_type, byte_size, content_hash, created_at
		FROM document_assets
		WHERE workspace_id = ?
		ORDER BY created_at DESC, id DESC;
	`, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("list document assets: %w", err)
	}
	defer rows.Close()

	assets := []DocumentAssetRecord{}
	for rows.Next() {
		var asset DocumentAssetRecord
		if err := rows.Scan(&asset.ID, &asset.DocumentID, &asset.WorkspaceID, &asset.Kind, &asset.Role, &asset.FileName, &asset.RelativePath, &asset.AbsolutePath, &asset.MimeType, &asset.ByteSize, &asset.ContentHash, &asset.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan document asset: %w", err)
		}
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate document assets: %w", err)
	}
	return assets, nil
}

func (s *configStore) SaveWorkspaceWikiScanJob(ctx context.Context, job WorkspaceWikiScanJob) (WorkspaceWikiScanJob, error) {
	if _, err := s.GetWorkspace(ctx, job.WorkspaceID); err != nil {
		return WorkspaceWikiScanJob{}, err
	}
	if strings.TrimSpace(job.JobID) == "" {
		job.JobID = newEntityID("wiki_job")
	}
	job.DocumentID = strings.TrimSpace(job.DocumentID)
	if strings.TrimSpace(job.StartedAt) == "" {
		job.StartedAt = nowRFC3339()
	}
	job.UpdatedAt = firstNonEmpty(strings.TrimSpace(job.UpdatedAt), job.StartedAt)
	if _, err := s.appDB.ExecContext(ctx, `
		INSERT INTO workspace_wiki_scan_jobs (job_id, workspace_id, document_id, status, total_items, processed_items, failed_items, current_item, current_stage, message, overall_progress, provider_id, model_id, error, started_at, updated_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id) DO UPDATE SET
			document_id = excluded.document_id,
			status = excluded.status,
			total_items = excluded.total_items,
			processed_items = excluded.processed_items,
			failed_items = excluded.failed_items,
			current_item = excluded.current_item,
			current_stage = excluded.current_stage,
			message = excluded.message,
			overall_progress = excluded.overall_progress,
			provider_id = excluded.provider_id,
			model_id = excluded.model_id,
			error = excluded.error,
			started_at = excluded.started_at,
			updated_at = excluded.updated_at,
			finished_at = excluded.finished_at;
	`, job.JobID, job.WorkspaceID, job.DocumentID, job.Status, job.TotalItems, job.ProcessedItems, job.FailedItems, job.CurrentItem, job.CurrentStage, job.Message, job.OverallProgress, job.ProviderID, job.ModelID, job.Error, job.StartedAt, job.UpdatedAt, job.FinishedAt); err != nil {
		return WorkspaceWikiScanJob{}, fmt.Errorf("save workspace wiki scan job: %w", err)
	}
	return job, nil
}

func (s *configStore) UpdateWorkspaceWikiScanJob(ctx context.Context, jobID string, update workspaceWikiScanJobUpdate) (WorkspaceWikiScanJob, error) {
	job, err := s.GetWorkspaceWikiScanJob(ctx, jobID)
	if err != nil {
		return WorkspaceWikiScanJob{}, err
	}
	if update.Status != "" {
		job.Status = update.Status
	}
	job.ProcessedItems = update.ProcessedItems
	job.FailedItems = update.FailedItems
	job.CurrentItem = update.CurrentItem
	job.CurrentStage = update.CurrentStage
	job.Message = update.Message
	job.OverallProgress = update.OverallProgress
	job.Error = update.Error
	job.UpdatedAt = nowRFC3339()
	if update.Finished {
		job.FinishedAt = job.UpdatedAt
	}
	return s.SaveWorkspaceWikiScanJob(ctx, job)
}

func (s *configStore) GetWorkspaceWikiScanJob(ctx context.Context, jobID string) (WorkspaceWikiScanJob, error) {
	var job WorkspaceWikiScanJob
	if err := s.appDB.QueryRowContext(ctx, `
		SELECT job_id, workspace_id, document_id, status, total_items, processed_items, failed_items, current_item, current_stage, message, overall_progress, provider_id, model_id, error, started_at, updated_at, finished_at
		FROM workspace_wiki_scan_jobs
		WHERE job_id = ?
		LIMIT 1;
	`, strings.TrimSpace(jobID)).Scan(&job.JobID, &job.WorkspaceID, &job.DocumentID, &job.Status, &job.TotalItems, &job.ProcessedItems, &job.FailedItems, &job.CurrentItem, &job.CurrentStage, &job.Message, &job.OverallProgress, &job.ProviderID, &job.ModelID, &job.Error, &job.StartedAt, &job.UpdatedAt, &job.FinishedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WorkspaceWikiScanJob{}, fmt.Errorf("wiki scan job %s not found", strings.TrimSpace(jobID))
		}
		return WorkspaceWikiScanJob{}, fmt.Errorf("get workspace wiki scan job: %w", err)
	}
	return job, nil
}

func (s *configStore) ListWorkspaceWikiScanJobs(ctx context.Context) ([]WorkspaceWikiScanJob, error) {
	rows, err := s.appDB.QueryContext(ctx, `
		SELECT job_id, workspace_id, document_id, status, total_items, processed_items, failed_items, current_item, current_stage, message, overall_progress, provider_id, model_id, error, started_at, updated_at, finished_at
		FROM workspace_wiki_scan_jobs
		ORDER BY updated_at DESC, started_at DESC, job_id DESC;
	`)
	if err != nil {
		return nil, fmt.Errorf("list workspace wiki scan jobs: %w", err)
	}
	defer rows.Close()

	jobs := []WorkspaceWikiScanJob{}
	for rows.Next() {
		var job WorkspaceWikiScanJob
		if err := rows.Scan(&job.JobID, &job.WorkspaceID, &job.DocumentID, &job.Status, &job.TotalItems, &job.ProcessedItems, &job.FailedItems, &job.CurrentItem, &job.CurrentStage, &job.Message, &job.OverallProgress, &job.ProviderID, &job.ModelID, &job.Error, &job.StartedAt, &job.UpdatedAt, &job.FinishedAt); err != nil {
			return nil, fmt.Errorf("scan workspace wiki scan job: %w", err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace wiki scan jobs: %w", err)
	}
	return jobs, nil
}

func (s *configStore) DeleteWorkspaceWikiScanJob(ctx context.Context, jobID string) error {
	if _, err := s.appDB.ExecContext(ctx, `DELETE FROM workspace_wiki_scan_jobs WHERE job_id = ?;`, strings.TrimSpace(jobID)); err != nil {
		return fmt.Errorf("delete workspace wiki scan job: %w", err)
	}
	return nil
}

func (s *configStore) ReplaceWorkspaceWikiPages(ctx context.Context, workspaceID string, pages []WorkspaceWikiPage) error {
	tx, err := s.appDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin workspace wiki page transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, `DELETE FROM workspace_wiki_pages WHERE workspace_id = ?;`, strings.TrimSpace(workspaceID)); err != nil {
		return fmt.Errorf("clear workspace wiki pages: %w", err)
	}
	for _, page := range pages {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO workspace_wiki_pages (id, workspace_id, source_document_id, title, slug, kind, markdown_path, summary, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`, page.ID, page.WorkspaceID, page.SourceDocumentID, page.Title, page.Slug, page.Kind, page.MarkdownPath, page.Summary, page.CreatedAt, page.UpdatedAt); err != nil {
			return fmt.Errorf("insert workspace wiki page: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit workspace wiki page transaction: %w", err)
	}
	return nil
}

func (s *configStore) ListWorkspaceWikiPages(ctx context.Context, workspaceID string) ([]WorkspaceWikiPage, error) {
	rows, err := s.appDB.QueryContext(ctx, `
		SELECT id, workspace_id, source_document_id, title, slug, kind, markdown_path, summary, created_at, updated_at
		FROM workspace_wiki_pages
		WHERE workspace_id = ?
		ORDER BY CASE kind
			WHEN 'index' THEN 0
			WHEN 'overview' THEN 1
			WHEN 'open_questions' THEN 2
			WHEN 'log' THEN 3
			WHEN 'document' THEN 4
			WHEN 'concept' THEN 5
			ELSE 6
		END,
		title COLLATE NOCASE ASC,
		id ASC;
	`, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("list workspace wiki pages: %w", err)
	}
	defer rows.Close()

	pages := []WorkspaceWikiPage{}
	for rows.Next() {
		var page WorkspaceWikiPage
		if err := rows.Scan(&page.ID, &page.WorkspaceID, &page.SourceDocumentID, &page.Title, &page.Slug, &page.Kind, &page.MarkdownPath, &page.Summary, &page.CreatedAt, &page.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace wiki page: %w", err)
		}
		pages = append(pages, page)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace wiki pages: %w", err)
	}
	return pages, nil
}

func (s *configStore) GetWorkspaceWikiPage(ctx context.Context, pageID string) (WorkspaceWikiPage, error) {
	var page WorkspaceWikiPage
	if err := s.appDB.QueryRowContext(ctx, `
		SELECT id, workspace_id, source_document_id, title, slug, kind, markdown_path, summary, created_at, updated_at
		FROM workspace_wiki_pages
		WHERE id = ?
		LIMIT 1;
	`, strings.TrimSpace(pageID)).Scan(&page.ID, &page.WorkspaceID, &page.SourceDocumentID, &page.Title, &page.Slug, &page.Kind, &page.MarkdownPath, &page.Summary, &page.CreatedAt, &page.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WorkspaceWikiPage{}, fmt.Errorf("wiki page %s not found", strings.TrimSpace(pageID))
		}
		return WorkspaceWikiPage{}, fmt.Errorf("get workspace wiki page: %w", err)
	}
	return page, nil
}

func (s *configStore) DeleteWorkspaceWikiPagesByWorkspace(ctx context.Context, workspaceID string) error {
	if _, err := s.appDB.ExecContext(ctx, `DELETE FROM workspace_wiki_pages WHERE workspace_id = ?;`, strings.TrimSpace(workspaceID)); err != nil {
		return fmt.Errorf("delete workspace wiki pages: %w", err)
	}
	return nil
}

func (s *configStore) DeleteWorkspaceWikiPageByDocument(ctx context.Context, workspaceID, documentID string) error {
	if _, err := s.appDB.ExecContext(ctx, `DELETE FROM workspace_wiki_pages WHERE workspace_id = ? AND source_document_id = ?;`, strings.TrimSpace(workspaceID), strings.TrimSpace(documentID)); err != nil {
		return fmt.Errorf("delete workspace wiki pages by document: %w", err)
	}
	return nil
}

func (s *configStore) SaveChatHistory(ctx context.Context, entry ChatHistoryEntry) (ChatHistoryEntry, error) {
	createdAt := nowRFC3339()
	result, err := s.appDB.ExecContext(ctx, `
		INSERT INTO ai_chat_history (workspace_id, document_id, item_id, item_title, page, kind, prompt, response, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, strings.TrimSpace(entry.WorkspaceID), strings.TrimSpace(entry.DocumentID), strings.TrimSpace(entry.ItemID), strings.TrimSpace(entry.ItemTitle), entry.Page, strings.TrimSpace(entry.Kind), strings.TrimSpace(entry.Prompt), strings.TrimSpace(entry.Response), createdAt)
	if err != nil {
		return ChatHistoryEntry{}, fmt.Errorf("save chat history: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return ChatHistoryEntry{}, fmt.Errorf("read chat history id: %w", err)
	}
	entry.ID = id
	entry.CreatedAt = createdAt
	return entry, nil
}

func (s *configStore) ListChatHistory(ctx context.Context, workspaceID, documentID, itemID string) ([]ChatHistoryEntry, error) {
	rows, err := s.appDB.QueryContext(ctx, `
		SELECT id, workspace_id, document_id, item_id, item_title, page, kind, prompt, response, created_at
		FROM ai_chat_history
		WHERE (workspace_id = ? AND document_id = ? AND workspace_id <> '' AND document_id <> '')
		   OR (? <> '' AND item_id = ?)
		ORDER BY id DESC;
	`, strings.TrimSpace(workspaceID), strings.TrimSpace(documentID), strings.TrimSpace(itemID), strings.TrimSpace(itemID))
	if err != nil {
		return nil, fmt.Errorf("list chat history: %w", err)
	}
	defer rows.Close()

	entries := []ChatHistoryEntry{}
	for rows.Next() {
		var entry ChatHistoryEntry
		if err := rows.Scan(&entry.ID, &entry.WorkspaceID, &entry.DocumentID, &entry.ItemID, &entry.ItemTitle, &entry.Page, &entry.Kind, &entry.Prompt, &entry.Response, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan chat history: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat history: %w", err)
	}
	return entries, nil
}

func (s *configStore) DeleteChatHistory(ctx context.Context, id int64) error {
	if _, err := s.appDB.ExecContext(ctx, `
		DELETE FROM ai_chat_history
		WHERE id = ?;
	`, id); err != nil {
		return fmt.Errorf("delete chat history: %w", err)
	}
	return nil
}

func (s *configStore) SaveReaderNote(ctx context.Context, entry ReaderNoteEntry) (ReaderNoteEntry, error) {
	createdAt := nowRFC3339()
	result, err := s.appDB.ExecContext(ctx, `
		INSERT INTO reader_notes (workspace_id, document_id, item_id, item_title, page, anchor_text, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?);
	`, strings.TrimSpace(entry.WorkspaceID), strings.TrimSpace(entry.DocumentID), strings.TrimSpace(entry.ItemID), strings.TrimSpace(entry.ItemTitle), entry.Page, strings.TrimSpace(entry.AnchorText), strings.TrimSpace(entry.Content), createdAt)
	if err != nil {
		return ReaderNoteEntry{}, fmt.Errorf("save reader note: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return ReaderNoteEntry{}, fmt.Errorf("read reader note id: %w", err)
	}
	entry.ID = id
	entry.CreatedAt = createdAt
	return entry, nil
}

func (s *configStore) ListReaderNotes(ctx context.Context, workspaceID, documentID, itemID string) ([]ReaderNoteEntry, error) {
	rows, err := s.appDB.QueryContext(ctx, `
		SELECT id, workspace_id, document_id, item_id, item_title, page, anchor_text, content, created_at
		FROM reader_notes
		WHERE (workspace_id = ? AND document_id = ? AND workspace_id <> '' AND document_id <> '')
		   OR (? <> '' AND item_id = ?)
		ORDER BY id DESC;
	`, strings.TrimSpace(workspaceID), strings.TrimSpace(documentID), strings.TrimSpace(itemID), strings.TrimSpace(itemID))
	if err != nil {
		return nil, fmt.Errorf("list reader notes: %w", err)
	}
	defer rows.Close()

	entries := []ReaderNoteEntry{}
	for rows.Next() {
		var entry ReaderNoteEntry
		if err := rows.Scan(&entry.ID, &entry.WorkspaceID, &entry.DocumentID, &entry.ItemID, &entry.ItemTitle, &entry.Page, &entry.AnchorText, &entry.Content, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan reader note: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reader notes: %w", err)
	}
	return entries, nil
}

const (
	pdfTranslateRuntimeSettingKeyDB = pdfTranslateRuntimeSettingKey
)

func defaultAIWorkspaceConfig() AIWorkspaceConfig {
	return AIWorkspaceConfig{
		SummaryMode:          "auto",
		SummaryChunkPages:    6,
		SummaryChunkMaxChars: 18000,
		AutoRestoreCount:     3,
		TableTemplate: `| 维度 | 内容 |
| --- | --- |
| 论文标题 | |
| 研究问题 | |
| 核心方法 | |
| 数据/实验设置 | |
| 关键结果 | |
| 创新点 | |
| 局限性 | |
| 我能直接借鉴什么 | |`,
		TablePrompt:         "请仔细阅读当前论文，并严格按照给定的 Markdown 表格模板填写。要求：1. 只输出填好的表格。2. 所有单元格用中文填写。3. 若原文未明确提及，填写“未明确说明”。4. 内容应简洁但能支持快速比较论文。",
		CustomPromptDraft:   "",
		FollowUpPromptDraft: "",
		DrawingPromptDraft:  "根据当前论文内容，生成一张适合组会汇报的科研概念图，突出问题、方法流程、关键结果和应用价值。",
		DrawingProviderID:   0,
		DrawingModel:        "gemini-3-pro-image-preview",
		WikiScanProviderID:  0,
		WikiScanModelID:     0,
	}
}

func normalizeAIWorkspaceConfig(input AIWorkspaceConfig) AIWorkspaceConfig {
	config := defaultAIWorkspaceConfig()

	switch strings.TrimSpace(input.SummaryMode) {
	case "single", "multi":
		config.SummaryMode = strings.TrimSpace(input.SummaryMode)
	default:
		config.SummaryMode = "auto"
	}

	if input.SummaryChunkPages >= 1 && input.SummaryChunkPages <= 30 {
		config.SummaryChunkPages = input.SummaryChunkPages
	}
	if input.SummaryChunkMaxChars >= 4000 && input.SummaryChunkMaxChars <= 120000 {
		config.SummaryChunkMaxChars = input.SummaryChunkMaxChars
	}
	if input.AutoRestoreCount >= 1 && input.AutoRestoreCount <= 12 {
		config.AutoRestoreCount = input.AutoRestoreCount
	}

	if trimmed := strings.TrimSpace(input.TableTemplate); trimmed != "" {
		config.TableTemplate = trimmed
	}
	if trimmed := strings.TrimSpace(input.TablePrompt); trimmed != "" {
		config.TablePrompt = trimmed
	}

	config.CustomPromptDraft = input.CustomPromptDraft
	config.FollowUpPromptDraft = input.FollowUpPromptDraft
	if trimmed := strings.TrimSpace(input.DrawingPromptDraft); trimmed != "" {
		config.DrawingPromptDraft = input.DrawingPromptDraft
	}
	if input.DrawingProviderID > 0 {
		config.DrawingProviderID = input.DrawingProviderID
	}
	if trimmed := strings.TrimSpace(input.DrawingModel); trimmed != "" {
		config.DrawingModel = trimmed
	}
	if input.WikiScanProviderID > 0 {
		config.WikiScanProviderID = input.WikiScanProviderID
	}
	if input.WikiScanModelID > 0 {
		config.WikiScanModelID = input.WikiScanModelID
	}

	return config
}

func (s *configStore) GetAIWorkspaceConfig(ctx context.Context, workspaceID string) (AIWorkspaceConfig, error) {
	config := defaultAIWorkspaceConfig()
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if trimmedWorkspaceID == "" {
		return config, nil
	}

	row := s.appDB.QueryRowContext(ctx, `
		SELECT config_json
		FROM workspace_ai_configs
		WHERE workspace_id = ?;
	`, trimmedWorkspaceID)

	var raw string
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return config, nil
		}
		return AIWorkspaceConfig{}, fmt.Errorf("get workspace ai config: %w", err)
	}

	var stored AIWorkspaceConfig
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return AIWorkspaceConfig{}, fmt.Errorf("decode ai workspace config: %w", err)
	}
	return normalizeAIWorkspaceConfig(stored), nil
}

func (s *configStore) SaveAIWorkspaceConfig(ctx context.Context, workspaceID string, input AIWorkspaceConfig) (AIWorkspaceConfig, error) {
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if trimmedWorkspaceID == "" {
		return AIWorkspaceConfig{}, fmt.Errorf("workspace id is required")
	}
	if _, err := s.GetWorkspace(ctx, trimmedWorkspaceID); err != nil {
		return AIWorkspaceConfig{}, err
	}

	normalized := normalizeAIWorkspaceConfig(input)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return AIWorkspaceConfig{}, fmt.Errorf("encode ai workspace config: %w", err)
	}

	if _, err := s.appDB.ExecContext(ctx, `
		INSERT INTO workspace_ai_configs (workspace_id, config_json, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(workspace_id) DO UPDATE SET
			config_json = excluded.config_json,
			updated_at = excluded.updated_at;
	`, trimmedWorkspaceID, string(encoded), nowRFC3339()); err != nil {
		return AIWorkspaceConfig{}, fmt.Errorf("save ai workspace config: %w", err)
	}

	return normalized, nil
}

func (s *configStore) GetPDFTranslateRuntimeConfig(ctx context.Context) (PDFTranslateRuntimeConfig, error) {
	config := defaultMissingPDFTranslateRuntimeConfig()

	raw, found, err := s.getSettingValue(ctx, pdfTranslateRuntimeSettingKeyDB)
	if err != nil {
		return PDFTranslateRuntimeConfig{}, err
	}
	if !found {
		return validateStoredPDFTranslateRuntimeConfig(config), nil
	}

	var stored PDFTranslateRuntimeConfig
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return PDFTranslateRuntimeConfig{}, fmt.Errorf("decode pdf translate runtime config: %w", err)
	}
	return validateStoredPDFTranslateRuntimeConfig(stored), nil
}

func (s *configStore) SavePDFTranslateRuntimeConfig(ctx context.Context, input PDFTranslateRuntimeConfig) (PDFTranslateRuntimeConfig, error) {
	normalized := validateStoredPDFTranslateRuntimeConfig(input)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return PDFTranslateRuntimeConfig{}, fmt.Errorf("encode pdf translate runtime config: %w", err)
	}
	if err := s.saveSettingValue(ctx, pdfTranslateRuntimeSettingKeyDB, string(encoded)); err != nil {
		return PDFTranslateRuntimeConfig{}, fmt.Errorf("save pdf translate runtime config: %w", err)
	}
	return normalized, nil
}

func (s *configStore) ClearPDFTranslateRuntimeConfig(ctx context.Context) error {
	if _, err := s.appDB.ExecContext(ctx, `DELETE FROM app_settings WHERE setting_key = ?;`, pdfTranslateRuntimeSettingKeyDB); err != nil {
		return fmt.Errorf("clear pdf translate runtime config: %w", err)
	}
	return nil
}

func (s *configStore) saveSettingValue(ctx context.Context, key, value string) error {
	if _, err := s.appDB.ExecContext(ctx, `
		INSERT INTO app_settings (setting_key, setting_value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(setting_key) DO UPDATE SET
			setting_value = excluded.setting_value,
			updated_at = excluded.updated_at;
	`, strings.TrimSpace(key), value, nowRFC3339()); err != nil {
		return fmt.Errorf("save app setting %s: %w", key, err)
	}
	return nil
}

func (s *configStore) getSettingValue(ctx context.Context, key string) (string, bool, error) {
	row := s.appDB.QueryRowContext(ctx, `
		SELECT setting_value
		FROM app_settings
		WHERE setting_key = ?;
	`, strings.TrimSpace(key))

	var value string
	if err := row.Scan(&value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("load app setting %s: %w", key, err)
	}

	return value, true, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func isValidProviderType(providerType ProviderType) bool {
	switch providerType {
	case ProviderTypeLLM, ProviderTypeDrawing, ProviderTypeTranslate:
		return true
	default:
		return false
	}
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (s *configStore) ensureProvidersRegionColumn() error {
	rows, err := s.appDB.Query(`PRAGMA table_info(providers);`)
	if err != nil {
		return fmt.Errorf("inspect providers schema: %w", err)
	}
	defer rows.Close()

	var (
		cid        int
		name       string
		colType    string
		notNull    int
		defaultV   sql.NullString
		primaryKey int
	)
	for rows.Next() {
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultV, &primaryKey); err != nil {
			return fmt.Errorf("scan providers schema: %w", err)
		}
		if strings.EqualFold(name, "region") {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate providers schema: %w", err)
	}

	if _, err := s.appDB.Exec(`ALTER TABLE providers ADD COLUMN region TEXT NOT NULL DEFAULT '';`); err != nil {
		return fmt.Errorf("add providers.region column: %w", err)
	}
	return nil
}

func (s *configStore) ensureHistoryDocumentColumns() error {
	for _, tableName := range []string{"ai_chat_history", "reader_notes"} {
		rows, err := s.appDB.Query(fmt.Sprintf("PRAGMA table_info(%s);", tableName))
		if err != nil {
			return fmt.Errorf("inspect %s schema: %w", tableName, err)
		}

		hasWorkspaceID := false
		hasDocumentID := false
		for rows.Next() {
			var (
				cid        int
				name       string
				colType    string
				notNull    int
				defaultV   sql.NullString
				primaryKey int
			)
			if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultV, &primaryKey); err != nil {
				_ = rows.Close()
				return fmt.Errorf("scan %s schema: %w", tableName, err)
			}
			if strings.EqualFold(name, "workspace_id") {
				hasWorkspaceID = true
			}
			if strings.EqualFold(name, "document_id") {
				hasDocumentID = true
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf("iterate %s schema: %w", tableName, err)
		}
		_ = rows.Close()

		if !hasWorkspaceID {
			if _, err := s.appDB.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN workspace_id TEXT NOT NULL DEFAULT '';", tableName)); err != nil {
				return fmt.Errorf("add %s.workspace_id column: %w", tableName, err)
			}
		}
		if !hasDocumentID {
			if _, err := s.appDB.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN document_id TEXT NOT NULL DEFAULT '';", tableName)); err != nil {
				return fmt.Errorf("add %s.document_id column: %w", tableName, err)
			}
		}
	}
	return nil
}

func (s *configStore) ensureWorkspaceWikiScanJobsSchema() error {
	rows, err := s.appDB.Query(`PRAGMA table_info(workspace_wiki_scan_jobs);`)
	if err != nil {
		return fmt.Errorf("inspect workspace wiki scan jobs schema: %w", err)
	}
	defer rows.Close()

	type columnInfo struct {
		name string
	}

	columns := map[string]columnInfo{}
	for rows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			defaultV   sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultV, &primaryKey); err != nil {
			return fmt.Errorf("scan workspace wiki scan jobs schema: %w", err)
		}
		columns[strings.ToLower(strings.TrimSpace(name))] = columnInfo{name: name}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate workspace wiki scan jobs schema: %w", err)
	}

	if _, ok := columns["job_id"]; ok {
		if _, ok := columns["document_id"]; !ok {
			if _, err := s.appDB.Exec(`ALTER TABLE workspace_wiki_scan_jobs ADD COLUMN document_id TEXT NOT NULL DEFAULT '';`); err != nil {
				return fmt.Errorf("add workspace wiki scan jobs document_id column: %w", err)
			}
		}
		return nil
	}

	tx, err := s.appDB.Begin()
	if err != nil {
		return fmt.Errorf("begin workspace wiki scan jobs migration: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`ALTER TABLE workspace_wiki_scan_jobs RENAME TO workspace_wiki_scan_jobs_legacy;`); err != nil {
		return fmt.Errorf("rename legacy workspace wiki scan jobs table: %w", err)
	}
	if _, err = tx.Exec(`
		CREATE TABLE workspace_wiki_scan_jobs (
			job_id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			document_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			total_items INTEGER NOT NULL DEFAULT 0,
			processed_items INTEGER NOT NULL DEFAULT 0,
			failed_items INTEGER NOT NULL DEFAULT 0,
			current_item TEXT NOT NULL DEFAULT '',
			current_stage TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			overall_progress REAL NOT NULL DEFAULT 0,
			provider_id INTEGER NOT NULL DEFAULT 0,
			model_id INTEGER NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '',
			started_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			finished_at TEXT NOT NULL DEFAULT '',
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);
	`); err != nil {
		return fmt.Errorf("create workspace wiki scan jobs table: %w", err)
	}
	if _, err = tx.Exec(`
		INSERT INTO workspace_wiki_scan_jobs (
			job_id, workspace_id, document_id, status, total_items, processed_items, failed_items,
			current_item, current_stage, message, overall_progress, provider_id, model_id, error,
			started_at, updated_at, finished_at
		)
		SELECT
			printf('wiki_job_legacy_%d', id),
			workspace_id,
			COALESCE(document_id, ''),
			status,
			0,
			CASE WHEN status IN ('completed', 'failed') THEN 1 ELSE 0 END,
			CASE WHEN status = 'failed' THEN 1 ELSE 0 END,
			'',
			current_stage,
			message,
			CASE WHEN status IN ('completed', 'failed') THEN 1 ELSE 0 END,
			provider_id,
			model_id,
			CASE WHEN status = 'failed' THEN message ELSE '' END,
			started_at,
			updated_at,
			finished_at
		FROM workspace_wiki_scan_jobs_legacy;
	`); err != nil {
		return fmt.Errorf("migrate workspace wiki scan jobs: %w", err)
	}
	if _, err = tx.Exec(`DROP TABLE workspace_wiki_scan_jobs_legacy;`); err != nil {
		return fmt.Errorf("drop legacy workspace wiki scan jobs table: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit workspace wiki scan jobs migration: %w", err)
	}
	return nil
}

func (s *configStore) purgeDeprecatedOCRProviders() error {
	if _, err := s.appDB.Exec(`DELETE FROM providers WHERE type = ?;`, string(ProviderTypeOCR)); err != nil {
		return fmt.Errorf("purge deprecated ocr providers: %w", err)
	}
	return nil
}

func (s *configStore) purgeDeprecatedOCRCache() error {
	if _, err := s.ocrDB.Exec(`DELETE FROM page_ocr_results;`); err != nil {
		return fmt.Errorf("purge deprecated ocr cache: %w", err)
	}
	return nil
}
