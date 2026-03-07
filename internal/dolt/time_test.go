package dolt

import (
	"testing"
	"time"
)

func TestParseDoltTime(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   time.Time
		isZero bool
	}{
		{
			name:  "standard format with microseconds",
			input: "2024-01-15 10:30:45.123456",
			want:  time.Date(2024, 1, 15, 10, 30, 45, 123456000, time.UTC),
		},
		{
			name:  "standard format without microseconds",
			input: "2024-01-15 10:30:45",
			want:  time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		},
		{
			name:  "ISO8601 with Z",
			input: "2024-01-15T10:30:45Z",
			want:  time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		},
		{
			name:  "ISO8601 with microseconds and Z",
			input: "2024-01-15T10:30:45.123456Z",
			want:  time.Date(2024, 1, 15, 10, 30, 45, 123456000, time.UTC),
		},
		{
			name:  "RFC3339 with timezone offset",
			input: "2024-01-15T10:30:45+00:00",
			want:  time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		},
		{
			name:   "invalid format",
			input:  "not-a-date",
			isZero: true,
		},
		{
			name:   "empty string",
			input:  "",
			isZero: true,
		},
		{
			name:   "wrong separator",
			input:  "2024/01/15",
			isZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDoltTime(tt.input)
			if tt.isZero {
				if !got.IsZero() {
					t.Errorf("parseDoltTime(%q) = %v, want zero time", tt.input, got)
				}
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("parseDoltTime(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
