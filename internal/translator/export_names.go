package translator

import (
	"path/filepath"
	"strings"
	"unicode"
)

const defaultExportOutputStem = "translated-document"

func exportOutputFileStem(request StartRequest) string {
	base := strings.TrimSpace(request.ItemTitle)
	if base == "" {
		base = strings.TrimSpace(filepath.Base(request.PDFPath))
		base = strings.TrimSuffix(base, filepath.Ext(base))
	}
	base = sanitizeExportOutputStem(base)
	if base == "" {
		base = defaultExportOutputStem
	}

	sourceLang := sanitizeExportOutputStem(strings.TrimSpace(request.SourceLang))
	targetLang := sanitizeExportOutputStem(strings.TrimSpace(request.TargetLang))
	if sourceLang == "" || targetLang == "" {
		return base
	}
	return base + "." + sourceLang + "-to-" + targetLang
}

func sanitizeExportOutputStem(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(value))
	lastWasSeparator := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			builder.WriteRune(r)
			lastWasSeparator = false
		case r == '.', r == '-', r == '_':
			builder.WriteRune(r)
			lastWasSeparator = false
		case unicode.IsSpace(r):
			if !lastWasSeparator {
				builder.WriteRune('-')
				lastWasSeparator = true
			}
		case strings.ContainsRune(`<>:"/\|?*`, r), unicode.IsControl(r):
			if !lastWasSeparator {
				builder.WriteRune('-')
				lastWasSeparator = true
			}
		default:
			if !lastWasSeparator {
				builder.WriteRune('-')
				lastWasSeparator = true
			}
		}
	}

	sanitized := strings.Trim(builder.String(), ".-_ ")
	sanitized = strings.ReplaceAll(sanitized, "--", "-")
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}
	return sanitized
}
