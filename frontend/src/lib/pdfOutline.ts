import * as pdfjsLib from 'pdfjs-dist';
import type { ReaderOutlineItem } from '../types/pdf';

interface PdfOutlineNode {
  title: string;
  dest?: string | unknown[] | null;
  items?: PdfOutlineNode[];
}

interface TextRowPart {
  height: number;
  text: string;
  width: number;
  x: number;
}

interface ExtractedTextLine {
  height: number;
  text: string;
  x: number;
  y: number;
}

const outlineCache = new Map<string, Promise<ReaderOutlineItem[]>>();

export async function loadReaderOutline(cacheKey: string, pdf: pdfjsLib.PDFDocumentProxy): Promise<ReaderOutlineItem[]> {
  const cached = outlineCache.get(cacheKey);
  if (cached) {
    return cached;
  }

  const pending = buildReaderOutline(pdf);
  outlineCache.set(cacheKey, pending);

  try {
    return await pending;
  } catch (error) {
    outlineCache.delete(cacheKey);
    throw error;
  }
}

async function buildReaderOutline(pdf: pdfjsLib.PDFDocumentProxy): Promise<ReaderOutlineItem[]> {
  const embeddedOutline = await resolveEmbeddedOutline(pdf, await pdf.getOutline() ?? []);
  if (embeddedOutline.length > 0) {
    return embeddedOutline;
  }

  return inferOutlineFromText(pdf);
}

async function resolveEmbeddedOutline(pdf: pdfjsLib.PDFDocumentProxy, items: PdfOutlineNode[]): Promise<ReaderOutlineItem[]> {
  const result: ReaderOutlineItem[] = [];

  for (const item of items) {
    result.push({
      title: normalizeExtractedLine(item.title) || 'Untitled Section',
      pageNumber: await resolveDestinationPageNumber(pdf, item.dest),
      items: await resolveEmbeddedOutline(pdf, item.items ?? []),
    });
  }

  return result;
}

async function resolveDestinationPageNumber(
  pdf: pdfjsLib.PDFDocumentProxy,
  dest: string | unknown[] | null | undefined,
): Promise<number | null> {
  if (!dest) {
    return null;
  }

  try {
    if (typeof dest === 'string') {
      const resolved = await pdf.getDestination(dest);
      return await referenceToPageNumber(pdf, resolved?.[0]);
    }

    if (Array.isArray(dest)) {
      return await referenceToPageNumber(pdf, dest[0]);
    }
  } catch {
    return null;
  }

  return null;
}

async function referenceToPageNumber(pdf: pdfjsLib.PDFDocumentProxy, reference: unknown): Promise<number | null> {
  if (typeof reference === 'number' && Number.isFinite(reference)) {
    return reference + 1;
  }

  if (isPDFReference(reference)) {
    return (await pdf.getPageIndex(reference)) + 1;
  }

  return null;
}

function isPDFReference(reference: unknown): reference is { gen: number; num: number } {
  return typeof reference === 'object'
    && reference !== null
    && typeof Reflect.get(reference, 'num') === 'number'
    && typeof Reflect.get(reference, 'gen') === 'number';
}

async function inferOutlineFromText(pdf: pdfjsLib.PDFDocumentProxy): Promise<ReaderOutlineItem[]> {
  const result: ReaderOutlineItem[] = [];
  const seen = new Set<string>();
  let currentSection: ReaderOutlineItem | null = null;

  for (let pageNumber = 1; pageNumber <= pdf.numPages; pageNumber += 1) {
    const page = await pdf.getPage(pageNumber);
    const lines = await extractPageTextLines(page);

    for (const line of lines) {
      const topLevelTitle = parseTopLevelHeading(line.text);
      if (topLevelTitle) {
        const cacheToken = `top:${topLevelTitle.toLowerCase()}`;
        if (seen.has(cacheToken)) {
          currentSection = result.find((item) => item.title.toLowerCase() === topLevelTitle.toLowerCase()) ?? currentSection;
          continue;
        }

        currentSection = { title: topLevelTitle, pageNumber, items: [] };
        result.push(currentSection);
        seen.add(cacheToken);
        continue;
      }

      const subSectionTitle = parseSubSectionHeading(line.text);
      if (!subSectionTitle || !currentSection) {
        continue;
      }

      const cacheToken = `sub:${currentSection.title.toLowerCase()}:${subSectionTitle.toLowerCase()}`;
      if (seen.has(cacheToken)) {
        continue;
      }

      currentSection.items.push({ title: subSectionTitle, pageNumber, items: [] });
      seen.add(cacheToken);
    }
  }

  return result;
}

