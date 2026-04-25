# Workspace PDF Wiki Architecture Design

## Background

OpenSciReader already has the core ingredients of a workspace knowledge system:

- workspace-scoped document import
- PDF-to-Markdown extraction via MarkItDown
- a workspace wiki scan service
- per-source structured extraction
- generated wiki pages and query flows

The remaining problem is architectural clarity. The current `raw / schema / wiki` layout mixes concerns:

- `raw` contains both original source material and derived extraction output
- `schema` contains both durable structured knowledge and transient runtime state
- the role of Markdown wiki pages versus machine-readable memory is not clearly separated

This design reframes each workspace as a research-project container with its own PDF corpus, its own derived Markdown input cache, its own machine state, and its own human-readable wiki.

It keeps the current file-first approach, keeps Markdown as the final human-facing memory artifact, and avoids forcing every internal layer into Markdown.

## Design Decisions

The following decisions are fixed by user requirements gathered during brainstorming:

1. Each workspace represents one research project or one research topic.
2. Each workspace stores its own source PDFs.
3. MarkItDown output is a reusable cache, not a long-term asset.
4. The wiki is incrementally maintained in the background after import.
5. Human-readable memory lives in Markdown pages.
6. Machine-readable runtime state lives in structured files, not in Markdown.

## Goals

- Make the workspace knowledge layout semantically clear.
- Separate source facts, derived input cache, machine state, and human-readable wiki output.
- Preserve the existing file-first workflow and avoid introducing a database requirement for the wiki layer.
- Support background incremental processing after PDF import.
- Make per-source failure, retry, and staleness visible and explainable.
- Keep the query path aligned with the LLM wiki pattern: prefer accumulated wiki knowledge before falling back to raw inputs.

## Non-Goals

- No vector database or embedding retrieval in this version.
- No attempt to make all machine state human-editable Markdown.
- No requirement to support cross-workspace shared PDF stores.
- No requirement to perform concept-level perfect incremental recompilation in the first implementation.
- No rewrite of the entire workspace pipeline from scratch.

## Final Workspace Layout

Each workspace should use the following structure:

```text
workspaces/<workspace-id>/
  files/...
  sources/
    pdfs/
      <imported-pdf-files>
  inputs/
    markitdown/
      <source-slug>.md
    manifests/
      <optional-import-and-conversion-manifests>
  state/
    sources.json
    by-source/
      <source-slug>.json
    jobs/
      <job-id>.json
    compile.json
  wiki/
    index.md
    overview.md
    open-questions.md
    log.md
    docs/
      <source-slug>.md
    concepts/
      <concept-slug>.md
```

## Layer Responsibilities

### `sources`

`sources/` stores original PDF facts for this workspace.

- Source PDFs are project assets.
- The agent does not rewrite them.
- They are the final fallback source of truth.

### `inputs`

`inputs/` stores derived, machine-friendly input caches.

- `inputs/markitdown/` stores Markdown generated from PDFs.
- This layer is disposable and reproducible.
- It is not treated as long-term knowledge memory.
- If a PDF changes, the corresponding Markdown can be regenerated.

### `state`

`state/` stores machine-readable runtime and compilation state.

- source inventory
- per-source extraction output
- job progress and failure state
- compile summaries and dirty markers

This layer exists for scheduling, retry, deduplication, and explainability. It is an implementation layer, not the primary reading surface.

### `wiki`

`wiki/` stores the human-readable memory artifact.

- `overview.md` is the workspace synthesis page.
- `docs/` contains per-document human-readable pages.
- `concepts/` contains cross-document topic pages.
- `open-questions.md` records unresolved issues and follow-up directions.
- `index.md` is the navigation and query entry point.
- `log.md` is a human-readable history of meaningful pipeline events.

This is the primary accumulated knowledge layer for both the user and the agent's first-pass retrieval.

## Data Model

The minimum stable model in this design has four runtime objects:

- source records
- by-source extracted state
- workspace build jobs
- workspace compile summaries

## Source Model

`state/sources.json` is the canonical workspace source manifest.

Each source record should include at least:

- `sourceId`
- `workspaceId`
- `title`
- `slug`
- `kind`
- `sourcePath`
- `markitdownPath`
- `contentHash`
- `markitdownStatus`
- `extractStatus`
- `lastIngestAt`
- `lastSuccessAt`
- `lastError`

