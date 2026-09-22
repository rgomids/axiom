# Evidence — MVP Slice S3: Intent to GitHub Work Item

## Claim and authority boundary

This record covers the explicitly authorized Specification 004 Slice S3 only:
T08–T09. Work started from `main` revision
`9c16322` (the S2 PR #84 merge). S4–S7, workflow execution/projection, issue
comments/labels/closure, repository or Git mutation, release publication, and
final MVP acceptance were not authorized and were not performed.

The results below establish technical implementation Evidence. They do not imply
human acceptance. A real GitHub observation requires separate exact per-run
authority plus reviewed public-safe content. That authority was not supplied, so
this run created no real Issue and does not claim the real-provider observation.

## Delivered behavior

### T08 — read-only structured Intent draft

- Intent or an explicit problem is normalized with desired outcome, context,
  scope, constraints, non-goals, and acceptance expectations into a
  provider-neutral draft.
- Every section retains `user` or `axiom` authorship. Missing questions are
  deterministic and limited to materially absent sections.
- Preview binds Project/repository/provider target, normalized draft, rendered
  provider document, exact effect set, expected local revision, correlation,
  provenance, and digest. Preparing it performs no Provider or local write.
- UTF-8/control, per-field and aggregate size, and structural secret-sentinel
  checks occur before rendering or effects. Provider rendering indents section
  content, preserving Markdown and HTML-like input as authored data rather than
  Issue structure.
- Guided CLI prompts only missing values, prints the complete preview, and treats
  EOF or any answer other than exact `yes` as cancellation without authority.

### T09 — exact create/select authority and truthful linkage

- Create requires the exact reviewed preview digest plus
  `--authorize-external`. Selection first reads and validates the exact Issue,
  then requires its separately reviewed digest plus `--authorize-local` for the
  protected local publication.
- The application port uses provider-neutral resource/external identity. GitHub
  Issue number, URL, state, request paths, and rendering remain in the
  `internal/githubissues` adapter.
- The adapter invokes an absolute `gh` executable with a 15-second deadline and
  256 KiB captured-output limit. Untrusted title/body are JSON on stdin, never
  shell-interpolated or placed in arguments. Responses must match the exact
  `https://github.com/<owner>/<repository>/issues/<number>` identity and
  `OPEN|CLOSED` state.
- Before create, and again after an ambiguous result, application code searches
  for the exact deterministic correlation marker. Zero matches permits one
  create; one valid match is reused; multiple/invalid matches fail closed. No
  blind create retry exists.
- A confirmed Provider effect followed by local read/write/conflict/recovery
  failure is canonical `partial` and retains the exact external reference.
  Committed-but-uncertain protected publication also remains truthful partial.

## Deterministic observations

| Condition | Test seam | Observed truth |
|---|---|---|
| incomplete or cancelled interview | application/CLI spies | ordered missing questions or `interrupted`; zero Provider effects |
| stale/absent create authority | exact preview digest test | `denied_authority`; zero create and local saves |
| stored identity differs from requested path | protected store decode test | unsafe record rejected before use |
| metacharacter/Markdown input | fake executable argument/stdin capture | content only in JSON stdin and indented authored sections |
| rate limit, timeout, oversized or mismatched response | bounded adapter fake | typed retry/failure boundary; invalid content rejected |
| ambiguous create | sequenced reconciliation fake | second reconciliation before result; no blind retry |
| existing/multiple correlation match | reconciliation ledger | one valid Issue reused; multiple matches fail before create |
| local revision changed after select preview | store revision fault | stale digest denied; zero save |
| confirmed Issue plus local failure | store fault injection | canonical `partial` with exact Issue reference |
| executable create/select | isolated fake `gh` black box | preview has zero mutation; one authorized create/select link |

## Validation environment and commands

Implementation and tests ran on macOS 27.0/arm64 with Go 1.26.1. The final
implementation snapshot is commit `e0bd2f1` (`feat: implement MVP S3 intent to
work item`). The branch validation commands are:

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
gitleaks detect --source . --no-git
git diff --check
```

The Go suites include pure unit tests, application/store integration, adapter
process-boundary tests, CLI tests, and installed executable-style black-box
tests. Every command above exited zero; Gitleaks reported no leaks. Provider
traffic in these tests uses synthetic deterministic fake
executables and temporary protected roots; no real GitHub mutation occurs.

## Limitations and pending gate

- The mandatory one bounded real GitHub observation remains pending exact
  per-run human authority, reviewed target/content/effects, cleanup ownership,
  and a public-safe test payload. It must not be inferred from this implementation
  request, tests, PR creation, review, or merge.
- Historical POC comment/complete and workflow code remain compatibility inputs;
  they are not S3 completion behavior and were not advanced.
- GitHub Issues is the only implemented Work Item capability. The application
  boundary is provider-neutral; a generic provider framework or second Provider
  remains explicitly out of scope.

## Acceptance status

S3 T08–T09 are technically implemented and ready for human review, except for the
separately gated real-provider observation recorded above. Human acceptance is not
claimed. No authority exists here to start T10 or any S4–S7 Task.
