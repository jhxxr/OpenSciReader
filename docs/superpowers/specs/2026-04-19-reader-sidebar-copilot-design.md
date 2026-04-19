# Reader Sidebar Knowledge-Grounded Copilot Design

## Overview

Redesign the ReaderAIPanel from a prompt-driven workbench to a knowledge-grounded copilot that leverages the workspace knowledge layer for question answering, evidence retrieval, and memory promotion.

## Goals

- Replace preset cards and chat history with a four-section copilot interface
- Ground answers in workspace knowledge (schema + wiki + raw extracts)
- Show evidence transparency with grouped, collapsible evidence sections
- Enable candidate memory promotion from answers into formal workspace memory
- Support multi-scope questioning (selection + page + document + workspace)

## Non-Goals

- Preserve legacy preset cards (summary, table, methods, selection interpretation)
- Maintain chat history as primary interaction pattern
- Vector search or embeddings (deferred to future versions)
- Automatic promotion of all chat output
- Graph visualization or advanced knowledge editing UI

## Architecture

### Four-Section Layout

The sidebar consists of four vertically stacked sections:

1. **Ask** (top) - Question input with multi-scope selection
2. **Answer** (upper middle) - Grounded answer in Markdown
3. **Evidence** (lower middle, collapsible) - Retrieved knowledge grouped by type
4. **Promote** (bottom, collapsible) - Candidate memories with expandable details

### Component Structure

Single-file approach: `ReaderAIPanel.tsx` completely rewritten with:

- Internal state management using React hooks
- Helper functions for formatting and rendering
- Event handlers for all user interactions
- Error handling at each section boundary

## State Management

```typescript
interface ReaderAIPanelState {
  // Ask section
  question: string;
  scope: {
    selection: boolean;
    page: boolean;
    document: boolean;
    workspace: boolean;
  };
  isAsking: boolean;

  // Answer section
  answer: string | null;
  answerError: string | null;

  // Evidence section
  evidence: {
    entities: WorkspaceKnowledgeEvidenceHit[];
    claims: WorkspaceKnowledgeEvidenceHit[];
    tasks: WorkspaceKnowledgeEvidenceHit[];
    sources: Array<{kind: string, id: string, title: string, excerpt: string}>;
  };
  expandedGroups: {
    entities: boolean;
    claims: boolean;
    tasks: boolean;
    sources: boolean;
  };

  // Promote section
  candidates: WorkspaceKnowledgeCandidate[];
  expandedCandidates: Set<string>;
  promotingIds: Set<string>;
  promoteError: string | null;
}
```

### State Update Flow

1. User submits question → `isAsking: true`, call API
2. API returns → update `answer` and `evidence`, `isAsking: false`
3. Answer contains candidates → update `candidates`
4. User toggles expand/collapse → update `expandedGroups` or `expandedCandidates`
5. User promotes → update `promotingIds`, call API, remove promoted candidates

## Ask Section

### Scope Selection (Multi-Select Checkboxes)

- ☐ Current Selection (enabled only when text is selected)
- ☐ Current Page
- ☐ Current Document
- ☐ Workspace Context

Users can combine multiple scopes (e.g., selection + workspace context).

### Input

- Multi-line textarea for question entry
- Placeholder: "向工作区知识提问..."
- Ask button with loading state: "提问中..." vs "Ask"
- Validation: disable button if question is empty

## Answer Section

### Display

- Markdown rendering using existing `MarkdownPreview` component
- Loading state while query is in progress
- Error message display on failure
- Empty state when no relevant information found

## Evidence Section

### Grouped, Collapsible Layout

Evidence is grouped into four collapsible sections:

1. **Entities / Concepts** - Extracted concepts and entities
2. **Key Claims** - Structured statements from knowledge
3. **Open Questions / Tasks** - Follow-up work items
4. **Sources** - Wiki pages and raw excerpts

Each section:
- Shows count badge in header
- Toggle button to expand/collapse
- Items sorted by relevance within groups

### Evidence Item Display

Each evidence item shows:
- Title (clickable to open wiki page if applicable)
- Type badge
- Confidence score percentage
- Summary or excerpt
- Source references (page numbers, document IDs)
- Kind indicator (entity/claim/task/wiki_page/raw_excerpt)

## Promote Section

### Preview-First Interaction

Candidates displayed with:
- Summary view by default (title + type + confidence)
- Expand/collapse button to show full details
- Independent Promote button per candidate

### Expandable Details

When expanded, candidates show:
- Full summary text
- Aliases (for entities)
- Linked entity IDs (for claims)
- Source references with page ranges
- Source count summary

### Promote Flow

1. User clicks Promote → button shows "Promoting..."
2. API call to `workspaceKnowledgeApi.promoteCandidates()`
3. On success: remove candidate from list
4. On error: show error message, keep candidate for retry

## API Integration

### Query API

