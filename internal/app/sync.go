package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/squoyster/hydracast/internal/config"
	"github.com/squoyster/hydracast/internal/download"
	"github.com/squoyster/hydracast/internal/joblog"
	"github.com/squoyster/hydracast/internal/media"
	"github.com/squoyster/hydracast/internal/publish"
	"github.com/squoyster/hydracast/internal/secrets"
	"github.com/squoyster/hydracast/internal/source"
	"github.com/squoyster/hydracast/internal/store"
	"github.com/squoyster/hydracast/internal/transform"
)

func RunSync(ctx context.Context, cfg *config.Config, db *store.Store, resolver *secrets.Resolver, logger *joblog.Logger, dryRun bool) error {
	component := logger.WithComponent("sync")

	if dryRun {
		component.Info("starting dry run")
	} else {
		component.Info("starting sync")
	}

	if err := media.CleanupStaleFiles(cfg.Storage.WorkDir, 24*time.Hour); err != nil {
		component.Warn("failed to cleanup stale files", "error", err)
	}

	if err := media.EnforceMaxWorkingBytes(cfg.Storage.WorkDir, int64(cfg.Limits.MaxWorkingBytes)); err != nil {
		component.Warn("failed to enforce max working bytes", "error", err)
	}

	items, err := scanSources(ctx, cfg, db, resolver, logger, dryRun)
	if err != nil {
		return fmt.Errorf("scan sources: %w", err)
	}

	component.Info("detected items", "count", len(items))

	if dryRun {
		return showDryRunPlan(cfg, items)
	}

	// The DB is the durable work queue: scans upsert new items, this drains the
	// oldest never-attempted ones up to MaxItemsPerRun. Failed items keep their
	// job (excluded here) and are owned by `retry --failed`. ponytail: no status
	// column needed — "pending" := no job row exists for the media item.
	pending, err := db.ListPendingItems(ctx, cfg.Limits.MaxItemsPerRun)
	if err != nil {
		return fmt.Errorf("list pending items: %w", err)
	}
	component.Info("processing pending items", "count", len(pending))

	for _, item := range pending {
		if err := ProcessItem(ctx, cfg, db, item, logger); err != nil {
			component.Error("item failed", "item", item.Title, "error", err)
		}
	}

	if err := db.PruneEvents(ctx, cfg.Limits.JobEventRetention); err != nil {
		component.Warn("failed to prune events", "error", err)
	}

	component.Info("sync complete")
	return nil
}

func RunScan(ctx context.Context, cfg *config.Config, db *store.Store, resolver *secrets.Resolver, logger *joblog.Logger, dryRun bool) error {
	component := logger.WithComponent("scan")

	items, err := scanSources(ctx, cfg, db, resolver, logger, dryRun)
	if err != nil {
		return fmt.Errorf("scan sources: %w", err)
	}

	component.Info("detected items", "count", len(items))

	for _, item := range items {
		fmt.Printf("  %s [%s] %s\n", item.Title, item.SourceName, item.ExternalID)
	}

	return nil
}

func scanSources(ctx context.Context, cfg *config.Config, db *store.Store, resolver *secrets.Resolver, logger *joblog.Logger, dryRun bool) ([]source.MediaItem, error) {
	var allItems []source.MediaItem

	for _, srcCfg := range cfg.Sources {
		if !srcCfg.Enabled {
			continue
		}

		logger.Info("scanning source", "source", srcCfg.Name, "type", srcCfg.Type)

		items, err := scanSource(ctx, srcCfg)
		if err != nil {
			logger.Warn("source scan failed", "source", srcCfg.Name, "error", err)
			continue
		}

		for _, item := range items {
			if db != nil {
				id, err := db.UpsertMediaItem(ctx, item.SourceName, item.SourceType, item.ExternalID, item.SourceURL, item.Title, item.MediaType, item.Fingerprint, "", nil)
				if err != nil {
					logger.Warn("failed to upsert media item", "title", item.Title, "error", err)
					continue
				}
				item.ID = id
			}
			allItems = append(allItems, item)
		}

		// Drain a url_list intake file once its items are consumed.
		// ponytail: items are durable in media_items; DB dedup (root R241) makes a
		// re-scan idempotent; failed items survive as job state for `retry --failed`.
		// Gated on !dryRun (root R281: dry-run is side-effect-free).
		if srcCfg.Type == "url_list" && !dryRun && len(items) > 0 && srcCfg.Path != "" {
			if err := os.Remove(srcCfg.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
				logger.Warn("failed to remove intake file", "path", srcCfg.Path, "error", err)
			}
		}
	}

	return allItems, nil
}

