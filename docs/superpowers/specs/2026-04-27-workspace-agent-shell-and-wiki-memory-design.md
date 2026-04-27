# Workspace Agent Shell And Wiki Memory Design

## Background

OpenSciReader already contains several pieces that point toward an agentic research workspace:

- a desktop shell built with Wails and React
- PDF reading and page-level context capture
- imported workspace documents
- a file-based workspace wiki pipeline
- a reader-side AI panel
- model/provider configuration and tool access

The current problem is not the absence of features. The problem is that the product center is unclear.

Today, the codebase mixes three different product shapes:

- a PDF reading assistant
- a workspace wiki browser
- a prompt-driven AI tools panel

That leaves the UI and logic fragmented:

- the workspace page tries to show documents, scan controls, source state, and wiki pages at the same time
- the reader panel mixes legacy prompt tools with the newer knowledge-grounded copilot direction
- the wiki layer exists, but it is still treated too much like a generated artifact instead of the assistant's persistent memory

The design in this document reframes the product around a clearer thesis:

**OpenSciReader should behave like an agentic research assistant with a persistent wiki memory system.**

The primary product is the agent. The wiki is the agent's strongest long-term memory substrate. Skills are the agent's capability system. The reader and workspace are two different entry surfaces into the same agent core.

## Design Decisions

The following decisions were fixed during brainstorming and should be treated as product requirements for this design:

1. The first version must support both paper understanding and research task execution. Neither is secondary.
2. The product shell should feel closer to an OpenClaw or Codex Desktop style agent workspace than to a dashboard-style wiki browser.
3. The workspace should use a `session-first` agent shell as its primary experience.
4. The wiki is a persistent memory system for the agent, not the top-level product itself.
5. Skills must support both explicit user triggering and agent-selected orchestration.
6. The wiki memory architecture should follow the LLM wiki pattern: accumulated wiki pages are the main memory layer, while structured runtime state remains a support layer.
7. Importing sources and building memory are separate operations.
8. Memory build is manual in version 1 and runs at the workspace level.
9. Version 1 memory build should only process documents that were explicitly imported into the workspace. It should not scan arbitrary loose files under the workspace directory.
10. Query grounding should prefer wiki memory first, then runtime state, then extracted source text, and finally raw source fallback.
11. Promote is required in version 1.
12. Promote should create new wiki insight pages in version 1 instead of directly editing core overview or concept pages.
13. The workspace surface and reader surface must share one agent core, one session model, one skill system, and one wiki memory system.

## Goals

- Make the product agent-first instead of wiki-first.
- Preserve the LLM wiki idea, but reposition it as the assistant's persistent memory layer.
- Make the workspace feel like an agent shell for ongoing research sessions.
- Make the reader feel like a focused entry point into the same agent, using current PDF context.
- Separate memory, skills, sessions, and UI surfaces so the system is easier to reason about and evolve.
- Ship a version 1 loop that is actually usable end to end:
  - import curated sources
  - manually build workspace memory
  - ask grounded questions
  - generate reading and research outputs
  - promote useful results back into wiki memory

## Non-Goals

- No attempt to make the workspace home page a general analytics dashboard.
- No filesystem-wide automatic source discovery in version 1.
- No version 1 requirement for autonomous background memory maintenance after every import.
- No attempt to make the runtime state the primary user-facing memory browser.
- No requirement to directly rewrite core wiki pages during promote in version 1.
- No requirement to replace the current entire tool stack with a new framework before restructuring the product model.

## Product Thesis

The core product idea is:

1. The user works inside a workspace that represents a research project.
2. The user interacts with an agent through long-running sessions.
3. The agent can invoke skills to read, compare, summarize, plan, build memory, and execute tools.
4. The agent grounds itself in a persistent wiki memory built from the workspace's curated sources.
5. Valuable outputs from both reading and research sessions can be promoted back into the wiki so the memory compounds over time.

This design keeps the assistant useful in two equally important modes:

- understanding a current PDF in detail
- executing longer-running cross-source research tasks

## Core Model

### Agent First

The product should be designed around one central agent experience, not around a generated wiki browser.

That means:

