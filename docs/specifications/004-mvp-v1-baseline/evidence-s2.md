# Evidence — MVP Slice S2: install, first run, and Project setup

## Claim and authority boundary

This record covers the explicitly authorized Specification 004 Slice S2 only:
T04–T07. Work started from `main` revision
`66c0782ffbad2263f6a095e1c2c5e88100872a21`, the merge of PR #83. S3–S7,
GitHub Release publication, automatic upgrade/recovery, Provider mutation, and
final MVP acceptance were not authorized and were not performed.

The results below establish technical implementation Evidence. They do not imply
human acceptance. Issue or PR state also does not supply human acceptance.

## Delivered behavior

### T04 — release archive and owned install

- `scripts/build-release-archives.sh` accepts an exact semantic version, requires
  clean source for release provenance, refuses a non-empty output directory, and
  builds the three approved archive names:
  `macos-27-arm64`, `ubuntu-26.04-amd64`, and `ubuntu-26.04-arm64`.
- Every archive includes `lingo`, `LICENSE`, installer, closed release metadata,
  the compatible five-skill manifest and files, and a complete inner SHA-256
  manifest. `SHA256SUMS` identifies the outer archives.
- `scripts/install-release.sh` stages and rechecks the selected archive before
  extraction, rejects unlisted files and unsafe archive types/paths, and validates
  the complete release row: exact macOS 27.0/arm64 or Ubuntu 26.04 distribution
  and version on amd64/arm64. Unknown host facts and every other row fail closed.
- Existing binary and receipt roots must be owner-only mode `0700` with no
  detected extended ACL. Unsafe roots and foreign permissions are preserved, not
  silently repaired.
- The closed receipt binds destination, binary/archive/skill-manifest digests,
  version, revision, source state, platform, architecture, and skill-set version.
  It also records `installedAt` once in RFC 3339 UTC. Equivalent reinstall
  preserves that value and is `unchanged`; unknown receipt fields and foreign,
  modified, symlinked, hard-linked, divergent, wrong-platform, or concurrent
  state are preserved and refused.
- The installer never edits a shell profile or `PATH`. S2 refuses owned version
  upgrade because resumable upgrade belongs to T20.

### T05 — five-skill compatibility and first run

- One versioned manifest closes the five hyphen-case embedded skills and records
  each SHA-256 digest plus binary compatibility version.
- `first-run` and `runtime codex status` report canonical completion/provenance
  plus every skill's expected digest and observed state: `missing`, `equivalent`,
  `owned_older`, or `modified_or_foreign`.
- Installation is idempotent, upgrades only known older Axiom bytes, and refuses
  changed/foreign/link-unsafe files plus roots/files with permissive modes or
  extended ACLs.
- The distributed `darwin && !cgo` binary inspects extended security through
  `fgetattrlist` on the already-open object, first confirming that the volume
  reports ACL support. Missing capability, malformed response, or syscall failure
  remains fail-closed; ACL-free objects are accepted and present ACLs are refused.
- Process coordination uses one private, schema-checked advisory lock file.
  A live holder returns `codex_skill_install_concurrent`; kernel lock release
  after real `SIGKILL` permits safe inspection and resume of exact missing/current
  skill state. Unknown content/type/permissions are preserved and return
  `recovery_required`.
- First run directs `runtime codex install` and then explicit `project configure`;
  it does not invoke Codex, configure a Project, inspect Git, or edit a profile.

Current skill digests:

| Skill | SHA-256 |
|---|---|
| `axiom-project-configure` | `d481dc61ecd7a9afd1ffd0a79908515f15f04a75002a7d501eea5517f1f4844e` |
| `axiom-project-show` | `a80b3038c497fe3f3b817e5d5d28bca68de96d9ceaf76630f08ad1e34c5db0f4` |
| `axiom-work-item-create` | `6750abfe6cb817ff4f011d7c6b59a27e4b12832a7c1bbae68a9357bee249d4bd` |
| `axiom-work-item-run` | `a75f21684d38f325840461fbe8e959ed9fd2b925ac630c7d471147fdfef124dd` |
| `axiom-work-item-status` | `4fbb6fda699dc50af88f96355cb9cbed05dbebf34a7ed3218bc26b72b7fd60c7` |

