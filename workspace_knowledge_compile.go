package main

import (
	"fmt"
	"os"
	"path/filepath"
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

type workspaceKnowledgeBySourceRecord struct {
	CanonicalSlug string
	Payload       WorkspaceKnowledgeBySourcePayload
}

type workspaceKnowledgeOutputFile struct {
	Path    string
	Content string
}

type workspaceKnowledgeWikiWritePlan struct {
	Overview      workspaceKnowledgeOutputFile
	OpenQuestions workspaceKnowledgeOutputFile
	Documents     []workspaceKnowledgeOutputFile
	Concepts      []workspaceKnowledgeOutputFile
}

func CompileWorkspaceKnowledge(files workspaceKnowledgeFiles, workspaceTitle string) (WorkspaceKnowledgeSnapshot, error) {
	if err := files.EnsureLayout(); err != nil {
		return WorkspaceKnowledgeSnapshot{}, err
	}

	records, err := readWorkspaceKnowledgeBySourceRecords(files)
	if err != nil {
		return WorkspaceKnowledgeSnapshot{}, err
	}

	snapshot := WorkspaceKnowledgeSnapshot{
		Sources:   mapSources(records),
		Entities:  mapEntities(records),
		Claims:    mapClaims(records),
		Relations: mapRelations(records),
		Tasks:     mapTasks(records),
	}

	if err := writeWorkspaceKnowledgeAggregates(files, snapshot); err != nil {
		return WorkspaceKnowledgeSnapshot{}, err
	}
	if err := writeWorkspaceKnowledgeWiki(files, strings.TrimSpace(workspaceTitle), snapshot, records); err != nil {
		return WorkspaceKnowledgeSnapshot{}, err
	}

	return snapshot, nil
}

func readWorkspaceKnowledgeBySourceRecords(files workspaceKnowledgeFiles) ([]workspaceKnowledgeBySourceRecord, error) {
	paths, err := files.BySourcePaths()
	if err != nil {
		return nil, err
	}

	records := make([]workspaceKnowledgeBySourceRecord, 0, len(paths))
	for _, path := range paths {
		var payload WorkspaceKnowledgeBySourcePayload
		if err := readWorkspaceKnowledgeJSON(path, &payload); err != nil {
			return nil, err
		}
		records = append(records, workspaceKnowledgeBySourceRecord{
			CanonicalSlug: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
			Payload:       payload,
		})
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].CanonicalSlug != records[j].CanonicalSlug {
			return records[i].CanonicalSlug < records[j].CanonicalSlug
		}
		return lessSource(records[i].Payload.Source, records[j].Payload.Source)
	})
	if err := validateUniqueWorkspaceKnowledgeSourceIDs(records); err != nil {
		return nil, err
	}
	return records, nil
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

func writeWorkspaceKnowledgeWiki(files workspaceKnowledgeFiles, workspaceTitle string, snapshot WorkspaceKnowledgeSnapshot, records []workspaceKnowledgeBySourceRecord) error {
	docsDir, err := files.docsDir()
	if err != nil {
		return err
	}

	conceptsDir, err := files.conceptsDir()
	if err != nil {
		return err
	}

	conceptSlugs := buildConceptSlugs(snapshot.Entities)
	sourceByID := buildSourceByID(snapshot.Sources)
	sourceDocSlugs := buildSourceDocSlugs(records)
	conceptPages := buildConceptPages(snapshot, conceptSlugs, sourceByID)
	plan, err := buildWorkspaceKnowledgeWikiWritePlan(files, workspaceTitle, snapshot, records, conceptPages, conceptSlugs, sourceByID, sourceDocSlugs)
	if err != nil {
		return err
	}
	if err := validateWorkspaceKnowledgeWikiWritePlan(plan); err != nil {
		return err
	}
	if err := clearWorkspaceKnowledgeMarkdownDir(docsDir); err != nil {
		return err
	}
	if err := clearWorkspaceKnowledgeMarkdownDir(conceptsDir); err != nil {
		return err
	}
	return writeWorkspaceKnowledgeWikiWritePlan(plan)
}