- the center of the product is an agent thread or working canvas
- the user's primary action is giving the agent work
- wiki memory, evidence, and tools are context systems that support the agent

### One Agent, Two Surfaces

There are two entry surfaces into the same agent core:

- `Workspace Surface`
- `Reader Surface`

They must not become two separate assistants with diverging histories and capabilities.

They should share:

- the same workspace binding
- the same session engine
- the same skill registry and router
- the same wiki memory system
- the same tool runner

The difference is not in the assistant identity. The difference is in the default context each surface contributes.

## Surface Design

### Workspace Surface

The workspace is the primary agent shell.

Its job is to support:

- long-running research sessions
- task planning and multi-step execution
- cross-source synthesis
- memory build operations
- browsing and inspecting the accumulated wiki memory
- reviewing promoted results

The workspace should feel like a research workbench rather than a management dashboard.

#### Layout Direction

The approved direction combines:

- the lighter left-rail structure from the user-provided reference UI
- the center-thread focus of Codex Desktop

The recommended shell has three persistent zones:

1. **Left rail**
   - workspace and project selector
   - session list
   - lightweight mode switching
   - pinned or recent items
2. **Center canvas**
   - the main agent thread
   - working trace
   - answers, plans, tool output, and promoted-result proposals
3. **Right context pane**
   - wiki memory hits
   - evidence panels
   - suggested skills
   - tools and related context

The center canvas is the default focus. The user should not land on a wiki tree or a panel grid by default.

#### Primary Modes

The workspace shell should support two top-level modes:

- `Sessions`
- `Knowledge`

`Sessions` is the default home.

`Knowledge` is a memory browser and maintenance surface, not the default first impression.

### Reader Surface

The reader is a focused surface into the same agent.

Its job is to support:

- current PDF understanding
- page-level or selection-level questioning
- quick reading outputs
- evidence-backed answers tied to the active document context
- rapid promote of useful results back into workspace memory

The reader should stay lighter than the workspace shell.

It should not host:

- full session management UI
- broad workspace maintenance controls
- complex build orchestration controls

Its main value is that it contributes stronger local context:

- current PDF
- current page
- current selection
- figure or snapshot context when available

## Workspace Modes

### Sessions Mode

Sessions mode is the main product experience.

It contains:

- current session thread
- session list for the workspace
- agent command bar
- working trace or step trace
- right-side context panels for memory, evidence, skills, and tools

Sessions are not just chat histories. A session is a research process container.

Each session should be able to accumulate:

- user goals
- task progress
- invoked skills
- generated outputs
- promoteable results
- links to relevant sources and wiki memory

### Knowledge Mode

Knowledge mode is the memory browser and memory maintenance surface.

Its role is to let the user inspect what the agent currently knows.

Recommended knowledge sections are:

- `Overview`
- `Sources`
- `Wiki`
- `Insights`
- `Builds`

This keeps memory browsing accessible without making memory browsing the product's default center.

## Memory Architecture

### Main Principle

Following the LLM wiki pattern, the main persistent memory is the wiki itself.

The wiki is not just a generated output folder. It is the agent's accumulated, human-readable, cross-linked memory graph.

Structured runtime state still matters, but it is a support layer, not the main memory surface.

### Core Memory Objects

The approved model uses four core object categories:

1. `Source`
2. `Build`
3. `Wiki`
4. `Runtime State`

#### Source

Sources are the curated documents explicitly imported into the workspace.

They are:

- immutable inputs
- the final provenance base
- the only documents version 1 build should process

#### Build

Build is a first-class user-understandable object.

It represents one explicit workspace-level memory build run.

The user should be able to understand:

- when a build ran
- whether it succeeded
- which sources failed
- what wiki memory changed

#### Wiki

Wiki is the primary persistent memory layer.

The agent reads from it, extends it, and files useful outputs back into it.

Version 1 wiki structure should include at least:

```text
wiki/
  index.md
  overview.md
  open-questions.md
  log.md
  docs/
    <source-pages>.md
  concepts/
    <concept-pages>.md
  insights/
    <promoted-analysis-pages>.md
```

Special roles:

- `index.md`
  - the main content-oriented navigation page for both the user and the agent
