# internal/publish — DOX

Parent: root `AGENTS.md`.

## Purpose

Destination plugins. Declares `Plugin` interface + `PublishResult`, implements `youtube` and `facebook_page` uploaders. Both wrap the `yt-dlp` subprocess as their upload mechanism.

## Ownership

- Package: `publish`.
- Files: `publisher.go` (interface + `PublishResult`), `youtube.go` (`YouTube`), `facebook_page.go` (`FacebookPage`).

## Local Contracts

```dox
R1 Plugin interface := {Name()string, Type()string, Publish(ctx, MediaItem, *LocalMedia)(*PublishResult, error)}.
R2 PublishResult := {RemoteID, RemoteURL, Status, Error}. Status ∈ {published, failed}.
R3 publish returns (result, nil) even on platform failure — Error is set inside PublishResult. Caller distinguishes transport error (err!=nil) from platform error (result.Error!=nil).
R4 YouTube.Type == "youtube". FacebookPage.Type == "facebook_page".
R5 upload_mechanism := yt-dlp subprocess (not native HTTP API). Both destinations depend on cfg.Downloaders.YtDlp.Binary path.
R6 YouTube metadata := --title (item.Title ∨ filename), --description (templated), optional --metadata-from-title for privacy ∧ category.
R7 FacebookPage requires cfg.PageID; missing -> PublishResult{Status:failed}.
R8 RemoteID extraction := regex-ish string scan over yt-dlp stdout (extractVideoID / extractFacebookVideoID); len==11 check for youtube.
R9 F upload_duplicate_content without explicit_policy. (root R315)
```

Known gaps:
- OAuth tokens/cookies not yet wired into the yt-dlp upload invocation — uploads will fail unauthenticated. `SetupYouTubeAuth` (internal/app) writes a token file but the upload path here does not consume it.
- `publish_results` table not written by app layer (root R243 defines the schema; persistence pending).

## Work Guidance

- Add new destination: new file, struct holding `config.DestinationConfig`, `Type()` returns the `knownDestinationTypes` key, `Publish` returns `*PublishResult`.
- Always prefer returning `PublishResult{Error: ...}` over `error` for platform-side failures so the caller logs and continues.

## Verification

```bash
go vet ./internal/publish
go build ./internal/publish
```

No tests (subprocess-bound, requires live credentials).