The source record represents the identity and processing state of one PDF inside one workspace.

## By-Source State

`state/by-source/<source-slug>.json` stores the extracted machine-readable result for one source.

Each file should include:

- the source record snapshot
- extracted entities
- extracted claims
- extracted relations
- extracted tasks
- `sourceHash`
- `generatedAt`

This file is the primary incremental compilation cache.

## Job Model

`state/jobs/<job-id>.json` stores workspace build jobs.

Each job should include at least:

- `jobId`
- `workspaceId`
- `type`
- `status`
- `stage`
- `sourceIds`
- `processedItems`
- `failedItems`
- `message`
- `startedAt`
- `updatedAt`
- `finishedAt`

Jobs describe workspace build activity. They do not replace per-source status.

## Compile Summary Model

`state/compile.json` should summarize the latest workspace-level compile run, including:

- last compile start and finish times
- source ids included in the run
- sources skipped or failed
- wiki pages updated
- whether the workspace is compile-dirty or wiki-dirty

This file exists to explain what the background builder most recently did.

## State Model

Per-source state should not be collapsed into a single status field.

At minimum, sources need two independent processing tracks:

### MarkItDown State

- `pending`
- `running`
- `ready`
- `failed`

### Extraction State

- `pending`
- `running`
- `ready`
- `failed`
- `stale`

`stale` means the existing extracted result still exists but no longer matches the current source content hash.

Workspace build state should remain separate from source state. At minimum, the workspace build job should distinguish:

- `idle`
- `pending_compile`
- `compiling`
- `updating_wiki`
- `completed`
- `failed`

## Operations

The workspace pipeline consists of four major operations.

### 1. Ingest

When the user imports a PDF into the workspace:

1. Copy or register the PDF under `sources/pdfs/`.
2. Register the source in `state/sources.json`.
3. Compute the source hash.
4. Queue background MarkItDown conversion.

Ingest creates the source identity and schedules the rest of the pipeline. It does not need to synchronously complete the wiki.

### 2. Build

The background build pipeline has three logical stages:

1. `extract-input`
   Generate or refresh `inputs/markitdown/<source>.md`.
2. `extract-state`
   Read one source Markdown file and write `state/by-source/<source>.json`.
3. `compile-and-write`
   Rebuild workspace-level compiled state and update wiki pages.

This keeps PDF conversion, per-source semantic extraction, and workspace-level synthesis separate.

### 3. Query

The query order must prefer accumulated knowledge over raw rediscovery.

The retrieval precedence is:

```text
wiki -> state -> inputs/markitdown -> source pdf
```

The expected query flow is:

1. Read `wiki/index.md`.
2. Read the most relevant wiki pages.
3. If needed, inspect `state` for structured evidence.
4. If needed, fall back to MarkItDown Markdown.
5. Only as a last step, return to the original PDF.

### 4. Lint

Lint checks the health of the workspace knowledge system rather than code quality.

Lint should detect at least:

- sources missing MarkItDown output
- stale by-source extraction files
- workspace compile-dirty state
- outdated wiki pages
- missing cross-references
- orphan concept pages
- duplicated concept pages
- unresolved open questions that need follow-up

## Background State Machine

Each source advances independently through the source pipeline.

Recommended source lifecycle:

```text
discovered
  -> md_pending
  -> md_running
  -> md_ready
  -> extract_pending
  -> extract_running
  -> extract_ready
```

Failure states remain local to the source:

```text
md_failed
extract_failed
```

Workspace-level activity is separate:

```text
pending_compile
  -> compiling
  -> updating_wiki
  -> completed
```

## Scheduling Rules

The scheduler should operate at two levels.

### Source-Level Work

Runs independently per source:

- hashing
- MarkItDown conversion
- by-source extraction

### Workspace-Level Work

Runs per workspace:

- compile aggregated state
- rewrite affected wiki pages

This prevents one failed source from blocking the entire workspace and makes build status understandable in the UI.

## Dirty Flags

The implementation should maintain at least three dirty categories:

1. `source dirty`
   The source content changed and needs fresh Markdown and extraction.
2. `compile dirty`
   By-source state changed and the workspace aggregate is outdated.
3. `wiki dirty`
   Compiled state changed and wiki pages need regeneration.

This allows the pipeline to advance cleanly:

