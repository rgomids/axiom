# Clarifications — Lingo Project Initialization

## Status

2026-09-10 — Specification/Clarification only. [Specification 002](spec.md)
is Proposed for human review. No answer below is a new human approval unless
explicitly attributed to the current request or an Accepted ADR.

## Resolved from authoritative input

| ID | Clarification | Basis and effect |
|---|---|---|
| R1 | Project is independent of Repository and Runtime | Current request; Accepted ADR-0001 and ADR-0003. No directory/provider identity or new ownership hierarchy. |
| R2 | Axiom owns contracts; Lingo executes them | Accepted ADR-0003. No domain vendor coupling or adapter implementation. |
| R3 | Spec-Kit is strategic upstream reference | Accepted ADR-0002. No compatibility, dependency, adapter or watch implementation. |
| R4 | Secrets never belong to portable Project configuration | Current request and security policy. References only; no credential store selected. |
| R5 | Repositories can be remote-only, non-sibling and on distinct Providers | Current request. Local binding is separate; submodules not required. |
| R6 | Integration does not mean MCP; Role does not mean concrete model | Current request and conceptual model. Preserve explicit boundaries. |
| R7 | This turn stops at Specification/Clarification | Current request. No code, Plan, tasks, new CI, external mutations or migrations. |

## Proposed resolutions, not accepted decisions

These defaults make behavior reviewable. They remain part of the Proposed
Specification; recording them here does not claim a human answered a question.

| ID | Proposed answer and rationale | Specification |
|---|---|---|
| P1 | Opaque persistent Project ID; unique Repository key within Project. Names/paths change independently. ID encoding/allocation remains Q1. | FR-001–002 |
| P2 | Require schema, ID, name, one Repository, and explicit Runtime/Provider/context states. Permit `unconfigured`, empty profiles and no credentials. Proves portability without setup adapters; human confirms MVP in Q5. | Minimum table |
| P3 | Keep portable intent authoritative; local records bind its stable keys and validated content revision. No implicit Workspace catalog or ownership claims. | Portable/local table |
| P4 | Init creates; identical rerun reopens without write; changed intent conflicts. No update/force mode. Safer than automatic merge before conflict semantics exist. | FR-012 |
| P5 | Reopen/install included in this slice; command name hypothetical. No automatic cloning. Missing local bindings are explicit, portable bytes unchanged. | J3–J4, FR-014 |
| P6 | Optional Provider selection and requested Capabilities may exist without operational Integration. No live verification claim. | FR-007 |
| P7 | Runtime availability and Model Profiles do not gate structural validity. Presence check does not prove capability readiness. No model discovery. | FR-005–006 |
| P8 | Credentials are logical references with machine-local source bindings. Do not retrieve values even to validate references. | FR-008 |
| P9 | Minimal Business Context is purpose text or explicit `unconfigured`; no ontology/chat import. | FR-009 |
| P10 | Strict core fields; unsupported versions fail without writes/migration. Exact format/version support remains Q2. | FR-015 |
| P11 | Guided interaction first; flags deferred, absent interactive input fails promptly. Preview before writes, cancel/retry and partial-persistence recovery explicit. | FR-010–013 |
| P12 | Separate portable validity, local installation, operational observations. Do not invent a Project lifecycle. | Validity and failures |

## Open questions for human review

No Plan should begin until Q1–Q6 have an explicit answer or a documented
scope reduction resolving their impact. Recommendations are proposals.

| ID | Decision and viable alternatives | Recommendation, trade-off, and what becomes harder to change |
|---|---|---|
| Q1 | Project identity allocation/encoding: generated opaque UUID-like value, user-assigned namespaced identifier, or content-derived identity | Prefer generated opaque ID retained on copy/rename. Avoids naming authority and mutable content identity; requires explicit fork/collision semantics. Changing IDs later disrupts every local binding. Specify format, collision detection, and copy-as-new boundary before planning; assess ADR need then. |
| Q2 | Portable format/layout/version: single structured manifest vs split documents; YAML vs JSON; initial schema token and supported-version matrix | Prefer a single manifest initially for coherent validation/writes; format remains undecided. Split files improve editing separation but complicate cross-file integrity. Explicit version plus strict core validation proposed; approving wire format creates long-lived compatibility cost. Future migration must require separate scope/authority. |
| Q3 | Local state root/platform contract: OS-native application directories vs uniform home directory vs explicitly configured external root; initial supported OS coverage | Prefer OS-native private storage with a testable explicit root override outside portable files. Portability across installations needs platform permission/path tests; supporting all platforms immediately increases scope. Confirm first release OS matrix, permission expectations, source relocation and multiple-copy policy. Location choice makes discoverability/migration harder later. |
| Q4 | Runtime validation without adapters: report unverified, check explicit local executable presence, or defer slice until a real Runtime probe is specified | Prefer honest unverified/presence-only observations now. Fits no-adapter scope but does not demonstrate runnable Runtime. Confirm this satisfies the requested availability validation; stronger validation needs a separately defined bounded contract, not fabricated adapter. Model availability stays unverified. |
| Q5 | MVP minimum: one Repository plus explicit unconfigured Runtime/Providers/context vs mandatory Runtime and purpose text vs connected environment | Prefer minimum table in spec. Proves portable Project round trip offline; delivers less immediate execution readiness. Confirm unresolved local bindings may still yield installed-with-gaps, and reopening/installation belongs in slice. Changing requiredness later affects existing configurations. |
| Q6 | Repository duplicate equivalence: exact/safe syntactic locator matching vs transport alias matching vs provider-resolved identity | Prefer strict key uniqueness and conservative syntactic matching with explicit alias conflicts. Avoids network/provider coupling; cannot deduplicate every alias offline. Define supported locator forms (including SSH shorthand), safe normalization and mismatch resolution examples before planning. No global Repository identity/ownership is selected. |

