# internal/app — DOX

Parent: root `AGENTS.md`.

## Purpose

Application orchestration. Runs the pipeline (scan → download → transform → publish), retries failed jobs, sets up destination auth, lists jobs/events. Composes all other internal packages; the only layer that imports the full plugin set together.

## Ownership

- Package: `app`.
- Files: `sync.go`, `scrape_reels.go`, `auth.go`, `jobs.go`, `retry.go`.

## Local Contracts

```dox
R1 entry_points := {RunSync, RunScan, RunScrapeReels, RetryFailed, SetupYouTubeAuth, ListJobs, ListEvents, CheckSecrets}.
R2 RunSync sequence := cleanup_stale ∧ enforce_max_bytes → scanSources (upsert new items, capture DB id onto item.ID, drain intake file) → [dry_run: showDryRunPlan] ∨ list_pending_items(MaxItemsPerRun) → per_item(processItem) → PruneEvents. The DB is the durable work queue; RunSync drains the oldest never-attempted items, NOT the just-scanned slice.
R3 processItem(ctx, cfg, db, resolver, item, logger) := CreateJob(item.ID, download_pending) → download → per_transform(transform, delete_prior) → per_destination(publish, tally published/failed) → UpdateJobStatus(failed ∨ published ∨ skipped) → [¬keep_successful: DeleteMedia]. Status never masks a failure as success: any dest error/result.Error → failed; zero dests published → skipped; else published. Returns error on failure so RunSync logs it (root R260). Resolver is threaded from RunSync ∧ RunScrapeReels (cmd creates it).
R4 item_failure -> M log ∧ continue; F abort_sibling_items. (root R260)
R5 max_items_per_run := the LIMIT passed to ListPendingItems; each run drains that many never-attempted items from the DB queue. Overflow stays pending for a later run (not orphaned); the intake file is drained on upsert regardless.
R6 route_resolution := resolveTransforms ∧ resolveDestinations match cfg.Routes by route.Source == item.SourceName; honor *.Enabled == false.
R7 dry_run -> F download ∧ F publish ∧ F db_mutation beyond UpsertMediaItem(scan path) ∧ F lock_bypass. (root R280/R281; see also scan path note below)
R8 publish.Plugin selection := switch dstCfg.Type ∈ {youtube: NewYouTube(dstCfg, resolver) (native Data API v3), facebook_page: NewFacebookPage(dstCfg, resolver) (native Graph API resumable chunked)}; unknown_type -> Warn ∧ skip.
R9 job_status_terminal_values := {published, failed, skipped}; transient values per root R250.
R10 ListJobs/ListEvents -> M honor --json flag ∧ --last N ∧ --failed filter.
```

Note: `scanSources` dispatches per source via `scanSource` (switch on `srcCfg.Type`): `url_list` is the first real plugin (reads a reels.json intake, `{items:[...]}` schema, `internal/source/urllist.go`); other types still emit the `example-001` placeholder. After upserting items it captures the DB id onto `item.ID` (so `processItem`'s `CreateJob` links the job to the media item — was `media_item_id=0`, a latent bug). A `url_list` source's intake file is then drained (`os.Remove`) gated on `!dryRun ∧ items>0` — items are already durable in `media_items`, so DB dedup (root R241) makes re-scans idempotent and `retry --failed` covers failures. RunSync then drains the DB pending queue (`ListPendingItems`, items with no job) rather than the just-scanned slice — so items beyond `max_items_per_run` are processed in a later run instead of being orphaned. `scanSources` still writes to DB even in dry-run via `UpsertMediaItem` (drift vs root R281); the file drain is the only scan-time side effect and is correctly dry-run-gated.

## Work Guidance

- `processItem` is the only place download/transform/publish are chained — edit there to change per-item flow.
- Errors are recorded as `job_events` (level=error, component=sync.*) AND `jobs.error_message`. Keep both in sync.
- New destination type: add `case` in `processItem` switch + constructor in `internal/publish`.
- YouTube OAuth: `ProcessItem` passes the `*secrets.Resolver` to `NewYouTube`; the plugin resolves `client_id_ref`, `client_secret_ref`, and `token_ref#refresh_token` itself (app code holds no OpenBao key names). The refresh token drives oauth2 auto-refresh; neither cred is logged. Facebook: `NewFacebookPage` takes the same resolver and resolves `page_token_ref` (whole) itself.

## Verification

```bash
go vet ./internal/app
go test ./internal/app
```

No tests yet. Pipeline depends on external subprocesses (yt-dlp, ffmpeg); integration tests deferred.
