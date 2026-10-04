# Issue #186 Evidence — upgrade receipt and Runtime skill convergence

Issue: [#186](https://github.com/rgomids/axiom/issues/186). Decision:
[ADR-0016](../../decisions/0016-candidate-derived-skill-set-receipt.md).
Executed 2026-10-04 on macOS 27.0.1 (`26A434`), arm64, APFS, `go1.26.1`.
This is new Evidence for the new behavior; historical S7/T20 Evidence is
unchanged.

## Root causes

Both were reproduced with the **published** v0.1.0–v0.4.1 macOS arm64 release
archives (checksums verified against each release's `SHA256SUMS`), installed
through each bundle's own `install.sh` facade into an isolated `HOME`, with
`codex`/`claude` stand-ins on `PATH` and `axiom first-run` run by the old
version before upgrading.

**A — Codex skill-set receipt never refreshed by the upgrade.** The owned
upgrade published Codex skill files but, by the S7/T20 interim design, never
`.axiom-skill-set.receipt`, because archive format 1 does not carry the
candidate's runtime constants. Every upgrade that changed skill text therefore
ended `install_status=partial` (exit 1) with `runtime codex status`
`validation_failure` and every skill `equivalent`. The installer, however, is
always executed by the candidate binary itself, which holds those constants.

**B — receipts of earlier releases re-derived with the current skill names.**
`claudeReceiptBytes` serialized every historical revision with this binary's
six skill names, emitting `skill.axiom-project-list=` (empty) for the
five-skill revision that v0.1.0–v0.1.2 published. The receipt those releases
really wrote was never recognized, so `runtime claude install` and
`first-run` refused every replacement (`claude_skill_conflict`) although each
skill was `missing`, `equivalent` or `owned_older`. The output named no
artifact, so the conflict was undiagnosable. The same pattern made
`installedSkillManifest` return "" for a five-skill Codex root.

Pre-fix baseline, published source → published v0.4.1:

| From | Installer | Claude after two `first-run` |
|---|---|---|
| v0.1.0 | `partial`, exit 1 | `failed claude_skill_conflict` |
| v0.1.1 | `partial`, exit 1 | `failed claude_skill_conflict` |
| v0.1.2 | `partial`, exit 1 | `failed claude_skill_conflict` |
| v0.2.0 | `partial`, exit 1 | `already_configured` |
| v0.2.1 | `partial`, exit 1 | `already_configured` |
| v0.3.0 | `partial`, exit 1 | `already_configured` |
| v0.4.0 | `upgraded`, exit 0 (no skill change) | `already_configured` |

The v0.1.1 row reproduces the issue report exactly (same messages, same
Claude skill states).

## Fix

- Historical receipts serialize each revision's own skill set
  (`skillSetRevision.names`); current receipts are byte-identical.
- The upgrade publishes the Codex skill-set receipt as an authorized
  `skill_receipt` effect when the running binary is the candidate release
  (ADR-0016); otherwise `refresh_required` is kept; an unrecognized receipt is
  preserved (`skillReceipt=conflict`).
- Runtime results classify each skill as `missing`, `equivalent`,
  `owned_older`, `modified`, `foreign` or `unsafe`, report the receipt state,
  and list every blocking artifact relative to the skill root.
- The Windows installer exits 1 on a `partial` upgrade (it exited 0).

## Automated regressions

| Test | Proves |
|---|---|
| `internal/codexruntime` `TestEveryPublishedReceiptIsRecognizedAsAxiomOwned` | every Codex/Claude receipt captured from the published v0.1.0–v0.4.1 binaries (`testdata/published-receipts`) is recognized; fails on the pre-fix code for v0.1.0/v0.1.1/v0.1.2 Claude |
| `TestEveryPublishedSkillDigestIsKnown`, `TestEveryPublishedClaudeInstallationConverges` | every published skill digest is owned; a root each release configured converges and reruns `unchanged` |
| `TestConflictsNameTheArtifactAndDistinguishModifiedFromForeign` | edited Axiom skill = `modified`, unknown = `foreign`, unrecognized receipt reported; nothing overwritten |
| `internal/install` `TestUpgradeAsCandidate*` | receipt effect is last, published, `runtime codex status` ready, rerun no-op; absent receipt created only for a configured root; unrecognized receipt preserved |
| `TestUpgradeNotRunningAsCandidateKeepsRefreshRequired`, `TestUpgradeCandidateIdentityRequiresEmbeddedSkills` | other revision, development, dirty, older or unset build, or different skills never derive a receipt |
| `TestUpgradeAsCandidateResumesAfterInterruptionAroundReceipt`, `TestUpgradeReceiptStageLeftoverIsRecoveredOnlyWithMarker` | interruption before/after the receipt resumes to success; a receipt stage leftover is removed only under the operation marker, otherwise `recovery_required` |
| `TestInstalledSkillManifestOmitsSkillsAnEarlierReleaseDidNotHave` | the v0.1.1 five-skill manifest digest (`98ba8954…`) is reconstructed |

## End-to-end matrix after the fix

Candidate: `0.4.2-local.1` built by `scripts/build-release-archives.sh` from
this branch (clean source, release metadata), installed through its own
bundle facade. Not a published release.

| From (published) | Installer | `runtime codex status` | `first-run` | 2nd `first-run` | Reinstall |
|---|---|---|---|---|---|
| v0.1.0, v0.1.1, v0.1.2, v0.2.0, v0.2.1, v0.3.0 | `upgraded`, 0 | `success` | Claude `configured`, 0 | both `already_configured`, 0 | `unchanged`, 0 |
| v0.4.0, v0.4.1 | `upgraded`, 0 | `success` | both `already_configured`, 0 | same, 0 | `unchanged`, 0 |
| v0.1.1 → published v0.4.1 (`partial`) → candidate | `upgraded`, 0 | `success` | Claude `configured`, 0 | `already_configured`, 0 | — |
| v0.1.1 → candidate, only Codex / only Claude / no Runtime | `upgraded`, 0 | — | `success`, 0 | `success`, 0 | — |
| v0.1.1 → candidate, Claude skill edited + foreign skill | `upgraded`, 0 | — | `partial`, 1, `conflict: artifact=axiom-work-item-run/SKILL.md state=modified …`, `axiom-project-list/SKILL.md state=foreign`; edits byte-identical | same, 1 | — |

Claude reports `owned_older` between the installer and `first-run`: the
installer converges the Codex root only, and its summary directs to
`axiom first-run`.

## Commands

All on this branch, clean worktree:

| Command | Result |
|---|---|
| `go build ./...`; `GOOS=windows go build ./...`; `GOOS=linux go build ./...`; `go vet ./...`; `GOOS=windows go vet ./cmd/... ./internal/install` | pass |
| `gofmt -l .` | empty |
| `go test ./... -count=1` | pass |
| `go test -race ./... -count=1` | pass |
| `scripts/test-install-bootstrap.sh`, `test-install-posix-facade.sh`, `test-install-axiom.sh`, `test-release-archives.sh`, `test-codex-skills.sh` | pass |
| `scripts/test-s7-security.sh` | `failures=0 result=pass` |
| `scripts/test-s7-native.sh` | `native_row=unsupported result=blocked` on 27.0.1 (its historical exact-`27.0` row gate); rerun with a `sw_vers -productVersion` shim reporting `27.0`: `failures=0 result=pass`, including `skills-apply-partial-receipt-refresh` (development binaries are not the candidate, so `refresh_required` stays) |
| `scripts/validate-repository.sh .` | pass |

## Not verified here

- Linux and Windows execution: covered by repository CI on the PR, not run
  locally. Windows `fresh_windows_test.go` compiles (`GOOS=windows go vet`).
- Upgrading from published releases is a manual matrix here; the automated
  published-release upgrade acceptance is delivered with Issue #153 I153-T03.