### T06 — read-only normalized setup preview

- Guided and complete flags enter the same `projectapp.PrepareSetup` path and
  normalize Repository order before producing portable intent, local bindings,
  capability readiness, destinations, observed revisions, effects, and one
  preview digest.
- Preview performs no publication. Repository paths and their observations remain
  local; they do not enter `axiom.yaml`. No CWD or Git remote becomes identity or
  capability. Missing and unsupported Work Item capability remain explicit and
  block that journey only.
- The guided path asks only missing values, displays the same JSON proposal, and
  publishes only after the exact `yes` confirmation.

### T07 — authorized separate publication

- Apply requires the preview's explicit Project ID and digest plus
  `--authorize-local`, then re-observes portable state, local state, and every
  Repository directory. Changed facts yield canonical `denied_authority`.
- Portable intent publishes before ID-addressed local bindings. A confirmed
  portable commit followed by local failure reports canonical `partial` with the
  portable reference; it never claims rollback or atomicity across roots.
- Exact replay is a no-op. Two independent processes using one absent-state
  preview produce one winner and one fail-closed refusal (`denied_authority` or
  an in-progress-state inspection failure), never mixed success.
- The resulting Project resolves by UUID or slug from an unrelated CWD. Work Item
  use verifies the declared portable GitHub capability instead of silently
  falling back to ambient configuration.

## Deterministic fault and conflict observations

| Stage / condition | Test seam or case | Observed truth |
|---|---|---|
| F0–F3 before binary commit | injected `before_binary`, checksum/manifest/platform/link/type conflicts | no binary or receipt; owned marker/stage removed |
| F3 stale setup authority | changed inputs, Repository replacement, changed revisions | `denied_authority`; no Project publication |
| F4–F5 coordination/publication | existing lock, foreign destination, hard link, two-process Project race | conflict preserved; exactly one Project winner, loser refuses |
| T05 process death during skill publication | child process holds the advisory lock after one exact skill publication, then receives `SIGKILL` | concurrent retry refuses while child lives; post-death retry validates the persistent lock marker and partial skill bytes, completes the set, and reaches `codex_ready` |
| T05 ambiguous lock state | legacy directory or invalid private lock state at the canonical lock path | `recovery_required`; unknown state remains untouched |
| F6 binary committed before receipt | injected `after_binary` | binary remains, receipt absent, marker says `binary_committed`, retry is `recovery_required` |
| F7 portable committed before local state | deterministic local-root replacement hook | canonical `partial`; portable digest remains readable |
| F8 cleanup | pre-commit marker cleanup plus S1 shared-publication F8 suite | commit truth preserved; unknown post-commit marker is not deleted automatically |

Timing sleeps are not race or fault oracles. Process coordination uses the shared
protected publication protocol and explicit lock/barrier state.

## Validation environment and commands

Native S2 execution environment:

- macOS 27.0, build `26A428`;
- Darwin 27.0.0, arm64, kernel `RELEASE_ARM64_T8103`;
- Go `go1.26.1 darwin/arm64`;
- filesystem tests use isolated temporary roots on the current host.

Commands used for the final branch validation are listed below. Each must exit
zero unless the command itself is a negative-path assertion inside a test:

```bash
bash -n scripts/build-release-archives.sh scripts/install-release.sh \
  scripts/test-release-archives.sh scripts/dogfood-poc.sh
CGO_ENABLED=0 go test ./internal/darwinacl ./internal/codexruntime ./internal/local -count=1
go test ./... -count=1
go test -race ./internal/codexruntime ./internal/projectapp ./internal/local ./internal/cli ./cmd/lingo -count=1
go vet ./...
go build ./...
go mod verify
./scripts/test-release-archives.sh
./scripts/test-codex-skills.sh
./scripts/dogfood-poc.sh
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
git diff --check
```

