# Implement Evidence 010 — Resumption 1, D1/D3 proposal gate

**2026-10-20 — BLOCKED at the requested human approval gate.**
This supersedes the operational checkpoint in [previous Evidence](evidence-implement.md)
without changing its historical measurements. Proposed contracts are ready for
review; no exact D1/D3 value or production mechanism was accepted/frozen.

## Authority and revalidation

- Human Tasks authorization: [PR comment](https://github.com/rgomids/axiom/pull/335#issuecomment-6298186645),
  exact `088eb89ac1d2a4328a956cec7047990b566e9934`; T01–T16/P1–P6 and G0–G3 preserved.
- Resumption authority: Rafael's instruction of 2026-10-20 17:36 BRT permits native
  Actions on Linux amd64/arm64, Docker experiments, complete **Proposed** D1/D3
  contracts and a PR summary. Stop before contract freeze and dependent T03+.
  No release/native Runtime/operating-account installation authority inferred.
- Initial local/remote/PR SHA matched `a6d37dca1a1ea47b545a0aa7c89dca422b429de6`,
  branch `docs/axm-37-offline-operational-documentation`, clean worktree; PR #335
  OPEN/Draft. Only delta since approved Tasks: old Evidence checkpoint.
  Baseline CI and delivery-metadata SUCCESS, not proof of this change.
- Read AXM-37 and newest eight comments: In Review, no blockedBy, still blocks
  AXM-39/AXM-12. No status/relation mutation. Existing implementation prompt and
  previous checkpoint continued; canonical Execution remains **unknown**. No
  start/restart/dispatch or invented Execution/acceptance record.
- No competing remote head observed before either experimental push. Final
  revalidation and CI are reported in the PR delivery comment at exact final SHA.
- No delegation: runtime exposes overrides but no evidence establishes a strictly
  inferior capability ordering. Main Orchestrator performed integration/review.

## E01 — Proposed contracts and reproducible coverage

[Contract proposal revision 1](contracts-proposed.md) contains separate versions,
closed schemas/fields, exact encoding/order, layout/permissions, quotas, source /
binary / docs / skills digest DAG, compatibility floor/matrix, public discovery
statuses/reasons/paths/exits and man selection. All are Proposed. Material decisions:
**D1-a–e and D3-a–c**, with alternatives/costs in the document. No ADR/approved
Spec/Clarification/Plan/Tasks rewritten; no production codec implemented.

[CLI coverage snapshot](cli-coverage-proposed.json): 105 public leaves and 549
flag registrations from actual `publicCommands` / `commandFlagSet`; all mapped to
proposed shared man pages. Future docs inspect adds one leaf. Proposed 11 text
outputs including notices + 15 pages = 26 outputs within candidate 64. Generated
content byte budgets, complete reference parity and future leaf still NOT_RUN.
No generator, public command or final corpus was implemented.

Inventory command: temporary `internal/cli/axm37_inventory_test.go` recursively
walked `publicCommands`, called `commandFlagSet(...).VisitAll`, emitted path and
`Name=DefValue`. `go test ./internal/cli -run '^TestAXM37Inventory$' -v` exit 0;
temporary file removed immediately. Final inventory JSON records its exact
bounded output, preserving all flags, including historical public spellings.

Current source-build size check, `go build -trimpath -ldflags '-s -w' -o <scratch>/axiom
./cmd/lingo`, exit 0 on macOS arm64: binary 9,748,018 bytes; LICENSE 11,358;
installer 10,276; four skills 31,120 / 14,404 / 35,980 / 20,296 bytes. These are
size observations, not release artifacts or Linux binary qualification. Proposed
archive overhead accounts for binary, installer/license, skills and control files;
final documentary bytes remain unavailable pending corpus implementation.

[Contract vectors](contract-vectors-proposed.json) preserve SHA-256 pins for v1
producer/reader/receipt source at baseline, literal synthetic metadata/inventory/
provenance/man-map/binding/receipt records, four proposed discovery records, and
negative mutation/test oracles. They are review fixtures, **not installable complete
archives**, not historical release bytes, and not production codec test results.
Document QA checks hashes and internal relations; T07 must execute the actual
shell/Go differential matrix and zero-candidate-execution sentinel. Native historical
v0.1.0 migration remains NOT_RUN; proposed floor is not certified support.

## E02 — Native image experiments, independent of contract freeze

[Sanitized outputs, hashes and job URLs](probe-results-resume-1.json) preserve four
successful environment rows and the failed emulated control. Fixed experiment SHA:
`9d3fd8ce27f1aa70c541f6cf35d0a1094436794f`. Native
[Actions run 39494151218](https://github.com/rgomids/axiom/actions/runs/39494151218)
completed SUCCESS on both authorized runner labels. This is experimental source
Evidence, distinct from final PR CI and product-native qualification.

| Host / architecture / observed filesystem | Command and measured outcome |
| --- | --- |
| macOS 27.0.1 arm64, Darwin 27, APFS; Apple clang 21 | `python3 <scratch>/probe.py`, exit 0; independent `PROC_PIDREGIONPATHINFO` on address of main plus descriptor identity, all eight cases PASS |
| ubuntu-24.04, x86_64, kernel 6.17.0-1022-azure, stat label `ext2/ext3` | Pinned workflow, Python 3 + GCC 13.3; `/proc/self/exe` descriptor identity, eight cases PASS; exact binaries/hashes in JSON |
| ubuntu-24.04-arm, aarch64, kernel 6.17.0-1022-azure, stat label `ext2/ext3` | Same source/oracles, GCC 13.3; eight cases PASS; native runner, not cross-build |
| Docker Linux VM aarch64, kernel 7.0.14-linuxkit, overlayfs, GCC 14.3 | `docker run --rm --network none --platform linux/arm64 -v <scratch>:/evidence:ro gcc:14-bookworm python3 /evidence/probe.py`, exit 0; native arm64 instructions, additional VM/container Evidence |
| Docker Linux amd64 under emulation on arm64 Mac | Same command with linux/amd64, exit 1; first explicit replacement oracle FAIL. Excluded from native proof, later cases NOT_RUN |

Docker 29.9.1, gcc multiarch manifest digest
`sha256:5e927c284bf55a7dc796262e311a0703344f62f41f5621eb56843111b1d37e15`.
Images acquired before `--network none` probe; image downloads are not offline
installation Evidence. Docker input mount was read-only, writes confined to
container temporary directory; no operator installation/configuration changed.

Protocol: compile A and B from same C source with distinct TAG constants; they
have different hashes, no product version/provenance contract. Spawn A, block on
stdin after independent OS proof and open descriptor. Parent waits for flushed
identity, replaces target with B via rename, then releases stdin. Compare original
OS image, post-change independent OS image, pinned descriptor and current pathname.
A still prints TAG A; current path hashes to B. Spawn fresh B and prove B image.
Process timeout 10 seconds after release, assertions return nonzero on violation.

Eight cases: explicit path, symlink invocation with spaces, invocation through PATH,
unlink/replacement, parent-directory replacement, hard-link invocation, relocation
and copy of A. Every case replaces the original pathname by B before resuming A.
The raw case label `switched-PATH` tests PATH resolution at launch only; an
independent A/B PATH-switch matrix remains NOT_RUN. It is not counted as full V2.
The PATH case proves resolution at launch; mutation of parent Python's environment
after fork would not change child's environment. No later PATH re-resolution used.
Symlink/hardlink cases establish OS identity behavior, not a qualified policy for
alias-chain validation; hard-link count is observable refusal input. Relocation/copy
are identity mismatches requiring explicit rebind/refusal in future resolver, not
automatically supported relocations. Unlink descriptor survives as OS reference,
not as authority to an unrelated new path/inode.

Single-file reader controls: pin fd to validated A content; rename B over leaf;
fd still hashes to A while unchecked reopen hashes B and is rejected. In-place
write after validation changes hash and is rejected before accepting content.
This proves primitive availability, not an integrated complete inventory reader.
Lock control: one process holds `flock` on common experimental lock; second
process with different receipt-root argument returns 7, then 0 after release.
This proves shared-file exclusion; receipt argument intentionally does not choose
lock. It does **not** prove derivation from installation identity, stale expected
revision handling, target-swap safety, production locks or T03 transitions.

Emulated amd64 counterexample: A start image/pin inode 60769; after replacement
TAG remains A, image/path report 60770 while pinned fd remains 60769. The oracle
correctly fails. No native support conclusion follows from emulated `/proc` behavior.

Initial [run 39494092374](https://github.com/rgomids/axiom/actions/runs/39494092374)
at `f76885bba477f4462f282309950510c4ebc74519` failed before image probes because
harness used BSD stat arguments on GNU stat. Fixed in `9d3fd8c`; corrected run above
PASS. Initial run is not omitted or counted as product FAIL/PASS. Baseline helper
also needed a local `diskutil` volume-selector correction before macOS success.

### Replay and temporary workflow removal

Temporary workflow `.github/workflows/axm37-identity-experiment.yml` existed only
in experimental commits and was registered with explicit deletion condition.
Triggers: push on exact AXM-37 branch; guarded workflow_dispatch on same ref;
permissions `contents: read`, no secrets/write token, pinned checkout with
`persist-credentials: false`, 10-minute job timeout. Existing workflows unchanged.
Workflow and temporary registry entry **removed before human review**; exact
replay source remains in experiment commit, no durable helper added.

To reconstruct the inspected probe without modifying current tree:

```bash
git show 9d3fd8ce27f1aa70c541f6cf35d0a1094436794f:.github/workflows/axm37-identity-experiment.yml > /tmp/axm37-workflow.yml
python3 - <<'PY'
from pathlib import Path
text = Path('/tmp/axm37-workflow.yml').read_text()
lines = text.split("          python3 - <<'PYPROBE'\n", 1)[1].splitlines()
probe = []
for line in lines:
    if line == '          PYPROBE':
        break
    probe.append(line[10:])
Path('/tmp/axm37-probe.py').write_text('\n'.join(probe) + '\n')
PY
shasum -a 256 /tmp/axm37-probe.py
python3 /tmp/axm37-probe.py
```

Expected Python SHA-256:
`aec77be29864a702adf6abeab4def1d1740b197bf1674303c5341cb1c8319b5d`;
C source SHA-256 `62d4e7a1bbd0ebd9130e4e2ea3ef260169f2c2ab8b942fd2c9844da9c07e1b66`.
Binary digests differ by host/toolchain and temporary build inputs; compare recorded
per-run digests and within-run A/B relations, not cross-host binary reproducibility.

## Remaining qualification and gate status

T02 is **BLOCKED / partial**, not completed: T01 unapproved; no adopted mechanism,
full-generation reader/inventory pinning, independent A/B PATH switches,
unavailable-proof/stale-fd negatives,
unsafe alias resolution, inode-reuse disposition, compiled-Go OS integration or
bound directory chain/rebind oracle. No unrestricted filesystem claim beyond
observed rows. Pin-at-start without independent OS comparison remains insufficient;
macOS experimental comparison supports refusal when pin differs from text vnode.
Linux native `/proc/self/exe` is viable in measured rows; emulation demonstrates why
unverified platform behavior cannot be generalized. T04 must review final mechanism.

| Task | Result | Evidence / next dependency |
| --- | --- | --- |
| T01 | BLOCKED | E01 proposal delivered; D1-a–e/D3-a–c approval and contract freeze outstanding |
| T02 | BLOCKED | E02 bounded native probes PASS; integrated/final contract proof not complete |
| T03 | NOT_RUN | T01/T02 dependency, no publication/recovery fixtures run |
| T04 | NOT_RUN | T01–T03 absent; G1 BLOCKED |
| T05 | NOT_RUN | G1 |
| T06 | NOT_RUN | G1/T05 |
| T07 | NOT_RUN | G1/T06 |
| T08 | NOT_RUN | G1/T07 |
| T09 | NOT_RUN | G1/T08 |
| T10 | NOT_RUN | G1/T09 |
| T11 | NOT_RUN | G1/T10 |
| T12 | NOT_RUN | G1/T11 |
| T13 | NOT_RUN | G1/G2/T12 |
| T14 | NOT_RUN | G2/T13 |
| T15 | NOT_RUN | G2/T13 |
| T16 | NOT_RUN | G3/T14/T15 |

AC-01, AC-02, AC-03, AC-04, AC-05, AC-06, AC-07, AC-08: **NOT_RUN**.
V1–V9 product validations: **NOT_RUN**. E01 documentary checks and E02 primitives
are partial inputs to V1/V2/V4/V6; they do not satisfy these full rows. Authoritative
OD-01–13 / AC-01–08 / V1–V9 / Task map remains [Tasks coverage](tasks.md#od--ac--validation--tasks-coverage).
No man reader, offline installation V8, native Codex/Claude V9, downgrade/independent
return, release artifact, loaded skill or full publisher concurrency claim made.
G0 scoped authority exercised; G1 BLOCKED; G2/G3 NOT_RUN.

## Validation and runtime-agnostic handoff

Executed local checks: inventory extraction, source-build size probe,
`go test ./internal/install ./internal/cli` (exit 0), three native OS/arch probes
plus Docker arm64 (exit 0); emulated amd64 expected experimental refusal (exit 1).
Additional executed local checks, all exit 0: ephemeral document QA (fixture
field counts/hash DAG, 33 negative test specifications, 105 leaf/549 flag coverage,
four environment records, source pins, preserved historical inputs, local links
and temporary-workflow removal), `./scripts/validate-repository.sh .`,
`python3 scripts/check-adr-governance.py .` (34 ADRs), `git diff --cached --check`,
`./scripts/check-sensitive-files.sh --staged .`, and
`gitleaks git --staged --redact --no-banner` (no leaks). QA validates proposed
records, not behavior of future codecs. Final-head CI is recorded in the PR delivery
comment after read-back. No global product PASS inferred from probes.

Next action: Rafael reviews exact proposed D1-a–e/D3-a–c revision and provides
named approval/edits. No approval inferred from silence, CI or this delivery.
Following acceptance, resume T01 freeze and remaining T02 coverage before T03/T04;
revalidate remote head, Execution/checkpoint and authority. Product review/acceptance,
AXM-39/AXM-12 blockers, release and merge stay separate. No successor dispatch.
