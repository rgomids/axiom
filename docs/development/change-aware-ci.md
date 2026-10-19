# PR CI and native qualification

The change-aware classifier of Issue #243 (`classify-ci-changes.py`, its plan,
row and conclusion artifacts) is retired by the
[AXM-8](https://linear.app/rgomids/issue/AXM-8/mvp-run-and-coordinate-stage-agents-through-cli-and-runtime-skills) maintainer decision (Linear comment `247f9812-31d2-4c62-af63-7fd972d18f54`)
recorded in [ADR-0029](../decisions/0029-minimal-pr-gate-native-qualification.md)
(Proposed). Its history remains in Git and in the Issue #243 Evidence.

## Pull requests: two required checks

| Required context | Workflow | What it proves |
| --- | --- | --- |
| `PR Checks` | `.github/workflows/ci.yml` | One Ubuntu job: `gofmt`, `go mod tidy -diff`, `go mod verify`, `go vet ./...`, pinned Staticcheck v0.8.1 and its gate contracts (`test-go-quality.py`), `go build ./...`, `go test ./...` and `scripts/validate-repository.sh` (governance and sensitive-file checks). The job's own exit status is the result. |
| `delivery-metadata` | `.github/workflows/delivery-metadata.yml` | Unchanged PR description and Issue metadata contract. |

There are no path filters, aggregators, artifacts or matrices in the PR gate.
Race detection, coverage and macOS are not part of it. The 2–3 minute target is
a hypothesis to measure on real runs, never a reason to skip or filter tests.

Reproduce locally with Go from `go.mod`:

```sh
./scripts/check-go-quality.sh all && go mod verify && go vet ./... && go build ./... && go test ./... && ./scripts/validate-repository.sh .
```

## Native suite outside the PR gate

`.github/workflows/native-suite.yml` is a reusable workflow on an exact SHA:

- `suite (linux|macos)`: `go test -race ./...` (Linux with coverage), vet,
  build, module verification, repository checks and the bounded dogfooding
  smoke; Linux retains a report-only code health snapshot.
- `contracts`: release pipeline, native acceptance, release flow, delivery,
  recovery, Release PR dispatch and code health contracts.
- `upgrade-journeys (linux|macos)`: owned N → candidate upgrades from public
  release assets. Release preparation proposes using its immutable workflow SHA
  for the harness alone; product builds, suites and contracts keep the release
  SHA ([ADR-0031](../decisions/0031-release-validation-harness-pin.md), Proposed).

It runs from three callers; a failed job fails the calling run:

| Caller | Trigger | Role |
| --- | --- | --- |
| `nightly.yml` | daily schedule and `workflow_dispatch` | visible native regression signal on the default branch or a dispatched ref |
| `native-acceptance-qualification.yml` | `workflow_dispatch` | non-publishing qualification; `verify-propagation` requires the native suite to succeed |
| `release-artifacts.yml` | release preparation | gates publication: `verify-prepared-release.sh` accepts only a fully successful run, so a failed or missing native suite at the release revision blocks the preview and publication |

## Release consumers of required checks

`release.sh`, `release-repair.sh` and `publish-release.yml` read the required
contexts from `.github/rulesets/main.json` (excluding the PR-only
`delivery-metadata`), so release CI state now means `PR Checks` on the merged
PR head. Pinned correction metadata keeps the policy committed at its own pin.

## Known limitation

`code-health-baseline.yml` promotes main baselines from PR CI artifacts. With
coverage moved to the native suite, PR runs no longer produce them, so main
baseline promotion reports "unavailable" (report-only; no gate depends on it).
