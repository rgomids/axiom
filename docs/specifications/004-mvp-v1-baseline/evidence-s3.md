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
- The provider-neutral domain link maps at the local adapter boundary to the
  existing Work Item `formatVersion: 1` fields `providerRepository` and `number`.
  Existing v1 records remain readable; no v2 schema or migration exists. The v1
  adapter rejects non-`github`, non-positive, non-numeric, or non-canonical Issue
  IDs before creating any local state.
- `gh api --include` supplies bounded response status and headers. `401` is
  unauthenticated/non-retryable; `429` and structured rate-limited `403` are
  retryable; `5xx` is unavailable/retryable; ordinary `403` and deterministic
  `400`/`404`/`410`/`422` failures are non-retryable. Missing reliable HTTP
  metadata fails closed without retry. Create timeout and retryable unavailable
  results remain ambiguous and require reconciliation; no blind create retry was
  added.

## Deterministic observations

| Condition | Test seam | Observed truth |
|---|---|---|
| incomplete or cancelled interview | application/CLI spies | ordered missing questions or `interrupted`; zero Provider effects |
| stale/absent create authority | exact preview digest test | `denied_authority`; zero create and local saves |
| existing or newly written v1 Work Item | protected store codec tests | old wire fields retained and mapped to provider-neutral domain identity |
| v1-unrepresentable provider/external ID | pre-publication adapter validation | unsafe input rejected before state-root creation |
| stored identity differs from requested path | protected store decode test | unsafe record rejected before use |
| metacharacter/Markdown input | fake executable argument/stdin capture | content only in JSON stdin and indented authored sections |
| 401/403/404/422/429/500/503 | included-status adapter matrix | only explicit rate-limit and 5xx cases are retryable |
| timeout, oversized, unstructured or mismatched response | bounded adapter fake | timeout ambiguous; other unknown/invalid content fails closed without retry |
| ambiguous create | sequenced reconciliation fake | second reconciliation before result; no blind retry |
| existing/multiple correlation match | reconciliation ledger | one valid Issue reused; multiple matches fail before create |
| local revision changed after select preview | store revision fault | stale digest denied; zero save |
| confirmed Issue plus local failure | store fault injection | canonical `partial` with exact Issue reference |
| executable create/select | isolated fake `gh` black box | preview has zero mutation; one authorized create/select link |

## Validation environment and commands

Implementation and tests ran on macOS 27.0/arm64 with Go 1.26.1. Review
remediation was validated in the working tree based on commit `757b894`; no
commit or push is claimed by this Evidence update. The validation commands are:

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
./scripts/check-sensitive-files.sh --staged .
./scripts/dogfood-poc.sh
gitleaks detect --source . --no-git
gitleaks git --staged --redact --no-banner
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

T08 implementation and deterministic Evidence are complete. T09 implementation
and deterministic tests are complete, but T09 and S3 are not fully validated:
the separately gated real-provider observation remains pending. Human acceptance
is not claimed. No authority exists here to start T10 or any S4–S7 Task.
