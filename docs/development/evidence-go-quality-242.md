# Issue #242: baseline and validation Evidence

## Specification and implementation scope

The authorized outcome is a cheap blocking PR structural gate, preserving all
existing product/repository/release checks. The acceptance contract is Issue
[#242](https://github.com/rgomids/axiom/issues/242); no coverage/complexity
threshold, mutation testing, change-aware skipping, Runtime calls or manual
release-candidate Evidence is introduced.

Implementation: independent bounded `go-quality` job; read-only formatting and
module checks; pinned curated analyzer; real-tool negative regression fixtures;
versioned required context and release-fixture integration; automation registry
and developer reproduction reference. This is CI policy, not a product
architecture boundary change; no ADR is required.

## Baseline provenance

Baseline: current `origin/main` at
`34103e5cfc1891f45caff71045d7c245f9301aad` (fetched on 2026-10-08). Go 1.26.0 linux/amd64;
Staticcheck 2026.2.1 (0.8.1). Commands:

```bash
go mod tidy -diff
staticcheck -checks=all ./...
```

Module tidiness passed. Full analysis reported 26 diagnostics: 15 ST1005,
ST1020 or ST1021 style findings (outside selected scope) and the following 11
selected findings. Original line numbers refer to the baseline revision.

| Baseline location | Check | Resolution |
| --- | --- | --- |
| cmd/lingo/blackbox_test.go:632 | SA4006 | Remove final revision increment with no subsequent reader. |
| cmd/lingo/execution_lifecycle_test.go:79 | SA4006 | Assert that listing preserves the captured seeded state before adding another record. |
| internal/local/validation.go:21 | SA4004 | Explicitly inspect the first identity issue; preserve existing first-error behavior. |
| internal/manifest/parser_contract_test.go:58 | SA4000 | Decode both stream documents explicitly, then assert EOF. |
| internal/codexruntime/upgrade.go:605 | U1000 | Remove unreferenced syncPath. |
| internal/install/domain_skills_upgrade_test.go:70 | U1000 | Remove unused changedCompatibilitySkills test data. |
| internal/install/upgrade.go:1028 | U1000 | Remove unreferenced readMarker wrapper; retain anchored readMarkerIn. |
| internal/install/upgrade.go:1169 | U1000 | Remove unreferenced syncDirectory. |
| internal/local/graph_store.go:188 | U1000 | Remove unreferenced graphStoreError. |
| internal/local/portable_store.go:249 | U1000 | Remove unreferenced PortableStore.clear. |
| internal/local/safe_fs.go:725 | U1000 | Remove unreferenced clearAttempt wrapper; retain removeProtocolState. |

No selected diagnostic is suppressed. Removal was checked against repository
references and platform-specific sources. The stream test and listing test
keep their original operations and strengthen diagnostic/assertion clarity.

Additional Windows analysis identified U1000 on `upgradeResumeNext`, whose
only caller is in `install_posix.go`. Move the helper unchanged from the
common file to that POSIX file, preserving behavior and making its platform
scope explicit rather than suppressing it.

## Validation

The final source was checked on Linux (native WSL checkout, preserving Git
modes and symlinks) and native Windows with Go 1.26.0. Results:

| Command / contract | Result |
| --- | --- |
| `./scripts/check-go-quality.sh` | PASS: formatting, module diff, selected analyzer; zero suppressions. |
| `python3 scripts/test-go-quality.py` | PASS: 10 tests; intentional format, syntax, tidiness, SA1000 and U1000 failures; clean/style-only success; wrong analyzer refused; suppression and workflow policy. |
| `./scripts/validate-repository.sh .` | PASS: harness, registry, ADR, sensitive-file and diff validation. |
| `go test -race ./...` (Linux) | PASS; final run 4m17.8s on this local host. |
| `go test ./... -timeout 10m` (native Windows) | PASS using a private temporary directory. |
| `go vet ./...`, `go build ./...`, `go mod verify` | PASS. |
| `GOOS=windows staticcheck -checks='SA1*,SA2*,SA3*,SA4*,SA5*,SA9*,U1000,-SA1019' ./...` | PASS after POSIX helper relocation. |
| `python3 scripts/test-release-pr-checks.py` | PASS: 14 tests. |
| `python3 scripts/test-release-corrections.py` | PASS: 12 tests. |
| `git diff --cached --check`, `check-sensitive-files.sh --staged .` | PASS. |

Negative fixtures run in isolated temporary Git/Go modules. Formatting and
tidiness failures preserve the input bytes. Analyzer failures show a source
location and check ID. The release contract also tests that missing, failed
or foreign-integration `go-quality` cannot authorize a repair.

An illustrative warm-cache local gate run took 1.75s and the nine initial
regressions took 3.43s; the final ten-test suite took 4.67s. These are local
measurements, not a claim about GitHub runner wall-clock impact. The new job
is independent and bounded to ten minutes, with Go setup/cache reused through
the established Actions configuration.

The first Windows run failed because the default temporary path under
AppData permits ancestor replacement by an untrusted principal. Re-running
with the documented private-directory procedure passed; no existing ACL or
security contract was relaxed. Initial Linux copy attempts altered modes and
symlinks; their repository/release results were excluded and validation was
repeated in a native checkout with only changed paths overlaid.

Full macOS tests, hosted release/upgrade jobs and hosted required-check success
remain GitHub CI evidence. The 2026-10-08 administrative read-back of ruleset
22828068 confirmed all seven existing required contexts and no `go-quality`.
Applying the eighth context after merge requires explicit administrative
authorization and a new read-back. No merge, publication, Issue closure or
external acceptance is asserted by these local checks.
