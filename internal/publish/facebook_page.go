package publish

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/squoyster/hydracast/internal/config"
	"github.com/squoyster/hydracast/internal/download"
	"github.com/squoyster/hydracast/internal/source"
)

type FacebookPage struct {
	cfg       config.DestinationConfig
	ytDlpPath string
}

func NewFacebookPage(cfg config.DestinationConfig, ytDlpPath string) *FacebookPage {
	return &FacebookPage{
		cfg:       cfg,
		ytDlpPath: ytDlpPath,
	}
}

func (f *FacebookPage) Name() string {
	return f.cfg.Name
}

func (f *FacebookPage) Type() string {
	return "facebook_page"
}

func (f *FacebookPage) Publish(ctx context.Context, item source.MediaItem, media *download.LocalMedia) (*PublishResult, error) {
	if f.ytDlpPath == "" {
		f.ytDlpPath = "/usr/local/bin/yt-dlp"
	}

	if _, err := os.Stat(f.ytDlpPath); err != nil {
		return nil, fmt.Errorf("yt-dlp binary not found at %s: %w", f.ytDlpPath, err)
	}

	if f.cfg.PageID == "" {
		return &PublishResult{
			Status: "failed",
			Error:  fmt.Errorf("page_id not configured"),
		}, nil
	}

	title := item.Title
	if title == "" {
		title = media.Filename
	}

	description := fmt.Sprintf("Published via HydraCast from %s", item.SourceName)

	args := []string{
		"--no-playlist",
		"--title", title,
		"--description", description,
		"--external-downloader", "aria2c",
	}

	args = append(args, media.Path)

	cmd := exec.CommandContext(ctx, f.ytDlpPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &PublishResult{
			Status: "failed",
			Error:  fmt.Errorf("facebook publish failed: %w\n%s", err, string(output)),
		}, nil
	}

	videoID := extractFacebookVideoID(string(output))

	return &PublishResult{
		RemoteID:  videoID,
		RemoteURL: fmt.Sprintf("https://www.facebook.com/%s/videos/%s", f.cfg.PageID, videoID),
		Status:    "published",
	}, nil
}

func extractFacebookVideoID(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "facebook.com") && strings.Contains(line, "/videos/") {
			idx := strings.LastIndex(line, "/videos/")
			if idx != -1 {
				id := line[idx+8:]
				id = strings.Split(id, "?")[0]
				id = strings.Split(id, "&")[0]
				id = strings.TrimSpace(id)
				if len(id) > 0 {
					return id
				}
			}
		}
	}
	return ""
}
