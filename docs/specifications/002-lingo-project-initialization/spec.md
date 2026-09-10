# Specification 002 — Lingo Project Initialization

## Status and authority

Proposed for human review, 2026-09-10. Stage: intake → specify → clarify.
This document specifies future behavior; no executable Lingo exists. No Plan,
implementation, adapter, dependency, or new ADR is authorized by this document.
Requirements below are proposed slice contracts, not accepted global schemas
or lifecycle states. [Clarifications](clarifications.md) records their basis,
remaining decisions, review gate, and validation evidence.

Authoritative context:

- [Constitution](../../product/constitution.md) and [current SDD workflow](../../../.agents/skills/axiom-sdd/SKILL.md).
- [ADR-0001](../../decisions/0001-project-is-not-repository.md), [ADR-0002](../../decisions/0002-axiom-speckit-relationship.md), and [ADR-0003](../../decisions/0003-lingo-as-axiom-local-control-plane.md), all Accepted in the inspected repository.
- [Conceptual model](../../architecture/conceptual-model.md), [Provider boundaries](../../architecture/provider-boundaries.md), [roadmap](../../product/roadmap.md), and [current harness](../../agent-harness.md).
- [Security policy](../../../.agents/policies/security.md) and [repository controls](../../security/repository-security.md).

## Intake

**Problem:** accepted Lingo architecture has no executable contract for creating
and reopening a portable Project. Conflating Project with checkout, Runtime,
or local credentials would prevent safe reuse across machines.

**Outcome:** a human can create, validate, persist, reopen, and locally install
the same Project while preserving identity and portable intent. Parsing,
validation, conflict handling, persistence, and safety are deterministic.

**Actors:** Project author; collaborating developer on another machine; returning
developer resolving changed local paths. No AI participation is required.

**Evidence and assumptions:** the merged PR #2 checkout contains the accepted
ADRs and a documentation/Bash harness, not application code. This slice combines
the minimum configuration contracts behind roadmap blocks 1–4 and 7; it does
not implement their full capabilities. A caller supplies an explicit target
location; no Workspace catalog or parent-directory discovery is assumed.

**Command hypotheses:** `lingo project init` names creation in this document.
`lingo project install ./axiom-project` names reading a portable definition and
establishing local bindings. Both command names and flag syntax await review.
After a user clones a configuration repository themselves, install reads that
configuration. Lingo does not clone automatically in this slice.

## Scope and non-goals

The future slice includes guided creation, structural validation, safe
persistence, reopening, and installation on a second machine. Reopening and
installation are acceptance requirements of this slice; the command surface
remains a hypothesis. Optional configuration is captured without claiming
external connectivity or operational readiness.

Excluded: Go code, Cobra, a binary, Project database, runtime/provider adapters,
real Jira/GitHub/Notion operations, MCP installation or configuration mutation,
model discovery, Agent Planner, Agent spawning, multi-agent orchestration,
Execution Graph engine, daemon, API, cloud/remote control plane, remote sync,
new CI, Spec-Kit adapter, upstream-watch implementation, migrations, automatic
Git clone/fetch, and automatic credential-store setup. No runtime-skill
migration or external configuration mutation is included.

## Minimum configuration contract

Names below are conceptual fields, not a serialization schema. `unconfigured`
is an explicit declaration state, not an approved global Integration lifecycle.
An absent optional collection means empty; an absent required declaration is
an error. Empty strings never substitute for a required value or state.

