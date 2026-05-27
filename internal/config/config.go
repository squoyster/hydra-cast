package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version     int         `yaml:"version"`
	App         AppConfig   `yaml:"app"`
	Storage     StorageConfig `yaml:"storage"`
	Secrets     SecretsConfig `yaml:"secrets"`
	Limits      LimitsConfig  `yaml:"limits"`
	Downloaders DownloadersConfig `yaml:"downloaders"`
	Sources     []SourceConfig    `yaml:"sources"`
	Transforms  []TransformConfig `yaml:"transforms"`
	Destinations []DestinationConfig `yaml:"destinations"`
	Routes      []RouteConfig       `yaml:"routes"`
}

type AppConfig struct {
	Name     string `yaml:"name"`
	Timezone string `yaml:"timezone"`
}

type StorageConfig struct {
	Database string `yaml:"database"`
	WorkDir  string `yaml:"work_dir"`
	CacheDir string `yaml:"cache_dir"`
}

type SecretsConfig struct {
	Provider string           `yaml:"provider"`
	OpenBao  OpenBaoConfig    `yaml:"openbao"`
	Fallback FallbackConfig   `yaml:"fallback"`
}

type OpenBaoConfig struct {
	Address   string        `yaml:"address"`
	Namespace string        `yaml:"namespace"`
	Mount     string        `yaml:"mount"`
	TokenFile string        `yaml:"token_file"`
	AppPath   string        `yaml:"app_path"`
	Timeout   time.Duration `yaml:"timeout"`
}

type FallbackConfig struct {
	Provider string `yaml:"provider"`
	Enabled  bool   `yaml:"enabled"`
	Root     string `yaml:"root"`
}

type LimitsConfig struct {
	MaxConcurrentJobs  int           `yaml:"max_concurrent_jobs"`
	MaxItemsPerRun     int           `yaml:"max_items_per_run"`
	MaxWorkingBytes    int64         `yaml:"max_working_bytes"`
	MaxMediaDuration   time.Duration `yaml:"max_media_duration"`
	KeepSuccessfulMedia bool         `yaml:"keep_successful_media"`
	KeepFailedMedia    bool          `yaml:"keep_failed_media"`
	JobEventRetention  int           `yaml:"job_event_retention"`
}

type DownloadersConfig struct {
	YtDlp YtDlpConfig `yaml:"yt_dlp"`
}

type YtDlpConfig struct {
	Binary         string `yaml:"binary"`
	CookiesRef     string `yaml:"cookies_ref"`
	OutputTemplate string `yaml:"output_template"`
	Format         string `yaml:"format"`
}

type SourceConfig struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	URL        string `yaml:"url"`
	Downloader string `yaml:"downloader"`
	Enabled    bool   `yaml:"enabled"`
}

type TransformConfig struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Enabled bool     `yaml:"enabled"`
	Preset  string   `yaml:"preset"`
	Args    []string `yaml:"args"`
}

type DestinationConfig struct {
	Name            string `yaml:"name"`
	Type            string `yaml:"type"`
	Enabled         bool   `yaml:"enabled"`
	ClientSecretRef string `yaml:"client_secret_ref"`
	TokenRef        string `yaml:"token_ref"`
	Privacy         string `yaml:"privacy"`
	CategoryID      string `yaml:"category_id"`
	PageID          string `yaml:"page_id"`
	PageTokenRef    string `yaml:"page_token_ref"`
}

type RouteConfig struct {
	Name        string   `yaml:"name"`
	Source      string   `yaml:"source"`
	Transforms  []string `yaml:"transforms"`
	Destinations []string `yaml:"destinations"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

func ApplyDefaults(cfg *Config) {
	if cfg.App.Name == "" {
		cfg.App.Name = "hydracast"
	}
	if cfg.App.Timezone == "" {
		cfg.App.Timezone = "UTC"
	}
	if cfg.Storage.Database == "" {
		cfg.Storage.Database = "/data/hydracast.db"
	}
	if cfg.Storage.WorkDir == "" {
		cfg.Storage.WorkDir = "/data/work"
	}
	if cfg.Storage.CacheDir == "" {
		cfg.Storage.CacheDir = "/data/cache"
	}
	if cfg.Secrets.Provider == "" {
		cfg.Secrets.Provider = "openbao"
	}
	if cfg.Secrets.OpenBao.Mount == "" {
		cfg.Secrets.OpenBao.Mount = "kv"
	}
	if cfg.Secrets.OpenBao.TokenFile == "" {
		cfg.Secrets.OpenBao.TokenFile = "/data/openbao-token"
	}
	if cfg.Secrets.OpenBao.AppPath == "" {
		cfg.Secrets.OpenBao.AppPath = "hydracast"
	}
	if cfg.Secrets.OpenBao.Timeout == 0 {
		cfg.Secrets.OpenBao.Timeout = 5 * time.Second
	}
	if cfg.Limits.MaxConcurrentJobs == 0 {
		cfg.Limits.MaxConcurrentJobs = 1
	}
	if cfg.Limits.MaxItemsPerRun == 0 {
		cfg.Limits.MaxItemsPerRun = 3
	}
	if cfg.Limits.MaxWorkingBytes == 0 {
		cfg.Limits.MaxWorkingBytes = 5000 * 1024 * 1024 // 5000MB
	}
	if cfg.Limits.MaxMediaDuration == 0 {
		cfg.Limits.MaxMediaDuration = 4 * time.Hour
	}
	if cfg.Limits.JobEventRetention == 0 {
		cfg.Limits.JobEventRetention = 1000
	}
	if cfg.Downloaders.YtDlp.Binary == "" {
		cfg.Downloaders.YtDlp.Binary = "/usr/local/bin/yt-dlp"
	}
	if cfg.Downloaders.YtDlp.Format == "" {
		cfg.Downloaders.YtDlp.Format = "bv*+ba/b"
	}
}
