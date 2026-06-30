package publish

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"

	"github.com/squoyster/hydracast/internal/config"
	"github.com/squoyster/hydracast/internal/download"
	"github.com/squoyster/hydracast/internal/secrets"
	"github.com/squoyster/hydracast/internal/source"
)

// YouTube uploads via the YouTube Data API v3 videos.insert (resumable upload),
// authenticating with an OAuth2 refresh token resolved from the secrets store.
type YouTube struct {
	cfg      config.DestinationConfig
	resolver *secrets.Resolver
}

func NewYouTube(cfg config.DestinationConfig, resolver *secrets.Resolver) *YouTube {
	return &YouTube{cfg: cfg, resolver: resolver}
}

func (y *YouTube) Name() string { return y.cfg.Name }
func (y *YouTube) Type() string { return "youtube" }

func (y *YouTube) Publish(ctx context.Context, item source.MediaItem, media *download.LocalMedia) (*PublishResult, error) {
	clientID, err := y.resolver.Resolve(y.cfg.ClientIDRef)
	if err != nil {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("resolve client_id: %w", err)}, nil
	}
	clientSecret, err := y.resolver.Resolve(y.cfg.ClientSecretRef)
	if err != nil {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("resolve client_secret: %w", err)}, nil
	}
	// token_ref points at a multi-field secret; #refresh_token selects the field.
	refreshToken, err := y.resolver.Resolve(y.cfg.TokenRef + "#refresh_token")
	if err != nil {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("resolve refresh_token: %w", err)}, nil
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{youtube.YoutubeUploadScope},
	}
	// oauth2Cfg.Client auto-refreshes the access token using the refresh token.
	httpClient := oauth2Cfg.Client(ctx, &oauth2.Token{RefreshToken: refreshToken})

	svc, err := youtube.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("build youtube service: %w", err)}, nil
	}

	title := item.Title
	if title == "" {
		title = filepath.Base(media.Path)
	}
	privacy := y.cfg.Privacy
	if privacy == "" {
		privacy = "public"
	}

	video := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:       title,
			Description: fmt.Sprintf("Published via HydraCast from %s", item.SourceName),
			CategoryId:  y.cfg.CategoryID,
		},
		Status: &youtube.VideoStatus{PrivacyStatus: privacy},
	}

	f, err := os.Open(media.Path)
	if err != nil {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("open media: %w", err)}, nil
	}
	defer f.Close()

	// Media() triggers a resumable upload for the file; the client chunks-retries it.
	resp, err := svc.Videos.Insert([]string{"snippet", "status"}, video).Media(f).Do()
	if err != nil {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("videos.insert: %w", err)}, nil
	}

	return &PublishResult{
		RemoteID:  resp.Id,
		RemoteURL: fmt.Sprintf("https://www.youtube.com/watch?v=%s", resp.Id),
		Status:    "published",
	}, nil
}
