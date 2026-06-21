# AGENTS.md — DOX-Min v1

Purpose: compact, agent-readable policy. Formal rules are authoritative. Prose is advisory.

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
R020 edit(any) -> M read(root/AGENTS.md) ∧ P:=expected_touch_paths(T) ∧ ∀p∈P:walk(Repo→p)∧read(AGENTS_on_route)∧read(child_if_listed_and_scope_contains(p))∧set(D(p),near(p)).
R021 edit(p) -> F rely(memory,DOX) ∧ M reread(D(p),current_session).
R022 navigate(repo) -> M read(.agent/tf.ctx) ≺ read(.agent/file.idx) ≺ read(.agent/symbol.idx) ≺ read(.agent/spec.idx) ≺ read(.agent/task.idx) ≺ select(files).
R023 routine_read -> M only(indexes ∪ changed_files ∪ directly_referenced_sources/tests ∪ relevant_docs).
R024 missing(file,indexes) -> M narrow(grep∨glob) ∧ update(.agent/index.overrides).
R025 skip_default -> M skip(session-ses_*.md ∪ specs/session-ses_*.md ∪ docs/archive/ ∪ .opencode/node_modules/ ∪ Volumes/ ∪ node_modules/).
```

## DOX Update / Hierarchy

```dox
R030 meaningful(Δ) -> M dox_pass(Δ) before done(T).
R031 affects(Δ,{purpose,scope,ownership,responsibility,durable_structure,contract,workflow,operating_rule,input,output,permission,constraint,side_effect,artifact,user_pref,AGENTS_lifecycle,index}) -> M update(near(Δ)).
R032 affects(parent_structure∨parent_ownership∨parent_workflow∨child_index) -> M update(parent_doc).
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

## Direct Git Model

```dox
R060 routine_work -> M use(git_directly).
R061 routine_work -> F use(taskforge checkpoint|submit|diff|pr).
R062 read_state -> P use(taskforge next|inspect|list|gates).
R063 lifecycle/worktree/branch/task_state -> Pref(git,taskforge_lifecycle).
R064 all -> M move_fast ∧ keep_gates_green ∧ maintain(task_state) ∧ F deliberate(facade_vs_git).
```

## Worktrees

```dox
R070 sequential(TASK-n,TASK-n+1) -> base(TASK-n+1):=tip(TASK-n).
R071 standalone(T) -> base(T):=clean(main_HEAD).
R072 create_wt(TASK-NNN) -> M run(`git -C /Volumes/Transcend/devel/task-forge worktree add -b agent/TASK-NNN-<slug> /Volumes/Transcend/devel/worktrees/task-forge/TASK-NNN <base-branch>`).
R073 fresh_wt(wt) -> M run(`ln -s /Volumes/Transcend/devel/task-forge/node_modules <wt>/node_modules`).
R074 all -> F work_in(/Volumes/Transcend/devel/task-forge). reason:main_checkout_swamp.
```

## Task State

```dox
R080 task_state_repo:=../task-state/. task_file(TASK-NNN):=../task-state/TASK-NNN.md.
R081 task_file(T) -> M frontmatter(status,assignee,claimed_at,completed_at,branch,worktree).
R082 complete(T) -> M fill(##Result).
R083 status_flow:=Inbox→NeedsSpec→Ready→InProgress→Review→Verify→Done.
R084 refactor_task(T)∧implemented(T)∧gates_pass(T) -> set(status,Done)∧set(completed_at,now)∧skip(Review/Verify unless requested).
R085 task_state_commit(TASK-NNN) -> M run(`cd ../task-state && TASKFORGE_INTERNAL=1 git add TASK-NNN.md && TASKFORGE_INTERNAL=1 git commit -m "TASK-NNN: ..." && TASKFORGE_INTERNAL=1 git push`).
```

## Gates / Commit / Cleanup

