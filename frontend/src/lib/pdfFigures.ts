import * as pdfjsLib from 'pdfjs-dist';
import type { TextContent, TextItem, TextStyle } from 'pdfjs-dist/types/src/display/api';
import { destroyPDFDocument, openPDFDocument } from './pdfDocument';

export interface ExtractedFigureCandidate {
  page: number;
  title: string;
  dataUrl: string;
  width: number;
  height: number;
}

interface FigureTextLine {
  text: string;
  left: number;
  top: number;
  right: number;
  bottom: number;
  width: number;
  height: number;
  centerX: number;
}

interface FigureCaptionBlock extends FigureTextLine {
  title: string;
  lineIndexes: number[];
}

interface FigureRect {
  left: number;
  top: number;
  right: number;
  bottom: number;
}

interface FigureRegionBounds {
  left: number;
  right: number;
  fullWidth: boolean;
}

interface FigureRegionScore {
  cropCanvas: HTMLCanvasElement;
  score: number;
}

interface TextMetrics {
  longLineCount: number;
  coverageRatio: number;
}

const FIGURE_CAPTION_START_RE = /^(?:extended\s+data\s+)?(?:figure|fig(?:ure)?\.?)\s*\d+[A-Za-z]?(?:\s*[:.)-]|\s+)/i;

export async function extractFigureCandidates(pdfPath: string, maxPages = 8): Promise<ExtractedFigureCandidate[]> {
  const { loadingTask, pdf } = await openPDFDocument(pdfPath);

  try {
    const results: ExtractedFigureCandidate[] = [];
    const seen = new Set<string>();
    const pageLimit = Math.min(pdf.numPages, maxPages);

    for (let pageNum = 1; pageNum <= pageLimit; pageNum += 1) {
      const page = await pdf.getPage(pageNum);
      const viewport = page.getViewport({ scale: 1.8 });
      const canvas = document.createElement('canvas');
      canvas.width = Math.ceil(viewport.width);
      canvas.height = Math.ceil(viewport.height);

      const context = canvas.getContext('2d');
      if (!context) {
        continue;
      }

      await page.render({ canvas, canvasContext: context, viewport }).promise;

      const textContent = await page.getTextContent();
      const lines = buildTextLines(textContent, viewport);
      const captions = findFigureCaptions(lines, canvas.width);

      for (const caption of captions) {
        const region = extractCaptionFigureRegion(canvas, lines, caption);
        if (!region) {
          continue;
        }

        const title = normalizeFigureTitle(caption.title);
        const key = `${pageNum}:${title}`;
        if (seen.has(key)) {
          continue;
        }

        seen.add(key);
        results.push({
          page: pageNum,
          title,
          dataUrl: region.cropCanvas.toDataURL('image/png'),
          width: region.cropCanvas.width,
          height: region.cropCanvas.height,
        });
      }
    }

    return results;
  } finally {
    await destroyPDFDocument(loadingTask);
  }
}

function buildTextLines(content: TextContent, viewport: pdfjsLib.PageViewport): FigureTextLine[] {
  const pageWidth = viewport.width;
  const rawItems = content.items
    .filter((item): item is TextItem => 'str' in item)
    .map((item) => buildTextLineItem(item, content.styles[item.fontName], viewport))
    .filter((item): item is FigureTextLine => item !== null)
    .sort((left, right) => {
      if (Math.abs(left.top - right.top) > 3) {
        return left.top - right.top;
      }
      return left.left - right.left;
    });

  const lines: FigureTextLine[] = [];

  for (const item of rawItems) {
    const itemMidY = item.top + (item.height / 2);
    let bestIndex = -1;
    let bestDistance = Number.POSITIVE_INFINITY;

    for (let index = lines.length - 1; index >= Math.max(0, lines.length - 8); index -= 1) {
      const line = lines[index];
      const lineMidY = line.top + (line.height / 2);
      const verticalTolerance = Math.max(4, Math.min(line.height, item.height) * 0.7);
      const horizontalGap = item.left - line.right;
      const likelySameColumn = horizontalGap <= Math.max(28, pageWidth * 0.09) || item.left <= line.right + 10;

      if (!likelySameColumn || Math.abs(lineMidY - itemMidY) > verticalTolerance) {
        continue;
      }

      const distance = Math.abs(lineMidY - itemMidY);
      if (distance < bestDistance) {
        bestDistance = distance;
        bestIndex = index;
      }
    }

    if (bestIndex === -1) {
      lines.push(item);
      continue;
    }

    const line = lines[bestIndex];
    const joiner = item.left - line.right > Math.max(4, item.height * 0.3) ? ' ' : '';
    const mergedText = `${line.text}${joiner}${item.text}`.trim();

    lines[bestIndex] = {
      text: normalizeWhitespace(mergedText),
      left: Math.min(line.left, item.left),
      top: Math.min(line.top, item.top),
      right: Math.max(line.right, item.right),
      bottom: Math.max(line.bottom, item.bottom),
      width: Math.max(line.right, item.right) - Math.min(line.left, item.left),
      height: Math.max(line.bottom, item.bottom) - Math.min(line.top, item.top),
      centerX: (Math.min(line.left, item.left) + Math.max(line.right, item.right)) / 2,
    };
  }

  return lines.map((line) => ({
    ...line,
    text: normalizeWhitespace(line.text),
    width: line.right - line.left,
    height: line.bottom - line.top,
    centerX: (line.left + line.right) / 2,
  }));
}

