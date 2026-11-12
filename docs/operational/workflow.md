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

Read a definition using the exact reference returned above:

```sh
axiom --json project workflow show --project <uuid-or-slug> --workflow <key> --revision <n> --digest <sha256>
```

Use `--source builtin` for a built-in reference. `show`, `select`, and `remove`
require all three reference fields; missing fields return `missing_required_input`
with `details` naming the first missing flag. Complete unknown references return
`workflow_not_found`. These operations never fall back to the current selection.

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
outputs require a completed successful predecessor in this Execution (any of
its bound Repositories). A schema-2 agent may declare `repository: <key>` to work
only in that bound Project Repository; absent, it works in the Work Item's.
A dependent agent also receives each same-stage dependency output as
`<agent>/<output>`.
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
<sha256:...> --note <text> --authorize-human --authorize-external`. Axiom
retains the human `--note` itself as the correlated Evidence (secret-like
content is refused); alternatively pass a pre-retained `--evidence <id>`,
exactly one of the two. Pre-retained Evidence must be successful, category/retention `evidence`, belong
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
eligibility. Cross-Runtime messages use the private canonical ingress.
For new Executions, a Stage with at least two configured writers targeting the
same Repository automatically uses isolated branches/worktrees, counting dependent
writers too. It requires configured post-merge commands and `--authorize-commands`
before launching those writers. The principal freezes each attempt's workspace
binding before creating it; writers commit their changes on the assigned
branch and leave the worktree clean. A sandboxed writer that cannot commit
(Codex keeps `.git` read-only) leaves them uncommitted: after it exits, the
principal records them as one commit on that branch; leftovers after a writer's
own commit are never adopted. The principal integrates validated results
serially into a new owned target, reruns the configured checks with the same
allowlisted environment as the attempt's own checks (`HOME`, `PATH`, `TMPDIR`,
locale; no Runtime configuration root), and records the
accepted target HEAD before unblocking dependents or completing the Stage.
An after-gated Stage with a same-Stage dependent consuming an enrolled writer
refuses before Git materialization or spawn; Axiom does not move an after gate
to partial results. To review the integrated result, put the human gate on a
separate later Stage that runs after the integrating Stage (decision recorded in
the AXM-43 implementation Evidence of Specification 015).
Other after gates wait for every agent to succeed and record a known gate wait;
actual human approval then permits integration.
Dependent Repository readers/writers receive the integrated tree. The operator's
checked-out branch is never advanced. Dirty/changed bases and branch/worktree
collisions refuse; existing paths/refs are never adopted. Conflicts and failed
post-merge checks stay `not_integrated`; uncertain effects stay `unknown` and
are never replayed automatically. All branches, commits and worktrees remain
for inspection; there is no automatic cleanup or public integration-resolution
command. Executions without enrollment retain their previous behavior.
Cancellation, failed required validation and unknown
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
`workflow delivery packet` reports validation readiness per Repository, read-only.
Status and run results expose `repositoryOutcomes`: attempt/stage/agent lineage,
artifact references and recorded source/target Git observations. Readiness
rechecks current output bytes and retained typed-artifact lineage. For recorded
integrations it also verifies local commit ancestry and the latest source/target
refs. Missing, changed or foreign evidence blocks readiness without rewriting a
completed attempt. Historical attempts and integration observations remain visible.
A Repository with no result is reported explicitly. Local completion needs no PR;
absence of an integration means no Git target is claimed. Readiness grants no
publication, merge or acceptance authority. Optional `workflow delivery advance` performs one exactly authorized push or PR
phase over `Execution.deliveries`, with intent before effect and exact observed
confirmation. Lost responses remain unknown; `--inspect` records only exact-target
observations under the original grant, without replay. That operator-authorized
inspection is the human disposition: when it conclusively observes exactly the
requested effect the delivery becomes confirmed; any other observation stays unknown. Use `axiom docs commands` for the full command reference.

Human rejection is recorded with `workflow stage reject --execution <id>
--stage <stage-id> --result-digest <digest> --note <text>
--authorize-human --authorize-local`. Status provides the exact stage digest.
Axiom retains the `--note` as the result-bound Evidence; a pre-retained
`--evidence <id>` is accepted instead and must belong to the same
Execution/stage/definition/result/Repository. Refusals name the failed
precondition: `invalid_review` (neither or both of `--note`/`--evidence`),
`result_digest_mismatch`, `review_evidence_invalid`, or `transition_refused`
when the Execution state does not admit the disposition. Sequential validation
refusals retain no new Evidence; pre-retained Evidence remains unchanged. With
`--note`, Evidence creation and the fenced Execution update are separate writes:
a concurrent state change or an update failure after creation can leave retained
Evidence unreferenced by the Execution.
The `rejected` Execution preserves successful process observations and all prior
history, while refusing further progression and delivery. An existing human
result approval is immutable and cannot be replaced by rejection.

Start linked rework through the ordinary `workflow run --rework-of <source-id>`
with the same Project, Work Item and Repository bindings, and fresh Workflow,
invocation selections and `--authorize-external`. It creates a new Execution;
`reworkOf` contains provenance only. No grants, gates, attempts, messages,
integration/delivery state or invocation snapshots transfer. Missing current
selection/authority refuses. Unknown effects cannot use rejection/rework to
bypass explicit recovery. Existing bounded retries, timeouts, leases and
`--new-selection` apply; no automatic replay, closure, merge or acceptance.

Legacy packet/publication/decision/rework producers and independent writers are
removed. Strict historical codecs and read-only attachment/inventory/retention
readers remain; they cannot start, resume or deliver an Execution. Spec 015 S3.4 records the source inventory and deletion evidence.

### Execution messages and authority

Codex and Claude use the same private attempt transport and canonical
`Execution.messages` record (`workflow.Message`). Only the principal engine
accepts candidates through `Execution.AcceptMessage`, binding sender, Runtime,
attempt and runner generation from the launcher. The child's JSON supplies only
`id`, `recipient`, `kind`, optional `replyTo`, and `content`; it cannot select
sender, generation, authority, a gate decision or an integration outcome.
Records are bounded to 256 messages and 4 KiB content each. Duplicate IDs with
identical content are idempotent; conflicting IDs and duplicate answers refuse.
Wrong Runtime/attempt/recipient, stale fences, ended askers, unknown kinds,
oversized content and post-cancel publications refuse without a state write.
The stage contract given to every child states this file shape (`replyTo` only
on answers) and that an asker keeps polling until answered; a file of any other
shape is skipped without a record. Writable attempts get a shell (Codex sandbox,
Claude `Bash`) to build, test and wait; read-only Claude attempts keep only the
Axiom CLI.

| Kind | Consumer | Effect on eligibility |
| --- | --- | --- |
| `question` | Principal pump delivers to the addressed running agent | None |
| `answer` | Pump delivers one correlated reply to the running asker | None |
| `blocker` | Recipient delivery and status/message inspection expose the diagnosis | None; it cannot cancel or approve work |
| `result` | Recipient delivery exposes reported data; engine output validation separately verifies the attempt result | Message alone never completes an attempt or unblocks a dependency |

Inspect the same stored records with `workflow coordination list` or `show`:

```sh
axiom --json workflow coordination list --project <slug-or-id> \
  --repository <key> --work-item github:<owner>/<repository>#<number> \
  --execution <id> --kind question
