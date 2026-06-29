# HydraCast

HydraCast is a compact, container-native media relay for scheduled video-first multimedia distribution.

It watches configured sources, detects new owned media, downloads a temporary local working copy, optionally transforms the media with `ffmpeg`, publishes it to configured destinations such as YouTube and Facebook, records minimal durable state, and cleans up local files to conserve VPS resources.

## Summary

HydraCast is designed for scheduled n-to-n content syndication.

```text
multiple sources → local working copy → optional transform → multiple destinations
```

Initial focus:

- Source media:
  - Video first
  - Audio second
- Initial destinations:
  - YouTube
  - Facebook Page
- Acquisition:
  - `yt-dlp`
- Transformation:
  - `ffmpeg`
- State:
  - Minimal SQLite database
- Secrets:
  - OpenBao as the preferred secrets backend
  - File-mounted secrets as a local/development fallback
- Deployment:
  - Go binary in a container
  - Podman or Docker
  - systemd timer on a VPS
- Configuration:
  - YAML file on an external mounted volume
  - Config validation command
- Operations:
  - Dry runs
  - Last-k job log
  - Retry failed jobs
  - Conservative disk usage

## Project Goals

HydraCast should be:

- Compact
- Reliable
- Container-native
- VPS-friendly
- Plugin-based
- Easy to configure
- Easy to inspect
- Cheap to operate
- Safe to run on a schedule
- Conservative with disk and memory usage

HydraCast is not intended to be a large media management suite. It is a focused scheduled relay for owned content.

## Core Use Case

A typical use case:

```text
1. Monitor a Facebook page for new owned videos.
2. Detect a video that has not been processed.
3. Download the video using yt-dlp.
4. Optionally normalize or repackage it with ffmpeg.
5. Upload it to YouTube.
6. Record the publish result.
7. Delete the temporary media file.
8. Keep only metadata, job status, and remote publish IDs.
```

## Architecture

```text
HydraCast
├── CLI / Runner
├── Scheduler Integration
├── Config Loader + Validator
├── Source Plugins
│   ├── facebook_page_videos
│   ├── youtube_channel
│   ├── rss_feed
│   └── local_directory
├── Downloader Plugins
│   └── yt_dlp
├── Transformer Plugins
│   └── ffmpeg
├── Destination Plugins
│   ├── youtube
│   └── facebook_page
├── Minimal State Store
│   └── SQLite
├── Working File Manager
│   ├── temp files
│   ├── cache files
│   └── cleanup policy
└── Job Log
    ├── recent jobs
    ├── recent events
    └── failed job inspection
```

## Technology Stack

| Layer | Technology |
|---|---|
| Main application | Go |
| CLI framework | Cobra or urfave/cli |
| Configuration | YAML |
| Config validation | JSON Schema-style validation or Go struct validation |
| State database | SQLite |
| SQLite driver | `modernc.org/sqlite` preferred for CGO-free builds |
| Downloads | `yt-dlp` subprocess |
| Media transformation | `ffmpeg` / `ffprobe` subprocesses |
| Container runtime | Podman or Docker |
| Scheduling | systemd timer |
| Deployment target | Small VPS |
| Logging | Structured JSON logs plus SQLite job events |
| Secrets | OpenBao preferred; mounted files under `/data/secrets` for local/development fallback |
| Runtime config volume | `/data` |

## Recommended Runtime Model

HydraCast should initially run as a scheduled one-shot job, not as a long-running daemon.

```text
systemd timer
    ↓
podman/docker run --rm hydracast sync
    ↓
scan sources
    ↓
download new media
    ↓
transform if needed
    ↓
publish to destinations
    ↓
record state
    ↓
cleanup
    ↓
exit
```

Advantages:

- No always-running process required
- Low memory footprint
- Simple VPS deployment
- Easy logging through systemd
- Easy failure isolation
- No Kubernetes, Redis, Celery, or external queue required
- Reboots are naturally handled by systemd timers