PR #84 blocker remediation reran this matrix on the environment above. The
explicit no-cgo adapter tests, Go unit/integration/black-box suite, selected race
suite, vet, build, module verification, release archive/install suite, Codex skill
suite, dogfood journey, repository validator, sensitive-file scan, and diff check
all exited zero. Negative cases retained exact-host-row rejection,
permissive-root and ACL refusal, unknown receipt-field refusal, active-process
refusal, ambiguous-lock `recovery_required`, safe resume after a real child-process
`SIGKILL`, and foreign skill preservation. Gitleaks was available and reported no
leaks.

Clean-source corrected-artifact observation from implementation commit
`3c528c18e63d5743c7d7ca5e4232c81fc0d987f7`, version `0.1.0-s2.2`:

| Archive | SHA-256 |
|---|---|
| `axiom-0.1.0-s2.2-macos-27-arm64.tar.gz` | `38fdfeca40170f406afe2395e206ba09f070176439679a40da6195265e4063af` |
| `axiom-0.1.0-s2.2-ubuntu-26.04-amd64.tar.gz` | `54e697a3c54ed4dab94d0d88546b204280db337d93bb7766a07fdd30f6465dae` |
| `axiom-0.1.0-s2.2-ubuntu-26.04-arm64.tar.gz` | `456d1230051dbaab395435522a47210545cb45e2c01834d01fbed9d5d3eec051` |

The corrected native macOS archive executed the real installed-binary sequence:

```text
install archive
-> lingo version
-> first-run
-> runtime codex install
-> runtime codex status
-> runtime codex install (idempotent rerun)
```

The archive install returned `install_status=installed`. The extracted binary
reported version `0.1.0-s2.2`, revision `3c528c18e63d`, and source state `clean`.
Its SHA-256 was
`37c450270d1314f5877aad7a841b10d0520f5e6c971cd150f5cdbaaae534abe0`;
the closed installation receipt SHA-256 was
`913fd89565d5949c4eb00954439dbf4caacf9ed954143ed01448b9fae6c22f57`.

Before installation, `first-run` exited nonzero with `validation_failure` and
exactly five `missing` skills whose names and digests matched the archive
manifest. `runtime codex install` returned `codex_configured`, installed exactly
five skill directories, and reported all five as `equivalent`. Each installed
`SKILL.md` digest matched the manifest values recorded above. `runtime codex
status` returned success with `Lingo and Codex skills are compatible`; the second
install returned `codex_already_configured` and preserved the complete runtime
file digest set.

The runtime root and five skill directories remained owner `501`, mode `0700`,
and ACL-free; the skill files, lock, and receipt remained owner `501`, mode
`0600`, link count one, and ACL-free. The archive regression also presented a
private foreign `axiom-project-configure/SKILL.md`; install returned
`codex_skill_conflict` and its SHA-256 remained unchanged.

These artifacts were generated only in an isolated temporary directory for
Evidence; they were not published as a prerelease or GitHub Release.

## Native Evidence and limitations

- The macOS 27/arm64 archive is built and installed natively in this Evidence run.
- Blocker remediation was exercised natively on macOS 27.0/arm64 with the actual
  no-cgo archive binary, including the complete first-run/install/status journey,
  exact-version negative case, destination and Runtime ACL refusal, and Codex
  skill-process `SIGKILL` resume.
- Ubuntu 26.04 amd64 and arm64 binaries/archives are cross-compiled and their
  structure/manifests/checksums are verified here. This is not substituted for
  native ext4 execution.
- Exact Ubuntu point-release/kernel/ext4 runs, ACL matrices, all target-specific
  primitives, owned upgrade/resume, and the complete target matrix remain explicit
  blocking obligations of T20, T22, and T24.
- The optional bundled Codex validator dependency was unavailable on this host;
  `scripts/test-codex-skills.sh` reported that limitation and completed its Go
  contract validation. No real Codex invocation is claimed by this remediation.
- Distribution trust in S2 is SHA-256 integrity only. Signing, notarization,
  package managers, automatic update, and GitHub Release publication remain out
  of scope.
- Tests cover supported local-process concurrency and deterministic injected
  faults, not physical power loss or malicious arbitrary same-UID interleavings.

## Acceptance status

S2 T04–T07 are technically implemented and ready for human review. Human
acceptance is not claimed. No authority exists here to start T08 or any S3–S7
Task.
