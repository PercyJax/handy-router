package router

import (
	"testing"
)

func TestParseRoute(t *testing.T) {
	prefixes := map[string]string{"transcribe": "enhance", "gemini": "gemini", "chat": "code"}

	tests := []struct {
		input     string
		wantRoute string
		wantQuery string
	}{
		{"transcribe hello world", "enhance", "hello world"},
		{"Transcribe, hello world", "enhance", "hello world"},
		{"gemini what is rust", "gemini", "what is rust"},
		{"Gemini, what is rust", "gemini", "what is rust"},
		{"gemini: what is rust", "gemini", "what is rust"},
		{"GEMINI what is rust", "gemini", "what is rust"},
		{"chat write a regex", "code", "write a regex"},
		{"Chat, write a regex", "code", "write a regex"},
		{"chat: write a regex", "code", "write a regex"},
		{"hello world", "default", "hello world"},
		{"gemini", "default", "gemini"},
		{"chat", "default", "chat"},
		{"", "default", ""},
		{"  gemini  what is rust  ", "gemini", "what is rust"},
		// Word boundary: must not match words starting with the prefix.
		{"chatbots are cool", "default", "chatbots are cool"},
		{"chatty person", "default", "chatty person"},
		{"gemini's answer", "default", "gemini's answer"},
		{"transcription is done", "default", "transcription is done"},
	}

	for _, tt := range tests {
		route := ParseRoute(tt.input, prefixes)
		if route.Name != tt.wantRoute || route.Query != tt.wantQuery {
			t.Errorf("ParseRoute(%q) = {Name:%q, Query:%q}, want {Name:%q, Query:%q}",
				tt.input, route.Name, route.Query, tt.wantRoute, tt.wantQuery)
		}
	}
}