A daemon mode can be added later:

```bash
hydracast run --config /data/config.yaml
```

But the initial preferred runtime is:

```bash
hydracast sync --config /data/config.yaml
```

## Data Lifecycle

HydraCast treats media files as ephemeral working data.

```text
1. Scan source
2. Identify new media item
3. Record media fingerprint
4. Download media to working directory
5. Probe media with ffprobe
6. Transform media with ffmpeg if configured
7. Publish to destination endpoints
8. Record remote publish IDs and URLs
9. Delete local media unless retention policy says otherwise
10. Keep minimal metadata and job history
```

Default retention policy:

```text
Successful media: delete
Failed media: delete unless explicitly retained
Metadata: keep
Publish results: keep
Recent job events: keep up to configured limit
```

## Plugin Architecture

HydraCast uses plugin-style interfaces for each major extension point.

The initial implementation can use compiled-in Go plugins/interfaces rather than dynamic shared-object plugins. This keeps the system simpler and container-friendly while preserving architectural modularity.

### Source Plugins

Source plugins discover media items.

Examples:

- `facebook_page_videos`
- `youtube_channel`
- `rss_feed`
- `local_directory`
- `manual_url_list`

Conceptual interface:

```go
type SourcePlugin interface {
    Name() string
    Type() string
    Scan(ctx context.Context, cfg SourceConfig) ([]MediaItem, error)
}
```

### Downloader Plugins

Downloader plugins acquire local working copies.

Initial downloader:

- `yt_dlp`

Conceptual interface:

```go
type DownloaderPlugin interface {
    Name() string
    Supports(item MediaItem) bool
    Download(ctx context.Context, item MediaItem, cfg DownloadConfig) (*LocalMedia, error)
}
```

### Transformer Plugins

Transformer plugins modify local media files.

Initial transformer:

- `ffmpeg`

Useful presets:

- `none`
- `faststart_mp4`
- `normalize_audio`
- `convert_to_mp4`
- `extract_audio`
- `scale_1080p`
- `burn_intro_outro`

Conceptual interface:

```go
type TransformerPlugin interface {
    Name() string
    Transform(ctx context.Context, media LocalMedia, cfg TransformConfig) (*LocalMedia, error)
}
```

### Destination Plugins

Destination plugins publish media to external platforms.

Initial destinations:

- `youtube`
- `facebook_page`

Conceptual interface:

```go
type DestinationPlugin interface {
    Name() string
    Type() string
    Publish(ctx context.Context, item MediaItem, media LocalMedia, cfg DestinationConfig) (*PublishResult, error)
}
```

## Repository Layout

```text
hydracast/
├── cmd/
│   └── hydracast/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── sync.go
│   │   ├── scan.go
│   │   ├── publish.go
│   │   └── retry.go
│   ├── config/
│   │   ├── config.go
│   │   └── validate.go
│   ├── source/
│   │   ├── source.go
│   │   └── facebook_page.go
│   ├── download/
│   │   ├── downloader.go
│   │   └── ytdlp.go
│   ├── transform/
│   │   ├── transformer.go
│   │   └── ffmpeg.go
│   ├── publish/
│   │   ├── publisher.go
│   │   ├── youtube.go
│   │   └── facebook_page.go
│   ├── store/
│   │   ├── store.go
│   │   ├── sqlite.go
│   │   └── migrations.go
│   ├── media/
│   │   ├── probe.go
│   │   ├── checksum.go
│   │   └── cleanup.go
│   ├── joblog/
│   │   └── events.go
│   └── lock/
│       └── flock.go
├── migrations/
│   └── 001_init.sql
├── systemd/
│   ├── hydracast-sync.service
│   └── hydracast-sync.timer
├── Dockerfile
├── compose.yaml
├── config.example.yaml
├── go.mod
└── README.md
```

## External Volume Layout

