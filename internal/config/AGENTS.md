# internal/config — DOX

Parent: root `AGENTS.md`.

## Purpose

YAML config loading, struct definitions, default application, and validation. The single source of truth for the `Config` type used across all packages.

## Ownership

- Package: `config`.
- Files: `config.go` (types, `Load`, `ApplyDefaults`), `validate.go` (`Validate`, `ValidateSecretRefs`), `config_test.go`.

## Local Contracts

```dox
R1 Config_version := 1 (only accepted value).
R2 ApplyDefaults := idempotent; called after every Load in cmd/hydracast.loadConfig.
R3 default_db := /data/hydracast.db. default_work_dir := /data/work. default_cache_dir := /data/cache. (root R216/R217)
R4 default_limits := {max_concurrent_jobs:1, max_items_per_run:3, max_working_bytes:5000MB, max_media_duration:4h, job_event_retention:1000}. (root R270/R274)
R5 secrets_provider ∈ {openbao, files}.
R6 known_source_types := {url_list, facebook_page_videos, youtube_channel, rss_feed, local_directory}. (root R231)
R7 known_transform_types := {ffmpeg}.
R8 known_destination_types := {youtube, facebook_page}.
R9 known_downloader_types := {yt_dlp}.
R10 secret_ref_format := "secret://" prefix (ValidateSecretRefs enforces).
R11 route integrity := every route.Source/Transforms/Destinations must reference existing named config entries.
R12 Validate := side_effect_creates WorkDir ∧ CacheDir via os.MkdirAll; F assume_pure_function.
```

## Work Guidance

- Adding a config field: add to struct + yaml tag, add to `ApplyDefaults` if defaulted, add to validation if constrained.
- Whitelists (`known*Types`) are package vars in `validate.go` — keep in sync with `internal/{source,download,transform,publish}` constructors.
- `Validate` returns `[]error` (not single error) — callers iterate and exit(1) if non-empty.

## Verification

```bash
go vet ./internal/config
go test ./internal/config
```

Tests: `TestLoadConfig`, `TestApplyDefaults`, `TestValidate`, `TestValidateSecretRefs`. Use table-driven style matching existing.
