package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"
)

type WorkspaceKnowledgeSnapshot struct {
	Sources   []WorkspaceKnowledgeSource   `json:"sources"`
	Entities  []WorkspaceKnowledgeEntity   `json:"entities"`
	Claims    []WorkspaceKnowledgeClaim    `json:"claims"`
	Relations []WorkspaceKnowledgeRelation `json:"relations"`
	Tasks     []WorkspaceKnowledgeTask     `json:"tasks"`
}

type workspaceKnowledgeConceptPage struct {
	Slug         string
	Entity       WorkspaceKnowledgeEntity
	Claims       []WorkspaceKnowledgeClaim
	SourceTitles []string
}

func CompileWorkspaceKnowledge(files workspaceKnowledgeFiles, workspaceTitle string) (WorkspaceKnowledgeSnapshot, error) {
	if err := files.EnsureLayout(); err != nil {
		return WorkspaceKnowledgeSnapshot{}, err
	}

	payloads, err := readWorkspaceKnowledgePayloads(files)
	if err != nil {
		return WorkspaceKnowledgeSnapshot{}, err
	}

	snapshot := WorkspaceKnowledgeSnapshot{
		Sources:   mapSources(payloads),
		Entities:  mapEntities(payloads),
		Claims:    mapClaims(payloads),
		Relations: mapRelations(payloads),
		Tasks:     mapTasks(payloads),
	}

	if err := writeWorkspaceKnowledgeAggregates(files, snapshot); err != nil {
		return WorkspaceKnowledgeSnapshot{}, err
	}
	if err := writeWorkspaceKnowledgeWiki(files, strings.TrimSpace(workspaceTitle), snapshot, payloads); err != nil {
		return WorkspaceKnowledgeSnapshot{}, err
	}

	return snapshot, nil
}

func readWorkspaceKnowledgePayloads(files workspaceKnowledgeFiles) ([]WorkspaceKnowledgeBySourcePayload, error) {
	paths, err := files.BySourcePaths()
	if err != nil {
		return nil, err
	}

	payloads := make([]WorkspaceKnowledgeBySourcePayload, 0, len(paths))
	for _, path := range paths {
		var payload WorkspaceKnowledgeBySourcePayload
		if err := readWorkspaceKnowledgeJSON(path, &payload); err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}

	sort.Slice(payloads, func(i, j int) bool {
		return lessSource(payloads[i].Source, payloads[j].Source)
	})
	return payloads, nil
}

func writeWorkspaceKnowledgeAggregates(files workspaceKnowledgeFiles, snapshot WorkspaceKnowledgeSnapshot) error {
	entitiesPath, err := files.EntitiesPath()
	if err != nil {
		return err
	}
	if err := writeWorkspaceKnowledgeJSON(entitiesPath, snapshot.Entities); err != nil {
		return err
	}

	claimsPath, err := files.ClaimsPath()
	if err != nil {
		return err
	}
	if err := writeWorkspaceKnowledgeJSON(claimsPath, snapshot.Claims); err != nil {
		return err
	}

	relationsPath, err := files.RelationsPath()
	if err != nil {
		return err
	}
	if err := writeWorkspaceKnowledgeJSON(relationsPath, snapshot.Relations); err != nil {
		return err
	}

	tasksPath, err := files.TasksPath()
	if err != nil {
		return err
	}
	if err := writeWorkspaceKnowledgeJSON(tasksPath, snapshot.Tasks); err != nil {
		return err
	}

	return nil
}

