# internal/app — DOX

Parent: root `AGENTS.md`.

## Purpose

Application orchestration. Runs the pipeline (scan → download → transform → publish), retries failed jobs, sets up destination auth, lists jobs/events. Composes all other internal packages; the only layer that imports the full plugin set together.

## Ownership

- Package: `app`.
- Files: `sync.go`, `auth.go`, `jobs.go`, `retry.go`.

## Local Contracts

```dox
R1 entry_points := {RunSync, RunScan, RetryFailed, SetupYouTubeAuth, ListJobs, ListEvents, CheckSecrets}.
R2 RunSync sequence := cleanup_stale ∧ enforce_max_bytes → scanSources → [dry_run: showDryRunPlan] ∨ per_item(processItem) → PruneEvents.
R3 processItem := CreateJob(download_pending) → download → per_transform(transform, delete_prior) → per_destination(publish) → UpdateJobStatus(published) → [¬keep_successful: DeleteMedia].
R4 item_failure -> M log ∧ continue; F abort_sibling_items. (root R260)
R5 max_items_per_run := hard ceiling on processItem invocations per RunSync.
R6 route_resolution := resolveTransforms ∧ resolveDestinations match cfg.Routes by route.Source == item.SourceName; honor *.Enabled == false.
R7 dry_run -> F download ∧ F publish ∧ F db_mutation beyond UpsertMediaItem(scan path) ∧ F lock_bypass. (root R280/R281; see also scan path note below)
R8 publish.Plugin selection := switch dstCfg.Type ∈ {youtube, facebook_page}; unknown_type -> Warn ∧ skip.
R9 job_status_terminal_values := {published, failed}; transient values per root R250.
R10 ListJobs/ListEvents -> M honor --json flag ∧ --last N ∧ --failed filter.
```

Note: `scanSources` currently emits a single placeholder `MediaItem` per enabled source (example-001) and is the integration point for real source plugins (root R153). `scanSources` writes to DB even in dry-run path via `UpsertMediaItem` — tracked as drift vs root R281; do not add new writes until resolved.

## Work Guidance

- `processItem` is the only place download/transform/publish are chained — edit there to change per-item flow.
- Errors are recorded as `job_events` (level=error, component=sync.*) AND `jobs.error_message`. Keep both in sync.
- New destination type: add `case` in `processItem` switch + constructor in `internal/publish`.
- `parseClientSecret` accepts `client_id=..\nclient_secret=..` form OR `id:secret` fallback. Both are parsed, neither is logged.

## Verification

```bash
go vet ./internal/app
go test ./internal/app
```

No tests yet. Pipeline depends on external subprocesses (yt-dlp, ffmpeg); integration tests deferred.
