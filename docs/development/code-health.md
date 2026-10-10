# Go coverage and code health

## Contract and implementation plan

Issue [#245](https://github.com/rgomids/axiom/issues/245), including its
[approved Socratic Gate](https://github.com/rgomids/axiom/issues/245#issuecomment-6090871554),
owns this report-only observability contract. Parent #241 prohibits duplicate
post-merge product CI; #242 owns the unchanged required `go-quality` check.

The approved decisions map to these implementation boundaries:

1. D1: `verify (linux)` adds atomic coverage to its existing race test invocation.
2. D2: `scripts/code-health.py` normalizes statements, produces a Job Summary,
   and retains profile, snapshot JSON, summary JSON and Markdown in an artifact.
3. D3: `code-health-baseline.yml` handles main pushes using successful PR CI
   artifacts and Git tree equivalence, without running any Go tests.
4. D4: `scripts/codehealth` uses the standard Go AST to compare changed-function
   complexity against the validated snapshot; no third-party analyzer is needed.
5. D5: collection, comparison and retention are report-only. Existing mandatory
   tests/checks keep their failure semantics. No numeric threshold is enforced.

Scope is CI/scripts/tests and this guide. No product contract, dependency or
runtime boundary changes; no additional ADR is warranted. Constitution check:
approved behavior precedes implementation; deterministic tests, provenance,
bounded permissions and explicit unavailable states support the contract.
Rollback removes reporting steps/workflow and restores the Linux test command.

## Reading the evidence

The tested SHA is checkout HEAD: normally the PR synthetic merge SHA, which can
differ from the workflow run's head SHA. `snapshot.json` schema 1 includes that
SHA, Git tree, UTC collection time, measured scope, Go version, AST algorithm
version, package statement counts and function inventory. `summary.json` includes
baseline identity/provenance, changed packages, total counts and at most 15
complexity entries. Artifacts use `code-health-linux-<tested-sha>` and
`code-health-main-<main-sha>`, retained for 30 days. The comparable function
inventory is needed for future matching; the human summary is capped at 15
packages and 15 functions. There are no automatic PR comments.
Changed packages take precedence; critical packages are `project`, `projectapp`,
`executiongraph`, `graphapplication`, `workflow`, `runtimeadapter`,
`runtimeapplication`, `local`, `portableconfig`, `windowsfs` and `darwinacl`.

Coverage is statement-weighted, not the average of package percentages. The
inventory is `go list ./...` on Linux, with default per-package Go test
instrumentation. Duplicate blocks merge counters without double-counting
statements. Packages without recorded statements show unavailable, never 100%.
Collection refuses a non-Linux Go target or tracked source drift from HEAD.
No-test packages with recorded zero counters are uncovered. Tests that launch
separately built, uninstrumented CLI binaries do not add coverage for those
subprocesses; this report is not an end-to-end coverage claim. Platform-specific
code not selected on Linux is outside coverage scope.

Changed-package results map changed Go source/test paths from the PR base SHA
to the tested revision's current packages. Deleted packages are absent from
current metrics. This is package coverage, not changed-line coverage; true
line-level diff coverage is explicitly unsupported. A renamed/moved path is a
deletion plus addition; no historical function match is invented.

Complexity starts at one and adds one for each `if`, `for`, `range`, non-default
switch/select case, and `&&`/`||`. Literal function decisions are included in
their enclosing declaration. The signal covers tracked non-test `cmd/` and
`internal/` Go sources across all platforms; build constraints are not evaluated.
Identity is file + printed receiver type + function name. Matched changed
functions are reported only for increases; unmatched functions are current
hotspots, not proven regressions. Deleted functions are omitted. With no
baseline, the summary shows ranked current hotspots and no fabricated deltas.

## Baseline lifecycle and trust

Main pushes find a unique merged PR from this repository, require the PR head
tree to equal the main tree, then find a successful `ci.yml` PR/dispatch run.
The downloaded snapshot's tested SHA is resolved through the GitHub commit API;
its tree must equal both that tested commit's tree and main's tree. The promoted
snapshot associates metrics with main and records original tested SHA, source
run, PR and publication time. No baseline is promoted from failed CI, fork PRs
or unequal trees. API lookup is bounded to 30 associated PRs, 10 CI runs and
100 artifacts per run. In particular, the first merge adding this workflow may
have no successful metrics yet: the following successful measured PR supplies
the first baseline. An existing main dispatch alone is not automatically promoted.

PR comparison chooses the latest available successful main snapshot among at
most 10 successful push runs, not the merge-base. Schema, scope, SHA, counts,
function identities, UTC timestamp (at most 30 days old) and tool versions are
validated. Missing, expired, stale, incompatible, inaccessible or malformed
snapshots produce an unavailable comparison; current metrics remain visible.
This retained history is not a permanent metrics database. No external backend
or scheduled/full test execution is added.

Both workflows use read-only contents/actions permissions, pinned checkout,
setup-go and upload-artifact, and checkout without persisted credentials.
Fork code runs only in `pull_request`, without secrets or write permission;
artifact access failures degrade gracefully. Baseline promotion executes only
main's script, reads one fixed JSON ZIP member without extracting files, limits
download/member size to 8 MiB and never sends the token to artifact-storage
redirects. No provider responses, tokens or environment dumps enter artifacts.
Pin updates follow normal review; changing the AST algorithm requires a scope
or algorithm version change and new compatible baseline history.

## Reproduction and validation

From the repository on Linux, using the Go version in `go.mod`:

```bash
mkdir -p coverage
go test -race -covermode=atomic -coverprofile=coverage/coverage.out ./...
python3 scripts/code-health.py collect
python3 scripts/code-health.py report --base <PR-base-SHA>
python3 scripts/test-code-health.py
go test ./scripts/codehealth
```

Local reporting without the Actions token reports baseline unavailable. Fetching
an Actions baseline uses `GITHUB_REPOSITORY` and `GH_TOKEN` with actions read
access; do not persist credentials. Collection does not run the product suite;
it only reads the profile, lists packages and runs the AST collector.

Integration evidence still requires a real successful measured PR (summary and
artifacts with tested SHA), its merge (promoted main artifact with equivalent
tree/source run), and a following PR (baseline comparison). Verify missing/fork
baseline behavior and that a failing product test still fails `verify (linux)`.
Local fixture tests cannot establish GitHub artifact availability or post-merge
timing. Future regression enforcement requires separate approval and sufficient
trustworthy baseline history.