axiom --json workflow coordination show --project <slug-or-id> \
  --repository <key> --work-item github:<owner>/<repository>#<number> \
  --execution <id> --child <sender-agent-id> --record <message-id>
```

`--child` is a sender-agent filter, not a child Execution. `--attempt` filters
sender attempts on list. Old nonzero `--after`/`--wait` cursors refuse. The
historical `coordination publish` spelling always refuses with
`child_context_required`; it is no alternate ingress. Children write only into
their launcher-issued area; the principal alone commits canonical state.
Same-UID filesystem access is not OS isolation (ADR-0028).

Only explicit human authority records configured before/after gates or exact
result review. Unknown effects/cancel/failure stop successors regardless of
message content. Enrolled same-Repository writers integrate through `workflow run`
under the principal's fence, with expected-head CAS, retained checks and truthful
unknown/recovery. Messages cannot integrate, dispose an unknown outcome, push
or open a PR. There is no separate public integration or integration-resolution
command. Repository delivery uses the canonical
Execution operation described above.

The retired format-1 stream has no writer, store, acceptance verifier or runtime
packet producer. Its read-only decoder survives solely for historical upgrade
inventory; it cannot feed new Executions. Old bytes remain untouched and cannot
resume. Cutover backup and disposition still require separate authority.

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
