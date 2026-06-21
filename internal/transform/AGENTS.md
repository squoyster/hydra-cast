# internal/transform — DOX

Parent: root `AGENTS.md`.

## Purpose

Transformer plugins. Declares `Plugin` interface and implements the `ffmpeg` transformer with named presets.

## Ownership

- Package: `transform`.
- Files: `transformer.go` (interface), `ffmpeg.go` (`FFmpeg` + preset table), `ffmpeg_test.go`.

## Local Contracts

```dox
R1 Plugin interface := {Name()string, Transform(ctx, *LocalMedia)(*LocalMedia, error)}. (NOTE: FFmpeg.Transform takes extra config.TransformConfig arg — interface conformance is pending; do not yet assign FFmpeg to transform.Plugin variable.)
R2 FFmpeg.Transform := subprocess(ffmpeg) with args from cfg.Args ∨ presetArgs(cfg.Preset).
R3 presets := {faststart_mp4, normalize_audio, convert_to_mp4, extract_audio, scale_1080p, none}.
R4 preset unknown -> error. preset=="none" -> copy-only.
R5 output_path := deriveOutputPath(input_path, preset); extension changes for {extract_audio→.mp3, convert_to_mp4/faststart_mp4→.mp4 (if not already)}.
R6 binary := cfg arg ∨ "ffmpeg"; resolved via os.Stat then exec.LookPath.
R7 output_path_exists -> overwrite (passes -y).
```

## Work Guidance

- Presets are hardcoded `switch` in `presetArgs` — add new presets there and document in `config.example.yaml`.
- Caller (`internal/app.processItem`) deletes the input file after each successful transform and reassigns `localMedia` to the output.

## Verification

```bash
go vet ./internal/transform
go test ./internal/transform
```

Tests in `ffmpeg_test.go`. Subprocess-bound cases require ffmpeg on PATH.
