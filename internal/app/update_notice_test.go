package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckForUpdateNoticeFetchesAndCachesAvailableUpdate(t *testing.T) {
	root := t.TempDir()
	checkedAt := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.1.1"})
	}))
	defer server.Close()

	service := NewService()
	service.version = "v1.1.0"
	service.resolveYeastHome = func() (string, error) { return root, nil }
	service.latestReleaseURL = server.URL
	service.now = func() time.Time { return checkedAt }

	notice, err := service.CheckForUpdateNotice(context.Background(), UpdateNoticeOptions{MaxAge: 24 * time.Hour})
	if err != nil {
		t.Fatalf("CheckForUpdateNotice returned error: %v", err)
	}
	if notice == nil {
		t.Fatal("expected update notice, got nil")
	}
	if notice.CurrentVersion != "v1.1.0" || notice.LatestVersion != "v1.1.1" {
		t.Fatalf("unexpected notice: %#v", notice)
	}
	if requests != 1 {
		t.Fatalf("expected one release request, got %d", requests)
	}

	raw, err := os.ReadFile(filepath.Join(root, "cache", "update-check.json"))
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	if !containsAll(string(raw), "v1.1.0", "v1.1.1") {
		t.Fatalf("cache did not record versions: %s", raw)
	}
}

func TestCheckForUpdateNoticeUsesFreshCacheWithoutNetwork(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	seedUpdateNoticeCache(t, root, updateNoticeCache{
		LastCheckedAt:   now.Add(-time.Hour),
		CurrentVersion:  "v1.1.0",
		LatestVersion:   "v1.1.1",
		UpdateAvailable: true,
	})

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("fresh cache should not call network")
	}))
	defer server.Close()

	service := NewService()
	service.version = "v1.1.0"
	service.resolveYeastHome = func() (string, error) { return root, nil }
	service.latestReleaseURL = server.URL
	service.now = func() time.Time { return now }

	notice, err := service.CheckForUpdateNotice(context.Background(), UpdateNoticeOptions{MaxAge: 24 * time.Hour})
	if err != nil {
		t.Fatalf("CheckForUpdateNotice returned error: %v", err)
	}
	if notice == nil || notice.LatestVersion != "v1.1.1" {
		t.Fatalf("expected cached update notice, got %#v", notice)
	}
	if requests != 0 {
		t.Fatalf("expected no network requests, got %d", requests)
	}
}

func TestCheckForUpdateNoticeRefreshesExpiredCache(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	seedUpdateNoticeCache(t, root, updateNoticeCache{
		LastCheckedAt:   now.Add(-25 * time.Hour),
		CurrentVersion:  "v1.1.0",
		LatestVersion:   "v1.1.1",
		UpdateAvailable: true,
	})

	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.1.2"})
	}))
	defer server.Close()

	service := NewService()
	service.version = "v1.1.0"
	service.resolveYeastHome = func() (string, error) { return root, nil }
	service.latestReleaseURL = server.URL
	service.now = func() time.Time { return now }

	notice, err := service.CheckForUpdateNotice(context.Background(), UpdateNoticeOptions{MaxAge: 24 * time.Hour})
	if err != nil {
		t.Fatalf("CheckForUpdateNotice returned error: %v", err)
	}
	if notice == nil || notice.LatestVersion != "v1.1.2" {
		t.Fatalf("expected refreshed update notice, got %#v", notice)
	}
	if requests != 1 {
		t.Fatalf("expected one refresh request, got %d", requests)
	}
}

func TestCheckForUpdateNoticeReturnsNilWhenAlreadyLatest(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.1.1"})
	}))
	defer server.Close()

	service := NewService()
	service.version = "1.1.1"
	service.resolveYeastHome = func() (string, error) { return root, nil }
	service.latestReleaseURL = server.URL
	service.now = func() time.Time { return now }

	notice, err := service.CheckForUpdateNotice(context.Background(), UpdateNoticeOptions{MaxAge: 24 * time.Hour})
	if err != nil {
		t.Fatalf("CheckForUpdateNotice returned error: %v", err)
	}
	if notice != nil {
		t.Fatalf("expected no update notice, got %#v", notice)
	}
}

func seedUpdateNoticeCache(t *testing.T, root string, cache updateNoticeCache) {
	t.Helper()
	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("marshal cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "update-check.json"), data, 0644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
}

func containsAll(text string, values ...string) bool {
	for _, value := range values {
		if !strings.Contains(text, value) {
			return false
		}
	}
	return true
}
