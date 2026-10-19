# ADR-0029 — Minimal PR gate with native qualification before release

## Status

Accepted on 2026-10-16 by explicit maintainer decision in
[AXM-8](https://linear.app/rgomids/issue/AXM-8/mvp-run-and-coordinate-stage-agents-through-cli-and-runtime-skills)
(Linear comment `fee15434-632b-4239-8d14-55653cab0ad7`). It records the [AXM-8](https://linear.app/rgomids/issue/AXM-8/mvp-run-and-coordinate-stage-agents-through-cli-and-runtime-skills)
maintainer decision of 2026-10-15 (Linear comment
`247f9812-31d2-4c62-af63-7fd972d18f54`, "simplificação T43 + CI mínimo").
The decision approves two required PR checks. The maintainer applied the remote
`main` ruleset `22828068` change on 2026-10-16 (required contexts exactly
`PR Checks` and `delivery-metadata`, strict; other protections unchanged),
read back on 2026-10-16. No merge or release is inferred.

## Context

PR CI had seven required contexts: `go-quality`, `verify (linux|macos)`,
`release-contract`, `upgrade-journeys (linux|macos)` and `delivery-metadata`.
A path classifier selected suites, and always-running conclusion jobs
re-derived each result from uploaded plan/row artifacts. On run 38832717686 the
Linux suite passed, but the `verify (linux)` conclusion failed because it was
coupled to the matrix result that the macOS row failed. The artifact plumbing
tested the plumbing, not the code, and every PR paid for macOS race suites,
release contracts and upgrade journeys.

## Decision

1. Every PR has exactly two required checks: `PR Checks` (one Ubuntu job:
   `gofmt`, `go mod tidy -diff`, `go mod verify`, `go vet ./...`, pinned
   Staticcheck, `go build ./...`, `go test ./...`, repository and
   sensitive-file validation) and the unchanged `delivery-metadata`.
2. The classifier, conclusion jobs and their artifacts are removed. No tests
   are deleted or filtered; they move to where they run.
3. A reusable `native-suite.yml` runs Linux/macOS `go test -race` (Linux with
   coverage), release and governance contracts and upgrade journeys on an
   exact SHA. It is called by a daily scheduled and manually dispatchable
   `nightly.yml`, by the manual native qualification workflow, and by
   `release-artifacts.yml` at the release revision.

> Superseded by [ADR-0031 — Pin upgrade-journey validation separately from release artifacts](0031-release-validation-harness-pin.md#decision), accepted 2026-10-18.
> Only the upgrade-journey harness pin is superseded; release artifacts and all other native-suite validation remain bound to the release revision.

4. Publication already accepts only a fully successful `release-artifacts.yml`
   run (`verify-prepared-release.sh`). A failed or missing native suite at the
   release revision therefore blocks preview and publication, even with a
   green PR. All other release gates are unchanged.
5. Release scripts keep reading required contexts from the versioned ruleset;
   historical correction pins keep their own committed policy.
6. The remote ruleset migrates in a safe order: it switches to the two contexts
   only together with the merge of the PR that introduces them. Review, Code
   Owners, strict up-to-date, thread resolution, squash-only, deletion and
   force-push protections stay as they are.

## Alternatives considered

| Alternative | Trade-offs |
| --- | --- |
| A — two PR checks, native qualification nightly and before release (chosen) | Fast, direct PR signal; platform and race regressions surface nightly or block a release rather than a PR. |
| B — keep change-aware routing and conclusion jobs | Preserves per-PR macOS and race coverage; complex, slow and produced the coupled false failure. |
| C — required macOS PR job without race | Catches macOS breaks per PR at higher cost and queue time; not chosen for the MVP. |

## Consequences

### Positive

The PR gate is one readable job whose exit status is the result. Native,
release and upgrade validation still runs on real runners and gates releases.

### Negative / trade-offs

A macOS-only or race regression can merge and is seen only by the next nightly
or release preparation. Release preparation takes longer and can fail late.
The code health baseline loses PR-sourced main snapshots (report-only). Required
check names are part of release and ruleset contracts; renaming them again
requires the same coordinated migration.

## Revisit when

Nightly failures repeatedly reach release preparation, PR CI time measurements
miss the target materially, or another supported platform enters scope.

## Disposition — 2026-10-16

Decision item 6 is satisfied: the maintainer switched ruleset `22828068` to
exactly `PR Checks` and `delivery-metadata` (strict) before PR #320 merged, and
the read-back on 2026-10-16 showed the other protections unchanged. The PR #320
`ci.yml`, `native-suite.yml`, `nightly.yml` and release-gate changes implement
items 1–5.
