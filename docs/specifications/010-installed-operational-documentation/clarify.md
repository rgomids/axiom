# Specification 010 — Clarification Q1–Q6

## Status, authority and boundary

**Proposed — decision pending, revision 1, 2026-10-20.**
Scope: Clarification only for [AXM-37](https://linear.app/rgomids/issue/AXM-37),
in the existing [PR #335](https://github.com/rgomids/axiom/pull/335).
No Plan, Tasks, implementation or product acceptance is delivered here.

The [Specification rev.1](spec.md) is **Accepted for the next SDD gate** by
the [human decision](https://github.com/rgomids/axiom/pull/335#issuecomment-6292970482),
bound to commit `99afe2a93c21bfe3767a61133eec8c374b690abd` and blob
`ce7e58630eda9a844d13b7422aaff4b9a91c2ee3`. Its historical `Proposed` header
is preserved. That acceptance explicitly leaves Q1–Q6 undecided. The technical
[review](https://github.com/rgomids/axiom/pull/335#pullrequestreview-5586081096)
was a native COMMENT, not GitHub approval.

The confirmed Linear gate comment `a89e5867-e3ca-4efe-984c-32e7864cbcb2`
reports `READY_FOR_clarify`; the present human instruction authorizes this
Clarification execution. Execution identity/revision/gate remain **unknown**.
No ledger is reconstructed. Linear status/blocking relations remain untouched.

This document contains public technical synthesis from repository sources;
no private Provider payload or raw chat history is imported.
[Clarification Evidence](evidence-clarify.md) records observations and limits.

## Accepted contract versus open choices

| Accepted Specification obligation | Open choice; not granted by acceptance |
| --- | --- |
| OD-01–02: minimal operational text, provenance, local links, exclusions | Q1: exact corpus strategy, language and maintenance boundary |
| OD-03–06: executable-bound, offline read-only resolution; identity and mismatch; help primacy | Q2/Q5: storage, bootstrap locator, public discovery surface and man integration |
| OD-04,07,11: binary/docs/skills/man identity and verified delivery | Q3: external versus embedded assets; format and migration contract |
| OD-08–11: logical consistency, truthful effects, recovery and preservation | Q4: commit boundary, retention and supported return to prior installation |
| OD-12–13 and AC-01–08: man coverage, supported hosts, concrete native Evidence | Q5/Q6: coverage boundary and sufficient observation protocol |

All requirements OD-01–OD-13 and acceptance scenarios AC-01–AC-08 remain in
force. Nothing below weakens them. Axiom != Lingo; Project != Repository;
Role != Model; Execution != Agent; Provider != Transport; Integration != MCP;
Skill != workflow source of truth; Evidence != chat history.

## Decision register

`Observed` describes verified sources, not a decision status. `Proposed` marks
the recommendation; `Pending` marks its human disposition. `Accepted` requires
a named human decision tied to the exact proposal/revision and scope.

| Question | Observed basis | Recommendation status | Human disposition |
| --- | --- | --- | --- |
| Q1 | Mixed operational/development reference and installation guide | Proposed: dedicated technical corpus plus generated reference, English first | Pending; reversible editorial default, no language preference inferred |
| Q2 | Independent binary/receipt roots; no installed-doc resolver | Proposed: installation-local assets and locator anchored to actual executable | Pending; material path/discovery contract |
| Q3 | Strict archive/metadata/receipt readers; skills already embedded | Proposed: external verified assets in release bundle; explicit format evolution | Pending; material packaging/migration decision, ask first |
| Q4 | Binary publication precedes separate receipt/skill effects; no automatic rollback | Proposed: prevalidated generations, binary effect boundary, fail-closed recovery, explicit retention | Pending; material lifecycle contract |
| Q5 | Custom Go command/help tree, no Cobra dependency | Proposed: read-only CLI discovery plus passive metadata; skills consume it | Pending; material public interface; syntax deferred |
| Q6 | Spec demands native source observation; no such trial performed here | Proposed: isolated native sessions plus OS/network observation and negative controls | Pending; proof protocol, feasibility unverified |

## Q1 — Minimum corpus and ownership

**Alternatives.** (a) Cut existing Markdown sections: least initial authorship,
but installation/development links and headings can change unnoticed.
(b) Dedicated versioned operational corpus: clear offline journey, additional
reconciliation work. (c) Generated command reference alone: low flag drift,
insufficient concepts, recovery explanation and troubleshooting.

**Recommendation (Proposed).** Combine (b) with generated reference from the
actual command/flag contracts. Use one English technical source initially,
consistent with the installation guide and canonical README language policy;
CLI tokens remain exact. Portuguese translation would need an explicit
reconciliation rule; neither the operator's language nor Spec language grants
that choice. Do not import Notion narrative.

Minimum content units, not prescribed filenames:

| Unit | Bounded content / canonical input |
| --- | --- |
| Index / concepts | Local navigation, identity/help priority and the invariants above; technical synthesis of [conceptual model](../../architecture/conceptual-model.md) |
| Installation / verification | Local verified archive, supported hosts/prerequisites, installed component inspection, legacy absence and mismatch; [installation](../../installation.md) |
| Operational guides | Configure/inspect Project and Runtime/Profile; inspect Work Item/Workflow and distinguish readiness from authority; only current operational portions of [commands](../../commands.md) |
| Reference | Public commands/flags, configuration precedence/scope, public metadata needed for operation; executable tree/introspection remains normative |
| Troubleshooting / recovery | Missing/invalid docs, PATH/man divergence, skills conflicts, links/permissions, interruption, foreign notes, inference/Provider unavailable; safe inspection and authorized handoff |
| Man / notices | Additional generated CLI reference and coverage map; applicable license/attribution and provenance inventory |

Every delivered item needs source path/section, source commit, transformation
identity when generated, output digest and local-link mapping. Required journey
links close inside the package; optional external links are labelled network-only
and never followed implicitly. Generation must reject unresolved local links,
unlisted outputs or excluded content. Dedicated explanations are reconciled when
their source contract changes; volatile flags are generated rather than copied.
Do not choose byte quotas here: bounded file/count/total limits must be specified
under a later authorized design and verified against this minimum.

**Compatibility / cost / residual risk.** No historical Specifications, ADRs,
Evidence, maintainer policies, source code, private data or wildcard `docs/`
distribution. This respects [documentation ownership](../../documentation.md).
Editorial default is reversible before distribution; released paths/links and
language coverage acquire compatibility cost. No ADR conflict identified;
sufficiency review and source-to-output reconciliation remain necessary.

## Q2 — Installation layout and resolution

**Observed.** [Bootstrap](../../../scripts/install.sh) defaults to
`$HOME/.local/bin/axiom` and `${XDG_STATE_HOME:-$HOME/.local/state}/axiom/install`.
[Installation](../../installation.md) permits independent `--bin-dir` and
`--receipt-dir`; no common parent prefix can be assumed. Source installer also
allows independent overrides. XDG here is installation state, not an adopted
documentation location. Shared Runtime skill roots are independent again.

**Alternatives.** (a) Assets adjacent to each installation's binary directory:
bounded discovery and A/B separation, but adds owned entries beside the binary.
(b) Conventional `share`/XDG data roots: familiar locations, but requires an
installation discriminator and binding to find the right instance.
(c) Explicit resolutive metadata: accommodates independent roots, but bootstrap
and integrity must themselves be bound to the executable; a central registry
or environment override alone risks selecting another installation.

**Recommendation (Proposed).** Use (a) plus minimal (c): a private owned asset
namespace and resolutive metadata local to the binary installation, with immutable
content-identified generations. Illustrative topology only: binary directory →
owned namespace → generation → text/man/inventory. No pathname/schema is adopted
here. Preserve existing independent binary/receipt destinations; never guess a
receipt directory by walking home, Project, checkout or XDG roots. Metadata must
bind the effective executable identity, installation target and exact generation;
it cannot merely advertise an arbitrary directory.

Resolution begins with the executable actually running, not `which` repeated
after PATH changes. Validate supported alias/symlink resolution to the physical
executable and anchored asset root; alias resolution gives no write authority
and package entries cannot escape via links. If the host cannot establish that
binding safely, refuse verified docs. A/B in separate bin directories resolves
independently, including equal version strings, arbitrary CWD and spaces.
Switching PATH requires fresh identity, not a cached directory. A moved root or
isolated copied binary requires revalidation/rebinding by an authorized installer;
unknown binding means absence/mismatch, never automatic repair or fallback.
An older process surviving replacement must retain a provable generation or
refuse; hashing only the replacement at its former pathname is insufficient.

**Compatibility / cost / residual risk.** Conventional man search directories
remain Q5. No global XDG doc cache, profile mutation or privileging is implied.
Adds a public discovery/migration contract and cross-root validation cost.
Selection is material: changing published paths/locators later needs migration.
Requires a new architectural decision before design adoption; no accepted ADR
is superseded by this proposal.

## Q3 — Packaging, identity and migration

**Observed.** [Builder](../../../scripts/build-release-archives.sh) builds
`cmd/lingo` into public binary `axiom`; it records version, 12-character revision,
source state and release provenance. It hashes binary, installer, LICENSE,
metadata and four skill files into `MANIFEST.sha256`, then hashes archives.
No operational docs/man generation exists there. Archive metadata skill constants
`1/1` are not the Runtime's `2/2`: [ADR-0016](../../decisions/0016-candidate-derived-skill-set-receipt.md)
explicitly prevents inferring candidate Runtime constants from archive format 1.
[Archive reader](../../../internal/install/archive.go),
[POSIX facade](../../../scripts/install-release.sh) and
[installation receipt](../../../internal/install/upgrade.go) validate schemas;
adding metadata fields is not transparently backward compatible.

**Alternatives.** External text/man assets in a verified bundle: readable without
execution and no extraction during consultation, but installer/build/parser
changes must publish a coherent set. Embed the bounded corpus and extract only
during explicitly authorized installation: closer binary coupling, larger binary,
extraction/path/permission logic and two materialized copies to maintain.
Embedding for lazy extraction at discovery conflicts with OD-06; compiling all
docs remains excluded. Either viable packaging option still needs integrity,
versioned output identity, man installation and migration.

**Recommendation (Proposed).** External files in the same verified release
bundle, with exact inventory and provenance. Extend formats explicitly under a
new reviewed compatibility contract; do not silently add fields to v1, repurpose
skill constants or rewrite receipts from published releases. No format version,
serialization, library or generator is adopted during Clarification.

The future contract must relate full source commit (short display remains
permitted), actual binary digest/build provenance, archive digest, doc inventory
digest and per-file hashes, generated man identity/coverage, expected embedded
skill inventory, and observed installed receipt/files for **each** Runtime.
Matching version strings or release labels alone cannot establish equality.
Record installed versus actually loaded skills separately: a native session
may cache older bytes. Unobservable loaded identity is `unknown`.
Preserve [ADR-0016](../../decisions/0016-candidate-derived-skill-set-receipt.md):
candidate proof gates Codex receipt publication; Claude convergence remains
separate via authorized `first-run`. Partial Runtime convergence is visible.

Legacy installs report docs absent until an explicitly supported migration;
unknown formats fail closed and older installer compatibility must be declared,
not assumed. Source-build installation must likewise deliver a source-derived
package or explicitly remain outside the qualified documentary path; a later
design may not silently call such an install OD-07 compliant.

**Security / cost / residual risk.** Release-channel verification and trusted
artifact acquisition remain necessary: a changed manifest plus changed files
does not establish authenticity. Local hashes detect observed drift under
ADR-0005, not arbitrary same-UID forgery. External bundles minimize extraction
complexity in the binary; migration across strict installer layers is real work.
New packaging/identity ADR candidate; any change to ADR-0016 would instead need
independent human acceptance and ADR-0018 supersession. No such change proposed.

## Q4 — Publication, recovery, return and cleanup

**Alternatives.** In-place independent copies are simple but cannot establish
the accepted complete-generation reader contract. Prevalidated immutable
generations cost disk space but preserve coherent reads/return. One global
pointer is insufficient when binary, receipt and user-global skills publish
separately. Automatic destructive rollback/GC would add unsupported authority
and cannot undo already confirmed effects.

**Recommendation (Proposed).** Prepare and verify complete candidate text/man/
inventory and executable binding before replacing the active binary. Keep
previous generations. Readers select by proven executing-binary identity;
staged candidate availability alone never selects it. Logical activation occurs
at the confirmed binary-publication effect, consistent with current
[upgrade publication](../../../internal/install/upgrade.go) and fresh-install
[binary commit](../../../internal/install/fresh_posix.go), not at a later receipt
acknowledgment. This is a proposed boundary, not proof that current code already
coordinates docs. A later authorized design must demonstrate the binding can be
ready before publication without exposing an unvalidated generation.
This boundary concerns the documentary installation path; it does not redefine
earlier confirmed preservation/retirement effects in existing compatibility
transitions or promise to roll them back. Those protocols remain unchanged.

| Boundary | Required behavior within ADR-0005 |
| --- | --- |
| Before binary publication | Existing complete installation remains usable; candidate files are staging/residue only. Failure does not replace old binary/docs. Foreign material is preserved |
| Binary publication confirmed | New binary effect is committed. Missing receipt/ack or subsequent failure cannot be described as rollback. Docs are accepted only with verified generation/binding and non-ambiguous operation state |
| Skills/receipts not converged | Report each Runtime's observed partial/mismatch state. Do not certify universal convergence or use another installation's skills as proof |
| Concurrent reader/installer | Coordinate conflicting publishers with current authority/revision and anchored identities. Reader pins one complete validated generation or refuses/retries on change; stale publisher cannot commit |
| Unknown publication or interrupted marker | Preserve residue; return recovery-required semantics and bounded read-back. No mixed state accepted; no new enum or recovery command asserted |
| Cleanup failure after commit | Report committed effect plus cleanup failure; retain bytes. Never roll back narration to pre-commit success/failure |

Ordinary downgrade remains refused; there is no automatic rollback. Proposed
AC-04 route: retain independently installed, verified A while installing B in
a separate eligible destination, then explicitly select A and its matching
Runtime configuration/skills/man after authorized inspection. This is a return
to an existing compatible installation, not `upgrade B→A`. Shared skill roots
must either be proven compatible or use separately configured supported roots;
do not silently overwrite B. Project/Execution state is not reverted; if A cannot
read the observed state, return is refused. Exact restoration Evidence remains
required; downgrade refusal alone cannot satisfy AC-04. If this route is judged
insufficient, in-place rollback is a separate human contract decision and ADR
candidate, not an exception inferred here.

Default retention: retain all recognized generations during this slice; no
automatic pruning. Explicit future cleanup requires an authorized object set,
positive ownership, unchanged inventory and proof no live reader/reference needs
it. If live-use proof is unavailable, retain. Unrecognized files and user notes
remain conflicts; do not inventory them as official or silently delete them.
Notes should live outside the managed corpus. This favors safety over disk use.

Every effect stays inside exact authorized roots; validate traversal, entry
types/link counts, owner/mode/ACL and replacement boundaries; bound reads and
artifact counts. No privilege escalation, chmod/ownership repair, following
unvalidated links or editing shell profiles. Preserve ADR-0005 exclusions for
arbitrary hostile same-UID interleavings, power loss and media durability.

**Contract / cost / residual risk.** Explicit adoption needs lifecycle ADR and
human acceptance of the AC-04 route/retention choice. No weakening of ADR-0005
or ADR-0016 is proposed. Disk growth and unprovable live-use cleanup remain
deliberate costs. Moving the binary commit point or promising filesystem-wide
transactions would require separate analysis/decision, not this document.

## Q5 — Discovery surface, skills and manual pages

**Alternatives.** Read-only CLI discovery binds inspection to the executable,
but adds a supported interface. Passive metadata is readable without execution,
but cannot by itself prove which binary the shell or process selected. Skill-only
path rules are inexpensive initially but duplicate resolution and drift.

**Recommendation (Proposed).** One read-only CLI surface for human and canonical
JSON discovery, backed by validated passive metadata. Expose binary/location/
provenance, doc root/index/generation/inventory identity, component compatibility,
expected/observed Runtime skills, man locations and absent/invalid/unknown states.
Loaded skills are never inferred merely from files on disk. Metadata allows
manual inspection but confers no independent authority. Extend existing
presentation contracts; no command name, flag, field name or schema adopted now.
Skills call executable discovery and help/introspection, never reproduce a path
registry or argument grammar. No writes, extraction, network, install/repair or
credential access during discovery. Help/version remain usable with no docs.

[Public command tree](../../../internal/cli/commands.go) and
[help renderer](../../../internal/cli/help.go) are custom Go structures using
`flag`; [go.mod](../../../go.mod) does not adopt Cobra. Prefer generation from
existing executable contracts, without selecting a generator now. `axiom help`,
`axiom --help`, hierarchical help and skill inspection retain priority for
arguments/authority. Text/reference divergence is diagnosed, never authority.

Generate root page and all public operational command paths represented by
the executable tree; a parent/shared page may cover a leaf only with a traceable
coverage-map entry and clear lookup. No hidden/internal test commands. The
minimum covers installation/upgrade/recovery, version/first-run, Project, Runtime,
Workflow/Execution, Work Item and skill inspection; public-tree changes must
reconcile the map. Every page identifies release/commit and its installed
generation/location binding, including an adjacent verified identity record when
destination-specific information cannot be baked into release bytes.

Use an installation-local conventional `man1` hierarchy and documented per-session
`MANPATH` selection for that installation, plus explicit local-page consultation
where supported by the host reader. No automatic profile edits, global symlink
creation or system-man cache mutation. Native tests must prove the actual macOS/
Linux readers recognize that layout and `man axiom`; no promise that arbitrary
custom prefixes are discovered without selection configuration. Plain `man`
cannot be controlled by Axiom: compare resolved executable identity with the
actual selected page path/identity. If `man` finds B under PATH A, label B and
show the supported explicit A selection. Test reader behavior per host; do not
assume identical locator flags across readers. Without a reader, text/help work.

**Compatibility / cost / residual risk.** Public discovery schema and man
selection instructions become compatibility contracts; a new interface ADR
candidate belongs with Q2/Q3. No new CLI or dependency now. Windows remains
withdrawn per [ADR-0021 Status note](../../decisions/0021-windows-server-amd64-native-acceptance.md#status),
with no acceptance row resurrected. External `man` may still select stale content;
observable origin/diagnosis and explicit correct selection are required.

## Q6 — Reproducible native proof protocol

**Alternatives.** Transcript/fixture-only proof is cheap but cannot establish
source confinement. Tool allowlists alone miss direct subprocess/network reads.
Clean native host/VM with effective denial and independent observation costs
more but tests the accepted claim. Containers are acceptable only where they
actually host a supported native Runtime/OS combination; a Linux container is
not macOS acceptance. No vendor sandbox capability is presumed.

**Recommendation (Proposed).** Retain all AC-01–AC-08. Use these minimum Evidence
conditions, subject to demonstrated host controls:

1. Record exact locally verified release archives/checksums, binary/doc/man/skill
   identities, Runtime executable versions/digests, supported OS/architecture/
   filesystem and tool/observer versions. Prepare a clean isolated user/host or
   VM; no checkout/mount, maintainer home, memories, prior chat/session or inherited
   integrations/configuration. Keep credentials outside published artifacts.
2. AC-01 runs without any Runtime and with network denied: clean installation
   from local artifacts, arbitrary CWD/spaces, discover/read the minimum and
   traverse required local links. Record paths and digests actually read.
3. AC-08 uses separate fresh **native Codex and Claude** sessions. Permit only
   verified installed skills, their matched docs, executable help/introspection,
   a controlled local Project/readiness fixture and necessary Runtime system
   files. Deny checkout/history/memories and documentary web/search/MCP sources.
   Verify effective controls including child processes; setting an agent prompt
   or inspecting only its tool transcript does not establish denial.
4. Capture independent OS-level file-access observations and network policy/
   connection observations covering the Runtime and descendants, correlated with
   session identity and tool events. Prove observer coverage with positive reads
   of installed docs and deliberate denied reads/network attempts to known
   disallowed fixtures. A denied attempt is distinguished from a successful
   escape; lost events or unobserved children leave confinement unverified.
5. Allow inference/auth transport only when required and explicitly separated
   from documentary sources. Endpoint/connection logs alone cannot prove that
   an opaque vendor service did no remote retrieval. Disable retrieval features
   where controls exist and document inaccessible server-side behavior. No
   air-gap or absence-of-model-prior-knowledge claim follows from this protocol.
6. Repeat per Runtime with A, B, explicit A/B paths, switched PATH and differing
   man selection; same-version/different-bytes fixtures; crossed skills/docs,
   absent package and tampered inventory. Use version-specific benign reference
   details to correlate actual reads, rather than judging fluent answers alone.
   Explain command and readiness from observed CLI facts; missing docs require
   explicit refusal of verified documentary guidance, without fallback.
7. Preserve bounded sanitized Evidence: environment recipe, real commands/control
   configuration, artifact hashes, coverage/denial checks, access summaries,
   correlated output excerpts, negative-control results and read-back. Keep raw
   private logs/credentials out of Git. Each matrix row is PASS, FAIL, NOT_RUN or
   UNVERIFIED; unavailable isolation/observation cannot be converted to PASS.

**Compatibility / cost / residual risk.** Do not invent vendor flags, dispatch
Runtimes or change local bindings during Clarification. The later authorized
stage must select demonstrably effective controls per host. Black-box server
inference and training knowledge remain outside observable source-access proof;
if AC-08 cannot be supported with these limits, submit an explicit human
clarification of that claim, not a silent acceptance downgrade. No ADR change
identified for this protocol; broader threat guarantees would require one.

## Human decisions and next gate

First material decision: **Q3 external verified release assets versus bounded
embed with authorized installation-time extraction**. Recommend external assets;
acceptance would approve that direction and explicit format evolution, not a
schema/library or Q2/Q4/Q5 by implication. After that decision, present Q2,
Q4, Q5, then remaining Q1/Q6 dispositions individually with exact revision and
scope. No human choice is recorded as Accepted in this revision.

Likely ADR subject: installation-bound documentary identity, packaging/discovery
and lifecycle, with independent acceptance before adopting material design.
No new ADR number is reserved and no accepted ADR edited. Any contradiction
requires a new Proposed ADR and [ADR-0018 governance](../../decisions/0018-adr-evolution-and-supersession-governance.md),
including later accepted reciprocal supersession where applicable.

Stop at **Clarification Proposed / decision pending**. No Plan readiness is
asserted. Resolve material choices, establish the appropriate next authority/gate
and preserve native Evidence obligations before Plan can be separately authorized.
