Place the bundled PDF translation runtime here for Windows amd64 packaging.

Expected contents:
- A private Python runtime (`python/python.exe` or `python.exe`)
- Installed `pdf2zh_next` dependencies inside that runtime
- Any extra native assets required by `pdf2zh_next` / BabelDOC on Windows

The installer copies this directory to:
`$INSTDIR\runtime\pdf2zh-next\windows-amd64`

The app resolves the worker runtime in this order:
1. `<install-dir>\runtime\pdf2zh-next\windows-amd64`
2. `<project-root>\runtime\pdf2zh-next\windows-amd64`
3. fallback to system `python` / `py -3` for development only
