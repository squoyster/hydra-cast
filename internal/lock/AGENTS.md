# internal/lock — DOX

Parent: root `AGENTS.md`.

## Purpose

Process-level file lock to prevent overlapping scheduled runs. Backed by `flock(2)` via syscalls on `/data/hydracast.lock`.

## Ownership

- Package: `lock`.
- Files: `flock.go` (`FileLock`, `New`, `TryLock`, `Unlock`, `Path`).

## Local Contracts

```dox
R1 lock_path := /data/hydracast.lock. (root R214)
R2 TryLock := open(O_CREAT|O_RDWR|O_CLOEXEC) → flock(LOCK_EX|LOCK_NB) → write pid.
R3 EWOULDBLOCK -> check isStale; stale(pid gone) -> os.Remove ∧ retry TryLock; ¬stale -> return "another instance is running".
R4 caller_policy := lock_active -> exit(0); lock_unacquirable -> record_event ∧ exit(0). (root R215)
R5 Unlock := flock(LOCK_UN) ∧ close(fd) ∧ os.Remove(path).
R6 isStale heuristic := /proc/<pid> existence check (Linux-only); on macOS/darwin this returns false (proc path absent) — stale detection is best-effort.
R7 StaleThreshold := 1*time.Hour (declared, currently unused; reserved for future age-based detection).
```

## Work Guidance

- This is the only concurrency guard — do not introduce a second lock primitive.
- darwin caveat: stale detection relies on `/proc`; container target is Linux so this is fine in production, but be aware during local dev on macOS.

## Verification

```bash
go vet ./internal/lock
go build ./internal/lock
```

No tests (syscall + filesystem timing).
