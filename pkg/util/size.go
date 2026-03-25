package util

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var byteSizePattern = regexp.MustCompile(`(?i)^\s*([0-9]+)\s*([kmgtp]?)(?:b)?\s*$`)

func ParseByteSize(raw string) (int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("size cannot be empty")
	}

	matches := byteSizePattern.FindStringSubmatch(trimmed)
	if len(matches) != 3 {
		return 0, fmt.Errorf("unsupported size %q (examples: 20G, 10240M, 21474836480)", raw)
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size value %q: %w", matches[1], err)
	}

	multiplier := int64(1)
	switch strings.ToUpper(matches[2]) {
	case "":
		multiplier = 1
	case "K":
		multiplier = 1024
	case "M":
		multiplier = 1024 * 1024
	case "G":
		multiplier = 1024 * 1024 * 1024
	case "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "P":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unsupported size suffix %q", matches[2])
	}

	if value > math.MaxInt64/multiplier {
		return 0, fmt.Errorf("size %q is too large", raw)
	}

	return value * multiplier, nil
}

func NormalizeByteSize(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("size cannot be empty")
	}

	matches := byteSizePattern.FindStringSubmatch(trimmed)
	if len(matches) != 3 {
		return "", fmt.Errorf("unsupported size %q (examples: 20G, 10240M, 21474836480)", raw)
	}

	suffix := strings.ToUpper(matches[2])
	if suffix == "" {
		return matches[1], nil
	}
	return matches[1] + suffix, nil
}
