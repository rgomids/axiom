# Workflow

A Workflow definition describes stages and their configuration. A Project may
select a definition; creating or editing a definition does not select it.
Inspect available definitions and current selection before planning work:

```sh
axiom --json project workflow list --project <slug-or-id>
axiom --json project show --selector <slug-or-id>
axiom --json project workflow create --project <slug-or-id> \
  --from-default --workflow team-sdd
```

Use `axiom project workflow ... --help` for the installed command's exact
inputs. Validate a complete definition before proposing create/edit. Review the
preview, effects, revision and digest. Apply by repeating the same operation
with its exact returned authority values and `--authorize-local`. Selection is
a separate explicit operation with its own preview and authorization.

`axiom workflow run` resolves the existing linked Work Item and selected Workflow
(or the minimal work stage), then progresses eligible work through one Execution.
It needs `--authorize-external` before spawning a Runtime. It does not require a
Stage Plan, Validator Policy or named Profile. Explicit per-agent fields override
Workflow values, then explicitly configured machine-local defaults:

```sh
export LINGO_INVOCATION_DEFAULTS='["runtime=codex","model=my-model","effort=runtime-default"]'
axiom workflow run --project <slug-or-id> --repository <key> \
  --work-item github:<owner>/<repository>#<number> --authorize-external
```

Defaults are a bounded JSON array of `field=value` strings; fields are `runtime`,
`model`, `effort`, `instructions`, and optional `profile`. Missing or conflicting
selections refuse. Instructions from the Workflow remain above local defaults.
The effective invocation and provenance freeze per attempt. Ordinary retry retains
that tuple, even if local defaults change; a fresh selection needs
`stage retry --new-selection`. Writable retries also require an inspected workspace
and `--disposition preserve`; no automatic reset is supported.

Required outputs must be present; declared optional outputs are validated when
present and may be absent. Work Item inputs come from `target`; earlier-stage
outputs require a completed successful predecessor in the same Repository.
`project-context` with source `project` freezes portable Project identity,
revision and Repository identities, without local paths. Artifact inputs freeze
retained successful artifacts with exactly one matching Repository, source stage
and source attempt reference, plus the source Execution identity, selected by
`--input <declared-source>=<retained-artifact-id>`. Source stage and attempt use
`workflow-stage` and `workflow-attempt`; missing or duplicate provenance refuses.
Earlier-stage input packets retain their producing Execution/stage/attempt IDs.
Undeclared bindings, missing
required inputs and unsupported policies refuse rather than disappearing.

A `human-review` validator waits for the exact successful stage result. Inspect
`resultDigest`, then after actual human review continue with
`workflow run --execution <id> --review-stage <stage> --result-digest
<sha256:...> --evidence <retained-id> --authorize-human --authorize-external`.
The retained Evidence must be successful, category/retention `evidence`, belong
to this Execution and correlate exactly once with `workflow-stage`,
`workflow-definition` (prefixed definition digest), `workflow-result` (the exact
result digest) and `workflow-repository`. Agent messages cannot grant approval.
Result review is distinct from before/after human gates: gates use
`--approve-gate <stage> --authorize-human`, and a before-gate starts no attempt.

Schema-2 `command` validators under `builtin-sdd-v1` declare bounded `argv`,
`timeoutSeconds` and `effectCeilings`. Running them additionally requires
`--authorize-commands` on run or retry, authorizing those exact configured
arguments and effects. No shell expansion, fallback command or implicit authority
is supplied. The frozen attempt retains selections and authority; ordinary
retries preserve the invocation and require current effect authorization.
Independent ready tasks run concurrently; the principal serializes their
Execution state commits. Dependencies and configured human gates still control
eligibility. Cross-Runtime collaboration transport and managed worktree
integration remain pending S2 work. Cancellation, failed required validation and unknown
effects stop successors. See the [command reference](cli.md) for installed flags.

Inspect `workflow stage status --execution <id>` after an attempt. Cancel records
intent and reports the observed stop separately; interrupting the foreground
runner (Ctrl-C, SIGTERM, hang-up) stops the agent's own process group and
records that stop. Reconcile never blindly replays unknown effects, and
abandoning an unknown attempt is refused while any recorded process is alive or
unobservable (stop it first). `workflow status`, `evidence` and `list` read the same canonical aggregate. Old
lifecycle gate/fact/Provider projection writers and Graph/Single dispatch are
retired. Old bytes are inert: there is no legacy resume or conversion. Public
child publication remains unavailable; S1 private ingress belongs to the engine.
`workflow delivery packet` reports canonical readiness read-only. Delivery
publication, acceptance and linked rework effects remain S3 work.

## Breaking cutover: stop or block

S1.8 certification depends on completing the S1.7 structural retirement and
required checks. No real installed-state switch is authorized by these fixtures.
The inventory/backup helpers are internal and tested only with isolated state;
no public cutover or migration command exists. At a separately authorized
deployment boundary:

1. Stop new writers and identify the exact binary/source, Project and state
   roots from deployment records. Establish all owned processes, leases,
   active/unknown/incomplete attempts and unresolved publication/effect intents.
   Missing, unreadable, linked or unexpected state makes the inventory incomplete
   and BLOCKS switching. An absent lease does not prove an attempt stopped.
2. Preserve raw legacy bytes privately outside the state root, with permissions,
   sizes and hashes. Legacy bytes include retired capability evidence and probe
   envelopes. Probe cleanup that stopped part-way BLOCKS. Projects, Work Items,
   workflows, Profiles, artifacts and Project context are live state: they
   stay in place and are not part of this backup. Re-read every backup byte and compare it to the manifest;
   a manifest alone is insufficient. Helpers compare the complete source inventory
   before and after copying and verify the complete backup inventory. Omitted or
   added files, changed source bytes, linked/unreadable backups or interrupted
   publication BLOCK. A failed copy can leave a partial backup; preserve it as
   unverified and choose a separate empty destination for another authorized copy.
   Never commit private state or
   interpret a backup as permission to execute it.
3. Stop only positively identified owned processes under actual authority,
   preserving PID/start/group correlation and escape limits. Confirm stopped
   observations and disposition every uncertain local/remote effect. Unknown
   effects or an unestablished process boundary BLOCK; never signal a stale PID
   or automatically repeat a remote mutation.
4. Record human disposition for the exact inventory and verified backup (or an
   explicit reason preservation is infeasible). Backup alone never establishes
   stopped work. Only after a complete safe boundary may new-format-only starts
   be enabled. Old format resume/retry is refused; no migration, conversion or
   compatibility engine is provided.
5. Before switching, rollback requires safe stop and separate authority to restore
   the prior binary/state. After new effects exist, preserve both namespaces and
   prefer a forward fix. Never point an old binary at new state, restart old
   attempts or replay effects from backup.

See [Work Item](work-item.md) and the [command reference](cli.md).
