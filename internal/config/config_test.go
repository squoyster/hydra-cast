package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `version: 1
app:
  name: test
  timezone: UTC
storage:
  database: /data/test.db
  work_dir: /data/work
  cache_dir: /data/cache
secrets:
  provider: files
  fallback:
    provider: files
    enabled: true
    root: /data/secrets
sources:
  - name: test-source
    type: facebook_page_videos
    url: "https://example.com"
    downloader: yt_dlp
    enabled: true
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if cfg.App.Name != "test" {
		t.Errorf("App.Name = %q, want %q", cfg.App.Name, "test")
	}
	if len(cfg.Sources) != 1 {
		t.Fatalf("Sources len = %d, want 1", len(cfg.Sources))
	}
	if cfg.Sources[0].Name != "test-source" {
		t.Errorf("Source[0].Name = %q, want %q", cfg.Sources[0].Name, "test-source")
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	ApplyDefaults(cfg)

	if cfg.App.Name != "hydracast" {
		t.Errorf("App.Name = %q, want %q", cfg.App.Name, "hydracast")
	}
	if cfg.App.Timezone != "UTC" {
		t.Errorf("App.Timezone = %q, want %q", cfg.App.Timezone, "UTC")
	}
	if cfg.Storage.Database != "/data/hydracast.db" {
		t.Errorf("Storage.Database = %q, want %q", cfg.Storage.Database, "/data/hydracast.db")
	}
	if cfg.Limits.MaxItemsPerRun != 3 {
		t.Errorf("MaxItemsPerRun = %d, want 3", cfg.Limits.MaxItemsPerRun)
	}
	if cfg.Limits.JobEventRetention != 1000 {
		t.Errorf("JobEventRetention = %d, want 1000", cfg.Limits.JobEventRetention)
	}
}

func TestValidate(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "work")
	cacheDir := filepath.Join(dir, "cache")

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Version: 1,
				Secrets: SecretsConfig{Provider: "files"},
				Storage: StorageConfig{WorkDir: workDir, CacheDir: cacheDir},
				Sources: []SourceConfig{
					{Name: "s1", Type: "facebook_page_videos", Downloader: "yt_dlp", Enabled: true},
				},
				Transforms: []TransformConfig{
					{Name: "t1", Type: "ffmpeg", Enabled: true},
				},
				Destinations: []DestinationConfig{
					{Name: "d1", Type: "youtube", Enabled: true},
				},
			},
			wantErr: false,
		},
		{
			name: "unknown source type",
			cfg: &Config{
				Version: 1,
				Secrets: SecretsConfig{Provider: "files"},
				Storage: StorageConfig{WorkDir: workDir, CacheDir: cacheDir},
				Sources: []SourceConfig{
					{Name: "s1", Type: "unknown_type", Downloader: "yt_dlp", Enabled: true},
				},
			},
			wantErr: true,
		},
		{
			name: "unknown destination type",
			cfg: &Config{
				Version: 1,
				Secrets: SecretsConfig{Provider: "files"},
				Storage: StorageConfig{WorkDir: workDir, CacheDir: cacheDir},
				Destinations: []DestinationConfig{
					{Name: "d1", Type: "unknown_dest", Enabled: true},
				},
			},
			wantErr: true,
		},
		{
			name: "disabled source skipped",
			cfg: &Config{
				Version: 1,
				Secrets: SecretsConfig{Provider: "files"},
				Storage: StorageConfig{WorkDir: workDir, CacheDir: cacheDir},
				Sources: []SourceConfig{
					{Name: "s1", Type: "unknown_type", Downloader: "yt_dlp", Enabled: false},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.cfg)
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("Validate() errs = %v, wantErr = %v", errs, tt.wantErr)
			}
		})
	}
}

func TestValidateSecretRefs(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid refs",
			cfg: &Config{
				Destinations: []DestinationConfig{
					{ClientSecretRef: "secret://openbao/kv/hydracast/youtube/client"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid ref format",
			cfg: &Config{
				Destinations: []DestinationConfig{
					{ClientSecretRef: "not-a-secret-ref"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateSecretRefs(tt.cfg)
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("ValidateSecretRefs() errs = %v, wantErr = %v", errs, tt.wantErr)
			}
		})
	}
}
