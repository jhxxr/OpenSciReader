package main

type PDFDocumentPayload struct {
	Path       string `json:"path"`
	DataBase64 string `json:"dataBase64"`
}