func buildWorkspaceKnowledgeWikiWritePlan(files workspaceKnowledgeFiles, workspaceTitle string, snapshot WorkspaceKnowledgeSnapshot, records []workspaceKnowledgeBySourceRecord, conceptPages []workspaceKnowledgeConceptPage, conceptSlugs map[string]string, sourceByID map[string]WorkspaceKnowledgeSource, sourceDocSlugs map[string]string) (workspaceKnowledgeWikiWritePlan, error) {
	overviewPath, err := files.OverviewPath()
	if err != nil {
		return workspaceKnowledgeWikiWritePlan{}, err
	}

	openQuestionsPath, err := files.OpenQuestionsPath()
	if err != nil {
		return workspaceKnowledgeWikiWritePlan{}, err
	}

	plan := workspaceKnowledgeWikiWritePlan{
		Overview: workspaceKnowledgeOutputFile{
			Path:    overviewPath,
			Content: buildOverviewWikiPage(workspaceTitle, snapshot, conceptSlugs, sourceDocSlugs),
		},
		OpenQuestions: workspaceKnowledgeOutputFile{
			Path:    openQuestionsPath,
			Content: buildOpenQuestionsPage(snapshot.Tasks, sourceByID),
		},
		Documents: make([]workspaceKnowledgeOutputFile, 0, len(records)),
		Concepts:  make([]workspaceKnowledgeOutputFile, 0, len(conceptPages)),
	}

	for _, record := range records {
		documentPath, err := files.DocumentWikiPath(record.CanonicalSlug)
		if err != nil {
			return workspaceKnowledgeWikiWritePlan{}, err
		}
		plan.Documents = append(plan.Documents, workspaceKnowledgeOutputFile{
			Path:    documentPath,
			Content: buildDocumentWikiPage(record.Payload, conceptSlugs),
		})
	}

	for _, page := range conceptPages {
		conceptPath, err := files.ConceptWikiPath(page.Slug)
		if err != nil {
			return workspaceKnowledgeWikiWritePlan{}, err
		}
		plan.Concepts = append(plan.Concepts, workspaceKnowledgeOutputFile{
			Path:    conceptPath,
			Content: buildConceptWikiPage(page),
		})
	}

	return plan, nil
}

func validateWorkspaceKnowledgeWikiWritePlan(plan workspaceKnowledgeWikiWritePlan) error {
	seen := map[string]struct{}{}
	files := []workspaceKnowledgeOutputFile{plan.Overview, plan.OpenQuestions}
	files = append(files, plan.Documents...)
	files = append(files, plan.Concepts...)

	for _, file := range files {
		if _, exists := seen[file.Path]; exists {
			return fmt.Errorf("duplicate workspace knowledge wiki output path %s", file.Path)
		}
		seen[file.Path] = struct{}{}

		info, err := os.Stat(file.Path)
		if err == nil {
			if info.IsDir() {
				return fmt.Errorf("workspace knowledge wiki output path %s is a directory", file.Path)
			}
			continue
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat workspace knowledge wiki output path %s: %w", file.Path, err)
		}
	}

	return nil
}

func writeWorkspaceKnowledgeWikiWritePlan(plan workspaceKnowledgeWikiWritePlan) error {
	files := []workspaceKnowledgeOutputFile{plan.Overview, plan.OpenQuestions}
	files = append(files, plan.Documents...)
	files = append(files, plan.Concepts...)
	for _, file := range files {
		if err := writeWorkspaceKnowledgeMarkdown(file.Path, file.Content); err != nil {
			return err
		}
	}
	return nil
}

func mapSources(records []workspaceKnowledgeBySourceRecord) []WorkspaceKnowledgeSource {
	sources := make([]WorkspaceKnowledgeSource, 0, len(records))
	for _, record := range records {
		sources = append(sources, record.Payload.Source)
	}
	sort.Slice(sources, func(i, j int) bool {
		return lessSource(sources[i], sources[j])
	})
	return sources
}

func mapEntities(records []workspaceKnowledgeBySourceRecord) []WorkspaceKnowledgeEntity {
	entities := make([]WorkspaceKnowledgeEntity, 0)
	for _, record := range records {
		for _, entity := range record.Payload.Entities {
			entities = append(entities, normalizeWorkspaceKnowledgeEntity(entity))
		}
	}
	sort.Slice(entities, func(i, j int) bool {
		return lessEntity(entities[i], entities[j])
	})
	return entities
}

func mapClaims(records []workspaceKnowledgeBySourceRecord) []WorkspaceKnowledgeClaim {
	claims := make([]WorkspaceKnowledgeClaim, 0)
	for _, record := range records {
		for _, claim := range record.Payload.Claims {
			claims = append(claims, normalizeWorkspaceKnowledgeClaim(claim))
		}
	}
	sort.Slice(claims, func(i, j int) bool {
		return lessClaim(claims[i], claims[j])
	})
	return claims
}

func mapRelations(records []workspaceKnowledgeBySourceRecord) []WorkspaceKnowledgeRelation {
	relations := make([]WorkspaceKnowledgeRelation, 0)
	for _, record := range records {
		for _, relation := range record.Payload.Relations {
			relations = append(relations, normalizeWorkspaceKnowledgeRelation(relation))
		}
	}
	sort.Slice(relations, func(i, j int) bool {
		return lessRelation(relations[i], relations[j])
	})
	return relations
}

func mapTasks(records []workspaceKnowledgeBySourceRecord) []WorkspaceKnowledgeTask {
	tasks := make([]WorkspaceKnowledgeTask, 0)
	for _, record := range records {
		for _, task := range record.Payload.Tasks {
			tasks = append(tasks, normalizeWorkspaceKnowledgeTask(task))
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		return lessTask(tasks[i], tasks[j])
	})
	return tasks
}

