# internal/download — DOX

Parent: root `AGENTS.md`.

## Purpose

Downloader plugins. Declares `Plugin` interface and `LocalMedia` struct, implements the `yt_dlp` downloader backed by the `yt-dlp` subprocess.

## Ownership

- Package: `download`.
- Files: `downloader.go` (interface + `LocalMedia`), `ytdlp.go` (`YtDlp`).

## Local Contracts

```dox
R1 Plugin interface := {Name()string, Supports(MediaItem)bool, Download(ctx, MediaItem)(*LocalMedia, error)}.
R2 LocalMedia := {Path, Filename, Size, Duration, MimeType}. Produced by Download; consumed by transform ∧ publish.
R3 YtDlp.Download := subprocess(yt-dlp) with args {--no-playlist, --format cfg.Format, --output cfg.OutputTemplate∨default, --no-mtime} + URL.
R4 cookies_ref present -> M materialize_cookies(temp_file, download_duration_only) ∧ M os.Remove(after). (root R224)
R5 output_discovery := scan workDir for filename containing item.ExternalID; failure -> error.
R6 binary missing -> error_before_subprocess; F invoke.
R7 yt-dlp path := cfg.Downloaders.YtDlp.Binary (default /usr/local/bin/yt-dlp).
```

Known gap: `materializeCookies` currently creates an empty temp file and does not resolve `cookies_ref` via the secrets resolver — cookies are not actually passed to yt-dlp yet. Resolve before relying on authenticated downloads.

## Work Guidance

- `YtDlp.Config` is `config.YtDlpConfig` (binary, cookies_ref, output_template, format).
- New downloader = new file implementing `Plugin` + constructor + wire into `internal/app`.
- Output template uses yt-dlp's `%(extractor)s-%(id)s.%(ext)s` form.

## Verification

```bash
go vet ./internal/download
go build ./internal/download
```

No tests (subprocess-bound). Verify with dry-run against a real URL.
