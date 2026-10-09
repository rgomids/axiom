# Native platform acceptance

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
No production behavior, dependency or support commitment changes; no new ADR
is needed. ADR-0005 filesystem confinement, ADR-0010 Windows client/amd64 bounds,
ADR-0015 host eligibility and constitution sections II–VII remain intact.

## Investigation and feasibility (2026-10-09)

| Platform/scenario | Feasibility / activation | Evidence and rationale |
| --- | --- | --- |
| Linux ARM64 infrastructure | VERIFIED; activate automated native lifecycle | [Hosted probe](https://github.com/rgomids/axiom/actions/runs/37958968405), job `113916618470`: Ubuntu 24.04.5, image `ubuntu-24.04-arm` version `20261004.142.1`, actual `uname -m=aarch64` |
| Linux ARM64 install/configure/upgrade/reinstall/errors/recovery | IMPLEMENTED / UNVERIFIED pending qualification | `accept-native-artifact.py`; candidate archive and genuine published v0.10.0 baseline; no emulation |
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
| Fresh install and executable discovery | Linux ARM64 | `fresh-install`, `discovery-provenance`, `private-permissions` | IMPLEMENTED / UNVERIFIED |
| CLI and configuration initialization/loading | Linux ARM64 | `configuration-cli`; real `project configure` preview/apply, show/list | IMPLEMENTED / UNVERIFIED |
| Genuine prior upgrade and persistent-state compatibility | Linux ARM64 | `genuine-prior-upgrade-state-reinstall`; previous CLI creates Project/state, candidate installer upgrades, hashes remain unchanged, compatibility inspect | IMPLEMENTED / UNVERIFIED |
| Reinstall/idempotence | Linux ARM64 | `reinstall-idempotent` and upgrade reinstall; binary/receipt/state digests preserved | IMPLEMENTED / UNVERIFIED |
| Invalid CLI/configuration and recovery | Linux ARM64 | `invalid-arguments`, `missing-project`, `malformed-config-recovery`, `install-interruption-recovery` | IMPLEMENTED / UNVERIFIED |
| Platform filesystem/security behavior | Linux ARM64 | spaces, 0700 binary/0600 receipt, `foreign-binary-preserved`, `symlink-destination-refused` | IMPLEMENTED / UNVERIFIED |
| Negative artifact and functional controls | Linux ARM64 | `corrupt-artifact-refused`; #244 wrong provenance/non-executable/lifecycle failure/timeout/mutation tests; qualification `rejection` job | IMPLEMENTED / UNVERIFIED |
| Strict release dependency | Enabled ARM64 | `native-linux-arm64` required preparation job; `test-release-flow.sh` failed preparation/authorized publication refusal; qualification rejection → skipped promotion | IMPLEMENTED / UNVERIFIED |
| Candidate identity | Enabled ARM64 | #244 smoke + native before/after inventories, binary/installed provenance, checksums for candidate and baseline | IMPLEMENTED / UNVERIFIED |
| Linux AMD64/macOS ARM64 regression | Existing rows | Original #244 jobs, `test-release-pipeline.sh`, `test-release-flow.sh` | Preserved; execution recorded separately |
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