function buildTextLineItem(item: TextItem, style: TextStyle | undefined, viewport: pdfjsLib.PageViewport): FigureTextLine | null {
  const text = normalizeWhitespace(item.str);
  if (!text) {
    return null;
  }

  const transform = pdfjsLib.Util.transform(viewport.transform, item.transform);
  const fontHeight = Math.max(1, Math.hypot(transform[2], transform[3]), Math.abs(item.height));
  const fontAscent = fontHeight * (typeof style?.ascent === 'number' ? style.ascent : 0.85);
  const left = transform[4];
  const top = transform[5] - fontAscent;
  const width = Math.max(1, Math.abs(item.width));
  const height = Math.max(1, fontHeight);
  const right = left + width;
  const bottom = top + height;

  return {
    text,
    left,
    top,
    right,
    bottom,
    width,
    height,
    centerX: (left + right) / 2,
  };
}

function findFigureCaptions(lines: FigureTextLine[], pageWidth: number): FigureCaptionBlock[] {
  const captions: FigureCaptionBlock[] = [];

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    if (!FIGURE_CAPTION_START_RE.test(line.text)) {
      continue;
    }

    const captionLines = [line];
    const lineIndexes = [index];

    for (let nextIndex = index + 1; nextIndex < lines.length && captionLines.length < 3; nextIndex += 1) {
      const nextLine = lines[nextIndex];
      const lastLine = captionLines[captionLines.length - 1];
      const verticalGap = nextLine.top - lastLine.bottom;

      if (verticalGap > Math.max(18, lastLine.height * 1.2)) {
        break;
      }
      if (FIGURE_CAPTION_START_RE.test(nextLine.text)) {
        break;
      }
      if (!isSameCaptionColumn(line, nextLine, pageWidth)) {
        break;
      }

      captionLines.push(nextLine);
      lineIndexes.push(nextIndex);
    }

    const merged = mergeLines(captionLines);
    captions.push({
      ...merged,
      title: normalizeWhitespace(captionLines.map((entry) => entry.text).join(' ')),
      lineIndexes,
    });
    index = lineIndexes[lineIndexes.length - 1];
  }

  return captions;
}

function isSameCaptionColumn(anchor: FigureTextLine, candidate: FigureTextLine, pageWidth: number) {
  if (anchor.width > pageWidth * 0.62) {
    if (candidate.width < pageWidth * 0.5 && Math.abs(candidate.centerX - anchor.centerX) > pageWidth * 0.12) {
      return false;
    }
    return horizontalOverlap(anchor, candidate) > Math.min(anchor.width, candidate.width) * 0.2;
  }

  const anchorIsLeftColumn = anchor.centerX < pageWidth / 2;
  if (anchorIsLeftColumn) {
    return candidate.centerX < pageWidth * 0.62;
  }

  return candidate.centerX > pageWidth * 0.38;
}

function mergeLines(lines: FigureTextLine[]): FigureTextLine {
  const left = Math.min(...lines.map((line) => line.left));
  const top = Math.min(...lines.map((line) => line.top));
  const right = Math.max(...lines.map((line) => line.right));
  const bottom = Math.max(...lines.map((line) => line.bottom));

  return {
    text: normalizeWhitespace(lines.map((line) => line.text).join(' ')),
    left,
    top,
    right,
    bottom,
    width: right - left,
    height: bottom - top,
    centerX: (left + right) / 2,
  };
}