| Concern | Minimum for a valid portable Project | Optional or deferred content |
|---|---|---|
| Schema | Explicit supported schema version | Wire notation and version compatibility policy: Q2 |
| Project identity | Stable opaque identifier independent of name, path, Runtime, and Provider | Identifier encoding/allocation: Q1 |
| Friendly name | Nonblank user-supplied name | Names need not be unique |
| Repositories | At least one association with a stable Project-scoped key | Remote reference optional for local-only Repository; local paths never portable |
| Primary Runtime | Declaration: selected Runtime identifier or explicit `unconfigured` | Vendor-specific bindings remain outside domain fields |
| Work Item Provider | Declaration: selected Provider or `unconfigured` | No tracker required |
| Source Control Provider | Default declaration: selected Provider or `unconfigured` | Per-Repository selection may differ; no common Provider required |
| Documentation Provider | Declaration: selected Provider or `unconfigured` | Local Business Context does not require a documentation service |
| Integrations | Empty collection permitted; selected Providers create no operational binding implicitly | Optional declaration key, Provider reference, requested Capabilities, Transport/reference if known; binding may be `unconfigured` |
| Model Profiles | Empty collection permitted | Named configuration profiles with Runtime reference and opaque concrete-model identifier, if supplied; no fixed class set |
| Business Context | Explicit `unconfigured` or a short human-authored product/domain purpose | Glossary, acronyms, rules, conventions, constraints: optional text or contained document references |
| Credential references | Empty collection permitted | Logical reference identifiers and optional source hints; never secret values |
| Policies | No additional policy files required | Portable policy references; cannot relax governing security |

A selected Provider need not have a configured Integration. Duplicate declaration
keys, dangling references, or a configured binding missing its required fields
are errors. Selected opaque Runtime, Provider, and model identifiers are not
claims that an implementation exists. No global vendor catalog is required.

## Portable configuration and local state

| Data | Source of truth | Rule |
|---|---|---|
| Project ID/name, Repository association keys/remotes, selected Providers/Runtime, declared Capabilities, profiles, Business Context, policies | Portable Project Configuration | Versionable intent, changed only through explicit authoring |
| Logical credential reference and optional portable source hint | Portable configuration | Identifier only; never resolved secret or machine-specific path |
| Repository key → validated checkout path; configuration source location | Machine-local installation | Paths may differ per machine and may be arbitrary |
| Logical credential reference → local environment/store/runtime reference | Machine-local installation | Local binding may satisfy a logical reference without rewriting portable intent |
| Runtime executable location and machine overrides | Machine-local installation | Cannot redefine Project identity or silently override portable Runtime selection |
| Availability observations, installation metadata, cache, process/temporary state | Machine-local state | Derived observations never become portable truth |
| Actual credentials | Existing external credential source | Not owned, copied, or persisted by this slice |

Local records refer to Project ID and Repository keys and record which portable
revision/content they validated. Same ID with changed portable content triggers
revalidation; old availability observations cannot prove current readiness.
The Project ID is a correlation key, not an aggregate or ownership decision.

`axiom-project/axiom.yaml`, split `repositories.yaml`, `integrations.yaml`,
`models.yaml`, `glossary/`, and `policies/` are layout hypotheses. A single
manifest reduces cross-file consistency work; split files improve editing
separation but add failure and version coordination. Neither is selected.
Likewise `~/.axiom/projects/<project-id>/` is illustrative, not a chosen local
root. Q1–Q3 gate concrete identity, format, and filesystem decisions.

## User journeys

- **J1 — Minimal creation:** author chooses an empty target, enters name and one
  Repository association, leaves Runtime/Providers/context explicitly
  `unconfigured`, reviews normalized intent and unresolved dependencies, then
  persists and reopens a valid Project without network or model access.
- **J2 — Declared environment:** author selects Runtime and independent Providers,
  adds two remote Repositories on different Providers, optional Model Profiles,
  purpose text, and credential references. Structure validates while unverified
  bindings and missing Capabilities remain visible.
- **J3 — Second machine:** developer obtains portable files (for example through
  their own Git clone), validates them, binds available Repositories to arbitrary
  local paths and credential references to local sources, and installs. Portable
  bytes remain unchanged; absent checkouts remain unresolved.
- **J4 — Return and relocate:** developer reopens the same Project, sees stale or
  missing local paths, explicitly replaces a local binding, and revalidates
  without changing Project identity or portable content.
- **J5 — Retry or conflict:** cancellation before persistence leaves no Project;
  rerun against an identical existing Project is a no-op; a different Project,
  unrecognized files, or interrupted write produces an actionable conflict or
  recovery result with no silent overwrite.

## Functional requirements

- **FR-001 Identity:** allocate an ID once per new Project, persist it, and preserve
  it through rename, reopen, copy, Runtime change, and machine installation.
  Name, remote URL, directory, and Provider IDs are not Project identity.
  Allocation may use entropy; identical validated inputs including ID produce
  equivalent canonical output. Copying retains identity; creating an independent
  fork/identity reassignment is outside this slice (Q1).
