# Specification 012 — Conversational Project and Stage presentation

## Status, scope and authority

Revision 1, 2026-10-23. **Specification output under approved Intake baseline;
not implementation or product acceptance.** Owner: [AXM-39][card]. Baseline:
`AXM-39:baseline:rev-2026-10-19T05:08:35.105Z`, explicitly approved by the
maintainer and recorded in [decision.approved][decision], followed by
[intake.completed][intake] and [READY_FOR_specify][gate]. All six original AC
and both approved clarifications remain intact. Only their authorized public
product intent is incorporated; no private transcript or installation data.

This document specifies presentation, not a replacement for [Specification
007][007]. Canonical fields, domain semantics, CLI JSON/text/flags, selection,
Execution authority, digests and historical Evidence retain their owners.
`project list/show/validate` below means conversational explanation of their
results, including natural-language entry; it does not redesign CLI output.
This attempt ends at Specify: no Plan, Tasks, code, skill changes, new ADR,
Runtime invocation, inference, credentials, merge, release or Done.

Origins on every material rule: `CARD` links to [six AC][card] plus the
[approved clarifications][decision]; `ACCEPTED` links to its existing contract.
Tables elaborate that baseline, not additional product AC. Examples are
synthetic expected behavior, never observed test outcomes. [Evidence](evidence-specify.md)
records technical choices, current dependency observations and limitations.

## Problem, actors and journeys

The card reports UUID-heavy Project responses and flat lists of Stage fields.
People need to recognize the Project, understand who does what, distinguish
configuration from execution and find the next safe action without reading a
payload. This report is not reproduced native-session Evidence here.

Actors: human operator; operating AI explaining canonical results; Lingo owning
facts/operations; host displaying responses; reviewer comparing source and
sanitized Evidence. Project is a product boundary, not a Repository [ACCEPTED:
007][007]. Multiple Repository bindings do not rename the Project.

| Journey | Observable result | Origin |
| --- | --- | --- |
| J1: “List Projects”, then “show/validate this one” | Recognizable label, unambiguous target, honest readiness; no automatic selection/mutation | CARD AC1/4 [AXM-39][card]; ACCEPTED [008][008] |
| J2: “Explain intake/implementation”, then “details of this Agent” | Essential operational summary, then requested configuration with provenance | CARD AC2/3 [AXM-39][card]; ACCEPTED [007][007] |
| J3: “Codex or Claude? Is it ready?” | Allowed, declared, selected, resolved and observed are separate facts; blocker and next action survive synthesis | CARD AC3/4 [AXM-39][card]; ACCEPTED [RA-1–4][ra] |
| J4: “Show the Workflow”, then inspect a Stage | Retain AXM-38 diagram and identity; concise text expands the same canonical definition | ACCEPTED [011 VP-01–15][011]; CARD AC2 [AXM-39][card] |

Non-goals [CARD AC6][card]: domain/CLI redesign, new flags registry, configuration
store or workflow model, Runtime preference/ranking, admission certification,
new architecture, Execution semantic changes, new host support, AXM-12 E2E
acceptance. Existing AXM-31/32/33/37/38 acceptance is not reopened.

## Project labels and exact identity

**P1 — CARD AC1 and accepted fallback clarification [AXM-39][decision].**
Use the following deterministic narrative policy over observed canonical fields.
“Present” means a supplied nonempty human identifier; never invent/repair a
missing field from Repository name, directory, prior chat or Project ID.
Display escaping may make control characters safe, but must not rewrite identity.

| Observed name | Observed slug | Context | Narrative label / action |
| --- | --- | --- | --- |
| Present | Present | Unique pair | `Nome (slug)`; preserve both even if texts coincide |
| Present | Absent | Unique label | Name only |
| Absent | Present | Unique label | Slug only |
| Absent | Absent | Any | `Project sem nome/slug — ID: <full canonical ID>`; explain identity fallback |
| Any | Any | Same display label for distinct IDs, or user reference matches multiple candidates | Show each available human label plus `ID: <full canonical ID>`; explain collision; obtain an exact target before a sensitive action |
| Any | Any | Inventory incomplete, unreadable or uniqueness not established | Label may aid reading; disclose unresolved target certainty; inspect canonical identity before action |
| Any | Any | User requests ID or operation requires it | Provide complete canonical ID and reason where operationally needed |

Collision is evaluated across the canonical candidate set for the current
request, not a claim of global uniqueness. Do not pick the first match, last
used Project or a longer UUID prefix. Repeated same input/candidate set produces
same labeling decision. List ordering remains source-owned. Duplicate records
for the same ID are not distinct Projects; contradictory metadata stays visible
as uncertainty and requires canonical read-back [CARD AC1][card].

**P2 — CARD AC1 [AXM-39][card]; ACCEPTED [ADR-0030][030] and [007][007].**
Human labels are navigation aids, never authority. Retain full selectors,
copyable commands, Project/Workflow/Execution references, Evidence identifiers,
preview digests and operation scopes byte-for-byte. An already unique observed
slug can remain a supported selector; this policy does not require converting
every command to UUID. Commands must use existing canonical syntax, never
concatenate untrusted labels as executable fragments. A missing canonical ID
means identity unavailable, not permission to generate one.

