package transform

import (
	"context"

	"github.com/squoyster/hydracast/internal/download"
)

type Plugin interface {
	Name() string
	Transform(ctx context.Context, media *download.LocalMedia) (*download.LocalMedia, error)
}
