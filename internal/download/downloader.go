package download

import (
	"context"

	"github.com/squoyster/hydracast/internal/source"
)

type LocalMedia struct {
	Path     string
	Filename string
	Size     int64
	Duration float64
	MimeType string
}

type Plugin interface {
	Name() string
	Supports(item source.MediaItem) bool
	Download(ctx context.Context, item source.MediaItem) (*LocalMedia, error)
}
