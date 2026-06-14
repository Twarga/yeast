package main

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{500 * time.Millisecond, "0.5s"},
		{1500 * time.Millisecond, "2s"},
		{30 * time.Second, "30s"},
		{65 * time.Second, "1m 5s"},
		{120 * time.Second, "2m"},
		{3661 * time.Second, "61m 1s"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.input)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
