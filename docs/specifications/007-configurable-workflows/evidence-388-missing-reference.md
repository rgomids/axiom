# #388 — Implementation Evidence

2026-11-11. Scope and acceptance: [amendment/plan/tasks](amendment-388-missing-reference.md).
Base: `e5b9d10880cfc210356b8c49d639b1718b494e01` (v0.23.6).

## Result and review

The shared `ApplyWorkflow` path now distinguishes absent exact reference fields
from a complete unknown reference for show/select/remove. Presence diagnostics
name the first missing flag in workflow/revision/digest order. The completion
adapter keeps the existing category/envelope and validation-failure exit, with
the stable `missing_required_input:--<flag>` detail and discovery guidance.
No implicit selection fallback or persistence/authority change was introduced.

Ponytail/Axiom connected-path review: no blocking correctness, security or
architecture finding. Reviewed `ApplyWorkflow` callers, lookup, completion and
operator diagnostic adapters, preview/apply authority, CLI examples and tests.
ADR-0020 immutable references and ADR-0022 skill/domain boundaries are preserved.
Load assumption: local bounded authoring operations; the added presence check is
constant work and performs no I/O. No new dependency or automation surface.

## Coverage

- Application table: all absent, each field absent, complete unknown, successful
  exact reference for show/select/remove; nil port proves missing input is
  rejected before observation/lookup; no writes on refusal or preview.
- Executable CLI: all absent and each field absent with a Workflow already
  selected; exact details/field/category and failure exit; complete unknown
  references still fail; existing show/select/remove success and authority tests
  retained. HOME, state and Project roots are synthetic for binary invocations.
- Human CLI: failure exit, missing flag and both reference-discovery commands.
- Shipped skill operation table and discovery example parity checks.
- Shared Runtime history records the complete previous four-skill revision;
  current digests are derived from embedded bytes. Historical fixtures retained.
- Command reference and installed Workflow guide reconciled. Installed docs
  generated through the repository builder into temporary output, with all 26
  inventory sizes/SHA-256 verified and the complete exact-reference example and
  diagnostic present. Generated release artifacts are not committed.

## Check matrix

Executed on the final tree before commit (attempt 3, Claude failover; the
earlier interrupted full run is not counted).

| Check | Result |
|---|---|
| `gofmt -l cmd internal` | PASS (empty) |
| `go vet ./...` | PASS |
| targeted `go test` projectapp, cli, codexruntime, cmd/lingo | PASS (codexruntime first FAILED: new embedded manifest digest not pinned in `publishedSharedRevisions`; pinned, rerun PASS) |
| `go test ./... -count=1` | 30 packages PASS; `cmd/lingo` hit the default 10m timeout under 2-CPU contention with a parallel validator run; rerun isolated with `-timeout 40m`: PASS (581s) |
| `./scripts/validate-repository.sh .` | FAIL, environmental: `test_model_advisor_eval.py::test_unavailable_runtime_exits_nonzero` removes its stub from PATH but this host has a real `claude` in `~/.local/bin`; no script touched by this change; CI is authoritative |
| `./scripts/check-sensitive-files.sh .` | PASS |
| `git diff --check` (staged and unstaged) | PASS |

## Boundaries

No material product/architecture/security/contract decision is pending.
Vendor inference/authentication probes and real-state E2E: NOT_RUN, explicitly
outside authority; synthetic tests only. No release, merge, tag, deployment,
Issue/card closure, status transition or acceptance of #278 / AXM-12 / AXM-1.
`gitleaks`: NOT_RUN, not installed; repository sensitive-file scanning is used.

The Work Item requests the closing phrase `Fixes #388`, while
`scripts/delivery-issues.sh check-pr` (CI `delivery-metadata`) refuses GitHub
closing keywords because Issues close at release publication. Repository policy
prevails: the PR declares `Completes-Issues: #388` instead; no validator change.

Initial validation found a test-only compile error (`completion.Failed` does not
exist); corrected to `completion.ValidationFailure`. The final matrix records
reruns, not inferred success. One obsolete overlapping targeted run was
interrupted to reduce host contention; it is not counted as PASS.
