# T01 — D1/D3 contract proposal, revision 1

**Proposed — every value, schema, pathname, quota and test below requires Rafael's
exact-revision approval. 2026-10-20. Nothing is frozen or accepted here.**

This implements the documentary portion of E01 under the resumption instruction
of 2026-10-20 17:36 BRT. Approved directions remain in [acceptance-plan.md](acceptance-plan.md)
and [ADR-0033](../../decisions/0033-installation-bound-operational-documentation.md).
[Tasks](tasks.md) at `088eb89ac1d2a4328a956cec7047990b566e9934` remain unchanged.
[Resumption Evidence](evidence-resume-1.md) distinguishes measured facts, proposals
and unexecuted product tests. G1 remains blocked; T03+ do not start.

## Baseline and implementation trade-offs

Source baseline: `a6d37dca1a1ea47b545a0aa7c89dca422b429de6`. The sole change since
approved Tasks was the previous exploratory Evidence. Current owners:

- `internal/install/archive.go`: gzip/tar standard library, closed 10-key metadata
  v1, four active skills, 64 archive entries, 256 MiB compressed/binary limits,
  1 MiB other entry, 64 KiB checksums; no documentary reader.
- `scripts/build-release-archives.sh`, `install-release.sh`: closed v1 producers
  and pre-execution consumers; shell executes only after validation.
- `internal/install/upgrade.go`: closed 15-key receipt v1, 64 KiB receipt bound;
  independent receipt-root lock is insufficient for same binary/different receipts.
- `internal/cli/commands.go`, `help.go`, `completion.go`: custom command tree,
  stdlib flags, canonical completion status/provenance; 16 KiB completion default.
- `internal/codexruntime`: four active skills; Runtime receipts/constants have
  their own ownership history. Archive `1/1` constants are not Runtime constants.
- `internal/local`: anchored directories and identity checks already exist;
  pathname `FileIdentity` alone does not prove an executing image.

Proposed choice: retain stdlib SHA-256, tar/gzip, JSON and existing anchored
filesystem helpers; use closed LF line/TSV package records so POSIX shell can
validate passive data with existing tools before executing Go. No `eval`, sourcing
metadata, custom crypto, package manager or new dependency. JSON package manifests
would require a JSON parser before candidate execution (current facade has none);
adding `jq` would change installation prerequisites. Existing JSON canonicalization
library remains available for contracts already using it; no signature/authenticity
protocol is introduced. Go-only local binding/discovery use stdlib JSON with explicit
duplicate-key rejection: `DisallowUnknownFields` alone does not reject duplicates.
Native proof options and limitations belong to T02/T04, not schema approval.

## Material approval requests

Every row is **Proposed**. Recommended values are the detailed contracts below.
An approval must name this document revision/commit and these rows (or precise edits).

| ID | Decision and recommendation | Alternative / cost / harder to change after shipment |
| --- | --- | --- |
| D1-a | Archive generation 2, release metadata 2, installation receipt 2; docs inventory/provenance 1, local binding 1; keep skill manifest/Runtime formats unchanged | JSON package envelope needs new pre-execution parser; shared version counter falsely couples independent protocols. Published readers become compatibility commitments |
| D1-b | Closed UTF-8 LF records/TSV, exact order and full source commit; flat constrained names, digest DAG below | Richer nesting increases shell/parser surface. Changing bytes later changes generation identifiers |
| D1-c | Private sibling `.axiom-docs/axiom/`, immutable generation and binding topology below; no mutable current pointer | Global/XDG registry adds shared resolution. Destination becomes a public locator; D2/T04 proof remains prerequisite |
| D1-d | 64 docs/man outputs, 8 MiB content, 192 tar headers, 288 MiB decompressed stream and remaining quota table | Raising arbitrary limits increases allocation/decompression surface. Actual future corpus must fit or return for approval; coverage cannot be dropped |
| D1-e | Legacy v1 preserved; initial supported migration floor v0.1.0 with each historical receipt variant positively recognized; fresh v2/reinstall/v2 upgrade matrix below | New v2 facade required for old binaries; old consumers intentionally reject v2. Historical native baseline verification remains pending |
| D3-a | Schema 1 additive `docs` member of existing completion envelope; global leading `--json`/`--human`; statuses/exits below | Suffix flags/new global statuses would contradict current CLI conventions. Public result becomes a compatibility obligation |
| D3-b | Summary paths only for verified, bounded per-Runtime observed facts, loaded always unknown without separate observation; downstream readers revalidate all bytes | Returning descriptors through JSON is impossible. Paths are locators, not pinned reads; verified observation cannot certify a later unchecked open |
| D3-c | Root + shared group man pages, explicit leaf/flag coverage and session-only selection | One page per 106 future leaves exceeds candidate 64-output budget. Shared pages grow but preserve all commands; no implicit global MANPATH |

