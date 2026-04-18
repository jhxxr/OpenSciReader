export function toReaderUrl(pdfPath: string): string {
  if (pdfPath.startsWith('http://') || pdfPath.startsWith('https://')) {
    return `/osr/remote-pdf?url=${encodeURIComponent(pdfPath)}`;
  }
  return `/osr/file?path=${encodeURIComponent(pdfPath)}`;
}