No UUID abbreviation is necessary for this minimal policy: collision fallback
uses the full ID with explicit operational justification. AC1 permits auxiliary
abbreviation but does not require it. A short narrative label must never replace
the exact target in a consent/preview view. Revalidate sensitive targets and
authority through existing operations; display text or a repeated label cannot
supply consent [ACCEPTED: RA-2][ra].

## Stage summary and progressive details

**S1 — CARD AC2 [AXM-39][card]; ACCEPTED [007 Stage/Agent semantics][007].**
For both `intake` and `implementation`, begin with actual Stage ID and purpose,
then meaningful inputs/outputs, human gates and their timing, Role/Agent ownership,
Runtime/Profile state and completion criteria. Explain domain terms briefly when
needed: Role describes responsibility; Agent identifies its configured owner;
Runtime executes; Profile supplies local model/effort intent. They are not synonyms.
Summarize from the inspected revision, never from the Stage's familiar name.
Missing source values stay “não informado nesta fonte”; explicit empty gates mean
“nenhum gate declarado”, not “gate aprovado”. Gate satisfaction needs Evidence.

| Initial summary | Expand on request | Promote immediately when material | Origin |
| --- | --- | --- | --- |
| Purpose, input/output meaning and source, human gates, Role/Agent, Runtime/Profile with status, completion criteria | Full instructions, I/O references and requiredness; validator identity/configuration, retries/attempts, limits, timeout, concurrency, effect ceilings, failure policy | Any constraint causing risk, blocked readiness, unknown effects, withheld authority, failure or required reconciliation | CARD AC2/4 [AXM-39][card] |
| Sequential vs same-Stage DAG context, dependency/gate warnings needed to understand work | Full agent predecessor list and declared concurrency | Concurrency/isolated writer limits or dependency failures preventing progress | ACCEPTED [011 VP-03–05/11–14][011] |

Essential does not mean a fixed field dump: short prose or compact lists may
group related facts. Advanced does not mean suppressed. Requested detail must
remain retrievable from the same canonical revision; if unavailable, identify
the missing source instead of fabricating configuration [CARD AC2][card].

**S2 — ACCEPTED [011][011], [ADR-0030][030].** Preserve rendered Workflow
overview on the qualified host, ordered Stage progression, same-Stage Agent
dependencies, exact Workflow reference/selection and definition-vs-Execution
labels. Text uses the same facts and offers Stage/Agent follow-up without
redundant full dumps. Unsupported, unknown or failed rendering retains complete
accessible textual fallback; do not claim image rendering from source fences.
This specification adds no renderer or host certification. A meaningful graph
constraint remains visible even when nominally an advanced Stage field.

## Runtime/Profile sources, readiness and failures

**R1 — CARD AC3 [AXM-39][card]; ACCEPTED [007][007], [RA-1–4][ra],
[ADR-0024][024].** These are presentation distinctions, not new wire enums or
fields. Associate every claim with its actual source/reference and scope.

| Distinction | Source sufficient for claim | Claim not authorized by that source alone |
| --- | --- | --- |
| Allowed | Workflow/Agent `runtimeConstraints`, Project policy, applicable local enabled configuration | Selected Runtime, compatibility certification or execution |
| Declared | Agent `profileRef`, Project policy/Profile declaration | Effective resolution; `policy-default` is resolver intent, not a concrete Profile |
| Selected | Exact canonical selection or explicit choice recorded for the operation | Successful resolution/admission, dispatch or success |
| Resolved | Matching canonical resolution/Stage Plan/Execution facts and revision | Runtime actually invoked, requested effort actually honored, task success |
| Observed | Correlated attempt/result/auth diagnostic Evidence | Universal capability, future success, billing identity or human acceptance |
| Unknown / absent / stale / conflicting | Missing or nonmatching source, incomplete observation | Default, preference or negative fact invented to fill the gap |

Project policy containing `codex-test` declares configuration; that text neither
selects a Runtime nor proves that Profile resolved for an Agent. Codex/Claude
permission is not choice. Host initiating the conversation has no implicit
preference. Model/effort comes from Profile/resolution, not Role/complexity;
`runtime-default` does not assert a concrete vendor effort. Historical schema
intent remains historical. Conflicting revisions are reported and re-read;
do not blend definition and old Execution facts into a current resolution.

**R2 — ACCEPTED [AXM-36 amendment RA-1–4][ra] and [ADR-0032][032]; CARD
approved clarification [decision][decision].** No registry, capability probe,
version/digest allowlist, model ping, qualification receipt or advance
certification as admission. Preserve installed executable/configuration/auth
observation, deterministic policy/binding checks, Profile effort ownership,
source/override/path safety, preview identity/digest/revision revalidation,
independent effect authority and honest confinement unknowns. Unknown version
alone is not failure. Actual unsupported invocation/model/effort is diagnosed
from observation; no hidden Runtime/Profile/API/permission fallback.

**R3 — CARD AC4 [AXM-39][card]; ACCEPTED [008][008], [RA-3/4][ra].**