// scanSource resolves a single source config to detected items.
func scanSource(ctx context.Context, srcCfg config.SourceConfig) ([]source.MediaItem, error) {
	switch srcCfg.Type {
	case "url_list":
		return source.NewURLList(srcCfg.Name, srcCfg.Path).Scan(ctx)
	default:
		// Other source types are not yet implemented; emit a placeholder so the
		// downstream pipeline stays wired for integration testing.
		return []source.MediaItem{
			{
				SourceName:  srcCfg.Name,
				SourceType:  srcCfg.Type,
				ExternalID:  "example-001",
				SourceURL:   srcCfg.URL,
				Title:       "Example Video",
				MediaType:   "video",
				DetectedAt:  time.Now(),
				Fingerprint: "pending",
			},
		}, nil
	}
}

func ProcessItem(ctx context.Context, cfg *config.Config, db *store.Store, item source.MediaItem, logger *joblog.Logger) error {
	component := logger.WithComponent("sync")

	component.Info("processing item", "title", item.Title)

	jobID, err := db.CreateJob(ctx, item.ID, "sync", "download_pending")
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	_ = db.RecordEvent(ctx, &jobID, "info", "sync", "processing item", fmt.Sprintf(`{"title":%q}`, item.Title))

	dl := download.NewYtDlp(cfg.Downloaders.YtDlp, cfg.Storage.WorkDir)

	localMedia, err := dl.Download(ctx, item)
	if err != nil {
		_ = db.UpdateJobStatus(ctx, jobID, "failed", err.Error())
		_ = db.RecordEvent(ctx, &jobID, "error", "sync.download", "download failed", fmt.Sprintf(`{"error":%q}`, err.Error()))
		return fmt.Errorf("download: %w", err)
	}

	component.Info("downloaded", "path", localMedia.Path, "size", localMedia.Size)
	_ = db.UpdateJobStatus(ctx, jobID, "downloaded", "")

	transforms := resolveTransforms(cfg, item.SourceName)
	for _, tCfg := range transforms {
		component.Info("transforming", "preset", tCfg.Preset)
		_ = db.RecordEvent(ctx, &jobID, "info", "sync.transform", "transforming media", fmt.Sprintf(`{"preset":%q}`, tCfg.Preset))

		ffmpeg := transform.NewFFmpeg("")
		transformedMedia, err := ffmpeg.Transform(ctx, localMedia, tCfg)
		if err != nil {
			_ = db.UpdateJobStatus(ctx, jobID, "failed", err.Error())
			_ = db.RecordEvent(ctx, &jobID, "error", "sync.transform", "transform failed", fmt.Sprintf(`{"error":%q}`, err.Error()))
			_ = media.DeleteMedia(localMedia.Path)
			return fmt.Errorf("transform: %w", err)
		}

		_ = media.DeleteMedia(localMedia.Path)
		localMedia = transformedMedia
		component.Info("transformed", "path", localMedia.Path, "size", localMedia.Size)
	}

	destinations := resolveDestinations(cfg, item.SourceName)
	for _, dstCfg := range destinations {
		component.Info("publishing", "destination", dstCfg.Name, "type", dstCfg.Type)
		_ = db.RecordEvent(ctx, &jobID, "info", "sync.publish", "publishing to destination", fmt.Sprintf(`{"destination":%q}`, dstCfg.Name))

		var pub publish.Plugin
		switch dstCfg.Type {
		case "youtube":
			pub = publish.NewYouTube(dstCfg, cfg.Downloaders.YtDlp.Binary)
		case "facebook_page":
			pub = publish.NewFacebookPage(dstCfg, cfg.Downloaders.YtDlp.Binary)
		default:
			component.Warn("unknown destination type, skipping", "type", dstCfg.Type)
			continue
		}

		result, err := pub.Publish(ctx, item, localMedia)
		if err != nil {
			_ = db.RecordEvent(ctx, &jobID, "error", "sync.publish", "publish failed", fmt.Sprintf(`{"destination":%q,"error":%q}`, dstCfg.Name, err.Error()))
			component.Error("publish failed", "destination", dstCfg.Name, "error", err)
			continue
		}

		if result.Error != nil {
			_ = db.RecordEvent(ctx, &jobID, "error", "sync.publish", "publish failed", fmt.Sprintf(`{"destination":%q,"error":%q}`, dstCfg.Name, result.Error.Error()))
			component.Error("publish failed", "destination", dstCfg.Name, "error", result.Error)
			continue
		}

		component.Info("published", "destination", dstCfg.Name, "remote_id", result.RemoteID, "url", result.RemoteURL)
		_ = db.RecordEvent(ctx, &jobID, "info", "sync.publish", "published successfully", fmt.Sprintf(`{"destination":%q,"remote_id":%q,"url":%q}`, dstCfg.Name, result.RemoteID, result.RemoteURL))
	}

	_ = db.UpdateJobStatus(ctx, jobID, "published", "")

	if !cfg.Limits.KeepSuccessfulMedia {
		component.Info("cleaning up media", "path", localMedia.Path)
		if err := media.DeleteMedia(localMedia.Path); err != nil {
			component.Warn("failed to delete media", "path", localMedia.Path, "error", err)
		}
	}

	return nil
}

