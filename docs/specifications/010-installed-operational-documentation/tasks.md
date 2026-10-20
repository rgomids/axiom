# Tasks 010 — Installation-bound operational documentation

## Status, authority and accepted inputs

**Proposed — tasks only, revision 1, 2026-10-20.** All T01–T16 are planned,
**NOT_RUN**; none is accepted, started or completed by this document.
[AXM-37](https://linear.app/rgomids/issue/AXM-37), sole Draft
[PR #335](https://github.com/rgomids/axiom/pull/335), existing branch
`docs/axm-37-offline-operational-documentation`.

This decomposes [Plan 010 rev.1 at the accepted SHA](https://github.com/rgomids/axiom/blob/a966510849d9dda0ab12f01eb4b396d51d7f4390/docs/specifications/010-installed-operational-documentation/plan.md),
`a966510849d9dda0ab12f01eb4b396d51d7f4390`, under the exact
[human Plan/D1–D4 decision](https://github.com/rgomids/axiom/pull/335#issuecomment-6296942470)
and [Linear trace](https://linear.app/rgomids/issue/AXM-37/distribuir-documentacao-operacional-versionada-junto-a-instalacao-do#comment-57738019).
[Acceptance addendum](acceptance-plan.md) reconciles historical Proposed language
without rewriting the accepted [Plan](plan.md), [Spec](spec.md),
[Q1–Q6](clarify.md#decision-register) or
[ADR-0033](../../decisions/0033-installation-bound-operational-documentation.md).
[Tasks Evidence](evidence-tasks.md) concerns documentary checks only.

The current instruction authorizes documentary Tasks, validation, commit/normal
push and updating this existing PR. No implementation, native experiment, new Work
Item/PR, dispatch, merge, release, closure or Linear status/blocker change.
Execution remains **unknown**: existing prompt/gate/dispatch records are not proof
of a canonical Execution. Do not reconstruct or start one from this backlog.

## Gates and common execution contract

- **G0 — future authority:** exact-revision Tasks review/acceptance plus separate
  explicit authority for the selected implementation unit and its effects. Every
  task below inherits G0. Dependencies and green CI grant no execution authority.
- **G1 — P1 fan-in:** T01–T04 deliver reviewed versioned contracts, preserved v1
  fixtures, native image/reader Evidence on all three host rows, and installation
  lock/commit/recovery fixtures. Record final algorithms, coverage/quotas and human
  disposition of material choices. All P2–P4 production tasks are **blocked by P1**
  until G1 passes. D2 remains unproven; inability to prove it stops dependent work
  and requires a named human decision, never pathname-only fallback.
- **G2 — qualification authority:** bounded disposable hosts, exact unpublished
  artifacts and supported observers must be available and authorized separately.
  No operator installation, credential change, privilege expansion or real Runtime
  startup is implicit. Missing host/tool/proof remains NOT_RUN or UNVERIFIED.
- **G3 — readiness fan-in:** T13–T15 Evidence reviewed, missing/failed rows explicitly
  dispositioned by humans before full qualification claims. Product acceptance,
  release/publication and AXM-39/AXM-12 unblocking remain separate decisions.

Each task's Evidence E01–E16 is a bounded entry in the existing Specification/PR
Evidence, not a new store, wire format or workflow state. Record exact source SHA,
commands/exit codes, fixture/producer/consumer hashes, expected/observed results,
negative cases, limitations and reviewer disposition. Native records additionally
include OS/build/architecture/filesystem, reader/Runtime versions, observer coverage
and sanitized controls. Preserve raw sensitive traces outside public artifacts.
No transcript, fake Runtime, cross-build or string assertion proves native PASS.

Common non-goals: no Windows support (ADR-0021), new dependency/service/global
registry, broad installer refactor, automatic repair/rollback/GC, historical artifact
rewrite or authentication claim from matching caller-controlled hashes. Source
install stays explicitly **unqualified**. Refuse in-place downgrade; return only to
independent compatible A, preserving B and shared state. Retain immutable generations,
including complete unactivated residue. Disk presence cannot prove skills loaded.
Apply ADR-0005's bounded filesystem guarantees and ADR-0016's candidate-running
receipt authority; Claude first-run remains separate. Consultation grants no effects.

## Dependency graph and integration ownership

IDs T01–T16 are stable within Specification 010; do not renumber completed work.
Dependencies below are causal, in addition to G0. Every row is Proposed / NOT_RUN.

| Task | Slice | Direct predecessors / additional gate | Bounded output |
| --- | --- | --- | --- |
| T01 | P1 | none | Versioned package/discovery/layout contract and compatibility fixtures |
| T02 | P1 | T01 | Native executing-image and reader-pinning experiments |
| T03 | P1 | T01, T02 | Installation-lock and publication/recovery transition fixtures |
| T04 | P1 | T01, T02, T03 | Contract/algorithm review and G1 disposition |
| T05 | P2 | T04 / G1 | Dedicated corpus, allowlist and provenance map |
| T06 | P2 | T05 / G1 | Deterministic CLI/reference/man generator and coverage |
| T07 | P3 | T06 / G1 | v2 builder, verifier, shell/Go archive consumers |
| T08 | P3 | T07 / G1 | Confined immutable generation/binding publication |
| T09 | P3 | T08 / G1 | Fresh/upgrade receipts, commit, concurrency and recovery |
| T10 | P4 | T09 / G1 | Read-only image-bound resolver |
| T11 | P4 | T10 / G1 | Public docs inspect adapter and help |
| T12 | P4 | T11 / G1 | Four active skills and session-local man consumption |
| T13 | P5 | T12 / G2 | Exact-artifact native V1–V7 qualification |
| T14 | P5 | T13 / G2 | V8 offline installation without checkout |
| T15 | P5 | T13 / G2 | V9 real independent Codex/Claude observations |
| T16 | P6 | T14, T15 / G3 | Documentation, Evidence and rollout readiness review |

T14/T15 may run in parallel only in separate disposable environments and separate
Evidence files; the Orchestrator integrates them at T16. No shared fixture, skill,
CLI, installer or workflow writes may overlap. Earlier units share contracts/files
and stay sequential. Within T02 host experiments may be independently scheduled
with read-only frozen fixtures and separate outputs; T04 is their mandatory fan-in.
No delegation or parallel execution is performed by this Tasks stage.

### T01 — Define closed versioned contracts and compatibility oracles

- **Objective/boundary:** specify v2 package evolution; independently assign archive,
  metadata, receipt/binding and discovery schema versions according to their own
  compatibility rules. Proposed contract/fixture artifacts under this Specification;
  inspect `internal/install/archive.go`, `scripts/install-release.sh`, receipt owners,
  `internal/cli/completion.go` and command/help owners. No production codec yet.
- **Dependencies/authority:** G0; D1/D3 accepted directions, exact details still need
  verification/review. Limits <=64 docs/man outputs, <=1 MiB each, <=8 MiB total,
  <=256 KiB manifest are candidates, not frozen limits. Include actual command/man
  coverage and binary/installer/license/skills overhead before freezing archive totals.
- **Acceptance:** closed wire/layout tables define fields, required/optional values,
  encoding/order/path rules, full-source/binary/docs/skills identity graph without
  self-hash cycles, and bounded counts/bytes/decompression. Discovery freezes field
  names/types, schema version, verified/absent/invalid/unknown reasons, path availability,
  exit codes and `--json`/`--human` selection using canonical CLI completion rules.
  Man map covers every public leaf with own/shared page and source/session identity.
- **Negative cases:** missing/extra/duplicate fields and entries, unknown versions,
  malformed/truncated metadata, overflow/quotas, crossed identities and unsupported
  receipts must have refusal fixtures; verified paths cannot accompany invalid state.
- **Validation/E01:** preserve pinned v1 reader/fixtures and historical receipts;
  describe expected outcomes for old builder/old consumer, old consumer/v2,
  new shell/new Go/v1 and v2, oldest supported upgrade baseline, equivalent v2
  reinstall and v2→v2. Check shell/Go pre-execution oracles and candidate-execution
  sentinel (zero calls on invalid input). Freeze values only with coverage Evidence.
- **Non-goals:** implementing codecs, executing candidate binaries or changing global
  output/provenance semantics. Full documentary commit is separately scoped.

### T02 — Falsify native image identity and pinned-reader mechanisms

- **Objective/boundary:** future disposable experimental harness, inspecting
  `internal/local/file_identity.go` and OS primitives; no adopted resolver algorithm
  until measurements. Host rows: macOS arm64, Linux amd64, Linux arm64.
- **Dependencies/authority:** T01 + G0 and separately authorized native experiments;
  D2 qualification. T01 field names stay frozen while mechanisms are tested.
- **Acceptance:** start process A with controlled pause before consultation; record
  OS-observed image identity and A generation. Atomically replace its pathname with
  B (including same version/different bytes), resume A; it proves and reads A's
  complete generation or explicitly refuses. Fresh B proves B. Test explicit path,
  switched PATH, alias, safe symlink, spaces, relocation/copy (requires authorized
  rebind), unlink/replacement and changed ancestry. Current-path hash alone fails
  the experimental oracle even when equal labels or successful lookup appear.
- **Negative cases:** unavailable OS proof, stale descriptor, hard link/unsafe alias,
  same pathname/different image, identity changes during read and replaced inventory
  or content must never return replacement docs as A. Reader interrupted between
  inventory validation and content open must consume pinned validated bytes or refuse;
  returning a path with an unchecked later open does not satisfy reader pinning.
- **Validation/E02:** reproducible synchronization points, expected/opened file hashes,
  OS APIs/error modes and timing, supported filesystem/host limits, control that
  deliberately naive pathname hashing is rejected. Compare viable pin-at-start versus
  descriptor-plus-independent-OS-proof options; report feasibility per host. Missing
  proof = blocker, not successful fallback or unsupported platform silently omitted.
- **Non-goals:** claiming experiment execution in Tasks, selecting an unproved API,
  weakening Q6/AC-08, same-UID adversary or power-loss guarantees beyond ADR-0005.

### T03 — Model installation locking and commit/recovery transitions

- **Objective/boundary:** bounded transition/fixture definitions for
  `internal/install/fresh_posix.go`, `upgrade.go`, `internal/local/install_posix.go`;
  cover binary, private namespace, installation receipt and independent Runtime roots.
- **Dependencies/authority:** T01/T02 + G0; use only a mechanism proven by T02.
- **Acceptance:** fixture starts two publishers for the same physical binary using
  different receipt roots and stale expected revisions; only one obtains valid
  installation authority. Also test distinct installations and alias identities.
  Preview binds exact target, identity, revisions, immutable objects and all effects.
  Complete objects plus binding precede executable rename; activation requires
  confirmed binary effect. Receipt/skills/marker are independently reported effects.
- **Negative cases:** stale publisher, lock collision/residue, target swap, stage
  short-write/disk-full/quota/permission/sync failure, conflicting immutable object,
  uncertain rename, confirmed binary then receipt/skills/ack/stage-removal failure.
  Readers accept complete old/new or refuse, never mixed state. Confirmed effect
  survives post-commit failure; uncertain effect cannot be guessed from marker.
- **Validation/E03:** expected transition/effect tables with barriers and read-back
  oracles, confinement sentinels and retained-generation snapshots. Include explicit
  inspection/recovery under fresh authority, no stale retry, blind lock deletion,
  automatic cleanup or rollback; preserve foreign files and complete unused objects.
- **Non-goals:** production locks, global transaction or invented generic recovery.

### T04 — Freeze P1 outputs through reviewed Evidence

- **Objective/boundary:** integrate E01–E03 into contract/algorithm decision record
  under Specification 010; reconcile concrete quotas and compatibility matrix.
- **Dependencies/authority:** T01/T02/T03 + G0; human review of material mechanisms,
  public schemas and limits. Accepted D1–D4 directions are not reopened.
- **Acceptance:** every host has adequate native image/reader proof, every format
  has closed fixtures and producer/consumer owner, same-bin/different-receipts lock
  oracle is complete, shell/Go validation precedes candidate execution, and final
  algorithm/layout/status/exit/man tables have exact-revision disposition.
- **Negative cases:** infeasible D2, missing observer/host, coverage exceeding quota,
  incompatible baseline or ADR conflict => name blocker and smallest human decision;
  do not authorize dependent production. Material Plan/ADR changes require distinct
  human decision and ADR-0018 evolution, not edits to accepted history.
- **Validation/E04:** trace every unresolved item to owner, experiment/fixture and
  reviewer; record G1 pass or blocked with reasons. No synthetic PASS for native gaps.
- **Non-goals:** running P2 or granting future execution/publication automatically.

### T05 — Author dedicated operational corpus and provenance allowlist

- **Objective/boundary:** proposed `docs/operational/` text, explicit source→output
  map, index/concepts, install/verification, Project/Runtime/Workflow/Work Item journeys,
  configuration and troubleshooting/recovery; notices and provenance inputs.
- **Dependencies/authority:** T04 / G1 + G0; **blocked by P1**.
- **Acceptance:** minimum journeys close within local inventory; optional network-only
  links labelled, authority boundaries explicit, executable help owns syntax. Sources
  pinned at full commit with transformations/notices/exclusions. Corpus fits reviewed
  quotas together with generated coverage without dropping required content.
- **Negative cases:** no history/Specs/ADRs/Evidence, maintainer paths/config, private
  Provider data, source tree or secrets; dead local links and unlisted files fail.
- **Validation/E05:** inventory/link/exclusion/security scan and source provenance
  comparison; preserve public/source-install limitations and independent A return.
- **Non-goals:** copying all docs, portal/Notion/Wiki migration or new grammar.

### T06 — Generate deterministic reference and man from actual CLI

- **Objective/boundary:** existing `internal/cli/commands.go`, `help.go` command/flag
  descriptors, bounded standard-library generator, man1 and command coverage map;
  register durable generator/checks in `scripts/automation-registry.json`.
- **Dependencies/authority:** T05 / G1 + G0; **blocked by P1**.
- **Acceptance:** every public operational command/flag/default maps to generated
  reference and own/shared man page; internal/test commands excluded deliberately.
  Generation records source/transform/skills identity without digest cycles.
  Two controlled builds from identical inputs yield identical documentary bytes.
- **Negative cases:** omitted/extra command or flag, changed default, control/terminal
  escapes and troff request injection, excessive output, broken local links fail.
  Shared pages need coverage justification, not omission. No generator on user host.
- **Validation/E06:** compare executable help and command descriptors of exact build,
  deterministic output hashes, escaping fixtures and quota/notice scans. Future T11
  must regenerate after docs inspect registration. Whole tar/gzip reproducibility
  is not inferred from reproducible generated text.
- **Non-goals:** Cobra/framework adoption or hand-maintained second CLI grammar.

### T07 — Implement coordinated v2 producers and pre-execution consumers

- **Objective/boundary:** `scripts/build-release-archives.sh`,
  `verify-release-artifacts.sh`, `install.sh`, `install-release.sh`,
  `internal/install/archive.go`; existing release/facade tests and pinned fixtures.
- **Dependencies/authority:** T06 / G1 + G0; **blocked by P1**.
- **Acceptance:** explicit version dispatch preserves v1 reader/fixtures and receipt
  meanings, emits closed v2 inventory/binding inputs/full documentary provenance for
  three release rows, and verifies archive checksum and contents before any candidate
  execution in shell and Go. Preserve ADR-0016 runtime constants/ownership proof.
- **Negative cases:** execute E01 producer/consumer matrix, malformed/unknown/extra
  entries, traversal/absolute/duplicate paths, links/special files, decompression
  limits, manifest tamper and mismatched binary/docs/skills; execution sentinel stays
  untouched. Historical consumers reject v2 rather than being assumed compatible.
- **Validation/E07:** existing `scripts/test-release-archives.sh`,
  `scripts/test-install-posix-facade.sh`, focused Go archive tests plus differential
  fixture outcomes and pinned old-consumer hashes. Trusted HTTPS release acquisition
  remains distinct from integrity of a local caller-supplied archive/checksum pair.
- **Non-goals:** rewriting releases, new signing/auth service or publishing assets.

### T08 — Publish confined immutable generations and executable bindings

- **Objective/boundary:** proposed `internal/operationaldocs` inventory/binding
  responsibilities, existing anchored `internal/local` primitives and installer
  preparation seams. Installer owns writes; read-only resolution remains T10.
- **Dependencies/authority:** T07 / G1 + G0; **blocked by P1**.
- **Acceptance:** use T04's proven layout/identity; bounded stage→full validation→
  immutable objects and complete binding before binary rename. Equivalent object is
  no-op only after full read-back; differing/foreign object is preserved conflict.
  Independent binaries never share mutable selection. Retain all generations.
- **Negative cases:** ancestry/leaf replacement, unsafe mode/ownership/ACL/types,
  symlink/hard link, crossed binding, missing/extra/changed files, short write,
  quota/space/permission failure and publication interruption preserve sentinels,
  notes/foreign files and old complete set; no prematurely activated generation.
- **Validation/E08:** focused local/install filesystem and failure-seam tests against
  E02/E03 oracles, inventories before/after, actual bounds and read-back effects.
- **Non-goals:** global/XDG registry, mutable docs-current pointer or GC.

### T09 — Integrate fresh/upgrade commit, receipts and explicit recovery

- **Objective/boundary:** `internal/install/fresh_posix.go`, `upgrade.go`,
  preview/digest/marker/receipt owners, shell facade and existing installer tests;
  `scripts/install-axiom.sh` source-path qualification reporting only as needed.
- **Dependencies/authority:** T08 / G1 + G0; **blocked by P1**.
- **Acceptance:** fresh, v1→v2, equivalent reinstall and v2→v2 obey E03 ordering,
  same-installation lock and expected revision across independent receipt roots.
  Confirm binary publication separately from installation receipt, Codex candidate-
  derived receipt/skills and marker acknowledgment. Claude convergence stays first-run.
  Source install unqualified; legacy docs absent; unknown owned state refuses.
- **Negative cases:** run all E03 pre-/post-commit/uncertain fault barriers and
  concurrent publishers/readers, stale preview/identity and interrupted fresh install.
  Report confirmed/remaining effects without rollback. Recovery inspects current
  exact binary/bindings/residue before fresh authorized action. Ordinary downgrade
  refused; independent compatible A return never mutates shared incompatible state.
- **Validation/E09:** targeted install/facade concurrency/failure tests with effect
  read-back and preservation snapshots; no stage cleanup deletes retained objects.
- **Non-goals:** automatic repair/retry, receipt-only installation locking or
  claiming native return qualification (T13).

### T10 — Resolve only the proven executing image's complete generation

- **Objective/boundary:** proposed read-only `internal/operationaldocs` resolver
  using T04 algorithm/OS adapters; inventory/content consumer API with pinned reads.
- **Dependencies/authority:** T09 / G1 + G0; **blocked by P1**.
- **Acceptance:** prove actual image, safely resolve allowed aliases to physical
  target/private binding, bound ancestry/reads and validate complete inventory;
  retain/revalidate pinned identity through content consumption. Expose honest
  verified/absent/invalid/unknown, including permission/concurrent/unsupported reasons.
- **Negative cases:** A surviving B replacement, same-label bytes, switched PATH,
  CWD, relocation/copy without rebind, stale binding, unsafe ancestry and inventory
  races resolve A's proven generation or refuse; never consult checkout, global root,
  arbitrary pointer or B's current pathname bytes as proof of A.
- **Validation/E10:** E02 synchronized races and no-side-effect fixtures: capture
  file/network/write/exec attempts; zero extraction, repair, download, auth lookup
  or Runtime startup. Missing docs does not break executable help/version.
- **Non-goals:** installer mutation or second skill-specific resolution mechanism.

### T11 — Expose versioned docs inspect JSON/human and executable help

- **Objective/boundary:** `internal/cli/commands.go`, `help.go`, completion/rendering
  owners and thin new discovery adapter; refresh T06 generated references/man.
- **Dependencies/authority:** T10 / G1 + G0; **blocked by P1**.
- **Acceptance:** `axiom docs inspect --json` / `--human` implement T01's public wire,
  exit codes and canonical completion behavior. Include actual image/binary/source,
  installation/generation/inventory, compatibility/reasons, man root/coverage and
  expected skill set versus observed per-Runtime receipts/files. Loaded unknown
  stays unknown without independent session observation; no success-shaped invalid path.
- **Negative cases:** all four statuses, malformed presentation flags, permission,
  missing/corrupt/unsupported docs and concurrent change; sanitized bounded diagnostics
  cannot inject terminal controls. No global status/envelope changes by implication.
- **Validation/E11:** public CLI golden/schema/exit fixtures, exact help/reference/man
  parity and resolver side-effect observations; help/version independently usable.
- **Non-goals:** readiness aggregation, ownership grants or repair commands.

### T12 — Wire four skills and qualify session-local man selection

- **Objective/boundary:** four active `internal/codexruntime/skills/*/SKILL.md`,
  `integration.go`, `internal/cli/skill_inspect.go` catalogue as needed; man recipe
  and mapping generated in T06/T11. Preserve historical ownership sets/fixtures.
- **Dependencies/authority:** T11 / G1 + G0; **blocked by P1**.
- **Acceptance:** Codex/Claude skills consume only discovery plus executable help,
  distinguish expected/installed/on-disk/loaded, and report unknown/refusal without
  fallback. Select man root from verified output, quote spaces; qualify session
  `MANPATH="$verified_man_root:" man axiom` and native explicit selection where
  supported. Detect actual page/path identity for PATH A/man B and select A explicitly.
- **Negative cases:** missing reader leaves Markdown/help usable; missing/corrupt
  docs, mixed skill root and same-version drift cannot self-certify readiness.
  No profile edits, persistent MANPATH, global symlinks or mandb/cache writes.
- **Validation/E12:** skill contract/static regressions, no second flag grammar,
  local supported-reader fixtures and regeneration parity. Disk/string/fake-runtime
  checks remain distinct from T15's real observed sessions and T13 native man matrix.
- **Non-goals:** installing skills in operator account, claiming loaded or AC-08 PASS.

### T13 — Qualify exact unpublished artifacts on supported native hosts

- **Objective/boundary:** extend `scripts/accept-native-artifact.py`, existing
  native acceptance workflows/tests and automation registry only where needed.
  No release dispatch or bypass of protected gates.
- **Dependencies/authority:** T12 + G0/G2; exact candidate bundle/binary/inventory/
  skills hashes, pinned producer/consumer baselines and verified harness identity.
- **Acceptance:** execute V1–V7 on macOS arm64, Linux amd64, Linux arm64; record
  native image proof, A/B identity, actual man reader/page selection, no-reader path,
  reinstall/upgrades and all E03 fault/concurrency cases. Independently invoke
  compatible A with A docs/man/skills/config after B, perform allowed local observed
  inspection, retain B and unchanged shared data. Incompatible state refuses return;
  ordinary downgrade refusal is a separate negative, never return proof.
- **Negative cases:** V6 traversal/links/ACL/mode/ancestry replacements, altered hashes,
  inventory and same-label bytes; foreign sentinels unchanged. Include old shell/new
  archive and new Go/v1 pre-execution compatibility, source unqualified and native
  command coverage/quota checks. Cross-compile or rewritten hashes cannot prove trust.
- **Validation/E13:** each AC/V row records exact artifacts, commands/exits, OS/
  filesystem/readers, barriers, snapshots and independent review. T13 contributes
  AC-01/08 but cannot close them: V8/T14 and V9/T15 are mandatory separate protocols.
- **Non-goals:** public release, all-filesystem claims or product acceptance.

### T14 — Prove V8 offline installation and consultation without checkout

- **Objective/boundary:** disposable native environments and bounded V8 collection
  in existing acceptance harness/Evidence; separate from Runtime-dependent protocol.
- **Dependencies/authority:** T13 + G0/G2; all three supported host rows.
- **Acceptance:** copy trusted exact local artifacts and prerequisites before OS
  isolation; remove checkout/mount/maintainer home access, use empty controlled CWD/
  config and deny network at OS. Install, discover and read index/concepts/guides/
  reference/troubleshooting and locally closed links. Record opened paths/bytes and
  binary/docs/man/skills identities, without requiring any Runtime.
- **Negative cases:** allowed file read plus planted forbidden file and network
  attempts calibrate observer and denial controls. Missing privilege/tool/coverage
  makes proof UNVERIFIED, never permission escalation or transcript-only PASS.
- **Validation/E14:** source/artifact/environment digests, effective denial policy,
  successful/denied file/network observations, exact commands/exits and independent
  assessment; this is AC-01's actual offline proof, not a claim of Runtime air gap.
- **Non-goals:** new release/install on operator machine or inference transport.

### T15 — Prove V9 installed-source use in real fresh Codex and Claude

- **Objective/boundary:** separately authorized disposable native Runtime sessions,
  local readiness fixtures and independent OS file/network observers; sanitized
  Evidence split by Runtime/host, no credentials/auth contents collected.
- **Dependencies/authority:** T13 + G0/G2; declare each Runtime's supported host rows,
  execute each eligible combination; unavailable rows NOT_RUN/UNVERIFIED with human
  disposition before full matrix claims. Can run beside T14 only with isolated writes.
- **Acceptance:** follow all six V9 Plan steps: record real Runtime/version/session/
  descendants, deny checkout/memory/config/documentary MCP/search/cache sources,
  permit separately declared inference/auth transport. Observe allowed file read
  and inference, denied forbidden/descendant read and documentary network control.
  Fresh independent Codex and Claude explain command/readiness using installed
  skills/discovery/help. Repeat A/B, same-version bytes, mixed skills and docs absent/
  corrupt; compare citations/identity with actual opened bytes and observed policies.
- **Negative cases:** failed controls, opaque encrypted traffic and observer gaps
  remain FAIL/UNVERIFIED as appropriate; no model assertion, fake session or string
  check substitutes. No web/checkout fallback, inferred loaded state or invented
  authority. Account separation alone is insufficient. Vendor-side invisible
  retrieval/model priors cannot be ruled out by unsupported claims.
- **Validation/E15:** exact process tree, artifacts, controls, OS/tool/transport
  limits, commands/exits, expected/local observed readiness and independent reviewer.
  Q6 infeasibility returns to human decision before weakening AC-08.
- **Non-goals:** credential changes, broad provider operations or AXM-12 acceptance.

### T16 — Reconcile operational docs and present rollout readiness

- **Objective/boundary:** `docs/installation.md`, `docs/commands.md`, affected Runtime
  guidance, Specification/Evidence/index and existing CI/readiness records only.
- **Dependencies/authority:** T14/T15 + G0/G3; blocked evidence may be summarized in
  a handoff, but does not complete qualification or pass rollout readiness.
- **Acceptance:** source→exact prepared artifact→verified installed set→AC row chain
  complete; bootstrap/facade compatibility, legacy/source exclusions, explicit recovery,
  retention costs, session man selection and independent compatible A return documented.
  No unsupported platform, loaded-skill, authenticity or rollback claim. Remote CI for
  exact final implementation head distinguished from local and historical CI.
- **Negative cases:** any missing/failed native host/Runtime/control or unresolved D2
  prevents full-ready claim; no auto-unblocking, next dispatch or publication from CI.
- **Validation/E16:** review full OD/AC/V/Task and producer/consumer matrices, run
  affected repository/document checks and exact-artifact CI, record unresolved risks
  and request separate human product/release decisions when appropriate.
- **Non-goals:** merge/tag/release/closure, new Work Items, state migration or GC.

## OD / AC → validation → Tasks coverage

The following preserves Plan V1–V9, with AC-02 additionally retaining Spec OD-11
coverage. All AC-01–AC-08 and implementation/native validations remain **NOT_RUN**.
A planning coverage cell is not executed Evidence.

| Validation | OD coverage | AC coverage | Tasks / future Evidence owners |
| --- | --- | --- | --- |
| V1 | OD-01, OD-02 | AC-01, AC-06 | T01, T05, T06, T07, T11, T13, T14 |
| V2 | OD-03, OD-04, OD-05, OD-06, OD-11 | AC-02, AC-08 | T01, T02, T04, T10, T11, T12, T13, T15 |
| V3 | OD-07 | AC-01, AC-03 | T01, T07, T09, T13, T14 |
| V4 | OD-08, OD-09 | AC-03, AC-04 | T02, T03, T08, T09, T10, T13 |
| V5 | OD-08, OD-09, OD-10, OD-11 | AC-04 | T03, T09, T12, T13, T16 |
| V6 | OD-10, OD-11 | AC-05 | T01, T03, T07, T08, T09, T10, T13 |
| V7 | OD-04, OD-05, OD-12, OD-13 | AC-07 | T01, T06, T11, T12, T13 |
| V8 | OD-01, OD-03, OD-07, OD-13 | AC-01 | T13, T14 |
| V9 | OD-03, OD-04, OD-05, OD-06 | AC-08 | T02, T04, T10, T11, T12, T13, T15 |

| Producer / consumer boundary | Contract / implementation / exact qualification |
| --- | --- |
| Release builder → archive/inventory/provenance | T01 → T05/T06/T07 → T13 |
| Verified acquisition/local archive → shell facade → candidate | T01 → T07/T09 → T13/T14; verify before execution |
| Archive → Go v1/v2 reader → fresh/upgrade | T01 → T07/T08/T09 → T13; preserve pinned old readers |
| Binary/namespace/generation → installation lock/receipts/recovery | T02/T03/T04 → T08/T09/T10 → T13 |
| Candidate proof → Codex receipt; Claude first-run separately | T01/T03 → T07/T09/T12 → T13/T15 |
| CLI command tree/help → reference/man; discovery → JSON/human | T01 → T06/T10/T11 → T13/T14 |
| Resolver/discovery → four skills and session-selected man | T02/T04 → T10/T11/T12 → T13/T15 |
| Exact artifacts → native harness/CI and readiness review | T04 → T13/T14/T15 → T16; no release authority |

## Stop and next candidate

This revision ends at documentary Tasks delivery/review. D2 native feasibility,
all P1 contracts/experiments and AC-01–AC-08 remain NOT_RUN. No implementation task
has started. Candidate next stage is `implement` only after exact Tasks review/
acceptance and new explicit bounded authority; P1 precedes dependent production.
No automatic dispatch or functional acceptance follows this document.

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