async function extractPageTextLines(page: pdfjsLib.PDFPageProxy): Promise<ExtractedTextLine[]> {
  const content = await page.getTextContent();
  const rows: Array<{ items: TextRowPart[]; maxHeight: number; y: number }> = [];

  for (const item of content.items) {
    if (!('str' in item)) {
      continue;
    }

    const text = normalizeTextFragment(item.str);
    if (!text) {
      continue;
    }

    const x = Number(item.transform?.[4] ?? 0);
    const y = Number(item.transform?.[5] ?? 0);
    const height = Math.max(Number(item.height ?? 0), Math.abs(Number(item.transform?.[3] ?? 0)), 1);
    const width = Math.max(Number(item.width ?? 0), 1);
    const part = { height, text, width, x };
    const row = rows.find((entry) => Math.abs(entry.y - y) <= Math.max(2, height * 0.4));

    if (row) {
      row.items.push(part);
      row.maxHeight = Math.max(row.maxHeight, height);
    } else {
      rows.push({ y, maxHeight: height, items: [part] });
    }
  }

  rows.sort((left, right) => right.y - left.y);

  const extractedLines = rows.flatMap((row) => {
    row.items.sort((left, right) => left.x - right.x);

    const segments: TextRowPart[][] = [];
    let currentSegment: TextRowPart[] = [];
    let previousEnd: number | null = null;

    for (const item of row.items) {
      const gap = previousEnd === null ? 0 : item.x - previousEnd;
      if (currentSegment.length > 0 && gap > Math.max(30, row.maxHeight * 2.2)) {
        segments.push(currentSegment);
        currentSegment = [];
      }
      currentSegment.push(item);
      previousEnd = item.x + item.width;
    }

    if (currentSegment.length > 0) {
      segments.push(currentSegment);
    }

    return segments
      .map((segment) => ({
        height: row.maxHeight,
        text: normalizeExtractedLine(segment.map((item) => item.text).join(' ')),
        x: segment[0]?.x ?? 0,
        y: row.y,
      }))
      .filter((line) => line.text.length > 0);
  });

  return orderExtractedLines(extractedLines, page.getViewport({ scale: 1 }).width);
}

function parseTopLevelHeading(input: string): string | null {
  const line = normalizeExtractedLine(input);
  if (!line) {
    return null;
  }

  if (/^abstract\b/i.test(line)) {
    return 'Abstract';
  }
  if (/^index terms\b/i.test(line)) {
    return 'Index Terms';
  }
  if (/^references\b/i.test(line)) {
    return 'References';
  }

  if (line.length > 80 || /[=:]|https?:\/\//i.test(line) || /\[[0-9]+\]/.test(line)) {
    return null;
  }

  const match = line.match(/^((?:[IVX]+|[1-9]\d*))\.\s+(.+)$/);
  if (!match) {
    return null;
  }

  const [, sectionIndex, sectionTitle] = match;
  const title = sectionTitle.trim();
  if (!looksLikeTopLevelHeading(title)) {
    return null;
  }

  return `${sectionIndex}. ${title}`;
}

function parseSubSectionHeading(input: string): string | null {
  const line = normalizeExtractedLine(input);
  if (!line || line.length > 96 || /[=:]|https?:\/\//i.test(line) || /\[[0-9]+\]/.test(line)) {
    return null;
  }

  const match = line.match(/^([A-Z])\.\s+(.+)$/);
  if (!match) {
    return null;
  }

  const [, sectionIndex, sectionTitle] = match;
  const title = sectionTitle.trim();
  if (!looksLikeSubSectionHeading(title)) {
    return null;
  }

  return `${sectionIndex}. ${title}`;
}

function looksLikeTopLevelHeading(input: string): boolean {
  if (input.length < 4 || input.length > 40 || /[.,;:]$/.test(input)) {
    return false;
  }

  const words = input.split(/\s+/);
  if (words.length > 6) {
    return false;
  }

  const letters = input.replace(/[^A-Za-z]/g, '');
  if (letters.length < 4) {
    return false;
  }

  const uppercaseCount = letters.split('').filter((char) => char === char.toUpperCase()).length;
  return uppercaseCount / letters.length >= 0.7;
}

function looksLikeSubSectionHeading(input: string): boolean {
  if (input.length < 3 || input.length > 72 || /[.,;:]$/.test(input)) {
    return false;
  }
  if (!/^[A-Za-z][A-Za-z0-9\s()/-]+$/.test(input)) {
    return false;
  }

  const words = input.split(/\s+/);
  if (words.length > 8) {
    return false;
  }

  const lowercase = input.toLowerCase();
  if (lowercase.includes('fig.') || lowercase.includes('table ') || lowercase.includes('http')) {
    return false;
  }

  return true;
}

function normalizeTextFragment(input: string): string {
  return input.replace(/\u0000/g, '').replace(/\s+/g, ' ').trim();
}

function normalizeExtractedLine(input: string): string {
  let value = input
    .replace(/\u0000/g, '')
    .replace(/\s+([,.;:!?])/g, '$1')
    .replace(/\s+/g, ' ')
    .trim();

  for (let index = 0; index < 4; index += 1) {
    const next = value.replace(/\b([A-Z])\s+([A-Z][A-Z]{2,})\b/g, '$1$2');
    if (next === value) {
      break;
    }
    value = next;
  }

  return value;
}

function orderExtractedLines(lines: ExtractedTextLine[], pageWidth: number): ExtractedTextLine[] {
  const leftColumn = lines.filter((line) => line.x <= pageWidth * 0.45);
  const rightColumn = lines.filter((line) => line.x >= pageWidth * 0.55);
  const centerColumn = lines.filter((line) => line.x > pageWidth * 0.45 && line.x < pageWidth * 0.55);

  if (leftColumn.length >= 6 && rightColumn.length >= 6) {
    return [
      ...sortLinesInReadingOrder(leftColumn),
      ...sortLinesInReadingOrder(rightColumn),
      ...sortLinesInReadingOrder(centerColumn),
    ];
  }

  return sortLinesInReadingOrder(lines);
}

function sortLinesInReadingOrder(lines: ExtractedTextLine[]): ExtractedTextLine[] {
  return [...lines].sort((left, right) => {
    if (Math.abs(right.y - left.y) <= 2) {
      return left.x - right.x;
    }
    return right.y - left.y;
  });
}