func resolveTransforms(cfg *config.Config, sourceName string) []config.TransformConfig {
	var transforms []config.TransformConfig

	for _, route := range cfg.Routes {
		if route.Source == sourceName {
			for _, tName := range route.Transforms {
				for _, t := range cfg.Transforms {
					if t.Name == tName && t.Enabled {
						transforms = append(transforms, t)
					}
				}
			}
		}
	}

	return transforms
}

func resolveDestinations(cfg *config.Config, sourceName string) []config.DestinationConfig {
	var destinations []config.DestinationConfig

	for _, route := range cfg.Routes {
		if route.Source == sourceName {
			for _, dName := range route.Destinations {
				for _, dst := range cfg.Destinations {
					if dst.Name == dName && dst.Enabled {
						destinations = append(destinations, dst)
					}
				}
			}
		}
	}

	return destinations
}

func showDryRunPlan(cfg *config.Config, items []source.MediaItem) error {
	fmt.Println("HydraCast dry run")
	fmt.Println()

	for _, srcCfg := range cfg.Sources {
		if !srcCfg.Enabled {
			continue
		}

		srcItems := 0
		for _, item := range items {
			if item.SourceName == srcCfg.Name {
				srcItems++
			}
		}

		fmt.Printf("Source: %s\n", srcCfg.Name)
		fmt.Printf("Detected items: %d\n", srcItems)
		fmt.Println()

		for _, item := range items {
			if item.SourceName != srcCfg.Name {
				continue
			}

			fmt.Printf("Planned job:\n")
			fmt.Printf("  item: %q\n", item.Title)
			fmt.Printf("  source: %s\n", item.SourceName)
			fmt.Printf("  media_type: %s\n", item.MediaType)
			fmt.Printf("  download: %s\n", srcCfg.Downloader)

			for _, route := range cfg.Routes {
				if route.Source == srcCfg.Name {
					fmt.Printf("  destinations:\n")
					for _, d := range route.Destinations {
						fmt.Printf("    - %s\n", d)
					}
				}
			}
			fmt.Println()
		}
	}

	fmt.Println("No files downloaded.")
	fmt.Println("No destinations published.")
	fmt.Println("No database writes performed.")

	return nil
}
