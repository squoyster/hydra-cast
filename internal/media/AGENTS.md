# internal/media — DOX

Parent: root `AGENTS.md`.

## Purpose

Media-file utilities: content fingerprinting, ffprobe inspection, size queries, and disk hygiene (stale cleanup, byte-budget enforcement, post-publish deletion).

## Ownership

- Package: `media`.
- Files: `checksum.go` (`Fingerprint`, `FileSize`), `probe.go` (`Probe`, `ProbeResult`), `cleanup.go` (`CleanupStaleFiles`, `EnforceMaxWorkingBytes`, `DeleteMedia`), `media_test.go`.

## Local Contracts

```dox
R1 Fingerprint(path) := sha256 over file bytes, returned as "sha256:<hex>". (matches secrets.Fingerprint format, different input)
R2 Probe(path, ffprobeBinary) := subprocess(ffprobe) → JSON → ProbeResult{Duration, Width, Height, Codec, Bitrate}.
R3 CleanupStaleFiles(dir, maxAge) := remove regular files with ModTime older than now-maxAge; dirs preserved; missing dir = noop.
R4 EnforceMaxWorkingBytes(dir, maxBytes) := if total>maxBytes, delete files oldest-first until under budget.
R5 DeleteMedia(path) := os.Remove, treats empty path ∧ nonexistent as success.
R6 ffprobe binary default := "ffprobe" (resolved by exec, not os.Stat — differs from FFmpeg check).
```

## Work Guidance

- `Fingerprint` reads the whole file — avoid calling on huge media inside hot loops; the `MediaItem.Fingerprint` field is the persisted form.
- `EnforceMaxWorkingBytes` deletes in reverse slice order (newest appended last) — effectively oldest-by-mtime first. Preserve this invariant if refactoring.

## Verification

```bash
go vet ./internal/media
go test ./internal/media
```

Tests: `TestFingerprint` (known sha256 vector), `TestFingerprintNonExistent`, `TestCleanupStaleFiles` (old/new mtime split), `TestDeleteMedia` (incl. empty/missing). `Probe` is subprocess-bound (no test).
