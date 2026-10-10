# Fast PR Go quality gate

Issue [#242](https://github.com/rgomids/axiom/issues/242) adds the `go-quality`
check to `ci.yml` on every PR and manual CI dispatch, including Release PRs.
Its applicable suite runs alongside verify, release-contract and upgrade
journeys after deterministic [change classification](change-aware-ci.md).
The required context remains present for every PR; documentation-only changes
need successful classification and repository checks rather than Go execution.

The job has a ten-minute timeout covering setup, analyzer installation and
execution. It uses the Go version in `go.mod`, read-only token permissions,
SHA-pinned Actions, and no credentials retained by checkout. It never
invokes a Runtime or Work Item Provider. Manual dispatch runs the full suite.

## Local reproduction

With Go 1.26.0, Bash and Git available, run from the checkout:

```bash
go install honnef.co/go/tools/cmd/staticcheck@v0.8.1
export PATH="$(go env GOPATH)/bin:$PATH"
./scripts/check-go-quality.sh
python3 scripts/test-go-quality.py
```

The gate prints the Go/analyzer versions and failing command. Individual
stages are `format`, `tidy` and `staticcheck`. `STATICCHECK` can point to the
installed executable; the gate refuses another analyzer version. The gate
disables workspace overlays, ambient Go flags and automatic toolchain switching.
Use the same Go version as CI for reproducible formatting and module output.

Formatting uses `gofmt -l` on tracked and non-ignored new Go files, including
tests and platform-specific sources. Parse errors and a nonempty list fail.
Run `gofmt -w` on the listed files to fix them. Module tidiness uses
`go mod tidy -diff`: any required change to `go.mod` or `go.sum` fails without
rewriting either. Run `go mod tidy` and review the diff to fix it.

## Analyzer policy and maintenance

Staticcheck 2026.2.1 (`honnef.co/go/tools@v0.8.1`) supports Go 1.26 and fixes
SA4023 false positives from 2026.2. `check-go-quality.sh` selects safety and
correctness families SA1–SA5 and SA9 plus U1000. U1000 catches unreachable
private code; the initial baseline contained seven obsolete private helpers
or test data, with no references across platform variants. Tests are analyzed.

Style (ST), simplification (S), performance (SA6) and deprecation (SA1019)
are outside this issue's correctness policy. Excluding these categories is
scope selection, not permission to suppress a selected finding.

There are no accepted suppressions in the initial selected baseline. Fix
findings first. A necessary exception must use `//lint:ignore CHECK reason`
immediately above the affected code, explain the invariant and why the report
does not indicate a defect, and receive review with the change. File-wide
`//lint:file-ignore`, global selected-check exclusions and baseline allowlists
are prohibited. Do not weaken the gate to accept a regression.

Toolchain/analyzer updates must update the workflow pin, script version check,
and regression expectations together; rerun the full baseline, selected checks
and negative fixtures before review. Consult the official
[Staticcheck checks](https://staticcheck.dev/docs/checks/),
[release notes](https://github.com/dominikh/go-tools/releases/tag/2026.2.1)
and [Go module reference](https://go.dev/ref/mod#go-mod-tidy).

## Blocking semantics and rollout

The versioned `.github/rulesets/main.json` adds `go-quality`, bound to the
GitHub Actions integration (15368), alongside every existing required context.
Release gates consume this policy, and the release contract fixtures include
the new context. The workflow never tolerates failure or skips these stages.

The remote ruleset is an administrative setting: merging this JSON alone does
not apply it. A maintainer must apply the reviewed policy and read back the
remote required context before claiming branch protection is active. No remote
ruleset update, merge or external acceptance is implied by local validation.

See [baseline and validation Evidence](evidence-go-quality-242.md).

Coverage and differential complexity are separate [report-only code health
signals](code-health.md); they do not change this required gate.