Approval of D1/D3 does not pass G1, accept D2 mechanisms, run T03+, accept ACs,
change ADRs, publish a release or unblock AXM-39/AXM-12.

## Common wire types and closed decoding

All definitions in this and following sections are **Proposed**.

`H` = lowercase 64-hex SHA-256. `C` = lowercase 40-hex full Git commit.
`R` = lowercase 12-hex legacy revision, equal to first 12 characters of C.
`V` = existing strict semantic version without leading `v`, <=128 ASCII bytes.
`P` = absolute clean UTF-8 POSIX path, <=4096 bytes, no `=`, NUL/control/DEL, `.`/`..`,
repeated separator or trailing slash except root; managed root cannot be `/`.
Spaces are allowed in local P, never in archive names. `U` = canonical unsigned
base-10 integer without leading zeros except `0`, <=2^53-1. `T` = UTC timestamp
`YYYY-MM-DDTHH:MM:SSZ` with calendar validation. Boolean = literal `true`/`false`.

KV wire: one `key=value\n` per field, required order below, exactly one final LF;
no BOM/CR/NUL/blank line/comment/extra `=`. All keys required exactly once, no
unknowns. TSV: tab-separated cells, LF rows, exact header and field counts; no
escaping or embedded tabs/LF. Data rows sorted bytewise under `LC_ALL=C`; duplicate
keys or paths rejected even when hashes equal. Limits checked before allocating,
adding or multiplying; reject overflow, trailing gzip members/data, invalid CRC,
truncation and excess decompressed bytes. JSON: UTF-8, no BOM, duplicate/unknown
members, trailing values, nonfinite numbers or unsupported versions; field order
insignificant on read, fixed struct order on emit, compact JSON + LF. Required
nullable members are present as `null`; no coercion/implicit defaults. JSON nested
objects are closed too. All text/diagnostics bounded and terminal-sanitized.

## Versions and archive layout

| Surface | Proposed emitted version / dispatch | Legacy behavior |
| --- | --- | --- |
| gzip/tar archive | Logical archive generation 2 identified by verified `release-metadata.txt` `formatVersion=2`; no new file executed to dispatch | v1 exact reader retained; strict metadata dispatch, never infer from filename |
| release-metadata.txt | 2 | 1 remains exact 10-key historical record |
| skills-manifest.txt | 1 unchanged, four active files, archive skillSetVersion=1 / binaryCompatibility=1 | Historical ownership variants retained by their owning contracts; not broadened into current active inventory |
| docs/inventory.tsv; docs/provenance.tsv; docs/man-map.tsv | independent format 1 | absent for v1 |
| installation.receipt | 2 for v2 installs, 1 for v1 | Historical bytes/meaning preserved; no v2 fields appended to v1 |
| local binding JSON | schemaVersion 1 | no manufactured binding from a v1 receipt |
| discovery JSON docs object | schemaVersion 1 | v1 install reported absent/legacy_installation |
| installation operation markers | No new version adopted here | Existing v1/v2 transition formats untouched; coordinated v2-doc effects require T03 proposal/review before production |
| Runtime skill receipts | No version change | ADR-0016 candidate-running proof and Claude first-run unchanged |

