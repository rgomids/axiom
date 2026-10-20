# Implement Evidence 010 — Resumption 2, qualified D1/D3 acceptance

**2026-10-20 — P1 PARTIAL / G1 BLOCKED; checkpoint, not Implement completion.**
[Acceptance reconciliation and pending C1](acceptance-resume-2.md) supersede the
operational approval blocker in [resumption 1](evidence-resume-1.md), without
rewriting its measurements or accepted revision-1 contracts. No production code,
codec, generator, lock, public discovery command or final corpus is added.

## Preflight and authority

- Local branch, remote PR head and latest local commit agreed on
  `f96d5dfb8efb4b4d6018b7a523e89532734c006f`; clean worktree at entry. Same branch
  `docs/axm-37-offline-operational-documentation`, sole OPEN/Draft PR #335.
- Re-read full human comments [6301616799](https://github.com/rgomids/axiom/pull/335#issuecomment-6301616799)
  and [6298186645](https://github.com/rgomids/axiom/pull/335#issuecomment-6298186645).
  T01–T04/P1 first; no inferred G1, functional acceptance or successor authority.
- Re-read AXM-37 through Linear: In Review, blockedBy empty, blocks AXM-39 and
  AXM-12. Status/relations unchanged. Existing PR/implementation continuation
  retained; canonical Execution remains unknown, not recreated or dispatched.
- Previous-head [CI](https://github.com/rgomids/axiom/actions/runs/39495410176/job/118879448827)
  and delivery-metadata checks SUCCESS, now verified rather than assumed. These
  runs do not prove the new head; final delivery comment records new-head CI.
- Latest Linear prompt record `413fcdba-4b91-4bef-ae78-48b306be9940` matches this
  resumption; previous prompt/gate receipts do not establish a canonical Execution.
  No newer conflicting human PR instruction observed before publication.
- Applicable AGENTS, implementation/orchestration/security/automation policies,
  relevant owners, Spec/Clarification/Plan/Tasks and ADR-0033/0005/0016/0018/0021
  constrain this checkpoint. Runtime/skill formats, ownership history and Windows
  withdrawal remain unchanged. Stdlib hashes, bounded passive tar reads and existing
  fixtures suffice; no dependency/parser/cryptographic mechanism introduced.
- No delegation: model overrides exist, but this runtime supplies no demonstrable
  strictly inferior capability ordering. Main Orchestrator performed bounded
  measurement/integration. No independent review or human mechanism acceptance claimed.

## E01 — coverage, initial quota budget and historical baselines

[Machine-readable measurements](measurements-resume-2.json) bind all inputs to the
unchanged source head above. No raw provider trace or personal path is included.
Public repository visibility was checked before recording public release asset
metadata/hashes. Private Linear description/comments were not imported.

| Measurement | Observed result / limit |
| --- | --- |
| Existing public CLI leaves / flag registrations | 105 / 549, exact path/default parity with revision-1 snapshot PASS |
| Rendered existing help nodes / UTF-8 bytes | 124 / 160,546; includes groups/root; per-node bytes/hashes recorded |
| Selected existing source inputs | 177,842 bytes; five public files with exact paths/bytes/hashes in JSON |
| Planned output count | 11 text + 15 shared man = 26 versus initial budget 64; planned count, not generated output |
| Explicit scenario estimate | `2*177842 + 4*160546 + 26*4096 = 1104364` bytes versus initial 8,388,608-byte budget |
| Final authored corpus, generated reference/man and complete local-link/provenance map | NOT_RUN; no final byte/coverage freeze |

Estimate rationale: duplicate selected text inputs twice, reserve four copies of
all help for Markdown/man/escaping, then 4 KiB per planned output. This is a
reviewable scenario, **not an upper bound or final corpus measurement**. Unknown
authored journeys, docs-inspect help and renderer behavior remain unmeasured;
source size does not establish sufficiency. No command, flag or journey removed
to fit a quota. Current source-build binary/installer/license/skill overhead remains
the measurement in E01 resumption 1; no new release-archive total is claimed.

### Coverage measurement replay

Temporary package-local test `TestAXM37MeasureResume2` recursively walked
`publicCommands` in definition order, starting path empty; each child appends its
name. For each node, `renderHelp` wrote to `bytes.Buffer` and had to return 0.
Record `len(buffer.Bytes())` and SHA-256. For leaves only,
`commandFlagSet(operation).VisitAll` recorded sorted `Name+"="+DefValue`.
Prefix recorded paths with `axiom`. Compare exact leaf sets and ordered flags to
`cli-coverage-proposed.json`; record no duplicate copy of flag strings in new JSON.
`go test ./internal/cli -run '^TestAXM37MeasureResume2$' -count=1` exit 0.
Temporary Go test removed after measurement; no production candidate executed.
The public command/help owners and pinned input hashes make this recipe replayable
without importing a new generator. Final T06 generation remains blocked.

### Historical baseline scope and actual tests

All **25 stable local release tags v0.1.0–v0.20.0** were inventoried by exact
peeled commit and SHA-256 of available builder/facade/archive/upgrade source.
Missing historical source files are JSON null, never invented. This is source
provenance, not a statement that every source hash denotes a distinct wire variant.
Existing published Codex/Claude receipt fixtures cover **11** release tags:
v0.1.0, v0.1.1, v0.1.2, v0.2.0, v0.2.1, v0.3.0, v0.4.0, v0.4.1, v0.5.0,
v0.6.0, v0.15.0. Exact fixture bytes/hashes retained without modification.

`go test ./internal/codexruntime -run
'^TestEveryPublished(ReceiptIsRecognizedAsAxiomOwned|SkillDigestIsKnown|ClaudeInstallationConverges)$'
-count=1 -v` exit 0. This positively tests existing receipt recognition and
registered skill digests, including v0.1.0. The Claude convergence test skips
seeding historical skill bytes absent from the current embedded set; it exercises
missing-skill convergence, **not full exact historical installations or real
Runtime sessions**. Codex receipt recognition is not Codex upgrade qualification.

All three public v0.1.0 archives plus SHA256SUMS were downloaded read-only via
`gh release download v0.1.0 --repo rgomids/axiom --dir <scratch>` (exit 0).
Passive bounded Python tarfile reads, without extraction/execution, checked each
archive checksum, safe single-root regular-file/directory names, exact manifest
closure and all member hashes. Collector bounds: compressed <16 MiB, <=64 headers,
each member <=16 MiB, total payload <64 MiB; links/special files refused.
All three PASS. Metadata, five skill names, binary/installer/member hashes and
asset URLs are in JSON. SHA256SUMS matches demonstrate integrity, not independent
release authenticity. Platform labels are historical macos-27/ubuntu-26.04, not
new v2 labels. These Linux assets were inspected on macOS, **not natively run**.

No v2 consumer exists yet, so all historical **native v1→v2 migrations remain
UNVERIFIED**. The 14 stable tags without retained receipt fixtures are explicitly
UNVERIFIED for fixture recognition here. Intermediate/prerelease variants and
installation receipts distinct from Runtime receipts still need positive
variant-specific Evidence. No new historical floor or compatibility claim is
declared; v0.1.0 remains the accepted objective. A complete migration matrix needs
the future v2 consumer and native tests, distinct from a pinned T01 test oracle.

`go test ./internal/install -run
'^(TestLoadCandidateMirrorsInstallerVerification|TestArchiveSkillSetIsTheEmbeddedSkillSet|TestUpgradeFromSixSkillReleaseAddsOnlyDomainSkills|TestUpgradeFromSixSkillReleaseNeverAdoptsForeignOrModifiedSkills)$'
-count=1` checks existing v1 validation and six-skill upgrade preservation only.
Synthetic fixtures are not released archives or a new shell/Go v2 matrix.
Command exit 0; all four selected existing installation tests PASS.

Local document validation: `./scripts/validate-repository.sh .` exit 0; Python
assertions for help counts/snapshot parity, unchanged v1 source pins, all retained
receipt fixture hashes and unqualified migration/gate classifications exit 0.
No new behavioral production tests are necessary for this documentary checkpoint.
Staged sensitive-file/secret scans and final diff checks accompany publication.

## E02 — native host matrix retained honestly

No new T02 execution after identifying T01's unresolved freeze dependency.
Detailed OS/API/filesystem/binary/harness hashes and job URLs remain in
[resumption 1 results](probe-results-resume-1.json). No native result relabelled.

| Environment | Executed prior primitive evidence | Full T02 reader/A-B/alias qualification |
| --- | --- | --- |
| macOS arm64 / APFS | PASS, image identity + eight cases + single-file reader/flock | NOT_RUN |
| Linux amd64 / native Ubuntu runner | PASS, same bounded primitives | NOT_RUN |
| Linux arm64 / native Ubuntu runner | PASS, same bounded primitives | NOT_RUN |
| Docker Linux arm64 / overlayfs | PASS, complementary VM/container row | NOT_RUN |
| Docker Linux amd64 emulated on arm64 | FAIL, image identity counterexample | Remaining cases NOT_RUN; never native Evidence |

Independent PATH A/B mutation after launch, full-generation inventory/content
pinning and late read revalidation, unsafe aliases, TOCTOU/ancestry safety and
failure/refusal oracles still need synchronized per-host experiments. Existing
pathname-only oracle is rejected; no pathname-only fallback adopted. Safe alias
policy and final OS algorithm remain unqualified. T03 cannot consume a selected
mechanism from these primitive measurements.

## E03 / E04 — gate submission and disposition

| Unit | Current status | Outstanding proof / exact unblock |
| --- | --- | --- |
| T01 / E01 | PARTIAL, BLOCKED at final freeze | D1/D3 exact acceptance reconciled; resolve C1 quota-freeze dependency, complete preserved historical variant/closed pre-execution fixture oracles; no false v0.1.0 support claim |
| T02 / E02 | PARTIAL, BLOCKED by T01 and remaining native proof | Adequate T01, then complete native image/full reader/alias matrix and mechanism review on all three hosts |
| T03 / E03 | BLOCKED, NOT_RUN | T01/T02 adequate; installation-derived lock across different receipt roots, concurrent publisher/revision, ordered commit/fault/read-back/recovery fixtures; primitive flock is insufficient |
| T04 / E04 | PARTIAL; G1 submitted BLOCKED | Fan-in explicitly incomplete; E02/E03 and material mechanism review/disposition still missing |
| T05–T16 | BLOCKED, NOT_RUN | G1 not passed; no production P2+ started |
| G2 / G3 | NOT_RUN | Later qualified artifacts/observers/hosts and reviewed readiness evidence |
| AC-01–AC-08 / V1–V9 | NOT_RUN | No full acceptance protocol executed; baseline tests/passive inspections cannot close ACs |

**C1:** final public quota freeze requires T05/T06 corpus/reference measurement,
while those tasks require G1's P1 freeze. Current acceptance does not explicitly
dispose this ordering conflict. The exact two alternatives and recommendation are
reviewable in [acceptance-resume-2.md](acceptance-resume-2.md#c1--minimum-human-decision-requested-quota-freeze-dependency).
No silent deferral, approximation-based G1 PASS, scope expansion, further dependent
experiment, automatic SDD review or functional acceptance.

## Recoverable handoff

- Objective: continue the same AXM-37 Implement in Draft PR #335; complete P1 and
  submit G1 only with complete reviewed proof and required human dispositions.
- Completed: exact D1/D3 reconciliation; E01 source/help estimates and receipt
  regression tests; bounded passive oldest-release integrity; E04 blocked fan-in.
- Remaining sequence: human C1 disposition → T01 contract/fixture closure → T02
  native reader/identity/alias proof → T03 lock/commit/recovery fixtures → T04 review
  and material disposition → evaluate G1. No new Execution or PR.
- Human decision needed now: C1's named gate-order alternative. This alone does
  not accept G1; D2/T03 final algorithms still need proof and material review.
- Authority retained: scoped edits/tests/disposable probes/normal commit/push and
  same Draft PR. No status/blocker changes, merge/release/closure or AC acceptance.
- Validation and exact delivered head/CI are recorded in the PR checkpoint comment.
  No dependency-sensitive task may use previous-head green CI as current Evidence.