All runtime data should live on a mounted external volume.

Recommended host path:

```text
/opt/hydracast/data
```

Container path:

```text
/data
```

Volume contents:

```text
/opt/hydracast/data
├── config.yaml
├── hydracast.db
├── auth/
│   └── role_id_secret_id   # AppRole creds (role_id=.., secret_id=..), 0600
├── openbao-token
├── secrets/
│   └── dev-only-fallback-files/
├── cookies/
│   └── facebook.txt
├── work/
├── cache/
└── logs/
```

## Secrets Management

HydraCast should use OpenBao as the preferred secrets backend.

OpenBao stores sensitive values such as:

- YouTube OAuth client credentials
- YouTube refresh/access tokens
- Facebook Page tokens
- Facebook cookie material if used
- Destination API keys
- Per-route webhook secrets
- Future plugin credentials

File-mounted secrets under `/data/secrets` are supported only as a development and bootstrap fallback. Production deployments should resolve secrets from OpenBao.

### Secrets Backend Model

HydraCast supports a configurable secrets provider:

```yaml
secrets:
  provider: openbao

  openbao:
    address: "http://openbao:8200"
    namespace: ""
    mount: "kv"
    auth_path: "approle"
    approle_file: "/data/auth/role_id_secret_id"
    token_file: "/data/openbao-token"
    app_path: "hydracast"
    timeout: 5s

  fallback:
    provider: files
    enabled: true
    root: "/data/secrets"
```

Recommended production behavior:

```text
1. Read OpenBao address and token source.
2. Authenticate to OpenBao.
3. Load required secrets by reference.
4. Validate required secret keys exist.
5. Keep secrets in memory only.
6. Do not write resolved secrets to logs, config, SQLite, or local files.
```

### Secret References

Configuration should reference secrets symbolically instead of embedding sensitive values.

Example destination configuration:

```yaml
destinations:
  - name: dkmc-youtube
    type: youtube
    enabled: true
    client_secret_ref: "secret://openbao/kv/hydracast/youtube/client"
    token_ref: "secret://openbao/kv/hydracast/youtube/token"
    privacy: public
    category_id: "27"

  - name: dkmc-facebook-page
    type: facebook_page
    enabled: false
    page_id: "YOUR_PAGE_ID"
    page_token_ref: "secret://openbao/kv/hydracast/facebook/page-token"
```

Downloader credentials can also be referenced:

```yaml
downloaders:
  yt_dlp:
    binary: /usr/local/bin/yt-dlp
    cookies_ref: "secret://openbao/kv/hydracast/facebook/cookies"
    output_template: "/data/work/%(extractor)s-%(id)s.%(ext)s"
    format: "bv*+ba/b"
```

### Recommended OpenBao Path Layout

Suggested KV layout:

```text
kv/
└── hydracast/
    ├── youtube/
    │   ├── client
    │   └── token
    ├── facebook/
    │   ├── page-token
    │   └── cookies
    └── plugins/
        └── example-plugin/
            └── credentials
```

Example values:

```text
kv/hydracast/youtube/client
  client_id
  client_secret

kv/hydracast/youtube/token
  access_token
  refresh_token
  expiry

kv/hydracast/facebook/page-token
  page_id
  access_token
  expires_at

kv/hydracast/facebook/cookies
  cookies_txt
```

A config ref selects one field within a secret via `#<field_name>`:

```yaml
client_id_ref: "secret://openbao/kv/hydracast/youtube/client#client_id"
client_secret_ref: "secret://openbao/kv/hydracast/youtube/client#client_secret"
```

Omitting `#key` returns the whole secret: multi-field secrets serialize as `key=value` lines (consumed by the client-secret parser historically); single-field secrets return the raw value.

### OpenBao Access From Containers

For a VPS deployment, OpenBao can run on the same host or on a private control-plane network.

Recommended local-container pattern:

