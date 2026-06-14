package images

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloadSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "ubuntu-24.04", ImageFileName)
	err := Download(TrustedImage{
		Name:     "ubuntu-24.04",
		URL:      server.URL,
		Checksum: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
	}, destination, DownloadOptions{Timeout: time.Second})
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	raw, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("expected destination file to exist: %v", err)
	}
	if string(raw) != "hello" {
		t.Fatalf("expected downloaded content hello, got %q", string(raw))
	}
}

func TestDownloadSkipsValidCachedFile(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.Error(w, "should not download", http.StatusInternalServerError)
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "ubuntu-24.04", ImageFileName)
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	if err := os.WriteFile(destination, []byte("hello"), 0644); err != nil {
		t.Fatalf("write cached image: %v", err)
	}

	err := Download(TrustedImage{
		Name:     "ubuntu-24.04",
		URL:      server.URL,
		Checksum: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
	}, destination, DownloadOptions{Timeout: time.Second})
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	if requests != 0 {
		t.Fatalf("expected no HTTP requests for valid cached image, got %d", requests)
	}
}

func TestDownloadHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "ubuntu-24.04", ImageFileName)
	err := Download(TrustedImage{
		Name:     "ubuntu-24.04",
		URL:      server.URL,
		Checksum: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
	}, destination, DownloadOptions{Timeout: time.Second})
	if err == nil {
		t.Fatal("expected HTTP failure")
	}
	if !strings.Contains(err.Error(), "unexpected HTTP status 404") {
		t.Fatalf("expected HTTP status error, got %v", err)
	}
	if _, statErr := os.Stat(destination + ".part"); !os.IsNotExist(statErr) {
		t.Fatalf("expected no temp file after failure, stat err=%v", statErr)
	}
}

func TestDownloadChecksumFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "ubuntu-24.04", ImageFileName)
	err := Download(TrustedImage{
		Name:     "ubuntu-24.04",
		URL:      server.URL,
		Checksum: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}, destination, DownloadOptions{Timeout: time.Second})
	if err == nil {
		t.Fatal("expected checksum failure")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Fatalf("expected destination file to be absent, stat err=%v", statErr)
	}
	if _, statErr := os.Stat(destination + ".part"); !os.IsNotExist(statErr) {
		t.Fatalf("expected temp file cleanup after checksum failure, stat err=%v", statErr)
	}
}

func TestDownloadPartialCleanup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		_, _ = w.Write([]byte("abc"))
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "ubuntu-24.04", ImageFileName)
	err := Download(TrustedImage{
		Name:     "ubuntu-24.04",
		URL:      server.URL,
		Checksum: fmt.Sprintf("%064x", 0),
	}, destination, DownloadOptions{Timeout: time.Second})
	if err == nil {
		t.Fatal("expected partial download failure")
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Fatalf("expected destination file to be absent, stat err=%v", statErr)
	}
	if _, statErr := os.Stat(destination + ".part"); !os.IsNotExist(statErr) {
		t.Fatalf("expected temp file cleanup after partial failure, stat err=%v", statErr)
	}
}