| Canonical observation | Required conversational interpretation / next step |
| --- | --- |
| Command `status: success`, `execution: blocked`, no dispatch | Command completed successfully; execution **not ready and not performed**; quote exact blocker code/reason, affected scope, risk/authority and smallest supported next action |
| Structurally `valid`, readiness absent | Validity confirmed only; readiness unknown; inspect applicable canonical readiness source |
| Resolved selection, no attempt | Resolution recorded; execution not observed; inspect authorized next preview/gate, never claim execution |
| Failure before spawn, confirmed zero effects | State failed boundary and known absence of execution; correction within existing authority, then fresh preview where required |
| Partial/unknown effects, timeout, lost supervision, stale or conflicting Evidence | State certainty separately from command result; reconcile before redispatch/retry; disclose required authority |
| Exit 0 / child prose, result missing or invalid | No validated Stage success; show result/validator failure and safe inspection |
| Source/tool unavailable, output truncated or inaccessible | State gap; request smallest safe read; do not hide failure or synthesize a full state |

A blocked result can also describe a previously attempted Execution: claim
“não realizada” only when no-dispatch is supported; otherwise state prior attempt
and uncertain/observed effects explicitly. A short answer still retains all
material blockers, warnings and uncertainty; never collapse multiple blockers
to a single inferred cause. Confirmed cause differs from labelled hypothesis.
Next action must exist on the installed surface or be stated unavailable;
inspection consent/login does not authorize inference, Provider effects or
repair. No automatic credential/permission edits, retry or new authority.

## Safety and legibility

**L1 — CARD AC1/2/5 [AXM-39][card]; ACCEPTED [ADR-0030][030],
[RA-2/4][ra], [security policy][security].** Treat labels, instructions and
diagnostics as data; escape unsafe display content and ignore embedded commands
as authority. No credentials, account identifiers, private paths, raw stderr or
unsanitized transcripts in Evidence. Synthetic public IDs in examples retain
their full bytes; sanitizer must not silently shorten authority identifiers.
Use synthetic fixtures or clearly labelled redacted copies kept distinct from
canonical authority records when real material cannot be published safely.

Preserve accented/Unicode names without identity normalization. Plain text must
carry meaning without color, icons, hover, clicks or diagram availability; avoid
wide payload tables in native conversation when compact prose is clearer.
Hosts may wrap/group prose differently; identity, blockers, source/status and
gates retain equivalent meaning. No fixed pixel/word budget or new platform
support promise is introduced.

## Acceptance and Evidence matrix

All rows specify future product verification. Documentary completeness is not
product PASS. [Examples and test protocol](examples.md) define representative
inputs and positive/negative oracles without a new schema or durable runner.

| Original AC / origin | Requirement coverage | Verification obligation | Current product result |
| --- | --- | --- | --- |
| AC1 [CARD][card] | P1–P2 | Name/slug matrix, collisions/absence, Unicode, exact command/authority/Evidence bytes, list/show/validate + natural entry | NOT_RUN |
| AC2 [CARD][card] | S1–S2 | Both Stages, essential/detail split, mandatory risk promotion, diagram/fallback fidelity | NOT_RUN |
| AC3 [CARD][card] | R1–R2 | Allowed vs declared vs selected vs resolved vs observed; `policy-default`, `codex-test`, stale/missing source; no certification | NOT_RUN |
| AC4 [CARD][card] | R3 | Success/blocked, valid/unknown, multiple blockers, prior attempts, unknown effects, invalid result and unavailable source | NOT_RUN |
| AC5 [CARD][card] | L1 and examples protocol | Deterministic content regressions plus one real native conversation in Codex and one in Claude; sanitized correlated Evidence | NOT_RUN / separate authorization |
| AC6 [CARD][card] | Scope, P2, S2, R2 | Diff/ownership review: no new flags/config registry, ADR, wire/Execution semantic change | Documentary review only; implementation NOT_RUN |

Specify may finish before a dependency release. Implementation and final
dependent qualification must not start while an actual dependency blocker
persists. Operational Done/release, compatible installed binary/skills and native
observations are distinct: recheck each before dependent work. AXM-12 owns real
inference/E2E and final human acceptance; this document transfers none of its
authority [CARD approved gate][gate]; [ACCEPTED ownership][ownership].

[card]: https://linear.app/rgomids/issue/AXM-39
[decision]: https://linear.app/rgomids/issue/AXM-39#comment-70d285fc
[intake]: https://linear.app/rgomids/issue/AXM-39#comment-a9e0c096
[gate]: https://linear.app/rgomids/issue/AXM-39#comment-054cdea2
[007]: ../007-configurable-workflows/spec.md
[008]: ../008-output-separation/spec.md
[011]: ../011-rendered-workflow-presentation/spec.md
[ra]: ../007-configurable-workflows/amendment-axm36-runtime-admission.md
[024]: ../../decisions/0024-model-profile-effort-ownership.md
[030]: ../../decisions/0030-canonical-results-terminal-and-host-presentation.md
[032]: ../../decisions/0032-runtime-admission-without-certification.md
[security]: ../../../.agents/policies/security.md
[ownership]: ../../development/linear-github-issue-tracking.md
