# Specification 007 Plan — Runtime admission without certification (AXM-36)

## Status, authority and checkpoint

**Proposed — SDD plan, revision 1, 2026-10-19.** Documentation only. This Plan
is not accepted and creates no executable Tasks or Implementation authority.

The maintainer [accepted ADR-0032 and the amendment](https://github.com/rgomids/axiom/pull/334#issuecomment-6278024276)
on 2026-10-19 at 21:41:36 UTC, exact commit
`b4baf106f02b89028149ff57b01bf960524692f8`, including the partial-supersession
map. [READY_FOR_plan](https://linear.app/rgomids/issue/AXM-36/simplificar-runtime-admission-e-tratamento-de-falhas-operacionais#comment-709c2872)
and this session authorize documentation, commit/push and updating the same
[Draft PR #334](https://github.com/rgomids/axiom/pull/334), branch
`docs/axm-36-runtime-admission-specify`. The accepted behavior is
[RA-1–RA-6 / RA-AC1–10](amendment-axm36-runtime-admission.md), not a new
architectural proposal. Status/backlink annotations materialize that prior
acceptance; they do not accept the design choices in this Plan.

Resume inspection: local and remote head both matched the accepted SHA; clean
worktree; remote `main` was `de19d3279b1c9cbd730ca8bb2ab2f098e1b76d12`.
GitHub listed #334 as the only open AXM-36 PR (#317 is unrelated). Linear had
no attached Plan/Tasks documents or blockedBy relation, and its comments
contained specify review, exact acceptance, readiness and the plan prompt;
no competing Plan/checkpoint was found. Repository search found only the
AXM-36 amendment; no AXM-36 Tasks/Execution artifact. No `axiom` executable
was available on PATH. External canonical Execution state is **unavailable**,
not proven inactive. Do not reconstruct it from Linear's `In Review`, start a
new Execution or advance any canonical gate. If another session/Execution or
remote commit appears, reconcile before writing/pushing; never overwrite it.

Inspection stayed local: strict lower model capability could not be verified.
Provider records were used only to verify this explicitly public-documentation
decision and authority; no operational payload or host configuration is imported.

## 1. Dependency boundary and target flow

Paths in this inventory are relative to the repository root, inspected at the
checkpoint above. No production behavior changes in this PR.

| Consumer / current coupling | Planned bounded change and retained owner |
| --- | --- |
| `internal/runtimeadapter/version_registry.go`, `runtime-versions.json`, `binding_source.go` | Delete registry/selector/probe preparation. Keep fresh path+bytes identity, pinned Axiom identity, invocation mode/effect identity and configuration inspection in runtimeadapter; return current facts directly, never facts read from proof |
| `internal/runtimeprofile/{runtimeprofile,v2,v3,v3_consumer}.go` | New active configuration and frozen tuple without declarations, ProfileProof or CapabilityProven prerequisites. Keep enabled/allowed Profile selection, explicit Runtime, model/effort ownership, ambiguity refusal, revisions and digests |
| `internal/runtimeadapter/{adapters,authpreflight,stage_invocation,frozen_stage,effort_sources,effort_inspector,effort_applicability,stage_effort}.go` | Remove exact-version invocation branches, declaration applicability and proof-derived source projection. Preserve auth status, environment filtering, canonical encoding, trusted roots, no-override checks and dispatch-time child equivalence |
| `internal/runtimeapplication/policy.go`; `cmd/lingo/{runtime_policy,execution_target,workflow_stage_plan,stage_dispatch,capability_composition}.go` | Replace proof-enriching observation with direct operational facts through the shared policy resolver. Workflow preview/start, stage plan/run and per-attempt checks use the same configuration/identity semantics; never swallow generic ErrBlocked |
| `internal/workflowcompiler/{compiler,stage_plan}.go`; `internal/workflow/configured.go` | Stop copying CapabilityEvidenceRef / CapabilityProofReference. Compile policy-required effects and frozen intent/identity; preserve immutable Workflow snapshot and authority ceiling |
| `internal/stageexecution/{preview,run}.go`; `internal/executiongraph` | Remove Capabilities.Show, required-Proven loops and proof-conditioned confinement admission. Keep exact preview comparison, pre-spawn authority, leases, limits, effect scopes, inbox lineage and persisted terminal classification for graph and single paths |
| `cmd/lingo/{stage_service,stage_supervisor,stage_single}.go` | Compose current auth/identity/effort checks before each spawn, including retry. Preserve detached supervisor, cancellation, locks, final inbox drain, atomic publication, unknown/reconciliation restrictions and terminal-save ownership |
| `internal/capabilityproof`; `internal/local/{capability_store,capability_selection}.go`; `cmd/lingo/*capability*`, `internal/cli/runtime_capability.go` | Retire active certification producers, selection, probes/envelopes and command/parser/service wiring. Retain only isolated historical codecs/read paths if referenced by supported historical inspection; no active proof store dependency |
| `internal/local/{runtime_profile_store,runtime_profile_authoring,graph_store,successor_graph_store,single_stage_store}.go` and workflow persistence | Explicit active-format replacement, strict historical read dispatch and no silent mutation/migration; existing publication/lock/recovery protocol remains |
| `internal/codexruntime/skills/{axiom-runtime,axiom-work-item}/SKILL.md`, `docs/commands.md`, CLI help/completion/tests | Replace show/prepare/probe/probe-extended journey with inspect → authorize → execute → observe → diagnose/recover; same facts for human and AI, no skill-owned state or capability assertions |

Target chain: explicit Runtime + allowed Profile intent → policy/binding and
safe current observations → immutable workflow/stage plan → exact preview →
separate effect authorities → fresh identity/auth/effort checks → durable
dispatch authority → supervised attempt → persisted result/Evidence → diagnosis
and, when necessary, reconciliation before separately authorized retry.
Required capabilities in Agent/Workflow/Plan remain declarations of requested
work and effect policy, not vendor certification claims. Preserve Project !=
Repository, Role != Model, Execution != Agent and Runtime != Model boundaries.

## 2. Design choices proposed for RA-6

### D1 — Remove certification ownership, preserve operational Evidence

Delete the active `internal/capabilityproof` service/observer/selector/probe,
canary/help classifier and envelope machinery once consumers are disconnected.
Do not rename it to admission/preflight or report unknown facts as Proven.
Move only non-certification identity/confinement data still required by preview
into existing adapter/stage structures. No replacement service, registry,
qualification toggle, version matrix or successful-canary receipt.

Remove all `runtime <codex|claude> capability` parser routes
(`show`, `invalidate`, `prepare`, `probe`, `probe-extended`),
help/completion entries, flags and service methods (including extended probe
wiring). The old syntax returns the ordinary invalid-command result with safe
updated guidance; it cannot launch any subprocess or mutate a store. Keeping
an inert qualification facade would prolong two journeys and is rejected.
Also retire `--effort-declaration` in `internal/cli/runtime_authoring.go`, its
file ingestion in `cmd/lingo/runtime_profile_v3.go` / declaration handlers and
tests that demand declaration installation. Binding/Profile create/edit/remove
remain configuration operations with their existing authority and revision
checks; no new configuration command or automatic retirement. Reconcile local
migrate targets that can only emit the retired declaration-bearing format;
invalid historical authoring options must fail with zero writes.

Stop creating/reading/selecting/invalidation writes to `capability-evidence/v1`
in active resolution. Existing files and quarantined probe envelopes remain
untouched. Historical Execution/graph readers preserve old proof references
as historical opaque references and disclose retired qualification semantics;
absence of the old store cannot prevent status/history inspection. If a
supported historical reader needs proof decoding, isolate the bounded read-only
codec from active runtime types; do not keep the service to support a codec.
Old Evidence bytes remain available to explicit archival inspection with the
prior release; no new archive browser/GC/import command is required. No startup
cleanup, automatic deletion or reinterpretation as current authorization.

Operational Evidence remains with the existing attempt/coordination and result
owners: correlation, identity digests, safe diagnostic categories, validated
result reference, exit/signal, certainty and reconciliation facts. It neither
proves future capability nor grants authority. Retain result validators and
completion/presentation separation; exit 0 or a child assertion is insufficient.

### D2 — One bounded format transition, no speculative migration

Removing required fields changes closed schemas and digest meaning. Do not
reuse their current format with blank/fake proof or synthetic version values.
Propose one successor generation per affected persisted owner:

| Owner / current contract | Proposed successor and reason |
| --- | --- |
| Local configuration 3 with `effortDeclarations` | Configuration 4, same Runtime/Profile/preferences and monotonic revision; remove declaration collections entirely. New authoring emits only 4 |
| `ResolvedProfile` in configured Execution 3 | Configured Execution 4 with proof-free tuple; retain Profile/model/effort, configuration revision/digest, executable digest, invocation mode, observation revision/digest and effort-source digest. Remove `capabilityProofReference`; optional diagnostic version is outside authority/eligibility digest |
| Successor Graph 3 and single Stage run 1 | Graph 4 and single run 2 where the frozen resolution is embedded. Same scheduler/classification semantics and single/graph authority limits; no graph redesign |
| Compiler result 2 and StagePlan envelope 2; stored dispatch preview payload in DispatchInputs 1 | Compiler result 3, StagePlan 3 and DispatchInputs 2 distinguish proof-free resolution/rights and their digest interpretation. Recompute preview from new inputs; never apply an old digest under the new reader |
| Workflow definition schema 2, coordination records, dispatch-authority receipt bodies and Project policy | Keep versions when bodies/meaning do not change. Receipts continue binding an opaque exact preview digest; new preview content does not itself require a new receipt body |

Numbers above are next unused versions at the inspected head, not reservations
against concurrent work. Tasks must recheck their owners and record the exact
closed-schema field lists/fixtures before implementation; if another change
claims one, adjust the proposed number explicitly, without double transitions.

Use the [pre-stable compatibility rule](amendment-276-axm22-binding-identity.md#pm2-pre-stable-compatibility-and-canonical-digestv3-bld-001b3c-2a):
old active local configuration requires explicit retirement and reconfiguration
using existing preview/revision/authority publication controls, not an implicit
3→4 conversion. Preserve tombstone digest/revision and interrupted-publication
recovery. New Profile intent may be re-entered; no probe declaration is needed.
Refuse future/corrupt/duplicate-key/unsafe configurations distinctly.

Distributed Workflow schema 2 and Profile-owned effort remain unchanged.
Historical Workflow/Execution/graph codecs and digest fixtures remain readable
through version dispatch. Frozen historical tuples are never rebound to mutable
local intent. Old Executions/graphs do not acquire new dispatch/retry authority
by format conversion: status, cancellation/stop observation, inbox quarantine
and append-only reconciliation remain available through existing recovery
paths; new proof-free work requires an explicitly authorized new Execution
and fresh preview. Live/orphan/unknown/partial old attempts must be reconciled
before any replacement work can be authorized. No universal continuation or
bulk migration engine. The old binary may be used for bounded historical
inspection; it must refuse new unsupported formats without writing them.

This sacrifices transparent continuation of pre-MVP work to avoid silently
changing immutable evidence. Rollback is code revert before adopting new state;
afterward preserve new state, stop/reconcile live work, and explicitly
reconfigure for the chosen binary. Never downgrade records or redispatch an
old attempt merely to roll back. This compatibility choice needs Plan review;
it does not broaden ADR-0032's accepted behavior.

### D3 — Optional version observation and real auth

`authpreflight.go` currently returns before auth status on `version_unavailable`
or `version_unrecognized`; `stage_invocation.go` hardcodes exact versions.
Remove both eligibility dependencies. The adapter keeps one reviewed invocation
per Runtime/operation; it never chooses options or fallback from version.

Use the existing bounded, separately consented status runner. Version may be
observed with its existing non-inference `--version` command under the same
limits and redaction boundary, with identity checked around subprocesses.
Missing/unparseable/nonzero version yields diagnostic `unknown` and continues
to actual auth status. Cancellation/deadline, executable disappearance/drift,
unsafe configuration or inability to observe auth still fail at their actual
boundary. Do not suppress these as optional-version errors. A hung optional
observation may consume the operation budget, never bypass cancellation.
Auth status must be independently observed; `subscription_observed` remains
login-only, with inference/billing unconfirmed. Retain credential stripping,
API/auth override rejection and no switch of authentication method.

Version is optional metadata, not part of invocation selection, tuple equality,
source applicability or authorization eligibility. Changes to path/bytes,
model/effort, source facts or configuration remain authority drift. A version
string alone cannot make equal executable/configuration identities stale.

### D4 — Adapter-owned source inspection without declarations

Replace `ApplicableEffortDeclaration`, `ObserveEffortApplicability`,
`precedenceEstablished` and `ProofReference` with small rules in the existing
two adapters/inspector. Rules are code reviewed and deterministically tested,
not operator-authored declarations selected by version/digest. Do not put a
source catalog or qualification state in local configuration 4.

Retain canonical explicit encoding already used: Codex's Profile-derived
`--config model_reasoning_effort=<token>` and Claude's Profile-derived
`CLAUDE_CODE_EFFORT_LEVEL=<token>` in the exact child environment. Runtime-default
adds neither selector and claims no effective vendor effort. Validate token
safety, not model/token availability through a support matrix; actual rejection
is operational failure. Do not drop a required sandbox/permission option to
make an unsupported invocation succeed.

Inspect recognized argv, inherited selectors and the bounded user/project/
managed sources already modeled by the inspector (Codex TOML config and managed
config; Claude user/project/local settings and managed settings/drop-ins).
Resolve roots independently, retain trusted ancestors, ownership/no-symlink,
size/count/parser bounds and no credential/auth-file reads. Current
`CODEX_HOME` / `CLAUDE_CONFIG_DIR` relocation refusal stays an explicit
unsupported configuration conflict until a separately reviewed safe path
mapping exists; no arbitrary extra root discovery.

For recognized effort selectors, reject competing sources even if they repeat
the same value; only the canonical Profile-derived selector is admitted.
Runtime-default also refuses recognized external selectors. This preserves the
one-owner contract without having to assert vendor precedence. Unrelated safe
settings are permitted. Malformed/unreadable sources or a concrete unhandled
selector/relocation in the invocation are configuration errors, not requests
for qualification. An unknown Runtime version or unrelated unknown setting
alone cannot block. Never claim universal detection of future vendor keys.

New canonical effort-source projection contains adapter-rule identity, selected
intent, current executable/mode identity and sorted sanitized source facts
(presence, safe selector key/category and location binding). It has no version,
proof or declared-precedence field, no raw values/paths. Hash with explicit
new projection semantics; retain old digest fixtures unchanged. Reobserve at
preview/apply and before each attempt. For read-only mode, inspect the actual
inbox CWD; for repository-write, compare the repository and reviewed child
workspace project-source views. Preserve byte/presence equivalence and safe
anchor checks without requiring version evidence that a CWD is eligible.

Tasks must pin exact production recognized keys/source paths from current
adapter parsing and primary vendor interface evidence. Claude fixture
`fixtureEffortKey` is synthetic and must never become a production setting.
This is a bounded implementation prerequisite, not an installation gate:
no per-user receipt, help/probe inspection or claimed vendor support. If a
real ambiguity cannot be resolved without relaxing a safety invariant, stop
that task for a material human decision; do not add certification.

### D5 — Error contract and CLI/skill UX

Keep canonical completion statuses and exit mapping in `internal/completion`
and `internal/cli/completion.go`. Extend existing stage/result structures,
not a parallel error framework. Pre-spawn validation errors have zero spawn;
an apply acknowledging dispatch is not terminal success. Status/history expose
the persisted attempt outcome independently of the CLI command's completion.

Use these proposed stable diagnostic codes, with existing `authority_required`,
`authority_denied`, `stale_preview`, policy/format/recovery codes retained:

| RA-4 boundary | Diagnostic code / next step |
| --- | --- |
| Lookup or spawn | `executable_unavailable` / `spawn_failed`; correct executable configuration, preview again |
| Actual rejected option/mode | `invocation_incompatible`; review adapter/configuration, no downgrade |
| Missing/expired/unobservable auth | `authentication_unavailable`; operator restores vendor auth, reobserve |
| Unsafe path/config or permission | `configuration_unsafe`, `configuration_invalid`, `permission_denied`; inspect safe role/errno, no chmod/repair |
| Recognized effort conflict | `effort_override_conflict`; remove conflicting intent with separate authority, preview again |
| Confirmed model/effort rejection | `model_unsupported` / `effort_unsupported`; operator selects allowed intent, no substitution |
| Timeout/cancel | `attempt_timeout` / `attempt_cancelled`; observe stop and effects before recovery |
| Other child failure | `subprocess_failed`; retain exit/signal, cause unknown unless established |
| Missing/invalid qualifying record | `result_missing` / `result_invalid`; inspect validator/inbox Evidence |
| Lost supervision/partial or unknown effects | `effects_uncertain`; reconcile, never auto redispatch |

Structured attempt diagnostics carry operation/attempt correlation, boundary
code, confirmed cause or labelled hypothesis, effect certainty (`none`, `known`,
`partial`, `unknown`), sanitized Evidence references, next action and required
authority. Missing cause is unknown; an exit code alone cannot confirm option,
auth or permission rejection. Use bounded recognized diagnostic patterns only;
otherwise classify generic subprocess failure. Allow only safe categories,
key names and references; discard raw stderr/transcript/config/env, account
identifiers and host paths before persistence or presentation. Test adversarial
payloads in both structured and human output. No new raw-log flag.

At terminal save, the existing supervisor classifies blocked/failed/unknown/
cancelled/succeeded once; failure class never changes effect certainty or
rewrites the outcome. Store diagnostic summary with the attempt's existing
atomic publication, not in a second store. The graph/single successor codecs
include the closed optional diagnostic summary; historical attempts retain
their original bytes and no cause is backfilled. Process failure after spawn may be
unknown even with a confirmed auth cause. Recovery appends facts; retry creates
a new budgeted attempt after fresh checks and authority. Thin packaged skills
repeat those facts/next actions and link history; no chat-derived state,
automatic repair, fallback or acceptance.

## 3. Execution order and approval points (future Tasks)

This is a dependency sequence, **not authorized Tasks**. Prefer three coherent
units over one task per file. No intermediate release enables unsafe dispatch.

1. **Contract foundation, dispatch still closed:** successor codecs/strict
   version dispatch, proof-free current identity/effort observations and optional
   version/auth separation; deterministic unit and historical digest fixtures.
   Keep new contracts unreachable from dispatch until the entire composition
   preserves pre-spawn controls. No enable flag or mixed-generation fallback.
   Evidence: field/version inventory, old-byte fixtures, zero-spawn failures,
   both-adapter encoding/source/auth tests. Review D2/D4 before active wiring.
2. **One coordinated production cutover:** workflow resolution/start, compiler,
   stage plan/preview/run, single+graph and supervisor checks use the same new
   tuple/observation; remove proof loops, hardcoded versions and proof-conditioned
   confinement refusal. Preserve mandatory independent safety controls and
   disclose advisory/unknown confinement facts, not synthetic os-enforced.
   Attach D5 diagnostics at failure/terminal-save boundaries. Authority is
   durable before any child starts; no product command can reach partially
   converted dispatch. Evidence: RA-AC matrix below, per-attempt/retry checks,
   installed positive path and negative controls. This yields usable admission
   without exposing a registry-only bypass as an intermediate product.
3. **Structural retirement and journey completion:** delete disconnected active
   certification code/store wiring/commands/fixtures, update help/commands and
   packaged skills, preserve historical codecs and security/result regressions.
   Re-run installed journeys from packaged binary/skills outside checkout.
   Evidence: production dependency search shows no registry/proof/probe gate,
   deleted commands have zero effects, no root/credential/canonical-store
   regression, all RA-AC rows cover both adapters.

Before each unit, recheck remote head, active work and canonical Execution if
available. Review may combine units 2–3 when it reduces dead wiring; they must
land together before delivery. No independent implementation PR/task is opened
by this Plan. Future task/PR organization requires its own stage authority and
must respect the single AXM-36 PR policy while #334 is open.

## 4. Deterministic acceptance design

All suites below are **future obligations, not executed by this Plan**. Extend
existing production composition tests; do not inject a Proven inventory or
replace the resolver/adapter in the installed success path. Synthetic vendor
executables implement bounded auth/status and attempt behavior, with spawn and
effect counters; they are not provider capability claims. Parameterize every
applicable row over Codex **and** Claude and single **and** graph execution.

| Criterion | Existing suite owners to extend / required Evidence |
| --- | --- |
| RA-AC1 | `cmd/lingo/stage_journey_blackbox_test.go`, `runtime_policy*_test.go`, `internal/stageexecution/preview_test.go`: installed binary + packaged skills from unrelated temporary CWD, no checkout/registry/store/receipt; auth + authority reach a valid synthetic result via workflow preview/start, stage plan/run |
| RA-AC2 | `internal/runtimeadapter/{stage_invocation,frozen_stage,authpreflight,stage_effort}_test.go`, workflow binding and stage preview/apply tests: arbitrary/unknown version and new model/effort before preview accepted; path/bytes/Profile/source drift after preview has zero spawn, new preview required |
| RA-AC3 | `internal/runtimeprofile/*consumer*_test.go`, `internal/runtimeapplication`, `internal/workflowcompiler/profile_chain_test.go`, CLI policy tests: Runtime/Profile disabled, Project denial, Agent/Workflow mismatch and ambiguity; deterministic reason, no fallback |
| RA-AC4 | `internal/stageexecution`, `cmd/lingo/stage_dispatch_test.go` and installed journey: local/external/inference authority each omitted, even with login; no unauthorized spawn/effect; authority record exists before positive spawn |
| RA-AC5 | `internal/runtimeadapter/{authpreflight,effort_sources*,effort_inspector,stage_effort}_test.go`, installed adversarial journey: missing/expired/unobservable auth, absent executable, symlink/unsafe root, permission denial, malformed/unreadable config, recognized override; concrete reason, no repair; missing version/proof alone passes |
| RA-AC6 | Adapter invocation and `cmd/lingo/stage_fake_runtime_test.go` with installed journeys: reject option/model/effort, exit/signal and unrecognized diagnostic; retain process result, supported confirmed causes only, no alternate dispatch |
| RA-AC7 | `internal/executiongraph` classification/inbox and installed journey: exit 0 with absent/invalid result fails contract, preserves result-validation reason; no success from login/terminal prose |
| RA-AC8 | `cmd/lingo/stage_adversarial_blackbox_test.go`, graph/single store/recovery and supervisor tests: timeout, unobserved stop, supervisor crash, partial writes/lost result; persisted certainty/Evidence, no redispatch; live process blocks retry, reconciliation appends, budgets/authority enforced |
| RA-AC9 | Effort path/source tests, stage preview and installed adversarial tests: canonical explicit/runtime-default intent, competing selectors, read-only inbox view and repository-write child drift; attempted canonical/out-of-scope writes cannot mutate Axiom state or gain scope; disclose actual control/unknown confinement, no sandbox claim from synthetic test |
| RA-AC10 | `internal/cli/output_separation_test.go`, presentation/completion, stage diagnostic tests and packaged-skill assertions: seeded secret/config/account/path/transcript payload absent in JSON/text/persisted Evidence; correlation, labelled hypotheses and next authority/action agree |

For installed coverage adapt existing `TestInstalledSyntheticSingleStageJourney`,
`TestInstalledSyntheticGraphStageJourney`, retry/reset, supervisor crash and
reconcile tests. Remove their registry/proof fixtures as positive prerequisites;
retain obsolete inputs only as historical/refusal fixtures. Exercise CLI
permission and invalid-option errors after spawn as well as preflight errors;
verify their effect certainty independently. Check installed help and both
Runtime skill packages contain no mandatory qualification guidance.

Future implementation checks: targeted suites for runtimeprofile/runtimeadapter,
runtimeapplication, workflowcompiler, workflow, stageexecution, executiongraph,
local and CLI/composition, then `go test ./...`, `go vet ./...`, `go build ./...`,
`go mod verify` and repository validation. No production dependency may still
import the certification service or embedded registry; intentional historical
codecs must be explicitly enumerated. Real Codex/Claude E2E remains
[AXM-12](https://linear.app/rgomids/issue/AXM-12) / [#278](https://github.com/rgomids/axiom/issues/278),
with separate consent for inference/external effects. Synthetic success neither
certifies sandbox enforcement nor satisfies that gate.

## 5. Risks, review boundary and ready for implementation

| Risk / trade-off | Containment and approval Evidence |
| --- | --- |
| Removing proof also removes a disguised safety dependency | Inventory includes preview confinement refusal, exact-version builder, auth version early return, per-attempt effort and compiler; negative tests precede cutover |
| New format prevents transparent continuation | Explicit local retirement/reconfiguration, isolated historical readers, stop/reconcile before new work, no rollback downgrade; review D2 and exact successor fixtures |
| Adapter knowledge can lag vendor changes | Reviewed bounded source rules and operational diagnosis; unfamiliar version alone never blocks, concrete unhandled selectors/unsafe configuration do; vendor support is unconfirmed |
| Failure summaries misstate cause or lose unknown effects | Classification remains terminal-save-owned, bounded diagnostics separate facts/hypotheses and process/contract/effects; crash/recovery and redaction tests |
| Concurrent contracts claim format numbers or change consumers | Re-read head and only direct owners before Tasks; reconcile rather than overwrite/renumber invisibly |

No new ADR is needed within the accepted removal boundary. A proposal to weaken
policy, authority, identity, source trust, supervision, immutable history or
reconciliation would exceed it and requires a new material human decision
before dependent work. Reintroducing a qualification gate is not a permissible
implementation workaround.

Next stage **tasks** must turn the three units into bounded objectives, exact
paths, dependencies, schema fixtures, acceptance commands and rollback checks;
resolve the mechanical D4 key/source inventory using primary evidence and
review D2 field/version ownership. Do not create task issues or execute code
from this document. Human validation of this exact Plan remains pending;
Implementation additionally needs accepted Tasks and explicit scope authority.

Ready for implementation means: Plan review/decision recorded at an exact SHA;
Tasks accepted/authorized with no unresolved material contract question; current
remote/canonical checkpoint reconciled; closed-schema and historical fixture
inventory explicit; RA-AC1–10 assigned in both adapters including installed
outside-checkout coverage; independent security/authority/recovery gates and
real-E2E limits carried into Tasks. Green documentation CI alone meets none of
the behavioral acceptance obligations and does not complete AXM-36.

Constitution check: intent remains RA-1–RA-6; this Plan consumes the exact
accepted decision and keeps Tasks/Implementation separate. Evidence and
hypotheses are explicit; policy, authority and security remain independently
observable. Scope uses existing adapter, persistence and supervisor owners,
with deterministic checks and no new dependency/infrastructure. The six
product boundaries in the amendment remain intact. No constitutional waiver
or amendment is requested.

## 6. Plan verification Evidence

- `./scripts/validate-repository.sh .`: PASS (repository bootstrap, 33 ADRs,
  56 ADR-governance fixtures, automation registration and sensitive-file checks).
  Its maintainer runtime behavioral scenarios explicitly SKIP/UNVERIFIED.
- `python3 scripts/check-adr-governance.py .`: PASS, including backlinks,
  acceptance date/title and index structure. This is structural validation.
- `git diff --check`: PASS. Ephemeral offline link/anchor check: PASS,
  278 local references across the ten changed Markdown documents.
- Ephemeral comparison against the accepted SHA: all historical ADR-0025 and
  amendment lines preserved in order; ADR-0032 Decision and its alternatives/
  consequences remain byte-identical. All RA-AC1–10 appear in the matrix;
  only documentation paths changed.
- Main-session review with `axiom-review`: no blocking semantic/security finding
  identified in this documentation scope. RA-1–RA-6, supersession boundaries,
  removed admission gates and retained controls were compared to the Plan.
  This is not an independent review or human acceptance.

Staged checks and remote publication Evidence are recorded in PR #334 after
commit. `gitleaks` is unavailable; repository sensitive-file checks do not
replace a dedicated scanner. No Go/provider/runtime tests, real auth or
credential inspection, probes, inference or canonical Execution effects ran.
RA-AC1–10 remain unverified behavioral obligations for Implementation; this
Plan does not establish functional vendor compatibility or confinement.
