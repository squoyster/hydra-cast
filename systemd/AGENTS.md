# systemd — DOX

Parent: root `AGENTS.md`.

## Purpose

systemd unit + timer for scheduled execution. Drives a `podman run` one-shot of the published container image every 10 minutes.

## Ownership

- No Go code — two unit files: `hydracast-sync.service`, `hydracast-sync.timer`.

## Local Contracts

```dox
R1 service Type=oneshot; Wants/After=network-online.target.
R2 ExecStart := podman run --rm --name hydracast-sync --network hydracast-net -v /opt/hydracast/data:/data:Z ghcr.io/squoyster/hydracast:latest sync --config /data/config.yaml.
R3 timer OnBootSec=2min; OnUnitActiveSec=10min; Persistent=true.
R4 WantedBy=timers.target.
R5 host_data_root := /opt/hydracast/data mounted at /data. (container-internal layout per root /data spec)
R6 F daemon. (root R312) Scheduling is the timer's job; the binary is one-shot.
```

## Work Guidance

- Install: copy both files to `/etc/systemd/system/`, `systemctl daemon-reload`, `systemctl enable --now hydracast-sync.timer`.
- Changing the cadence = edit `OnUnitActiveSec` only; keep `Persistent=true` for missed-run catch-up.
- Image tag is pinned to `:latest` — consider pinning to a digest for production reproducibility.

## Verification

```bash
systemd-analyze verify systemd/hydracast-sync.service systemd/hydracast-sync.timer
```

Runtime verification happens on the target host via `systemctl status hydracast-sync.timer` and `journalctl -u hydracast-sync.service`.