```text
hydracast container
    ↓
OpenBao HTTP API
    ↓
file-backed OpenBao storage or other OpenBao storage backend
```

HydraCast should receive only enough OpenBao access to read its own path:

```text
kv/data/hydracast/*
kv/metadata/hydracast/*
```

A minimal policy should allow:

```text
read on kv/data/hydracast/*
list on kv/metadata/hydracast/*
```

HydraCast does not normally need permission to create, update, or delete secrets during scheduled sync runs.

### Token Delivery

HydraCast obtains an OpenBao client token using this precedence:

```text
1. BAO_TOKEN / VAULT_TOKEN env var           (ops override / debugging)
2. AppRole login  (production default)
   POST {address}/v1/auth/{auth_path}/login
   creds file: /data/auth/role_id_secret_id   (role_id=.., secret_id=..)
3. Static token file /data/openbao-token      (last-resort fallback)
```

AppRole is the production default. The obtained client token (24h TTL) is
cached for the duration of a run, so a scheduled sync performs at most one
AppRole login per run. Env vars and the static token file remain honoured for
ops/debugging when AppRole creds are absent.

Provisioning AppRole creds on the host:

```bash
sudo install -d -m 700 /opt/hydracast/data/auth
sudo tee /opt/hydracast/data/auth/role_id_secret_id >/dev/null <<'EOF'
role_id=08ab1365-...
secret_id=4a7e5d7d-...
EOF
sudo chmod 600 /opt/hydracast/data/auth/role_id_secret_id
```

A static token file is still a valid fallback; readable only by the deployment
user (`chmod 600 /opt/hydracast/data/openbao-token`).

### Secret Resolution Rules

HydraCast should resolve secrets at runtime using this order:

```text
1. Explicit OpenBao secret reference
2. Configured default OpenBao path
3. File fallback, if enabled
4. Environment fallback, if enabled
```

HydraCast should fail validation if a required production secret cannot be resolved.

Dry runs should validate secret references by default, but should not print secret values.

### Secret Safety Rules

HydraCast must not:

- Log secret values
- Store secret values in SQLite
- Write resolved secrets into transformed config files
- Include secret values in dry-run output
- Include secret values in job events
- Preserve temporary cookie files after a run unless explicitly configured

HydraCast may log:

- Secret reference path
- Whether a secret was found
- Which required keys were present
- Redacted fingerprints such as `sha256:abcd1234...`


## Configuration

HydraCast uses a YAML configuration file.

Default path:

```text
/data/config.yaml
```

Example:

```yaml
version: 1

app:
  name: hydracast
  timezone: America/Denver

storage:
  database: /data/hydracast.db
  work_dir: /data/work
  cache_dir: /data/cache

secrets:
  provider: openbao
  openbao:
    address: "http://openbao:8200"
    mount: "kv"
    auth_path: "approle"
    approle_file: "/data/auth/role_id_secret_id"
    token_file: "/data/openbao-token"
    app_path: "hydracast"
    timeout: 5s
  fallback:
    provider: files
    enabled: true
    root: "/data/secrets"

limits:
  max_concurrent_jobs: 1
  max_items_per_run: 3
  max_working_bytes: 5000MB
  keep_successful_media: false
  keep_failed_media: false
  job_event_retention: 1000

downloaders:
  yt_dlp:
    binary: /usr/local/bin/yt-dlp
    cookies_ref: "secret://openbao/kv/hydracast/facebook/cookies"
    output_template: "/data/work/%(extractor)s-%(id)s.%(ext)s"
    format: "bv*+ba/b"

sources:
  - name: dkmc-facebook
    type: facebook_page_videos
    url: "https://www.facebook.com/DKMCYoga/videos"
    downloader: yt_dlp
    enabled: true

transforms:
  - name: youtube-default
    type: ffmpeg
    enabled: true
    preset: faststart_mp4
    args:
      - "-movflags"
      - "+faststart"

destinations:
  - name: dkmc-youtube
    type: youtube
    enabled: true
    client_id_ref: "secret://openbao/kv/hydracast/youtube/client#client_id"
    client_secret_ref: "secret://openbao/kv/hydracast/youtube/client#client_secret"
    token_ref: "secret://openbao/kv/hydracast/youtube/token"
    privacy: public
    category_id: "27"

  - name: dkmc-facebook-page
    type: facebook_page
    enabled: false
    page_id: "YOUR_PAGE_ID"
    page_token_ref: "secret://openbao/kv/hydracast/facebook/page-token"

routes:
  - name: facebook-to-youtube
    source: dkmc-facebook
    transforms:
      - youtube-default
    destinations:
      - dkmc-youtube
```

