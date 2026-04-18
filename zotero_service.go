package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	zoteroLocalAPIEndpoint = "http://127.0.0.1:23119/api"
	zoteroAPIVersion       = "3"
	zoteroPageSize         = 100
)

var yearPattern = regexp.MustCompile(`\d{4}`)

type zoteroService struct {
	endpoint string
	client   *http.Client
}

type zoteroScope struct {
	kind      string
	apiID     int
	libraryID int
	name      string
}

type zoteroAPIGroup struct {
	ID   int                `json:"id"`
	Data zoteroAPIGroupData `json:"data"`
}

type zoteroAPIGroupData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type zoteroAPILibrary struct {
	Type string `json:"type"`
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type zoteroAPILink struct {
	Href           string `json:"href"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	AttachmentType string `json:"attachmentType"`
	AttachmentSize int    `json:"attachmentSize"`
}

type zoteroAPICollection struct {
	Key     string                     `json:"key"`
	Library zoteroAPILibrary           `json:"library"`
	Data    zoteroAPICollectionData    `json:"data"`
	Links   map[string]zoteroAPILink   `json:"links"`
	Meta    map[string]json.RawMessage `json:"meta"`
}

type zoteroAPICollectionData struct {
	Key              string `json:"key"`
	Name             string `json:"name"`
	ParentCollection any    `json:"parentCollection"`
}

type zoteroAPICreator struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Name      string `json:"name"`
}

type zoteroAPIItem struct {
	Key     string             `json:"key"`
	Library zoteroAPILibrary   `json:"library"`
	Links   zoteroAPIItemLinks `json:"links"`
	Meta    zoteroAPIItemMeta  `json:"meta"`
	Data    zoteroAPIItemData  `json:"data"`
}

type zoteroAPIItemLinks struct {
	Self       zoteroAPILink `json:"self"`
	Attachment zoteroAPILink `json:"attachment"`
	Enclosure  zoteroAPILink `json:"enclosure"`
	Up         zoteroAPILink `json:"up"`
}

type zoteroAPIItemMeta struct {
	CreatorSummary string `json:"creatorSummary"`
	ParsedDate     string `json:"parsedDate"`
	NumChildren    int    `json:"numChildren"`
}

type zoteroAPIItemData struct {
	Key         string             `json:"key"`
	ItemType    string             `json:"itemType"`
	Title       string             `json:"title"`
	Date        string             `json:"date"`
	Creators    []zoteroAPICreator `json:"creators"`
	Collections []string           `json:"collections"`
	ParentItem  string             `json:"parentItem"`
	ContentType string             `json:"contentType"`
	LinkMode    string             `json:"linkMode"`
	Filename    string             `json:"filename"`
	URL         string             `json:"url"`
	Path        string             `json:"path"`
}

func newZoteroService() *zoteroService {
	return &zoteroService{
		endpoint: zoteroLocalAPIEndpoint,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (s *zoteroService) GetCollections(ctx context.Context, source string) ([]CollectionTree, error) {
	if source != "" && source != "http" && source != "bbt" {
		return nil, fmt.Errorf("unsupported zotero source: %s", source)
	}

	userScope := zoteroScope{kind: "user", apiID: 0}
	userCollections, err := s.fetchCollections(ctx, userScope)
	if err != nil {
		return nil, err
	}
	userScope = hydrateScope(userScope, userCollections)

	trees := []CollectionTree{
		buildCollectionTree(userScope, userCollections),
	}

	groups, err := s.fetchGroups(ctx)
	if err != nil {
		return trees, nil
	}

	groupTrees := make([]CollectionTree, 0, len(groups))
	for _, group := range groups {
		groupID := firstPositive(group.Data.ID, group.ID)
		scope := zoteroScope{
			kind:      "group",
			apiID:     groupID,
			libraryID: groupID,
			name:      firstNonEmpty(group.Data.Name, fmt.Sprintf("Group %d", groupID)),
		}

		collections, err := s.fetchCollections(ctx, scope)
		if err != nil {
			groupTrees = append(groupTrees, buildCollectionTree(scope, nil))
			continue
		}

		scope = hydrateScope(scope, collections)
		groupTrees = append(groupTrees, buildCollectionTree(scope, collections))
	}

	sort.Slice(groupTrees, func(i, j int) bool {
		return strings.ToLower(groupTrees[i].Name) < strings.ToLower(groupTrees[j].Name)
	})

	return append(trees, groupTrees...), nil
}

func (s *zoteroService) GetItemsByCollection(ctx context.Context, collectionID string) ([]ZoteroItem, error) {
	scope, rawCollectionKey, isLibraryRoot, err := parseScopedID(collectionID)
	if err != nil {
		return nil, err
	}

	var endpoint string
	if isLibraryRoot || rawCollectionKey == "" {
		endpoint = fmt.Sprintf("%s/items/top?format=json", scope.apiPath())
	} else {
		endpoint = fmt.Sprintf("%s/collections/%s/items/top?format=json", scope.apiPath(), url.PathEscape(rawCollectionKey))
	}

	items, err := s.fetchItems(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	results := make([]ZoteroItem, 0, len(items))
	for _, item := range items {
		itemKey := firstNonEmpty(item.Data.Key, item.Key)
		if itemKey == "" || isAttachmentLike(item.Data.ItemType) {
			continue
		}

		pdfPath := ""
		attachmentCount := 0
		hasPDF := strings.Contains(strings.ToLower(item.Links.Attachment.AttachmentType), "pdf")
		if item.Meta.NumChildren > 0 || hasPDF {
			pdfPath, attachmentCount = s.resolvePDFPathFromItem(ctx, scope, itemKey)
			if pdfPath != "" {
				hasPDF = true
			}
		}

		collectionIDs := make([]string, 0, len(item.Data.Collections))
		for _, rawCollectionID := range item.Data.Collections {
			if strings.TrimSpace(rawCollectionID) == "" {
				continue
			}
			collectionIDs = append(collectionIDs, scope.scopedKey(rawCollectionID))
		}

		results = append(results, ZoteroItem{
			ID:              scope.scopedKey(itemKey),
			Key:             itemKey,
			CiteKey:         itemKey,
			Title:           firstNonEmpty(item.Data.Title, "Untitled"),
			Creators:        joinCreators(item.Data.Creators),
			Year:            firstNonEmpty(item.Meta.ParsedDate, extractYear(item.Data.Date)),
			ItemType:        item.Data.ItemType,
			LibraryID:       firstPositive(item.Library.ID, scope.libraryID),
			CollectionIDs:   collectionIDs,
			AttachmentCount: attachmentCount,
			HasPDF:          hasPDF,
			PDFPath:         pdfPath,
			RawID:           firstNonEmpty(item.Links.Self.Href, itemKey),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return strings.ToLower(results[i].Title) < strings.ToLower(results[j].Title)
	})

	return results, nil
}

func (s *zoteroService) ResolvePDFPath(ctx context.Context, itemID string) (string, error) {
	scope, rawItemKey, _, err := parseScopedID(itemID)
	if err != nil {
		return "", err
	}

	path, _ := s.resolvePDFPathFromItem(ctx, scope, rawItemKey)
	if path == "" {
		return "", fmt.Errorf("no pdf attachment found for %s", itemID)
	}

	return path, nil
}

func (s *zoteroService) resolvePDFPathFromItem(ctx context.Context, scope zoteroScope, itemKey string) (string, int) {
	attachments, err := s.fetchItemAttachments(ctx, scope, itemKey)
	if err != nil {
		return "", 0
	}

	directAttachments := make([]zoteroAPIItem, 0, len(attachments))
	for _, attachment := range attachments {
		if attachment.Data.ParentItem == itemKey {
			directAttachments = append(directAttachments, attachment)
		}
	}

	if path := pickPDFPath(directAttachments); path != "" {
		return path, len(directAttachments)
	}
	if path := pickPDFPath(attachments); path != "" {
		if len(directAttachments) > 0 {
			return path, len(directAttachments)
		}
		return path, len(attachments)
	}

	if len(directAttachments) > 0 {
		return "", len(directAttachments)
	}
	return "", len(attachments)
}

func (s *zoteroService) fetchGroups(ctx context.Context) ([]zoteroAPIGroup, error) {
	return fetchPaginatedJSON[zoteroAPIGroup](ctx, s, "users/0/groups?format=json")
}

func (s *zoteroService) fetchCollections(ctx context.Context, scope zoteroScope) ([]zoteroAPICollection, error) {
	return fetchPaginatedJSON[zoteroAPICollection](ctx, s, fmt.Sprintf("%s/collections?format=json", scope.apiPath()))
}

func (s *zoteroService) fetchItems(ctx context.Context, endpoint string) ([]zoteroAPIItem, error) {
	return fetchPaginatedJSON[zoteroAPIItem](ctx, s, endpoint)
}

func (s *zoteroService) fetchItemAttachments(ctx context.Context, scope zoteroScope, itemKey string) ([]zoteroAPIItem, error) {
	return fetchPaginatedJSON[zoteroAPIItem](ctx, s, fmt.Sprintf("%s/items/%s/children?format=json&itemType=attachment", scope.apiPath(), url.PathEscape(itemKey)))
}

func fetchPaginatedJSON[T any](ctx context.Context, service *zoteroService, endpoint string) ([]T, error) {
	results := make([]T, 0)
	for start := 0; ; start += zoteroPageSize {
		path := appendQuery(endpoint, map[string]string{
			"limit": strconv.Itoa(zoteroPageSize),
			"start": strconv.Itoa(start),
		})

		batch := make([]T, 0)
		if _, err := service.getJSON(ctx, path, &batch); err != nil {
			return nil, err
		}

		results = append(results, batch...)
		if len(batch) < zoteroPageSize {
			return results, nil
		}
	}
}

func (s *zoteroService) getJSON(ctx context.Context, endpoint string, target any) (http.Header, error) {
	rawURL := endpoint
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = strings.TrimRight(s.endpoint, "/") + "/" + strings.TrimLeft(endpoint, "/")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create zotero request: %w", err)
	}

	req.Header.Set("Zotero-API-Version", zoteroAPIVersion)
	req.Header.Set("X-Zotero-Connector-API-Version", zoteroAPIVersion)
	req.Header.Set("User-Agent", "OpenSciReader/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call zotero local api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Header, fmt.Errorf("read zotero response: %w", err)
	}

	if resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return resp.Header, fmt.Errorf("zotero api request failed: %s", message)
	}

	if target != nil && len(body) > 0 {
		if err := json.Unmarshal(body, target); err != nil {
			return resp.Header, fmt.Errorf("decode zotero response: %w", err)
		}
	}

	return resp.Header, nil
}

func buildCollectionTree(scope zoteroScope, collections []zoteroAPICollection) CollectionTree {
	libraryName := firstNonEmpty(scope.name, defaultLibraryName(scope))
	libraryID := firstPositive(scope.libraryID, scope.apiID)

	root := CollectionTree{
		ID:        scope.rootID(),
		Name:      libraryName,
		LibraryID: libraryID,
		Library:   libraryName,
		Path:      "/" + libraryName,
		Children:  []CollectionTree{},
	}

	if len(collections) == 0 {
		return root
	}

	childrenByParent := make(map[string][]zoteroAPICollection)
	for _, collection := range collections {
		parentKey := parentCollectionKey(collection.Data.ParentCollection)
		childrenByParent[parentKey] = append(childrenByParent[parentKey], collection)
	}

	root.Children = buildCollectionChildren(scope, childrenByParent, "", root.ID, root.Path)
	return root
}

func buildCollectionChildren(scope zoteroScope, childrenByParent map[string][]zoteroAPICollection, parentCollectionKeyValue, parentID, parentPath string) []CollectionTree {
	rawChildren := childrenByParent[parentCollectionKeyValue]
	children := make([]CollectionTree, 0, len(rawChildren))

	for _, collection := range rawChildren {
		rawKey := firstNonEmpty(collection.Data.Key, collection.Key)
		name := firstNonEmpty(collection.Data.Name, rawKey)
		path := parentPath + "/" + name

		node := CollectionTree{
			ID:        scope.scopedKey(rawKey),
			Name:      name,
			LibraryID: firstPositive(scope.libraryID, collection.Library.ID, scope.apiID),
			Library:   firstNonEmpty(scope.name, collection.Library.Name, defaultLibraryName(scope)),
			ParentID:  parentID,
			Path:      path,
		}
		node.Children = buildCollectionChildren(scope, childrenByParent, rawKey, node.ID, path)
		children = append(children, node)
	}

	sort.Slice(children, func(i, j int) bool {
		return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
	})

	return children
}

func hydrateScope(scope zoteroScope, collections []zoteroAPICollection) zoteroScope {
	if len(collections) == 0 {
		scope.name = firstNonEmpty(scope.name, defaultLibraryName(scope))
		scope.libraryID = firstPositive(scope.libraryID, scope.apiID)
		return scope
	}

	scope.name = firstNonEmpty(collections[0].Library.Name, scope.name, defaultLibraryName(scope))
	scope.libraryID = firstPositive(scope.libraryID, collections[0].Library.ID, scope.apiID)
	return scope
}

func parseScopedID(id string) (zoteroScope, string, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return zoteroScope{}, "", false, fmt.Errorf("id is required")
	}

	isLibraryRoot := false
	prefix := ""
	rawKey := ""

	switch {
	case strings.HasPrefix(id, "library/"):
		isLibraryRoot = true
		prefix = strings.TrimPrefix(id, "library/")
	case strings.Contains(id, "/"):
		parts := strings.SplitN(id, "/", 2)
		prefix = parts[0]
		rawKey = parts[1]
	default:
		return zoteroScope{kind: "user", apiID: 0, libraryID: 0, name: "My Library"}, id, false, nil
	}

	scope, err := parseScopePrefix(prefix)
	if err != nil {
		return zoteroScope{}, "", false, err
	}

	return scope, rawKey, isLibraryRoot, nil
}

func parseScopePrefix(prefix string) (zoteroScope, error) {
	switch {
	case prefix == "user":
		return zoteroScope{kind: "user", apiID: 0, name: "My Library"}, nil
	case strings.HasPrefix(prefix, "group:"):
		groupID, err := strconv.Atoi(strings.TrimPrefix(prefix, "group:"))
		if err != nil {
			return zoteroScope{}, fmt.Errorf("invalid group scope: %s", prefix)
		}
		return zoteroScope{
			kind:      "group",
			apiID:     groupID,
			libraryID: groupID,
			name:      fmt.Sprintf("Group %d", groupID),
		}, nil
	default:
		return zoteroScope{}, fmt.Errorf("invalid scoped id: %s", prefix)
	}
}

func pickPDFPath(attachments []zoteroAPIItem) string {
	for _, attachment := range attachments {
		if !looksLikePDF(attachment) {
			continue
		}

		candidates := []string{
			attachment.Links.Enclosure.Href,
			attachment.Data.Path,
			attachment.Data.URL,
		}

		for _, candidate := range candidates {
			if normalized := normalizeAttachmentPath(candidate); normalized != "" {
				return normalized
			}
		}
	}

	return ""
}

func looksLikePDF(item zoteroAPIItem) bool {
	contentType := strings.ToLower(firstNonEmpty(
		item.Links.Enclosure.Type,
		item.Data.ContentType,
		item.Links.Attachment.AttachmentType,
	))
	title := strings.ToLower(firstNonEmpty(item.Data.Title, item.Data.Filename, item.Links.Enclosure.Title))
	location := strings.ToLower(firstNonEmpty(item.Links.Enclosure.Href, item.Data.Path, item.Data.URL))

	return strings.Contains(contentType, "pdf") ||
		strings.HasSuffix(location, ".pdf") ||
		strings.Contains(title, "pdf")
}

func normalizeAttachmentPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
		return raw
	case strings.HasPrefix(lower, "file://"):
		return decodeFileURL(raw)
	case strings.HasPrefix(lower, "storage:"):
		return ""
	default:
		if unescaped, err := url.PathUnescape(raw); err == nil {
			raw = unescaped
		}
		return filepath.Clean(raw)
	}
}

func decodeFileURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return filepath.Clean(strings.TrimPrefix(raw, "file://"))
	}

	path := parsed.Path
	if parsed.Opaque != "" && path == "" {
		path = parsed.Opaque
	}

	if unescaped, err := url.PathUnescape(path); err == nil {
		path = unescaped
	}

	if parsed.Host != "" && !strings.EqualFold(parsed.Host, "localhost") {
		path = "//" + parsed.Host + path
	}

	if len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}

	return filepath.Clean(filepath.FromSlash(path))
}

func appendQuery(rawURL string, query map[string]string) string {
	separator := "?"
	if strings.Contains(rawURL, "?") {
		separator = "&"
	}

	parts := make([]string, 0, len(query))
	for key, value := range query {
		parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(value))
	}
	sort.Strings(parts)

	return rawURL + separator + strings.Join(parts, "&")
}

func parentCollectionKey(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		return ""
	default:
		return ""
	}
}

func joinCreators(creators []zoteroAPICreator) string {
	parts := make([]string, 0, len(creators))
	for _, creator := range creators {
		name := firstNonEmpty(
			strings.TrimSpace(creator.Name),
			strings.TrimSpace(strings.TrimSpace(creator.LastName)+" "+strings.TrimSpace(creator.FirstName)),
		)
		if name != "" {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, ", ")
}

func extractYear(input string) string {
	if match := yearPattern.FindString(input); match != "" {
		return match
	}
	return ""
}

func defaultLibraryName(scope zoteroScope) string {
	if scope.kind == "group" {
		return fmt.Sprintf("Group %d", scope.apiID)
	}
	return "My Library"
}

func parsePositiveInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func isAttachmentLike(itemType string) bool {
	return strings.EqualFold(strings.TrimSpace(itemType), "attachment") ||
		strings.EqualFold(strings.TrimSpace(itemType), "note")
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (s zoteroScope) apiPath() string {
	if s.kind == "group" {
		return fmt.Sprintf("groups/%d", s.apiID)
	}
	return "users/0"
}

func (s zoteroScope) prefix() string {
	if s.kind == "group" {
		return fmt.Sprintf("group:%d", s.apiID)
	}
	return "user"
}

func (s zoteroScope) rootID() string {
	return "library/" + s.prefix()
}

func (s zoteroScope) scopedKey(rawKey string) string {
	return s.prefix() + "/" + rawKey
}
