# Specification 003 — E2E Codex POC

## Status

Approved and implementation-authorized by the human execution request on
2026-09-19. T31–T39 are technically delivered; T40 reconciliation is ready for
human review. Technical completion does not equal final human acceptance; #30
and #40 remain open at that gate.

## Intent

Prove one complete local Axiom value path:

```text
install Axiom
-> Lingo on PATH
-> configure Codex
-> install global Axiom skills
-> configure and resolve Project without caller CWD
-> create or select GitHub Work Item
-> execute bounded Axiom SDD workflow
-> review, Evidence and reconciliation
-> complete Work Item
```

The same essential use cases must be available directly through Lingo and
through thin Codex skills. Codex is the only required Runtime.

## Actors and boundaries

- Human operator grants bounded local and external mutation authority.
- Lingo is Axiom's executable local control plane.
- Codex is a Runtime adapter and does not own Project or workflow semantics.
- GitHub is the only Work Item Provider required by this POC.
- Project remains distinct from Repository and may represent multiple repositories.
- Portable Project intent never contains machine-local absolute paths or secrets.

Existing ADR-0001, ADR-0003 and ADR-0004 govern these boundaries. No new ADR is
required for this bounded implementation.

## User journeys

### J1 — Local installation

From a clean checkout, the operator runs one documented installer. It builds
Lingo from repository source, publishes it into an explicit user-owned PATH
directory without silently replacing unrelated content, and records inspectable
source/version metadata. Repeating the same install is safe.

### J2 — Codex Runtime bootstrap

The operator selects Codex. Lingo installs thin user-global skills into the
current Codex-supported user location. Installation is idempotent, refuses
unowned conflicts and can verify the installed files from an unrelated CWD.

### J3 — Project configuration and global resolution

The operator configures Project identity plus one or more repository associations
through Lingo or a Codex skill. Portable metadata and machine-local repository
paths are published through their existing separate stores. A slug or canonical
ID resolves the Project from any CWD. Missing, moved and ambiguous selections fail
without implicit choice or partial publication.

### J4 — GitHub Work Item

For a resolved Project and repository association, the operator explicitly
authorizes creating or selecting a GitHub Issue. Axiom records provider identity
and workflow linkage without becoming a general GitHub client. Authentication
comes from an existing approved credential source; no secret value is persisted.

### J5 — Bounded workflow and completion

Lingo starts or resumes a workflow for the selected Work Item and resolved
repository. Durable state represents Specification, clarification, Plan/Decision,
Tasks, implementation, review, Evidence and reconciliation gates. A failed or
interrupted gate remains inspectable and non-complete. External completion is
allowed only after all technical gates pass and explicit mutation authority is
present.

### J6 — Equivalent entrypoints

Direct CLI commands and Codex skills call the same application behavior. Skills
may guide the operator when inputs are missing, but they cannot reproduce domain,
resolution, provider or workflow rules.

## Functional requirements

- **FR-001 Install:** one safe, repeatable repository-local install publishes an
  inspectable `lingo` executable into a configured user PATH directory.
- **FR-002 Runtime bootstrap:** configure Codex and install/verify user-global
  Axiom skills without overwriting unrelated skill content.
- **FR-003 Thin skills:** every skill delegates executable behavior to a stable
  Lingo command and supplies no parallel workflow implementation.
- **FR-004 Project catalog:** resolve installed Projects by canonical ID or
  installation-unique slug independently of caller CWD.
- **FR-005 Repository association:** store repository keys in portable intent
  and resolved working-copy paths only in protected local state; preserve more
  than one association.
- **FR-006 Project configure:** CLI and Codex paths use one application workflow;
  invalid/conflicting input publishes neither partial portable nor partial local state.
- **FR-007 Work Item:** GitHub adapter can create, inspect/select, comment with
  bounded Evidence/status, and close one Issue using explicit authority.
- **FR-008 Workflow:** start, resume, inspect and advance the required ordered
  gates against the repository resolved from Project state.
