package components

import (
	"testing"
	"time"
)

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		name string
		dur  time.Duration // subtracted from time.Now()
		want string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"hours", 3 * time.Hour, "3h ago"},
		{"days", 7 * 24 * time.Hour, "7d ago"},
		{"months", 60 * 24 * time.Hour, "2mo ago"},
		{"years", 400 * 24 * time.Hour, "1y ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := time.Now().Add(-tt.dur)
			got := relativeTime(input)
			if got != tt.want {
				t.Errorf("relativeTime(%v ago) = %q, want %q", tt.dur, got, tt.want)
			}
		})
	}

	t.Run("zero time", func(t *testing.T) {
		got := relativeTime(time.Time{})
		if got != "" {
			t.Errorf("relativeTime(zero) = %q, want empty string", got)
		}
	})
}
