# internal/store/migrations — DOX

Parent: `internal/store/AGENTS.md`.

## Purpose

Embedded SQL migrations. Loaded at runtime via `//go:embed migrations/*.sql` in `internal/store/store.go` and executed by `Store.Migrate`.

## Ownership

- No Go package — directory of `.sql` files consumed by `internal/store`.

## Local Contracts

```dox
R1 filename_pattern := NNN_name.sql (zero-padded 3 digits, lexical order == apply order).
R2 every statement := CREATE ... IF NOT EXISTS ∨ idempotent DDL; M no destructive ALTER without guard.
R3 current_files := {001_init.sql}.
R4 001_init.sql tables := {media_items, jobs, publish_results, job_events} + indexes {idx_job_events_timestamp, idx_jobs_media_item_id, idx_publish_results_media_item_id}.
R5 UNIQUE constraints := media_items(source_name, external_id); publish_results(media_item_id, destination_name). (root R241/R243)
R6 FK := jobs.media_item_id → media_items.id; publish_results.media_item_id → media_items.id.
```

## Work Guidance

- Adding a migration: append `NNN_<topic>.sql`. Do not edit earlier files in a way that breaks an already-applied DB.
- Keep migrations pure DDL — no data backfill in the same file as schema changes unless atomicity demands it.

## Verification

`go test ./internal/store` runs `Migrate` against a fresh temp DB and asserts all four tables exist.
