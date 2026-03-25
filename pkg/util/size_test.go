package util

import "testing"

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		raw  string
		want int64
	}{
		{raw: "20G", want: 20 * 1024 * 1024 * 1024},
		{raw: "10240M", want: 10240 * 1024 * 1024},
		{raw: "4096", want: 4096},
		{raw: "25gb", want: 25 * 1024 * 1024 * 1024},
	}

	for _, tc := range tests {
		got, err := ParseByteSize(tc.raw)
		if err != nil {
			t.Fatalf("ParseByteSize(%q) returned error: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("ParseByteSize(%q)=%d, want %d", tc.raw, got, tc.want)
		}
	}
}

func TestParseByteSizeRejectsInvalidValues(t *testing.T) {
	for _, raw := range []string{"", "G", "20GiB", "abc", "10XB"} {
		if _, err := ParseByteSize(raw); err == nil {
			t.Fatalf("expected ParseByteSize(%q) to fail", raw)
		}
	}
}

func TestNormalizeByteSize(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "20g", want: "20G"},
		{raw: "25 GB", want: "25G"},
		{raw: "4096", want: "4096"},
	}

	for _, tc := range tests {
		got, err := NormalizeByteSize(tc.raw)
		if err != nil {
			t.Fatalf("NormalizeByteSize(%q) returned error: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeByteSize(%q)=%q, want %q", tc.raw, got, tc.want)
		}
	}
}
