# Native platform acceptance

## Current Server amendment

Explicit maintainer authorization expanded this PR to member Windows Server AMD64.
The reusable required `native-acceptance` now includes `windows-server-amd64` on
`windows-2022`, alongside Linux ARM64. Both consume exact prepared bytes and must
pass before preparation can be successful. Workstation AMD64 remains a separate
native coverage gap; Windows ARM64 and domain controllers remain unsupported.

Server covers fresh install, PATH discovery/provenance, DACL protection, real CLI
configuration/show/list/validate, reinstall/idempotency, invalid arguments, missing
projects, corrupt/missing configuration recovery, corrupt archive refusal, foreign
binary preservation, and junction destination refusal. Native existing installer
and bootstrap regressions now run positive Server paths, including fixture upgrade,
locked binary refusal, permission repair, default paths and both PowerShell versions.

A genuine prior-supported-release Server upgrade is N/A for this first supported
candidate; fixture upgrade does not substitute for historical release evidence.
The POSIX installer interruption hook is N/A to Windows; Windows refused-install
retry/recovery is covered by native regressions. Native execution results will be
recorded after qualification. The earlier investigation/evidence below is historical.

## Authority and design

Issue [#256](https://github.com/rgomids/axiom/issues/256), its
[approved Socratic Gate](https://github.com/rgomids/axiom/issues/256#issuecomment-6084847176),
parent [#241](https://github.com/rgomids/axiom/issues/241) and delivered
[#244](https://github.com/rgomids/axiom/issues/244) govern this boundary.
The five approved decisions require feasible full functional acceptance,
GitHub-hosted runners, strict release blocking and actual native Evidence.

The smallest design adds a read-only reusable native acceptance job to
`release-artifacts.yml`. It downloads the already-built candidate; there is no
build/toolchain in the native job. The existing Linux amd64 and macOS arm64
smokes remain intact. `verify-prepared-release.sh` already rejects any preparation
run that is not completed/successful, both before preview and before publication.
Acceptance cannot authorize publication or obtain publication credentials.
The maintainer subsequently authorized member Server AMD64 support in the same PR.
[ADR-0021](../decisions/0021-windows-server-amd64-native-acceptance.md) records that
bounded amendment; local NTFS and security boundaries remain intact.

## Initial investigation and feasibility (before Server authorization)

| Platform/scenario | Feasibility / activation | Evidence and rationale |
| --- | --- | --- |
| Linux ARM64 infrastructure | VERIFIED; activate automated native lifecycle | [Hosted probe](https://github.com/rgomids/axiom/actions/runs/37958968405), job `113916618470`: Ubuntu 24.04.5, image `ubuntu-24.04-arm` version `20261004.142.1`, actual `uname -m=aarch64` |
| Linux ARM64 install/configure/upgrade/reinstall/errors/recovery | VERIFIED in non-publishing qualification | [Native job](https://github.com/rgomids/axiom/actions/runs/37961209760/job/113924616021): all 14 scenarios passed on `aarch64`; candidate archive and genuine published v0.10.0 baseline |
| Linux AMD64 | Existing #244 native smoke preserved | `prepare` on `ubuntu-24.04`; no added full-release upgrade matrix |
| macOS ARM64 | Existing #244 native smoke preserved | `smoke-macos` on `macos-15`; legacy `macos-27` asset naming is not an OS version bound |
| Windows AMD64 client lifecycle | BLOCKED; not activated | Standard hosted amd64 Windows runners are Server; client runner `windows-11-arm` is ARM64. Both violate the supported client/amd64 conjunction. `scripts/install.ps1` and `internal/install/space_windows.go` refuse Server |
| Windows Server / Windows ARM64 acceptance | NOT APPLICABLE | Outside approved product support; Server regression tests remain useful but are not client acceptance |
| Additional Linux distributions, macOS releases, unusual filesystems | Residual risk; not activated | No new incident or concrete gap justifies further rows in this issue |

[GitHub's runner reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)
lists the native ARM64 label and hosted Windows OS/architecture combinations.
Repository API confirmed public visibility and Actions execution was demonstrated
by the probe. Standard hosted runner execution is free for public repositories;
normal concurrency/usage limits still apply. The new native release job is bounded
to 15 minutes, qualification build to 20 minutes, subprocesses to 60 seconds,
downloads to 120 seconds and diagnostic retention to 30 days. No larger runner,
paid service, new secret, admin setting or self-hosted runner is used.
The Actions-policy API returned HTTP 403, but actual probe execution establishes
eligibility without changing policy. This does not prove future quota availability:
once activated, infrastructure failure fails preparation closed.

GitHub also documents a
[Base Windows 11 Desktop larger-runner image](https://docs.github.com/en/actions/reference/runners/larger-runners).
This is not an available standard label for this repository: the API identifies
`rgomids` as a **User**, while
[larger runners require an organization/enterprise on Team or Enterprise Cloud](https://docs.github.com/en/actions/concepts/runners/larger-runners).
They are [paid even for public repositories](https://docs.github.com/en/billing/reference/actions-runner-pricing).
No eligible existing client/amd64 pool was established, and changing account,
billing or infrastructure provisioning is outside the authorized scope.

## Functional contract and traceability

The suite runs the candidate's own verified `install.sh` facade. v0.10.0 is a
genuine release with a Linux ARM64 distribution, published checksums and the
supported install/upgrade protocol. It is a fixed, reproducible supported baseline,
not a rebuild or invented historic version. Candidates must upgrade from it;
historical candidates at or below that baseline are outside this new contract.
The release downloader is isolated from CLI execution. CLI commands inherit a
closed environment with temporary HOME/state/projects/skills and no credentials
or discovered Providers/Runtimes. Repository paths and HOME include spaces.

| Requirement | Platform | Scenario / workflow | Evidence status |
| --- | --- | --- | --- |
| Fresh install and executable discovery | Linux ARM64 | `fresh-install`, `discovery-provenance`, `private-permissions` | PASS; [native job](https://github.com/rgomids/axiom/actions/runs/37961209760/job/113924616021) |
| CLI and configuration initialization/loading | Linux ARM64 | `configuration-cli`; real `project configure` preview/apply, show/list/validate | PASS; [native job](https://github.com/rgomids/axiom/actions/runs/37961209760/job/113924616021) |
| Genuine prior upgrade and persistent-state compatibility | Linux ARM64 | `genuine-prior-upgrade-state-reinstall`; previous CLI creates Project/state, failed corrupt upgrade preserves state, candidate installer upgrades, hashes remain unchanged, compatibility inspect | PASS; [native job](https://github.com/rgomids/axiom/actions/runs/37961209760/job/113924616021) |
| Reinstall/idempotence | Linux ARM64 | `reinstall-idempotent` and upgrade reinstall; binary/receipt/state digests preserved, no duplicate bin entry | PASS; [native job](https://github.com/rgomids/axiom/actions/runs/37961209760/job/113924616021) |
| Invalid CLI/configuration and recovery | Linux ARM64 | `invalid-arguments`, `missing-project`, `malformed-config-recovery`, `missing-config-recovery`, `install-interruption-recovery` | PASS; [native job](https://github.com/rgomids/axiom/actions/runs/37961209760/job/113924616021) |
| Platform filesystem/security behavior | Linux ARM64 | spaces, 0700 binary/0600 receipt, `foreign-binary-preserved`, `symlink-destination-refused` | PASS; [native job](https://github.com/rgomids/axiom/actions/runs/37961209760/job/113924616021) |
| Negative artifact and functional controls | Linux ARM64 | `corrupt-artifact-refused`; #244 wrong provenance/non-executable/lifecycle failure/timeout/mutation tests; qualification `rejection` job | PASS; [native](https://github.com/rgomids/axiom/actions/runs/37961209760/job/113924616021) and [deliberate rejection](https://github.com/rgomids/axiom/actions/runs/37961209760/job/113924615975) |
| Strict release dependency | Enabled ARM64 | `native-acceptance` required preparation job; `test-release-flow.sh` failed preparation/authorized publication refusal; qualification rejection → skipped promotion | PASS; [propagation assertion](https://github.com/rgomids/axiom/actions/runs/37961209760/job/113925949826) and [release contracts](https://github.com/rgomids/axiom/actions/runs/37961209760/job/113924213282) |
| Candidate identity | Enabled ARM64 | #244 smoke + native before/after inventories, binary/installed provenance, checksums for candidate and baseline | PASS; [native report](native-platform-acceptance-evidence.json) |
| Linux AMD64/macOS ARM64 regression | Existing rows | Original #244 jobs, `test-release-pipeline.sh`, `test-release-flow.sh` | Both native smokes PASS in [qualification](https://github.com/rgomids/axiom/actions/runs/37961209760); release contracts recorded below |
| Windows client lifecycle | Windows AMD64 | No eligible hosted runner | BLOCKED; no pending required job or simulated PASS |

Installer failures, nonzero/unexpected CLI statuses, invalid artifacts, changed
inputs and timeouts all fail acceptance. The supported first-install pre-binary
fault seam verifies that no binary was committed and a normal retry succeeds.
Malformed Project configuration is refused without rewriting it; restoring the
original bytes permits normal loading. This is guided recovery of controlled
temporary state, not a claim of power-loss durability or every filesystem case.

## Reproduction and evidence

On native Linux ARM64 (use `linux-amd64` for local x86_64 diagnostics):

```bash
python3 scripts/accept-native-artifact.py --dir /absolute/prepared/artifacts \
  --previous /absolute/v0.10.0-assets --version VERSION --revision FULL_SHA \
  --row linux-arm64 --log /absolute/diagnostics/native.log > native.json
python3 scripts/test-native-acceptance.py
./scripts/test-release-pipeline.sh
./scripts/test-release-flow.sh
```

`native.json` has schema `axiom-native-acceptance/v1`, candidate version/full
revision, archive and binary digests, baseline identity, actual OS/architecture,
runner image/version, run URL/job, individual expected/observed scenario outcomes
and input preservation. Logs are retained separately, never publication assets
or envelope inputs. If infrastructure fails before the suite starts, the job
fails and raw Actions logs diagnose the missing report; no PASS is fabricated.

The manual, non-publishing `native-acceptance-qualification.yml` builds once,
transfers the same candidate to successful acceptance and deliberate corruption
rejection jobs. `promotion` requires rejection and must be skipped; the
`verify-propagation` observer asserts success/failure/skipped results. The
qualification run is intentionally red because the rejection remains a real
failed required job; no ignored exit/continue-on-error masks it. This observer
does not run in release preparation and cannot waive the release gate.

Rollback removes the new required preparation job after a reviewed decision;
there is no data migration or change to prepared/publication bytes. Do not waive
an activated failing row by marking it unavailable within a release run.

## Open acceptance and follow-up

Record qualification run URL, tested digest, native scenario outcomes and strict
gate proof before marking a row VERIFIED. A green infrastructure probe alone is
not product acceptance. Full release-preparation dispatch requires a reviewed
revision on main; branch qualification does not authorize release readiness.
Keep #256 open unless all applicable approved criteria have Evidence and closure
is explicitly authorized. Future follow-up: activate Windows AMD64 client only
when an eligible GitHub-hosted client/amd64 runner exists, and reconsider the
fixed baseline when its compatibility/support contract changes.

## Engineering report / Evidence snapshot — 2026-10-09

Final qualification [run 37961209760](https://github.com/rgomids/axiom/actions/runs/37961209760)
tested revision `e8157bc30383bc72ab27b934596dc2fb9d689d5f`, version
`9999.0.0-acceptance`, using the normal release archive builder with clean,
release-marked provenance. These are real prepared distribution bytes built
once and transferred to native execution; the qualification version is never
published. Later delivery changes only reconcile this Evidence.

| Job | Actual host | Observed result |
| --- | --- | --- |
| `prepare` / `113924213434` | `ubuntu-24.04`, x86_64 | PASS: build, Python contracts (10 smoke + 2 native tests), exact Linux AMD64 smoke |
| `acceptance / linux-arm64` / `113924616021` | `ubuntu-24.04-arm`, aarch64; image `20261004.142.1`, kernel `6.17.0-1022-azure`, glibc 2.39 | PASS: all 14 lifecycle scenarios, candidate and baseline inputs unchanged |
| `smoke-macos` / `113924615652` | `macos-15`, arm64; macOS 15.7.9, image `20260907.0337.1` | PASS: exact-byte provenance, Project lifecycle, unchanged inputs |
| `rejection / linux-arm64` / `113924615975` | `ubuntu-24.04-arm`, aarch64 | FAIL as required: `prepared archive checksum mismatch` after deliberate corruption |
| `promotion` / `113924692019` | No runner allocated | SKIPPED automatically because required rejection failed |
| `release-contract` / `113924213282` | `ubuntu-24.04`, x86_64 | PASS: `test-release-pipeline.sh`, `test-release-flow.sh`, `validate-repository.sh .` |
| `verify-propagation` / `113925949826` | `ubuntu-24.04` | PASS: successful acceptance/macOS/contracts, real failed checksum rejection, skipped promotion |

The overall qualification conclusion is **failure by design**, not PASS:
there is no masked rejection exit. This is safe negative-control Evidence,
not a publishable preparation run. Release-flow regressions separately observed
`next_action=blocked`, refusal even with explicit publication authority and
zero release effects for a failed preparation run. Production preparation calls
the same reusable native job without the corruption control; publication's
existing successful-preparation-run check remains enforced. A live dispatch
of the modified production preparation workflow is still unverified because
the implementation is on a feature branch, not an approved revision on main.

| Subject | SHA-256 |
| --- | --- |
| Linux ARM64 candidate archive | `d5dfe8c2353d7be25f687ac7df351195c72c89c0001fff9d8af863bca7f5f22d` |
| Linux ARM64 candidate binary / installed binary | `5a9180f56c3baa66c9e3ba5f8f4b06d4ec6d36a9fe01bf0ee653ce1ed8b47292` |
| Genuine v0.10.0 baseline archive | `905b3f729a824d0c2e3530d638683209e1cc5d0011347b4c7182a518d5c854ba` |
| Genuine v0.10.0 baseline binary, revision `c7260797aad7` | `9ee3aec3106b96cb86b44cc902cc19d121727fb92a40005f0d62482b0915be20` |
| macOS ARM64 candidate archive | `4fb656c6d2df69512db700ff95a1c575d61fc24527356ed6f5e047f762d6b10a` |

The sanitized [native report](native-platform-acceptance-evidence.json) preserves
every scenario's observed result and exact tested identity. Diagnostic logs and
macOS/rejection reports are available as 30-day artifacts of the run; job logs
provide the raw execution evidence. They contain only synthetic test state.

Changed surfaces: three release/qualification workflows, the CI contract step,
one native acceptance script, #244's ARM64 smoke adapter, native/release-flow/
release-pipeline contracts, the automation registry and this guide/report plus
the reconciled #244 smoke guide. No product Go source or production support
contract changes. Local native-contract tests, Python/Bash syntax, registry
coverage (81 surfaces), diff whitespace and staged sensitive-file checks passed.
The existing smoke unit suite has one OS-dependent failure when run on Windows
(POSIX executable fixture); all ten tests passed on hosted Linux. Local Go was
not on PATH, local repository validation stopped at Windows symlink checkout
representation, and `gitleaks` was unavailable. Hosted repository validation and
release builds passed; no unsupported local check is reported as PASS.

### Approved acceptance criteria audit

| Gate acceptance criterion | Evaluation |
| --- | --- |
| Investigation of repository/#244/platform/CI boundaries | Satisfied; source and approved gate inspected, boundaries preserved |
| Evidence-based feasibility/activation matrix | Satisfied; real ARM64 eligibility probe, public/user-owned account, Windows and paid-runner gap recorded |
| Full applicable feasible native lifecycle | Satisfied for activated Linux ARM64; 14 real native scenarios; Windows client remains BLOCKED |
| Real hosted execution for VERIFIED rows | Satisfied; job identities and observed architecture above |
| Strict failure propagation/publication blocking | Satisfied in safe non-publishing qualification and release-flow refusal contracts; live production dispatch remains unverified |
| Immutable candidate identity | Satisfied; digest-bound archive/binary, installed-byte match, input inventories unchanged |
| Existing #244/platform coverage preserved | Satisfied; both original native smokes and release contracts passed |
| Positive/negative cases and diagnostics | Satisfied; retained reports/logs, real failed corruption control and recovery cases |
| Traceability and residual risks | Satisfied; requirement/scenario/run mapping above and Windows gap explicit |
| No forbidden infrastructure/manual gates/support expansion | Satisfied; standard hosted runners, read-only tokens, automatic required release job; qualification dispatch is a developer check, not routine release authority |

Review found no remaining blocking code/security finding for the feasible
Linux implementation: verified extraction, isolated state, bounded processes,
unchanged publication credentials/authority and no unrelated changes.
**Overall handoff: PARTIALLY COMPLETE** — feasible native implementation and
qualification are verified; Windows AMD64 client execution is blocked and live
main preparation integration awaits normal review/merge. No release readiness,
merge, publication or Issue closure is claimed or authorized.
