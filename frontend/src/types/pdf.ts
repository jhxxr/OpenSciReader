export interface ReaderOutlineItem {
  title: string;
  pageNumber: number | null;
  items: ReaderOutlineItem[];
}

export interface PDFDocumentPayload {
  path: string;
  dataBase64: string;
}
