# AI Workspace Knowledge Layer Design

## Background

OpenSciReader already has an AI workspace UI, document import flow, PDF-to-Markdown extraction, and an initial workspace wiki scan pipeline. However, the current AI sidebar is still centered on legacy prompt-driven interactions rather than a durable workspace knowledge layer.

This design defines the first full version of the AI workspace system around two goals:

1. Scan workspace documents and build a persistent wiki.
2. Build structured memory for the workspace and let the reader sidebar query that memory directly.

This version intentionally does not adapt to legacy sidebar interaction patterns. Instead, it establishes a new source of truth that later UI iterations can build on.

## Goals

- Build a file-based `raw / wiki / schema` knowledge layer inside each workspace.
- Support scanning imported workspace documents plus `pdf/md/markdown` files discovered under the workspace directory.
- Convert PDF files with MarkItDown and normalize all supported sources into reusable extracted markdown.
- Distill structured memory from each source into machine-readable files with source traceability.
- Compile per-source structured memory into workspace-level aggregated knowledge files.
- Generate readable workspace wiki pages from extracted content and structured memory.
- Replace the current reader AI sidebar core flow with a knowledge-grounded ask/answer/evidence/promote flow.
- Let users promote chat-derived candidate memories into the formal workspace memory store.

## Non-Goals

- No vector database or embedding-based retrieval in this version.
- No background file watching or automatic rescan on filesystem changes.
- No compatibility work to preserve the old reader AI interaction model.
- No automatic write-back of all chat turns into formal memory.
- No graph visualization editor or advanced graph tooling.
- No cloud sync or multi-user collaboration.

## Knowledge Layer Layout

Each workspace gets a dedicated knowledge directory structure:

```text
workspaces/<workspace-id>/
  files/...
  raw/
    sources.json
    extracts/
      <source-slug>.md
  wiki/
    overview.md
    open-questions.md
    docs/
      <source-slug>.md
    concepts/
      <concept-slug>.md
  schema/
    by-source/
      <source-slug>.json
    entities.json
    claims.json
    relations.json
    tasks.json
    conversation-log.jsonl
    scan-runs/
      <job-id>.json
```

### Layer Responsibilities

- `raw`
  Stores source inventory and normalized extracted text only. No summarization, no rewriting, no structured interpretation.
- `wiki`
  Stores human-readable knowledge artifacts generated from raw sources and compiled memory.
- `schema`
  Stores structured workspace memory and operational logs used by retrieval, compilation, and promotion flows.

## Architecture

The system is organized around a new workspace knowledge service with four explicit stages:

1. `scan`
   Discover sources, hash them, extract normalized text into `raw/extracts`.
2. `distill`
   Process one source at a time with the LLM and emit structured memory into `schema/by-source/<source-slug>.json`.
3. `compile`
   Merge all per-source structured memory into aggregated workspace-level schema files and regenerate wiki outputs.
4. `query`
   Answer reader sidebar questions by retrieving from `schema`, `wiki`, and then `raw` as a fallback.

This keeps the file-first design workable while still separating extraction, memory generation, aggregation, and runtime querying.

## Source Discovery and Extraction

### Supported Sources

- Imported workspace document assets whose kind is `pdf` or `markdown`
- Standalone `.pdf`, `.md`, and `.markdown` files found under the workspace root and workspace `files/` directory

### Source Inventory

`raw/sources.json` is the canonical manifest for discovered sources. Each record should include:

- `sourceId`
- `workspaceId`
- `title`
- `slug`
- `kind`
- `absolutePath`
- `contentHash`
- `extractPath`
- `documentId`
- `status`
- `lastScanAt`
- `lastError`

### Extraction Rules

- PDF sources use MarkItDown and produce markdown under `raw/extracts/<source-slug>.md`.
- Markdown sources are normalized and copied into the same extracts directory.
- Extracted text is reusable input for distillation, wiki generation, and retrieval.
- Extraction is skipped when the source hash is unchanged and a valid extract already exists.

## Structured Memory Model

The first version uses four formal memory object types plus one append-only conversation log.

### Entities

`schema/entities.json` stores stable concepts such as:

- methods
- models
- datasets
- metrics
- tasks
- terms
- papers
- systems

Each entity must include:

