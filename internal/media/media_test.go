package media

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFingerprint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	fp, err := Fingerprint(path)
	if err != nil {
		t.Fatalf("Fingerprint() error: %v", err)
	}

	expected := "sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if fp != expected {
		t.Errorf("Fingerprint() = %q, want %q", fp, expected)
	}
}

func TestFingerprintNonExistent(t *testing.T) {
	_, err := Fingerprint("/nonexistent/file.txt")
	if err == nil {
		t.Error("Fingerprint() expected error for nonexistent file")
	}
}

func TestCleanupStaleFiles(t *testing.T) {
	dir := t.TempDir()

	oldFile := filepath.Join(dir, "old.txt")
	if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	newFile := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	if err := CleanupStaleFiles(dir, 1*time.Hour); err != nil {
		t.Fatalf("CleanupStaleFiles() error: %v", err)
	}

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("old file should have been removed")
	}

	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Error("new file should not have been removed")
	}
}

func TestDeleteMedia(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp4")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := DeleteMedia(path); err != nil {
		t.Fatalf("DeleteMedia() error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}

	if err := DeleteMedia(""); err != nil {
		t.Error("DeleteMedia(\"\") should not error")
	}

	if err := DeleteMedia("/nonexistent/path"); err != nil {
		t.Error("DeleteMedia() should not error for nonexistent file")
	}
}