Archive root is exactly `axiom-<V>-<platform>-<architecture>/` (`macos/arm64`,
`linux/amd64`, `linux/arm64`, goos `darwin`/`linux` respectively). Regular files:
`axiom`, `LICENSE`, `install.sh`, `release-metadata.txt`, `skills-manifest.txt`,
`MANIFEST.sha256`, four `skills/<active-name>/SKILL.md`, three TSV control files,
and exactly the outputs listed in docs inventory. No Windows row. Directory
entries may occur once before children, from exact parent closure; no extra roots,
PAX/GNU extension headers, symlinks, hard links, sparse entries, devices or other
types. Filenames use ASCII `[a-z0-9][a-z0-9._-]*`, component <=128 bytes, relative
path <=240 bytes, no leading dot, slash escape or backslash. Allowed output paths:
`docs/text/<name>.md` and `docs/man/man1/<page>.1`, flat within those directories.
No silent extraction, no extraneous docs tree. Archive mode is 0700 for binary/
installer, 0600 text/control; directories 0700; no set-ID bits. Gzip/tar determinism
is not inferred from documentary determinism.

`MANIFEST.sha256`: unchanged line syntax `H  relative-path\n`, strictly sorted;
exactly every regular file except itself, no self-hash. `SHA256SUMS`: same syntax,
exactly one match for selected archive basename, <=64 KiB, duplicate match refused;
non-selected release rows may remain. Acquisition authenticity remains separate.

### Release metadata 2 — ordered closed KV schema

| Key in wire order | Type / required relation |
| --- | --- |
| formatVersion | literal `2` |
| product | literal `Axiom` |
| version | V |
| revision | R, prefix of sourceCommit |
| sourceState | `clean` or `dirty`; dirty only development, not qualified release |
| release | boolean; true requires clean |
| platform | `macos` or `linux` |
| goos | `darwin` iff macos, else linux |
| architecture | `arm64` or `amd64`, only supported row tuples |
| skillSetVersion | literal `1`, archive meaning unchanged |
| sourceCommit | C |
| archiveGeneration | literal `2` |
| docsInventorySha256 | H of exact docs/inventory.tsv bytes |
| docsProvenanceSha256 | H of exact docs/provenance.tsv bytes |
| docsManMapSha256 | H of exact docs/man-map.tsv bytes |
| binarySha256 | H of axiom bytes |

Full source commit is additionally linked into the candidate build as a new
independent documentary build variable, not a reinterpretation of current short
provenance. Verified source comparison requires the running build to expose this
field; missing field is unknown, never padded/inferred from R. No build-time docs
digest is embedded in a binary whose hash participates in the generation DAG.

### Documentary records — closed TSV schemas

Each begins with a line `formatVersion=1\n`, then the exact header below.
All files are UTF-8 LF; source paths use the same constrained relative path grammar
(with ordinary repository component capitalization permitted for source files).
Source paths <=240 bytes; only allowlisted public tracked sources, no absolute paths.

| File | Header (cells separated by literal TAB) | Row validation |
| --- | --- | --- |
| inventory.tsv | `path`, `kind`, `bytes`, `sha256` | sorted unique output path; kind `markdown` or `man1` consistent with path; U bytes >0 within limits; H exact file hash |
| provenance.tsv | `output`, `source`, `sourceCommit`, `sourceSha256`, `transform` | sorted unique (output,source); every inventory output has >=1 row; source C equals metadata C, H exact source bytes; transform `copy-v1`, `operational-synthesis-v1` or `cli-render-v1`; no unlisted outputs |
| man-map.tsv | `command`, `page`, `anchor` | sorted unique command; command `axiom` or public command words joined by one ASCII space, <=256 bytes; page listed inventory man path; anchor `[A-Z][A-Z0-9_-]{0,63}` section emitted in page |

Dedicated operational synthesis remains reviewed source, not runtime LLM output.
Provenance may map multiple source files to one output and one source to several
outputs; source digest checks occur at build, output and cross-record checks at
both consumers. Consumers do not need or fetch a checkout. Transform identity
changes when rendering/escaping/options change. `docs/text/notices.md` required;
source/transform licenses recorded in reviewed notices. `docs/text/index.md`,
`concepts.md`, `installation.md`, `project.md`, `runtime.md`, `workflow.md`,
`work-item.md`, `configuration.md`, `troubleshooting.md`, `cli.md` are required
initial outputs, not authored/implemented in this proposal. Local required links
resolve only within inventory; optional HTTPS links explicitly marked network-only.

