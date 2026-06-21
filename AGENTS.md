# HydraCast AGENTS — DOX v1

Purpose: compact, agent-readable policy for HydraCast. Formal rules are authoritative. Prose is advisory. README.md is the authoritative architecture spec — read it first.

## Notation

```dox
# Logic
□=always; ◇=before closeout; ¬=not; ∧=and; ∨=or; →=implies; ≺=before; ≻=higher priority; :=define; ∅=none.

# Norms
M x=must x. F x=must-not x. S x=should x unless blocked by stronger rule. P x=may x. Pref(a,b)=prefer a over b unless stronger rule blocks.

# Core vars
Repo=repo root. p=path. T=task. Δ=changed paths. d=AGENTS.md. D(p)=root→nearest AGENTS chain for p. near(p)=nearest governing AGENTS. wt=worktree. S=symbol.

# Rule shape
R[id]: scope | trigger -> norm/action [verify] [except] [effect]

# Directive schema
Dir := {scope,trigger,norm∈{M,F,S,P,Pref},action,verify?,except?,effect?}

# Priority
safety ≻ DOX_root ≻ near(p) ≻ parent(D(p)) ≻ task_instruction ≻ preference
conflict(a,b)->choose(max_priority); tie->choose(more_specific); unresolved->stop_report(conflict)
```

## Meta Rules

```dox
R000 global | nontrivial(T) -> M translate(relevant_directives(T),dox) ∧ reason_over(dox) ∧ execute(derived_plan) ∧ verify(postconditions).
R001 global | new_agent_directive(x) -> M encode_as(Dir) ∧ Pref(dox_notation,prose) ∧ allow(prose_if_human_clarity_needed).
R002 global | acting_on_directive(x) -> M parse(x) ∧ classify(x,{invariant,precondition,postcondition,permission,prohibition,preference,exception}) ∧ encode(x,dox).
R003 global | report(T) -> S include(assumptions ∧ selected_rules ∧ actions_taken ∧ verification_results ∧ unresolved_conflicts?).
R004 global | reasoning_trace -> F expose_long_chain_of_thought ∧ Pref(compact_rule_trace,deliberation_prose).
```

## DOX Authority

```dox
R010 all | work_on(p) -> M comply(D(p)).
R011 all | artifacts(p) -> M understandable_from(D(p)).
R012 all | conflict(parent,child) -> local_detail:=child.
R013 all | weaken(child,DOX) -> invalid(child_rule).
R014 all | user_requests(durable_behavior_change) -> M record(root_AGENTS ∨ relevant_child_AGENTS).
```

## Read Before Edit

```dox
R020 edit(any) -> M read(root/AGENTS.md) ∧ P:=expected_touch_paths(T) ∧ ∀p∈P:walk(Repo→p)∧read(AGENTS_on_route)∧set(D(p),near(p)).
R021 edit(p) -> F rely(memory,DOX) ∧ M reread(D(p),current_session).
```

## DOX Update / Hierarchy

```dox
R030 meaningful(Δ) -> M dox_pass(Δ) before done(T).
R031 affects(Δ,{purpose,scope,ownership,responsibility,durable_structure,contract,workflow,operating_rule,input,output,permission,constraint,side_effect,artifact,user_pref,AGENTS_lifecycle,index}) -> M update(near(Δ)).
R032 affects(Δ,{parent_structure∨parent_ownership∨parent_workflow∨child_index}) -> M update(parent_doc).
R033 parent_change_alters(local_rules) -> M update(child_doc).
R034 stale(text)∨contradictory(text) -> M delete(text).
R035 small(Δ)∧¬changes_behavior(Δ)∧¬changes_contract(Δ) -> P leave_docs_unchanged ∧ M dox_pass.
R036 root_AGENTS -> M own(global_rules ∪ user_preferences ∪ workflow_rules ∪ top_child_index).
R037 child_AGENTS -> M own(domain_rules ∪ local_child_index).
R038 parent(d) -> M explain(direct_children ∧ parent_owned_scope).
R039 closer(d,p) -> M more_specific(d,p) ∧ more_practical(d,p).
R040 durable_boundary(folder)∧has(folder,{purpose,rules,responsibilities,workflow,materials,quality}) -> M create(folder/AGENTS.md).
R041 child_AGENTS -> S sections([Purpose,Ownership,Local Contracts,Work Guidance,Verification,Child DOX Index]).
R042 ¬specific_standards -> Work_Guidance:=∅.  ¬existing_check -> Verification:=∅.
```

