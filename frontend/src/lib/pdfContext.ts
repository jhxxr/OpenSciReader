import { pdfApi } from "../api/pdf";
import type { PDFMarkdownPayload, PDFMarkdownSection } from "../types/pdf";
import { extractPDFPageTexts } from "./pdfText";

export interface PDFTextContext {
  text: string;
  totalPages: number;
  includedPages: number[];
  totalCharacters: number;
  truncated: boolean;
  source: "markitdown" | "page_text";
  sourceLabel: string;
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
const markdownCache = new Map<string, Promise<PDFMarkdownPayload>>();

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

async function loadPDFMarkdown(pdfPath: string) {
  const cached = markdownCache.get(pdfPath);
  if (cached) {
    return cached;
  }

  const pending = pdfApi.extractPDFMarkdown(pdfPath).catch((error) => {
    markdownCache.delete(pdfPath);
    throw error;
  });
  markdownCache.set(pdfPath, pending);
  return pending;
}

export async function loadPDFTextContext(
  pdfPath: string,
  maxChars: number,
): Promise<PDFTextContext> {
  try {
    const markdown = await loadPDFMarkdown(pdfPath);
    return buildMarkdownContext(markdown, maxChars);
  } catch {
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
      source: "page_text",
      sourceLabel: "pdf.js 逐页文本",
    };
  }
}

export async function loadPDFTextChunks(
  pdfPath: string,
  chunkPages: number,
  maxChars: number,
): Promise<PDFTextChunk[]> {
  try {
    const markdown = await loadPDFMarkdown(pdfPath);
    const markdownChunks = buildMarkdownChunks(markdown.sections, maxChars);
    if (markdownChunks.length > 0) {
      return markdownChunks;
    }
  } catch {
    // Fall back to page text extraction below.
  }

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

function buildMarkdownContext(
  markdown: PDFMarkdownPayload,
  maxChars: number,
): PDFTextContext {
  const safeLimit = Math.max(4000, maxChars);
  const sections = markdown.sections.filter((section) => section.text.trim() !== "");
  const includedPages: number[] = [];
  let totalCharacters = 0;
  let text = "";
  let includedSections = 0;

  for (const section of sections) {
    totalCharacters += section.text.length;
    const nextText = text ? `${text}\n\n${section.text.trim()}` : section.text.trim();
    if (nextText.length > safeLimit && text !== "") {
      break;
    }
    text = nextText;
    includedSections += 1;
  }

  return {
    text: text || markdown.markdown.trim(),
    totalPages: sections.length,
    includedPages,
    totalCharacters: markdown.totalChars,
    truncated: includedSections < sections.length,
    source: "markitdown",
    sourceLabel: markdown.cached ? "MarkItDown Markdown（缓存）" : "MarkItDown Markdown",
  };
}

function buildMarkdownChunks(
  sections: PDFMarkdownSection[],
  maxChars: number,
): PDFTextChunk[] {
  const usableSections = sections.filter((section) => section.text.trim() !== "");
  const safeMaxChars = Math.max(4000, maxChars);
  const chunks: PDFTextChunk[] = [];
  let index = 0;
  let startPage = 0;
  let endPage = 0;
  let text = "";

  const flush = () => {
    if (!text.trim()) {
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

  for (const section of usableSections) {
    const segment = section.text.trim();
    const nextText = text ? `${text}\n\n${segment}` : segment;
    if (nextText.length > safeMaxChars && text !== "") {
      flush();
    }

    if (!text && segment.length > safeMaxChars) {
      let offset = 0;
      while (offset < segment.length) {
        const chunkText = segment.slice(offset, offset + safeMaxChars).trim();
        if (!chunkText) {
          break;
        }
        index += 1;
        chunks.push({
          index,
          startPage: section.startPage,
          endPage: section.endPage,
          text: chunkText,
          characters: chunkText.length,
        });
        offset += safeMaxChars;
      }
      continue;
    }

    if (startPage === 0) {
      startPage = section.startPage;
    }
    endPage = section.endPage;
    text = text ? `${text}\n\n${segment}` : segment;
  }

  flush();
  return chunks;
}
