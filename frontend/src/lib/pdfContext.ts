import { extractPDFPageTexts } from "./pdfText";

export interface PDFTextContext {
  text: string;
  totalPages: number;
  includedPages: number[];
  totalCharacters: number;
  truncated: boolean;
}

export interface PDFTextChunk {
  index: number;
  startPage: number;
  endPage: number;
  text: string;
  characters: number;
}

type PDFPageTextsPromise = ReturnType<typeof extractPDFPageTexts>;

const pageTextCache = new Map<string, PDFPageTextsPromise>();

async function loadAllPageTexts(pdfPath: string) {
  const cached = pageTextCache.get(pdfPath);
  if (cached) {
    return cached;
  }

  const pending = extractPDFPageTexts(pdfPath).catch((error) => {
    pageTextCache.delete(pdfPath);
    throw error;
  });

  pageTextCache.set(pdfPath, pending);
  return pending;
}

export async function loadPDFTextContext(
  pdfPath: string,
  maxChars: number,
): Promise<PDFTextContext> {
  const pages = (await loadAllPageTexts(pdfPath)).filter(
    (page) => page.text.trim() !== "",
  );
  const safeLimit = Math.max(4000, maxChars);
  const includedPages: number[] = [];
  let totalCharacters = 0;
  let text = "";

  for (const page of pages) {
    totalCharacters += page.text.length;
    const segment = `[Page ${page.page}]\n${page.text.trim()}`;
    const nextText = text ? `${text}\n\n${segment}` : segment;
    if (nextText.length > safeLimit && text !== "") {
      break;
    }
    text = nextText;
    includedPages.push(page.page);
  }

  return {
    text,
    totalPages: pages.length,
    includedPages,
    totalCharacters,
    truncated: includedPages.length < pages.length,
  };
}

export async function loadPDFTextChunks(
  pdfPath: string,
  chunkPages: number,
  maxChars: number,
): Promise<PDFTextChunk[]> {
  const pages = (await loadAllPageTexts(pdfPath)).filter(
    (page) => page.text.trim() !== "",
  );
  const safeChunkPages = Math.max(1, chunkPages);
  const safeMaxChars = Math.max(4000, maxChars);
  const chunks: PDFTextChunk[] = [];

  let index = 0;
  let startPage = 0;
  let endPage = 0;
  let text = "";

  const flush = () => {
    if (!text.trim() || startPage === 0 || endPage === 0) {
      return;
    }

    index += 1;
    chunks.push({
      index,
      startPage,
      endPage,
      text: text.trim(),
      characters: text.length,
    });

    startPage = 0;
    endPage = 0;
    text = "";
  };

  for (const page of pages) {
    const segment = `[Page ${page.page}]\n${page.text.trim()}`;
    const nextText = text ? `${text}\n\n${segment}` : segment;
    const currentPageSpan = startPage === 0 ? 1 : page.page - startPage + 1;
    const exceedsPages = startPage !== 0 && currentPageSpan > safeChunkPages;
    const exceedsChars = nextText.length > safeMaxChars && text !== "";

    if (exceedsPages || exceedsChars) {
      flush();
    }

    if (startPage === 0) {
      startPage = page.page;
    }
    endPage = page.page;

    if (!text && segment.length > safeMaxChars) {
      text = `${segment.slice(0, safeMaxChars)}\n\n[Truncated chunk due to size limit]`;
      flush();
      continue;
    }

    text = text ? `${text}\n\n${segment}` : segment;
  }

  flush();
  return chunks;
}
