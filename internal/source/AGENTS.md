# internal/source — DOX

Parent: root `AGENTS.md`.

## Purpose

Source plugin contract. Defines the `MediaItem` domain record and the `Plugin` interface implemented by source scanners.

## Ownership

- Package: `source`.
- Files: `source.go` (interface + `MediaItem` struct), `urllist.go` (`URLList`, `NewURLList`).

## Local Contracts

```dox
R1 Plugin interface := {Name()string, Type()string, Scan(ctx)([]MediaItem, error)}.
R2 MediaItem := {ID int64 (DB-assigned; 0 until upserted), SourceName, SourceType, ExternalID, SourceURL, Title, MediaType, PublishedAt*, DetectedAt, Fingerprint, RawMetadata}.
R3 ExternalID := stable_unique_key_per_source; combined with SourceName forms the dedup key (see store UNIQUE(source_name, external_id), root R241).
R4 Fingerprint := opaque content hash string (e.g. "sha256:..."); see internal/media.Fingerprint for file-based impl.
R5 url_list plugin := reads a reels.json intake from SourceConfig.Path (default /data/reels.json) produced by the hydracast-mac-playwright collector. Schema: {items:[{url, external_id, url_sha256, first_seen_in_run_at}]}. ExternalID := item.external_id (fallback: url) — dedup key with SourceName (root R241). Fingerprint := item.url_sha256 (fallback: "pending"). DetectedAt := parsed first_seen_in_run_at (fallback: now). Scan is NON-destructive (pure read); the app layer drains the file after items are upserted, gated on !dryRun ∧ items>0. Missing file := idle (nil, nil).
```

## Work Guidance

- `url_list` (`urllist.go`) is the first real plugin (consumed by `app.scanSources`); this package remains the extension point for the remaining root R231 source plugins (facebook_page_videos, youtube_channel, rss_feed, local_directory).
- `MediaItem` is shared across `download`/`transform`/`publish` Plugin signatures; changing its shape ripples widely — run impact analysis first.

## Verification

```bash
go test ./internal/source
```

`urllist_test.go` covers the url_list plugin (reels schema parse, missing/empty/malformed).