## Style / Closeout

```dox
R050 docs -> M concise ∧ current ∧ operational ∧ stable_contracts_only ∧ F diary_entries.
R051 docs -> M broad_rules_in_parent ∧ concrete_details_in_child ∧ direct_bullets ∧ explicit_names.
R052 docs -> F duplicate_rules_unless_scope_needs_local_copy; M trim(obvious∨repeated∨misplaced∨obsolete_warning).
R053 closeout(T) -> M recheck(Δ,D(Δ)) ∧ update(nearest_docs∪affected_parents∪affected_children) ∧ refresh(child_indices) ∧ delete(stale∨contradictory) ∧ run(existing_verification_if_relevant) ∧ report(unchanged_docs,reason).
R054 response -> M terse ∧ act_then_report_briefly ∧ F restate_obvious_task ∧ F excessive_deliberation_prose.
R055 nontrivial(T) -> M plan:=ordered_steps(T) ∧ execute_in_order(plan) ∧ verify(each_step_before_next).
R056 all -> M correctness>speed ∧ read_before_edit ∧ confirm_assumptions ∧ run_gates ∧ double_check(commands∧paths).
```

## Hard Rules

```dox
R100 all -> F force_push.
```

## Project Context

Prose: HydraCast is a Go CLI app for scheduled video syndication. Pipeline: sources → download → optional ffmpeg transform → publish to YouTube/Facebook. Runs as a one-shot scheduled job via systemd timer + Podman/Docker, ¬daemon. MVP_1 skeleton implemented (cmd + internal/{app,config,source,download,transform,publish,store,media,joblog,lock,secrets} + systemd + Dockerfile); source plugins, OpenBao client, and publish auth wiring are stubs/pending. README.md is authoritative architecture spec.

## Tech Stack

```dox
R200 stack := {
  language: Go(CGO-free_preferred),
  sqlite_driver: modernc.org/sqlite(no_CGO),
  cli: Cobra ∨ urfave/cli,
  config: YAML + struct_validation,
  external_deps: {yt-dlp(subprocess), ffmpeg/ffprobe(subprocess)},
  secrets: OpenBao_preferred + file_mounted_fallback(/data/secrets),
  container: Go_binary + python:3.12-slim + yt-dlp + ffmpeg,
  go_version: 1.22+(per_Dockerfile)
}.
```

## Critical Constraints

```dox
R210 secret_value -> F log ∧ F store(SQLite) ∧ F write_to_disk.
R211 dry_run_output ∧ job_events -> F include(secret_values).
R212 may_log := {secret_reference_path, found_status, redacted_fingerprints(sha256:abcd1234...)}.
R213 media_files := ephemeral; M delete(after_successful_publish) by default.
R214 file_lock := /data/hydracast.lock; purpose := prevent_overlapping_scheduled_runs.
R215 lock_active -> exit(0). lock_stale -> M remove ∧ continue. lock_unacquirable -> M record_event ∧ exit(0).
R216 runtime_data -> M live_on(/data_volume).
R217 config_path := /data/config.yaml. db_path := /data/hydracast.db.
```

## External Volume Layout (/data)

```
/data
├── config.yaml
├── hydracast.db
├── openbao-token
├── secrets/          # dev-only fallback files
├── cookies/          # facebook.txt (dev fallback)
├── work/             # temp media files
├── cache/
└── logs/
```

## Secrets Management

```dox
R220 config -> M reference_secrets_symbolically (e.g. secret://openbao/kv/hydracast/youtube/client).
R221 resolution_order := explicit_openbao_ref → default_openbao_path → file_fallback → env_fallback.
R222 required_production_secret_unresolvable -> M fail_validation.
R223 openbao_token_delivery := /data/openbao-token ∨ BAO_TOKEN ∨ VAULT_TOKEN(env).
R224 cookie_data -> M materialize_into_temp_files(download_duration_only) ∧ M remove(after).
```

## Plugin Architecture

```dox
R230 pipeline := source → downloader → transformer → destination. compiled_in(¬dynamic).
R231 initial_plugins := {
  sources: {facebook_page_videos, youtube_channel, rss_feed, local_directory},
  downloader: {yt_dlp},
  transformer: {ffmpeg},
  destinations: {youtube, facebook_page}
}.
```