## Configuration Validation

HydraCast should validate configuration before running jobs.

Command:

```bash
hydracast validate --config /data/config.yaml
```

Validation should check:

- YAML syntax
- Config schema version
- Required fields
- Unknown fields
- Source plugin exists
- Downloader plugin exists
- Transformer plugin exists
- Destination plugin exists
- Route references are valid
- OpenBao address is configured when OpenBao provider is enabled
- OpenBao token file or token environment variable is available
- Required OpenBao secret references resolve
- Required keys exist inside each resolved secret
- Secret files exist when file fallback is enabled
- Cookie secrets exist where needed
- Work/cache directories are writable
- `yt-dlp` exists and is executable
- `ffmpeg` exists and is executable
- SQLite database is creatable/openable
- Retention and size limits parse correctly

Example output:

```text
OK config: /data/config.yaml
OK database: /data/hydracast.db
OK source plugin: facebook_page_videos
OK downloader: yt_dlp
OK transform: ffmpeg.faststart_mp4
OK destination plugin: youtube
OK secrets provider: openbao
OK secret ref: secret://openbao/kv/hydracast/youtube/client
OK secret ref: secret://openbao/kv/hydracast/youtube/token
ERROR destination dkmc-facebook-page: secret ref does not resolve: secret://openbao/kv/hydracast/facebook/page-token
```

Suggested exit codes:

| Exit Code | Meaning |
|---:|---|
| `0` | Valid |
| `1` | Config invalid |
| `2` | Runtime dependency missing |
| `3` | Auth or credential issue |
| `4` | Storage issue |

## CLI

Primary commands:

```bash
hydracast validate --config /data/config.yaml
hydracast scan --config /data/config.yaml
hydracast sync --config /data/config.yaml
hydracast retry --failed --config /data/config.yaml
hydracast jobs --last 20 --config /data/config.yaml
hydracast jobs --failed --config /data/config.yaml
hydracast log --last 100 --config /data/config.yaml
hydracast auth youtube --destination dkmc-youtube --config /data/config.yaml
hydracast secrets check --config /data/config.yaml
```

Dry run examples:

```bash
hydracast scan --dry-run --config /data/config.yaml
hydracast sync --dry-run --config /data/config.yaml
hydracast publish --dry-run --item 123 --to dkmc-youtube --config /data/config.yaml
```

Scheduled execution command:

```bash
hydracast sync --config /data/config.yaml
```

## Dry Runs

Dry runs should be non-mutating by default.

A dry run should:

- Load config
- Validate config
- Scan sources
- Detect candidate new items
- Resolve routes
- Show intended downloads
- Show intended transforms
- Show intended publishes
- Avoid downloads
- Avoid uploads
- Avoid database writes unless explicitly requested

Example:

```text
HydraCast dry run

Source: dkmc-facebook
Detected items: 2
New items: 1
Already processed: 1

Planned job:
  item: "Sunday Dharma Talk"
  source: dkmc-facebook
  media_type: video
  download: yt_dlp
  transform: youtube-default
  destinations:
    - dkmc-youtube

No files downloaded.
No destinations published.
No database writes performed.
```

Optional discovery-writing mode:

```bash
hydracast sync --dry-run --write-discovery --config /data/config.yaml
```

## State Model

HydraCast uses SQLite for minimal durable state.

The database records:

- Source media identity
- Fingerprints
- Download status
- Transform status
- Publish status
- Remote destination IDs
- Recent job events
- Failure information

It should not store large media blobs or resolved secret values.

### `media_items`

Tracks discovered source items.

```sql
CREATE TABLE media_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    source_type TEXT NOT NULL,
    external_id TEXT NOT NULL,
    source_url TEXT NOT NULL,
    title TEXT,
    media_type TEXT NOT NULL,
    published_at TEXT,
    detected_at TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    raw_metadata_json TEXT,
    UNIQUE(source_name, external_id)
);
```

### `jobs`

Tracks processing jobs.

```sql
CREATE TABLE jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_item_id INTEGER NOT NULL,
    job_type TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    FOREIGN KEY(media_item_id) REFERENCES media_items(id)
);
```

### `publish_results`

Tracks one publish result per media item and destination.

```sql
CREATE TABLE publish_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_item_id INTEGER NOT NULL,
    destination_name TEXT NOT NULL,
    destination_type TEXT NOT NULL,
    status TEXT NOT NULL,
    remote_id TEXT,
    remote_url TEXT,
    published_at TEXT,
    error_message TEXT,
    UNIQUE(media_item_id, destination_name),
    FOREIGN KEY(media_item_id) REFERENCES media_items(id)
);
```

### `job_events`

Tracks recent operational events.

```sql
CREATE TABLE job_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TEXT NOT NULL,
    job_id INTEGER,
    level TEXT NOT NULL,
    component TEXT NOT NULL,
    message TEXT NOT NULL,
    context_json TEXT
);
```

Recent events query:

```sql
SELECT *
FROM job_events
ORDER BY id DESC
LIMIT ?;
```

## Job States

Recommended media/job states:

```text
new
detected
download_pending
downloading
downloaded
transform_pending
transforming
transformed
publish_pending
publishing
published
failed
retryable_failed
permanent_failed
skipped
```

Destination publish states:

```text
pending
uploading
published
failed
retryable_failed
permanent_failed
auth_required
quota_limited
skipped
```

## Failure Handling

HydraCast should distinguish item-level failures from system-level failures.

| Failure Type | Example | Suggested Exit Behavior |
|---|---|---|
| Item failure | One video fails to upload | Record failed job, process others, usually exit `0` |
| System failure | Bad config, DB unavailable | Exit nonzero |
| Auth failure | YouTube token invalid | Mark destination `auth_required`, exit nonzero if no work can continue |
| Partial failure | 3 jobs succeed, 1 fails | Exit `0`, record failed job |
| Quota failure | YouTube quota exceeded | Mark `quota_limited`, retry later |

Retryable failures:

- Network timeout
- HTTP 429
- Temporary platform error
- Temporary DNS failure
- Upload interruption

Permanent failures:

- Unsupported media
- Deleted source item
- Invalid destination credentials
- Duplicate destination policy violation
- Missing required metadata

## Disk Usage Policy

HydraCast should minimize VPS disk usage.

Default limits:

```yaml
limits:
  max_concurrent_jobs: 1
  max_items_per_run: 3
  max_working_bytes: 5000MB
  max_media_duration: 4h
  keep_successful_media: false
  keep_failed_media: false
  job_event_retention: 1000
```

Cleanup policy:

```text
Before run:
  remove stale temp files
  enforce max_working_bytes
  remove old cache files

After successful publish:
  delete downloaded original
  delete transformed copy
  keep metadata and publish result

After failure:
  default: delete media, keep error state
  optional: retain failed media for debugging
```

## Locking

Scheduled jobs must not overlap.

HydraCast should use a lock file or SQLite-based lock.

Recommended lock path:

```text
/data/hydracast.lock
```

Behavior:

