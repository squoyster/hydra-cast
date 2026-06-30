# internal/publish — DOX

Parent: root `AGENTS.md`.

## Purpose

Destination plugins. Declares `Plugin` interface + `PublishResult`, implements `youtube` (native Data API v3) and `facebook_page` (native Graph API) uploaders.

## Ownership

- Package: `publish`.
- Files: `publisher.go` (interface + `PublishResult`), `youtube.go` (`YouTube`), `facebook_page.go` (`FacebookPage`).

## Local Contracts

```dox
R1 Plugin interface := {Name()string, Type()string, Publish(ctx, MediaItem, *LocalMedia)(*PublishResult, error)}.
R2 PublishResult := {RemoteID, RemoteURL, Status, Error}. Status ∈ {published, failed}.
R3 publish returns (result, nil) even on platform failure — Error is set inside PublishResult. Caller distinguishes transport error (err!=nil) from platform error (result.Error!=nil).
R4 YouTube.Type == "youtube". FacebookPage.Type == "facebook_page".
R5 upload_mechanism := youtube=NATIVE Data API v3 videos.insert (resumable, google.golang.org/api/youtube/v3); facebook_page=NATIVE Graph API resumable chunked upload (start/transfer/finish phases, net/http). Neither shells yt-dlp.
R6 YouTube metadata := Video.Snippet{Title, Description, CategoryId} + Video.Status{PrivacyStatus}. Title := item.Title ∨ filename basename. Privacy default public.
R7 FacebookPage requires cfg.PageID ∧ cfg.PageTokenRef; missing -> PublishResult{Status:failed}. Page access token resolved via resolver.Resolve(PageTokenRef) (whole ref; page-token secret is single-field {token} -> raw value).
R8 RemoteID := youtube: response.Id from videos.insert; facebook_page: video_id from the start-phase response. No stdout parsing (both native APIs).
R9 F upload_duplicate_content without explicit_policy. (root R315)
R10 YouTube OAuth := NewYouTube(cfg, *secrets.Resolver). Publish resolves client_id (ClientIDRef), client_secret (ClientSecretRef), refresh_token (TokenRef+"#refresh_token") via the resolver #key selector. oauth2.Config{Endpoint:google.Endpoint, Scopes:[youtube.YoutubeUploadScope]}; oauth2Cfg.Client auto-refreshes the access token from the refresh token. publish pkg imports secrets (leaf util; acyclic).
R11 FacebookPage auth := NewFacebookPage(cfg, *secrets.Resolver). Publish resolves the page access token via resolver.Resolve(PageTokenRef) (whole ref). Resumable chunked upload to Graph API {fbGraphVersion}/{pageID}/videos: start(file_size -> upload_session_id, video_id, start_offset) -> transfer(8MiB chunks, multipart field video_file_chunk, until start_offset>=file_size) -> finish(title, description). fbGraphVersion := v21.0. Page videos are public by default (no privacy param wired).
```

Known gaps:
- YouTube OAuth tokens must exist in the secret store (youtube/&lt;...&gt;/client#client_id, #client_secret; token#refresh_token). `SetupYouTubeAuth` (internal/app) one-time flow is broken (discards its token dir); consent must be run out-of-band and the refresh_token stored. The upload path relies on oauth2 token refresh, NOT that command.
- FacebookPage upload requires a page access token at PageTokenRef (resolved whole; secret is single-field {token}) and cfg.PageID. Page must have video publish permission. No FB SDK dependency (native net/http).
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
