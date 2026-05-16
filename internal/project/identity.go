package project

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
)

const ProjectIDPrefix = "proj_"

var projectIDPattern = regexp.MustCompile(`^proj_[a-f0-9]{24}$`)

func GenerateID() (string, error) {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate project id: %w", err)
	}
	return ProjectIDPrefix + hex.EncodeToString(raw[:]), nil
}

func IsValidID(id string) bool {
	return projectIDPattern.MatchString(id)
}
