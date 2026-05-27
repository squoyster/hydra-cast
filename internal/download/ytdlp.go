package download

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/squoyster/hydracast/internal/config"
	"github.com/squoyster/hydracast/internal/source"
)

type YtDlp struct {
	cfg    config.YtDlpConfig
	workDir string
}

func NewYtDlp(cfg config.YtDlpConfig, workDir string) *YtDlp {
	return &YtDlp{
		cfg:     cfg,
		workDir: workDir,
	}
}

func (y *YtDlp) Name() string {
	return "yt_dlp"
}

func (y *YtDlp) Supports(item source.MediaItem) bool {
	return true
}

func (y *YtDlp) Download(ctx context.Context, item source.MediaItem) (*LocalMedia, error) {
	if _, err := os.Stat(y.cfg.Binary); err != nil {
		return nil, fmt.Errorf("yt-dlp binary not found at %s: %w", y.cfg.Binary, err)
	}

	outputTemplate := y.cfg.OutputTemplate
	if outputTemplate == "" {
		outputTemplate = filepath.Join(y.workDir, "%(extractor)s-%(id)s.%(ext)s")
	}

	args := []string{
		"--no-playlist",
		"--format", y.cfg.Format,
		"--output", outputTemplate,
		"--no-mtime",
	}

	if y.cfg.CookiesRef != "" {
		cookieFile, err := y.materializeCookies(ctx)
		if err != nil {
			return nil, fmt.Errorf("materialize cookies: %w", err)
		}
		defer os.Remove(cookieFile)
		args = append(args, "--cookies", cookieFile)
	}

	args = append(args, item.SourceURL)

	cmd := exec.CommandContext(ctx, y.cfg.Binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp download failed: %w", err)
	}

	downloadedPath, err := y.findDownloadedFile(outputTemplate, item.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("find downloaded file: %w", err)
	}

	info, err := os.Stat(downloadedPath)
	if err != nil {
		return nil, fmt.Errorf("stat downloaded file: %w", err)
	}

	return &LocalMedia{
		Path:     downloadedPath,
		Filename: info.Name(),
		Size:     info.Size(),
	}, nil
}

func (y *YtDlp) materializeCookies(ctx context.Context) (string, error) {
	tmpFile, err := os.CreateTemp(y.workDir, "cookies-*.txt")
	if err != nil {
		return "", fmt.Errorf("create temp cookie file: %w", err)
	}
	defer tmpFile.Close()

	return tmpFile.Name(), nil
}

func (y *YtDlp) findDownloadedFile(template, externalID string) (string, error) {
	dir := y.workDir
	if idx := strings.LastIndex(template, "/"); idx != -1 {
		dir = template[:idx]
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.Contains(entry.Name(), externalID) {
			return filepath.Join(dir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("no file found matching external_id %s in %s", externalID, dir)
}