function extractCaptionFigureRegion(
  pageCanvas: HTMLCanvasElement,
  lines: FigureTextLine[],
  caption: FigureCaptionBlock,
): FigureRegionScore | null {
  const bounds = inferRegionBounds(caption, pageCanvas.width);
  const candidates = [
    {
      direction: 'above' as const,
      rect: buildCaptionSideRect('above', caption, bounds, lines, pageCanvas.width, pageCanvas.height),
    },
    {
      direction: 'below' as const,
      rect: buildCaptionSideRect('below', caption, bounds, lines, pageCanvas.width, pageCanvas.height),
    },
  ]
    .filter((entry): entry is { direction: 'above' | 'below'; rect: FigureRect } => entry.rect !== null)
    .map((entry) => scoreFigureRegion(pageCanvas, entry.rect, lines, entry.direction))
    .filter((entry): entry is FigureRegionScore => entry !== null)
    .sort((left, right) => right.score - left.score);

  return candidates[0] ?? null;
}

function inferRegionBounds(caption: FigureCaptionBlock, pageWidth: number): FigureRegionBounds {
  const pagePadding = pageWidth * 0.06;
  const gutter = pageWidth * 0.04;
  const fullWidth = caption.width > pageWidth * 0.62 || (caption.left < pageWidth * 0.18 && caption.right > pageWidth * 0.82);

  if (fullWidth) {
    return { left: pagePadding, right: pageWidth - pagePadding, fullWidth: true };
  }

  if (caption.centerX < pageWidth / 2) {
    return { left: pagePadding, right: (pageWidth / 2) - (gutter / 2), fullWidth: false };
  }

  return { left: (pageWidth / 2) + (gutter / 2), right: pageWidth - pagePadding, fullWidth: false };
}

function buildCaptionSideRect(
  direction: 'above' | 'below',
  caption: FigureCaptionBlock,
  bounds: FigureRegionBounds,
  lines: FigureTextLine[],
  pageWidth: number,
  pageHeight: number,
): FigureRect | null {
  const captionPadding = Math.max(10, caption.height * 0.6);
  const pagePadding = pageHeight * 0.05;
  const regionWidth = bounds.right - bounds.left;
  const sameRegionLines = lines
    .filter((line) => horizontalOverlap(line, bounds) > Math.min(line.width, bounds.right - bounds.left) * 0.15)
    .filter((line) => !caption.lineIndexes.some((index) => lines[index] === line));
  const paragraphLines = sameRegionLines.filter((line) => isParagraphLikeLine(line, sameRegionLines, regionWidth));

  let top: number;
  let bottom: number;

  if (direction === 'above') {
    const previousLine = [...paragraphLines].reverse().find((line) => line.bottom < caption.top - captionPadding);
    top = previousLine ? previousLine.bottom + (captionPadding * 0.7) : pagePadding;
    bottom = caption.top - captionPadding;
  } else {
    const nextLine = paragraphLines.find((line) => line.top > caption.bottom + captionPadding);
    top = caption.bottom + captionPadding;
    bottom = nextLine ? nextLine.top - (captionPadding * 0.7) : pageHeight - pagePadding;
  }

  const maxHeight = pageHeight * (bounds.fullWidth ? 0.55 : 0.46);
  if (bottom - top > maxHeight) {
    if (direction === 'above') {
      top = bottom - maxHeight;
    } else {
      bottom = top + maxHeight;
    }
  }

  const rect = clampRect({
    left: bounds.left,
    top,
    right: bounds.right,
    bottom,
  }, pageHeight, pageWidth);

  if (rect.bottom - rect.top < pageHeight * 0.08 || rect.right - rect.left < pageWidth * 0.18) {
    return null;
  }

  return rect;
}

function isParagraphLikeLine(
  line: FigureTextLine,
  regionLines: FigureTextLine[],
  regionWidth: number,
) {
  const isSubstantial = line.text.length >= 28 || line.width >= regionWidth * 0.34;
  if (!isSubstantial) {
    return false;
  }

  let neighborCount = 0;

  for (const candidate of regionLines) {
    if (candidate === line) {
      continue;
    }

    const isNearby = Math.abs((candidate.top + candidate.bottom) - (line.top + line.bottom)) / 2
      <= Math.max(26, Math.max(line.height, candidate.height) * 2.4);
    const overlapsEnough = horizontalOverlap(line, candidate) > Math.min(line.width, candidate.width) * 0.45;
    const candidateIsSubstantial = candidate.text.length >= 18 || candidate.width >= regionWidth * 0.25;

    if (!isNearby || !overlapsEnough || !candidateIsSubstantial) {
      continue;
    }

    neighborCount += 1;
    if (neighborCount >= 1) {
      return true;
    }
  }

  return false;
}

