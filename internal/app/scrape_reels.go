package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/squoyster/hydracast/internal/config"
	"github.com/squoyster/hydracast/internal/joblog"
	"github.com/squoyster/hydracast/internal/secrets"
	"github.com/squoyster/hydracast/internal/source"
	"github.com/squoyster/hydracast/internal/store"
)

// ponytail: FB serves a login wall to default Go http client; one browser UA bypasses it for public pages.
// If FB starts enforcing auth, this command is void — no point making it pluggable now.
var reelURLRe = regexp.MustCompile(`https?://(?:www\.)?facebook\.com/[^"'\s]*?/reel/\d+[^"'\s]*`)

type ScrapeReelsOptions struct {
	URL        string
	SourceName string
}

func RunScrapeReels(ctx context.Context, cfg *config.Config, db *store.Store, resolver *secrets.Resolver, opts ScrapeReelsOptions, logger *joblog.Logger, dryRun bool) error {
	component := logger.WithComponent("scrape-reels")

	if !routeExists(cfg, opts.SourceName) {
		return fmt.Errorf("no route configured for source %q (add a route with destinations to publish scraped reels)", opts.SourceName)
	}

	body, err := fetchPage(ctx, opts.URL)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", opts.URL, err)
	}

	reelURLs := extractReelURLs(body, opts.URL)
	if len(reelURLs) == 0 {
		component.Warn("no reel URLs found", "url", opts.URL)
		return nil
	}

	component.Info("found reels", "count", len(reelURLs))
	for _, u := range reelURLs {
		fmt.Println(u)
	}

	if dryRun {
		fmt.Println("\nNo downloads performed (dry-run).")
		return nil
	}

	processed := 0
	for _, u := range reelURLs {
		if processed >= cfg.Limits.MaxItemsPerRun {
			component.Info("max items per run reached", "limit", cfg.Limits.MaxItemsPerRun)
			break
		}

		item := source.MediaItem{
			SourceName:  opts.SourceName,
			SourceType:  "facebook_reels_unauth",
			ExternalID:  reelExternalID(u),
			SourceURL:   u,
			Title:       fmt.Sprintf("reel %s", reelExternalID(u)),
			MediaType:   "video",
			DetectedAt:  time.Now(),
			Fingerprint: "pending",
		}

		if db != nil {
			id, err := db.UpsertMediaItem(ctx, item.SourceName, item.SourceType, item.ExternalID, item.SourceURL, item.Title, item.MediaType, item.Fingerprint, "", nil)
			if err != nil {
				component.Warn("failed to upsert media item", "url", u, "error", err)
				continue
			}
			item.ID = id
		}

		if err := ProcessItem(ctx, cfg, db, resolver, item, logger); err != nil {
			component.Error("item failed", "url", u, "error", err)
			continue
		}
		processed++
	}

	return nil
}

func routeExists(cfg *config.Config, sourceName string) bool {
	for _, r := range cfg.Routes {
		if r.Source == sourceName && len(r.Destinations) > 0 {
			return true
		}
	}
	return false
}

func fetchPage(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func extractReelURLs(body, baseURL string) []string {
	_ = baseURL // ponytail: kept for future og:url-based reconstruction; not needed today.
	seen := make(map[string]struct{})
	var out []string

	for _, m := range reelURLRe.FindAllString(body, -1) {
		clean := strings.SplitN(m, "?", 2)[0]
		clean = strings.TrimRight(clean, "/\"'")
		if _, ok := seen[clean]; !ok {
			seen[clean] = struct{}{}
			out = append(out, clean)
		}
	}

	return out
}

func reelExternalID(url string) string {
	// ponytail: pull the numeric ID out of /reel/<id>/ for dedup; everything else is decoration.
	url = strings.TrimRight(url, "/")
	if i := strings.LastIndex(url, "/"); i != -1 {
		return url[i+1:]
	}
	return url
}
