# Explicit workflow gate actions — Issue #138

## Scope and plan

This follow-up implements the gate-action scope of
[#138](https://github.com/rgomids/axiom/issues/138). Canonical Execution history
and the existing ten gates remain the source of truth. This follow-up changes
application gate guidance, the shared CLI adapter and the compatible Runtime run
skill; no new per-gate skills, state format or Provider mutation is introduced.

The MVP epic places final domain-skill convergence after
[#229](https://github.com/rgomids/axiom/issues/229), which remains open. This
implementation retains `axiom-work-item-run` as the compatibility surface. The
future domain skill must route to these same application operations. It does
not claim completion of #229 or waive the epic's delivery sequencing.

## Behavior

- Status and workflow results expose `workflow.gateAction` and an argument-array
  `workflow.gateCommand` with exact persisted selectors and revision. Reading
  status never mutates state. Command placeholders require actual inputs.
- Read-only `skill inspect axiom-work-item-run` discovers `--automatic`; gate
  and outcome are required only when automatic mode is false or absent. The
  inspection requirements and executable parser share the same declarations.
- `workflow advance --automatic` evaluates Intake only. Its prerequisites are
  the already validated Project/Repository/linked Work Item/Execution scope.
  It commits one observable transition using the ordinary transition service.
  No gate, outcome, reference or next override may accompany automatic mode.
- Specification, clarification, planning, tasks, implementation, review,
  Evidence, reconciliation and completion require observed technical results.
  The Runtime performs authorized work and records those results without asking
  the user to name an internal gate. Artifact integrity alone cannot establish
  correctness or the absence of material clarification questions.
- Planning and implementation authority, review start and terminal human
  acceptance use the existing revisioned `workflow fact` operation, validated
  references and explicit local authority. Merely showing a command containing
  `--authorize-local` grants no authority. Human acceptance remains distinct
  from technical completion.
- Blocked, needs-decision and needs-approval conditions prohibit passing a gate.
  The next action identifies the condition to resolve and explicitly clear,
  with reference and authority. Failed outcomes interrupt at the same gate;
  resume is explicit and revision-bound.
- Invalid, stale, denied or unavailable operations do not advance persisted
  state. Corrupt history exposes no mutation action. Provider projection remains
  a separately reviewed and authorized operation.

## Acceptance and verification

| Issue criterion | Contract / verification |
|---|---|
| No magic phrase | Explicit run operations, action metadata, exact command arguments; no conversational gate-name requirement |
| Deterministic progress | Automatic Intake advances through canonical transition; stops before work requiring observed results |
| Human action surface | Fact actions identify authority, reference and active/clear operation before mutation |
| Failed/denied gates stay put | Tests compare full persisted snapshots for stale input, conditions, missing authority, invalid Evidence, cancellation and storage failure |
| CLI/Runtime convergence | Execute the returned CLI command and direct application operation for Codex and Claude; compare state and canonical workflow output |

Preserve shared skill revision history for safe upgrade in both Runtimes.
Verification also covers the ten-gate authority matrix and terminal acceptance.
Repository-wide tests, static checks and validators precede publication of the PR;
the second review inspects behavior, security boundaries, docs and the final diff.

## Compatibility and decisions

Existing explicit `workflow advance --gate ... --outcome ...` remains available.
Missing authority now returns a specific denial before constructing a transition,
rather than reporting recovery for the attempted next state. Existing history is
not rewritten. No new persistence or authority model is adopted, so this bounded
follow-up reconciles the current Specification without a new ADR.

Native host CI and real Runtime dogfooding remain separate observable evidence;
local automated tests do not constitute human acceptance or release approval.

## Implementation Evidence — 2026-10-06

Base: `cfd688d` (`origin/main`, incorporating #131). Verification uses Go 1.26.0 on Ubuntu/WSL,
with a Linux checkout of that base plus the exact changed/new source files to
preserve repository symlinks and Unix filesystem contracts.

- `go test -race ./...`: passed across all packages, including CLI/application
  convergence for Codex/Claude, explicit authority refusals and the full journey
  from the returned Intake action to terminal human acceptance.
- `go vet ./...`, `go build ./...`, `go mod verify`: passed.
- `GOOS=windows GOARCH=amd64 go build ./...` and
  `GOOS=darwin GOARCH=arm64 go build ./...`: passed (cross-compilation only).
- `./scripts/validate-repository.sh .`: passed, including harness structure,
  automation registry, release-Evidence schemas, ADR governance and secret/path
  checks. Its real Runtime behavioral lane explicitly reports unverified.
- `./scripts/dogfood-poc.sh`: passed; six global skills, completed workflow,
  revision 17 and reconciled projection through its bounded fake Provider.
- `go run ./scripts/check-architecture.go domain`: passed.
- The optional `application` architecture profile fails on the pre-existing
  `internal/testfs` test import. Re-running on an untouched archive of the base
  produces the same failure. No application-policy or projectapp files change.

The second review checked all five Issue criteria, the ten-stage action/authority
matrix, terminal acceptance, negative paths, unchanged persistence and separate
Provider authority. It added strict rejection of empty conflicting automatic
arguments, a test executing every returned command through the journey, and
discovery/parser convergence for the conditional automatic-mode arguments.
No new blocking correctness/security finding remains in this bounded scope.
Final domain-skill integration after #229, remote CI, native Windows/macOS test
execution and real installed Runtime behavior remain unverified or pending.
