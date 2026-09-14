# Clarifications — Lingo Project Initialization

## Current status — 2026-09-14

**Ready for human re-review.** Subsequent human review of PR #4 explicitly
reopened Specification 002's location, minimum and update contracts. Original
2026-09-11 approval is preserved below as historical evidence. H1–H8 supersede
only the stated portions; original Q1–Q6 are not silently relabeled or erased.
H9 below records the latest human confirmation and update-contract refinement.
Specification and Plan now consume these later human-approved decisions. Tasks
and Implementation remain blocked pending explicit final Plan approval and
authorization to advance.

## Latest human decision — 2026-09-14

Authority: [Human review decision — Project update contract](https://github.com/rgomids/axiom/pull/4#issuecomment-5664182508),
posted by `rgomids` on 2026-09-14 at 12:47:22 UTC. This is the authority for
this reconciliation; it confirms rather than replaces the prior Project model.

| ID | Approved revision | Supersession / traceability |
|---|---|---|
| H9 | Commands may express partial patches/intents. Lingo loads current Project; the domain materializes a complete proposed state and validates all invariants; application enforces applicable authority/version/concurrency, presents the required safe diff and persists only the complete valid result. No direct partial mutation of `axiom.yaml` or domain/application validation bypass. Init stays create/no-op/conflict; update is explicit mutation. | Refines H4, FR-012/018, AC-15–16 and Plan use-case/test boundaries; extends ADR-0004 without replacing H1–H8 |

The same comment reconfirms Project != Repository; immutable canonical UUID v4;
mutable slug/name with slug locating the portable working copy; portable
`projects/<slug>/` versus local `state/projects/<id>/`; minimal init with later
Repository/document association; independent optional Git backing; and separate
local mutation, optional local Git commit and remote push/sync authority. No
remote mutation is implicit. Earlier declaration-state and formatVersion decisions
remain unchanged. No Tasks or Implementation authorized; final Plan approval is pending.

Partial input is resolved, not an open choice between patch-only validation and
complete-state validation. Remaining init/update and Git/authority concerns are
listed in [Plan review concerns](plan.md#remaining-review-concerns-and-deferred-design).

## Subsequent human decisions — 2026-09-12

Authority: explicit human reconciliation request during PR #4 Plan review, after
original Specification approval. No agent approval is inferred.

| ID | Approved revision | Supersession / traceability |
|---|---|---|
| H1 | Per-user logical Axiom root: `projects/<slug>` is portable/shared working copy; `state/projects/<id>/installation.json` is machine-local only. No config created inside associated Repository. Native Linux/macOS state mappings and test override retained. | Refines Q3/P3; FR-017; SEC-003/005; AC-03, AC-13–14, AC-17 |
| H2 | UUID v4 canonical immutable id; installation-unique mutable slug; mutable nonunique display name. Full slug grammar `[a-z0-9]+(-[a-z0-9]+)*`; explicit collision resolution and protected rename/move with ID/state continuity. | Extends Q1; FR-001/017; AC-13–14/16 |
| H3 | Init requires only schemaVersion and Project id/slug/name. Repositories may be absent/empty. Context, policies and documents can arrive later through update. Minimal directories require no preexisting referenced files. | Supersedes Q5's >=1 Repository minimum; FR-010/018; AC-01/11/15 |
| H4 | Init remains create/no-op/conflict with no force; explicit update legitimately evolves same Project, validates proposed state, previews safe diff, checks authority/revision, persists atomically and preserves old state on pre-commit failure. | Supersedes P4's exclusion of update and create-only question 8–11 answer; FR-012/018; AC-15–16 |
| H5 | AI may propose/explain drafts; application/domain validation, authority, filesystem, versioning, identity and persistence remain deterministic. | Extends P9's no required AI boundary; FR-019; SEC-002; AC-15 |
| H6 | Optional dedicated portable Git backing, independent of Repository associations; only supported portable artifacts exported. No primary symlink sync strategy. | Refines previously open distribution boundary; FR-020; AC-17–18 |
| H7 | Local mutation, optional local Git commit and optional remote push/sync have separate authority and outcomes. No implicit remote mutation; deliberate future autoPush requires explicit authority. Git execution remains deferred. | FR-021; SEC-002; AC-18 |
| H8 | Extend Accepted ADR-0004 for these durable portable/local boundary decisions; preserve original approval and earlier Plan decisions on declaration forms and local formatVersion. Final reconciled Plan needs human re-review. | ADR-0001–0003 unchanged; no new ADR, Tasks or implementation |

These changes carry compatibility cost: slug becomes required and zero associations
becomes valid in the unreleased schema proposal. No executable schema or migration
exists; do not imply compatibility with old drafts or silently migrate files.
No global slug uniqueness, aggregate ownership, Git adapter or sync engine decided.

The remaining sections record the original review only; their dated statements
about PR #3, create-only scope, minimum and next Plan phase are historical and
are superseded where H1–H8 say so. Current requirements live in [spec.md](spec.md)
and current design in [plan.md](plan.md).

## Original approval — 2026-09-11

2026-09-11 — Specification 002 approved. [Specification 002](spec.md) is
Approved by final human review on 2026-09-11. Q1–Q6 remain Resolved by human review.
No blocking clarification remains for planning. Approval closes intake → specify
→ clarify and authorizes Plan as the next phase after PR #3 is merged, in a new
change. It does not authorize direct implementation; an approved Plan is still
required. This PR creates no Plan, Tasks or implementation.

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

## Resolved by human review

Basis: [PR #3 human review](https://github.com/rgomids/axiom/pull/3#issuecomment-5629229463)
and the follow-up revision request. These decisions replace the earlier open
questions and proposed defaults where they differed. No Q1–Q6 blocker remains.

| ID | Human-approved decision | Rationale, trade-offs and traceability |
|---|---|---|
| Q1 — Project identity | Opaque UUID v4 generated by Lingo, persistent and immutable. Rename, reopen, Runtime change, machine installation and copy preserve ID. Independent identity/fork is outside this slice. No aggregate ownership decision. | Avoids naming authority and mutable content identity, unlike user-assigned or content-derived IDs. Copies retain correlation; independent forks need future scope. Changing identity later disrupts bindings. FR-001–002; AC-01, AC-03–04. |
| Q2 — Portable format | One portable manifest, `axiom.yaml`, with explicit `schemaVersion` and strict parsing. Unknown core fields, duplicate fields and unsupported schema versions fail before writes. No automatic migration. Split manifests are outside MVP and may be reconsidered later. | Single manifest reduces cross-file consistency/persistence cost; split manifests improve editing separation but complicate integrity. YAML is selected over leaving YAML/JSON open. Format creates compatibility cost; avoid further schema choices beyond this contract. FR-015; AC-01, AC-06. |
| Q3 — Machine-local state | OS-native directories; first support Linux and macOS. Explicit testable root override for development/tests; exact override name remains a Plan detail. State never enters axiom.yaml, never depends solely on .gitignore, and uses platform-appropriate permissions. Windows outside MVP; architecture must permit future support. | Native storage fits platform conventions versus a uniform home directory; override makes isolation testable. Multiple platforms require permission/path coverage; root/discovery changes later need care. Portable/local table, SEC-005; AC-03, AC-09. |
| Q4 — Runtime validation | Observable available / unavailable / unverified with check basis. Safe presence-check permitted; absence may be reported. No capability or model readiness inference; no invented arbitrary Runtime execution. Unavailable/unverified does not invalidate portable Project or prevent installation. | Presence-only evidence fits the no-adapter slice but cannot prove runnable workflows. Stronger validation needs a future Capability contract; waiting for real probes would unnecessarily expand this slice. FR-005–006; AC-05, AC-12. |
| Q5 — Minimum MVP | Only Project ID, Project name, >=1 Repository association and schemaVersion required. Runtime, Providers, Integrations, Model Profiles, Business Context, credential references and additional policies may be absent/unconfigured. No mandatory Provider without concrete workflow/Capability need. Creation, reopen and second-machine install remain included. Installation may complete with unresolved bindings: valid portable Project, explicit installation gaps, no operational readiness claim. | Minimal offline round trip avoids artificial Provider cardinality/defaults and connected-environment requirements; immediate execution readiness is lower. Later mandatory fields affect compatibility. Minimum table, J1–J4, FR-007, FR-009–010, FR-014; AC-01, AC-03, AC-05, AC-11–12. |
| Q6 — Repository equivalence | Conservative syntactic matching. Project-scoped association key is identity; remote locator is not. Exact/safely normalized duplicates may be rejected (FR-004 retains rejection); SSH shorthand and HTTPS are not automatically equivalent. Ambiguous aliases require human resolution. Provider-resolved semantic identity is a future Capability; no network dependency solely for deduplication. | Avoids provider/network coupling versus semantic resolution; cannot deduplicate every alias offline. Preserve case-sensitive paths and avoid speculative normalization. No global identity/ownership decision. FR-002–004; AC-02, AC-04. |

## Security refinements from human review

- **SEC-001 / AC-08:** deterministic structural rejection covers credential values
  in reference-only or otherwise prohibited fields, credential-bearing URL user-info,
  known sensitive URL parameters and explicit secret-value structures. Best-effort
  scanning of free text, Business Context and documents is separate: safe warnings
  and Evidence, no proof of secret absence, no absolute confidentiality claim, and
  no structural invalidity solely because an optional scanner is unavailable.
- **SEC-003 / AC-09:** no write may escape, be redirected or reach a destination
  other than the explicitly authorized target. Cover traversal, symlink redirection,
  filesystem races/TOCTOU and approved boundaries. Concrete techniques remain for
  Plan/implementation. Safe read-only Repository path resolution authorizes no write.

## Approved slice behavior

These behaviors are covered by final Specification approval on 2026-09-11.
They do not reopen Q1–Q6 or establish additional global architectural decisions.
Original P identifiers are retained for decision traceability.

| ID | Approved slice behavior and rationale | Specification |
|---|---|---|
| P3 | Portable intent authoritative; local records bind stable keys and validated content revision. No implicit Workspace catalog/ownership. | Portable/local table |
| P4 | Init creates; identical rerun is no-op; changed intent conflicts. No update/force mode or automatic merge. | FR-012 |
| P5 | Reopen/install commands remain hypotheses; no automatic cloning. Portable bytes unchanged. Inclusion in slice confirmed by Q5. | J3–J4, FR-014 |
| P6 | Optional Provider/Capability declarations do not imply operational Integration or live verification. Requiredness settled by Q5. | FR-007 |
| P8 | Logical credential references with local source bindings; never retrieve values to validate references; no store selected. | FR-008 |
| P9 | Optional purpose text/documents, no ontology/chat import; absent/unconfigured context settled by Q5. | FR-009 |
| P11 | Guided interaction first; flags deferred; missing required input fails promptly. Preview, cancel/retry and persistence recovery explicit. | FR-010–013 |
| P12 | Portable validity, local installation and operational observations remain separate, not a Project lifecycle. Gaps settled by Q5. | Validity and failures |

Earlier P1/P2/P7/P10 defaults are superseded by Q1/Q5/Q4/Q2 respectively.

## Planning details and architectural boundary

No material blocking question remains from this review. Exact override name,
CLI syntax, initial schema version token/support rule and filesystem techniques
are later Plan details within the approved behavioral constraints, not new
clarification blockers. No additional schema, CLI framework, persistence engine,
credential store, Runtime integration or adapter is selected here.

No ADR is created or automatically accepted. Accepted ADR-0001–0003 remain
unchanged; no conflict was identified. Q1 does not settle aggregate ownership.
Future cross-cutting choices still follow the normal ADR process when justified.

## Coverage of requested questions

| User question | Reconciled answer |
|---|---|
| 1. Unique Project identity | FR-001; Q1 UUID v4 |
| 2–3. Required fields / unconfigured | Minimum table; Q5 minimal contract |
| 4–5. Portable/local relation / source of truth | Portable/local table; Q2–Q3; content revision binding |
| 6–7. Repository representation / paths | FR-002–004; Q6 conservative locator matching |
| 8–11. Existing Project / idempotency / files / update | FR-012–013: explicit target, no-op or conflict, create-only |
| 12. Another machine | J3, FR-014, AC-03; Q3, Q5 |
| 13–14. Schema version / incompatible input | FR-015; Q2 strict axiom.yaml |
| 15. Credentials | FR-008, SEC-001, AC-08; no store chosen |
| 16–17. Runtime selection / unavailable | FR-005; Q4 presence-only evidence |
| 18. Model Profile required? | No; FR-006, Q5 |
| 19–20. Optional Provider / unconfigured Integration | Minimum table, FR-007, Q5 |
| 21. Valid Project | Three validation dimensions; explicit local gaps; Q5 |
| 22. Minimum acceptance Evidence | AC-01–12 and future delivery Evidence paragraph |

## Review and documentation reconciliation

Reconciled the minimum table, journeys, FRs, security requirements and all
AC-01–AC-12 against human review. Updated specification index, README, roadmap
and CHANGELOG to remove stale open-question gates and artificial Provider
requirements. Conceptual model, Provider boundaries, accepted ADRs, harness,
commands and security policies need no change.

Final human decision on 2026-09-11: Specification 002 — Approved; Q1–Q6 —
Resolved by human review. This records the Constitution/SDD gate:
intake → specify → clarify → [human approval] → plan.
Next step: merge PR #3, then start Plan in a new change. Implementation is not
authorized by this PR and remains gated by an approved Plan. No Plan, Tasks,
implementation, adapters or dependencies are created.
Executable Lingo acceptance remains unverified because no Lingo code exists.

## Validation of this documentation change

Executed from repository root on 2026-09-11:

- `./scripts/validate-repository.sh .` — passed, including package validation,
  both validator regression suites and worktree sensitive-file scanning.
- `./scripts/check-sensitive-files.sh --staged .` — passed against staged content.
- `git diff --check` and `git diff --cached --check` — passed.
- Local Markdown link check — passed: 34 relative targets across the six changed
  documents exist.
- Identifier/reference check — passed: unique FR-001–016, SEC-001–005 and
  AC-01–12 definitions, with all referenced identifiers/ranges resolving.
- Q1–Q6 reconciliation and semantic review — passed: all six recorded as resolved;
  no remaining open-question blocker, artificial Provider requirement or conflict
  with Constitution, accepted ADRs, conceptual model or Provider boundaries found.
- PR diff scope — exactly six Markdown documents; no plan.md, tasks.md,
  implementation, dependencies, accepted ADR edits or sensitive files introduced.
  Idempotency, persistence safety, conflict detection, deterministic diagnostics,
  concurrency, version validation and existing-data preservation coverage retained.
- `command -v gitleaks` — unavailable (exit 1); dedicated scanner coverage remains
  unverified. No dependency installed. Local scans and content review found no secrets.

Existing harness checks validate documentation/harness hygiene only, not executable
AC-01–AC-12 behavior. The acceptance matrix retains future unit, integration and
functional coverage; no application tests or implementation are introduced here.
