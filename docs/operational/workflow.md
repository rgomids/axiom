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

Supported before-gates require actual human approval before any attempt starts;
after-gates require successful outputs first. Use the returned next action for
that stage. Configured validators currently refuse before Execution creation
until their exact controls are implemented. Parallel scheduling, collaboration
and integration are not delivered by this serial path.

Inspect `workflow stage status --execution <id>` after an attempt. Cancel records
intent and reports the observed stop separately. Reconcile never blindly replays
unknown effects. Pre-cutover lifecycle, coordination and delivery commands still
remain in this partial implementation and do not operate on the new aggregate.

## Breaking cutover: stop or block

Do not switch an installed state root using this partial implementation. The
read-only inventory/backup helpers are internal and tested only with isolated
fixtures; no public cutover or migration command exists. At a separately
authorized deployment boundary:

1. Stop new writers and identify the exact binary/source, Project and state
   roots from deployment records. Establish all owned processes, leases,
   active/unknown/incomplete attempts and unresolved publication/effect intents.
   Missing, unreadable, linked or unexpected state makes the inventory incomplete
   and BLOCKS switching. An absent lease does not prove an attempt stopped.
2. Preserve raw legacy bytes privately outside the state root, with permissions,
   sizes and hashes. Re-read every backup byte and compare it to the manifest;
   a manifest alone is insufficient. Changed source bytes, linked/unreadable
   backups or interrupted publication BLOCK. Never commit private state or
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
