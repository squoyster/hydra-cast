# cmd/hydracast — DOX

Parent: root `AGENTS.md`.

## Purpose

CLI entry point. Wires Cobra subcommands to `internal/app` orchestrators. Owns flag parsing and the bootstrap sequence (config → lock → store → migrate → app). No business logic.

## Ownership

- Package: `main`.
- One file: `main.go`.
- Binary: `hydracast`.

## Local Contracts

```dox
R1 commands := {sync, validate, scan, jobs, log, retry, auth youtube, secrets check, scrape-reels}.
R2 persistent_flags := {--config(default /data/config.yaml), --lock-file(default /data/hydracast.lock), --dry-run, --json}.
R3 sync ∧ retry ∧ scrape-reels -> M acquire(--lock-file) via internal/lock; lock_active -> exit(0).
R4 config_invalid -> exit(1).
R5 store_open ∨ migrate_fail -> propagate(nonzero).
R6 dependencies := {app, config, joblog, lock, secrets, store}; F import(download, transform, publish, source) directly — they belong to app layer.
```

## Work Guidance

- Adding a command = new `xCmd()` returning `*cobra.Command` + `rootCmd.AddCommand` in `main()`.
- `loadConfig()` always pairs `config.Load` + `config.ApplyDefaults`.
- Exit codes: see root R290–R294. `rootCmd.Execute()` error → `os.Exit(1)`.

## Verification

```bash
go build ./cmd/hydracast
go vet ./cmd/hydracast
```

No tests yet — CLI surface is thin; behavior lives in `internal/app`.
