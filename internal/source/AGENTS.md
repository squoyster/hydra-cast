# internal/source — DOX

Parent: root `AGENTS.md`.

## Purpose

Source plugin contract. Defines the `MediaItem` domain record and the `Plugin` interface implemented by source scanners.

## Ownership

- Package: `source`.
- Files: `source.go` (interface + `MediaItem` struct).

## Local Contracts

```dox
R1 Plugin interface := {Name()string, Type()string, Scan(ctx)([]MediaItem, error)}.
R2 MediaItem := {SourceName, SourceType, ExternalID, SourceURL, Title, MediaType, PublishedAt*, DetectedAt, Fingerprint, RawMetadata}.
R3 ExternalID := stable_unique_key_per_source; combined with SourceName forms the dedup key (see store UNIQUE(source_name, external_id), root R241).
R4 Fingerprint := opaque content hash string (e.g. "sha256:..."); see internal/media.Fingerprint for file-based impl.
```

## Work Guidance

- No concrete plugins implemented yet — `internal/app.scanSources` emits placeholder items. This package is the extension point for root R153 plugins.
- `MediaItem` is shared across `download`/`transform`/`publish` Plugin signatures; changing its shape ripples widely — run impact analysis first.

## Verification

```bash
go vet ./internal/source
```

No tests (pure type declarations).
