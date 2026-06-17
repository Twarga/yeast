package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"yeast/internal/project"
)

const (
	defaultLatestReleaseURL    = "https://api.github.com/repos/Twarga/yeast/releases/latest"
	defaultUpdateNoticeMaxAge  = 24 * time.Hour
	defaultUpdateNoticeTimeout = 2 * time.Second
	updateNoticeCacheFileName  = "update-check.json"
	updateNoticeUserAgent      = "yeast-update-notice"
)

var releaseVersionPattern = regexp.MustCompile(`^v?([0-9]+)\.([0-9]+)\.([0-9]+)$`)

type UpdateNoticeOptions struct {
	MaxAge  time.Duration
	Timeout time.Duration
}

type UpdateNotice struct {
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version"`
	CheckedAt      time.Time `json:"checked_at"`
	CacheHit       bool      `json:"cache_hit"`
}

type updateNoticeCache struct {
	LastCheckedAt   time.Time `json:"last_checked_at"`
	CurrentVersion  string    `json:"current_version"`
	LatestVersion   string    `json:"latest_version"`
	UpdateAvailable bool      `json:"update_available"`
}

type latestReleasePayload struct {
	TagName string `json:"tag_name"`
}

func (s *Service) CheckForUpdateNotice(ctx context.Context, options UpdateNoticeOptions) (*UpdateNotice, error) {
	currentVersion, ok := normalizeReleaseVersion(s.Version())
	if !ok {
		return nil, nil
	}

	now := time.Now
	if s != nil && s.now != nil {
		now = s.now
	}
	checkedAt := now().UTC()

	maxAge := options.MaxAge
	if maxAge <= 0 {
		maxAge = defaultUpdateNoticeMaxAge
	}

	cachePath, err := s.updateNoticeCachePath()
	if err != nil {
		return nil, err
	}

	if cache, ok := readUpdateNoticeCache(cachePath); ok && checkedAt.Sub(cache.LastCheckedAt) < maxAge {
		if cache.UpdateAvailable && compareReleaseVersions(cache.LatestVersion, currentVersion) > 0 {
			return &UpdateNotice{
				CurrentVersion: currentVersion,
				LatestVersion:  cache.LatestVersion,
				CheckedAt:      cache.LastCheckedAt,
				CacheHit:       true,
			}, nil
		}
		return nil, nil
	}

	latestVersion, err := s.fetchLatestReleaseVersion(ctx, options.Timeout)
	if err != nil {
		return nil, err
	}
	latestVersion, ok = normalizeReleaseVersion(latestVersion)
	if !ok {
		return nil, nil
	}

	updateAvailable := compareReleaseVersions(latestVersion, currentVersion) > 0
	cache := updateNoticeCache{
		LastCheckedAt:   checkedAt,
		CurrentVersion:  currentVersion,
		LatestVersion:   latestVersion,
		UpdateAvailable: updateAvailable,
	}
	if err := writeUpdateNoticeCache(cachePath, cache); err != nil {
		return nil, err
	}
	if !updateAvailable {
		return nil, nil
	}

	return &UpdateNotice{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		CheckedAt:      checkedAt,
	}, nil
}

func (s *Service) updateNoticeCachePath() (string, error) {
	if s == nil || s.resolveYeastHome == nil {
		return "", fmt.Errorf("resolve yeast home is not configured")
	}
	yeastHome, err := s.resolveYeastHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(yeastHome, project.CacheDirName, updateNoticeCacheFileName), nil
}

func readUpdateNoticeCache(path string) (updateNoticeCache, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return updateNoticeCache{}, false
	}
	var cache updateNoticeCache
	if err := json.Unmarshal(raw, &cache); err != nil {
		return updateNoticeCache{}, false
	}
	if cache.LastCheckedAt.IsZero() || cache.LatestVersion == "" {
		return updateNoticeCache{}, false
	}
	return cache, true
}

func writeUpdateNoticeCache(path string, cache updateNoticeCache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func (s *Service) fetchLatestReleaseVersion(ctx context.Context, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = defaultUpdateNoticeTimeout
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := defaultLatestReleaseURL
	if s != nil && s.latestReleaseURL != "" {
		url = s.latestReleaseURL
	}
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", updateNoticeUserAgent)

	client := http.DefaultClient
	if s != nil && s.httpClient != nil {
		client = s.httpClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("latest release check returned HTTP %d", resp.StatusCode)
	}

	var payload latestReleasePayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.TagName, nil
}

func normalizeReleaseVersion(version string) (string, bool) {
	version = strings.TrimSpace(version)
	matches := releaseVersionPattern.FindStringSubmatch(version)
	if matches == nil {
		return "", false
	}
	return "v" + matches[1] + "." + matches[2] + "." + matches[3], true
}

func compareReleaseVersions(left, right string) int {
	leftParts, leftOK := releaseVersionParts(left)
	rightParts, rightOK := releaseVersionParts(right)
	if !leftOK || !rightOK {
		return strings.Compare(left, right)
	}
	for i := range leftParts {
		if leftParts[i] > rightParts[i] {
			return 1
		}
		if leftParts[i] < rightParts[i] {
			return -1
		}
	}
	return 0
}

func releaseVersionParts(version string) ([3]int, bool) {
	var parts [3]int
	matches := releaseVersionPattern.FindStringSubmatch(strings.TrimSpace(version))
	if matches == nil {
		return parts, false
	}
	for i := 0; i < 3; i++ {
		value, err := strconv.Atoi(matches[i+1])
		if err != nil {
			return parts, false
		}
		parts[i] = value
	}
	return parts, true
}
