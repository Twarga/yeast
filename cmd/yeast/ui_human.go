package main

import (
	"fmt"
	"os"
	"strings"
)

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiCyan   = "\033[36m"
)

var humanColorEnabled = detectHumanColor(os.Stdout)

func detectHumanColor(f *os.File) bool {
	if f == nil {
		return false
	}
	if os.Getenv("NO_COLOR") != "" || strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func humanTTY() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func humanStyle(text string, codes ...string) string {
	if !humanColorEnabled {
		return text
	}
	return strings.Join(codes, "") + text + ansiReset
}

func humanMuted(text string) string {
	return humanStyle(text, ansiDim)
}

func humanAccent(text string) string {
	return humanStyle(text, ansiBold, ansiCyan)
}

func humanSuccessText(text string) string {
	return humanStyle(text, ansiBold, ansiGreen)
}

func humanWarningText(text string) string {
	return humanStyle(text, ansiBold, ansiYellow)
}

func humanErrorText(text string) string {
	return humanStyle(text, ansiBold, ansiRed)
}

func humanSection(title string) {
	fmt.Printf("%s %s\n", humanStyle("==>", ansiBold, ansiBlue), humanStyle(title, ansiBold))
}

func humanInfof(format string, args ...any) {
	fmt.Printf("%s %s\n", humanStyle("[info]", ansiBlue), fmt.Sprintf(format, args...))
}

func humanSuccessf(format string, args ...any) {
	fmt.Printf("%s %s\n", humanStyle("[ok]", ansiGreen), fmt.Sprintf(format, args...))
}

func humanWarnf(format string, args ...any) {
	fmt.Printf("%s %s\n", humanStyle("[warn]", ansiYellow), fmt.Sprintf(format, args...))
}

func humanErrorf(format string, args ...any) {
	fmt.Printf("%s %s\n", humanStyle("[fail]", ansiRed), fmt.Sprintf(format, args...))
}

func humanKeyValue(label, value string) {
	fmt.Printf("    %s %s\n", humanMuted(label+":"), value)
}

func humanStatusLabel(status string) string {
	switch status {
	case "running":
		return humanSuccessText(status)
	case "stopped":
		return humanMuted(status)
	case "error":
		return humanErrorText(status)
	default:
		return status
	}
}
