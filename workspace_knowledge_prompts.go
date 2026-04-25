package main

import (
	"fmt"
	"strings"
)

func buildWorkspaceKnowledgeBySourcePrompt(workspace Workspace, source WorkspaceKnowledgeSource, markdown string) string {
	return strings.TrimSpace(fmt.Sprintf(`
Extract structured workspace knowledge from one entry in sources/ using the markdown already prepared in inputs/markitdown/.
Return JSON only for the machine-readable state/by-source/ payload that will later compile into aggregate state/ and human-readable wiki/ pages.

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
    "sourcePath": "",
    "contentHash": "",
    "markItDownPath": "",
    "documentId": "...",
    "markItDownStatus": "ready",
    "extractStatus": "ready",
    "lastIngestAt": "",
    "lastSuccessAt": "",
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

Markdown from inputs/markitdown:
%s
`, workspace.ID, workspace.Name, source.ID, source.Title, source.Slug, source.Kind, source.DocumentID, strings.TrimSpace(markdown)))
}
