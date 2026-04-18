# OpenSciReader

OpenSciReader is a Wails desktop reader for local academic PDF workflows.

This branch includes a local PDF layout-translation pipeline built around:
- Go job orchestration
- a private Python worker
- `pdf2zh_next.high_level.do_translate_async_stream`
- React page-by-page preview replacement

## What is implemented

- Local PDF layout translation preview in the reader
- Dual-column reading mode
  - left: original PDF
  - right: translated pages replaced chunk by chunk
- Preview translation in 25-page chunks
- Whole-book export mode that reruns translation and outputs mono + dual PDFs
- Cancel support
- Retry support by starting a new preview job with `retryJobId`
- WebSocket event streaming for progress updates
- Persisted job metadata and output file paths under the user config directory

## Runtime packaging

The PDF layout-translation feature still uses the existing Go orchestration + private Python worker contract, but the heavy pdf2zh/BabelDOC runtime is no longer shipped inside the main installer.

The Windows installer now includes only:
- `python_worker/`
- optional `runtime/webview2/windows-amd64/`

Users import a standalone runtime package after installing the app.

Expected imported runtime layout inside the zip:

```text
pdf2zh/
  manifest.json
  runtime/
    python.exe
    pythonXY._pth
  site-packages/
  offline_assets_*.zip
  ...runtime files...
python_worker/
  worker.py
```

At runtime, the app resolves translation assets in this order:
1. installed runtime from user config storage
2. development fallback worker under `<install-dir>/python_worker/worker.py`
3. development fallback runtime under repo / executable `runtime/pdf2zh-next/...`

The app stores imported runtimes under the user config directory and validates the runtime before allowing translation jobs to start.

## Start in development

Prerequisites:
- Go
- Node.js / npm
- Wails CLI

Run:

```bash
wails dev
```

Build frontend only:

```bash
cd frontend
npm run build
```

## Package for Windows

1. Ensure `python_worker/worker.py` is present
2. Build the slim installer:

```bash
wails build --target windows/amd64 --nsis
```

The installer script now copies:
- `python_worker`
- optional `runtime/webview2/windows-amd64`

The heavy pdf2zh runtime is published as a separate release asset and is no longer embedded into the installer.

## GitHub Actions release packaging

The repository includes a Windows release workflow at:
- `.github/workflows/release-windows.yml`

The workflow now produces two release assets:
- slim installer: `OpenSciReader-<version>-windows-amd64-installer.exe`
- standalone runtime: `OpenSciReader-pdf-runtime-windows-amd64-<version>.zip`

The runtime package is assembled in CI from the pinned stack:
- embeddable CPython archive
- `pdf2zh-next` / `BabelDOC` / `PyMuPDF`
- `onnxruntime-directml`
- generated `offline_assets_*.zip`
- generated `manifest.json`

The workflow will:
- assemble and prune the runtime
- validate the worker import path against the packaged runtime
- zip the runtime as a separate release asset
- build the NSIS installer without embedding the heavy runtime
- optionally bundle the offline WebView2 installer
- publish both assets to the GitHub Release in the same run

## Reader workflow

1. Install OpenSciReader
2. Open settings and import `OpenSciReader-pdf-runtime-windows-amd64-<version>.zip`
3. Open a PDF in the reader
4. Go to the `翻译` tab
5. Select the existing LLM provider + model used for layout translation
6. Click `开始保留格式翻译预览`
7. After the first 25-page chunk finishes, the right column replaces those pages with translated output
8. Optionally start `开始导出 mono + dual`
9. Download the translated-only PDF or the dual PDF from the export section

## Backend API

These endpoints are served by the app-local HTTP handler:

- `POST /api/pdf-translate/start`
- `POST /api/pdf-translate/{jobId}/cancel`
- `GET /api/pdf-translate/{jobId}/status`
- `WS /api/pdf-translate/{jobId}/events`

### `POST /api/pdf-translate/start`

Request:

