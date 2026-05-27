package transform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/squoyster/hydracast/internal/config"
	"github.com/squoyster/hydracast/internal/download"
)

type FFmpeg struct {
	binary string
}

func NewFFmpeg(binary string) *FFmpeg {
	if binary == "" {
		binary = "ffmpeg"
	}
	return &FFmpeg{binary: binary}
}

func (f *FFmpeg) Name() string {
	return "ffmpeg"
}

func (f *FFmpeg) Transform(ctx context.Context, media *download.LocalMedia, cfg config.TransformConfig) (*download.LocalMedia, error) {
	if _, err := os.Stat(f.binary); err != nil {
		if _, err := exec.LookPath(f.binary); err != nil {
			return nil, fmt.Errorf("ffmpeg binary not found: %w", err)
		}
	}

	args, err := buildArgs(cfg, media.Path)
	if err != nil {
		return nil, fmt.Errorf("build ffmpeg args: %w", err)
	}

	outputPath := deriveOutputPath(media.Path, cfg.Preset)

	cmd := exec.CommandContext(ctx, f.binary, append(args, outputPath)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg transform failed: %w", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("stat output file: %w", err)
	}

	return &download.LocalMedia{
		Path:     outputPath,
		Filename: info.Name(),
		Size:     info.Size(),
	}, nil
}

func buildArgs(cfg config.TransformConfig, inputPath string) ([]string, error) {
	args := []string{"-i", inputPath}

	if len(cfg.Args) > 0 {
		args = append(args, cfg.Args...)
	} else {
		presetArgs, err := presetArgs(cfg.Preset)
		if err != nil {
			return nil, err
		}
		args = append(args, presetArgs...)
	}

	args = append(args, "-y")

	return args, nil
}

func presetArgs(preset string) ([]string, error) {
	switch preset {
	case "faststart_mp4":
		return []string{
			"-c", "copy",
			"-movflags", "+faststart",
			"-f", "mp4",
		}, nil
	case "normalize_audio":
		return []string{
			"-c:v", "copy",
			"-af", "loudnorm=I=-16:TP=-1.5:LRA=11",
			"-c:a", "aac",
			"-b:a", "192k",
		}, nil
	case "convert_to_mp4":
		return []string{
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
			"-c:a", "aac",
			"-b:a", "192k",
			"-movflags", "+faststart",
			"-f", "mp4",
		}, nil
	case "extract_audio":
		return []string{
			"-vn",
			"-c:a", "libmp3lame",
			"-b:a", "192k",
			"-f", "mp3",
		}, nil
	case "scale_1080p":
		return []string{
			"-vf", "scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2",
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
			"-c:a", "copy",
		}, nil
	case "none":
		return []string{"-c", "copy"}, nil
	default:
		return nil, fmt.Errorf("unknown preset: %s", preset)
	}
}

func deriveOutputPath(inputPath, preset string) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	outputExt := ext
	switch preset {
	case "extract_audio":
		outputExt = ".mp3"
	case "convert_to_mp4", "faststart_mp4":
		if ext != ".mp4" {
			outputExt = ".mp4"
		}
	}

	return filepath.Join(dir, fmt.Sprintf("%s_transformed%s", name, outputExt))
}