### Identity DAG and local publication

Proposed content identity: generation is SHA256 of the exact ASCII bytes
`axiom-docs-generation-v1\n<inventorySha256>\n<provenanceSha256>\n<manMapSha256>\n`
(substitute lowercase hex digests, LF separators, final LF). All three control
records therefore participate; unchanged output bytes with a new source commit do
not collide with a previous generation. Metadata and binding also carry the
individual control hashes for independent validation. No hashes of a
whole archive appear inside that archive. Relationship, in dependency order:

1. Full source C + source digests + transform generate docs/man and embedded skill
   bytes; binary built with C, V and existing provenance.
2. Output bytes -> inventory hashes; inventory/provenance/man-map -> metadata;
   binary and skills -> their separate hashes.
3. All files -> MANIFEST; complete archive -> external SHA256SUMS.
4. Verified archive hash + metadata hash + documentary control hashes + binary
   digest + target filesystem facts -> destination-specific binding.
5. Binding hash -> new installation receipt. No edge points back into archive.

Pages carry release V/C and transform identity. Generation hash and installed
target are adjacent in binding, not printed into hashed pages (avoids self-cycle).
Conflicting existing same-key objects/control files are refused, never overwritten
or called equivalent. Local hashes still provide no acquisition authenticity.

Proposed namespace: `<physical-binary-parent>/.axiom-docs/axiom/`, owner-only 0700.
Installed leaf is currently exactly `axiom`; do not widen installer filename scope.
`generations/<generation>/` stores complete `docs/` control and output trees;
`bindings/<binarySha256>/<bindingSha256>.json` stores immutable bindings. Resolver
considers only that binary digest directory, bounded <=64 binding records per
consultation; zero matching target => absent, multiple conflicting matches => invalid.
This per-consultation bound is not a retention/GC permission: excess => unknown/quota_exceeded
and human review. Immutable text/control/binding files 0400, generation directories
0500 after publication; writable parent remains 0700. No chmod of existing foreign
material, automatic ownership/ACL repair, mutable current pointer or receipt-root
locator. Root, parent and leaf ownership, modes, ACL, type, link count and anchored
identity verified at declared boundaries. No privileged/shared deployment promised.

Proposed local binding 1: JSON object, all fields required:
`schemaVersion:1`, `binaryPath:P`, `parentDevice:U`, `parentInode:U`,
`binaryDevice:U`, `binaryInode:U`, `binarySha256:H`, `sourceCommit:C`, `version:V`,
`archiveSha256:H`, `metadataSha256:H`, `generation:H`, `provenanceSha256:H`,
`manMapSha256:H`, `skillManifestSha256:H`. No receipt path, timestamp, arbitrary
root hint or loaded-skill assertion. Device/inode additionally checked against
independent OS image proof; they are not durable authentication/inode-reuse proof.
Binding filename H = exact binding bytes digest, excludes its own name/hash.
Moved/copied binaries require explicit authorized rebind; inspector never writes.
How to retain/select a previous binding for surviving A is T02/T04-qualified, not
solved by hashing B's current pathname. Safe refusal remains an allowed outcome.

Proposed receipt 2 ordered KV: `formatVersion=2`, then the exact existing receipt
fields in existing order after formatVersion (`destination`, `sha256`, `product`,
`version`, `revision`, `sourceState`, `release`, `platform`, `goos`, `architecture`,
`skillSetVersion`, `archiveSha256`, `skillManifestSha256`, `installedAt`), then
`sourceCommit`, `docsGeneration`, `docsBindingSha256`, `docsInventorySha256`,
`docsProvenanceSha256`, `docsManMapSha256`. Types as above; destination=P,
sha256=binarySha256, product=Axiom, installedAt=T; cross-values equal candidate/
binding. docsGeneration equals the combined content identity above, not
inventorySha256. Receipt remains a separate confirmed effect after binary commit. It neither activates docs nor grants Runtime ownership.
Unknown receipt format/field or foreign preimage is preserved/refused.

