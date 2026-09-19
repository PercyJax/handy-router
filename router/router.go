package router

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type Route struct {
	Name  string
	Query string
}

// IsCancel returns true if "cancel" appears three or more times in a row
// (accounting for punctuation the transcription model might add).
// e.g. "cancel cancel cancel", "Cancel, Cancel, Cancel", "I said cancel cancel cancel!"
func IsCancel(text string) bool {
	words := strings.Fields(strings.ToLower(text))
	cancelRun := 0
	for _, w := range words {
		w = strings.TrimFunc(w, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if w == "cancel" {
			cancelRun++
			if cancelRun >= 3 {
				return true
			}
		} else {
			cancelRun = 0
		}
	}
	return false
}

func ParseRoute(transcription string, prefixes map[string]string) Route {
	text := strings.TrimSpace(transcription)
	lower := strings.ToLower(text)

	for prefix, name := range prefixes {
		lowerPrefix := strings.ToLower(prefix)
		if !strings.HasPrefix(lower, lowerPrefix) {
			continue
		}

		rest := text[len(prefix):]

		// Require a word boundary: the character after the prefix must not be
		// a letter, digit, or word-internal punctuation, so "chat" does not
		// match "chatbots", "gemini" does not match "gemini's".
		if rest != "" {
			r, _ := utf8.DecodeRuneInString(rest)
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || r == '\u2019' || r == '-' {
				continue
			}
		}

		// Strip punctuation and whitespace after the prefix
		// (e.g. "Gemini, what" -> "what").
		rest = strings.TrimLeftFunc(rest, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		rest = strings.TrimSpace(rest)
		if rest != "" {
			return Route{Name: name, Query: rest}
		}

		// Prefix with no query after it (e.g. just "gemini")
		return Route{Name: "default", Query: text}
	}

	return Route{Name: "default", Query: text}
}
