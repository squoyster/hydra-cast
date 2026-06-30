package source

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// reelsIntake mirrors the JSON schema written by the hydracast-mac-playwright
// collector (see reels.json). Only items[].url is required; the collector also
// pre-computes external_id (reel id) and url_sha256, which are reused so dedup
// and provenance survive intact.
type reelsIntake struct {
	SchemaVersion string     `json:"schema_version"`
	Collector     string     `json:"collector"`
	Items         []reelItem `json:"items"`
}

type reelItem struct {
	URL              string `json:"url"`
	ExternalID       string `json:"external_id"`
	URLSHA256        string `json:"url_sha256"`
	FirstSeenInRunAt string `json:"first_seen_in_run_at"`
}

// URLList is a source plugin that reads video URLs from a reels.json intake
// file produced by the hydracast-mac-playwright collector. Each run Scans the
// file; the app layer deletes the file once items are consumed (gated on
// !dry-run). Missing file = idle (no work), not an error.
type URLList struct {
	name string
	path string
}

func NewURLList(name, path string) *URLList {
	return &URLList{name: name, path: path}
}

func (u *URLList) Name() string { return u.name }
func (u *URLList) Type() string { return "url_list" }

// Scan reads the intake file. Scan is non-destructive; the caller owns file
// lifecycle (drain after consumption). Missing file = idle (nil, nil).
func (u *URLList) Scan(ctx context.Context) ([]MediaItem, error) {
	data, err := os.ReadFile(u.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", u.path, err)
	}
	var intake reelsIntake
	if err := json.Unmarshal(data, &intake); err != nil {
		return nil, fmt.Errorf("parse %s (expect reels intake {items:[...]}): %w", u.path, err)
	}
	now := time.Now()
	items := make([]MediaItem, 0, len(intake.Items))
	for _, it := range intake.Items {
		url := strings.TrimSpace(it.URL)
		if url == "" {
			continue
		}
		externalID := it.ExternalID
		if externalID == "" {
			externalID = url
		}
		fp := it.URLSHA256
		if fp == "" {
			fp = "pending"
		}
		detected := now
		if it.FirstSeenInRunAt != "" {
			if t, err := time.Parse(time.RFC3339, it.FirstSeenInRunAt); err == nil {
				detected = t
			}
		}
		items = append(items, MediaItem{
			SourceName:  u.name,
			SourceType:  "url_list",
			ExternalID:  externalID,
			SourceURL:   url,
			MediaType:   "video",
			DetectedAt:  detected,
			Fingerprint: fp,
		})
	}
	return items, nil
}