```text
source changed -> source dirty
source extracted -> compile dirty
compile completed -> wiki dirty
wiki written -> clean
```

## Debounce and Batching

Workspace compile should be debounced.

If multiple sources finish within a short interval, the system should merge those updates into one workspace compile instead of recompiling after every single PDF. This is required to avoid unnecessary repeated wiki rewrites during batch import.

## Failure Handling

Failures should be isolated and should not destroy already-usable outputs.

### MarkItDown Failure

- Only the affected source is blocked.
- The failure is recorded on the source.
- Retry is allowed without affecting other sources.

### Extraction Failure

- Only the affected source is blocked from contributing new state.
- Existing successful by-source state may remain until replaced.
- Other sources continue normally.

### Compile Failure

- Existing wiki pages remain untouched.
- Existing compiled state remains the last known good output.

### Wiki Write Failure

- The system should prefer staging and atomic replacement.
- Partial page overwrite should be avoided.

## Wiki Output Responsibilities

The wiki pages should have stable roles.

### `wiki/overview.md`

The top-level synthesis for the research workspace:

- current thesis
- main concepts
- notable findings
- active uncertainties

### `wiki/docs/<source>.md`

One page per source, focused on:

- what the source covers
- what matters for this workspace
- key takeaways grounded in the source

### `wiki/concepts/<concept>.md`

Cross-source concept pages focused on:

- concept definition in workspace context
- evidence across sources
- comparisons and tensions
- links to relevant docs and related concepts

### `wiki/open-questions.md`

The structured uncertainty page:

- unresolved questions
- contradictions
- recommended follow-up reading
- suggested next investigation steps

### `wiki/index.md`

The main navigation page for both users and the agent.

### `wiki/log.md`

A concise human-readable operation history, for example:

```text
## [2026-04-25 14:32] ingest | Attention Is All You Need
- source registered
- markitdown completed
- extraction completed
- queued for compile

## [2026-04-25 14:35] compile | workspace
- merged 12 sources
- updated overview and 3 concept pages
```

## Migration from Current Layout

The current layout should be migrated conceptually as follows:

- old `raw/` becomes split into `sources/` and `inputs/`
- old `schema/` becomes `state/`
- old `wiki/` remains `wiki/`

The most important concrete path migrations are:

- `raw/extracts/*.md` -> `inputs/markitdown/*.md`
- `schema/by-source/*.json` -> `state/by-source/*.json`
- `schema/scan-runs/*.json` -> `state/jobs/*.json`
- old source manifest -> `state/sources.json`

The implementation may support compatibility reads for older workspaces during transition, but all new writes should target the new layout.

## Minimal Implementation Strategy

This design intentionally favors the smallest correct change over a full rewrite.

### Phase 1

Rename and reorganize the filesystem layout through `workspaceKnowledgeFiles` while keeping most service logic intact.

### Phase 2

Promote MarkItDown output to a first-class `inputs/markitdown` cache and add explicit `markitdownStatus` and `extractStatus` to source records.

### Phase 3

Keep by-source machine memory as JSON under `state/by-source/` rather than forcing it into Markdown.

### Phase 4

Keep workspace compile simple:

- source-level work stays incremental
- workspace aggregate can be recomputed from all `state/by-source/*.json`
- wiki page writing may start coarse and become more selective later

This is the recommended first implementation because it keeps consistency risks low while preserving a path to later optimization.

## Verification

The implementation should be considered correct only if it demonstrates all of the following:

1. Importing one PDF generates:
   - a source PDF
   - a MarkItDown Markdown cache
   - a by-source state file
   - a corresponding wiki document page
2. Importing multiple PDFs batches workspace compile instead of recompiling after every single source completion.
3. Updating one PDF marks only that source stale and reprocesses it without forcing a full source rescan of the workspace.
4. One failed source does not corrupt existing wiki output or block unrelated sources.
5. Query flows prefer `wiki` over `state`, and `state` over derived Markdown or PDF fallback.

## Summary

The final architecture for each workspace is:

- `sources` for original PDFs
- `inputs` for reproducible MarkItDown cache
- `state` for machine-readable runtime and extraction state
- `wiki` for human-readable accumulated memory

This preserves the LLM wiki idea while making the implementation more explicit, more incremental, and easier to reason about in the current OpenSciReader codebase.