## Proposed quotas and measured coverage

| Surface | Proposed bound (inclusive) |
| --- | --- |
| compressed archive / binary | 256 MiB / 256 MiB (existing individual bounds retained) |
| total decompressed tar stream including headers/padding | 288 MiB, one gzip member, consume/validate stream to end |
| total regular-file payload | 280 MiB, checked separately from stream |
| tar headers / regular files | 192 / 77 maximum (13 fixed + 64 output files) |
| docs/man outputs / individual / aggregate content | 64 / 1 MiB / 8 MiB |
| each documentary TSV control / combined controls | 256 KiB / 768 KiB |
| release metadata / bundle manifest / skills manifest | 16 KiB / 64 KiB / 16 KiB |
| LICENSE / install.sh / each of four skills | 1 MiB each |
| provenance rows / man-map rows | 1024 / 256 |
| receipt / binding JSON | 64 KiB / 16 KiB |
| alias hops / absolute path bytes | 16 / 4096 |
| discovery JSON / human rendering | 64 KiB / existing bounded human expansion of 64 KiB |
| diagnostic reason entries / each diagnostic text | 8 / 512 bytes |
| selected binding records per read | 64, excess refuses; no deletion |

Arithmetic: binary 256 + docs 8 + seven MiB-cap files (license, installer, four
skills plus conservative slack) + controls 0.75 + manifests/metadata <0.1 <280 MiB.
77 regular files plus exact unique directory closure <192 headers; stream bound
includes padding and rejects redundant unbounded directory/end records.

Measured command graph at baseline: **105 public leaves / 549 flag registrations**
(counts registrations, not distinct spellings; global presentation flags separate).
`docs inspect` would add one leaf: 106. Recommend 15 man pages: root, first-run,
skill, runtime, runtime-binding, runtime-profile, project, integration, work-item,
workflow, compatibility, artifact, recovery, upgrade, docs. Eleven text outputs (including notices) + 15
pages = 26 outputs, leaving 38 outputs slack. This proves count headroom, not future
content byte sizes. Current binary/installer/license/skills measured sizes and
inventory fingerprint are recorded in Evidence. Actual generated corpus does not
exist yet: byte-size/coverage freeze remains conditional on T05/T06 checks; overflow
returns to human review rather than trimming commands or silently raising limits.

## Proposed D3 public discovery contract

Existing parser accepts presentation flags only before command words. Canonical
forms: `axiom --json docs inspect`, `axiom --human docs inspect`, `axiom docs inspect`
(default human). Informal `axiom docs inspect --json/--human` means selection, not a
new suffix grammar: suffix, conflicting/repeated mode flags, positional operands or
other options are invalid input, exit 1. Confirm through existing CLI tests in T11;
no global parser change. `axiom docs --help`, `axiom docs inspect --help`, root
help/version remain usable without assets. No root/receipt/runtime override flags.

Proposed JSON is existing completion envelope (`status`, `result`, optional
`references`, `next`, `details`, `provenance`) plus required **`docs`**. Existing
provenance keys/short revision semantics unchanged. Docs schema 1 fields below
are all required; `?` means nullable, not optional. Hash/path values only appear
when independently verified, otherwise null. Arrays are present even when empty.

| Member | Closed type and semantics |
| --- | --- |
| schemaVersion | integer 1 |
| state | `verified`, `absent`, `invalid`, `unknown` |
| reason | enum in state table below |
| binary | object: `path:P?`, `sha256:H?`, `sourceCommit:C?`, `version:V?`, `imageProof:enum` (`linux_proc_exe`, `darwin_text_vnode`, `unavailable`); candidate mechanism labels do not themselves prove anything |
| installation | object: `bindingSha256:H?`, `generation:H?`, `compatibility:enum` (`compatible`, `legacy`, `unqualified`, `mismatch`, `unknown`) |
| paths | object: `root:P?`, `index:P?`, `inventory:P?`, `provenance:P?`, `manRoot:P?`, `manMap:P?`; all null unless state=verified |
| integrity | object: `inventorySha256:H?`, `provenanceSha256:H?`, `manMapSha256:H?`, `files:U`, `bytes:U`; counts observed verified files, else 0 |
| skills | object: `expectedManifestSha256:H?`, `expected:array`, `observed:array`; expected elements `{name:string,sha256:H}`, exact four active names when known |
| skills.observed element | `{runtime:enum,receipt:enum,files:enum,manifestSha256:H?,loaded:enum}`; runtime codex/claude; receipt absent/recognized/conflict/unknown; files match/mismatch/unknown; loaded unknown only in this read-only inspector |
| man | object: `state:enum`, `page:P?`, `readerSelection:enum`; state available/absent/unknown; readerSelection unknown (no reader launched); page root page only when verified |
| diagnostics | array of `{code:enum,message:string}` <=8, codes from reason enum; bounded sanitized text, no OS error dump or credentials |

