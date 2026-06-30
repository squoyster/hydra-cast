package source

import (
	"context"
	"time"
)

type MediaItem struct {
	ID          int64
	SourceName  string
	SourceType  string
	ExternalID  string
	SourceURL   string
	Title       string
	MediaType   string
	PublishedAt *time.Time
	DetectedAt  time.Time
	Fingerprint string
	RawMetadata string
}

type Plugin interface {
	Name() string
	Type() string
	Scan(ctx context.Context) ([]MediaItem, error)
}
