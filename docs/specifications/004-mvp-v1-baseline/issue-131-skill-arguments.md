# Issue #131 — Inspect Runtime skill arguments

## Intake and specification

Implement the authorized request [#131](https://github.com/rgomids/axiom/issues/131)
on main `e28dc51`. Actors are users and Runtime adapters inspecting Axiom's six
embedded product skills before invocation. Maintainer skills under `.agents/`
are a separate harness (ADR-0014), not executable product commands.

The existing CLI owns flag registration, strict long-option parsing, missing
selector validation and guided acquisition. Embedded SKILL.md files own thin
workflow instructions; their versioned manifest owns the installed inventory.
ADR-0003 and Specification 004 FR-018–020 require that authority remain in Lingo.

## Contract and clarification

`axiom [--human|--json] skill inspect <exact-skill-name>` inspects the binary's
embedded skill contract, independently of installation or Project state. It
accepts no workflow arguments. Unknown names, extra inputs and malformed
inspection requests return canonical `validation_failure`, never partial success.

JSON carries canonical completion fields and a separate `skill` payload with
`name`, `commands`; each command has `command` and `arguments`. Each argument
has `name`, `required` (always serialized), optional `requiredWhen`, `description`,
`acceptedForms` and `repeatable`. Required means needed by the noninteractive CLI
request, not required to start the conversational skill. Conditional requirements
are explicit; optional content can still be requested by the guided/domain
validation layer before an effect. Descriptions state such downstream conditions.
No new host-specific argument syntax is promised.

Names, types, forms and descriptions come from the actual Go flag registrations.
Required-input rules are shared by request validation and discovery. Repeatability
comes from the registered flag value type. Existing conflict/value validation
remains authoritative. No independent argument registry or static duplicated
argument table is introduced. Missing required values may still be gathered by
the existing guided path; discovery never reads stdin or invokes that path.

Inspection is handled before service composition and dispatch. It reads no local
Project, Runtime or Provider state, runs no process, writes no persistence and
never advances workflow. Only bounded output is written. It works with invalid
state-root configuration and unavailable application services.

## Plan and tasks (before implementation)

1. Extract existing flag-set construction without changing accepted inputs; add
   descriptions at registration and share required-input/repeatability rules.
2. Add bounded CLI inspection rendering, early routing and an embedded-inventory
   consistency test. Expose all commands used by each skill, including workflow
   fact/status/evidence used by the run skill.
3. Point thin skills and command reference to inspection; preserve the replaced
   skill revision in shared upgrade history.
4. Test required, optional, mixed and conditional inputs, descriptions/forms,
   unknown/conflicting inputs, inventory/parser consistency, zero dispatch/input
   during discovery and guided invocation after discovery. Run focused then broad
   validation and review specification, architecture, verification and security.
5. Reconcile evidence and acceptance. No merge, release or Issue closure implied.

Constitution/ADR assessment: this additive CLI projection preserves existing
control-plane, storage and Runtime boundaries, uses the standard library and
needs no new ADR. The reversible schema is specified here. No hierarchical help,
sibling implementation, arbitrary skill loading, dependencies, telemetry or new
persistence. Rollback removes inspection and restores prior skill text; existing
workflow invocations retain their semantics.

## Verification and review evidence (2026-10-05)

Validated on Go 1.26.0. Linux checks use an ext4 Ubuntu/WSL checkout of the
same base with the changed source files copied byte-for-byte; native Windows
checks use the isolated feature worktree. No Provider or Runtime invocation
was needed for these checks.

| Command | Result |
|---|---|
| `go test ./internal/cli -count=1` | PASS, native Windows, including existing guided, strict-selector and completion tests. |
| `go test ./internal/cli ./cmd/lingo ./internal/codexruntime -run 'TestSkillDiscovery\|TestExecutableSkillDiscovery\|TestEveryPublishedSharedRevisionStaysOwned' -count=1` | PASS, native Windows; actual binary discovery, required-field serialization and historical manifest ownership. |
| `go test ./internal/cli ./internal/codexruntime ./cmd/lingo -count=1` | PASS, Linux; all affected packages including installation/upgrade and real guided invocation. |
| `go test -race ./...` | PASS, Linux, all packages. |
| `go vet ./...` | PASS, Linux. |
| `go build ./...` | PASS, Linux. |
| `go mod verify` | PASS, all modules verified. |
| `./scripts/validate-repository.sh .` | PASS, Linux; includes harness, automation/ADR governance and sensitive-file checks. |
| `./scripts/test-codex-skills.sh` | PASS, Linux Go contracts; optional external Codex Python validator unavailable. |
| `go test -race ./internal/cli ./cmd/lingo` | PASS, Linux, repeated after final description/test refinements. |
| `./scripts/dogfood-poc.sh` | PASS, Linux, `globalSkillCount=6`, `workflow=completed`, `result=pass`. |
| `git diff --check` | PASS. |

Native Windows's broad affected-package run failed in existing filesystem
publication/ownership tests (including `codex_skill_root_unavailable` and
`Portable Project publication failed`); retrying with a private temporary root
did not resolve them. Both `TestInstallRefusesRootReplacedBeforeAnyChange` and
`TestExecutableGuidedProjectConfiguration` reproduce the same failures on the
unmodified base `e28dc51`. This change does not claim to fix or validate those
Windows filesystem paths. The new inspection boundary passes natively without
requiring those paths. macOS and live Codex/Claude conversational behavior were
not exercised locally; no claim of external host acceptance is made. The optional
`gitleaks` executable was unavailable; repository sensitive-file scans passed.

### Acceptance matrix

The GitHub issue contains four explicit acceptance criteria. The fifth row below
records its separate workflow-isolation scope requirement, as requested by the
implementation brief; it is not a rewritten Issue criterion.

| Criterion | Result | Concrete evidence |
|---|---|---|
| Inspect accepted arguments without executing a skill workflow | PASS | `TestSkillDiscoveryInventoryAndNoExecution`, `TestExecutableSkillDiscoveryWithoutStateOrWorkflow`: all six skills, nil/panic-on-use service, unread stdin, invalid state configuration, no state files. |
| Distinguish required and optional arguments | PASS | `TestSkillDiscoveryRequiredOptionalDescriptionForms`, `TestSkillDiscoveryRequirementsMatchRequests`, executable test's assertion that every argument serializes a boolean `required`; conditional requirements and zero-input listing included. |
| Documentation and actual parsing remain consistent | PASS | `TestSkillDiscoveryMatchesParserAndAcceptedForms`: same registered flags, descriptions, both long-option forms, repeatability and unknown-input rejection for every mapped command. Shared executable presence rules; removing each required fixture input fails. |
| Existing fully guided invocation remains supported | PASS | `TestSkillDiscoveryPreservesFullyGuidedInvocation`; existing guided create, missing-only and exact-preview authorization tests; Linux `TestExecutableGuidedProjectConfiguration` in the affected-package and race runs. |
| Discovery stays separate from domain/workflow execution | PASS | Early `InspectSkill` route before composition in `cmd/lingo/main.go`, before dispatch/prompts in `RunInteractive`; binary and panic-on-use boundary tests. Unknown/conflicting inputs retain canonical validation failure in `TestSkillDiscoveryRejectsInvalidRequestsAndWorkflowConflicts` and the strict-selector suite. |

### Four review lenses

- Specification: all six embedded product skills expose their delegated command
  arguments; required/conditional/optional semantics and inspection errors are
  explicit. No hierarchical help or sibling dependency was introduced.
- Architecture: names/types/descriptions derive from existing Go `FlagSet`
  registrations; shared presence rules drive parsing and inspection. Domain
  validation, conflict ordering, authority and guided acquisition remain in
  their existing layers. The only mapping is skill-to-existing-command routing.
- Verification: behavioral tests cover the returned contract and public binary,
  regression checks cover guided acquisition and strict failures, and shared
  history preserves the replaced skill manifest. No implementation-only success
  claim substitutes for the acceptance evidence above.
- Security: no executable skill loading, service composition, stdin, network,
  credentials, persistence, permission expansion or Provider effect on inspection.
  Exact closed names, bounded output and sanitized canonical failures. Existing
  unknown, duplicate and conflict behavior is preserved.

No unresolved blocker, critical or major finding was identified in this scoped
review. Human review/acceptance, merge and release remain separate; the Issue
must not be closed by this implementation record.