func writeWorkspaceKnowledgeWiki(files workspaceKnowledgeFiles, workspaceTitle string, snapshot WorkspaceKnowledgeSnapshot, payloads []WorkspaceKnowledgeBySourcePayload) error {
	docsDir, err := files.docsDir()
	if err != nil {
		return err
	}
	if err := clearWorkspaceKnowledgeMarkdownDir(docsDir); err != nil {
		return err
	}

	conceptsDir, err := files.conceptsDir()
	if err != nil {
		return err
	}
	if err := clearWorkspaceKnowledgeMarkdownDir(conceptsDir); err != nil {
		return err
	}

	overviewPath, err := files.OverviewPath()
	if err != nil {
		return err
	}
	conceptSlugs := buildConceptSlugs(snapshot.Entities)
	sourceByID := buildSourceByID(snapshot.Sources)
	conceptPages := buildConceptPages(snapshot, conceptSlugs, sourceByID)

	if err := writeWorkspaceKnowledgeMarkdown(overviewPath, buildOverviewWikiPage(workspaceTitle, snapshot, conceptSlugs)); err != nil {
		return err
	}

	openQuestionsPath, err := files.OpenQuestionsPath()
	if err != nil {
		return err
	}
	if err := writeWorkspaceKnowledgeMarkdown(openQuestionsPath, buildOpenQuestionsPage(snapshot.Tasks, sourceByID)); err != nil {
		return err
	}

	for _, payload := range payloads {
		documentPath, err := files.DocumentWikiPath(firstNonEmptyText(payload.Source.Slug, payload.Source.ID))
		if err != nil {
			return err
		}
		if err := writeWorkspaceKnowledgeMarkdown(documentPath, buildDocumentWikiPage(payload, conceptSlugs)); err != nil {
			return err
		}
	}

	for _, page := range conceptPages {
		conceptPath, err := files.ConceptWikiPath(page.Slug)
		if err != nil {
			return err
		}
		if err := writeWorkspaceKnowledgeMarkdown(conceptPath, buildConceptWikiPage(page)); err != nil {
			return err
		}
	}

	return nil
}

func mapSources(payloads []WorkspaceKnowledgeBySourcePayload) []WorkspaceKnowledgeSource {
	sources := make([]WorkspaceKnowledgeSource, 0, len(payloads))
	for _, payload := range payloads {
		sources = append(sources, payload.Source)
	}
	sort.Slice(sources, func(i, j int) bool {
		return lessSource(sources[i], sources[j])
	})
	return sources
}

func mapEntities(payloads []WorkspaceKnowledgeBySourcePayload) []WorkspaceKnowledgeEntity {
	entities := make([]WorkspaceKnowledgeEntity, 0)
	for _, payload := range payloads {
		entities = append(entities, payload.Entities...)
	}
	sort.Slice(entities, func(i, j int) bool {
		return lessEntity(entities[i], entities[j])
	})
	return entities
}

func mapClaims(payloads []WorkspaceKnowledgeBySourcePayload) []WorkspaceKnowledgeClaim {
	claims := make([]WorkspaceKnowledgeClaim, 0)
	for _, payload := range payloads {
		claims = append(claims, payload.Claims...)
	}
	sort.Slice(claims, func(i, j int) bool {
		return lessClaim(claims[i], claims[j])
	})
	return claims
}

func mapRelations(payloads []WorkspaceKnowledgeBySourcePayload) []WorkspaceKnowledgeRelation {
	relations := make([]WorkspaceKnowledgeRelation, 0)
	for _, payload := range payloads {
		relations = append(relations, payload.Relations...)
	}
	sort.Slice(relations, func(i, j int) bool {
		return lessRelation(relations[i], relations[j])
	})
	return relations
}

func mapTasks(payloads []WorkspaceKnowledgeBySourcePayload) []WorkspaceKnowledgeTask {
	tasks := make([]WorkspaceKnowledgeTask, 0)
	for _, payload := range payloads {
		tasks = append(tasks, payload.Tasks...)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return lessTask(tasks[i], tasks[j])
	})
	return tasks
}

