# ADR-0038 — Out-of-sandbox helper for sandboxed Codex agents

## Status

Proposed for human review on 2026-11-11 (research Issue
[#394](https://github.com/rgomids/axiom/issues/394)). This is a design
proposal only: it is not in force, authorizes no implementation, and is
accepted only by an explicit decision of the maintainer (Rafael). It
supersedes no ADR.

## Context

Inside the Codex Linux sandbox (`workspace-write` or `read-only`), bubblewrap
runs the child in a user namespace where root-owned ancestors such as `/` and
`/home` appear owned by the overflow uid 65534. `ancestorSafe`
(`internal/local/platform_posix.go`) accepts only uid 0 or the effective uid,
so every Axiom command that opens local state refuses
([#387](https://github.com/rgomids/axiom/issues/387)). PR
[#392](https://github.com/rgomids/axiom/pull/392) reports that refusal as its
own diagnostic, `untrusted_ancestor_owner`, and stays as is.

The maintainer decided on 2026-11-11 (literal): "Mantém a recusa explícita e
desenha o processo auxiliar fora do sandbox à parte". Therefore:

- the ancestor-ownership rule of [ADR-0005](0005-bounded-local-filesystem-threat-model.md),
  consumed by [ADR-0007](0007-local-publication-and-recovery-protocol.md), is
  unchanged. Treating the overflow uid as root or anchoring trust at `$HOME`
  is out of scope ([options comment](https://github.com/rgomids/axiom/issues/387#issuecomment-6796782134));
- `danger-full-access` is not a mitigation;
- any helper runs outside the namespace, as the real user, with the normal
  unsandboxed trust checks.

Two callers are affected and must not be conflated:

1. **Engine-launched stage agents.** The Axiom engine already runs outside the
   sandbox and launches Codex children with `exec --sandbox workspace-write`,
   `--ignore-user-config`, `--ephemeral` and an attempt inbox
   (`internal/runtimeadapter/stage_invocation.go`). Claude children are
   granted `Bash(<axiom>:*)`, so stage agents are expected to call Axiom; Codex
   children on Linux cannot. This is the case blocking T08/K6 and AXM-37 V9/Q6.
2. **Interactive Codex sessions** started by a human outside any Execution
   (for example using the Axiom skills). No Axiom process runs beside them.

The engine already has an attempt-attributed child-to-engine channel: the
`RuntimeLauncher` `@send`/`@recv` exchange (Spec 015 S2.2,
`docs/specifications/015-minimum-viable-operational-workflow/implementation-s2-02.md`).
The engine reads bounded (5 KiB), regular, non-symlink, strictly decoded
candidates, binds each one to an engine-issued `IngressContext` derived from
the persisted attempt (the child never names its sender) and commits through
the principal writer under the runner fence
([ADR-0037](0037-one-canonical-execution-for-new-runs.md)).

## Decision

Everything below is proposed, pending Rafael's acceptance (ADR-0018).

1. **Scope: read-only first.** The helper serves only reads of the Project
   bound to the calling attempt: Project show/validate, Workflow and Work Item
   list/show and Execution/attempt status, with the same canonical JSON
   envelope ([ADR-0030](0030-canonical-results-terminal-and-host-presentation.md))
   the CLI returns. The caller cannot name another Project, a path, a state
   root or an environment variable.
2. **All domain mutations stay out.** No create/update/comment, run/start,
   delivery, provider synchronization or arbitrary command execution. No `--authorize-*` operation,
   gate decision, acceptance, cleanup, retirement, Runtime/Profile change or
   repair passes through the helper. A sandboxed agent is the subject of those
   authorities, not their holder; nothing on the channel proves a human
   decision, so allowing it would create a confused deputy. Revisiting this
   needs a separate ADR with a human-authority proof that the agent cannot
   produce (for example an engine-recorded human gate decision referenced by
   ID), not a flag forwarded by the agent. A future proof would bind the human
   principal, exact operation/objects/effects, preview digest, expected
   revisions, attempt and expiry, be issued outside the child's writable
   scope, consumed once and revalidated at commit. An ID or peer UID alone
   proves none of that. Changed targets/revisions require fresh authority;
   an uncertain result requires inspection, never automatic replay.
3. **Mechanism for engine-launched agents (recommended: option D).** Prefer
   selected reads in the stdin packet at launch. This no-channel variant is the preferred first
   step: the orchestrator selects the reads and passes a bounded snapshot,
   labelled with observation time and revisions, without accepting child
   requests. Add live read requests only if a documented stage needs fresh
   state after launch. In that case, extend the existing `@send`/`@recv`
   exchange with a closed set of read-request kinds; the engine answers into
   `@recv` using in-process reads with normal trust checks. Neither variant
   grants mutation authority.
4. **Interactive sessions: no helper now.** A human-started Codex session
   keeps the explicit `untrusted_ancestor_owner` refusal. The human may run
   the Axiom command outside the sandbox, or through Codex's own per-command
   approval if it is verified to exist (open question Q3). A per-user daemon
   or MCP server is deferred until a concrete need is recorded (revisit
   trigger below).
5. **Unchanged:** `ancestorSafe`, `trustedCanonical`, ADR-0005/ADR-0007, the
   Codex sandbox flags Axiom emits, and every existing CLI command and flag.

## Alternatives considered

| Option | How it works | Fit against #394 criteria |
| --- | --- | --- |
| A — per-user daemon, Unix socket exposed in the sandbox | Long-lived helper outside the namespace; socket path placed under a writable root or bind-mounted | Preserves the ancestor rule. Adds the largest new surface: lifecycle (who starts/stops it, stale sockets), caller authentication (`SO_PEERCRED` sees the same real uid for every process of the user, so it cannot tell a sandboxed agent from any other process; namespace identity is not a stable credential), request authority and replay, socket-path TOCTOU, a new install/config surface, and no clean macOS/Windows parity (Windows named pipes, macOS Seatbelt rules). Highest cost. |
| B — Axiom MCP server outside the sandbox | Externally launched server connected through MCP, or a Codex-spawned configured server if external placement is proven | Engine runs emit `--ignore-user-config`; relying on user registration is therefore not an established integration contract. Both config handling and server placement in 0.165 remain unverified (Q3). Needs explicit adapter configuration and tool schema; preserve the JSON result rather than invent another envelope. Medium–high cost. |
| C — Codex approved-command/escalation | Codex asks a human to run a command outside the sandbox | No Axiom code. Unusable for engine runs, which are non-interactive. When available, it runs the whole CLI unsandboxed per approved command: the human is the authority. Mechanism in 0.165 not verified locally (Q3). Suitable only for interactive sessions. |
| D — orchestrator-mediated (recommended) | Engine supplies selected reads in the launch packet without a child channel; only demonstrated freshness needs justify extending `@send`/`@recv` | Reuses bounded, fenced attempt attribution, not strong same-UID authentication. No socket, daemon or new install step; portable across engine-supported platforms. Covers only engine-launched agents. Snapshot-only has the lowest cost; live requests add protocol and audit work. |
| E — relax `ancestorSafe` | Overflow uid as root, or trust from `$HOME` | Rejected by the 2026-11-11 decision. |
| F — `danger-full-access` | Disable the sandbox | Removes agent isolation. Rejected. |

## Threat model (option D, with notes for A/B)

- **Confused deputy.** The helper holds the user's full local authority. The
  request names only an operation kind and bounded identifiers; the Project,
  state roots and attempt come from the engine-issued `IngressContext`.
  Mutations and `--authorize-*` are not representable in the message kinds.
- **Caller authentication.** D: the attempt that owns the `@send` area is the
  caller; the engine never trusts a sender field. A/B would need their own
  authentication; `SO_PEERCRED`/peer uid alone is insufficient (see table).
  A same-UID peer that discovers an inbox can write into it; attribution is
  not proof of process identity. ADR-0005 excludes arbitrary malicious
  same-UID interleavings and ADR-0028 does not provide filesystem isolation.
  No stronger authentication is claimed here. A/B would additionally require
  an out-of-band, attempt-scoped capability, private endpoint and revocation;
  namespace/PID identity is corroboration only, subject to PID reuse and
  namespace mapping. If peer isolation becomes required, this recommendation
  is invalid until a stronger OS boundary is separately approved.
- **Replay.** Reads are idempotent and change no state; each request has an
  ID unique within the attempt and duplicates return the first answer.
  Requests from ended or fenced-out attempts are not answered.
  A duplicate ID with different bytes is refused; retries never extend the
  lifetime of an attempt. Snapshot-only has no replayable request channel.
- **TOCTOU on paths.** The request carries no paths. The engine opens state
  through the existing anchored, trust-checked roots; the exchange already
  refuses symlinks, non-regular files and unsafe permissions in `@send`.
- **Argument/flag allowlist.** A closed set of message kinds with strictly
  decoded fields; unknown kinds or fields are refused. The engine never builds
  a shell command line or argv from request text.
- **Output redaction and secrets.** Answers are the same canonical JSON the
  CLI prints for approved read views, not every CLI output indiscriminately.
  Before admitting a view, review its fields for secrets, local paths and
  free-form provider/Work Item text. Exclude unsafe views rather than silently
  changing the envelope. No environment, credentials, provider tokens, raw
  logs, arbitrary detail artifacts or unrelated Project data may pass.
  Returned text is untrusted data, never instructions. Stage environment filtering
  (`StageEnvironment`) is unchanged.
- **DoS.** Existing exchange limits (5 KiB per candidate, 256 entries) plus a
  per-attempt request budget and bounded answer size; excess is refused
  without stopping scheduling.
  The plan must set numeric request/response byte, time, concurrency and total
  audit limits; no unbounded pagination or expensive validation/provider
  refresh is allowed. Project validate is eligible only as a local,
  side-effect-free inspection, never repair or network validation.
- **Lifecycle.** D has none of its own: it lives and dies with the attempt's
  message pump. A would need explicit start/stop, single-instance and
  stale-socket rules. Only the human or an explicitly configured user service
  could start A/B outside the sandbox; the child cannot bootstrap or restart
  its own privilege boundary. A/B must refuse foreign/live socket replacement,
  revoke capabilities on stop and preserve unknown files on cleanup.
- **Logging/audit.** Each request and its outcome (kind, attempt, accepted or
  refused, size) is recorded with the attempt; answer bodies are not logged.
  This is a bounded protocol/audit write, not a domain mutation authorized by
  a read. Persist through the principal writer; preserve ADR-0007 commit truth
  if delivery fails. Read handlers themselves cannot acquire mutation
  authority. Audit retention and replay cache must fit aggregate bounds.
- **Platform parity.** D is file-based and uses the same exchange on
  Linux/macOS (and Windows if support returns). macOS Seatbelt does not use a
  user namespace, so the reported failure is Linux-specific. Equivalent D
  behaviour on macOS remains a validation requirement, not native evidence.
- **Ancestor rule.** Unchanged. The helper is the engine itself, running
  unsandboxed with normal trust checks; inside the sandbox Axiom still
  refuses with `untrusted_ancestor_owner`.

## Contract impact (if accepted)

- **New public CLI commands/flags/config:** none for option D.
- **Current delivery:** only this Proposed ADR and its index entry; no
  installed docs, CLI help, skills, configuration or persisted schema changes.
  No feature is available as a result of merging this document.
- **Changed contracts:** Spec 015 exchange message kinds (new read-request and
  read-answer kinds), the stage stdin packet text that tells agents how to ask,
  and the Codex stage-agent guidance in the Runtime skills. These need a Spec
  015 amendment and plan before implementation.
  A/B would additionally require a reviewed endpoint/tool schema, lifecycle
  and opt-in configuration plus skills/help/installed-docs parity; none is
  adopted here. Snapshot-only D changes packet content rather than message
  kinds. Live D would need explicit schema/version compatibility and failure
  semantics before any implementation.
- **ADRs:** none superseded. ADR-0005/ADR-0007 unchanged; ADR-0037 gains a
  proposed extension beyond its current four message kinds. Its rule that
  messages never grant authority remains; before implementation reconcile
  that extension explicitly under ADR-0018. No second aggregate is proposed.
- **JSON envelope:** answers reuse the canonical envelope unchanged.

## Consequences

- Engine-launched Codex agents could read their own Project state on Linux
  without weakening the ancestor rule or the sandbox.
- Interactive Codex sessions keep the explicit refusal; that is a known,
  proposed limitation requiring Rafael's decision.
- Agents get a second way to read state (exchange instead of CLI); skills must
  say which applies to Codex stage agents.
- Harder to change later: once message kinds ship, their schema and the
  read-only boundary become a compatibility contract for agents.

## Revisit when

A recorded need for Axiom operations from interactive
sandboxed Codex sessions that the human cannot run outside the sandbox, or a
need for mutations from sandboxed agents.

## Open questions for the maintainer

- **Q1.** Accept option D for engine-launched agents and "no helper" for
  interactive sessions, or require a helper for interactive sessions now
  (A or B)?
- **Q2.** Is the read-only list in decision 1 sufficient for T08/K6 and
  AXM-37 V9/Q6, or are other reads needed?
- **Q3.** Codex 0.165 behaviour (approval escalation, whether MCP servers run
  outside the sandbox, and whether either works with `--ignore-user-config`)
  was not established by installed documentation. Read-only inspection of
  the resolved standalone `0.165.0-aarch64-unknown-linux-musl` package on
  2026-11-11 found only `bin/codex` and `bin/codex-code-mode-host`; a search
  of the installed package tree found no Markdown docs, config TOML or JSON
  schema. No user configuration, executable help, binary execution or vendor
  inference was used. The repository adapter proves only the flags it emits,
  not vendor support. Obtain version-matched documentation before choosing
  B/C; do not infer availability from newer online docs.
- **Q4.** Confirm that `--authorize-*` and every other authority-bearing
  mutation stay out of any helper until a separate ADR defines a
  human-authority proof.

## Validation plan (proposal)

Deterministic, with fake runners and synthetic state roots only:

- message-kind decoding: unknown kind/field, oversize, symlink, non-regular,
  duplicate ID and misaddressed request are refused without state change;
- binding: a request is answered only for the Project of its own attempt;
  requests after attempt end or fence loss get no answer;
- read-only: a test asserts no domain store write, mutation lease/CAS or
  provider call happens in the read handler; protocol/audit writes are
  separately bounded and fenced;
- negative disclosure tests cover secrets, local paths and untrusted text;
  budget exhaustion, cancellation, conflicting replay and delivery failure
  preserve existing domain state and truthful outcomes;
- parity: the answer for each kind equals the CLI's canonical JSON for the
  same synthetic state;
- regression: `ancestorSafe` tests and the `untrusted_ancestor_owner`
  diagnostic are unchanged.

Later, first a bounded native Linux check with a synthetic HOME/state and a
fake runner under a real restricted namespace: verify the external reader
sees normal ownership, the child still refuses direct state access, and
only the selected read crosses the boundary. macOS needs its own sandbox
check; Windows is not claimed supported by this POSIX proposal (see #265).
No real user state or vendor inference is needed for that step.

After separate human authority, a bounded native check on Linux: one engine-launched Codex stage
agent under `workspace-write` reads its Project state through the exchange
and is still refused when it calls the CLI directly. That check needs
separate authority because it runs real vendor inference.

## Related

- Issues [#394](https://github.com/rgomids/axiom/issues/394),
  [#387](https://github.com/rgomids/axiom/issues/387),
  [#265](https://github.com/rgomids/axiom/issues/265); PR
  [#392](https://github.com/rgomids/axiom/pull/392).
- [ADR-0005](0005-bounded-local-filesystem-threat-model.md),
  [ADR-0007](0007-local-publication-and-recovery-protocol.md),
  [ADR-0018](0018-adr-evolution-and-supersession-governance.md),
  [ADR-0028](0028-mvp-posix-process-group-supervision-boundary.md),
  [ADR-0030](0030-canonical-results-terminal-and-host-presentation.md),
  [ADR-0037](0037-one-canonical-execution-for-new-runs.md).