func buildOverviewWikiPage(workspaceTitle string, snapshot WorkspaceKnowledgeSnapshot, conceptSlugs map[string]string, sourceDocSlugs map[string]string) string {
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
			builder.WriteString(firstNonEmptyText(sourceDocSlugs[source.ID], workspaceKnowledgeSourceWikiSlug(source)))
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

		sourceLabels := collectSourceLabels(task.SourceRefs, sourceByID)
		if len(sourceLabels) > 0 {
			builder.WriteString(" (Sources: ")
			builder.WriteString(strings.Join(sourceLabels, ", "))
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
			SourceTitles: collectSourceLabels(entity.SourceRefs, sourceByID),
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

func buildSourceDocSlugs(records []workspaceKnowledgeBySourceRecord) map[string]string {
	sourceDocSlugs := make(map[string]string, len(records))
	for _, record := range records {
		sourceDocSlugs[record.Payload.Source.ID] = record.CanonicalSlug
	}
	return sourceDocSlugs
}

func validateUniqueWorkspaceKnowledgeSourceIDs(records []workspaceKnowledgeBySourceRecord) error {
	seen := make(map[string]string, len(records))
	for _, record := range records {
		sourceID := record.Payload.Source.ID
		if firstSlug, exists := seen[sourceID]; exists {
			return fmt.Errorf("duplicate workspace knowledge source id %q in by-source files %q and %q", sourceID, firstSlug, record.CanonicalSlug)
		}
		seen[sourceID] = record.CanonicalSlug
	}
	return nil
}

func normalizeWorkspaceKnowledgeEntity(entity WorkspaceKnowledgeEntity) WorkspaceKnowledgeEntity {
	entity.Aliases = append([]string(nil), entity.Aliases...)
	sort.Strings(entity.Aliases)
	entity.SourceRefs = normalizeWorkspaceKnowledgeSourceRefs(entity.SourceRefs)
	return entity
}

func normalizeWorkspaceKnowledgeClaim(claim WorkspaceKnowledgeClaim) WorkspaceKnowledgeClaim {
	claim.EntityIDs = append([]string(nil), claim.EntityIDs...)
	sort.Strings(claim.EntityIDs)
	claim.SourceRefs = normalizeWorkspaceKnowledgeSourceRefs(claim.SourceRefs)
	return claim
}

func normalizeWorkspaceKnowledgeRelation(relation WorkspaceKnowledgeRelation) WorkspaceKnowledgeRelation {
	relation.SourceRefs = normalizeWorkspaceKnowledgeSourceRefs(relation.SourceRefs)
	return relation
}

func normalizeWorkspaceKnowledgeTask(task WorkspaceKnowledgeTask) WorkspaceKnowledgeTask {
	task.SourceRefs = normalizeWorkspaceKnowledgeSourceRefs(task.SourceRefs)
	return task
}

func normalizeWorkspaceKnowledgeSourceRefs(sourceRefs []WorkspaceKnowledgeSourceRef) []WorkspaceKnowledgeSourceRef {
	normalized := append([]WorkspaceKnowledgeSourceRef(nil), sourceRefs...)
	sort.Slice(normalized, func(i, j int) bool {
		return lessWorkspaceKnowledgeSourceRef(normalized[i], normalized[j])
	})
	return normalized
}

func lessWorkspaceKnowledgeSourceRef(left, right WorkspaceKnowledgeSourceRef) bool {
	if left.SourceID != right.SourceID {
		return left.SourceID < right.SourceID
	}
	if left.PageStart != right.PageStart {
		return left.PageStart < right.PageStart
	}
	if left.PageEnd != right.PageEnd {
		return left.PageEnd < right.PageEnd
	}
	return left.Excerpt < right.Excerpt
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

func collectSourceLabels(refs []WorkspaceKnowledgeSourceRef, sourceByID map[string]WorkspaceKnowledgeSource) []string {
	if len(refs) == 0 {
		return []string{}
	}

	type sourceLabel struct {
		SourceID string
		Title    string
		Label    string
	}

	seen := make(map[string]struct{}, len(refs))
	labels := make([]sourceLabel, 0, len(refs))
	for _, ref := range refs {
		source, ok := sourceByID[ref.SourceID]
		if !ok {
			continue
		}
		if _, exists := seen[source.ID]; exists {
			continue
		}
		seen[source.ID] = struct{}{}

		title := firstNonEmptyText(source.Title, source.Slug, source.ID)
		label := fmt.Sprintf("%s (`%s`)", title, source.ID)
		if title == source.ID {
			label = fmt.Sprintf("`%s`", source.ID)
		}
		labels = append(labels, sourceLabel{
			SourceID: source.ID,
			Title:    title,
			Label:    label,
		})
	}
	sort.Slice(labels, func(i, j int) bool {
		if labels[i].Title != labels[j].Title {
			return labels[i].Title < labels[j].Title
		}
		return labels[i].SourceID < labels[j].SourceID
	})

	result := make([]string, 0, len(labels))
	for _, label := range labels {
		result = append(result, label.Label)
	}
	return result
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

func workspaceKnowledgeSourceWikiSlug(source WorkspaceKnowledgeSource) string {
	if trimmedSlug := strings.TrimSpace(source.Slug); trimmedSlug != "" {
		return trimmedSlug
	}
	if slug := workspaceKnowledgeSlug(source.ID); slug != "" {
		return slug
	}
	return workspaceKnowledgeSlug(firstNonEmptyText(source.Title, "source"))
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
