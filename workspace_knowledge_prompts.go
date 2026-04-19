package main

import (
	"fmt"
	"strings"
)

func buildWorkspaceKnowledgeBySourcePrompt(workspace Workspace, source WorkspaceKnowledgeSource, markdown string) string {
	return strings.TrimSpace(fmt.Sprintf(`
Extract structured workspace knowledge from a single source and return JSON only.

Workspace:
- workspaceId: %s
- workspaceName: %s

Source:
- sourceId: %s
- title: %s
- slug: %s
- kind: %s
- documentId: %s

Return a JSON object with this exact top-level shape:
{
  "source": {
    "sourceId": "...",
    "workspaceId": "...",
    "title": "...",
    "slug": "...",
    "kind": "...",
    "absolutePath": "",
    "contentHash": "",
    "extractPath": "",
    "documentId": "...",
    "status": "ready",
    "lastScanAt": "",
    "lastError": ""
  },
  "entities": [],
  "claims": [],
  "relations": [],
  "tasks": []
}

Rules:
- Return valid JSON only. No markdown fences. No prose outside JSON.
- Preserve the provided workspaceId and sourceId in every extracted object.
- Include evidence in sourceRefs whenever the source supports it.
- Use empty arrays instead of null.
- Do not invent facts that are not grounded in the source text.

Extracted markdown:
%s
`, workspace.ID, workspace.Name, source.ID, source.Title, source.Slug, source.Kind, source.DocumentID, strings.TrimSpace(markdown)))
}
