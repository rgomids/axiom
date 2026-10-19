# Plan — AXM-32 output separation

## Status and boundaries

**Proposed for review — plan only, 2026-10-18.** Translates the direction of
[Specification 008](spec.md), explicitly [accepted at `18d29a6`](https://github.com/rgomids/axiom/pull/328#issuecomment-6246060142),
and [Accepted ADR-0030](../../decisions/0030-canonical-results-terminal-and-host-presentation.md).
The [READY_FOR_plan record](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia#comment-9e16edb9)
and this request authorize documentary work on the single Draft PR #328.
This Plan has not been accepted; it creates no Tasks or implementation authority.

The [R1 inventory](clarify.md) and two subsequent maintainer decisions are inputs.
The [ADR acceptance / G1–G2 decision](https://github.com/rgomids/axiom/pull/328#issuecomment-6250321308)
keeps G1/G2 as observations, not preventive repairs or blockers. The later
[explicit R1 decision](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia#comment-f5c308fd)
confirms that Clarify's concrete producer/consumer, fact and gap inventory satisfies
ADR-0030's pre-Plan inventory requirement **for planning only**, despite unverified
dynamic variants. Inventory with Evidence is distinct from universal equivalence
proof: this decision neither asserts universally lossless JSON nor waives R1 or
AC1–AC9. Gaps and unverified coverage still require Evidence before behavioral
acceptance; a material defect preventing the accepted contract needs its own
review and authority. The [reconciliation Evidence](evidence.md#plan-correction--r1-decision-and-index-2026-10-18)
records the round-3 responses and verification limits.

Earlier stage/status statements in accepted artifacts and Clarify remain
historical; no accepted Spec/ADR text is rewritten. D1–D3 are closed decisions.
No new ADR candidate is needed within this boundary; a material contradiction
requires specific Evidence and a human decision before changing the contract.

Scope is presentation and coordinated consumption, within AXM-32. No domain,
readiness, persisted-state or JSON-schema migration; no #305 policy editor,
Execution reconstruction, Provider/Runtime invocation, lifecycle change or
AXM-12/#278 E2E acceptance. Preserve Axiom != Lingo, Project != Repository,
Role != Model, Runtime != Model, Execution != Agent, Provider != Transport,
Integration != MCP, Skill != workflow source of truth, Evidence != raw chat.

## Components and contracts

The flow branches after serialization: application/domain result → existing typed
DTO and canonical JSON → native Lingo terminal presentation **or** explicit JSON
consumption by the operating AI → host Markdown. The host never parses terminal
text. There is one operation observation and no second use-case invocation to
obtain another view.

| Boundary / concrete surfaces | Planned responsibility and preserved contract |
|---|---|
| Producers: `cmd/lingo/project_lifecycle.go`, `internal/projectapp/readiness.go`, `internal/workitem`, `internal/workflow`, `internal/completion` | Keep classification, readiness, authority, effects and recovery in application/domain services. Existing `cli.Result` and invocation supply observation/identity; renderer and prompts cannot decide policy or transitions. |
| DTO/emission: `internal/cli/cli.go::emitResponse`, `completion.go`, `readiness.go`, operation-specific emitters | Retain typed payloads and explicit JSON, names/types/omission rules, status/enums, limits, revision/digest, provenance, refs and diagnostic schemaVersion 1. No new root operation/target field or envelope. Invocation identity supplements typed targets; unresolved identity stays unknown. Do not extract typed meaning from `result`/`next` prose. |
| Terminal: `internal/cli/presentation.go::presentEvent`, currently `renderMarkdown` / fenced fallback | Replace Markdown traversal with a bounded deterministic native view over the same event. Operation-specific projections select material facts from typed DTO members, not prose heuristics or a parallel authoritative result tree. Unknown payload/required fact must be explicit, never silently dropped. Keep write/size failures visible, without re-execution. |
| Entry/channel: `cmd/lingo/main.go`, `internal/cli/cli.go::RunInteractive`, `parseOutputMode`; help/version/legacy writers | Keep existing leading global flags and refusal behavior. `Run` deliberately prepends JSON for programmatic callers; retain that distinction from interactive executable defaults. Classify help/version/skill-inspect and legacy POC emission separately before editing; no forced universal schema. |
| Host: `internal/codexruntime/skills/{axiom-project,axiom-workflow,axiom-work-item,axiom-runtime}/SKILL.md` | All four request leading `--json` internally, including explicit skill invocation; only an explicit raw-JSON request relays JSON unchanged. Normal reply uses objective → effective result → material blockers/risk/decision → next action; empty categories need no heading. Replace mandatory literal relay and Auxiliary/dump obligations together. No second formatting LLM call, new agent or Runtime hop. |
| Packaging: `internal/codexruntime/runtime.go`, `integration.go::NewClaude`, `manifest.go`, `upgrade.go` | Shared embedded skill content reaches both hosts through distinct roots/receipts. Preserve ownership, exact bytes/digests, upgrade conflicts and published history; do not invent separate Claude production skill sources. |

Channel policy: no flag and `--human` select native human output in TTY **and**
pipe/redirection; only `--json` selects machine output. Keep conflicting/repeated/
misplaced flag refusals and help behavior. Pass presentation capability information
from the executable boundary without putting terminal detection into domain code.
Start with plain static text, which meets the contract; enable ANSI only on positively
supported terminals. Unknown capability/non-TTY stays plain; presence of `NO_COLOR`
suppresses color. No spinner/cursor rewrites, essential color, Markdown layout or
fenced JSON fallback. Width may wrap prose, never crop actionable IDs/digests.
JSON stdout contains exactly one document; prompts/transport diagnostics remain
separate. Broken output returns failure even after confirmed effects; it cannot
imply rollback. Keep success → 0, interrupted → 2, all five other completion
statuses → 1. Add no blocked/no-op/recovery completion enum.

Both human surfaces retain all material blockers with scope, warnings, confirmed
and uncertain effects, authority conditions and trust qualifications. Details and
canonical provenance remain retrievable by existing references; dirty/unknown or
mismatched source state material to trust is visible. Preview effects are proposed,
not committed. Consent binds exact target, preview, digest and revision; drift needs
renewed authorization and apply requires canonical read-back. Summary assent grants
nothing. Partial retains committed effects; no-op claims no new effects; denied,
interrupted and recovery-required outcomes preserve their actual status and safe
inspection boundary. Retryable does not prove no effects or authorize replay.
Malformed/truncated/unsupported JSON, missing status, wrong target or contradictory
observations stop dependent actions and lead to bounded uncertainty/handoff with
known refs. Never invent capability, effect, policy cause or acceptance.

Smallest proving example: `project validate` returns completion `success` / exit 0,
`readiness.structure=valid`, effective `partial`, Work Item ready and an Execution
blocker `runtime_policy_unavailable`. Native text and conversation must say
“Project estruturalmente válido; execução bloqueada por política de Runtime”,
retain the Work Item readiness and all other blockers, and avoid calling the whole
Project invalid or ready. That policy explanation is conditional on actual typed
facts; `runtime_resolution_blocked` alone does not establish its cause. Missing
readiness is unobserved. #305 remains a separate unsupported remediation boundary.

## Delivery sequence and compatibility

These are implementation phases for later decomposition, not Tasks or authority
to execute them. Each phase depends on the preceding evidence; none is independently
released with an incompatible skill/binary combination.

1. **Freeze consumer baseline and verification inputs.** Extend the bounded R1
   inventory only for touched event variants. Record invocation, target, DTO shape,
   emitted facts, missing/unknown behavior, size limits and downstream callers.
   Preserve current JSON goldens and #232/AXM-31 acceptance as historical evidence.
   Inventory `internal/cli/testdata/presentation`, `presentation_test.go`,
   `completion_test.go`, `domain_skill_routing_test.go`, `cmd/lingo/skill_protocol_test.go`,
   installed-skill assertions and command/help examples. Search scripts and release
   packaging for default-human parsers; change machine callers to explicit JSON.
   `scripts/smoke-prepared-artifact.py` already parses JSON but has its own 64KiB
   limit; do not conflate that with operation-specific emission budgets.
   Exit: named consumers/fixtures and an explicit unknown external-consumer boundary.
2. **Prove a narrow vertical slice.** Use existing read-only `project validate`
   producer/readiness emitter, native projection and the Project skill's JSON
   consumption in isolated host roots. Cover success/0 with blocked Execution,
   exact facts and zero additional service calls. Introduce only presentation-local
   helpers and injectable capability context needed for deterministic checks;
   no domain refactor, shared result schema or generic formatting service.
   Exit: the proving example and channel variants satisfy the future AC checks.
3. **Complete the coordinated presentation replacement.** Expand native views to
   every canonical emitter in the inventory and all Spec matrix outcomes. Remove
   Markdown/fenced fallback from the CLI human path. Update all four shared skill
   sources and their content/routing tests in this same AXM-32 delivery, retaining
   bounded AXM-31 diagnostics, observed/not-observed distinctions, budgets and
   domain ownership. Exit: no mandatory literal relay/Auxiliary duplicate remains;
   JSON and semantic equivalence regressions pass for the touched variants.
4. **Protect distribution and rollback of the set.** Record the replaced embedded
   revision in `manifest.go::sharedSkillHistory`; append immutable published-byte
   fixtures/receipt recognition as appropriate, never rewrite old pins. Publish a
   distinguishable skill-set revision using existing manifest/receipt versioning;
   validate whether `BinaryCompatibility` needs a presentation-protocol bump with
   the old/new pair matrix, rather than changing JSON/schema versions. Numeric
   allocation belongs to implementation review. Verify new/new and old/old pairs;
   new binary + old skills must require recognized upgrade before relying on the
   new host contract; old binary + new skills must be refused/held for a compatible
   pair, never treated as safe merely because JSON parses. Use existing runtime
   inspect/install and protected upgrade boundaries; check gaps in enforcement
   before claiming mismatch fail-closed. Modified/foreign/unsafe roots stop replacement.
   Exit: both host roots converge on verified digests/receipts; no-op reinstall and
   interrupted upgrade preserve ownership/state, with mismatch evidence.
5. **Reconcile and collect delivery evidence.** Update command/help references and
   prospective #232/AXM-31 presentation references under later authority, preserving
   their dated history. Run isolated deterministic suites, then separately authorized
   native observations in both hosts. Record exact binary SHA, skill digests, commands,
   statuses/exits and sanitized facts. Only a verified compatible set is eligible
   for later release review; no merge/publication or E2E acceptance follows here.

Rollback restores an explicitly verified earlier binary **and all four skills**
with matching receipts in each host, preserving operational state and user-modified
content. Do not assume an old installer recognizes newer bytes: if protected
replacement refuses them, stop and use an authorized pair-restoration procedure
proved in isolation. No forced overwrite, receipt forgery, state deletion or blind
mutation retry. Automation stays on explicit compatible JSON. External human-text
parsers are a communication/upgrade risk, not a new parsing contract.

## Future verification and Evidence

All checks below are **NOT_RUN in plan**. Extend named existing surfaces; proposed
fixture/oracle additions are test artifacts, not a new product wire schema.

| AC | Deterministic future check / source surface | Required evidence |
|---|---|---|
| AC1 | `internal/cli/completion_test.go`, `completion_structure_test.go`, `presentation_test.go`: decode one complete document, preserve typed JSON goldens, statuses/exits/limits and count fake use-case calls for both selections | Same invocation/target and observation, one call, no prose extraction, no banner/ANSI on JSON stdout; visible short/failed writes and no replay |
| AC2 | `presentation_test.go` / `testdata/presentation`: native snapshots for every Spec outcome; repeated equal input/capability gives identical bytes | No renderer-generated headings/emphasis/fences/tables or dump fallback; literal hostile user data distinguished from renderer syntax; all material facts retained |
| AC3 | `internal/codexruntime/runtime_test.go`, `workflow_skill_content_test.go`, `internal/cli/domain_skill_routing_test.go`, `cmd/lingo/skill_protocol_test.go` | Four skills, natural/explicit entry, internal leading JSON without a user raw-output request; reject mandatory literal/Auxiliary relay and extra formatter calls |
| AC4 | Extend presentation scenarios with a typed fact oracle and conversation fixtures, plus isolated four-skill journeys using fake tool results | Mandatory facts and exact actionable tokens match; unsupported claims fail; zero extra formatting/tool effects. Native observations supplement, never replace, deterministic checks (method below) |
| AC5 | `cmd/lingo/bootstrap_readiness_test.go`, `operator_journey_test.go`, readiness fixtures | Valid/ready, valid/partial with blocked Execution, valid/blocked, invalid structure, unknown readiness, non-policy and multiple blockers; policy phrase only with R4 predicates, including success/0 |
| AC6 | CLI parser/help tests, `cmd/lingo/blackbox_test.go`, `hierarchical_help_blackbox_test.go`, presentation writer/capability fixtures | Default/human/JSON; conflicting/repeated/misplaced flags; TTY/pipe/redirect; separate stderr; NO_COLOR present including empty; unsupported ANSI; narrow width; write errors; equal status/effects and no automatic JSON |
| AC7 | Existing work-item lifecycle/digest tests, `cmd/lingo/operator_diagnostic_test.go`, `operator_budget_test.go`, stage adversarial fixtures | Preview vs committed effects, exact consent, digest/revision drift, canonical read-back, no-op/denied/partial/interrupted/recovery and uncertain acknowledgement; mutation counters prove no replay or new authority |
| AC8 | Hostile presentation fixtures, stage adversarial tests and isolated skill-tool harness canaries | Synthetic secrets redacted before JSON/detail publication; safe facts remain. ANSI/bidi escaped, hostile Markdown and Provider/workflow instructions treated as data, malicious refs never followed; zero leaked canary or unintended tool effects |
| AC9 | `internal/codexruntime/domain_skills_upgrade_test.go`, `shared_history_test.go`, `published_receipts_test.go`, `upgrade_test.go`; prepared artifact smoke | Exact old published bytes in isolated Codex/Claude roots; old/new pair matrix, recognized upgrade, mismatch stop, no-op, partial/interrupted recovery and modified/foreign preservation; immutable pins and persisted snapshots unchanged |

For AC2/AC4 build expected obligations from typed fixture members and invocation:
completion, readiness per operation, each material blocker/warning/effect and its
certainty, authority, exact refs/revisions/digests and provenance qualifications.
Keep an explicit unsupported/unknown list; absence grants nothing. Compare native
output with deterministic expected lines/tokens and negative assertions. For host
fixtures pair candidate Markdown with a test-only claim ledger pointing to source
facts; automated checks require every obligation, exact tokens and no unsupported
status/effect/authority claim. Include negative candidates that omit a second
blocker, invent success, misattribute policy, alter a digest or hide partial effects;
they must fail. The ledger does not excuse contradictory prose: checked fixture
phrases/negative assertions must cover those claims. This bounded oracle tests
known scenarios, not arbitrary natural language; static skill-string tests or an
LLM judge alone cannot prove native fidelity.

Separately authorized native observations must exercise ordinary success, blocked
validation, denied consent and interrupted/partial recovery **in both Codex and
Claude**, using synthetic isolated Projects, roots and fake Provider/Runtime effects.
Capture bounded sanitized response/fact comparisons and tool-call traces to establish
internal JSON, normal host synthesis and absence of a dedicated formatting call.
Do not invent host invocation flags/APIs; verify the installed native interface at
that later gate. Isolation smoke is deterministic integration evidence; native
conversation observation is a distinct required claim, still unverified now.
Store commands/exits, expected/observed obligations, revision/digests and limits in
[Evidence](evidence.md), never raw chat, secrets or personal incident data.

## Risks and exit criteria

G1 category loss on no-payload ambiguous create and G2's 64-item transfer view stay
as [observations](clarify.md#subsequent-maintainer-decision--2026-10-18-after-clarify),
without planned fixes, schema additions, extra migration or Work Items. Actual
ambiguous-create impact or missing transfer effect identities, especially partial
without manifest, triggers a minimal reproduced report and a separate focused
correction per gap. Never replay uncertain external effects or call a truncated
ledger complete. Essential uncertainty/limit checks remain; no full JSON coverage
is claimed. A material contradiction preventing the accepted Spec in this scope
requires specific Evidence and a decision, not a silent waiver or preventive fix.

Other risks are host omission/injection, incomplete dynamic-payload inventory,
unknown downstream human-text parsers and mixed installed revisions. The typed
oracle, explicit JSON, fail-closed dependent actions, protected pair upgrades and
native observations address the planned boundary without transferring authority.
Redact secrets before canonical/detail emission; escaping display controls must
not alter authorized actionable tokens. A discovered redaction gap must be reported
rather than silently publishing a secret to meet “complete JSON”.

Plan-correction exit: only this Plan, appended Evidence and the Specifications
index change to reconcile R1 and the round-3 Minor; documentary checks and PR
metadata are validated; accepted artifacts remain intact; no
behavioral PASS, Tasks, Execution transition or acceptance is claimed. Future
implementation exit requires AC1–AC9 evidence, compatible binary/four-skill pairs
in both hosts, rollback proof and explicit limits on unobserved consumers.
Next candidate stage is **tasks**, subject to separate review/acceptance of this
exact Plan and its own gate/authority. No automatic advancement.
