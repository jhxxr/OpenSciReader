package main

type PDFDocumentPayload struct {
	Path       string `json:"path"`
	DataBase64 string `json:"dataBase64"`
}

type PDFMarkdownSection struct {
	Index      int    `json:"index"`
	Title      string `json:"title"`
	Level      int    `json:"level"`
	StartPage  int    `json:"startPage"`
	EndPage    int    `json:"endPage"`
	Text       string `json:"text"`
	Characters int    `json:"characters"`
}

type PDFMarkdownPayload struct {
	PDFPath     string               `json:"pdfPath"`
	Source      string               `json:"source"`
	Markdown    string               `json:"markdown"`
	Sections    []PDFMarkdownSection `json:"sections"`
	TotalChars  int                  `json:"totalChars"`
	Cached      bool                 `json:"cached"`
	GeneratedAt string               `json:"generatedAt"`
}