function clampRect(rect: FigureRect, pageHeight: number, pageWidth: number): FigureRect {
  const left = Math.max(0, Math.min(rect.left, pageWidth - 1));
  const right = Math.max(left + 1, Math.min(rect.right, pageWidth));
  const top = Math.max(0, Math.min(rect.top, pageHeight - 1));
  const bottom = Math.max(top + 1, Math.min(rect.bottom, pageHeight));

  return { left, top, right, bottom };
}

function scoreFigureRegion(
  sourceCanvas: HTMLCanvasElement,
  rect: FigureRect,
  lines: FigureTextLine[],
  direction: 'above' | 'below',
): FigureRegionScore | null {
  const regionCanvas = cropCanvas(sourceCanvas, rect);
  const trimmedCanvas = trimCanvasToContent(regionCanvas);
  if (trimmedCanvas.width < 120 || trimmedCanvas.height < 90) {
    return null;
  }

  const areaRatio = (trimmedCanvas.width * trimmedCanvas.height) / Math.max(1, sourceCanvas.width * sourceCanvas.height);
  const imageStats = collectImageStats(trimmedCanvas);
  const textMetrics = collectTextMetrics(lines, rect);

  if (areaRatio < 0.025 || areaRatio > 0.72) {
    return null;
  }
  if (imageStats.nonWhiteRatio < 0.018 || imageStats.nonWhiteRatio > 0.72) {
    return null;
  }

  let score = 0;
  score += Math.min(imageStats.nonWhiteRatio * 7, 2.5);
  score += Math.min(areaRatio * 5, 1.9);
  score += direction === 'above' ? 0.3 : 0.12;
  score += imageStats.colorfulRatio > 0.01 ? 0.2 : 0;
  score -= Math.max(0, textMetrics.coverageRatio - 0.24) * 5;
  score -= Math.max(0, textMetrics.longLineCount - 1) * 0.95;
  score -= Math.max(0, imageStats.darkRatio - 0.42) * 2;

  if (Math.max(trimmedCanvas.width / trimmedCanvas.height, trimmedCanvas.height / trimmedCanvas.width) > 4.2) {
    score -= 0.8;
  }
  if (score < 0.9) {
    return null;
  }

  return {
    cropCanvas: trimmedCanvas,
    score,
  };
}

function cropCanvas(sourceCanvas: HTMLCanvasElement, rect: FigureRect): HTMLCanvasElement {
  const left = Math.max(0, Math.floor(rect.left));
  const top = Math.max(0, Math.floor(rect.top));
  const right = Math.min(sourceCanvas.width, Math.ceil(rect.right));
  const bottom = Math.min(sourceCanvas.height, Math.ceil(rect.bottom));
  const width = Math.max(1, right - left);
  const height = Math.max(1, bottom - top);
  const canvas = document.createElement('canvas');
  canvas.width = width;
  canvas.height = height;

  const context = canvas.getContext('2d');
  if (!context) {
    return canvas;
  }

  context.drawImage(sourceCanvas, left, top, width, height, 0, 0, width, height);
  return canvas;
}

function trimCanvasToContent(canvas: HTMLCanvasElement): HTMLCanvasElement {
  const context = canvas.getContext('2d');
  if (!context) {
    return canvas;
  }

  const { data, width, height } = context.getImageData(0, 0, canvas.width, canvas.height);
  const sampleX = Math.max(1, Math.floor(width / 240));
  const sampleY = Math.max(1, Math.floor(height / 240));

  let top = 0;
  while (top < height && !hasRowContent(data, width, top, sampleX)) {
    top += sampleY;
  }

  let bottom = height - 1;
  while (bottom > top && !hasRowContent(data, width, bottom, sampleX)) {
    bottom -= sampleY;
  }

  let left = 0;
  while (left < width && !hasColumnContent(data, width, height, left, sampleY)) {
    left += sampleX;
  }

  let right = width - 1;
  while (right > left && !hasColumnContent(data, width, height, right, sampleY)) {
    right -= sampleX;
  }

  if (top >= bottom || left >= right) {
    return canvas;
  }

  const padding = 8;
  const finalLeft = Math.max(0, left - padding);
  const finalTop = Math.max(0, top - padding);
  const finalRight = Math.min(width, right + padding);
  const finalBottom = Math.min(height, bottom + padding);

  if (finalLeft === 0 && finalTop === 0 && finalRight === width && finalBottom === height) {
    return canvas;
  }

  const trimmed = document.createElement('canvas');
  trimmed.width = Math.max(1, finalRight - finalLeft);
  trimmed.height = Math.max(1, finalBottom - finalTop);

  const trimmedContext = trimmed.getContext('2d');
  if (!trimmedContext) {
    return canvas;
  }

  trimmedContext.drawImage(
    canvas,
    finalLeft,
    finalTop,
    trimmed.width,
    trimmed.height,
    0,
    0,
    trimmed.width,
    trimmed.height,
  );
  return trimmed;
}

