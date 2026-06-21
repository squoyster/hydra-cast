# internal/joblog — DOX

Parent: root `AGENTS.md`.

## Purpose

Structured logging (`log/slog` JSON to stdout) plus an in-process `EventRecorder` abstraction. Two channels: human/operational logs (slog) and durable `job_events` rows (written by `store.RecordEvent`).

## Ownership

- Package: `joblog`.
- Files: `events.go` (`Logger`, `New`, `WithComponent`, `WithMediaItemID`, `EventRecorder`, `JobEvent`).

## Local Contracts

```dox
R1 Logger := wrapper over *slog.Logger (JSON handler, LevelInfo, AddSource=false, dest=stdout).
R2 WithComponent(name) := returns derived Logger with slog attr component=<name>; used as "component" namespace across app layer.
R3 WithMediaItemID(id) := derived Logger with media_item_id attr.
R4 EventRecorder.Record := constructs JobEvent, marshals context to ContextJSON, emits via slog; NOTE current impl does NOT persist to job_events table — persistence is done separately by store.RecordEvent in the app layer.
R5 slogLevel mapping := {debug, info, warn, error}; unknown -> info.
R6 F log(secret_value). (root R210) Redact via internal/secrets.Redact if a value must appear.
```

Known drift: `EventRecorder` is declared but not used by `internal/app` — the app calls `store.RecordEvent` directly. Either wire `EventRecorder` to persist or remove it.

## Work Guidance

- Logging handler is constructed once in `New()` — to change verbosity or output, edit there (consider accepting a `Level`/`io.Writer` if reuse grows).
- `JobEvent.ContextJSON` is a JSON string column, not structured fields — keep values small and serializable.

## Verification

```bash
go vet ./internal/joblog
go build ./internal/joblog
```

No tests.
