package main

type CollectionTree struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	LibraryID int              `json:"libraryId"`
	Library   string           `json:"library"`
	ParentID  string           `json:"parentId"`
	Path      string           `json:"path"`
	Children  []CollectionTree `json:"children"`
}

type ZoteroItem struct {
	ID              string   `json:"id"`
	Key             string   `json:"key"`
	CiteKey         string   `json:"citeKey"`
	Title           string   `json:"title"`
	Creators        string   `json:"creators"`
	Year            string   `json:"year"`
	ItemType        string   `json:"itemType"`
	LibraryID       int      `json:"libraryId"`
	CollectionIDs   []string `json:"collectionIds"`
	AttachmentCount int      `json:"attachmentCount"`
	HasPDF          bool     `json:"hasPdf"`
	PDFPath         string   `json:"pdfPath"`
	RawID           string   `json:"rawId"`
}
