package transform

import (
	"testing"

	"github.com/squoyster/hydracast/internal/config"
)

func TestPresetArgs(t *testing.T) {
	tests := []struct {
		preset  string
		wantLen int
		wantErr bool
	}{
		{"faststart_mp4", 6, false},
		{"normalize_audio", 8, false},
		{"convert_to_mp4", 14, false},
		{"extract_audio", 7, false},
		{"scale_1080p", 10, false},
		{"none", 2, false},
		{"unknown", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.preset, func(t *testing.T) {
			args, err := presetArgs(tt.preset)
			if (err != nil) != tt.wantErr {
				t.Errorf("presetArgs(%q) error = %v, wantErr = %v", tt.preset, err, tt.wantErr)
			}
			if !tt.wantErr && len(args) != tt.wantLen {
				t.Errorf("presetArgs(%q) len = %d, want %d", tt.preset, len(args), tt.wantLen)
			}
		})
	}
}

func TestDeriveOutputPath(t *testing.T) {
	tests := []struct {
		inputPath string
		preset    string
		wantExt   string
	}{
		{"/data/work/video.mp4", "faststart_mp4", ".mp4"},
		{"/data/work/video.webm", "convert_to_mp4", ".mp4"},
		{"/data/work/video.mp4", "extract_audio", ".mp3"},
		{"/data/work/video.mp4", "none", ".mp4"},
	}

	for _, tt := range tests {
		t.Run(tt.preset, func(t *testing.T) {
			got := deriveOutputPath(tt.inputPath, tt.preset)
			ext := got[len(got)-len(tt.wantExt):]
			if ext != tt.wantExt {
				t.Errorf("deriveOutputPath(%q, %q) ext = %q, want %q", tt.inputPath, tt.preset, ext, tt.wantExt)
			}
		})
	}
}

func TestFFmpegName(t *testing.T) {
	f := NewFFmpeg("")
	if f.Name() != "ffmpeg" {
		t.Errorf("Name() = %q, want %q", f.Name(), "ffmpeg")
	}
}

func TestBuildArgsWithCustomArgs(t *testing.T) {
	cfg := config.TransformConfig{
		Preset: "none",
		Args:   []string{"-c", "copy", "-f", "mp4"},
	}

	args, err := buildArgs(cfg, "/data/work/input.mp4")
	if err != nil {
		t.Fatalf("buildArgs() error: %v", err)
	}

	if len(args) < 6 {
		t.Errorf("buildArgs() len = %d, want >= 6", len(args))
	}

	found := false
	for _, a := range args {
		if a == "-f" {
			found = true
			break
		}
	}
	if !found {
		t.Error("buildArgs() should include custom args")
	}
}
