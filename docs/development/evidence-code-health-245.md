# Issue #245 validation evidence

Local validation on 2026-10-09, Windows AMD64, Go 1.26.0 and Python 3.12.13.
Implementation follows the [approved issue comment](https://github.com/rgomids/axiom/issues/245#issuecomment-6090871554).
The [code health guide](code-health.md) records the contract, implementation
plan, reproducible commands, trust boundary and integration acceptance steps.

## Executed checks

| Command | Observed result |
|---|---|
| `python scripts/test-code-health.py` | 11 offline tests pass: profile parsing/normalization, counts, schema, freshness, changed-package mapping, deterministic top limit, formatting, rename/deletion, ZIP validation, missing/expired/permission paths, successful tree-equivalent promotion and refusal of forks/failed runs/wrong trees |
| `go test ./scripts/codehealth` | pass: AST decisions, method/generic identity, empty function and invalid source |
| `go test -covermode=atomic -coverprofile=<temporary-profile> ./scripts/codehealth` | pass; real Go atomic text profile generated (narrow collector tests only, not a product coverage baseline) |
| `go run ./scripts/codehealth` | pass; repository product function inventory generated in a temporary file |
| `go vet ./...` | pass |
| `go build ./...` | pass |
| `go mod verify` | all modules verified |
| `./scripts/check-go-quality.sh format` | pass |
| `./scripts/check-go-quality.sh tidy` | pass, no module changes |
| `python scripts/check-automation-registry.py` | pass, 90 governed surfaces |
| `git diff --check` | pass |
| `./scripts/check-sensitive-files.sh .` | pass |

The full `go test ./... -timeout 10m` run failed in existing Windows filesystem
and runtime tests. Diagnostics include refusal of the local `AppData` ancestor
ACL, portable Project publication failure, unavailable runtime skill roots and
an existing runtimeadapter fixture panic. No product files were changed by
this implementation. These results do not establish a passing Windows suite.

`./scripts/validate-repository.sh .` stopped at the existing Claude adapter
symlink contract: this Windows checkout materialized the symlink as a copy.
`./scripts/test-release-flow.sh` passed its early release fixtures but stopped
when delivery resolution required unavailable `jq`. Neither check is claimed
to have passed. Staticcheck was not locally executed; its existing pinned
mandatory CI step is preserved.

## Engineering and security review

No blocking finding identified in the scoped diff. Test execution remains
mandatory, with Linux atomic coverage added to the single existing race run.
macOS/Windows commands and required check identities remain. New metric
collection/comparison/upload failures are informational and explicitly reported.
No numeric coverage/complexity threshold, PR comment, new dependency, privileged
PR event, credential persistence or product behavior change was introduced.

Artifact provenance is verified against successful CI and Git trees rather than
branch names alone. Baseline downloads are read-only, bounded, validated JSON;
ZIP members are never extracted. Tokens are absent from outputs and withheld
from storage redirects. Actions use full SHA pins and explicit 30-day retention.
Tool versions and measurement scope gate compatibility.

## Remaining integration evidence

Linux race+coverage, actual Actions summaries/uploads and trusted main promotion
cannot be proven by the local Windows checks. The implementation PR must show
the Linux artifact/summary at its tested SHA. After a successful measured PR
merges, inspect the main baseline artifact for matching main SHA/tree and source
run; then inspect a following PR comparison. Missing baseline must remain
unavailable, and existing failing tests must remain failing checks. Fork metrics
may be visible while comparison is unavailable; fork snapshots cannot become
main baselines. No main baseline or real PR comparison is claimed here.