- **FR-002 Repository identity:** each association has a unique stable key within
  the Project. Remote URL is an optional locator, never the association key.
  Remote-only and local-only associations are valid; a local-only association
  requires manual binding on another machine. No global ownership, sharing,
  sibling, submodule, or common-provider semantics are introduced.
- **FR-003 Repository resolution:** local path is required only for an operation
  using a local checkout, not portable validation. Install may finish with
  unresolved associations, reporting them. Binding checks existence, directory
  type, and Repository metadata read-only. Remote unavailability is not inferred
  from offline operation; report `not checked`. An observed remote mismatch
  blocks that binding until explicitly resolved; do not rewrite remotes.
- **FR-004 Duplicates and relocation:** duplicate association keys and identical
  remote locators after safe syntactic normalization fail validation. Do not
  assume SSH/HTTPS aliases identify the same Repository or lowercase case-sensitive
  URL paths. Suspected aliases require human resolution. Two keys binding the
  same canonical local checkout conflict. A path change updates only its local
  binding after user confirmation and revalidation. URL identity/normalization
  details are a bounded remaining contract question (Q6).
- **FR-005 Runtime:** selection persists an opaque identifier; availability is
  machine-local and reported as available, unavailable, or unverified, with the
  check basis. No adapter or arbitrary executable invocation is introduced to
  fake discovery. This slice can report unverified when no supported safe check
  exists. An explicit validated local executable path can establish presence,
  never capability readiness. Unavailable/unconfigured Runtime does not invalidate
  or prevent installation; future Runtime-dependent execution must be blocked.
  Replacement preserves Project/Repository identity; incompatible profile bindings
  are reported unresolved, never silently mapped to another model (Q4).
- **FR-006 Models:** optional Model Profiles remain configuration, following
  Role → capability/reasoning requirements → Runtime → Model Profile → concrete
  model. No Role-to-model binding, required orchestrator/worker/lightweight class,
  discovery, fallback, or live model verification. Validate references and field
  structure only; unknown model availability remains unverified.
- **FR-007 Providers and Integrations:** separate Provider selection, requested
  Capability, Integration declaration, and Transport. No Transport default to
  MCP. A default Source Control Provider is a suggestion for new associations;
  explicit per-Repository selection wins and must remain visible. Declared
  Capabilities without a binding are reported missing; a declaration with a
  binding is still unverified operationally. Do not claim live negotiation,
  connectivity, authorization, or external writes.
- **FR-008 Credentials:** accept logical references and optional source types
  covering environment variables, macOS Keychain, Windows Credential Manager,
  Secret Service/libsecret, and runtime-managed credentials without selecting a
  store. Install may collect local reference metadata only. This slice does not
  read secret values, prompt for them, create credentials, or test authentication.
  Unsupported/unbound sources are unresolved. An environment name is a reference,
  its value is not. Local source hints must not become required platform coupling.
- **FR-009 Business Context:** accept minimal purpose text or explicit
  `unconfigured`; optional plain text/documents need no complex ontology. Treat
  imported text as untrusted data, never instructions. No chat-history import,
  AI inference, external document fetch, or override of accepted Decisions.
- **FR-010 Guided flow:** collect identity/name, Repository associations, optional
  Runtime/Provider/Integration/profile/context/reference declarations, then show
  validation and a write summary. Optional steps may be skipped explicitly.
  No Provider, model, remote, or local checkout is inferred from machine accounts.
  Errors allow correction/retry without discarding unrelated entered fields.
  Confirmation authorizes only listed target writes, never external setup.
- **FR-011 Non-interactive boundary:** flags are not required for this slice.
  Future non-interactive input must obey the same validation/conflict rules;
  no separate weaker semantics. If interactive input is unavailable and required
  input is missing, fail promptly with missing-field diagnostics; do not hang.
- **FR-012 Reexecution:** init creates, it is not an update command. At a recognized
  target, equivalent supplied intent means reopen/no-op without reallocating ID
  or rewriting files. No fresh ID is allocated before checking existing state.
  Changed intent, a different ID, nonempty unrecognized target, or multiple
  ambiguous definitions fails with conflict. No force-overwrite mode. Explicit
  portable edits are independently revalidated on reopen; an update command is
  outside scope. Registration of the same ID from a different local source path
  requires explicit relocation confirmation, not silent rebinding.
