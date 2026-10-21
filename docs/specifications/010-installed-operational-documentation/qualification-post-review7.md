# AXM-37 — Post-review 7 qualification handoff

Prepared 2026-10-21 for the same Implement/PR #335. R1/H1 remains the accepted
baseline; [E01–E16](evidence-implement-v3.md) retains historical results and
limits. This preparation does not accept product behavior, replace Q6 or grant
host, network, credential, Runtime, release or Linear-transition authority.

Later bounded continuation: [v2/gzip r2 Evidence](evidence-bounded-v2-gzip-r2.md).
The five decisions and explicit patch authority were subsequently verified;
this packet's historical qualification limits remain unchanged.

## Entry and reproducible development checks

Entry head: `78ecd6cac911d6f53726bd51cda7b2817413365b`. Review 7 resolved
CR-002; no correction or dispatch is repeated. Entry CI and delivery checks:
[CI](https://github.com/rgomids/axiom/actions/runs/39525580088),
[metadata](https://github.com/rgomids/axiom/actions/runs/39525580176).
The live Linear read found **In Progress**, no blockedBy, and AXM-39/AXM-12 still
blocked. The prompt's In Review snapshot is stale; no state was changed here.
The ledger records dispatch acceptance, not canonical product Execution proof.

```sh
go test ./...
go vet ./...
go build ./...
./scripts/test-docs-offline.sh
./scripts/validate-repository.sh .
python3 scripts/check-adr-governance.py .
```

The existing offline harness now also accepts an exact local artifact pair:

```sh
./scripts/test-docs-offline.sh --archive /absolute/local/axiom-VERSION-ROW.tar.gz \
  --checksums /absolute/local/SHA256SUMS
```

Both files must already exist. The harness never downloads or starts Runtimes;
it installs/mutates only temporary test copies. The default still builds a
development artifact. Supplied mode does not rebuild candidate bytes. It records
archive/binary hashes, metadata, source commit, generation/binding and documentary
integrity before negative mutations. Retain that output together with the final
exit status; an identity line alone is not a successful run. Inputs and their
hashes must remain unchanged. `release=true` records builder metadata, not actual
publication or trusted acquisition. A dirty development artifact never becomes
release Evidence by passing this script.

The harness needs preparation tooling (including repository scripts); `env -i`,
CWD `/`, local links and a fresh HOME do **not** deny source/network access. Even
supplied mode is only a development rehearsal, not the isolated V8 run. Its man
lookup is advisory, not V7. It cannot run genuine lifecycle/migration/Q6 matrices
or substitute for the separate native suites below.

## Native execution packet — prepared, NOT_RUN

Before any controlled run, bind authorization to three native disposable hosts:
macOS arm64, Linux amd64, Linux arm64. Record exact OS, filesystem, observer/tool
versions, privileges, resource budget and operator/reviewer. A cross-build or
ubuntu PR job does not qualify a row. No existing harness supplies demonstrated
independent Q6 observer coverage. The operator must select/calibrate actual
host controls before execution; no undocumented vendor flag or sandbox feature
is presumed.

Preparation host: verify acquisition of immutable A/B archives and checksum sets,
source SHA, format, version, binary/descriptor, inventory/provenance/man-map and
embedded/installed skill identities. Record archive hashes on transfer/read-back.
Include same-label/different-content variants, pinned genuine v1 artifacts and
each claimed v0.1.0 historical variant with exact provenance. Missing historical
bytes or eligibility keeps the floor **UNVERIFIED**. Current fixtures and the
pinned v1 installer at H1 test selected compatibility only.

Execution host: copy trusted local inputs before denying access, retain the
verified installer as standalone tooling, prepare private disposable prefixes A/B,
separate receipt/state/skill roots, CWD with spaces, user/foreign sentinels and
before snapshots. No checkout, maintainer home/config/memory or prior sessions.
Run the copied installer with explicit local `--archive`, `--checksums`,
`--bin-dir` and `--receipt-dir`; use the installed binary's real `--help` for
further commands. Do not copy a working checkout onto the isolated host merely
to run a development harness. Archive extraction follows validated installer
handling, never arbitrary third-party archive instructions.

| Suite / existing preparation | Required oracle before PASS |
| --- | --- |
| V1 / corpus, coverage map, deterministic generator tests | All required classes and local links actually read; generated CLI/help/man parity, measured quotas and exclusions match exact artifact |
| V2 / `internal/operationaldocs/resolve_test.go` | Explicit A/B paths, switched PATH, spaces/aliases/copy/move, surviving A after replacement, same-label B/C2 and crossed skills/docs verify own association or refuse; imageProof stays unavailable |
| V3 / archive codecs, `archive_v2_test.go`, offline harness pinned old installer | Clean/repeat install, supported v1→v2 and every claimed historical variant; old/new shell and Go readers refuse unsupported/malformed input before candidate execution |
| V4 / `internal/install/docs_publish_test.go` | Real installer/readers and competing publishers with distinct receipt roots; process interruption/fault matrix from Plan V4, confirmed effects/read-back, no mixed publication; in-process seams alone remain PARTIAL |
| V5 / transition fixtures | Actual compatible independent A return with A docs/skills/man and preserved B/user snapshots; incompatible shared state and ordinary downgrade refuse. No automatic rollback |
| V6 / archive, resolver and anchored cleanup negatives | Unsafe paths/types/links/counts/ACLs/hashes and swaps at declared boundaries refuse; foreign sentinel/snapshots survive. ADR-0005 scope retained |
| V7 / generated pages and session recipe | Native reader actually selects and renders expected pages; PATH A/man B divergence detected, explicit A selection proven, upgrade/return/spaces/no-reader case exercised |
| V8 / offline development rehearsal | Independent observer proves all installed content reads, hashes and local link closure under effective source and network denial, including deliberate denied controls |
| V9 / Q6 below | Real fresh independent Codex and Claude sessions, effective controls and correlated independent observations; no synthetic substitute |

The generic `accept-native-artifact.py` currently targets prior release v0.10.0
and Linux lifecycle assertions; it is not complete docs/man/Q6 qualification.
Existing fixture models and historical native probes remain bounded to their
own sources/SHAs. Do not relabel them as these exact-artifact suites.

## Q6 execution and observer contract

Apply [all seven original controls](clarify.md#q6--reproducible-native-proof-protocol),
without replacing their requirements:

1. Record artifact/host/Runtime executable/version/digest and clean session
   identity; install only verified local inputs and controlled readiness fixture.
2. Run V8 separately, with all network denied and no Runtime. Record actual opens,
   corpus bytes/digests, local links and discovery read-back.
3. Start separate fresh native Codex and Claude sessions only after explicit
   authorization. Permit matched installed docs/skills, executable help, required
   system files and a controlled Project/readiness fixture. Deny documentary
   network tools and checkout/history/memories/configuration sources effectively.
4. An observer outside the Runtime records successful/denied canonical file opens,
   PID/parent/descendants, timestamps and network policy/connections. Calibrate an
   allowed installed read, forbidden planted source read, forbidden descendant
   read, denied documentary connection and (for V9) allowed inference connection.
   Missing descendants/lost events => UNVERIFIED; successful escape => FAIL.
5. Authorize exact inference/auth destinations separately. Record transport
   opacity and disabled retrieval capabilities; do not claim to observe opaque
   server retrieval, erase model knowledge or establish Runtime air gap.
6. Per Runtime and eligible native host, execute A, B, explicit paths, switched
   PATH/man, same-label/different-content, crossed skills/docs, absent and tampered
   inventory. The answer must explain a real command/readiness from observed
   installed reads/help, correlate a benign version-specific detail, and refuse
   verified documentary guidance on invalid state without fallback. A prompt or
   fluent answer alone cannot establish any control.
7. Independent reviewer reconciles sanitized access/denial summaries with output,
   fixture expectations and hashes/read-back. Preserve private raw observations
   outside Git; publish bounded summaries only. Failed/unavailable controls never
   become PASS by self-report.

For each scenario retain: host/Runtime, suite/case, artifact/source hashes,
policy/observer recipe and version, session/process correlation, expected result,
observed exit/state/reason, access/denial counters and gaps, before/after sentinel
digests, sanitized output reference, independent reviewer and PASS/FAIL/NOT_RUN/
UNVERIFIED. Credentials, personal paths and raw private logs are excluded.

| Native host | T13 V1–V7 | T14 V8 controlled | T15 V9 Codex | T15 V9 Claude |
| --- | --- | --- | --- | --- |
| macOS arm64 | NOT_RUN | NOT_RUN | NOT_RUN | NOT_RUN |
| Linux amd64 | NOT_RUN | NOT_RUN | NOT_RUN | NOT_RUN |
| Linux arm64 | NOT_RUN | NOT_RUN | NOT_RUN | NOT_RUN |

Runtime eligibility/observer feasibility must be verified per row; unavailable
combinations need human disposition, not a silent matrix reduction. T14 remains
PARTIAL overall from development rehearsal. T16 remains PARTIAL pending this
qualification and technical review; no AC-08 PASS or product acceptance follows.

## One grouped human decision — pending

| Decision | Minimum recommendation; alternative and trade-off | AC/gate effect |
| --- | --- | --- |
| Hosts/observers/Q6 | Authorize bounded disposable three-host qualification and fresh Codex/Claude sessions, naming operator, independent observer/reviewer, effective policy and subscription/credential budget. Alternative: defer execution and retain NOT_RUN; saves cost, leaves native claims unproven | Enables T13–T15, AC-01–08 proof; controls must still pass |
| Archive v2/default/bootstrap | Approve a subsequent bounded coordinated change of producer default, `install.sh` and release consumers, with canonical v2 `macos` token and explicit v1 `macos-27` compatibility. Alternative: retain v2 opt-in; fewer distribution changes, normal release path still lacks docs | Needed for ordinary distributed AC-01/05/06/07; naming/default is a lasting compatibility choice. No publication granted |
| v1 concatenated gzip | Require preservation of historical v1 multi-member reading, confined to v1, with regression proof; alternative: explicitly accept first-member-only behavior after historical artifact inventory. Preservation costs a bounded fix; narrowing risks rejecting old inputs | AC-05; current behavior/decision remains unchanged pending authority |
| Residual lock recovery | Retain fail-closed manual recovery only after proving no live publisher, exact lock/namespace identity and preserved snapshots. Alternative: separately specify owner-aware recovery; broader race/state contract. No blind removal or automatic cleanup | AC-03/05 recovery disposition; no removal authorized here |
| T04 sufficiency | Accept bounded author internal review plus review-7 CR-002 scope as an internal checkpoint only, then require independent final implementation/native Evidence review at T16. Alternative: commission independent T01–T03 audit before qualification; stronger independence, more cost | T04/E04 disposition; neither option makes whole PR/native ACs accepted |

No decision above is enacted by this document. Until resolved, keep default v1,
existing gzip/lock behavior, historical C2 FAIL, v0.1.0 UNVERIFIED and no GC or
automatic rollback. Resume this PR at its freshly verified head after the grouped
decision; reconcile concurrent changes first. Do not create another Execution,
PR, dispatch or successor work item. Merge/release/human acceptance/Done remain
outside authority.

## Observed preparation Evidence

Environment: author's macOS arm64, Darwin 27.0.0, Go 1.26.1; not an independent
native acceptance host. No real Runtime session, host provisioning, network/path
denial or user-environment change was performed. Source head for these precommit
checks is the entry SHA plus this harness/documentation delta.

- PASS: `go vet ./...`, `go build ./...`, focused
  `go test ./internal/operationaldocs ./internal/install ./internal/cli` (cached).
- PASS: `bash -n scripts/test-docs-offline.sh`, `--help`, unknown option,
  relative/missing input and missing paired-option refusals.
- PASS: default development harness and supplied-archive harness, including all
  four resealed negatives, pinned v1 refusal, copy/tamper/modified-binary refusal,
  26 corpus files/closed local links, and final exit 0. Supplied artifact and
  checksum-file SHA-256 read-back unchanged; bad supplied checksum refused.
- PASS: repository validator, ADR governance (35 ADRs), local file-link check and
  `git diff --check`. Final staged scans and exact-head CI go in the PR receipt.
- One initial harness execution was invalidated by editing the running script
  (shell EOF); it is discarded. The subsequent stable-script runs above passed.

Supplied development rehearsal (`release=false`, `sourceState=dirty`, version
`0.0.0-postreview7`, source `78ecd6cac911d6f53726bd51cda7b2817413365b`):

| Identity | SHA-256 |
| --- | --- |
| Archive | `4f05c3c353f4e168fc49ceca8d6b698f72a3d47c7ae94631547627ed13b8d01c` |
| Binary | `8f3eaeeafa830bd3acd026c9f073dc81cf664023797ae93e6fd79003f543bf10` |
| Generation | `61d9a5fdd763e6b0b0619fc4b8684542e868b6cb0c88e0fcb4a7988d0e3ac6b2` |
| Inventory | `26bfb00eb04b93fa77112d1d5a11b519649354cd644865b101c20acd659a90bd` |

29 inventory/control files, 414828 content bytes; never reuse these development
hashes as the eventual release/test qualification identity. The final-head
artifact, test/CI outcomes and recoverable commit are recorded in the linked PR
completion receipt, avoiding a self-referential source/hash claim.

| Task/Evidence checkpoint | Current status |
| --- | --- |
| T01/E01, T05/E05, T06/E06, T08/E08, T10/E10, T11/E11 | Retain scoped PASS in prior Evidence; no promotion to native acceptance |
| T02/E02, T03/E03, T04/E04, T07/E07, T09/E09, T12/E12 | PARTIAL, prior limits retained; T04 disposition still pending |
| T13/E13 | NOT_RUN; exact-artifact lifecycle/man/migration on three controlled hosts missing |
| T14/E14 | PARTIAL; development checks passed, effective source/network denial missing on all hosts |
| T15/E15 | NOT_RUN; fresh real sessions, independent Q6 controls/observations missing |
| T16/E16 | PARTIAL; preparation/traceability ready, full qualification/review absent |

AC-01/02/03/05/07 remain PARTIAL, AC-04 and AC-08 NOT_RUN; AC-06 retains scoped
reference-parity PASS, not full distribution qualification. V1–V7/V9 native
qualification NOT_RUN, V8 PARTIAL only in the development sense above. Final
handoff gate: one grouped human disposition of the five decisions, then the
authorized controlled proof, not merge or delivery completion.
