# Issue #140 — implementation Evidence

Date: 2026-10-07. Scope: local implementation of
[Issue #140](https://github.com/rgomids/axiom/issues/140), its
[Plan](https://github.com/rgomids/axiom/issues/140#issuecomment-6029914946) and
[Tasks](https://github.com/rgomids/axiom/issues/140#issuecomment-6029934192).
Human instruction explicitly authorized implementation; published decomposition
alone is not treated as execution/publication authority.

## Identity and reproducibility

Base HEAD: `cfd688d8ec63aa9c4fc51777bd84d6445b519279`.
Pre-publication validation snapshot:
Local branch: `agent/project-runtime-policy-140`; implementation is an uncommitted
reviewable working-tree change, not a published revision. No local commits, push,
PR, merge, Issue closure, Runtime authentication or release publication performed.
Local test platform: `darwin/arm64`, Go module's existing Go 1.26 toolchain.

The 45 changed Go/manifest fixture files have aggregate SHA-256
`f3dddf9f74ea6dd88dcf6191f56214b8cce034d1b7d23f02f052eb0587407849`.
Reproduce from the base checkout and this complete working-tree patch: take the
sorted union of `git diff --name-only cfd688d8ec63aa9c4fc51777bd84d6445b519279`
and `git ls-files --others --exclude-standard`,
retain `internal/` or `cmd/` paths ending in `.go` or `.yaml`, and hash the
concatenation of each UTF-8 path, NUL, and its binary SHA-256. This binds code and
fixtures without importing raw tool/chat output into Evidence.

## Task results

| Task | Implemented behavior / executable Evidence |
|---|---|
| T140-01 | [Closed v2 contract](runtime-policy-v2.md), Specification/Plan amendment; v1 semantics preserved |
| T140-02 | Project State/Intent, strict version-gated codec, v2 fixtures/goldens and reference/declaration/canonical tests |
| T140-03 | `runtimeapplication` intersects portable policy, local allowlists/model/configuration and requirements; delegates selection to the existing resolver |
| T140-04 | Safe preview binds Project/configuration/observations by digests, records configuration/observation revisions, and returns explicit blockers |
| T140-05 | Production graph composition requires per-child previews; exact child/model binding and freshness checked before credentials/attempts/processes |
| T140-06 | `runtime profile preview`, human/JSON rendering and workflow-start preview token; truthful CLI argument discovery; Codex default removed |
| T140-07 | Focused, full Go, race, static/build/module, repository and secret checks below |
| T140-08 | This Evidence, contract, commands, architecture, changelog and Specifications index reconciliation |

## Acceptance proof

| Required proof | Tests / source |
|---|---|
| One or multiple portable Runtimes/Profiles; duplicate IDs, dangling refs, invalid preferences rejected | `internal/project/runtime_policy_test.go`: `TestRuntimePolicyValidation`, `TestRuntimePolicyVersionGates`, `TestRuntimePolicyBounds` |
| Strict new-version shape, presence distinctions, safe canonical round trip | `internal/manifest/runtime_policy_test.go`: `TestV2VersionBoundary`, `TestV2RejectsMalformedPolicy`, `TestV2DeclarationStatesIndependentRoundTrip`, security/boundary tests; v1 goldens unchanged |
| Legacy v1 read/execute projection stays explicit without rewriting | `runtimeapplication.TestLegacyV1ProjectionNeverRewritesProject`; existing v1 codec tests and frozen stable compatibility corpus |
| Both Codex and Claude explicitly allowed | `runtimeapplication.TestProjectResolutionBothExplicitRuntimes`; `graphapplication.TestPolicyInvocationChecksExactSelectionBeforeCredentials`; `TestProductionDispatchUsesPreviewedModelBothAdapters` |
| Missing/unconfigured/incompatible/unavailable/unproven/ambiguous state blocks with no fallback | `runtimeapplication.TestProjectResolutionBlocksWithoutFallback`, `TestProjectPreferenceAndExplicitNarrowing`; resolver regressions |
| Exact preview used at process dispatch | Production dispatch test creates isolated synthetic Go executables named Codex/Claude, verifies adapter argv/model, executes OSProcessRunner and reads persisted successful attempts |
| Configuration/Project/observation drift blocks | `TestPreviewRejectsContentAndRevisionDrift`; `TestProductionDispatchStalePolicyHasNoAttemptsOrCredentials`; zero credential calls, allocations, attempts and persistence changes |
| Runtime model overrides/fallback cannot bypass reviewed model | `TestAlternateModelSelectionRejectedBeforeCredentials`; explicit denial of model/config/profile/agent/fallback/session override flags |
| Choice/blocker inspectable, deterministic, bounded and safe | `cli.TestRuntimeResolutionCompletionHumanAndJSON`, `TestRuntimeResolutionCompletionRejectsOversizedOutput`; source/inventory/workflow tests in `cmd/lingo/runtime_policy_test.go` |
| Observation JSON cannot widen state via duplicate/case/Unicode aliases | `TestRuntimeObservationInventoryRejectsUntrustedData`, including `installed` versus Unicode long-s alias; canonical keys only |
| Runtime-independent Execution contracts preserved | Full existing executiongraph/coordination/workflow/gitworkspace tests; persisted graph/ledger formats unchanged |

Preview is read-only and grants no new effect authority. Workflow start without a
review token returns preview only; a token requires fresh checks before ledger
creation. Production graph invocation also checks the corresponding child scope,
role, complexity, capability set and existing resolution revisions. Global local
preferences cannot synthesize Project preferences. Explicit Runtime input narrows
candidates; unavailable preferred profiles cannot fall back to another model.

## Validation

All commands below were executed locally; each final run exited 0:

- `go test ./internal/project ./internal/manifest`
- `go test ./internal/runtimeapplication ./internal/graphapplication ./internal/runtimeprofile ./internal/runtimeadapter`
- `go test ./internal/compatibility ./internal/runtimeapplication ./internal/runtimeprofile`
- `go test ./internal/cli ./cmd/lingo`
- `go test ./... -timeout 10m`
- `go test -race ./... -timeout 10m`
- `go vet ./...`
- `go build ./...`
- `go mod verify`
- `./scripts/validate-repository.sh .`
- `gitleaks dir . --no-banner --redact`
- `git diff --check`

An early full run found historical fixtures assuming implicit Codex and schema 2
as unsupported. Fixtures now declare explicit policy/observations and unsupported
schema tests use version 3; no frozen stable corpus was changed. Independent
engineering/security review found two blocking issues: adapter fallback-model
arguments and Unicode JSON key aliasing. Both were corrected and regression-tested.
The local review identified no remaining blocking finding after remediation.

## Compatibility and architectural checks

Portable v1 stays readable/encodable with unchanged golden bytes and no automated
rewrite. V2 replaces singular `runtime` with `runtimes` and adds optional
`runtimePreferences`; invalid mixtures and unsupported versions fail closed.
Absent/empty preferences impose no preference; explicit unconfigured preferences
block until reconciled. Missing configured Runtime/Profile collections block.
Capabilities, adapter bindings, executable paths, credentials, environment,
availability and observations remain machine-local. Runtime != Model,
Role != Model, Execution != Agent and Project != Repository remain intact.

The portable v2 shape is declared in the contract's release-generation section.
Its first published generation baseline and representative frozen corpus remain
release-acceptance obligations; this local implementation creates no release.

## Limits and remaining human actions

- Concrete adapter/process coverage uses synthetic executables and isolated Git
  worktrees. No actual Codex/Claude model, credential or Provider run is claimed.
- CLI reads an explicitly supplied operator inventory, not live vendor capability
  discovery. It detects changes to that inventory; it cannot prove unreported
  machine changes. Programmatic dispatch sources must observe current machine
  state on each check. Observation timestamps alone do not constitute revisions.
- Native Linux/Windows runs and remote CI were not executed in this local macOS
  validation. Existing platform-specific coverage runs locally where supported.
- Embedded Runtime skills/receipts and installation/bootstrap formats remain
  unchanged; binary argument discovery advertises the new required policy inputs.
- Human review/acceptance, commit if desired, push/PR and any later release need
  separate authority. Issue #140 remains open; no successor work is authorized.

## Subsequent publication authority

After the local validation snapshot, the human requested creation of the PR.
That request authorizes the necessary commit, branch push and PR creation only.
It does not authorize merge, Issue closure, human acceptance or release. Published
commit/head and remote CI status belong to the PR; the local test snapshot above
remains historical Evidence rather than a claim that remote CI already passed.