```dox
R090 done(T) -> M run_in_wt(`npm run typecheck`) ≺ run(`npm run lint`) ≺ run(`npm run build`) ≺ run(`npm test -- --run`).
R091 gates -> M lint_errors=0 ∧ P preexisting_warnings ∧ F bypass(gates).
R092 code_commit -> M run_in_wt(`git add -A && git commit -m "TASK-NNN: <summary>"`).
R093 push_useful -> P run_in_wt(`git push -u origin <branch>`).
R094 commit -> M location(task_worktree) ∧ branch(task_branch) ∧ respect_gitignore(node_modules∪dist).
R095 done(T) -> M remove_worktree(wt) ∧ if merged_or_superseded(branch) then delete(branch) ∧ if base_for_next_task(branch) then keep(branch).
R096 remove_wt -> M run(`git -C /Volumes/Transcend/devel/task-forge worktree remove <wt>`); dirty(wt)∧required -> P `--force`.
```

## Hard Rules / Permissions

```dox
R100 all -> F force_push.
R101 exists(.doctor-lock) -> M stop_all_work.
R102 push(main∨task-state,from_worktree) -> F push.
R103 task_state_push -> M use(task-state_worktree) ∧ env(TASKFORGE_INTERNAL=1).
R104 permissions -> allow("git *") ∧ deny("git push --force") ∧ allow("edit ../task-state/**") ∧ deny("tasks/**") ∧ deny(".git/**").
R105 edit(opencode.json) -> M user_restart(opencode_required).
```

## Durable Agent Identity

```dox
R110 identity -> M authoritative(durable_state) ∧ F source_of_truth(conversation_memory∨summaries∨prompt_text) ∧ prompt_identity:=projection(durable_identity).
R111 IDs := {agentId:AgentRuntime, sessionId:ModelSession, runId:ExecutionAttempt, taskId?:DurableWorkItem, claimId?:OwnershipClaim}; Pref(UUIDv7∨ULID,other_id).
R112 paths := {.taskforge/agents/<agentId>.json, .taskforge/sessions/<sessionId>.json, .taskforge/runs/<runId>.json}.
R113 before(model_call) -> M load(identity,durable_state) ∧ validate(repo∧worktree∧task∧claim_scope) ∧ inject(identity,model_context) ∧ if missing_or_inconsistent(required_identity) then refuse(identity_sensitive_work).
R114 available(agentId∧sessionId∧runId) -> M include_in(task_claims∧checkpoints∧logs∧summaries∧handoff_notes∧PR_metadata∧submission_metadata).
R115 exists(durable(agentId)) -> F regenerate(agentId). new(agentId) allowed_only_if initialize_new_identity∨explicit_fork.
R116 subagent(s) -> M own(agentId_s) ∧ explicit_parent_link(s,parent). handoff -> M include(source_identity∧target_identity).
```

## GitNexus

```dox
R120 gitnexus_repo:=task-forge; symbols≈2769; relationships≈6157; flows≈233.
R121 reindex -> M run(`node .gitnexus/run.cjs analyze`); fallback:`npx gitnexus analyze`; npm11_crash:`npm i -g gitnexus`.
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

## Gotchas

```dox
R140 commander_hide -> use `.command("name",{hidden:true})`; F `.hidden()`.
R141 tests -> use Vitest_temp_git(fs.mkdtempSync+git_init); mock_execa_by_command_string; CLI_regression may spawn dist/cli.js after build.
R142 stale_taskforge -> if hook emits `unknown command '_hook'` then reinstall_or_upgrade_taskforge ∨ prepend_wrapper_on_PATH.
R143 old_prepush -> if new_branch_push misclassified_force then `taskforge init --agent-framework opencode` ∨ `git -c core.hooksPath=/dev/null push ...`.
R144 commit_author_hint_after_success -> ignore.
R145 max_context_warning -> often_false_alarm; F overcompress_context. window≈1M_tokens.
```

## Child DOX Index

```dox
R150 child(src/core)=core engine: state machine, lifecycle, git, audit, hooks, config, agents, sessions, validation, sweeper, continuation, errors, templates, publication.
R151 child(src/commands)=CLI handlers + deps; thin delegation to core.
R152 child(tests)=Vitest suite mirroring src.
R153 child(docs)=workflow, architecture, deployment, design docs.
R154 child(.agent)=routing indices tf.ctx,file.idx,symbol.idx,spec.idx,task.idx.
R155 child(specs)=specs, gap analyses, task packs, roadmap, compact guide.
R156 root_owns := src/agent-frameworks/ ∪ src/integrations/ ∪ src/util/ ∪ src/markdown/ ∪ scripts/ ∪ tasks/ ∪ .taskforge/.
```
