package config

import (
	"fmt"
	"os"
	"strings"
)

var knownSourceTypes = map[string]bool{
	"facebook_page_videos": true,
	"youtube_channel":      true,
	"rss_feed":             true,
	"local_directory":      true,
}

var knownDownloaderTypes = map[string]bool{
	"yt_dlp": true,
}

var knownTransformTypes = map[string]bool{
	"ffmpeg": true,
}

var knownDestinationTypes = map[string]bool{
	"youtube":       true,
	"facebook_page": true,
}

func Validate(cfg *Config) []error {
	var errs []error

	if cfg.Version != 1 {
		errs = append(errs, fmt.Errorf("unsupported config version: %d (expected 1)", cfg.Version))
	}

	if cfg.Secrets.Provider != "openbao" && cfg.Secrets.Provider != "files" {
		errs = append(errs, fmt.Errorf("unknown secrets provider: %s", cfg.Secrets.Provider))
	}

	if cfg.Secrets.Provider == "openbao" && cfg.Secrets.OpenBao.Address == "" {
		errs = append(errs, fmt.Errorf("openbao address is required when provider is openbao"))
	}

	for _, src := range cfg.Sources {
		if !src.Enabled {
			continue
		}
		if !knownSourceTypes[src.Type] {
			errs = append(errs, fmt.Errorf("unknown source type: %s", src.Type))
		}
		if !knownDownloaderTypes[src.Downloader] {
			errs = append(errs, fmt.Errorf("unknown downloader: %s (source: %s)", src.Downloader, src.Name))
		}
	}

	for _, t := range cfg.Transforms {
		if !t.Enabled {
			continue
		}
		if !knownTransformTypes[t.Type] {
			errs = append(errs, fmt.Errorf("unknown transform type: %s", t.Type))
		}
	}

	for _, dst := range cfg.Destinations {
		if !dst.Enabled {
			continue
		}
		if !knownDestinationTypes[dst.Type] {
			errs = append(errs, fmt.Errorf("unknown destination type: %s", dst.Type))
		}
	}

	sourceNames := make(map[string]bool)
	for _, src := range cfg.Sources {
		sourceNames[src.Name] = true
	}

	transformNames := make(map[string]bool)
	for _, t := range cfg.Transforms {
		transformNames[t.Name] = true
	}

	destinationNames := make(map[string]bool)
	for _, dst := range cfg.Destinations {
		destinationNames[dst.Name] = true
	}

	for _, route := range cfg.Routes {
		if !sourceNames[route.Source] {
			errs = append(errs, fmt.Errorf("route %q references unknown source: %s", route.Name, route.Source))
		}
		for _, tName := range route.Transforms {
			if !transformNames[tName] {
				errs = append(errs, fmt.Errorf("route %q references unknown transform: %s", route.Name, tName))
			}
		}
		for _, dName := range route.Destinations {
			if !destinationNames[dName] {
				errs = append(errs, fmt.Errorf("route %q references unknown destination: %s", route.Name, dName))
			}
		}
	}

	if cfg.Storage.WorkDir != "" {
		if err := os.MkdirAll(cfg.Storage.WorkDir, 0755); err != nil {
			errs = append(errs, fmt.Errorf("cannot create work dir %s: %w", cfg.Storage.WorkDir, err))
		}
	}

	if cfg.Storage.CacheDir != "" {
		if err := os.MkdirAll(cfg.Storage.CacheDir, 0755); err != nil {
			errs = append(errs, fmt.Errorf("cannot create cache dir %s: %w", cfg.Storage.CacheDir, err))
		}
	}

	return errs
}

func ValidateSecretRefs(cfg *Config) []error {
	var errs []error

	var refs []string
	for _, src := range cfg.Sources {
		if src.Downloader == "yt_dlp" && cfg.Downloaders.YtDlp.CookiesRef != "" {
			refs = append(refs, cfg.Downloaders.YtDlp.CookiesRef)
		}
	}

	for _, dst := range cfg.Destinations {
		if dst.ClientIDRef != "" {
			refs = append(refs, dst.ClientIDRef)
		}
		if dst.ClientSecretRef != "" {
			refs = append(refs, dst.ClientSecretRef)
		}
		if dst.TokenRef != "" {
			refs = append(refs, dst.TokenRef)
		}
		if dst.PageTokenRef != "" {
			refs = append(refs, dst.PageTokenRef)
		}
	}

	for _, ref := range refs {
		if !strings.HasPrefix(ref, "secret://") {
			errs = append(errs, fmt.Errorf("invalid secret ref format: %s", ref))
		}
	}

	return errs
}