- `id`
- `workspaceId`
- `title`
- `type`
- `summary`
- `aliases`
- `sourceRefs`
- `origin`
- `status`
- `confidence`
- `createdAt`
- `updatedAt`

### Claims

`schema/claims.json` stores structured statements such as:

- research goals
- method descriptions
- results
- limitations
- comparisons
- conclusions

Each claim must include:

- `id`
- `workspaceId`
- `title`
- `type`
- `summary`
- `entityIds`
- `sourceRefs`
- `origin`
- `status`
- `confidence`
- `createdAt`
- `updatedAt`

### Relations

`schema/relations.json` stores links across entities and claims, such as:

- `uses`
- `improves`
- `compares_with`
- `evaluated_on`
- `limited_by`
- `supports`
- `contradicts`

Each relation must include:

- `id`
- `workspaceId`
- `type`
- `fromId`
- `toId`
- `summary`
- `sourceRefs`
- `origin`
- `status`
- `confidence`
- `createdAt`
- `updatedAt`

### Tasks

`schema/tasks.json` stores open questions and next-step work items, such as:

- follow-up reading tasks
- unresolved conflicts
- verification tasks
- open research questions

Each task must include:

- `id`
- `workspaceId`
- `title`
- `type`
- `summary`
- `priority`
- `sourceRefs`
- `origin`
- `status`
- `confidence`
- `createdAt`
- `updatedAt`

### Source References

Every formal memory object must carry traceability:

```json
{
  "sourceId": "source:paper-a",
  "pageStart": 5,
  "pageEnd": 6,
  "excerpt": "..."
}
```

This traceability is required for evidence display in the reader sidebar and for wiki generation.

### Conversation Log

`schema/conversation-log.jsonl` is append-only and records:

- query input metadata
- retrieved evidence ids
- final answer
- candidate memories returned by the LLM
- promotion actions

This file is not the formal knowledge store. It is an operational event log.

## Per-Source Distillation

Structured memory extraction is performed per source, not directly against the aggregated workspace schema.

Each source emits:

`schema/by-source/<source-slug>.json`

This file contains:

- source metadata
- extracted entities
- extracted claims
- extracted relations
- extracted tasks
- source-local sourceRefs

Benefits of the per-source model:

- incremental reprocessing for only changed sources
- deterministic deletion when a source disappears
- simpler debugging of LLM output quality
- simpler aggregation and conflict detection

## Compilation

Compilation merges all `schema/by-source/*.json` files into aggregated schema outputs and wiki pages.

### Compilation Responsibilities

- normalize titles and aliases
- assign stable ids
- merge duplicates conservatively
- preserve conflicting claims instead of forcing a single truth
- remove knowledge contributions from deleted sources
- regenerate concept and overview pages

### Stable ID Rule

IDs must be stable across rescans. They should be derived from:

- object type
- normalized title
- anchored source identity

Random ids are not acceptable for aggregated schema objects because they would break references across rescans.

## Wiki Generation

Wiki pages are generated from `raw` plus compiled `schema`.

### Required Wiki Outputs

- `wiki/overview.md`
- `wiki/docs/<source-slug>.md`
- `wiki/concepts/<concept-slug>.md`
- `wiki/open-questions.md`

### Generation Intent

- `overview.md`
  High-level workspace map, document index, major themes, and important next questions
- `docs/*.md`
  One page per source focused on document summary, key methods, results, limitations, and related concepts
- `concepts/*.md`
  One page per concept/entity showing definition, related claims, related sources, and related concepts
- `open-questions.md`
  Aggregated unresolved tasks, conflicts, and follow-up directions

## Reader Sidebar Redesign

The current sidebar should stop behaving like a prompt workbench. The new first-version interaction model is a knowledge-grounded copilot.

### Sidebar Structure

- `Ask`
  A question input with scope selection:
  - current selection
  - current page/document
  - workspace context
- `Answer`
  Final grounded answer in markdown
- `Evidence`
  Retrieved structured memory items, wiki pages, and raw excerpts used for the answer
- `Promote`
  Candidate memories extracted from the answer that the user can save into formal memory

### First-Version Sidebar Actions

- ask a question
- inspect evidence
- open referenced wiki pages
- promote candidate memories
- trigger rescan for current document or full workspace

