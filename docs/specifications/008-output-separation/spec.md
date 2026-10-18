# AXM-32 — Canonical results, terminal output and host conversation

## Status and authority

**Proposed — specify only, 2026-10-18.** Related Work Item:
[AXM-32](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia).
This proposal does not replace accepted contracts until an explicit human decision
identifies the exact reviewed revision and scope. It authorizes no Plan, Tasks,
implementation, Runtime invocation, Provider mutation, acceptance or release.
[ADR-0030](../../decisions/0030-canonical-results-terminal-and-host-presentation.md)
proposes the durable boundary; it also remains Proposed.

The supplied gate report says READY_FOR_specify, Linear Todo, no Blocked by;
these are intake facts, not independently observed local Execution truth.
Canonical Execution, revision, current gate and concurrent execution remain
unobservable in this session. No Execution is reconstructed or advanced, no
derived stage or acceptance is inferred, and no Linear metadata is changed.

## Problem, outcome and actors

The maintainer reports that personal testing of v0.19.0 on 2026-10-18 exposed
verbose technical Markdown, internal structures, Provenance, Auxiliary explanation
and JSON dumps, plus “Succeeded” while `project validate` reported an Execution
blocker. This is a reported experience, not a captured real JSON event reproduced
by this change. Existing code explains how the experience is possible; it does
not establish the precise facts of that particular Project.

Desired outcome: one authoritative structured operation result, deterministic
native terminal presentation by Lingo, and concise Markdown synthesized by the
already operating AI in its own host. The human can distinguish what happened,
what remains blocked and which decision or action is needed without interpreting
an internal dump. Automation retains a complete, stable machine contract.

Actors are the human operator, direct CLI user, automation consumer, operating AI
in Codex/Claude/future hosts, application/domain result producer and Evidence
reviewer. Axiom is the product; Lingo is its local control plane. Application/domain
contracts own facts and authority; neither terminal nor AI presentation owns them.

## Confirmed baseline and conflicting contracts

Investigation baseline is `14cb721ac7b8ce09a7f20f2bc2175adf61b7934b`
(v0.19.0); remote main matched it at inspection. Relative source links below refer
to this baseline unless changed by a later reviewed delivery. Public GitHub state
was read on 2026-10-18. No private provider text is imported.