Expected skill inventory derives from verified candidate/binding. Observations use
only existing configured local roots established by owning contracts, no scans of
arbitrary home/config trees or credential reads. Unavailable root => unknown, not
match. Expected may remain known despite a Runtime mismatch; that mismatch is
reported independently and prevents a claim of Runtime readiness. Documentation
state=verified certifies documentation binding only. No session-loaded inference.

| Docs state / reason enum | Completion status / exit | Verified paths and behavior |
| --- | --- | --- |
| verified / `ok` | success / 0 | All six paths, complete hashes and binding validated within one inspection; man root page available |
| absent / `legacy_installation`, `binding_absent`, `generation_absent` | failure / 1 | paths null; no fallback. Legacy means positively recognized v1; missing state alone cannot establish legacy |
| unknown / `source_unqualified`, `image_proof_unavailable`, `permission_denied`, `concurrent_change`, `quota_exceeded`, `io_unavailable` | failure / 1 | paths null, unknown facts null; next advises explicit inspection/help, no repair |
| invalid / `unsupported_format`, `malformed_record`, `unsafe_path`, `unsafe_permissions`, `identity_mismatch`, `hash_mismatch`, `inventory_mismatch`, `binding_conflict` | validation_failure / 1 | paths null, refuse invalid bytes; preserve all artifacts |
| command interrupted before result | interrupted / 2 | existing global semantics; no partial verified paths |

Check failures have deterministic priority: safety/type/permissions, closed format
and quotas, image/target identity, cross-record identity, inventory/content hashes,
then absence classification; successful observation cannot override earlier failure.
Unsupported schema is invalid, not guessed legacy. If permission denies inspection,
state unknown rather than claiming absence. No `denied_authority` for mere missing
docs. Unknown/malformed command follows existing CLI validation independently.

Human rendering uses existing canonical renderer from same JSON semantics and
sanitization rules. Diagnostic output <=bound before writing; no success-shaped
truncated output. Discovery never starts `man`/Runtime, writes, downloads, extracts,
authenticates or repairs. Returning locators cannot pin a future reader: a consumer
must validate expected inventory and complete selected content into a bounded
snapshot, holding descriptors/bytes and rechecking identity/hash at read boundaries.
Later path reads require fresh validation or refusal. The single-file T02 probe is
not proof of an entire multi-file consumer. Skills may not treat JSON paths as a
capability or implement another installation locator.

## Proposed man coverage and selection

Snapshot [CLI coverage](cli-coverage-proposed.json) maps every currently public
leaf to a page, exact flag defaults and section. Group/root help use their group
pages. New docs inspect -> `axiom-docs.1`, section `DOCS_INSPECT`; root includes
leading presentation flags. Existing historical public Project commands remain
covered because they remain executable. Hidden `install-release` and test commands
are excluded because absent from public tree, not by a new guessed list.

Rule: first command word selects page, except runtime binding/profile use their
own pages. Root/help/version -> axiom. All leaf names, aliases, options/defaults
and help text appear in mapped section; generated parity fails on any missing/extra
leaf/flag/default. Page identity V/C/transform visibly printed. Roff rendering
escapes leading `.`/`'`, backslashes, controls and terminal sequences; reference
cannot inject requests. No external generator/library required on user host.