The task can complete as a reviewable Specification with these questions open.
They block planning, not this documentation delivery. No human answer was
solicited solely to choose speculative implementation details during authoring.

## Possible ADRs, not created

- **Identity (Q1):** candidate because durable references make replacement costly.
- **Manifest format/version boundary (Q2):** candidate if adopted as a public,
  long-lived portability contract; alternatives and compatibility costs need review.
- **Local state location (Q3):** candidate if cross-platform discovery, permission,
  and migration rules become durable architecture. A reversible implementation
  detail alone would not warrant an ADR.
- CLI framework and persistence engine remain outside this phase; no current
  requirement justifies selecting either or creating an ADR for them now.
- Credential store remains deliberately unselected. This slice's reference-only
  behavior does not require a store ADR; actual resolution may justify one later.

No ADR is created or accepted automatically. Accepted ADR-0001–0003 are unchanged.

## Coverage of requested questions

| User question | Proposed answer / remaining decision |
|---|---|
| 1. Unique Project identity | FR-001; Q1 encoding/allocation |
| 2–3. Required fields / unconfigured | Minimum configuration table; Q5 approval |
| 4–5. Portable/local relation / source of truth | Portable/local table, content revision binding |
| 6–7. Repository representation / paths | FR-002–004; Q6 locator rules |
| 8–11. Existing Project / idempotency / files / update | FR-012–013: explicit target, no-op or conflict, create-only |
| 12. Another machine | J3, FR-014, AC-03 |
| 13–14. Schema version / incompatible input | FR-015; Q2 wire representation/support matrix |
| 15. Credentials | FR-008, SEC-001; no store chosen |
| 16–17. Runtime selection / unavailable | FR-005; Q4 availability evidence |
| 18. Model Profile required? | No; FR-006, subject to Q5 minimum approval |
| 19–20. Optional Provider / unconfigured Integration | Minimum table, FR-007 |
| 21. Valid Project | Three validation dimensions; explicit gaps |
| 22. Minimum acceptance Evidence | AC-01–12 and future delivery Evidence paragraph |

## Review and documentation reconciliation

README had stale Proposed references to ADR-0003 despite the ADR's Accepted
status. Reconciled those references to current authoritative state. Added
Specification discovery links in the specification index, README and roadmap,
and recorded this documentation change in CHANGELOG. No conceptual-model,
Provider-boundary, ADR, harness, command, or security-policy rewrite was needed.
Future command examples stay in the Specification, not executable commands docs.

Review recommendation: ready for human review of behavior and open questions;
not approved, not ready for planning until Q1–Q6 are resolved. No blocking
architectural conflict identified in document review. Executable acceptance
remains entirely unverified because no Lingo code exists.

## Validation of this documentation change

Executed from repository root on 2026-09-10:

- `./scripts/validate-repository.sh .` — passed; includes harness structure,
  package-validator regression tests, sensitive-file-checker regression tests,
  worktree sensitive-file scan and `git diff --check`.
- Local Markdown link existence check over both new documents and changed
  README/index/roadmap — passed.
- Requirement identifier presence check — passed: FR-001–016, SEC-001–005,
  AC-01–12. This checks coverage labels, not semantic correctness.
- Diff and document review against ADR-0001–0003, conceptual model, Provider
  boundaries, roadmap, Constitution and SDD workflow — no blocking conflict
  identified; Q1–Q6 remain explicit planning gates.
- `command -v gitleaks` — unavailable (exit 1); consolidated secret-scanner
  coverage remains unverified. No scanner was installed.

These checks validate documentation/harness hygiene only; they do not execute
AC-01–12. No new software tests are created for this Specification-only change;
future unit, integration and functional test obligations are specified in the
acceptance matrix. During PR preparation, the repository validation and
`./scripts/check-sensitive-files.sh --staged .` also passed for this change.
