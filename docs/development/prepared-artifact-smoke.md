# Automated prepared-artifact smoke

Issue [#244](https://github.com/rgomids/axiom/issues/244) authorizes this bounded
release-preparation contract, replacing the reverted manual/unavailable gates
described in PR #250. It does not alter publication authority or product support.

## Contract and acceptance

Preparation builds once, retains structural/digest/provenance verification, then
executes the exact prepared `linux-amd64` archive on `ubuntu-24.04`. A dependent
job downloads that same prepared set and executes `macos-27-arm64` on `macos-15`.
These are the existing CI-available native environments; the macOS archive name
is legacy naming, not an OS-version restriction (ADR-0015, Specification 006).

Each smoke checks executable permission and exit status, expected release
version and revision, and a small Project configure (preview plus authorized
local publication), show and list lifecycle. HOME, PATH, temporary files,
Projects, local state and skill roots are isolated. No provider, Runtime,
installer, source rebuild or full PR suite is invoked. Every command has a
20-second timeout; workflow jobs retain their preparation timeout (20 minutes)
and a bounded macOS timeout (10 minutes).

The archive checksum is checked against prepared `SHA256SUMS`; archive paths
and entry types are validated before extraction. All prepared artifact-directory
file digests and directory entries are compared before/after execution,
including on CLI failure. The extracted binary digest must also remain unchanged.
Any smoke failure fails its job and thus preparation. Publication's existing
successful-run check rejects a run whose dependent macOS smoke failed.

## Retained summary

Each row writes a single JSON record with schema
`axiom-prepared-artifact-smoke/v1`, exact version/full revision, row, archive and
binary SHA-256, observed OS/kernel/architecture/platform and runner image/version,
input-preservation result and pass/fail. Failed attempts record the available
subject facts and a bounded reason. Raw CLI output remains temporary diagnostic
data. Summaries are uploaded with `always()` for 30 days as separate
`axiom-release-smoke-<row>` workflow artifacts. They are not publishable assets,
publication-envelope inputs or `axiom-gate-evidence/v1` authorization records.

Publication still verifies artifact identity and the authorized envelope and
publishes only prepared bytes. It does not rerun this smoke or rebuild. Historical
prepared runs remain verifiable through their pinned source/control contracts.
Recovery preparation uses the validated pinned control revision for smoke code
when correction pins are supplied; the tested bytes/provenance remain those of
the original source revision.

## Implementation and validation

`scripts/smoke-prepared-artifact.py` uses only Python's standard library.
`scripts/test-release-pipeline.sh` exercises real prepared archives and workflow
transfer/order contracts; `scripts/test-prepared-artifact-smoke.py` covers
deterministic refusal, isolation, timeout and mutation cases.
`scripts/test-release-flow.sh` guards the no-rebuild/no-smoke publication boundary.
No new ADR is required: the approved build-once, explicit publication authority
and platform eligibility boundaries remain intact. This satisfies constitution
requirements for explicit intent, deterministic validation and bounded impact.
Rollback removes these preparation steps; prepared bytes and publication
contracts remain unchanged.

## Residual risk

Hosted smoke proves this bounded lifecycle in the recorded environment,
not every filesystem/install/upgrade journey. Issue #256 adds required Linux
arm64 and explicitly authorized member Windows Server AMD64 installation lifecycle acceptance, described in
[native platform acceptance](native-platform-acceptance.md). Supported Windows
client native acceptance remains blocked by hosted runner eligibility and does
not block publication.
Unavailable rows remain residual risk under
[#256](https://github.com/rgomids/axiom/issues/256), never simulated as PASS.
Any future change of the automated runner set follows an explicit reviewed
support/infrastructure change; a failed automated smoke cannot be waived as
an unavailable platform within this workflow.
