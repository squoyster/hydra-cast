package publish

import (
	"context"

	"github.com/squoyster/hydracast/internal/download"
	"github.com/squoyster/hydracast/internal/source"
)

type PublishResult struct {
	RemoteID  string
	RemoteURL string
	Status    string
	Error     error
}

type Plugin interface {
	Name() string
	Type() string
	Publish(ctx context.Context, item source.MediaItem, media *download.LocalMedia) (*PublishResult, error)
}
