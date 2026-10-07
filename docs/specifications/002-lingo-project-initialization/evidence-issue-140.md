# Issue #140 — implementation Evidence

Date: 2026-10-07. Scope: local implementation of
[Issue #140](https://github.com/rgomids/axiom/issues/140), its
[Plan](https://github.com/rgomids/axiom/issues/140#issuecomment-6029914946) and
[Tasks](https://github.com/rgomids/axiom/issues/140#issuecomment-6029934192).
Human instruction explicitly authorized implementation; published decomposition
alone is not treated as execution/publication authority.

## Identity and reproducibility

This Evidence describes the PR #249 review-correction head, not the first
published head. The first head `c43ca6b8bec909cd1524a25d7e59f08814c31152`
(base `21bd37fc58397cb1cfcd49894915f8c4eb378fa9`) received REQUEST_CHANGES with
findings CR-001..CR-003; its local snapshot and CI results are superseded and are
not proof of the corrected head.

- Base: `main` at `641901dcc2ec5eb681f45f6d6f7b07d50705f7a8`, merged into the PR
  branch by `7d33cc4` (history preserved, no rebase or force-push; only
  `CHANGELOG.md` conflicted, resolved by keeping `main`'s 0.6.0 section and this
  Unreleased entry).
- Implementation under test: `fccea0eb5e2b474bb99d6c71fe21bd16542613fd` (commits `b365624` and `fccea0e`: code, tests, skill, dogfood
  and docs). The commit that adds this Evidence changes only this file.
- Platform of the local runs below: `darwin/arm64`, Go 1.26 toolchain. Remote CI
  for Linux, macOS and Windows belongs to the PR head and is reported there.

No merge, Issue closure, tag, release, credential change or Runtime
authentication was performed.

## Task results

| Task | Implemented behavior / executable Evidence |
|---|---|
| T140-01 | [Closed v2 contract](runtime-policy-v2.md), Specification/Plan amendment; v1 semantics preserved; authoritative observation, concrete binding and production authoring path documented |
| T140-02 | Project State/Intent, strict version-gated codec, v2 fixtures/goldens and reference/declaration/canonical tests |
| T140-03 | `runtimeapplication` intersects portable policy, local allowlists/model/configuration and requirements with observations Lingo derives itself (`runtimeadapter.ExecutableObserver`); delegates selection to the existing resolver |
| T140-04 | Safe preview binds Project/configuration/observation digests, revisions and `choice.executableDigest`; explicit blockers |
| T140-05 | Dispatch re-observes and checks the exact preview, then the concrete binding (Runtime, Model Profile, model, credential reference, executable identity) before credentials, attempts and processes |
| T140-06 | `runtime profile preview`, workflow-start preview token, `project install --repository`; no observation input; Codex default removed; embedded `axiom-work-item-run` teaches the reviewed protocol |
| T140-07 | Focused, full Go, race, static/build/module, repository, dogfood and secret checks below |
| T140-08 | This Evidence, contract, commands, architecture, help, changelog and Specifications index reconciliation |

## Review findings

| Finding | Status at `fccea0eb5e2b474bb99d6c71fe21bd16542613fd` | Evidence |
|---|---|---|
| CR-001 workflow-start consumers | Resolved | `scripts/dogfood-poc.sh` installs an authored v2 policy, proves `policy_unconfigured`/unobserved-Claude/unprovable-capability blocks, previews, blocks a replaced executable as `stale_preview`, then starts with the reviewed digest; `axiom-work-item-run` teaches preview → `previewDigest` → `--runtime-preview`, never defaults to Codex and preserves `runtimeResolution`/`previewDigest`; `TestExecutableRuntimeAndFirstProjectionOfCreatedWorkItem/{codex,claude}` executes the protocol parsed from the installed skill; `TestWorkItemRunSkillTeachesReviewedRuntimePreview`; payload contract test |
| CR-002 observation bound to dispatch target | Resolved | `--observations` removed; `TestExecutableIdentityBindsPathAndContent`, `TestExecutableObserverProvesOnlyWhatLingoVerifies`, `runtimeapplication.TestExecutableObserverBindsReviewedRuntime`, `cmd/lingo` `TestRuntimePolicyPreviewReadsRecordedSourceAndMachineState`, `TestRuntimePolicyProvesOnlyLingoVerifiedFacts`, `TestWorkflowStartRequiresReviewedFreshPolicy`; dispatch scenarios below |
| CR-003 credential bound to invocation | Resolved | `graphapplication.TestProductionDispatchBlocksUnreviewedBindingWithZeroEffects/{codex,claude}/credential_reference_differs_from_reviewed_configuration`, `TestPolicyInvocationResolvesOnlyTheReviewedCredentialReference`, environment overrides in `TestPolicyInvocationRejectsMissingMismatchedAndOverrideBindings` |

`TestProductionDispatchBlocksUnreviewedBindingWithZeroEffects` runs, for Codex
and Claude, credential B against reviewed credential A, executable replaced after
preview, executable removed after preview, a command profile naming another
same-basename executable, and Runtime unavailable after preview. Each asserts
zero credential resolver calls, zero attempt allocations, zero attempts, an
unchanged persisted graph and no process-start marker. Removing the credential or
executable guard makes the four binding scenarios fail (checked locally).

## Acceptance proof

| Required proof | Tests / source |
|---|---|
| One or multiple portable Runtimes/Profiles; duplicate IDs, dangling refs, invalid preferences rejected | `internal/project/runtime_policy_test.go`: `TestRuntimePolicyValidation`, `TestRuntimePolicyVersionGates`, `TestRuntimePolicyBounds` |
| Strict new-version shape, presence distinctions, safe canonical round trip | `internal/manifest/runtime_policy_test.go`; v1 goldens unchanged |
| Legacy v1 read/execute projection stays explicit without rewriting | `runtimeapplication.TestLegacyV1ProjectionNeverRewritesProject`; frozen stable compatibility corpus |
| Production authoring path for a policy | `TestProjectInstallBindsEveryDeclaredRepository`, `TestAuthoredPolicyPreviewThenStartBothRuntimes/{codex,claude}`; dogfood |
| Both Codex and Claude explicitly allowed | `runtimeapplication.TestProjectResolutionBothExplicitRuntimes`; graph tests per adapter; black-box journey per Runtime |
| Missing/unconfigured/incompatible/unavailable/unproven/ambiguous state blocks with no fallback | `TestProjectResolutionBlocksWithoutFallback`, `TestWorkflowStartNeverFallsBackToCodex`, `TestRuntimePolicyProvesOnlyLingoVerifiedFacts`; dogfood blocks |
| Exact preview used at process dispatch | `TestProductionDispatchUsesPreviewedModelBothAdapters` (synthetic executables, OSProcessRunner, persisted attempts) |
| Configuration/Project/observation/executable/credential drift blocks with zero effects | `TestPreviewRejectsContentAndRevisionDrift`, `TestProductionDispatchStalePolicyHasNoAttemptsOrCredentials`, `TestProductionDispatchBlocksUnreviewedBindingWithZeroEffects` |
| Runtime model/credential overrides cannot bypass the reviewed binding | `TestAlternateModelSelectionRejectedBeforeCredentials`; environment cases in `TestPolicyInvocationRejectsMissingMismatchedAndOverrideBindings` |
| Choice/blocker inspectable, deterministic, bounded and safe | `cli.TestRuntimeResolutionCompletionHumanAndJSON`, `TestRuntimeResolutionCompletionRejectsOversizedOutput`; no credential reference, executable path or state path in output (cmd/lingo tests, dogfood) |
| Runtime-independent Execution contracts preserved | Full executiongraph/coordination/workflow/gitworkspace tests; persisted graph/ledger formats unchanged |

Preview is read-only and grants no new effect authority. Workflow start without a
review token returns preview only; a token requires fresh checks before ledger
creation. Explicit Runtime input narrows candidates; unavailable preferred
profiles cannot fall back to another model.

## Validation

Executed locally on `fccea0eb5e2b474bb99d6c71fe21bd16542613fd`; every command exited 0:

- `go test ./internal/project ./internal/manifest`
- `go test ./internal/runtimeprofile ./internal/runtimeapplication ./internal/runtimeadapter ./internal/graphapplication`
- `go test ./internal/cli ./internal/codexruntime ./cmd/lingo`
- `go test ./... -timeout 10m`
- `go test -race ./... -timeout 10m`
- `go vet ./...`, `go build ./...`, `go mod verify`
- `./scripts/validate-repository.sh .`
- `./scripts/dogfood-poc.sh` (`"runtimeId":"codex"`, `"workflow":"completed"`, `"result":"pass"`)
- `gitleaks dir . --no-banner --redact` (no leaks found)
- `git diff --check`

Embedded skill text changed, so the replaced shared revision was appended to the
installer history and the new embedded manifest digest pinned; published
receipts and historical snapshots are unchanged.

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

- Observation revisions equal the local configuration revision; machine changes
  are visible only through `observationDigest` and `executableDigest`.
- Lingo proves only `axiom-skills`. Capabilities such as `go` or
  `repository-write` have no Lingo-verifiable source and block; this is the
  intended fail-closed outcome, not a missing feature of the resolver.
- Executable identity covers the resolved file (path and content), not
  interpreters or libraries it loads (for example an npm shim's package). A
  replacement between the final identity check and process start is not
  detected. Version stays unknown because Lingo never runs the Runtime.
- Graph dispatch takes its observer from programmatic composition (trusted
  code); the dispatch guard nevertheless re-verifies the executable identity and
  credential reference itself. The environment check is a denylist of known
  credential/selector keys; an explicitly configured `HOME` (or platform
  equivalent) still exposes the Runtime's own ambient login, which Lingo neither
  reads nor binds. No `cmd/lingo` path composes graph dispatch yet, so the
  dispatch guard is exercised by tests and acceptance tooling only.
- `executableDigest` hides the path but lets someone who knows a candidate path
  and binary confirm it; identity hashing reads the whole file on each check.
- `project install --repository` records bindings without a preview/digest, like
  the existing `install`; editing an installed manifest blocks resolution
  (`policy_unavailable`) and has no re-adoption path in this slice.
- `scripts/acceptance/s9-graph-runner.go` (`//go:build ignore`, maintainer
  acceptance tooling) still composes `NewLocalService` without a policy and was
  not migrated; it fails closed if built.
- Concrete adapter coverage uses synthetic executables; no actual Codex/Claude
  model, credential or Provider run is claimed.
- Human review/acceptance, merge and any release need separate authority.
  Issue #140 remains open.

## Publication authority

The original PR creation was explicitly requested by the human. This correction
round was authorized to edit the PR branch, run local validation, commit and push
to the existing branch only. It does not authorize merge, Issue closure, human
acceptance, tag or release.
