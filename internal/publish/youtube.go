package publish

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/squoyster/hydracast/internal/config"
	"github.com/squoyster/hydracast/internal/download"
	"github.com/squoyster/hydracast/internal/source"
)

type YouTube struct {
	cfg       config.DestinationConfig
	ytDlpPath string
}

func NewYouTube(cfg config.DestinationConfig, ytDlpPath string) *YouTube {
	return &YouTube{
		cfg:       cfg,
		ytDlpPath: ytDlpPath,
	}
}

func (y *YouTube) Name() string {
	return y.cfg.Name
}

func (y *YouTube) Type() string {
	return "youtube"
}

func (y *YouTube) Publish(ctx context.Context, item source.MediaItem, media *download.LocalMedia) (*PublishResult, error) {
	if y.ytDlpPath == "" {
		y.ytDlpPath = "/usr/local/bin/yt-dlp"
	}

	if _, err := os.Stat(y.ytDlpPath); err != nil {
		return nil, fmt.Errorf("yt-dlp binary not found at %s: %w", y.ytDlpPath, err)
	}

	title := item.Title
	if title == "" {
		title = filepath.Base(media.Path)
	}

	description := fmt.Sprintf("Published via HydraCast from %s", item.SourceName)

	args := []string{
		"--no-playlist",
		"--title", title,
		"--description", description,
	}

	if y.cfg.Privacy != "" {
		args = append(args, "--metadata-from-title", fmt.Sprintf("privacy=%s", y.cfg.Privacy))
	}

	if y.cfg.CategoryID != "" {
		args = append(args, "--metadata-from-title", fmt.Sprintf("category=%s", y.cfg.CategoryID))
	}

	args = append(args, media.Path)

	cmd := exec.CommandContext(ctx, y.ytDlpPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &PublishResult{
			Status: "failed",
			Error:  fmt.Errorf("yt-dlp upload failed: %w\n%s", err, string(output)),
		}, nil
	}

	videoID := extractVideoID(string(output))

	return &PublishResult{
		RemoteID:  videoID,
		RemoteURL: fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
		Status:    "published",
	}, nil
}

func extractVideoID(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "youtube.com/watch?v=") {
			idx := strings.Index(line, "watch?v=")
			if idx != -1 {
				id := line[idx+8:]
				id = strings.Split(id, "&")[0]
				id = strings.TrimSpace(id)
				if len(id) == 11 {
					return id
				}
			}
		}
	}
	return ""
}