```text
If lock is active:
  exit 0

If lock is stale:
  remove stale lock and continue

If lock cannot be acquired:
  record event and exit 0
```

## Logging and Job Inspection

HydraCast should support last-k job and event inspection.

Commands:

```bash
hydracast jobs --last 20 --config /data/config.yaml
hydracast jobs --failed --config /data/config.yaml
hydracast log --last 100 --config /data/config.yaml
```

Example output:

```text
ID   STATUS     SOURCE          DESTINATION     TITLE                  FINISHED
184  published  dkmc-facebook   dkmc-youtube    Sunday Dharma Talk      2026-05-24 09:41
183  skipped    dkmc-facebook   dkmc-youtube    Friday Meditation       2026-05-23 21:10
182  failed     dkmc-facebook   dkmc-youtube    Morning Practice        2026-05-22 07:15
```

JSON output should be available:

```bash
hydracast jobs --last 20 --json --config /data/config.yaml
```

Structured log example:

```json
{
  "ts": "2026-05-24T09:42:00-06:00",
  "level": "info",
  "component": "publisher.youtube",
  "media_item_id": 123,
  "message": "published video",
  "remote_id": "abc123"
}
```

## Container Image

Preferred runtime image:

```text
Go binary + Python slim + yt-dlp + ffmpeg
```

Reason:

- Go keeps HydraCast compact.
- `yt-dlp` changes often.
- Installing `yt-dlp` through Python/pip is usually fresher than distro packages.
- `ffmpeg` is available through the base OS package manager.

Example Dockerfile:

```dockerfile
FROM golang:1.22-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/hydracast \
    ./cmd/hydracast

FROM python:3.12-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    ffmpeg \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir yt-dlp

COPY --from=build /out/hydracast /usr/local/bin/hydracast

VOLUME ["/data"]

ENTRYPOINT ["hydracast"]
CMD ["sync", "--config", "/data/config.yaml"]
```

## Podman Deployment

Recommended host location:

```bash
sudo mkdir -p /opt/hydracast/data
sudo mkdir -p /opt/hydracast/data/{secrets,cookies,work,cache,logs}
```

Manual validation:

```bash
podman run --rm \
  --network hydracast-net \
  -v /opt/hydracast/data:/data:Z \
  ghcr.io/squoyster/hydracast:latest \
  validate --config /data/config.yaml
```

Manual dry run:

```bash
podman run --rm \
  --network hydracast-net \
  -v /opt/hydracast/data:/data:Z \
  ghcr.io/squoyster/hydracast:latest \
  sync --dry-run --config /data/config.yaml
```

Manual sync:

```bash
podman run --rm \
  --network hydracast-net \
  -v /opt/hydracast/data:/data:Z \
  ghcr.io/squoyster/hydracast:latest \
  sync --config /data/config.yaml
```

## systemd Timer Deployment

### `/etc/systemd/system/hydracast-sync.service`

```ini
[Unit]
Description=HydraCast scheduled sync
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
WorkingDirectory=/opt/hydracast
ExecStart=/usr/bin/podman run --rm \
  --name hydracast-sync \
  --network hydracast-net \
  -v /opt/hydracast/data:/data:Z \
  ghcr.io/squoyster/hydracast:latest \
  sync --config /data/config.yaml
```

### `/etc/systemd/system/hydracast-sync.timer`

```ini
[Unit]
Description=Run HydraCast scheduled sync

[Timer]
OnBootSec=2min
OnUnitActiveSec=10min
Persistent=true

[Install]
WantedBy=timers.target
```

Enable:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now hydracast-sync.timer
```

Check timer:

```bash
systemctl list-timers | grep hydracast
```

Check logs:

```bash
journalctl -u hydracast-sync.service -n 100 --no-pager
```

## Docker Compose Alternative

```yaml
services:
  hydracast:
    image: ghcr.io/squoyster/hydracast:latest
    volumes:
      - ./data:/data
    environment:
      - TZ=America/Denver
      - BAO_ADDR=http://openbao:8200
    networks:
      - hydracast

