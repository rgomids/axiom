# Specification 012 — Implementation Plan r1 (AXM-39)

## Scope, authority and baseline

2026-10-23; PLAN / attempt-01. Documentary implementation strategy for
[Specification 012 r1](spec.md), preserving its six AC and all non-goals.
[Human baseline](https://linear.app/rgomids/issue/AXM-39#comment-70d285fc),
[READY_FOR_plan](https://linear.app/rgomids/issue/AXM-39#comment-70d2c0f7),
[Specify receipt](https://linear.app/rgomids/issue/AXM-39#comment-ce984f36).
Source inspected: `fc4b00b06d0f69d74b50b9b6222f9a71ca38abb2`;
[Draft PR #343](https://github.com/rgomids/axiom/pull/343), branch
`docs/axm-39-conversational-presentation`, base `main`. Reuse that PR only.
This attempt creates no Tasks, implementation, installed changes or native runs.
The historical Specify stopping boundary remains unchanged in its artifacts.

Accepted boundaries remain [007](../007-configurable-workflows/spec.md),
[008](../008-output-separation/spec.md), [011](../011-rendered-workflow-presentation/spec.md),
[AXM-36 RA-1–6](../007-configurable-workflows/amendment-axm36-runtime-admission.md),
[ADR-0030](../../decisions/0030-canonical-results-terminal-and-host-presentation.md)
and [ADR-0032](../../decisions/0032-runtime-admission-without-certification.md).
No domain/CLI JSON or text/flags, Workflow/Execution, Registry, consent/auth,
architecture, renderer, service or certification change. No ADR candidate is
needed within this strategy. No private Axiom Workflow plugin enters the product.

## 1. Observed owners and future touchpoints

Paths below are source-relative from this directory. Implementation edits are
future and bounded; an inspected owner is not permission to change its contract.

| Owner / inspected source | Future responsibility / change boundary |
| --- | --- |
| [Packaged axiom-project](../../../internal/codexruntime/skills/axiom-project/SKILL.md), operation routing, guided readiness, canonical presentation | P1–P2 labels and candidate ambiguity for list/show/validate and natural follow-up; R3 blocker and next-action explanation; preserve exact selectors and existing operations |
| [Packaged axiom-workflow](../../../internal/codexruntime/skills/axiom-workflow/SKILL.md), configuration readiness, Result contract, Workflow visualization | S1 summary/detail and risk promotion; R1 source distinctions; preserve S2 overview, Stage order, same-Stage DAG, full fallback and reference checks |
| [Packaged axiom-runtime](../../../internal/codexruntime/skills/axiom-runtime/SKILL.md), canonical presentation and guided setup | R1–R3 explanation of local Profile/executable/auth observations; no Runtime choice from current host, login or unknown version |
| [Packaged axiom-work-item](../../../internal/codexruntime/skills/axiom-work-item/SKILL.md), existing result/Stage planning guidance | Consistent Stage Plan/Execution and effect certainty handoff. Edit only if cross-entry regressions show missing 012 presentation rules; no new execution behavior |
| [CLI DTOs](../../../internal/cli/cli.go), [Project list](../../../internal/projectapp/list.go), [readiness](../../../internal/projectapp/readiness.go), [Workflow report](../../../internal/projectapp/workflow_authoring.go) | Read-only source owners: ProjectListView has name/slug/ID; ProjectView has slug/ID, no name; WorkflowReport has reference/selection/definition and diagnostics. Consume existing facts; do not add name or presentation fields to wire contracts |
| [Content contracts](../../../internal/codexruntime/workflow_skill_content_test.go), [runtime contracts](../../../internal/codexruntime/runtime_test.go) | Extend instruction/package checks for changed skills; tests demonstrate embedded guidance, not AI compliance |
| [Semantic candidates](../../../internal/cli/output_separation_test.go), [presentation specimens](../../../internal/cli/testdata/presentation) | Extend test-only typed claim evaluation and positive/negative conversation specimens for 012; no production formatter |
| [AXM-38 oracle](../../../internal/codexruntime/workflow_presentation_test.go), [rendered fixtures](../../../internal/codexruntime/testdata/rendered-workflow), [producer check](../../../internal/cli/workflow_presentation_fixture_test.go) | Reuse full reference/order/edge/gate and fallback oracles; preserve old fixtures, add separate synthetic cases when needed |
| [Shared manifest/history](../../../internal/codexruntime/manifest.go), [integration](../../../internal/codexruntime/integration.go), [upgrade tests](../../../internal/codexruntime/rendered_workflow_upgrade_test.go), [shared-history tests](../../../internal/codexruntime/shared_history_test.go) | Register complete replaced four-skill revision before changing embedded bytes; retain historical digests/receipts; test Codex and Claude convergence, interruption and conflicts |
| [Bootstrap](../../../internal/runtimebootstrap/bootstrap.go), [pair restoration precedent](../../../internal/codexruntime/output_separation_test.go) | Regression consumers only; no bootstrap/install redesign. Temporary test roots do not prove actual installed hosts |

Maintainer `.agents/skills` are repository policy, not the product guidance above.
Runtime-specific copies or adapters are unnecessary: both integrations consume
the shared embedded skill set. No private plugin installation or duplicate rules.
Documentation reconciliation stays in this directory and the 012 index entry.
Future product guidance documentation changes require a demonstrated affected
reference, never a broad README, commands or historical-status rewrite.

## 2. Presentation interfaces and causal strategy

These are host reasoning steps over existing canonical observations, not a new
serialized schema, stored object, resolver or executable API. Keep source result
and provenance accessible; canonical references outlive any narrative summary.

1. **Observe and correlate.** Use installed skill/operation discovery and JSON
   results as already required by ADR-0030. Capture exact Project ID, source,
   relevant revision, Workflow full reference and any actual Stage Plan/attempt.
   Refuse to merge stale or conflicting observations. Missing/truncated output
   stays unknown; read only the smallest supported missing source.
2. **Label Project (P1–P2).** Apply the Spec matrix to the current canonical
   candidate set: name+slug, one field, neither field, collision or incomplete
   inventory. Preserve Unicode and source ordering. Same ID repeated is not
   two Projects; contradictory metadata triggers read-back. Full ID is required
   for absence/collision/explicit request/operational necessity, with reason.
   No prefix algorithm, first match, Repository-derived name or generated ID.
3. **Correlate missing display metadata safely.** `show` does not carry name.
   An existing list observation may contribute name only through exact full ID
   and compatible current source; refresh via supported read if necessary.
   A validate result may identify its target without display metadata: correlate
   the exact target similarly. If unavailable/stale, use observed slug/ID fallback
   and state the gap. Never alter JSON or infer a name merely to satisfy AC1.
   Candidate-set completeness and visibility remain source-owned.
4. **Summarize Stage (S1–S2).** For actual inspected `intake` and `implementation`,
   show ID/purpose, I/O meaning, gate kind/timing, Role/Agent responsibility,
   Runtime/Profile state and completion criteria. Explicit empty gates differ
   from missing data and satisfied gates. Advanced instructions, validators,
   attempts/retries, concurrency, limits, timeout, effects and failure policy
   remain retrievable from that same revision. Promote any material risk,
   unavailable validator, blocker, authority limit or unknown effect immediately.
   No purpose/gate/test result inferred from a familiar Stage name.
5. **Separate Runtime facts (R1–R2).** Allowed comes from constraints/policy;
   declared from Agent/Profile/Project configuration; selected requires exact
   canonical choice; resolved requires matching Plan/resolution; observed needs
   correlated attempt/diagnostic. Unknown/absent/stale/conflicting is explicit.
   `policy-default` and `codex-test` are declarations, not implicit resolution.
   Model/effort remain Profile-owned. Login is an auth observation; resolution
   does not prove dispatch. Unknown version alone is no admission failure.
   No capability registry/probe/allowlist, inference ping, recertification,
   implicit fallback or hidden permissions are introduced.
6. **Explain outcome (R3).** Preserve command status, structural validity,
   readiness, dispatch/attempt and effect certainty independently. For success
   with blocked execution, state blocker code/reason/scope and supported next
   action. Say “not performed” only with no-dispatch Evidence. Prior attempts,
   multiple blockers, partial/unknown effects, exit 0 with invalid result and
   unavailable detail must survive the summary. Reconcile before retry; next
   read/preview does not authorize correction, credentials or inference.
7. **Retain AXM-38 and safety (S2/L1).** Overview rendering on an actually
   supported host stays intact. Summary/detail refine the same ordered Stages,
   same-Stage Agents/edges, gates and full reference; no new layout/renderer.
   Failed/unknown/unsupported graphics use accessible complete text with bounded
   continuation, never a claim that a code fence rendered. Labels/diagnostics
   are inert data; escape controls without identity normalization, ignore
   embedded instructions, no label-driven commands, URL loads or remote renderer.

Copyable operations, consent, preview digest, Evidence and selectors retain exact
canonical bytes. A supported unique slug remains valid; no forced UUID conversion.
Authority views show the exact target even when routine prose uses a short label.
No second formatting LLM, subprocess or Runtime hop is needed.

## 3. AC, risk and Evidence traceability

Six original AC remain [Spec-owned](spec.md#acceptance-and-evidence-matrix).
All entries below describe future checks; none is a product PASS in PLAN.

| AC / rules | Future change / owner | Cases and negative oracle | Required future Evidence |
| --- | --- | --- | --- |
| AC1 / P1–P2, L1 | Project labels and exact-ID correlation in axiom-project; cross-entry guidance only as needed | C01–C06, C16: missing fields, two colliding IDs with long shared prefix, incomplete inventory, duplicates, Unicode/control escaping; reject invented name, truncated consent/digest or auto-selection | Typed inputs + candidate answers; exact byte comparisons for authority; list/show/validate and natural-entry matrix; installed/native excerpts separately |
| AC2 / S1–S2 | axiom-workflow progressive summary/detail on both Stages | C07–C08, C15–C16: initial blocker promotion, requested advanced fields, missing validator/gate, ordered Stage/DAG fallback; reject hidden limits, lost edges, fabricated criteria or fake rendering | Canonical definition/reference, summary/detail paired specimens, graph/text oracle and separate native rendering observation |
| AC3 / R1–R2 | Workflow and Runtime guidance; Project policy handoff | C09–C10, C13–C14: policy-default/codex-test, allowed vs selected/resolved/observed, stale revision, login-only, unknown version; reject host preference, made-up effort/model or certification remedy | Source-attributed claims; mutations that falsely promote a state fail; actual attempt only if separately authorized |
| AC4 / R3 | Project and shared result guidance; Work Item handoff if needed | C11–C13: success+blocked, valid+unknown, no dispatch vs prior attempt, multiple blockers, timeout/invalid result/partial effects; reject “executed”, unsafe retry or lost blocker | Full result/diagnostic + supported next action, correlated certainty; positive and omission/invention negatives |
| AC5 / L1 | Existing content harness extensions + two separate native conversations | C01–C16; reject omitted claim, changed exact identifier, invented state, injected instruction; sanitized publication | Deterministic command/results/case IDs; Codex record and Claude record under separate authorized host protocols; gaps retained |
| AC6 / scope/P2/S2/R2 | Diff/ownership and packaging review | No flag registry, CLI/domain/Execution semantic edits, new ADR or certification; historical fixtures immutable; synthetic host never labelled native | Scoped diff, unchanged contract review, shared-history/upgrade results, no unauthorized effects |

## 4. Test design using discovered harnesses

Reuse Go `testing` and current fixture conventions. No new framework, durable
runner, production presentation mapper or flags registry. Organize future 012
cases adjacent to existing presentation tests/specimens, labelled synthetic.
Each C01–C16 row in [examples.md](examples.md) gets explicit coverage, positive
assertions plus omission/invention negatives. Candidate wording may vary;
required facts, forbidden claims and authority-bearing bytes are the oracles.

The existing `semanticObservation` / `semanticClaims` / `checkCandidate` in
`internal/cli/output_separation_test.go` covers completion, references and several
diagnostic/effect claims. It does **not** yet model Project name/slug/candidate
sets or all Stage/Profile distinctions. Extend test-only typed extraction and
case-specific assertions; never weaken its existing status/effect checks or
force the old all-reference prose pattern onto ordinary name-only narratives.
Keep full identifiers in authority-bearing views; ordinary labels need not
repeat every UUID. Construct producer-shaped canonical input with existing DTOs;
preserve field absence vs explicit empty. For missing invalid observations use
explicit negative fixtures rather than pretending they are valid producer output.

Group coverage: C01–C06 identity/candidate cases; C07–C08 two-Stage summaries and
risk promotion; C09–C10 runtime source/certainty cases; C11–C14 readiness/failure/
unknown/stale cases; C15 graph/text fallback; C16 cross-entry/plain-text fidelity.
Change input facts and require the corresponding claims to change. Mutate good
specimens by dropping a blocker, replacing full ID/digest, inventing selected
Runtime or execution, hiding uncertainty, adding unsafe retry or injected label
instruction; checker must reject each. A checker that accepts these negatives
fails even when every positive specimen passes.

Reuse `TestWorkflowShowSyntheticPresentationEnvelopeMatchesProducer` for Workflow
envelope equality; add analogous targeted producer assertions for new fixtures
where needed. Reuse AXM-38 manual oracles and fallback checks instead of deriving
both expected and actual facts through one helper. Instruction-string assertions
only establish packaged guidance. Candidate/oracle tests only establish synthetic
semantic evaluation, not model fluency, compliance, graphical rendering or native AC5.
`scripts/test-native-acceptance.py` is artifact trust-boundary coverage, not this
conversational harness; do not relabel it as native conversation Evidence.

Future validation order after authorized implementation:

1. Focused content and semantic tests:
   `go test ./internal/cli -run 'Test(OutputSeparation|WorkflowShowSynthetic|Presentation)'`
   and `go test ./internal/codexruntime -run 'Test(RenderedWorkflow|OperatorConversational|WorkflowSkill)'`;
   include the actual new 012 test names selected during implementation.
2. Shared packaging/history/rollback: `go test ./internal/codexruntime ./internal/runtimebootstrap ./internal/cli ./cmd/lingo`.
   Existing `scripts/test-codex-skills.sh` invokes those packages and optionally
   a locally available skill validator; use real underlying commands and report
   skipped optional validation. Broaden `go test ./...` only for relevant impact/failures.
3. `./scripts/validate-repository.sh .`, `git diff --check`, staged sensitive-file
   check and scoped redacted secret scan; inspect full changed diff and ownership.
4. Two authorized native conversations below. Record deterministic and native
   results separately; unavailable sessions remain NOT_RUN, not synthetic PASS.

## 5. Dependency gates and future executable decomposition

Current authenticated read-back: AXM-36/37/38 **Done**, AXM-12 **In Progress**.
Source HEAD contains operational-admission guidance, installed-doc inspection and
AXM-38 rendering/text oracles. These facts support planning, not actual host
installation, loaded skills, inference or rendering. Re-read source/PR/dependency
state at Tasks/implementation entry, serialize shared-skill edits and reconcile
any intervening changes before preparing a new four-skill history record.

The following work packages are decomposition guidance for a later Tasks stage;
they are not Task records, IDs, assignments or authorization to implement now.

| Future package / accountable responsibility | Dependency and observable exit | Negative / Evidence |
| --- | --- | --- |
| Source/gate check — executor | Exact canonical PR/head, Spec revision, installed-dependency needs and shared-file conflicts reconciled before edits | Drift/active overlapping skill writer blocks dependent edit; record read-back and scoped paths |
| Regression preparation — test owner | After source check; producer-shaped C01–C16 fixtures and candidate assertions, preserving old oracles | Each omission/invention mutant rejected; guidance checks fail against unmodified guidance where applicable; no LLM involved |
| Project presentation — guidance owner | After regression preparation; P1–P2 in product Project guidance, correlated show/validate label and exact authority | C01–C06/C16; no name source means honest fallback, not wire change |
| Stage/Runtime presentation — guidance owner | After regression preparation; S1–S2/R1–R3 aligned across affected skills, sequential shared-file writes | C07–C16; missing detail, stale resolution and prior unknown effects retain blockers |
| Package compatibility — runtime integration owner | After final guidance bytes; append replaced revision to sharedSkillHistory, retain history and current compatibility unless actual incompatibility demonstrated | Both temporary Runtime roots converge; modified/foreign skills refused; interruption not ready; exact restoration preserves state |
| Synthetic verification/review — executor/reviewer | After both guidance packages and packaging; focused/package/document/security checks; unchanged public contracts | Full AC/case/result ledger and independent negative review; product/native claims remain NOT_RUN |
| Native conversations — authorized host operator + reviewer | After compatible installed binary/skills read-back and separate session/effect authority; one Codex and one Claude conversation | Per-host observations/cases/limits, no dispatch or inference inferred; failure remains failure; AXM-12 acceptance retained |
| Reconciliation/handoff — executor | Gather synthetic/native statuses, exact PR/SHA, deferred findings and unresolved gates | No unsupported AC5 completion, merge/release/Done or native approval; readiness follows actual Evidence |

Project and Stage work can share prepared fixtures but cannot race on common
skills/manifest. Execute directly while capability hierarchy for smaller agents
is unverified; no speculative delegation or parallel edits.

### Installed read-back and separate authorization

Offline fixtures, test roots, diff/document checks and test design can be prepared
without vendor execution. No user-global root is inspected or changed in PLAN.
Future host operator first establishes authorized target/surface/build and reads
actual installed Axiom binary provenance, packaged skill identity and runtime
status using supported `axiom --json runtime codex status` and
`axiom --json runtime claude status`; inspect only the relevant authorized host.
Use `axiom --json docs inspect` and its returned verified paths, and
`axiom --json skill inspect <domain-skill>` for installed operation discovery.
These are future supported read-back commands, not executed observations here.
Do not run bootstrap `Run` to inspect: it performs installation.

Record absent/modified/foreign/partial or incompatible pair and stop its dependent
session. Installed/verified/loaded/authenticated/authorized/observed remain distinct;
on-disk skill presence does not prove loaded. Refresh an authorized session when
required by the host; do not edit receipts, auto-install or certify a vendor.
Any installed update, actual Codex/Claude invocation, inference, credential/auth
inspection requiring consent, expense or external effect without specific grant
is **AUTHORIZATION_REQUIRED** for that action. No baseline or predecessor Done
supplies that grant. Preserve safe independent work; no ceremonial transition gate.

Use the [existing two-conversation protocol](examples.md#two-native-conversations--future-protocol-not_run)
outside the maintainer checkout. Separate Codex and Claude records include date,
host/build/surface, binary/skill revision/content, authorization ref, canonical
input/revision, AC/case, sanitized visible output, PASS/FAIL/NOT_RUN, gaps/next action.
Exercise natural list/show/validate, requested ID, both Stages and Agent details,
allowed/declared/resolved/observed and blocked/unknown; preserve graphic/text limits.
If collision or blocked input is replayed, label it synthetic within that session;
it cannot prove a native collision/blocker occurred. Never dispatch solely to
manufacture a positive example. AXM-12 retains real inference/E2E and human acceptance.

## 6. Risks, rollback and Evidence gates

| Risk | Control / rollback |
| --- | --- |
| Shared paths drift under AXM-36/37/38 or other work | Read current head/dependencies before Tasks and implementation; serialize edits; compare replaced full skill set; never rewrite historical receipts/fixtures |
| Name unavailable in show/validate or catalog incomplete | Exact-ID correlation from supported observed list, otherwise truthful fallback/uncertainty; no JSON/schema expansion |
| Concision hides gate, blocker or effects; Runtime states blended | Positive and adversarial content checks; immediate risk promotion; source/revision-tagged claims; native review |
| Rendering or host behavior differs | Preserve AXM-38 path/text; per-host observed rendering, no new renderer; text mitigation does not count graphical success |
| Mixed installed skill pair or user changes | Existing history/conflict/interruption tests; preserve user content and refuse adoption; supported separately authorized upgrade only |
| Malicious label/diagnostic, leaked private Evidence | Inert display/control escaping, no execution or remote fetch; synthetic public fixtures; classified sanitized excerpts; exact authority separate from redacted copies |
| Rollback erases state or revives obsolete guidance | Revert only 012 guidance/test/manifest change through reviewed normal history, preserving appended published ownership history if already released; restore compatible binary/skills through supported authorized mechanism; preserve Project/Workflow/Execution/Profile state and foreign/modified files |

No state migration, new persistence, logger or external service. Evidence uses
existing test output, fixtures, PR and versioned Evidence; no raw stderr, private
paths/accounts or chat exports. Retain original canonical authority records;
published redactions are labelled copies, never operational selectors.

Tasks readiness checks Plan/Spec alignment and decomposability. Implementation
checks actual source/dependency blockers and scoped Tasks. Native verification
checks installed pair and specific authority. Pre-merge shows DF-01 prominently;
merge, release, closure, Done and human acceptance remain separate. No callback
continuation API has been demonstrated; completion event hands off at
`AWAITING_EXECUTION_REPORT` for `readiness_for_tasks`, without creating Tasks here.

## 7. Local technical decisions and actual result

All choices: TECHNICAL_LOCAL, `scope_impact: none`, `decision_maker: executor-ai`.
Recoverable records and actual checks are in [Plan Evidence](evidence-plan.md).

| ID / question | Chosen option / alternatives considered | Rationale / source |
| --- | --- | --- |
| PLAN-TD-01: where to implement presentation? | Existing shared product skills; alternatives Go formatter, maintainer rules or private plugin | ADR-0030 host ownership; shared integration source; no new API/architecture |
| PLAN-TD-02: which content harness? | Extend existing test-only typed semantic candidates + AXM-38 oracles; alternatives new runner, wording snapshots, string-only skill tests | Existing checkCandidate lacks some 012 claims; extend narrowly; deterministic tests cannot prove model/native behavior |
| PLAN-TD-03: how to show name without show.name? | Exact-ID-correlated observed list metadata or Spec fallback; alternatives new wire field, inferred Repository/chat name | Actual ProjectView lacks name; approved P1 absence/stale policy already resolves gap |
| PLAN-TD-04: packaging/version and execution organization? | Shared complete replaced revision, current compatibility 3 unless incompatibility proven; direct sequential execution; alternatives per-host copies, unconditional version bump or unverified delegation | manifest.go/integration.go history contract, same four skills for both hosts; smaller capability hierarchy unverified |

PLAN result: documentary strategy only. C01–C16 execution, new guidance/package
behavior, native Codex/Claude, installed readiness, inference/E2E and acceptance
remain NOT_RUN. Deferred DF-01 remains DEFERRED_NOT_IMPLEMENTED as recorded in
[Specify Evidence](evidence-specify.md); do not rewrite predecessor history.