```typescript
async queryWorkspaceKnowledge(
  workspaceId: string,
  providerId: number,
  modelId: number,
  question: string,
  scope: {
    selection?: string;
    currentPage?: number;
    documentId?: string;
  }
): Promise<WorkspaceKnowledgeQueryResult>
```

### Result Structure

```typescript
interface WorkspaceKnowledgeQueryResult {
  answer: string;  // Final grounded answer in Markdown
  evidence: WorkspaceKnowledgeEvidenceHit[];  // Retrieved evidence items
  candidates: WorkspaceKnowledgeCandidate[];  // Candidate memories for promotion
}

interface WorkspaceKnowledgeEvidenceHit {
  kind: "entity" | "claim" | "task" | "wiki_page" | "raw_excerpt";
  id: string;
  title: string;
  summary: string;
  excerpt: string;
  sourceRefs: WorkspaceKnowledgeSourceRef[];
}
```

### Evidence Grouping Logic

Evidence array is grouped by `kind` field:
- `entity`, `claim`, `task` → map to schema knowledge types
- `wiki_page` → wiki page references
- `raw_excerpt` → raw text extracts

## Error Handling

### Network/API Errors

- **Ask failure**: Display error in Answer section, reset `isAsking`
- **Promote failure**: Display error in Promote section, keep candidate for retry
- Use `getErrorMessage(error, context)` for consistent formatting

### Empty States

- No query results: "No relevant information found in workspace knowledge."
- No evidence: Group-specific empty messages
- No candidates: "No candidate memories extracted from this answer."

### Edge Cases

- Empty question input: Disable Ask button
- No text selection: Disable "当前选区" checkbox
- Workspace not active: Disable Ask section with guidance message
- Model not configured: Show error directing user to configure model first

### Race Conditions

- Prevent duplicate Ask requests while one is in progress
- Handle independent promote operations with Set tracking
- Cleanup pending requests on component unmount

## Performance Considerations

1. **Default Collapsed State**: Evidence groups collapsed by default to reduce initial render
2. **Memoization**: Use React.memo for MarkdownPreview and list items
3. **Virtualization**: Implement virtual scrolling if evidence exceeds 20 items
4. **Debouncing**: Consider debouncing question input (optional for future)

## Styling

### CSS Classes

- `.reader-ask-section` - Top question input area
- `.reader-answer-section` - Answer display area
- `.reader-evidence-section` - Evidence container
- `.reader-promote-section` - Promote candidate list
- `.evidence-group` - Individual evidence type group
- `.evidence-group-header` - Expandable group header
- `.evidence-list` - Items within a group
- `.candidate-item` - Single promote candidate
- `.candidate-summary` - Default collapsed view
- `.candidate-details` - Expanded details view

### Responsive Design

- Minimum width: 300px for sidebar
- Collapsible sections help manage vertical space
- Textareas adapt to container width

## Testing Strategy

### Unit Tests

- State management: verify all state transitions
- Helper functions: formatting and rendering utilities
- Error handling: mock API failures
- Edge cases: empty inputs, disabled states

### Integration Tests

- Full query flow: Ask → Answer → Evidence → Promote
- Multi-scope queries: verify scope combinations
- Expand/collapse: verify state persistence
- Error recovery: verify retry after failures

### Manual Testing

- Test with actual workspace knowledge data
- Verify evidence grouping matches backend response
- Test promote flow with real API calls
- Validate markdown rendering of answers

## Migration Notes

### Breaking Changes

- Preset cards removed: `summary`, `table`, `methods`, `selection`
- Chat history no longer displayed in sidebar
- Legacy prompt patterns not supported

### Preserved Functionality

- Workspace context integration remains
- Model/provider selection continues to work
- PDF text context loading unchanged

## Future Enhancements (Out of Scope)

- Vector search with embeddings
- Automatic knowledge sync on document changes
- Graph visualization of knowledge relationships
- Batch promote all candidates
- Candidate editing before promote
- Knowledge conflict resolution UI
- Wiki page inline preview

## Implementation Checklist

- [ ] Rewrite ReaderAIPanel.tsx with four-section layout
- [ ] Implement state management with React hooks
- [ ] Build Ask section with multi-scope checkboxes
- [ ] Integrate query API with scope parameters
- [ ] Implement Answer section with Markdown rendering
- [ ] Build Evidence section with grouped, collapsible layout
- [ ] Implement Promote section with expandable candidates
- [ ] Add error handling for all API calls
- [ ] Handle edge cases and empty states
- [ ] Add loading states and disabled states
- [ ] Style all sections with consistent CSS
- [ ] Write unit tests for core logic
- [ ] Manual testing with real data
- [ ] Update documentation and commit

## Success Criteria

- Users can ask questions with configurable scope
- Answers are grounded in workspace knowledge
- Evidence is transparently displayed and organized
- Candidates can be reviewed and promoted
- All error states are handled gracefully
- No preset cards or legacy chat history in sidebar
