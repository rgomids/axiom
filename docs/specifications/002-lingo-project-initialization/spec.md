# Specification 002 — Lingo Project Initialization

## Status and authority

**Ready for final human Plan review** — Specification and Plan reconciliation, 2026-09-14.

Specification originally approved by human review on 2026-09-11; Q1–Q6 were
resolved at that gate. Subsequent human Plan review of PR #4 on 2026-09-12
reopened the affected contracts and explicitly approved Project location,
id/slug/name, minimal creation, incremental update and optional portable Git
distribution/synchronization boundaries. These are later human decisions, not
part of the original approval. [Clarifications](clarifications.md) preserves the
original decisions and records their supersession/refinement as H1–H8.
The [partial-input human decision, 2026-09-14](https://github.com/rgomids/axiom/pull/4#issuecomment-5664182508)
confirms that model and refines update input versus persisted state as H9.
Partial command intent is permitted; complete domain-validated state is required.
The [latest atomicity decision](https://github.com/rgomids/axiom/pull/4#issuecomment-5664710882), recorded as H10,
approves the Project model and init/update contract with logical atomicity.
Concrete persistence mechanisms remain implementation choices requiring Evidence.
The [latest Git Authority decision](https://github.com/rgomids/axiom/pull/4#issuecomment-5665679754), H11,
approves the independent Project mutation, local Git commit and remote sync/push
boundary. Project model, init/update and logical atomicity approvals are preserved.

This revision reconciles Specification → Clarifications → [Plan](plan.md).
Final human Plan review remains required; Tasks and Implementation remain blocked
until explicit Plan approval and authorization to advance. No executable Lingo,
adapters, dependencies or application tests are introduced by this change.

Authoritative context:

- [Constitution](../../product/constitution.md) and [current SDD workflow](../../../.agents/skills/axiom-sdd/SKILL.md).
- [ADR-0001](../../decisions/0001-project-is-not-repository.md), [ADR-0002](../../decisions/0002-axiom-speckit-relationship.md), [ADR-0003](../../decisions/0003-lingo-as-axiom-local-control-plane.md), and [ADR-0004](../../decisions/0004-portable-project-manifest.md), all Accepted in the inspected repository.
- [Conceptual model](../../architecture/conceptual-model.md), [Provider boundaries](../../architecture/provider-boundaries.md), [roadmap](../../product/roadmap.md), and [current harness](../../agent-harness.md).
- [Security policy](../../../.agents/policies/security.md) and [repository controls](../../security/repository-security.md).

## Intake

**Problem:** accepted Lingo architecture has no executable contract for creating
and reopening a portable Project. Conflating Project with checkout, Runtime,
or local credentials would prevent safe reuse across machines.

**Outcome:** a human can create, explicitly update, validate, persist, reopen, and locally install
the same Project while preserving identity and portable intent. Parsing,
validation, conflict handling, persistence, and safety are deterministic.

**Actors:** Project author; collaborating developer on another machine; returning
developer resolving changed local paths. No AI participation is required.

**Evidence and assumptions:** the merged PR #2 checkout contains the accepted
ADRs and a documentation/Bash harness, not application code. This slice combines
the minimum configuration contracts behind roadmap blocks 1–4 and 7; it does
not implement their full capabilities. A validated installation-unique slug selects the default portable working copy;
no associated Repository, Workspace ownership or parent-directory discovery is assumed.

**Command hypotheses:** `lingo project init` names creation in this document.
`lingo project install ./axiom-project` names reading a portable definition and
establishing local bindings. `lingo project update <slug>` names explicit evolution. Command names and flag syntax remain illustrative.
After a user clones a configuration repository themselves, install reads that
configuration. Lingo does not clone automatically in this slice.

## Scope and non-goals

The future slice includes guided creation, structural validation, safe
persistence, explicit incremental update, reopening, and installation on a second machine. Reopening and
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

Names below are conceptual fields except the selected manifest name and
`schemaVersion`. `unconfigured` is an optional declaration state, not an approved
global Integration lifecycle. Optional concepts admit only the forms listed below;
optional collections may be empty. Empty strings cannot replace required values.

| Concern | Minimum for a valid portable Project | Optional or deferred content |
|---|---|---|
| Schema | Explicit supported `schemaVersion` in one `axiom.yaml` | Strict parsing; no split manifests or automatic migration in MVP |
| Project identity | Opaque UUID v4 generated once by Lingo, persistent and immutable | Independent identity/fork outside this slice |
| Slug | Required installation-unique CLI name matching `[a-z0-9]+(-[a-z0-9]+)*` | Explicit rename preserves ID; not globally unique |
| Friendly name | Nonblank user-supplied name | Names need not be unique |
| Repositories | May be absent or empty; each association has a stable Project-scoped key | Remote locator optional for local-only Repository; local paths never portable |
| Runtime | May be absent or unconfigured | Optional selected Runtime identifier; vendor-specific local bindings remain separate |
| Providers | May be absent or unconfigured | No required Work Item, Source Control or Documentation Provider declaration, cardinality or default |
| Integrations | May be absent, empty or unconfigured; Provider selection creates no implicit binding | Optional declaration key, Provider reference, requested Capabilities, Transport/reference if known |
| Model Profiles | May be absent, empty or unconfigured | Named profiles with Runtime reference and opaque model identifier when configured; no fixed class set |
| Business Context | May be absent or unconfigured | Human-authored purpose, glossary, acronyms, rules, conventions, constraints: optional text or contained document references |
| Credential references | May be absent, empty or unconfigured | Logical reference identifiers and optional source hints; never secret values |
| Policies | Additional policies may be absent or unconfigured | Portable policy references; cannot relax governing security |

Only Project ID, Project slug, Project name and
`schemaVersion` are required. No Provider becomes mandatory without a concrete
workflow/Capability that requires it; this slice introduces no such requirement.

A selected Provider need not have a configured Integration. Duplicate declaration
keys, dangling references, or a configured binding missing its required fields
are errors. Selected opaque Runtime, Provider, and model identifiers are not
claims that an implementation exists. No global vendor catalog is required.

## Portable configuration and local state

| Data | Source of truth | Rule |
|---|---|---|
| Project ID/slug/name, Repository association keys/remotes, selected Providers/Runtime, declared Capabilities, profiles, Business Context, policies | Portable Project Configuration | Versionable intent, changed only through explicit authoring |
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

The MVP uses one portable manifest, `axiom.yaml`, with explicit `schemaVersion`
and strict parsing. Unknown core fields, duplicate fields and unsupported schema
versions fail before writes. Split manifests are outside the MVP and may be
reconsidered later; no automatic migration or additional schema is selected here.
Optional context/policy document references do not create split configuration
manifests.

Machine-local state uses native operating-system application directories, initially
on Linux and macOS, with an explicit, testable root override for development/tests.
The override name and concrete filesystem strategy remain Plan details. Local state
never enters `axiom.yaml`, never relies solely on `.gitignore`, and uses permissions
appropriate to each platform. Windows is outside the MVP; these boundaries must
permit future Windows support without embedding platform state in portable intent.

### Per-user logical layout

```text
~/.axiom/
├── projects/
│   └── <slug>/
│       ├── axiom.yaml
│       ├── context/
│       └── policies/
└── state/
    └── projects/
        └── <project-id>/
            └── installation.json
```

`projects/` holds portable/shared Project working copies even though located in
the user's home. `state/projects/` holds exclusively machine-local installation
state. The concrete state root retains Q3's native Linux/macOS mapping and explicit
test override; the Plan documents the mapping. No requirement to physically nest
native state below `~/.axiom` supersedes that platform decision.

Creation must not put configuration inside an associated Repository or use
`repository/axiom.yaml` as a convention. The portable working copy contains only
manifest, optional Business Context/documents, portable policies and other artifacts
explicitly supported by the contract. Empty `context/` and `policies/` directories
need no documents at init. Unsupported artifacts are not silently imported.
Absolute Repository/Runtime paths, credentials/tokens, caches, PIDs, local
observations, Execution data and machine bindings are forbidden throughout the
portable content, not just in YAML. Local records keep `installation.json` and
`formatVersion`, distinct from portable `schemaVersion`.

### Identity and evolution

`id` is canonical immutable UUID v4; `slug` is CLI/location ergonomics; `name`
is mutable nonunique presentation. Slug is unique only within an Axiom installation.
Validate its full ASCII grammar deterministically before using it as a path;
reject empty strings, `/`, `..`, separators and unsafe/ambiguous filesystem or CLI
forms. Do not silently normalize invalid input. Rename, update, copy, export, sync
and another installation preserve ID. Different IDs claiming one slug require
explicit resolution, never overwrite or a fresh ID. Slug rename may safely move
working copy; local state remains addressed by ID and preserves binding continuity.

## User journeys

- **J1 — Minimal creation:** author chooses a valid available slug, enters name, creates the minimal
  portable structure with no Repository or documents required, omits optional concepts or leaves them
  `unconfigured`, reviews normalized intent and unresolved dependencies, then
  persists and reopens a valid Project without network or model access.
- **J2 — Declared environment:** author selects Runtime and independent Providers,
  adds two remote Repositories on different Providers, optional Model Profiles,
  purpose text, and credential references. Structure validates while unverified
  bindings and missing Capabilities remain visible.
- **J3 — Second machine:** developer obtains `axiom.yaml` and any referenced documents
  (for example through
  their own Git clone), validates them, binds available Repositories to arbitrary
  local paths and credential references to local sources, and installs. Portable
  bytes remain unchanged; absent checkouts remain unresolved. Installation records gaps
  without
  invalidating the Project or declaring operational readiness.
- **J4 — Return and relocate:** developer reopens the same Project, sees stale or
  missing local paths, explicitly replaces a local binding, and revalidates
  without changing Project identity or portable content.
- **J5 — Retry or conflict:** cancellation before persistence leaves no Project;
  rerun against an identical existing Project is a no-op; a different Project,
  unrecognized files, or interrupted write produces an actionable conflict or
  recovery result with no silent overwrite.

- **J6 — Incremental evolution:** author explicitly updates the selected Project,
  adds a Repository or context document later, reviews the complete proposed state
  and safe diff, authorizes persistence, then reopens the same UUID. Rename of slug
  moves the portable directory safely while local state remains continuous by ID.
- **J7 — Optional distribution boundary:** a future export may establish dedicated
  Git backing for the portable working copy. Associated Repositories remain distinct;
  local mutation, optional Git commit and explicit remote sync are separate actions.
  This journey specifies boundaries only; Git execution is deferred.

## Functional requirements

- **FR-001 Identity:** generate an opaque, immutable UUID v4 once per new Project,
  persist it, and preserve
  it through name/slug rename, update, reopen, copy, export, sync, Runtime change,
  and machine installation.
  Slug, name, remote URL, directory, and Provider IDs are not Project identity.
  UUID allocation uses entropy; identical validated inputs including ID produce
  equivalent canonical output. Copying retains identity; creating an independent
  fork/identity reassignment is outside this slice.
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
  binding after user confirmation and revalidation. Matching is conservative and
  syntactic: for example,
  `git@github.com:org/repo.git` and `https://github.com/org/repo.git` are not
  automatically equivalent. Provider-resolved semantic identity is a future
  Capability; no network dependency is allowed solely for deduplication.
- **FR-005 Runtime:** selection persists an opaque identifier; availability is
  machine-local and reported as available, unavailable, or unverified, with the
  check basis. No adapter or arbitrary executable invocation is introduced to
  fake discovery. This slice can report unverified when no supported safe check
  exists. An explicit validated local executable path can establish presence,
  never capability or model readiness. An absent, unavailable, unverified or
  unconfigured Runtime does not invalidate the portable Project or prevent
  installation; future Runtime-dependent execution must be blocked.
  Replacement preserves Project/Repository identity; incompatible profile bindings
  are reported unresolved, never silently mapped to another model.
- **FR-006 Models:** optional Model Profiles remain configuration, following
  Role → capability/reasoning requirements → Runtime → Model Profile → concrete
  model. No Role-to-model binding, required orchestrator/worker/lightweight class,
  discovery, fallback, or live model verification. Validate references and field
  structure only; unknown model availability remains unverified.
- **FR-007 Providers and Integrations:** separate Provider selection, requested
  Capability, Integration declaration, and Transport. No Transport default to
  MCP. Providers and Integrations may be absent or unconfigured. No mandatory
  Provider declarations, cardinalities or defaults are introduced without a concrete
  workflow/Capability requirement. Optional selections remain independent. Declared
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
- **FR-009 Business Context:** allow absent or unconfigured context, or optional purpose
  text; optional plain text/documents need no complex ontology. Treat
  imported text as untrusted data, never instructions. No chat-history import,
  automatic AI inference, external document fetch, or override of accepted Decisions.
  Future optional AI draft assistance follows FR-019; it is not executed in this slice.
- **FR-010 Guided flow:** generate identity once for a new Project and collect slug/name,
  optional Repository associations and optional
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
  portable edits are independently revalidated on reopen; explicit update is
  specified separately by FR-018. Changed init intent directs the user to update. Registration of the same ID from a different local source path
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
- **FR-015 Versioning:** use a single `axiom.yaml` with strict parsing and explicit
  `schemaVersion`; reject missing, malformed,
  or unsupported versions before writes. Supported versions must have a published
  compatibility rule. No promise to accept an older/newer version merely because
  its number is close. Unknown core fields and duplicate fields are errors, not
  ignored typos. Extension containers, if later approved, need explicit rules;
  this slice does not accept arbitrary executable extensions. No automatic migration,
  down-conversion, or lossy rewrite. Split manifests are outside the MVP; the
  concrete initial version token/support rule is a Plan detail under this contract.
- **FR-016 Deterministic reporting:** same input, version, and filesystem observations
  yield the same validation classification and stable issue ordering. Diagnostics
  identify safe field/key, category, and remedy without echoing raw rejected values.
  Distinguish success, success with unresolved dependencies, validation error,
  conflict, cancelled, and persistence/local-install failure; failure/cancellation
  returns non-success process status. Exact numeric codes await command design.
  Structured events use consistent `start`, `success`, `error`, or `warning` events,
  operation correlation, and safe result fields. Logs/Evidence are local by default;
  no Execution engine or portable event history is required.

- **FR-017 Slug and location:** enforce the identity/location contract above during
  init, install, reopen and update. Validate slug before lookup/path construction;
  detect installation-wide collision between different IDs. Rename requires explicit
  preview/authority over old/new paths, collision and expected-revision checks,
  TOCTOU-safe move, unchanged UUID and ID-addressed local state continuity.
- **FR-018 Explicit update:** commands may receive partial patches/intents. Load
  current portable state; the domain must apply the requested intent to materialize
  a complete proposed Project state, preserving immutable identity. Validate all
  invariants against that complete state, including schema, references, policies and safety;
  show safe diff/preview; confirm human/system authority tied to the exact revision
  and write set; enforce applicable version and concurrency requirements, then
  persist only the complete valid result with logical atomicity and expected-revision
  control. Never expose a partially updated Project as valid. Before commit the
  previous valid Project remains authoritative; after successful commit the complete
  new valid Project is authoritative. No multi-file transaction technique is prescribed.
  Direct partial mutation of `axiom.yaml`, patch persistence, and any bypass of
  domain/application validation are forbidden, regardless of caller.
  Preserve prior valid state on pre-commit failure; after commit report actual committed
  state without falsely claiming rollback. Concurrent conflicting updates cannot
  both succeed. Refresh/invalidate local observations only when affected. Updates
  may add/remove/change Repository associations, name, slug, Runtime, Providers,
  Integrations, Model Profiles, Business Context/documents, policies and credential
  references. Remove/alter operations must resolve dangling references explicitly;
  machine bindings never redefine portable intent. Documents can be supplied with
  an update draft and validated before its commit; they need not predate init.
- **FR-019 AI and authority:** optional future AI interprets intent, proposes drafts
  and explains consequences only. AI proposes → Lingo validates → human/system
  authority approves → deterministic persistence commits. Schema, filesystem,
  identity, conflict/path checks, authority and versioning remain deterministic
  application/domain responsibilities. No direct AI write bypass or AI requirement.
  AI may propose Git operations but receives no implicit authority for local or remote Git effects.
- **FR-020 Optional Git backing:** future export may initialize Git in the portable
  working copy, configure an explicitly supplied origin, commit/publish and record
  backing/sync configuration. This dedicated Git repository is not automatically
  an associated Repository; `.git` in the portable working copy creates no association.
  Export includes only supported portable artifacts;
  local state/bindings and secrets never enter it. Symlinks are filesystem security
  concerns, not the primary distribution/sync mechanism.
- **FR-021 Sync authority:** Project mutation → optional local Git commit → optional
  explicitly authorized remote sync/push are three separate operations with their
  own authority and outcomes. Local mutation implies no Git; its success depends
  on neither Git commit nor push. Later commit/push failures must not undo or
  misreport rollback of an already confirmed local Project mutation. Init, update,
  installation, validation and local Git commit never imply remote publication.
  Machine-local state, credentials, local bindings and observations are excluded
  from portable Git backing. No implicit network mutation, hooks, credential-helper
  changes, remote creation, branch rewrite, force push, merge/rebase or other Git
  effects are authorized. Future autoPush/equivalent automation requires explicit
  authority and separate human-approved design. Git execution remains deferred.

### Git design boundary (H11)

Concrete Git backing schema/metadata, branch strategy, merge/rebase, conflict
resolution, sync protocol, transport, credential helpers, hooks, metadata
preservation, dirty-tree handling, recovery protocol, autoPush implementation and
remote-authority representation remain future design/implementation obligations.
None is selected by this boundary approval; appropriate Evidence and authorized
scope are required before delivery. Any autoPush/equivalent automation requires
explicit authority and separate human-approved design.

## Validity and failures

These are validation dimensions, not a new domain lifecycle:

1. **Portable validity:** supported schemaVersion, Project ID/slug/name, optional
   Repository associations, reference
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

- **SEC-001 Secret exclusion:** secrets are forbidden in portable configuration,
  local Project records, output, logs, temporary diagnostics and Evidence. Accept
  credential references only; never expand environment/store values or echo rejected
  values. Structural validation must deterministically reject credential values in
  reference-only fields, credential-bearing URL user-info, known sensitive URL
  parameters, structures explicitly intended for secret values, and secret material
  in any field whose contract forbids it. These are structural validity failures
  before writes, independent of optional scanners. Free text, Business Context and
  documents may receive best-effort heuristic scanning and human review/sanitization.
  Heuristic findings and scanner failure/unavailability produce appropriate safe
  warnings/Evidence, separate from structural validity. An optional scanner being
  unavailable cannot alone invalidate otherwise valid configuration. Scanning never
  proves complete absence of secrets or justifies absolute confidentiality claims.
- **SEC-002 Authority:** no external configuration, runtime settings, MCP files,
  credential stores, Git config/remotes, or Provider state changed by init/install/update. Future optional export/sync needs separate
  explicit authority; no remote Git effect is implicit.
  No implicit network access, permission escalation, or repository hook execution.
- **SEC-003 Write boundary:** no Lingo write may escape, be redirected, or reach
  any destination other than the target explicitly authorized by the user. Protect
  all writes against path traversal, symlink redirection, filesystem races/TOCTOU
  and writes outside the approved boundary, including during persistence. Portable
  file references remain relative and contained, without absolute machine paths or
  executable interpolation. Read-only Repository bindings may resolve paths safely,
  including symlinks, provided that no implicit write is authorized; bindings remain
  read-only in this slice. Concrete filesystem techniques belong to Plan/implementation,
  not this invariant.
- **SEC-004 Existing data:** no silent overwrite, recursive cleanup of unknown
  content, or partial Project presented as valid. Detect races and fail safely. Update requires expected revision; failed
  pre-commit update preserves the old state; slug rename protects collision/TOCTOU
  and preserves ID-addressed installation continuity.
- **SEC-005 Sensitive metadata:** local metadata and temporary files use restrictive
  permissions/ACLs appropriate to the platform; no broader inherited exposure.
  Local data is excluded from portable output by construction, not merely by
  `.gitignore`. Linux and macOS native directories and the explicit local-root
  override must satisfy the same protections; state never enters `axiom.yaml`. Only
  sanitized fixtures
  and public-safe Evidence follow the current repository security policy.

## Acceptance criteria and future executable evidence

All cases below are required for future slice completion, not claimed as tested
Lingo behavior today. Use controlled filesystems and synthetic safe inputs;
no real Provider, credential value, model service, or fake production adapter.

| ID | Observable acceptance | Required evidence / coverage |
|---|---|---|
| AC-01 | J1 creates and reopens a valid Project using only generated UUID v4, slug, name and schemaVersion, with zero Repositories/documents; each required field enforced; optional concepts admit their documented absent/unconfigured/empty forms | Black-box minimal round trip and unit required/optional-field matrix (FR-001, FR-010, FR-015) |
| AC-02 | Different-provider remote Repositories need no checkout or Provider declaration; local-only association permits manual binding; SSH/HTTPS aliases are not automatically equated and ambiguous aliases require human resolution without network deduplication | Unit key/locator rules, conservative normalization cases and two-machine filesystem integration (FR-002–004) |
| AC-03 | J3 preserves portable bytes/UUID exactly across Linux and macOS installations, binds unrelated paths, records unresolved checkouts/credentials and equivalent intent without declaring readiness | Before/after hashes and black-box install/reopen using native local directories and isolated explicit root overrides; local state absent from manifest (FR-014, SEC-005) |
| AC-04 | Same init/install is no-op; rename/edit, Runtime change and copy preserve immutable UUID; changed init intent/different ID, duplicate keys/exact or safely normalized remotes/checkouts, and ambiguous targets conflict | Unit UUID/duplicate cases and black-box conflict/relocation matrix; no independent fork identity (FR-001, FR-004, FR-012) |
| AC-05 | Runtime/Providers/Integrations/profiles may be absent or unconfigured without artificial declarations; available/unavailable/unverified observations state their basis; unavailable/unverified does not invalidate Project/install or prove capability/model readiness | Unit optional/reference matrix; controlled presence-check cases and black-box J2 without network, model or arbitrary Runtime execution (FR-005–007) |
| AC-06 | One axiom.yaml requires schemaVersion; missing/malformed/unsupported version, unknown/duplicate core field, dangling reference and malformed data fail before writes; no split manifest or automatic migration | Strict parser/validator unit matrix and unchanged filesystem assertions (FR-015) |
| AC-07 | Cancel/retry, permissions/read-only failure, partial local-install failure, interrupted commit and concurrent writer preserve existing data; conflicting writers cannot both succeed | Controlled filesystem fault integration plus black-box recovery (FR-013) |
| AC-08 | Deterministic rejection covers credential values in reference-only/prohibited fields, credential-bearing URL user-info, known sensitive URL parameters and secret-value structures before writes, without leakage; optional free-text/context/document scanning gives safe findings or unavailable/failure warnings/Evidence, never proof of secret absence or structural invalidity solely from scanner unavailability; credential values never read and sources may remain unresolved | Synthetic structural rejection/redaction matrix over output/logs/Evidence/files, scanner available/finding/unavailable/failure cases and controlled no-secret-read assertions (FR-008, SEC-001) |
| AC-09 | No write escapes, is redirected or reaches a destination other than the explicitly authorized target under traversal, symlink redirection, races/TOCTOU or outside-boundary attempts; absolute machine paths in portable references fail; safe read-only bindings permit no implicit writes | Filesystem security integration with adversarial races, unchanged unauthorized-target assertions, safe binding relocation/resolution and Linux/macOS permission checks; no prescribed filesystem technique (SEC-003–005) |
| AC-10 | Init/install/update perform no external config/Git/Provider mutation; no hooks, implicit network, or AI invocation | Black-box side-effect assertions in isolated environment (SEC-002) |
| AC-11 | Business Context may be absent/unconfigured or persist as untrusted data; invalid fields corrected independently; missing required interactive input fails promptly | Unit optional context/reference validation and black-box wizard/input tests (FR-009–011) |
| AC-12 | Identical inputs/observations produce equivalent output and stable diagnostics; portable validity, installed-with-gaps and installation failure stay distinct; no creation/install or presence check implies operational readiness | Repeated-run comparison excluding UUID allocation/attempt metadata; classification matrix including valid Project with local gaps and failed local persistence (FR-016) |
| AC-13 | Invalid/empty/traversal slug fails before path use; different IDs with same installation slug conflict; no global uniqueness inferred | Unit grammar/identity matrix and two-process init/install collision integration; unchanged occupied destination (FR-017, SEC-003–004) |
| AC-14 | Explicit slug rename moves portable directory, preserves project.id and local state continuity; name need not be unique | Black-box old/new lookup, document hashes, same ID-addressed installation/bindings; collision, symlink/TOCTOU and move-failure integration (FR-001, FR-017, SEC-003–005) |
| AC-15 | Explicit update succeeds where changed init conflicts; minimal init needs no documents; partial update intent materializes a complete proposed state, validates all invariants, shows safe diff and enforces authority/version/concurrency before complete persistence | Unit partial-intent add/remove/reference matrix (including invalid retained references), no-direct-patch persistence, no-authority/stale-preview cases and black-box J6 including document addition, declaration states and unchanged UUID (FR-012, FR-018–019) |
| AC-16 | Logical atomicity: no partial Project accepted as valid; pre-commit failure preserves prior authoritative state; successful commit makes complete new state authoritative; conflicting updates cannot both succeed; post-commit failures report actual state without false rollback | Linux/macOS fault/crash tests around adapter-defined commit, concurrent readers/writers, confinement/collision tests including manifest/documents and rename versus update, identity/local-state continuity recovery (FR-013, FR-017–018, SEC-004) |
| AC-17 | Entire portable working copy excludes machine state and secrets; home location does not make it private local state | Unit artifact allowlist, integration portable/local root overlap and export-set exclusion, black-box portable snapshot checks (FR-020, SEC-001, SEC-005) |
| AC-18 | Optional backing/`.git` creates no Repository association; local mutation, optional Git commit and explicit remote sync have independent authority/outcomes; local success depends on neither commit nor push; no false rollback or local-state export; AI has no implicit Git authority | Denied Git/network spies for current init/update/install/validate; future authorized Git tests for association independence, portable-only backing, AI/authority denial, no implicit publication, commit/push failures preserving confirmed local success, and separately approved automation before Git delivery (FR-020–021, SEC-002) |

Minimum delivery Evidence: approved Specification and resolved blocking questions;
traceable unit, filesystem integration, and functional black-box results for
AC-01–18, with Git execution evidence in AC-18 deferred until authorized Git
delivery (current slice must prove denied side effects and boundary contracts);
sanitized command/exit-status reports, content comparisons and fault
results; platform coverage named explicitly; security review and remaining
limitations. Existing harness checks do not prove these future behaviors.

## Constitution check and review gate

Intent and acceptance precede implementation. Constitution I–IV require reopening
materially changed approved intent: the 2026-09-11 approval remains historical,
while H1–H8 record explicit subsequent human authority on 2026-09-12 and H9
records partial-input refinement; H10 approves the model and init/update contract
with logical atomicity on 2026-09-14; H11 approves Git Authority. ADR-0004
is extended within its portable/local boundary; ADR-0001–0003 remain unchanged.
Constitution V–VIII retain deterministic authority, least privilege and
Project != Repository without imposing aggregate/Workspace ownership.

H9/H10 resolve partial input, complete-state validation and logical atomicity.
H11 resolves Git Authority; final human Plan review and future Evidence obligations are identified in
[Plan review concerns](plan.md#remaining-review-concerns-and-deferred-design).
They are not implicit decisions or permission to implement. Current gate:
**Ready for final human Plan review** of reconciled Specification/Clarifications/Plan.
Tasks and Implementation remain blocked pending explicit human Plan approval and
authorization to advance. All acceptance Evidence above remains future work.