- `log.md`
  - the chronological activity ledger for builds, promotes, and important events
- `overview.md`
  - top-level workspace synthesis
- `docs/`
  - per-source pages
- `concepts/`
  - cross-source concept pages
- `insights/`
  - promoted analysis outputs from agent sessions and reader interactions

#### Runtime State

Runtime state supports:

- source processing state
- extracted intermediate representations
- build summaries
- evidence anchors and source provenance
- incremental build logic
- query support and lint support

Runtime state is allowed to stay structured and machine-oriented.

It should not replace the wiki as the primary user-facing memory layer.

## Filesystem Direction

The clarified workspace memory layout remains compatible with the earlier file-first architecture direction:

```text
workspaces/<workspace-id>/
  files/
  sources/
    pdfs/
  inputs/
    markitdown/
    manifests/
  state/
    sources.json
    by-source/
    jobs/
    compile-summary.json
  wiki/
    index.md
    overview.md
    open-questions.md
    log.md
    docs/
    concepts/
    insights/
```

In this design:

- `sources/` stores immutable curated assets
- `inputs/` stores reproducible extract caches
- `state/` stores machine-readable support state
- `wiki/` stores the accumulated human-readable memory graph

## Session Model

Sessions are central to the agent shell.

A session is not merely a transcript. It is the unit of ongoing research activity.

Each session should carry:

- `workspaceId`
- session title
- chronological agent thread
- related task intent
- invoked skill history
- references to relevant memory artifacts
- candidate promote outputs

Reader-originated asks and outputs should attach to the same session system instead of creating a second isolated history model.

That allows:

- a reading question to continue as a broader research task
- a workspace task to jump into reader-grounded evidence
- promoted results to remain traceable to the session where they were produced

## Skills System

### Skill Role

Skills are capability units, not pages and not storage layers.

They define what the agent can do.

The first version must support both:

- explicit user-triggered skill invocation
- agent-selected skill routing when the user gives a higher-level task

### Must-Have Skills For Version 1

The approved initial skill set is:

- `Ask with evidence`
- `Reading outputs`
- `Task planning`
- `Build memory`
- `Cross-source synthesis`
- `Promote to wiki`
- `Tool execution`

### Skill Responsibilities

#### Ask With Evidence

Ground an answer in current session context plus workspace memory and evidence.

#### Reading Outputs

Generate structured outputs such as summaries, method breakdowns, tables, and reading notes.

#### Task Planning

Turn a research objective into explicit tasks and a working plan inside the current session.

#### Build Memory

Trigger the workspace-level memory build workflow and report results clearly.

#### Cross-Source Synthesis

Compare, merge, contrast, and synthesize across multiple papers or sources.

#### Promote To Wiki

File valuable agent output back into `wiki/insights/` and update navigation pages.

#### Tool Execution

Run external tools or scripts when a skill needs non-LLM support.

## Agent Core Responsibilities

The agent core should be the orchestrator of the system.

Its responsibilities are:

- maintain session continuity
- decide whether a request needs one skill or several
- request memory evidence from the wiki memory adapter
- coordinate tool execution
- collect outputs and proposed promote candidates
- present clear step traces to the center canvas

The UI surfaces should not each implement their own orchestration logic.

## Operations

### 1. Import

Import is the act of registering a curated document into the workspace.

Import should:

1. store or register the source in the workspace's curated source set
2. update the workspace source manifest
3. make the source available for future memory build runs

Import does **not** automatically imply memory build in version 1.

### 2. Build Workspace Memory

Build is a manual, workspace-level action in version 1.

The user triggers `Build Workspace` from the workspace shell.

The build pipeline should:

1. read the explicitly imported workspace sources
2. generate or reuse extract caches
3. update per-source runtime state
4. compile a wiki rewrite plan
5. update wiki memory atomically
6. write or update build summaries and logs

The user-visible result of a build should be understandable in terms of:

- sources processed
- failures
- updated wiki pages
- current memory freshness

### 3. Ask

Ask can originate from either surface.

Reader asks contribute stronger local document context.
Workspace asks contribute broader session and research context.

The grounding order for factual retrieval should be:

1. session context for conversational continuity
2. `wiki/index.md` and relevant wiki pages
3. runtime state for structured support and provenance anchors
4. extracted source text caches
5. raw source fallback when needed

This preserves the compounding-memory property of the LLM wiki pattern.

### 4. Reading Outputs

Reading outputs are not separate standalone products. They are agent work products.

They should be generated through the skill system and remain attachable to the current session, with optional later promote.

### 5. Task Planning

Task planning should be available inside the agent shell because the first version must support ongoing research execution, not just document Q&A.

Planned tasks live in session context first. They do not automatically become wiki memory.

### 6. Cross-Source Synthesis

Cross-source synthesis is a first-class skill because the product must support both reading and research.

This skill should read across:

- relevant wiki pages
- related source pages
- extracted evidence

The result may become a promote candidate.

### 7. Promote

Promote is required in version 1.

The first promote target should be:

- a new page under `wiki/insights/`

Promote should:

1. let the user confirm the result is worth filing
2. create a new insight or analysis page
3. update `index.md`
4. append a `log.md` entry
5. make the new page available for future retrieval

Promote should **not** directly rewrite `overview.md` or concept pages in version 1.

That constraint keeps the workflow safer while still honoring the key LLM wiki principle that good answers can be filed back into memory.

## Knowledge Mode Contents

Although the product is agent-first, the knowledge browser still needs a clear structure.

The approved knowledge sections are:

- `Overview`
- `Sources`
- `Wiki`
- `Insights`
- `Builds`

Purpose of each section:

- `Overview`
  - memory-level summary of the workspace
- `Sources`
  - what was imported, what failed, what is available to build
- `Wiki`
  - core memory pages such as overview, docs, concepts, open questions, and log
- `Insights`
  - promoted analysis pages from sessions and reader interactions
- `Builds`
  - current and recent build results

This gives the user a clear memory browser without turning memory browsing into the default product home.

## Context Pane Model

The right-side context pane in the workspace shell should support at least four context categories:

- `Skills`
- `Memory`
- `Evidence`
- `Tools`

This pane is not the main stage.

Its purpose is to let the user inspect and steer the agent while the center canvas remains focused on the ongoing thread.

## PDF Understanding Integration

PDF understanding remains a primary use case, but it should now fit into the broader agent model instead of living as a special-case side tool.

The reader contributes:

- current document identity
- current page
- selected text
- visual context where available

That local context should be passed into the shared agent core.

The agent then decides whether to:

- answer directly from relevant wiki memory
- retrieve additional evidence from runtime state or source extracts
- invoke a reading-output skill
- create a promote candidate

## Implementation Direction

This design intentionally favors a re-architecture of product boundaries before large-scale feature expansion.

The most important implementation shift is:

- move from `workspace page + reader panel + wiki pipeline` as loosely related subsystems
- toward `shared agent core + sessions + skills + wiki memory` as the product backbone

That means future implementation work should prioritize:

1. shared session model
2. agent shell layout
3. skill registry and routing layer
4. reader integration into the same agent core
5. wiki memory build and promote flows

## Verification Expectations

The design should be considered correctly implemented only if it demonstrates all of the following:

1. The workspace opens into an agent-first shell, not a mixed dashboard.
2. The reader and workspace use the same underlying agent session system.
3. Memory build is manual, workspace-level, and only uses explicitly imported sources.
4. Query grounding prefers wiki memory before dropping into lower layers.
5. Promote creates reusable wiki insight pages that become future memory.
6. The skill system can support both user-triggered and agent-selected execution.

## Summary

The approved direction is:

- an agent-first workspace shell inspired by OpenClaw-style research UI and Codex Desktop center-thread layout
- one shared agent core used by both workspace and reader surfaces
- a skill system as the capability layer
- a wiki memory system as the primary long-term memory layer
- runtime state as a support layer for provenance, build, and retrieval
- a version 1 loop of curated import, manual memory build, grounded ask, useful output generation, and promote back into wiki insight pages

This preserves the strongest part of the current LLM wiki work while fixing the larger product problem: the agent, memory, sessions, and surfaces now have clear roles.
