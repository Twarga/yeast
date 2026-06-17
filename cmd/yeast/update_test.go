package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

func TestExtractBinaryFromTarGzAcceptsLegacyReleaseBinaryName(t *testing.T) {
	var archive bytes.Buffer

	gzw := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gzw)

	content := []byte("legacy-binary")
	header := &tar.Header{
		Name: "yeast_linux_amd64",
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	got, err := extractBinaryFromTarGz(archive.Bytes(), "yeast", "yeast_linux_amd64")
	if err != nil {
		t.Fatalf("extractBinaryFromTarGz returned error: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("unexpected binary contents: got %q want %q", got, content)
	}
}