## State Model (SQLite)

```dox
R240 tables := {media_items, jobs, publish_results, job_events}.
R241 media_items := {source_identity, fingerprint, external_id, metadata}. UNIQUE(source_name, external_id).
R242 jobs := {processing_status, attempts, error_tracking}. FK→media_items.
R243 publish_results := one_row_per(media_item × destination). UNIQUE(media_item_id, destination_name).
R244 job_events := {level, component, context_json}. recent_operational_events.
R245 DB -> F store(large_media_blobs ∨ resolved_secret_values).
```

## Job States

```dox
R250 media_state_flow := new → detected → download_pending → downloading → downloaded → transform_pending → transforming → transformed → publish_pending → publishing → published | failed | retryable_failed | permanent_failed | skipped.
R251 destination_state_flow := pending → uploading → published | failed | retryable_failed | permanent_failed | auth_required | quota_limited | skipped.
```

## Failure Handling

```dox
R260 item_failure -> M record_failed_job ∧ process_others ∧ exit(0).
R261 system_failure({bad_config, DB_unavailable}) -> exit(nonzero).
R262 auth_failure -> M mark_destination(auth_required); exit(nonzero) if ¬work_can_continue.
R263 partial_failure -> exit(0) ∧ record_failed_jobs.
R264 retryable := {network_timeout, HTTP_429, temp_platform_error, DNS_failure, upload_interruption}.
R265 permanent := {unsupported_media, deleted_source, invalid_credentials, duplicate_policy_violation, missing_metadata}.
```

## Disk Usage Policy

```dox
R270 default_limits := {max_items_per_run:3, max_working_bytes:5000MB, max_media_duration:4h}.
R271 before_run -> M remove(stale_temp_files) ∧ enforce(max_working_bytes) ∧ remove(old_cache).
R272 after_success -> M delete(original ∧ transformed_copies) ∧ keep(metadata).
R273 after_failure -> M delete(media) ∧ keep(error_state); P retain_for_debugging.
R274 job_event_retention := 1000_events.
```

## Dry Run Behavior

```dox
R280 dry_run -> M load_config ∧ validate ∧ scan_sources ∧ detect_new_items ∧ resolve_routes ∧ show({intended_downloads, transforms, publishes}).
R281 dry_run -> F download ∧ F upload ∧ F db_writes unless --write-discovery.
```

## Exit Codes

```dox
R290 exit(0) := valid ∨ item_level_failures(non_blocking).
R291 exit(1) := config_invalid.
R292 exit(2) := missing_runtime_dependency.
R293 exit(3) := auth ∨ credential_issue.
R294 exit(4) := storage_issue.
```

## Key Commands (when implemented)

```
hydracast sync --config /data/config.yaml      # primary scheduled run
hydracast validate --config /data/config.yaml  # config check
hydracast scan --config /data/config.yaml      # scan sources only
hydracast sync --dry-run --config /data/config.yaml
hydracast jobs --last 20 --config /data/config.yaml
hydracast jobs --failed --config /data/config.yaml
hydracast retry --failed --config /data/config.yaml
hydracast log --last 100 --config /data/config.yaml
hydracast auth youtube --destination <name> --config /data/config.yaml
hydracast secrets check --config /data/config.yaml
```

All commands support `--json` output where applicable.

## Repo Layout (actual)

```
cmd/hydracast/main.go
internal/{app,config,source,download,transform,publish,store,media,joblog,lock,secrets}/
internal/store/migrations/        # embedded SQL migrations (not repo-root migrations/)
systemd/
Dockerfile
compose.yaml
config.example.yaml
```

Each `internal/*` folder and `cmd/hydracast`, `systemd` has its own `AGENTS.md` (see Child DOX Index).

## MVP Roadmap

```dox
R300 MVP_1 := {config, validation, SQLite, yt-dlp_source, download, cleanup, dry_run, jobs/log_inspection}.
R301 MVP_2 := {youtube_destination, OAuth_setup, upload, retry, publish_dry_run}.
R302 MVP_3 := {ffprobe_inspection, ffmpeg_transform, presets(faststart_mp4, normalize_audio)}.
R303 MVP_4 := {facebook_page_destination, token_validation, publish_status}.
R304 MVP_5 := {multi_source/destination_routing, per_route_transforms, per_route_limits}.
```

## Design Constraints

