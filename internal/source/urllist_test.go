package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestURLListScan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reels.json")
	if err := os.WriteFile(path, []byte(`{
		"schema_version":"1.0",
		"collector":"hydracast-mac-playwright",
		"items":[
			{"url":"https://www.facebook.com/reel/1014475400998131","external_id":"1014475400998131","url_sha256":"abc123","first_seen_in_run_at":"2026-06-29T23:38:53.469Z"},
			{"url":"  https://www.facebook.com/reel/1489785266162269  ","external_id":"1489785266162269"},
			{"url":"https://no-id.example/reel/999"},
			{"url":""}
		]
	}`), 0600); err != nil {
		t.Fatal(err)
	}

	u := NewURLList("intake", path)
	items, err := u.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3 (blank entry skipped)", len(items))
	}
	if items[0].ExternalID != "1014475400998131" {
		t.Errorf("ExternalID = %q, want collector external_id", items[0].ExternalID)
	}
	if items[0].Fingerprint != "abc123" {
		t.Errorf("Fingerprint = %q, want url_sha256", items[0].Fingerprint)
	}
	if items[0].DetectedAt.Year() != 2026 {
		t.Errorf("DetectedAt = %v, want first_seen_in_run_at parsed (2026)", items[0].DetectedAt)
	}
	if items[1].Fingerprint != "pending" {
		t.Errorf("Fingerprint = %q, want pending (no url_sha256)", items[1].Fingerprint)
	}
	if items[1].SourceURL != "https://www.facebook.com/reel/1489785266162269" {
		t.Errorf("SourceURL = %q, want trimmed", items[1].SourceURL)
	}
	if items[2].ExternalID != "https://no-id.example/reel/999" {
		t.Errorf("ExternalID = %q, want URL fallback (no external_id)", items[2].ExternalID)
	}
	for _, it := range items {
		if it.SourceType != "url_list" {
			t.Errorf("SourceType = %q, want url_list", it.SourceType)
		}
		if it.MediaType != "video" {
			t.Errorf("MediaType = %q, want video", it.MediaType)
		}
	}
}

func TestURLListScanMissingFile(t *testing.T) {
	u := NewURLList("intake", filepath.Join(t.TempDir(), "absent.json"))
	items, err := u.Scan(context.Background())
	if err != nil {
		t.Fatalf("missing file should be idle, got error: %v", err)
	}
	if items != nil {
		t.Fatalf("expected nil items for missing file, got %d", len(items))
	}
}

func TestURLListScanEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reels.json")
	if err := os.WriteFile(path, []byte(`{"items":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	u := NewURLList("intake", path)
	items, err := u.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items for empty intake, got %d", len(items))
	}
}

func TestURLListScanMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reels.json")
	if err := os.WriteFile(path, []byte(`{not json}`), 0600); err != nil {
		t.Fatal(err)
	}
	u := NewURLList("intake", path)
	if _, err := u.Scan(context.Background()); err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}