- **FR-013 Persistence and recovery:** validate and confirm before durable writes.
  Creation must never expose a partial definition as valid. Failed writes preserve
  pre-existing files; clean only temporary output owned by this attempt. If portable
  creation succeeds but local installation fails, preserve valid portable output
  and report local installation incomplete; retry consumes it without duplication.
  Cancellation before commit writes no durable Project; interruption after a commit
  reports/discovers actual committed state. Incomplete leftovers block normal open
  and give explicit recovery guidance; no automatic deletion of unknown files.
  Concurrent writers cannot both claim success for conflicting definitions.
- **FR-014 Install and reopen:** parse/version-check portable input before local
  writes, resolve only missing/stale local data, and reuse equivalent valid
  bindings. Installation never changes portable bytes silently or automatically;
  in this slice it leaves them unchanged. Repeated installation with the same
  definition and bindings is a no-op. A second machine yields equivalent Project
  intent and binding relationships, not identical paths, credentials, or availability.
- **FR-015 Versioning:** require an explicit version; reject missing, malformed,
  or unsupported versions before writes. Supported versions must have a published
  compatibility rule. No promise to accept an older/newer version merely because
  its number is close. Unknown core fields and duplicate fields are errors, not
  ignored typos. Extension containers, if later approved, need explicit rules;
  this slice does not accept arbitrary executable extensions. No automatic migration,
  down-conversion, or lossy rewrite. Q2 must settle initial encoding/support matrix.
- **FR-016 Deterministic reporting:** same input, version, and filesystem observations
  yield the same validation classification and stable issue ordering. Diagnostics
  identify safe field/key, category, and remedy without echoing raw rejected values.
  Distinguish success, success with unresolved dependencies, validation error,
  conflict, cancelled, and persistence/local-install failure; failure/cancellation
  returns non-success process status. Exact numeric codes await command design.
  Structured events use consistent `start`, `success`, `error`, or `warning` events,
  operation correlation, and safe result fields. Logs/Evidence are local by default;
  no Execution engine or portable event history is required.

## Validity and failures

These are validation dimensions, not a new domain lifecycle:

1. **Portable validity:** supported schema, required declarations, reference
   integrity, no forbidden data, and safe contained file references. Explicit
   `unconfigured` values are valid where allowed in the minimum table.
2. **Local installation:** validated portable identity has a local installation
   record. It may contain unresolved Repository or credential bindings; report
   each. A failed record write is incomplete installation, never success.
3. **Operational readiness:** separate dependency observations. This slice does
   not execute workflows or prove Runtime/Integration/model readiness. Never label
   the Project globally ready merely because creation or installation succeeded.

Malformed fields, duplicates, dangling references, unsafe input, incompatible
versions, and conflicting identity prevent writes. Offline remote, absent Runtime,
missing optional Provider, unbound credentials, and absent checkout allow valid
configuration with explicit unresolved/unverified results. Permission denial,
read-only destination, interrupted writes, and concurrent conflict fail without
silently changing existing data. Retry revalidates observed state; it never retries
external side effects because none are authorized.

## Security requirements

- **SEC-001 Secret exclusion:** no secret in portable files, local Project records,
  output, logs, temporary diagnostics, or Evidence. Accept references only; never
  expand environment/store values. Reject secret-bearing URL user-info, sensitive
  URL query parameters, embedded credential fields, and detected secret content
  before writing; do not echo rejected values. Free text must be reviewed/sanitized:
  no detector can prove absence of all arbitrary secrets. Detection failure cannot
  justify claiming complete confidentiality coverage.
- **SEC-002 Authority:** no external configuration, runtime settings, MCP files,
  credential stores, Git config/remotes, or Provider state changed by init/install.
  No implicit network access, permission escalation, or repository hook execution.
- **SEC-003 Paths:** resolve and validate caller-selected target and local paths;
  portable file references must be relative, contained, and free of traversal,
  absolute machine paths, or executable interpolation. Reject symlinks in managed
  write targets/ancestors and portable referenced files; validate final targets
  again at write time to prevent redirection. Read-only local Repository paths
  may resolve symlinks only to a displayed canonical target explicitly confirmed
  for binding; never write through that binding in this slice.
