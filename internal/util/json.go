package util

import (
	"strings"
)

// ExtractJSON attempts to find the valid JSON object or array within a string
// that might contain other log output. It finds the first '{' or '['
// and the last '}' or ']'.
func ExtractJSON(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return ""
	}

	firstBrace := strings.Index(s, "{")
	firstBracket := strings.Index(s, "[")

	start := -1
	if firstBrace != -1 && firstBracket != -1 {
		if firstBrace < firstBracket {
			start = firstBrace
		} else {
			start = firstBracket
		}
	} else if firstBrace != -1 {
		start = firstBrace
	} else if firstBracket != -1 {
		start = firstBracket
	}

	if start == -1 {
		return s // Return original if no JSON start found
	}

	lastBrace := strings.LastIndex(s, "}")
	lastBracket := strings.LastIndex(s, "]")

	end := -1
	if lastBrace != -1 && lastBracket != -1 {
		if lastBrace > lastBracket {
			end = lastBrace
		} else {
			end = lastBracket
		}
	} else if lastBrace != -1 {
		end = lastBrace
	} else if lastBracket != -1 {
		end = lastBracket
	}

	if end == -1 || end < start {
		return s
	}

	return s[start : end+1]
}
