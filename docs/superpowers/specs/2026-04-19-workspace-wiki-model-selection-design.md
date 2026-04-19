# Workspace Wiki Model Selection Design

**Date:** 2026-04-19
**Status:** Draft
**Scope:** Add provider/model selection UI to Workspace Wiki scan panel with per-workspace persistence

## Problem

The Workspace Wiki scan currently hardcodes the model selection: it always uses "the first provider with at least one model, and that provider's first model." This causes:

1. No visibility into which model is actually being used for the scan
2. No ability to switch models for different scanning tasks
3. Confusion when scan fails - user cannot tell if wrong model type was selected
4. The error message "Cannot read 'clipboard' (this model does not support image input)" appeared because a drawing/image model was incorrectly used for text-only wiki generation

## Solution

Add explicit provider/model selection controls to the Workspace Wiki panel with per-workspace persistence.

## Design

### UI & Interaction

**Placement:** Inside the `Workspace Wiki` panel, adjacent to the `Scan / Cancel / Refresh` button group. Same visual context as the scan action.

**Controls:** Two compact dropdowns in sequence:
- Provider dropdown: shows all configured LLM providers with at least one model
- Model dropdown: shows only models belonging to the selected provider

**Default behavior:**
- On entering a workspace, restore the last-used provider/model for that workspace
- If no saved selection exists, auto-select "first available provider + first model"
- If saved selection is invalid (provider deleted/disabled), reset to first available and save

**Button state:**
- `Scan` button disabled when no complete provider/model selection
- Replace current generic message with specific prompt: "请先为当前工作区选择扫描模型"

**Transparency during scan:**
- Status display shows current model info, e.g., "Using OpenAI / gpt-4.1"
- Helps user understand which model caused failures

### State Persistence

**Storage location:** Extend existing `AIWorkspaceConfig` structure (already persisted via `getAIWorkspaceConfig` / `saveAIWorkspaceConfig`).

**New fields:**
- `wikiScanProviderId: number` - provider ID from providers table
- `wikiScanModelId: number` - model record ID from models table (primary key, not the `modelId` string)

**Save timing:** Auto-save after selection change with 450ms debounce (same pattern as existing drawing config).

**Fallback logic:**
- On workspace load, validate saved provider/model still exists and is active
- If invalid, auto-reset to first available combination and trigger save
- Prevents stale configuration from blocking future scans

### Backend Validation

**Scan start validation:**
- Frontend ensures complete selection before calling `StartWorkspaceWikiScan`
- Backend `loadOpenAICompatibleProviderModel` validates:
  - Provider is active
  - Model belongs to provider
  - Provider has API key configured
- If validation fails, job immediately marked as `failed` with specific error

### Frontend Changes

| File | Changes |
|------|---------|
| `WorkspaceTab.tsx` | Add `wikiScanProviderId` / `wikiScanModelId` state; render provider/model dropdowns; pass selection to `onStartWikiScan` |
| `App.tsx` | Update `onStartWikiScan` callback to accept provider/model from WorkspaceTab instead of hardcoding |
| `config.ts` | Add `wikiScanProviderId` and `wikiScanModelId` to `AIWorkspaceConfig` type; default to `0` |
| `configApi.ts` | No changes needed - existing API handles arbitrary fields |

### Backend Changes

| File | Changes |
|------|---------|
| `config_types.go` | Add `WikiScanProviderID` and `WikiScanModelID` fields to `AIWorkspaceConfig` struct |
| `config_store.go` | Extend `saveAIWorkspaceConfig` / `getAIWorkspaceConfig` SQL to include new columns; add schema migration |

### No Changes Required

- `workspaceWikiApi.ts` - existing `start` already accepts `providerId` / `modelId`
- `workspace_wiki_service.go` - existing `StartScan` already uses `input.ProviderID` / `input.ModelID`
- `gateway_service.go` - existing `GenerateWorkspaceKnowledgeBySource` already validates provider/model

## Migration Path

1. Backend: Add new fields to `AIWorkspaceConfig` with `0` default for existing workspaces
2. Frontend: On first load, detect `0` values and auto-select first available
3. User selection auto-saves, future loads restore correctly

## Edge Cases

- **No configured LLM providers:** Dropdowns empty, `Scan` disabled, show "请先配置 LLM provider 和 model"
- **Provider deleted after selection:** Auto-reset on next workspace load
- **Model deleted after selection:** Auto-reset on next workspace load  
- **Provider deactivated:** Auto-reset on next workspace load
- **Multiple providers with same name:** Dropdown shows provider name (unique constraint already exists)

## Success Criteria

1. User can see and change which model will be used for wiki scan
2. Selection persists per-workspace across sessions
3. Invalid selections auto-reset without blocking user
4. Scan failures show clear model information in error display
5. No regression in existing wiki scan functionality