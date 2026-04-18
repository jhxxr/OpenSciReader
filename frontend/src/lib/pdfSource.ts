import { pdfApi } from '../api/pdf';
import { toReaderUrl } from './pdfUrl';

const pdfBytesCache = new Map<string, Promise<Uint8Array>>();

export async function loadPDFBytes(pdfPath: string): Promise<Uint8Array> {
  const cached = pdfBytesCache.get(pdfPath);
  if (cached) {
    return (await cached).slice();
  }

  const pending = (async () => {
    try {
      const payload = await pdfApi.loadPDFDocument(pdfPath);
      return decodeBase64ToUint8Array(payload.dataBase64);
    } catch {
      const response = await fetch(toReaderUrl(pdfPath));
      if (!response.ok) {
        throw new Error(`Failed to load PDF: ${response.status} ${response.statusText}`);
      }
      const bytes = new Uint8Array(await response.arrayBuffer());
      ensurePDFBytes(bytes);
      return bytes;
    }
  })();

  pdfBytesCache.set(pdfPath, pending);

  try {
    return (await pending).slice();
  } catch (error) {
    pdfBytesCache.delete(pdfPath);
    throw error;
  }
}

function decodeBase64ToUint8Array(base64: string): Uint8Array {
  const binary = window.atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  ensurePDFBytes(bytes);
  return bytes;
}

function ensurePDFBytes(bytes: Uint8Array) {
  const signature = String.fromCharCode(...Array.from(bytes.slice(0, 5)));
  if (signature !== '%PDF-') {
    throw new Error('Loaded content is not a valid PDF file');
  }
}
