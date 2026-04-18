export interface ReaderOutlineItem {
  title: string;
  pageNumber: number | null;
  items: ReaderOutlineItem[];
}

export interface PDFDocumentPayload {
  path: string;
  dataBase64: string;
}

export interface PDFMarkdownSection {
  index: number;
  title: string;
  level: number;
  startPage: number;
  endPage: number;
  text: string;
  characters: number;
}

export interface PDFMarkdownPayload {
  pdfPath: string;
  source: string;
  markdown: string;
  sections: PDFMarkdownSection[];
  totalChars: number;
  cached: boolean;
  generatedAt: string;
}