func buildOverviewWikiPage(workspaceTitle string, snapshot WorkspaceKnowledgeSnapshot, conceptSlugs map[string]string) string {
	title := firstNonEmptyText(strings.TrimSpace(workspaceTitle), "Workspace Knowledge")

	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	builder.WriteString("- Documents: ")
	builder.WriteString(fmt.Sprintf("%d", len(snapshot.Sources)))
	builder.WriteString("\n")
	builder.WriteString("- Concepts: ")
	builder.WriteString(fmt.Sprintf("%d", len(snapshot.Entities)))
	builder.WriteString("\n")
	builder.WriteString("- Claims: ")
	builder.WriteString(fmt.Sprintf("%d", len(snapshot.Claims)))
	builder.WriteString("\n")
	builder.WriteString("- Open Questions: ")
	builder.WriteString(fmt.Sprintf("%d", len(snapshot.Tasks)))
	builder.WriteString("\n\n")
	builder.WriteString("## Documents\n\n")
	if len(snapshot.Sources) == 0 {
		builder.WriteString("None.\n")
	} else {
		for _, source := range snapshot.Sources {
			builder.WriteString("- [")
			builder.WriteString(firstNonEmptyText(source.Title, source.Slug, source.ID))
			builder.WriteString("](docs/")
			builder.WriteString(firstNonEmptyText(source.Slug, workspaceKnowledgeSlug(source.ID)))
			builder.WriteString(".md)\n")
		}
	}

	builder.WriteString("\n## Concepts\n\n")
	if len(snapshot.Entities) == 0 {
		builder.WriteString("None.\n")
	} else {
		for _, entity := range snapshot.Entities {
			builder.WriteString("- [")
			builder.WriteString(firstNonEmptyText(entity.Title, entity.ID))
			builder.WriteString("](concepts/")
			builder.WriteString(conceptSlugs[entity.ID])
			builder.WriteString(".md)")
			if strings.TrimSpace(entity.Summary) != "" {
				builder.WriteString(": ")
				builder.WriteString(strings.TrimSpace(entity.Summary))
			}
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func buildOpenQuestionsPage(tasks []WorkspaceKnowledgeTask, sourceByID map[string]WorkspaceKnowledgeSource) string {
	var builder strings.Builder
	builder.WriteString("# Open Questions\n\n")
	if len(tasks) == 0 {
		builder.WriteString("None.\n")
		return builder.String()
	}

	for _, task := range tasks {
		builder.WriteString("- ")
		builder.WriteString(strings.TrimSpace(task.Title))
		if strings.TrimSpace(task.Priority) != "" {
			builder.WriteString(" [")
			builder.WriteString(strings.TrimSpace(task.Priority))
			builder.WriteString("]")
		}
		if strings.TrimSpace(task.Summary) != "" {
			builder.WriteString(": ")
			builder.WriteString(strings.TrimSpace(task.Summary))
		}

		sourceTitles := collectSourceTitles(task.SourceRefs, sourceByID)
		if len(sourceTitles) > 0 {
			builder.WriteString(" (Sources: ")
			builder.WriteString(strings.Join(sourceTitles, ", "))
			builder.WriteString(")")
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func buildDocumentWikiPage(payload WorkspaceKnowledgeBySourcePayload, conceptSlugs map[string]string) string {
	entities := append([]WorkspaceKnowledgeEntity(nil), payload.Entities...)
	sort.Slice(entities, func(i, j int) bool {
		return lessEntity(entities[i], entities[j])
	})

	claims := append([]WorkspaceKnowledgeClaim(nil), payload.Claims...)
	sort.Slice(claims, func(i, j int) bool {
		return lessClaim(claims[i], claims[j])
	})

	tasks := append([]WorkspaceKnowledgeTask(nil), payload.Tasks...)
	sort.Slice(tasks, func(i, j int) bool {
		return lessTask(tasks[i], tasks[j])
	})

	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(firstNonEmptyText(payload.Source.Title, payload.Source.Slug, payload.Source.ID))
	builder.WriteString("\n\n")
	builder.WriteString("- Source ID: `")
	builder.WriteString(payload.Source.ID)
	builder.WriteString("`\n")
	if strings.TrimSpace(payload.Source.Kind) != "" {
		builder.WriteString("- Kind: `")
		builder.WriteString(strings.TrimSpace(payload.Source.Kind))
		builder.WriteString("`\n")
	}

	builder.WriteString("\n## Concepts\n\n")
	if len(entities) == 0 {
		builder.WriteString("None.\n")
	} else {
		for _, entity := range entities {
			builder.WriteString("- [")
			builder.WriteString(firstNonEmptyText(entity.Title, entity.ID))
			builder.WriteString("](../concepts/")
			builder.WriteString(conceptSlugs[entity.ID])
			builder.WriteString(".md)")
			if strings.TrimSpace(entity.Summary) != "" {
				builder.WriteString(": ")
				builder.WriteString(strings.TrimSpace(entity.Summary))
			}
			builder.WriteString("\n")
		}
	}

	builder.WriteString("\n## Claims\n\n")
	if len(claims) == 0 {
		builder.WriteString("None.\n")
	} else {
		for _, claim := range claims {
			builder.WriteString("- ")
			builder.WriteString(strings.TrimSpace(claim.Title))
			if strings.TrimSpace(claim.Summary) != "" {
				builder.WriteString(": ")
				builder.WriteString(strings.TrimSpace(claim.Summary))
			}
			builder.WriteString("\n")
		}
	}

	builder.WriteString("\n## Open Questions\n\n")
	if len(tasks) == 0 {
		builder.WriteString("None.\n")
	} else {
		for _, task := range tasks {
			builder.WriteString("- ")
			builder.WriteString(strings.TrimSpace(task.Title))
			if strings.TrimSpace(task.Summary) != "" {
				builder.WriteString(": ")
				builder.WriteString(strings.TrimSpace(task.Summary))
			}
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func buildConceptWikiPage(page workspaceKnowledgeConceptPage) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(firstNonEmptyText(page.Entity.Title, page.Entity.ID))
	builder.WriteString("\n\n")
	if strings.TrimSpace(page.Entity.Type) != "" {
		builder.WriteString("- Type: `")
		builder.WriteString(strings.TrimSpace(page.Entity.Type))
		builder.WriteString("`\n")
	}
	if len(page.Entity.Aliases) > 0 {
		builder.WriteString("- Aliases: ")
		builder.WriteString(strings.Join(page.Entity.Aliases, ", "))
		builder.WriteString("\n")
	}
	if len(page.SourceTitles) > 0 {
		builder.WriteString("- Sources: ")
		builder.WriteString(strings.Join(page.SourceTitles, ", "))
		builder.WriteString("\n")
	}

	builder.WriteString("\n## Summary\n\n")
	if strings.TrimSpace(page.Entity.Summary) == "" {
		builder.WriteString("No summary available.\n")
	} else {
		builder.WriteString(strings.TrimSpace(page.Entity.Summary))
		builder.WriteString("\n")
	}

	builder.WriteString("\n## Related Claims\n\n")
	builder.WriteString(renderConceptClaims(page.Claims))
	return builder.String()
}

func buildConceptPages(snapshot WorkspaceKnowledgeSnapshot, conceptSlugs map[string]string, sourceByID map[string]WorkspaceKnowledgeSource) []workspaceKnowledgeConceptPage {
	claimsByEntityID := make(map[string][]WorkspaceKnowledgeClaim)
	for _, claim := range snapshot.Claims {
		for _, entityID := range claim.EntityIDs {
			claimsByEntityID[entityID] = append(claimsByEntityID[entityID], claim)
		}
	}

	pages := make([]workspaceKnowledgeConceptPage, 0, len(snapshot.Entities))
	for _, entity := range snapshot.Entities {
		claims := append([]WorkspaceKnowledgeClaim(nil), claimsByEntityID[entity.ID]...)
		sort.Slice(claims, func(i, j int) bool {
			return lessClaim(claims[i], claims[j])
		})
		pages = append(pages, workspaceKnowledgeConceptPage{
			Slug:         conceptSlugs[entity.ID],
			Entity:       entity,
			Claims:       claims,
			SourceTitles: collectSourceTitles(entity.SourceRefs, sourceByID),
		})
	}

	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Slug != pages[j].Slug {
			return pages[i].Slug < pages[j].Slug
		}
		return lessEntity(pages[i].Entity, pages[j].Entity)
	})
	return pages
}

func buildConceptSlugs(entities []WorkspaceKnowledgeEntity) map[string]string {
	slugs := make(map[string]string, len(entities))
	used := make(map[string]int, len(entities))
	for _, entity := range entities {
		base := workspaceKnowledgeSlug(firstNonEmptyText(entity.Title, entity.ID, "concept"))
		used[base]++
		if used[base] == 1 {
			slugs[entity.ID] = base
			continue
		}
		slugs[entity.ID] = fmt.Sprintf("%s-%d", base, used[base])
	}
	return slugs
}

func buildSourceByID(sources []WorkspaceKnowledgeSource) map[string]WorkspaceKnowledgeSource {
	sourceByID := make(map[string]WorkspaceKnowledgeSource, len(sources))
	for _, source := range sources {
		sourceByID[source.ID] = source
	}
	return sourceByID
}

func renderConceptClaims(claims []WorkspaceKnowledgeClaim) string {
	if len(claims) == 0 {
		return "None.\n"
	}

	var builder strings.Builder
	for _, claim := range claims {
		builder.WriteString("- ")
		builder.WriteString(strings.TrimSpace(claim.Title))
		if strings.TrimSpace(claim.Summary) != "" {
			builder.WriteString(": ")
			builder.WriteString(strings.TrimSpace(claim.Summary))
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func collectSourceTitles(refs []WorkspaceKnowledgeSourceRef, sourceByID map[string]WorkspaceKnowledgeSource) []string {
	if len(refs) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(refs))
	titles := make([]string, 0, len(refs))
	for _, ref := range refs {
		source, ok := sourceByID[ref.SourceID]
		if !ok {
			continue
		}
		title := firstNonEmptyText(source.Title, source.Slug, source.ID)
		if _, exists := seen[title]; exists {
			continue
		}
		seen[title] = struct{}{}
		titles = append(titles, title)
	}
	sort.Strings(titles)
	return titles
}

func clearWorkspaceKnowledgeMarkdownDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read workspace knowledge markdown directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepathExt(entry.Name()), ".md") {
			continue
		}
		if err := os.RemoveAll(dir + string(os.PathSeparator) + entry.Name()); err != nil {
			return fmt.Errorf("remove workspace knowledge markdown %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func filepathExt(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i:]
		}
		if name[i] == '/' || name[i] == '\\' {
			break
		}
	}
	return ""
}

func workspaceKnowledgeSlug(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "item"
	}

	var builder strings.Builder
	lastHyphen := false
	for _, r := range trimmed {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if r > unicode.MaxASCII {
				continue
			}
			builder.WriteRune(r)
			lastHyphen = false
		case !lastHyphen && builder.Len() > 0:
			builder.WriteByte('-')
			lastHyphen = true
		}
	}

	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "item"
	}
	return slug
}

func lessSource(left, right WorkspaceKnowledgeSource) bool {
	if left.Slug != right.Slug {
		return left.Slug < right.Slug
	}
	if left.Title != right.Title {
		return left.Title < right.Title
	}
	return left.ID < right.ID
}

func lessEntity(left, right WorkspaceKnowledgeEntity) bool {
	if left.ID != right.ID {
		return left.ID < right.ID
	}
	if left.Title != right.Title {
		return left.Title < right.Title
	}
	return left.UpdatedAt < right.UpdatedAt
}

func lessClaim(left, right WorkspaceKnowledgeClaim) bool {
	if left.ID != right.ID {
		return left.ID < right.ID
	}
	if left.Title != right.Title {
		return left.Title < right.Title
	}
	return left.UpdatedAt < right.UpdatedAt
}

func lessRelation(left, right WorkspaceKnowledgeRelation) bool {
	if left.ID != right.ID {
		return left.ID < right.ID
	}
	if left.Type != right.Type {
		return left.Type < right.Type
	}
	return left.UpdatedAt < right.UpdatedAt
}

func lessTask(left, right WorkspaceKnowledgeTask) bool {
	if left.ID != right.ID {
		return left.ID < right.ID
	}
	if left.Title != right.Title {
		return left.Title < right.Title
	}
	return left.UpdatedAt < right.UpdatedAt
}
