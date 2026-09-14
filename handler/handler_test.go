package handler

import (
	"testing"
)

func TestExtractTranscription(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with transcript tags",
			input: "<transcript>\nGemini test\n</transcript>\n\nThe above is a transcript...",
			want:  "Gemini test",
		},
		{
			name:  "without transcript tags",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "multiline transcript",
			input: "<transcript>\nline one\nline two\n</transcript>\n\nInstructions here",
			want:  "line one\nline two",
		},
		{
			name:  "gemini prefix in transcript",
			input: "<transcript>\nGemini what is rust\n</transcript>\n\nClean it up",
			want:  "Gemini what is rust",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTranscription(tt.input)
			if got != tt.want {
				t.Errorf("extractTranscription() = %q, want %q", got, tt.want)
			}
		})
	}
}
