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
