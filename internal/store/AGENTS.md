# internal/store — DOX

Parent: root `AGENTS.md`. Child: `migrations/AGENTS.md`.

## Purpose

SQLite persistence layer. Opens the DB, runs embedded migrations, and provides typed accessors over the four core tables (`media_items`, `jobs`, `publish_results`, `job_events`). CGO-free via `modernc.org/sqlite`.

## Ownership

- Package: `store`.
- Files: `store.go` (`Store`, `New`, `Migrate`, accessors), `store_test.go`. Embedded migrations live under `migrations/`.

## Local Contracts

```dox
R1 dsn := dbPath+"?_journal_mode=WAL&_busy_timeout=5000".
R2 Store embeds nothing exported; access raw *sql.DB via Store.DB() for queries not yet wrapped.
R3 Migrate := embed.FS read migrations/*.sql, exec each in lexical order. Idempotent per-statement (CREATE ... IF NOT EXISTS).
R4 UpsertMediaItem := SELECT existing by (source_name, external_id); if present return existing id WITHOUT update; else INSERT. NOT a true upsert — later fields ignored on conflict.
R5 CreateJob := INSERT(media_item_id, job_type, status, started_at, attempts=0). Returns LastInsertId.
R6 UpdateJobStatus := sets status ∧ finished_at=now ∧ error_message. Does NOT bump attempts.
R7 RecordEvent := INSERT(job_events). job_id nullable.
R8 PruneEvents(maxRetention) := DELETE rows not in top-N by id DESC. Called by RunSync tail. (root R274)
R9 GetFailedJobs := status ∈ {failed, retryable_failed}, ordered by id ASC.
R10 F store(large_media_blobs ∨ resolved_secret_values). (root R245)
R11 timestamps := RFC3339 strings (UTC), stored as TEXT.
```

## Work Guidance

- Schema changes go in `migrations/` as a new numbered file — never edit `001_init.sql` in a breaking way on an existing deployment.
- New accessor = method on `*Store` taking `ctx context.Context` first; mirror existing error-wrapping style.
- `UpsertMediaItem` dedup semantics: callers must not expect field refresh on re-scan.

## Verification

```bash
go vet ./internal/store
go test ./internal/store
```

Tests: `TestNewStore`, `TestMigrate`, `TestUpsertMediaItem` (incl. idempotency), `TestCreateJob`, `TestUpdateJobStatus`, `TestRecordEvent`, `TestPruneEvents`, `TestGetFailedJobs`, `TestUpsertMediaItemWithPublishedAt`. `newTestStore` helper builds a temp-file store + runs Migrate.

## Child DOX Index

```dox
R1 child(migrations) = embedded SQL migrations loaded via go:embed.
```