- **SEC-004 Existing data:** no silent overwrite, recursive cleanup of unknown
  content, or partial Project presented as valid. Detect races and fail safely.
- **SEC-005 Sensitive metadata:** local metadata and temporary files use restrictive
  permissions/ACLs appropriate to the platform; no broader inherited exposure.
  Local data is excluded from portable output by construction, not merely by
  `.gitignore`. Exact platform/location contract is Q3. Only sanitized fixtures
  and public-safe Evidence follow the current repository security policy.

## Acceptance criteria and future executable evidence

All cases below are required for future slice completion, not claimed as tested
Lingo behavior today. Use controlled filesystems and synthetic safe inputs;
no real Provider, credential value, model service, or fake production adapter.

| ID | Observable acceptance | Required evidence / coverage |
|---|---|---|
| AC-01 | J1 creates and reopens identical Project intent/ID; required fields enforced; optional explicit gaps valid | Black-box round trip plus unit cases for every minimum-table field (FR-001, 010) |
| AC-02 | Two different-provider remote Repositories need no checkouts; local-only association can be manually installed | Unit Repository rules and two-machine filesystem integration (FR-002–004) |
| AC-03 | J3 preserves portable bytes exactly, binds unrelated paths, reports unavailable checkout, and produces equivalent intent | Before/after hashes and black-box install/reopen on isolated local roots (FR-014) |
| AC-04 | Same init/install is no-op; rename/edit preserves ID; different intent/ID, duplicate keys/remotes/checkouts, and ambiguous targets fail | Unit identity/duplicate cases and black-box conflict matrix (FR-001, 004, 012) |
| AC-05 | Runtime absent/unverified, empty Model Profiles, independent Providers and unconfigured Integrations are represented honestly | Unit reference validation and black-box J2 with no network/model execution (FR-005–007) |
| AC-06 | Missing/unsupported version, unknown/duplicate field, dangling reference and malformed data fail before writes | Parser/validator unit matrix and filesystem unchanged assertions (FR-015) |
| AC-07 | Cancel/retry, permissions/read-only failure, partial local-install failure, interrupted commit and concurrent writer preserve data | Controlled filesystem fault integration plus black-box recovery (FR-013) |
| AC-08 | Secret-bearing inputs rejected without leakage; credential source values never read; all listed source types can remain unresolved | Synthetic redaction cases over output/logs/Evidence/files and controlled no-secret-read assertions (FR-008, SEC-001) |
| AC-09 | Traversal, absolute portable paths, symlink redirection and unsafe targets rejected; local paths can relocate safely | Filesystem security integration, including race tests and platform permission checks (SEC-003–005) |
| AC-10 | No external config/Git/Provider mutation; no hooks, implicit network, or AI invocation | Black-box side-effect assertions in isolated environment (SEC-002) |
| AC-11 | Business Context persists as data; invalid fields corrected independently; missing interactive input fails promptly | Unit context/reference validation and black-box wizard/input tests (FR-009–011) |
| AC-12 | Identical normalized inputs/observations produce equivalent output and stable diagnostics; invalid/local-failed results never report ready | Repeated-run comparison excluding explicit identity allocation and attempt metadata (FR-016) |

Minimum delivery Evidence: approved Specification and resolved blocking questions;
traceable unit, filesystem integration, and functional black-box results for
AC-01–12; sanitized command/exit-status reports, content comparisons and fault
results; platform coverage named explicitly; security review and remaining
limitations. Existing harness checks do not prove these future behaviors.

## Constitution check and review gate

Intent, actors, non-goals and acceptance precede implementation. Deterministic
rules preserve the accepted conceptual distinctions. Optional declarations do
not promote candidate concepts to accepted schemas. No dependency, external
mutation, implementation, or change to an accepted ADR is introduced.

Ready for human review of the proposed scope and behavior. Not approved and
not ready for implementation planning while Q1–Q6 remain unresolved. Stop at
Specification/Clarification; do not create `plan.md` or `tasks.md` in this turn.