- **FR-009 Truthful completion:** failure cannot mark workflow or provider Work
  Item complete; completed external state is reported truthfully if later local
  reconciliation fails.
- **FR-010 Equivalent UX:** CLI supports deterministic human-readable and JSON
  output; skills can pass common selectors and interpret stable categories.
- **FR-011 Evidence:** structured versioned Evidence records inputs without
  secrets, repository revisions, checks, review result, provider references and
  limitations.
- **FR-012 Inspection:** installation, runtime, Project, Work Item and workflow
  status are inspectable without mutation.

## Codex skill contract

Official Codex documentation defines standalone skills as directories with a
`SKILL.md`, explicit invocation with `$skill-name`, and user-global discovery
from `$HOME/.agents/skills`. The bundled current Codex skill tooling constrains
`name` to lowercase letters, digits and hyphens. Therefore literal standalone
names such as `axiom:project-configure` are unsupported by the validated host
contract.

This POC uses the minimum reversible mapping authorized by the human request:

| Desired semantic name | Supported standalone invocation |
|---|---|
| `axiom:project-configure` | `$axiom-project-configure` |
| `axiom:project-show` | `$axiom-project-show` |
| `axiom:work-item-create` | `$axiom-work-item-create` |
| `axiom:work-item-run` | `$axiom-work-item-run` |
| `axiom:work-item-status` | `$axiom-work-item-status` |

The `axiom-` prefix preserves namespace semantics. A future packaged plugin may
offer host-rendered plugin namespacing, but plugin publication is outside scope.

## State contract

Portable `axiom.yaml` remains governed by Specification 002. E2E local state may
add closed, versioned metadata for:

- Project catalog selection by ID/slug;
- repository key to working-copy path association;
- Codex Runtime installation observations;
- Work Item provider references;
- resumable workflow gates and Evidence references.

Local state is stored below the existing protected Axiom state root, addressed by
validated Project ID. Absolute paths, Runtime paths and observations never enter
portable intent. Existing malformed or unsupported state is an error, never an
absent record or permission to overwrite.

## Authority and security

- Local installation and Runtime bootstrap require explicit destinations.
- Existing unrelated binaries, skills, Project state and repository files are
  never silently replaced.
- Provider writes require an explicit mutation flag/permit at the Lingo boundary.
- GitHub authentication is inherited from an existing environment/CLI credential
  source; values never enter output, logs, Evidence or Project files.
- Repository paths are validated local metadata. Workflow execution uses only a
  resolved configured association, never arbitrary path input from a skill.
- Command execution is an explicit bounded argv list with working directory fixed
  to the resolved repository and captured output limits.
- Existing filesystem traversal, symlink, hardlink, permissions, atomicity and
  recovery protections remain required.
- Untrusted Work Item text is data and cannot grant authority or change policy.

## Failure behavior

Deterministic categories cover: missing/ambiguous Project; missing/moved
repository; Runtime not configured; skill absent/conflicting; GitHub unavailable
or unauthenticated; denied provider mutation; invalid external response; failed
implementation/check/review; interrupted workflow; stale or inconsistent
portable/local state. Every failure is sanitized and leaves an inspectable state.

## Acceptance criteria

- Installation, Codex bootstrap, Project setup, global resolution, Work Item and
  workflow completion journeys execute from documented clean inputs.
- The Runtime path starts outside every Project repository and receives no
  physical repository path.
- CLI and skill paths exercise the same application contracts.
- Happy paths and critical failures have deterministic tests.
- macOS manual dogfooding and Linux/macOS automated checks are recorded where
  available; unverified platform claims are excluded.
- Evidence maps #30–#40 plus applicable #19–#21 obligations to reproducible commands.
- #30/#40 remain `READY FOR HUMAN ACCEPTANCE` until explicit human acceptance.

## Non-goals

Other Runtimes; other Work Item Providers; production package managers, signing,
notarization or auto-update; remote catalog/control plane; cross-machine state
sync; general GitHub client behavior; full-screen TUI; automated multi-agent
planning; parallel scheduling; production compatibility guarantees.