networks:
  hydracast:
    external: true
```

Run manually:

```bash
docker compose run --rm hydracast validate --config /data/config.yaml
docker compose run --rm hydracast sync --dry-run --config /data/config.yaml
docker compose run --rm hydracast sync --config /data/config.yaml
```

Cron example:

```cron
*/10 * * * * cd /opt/hydracast && docker compose run --rm hydracast sync --config /data/config.yaml
```

systemd timers are preferred over cron because they provide cleaner logging, dependency ordering, and missed-run handling.

## Authentication

### Facebook Source

Initial Facebook source acquisition should use `yt-dlp`.

For private or restricted content, `yt-dlp` may require browser cookie material. In production, store this in OpenBao:

```text
kv/hydracast/facebook/cookies
```

HydraCast should materialize cookie data only into a temporary file for the duration of the download, then remove it. File-mounted cookies under `/data/cookies/facebook.txt` are acceptable only as a local/development fallback.

HydraCast should treat Facebook extraction as a downloader concern rather than implementing Facebook scraping internally.

### YouTube Destination

YouTube publishing should use the official YouTube Data API with OAuth.

Expected OpenBao paths:

```text
kv/hydracast/youtube/client
kv/hydracast/youtube/token
```

Initial auth command:

```bash
hydracast auth youtube --destination dkmc-youtube --config /data/config.yaml
```

Scheduled runs should use the stored refresh token.

### Facebook Page Destination

Facebook Page publishing should use a Page token stored in OpenBao:

```text
kv/hydracast/facebook/page-token
```

The destination plugin should validate:

- Page ID
- Token file existence
- Token permissions
- Expiration if available
- Publish capability

## MVP Roadmap

### MVP 1: Discovery, Download, State, Cleanup

Required:

- YAML config
- Config validation
- SQLite state
- Facebook/video source through `yt-dlp`
- New-item detection
- Download to working directory
- Minimal metadata capture
- Cleanup after success
- Dry run
- `jobs --last k`
- `log --last k`

No upload required yet.

### MVP 2: YouTube Publishing

Required:

- YouTube destination plugin
- OAuth setup command
- Upload video
- Record remote ID and URL
- Retry failed uploads
- Publish dry run

### MVP 3: ffmpeg Transformation

Required:

- `ffprobe` metadata inspection
- `ffmpeg` transform plugin
- Preset support
- `faststart_mp4`
- Audio normalization
- Optional audio extraction

### MVP 4: Facebook Page Publishing

Required:

- Facebook Page destination plugin
- Page token validation
- Upload/publish support
- Publish status tracking

### MVP 5: Multi-Source / Multi-Destination Routing

Required:

- Multiple source configs
- Multiple destination configs
- Route-level transforms
- Route-level enable/disable
- Per-route limits

## Design Constraints

HydraCast should avoid:

- Kubernetes
- Redis
- RabbitMQ
- Celery
- Heavy database servers
- Permanent local media archive by default
- Long-running daemon as the only execution mode
- Large framework dependencies
- Hardcoded platform behavior
- Uploading duplicate content without an explicit policy

HydraCast should prefer:

- One-shot scheduled execution
- SQLite
- External volume config
- OpenBao-backed secrets
- Small Go codebase
- Clear plugin interfaces
- Explicit job state
- Explicit dry runs
- Deterministic cleanup
- Good logs
- Simple failure recovery

## Final Architecture Statement

HydraCast is a scheduled, container-native Go application for video-first multimedia syndication. It uses pluggable source, downloader, transformer, and destination adapters; `yt-dlp` for acquisition; `ffmpeg` for basic transformation; SQLite for minimal durable state; OpenBao for secrets management; and an external-volume YAML configuration with validation, dry-run execution, conservative cleanup, retry support, and recent-job reporting.
