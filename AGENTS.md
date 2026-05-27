# HydraCast Agent Notes

## Project State
- Early stage: design doc only, no source code committed yet.
- README.md is the authoritative architecture spec. Read it first.

## What It Is
- Go CLI app for scheduled video syndication (sources → download → optional ffmpeg transform → publish to YouTube/Facebook).
- Runs as a one-shot scheduled job via systemd timer + Podman/Docker, not a daemon.

## Key Commands (when implemented)
```
hydracast sync --config /data/config.yaml      # primary scheduled run
hydracast validate --config /data/config.yaml  # config check
hydracast scan --config /data/config.yaml      # scan sources only
hydracast sync --dry-run --config /data/config.yaml
hydracast jobs --last 20 --config /data/config.yaml
hydracast jobs --failed --config /data/config.yaml
hydracast retry --failed --config /data/config.yaml
hydracast log --last 100 --config /data/config.yaml
hydracast auth youtube --destination <name> --config /data/config.yaml
hydracast secrets check --config /data/config.yaml
```
All commands support `--json` output where applicable.

## Tech Stack
- **Language**: Go (CGO-free preferred)
- **SQLite driver**: `modernc.org/sqlite` (no CGO)
- **CLI**: Cobra or urfave/cli
- **Config**: YAML with struct validation
- **External deps**: `yt-dlp` (subprocess), `ffmpeg`/`ffprobe` (subprocess)
- **Secrets**: OpenBao preferred, file-mounted fallback at `/data/secrets`
- **Container image**: Go binary + python:3.12-slim + yt-dlp + ffmpeg
- **Go version**: 1.22+ (per Dockerfile)

## Critical Constraints
- **Never** log, store in SQLite, or write resolved secret values to disk.
- **Never** include secret values in dry-run output or job events.
- May log: secret reference path, whether found, redacted fingerprints (sha256:abcd1234...).
- Media files are ephemeral — delete after successful publish by default.
- Use file lock (`/data/hydracast.lock`) to prevent overlapping scheduled runs.
  - If lock active: exit 0. If stale: remove and continue. If cannot acquire: record event, exit 0.
- All runtime data lives on external volume mounted at `/data`.
- Config path: `/data/config.yaml`, DB: `/data/hydracast.db`.

## External Volume Layout (/data)
```
/data
├── config.yaml
├── hydracast.db
├── openbao-token
├── secrets/          # dev-only fallback files
├── cookies/          # facebook.txt (dev fallback)
├── work/             # temp media files
├── cache/
└── logs/
```

## Secrets Management
- Config references secrets symbolically: `secret://openbao/kv/hydracast/youtube/client`
- Resolution order: explicit OpenBao ref → default OpenBao path → file fallback → env fallback
- Fail validation if required production secret cannot be resolved.
- OpenBao token delivery: `/data/openbao-token`, `BAO_TOKEN`, or `VAULT_TOKEN` env var.
- Materialize cookie data only into temp files for download duration, then remove.

## Plugin Architecture (compiled-in, not dynamic)
- Source → Downloader → Transformer → Destination pipeline.
- Initial plugins: `facebook_page_videos`, `youtube_channel`, `rss_feed`, `local_directory` (sources); `yt_dlp` (downloader); `ffmpeg` (transformer); `youtube`, `facebook_page` (destinations).

## State Model (SQLite)
Four tables: `media_items`, `jobs`, `publish_results`, `job_events`.
- `media_items`: source identity, fingerprint, external_id, metadata. UNIQUE(source_name, external_id).
- `jobs`: processing status, attempts, error tracking. FK to media_items.
- `publish_results`: one row per media_item + destination. UNIQUE(media_item_id, destination_name).
- `job_events`: recent operational events with level, component, context_json.
- Never store large media blobs or resolved secret values.

## Job States
Media: new → detected → download_pending → downloading → downloaded → transform_pending → transforming → transformed → publish_pending → publishing → published | failed | retryable_failed | permanent_failed | skipped
Destination: pending → uploading → published | failed | retryable_failed | permanent_failed | auth_required | quota_limited | skipped

## Failure Handling
- Item failure (one video fails): record failed job, process others, exit 0.
- System failure (bad config, DB unavailable): exit nonzero.
- Auth failure: mark destination `auth_required`, exit nonzero if no work can continue.
- Partial failure: exit 0, record failed jobs.
- Retryable: network timeout, HTTP 429, temp platform error, DNS failure, upload interruption.
- Permanent: unsupported media, deleted source, invalid credentials, duplicate policy violation, missing metadata.

## Disk Usage Policy
- Default limits: max_items_per_run=3, max_working_bytes=5000MB, max_media_duration=4h.
- Before run: remove stale temp files, enforce max_working_bytes, remove old cache.
- After success: delete original and transformed copies, keep metadata.
- After failure: default delete media, keep error state. Optional retain for debugging.
- Job event retention: 1000 events.

## Dry Run Behavior
- Load config, validate, scan sources, detect new items, resolve routes.
- Show intended downloads, transforms, publishes.
- Avoid: downloads, uploads, database writes (unless --write-discovery).

## Exit Codes
- `0` = valid / item-level failures (non-blocking)
- `1` = config invalid
- `2` = missing runtime dependency
- `3` = auth/credential issue
- `4` = storage issue

## Repo Layout (planned)
```
cmd/hydracast/main.go
internal/{app,config,source,download,transform,publish,store,media,joblog,lock}/
migrations/
systemd/
Dockerfile
compose.yaml
config.example.yaml
```

## MVP Roadmap
1. **MVP 1**: Config, validation, SQLite, yt-dlp source, download, cleanup, dry run, jobs/log inspection.
2. **MVP 2**: YouTube destination, OAuth setup, upload, retry, publish dry run.
3. **MVP 3**: ffprobe inspection, ffmpeg transform, presets (faststart_mp4, normalize_audio).
4. **MVP 4**: Facebook Page destination, token validation, publish status.
5. **MVP 5**: Multi-source/destination routing, per-route transforms and limits.

## Design Constraints to Honor
- No Kubernetes, Redis, RabbitMQ, Celery, or heavy DB servers.
- No large framework dependencies.
- One-shot execution preferred; daemon mode optional later.
- Conservative disk usage; deterministic cleanup.
- No hardcoded platform behavior.
- No uploading duplicate content without explicit policy.
