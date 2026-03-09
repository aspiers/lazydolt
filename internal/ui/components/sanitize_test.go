package components

import "testing"

func TestSanitizeForDisplay(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text unchanged",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "newlines preserved",
			input: "line1\nline2\nline3",
			want:  "line1\nline2\nline3",
		},
		{
			name:  "tabs preserved",
			input: "col1\tcol2",
			want:  "col1\tcol2",
		},
		{
			name:  "NUL replaced",
			input: "before\x00after",
			want:  "before·after",
		},
		{
			name:  "BEL replaced",
			input: "alert\x07here",
			want:  "alert·here",
		},
		{
			name:  "backspace replaced",
			input: "back\x08space",
			want:  "back·space",
		},
		{
			name:  "carriage return replaced",
			input: "line\roverwrite",
			want:  "line·overwrite",
		},
		{
			name:  "escape char replaced",
			input: "esc\x1bhere",
			want:  "esc·here",
		},
		{
			name:  "multiple control chars",
			input: "\x00\x01\x02\x03hello\x07\x08world\x1b",
			want:  "····hello··world·",
		},
		{
			name:  "unicode text preserved",
			input: "日本語テスト",
			want:  "日本語テスト",
		},
		{
			name:  "emoji preserved",
			input: "hello 🌍 world",
			want:  "hello 🌍 world",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "mixed binary and text",
			input: "id=\x00\x01\x02 name=test\x07",
			want:  "id=··· name=test·",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeForDisplay(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeForDisplay(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