| Source/interface | Confirmed fact | Proposed change / preserved boundary |
|---|---|---|
| [#232](https://github.com/rgomids/axiom/issues/232), [PR #291](https://github.com/rgomids/axiom/pull/291), [historical Evidence](../004-mvp-v1-baseline/evidence-232.md) | Delivered, explicitly human-accepted 2026-10-09, stable v0.13.0. Default deterministic Markdown and literal skill relay were intentional | Replace those presentation requirements prospectively; preserve acceptance, Evidence, provenance, authority and no extra formatting LLM call |
| [presentation.go](../../../internal/cli/presentation.go), [completion.go](../../../internal/cli/completion.go) | JSON event is rendered into Markdown by default; every field is printed; oversized human output falls back to fenced canonical JSON; status label for `success` is Succeeded | Native terminal renderer; no mandatory dump or Markdown fallback. JSON stays complete |
| [CLI parser/emission](../../../internal/cli/cli.go), [commands](../../commands.md#cli-output-and-help) | Leading `--json` selects JSON, leading `--human` explicitly selects default human output; default is human, including pipes. Some legacy POC events differ from MVP completion | Preserve existing flags and error behavior pending D1/D2; audit legacy shapes rather than claim an already uniform schema |
| [Spec 007 WF-013 and public DTOs](../007-configurable-workflows/spec.md#public-operations-and-boundary-dtos-wf-009011013) | Complete JSON; status success for read/preview is operation success, not Workflow success; `blocked`/`unknown` must not become completion statuses | Preserve status taxonomy, payload truth, ownership, revisions, lifecycle and authority; replace reliance on #232 presentation only |
| [AXM-31 amendment](../007-configurable-workflows/amendment-axm31-operator-autonomy.md), [accepted Plan](../007-configurable-workflows/plan-axm31-operator-autonomy.md), [Evidence](../007-configurable-workflows/evidence-axm31-operator-autonomy.md), [PR #325](https://github.com/rgomids/axiom/pull/325) | PR merged 2026-10-18 at `5b6bd0b354ec38794be33fd8847333bf5132049f`; diagnostic schemaVersion 1 and literal ordered relay with separately labelled auxiliary prose were integrated | Change only presentation/consumption obligations. Preserve bounded diagnostics, observed/not-observed distinctions, budgets, no automatic authority or retry |
| Four [embedded skills](../../../internal/codexruntime/skills/axiom-project/SKILL.md), [Workflow](../../../internal/codexruntime/skills/axiom-workflow/SKILL.md), [Work Item](../../../internal/codexruntime/skills/axiom-work-item/SKILL.md), [Runtime](../../../internal/codexruntime/skills/axiom-runtime/SKILL.md) | Skills request default human view, relay it exactly and permit JSON only on explicit user request; three carry Auxiliary presentation sections | All four must consume canonical JSON internally and use host Markdown; explicit raw JSON request remains supported |
| [Readiness emission](../../../internal/cli/readiness.go), [application operation](../../../cmd/lingo/project_lifecycle.go), [readiness contract](../../../internal/projectapp/readiness.go) | Structure and readiness are separate; structurally valid validation completes successfully even if Execution is blocked. Readiness is read-only and grants no authority | Show both facts, with precise operation-specific blockers; do not change validation status/exit semantics |
| [Presentation fixtures](../../../internal/cli/testdata/presentation), [golden tests](../../../internal/cli/presentation_test.go), [skill contract tests](../../../internal/codexruntime/runtime_test.go), [auxiliary contracts](../../../internal/codexruntime/workflow_skill_content_test.go), [routing tests](../../../internal/cli/domain_skill_routing_test.go) | Tests pin Markdown bytes, explicit JSON routing restrictions and relay instructions | Future tests must assert new approved contracts, preserving old fixtures as historical evidence where required; do not mechanically regenerate away semantic differences |
| [Skill revision history](../../../internal/codexruntime/shared_history_test.go), [skill validation script](../../../scripts/test-codex-skills.sh), [prepared smoke](../../../scripts/smoke-prepared-artifact.py) | Installed skill compatibility, scripted consumers and release smoke are additional migration surfaces | Inventory callers, receipts, digests, snapshots and parser assumptions before implementation; immutable published pins and state corpus stay unchanged |
| [Tracking governance](../../development/linear-github-issue-tracking.md) | Repository technical truth, local Execution workflow truth and Linear operational status have distinct authorities | Preserve all boundaries; a clearer message cannot grant acceptance or advance a gate |

Accepted architecture checked: [ADR-0003](../../decisions/0003-lingo-as-axiom-local-control-plane.md),
[ADR-0006](../../decisions/0006-machine-local-detail-artifacts.md),
[ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md),
[ADR-0025](../../decisions/0025-public-stage-dispatch-and-capability-proof.md).
No accepted ADR fragment requiring supersession was identified: central control,
detail artifacts, skill ownership and dispatch authority remain intact. If later
review identifies a conflict, acceptance must name its exact fragment and use
[ADR-0018](../../decisions/0018-adr-evolution-and-supersession-governance.md).
Do not annotate an old ADR as superseded while this proposal is Proposed.

Hypothesis: exhaustive field relay contributes to the reported verbosity.
It is not evidence that every emitted field is unnecessary, nor proof of the
reported Project's Runtime policy. Gaps in JSON, if found, are application
contract gaps to review explicitly, never facts for the host to invent.

## Scope, invariants and non-goals

Scope covers result consumption and presentation for existing canonical CLI
operations and the four installed domain skills in Codex and Claude; future hosts
inherit the same behavior only after supported integration and validation.
Help/version and legacy events must be classified separately during migration;
this proposal does not silently promise a new schema for every text entrypoint.

Preserve Axiom != Lingo, Project != Repository, Role != Model, Runtime != Model,
Execution != Agent, Skill != workflow source of truth, Evidence != raw chat.
Preserve all operation semantics, status classification, consent, preview/apply,
expected revisions, immutable bindings, dispatch proof, retry/recovery budgets,
Provider boundaries and accepted history. Readiness never grants authority.

Non-goals: implement renderers/adapters/tests, edit domain/status/authorization,
fix [#305](https://github.com/rgomids/axiom/issues/305), author a missing policy-edit
operation, run a Runtime, alter integrations/infrastructure, reopen #232, accept
an ADR, merge, release, close AXM-32, remove dependencies or claim
[AXM-12/#278](https://github.com/rgomids/axiom/issues/278) acceptance.

## Observable journeys

1. A direct CLI user inspects a Project. Native text states the operation result
   and material exceptions; explicit JSON yields the complete same observation.
   Piping the result cannot add effects or change its facts.
2. A human asks the operating AI whether a Project can execute work. The skill
   consumes JSON, separates structural validity from operation readiness and
   reports the actual Execution blocker. Missing policy-edit capability remains
   visible as #305; presentation does not repair or bypass it.
3. A human requests a mutation. The skill reads canonical preview effects,
   revision and digest, presents the material decision and preserves exact consent
   binding. A digest change requires renewed authorization, even if the summary
   would look identical. Confirmed apply effects are checked by canonical read-back.
4. A Provider acknowledgement is uncertain or an operation is interrupted after
   an effect. Both surfaces preserve confirmed/unknown effects and recovery refs;
   the host proposes inspection/handoff within existing budgets, without replaying
   the mutation or claiming the requested end state.

## Behavioral requirements (proposed)

### R1 — One canonical structured result

For the same operation observation there MUST be one canonical structured result.
JSON MUST be a complete, bounded, parseable serialization of that result, independent
of host, terminal width, color, locale, TTY and prose style. No rendered Markdown,
terminal layouts, ANSI, host-specific message blocks or duplicated human-result
tree may become authoritative JSON fields. Existing `result`/`next` text is retained
as canonical data for compatibility; it is not a layout instruction or a required
conversational sentence. No parallel result source is introduced.

It MUST preserve effective completion status, operation/target identity, confirmed
effects, uncertain/unconfirmed effects where known, errors/categories, warnings,
blockers and their scope, references, consent/authority conditions, revisions,
digests, provenance and Evidence/detail references. Absence of a field MUST NOT
be interpreted as evidence of no effects, authority, readiness or success.
Unsupported/missing facts remain explicitly unknown to the consumer.
Operation identity currently follows invocation and typed payloads in some MVP
events; no new `operation` field or envelope version is approved by this proposal.
An additive field or schema migration needs explicit compatibility review (D3).

Existing field names, types, enum meanings, omission rules, limits and diagnostic
schemaVersion 1 MUST remain compatible unless a separately reviewed change names
the affected consumers. No prose parsing may substitute for typed status,
readiness, effects, revisions or digests. JSON retains every applicable technical
detail, even when presentation summarizes it. Oversized/unsupported output must
fail visibly, never silently truncate or emit a success-shaped partial JSON value.

### R2 — Native human CLI

Lingo MUST render the same result deterministically as native terminal text,
without Markdown headings, emphasis markers, code fences, Markdown tables or a
fenced JSON dump. Literal user values containing such characters are data and
must remain distinguishable from renderer syntax. Present the operation and
effective result first, followed by material blockers/warnings, confirmed or
uncertain effects, authority/decision and next action. Internal implementation
structures MUST NOT be expanded by default.

TTY may enable color/layout only when supported; all meaning MUST survive
plain text. `NO_COLOR` when present MUST suppress color. Non-TTY pipes,
redirection and terminals without ANSI support MUST receive plain static output,
without cursor movement, spinner, progress rewrites or essential color cues.
Unknown terminal capability MUST degrade to plain output. Width changes may wrap
prose but MUST NOT alter/crop identifiers or digests. Machine consumers select
JSON explicitly; human text is not a new machine parsing contract.

`--json` MUST keep its leading global selection and complete machine semantics;
`--human` MUST remain an explicit human selector, proposed to mean native terminal
text. No new `--terminal`, `--output`, auto-JSON rule or flag precedence is assumed.
D1/D2 below require human decision before this migration is authoritative.
Invalid/conflicting/repeated or misplaced flags retain current refusal behavior
unless the approved migration explicitly changes it. Help remains deterministic
human text even with output flags; it is not an operation result to parse as JSON.

In JSON mode stdout MUST carry exactly one result document for a completed
canonical operation, with no prompts, banners, progress or renderer prose.
Prompts/transport diagnostics must stay separate from the result channel; an
unavailable/truncated result MUST be reported as unconfirmed, not fabricated.
Redirection/write failure must return failure and MUST NOT trigger another
execution of the operation. Stderr handling cannot conceal an already confirmed
effect. Selection MUST NOT cause another application call to obtain a second view.

### R3 — Host-owned conversational Markdown

All four skills MUST request the existing explicit JSON mode for internal result
consumption, whether entry was natural intent or explicit skill invocation.
The operating AI MUST use the resulting canonical data to compose its normal host
Markdown response in the order: objective → effective result → material
blocker/decision/risk → next action. Empty categories need no boilerplate heading.
No dedicated second LLM formatting call, new model, formatter agent or runtime hop
is allowed solely for presentation. A user asking for raw JSON receives the
canonical JSON without rewriting; that is separate from internal consumption.

The host MUST NOT relay a mandatory Lingo dump or append a compulsory Auxiliary
explanation after it. It may omit nonmaterial technical detail, repeat neither
payload trees nor generic Provenance footers, and group equivalent warnings only
when their scope and consequence remain clear. It MUST disclose every material
confirmed/uncertain effect, warning, blocker, authority constraint and provenance
qualification; retain exact references, revision and digest needed for the next
operation. It MUST NOT hide an additional blocker behind one convenient cause.
Detailed canonical provenance and Evidence must remain retrievable; source state
dirty/unknown or mismatched revision material to trust MUST be stated.

Consent MUST bind the actual canonical preview and digest; paraphrase, a diagram,
“yes” to a summary or a previous approval cannot substitute for exact authority.
Read-back/revalidation remains required. The AI MUST copy identifiers/digests from
typed JSON, never reconstruct them from prose or display abbreviations. It MUST
distinguish recommendation/hypothesis from observed facts and state when no
supported next operation exists, including #305. Presentation does not authorize
remediation, retry, Runtime invocation or Provider mutation.

Malformed/unsupported/truncated JSON, mismatched target, missing required status
or contradictory observations MUST stop dependent actions and produce a bounded
uncertainty/handoff message with known refs and the smallest safe inspection.
Do not rerun a mutation to get another view, infer success from exit code alone,
or reconstruct missing local Execution truth from a conversation.

### R4 — Status, readiness and effects remain separate

Keep completion statuses and exit codes: success → 0; interrupted → 2; failure,
validation_failure, denied_authority, partial and retryable_failure → 1. No new
`blocked`, `no-op` or `recovery` completion enum is introduced. No-op is an
operation outcome under the existing status; recovery is a condition/next step.
Partial completion retains confirmed effects even when the command exits 1.
Neither CLI nor host may promote technical success into Workflow/product acceptance.

For `project validate`, if JSON reports `readiness.structure=valid` and an
`execution` entry in `readiness.operations` is blocked, both human surfaces MUST
state structural validity AND blocked Execution. `readiness.effective=partial`
can coexist with that blocker when Work Item operations are ready. Do not call the
whole Project invalid, or say all operations are blocked. Preserve status success
and exit 0 for successful structural validation; this specification changes no
readiness algorithm or domain result.

“Project estruturalmente válido; execução bloqueada por política de Runtime” is
permitted if and only if the actual JSON confirms both structural validity and
the Execution blocker attributable to that policy (for example
`runtime_policy_unavailable`). `runtime_resolution_blocked` alone is insufficient
to diagnose a policy cause without supporting payload. Other or multiple blockers
must be described with their actual scope. Missing readiness means readiness is
unobserved, not ready. No actual test-report JSON for the personal incident is
available here; the phrase is a conditional acceptance example, not its diagnosis.

## Semantic equivalence and measurable acceptance

Equivalence means preservation of material facts and exact actionable tokens,
not byte equality across JSON, terminal and conversation. JSON is lossless;
terminal is deterministic; conversation can paraphrase within R3. For each row,
future fixtures MUST compare typed facts and assert zero additional effects or
authority caused by selecting presentation.

| Case / canonical facts | Native CLI obligation | Host conversation obligation | Exit |
|---|---|---|---|
| success with confirmed effect | Operation and exact effect/target | Goal achieved only within confirmed scope; next action if needed | 0 |
| success with no-op | Explicit already-equivalent/no change | No new effect claimed; no new consent needed solely for no-op | 0 |
| partial, effect confirmed plus secondary failure | Both effect and failure; recovery refs | Explain what happened, what remains and safe inspection/recovery | 1 |
| failure or validation_failure | Actual error/category and no unsupported effect claim | Explain refusal/error and smallest supported correction | 1 |
| denied_authority | Denial and exact required preview/consent | Ask only the missing material authority; no automatic grant | 1 |
| interrupted or uncertain outcome | Interruption and known/unknown effects | No blind retry; inspect/reconcile using retained refs | 2 for interrupted; otherwise existing status mapping |
| retryable_failure | Retry-safe only if canonically established | No retry expansion beyond existing budgets/authority | 1 |
| recovery-required condition | Preserve underlying status and recovery scope | No completion/acceptance claim; handoff with observations and revisions | Underlying status |
| valid structure, execution blocked, Work Item ready | Structural validity, Execution blocker and Work Item readiness | Conditional R4 wording; no policy edit invented, no workflow run implied | 0 |
| missing/unsupported JSON or failed output write | Transport failure distinct from domain outcome | Unconfirmed result; no dependent mutation; known effects retained | Failure for output write; host does not recode domain status |

Acceptance criteria:

- **AC1:** Each canonical event has one parseable JSON document with all applicable
  facts and unchanged status/exit semantics; presentation makes zero extra use-case
  calls. Existing consumers parse old-compatible fields without prose extraction.
- **AC2:** For every matrix row, terminal output has zero renderer-generated
  Markdown constructs and zero loss of material facts; repeated equal input and
  terminal conditions produce identical output. Plain text retains every decision.
- **AC3:** All four skills in both Codex and Claude consume JSON internally;
  assertions reject mandatory exact Markdown relay and labelled duplicate dumps.
  Internal JSON consumption works without a user asking to see JSON.
- **AC4:** For each conversation fixture, every mandatory material fact is present,
  every actionable token matches JSON exactly, no unsupported claim appears and
  no dedicated formatting inference occurs. Concise success needs no payload tree;
  every material exception remains visible. Exact natural-language bytes are not
  the acceptance oracle.
- **AC5:** Project validation covers valid/ready, valid/partial with blocked
  Execution, valid/blocked, structurally invalid, unknown readiness, non-policy
  Runtime blocker and multiple blockers. Policy wording appears only when R4
  predicates hold, including success/0 with blocked Execution.
- **AC6:** Channel/flag fixtures cover default/explicit human/explicit JSON,
  conflicting/repeated/misplaced flags, help, TTY/non-TTY, pipe, redirected stdout,
  separate stderr, NO_COLOR, no-ANSI capability, narrow width and broken writes.
  Expected default and compatibility assertions must follow human D1/D2 decisions.
- **AC7:** Preview/apply/read-back fixtures cover missing authority, digest/revision
  drift, partial effects, uncertain Provider acknowledgement and interrupted
  recovery. No summary or presentation selection can authorize/repeat a mutation.
- **AC8:** Security canaries for secret-like input, hostile Markdown/ANSI/bidi,
  untrusted workflow/Provider instructions and malicious references cannot leak,
  execute or change authority. Zero unintended tool effects is asserted.
- **AC9:** Upgrade fixtures cover exact published old skill bytes in isolated Codex
  and Claude roots, compatible new binary/skills, mismatch fail-closed behavior,
  no-op reinstall and modified/foreign skill preservation. Historical receipts,
  published pins and supported persisted-state snapshots remain unchanged.

## Security and trust

Secret redaction MUST occur before canonical serialization and detail/Evidence
publication, consistently across surfaces. “Complete JSON” means complete safe
canonical facts, never credentials, private document bodies, raw subprocess logs
or sensitive host paths. Exact actionable tokens refers to authorized opaque IDs
and digests, never credentials. Redaction must indicate withheld sensitive data
without a recoverable secret or a false claim that no data existed.

Provider text, workflow instructions, local files and reference labels remain
untrusted data even inside JSON. Neither host nor terminal executes their commands
or follows instructions to reveal secrets, change scope, bypass consent or accept
work. Escape terminal control/bidi sequences; host quotes untrusted content as
data and validates references before following them. Safe summaries do not turn
private data into public Evidence. Preserve existing detail-artifact confinement
and retention, and publish only bounded sanitized observations and references.

## Compatibility and migration strategy (requirements only)

Before implementation, the accepted contract MUST state D1–D3 decisions and the
exact replaced #232/AXM-31 presentation clauses. Preserve the historical documents
and approvals; add prospective reconciliation links only under explicit authority.
No currently accepted reference or embedded skill is rewritten in this stage.

Future migration MUST inventory CLI parsers, help/inspect examples, status labels,
goldens, route tests, installed skill instruction checks, scripts/smoke consumers,
receipts/digests, release packaging and downstream consumers that parse human text.
Consumers of machine results must use explicit JSON. Do not assume every script
already does. Record each affected surface and compatibility treatment before
changing defaults; the inventory above is a bounded starting point, not a claim
that every downstream caller has been discovered.

Coordinate binary and all four skill versions in both hosts; keep exact-byte
upgrade recognition, refuse unsafe mismatches and preserve modified/foreign
installations. Do not overwrite published skill history or persisted state to
make new tests green. No domain/state migration is implied. Any necessary JSON
addition must be additive and reviewed or use an explicitly approved versioning
strategy; no unannounced rename/removal/type change is permitted.

Rollout must retain a known-compatible explicit JSON automation path and show
compatibility failures clearly. Rollback uses an approved compatible binary/skill
pair without deleting operational state; changed human output must not break
automation silently. Exact release/deprecation window is a human D2 decision,
not implementation authorization from this document.

## Evidence and validation boundaries

[Specify Evidence](evidence.md) records investigation, checks and limits for this
documentation change. Future implementation Evidence MUST map R1–R4 / AC1–AC9 to
named tests, sanitized canonical fixtures, semantic projections and exact revisions
and skill digests. Preserve JSON/terminal goldens plus concise conversation examples
and negative assertions; record command, exit code, target identity, observed
effects and authority, not raw chat. Safety/authority assertions must be
deterministic where possible; model judgment alone cannot validate facts.

Synthetic installed-skill journeys MUST run against isolated Codex and Claude
roots, prove JSON consumption and preview/read-back invariants, and keep zero real
Provider/Runtime effects. They cannot prove native host compliance. Separately
authorized native conversational observations in both hosts must check ordinary
success, blocked validation, denied consent and interrupted/partial recovery,
with bounded sanitized evidence and no extra formatting call. Future hosts require
their own evidence. AXM-12/#278 retains E2E/product acceptance; passing this matrix
does not implicitly satisfy or unblock it.

All AC1–AC9 future behavior/integration tests and native observations are
**NOT_RUN: implementation is outside this specify stage**. Current documentation
checks prove document integrity only, not delivered behavior or human acceptance.

## Open human decisions and next gate

| Decision | Objective question | Recommendation and consequence |
|---|---|---|
| D1 — default/channel selection | Should no-flag output remain human in TTY AND pipes/redirection, with `--human` selecting terminal text and `--json` selecting JSON? | Yes; preserve current selection, change only human syntax. Auto-JSON in pipes is a viable alternative but changes existing no-flag pipe behavior. No such default change is inferred here |
| D2 — historical presentation replacement | Approve prospective replacement of #232 default Markdown/exhaustive relay and AXM-31 intact-relay/Auxiliary presentation, immediately in one compatible binary/skill release or through an explicit compatibility window? | One coordinated release, retaining both existing flags and stable JSON. A window reduces consumer disruption but prolongs dual instructions. Human must choose timing and whether any legacy human mode is required; none is invented |
| D3 — canonical contract coverage | Can existing typed payloads plus invocation identity carry all R1 facts, or must specific gaps be added under separately reviewed JSON evolution? | Preserve current wire shape and application semantics; inventory gaps before Plan. The missing incident JSON prevents its exact diagnosis but does not prevent specifying conditional behavior |

D1/D2 block an authoritative selection/migration contract and implementation;
D3 requires explicit resolution of any material gap before designing a schema
change. These questions do not prevent review of the completed specify proposal.
Next gate is explicit human review/clarification and acceptance or revision of
this exact Specification/ADR proposal. No progression to Plan/Tasks is performed.

Constitution check: intent precedes implementation (I–II), source/decision/Evidence
links are durable and distinguish proposal from accepted history (III–IV), typed
equivalence and deterministic checks precede model judgment (V), authority and
redaction remain boundaries (VI), scope adds no runtime or dependency (VII), and
Project/Repository ownership remains unchanged (VIII). No constitutional waiver
or product acceptance is claimed.