function hasRowContent(data: Uint8ClampedArray, width: number, row: number, sampleX: number) {
  let active = 0;
  let total = 0;

  for (let column = 0; column < width; column += sampleX) {
    const index = ((row * width) + column) * 4;
    if (isInkPixel(data[index] ?? 255, data[index + 1] ?? 255, data[index + 2] ?? 255)) {
      active += 1;
    }
    total += 1;
  }

  return active / Math.max(1, total) > 0.012;
}

function hasColumnContent(data: Uint8ClampedArray, width: number, height: number, column: number, sampleY: number) {
  let active = 0;
  let total = 0;

  for (let row = 0; row < height; row += sampleY) {
    const index = ((row * width) + column) * 4;
    if (isInkPixel(data[index] ?? 255, data[index + 1] ?? 255, data[index + 2] ?? 255)) {
      active += 1;
    }
    total += 1;
  }

  return active / Math.max(1, total) > 0.012;
}

function collectImageStats(canvas: HTMLCanvasElement) {
  const context = canvas.getContext('2d');
  if (!context) {
    return { nonWhiteRatio: 0, darkRatio: 0, colorfulRatio: 0 };
  }

  const { data, width, height } = context.getImageData(0, 0, canvas.width, canvas.height);
  const sampleStep = Math.max(1, Math.floor((width * height) / 15000));
  let nonWhite = 0;
  let dark = 0;
  let colorful = 0;
  let total = 0;

  for (let index = 0; index < data.length; index += 4 * sampleStep) {
    const red = data[index] ?? 255;
    const green = data[index + 1] ?? 255;
    const blue = data[index + 2] ?? 255;
    const average = (red + green + blue) / 3;

    if (average < 246) {
      nonWhite += 1;
    }
    if (average < 112) {
      dark += 1;
    }
    if (Math.max(red, green, blue) - Math.min(red, green, blue) > 28 && average < 245) {
      colorful += 1;
    }
    total += 1;
  }

  return {
    nonWhiteRatio: nonWhite / Math.max(1, total),
    darkRatio: dark / Math.max(1, total),
    colorfulRatio: colorful / Math.max(1, total),
  };
}

function collectTextMetrics(lines: FigureTextLine[], rect: FigureRect): TextMetrics {
  let longLineCount = 0;
  let totalHeight = 0;

  for (const line of lines) {
    const verticalOverlap = Math.min(line.bottom, rect.bottom) - Math.max(line.top, rect.top);
    const horizontal = horizontalOverlap(line, rect);
    if (verticalOverlap <= 0 || horizontal <= 0 || verticalOverlap / line.height < 0.45) {
      continue;
    }

    totalHeight += Math.min(line.height, verticalOverlap);
    if (line.text.length >= 24 || line.width >= (rect.right - rect.left) * 0.55) {
      longLineCount += 1;
    }
  }

  return {
    longLineCount,
    coverageRatio: totalHeight / Math.max(1, rect.bottom - rect.top),
  };
}

function horizontalOverlap(
  subject: Pick<FigureTextLine, 'left' | 'right'> | FigureRegionBounds,
  target: Pick<FigureTextLine, 'left' | 'right'> | FigureRect | FigureRegionBounds,
) {
  return Math.max(0, Math.min(subject.right, target.right) - Math.max(subject.left, target.left));
}

function isInkPixel(red: number, green: number, blue: number) {
  return ((red + green + blue) / 3) < 246;
}

function normalizeFigureTitle(title: string) {
  const normalized = normalizeWhitespace(title);
  return normalized.length > 96 ? `${normalized.slice(0, 93)}...` : normalized;
}

function normalizeWhitespace(value: string) {
  return value.replace(/\s+/g, ' ').trim();
}