When discovery verified, session recipe is `MANPATH="$verified_man_root:" man axiom`.
`$verified_man_root` is a validated locator, not shell code. Qualify native reader
trailing-colon semantics, `man -w` lookup and direct local-page invocation separately
on macOS/Linux before documenting reader-specific flags as supported. Compare actual
selected page digest/source against inventory: PATH A + man B => mismatch; select A
explicitly and verify again. Inspector itself reports readerSelection=unknown.
No shell profile, persistent MANPATH, global link, `mandb` or cache mutation.
No man reader does not invalidate Markdown/help; a missing inventoried page is drift.

## Proposed compatibility and test oracles

Pin historical reader/fixture bytes at baseline above (and historical release
artifacts by SHA256 before fetching/executing). Do not rewrite historical fixtures.
Oldest proposed upgrade release is v0.1.0: every recognized receipt variant from
that release through current v1 needs retained positive ownership Evidence; if
one cannot be proven, narrow floor only with explicit human approval. No promise
that current four-skill archive inventory accepts older five/six/eight inventories.
Existing ownership history and archive support are separate.

| Producer / consumer | Expected proposed test result before publication |
| --- | --- |
| baseline v1 builder + pinned v1 shell/Go | unchanged positive oracle, docs absent; recorded fixture digest stable |
| v2 builder + pinned v1 shell/Go | refusal before candidate execution, sentinel calls=0 |
| v1 current builder + new shell/Go | preserved positive installation semantics, no invented docs/Runtime ownership |
| v2 builder + new shell/Go | same accepted/rejected set and normalized identity; validate all passive records before candidate call |
| historical v0.1.0..current v1 installation + v2 facade/candidate | recognized ownership only, explicit preview of namespace/binding/receipt effects; candidate proof gates Codex receipt, Claude separate |
| v2 equivalent reinstall / v2 A->B | exact no-op / confirmed binary commit plus separate effects; no mutable generation pointer |
| old binary directly loading v2 archive/receipt | explicit unsupported refusal; use verified matching v2 facade, not relaxed old reader |
| source build / copied or moved executable / ordinary downgrade | unqualified or needs explicit rebind / downgrade refused; independent A return separately qualified |

[Contract vectors](contract-vectors-proposed.json) enumerate expected tests and
owners. They are **Proposed test specifications**, not executable codec PASS.
T01 specifically excludes production codecs; they belong to T07. Prototype/document
checks in Evidence establish internal consistency only. Required future tests:

- Positive exact metadata, TSV, binding, receipt, four discovery states and both
  presentations; shell/Go equivalent normalized fields; preserve pinned v1 hashes.
- Remove/add/duplicate/reorder every KV key; BOM/CRLF/no-final-LF/control/empty/
  invalid UTF-8; unknown versions; bad commit/revision/row relation; truncation at
  every boundary; duplicate JSON member including escaped spelling; trailing JSON.
- Duplicate/extra/traversal/absolute/archive links/special/sparse/PAX entries;
  decompression bomb, trailing member/CRC, each limit N-1/N/N+1, integer overflow,
  >64 outputs or >77 files/192 headers; zero candidate execution on any invalidity.
- Change binary/docs/man/skills/control bytes singly; cross same-version A/B;
  tamper manifest and content together must never be called authenticated; source
  provenance or expected/installed/loaded mismatch remains truthful.
- Valid path only for verified; unknown proof/permissions cannot become absent;
  runtime mismatch cannot become loaded; default/global flag placement parity,
  exit 0/1/2 and bounded JSON/human output including terminal/troff injection.
- Measured final generated bytes, all public leaf/flag map coverage and deterministic
  two-build doc hashes. Full reader/publisher faults, binding identity and native
  man/runtime/offline qualification stay T02–T15 with their gates.

## Gate disposition

**T01 BLOCKED pending approval and later freeze Evidence.** Proposal ready for human
review; no new versions, layouts or schema values are in force. T02 host probes may
continue only as independently authorized experiments without adopting algorithms.
T03–T16 NOT_RUN; G1 BLOCKED, G2/G3 NOT_RUN; AC-01–AC-08 NOT_RUN. Next action:
Rafael disposition of D1-a–e / D3-a–c at exact proposed commit, then revalidate gates.
