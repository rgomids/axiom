# Issue #136 — classification validation evidence

Validated locally on 2026-10-02. Scope:
[Work Item classification contract](work-item-classification.md).

## Environment and reproducibility

Ubuntu 26.04 under WSL, Go 1.26.0, Linux amd64, GCC with `CGO_ENABLED=1`.
The Go archive was verified against its official SHA-256 before extraction.
Validation used an isolated local clone at base revision
`8071c9b780198b3b144f6baeb38d8f95d3dd93b7`, overlaid with this change's
canonical LF source bytes. This avoids Windows CRLF changing embedded skill
and fixture hashes. The initial CRLF-related failures were not bypassed by
weakening the historical digest checks; the replaced canonical skill set was
registered in shared history and the new manifest pinned.

Run from a Linux checkout with Go 1.26+ and a C compiler:

```bash
CGO_ENABLED=1 go test -race ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
./scripts/dogfood-poc.sh
./scripts/test-release-pipeline.sh
./scripts/test-release-flow.sh
git diff --check
```

All commands passed. The full race suite covered every Go package. After the
final confirmed-item reuse correction, race tests were rerun for Work Item,
GitHub adapter, CLI, runtime skills, and the executable CLI; static/build,
repository checks and the bounded smoke test were also rerun successfully.
The smoke test reported `axiom_e2e_dogfood` with `result: pass`.

## Acceptance and regression checks

| Criterion | Result and evidence |
|---|---|
| Explicit type on every created item | Passed: supported types, default task, invalid type rejection, typed provider body, actual executable bug creation |
| Story delivery value | Passed: beneficiary/value required separately from implementation sections; guided collection, preserved authorship, rendered value; skill requires concrete benefit or task classification |
| Proposed labels visible before mutation | Passed: executable read-only preview contains selected labels and type with zero mutations; metadata/type changes invalidate authority digest |
| Existing/authorized provider labels only | Passed: existing-catalog selection, explicit unknown rejection, deduplication, paginated lookup, removed-label rejection before POST, safe stdin payload |
| Explicit unsupported/degraded outcomes | Passed: absent inferred labels emit notice; dropped labels or unavailable verification yield partial result with existing Issue reference |
| Adapter boundary | Reviewed: only GitHub adapter interprets label catalogs, concrete metadata fields and applied labels; core uses neutral classification intent and opaque reviewed document |
| No duplicate during recovery/metadata change | Passed: confirmed retries reverify, inferred metadata changes reuse confirmed item, reconciled verification failure preserves reference without POST |
| Runtime installation compatibility | Passed: published shared skill history remains owned; historical POC fixture and both runtime integration tests pass |
| Security and repository hygiene | Passed: input text/secret limits, strict metadata fields, malformed catalogs rejected; sensitive-file checks and final diff review |

No blocking engineering finding remains from the local review. The change adds
no dependency and preserves existing authorization, attempt fences and local
state formats. No new ADR is required: the adapter-owned translation implements
the existing provider-boundary constraint.

## Limits and acceptance authority

GitHub transport behavior was tested through deterministic fake `gh` processes
and an executable CLI integration. No live Issue or label was created, and real
account permissions were not tested. macOS execution and remote CI remain
unverified locally. Natural-language story value is presented for human review;
required fields are deterministic checks, not proof that arbitrary prose
communicates good product value.

These results support technical review of the five issue criteria. They do not
declare maintainer acceptance, close the issue, merge, or publish a release.

## PR integration on current main

For PR preparation, the task-only commit was applied to current main
`db8ed3a` in a separate checkout. The earlier installer commits are excluded
from this PR. Current main's six-skill runtime set and every published historical
revision are preserved; the replacement six-skill manifest is pinned separately.
The full race suite, static/build/dependency checks, repository validation,
bounded smoke test and both release-contract scripts were repeated on this base.
The smoke test reported six installed skills and `result: pass`.