Preset prompt cards from the legacy sidebar are not part of the new core flow.

## Query Flow

Reader sidebar queries use:

- `workspaceId`
- active document metadata
- current page
- selected text
- user question

Retrieval order:

1. aggregated `schema`
2. relevant `wiki` pages
3. `raw/extracts` excerpts if more grounding is needed

The model response contract should internally produce:

- `answer`
- `candidates`

Where `candidates` are structured memory suggestions of type:

- `entity`
- `claim`
- `task`

This internal split is required even if the UI initially renders it as one answer panel plus a promote section.

## Write-Back Policy

### Scan Output

Scan and distillation results are automatically written into per-source schema files and compiled into formal memory.

### Chat Output

Chat output does not automatically become formal memory.

Instead:

- every query is logged to `conversation-log.jsonl`
- the LLM may return candidate memories
- the user explicitly promotes selected candidates into formal `schema` files

This avoids polluting workspace memory with noisy or speculative chat output while keeping the system efficient enough to use.

## Workspace Page Changes

The workspace page should evolve from a wiki-only result browser into a knowledge management surface.

### Required Workspace Views

- wiki pages
- entities/concepts
- key claims
- open questions/tasks
- recent promoted chat memories

The reader sidebar is responsible for question answering. The workspace page is responsible for browsing and managing accumulated workspace knowledge.

## Error Handling

### Source-Level Failure

Failures in extraction, distillation, or wiki generation for a single source should not fail the whole run.

These errors must be written to:

- `raw/sources.json`
- `schema/scan-runs/<job-id>.json`

### Run-Level Failure

A full run should only fail for workspace-wide errors such as:

- workspace cannot be read
- knowledge directories cannot be written
- compilation cannot produce coherent aggregated outputs

### Query Degradation

If one layer is unavailable:

- fall back from `schema` to `wiki`
- fall back from `wiki` to `raw`
- if no sufficient evidence exists, return an explicit uncertainty message instead of fabricating

### Conflict Handling

Conflicting claims from different sources must be preserved and surfaced as conflicts or open questions. They should not be force-merged into a single claim.

## Incremental Update Strategy

Incremental processing is mandatory in the first version.

- unchanged source hash means skip extraction and distillation
- changed source means regenerate only that source's per-source schema
- compilation then rebuilds aggregated schema and affected wiki outputs
- deleted sources remove their contributions during compilation

Manual triggers in the first version:

- rescan current document
- rescan entire workspace

No filesystem watching is included in this version.

## Testing Strategy

The test focus is contract stability and flow correctness, not exact wording from the model.

### Unit Tests

- source discovery
- hash-based incremental behavior
- path and directory layout
- schema read/write helpers
- duplicate merge rules
- deletion cleanup
- retrieval ranking and fallback selection

### Contract Tests

- per-source schema file shape
- aggregated schema file shape
- query service response structure

### Integration Tests

At least one complete flow:

1. import source
2. scan
3. distill
4. compile
5. generate wiki
6. run sidebar query
7. produce candidate memory
8. promote candidate memory

### Mocking Rule

All LLM-dependent tests should use deterministic mocked responses. Tests must not depend on real providers.

## Implementation Scope for Version 1

### Must Have

- file-based `raw / wiki / schema` knowledge directory
- workspace scan task for PDF and Markdown
- per-source structured memory distillation
- compile stage for entities, claims, relations, and tasks
- wiki generation for overview, document, concept, and open-question pages
- new reader sidebar flow using structured memory and wiki retrieval
- candidate memory promotion from chat into formal schema
- workspace page browsing for wiki and memory

### Explicitly Deferred

- embeddings
- vector retrieval
- automatic background sync
- graph editor
- legacy sidebar compatibility
- unreviewed automatic promotion of all chat output
- collaborative sync

## Migration Guidance

Existing workspace wiki job infrastructure, PDF extraction, provider/model configuration, and workspace UI shells should be reused where useful. However, legacy reader AI behaviors, old chat-history-centric flows, and old prompt-card assumptions are not a compatibility target for this version.

The new knowledge layer becomes the primary source of truth. The reader sidebar and future AI workspace features should consume this layer rather than building parallel memory flows.
