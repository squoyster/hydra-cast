package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Bytes is an int64 that unmarshals from human size strings like "5000MB",
// "1.5GB", or a bare integer. Units use binary multipliers (MB = 1024*1024)
// to match disk-size conventions used elsewhere in the codebase.
type Bytes int64

func (b *Bytes) UnmarshalYAML(value *yaml.Node) error {
	// Bare integer.
	var n int64
	if err := value.Decode(&n); err == nil {
		*b = Bytes(n)
		return nil
	}
	var s string
	if err := value.Decode(&s); err != nil {
		return fmt.Errorf("%s: expected integer or size string, got %q", value.Value, value.Value)
	}
	parsed, err := ParseBytes(s)
	if err != nil {
		return fmt.Errorf("%s: %w", value.Value, err)
	}
	*b = Bytes(parsed)
	return nil
}

var byteUnits = map[string]int64{
	"": 1, "B": 1,
	"K": 1024, "KB": 1024, "KIB": 1024,
	"M": 1024 * 1024, "MB": 1024 * 1024, "MIB": 1024 * 1024,
	"G": 1024 * 1024 * 1024, "GB": 1024 * 1024 * 1024, "GIB": 1024 * 1024 * 1024,
	"T": 1024 * 1024 * 1024 * 1024, "TB": 1024 * 1024 * 1024 * 1024, "TIB": 1024 * 1024 * 1024 * 1024,
}

func ParseBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, nil
	}
	i := len(s)
	for i > 0 && !isByteDigitOrDot(s[i-1]) {
		i--
	}
	numStr := s[:i]
	unit := strings.ToUpper(strings.TrimSpace(s[i:]))
	mult, ok := byteUnits[unit]
	if !ok {
		return 0, fmt.Errorf("unknown size unit %q in %q", unit, s)
	}
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", s, err)
	}
	return int64(num * float64(mult)), nil
}

func isByteDigitOrDot(r byte) bool {
	return (r >= '0' && r <= '9') || r == '.'
}

type Config struct {
	Version      int                 `yaml:"version"`
	App          AppConfig           `yaml:"app"`
	Storage      StorageConfig       `yaml:"storage"`
	Secrets      SecretsConfig       `yaml:"secrets"`
	Limits       LimitsConfig        `yaml:"limits"`
	Downloaders  DownloadersConfig   `yaml:"downloaders"`
	Sources      []SourceConfig      `yaml:"sources"`
	Transforms   []TransformConfig   `yaml:"transforms"`
	Destinations []DestinationConfig `yaml:"destinations"`
	Routes       []RouteConfig       `yaml:"routes"`
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
	Provider string         `yaml:"provider"`
	OpenBao  OpenBaoConfig  `yaml:"openbao"`
	Fallback FallbackConfig `yaml:"fallback"`
}

type OpenBaoConfig struct {
	Address     string        `yaml:"address"`
	Namespace   string        `yaml:"namespace"`
	Mount       string        `yaml:"mount"`
	AuthPath    string        `yaml:"auth_path"`
	AppRoleFile string        `yaml:"approle_file"`
	TokenFile   string        `yaml:"token_file"`
	AppPath     string        `yaml:"app_path"`
	Timeout     time.Duration `yaml:"timeout"`
}

type FallbackConfig struct {
	Provider string `yaml:"provider"`
	Enabled  bool   `yaml:"enabled"`
	Root     string `yaml:"root"`
}

type LimitsConfig struct {
	MaxConcurrentJobs   int           `yaml:"max_concurrent_jobs"`
	MaxItemsPerRun      int           `yaml:"max_items_per_run"`
	MaxWorkingBytes     Bytes         `yaml:"max_working_bytes"`
	MaxMediaDuration    time.Duration `yaml:"max_media_duration"`
	KeepSuccessfulMedia bool          `yaml:"keep_successful_media"`
	KeepFailedMedia     bool          `yaml:"keep_failed_media"`
	JobEventRetention   int           `yaml:"job_event_retention"`
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
	Path       string `yaml:"path"`
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
	ClientIDRef     string `yaml:"client_id_ref"`
	ClientSecretRef string `yaml:"client_secret_ref"`
	TokenRef        string `yaml:"token_ref"`
	Privacy         string `yaml:"privacy"`
	CategoryID      string `yaml:"category_id"`
	PageID          string `yaml:"page_id"`
	PageTokenRef    string `yaml:"page_token_ref"`
}

type RouteConfig struct {
	Name         string   `yaml:"name"`
	Source       string   `yaml:"source"`
	Transforms   []string `yaml:"transforms"`
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
	if cfg.Secrets.OpenBao.AuthPath == "" {
		cfg.Secrets.OpenBao.AuthPath = "approle"
	}
	if cfg.Secrets.OpenBao.AppRoleFile == "" {
		cfg.Secrets.OpenBao.AppRoleFile = "/data/auth/role_id_secret_id"
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
		cfg.Limits.MaxWorkingBytes = Bytes(5000 * 1024 * 1024) // 5000MB
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
	for i := range cfg.Sources {
		if cfg.Sources[i].Type == "url_list" && cfg.Sources[i].Path == "" {
			cfg.Sources[i].Path = "/data/reels.json"
		}
	}
}
