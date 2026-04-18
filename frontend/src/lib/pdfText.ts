import { destroyPDFDocument, openPDFDocument } from './pdfDocument';

export interface ExtractedPageText {
  page: number;
  text: string;
}

export async function extractPDFPageTexts(pdfPath: string): Promise<ExtractedPageText[]> {
  const { loadingTask, pdf } = await openPDFDocument(pdfPath);

  try {
    const pages: ExtractedPageText[] = [];

    for (let pageNum = 1; pageNum <= pdf.numPages; pageNum += 1) {
      const page = await pdf.getPage(pageNum);
      const content = await page.getTextContent();
      const text = content.items
        .map((item) => (('str' in item ? item.str : '') as string))
        .join(' ')
        .replace(/\s{2,}/g, ' ')
        .trim();
      pages.push({ page: pageNum, text });
    }

    return pages;
  } finally {
    await destroyPDFDocument(loadingTask);
  }
}
