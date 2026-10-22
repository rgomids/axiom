# Plan 010 — Installation-bound operational documentation

## Status, authority and boundaries

**Proposed — plan only, revision 1, 2026-10-20.**
[AXM-37](https://linear.app/rgomids/issue/AXM-37), sole Draft [PR #335](https://github.com/rgomids/axiom/pull/335),
branch `docs/axm-37-offline-operational-documentation`.
Baseline inspected: `c7ef060d71849c9dc82275c23f0ec9f9256f97a0`.
The [gate READY_FOR_plan](https://linear.app/rgomids/issue/AXM-37#comment-deff827e)
and current executor instruction authorize this documentary proposal and delivery.
They do not prove canonical Execution, which remains **unknown**. Existing gate,
prompt and dispatch checkpoints were read; no restart, dispatch or duplicate is needed.

Authority inputs: [Specification rev.1](spec.md), accepted by its exact
[human decision](https://github.com/rgomids/axiom/pull/335#issuecomment-6292970482);
[Clarification rev.2 / Q1–Q6 accepted directions](clarify.md#decision-register);
[ADR-0033](../../decisions/0033-installation-bound-operational-documentation.md),
accepted by the distinct [exact-revision decision](https://github.com/rgomids/axiom/pull/335#issuecomment-6296075261).
ADR status reconciliation changes only its Status section. Historical Spec,
Clarification and Evidence bytes are preserved. [Plan Evidence](evidence-plan.md)
records checks separately from future qualification.

No Tasks, implementation, native acceptance, API/schema adoption, merge, release,
Linear status/relation changes or AXM-39/AXM-12 unblocking occur here. Sections
below describe future work, conditional on Plan acceptance and separate authority.
P1–P6 are planning slices, not Task IDs or persisted workflow states.

Axiom != Lingo; Project != Repository; Role != Model; Execution != Agent;
Provider != Transport; Integration != MCP; Skill != workflow source of truth;
Evidence != chat history. Consultation cannot grant execution authority.

## Integration map and responsibility

| Existing owner / concrete files | Observed boundary and planned responsibility |
| --- | --- |
| `scripts/build-release-archives.sh`, `scripts/verify-release-artifacts.sh` | Three release rows; binary, LICENSE, installer, strict metadata, skill inventory and SHA256 manifests. Add explicitly enumerated operational outputs and provenance after format decision; verify prepared and published exact bytes |
| `scripts/install.sh`, `scripts/install-release.sh` | HTTPS acquisition followed by local checksum/archive validation; shell verifies before executing candidate. Its closed metadata/entry rules must evolve together with Go; preserve local artifact installation and existing host checks |
| `internal/install/archive.go` | `LoadCandidate`, bounded regular-file archive reader, exact metadata/skill inventories. Introduce version-dispatched documentary validation; do not loosen v1 or execute unverified candidate |
| `internal/install/fresh_posix.go`, `internal/install/upgrade.go` | Fresh/owned upgrade, exact targets, markers/locks, preview effects/digests, binary publication, receipt and skills. Integrate generation preparation before binary commit, read-back and truthful recovery; keep compatibility preservation boundaries |
| `internal/local/install_posix.go`, `internal/local/file_identity.go`, `internal/local/directory_identity.go` | Existing anchored publication directories and object checks. Reuse for confined reads/publication; extend only for demonstrated identity gaps |
| `internal/provenance/provenance.go`, `cmd/lingo` | Current build identity uses shortened revision; add separately versioned full documentary source/build identity without changing existing provenance fields by inference |
| `internal/cli/commands.go`, `internal/cli/help.go`, `internal/cli/completion.go`, `internal/cli/skill_inspect.go` | Custom Go command/flag tree and canonical JSON/human presentation, no Cobra. Generate reference from this tree; add one read-only discovery adapter under reviewed interface contract |
| `internal/codexruntime/integration.go`, `internal/codexruntime/skills/*/SKILL.md` | Same integration package serves Codex and Claude; four active skills. Consume discovery/help, preserve historical ownership sets, distinguish expected/installed/loaded and separate Runtime convergence |
| `scripts/install-axiom.sh` | Source install has a distinct receipt/build path. Must explicitly report documentary delivery as unqualified until separately supported; never borrow release docs or rewrite source receipts |
| `scripts/test-release-archives.sh`, `scripts/test-install-posix-facade.sh`, `internal/install/*_test.go` | Extend existing archive/facade and failure seams with v1/new-format differential fixtures and lifecycle/readers tests |
| `scripts/accept-native-artifact.py`, `.github/workflows/native-suite.yml`, `.github/workflows/native-artifact-acceptance.yml`, `.github/workflows/native-acceptance-qualification.yml`, `.github/workflows/ci.yml` | Extend exact-artifact qualification, not publication authority. Add docs/man host rows and separately controlled native Runtime evidence; never count fake Runtime tests as AC-08 |

Proposed new responsibility: a bounded operational-document package (candidate
`internal/operationaldocs`) owns inventory, identity validation and read-only
resolution. `install` owns mutations; CLI owns presentation. No service, database,
global registry, parallel skill resolver or new third-party dependency. New path
is proposed, not an existing capability. Durable new generators/harnesses require
registration in `scripts/automation-registry.json` under existing ownership policy.

## Material design dispositions before Tasks

ADR-0033 accepts direction but deliberately leaves concrete wire/layout/mechanism
open. These are **recommendations for human review, not adopted contracts**.
Do not create downstream Tasks for dependent production work until dispositions
are explicit. A conflicting change needs a new Proposed ADR and separate acceptance
under [ADR-0018](../../decisions/0018-adr-evolution-and-supersession-governance.md);
never edit older accepted decisions. Q1–Q6 are not reopened.

| Decision / authority gate | Bounded recommendation and alternative | Trade-off / hard-to-change consequence |
| --- | --- | --- |
| D1 — package, metadata and receipt evolution: HUMAN_DECISION_REQUIRED | Recommend explicit new archive/metadata generation, strict codecs and fixtures; candidate label v2, with separately versioned local binding/discovery. Alternative: different new envelope with separate docs manifest. Retain unchanged v1 read path for history; never append fields to v1 | Every producer/consumer must agree before shipment; historical bytes and ADR-0016 Runtime constants stay independent. Exact schemas, numeric versions, encodings and quotas need approval before code |
| D2 — bootstrap and executing-image proof: HUMAN_DECISION_REQUIRED | Recommend deterministic private sibling namespace associated with physical installed binary leaf, immutable records keyed by verified executable digest plus generation identity. Example topology only: `<binary-parent>/<private-install-namespace>/<generation>/`; receipt target remains independent. Alternative: compiled relative locator plus verified binding. No global search/registry | Sibling placement must satisfy eligible publication-directory permissions without chmod/ownership repair. Binding persists installed target identity; copying/moving requires explicit authorized rebind. OS image proof and surviving-reader strategy must be demonstrated on all host rows before adopting paths/algorithm |
| D3 — discovery wire and man mapping: HUMAN_DECISION_REQUIRED | Recommend one command beneath existing CLI tree, illustrative `axiom docs inspect`, with existing `--json` / `--human` presentation selection. Alternative: an inspection child under another existing group. Propose versioned result described below and root/shared man coverage from same command graph | Command names, JSON keys/status/exit semantics, page names and public layout become compatibility contracts. No command is created here; no global completion-envelope change is implied |
| D4 — generation commit/recovery detail: HUMAN_DECISION_REQUIRED | Recommend stage complete generation and executable binding under anchored owned root, publish immutable objects before binary rename, then confirm binary effect; reuse locks/expected revisions with explicit installation identity across independent receipt roots. Alternative: stronger prevalidated executable-derived locator if platform proof requires it | Root/lock/marker identity, ordering, stale-process proof and uncertainty recovery require reviewed state-transition fixtures. Cannot claim receipt-only locking protects same binary target with different receipt roots. No global transaction or post-commit rollback |

Generator recommendation: use Go standard-library rendering from the existing
command/flag tree plus dedicated text; no framework migration. Alternative existing
external man generator would add a dependency/build trust surface and requires
explicit adoption. Rendering choice may be accepted with Plan if no public contract
changes. Proposed corpus budget: <=64 text/man outputs, <=1 MiB each, <=8 MiB total
uncompressed documentary content; manifest <=256 KiB. Approve/tune against actual
command coverage before implementation; update archive entry/aggregate limits
explicitly rather than silently inheriting current 64 total entries / 1 MiB file
limits. Never discard required command coverage to fit a budget.

## Corpus, generation and identity contracts

Dedicated English corpus candidate: `docs/operational/`, with an explicit allowlist;
not a wildcard over `docs/`. Names below are proposed output roles, not committed
files in this stage.

| Output | Canonical input / minimum journey |
| --- | --- |
| Local index and concepts | Dedicated authored synthesis of product/domain boundaries, authority and observed versus unknown; concise source links recorded at full source commit |
| Installation and verification guide | Release/local install prerequisites, identity read-back, legacy/source limits, independent destinations and A/B selection; synthesize current installation contracts |
| Operational guides | Project, Runtime, Workflow and Work Item inspection/setup journeys; external effects and network requirements explicit, help authoritative |
| Generated CLI/configuration reference | Actual `publicCommands` and `commandFlagSet` contracts, registered options/defaults/required values, public help parity; versioned configuration descriptions from owning contracts, no inferred flags |
| Troubleshooting/recovery | Missing/corrupt docs, stale binding, partial Runtime state, interrupted publication, compatible return and safe next action; no autonomous repair instruction |
| Man root/public command pages, coverage map | Same canonical command reference; map every public operational leaf to own or shared page, justify shared coverage, exclude internal/test commands |
| Inventory, provenance and notices | Canonical sources/full commit, transform version/options, source and output hashes, licenses/attributions and explicit exclusions |

Required journey links must close within delivered inventory; optional external
references are labelled network-only and never followed automatically. Escape
Markdown/man control syntax, control characters, terminal escapes and troff requests
so source text cannot inject executable reader instructions. No historical Spec,
ADR, Evidence, maintainer harness, private Provider data, secrets or personal paths.

Build order avoids self-hash cycles: freeze source corpus + command/flag descriptor
and transform identity; derive documentary generation descriptor; build platform
binary with a non-self-referential descriptor where approved; hash final binary;
produce external inventory/binding inputs; hash bundle; then outer archive checksum.
Generation identity is carried in detached metadata when embedding it would create
a self-hash cycle. It covers canonical ordered path/hash inventory, full source commit,
transform identity, command map and expected embedded skill inventory. Archive digest
is recorded externally/local installation metadata, never embedded into bytes whose
hash defines that archive. Same version strings do not identify same bytes.

Release acquisition trusts the authorized exact repository/tag/assets over verified
HTTPS using existing bootstrap policy, then checks external archive SHA256SUMS and
internal inventory. Local mode requires a previously trusted archive + checksum
pair and records acquisition provenance; a caller-controlled archive and matching
manifest alone are only integrity, not authenticated origin. No new signing service,
credentials or stronger attestation guarantee is presumed. Verification must precede
candidate execution, including the shell facade. Full source commit is a new scoped
contract: preserve current abbreviated public provenance behavior unless separately
approved. Dirty/source builds cannot masquerade as official release assets.

## Producer-consumer and migration matrix

`N` below means the new format after D1 approval, not a chosen numeric version.

| Producer / consumer | v1 historical artifacts | N documentary artifacts |
| --- | --- | --- |
| Existing builder / existing Go and shell | Preserve current supported behavior and fixtures; no docs compliance | Reject unknown format/extra entries before candidate execution; must test with pinned old consumer |
| Updated builder / updated Go and shell | Do not rewrite existing releases or silently emit new fields as v1 | Emit/consume only explicitly versioned closed schema, complete docs/man hashes and binary binding inputs |
| New Go consumer / v1 archive | Preserve supported legacy installation semantics, docs absent; no manufactured generation/Runtime receipt | Validate exact metadata/inventory, limits, release row and identity; publish only after authority/read-back |
| Historical installed receipt / new installer | Read only known historic variants for ownership and explicit supported upgrade; preserve preimage in evidence/compatibility contract | New receipt/binding publication only as previewed effect, with exact expected revision; reject unrecognized version |
| Older installer / new receipt or marker | No backward compatibility claim; protect earlier installation by refusing unknown state | No downgrade in place; explicit separate A remains return path |
| Runtime receipts / any archive | ADR-0016 candidate-running proof and historical ownership remain authoritative | Documentary manifest does not replace Runtime constants/receipt; candidate-derived Codex receipt only under existing proof; Claude first-run separate |
| Source/raw copied binary / discovery | Help/version remain, docs absent or unqualified | Initial rollout recommendation excludes source installs from qualified docs; future source-derived package needs own approved matrix and tests |

Migration is an explicit authorized upgrade from recognized owned state, including
new docs/binding/receipt effects in preview digest. Preserve old formats and foreign
files; no background migration during inspection. An older running binary may reject
N; documented route uses N's verified shell facade/candidate, never weakens validation
or assumes every old binary can consume new archives. Bootstrap release selection
must use matching verified facade; old bootstrap/new archive rejection is documented.
Before first N release, qualify earliest supported baselines, current v1, equivalent
N reinstall and N→N upgrade with exact producer/consumer versions/hashes.

## Resolution, publication and recovery

Read-only resolver starts from OS-observed executing-image identity, validates
supported alias/symlink chain to physical binary and bounded ancestry, and obtains
only that target's private binding. `argv[0]`, CWD, repeated PATH lookup, Project,
XDG/global roots, same version and arbitrary absolute metadata pointers are not
proof. Aliases may grant read resolution only; install destinations keep existing
symlink restrictions. Reject unsafe hard links/types/roots, escaped names and
changed identity at declared boundaries. Cache only within a proven pinned snapshot;
revalidate before returning paths/reading files. Reading a path later is not proof
of its earlier bytes; consumers must verify selected inventory or use pinned reads.

Identity proof must distinguish a surviving process from a replacement at its old
path. Candidate options: pin OS image/file identity at process start with rechecks,
or use a compiled generation descriptor plus independently validated immutable
binding and OS image proof. Reading/hash of current pathname alone is insufficient.
If proof cannot be established, explicitly refuse; do not serve replacement docs.
Linux/macOS primitives and limitations remain D2 qualification obligations, not
assumed capabilities. Changed targets invalidate prior preview authority.

| Lifecycle boundary | Required future mechanism / observed outcome |
| --- | --- |
| Inspection and authority | Capture binary, namespace, receipt and Runtime roots separately; include exact revisions, planned immutable objects and effects in preview digest; lock same installation even with differing receipt destinations; recheck before mutation |
| Stage | Confined private writes with bounded inventory/hash validation; stage complete docs/man/coverage/binding before binary. Insufficient space, short write, permission or sync failure preserves old set; no activation |
| Immutable generation publication | Publish only verified complete generation + per-executable binding; equivalent object is no-op after full read-back; differing/foreign object is preserved conflict. Old binary still resolves old generation |
| Binary publication commit | Existing anchored rename + effect confirmation is logical activation boundary. Verify candidate identity and prepared binding immediately before commit; read back binary identity. No mutable docs-current pointer needed to declare activation |
| Confirmed binary / later failure | Binary effect stays confirmed even if installation receipt, skill publication, marker acknowledgment or stage removal fails. Report exact committed/remaining effects; no rollback wording |
| Unknown binary effect | `publication_uncertain` / recovery required semantics preserved; inspect exact installed binary plus immutable bindings and residues. Do not retry with stale authority, delete lock blindly or report old/new from marker alone |
| Readers and publishers overlap | Reader pins proven complete generation or refuses/retries on detected change; stale second publisher fails expected-revision/lock checks. Never mix old text/new inventory or newer pathname identity with old process |
| Recovery | Bounded inspection first; exact verified candidate and recognized operation may resume only under supported current authorization. Fresh interrupted installs still require operator recovery. Unknown residues/foreign data preserved; no general cleanup/recovery capability invented |
| Retention | Keep every recognized generation, including unactivated complete residue. Existing authorized transient-stage handling must not delete retained generations, notes or foreign objects. No GC, automatic rollback or profile changes |

Compatible return AC-04: retain A in independent eligible destination before B
upgrade; inspect current Project/Runtime/state compatibility using A under explicit
authority. If incompatible, refuse return without migrating/reverting shared state.
Select A explicitly, verify its binary/docs/man and matching installed Runtime
skills/configuration; start fresh sessions to avoid loaded B skills. Shared skill
roots require separate authorized convergence or conflict; no invisible overwrite.
Keep B intact, exercise actual A consultation/allowed local inspection, verify data
preservation. Separately prove ordinary in-place B→A downgrade remains refused.

## Read-only presentation, skills and manual reader

Proposed discovery result semantic fields (D3 must freeze names/wire): schema
identity; actual binary target/image proof, digest/build/source; installation and
generation binding; validated root/index/inventory/digest; compatibility and reason;
expected skill hashes plus separately observed installed receipt/files per Runtime;
loaded skill observation or unknown; local man root/coverage/source identities.
Represent absence, invalidity, permission failure, unsupported format, concurrent
change and unavailable observations distinctly. Use existing canonical completion
and terminal rendering rules, bounded sanitized diagnostics and deterministic JSON;
review schema/exit fixtures without redefining global statuses. Invalid docs never
produce a success-shaped verified path. Help/version remain independently usable.

Discovery performs no download, writes, extraction, repair, auth lookup or Runtime
startup. Four active skills invoke this single interface and executable help; no
hardcoded namespace guesses or second flag grammar. Expected/installed/on-disk and
loaded facts are distinct. Skill text cannot self-certify readiness or bypass gates.

Generate local `man1` hierarchy before installation; each page identifies source
release/full commit and generation, with destination-specific binding supplied by
verified adjacent installed metadata. Supported session recipe is derived from
verified discovery output: `MANPATH="$verified_man_root:" man axiom` for readers
whose trailing-colon default behavior has been qualified. Quote paths with spaces.
Also qualify explicit local-page selection where supported; do not assume identical
flags across man implementations. Plain `man axiom` under deliberately configured
session must work. Unconfigured man may select B under PATH A: inspect actual reader
selection (native supported lookup) and page identity, compare with A, report mismatch
and select A explicitly. No shell-profile edits, persistent MANPATH, global symlinks,
`mandb`/system cache writes or install failure solely because man is absent.

## Incremental delivery and completion boundaries

| Slice / dependencies | Bounded deliverable and future exit Evidence |
| --- | --- |
| P1 — contract freeze; no predecessor | Resolve D1–D4 against accepted ADRs; versioned wire/layout/limits/preview/exit/reader state tables and producer-consumer fixtures. Feasibility plan for OS image and Q6 controls. Human acceptance of material choices; no production implementation while unresolved |
| P2 — corpus and generation; P1 | Explicit dedicated allowlist, source→output mapping, standard-library generator, command/man coverage and provenance graph. Two controlled builds yield same generated docs for same inputs; no requirement that current tar/gzip archives already be reproducible. Link/exclusion/budget/secret/help-parity tests pass |
| P3 — delivery and lifecycle; P1 + P2 | Coordinated builder, verification, shell/Go codecs, owned namespace/binding, fresh/upgrade/marker/read-back. Fault and concurrent-process/readers matrix passes; retained generations and ADR-0016 receipts remain correct; no public release yet |
| P4 — consumption; P1 + P3 | Read-only resolver, canonical CLI human/JSON, four skills and local man1 selection. Absence/corruption/alias/A-B/same-label mismatch and no-side-effect tests pass, help remains usable |
| P5 — exact-artifact qualification; P2 + P3 + P4 | AC-01–AC-07 all supported host rows, native man and independent return; AC-08 separate fresh native Codex/Claude controls with OS observation. Sanitized Evidence and limits reviewed; unavailable proof remains UNVERIFIED, never synthetic PASS |
| P6 — rollout readiness; P5 | Install/commands/runtime docs reconciled, migration and explicit recovery/return instructions verified, CI exact prepared artifacts traced to source. Human product acceptance and release authority remain separate; no predecessor completion auto-unblocks Linear |

Do not start a broad installer refactor or unrelated documentation cleanup. If
P1 reveals conflict or Q6 infeasibility, stop dependent work and present named human
decision; independently useful documentary analysis may remain proposed.

## Qualification protocol and traceability

All product rows below are **NOT_RUN** in this Plan. Future statuses:
**PASS** = executed with required reproducible evidence and controls;
**FAIL** = observed violation; **NOT_RUN** = no execution;
**UNVERIFIED** = execution/observations insufficient to prove claim.
CI documentary PASS proves none of these product rows.

Evidence record per row: source SHA + artifact/binary/inventory hashes, producer
and consumer versions, OS/build/architecture/filesystem, Runtime/version when used,
sanitized exact commands/fixtures, expected/observed result, exit/read-back,
file/network policy + observer coverage, controls, limits and reviewer disposition.
Keep raw sensitive traces outside public repository; retain sanitized bounded
summaries with artifact digests and reproducible collection instructions. Record
missing tooling/privilege as UNVERIFIED, without permission expansion/bypass.

| Step / OD coverage | AC coverage / required future Evidence |
| --- | --- |
| V1 — OD-01, OD-02 | AC-01, AC-06: allowlist sources/provenance/notice map, complete offline journey links; no historical/private files, generated CLI/man command/flag parity and deterministic docs outputs |
| V2 — OD-03, OD-04, OD-05, OD-06 | AC-02, AC-08: A/B explicit invocation, switched PATH, arbitrary CWD, independent prefixes/receipts, spaces, safe aliases, moved/copied binary and old process surviving replacement. Cross same-version different binary/docs/skills bytes; expected/installed/loaded unknown and mismatch, help primacy/no fallback |
| V3 — OD-07 | AC-01, AC-03: local trusted artifacts clean install/reinstall no-op, legacy absent/source unqualified, unknown formats/duplicate or extra fields/unsupported version/truncated metadata; compare old shell/new archive and new Go/v1 fixtures before execution |
| V4 — OD-08, OD-09 | AC-03, AC-04: inject stage short-write, space/quota/permissions, immutable-generation publication, pre-binary commit, rename/sync uncertainty, confirmed binary before receipt/skills/ack, stage-removal failure. Concurrent readers/publishers incl. same bin/different receipt roots; read-back proves whole old/new or refusal and correct committed effects |
| V5 — OD-08, OD-09, OD-10, OD-11 | AC-04: actual compatible independent A return with A skills/config/man and local observed state; B retained, snapshots unchanged; incompatible shared state refusal and separate ordinary downgrade refusal |
| V6 — OD-10, OD-11 | AC-05: traversal/absolute/duplicate archive paths, symlink/hard-link/special types, ancestry/leaf replacement at supported boundaries, unsafe ownership/mode/ACL, count/size/decompression limits; missing/extra/changed files, altered manifest, crossed generation and same-label bytes. Verify external sentinel/notes/foreign files unchanged; no silent authenticity claim from rewritten hashes |
| V7 — OD-04, OD-05, OD-12, OD-13 | AC-07: native `man axiom` and mapped public subcommands in qualified session; actual reader path/page identity, PATH A/man B mismatch, explicit A selection, upgrade/return and no-reader Markdown/help case |
| V8 — OD-01, OD-03, OD-07, OD-13 | AC-01: disposable host/account, trusted artifacts copied in before isolation, prerequisites installed; no checkout/mount/maintainer home, empty controlled CWD/config, network denied by OS. Install/discover/read all minimum content; capture file opens/bytes and denied network controls. No Runtime needed |
| V9 — OD-03, OD-04, OD-05, OD-06 | AC-08: separate fresh native Codex and Claude sessions; protocol below with file/network observation, controls and A/B/absent cases. No string test, fake runtime or model claim substitutes for native execution |

Host product matrix: run V1–V8 applicable native portions on macOS arm64,
Linux amd64 and Linux arm64; cross-compilation is not host acceptance. Record OS
versions, filesystem and reader implementation actually tested; no all-filesystem
claim. V9 records each supported Runtime's real host eligibility; qualify each
Runtime on declared supported host rows, mark unavailable combinations NOT_RUN or
UNVERIFIED and obtain human disposition before claiming full matrix. Windows
remains withdrawn under [ADR-0021](../../decisions/0021-windows-server-amd64-native-acceptance.md#status),
not an implicitly skipped supported row.

### V9 — native Runtime source-access proof

1. Prepare disposable clean machine/VM or isolated account for each Runtime.
   Record native executable/version, session identity and descendant process tree.
   Install only trusted exact A/B artifacts, matching runtime skills and controlled
   local readiness fixture. No checkout, maintainer memory/config mount, historical
   chat/session, external documentary MCP, search/browser tool or retrieval cache.
2. Before prompts, establish effective OS path denial and independent file-access
   observation (successful and denied opens, canonical paths, process/descendants,
   timestamps) plus OS network policy/connection/DNS observation. Linux candidates:
   isolated mount namespace and audited syscall/process/network collection;
   macOS candidates: disposable VM/account with effective filesystem/network rules
   and host observation. Tools/privileges must be actually verified, not assumed;
   lack of supported observation makes result UNVERIFIED. Account separation alone
   or deleting a checkout is insufficient proof.
3. Calibrate controls under the same policy and process tree: known allowed file
   read and allowed inferencing connection observed; planted forbidden local source
   read, descendant read and disallowed documentary network request denied and
   observed. If encrypted traffic cannot distinguish allowed inference from remote
   documentary retrieval, record opacity; do not claim its absence. No collection
   of auth contents or credential changes.
4. Permit declared vendor inference/auth transport separately from documentary
   sources, with exact endpoints/observed limits. No claim of Runtime air gap,
   model-prior erasure or invisible vendor-side retrieval exclusion. AC-01 carries
   the actual offline proof, independently of Runtime network requirements.
5. Fresh Codex session consumes installed skill/discovery/help to explain command
   and observed local readiness, citing local generation/paths. Repeat independently
   in fresh Claude session. Repeat each with A, B, same-version different bytes,
   shared-root skill mismatch and docs absent/corrupt; expected outcome is honest
   unknown/refusal with executable help, no checkout/web fallback or invented authority.
6. Compare output citations/identity with OS opens and policy/connection records;
   preserve controlled fixture expectations, controls, sanitized commands, observer
   coverage/gaps and independent reviewer assessment. Transcript alone never PASS.
   Failed controls = FAIL or UNVERIFIED as appropriate. Infeasible Q6 proof requires
   new human decision before weakening AC-08 or declaring product acceptance.

## Rollout, drift and next gate

First qualify unpublished N artifacts against preserved historic fixtures and all
host rows. Do not publish until consumers, bootstrap/facade, docs/man, Runtime
receipt handling and migration instructions pass exact-artifact checks and normal
release gates. Initial qualified path is release-archive installs; source installs
remain explicitly unqualified under the D1 proposal. No release/tag is created here.

Drift checks compare executable proof, source/build identity, binding, generation,
full inventory and expected/observed skill sets, with bounded reads on consultation.
Corruption or uncertain state fails closed; diagnostics expose phase/confirmed effects
and authorized next inspection, never download fallback. Independent preserved A is
operational return only after compatibility proof; no automatic rollback, downgrade,
Project/Execution migration or cleanup. Retention disk costs are explicit.

Future `tasks` readiness recommendation: **BLOCKED pending review and exact-revision
human acceptance of this Plan plus explicit D1–D4 dispositions**. Required exit:
complete OD/AC mapping; accepted material contracts or named narrower blocked work;
unchanged accepted history; current documentary checks/CI and no unresolved authority
drift. No outstanding Q1–Q6 question needs another answer. Plan approval alone does
not run Tasks/implementation or accept native/product behavior. Stop at this proposal.

## Resumption 3 addendum — human C1, 2026-10-20 18:40 BRT

This addendum changes only the quota-freeze ordering in the historical revision
above. Rafael's explicit decision in the resumption-3 instruction is recorded
literally in [C1 acceptance](acceptance-resume-3.md). G1 may use the exact accepted
initial quotas as development/security budgets. Public final quotas remain
conditional on integral corpus/reference measurement and coverage validation in
T05/T06. **T07/packaging is blocked** if any limit is exceeded or coverage is
incomplete. Do not reduce coverage or change a material limit without a comparative
proposal and explicit human disposition. All other G1 requirements remain intact.
**Human approval of G1 is required before T05+**; C1 does not approve G1.

T01's coverage-based final quota freeze and T04's quota reconciliation are read
with this bounded amendment. Task IDs, DAG, historical dispositions, D1-e's
unqualified v0.1.0 objective, G2/G3 and AC-01–AC-08 remain unchanged. T11's historical
suffix notation is governed by accepted D3-a: `axiom --json docs inspect` and
`axiom --human docs inspect`. Current execution/Evidence status belongs to the
[resumption-3 checkpoint](evidence-resume-3.md), not historical NOT_RUN headers.