```dox
R310 F use({Kubernetes, Redis, RabbitMQ, Celery, heavy_DB_servers}).
R311 F large_framework_dependencies.
R312 Pref(one_shot_execution, daemon_mode); daemon_mode := optional_later.
R313 M conservative_disk_usage ∧ deterministic_cleanup.
R314 F hardcoded_platform_behavior.
R315 F upload_duplicate_content without explicit_policy.
```

## GitNexus

```dox
R120 gitnexus_repo:=hydra-cast; indexed:=true; symbols:=540; relationships:=1394; flows:=41.
R121 reindex -> M run(`npx gitnexus analyze`); if repo_has(.gitnexus/run.cjs) then M run(`node .gitnexus/run.cjs analyze`); npm11_crash:`npm i -g gitnexus`.
R122 edit(symbol S) -> M impact({target:S,direction:"upstream"}) before edit(S); if risk(S)∈{HIGH,CRITICAL} then warn_user.
R123 before(commit) -> M detect_changes().
R124 regression_review -> M detect_changes({scope:"compare",base_ref:"main"}).
R125 discover_flow(concept) -> Pref(query({query:concept}),grep).
R126 need_context(S) -> M context({name:S}).
R127 rename(S) -> M use(GitNexus.rename) ∧ F find_replace_rename.
R128 all -> F ignore(HIGH∨CRITICAL risk) ∧ F commit_before(detect_changes).
R129 resources := {context:overview+freshness, clusters:functional_areas, processes:flows, process/X:trace(X)}.
R130 skills := {architecture:exploring, blast_radius:impact-analysis, debug:debugging, refactor:refactoring, guide:guide, cli:cli} under `.claude/skills/gitnexus/gitnexus-*`.
```

## Child DOX Index

```dox
# Status: MVP_1 skeleton implemented. Each folder below has its own AGENTS.md (created once durable, per R040).
R150 child(cmd/hydracast)=entry point: main.go, CLI wiring (Cobra). OWNED.
R151 child(internal/app)=application orchestration, pipeline runner. OWNED.
R152 child(internal/config)=YAML config load + struct validation. OWNED.
R153 child(internal/source)=source plugin interface (MediaItem + Plugin); concrete plugins pending. OWNED.
R154 child(internal/download)=downloader plugin: yt-dlp subprocess. OWNED.
R155 child(internal/transform)=transformer plugin: ffmpeg/ffprobe subprocess, presets (faststart_mp4, normalize_audio, convert_to_mp4, extract_audio, scale_1080p, none). OWNED.
R156 child(internal/publish)=destination plugins: youtube, facebook_page; yt-dlp-based upload + status. OWNED.
R157 child(internal/store)=SQLite persistence: media_items, jobs, publish_results, job_events. OWNED.
R157a child(internal/store/migrations)=embedded SQL migrations (currently 001_init.sql). OWNED.
R158 child(internal/media)=media utilities: sha256 fingerprint, ffprobe probe, cleanup, byte-budget enforcement. OWNED.
R159 child(internal/joblog)=slog JSON logger + EventRecorder; durable events written via store.RecordEvent. OWNED.
R160 child(internal/lock)=file lock (/data/hydracast.lock) via flock(2). OWNED.
R160a child(internal/secrets)=secret resolver: secret:// refs → OpenBao (pending) ∨ file fallback; Fingerprint + Redact helpers. OWNED.
R162 child(systemd)=systemd timer + service units (podman one-shot, 10min cadence). OWNED.
R163 root_owns := Dockerfile ∪ compose.yaml ∪ config.example.yaml ∪ top-level tooling.
```

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **hydra-cast** (540 symbols, 1394 relationships, 41 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> Index stale? Run `node .gitnexus/run.cjs analyze` from the project root — it auto-selects an available runner. No `.gitnexus/run.cjs` yet? `npx gitnexus analyze` (npm 11 crash → `npm i -g gitnexus`; #1939).

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows. For regression review, compare against the default branch: `detect_changes({scope: "compare", base_ref: "main"})`.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `rename` which understands the call graph.
- NEVER commit changes without running `detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/hydra-cast/context` | Codebase overview, check index freshness |
| `gitnexus://repo/hydra-cast/clusters` | All functional areas |
| `gitnexus://repo/hydra-cast/processes` | All execution flows |
| `gitnexus://repo/hydra-cast/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
