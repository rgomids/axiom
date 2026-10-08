# Safe Work Item Execution Targeting — Issue #137

**Status:** Specification, Plan and T137-01–T137-06 approved for bounded implementation by the maintainer on 2026-10-08 (America/Sao_Paulo). Implementation remains subject to engineering review and separate human acceptance.

**Baseline:** `b69fa837cacaa92bee865628a09edf8797777110` (`main`, v0.9.0).

Approved sources:

- [Specification](https://github.com/rgomids/axiom/issues/137#issuecomment-6051987244)
- [Implementation Plan](https://github.com/rgomids/axiom/issues/137#issuecomment-6052022664)
- [Implementation Tasks](https://github.com/rgomids/axiom/issues/137#issuecomment-6052051324)

The linked comments retain their historical Draft/Proposed and authority wording.
The maintainer's implementation request supplies the later approval and bounded
local implementation authority; it grants no push, merge, release, Issue closure,
Provider mutation or human acceptance. This artifact is a bounded amendment to
[Specification 004](spec.md), preserving its historical requirements and Evidence.

## 1. Problem

A Work Item execution must never begin against an implicitly inferred or ambiguous target.

Runtime conversations, working directories, Git remotes, and recent activity are not authoritative sources for selecting an execution target.

The execution boundary must guarantee that the selected Project and Work Item are explicit or deterministically resolved before any workflow execution begins.

## 2. Desired Outcome

Every Work Item execution has one validated, observable, and immutable target.

A user can invoke the run operation without repeatedly supplying a Project selector when an effective Project is already available.

A Work Item identifier is always supplied explicitly.

## 3. Actors

- Operator initiating a Work Item execution.
- Runtime invoking the domain Work Item skill.
- Lingo resolving and validating execution targets.

## 4. Functional Requirements

### FR-001 — Explicit Work Item identity

Every new run MUST receive an explicit Work Item identifier.

The identifier MAY use an existing supported CLI selector form, provided that it identifies exactly one Work Item within the selected Project and Repository context.

Missing, malformed, conflicting, or ambiguous identifiers MUST fail closed.

The Work Item MUST NOT be inferred from conversation history, recent activity, previous Executions, or Runtime state.

### FR-002 — Effective Project resolution

Project resolution MUST reuse the contract established by Issue #233.

Precedence:

1. Explicit operation Project
2. Session Project override
3. Persistent local default Project
4. Unresolved — fail closed

The first available selector MUST be validated.

An invalid, stale, deleted, or ambiguous winning selector MUST fail without falling back to another source.

An explicit Project MUST override session and default preferences, including stale lower-priority preferences.

### FR-003 — Project-scoped Work Item resolution

The Work Item MUST be resolved exclusively within the validated effective Project.

The existing Project-scoped Repository selector and Work Item Provider linkage contracts remain authoritative.

Resolution MUST establish one canonical target containing:

- Project identity
- Repository identity
- Provider identity
- Work Item resource and identifier

Missing, conflicting, or invalid target components MUST prevent Execution creation.

Cross-Project Work Item fallback is prohibited.

### FR-004 — Target visibility

Before the first execution gate, the operator MUST be able to inspect the resolved target.

The presentation MUST identify:

- Resolved Project
- Project resolution source
- Repository
- Explicit Work Item
- Canonical Work Item identity

Target information MUST originate from validated application state, not Runtime-generated assumptions.

The canonical execution result MUST preserve the resolved target identity.

### FR-005 — Pre-execution validation

All target validation MUST succeed before:

- Creating a new Execution
- Launching an execution Runtime
- Advancing a workflow gate
- Performing authorized workflow mutations

Invalid targeting MUST produce an actionable, structured failure.

Validation failure MUST NOT modify existing Execution or workflow state.

Read-only inspection required for validation is allowed.

### FR-006 — Stable Execution binding

The validated target MUST be bound to the Execution using canonical identities.

Changes to session or default Project preferences after Execution creation MUST NOT change its target.

An existing Execution MUST NOT be silently rebound to another Project or Work Item.

A request targeting a different scope MUST fail explicitly.

### FR-007 — Runtime skill consistency

Both the canonical axiom-work-item run operation and the compatible axiom-work-item-run alias MUST follow the same effective-Project contract.

Skills MUST NOT require redundant Project selection when a valid effective Project exists.

Skills MUST collect an explicit Work Item identifier.

Skills MUST delegate authoritative resolution and validation to Lingo.

Skills MUST NOT independently infer or persist execution targets.

### FR-008 — Preview and authority consistency

The reviewed Runtime/Profile start protocol remains mandatory.

A reviewed start MUST use the same validated target presented to the operator.

If the target or relevant validation context changes before confirmation, the request MUST fail closed and require a fresh review.

Target resolution MUST NOT grant execution, local, Provider, or human-acceptance authority.

## 5. Failure Cases

The system MUST refuse execution when:

- Work Item identifier is missing.
- Work Item identifier is malformed.
- Work Item is not linked to the selected Project.
- Repository is absent or invalid.
- Project context cannot be resolved.
- The winning Project context is stale.
- The selected Project is ambiguous.
- Project configuration is invalid or unavailable.
- Execution selectors conflict with persisted scope.
- The reviewed start context is stale.

Failures MUST expose a stable error category and actionable recovery guidance.

## 6. Acceptance Criteria

- **AC-001:** Run without an explicit Work Item identifier fails.
- **AC-002:** Run with an explicit Project and valid Work Item succeeds when all existing execution prerequisites are satisfied.
- **AC-003:** Run without an explicit Project uses the valid session override when available.
- **AC-004:** Run without explicit/session Project uses the valid persistent local default.
- **AC-005:** An explicit Project overrides session and default context.
- **AC-006:** Stale, ambiguous, or unresolved winning Project context fails without implicit fallback.
- **AC-007:** The resolved Project and Work Item are visible before the first execution gate.
- **AC-008:** Invalid targeting does not create an Execution, start a Runtime, or advance workflow state.
- **AC-009:** An existing Execution retains its original target after session/default context changes.
- **AC-010:** Canonical Work Item skills and compatible aliases produce equivalent targeting behavior.

## 7. Required Evidence

Automated verification MUST cover:

- Missing Work Item identifier
- Malformed Work Item identifier
- Explicit Project precedence
- Session Project precedence
- Default Project resolution
- Unresolved Project context
- Stale and ambiguous Project context
- Cross-Project targeting rejection
- Valid Project and Work Item binding
- Existing Execution immutability
- Preview/start target consistency
- No execution side effects on validation failure
- CLI and Runtime skill contract consistency

Tests SHOULD verify canonical results and persisted state, not only process exit codes.

## 8. Non-goals

This change does not introduce:

- A new Project identity model
- Automatic Work Item discovery
- CWD or Git remote inference
- Chat-history-based targeting
- Cross-Project execution discovery
- New Runtime or Model selection policies
- New workflow stages
- New authority semantics
- New persisted Execution format
- Automatic Provider mutations

## 9. Invariants

- Project != Repository
- Execution != Agent
- Runtime != Model
- Effective Project resolution is deterministic
- Work Item selection is explicit
- Persisted Execution identity is immutable
- Invalid or ambiguous state fails closed
- Workflow truth remains Axiom-owned
- Authority remains explicit

## 10. Constitution and ADR Assessment

This Specification preserves existing Project, Runtime, Execution, authority, and persistence contracts.

It reuses the effective-Project behavior delivered by Issue #233.

No new ADR is required unless implementation reveals a necessary change to an accepted architectural contract.

## 11. Dependencies

- #15 — MVP completion boundary
- #140 — Runtime and Model Profiles
- #231 — Project bootstrap and readiness
- #233 — Effective Project context resolution

Issue #233 is delivered. No unresolved dependency currently prevents planning this change.

## 12. Approved implementation plan and Tasks

The approved Plan orders baseline analysis, authoritative target validation,
target-bound review, immutable Execution/skill compatibility, then integrated
verification and reconciliation. Read-only inspection is allowed; every invalid
or stale target must fail before an execution effect. No persistence migration,
new workflow stage, Runtime selection policy, cross-Project discovery or authority
semantics are introduced.

| Task | Scope and dependency | Required Evidence |
|---|---|---|
| T137-01 | Baseline selectors, direct/CLI boundary, Runtime digest gaps; first | Reproduced failing regressions and baseline gap matrix |
| T137-02 | Canonical Project/Repository/Provider/linked Work Item validation; after T137-01 | Precedence matrix, structured refusal, no-effect snapshots |
| T137-03 | Target disclosure and reviewed-start binding; after T137-02 | JSON/human target, stale review and fresh validation tests |
| T137-04 | Immutable existing Execution and compatibility; after T137-03 | Equivalent/conflicting start, context changes, cancellation/recovery preservation |
| T137-05 | Canonical/alias parity, discovery and published skill compatibility; after T137-03 | Shared routing, parser/discovery and Codex/Claude receipt tests |
| T137-06 | Full verification, independent review, durable reconciliation; after T137-04/05 | AC-001–AC-010 traceability, executed checks, explicit limitations |

T137-04 and T137-05 may run independently when ownership does not overlap.
Approved validation commands and actual results live in [Evidence](evidence-issue-137.md).

## 13. Implementation details (distinct from approved requirements)

- `workflow.Service.ValidateTarget` shares the authoritative resolver with
  `Start`. The internal target requires the existing canonical positive numeric
  identifier; the application accepts the exact qualified selector or explicit
  legacy number and rejects conflicting/malformed fields even for direct calls.
- Resolution uses the canonical Project UUID for linked-item reads. Provider,
  resource and identifier must match the explicit inputs. Ambiguous linkage and
  recovery remain distinct failures. The winning effective Project comes from
  the unchanged #233 resolver; CLI parser injection retains the original source.
- `workflow start` returns `executionTarget` before any Execution exists:
  `projectId`, `projectSource`, `repositoryKey`, `provider`, `resource`, `externalId`,
  and the qualified `workItem`. No local Repository path is disclosed.
- The start `previewDigest` is SHA-256 over a domain-separated
  `axiom-workflow-start-v1` envelope: canonical target (including Repository
  binding and complete linked Work Item), winning Project source and existing
  Runtime/Profile preview digest. The independent `runtime profile preview`
  API and policy digest are unchanged. Its token, and older policy-only start
  tokens, cannot approve a start under this contract; request fresh start review.
- Confirmed start re-resolves all inputs, compares the composed digest, calls the
  existing fresh Runtime policy `Check`, rechecks the effective Project winner,
  then revalidates the Repository/link in `Start` against the reviewed snapshot.
  Source, binding or linkage drift requires fresh review. UUID/slug equivalents
  remain equivalent when they resolve the same explicit Project.
- The ephemeral validation snapshot is never persisted. Execution v1 and its
  equivalent/conflicting-start behavior remain canonical. Successful results use
  persisted `workflow.workItem`/Runtime/Execution identity; existing transitions,
  history, Evidence and provenance remain bound to that identity.
- Skills route both Runtime integrations to the same Lingo application. They
  preserve identical selectors and session for reviewed start; later operations
  use the persisted canonical selectors. Previous published skill receipts remain
  recognized through the append-only shared revision history.

No new ADR is required: this implements the approved amendment using existing
Project, authority, Runtime and local-state boundaries. ADR-0003, ADR-0005,
ADR-0007, ADR-0008/0009 and ADR-0014/0016 remain applicable and unchanged.
