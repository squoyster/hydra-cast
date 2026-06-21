# internal/secrets — DOX

Parent: root `AGENTS.md`.

## Purpose

Secret resolution. Parses `secret://` references from config and returns the plaintext value from OpenBao (preferred) or the file fallback. Also provides redaction + fingerprinting helpers for safe logging.

## Ownership

- Package: `secrets`.
- Files: `resolver.go` (`Resolver`, `NewResolver`, `Resolve`, `getOpenBaoToken`, `resolveFileFallback`, `Fingerprint`, `Redact`), `resolver_test.go`.

## Local Contracts

```dox
R1 ref_format := "secret://" + scheme_path. Currently only "secret://openbao/..." is accepted; other schemes -> error.
R2 resolution_order := explicit_openbao_ref → (openbao_configured? openbao : fallback) → fallback. (root R221)
R3 openbao_not_configured ∧ fallback_enabled -> resolveFileFallback.
R4 openbao_token sources := BAO_TOKEN env → VAULT_TOKEN env → OpenBao.TokenFile. (root R223)
R5 fallback_path := Fallback.Root + "/" + strip("openbao/kv/hydracast/" prefix) from ref.
R6 Fingerprint(value) := sha256 first 4 bytes → "sha256:<8 hex>". Safe to log. (root R212)
R7 Redact(value) := first2 + "****" + last2, or "****" if len<=4. For short display contexts.
R8 F log ∧ F return(resolved_plaintext) from non-Resolve code paths. (root R210)
```

Known gap: `resolveOpenBao` returns `"openbao client not yet implemented"` for the configured-and-tokened path — only the file fallback actually returns values today. OpenBao HTTP client integration is pending.

## Work Guidance

- Plaintext secrets must never be logged — callers use `Fingerprint`/`Redact` from this package when a reference must be acknowledged in output.
- New scheme: add dispatch in `Resolve` (e.g. `secret://env/...`) and a corresponding `resolveX` method.

## Verification

```bash
go vet ./internal/secrets
go test ./internal/secrets
```

Tests in `resolver_test.go`. Plaintext-logging regressions are review-caught, not test-caught — keep a human check on any new accessor.