```json
{
  "pdfPath": "E:/papers/demo.pdf",
  "pageCount": 86,
  "itemId": "zotero-item-id",
  "itemTitle": "Demo Paper",
  "sourceLang": "en",
  "targetLang": "zh-CN",
  "mode": "preview",
  "previewChunkPages": 25,
  "maxPagesPerPart": 120,
  "retryJobId": "",
  "llmProviderId": 1,
  "llmModelId": 2
}
```

Response: `PDFTranslateJobSnapshot`

### `GET /api/pdf-translate/{jobId}/status`

Response shape:

```json
{
  "jobId": "uuid",
  "mode": "preview",
  "status": "running",
  "pdfPath": "E:/papers/demo.pdf",
  "localPdfPath": "E:/papers/demo.pdf",
  "pageCount": 86,
  "sourceLang": "en",
  "targetLang": "zh-CN",
  "providerId": 1,
  "providerName": "OpenAI Compatible",
  "modelId": "gpt-4.1",
  "currentStage": "translate",
  "overallProgress": 0.42,
  "outputs": {
    "monoPdfPath": "",
    "dualPdfPath": ""
  },
  "chunks": [
    {
      "index": 1,
      "startPage": 1,
      "endPage": 25,
      "status": "completed",
      "translatedPdfPath": "C:/Users/.../preview-chunk-001/output.no_watermark.pdf"
    }
  ]
}
```

## Event protocol

WebSocket messages are JSON objects shaped like `PDFTranslateEvent`.

Common event types:
- `status_snapshot`
- `job_started`
- `chunk_started`
- `stage_summary`
- `progress_start`
- `progress_update`
- `progress_end`
- `chunk_finished`
- `finish`
- `cancelled`
- `error`

Example progress event:

```json
{
  "sequence": 7,
  "jobId": "uuid",
  "mode": "preview",
  "type": "progress_update",
  "timestamp": "2026-04-17T10:00:00Z",
  "jobStatus": "running",
  "stage": "translate",
  "stageProgress": 0.56,
  "overallProgress": 0.31,
  "stageCurrent": 14,
  "stageTotal": 25,
  "chunk": {
    "index": 1,
    "startPage": 1,
    "endPage": 25,
    "status": "running"
  }
}
```

Example finish event:

```json
{
  "sequence": 19,
  "jobId": "uuid",
  "mode": "export",
  "type": "finish",
  "timestamp": "2026-04-17T10:03:21Z",
  "jobStatus": "completed",
  "output": {
    "monoPdfPath": "C:/Users/.../mono.pdf",
    "dualPdfPath": "C:/Users/.../dual.pdf",
    "totalSeconds": 201.6
  }
}
```

## Python worker contract

Worker file:
- [python_worker/worker.py](/E:/0JHX/Project/OpenSciReader/python_worker/worker.py)

Input:
- reads one JSON request from `stdin`

Output:
- writes one JSON event per line to `stdout`

Behavior:
- imports `pdf2zh_next.high_level.do_translate_async_stream`
- builds settings dynamically for compatibility with different upstream layouts
- forwards `stage_summary / progress_* / finish / error`
- normalizes `finish.translate_result` into plain JSON

Normalized output fields:
- `original_pdf_path`
- `mono_pdf_path`
- `dual_pdf_path`
- `no_watermark_mono_pdf_path`
- `no_watermark_dual_pdf_path`
- `auto_extracted_glossary_path`
- `total_seconds`
- `peak_memory_usage`

## Directory structure

```text
OpenSciReader/
  frontend/
    src/
      api/
        pdfTranslate.ts
      components/
        DualPdfReader.tsx
        ReaderTab.tsx
      types/
        pdfTranslate.ts
  internal/
    translator/
      manager.go
      storage.go
      types.go
      worker.go
  python_worker/
    worker.py
  runtime/
    pdf2zh-next/
      windows-amd64/
        README.md
  build/windows/installer/
    project.nsi
  pdf_translate_http.go
  pdf_translate_runtime.go
```

## Notes

- Preview mode uses:
  - `pages`
  - `only_include_translated_page=true`
  - `no_dual=true`
  - `no_mono=false`
- Export mode reruns the whole document and returns mono + dual output paths
- The current implementation persists job metadata to:
  - `%AppData%/.openscireader/reader_translate/jobs/<jobId>/job.json`
