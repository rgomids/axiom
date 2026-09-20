# Specification 004 — Usable MVP v1 Baseline

## Status and authority

**Draft — human decisions recorded; ready for final human approval.**

Tracked by [#62](https://github.com/rgomids/axiom/issues/62) under the
[MVP tracker #15](https://github.com/rgomids/axiom/issues/15). This artifact is
Specification work only. It does not authorize a Plan, Tasks, implementation,
release, migration, external Provider mutation, or acceptance claim.

The E2E Codex POC was explicitly accepted on 2026-09-20 and is historical
Evidence for this Specification. Its experimental commands, formats, storage
choices, and adapters become MVP contracts only where this Specification or an
existing accepted decision says so.

Normative terms `MUST`, `MUST NOT`, `SHOULD`, and `MAY` follow the
[Axiom Constitution](../../product/constitution.md).

## Source baseline

This draft reconciles:

- MVP outcome and delivery boundary in #15;
- guided Project setup in #54;
- Intent-driven Work Item creation in #55;
- Provider-visible workflow projection in #56;
- Runtime skill selectors in #57;
- completion output and detailed artifacts in #63;
- generated-text provenance in #64;
- persistence, recovery, compatibility, and migrations in #66;
- installation, upgrade, and onboarding in #67;
- clean-environment release-candidate acceptance in #68;
- accepted Specifications 002 and 003, their clarifications, Plans, and Evidence;
- ADR-0001 through ADR-0004, the conceptual model, Provider boundaries, product
  roadmap, and Constitution.

## Reconciliation findings

### Already decided

- POC acceptance is historical Evidence, not MVP implementation authority.
- Axiom/Lingo, Project/Repository, Runtime/domain, Provider/Transport,
  Integration/MCP, portable/local, authority, and human-acceptance boundaries are
  already accepted and are preserved below.
- The current supported experimental path is Codex + Lingo + GitHub Issues + one
  sequential workflow; broad multi-runtime/provider/orchestration support is not
  required for MVP.
- Project mutation, Provider mutation, optional Git effects, technical validation,
  and human acceptance are separate outcomes/authority gates.
- Unknown or invalid persisted state fails closed; it is never treated as absent
  or permission to overwrite.

### Human decisions recorded on 2026-09-20

Human review of PR #69 records the following directions:

- **HD-1:** use published checksummed binaries for macOS/Linux, retaining source
  installation for contributors. The authorized Plan/release contract MUST declare
  the exact supported OS/architecture matrix before distribution.
- **HD-2:** keep detailed artifacts durable and machine-local while required by
  workflow/Evidence, with explicit safe cleanup. They are not portable Project
  content. An ADR MUST define their final ownership/lifecycle and the relationship
  between Execution and any pre-Execution correlation identity.
- **HD-3:** use a bounded local filesystem threat model that preserves authorized
  target confinement, supported path/link protection, process concurrency,
  deterministic fault stages, fail-closed uncertainty, guided recovery, and
  protection against partial/invalid canonical state. Malicious same-UID arbitrary
  interleavings and physical power-loss/media-durability guarantees are excluded.
  Specification 002 plus affected security/architecture contracts MUST be
  reconciled before any Plan for Specification 004.
- **HD-4:** use clean v1 installation as the officially supported compatibility
  baseline. Historical POC state is detected and preserved with actionable
  backup/export/reconfigure behavior when applicable; no in-place POC migration
  commitment exists without a later explicit compatibility decision and Evidence.

### Apparent conflicts resolved without changing accepted decisions

- Specification 002 allows a portable-valid Project with no Provider; #54 asks
  setup to collect a Work Item Provider. This draft keeps Provider optional for
  portable validity but requires a supported Work Item capability before the MVP
  Work Item workflow can run.
- #56 requests Provider-visible stages while local state remains authoritative.
  This draft treats Provider metadata as an idempotent projection, not workflow
  ownership.
- #57 shows illustrative command-like skill arguments, while the accepted POC
  proves only the current Codex standalone skill contract. This draft specifies
  selector semantics and defers exact host syntax to validated planning/Evidence.
- #64 applies provenance to Axiom-authored text but forbids claiming transported
  user content. This draft uses one central provenance envelope and requires
  renderers to distinguish Axiom framing from unchanged user text.

No approved Specification or ADR is superseded merely by this draft. The
recorded HD-3 decision requires Specification 002 and the affected security and
architecture contracts to be reconciled before Plan authorization. HD-4 establishes
clean v1 as the compatibility baseline; any future in-place historical migration
requires a separate explicit compatibility decision before implementation.

## Intake

### Problem

The accepted POC proves one local value path, but another developer still needs
project-history knowledge to configure a Project, capture Intent, understand
workflow progress, interpret completion, diagnose failures, identify generated
content, recover state, and upgrade safely. Current behavior is useful Evidence,
not a stable v1 user contract.

### Desired outcome

A developer in a clean supported environment can install Axiom, discover Lingo
and the supported Runtime integration, intentionally configure a Project, create
or select a Work Item, run the bounded Axiom workflow, inspect Evidence, recover
truthfully from supported failures, upgrade through the supported path, and reach
a concise completion result ready for explicit human acceptance.

### Primary actors

- developer configuring and operating a local Axiom Project;
- reviewer deciding Specification, release-candidate, and final MVP acceptance;
- Runtime adapter invoking thin skills;
- Work Item Provider adapter projecting authorized workflow state.

### Constraints

- local-first operation;
- Codex is the only required Runtime for this MVP;
- GitHub Issues is the first supported Work Item Provider adapter, without
  making GitHub a domain requirement;
- macOS and Linux are the selected release OS boundary; the authorized
  Plan/release contract MUST declare the exact supported OS/architecture
  combinations before distribution;
- deterministic application/domain contracts remain usable without model
  judgment where interpretation or generation is not required;
- external mutation, migration, and destructive recovery require explicit
  authority appropriate to their effects.

## Non-goals

The following exclusions consolidate the existing MVP tracker and related-issue
boundaries. They constrain the MVP; they do not establish new behavior or relax
requirements inside the supported local contract:

- mandatory support for multiple external Providers;
- broad multi-runtime support beyond the required Codex path;
- multi-agent orchestration;
- automated Agent Planning;
- a cloud control plane;
- multi-tenancy;
- remote collaboration or synchronization;
- distributed persistence or remote locking;
- automatic updating;
- support for every package manager;
- Windows support for the MVP;
- signing, notarization, cryptographic provenance, attestation, or equivalent
  release trust mechanisms unless a future specific decision authorizes them;
  the informational provenance contract below remains required;
- production-grade durability beyond the explicitly supported local MVP contract;
- storing raw chat history as Evidence;
- making detailed artifacts part of portable Project intent;
- general backwards-compatibility guarantees across arbitrary historical
  versions.

## Preserved architectural boundaries

The MVP MUST preserve:

- `Axiom != Lingo`: Axiom owns product/domain/policies/contracts; Lingo executes
  or coordinates them locally;
- `Project != Repository`: one Project may associate multiple independent
  Repositories and no caller CWD becomes Project identity;
- `Role != Model`;
- `Execution != Agent`;
- `Provider != Transport`;
- `Integration != MCP`;
- `Skill != workflow source of truth`;
- `Evidence != raw chat history`;
- portable Project intent is distinct from machine-local state;
- `Runtime skill -> Lingo/application -> Axiom contracts`;
- Provider projection is reconcilable output; local validated workflow state
  remains authoritative for execution;
- user authority, green checks, technical review, merge, Provider closure, and
  human acceptance remain distinct states.

Runtime skills MUST remain thin entrypoints. They MUST NOT duplicate domain,
workflow, output, provenance, authority, persistence, or recovery rules.

## Supported MVP boundary

The MVP supports one local, sequential, single-agent delivery workflow through
Codex and Lingo, with GitHub Issues as the first Work Item Provider adapter. A
Project MAY remain portable-valid without a configured Work Item Provider, but
the supported Work Item journey MUST stop with an actionable missing-capability
result until a supported Provider is explicitly configured.

The supported workflow stages are:

```text
intake -> specification -> clarification -> plan -> tasks
-> implementation -> review -> evidence -> reconciliation -> completion
```

Clarification MAY be satisfied without a separate artifact when no material
question exists, but its gate result remains observable. Completion requires
all applicable technical gates plus explicit authority for any external
mutation. Final MVP acceptance remains a separate human decision.

## User journeys

### J1 — Clean installation and first run

From a clean supported environment, the developer follows published instructions,
installs a traceable Axiom build, discovers `lingo`, installs/verifies the Codex
integration, and receives first-run guidance into explicit Project setup. No
repository CWD, shell-profile mutation, existing maintainer state, or undocumented
credential setup is assumed.

### J2 — Intentional Project setup

The developer starts setup with complete arguments or a guided interview. Axiom
collects Project name/slug, Repository associations, machine-local working-copy
paths, and the Work Item Provider declaration needed by the intended workflow.
It shows portable intent separately from local bindings, validates a complete
proposal, and requires confirmation before publication.

### J3 — Intent-driven Work Item

The developer supplies an Intent. Axiom reuses supplied facts, asks only for
material missing information, and prepares a structured draft covering problem,
outcome, context, scope, constraints, non-goals, and acceptance expectations.
The complete draft is shown before any Provider mutation. Cancellation or denied
authority leaves Provider state unchanged.

### J4 — Visible workflow

The developer starts or resumes the bounded workflow. Local state records the
truthful current gate. With explicit Provider authority, the GitHub Issue shows
exactly one current Axiom stage marker and bounded transition comments referencing
relevant artifacts/Evidence. Failed or interrupted transitions do not project a
later stage as complete.

### J5 — Argument-driven Runtime invocation

A fully specified Runtime invocation supplies Project, Repository, and Work Item
selectors directly and completes without an avoidable question. A partial
invocation asks only for missing or ambiguous values. Unknown or conflicting
arguments produce validation failure rather than silent fallback.

### J6 — Concise completion and details

Every relevant CLI or Runtime operation ends with the same semantic completion
result. The primary summary is short and scannable. When diagnostics, context,
or Evidence exceed that summary, Axiom materializes a Markdown detail artifact
and returns a stable reference to it.

### J7 — Failure, recovery, and retry

The developer can distinguish invalid input, denied authority, interrupted or
partial work, retryable external failure, incompatible state, and local storage
failure. Axiom preserves the last known authoritative state, reports confirmed
external effects, and provides the next safe action. Ambiguous recovery never
claims rollback or deletes unknown content.

### J8 — Supported upgrade

The developer inspects the installed version, follows the documented supported
upgrade path, previews any state migration and backup effects, authorizes mutation,
and verifies the resulting schemas and Runtime integration. Unsupported downgrade,
newer schema, or missing migration path fails before destructive mutation.

### J9 — Release-candidate acceptance

A reviewer executes the complete journey in an isolated clean environment using
an identified release candidate. The run exercises CLI and Runtime entrypoints,
Provider projection, output/provenance, representative failure/recovery, supported
upgrade, Evidence, and documentation. Automated success prepares but never makes
the final human acceptance decision.

## Functional requirements

### Project setup

- **FR-001 Explicit selection:** Project setup MUST NOT infer that CWD is the
  Project or Repository. Logical selectors and explicit paths are inputs, never
  ambient identity.
- **FR-002 Guided completeness:** setup MUST ask only for missing required or
  materially ambiguous inputs and MUST support non-interactive complete input.
- **FR-003 Portable/local separation:** Project identity, Repository keys, and
  Provider declarations belong to portable intent; absolute working-copy paths,
  credentials, observations, and executable locations remain machine-local.
- **FR-004 Review before publication:** Axiom MUST show a safe normalized summary
  of portable and local changes and require exact mutation authority before write.
- **FR-005 Multi-repository:** the flow MUST accept multiple independent
  Repository associations without common paths, Providers, or submodules.
- **FR-006 Provider capability:** the intended Work Item workflow MUST identify a
  supported Provider/capability explicitly. Unsupported Providers MUST remain
  explicit and MUST NOT be silently mapped to GitHub.
- **FR-007 Reexecution:** equivalent setup is deterministic and idempotent;
  conflicting identity, slug, source, binding, or state is explicit.

### Work Item capture

- **FR-008 Intent first:** Work Item creation MUST begin from supplied Intent,
  not require a prewritten Provider title/body.
- **FR-009 Bounded interview:** questions MUST be limited to missing information
  that changes the problem, outcome, context, scope, constraints, non-goals, or
  acceptance expectations.
- **FR-010 Structured draft:** Axiom-authored drafts MUST preserve those sections
  in a Provider-appropriate representation suitable as Specification input.
- **FR-011 Mutation gate:** Provider create/update MUST occur only after the
  complete draft is shown and explicit external-mutation authority is granted.
- **FR-012 Neutral application boundary:** Provider-specific formatting and
  transport remain adapters; the application behavior does not depend on GitHub.

### Workflow visibility

- **FR-013 Local truth:** validated local workflow state is the execution source
  of truth; Provider state is an authorized, idempotent, reconcilable projection.
- **FR-014 Current-stage marker:** the supported GitHub adapter MUST expose exactly
  one canonical current Axiom stage marker. Naming is an adapter UX detail and
  MUST be documented and namespaced.
- **FR-015 Transition comment:** a material transition comment MUST contain stage,
  outcome, relevant references, provenance, and next action when applicable.
- **FR-016 Truthful failure:** failed, denied, interrupted, or partial transitions
  MUST NOT present a later stage or external completion as successful.
- **FR-017 Reconciliation:** replay/reconciliation MUST avoid duplicate markers
  and semantically duplicate comments while preserving confirmed Provider effects.

### Runtime invocation

- **FR-018 Known inputs:** supported Runtime skills MUST accept available Project,
  Repository, Work Item, and operation-specific inputs directly under the validated
  host invocation contract.
- **FR-019 Missing-only questions:** supplied valid values MUST be reused; only
  missing or ambiguous values may trigger questions.
- **FR-020 Strict arguments:** unknown, duplicated, or conflicting inputs MUST fail
  clearly. They MUST NOT be ignored or converted into ambient-CWD inference.
- **FR-021 Thin delegation:** parsing/adaptation MAY be Runtime-specific, but every
  validated selector and action MUST reach the same Lingo/application contract as
  the direct CLI path.

## Completion output contract

Every relevant completed CLI command and Runtime skill invocation MUST expose one
canonical semantic result. Human text, Runtime rendering, and structured JSON MAY
vary in syntax, but MUST NOT vary in meaning.

### Required result fields

| Field | Contract |
|---|---|
| `status` | One canonical terminal status from the set below. |
| `result` | Short primary outcome, bounded to one scannable statement. |
| `references` | Relevant Project, Repository, Work Item, Execution, local operation-correlation, Provider, or artifact identifiers/links. Omit only when none exists. |
| `next` | Next safe action when further action is applicable. |
| `details` | Stable reference to a Markdown detail artifact when one exists. |
| `provenance` | Canonical Axiom product/version/build identity. |

Human output SHOULD fit in a compact block and MUST NOT emit large diagnostics,
raw logs, raw model reasoning, or full Evidence by default. Structured output MUST
use stable fields/categories rather than require prose parsing.

### Canonical terminal statuses

| Status | Meaning |
|---|---|
| `success` | Requested operation completed with its promised effects confirmed. |
| `failure` | Non-validation, non-retryable operation failure; no later success is implied. |
| `validation_failure` | Input, state, schema, compatibility, or invariant failed before the requested valid transition completed. |
| `denied_authority` | Required authority was absent, refused, stale, or did not cover the exact effects. |
| `partial` | Some effects are confirmed and others did not complete; confirmed effects and remaining work are explicit. |
| `interrupted` | Execution stopped before a terminal operation result; persisted/externally confirmed state is reported without inferred rollback. |
| `retryable_failure` | A transient external or environmental dependency failed and a bounded retry is safe under the stated preconditions. |

An operation MUST NOT classify an unknown or ambiguous side effect as `success`.
`retryable_failure` MUST name the retry boundary and MUST NOT authorize duplicate
external mutation. Application contracts own the classification; skills and
Provider adapters only render it.

## Detailed Markdown artifact contract

A detail artifact MUST be created when information necessary for diagnosis,
review, Evidence, recovery, or continuation would exceed the concise summary;
when a retained failure matrix or validation report is required; or when a
workflow/release gate requires durable Evidence. The summary alone is sufficient
for routine success, expected no-op, simple validation failure, or denied authority
when all actionable facts fit in the primary contract.

Artifacts MUST:

- be Markdown, bounded in size, sanitized, and free of secrets/raw chat history;
- have stable identity independent of a display filename;
- record the applicable Execution reference or local operation-correlation
  identity, creation time, provenance, source references, outcome, and retention
  class;
- be referenced by the completion result and, when it supports a claim, by the
  corresponding Evidence record;
- distinguish diagnostic context from Evidence claims;
- remain machine-local unless an explicit later publication action is separately
  authorized;
- never enter the Portable Project Manifest or portable Project working copy by
  implementation convenience.

HD-2 records machine-local durable ownership as the MVP direction. Before
implementation, an ADR MUST define the final storage root, relationship to
Execution/pre-Execution correlation, retention, Evidence relation, and explicit
safe cleanup semantics.

## Provenance contract

Axiom-authored textual content on every supported MVP surface MUST carry one
canonical informational provenance model:

| Field | Required behavior |
|---|---|
| product | Exact identity `Axiom`. |
| version | Release semantic version when built as a release; otherwise explicit `development`/unreleased identity. Never invent a release version. |
| revision | Short source revision when embedded or truthfully observable; otherwise explicit `unavailable`. |
| source state | `clean`, `dirty`, or `unknown`. Dirty MUST NOT be presented as the clean revision alone. |

Presentation MAY be a compact footer, header, subtitle, comment, or valid metadata,
depending on surface. CLI/Runtime summaries, Provider Issue/PR bodies and comments,
generated Markdown, and generated source/text files MUST expose equivalent facts
without breaking syntax or overwhelming the summary.

Provenance MUST be produced centrally and supplied to renderers/adapters. A skill,
template, or Provider adapter MUST NOT hardcode an independent version rule.
Transported user text remains user-authored. If Axiom wraps or transforms it, the
marker MUST distinguish Axiom-authored framing from unchanged user content and
MUST NOT claim authorship over the transported content.

## Persistence, recovery, compatibility, and migration

- **FR-022 Authoritative state:** publication MUST expose either the prior complete
  valid state or the new complete valid state, never a partial state as canonical.
- **FR-023 Commit truth:** pre-commit failure preserves prior authority; a confirmed
  post-commit effect is reported as committed even if later sync, local persistence,
  Provider projection, or cleanup fails.
- **FR-024 Filesystem confinement:** writes MUST remain within the exact authorized
  target, with traversal, symlink, hard-link, collision, and replacement protections
  appropriate to the approved threat model.
- **FR-025 Recovery:** recognized interrupted artifacts MUST fail closed with
  deterministic classification and guided recovery. Unknown artifacts MUST be
  preserved. Ambiguous state MUST require operator review and MUST NOT be repaired
  or deleted automatically.
- **FR-026 Schema compatibility:** every portable and local persisted format MUST
  have an explicit version and reader compatibility rule. Missing, malformed,
  unsupported-newer, and unsupported-older versions are distinct failures. State
  safely identified as belonging to the historical POC MUST NOT be treated as
  absent or overwritten; HD-4 requires preservation plus actionable
  backup/export/reconfigure handling, without an automatic in-place migration
  commitment.
- **FR-027 Migration preview:** any mutating migration MUST validate source state,
  show source/target versions and affected roots, identify backup/rollback behavior,
  require exact authority, and revalidate the result.
- **FR-028 Backup and rollback:** migration MUST preserve a recoverable pre-migration
  copy before canonical replacement when rollback cannot otherwise be guaranteed.
  Backup identity, permissions, retention, and cleanup MUST be explicit.
- **FR-029 No silent downgrade:** downgrade or lossy conversion is refused unless a
  separately specified reversible path exists. Newer unknown state is never
  overwritten as if absent.
- **FR-030 Portable/local boundary:** migration MUST NOT move secrets, absolute
  paths, local observations, workflow state, or Execution artifacts into portable
  Project intent.

The residual POC gaps are not silently waived: physical power-loss durability,
remaining fault stages, ACL mutation, arbitrary hostile same-UID interleavings,
automatic recovery, and compatibility or migration Evidence must be mapped to the
approved threat model and test matrix. HD-3 fixes the bounded threat boundary;
HD-4 fixes clean v1 as the compatibility baseline.

## Installation, onboarding, and upgrade

- **FR-031 Supported path:** one published, documented install path MUST work from
  a clean supported environment and yield an inspectable version/provenance result.
- **FR-032 Prerequisites:** missing tools, incompatible platform, invalid destination,
  and unavailable Runtime integration MUST fail with actionable next steps.
- **FR-033 Ownership:** installed binary, receipt, Runtime skills, portable Project
  data, local state, Evidence, and detail artifacts MUST be documented as separate
  ownership/lifecycle categories.
- **FR-034 Safe repetition:** reinstall and upgrade MUST be idempotent for equivalent
  owned content and refuse unowned/conflicting replacement.
- **FR-035 First run:** first-run guidance MUST lead to explicit Project setup and
  MUST NOT configure the CWD implicitly.
- **FR-036 Runtime compatibility:** Lingo and installed Runtime skills MUST expose
  a detectable compatibility result. An incompatible known skill set is upgraded
  only through the authorized install/upgrade flow.
- **FR-037 Upgrade ordering:** binary/Runtime integration and persisted-state
  compatibility checks MUST be ordered so a failed upgrade does not silently leave
  an unusable mixed state. Partial confirmed effects use the completion contract.

HD-1 selects checksummed macOS/Linux binaries plus source installation for
contributors. The exact supported OS/architecture combinations remain a required
Plan/release declaration before distribution.

## Failure cases and invariants

The MVP MUST handle at least:

- absent, invalid, ambiguous, or conflicting Project/Repository/Work Item selector;
- unsupported Provider or missing Work Item capability;
- incomplete interview, cancellation, and denied mutation authority;
- invalid/duplicate Runtime arguments;
- Provider unavailable, unauthenticated, rate-limited, invalid response, or
  confirmed mutation followed by local failure;
- failed, interrupted, resumed, stale, or already-completed workflow transition;
- completion summary rendering failure after confirmed domain/external effect;
- detail-artifact write failure without hiding the primary operation truth;
- released, development, dirty, and revision-unavailable provenance;
- missing/corrupt/newer/older persisted schema;
- interrupted publication or migration, insufficient space, permission denial,
  collision, unsafe links, concurrent writer, and recovery-required state;
- incompatible binary/Runtime skill versions and partial upgrade.

Global invariants:

1. CWD never supplies Project or Repository identity implicitly.
2. Portable intent never contains secrets or machine-local state.
3. A valid Project does not imply operational workflow readiness.
4. No external mutation occurs without exact authority and review where required.
5. CLI and Runtime skills share application semantics, terminal statuses, and
   provenance; presentation differences do not create parallel contracts.
6. Provider projection cannot advance local workflow truth.
7. Confirmed effects survive truthful reporting; failure never invents rollback.
8. Evidence is inspectable support for a claim, not raw chat history.
9. Axiom provenance never claims authorship over unchanged user content.
10. Automated validation never grants final human acceptance.

## Non-functional requirements

- output and diagnostics MUST be deterministic for equivalent input/state, except
  for explicitly identified time/identity fields;
- secrets and rejected sensitive values MUST not enter output, logs, artifacts,
  Evidence, Provider comments, or migration backups;
- user-facing summaries MUST remain bounded and scannable;
- file reads/writes and captured external output MUST be bounded;
- local state and generated artifacts MUST use restrictive platform-appropriate
  permissions and reject unsafe ownership/link conditions;
- supported operations MUST be reproducible without maintainer-local state;
- observability MUST correlate one operation across summary, artifact, Evidence,
  local state, and Provider projection without exposing credentials; under HD-2,
  the required ADR decides whether that identity is subordinate to an Execution or
  is only local operational correlation before an Execution exists.

## Observable acceptance criteria

| ID | Observable acceptance |
|---|---|
| AC-01 | Clean installation yields discoverable `lingo`, truthful provenance, and verified Codex integration using only published instructions. |
| AC-02 | First run enters explicit Project setup without treating CWD as Project/Repository. |
| AC-03 | Guided and fully argument-driven setup produce equivalent validated Project/application behavior. |
| AC-04 | Multi-repository portable intent and local absolute bindings remain separate; review precedes publication. |
| AC-05 | An unsupported/missing Work Item Provider capability blocks the workflow explicitly without silently selecting GitHub. |
| AC-06 | Intent-only Work Item creation asks bounded missing questions, presents a structured draft, and mutates GitHub only after authority. |
| AC-07 | Supplied Work Item fields are reused; cancellation and denied authority leave Provider state unchanged. |
| AC-08 | Provider state shows exactly one current workflow-stage marker and idempotent bounded transition comments. |
| AC-09 | Failed/interrupted/resumed workflows remain truthful locally and on the Provider; no later stage is projected early. |
| AC-10 | Fully specified skill invocation needs no selector round trip; partial invocation asks only for missing/ambiguous values. |
| AC-11 | Invalid/conflicting skill arguments fail predictably and never fall back to CWD. |
| AC-12 | Representative CLI and Runtime operations produce semantically equivalent `success`, `failure`, `validation_failure`, `denied_authority`, `partial`, `interrupted`, and `retryable_failure` results. |
| AC-13 | Routine outcomes use only the short summary; oversized diagnosis/Evidence creates one referenced Markdown artifact under the approved ownership/lifecycle contract. |
| AC-14 | Detail artifacts are sanitized, bounded, identity-addressable, correlated, and never silently portable. |
| AC-15 | CLI, Runtime, Provider comments, Markdown, and generated source/text expose equivalent truthful provenance for release, development, dirty, and unavailable-revision builds. |
| AC-16 | Unchanged transported user content is not labeled as Axiom-authored. |
| AC-17 | Supported publication faults preserve old/new complete authority, classify recovery deterministically, and preserve unknown artifacts. |
| AC-18 | Unsupported schema/version combinations fail before destructive mutation; the approved compatibility path detects historical POC state without overwrite and provides its documented clean-install, export/reconfigure, backup, recovery, and, only if separately approved, migration behavior. |
| AC-19 | Reinstall/upgrade is safe and idempotent for owned equivalent content and refuses unowned conflicts. |
| AC-20 | A representative confirmed Provider effect followed by local failure reports `partial` with the confirmed reference and a safe next action. |
| AC-21 | Clean-environment RC dogfood completes install -> first run -> Project -> Work Item -> workflow -> Evidence -> completion through Runtime and direct CLI entrypoints. |
| AC-22 | RC dogfood exercises validation, authority denial, interruption/resume, retryable external failure, recovery-required state, and supported upgrade. |
| AC-23 | Versioned Evidence identifies candidate build, environment, commands, exits, hashes/references, exclusions, and known limitations without raw chat history or secrets. |
| AC-24 | Human explicitly accepts or rejects the Specification and, later, the release candidate; automation never fills either decision. |

## Required acceptance Evidence

Release-candidate Evidence MUST include:

- exact Axiom version, source revision/state, platform, architecture, Runtime and
  Provider adapter versions;
- clean-environment construction and prerequisite inventory;
- installation and upgrade commands with exit statuses and installed checksums;
- CLI/Runtime semantic-equivalence matrix for every terminal status;
- Project portable/local separation and multi-repository observations;
- Work Item draft/authority and Provider projection references;
- workflow gate transitions, interruption/resume, artifact identities/digests,
  and final completion reference;
- provenance observations on each supported generated surface;
- filesystem fault/recovery and compatibility-transition matrix tied to the
  approved threat model, including migration Evidence only when migration is part
  of the approved path;
- secret/sensitive-file and security review results;
- relevant unit, integration, functional black-box, race, vet, build, module,
  repository, and documentation validation results;
- known limitations, unexecuted checks, and explicit human acceptance/rejection.

At least one acceptance run MUST use an isolated environment without existing
Axiom state. Controlled fakes MAY prove deterministic external failures, but the
supported Provider/Runtime journey requires a bounded real integration observation
before final acceptance. Every external mutation remains explicitly authorized.

## Delivery dependencies

```text
#62 Specification approval
  |
  +--> #54 Project setup
  +--> #55 Work Item intent
  +--> #56 Provider projection
  +--> #57 Runtime arguments
  +--> #63 output + detail artifacts
  +--> #64 provenance
  +--> #66 persistence + migration
  +--> #67 install + upgrade + onboarding
         |
         +--> #68 clean RC dogfood + docs + human acceptance
```

If authorized after the human decisions and required reconciliations, a future
Plan may reorder independently implementable units but MUST preserve
contract dependencies: provenance and completion semantics precede surface-wide
adoption; artifact ownership precedes persistence; migration policy precedes
upgrade; all delivered blocks precede #68. Approval of this Specification permits
the next expressly authorized phase only and does not itself authorize Plan work,
any delivery issue, implementation, or release.

## Requirements, details, and open questions

| Classification | This Specification |
|---|---|
| Requirement | Observable journeys, FR/AC contracts, failure states, invariants, Evidence, and preserved boundaries above. |
| Implementation detail deferred to Plan | CLI framework, concrete Go packages/interfaces, exact JSON schema, Provider label spelling, prompt UI, filesystem syscalls, migration algorithm, installer implementation, artifact filename rendering. |
| Human decisions recorded | HD-1 through HD-4 are recorded below; required ADR/reconciliation work remains gated, and final Specification approval is still pending. |

## Human decisions recorded — 2026-09-20

These decisions were explicitly recorded during human review of PR #69. They
resolve the material directions needed by this Specification but do not by
themselves approve the complete Specification or authorize Plan, Tasks,
implementation, accepted ADRs, or release work.

### HD-1 — Supported release installation and platforms

**Decision:** use published checksummed binaries for macOS/Linux, while retaining
source installation for contributors.

The authorized Plan and release contract MUST explicitly declare the supported
OS/architecture combinations before distribution. Candidate combinations include
`darwin/arm64`, `darwin/amd64`, `linux/amd64`, and `linux/arm64`; this
decision does not silently promise every candidate. Any combination not explicitly
declared is unsupported for that release.

Package-manager distribution, automatic updating, Windows support, signing, and
notarization remain outside the MVP unless separately decided.

### HD-2 — Detailed artifact ownership and lifecycle

**Decision:** detail artifacts are durable, machine-local artifacts retained while
needed by workflow/Evidence and removed only through explicit safe cleanup. They
MUST NOT enter the Portable Project Manifest or portable Project working copy.

This decision intentionally does not create an `operation-attempt` entity parallel
to Execution. Before implementation, an ADR MUST define final ownership/location,
Evidence relation, retention and cleanup, and decide whether attempt identity is
subordinate to Execution or whether pre-Execution commands use only non-domain
local correlation identity.

### HD-3 — MVP filesystem threat model

**Decision:** adopt the bounded local threat model.

The supported boundary retains:

- confinement to the exact authorized target;
- traversal and supported symlink/link protections;
- process concurrency protection;
- deterministic injected fault stages;
- fail-closed uncertainty;
- guided recovery;
- protection against partial or invalid canonical state.

The MVP explicitly excludes guarantees against malicious same-UID actors performing
arbitrary interleavings and against physical power loss/media durability. These are
unsupported guarantees, not solved risks.

Because this narrows previously approved proof expectations, Specification 002 and
all affected security/architecture contracts MUST be reconciled before any Plan for
Specification 004 is authorized. The reconciliation/ADR MUST state supported and
excluded threats, still-mandatory properties, required Evidence, and the exact
relationship to Specification 002 SEC-003/SEC-005.

### HD-4 — Compatibility baseline and historical POC state

**Decision:** clean v1 installation is the officially supported compatibility
baseline.

`v0.1.0-poc.1` remains a historical snapshot without an automatic compatibility
or migration commitment. When POC-owned state is recognized, v1 MUST NOT treat it
as absent or overwrite it. The supported path provides actionable diagnostics and
preserves backup/export where applicable, followed by explicit export/reconfigure.

In-place migration from the POC remains unsupported unless a later bounded
compatibility/ADR decision proves it safe and explicitly authorizes it. Unknown
newer formats fail closed; unsupported older formats are diagnosed explicitly; no
migration or downgrade is silent; no generic N-1 or arbitrary historical
compatibility is promised.

## ADR candidates

- **Required by HD-2 before implementation:** detail-artifact ownership and
  relationship to Execution, pre-Execution correlation identity, local location
  boundary, Evidence relation, retention, and cleanup.
- **Required by HD-3 before Plan authorization:** local persistence threat model and
  durability/recovery guarantees, with explicit reconciliation to Specification
  002 SEC-003/SEC-005 and affected security/architecture contracts.
- **Required before any in-place POC migration:** specific persisted-schema
  compatibility and migration/rollback decision; clean install plus
  export/reconfigure does not itself accept such migration.
- **Assess during an authorized Plan under HD-1:** distribution/supply-chain ADR
  if published binaries establish a durable release topology or trust boundary. A
  routine source installer alone does not justify an ADR.

No ADR is accepted by this draft.

## Constitution check

- Intent, actors, constraints, non-goals, failures, acceptance, and Evidence
  precede implementation.
- POC facts remain Evidence; experimental implementation does not silently define
  MVP contracts.
- Material architecture, security, compatibility, and ownership choices are
  recorded as explicit human decisions; required reconciliation and ADR gates
  remain enforceable.
- Deterministic classifications and stable references take precedence over prose
  or model judgment.
- Least privilege, exact mutation authority, portable/local separation, and
  honest partial outcomes are normative.
- Project remains distinct from Repository and runtime/provider products remain
  outside the domain core.
- Final acceptance remains human and separate from validation or merge.

## Review gate

HD-1 through HD-4 are recorded. This draft is ready for final human review of
the complete Specification. HD-3 requires Specification 002 and affected
security/architecture reconciliation before any Plan authorization; HD-2 requires
its artifact-lifecycle ADR before implementation. Final Specification approval
authorizes only the next phase explicitly recorded by the human decision and does
not implicitly authorize Plan, Tasks, implementation, accepted ADRs, or release.

**Next artifact: Final human review / approval of MVP Specification**
