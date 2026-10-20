# ADR-0033 — Installation-bound operational documentation

## Status

Accepted on 2026-10-20 by the explicit human
[decision in PR #335](https://github.com/rgomids/axiom/pull/335#issuecomment-6296075261),
for revision `c7ef060d71849c9dc82275c23f0ec9f9256f97a0`, blob
`5a8a1efc2b802285f291b31ece9823719a494057`.
Only this Status section reconciles that acceptance; the accepted decision text
remains unchanged. Plan review/acceptance, Tasks, implementation, merge, release
and product acceptance remain separate gates.

## Context

[Specification 010 rev.1](../specifications/010-installed-operational-documentation/spec.md)
requires offline operational guidance without a checkout, tied to the binary,
documents, skills and manual pages actually selected. Matching version labels
cannot distinguish independent installations or different bytes. The human
[consolidation](https://github.com/rgomids/axiom/pull/335#issuecomment-6294090913)
requires documentary reconciliation and a durable architecture proposal while
keeping the Plan gate closed. Historical Spec/header and prior Evidence remain.

Current [installation](../installation.md) supports independent binary and
receipt destinations; Runtime skill roots are independent again. The
[builder](../../scripts/build-release-archives.sh) has no operational text/man
assets. The [archive reader](../../internal/install/archive.go) and
[installer facade](../../scripts/install-release.sh) enforce strict formats;
adding files or metadata is not transparently compatible. The
[upgrade](../../internal/install/upgrade.go) confirms binary publication before
separate receipt/skill effects. [CLI commands](../../internal/cli/commands.go)
and [help](../../internal/cli/help.go) already own executable syntax.

Identity, packaging, activation and discovery must agree on one generation.
Separate ADRs for each would split this invariant without an independent design
choice. One bounded ADR is proposed; concrete formats and mechanisms remain
future authorized work.

## Decision

Propose one installation-bound contract: verified external operational assets
belong to an immutable generation selected through ~~proof of the executing
binary~~ the release-content and installation association defined by ADR-0034,
with explicit package evolution, truthful publication and a single read-only
discovery authority.

> Superseded by [ADR-0034 — Release-content documentation association](0034-release-content-documentation-association.md#decision), accepted 2026-10-20.
> Only the requirement to prove original whole-file or mapped-image identity is superseded; this contract and its other requirements remain in force. The historical wording is preserved for traceability.

### Corpus and generation identity

Use dedicated versioned English operational text plus reference generated from
actual public commands/flags, as accepted in Q1. The minimum comprises local
index/concepts, installation/verification, operational guides, command/configuration
reference, troubleshooting/recovery, man coverage, provenance and notices.
Required journey links close locally; optional network links are labelled and
never followed implicitly. Each output has a canonical source/commit,
transformation identity where applicable and inventory/digest mapping. Do not
bundle historical Specifications, ADRs, Evidence, maintainer policies, source,
private data, Notion narrative or all of `docs/`.

Use a private owned namespace associated with each binary installation and
immutable complete content-identified generations. Preserve independent binary,
receipt and Runtime destinations. Locally verifiable metadata binds the actual
executing binary, installation target and exact generation; it is not an
arbitrary path hint or a second source of authority. No concrete pathname,
metadata encoding or generation-key algorithm is chosen by this ADR.

Resolve from the executing binary, including supported aliases/symlinks safely
resolved to the physical executable. CWD, repeated PATH lookup, checkout,
Project, global/XDG search and equal version strings cannot select a generation.
Read-only alias resolution gives no write authority. A/B installations in separate
eligible destinations resolve independently, including spaces and switched PATH.
A moved/copied binary needs authorized revalidation/rebinding; an old process
surviving replacement must ~~prove its own generation~~ compare its compiled
release-content descriptor with the validated current installation association,
or refuse. ~~Hashing only the replacement at the former pathname cannot prove
that old process's identity.~~

> Superseded by [ADR-0034 — Release-content documentation association](0034-release-content-documentation-association.md#decision), accepted 2026-10-20.
> Only original executing-image identity proof is superseded. Refusal on a missing, crossed, changed, unsafe or ambiguous association remains required; the historical wording is preserved for traceability.
Unprovable binding, absent or corrupt assets produce explicit absence/mismatch,
without fallback or repair.

### Verified external package and explicit compatibility

Ship text/man as external assets inside the same verified release as the binary.
Consultation neither downloads nor extracts assets. Package identity relates
full source commit, binary digest/build provenance, archive digest, document
inventory/per-file hashes, generated man identity/coverage and expected skill
inventory. Record observed installed receipts/files separately per Runtime;
loaded skills cannot be inferred from disk. Unobservable facts remain unknown.

Evolve archive, metadata, installation receipt and discovery formats explicitly
where needed. Declare the producer/consumer and migration support matrix before
implementation; do not append fields to strict v1 and call that compatible,
repurpose skill constants or rewrite historical receipts/releases. This ADR
adopts no version number, serialization, dependency or generator. Legacy installs
report docs absent until an explicitly supported migration; unknown formats fail
closed. Source installs must deliver a source-derived coherent package or be
explicitly excluded from qualified documentary delivery, never silently called
Specification 010 compliant.

Preserve [ADR-0016](0016-candidate-derived-skill-set-receipt.md): only existing
candidate-running proof authorizes candidate-derived Codex receipt publication;
provenance does not grant ownership. Archive v1 constants cannot substitute for
Runtime constants. Claude convergence remains separate through authorized
`first-run`. Documentary success never certifies unobserved Runtime convergence.

### Publication, recovery and compatible return

Prepare and verify a complete generation, man inventory and executable binding
before binary publication. Availability in staging does not activate it.
Activation is the confirmed binary-publication effect, not later receipt/skill
acknowledgment. A future mechanism must demonstrate this ordering while readers
accept only their complete validated generation or refuse. This proposal does
not claim current code implements documentary coordination or a global transaction.

| Boundary | Required outcome |
| --- | --- |
| Before binary publication | Prior complete installation remains usable; candidate staging/residue never replaces prior documentary authority |
| Confirmed binary publication | Report that committed effect even if receipts, skills or cleanup subsequently fail; do not narrate rollback |
| Concurrent or surviving readers | Pin a complete proven generation or refuse/retry on change; stale publishers cannot commit |
| Partial Runtime convergence | Expose observed expected/installed/loaded distinctions and conflicts; shared global roots prove no compatibility |
| Interrupted or uncertain publication | Preserve residue and foreign files; bounded inspection and explicit authorized recovery; no mixed/ambiguous generation accepted |
| Cleanup failure | Retain bytes and report committed effects plus failure truthfully |

Existing compatibility preservation/retirement effects retain their earlier
confirmed boundaries. This does not roll back or redefine them.

Ordinary in-place downgrade stays refused. The accepted AC-04 direction is an
explicit return to independently retained, verified compatible A while B remains
in a separate eligible installation. Check matching Runtime skills/configuration
and man selection; preserve shared-root conflicts. Do not revert Project or
Execution state. If A cannot read current state, refuse return. A future test
must demonstrate actual compatible return; downgrade refusal alone proves no
AC-04 restoration. In-place rollback would require a distinct human decision.

Retain all recognized generations for this slice. No automatic rollback,
pruning or cleanup. Future explicit cleanup requires exact authorized objects,
positive ownership, unchanged inventory and proof no live reader/reference needs
them; absent proof means retain. Foreign material/notes remain preserved conflicts,
never official inventory. Notes belong outside the managed corpus.

### CLI, skills and manual pages

Expose one read-only discovery surface using existing canonical JSON and human
presentation contracts. Report verified binary/location/provenance, generation,
root/index/inventory, compatibility, expected/observed Runtime skills, man origins
and absent/invalid/unknown facts. Passive metadata permits inspection but confers
no independent authority. Syntax, JSON field names and wire schema remain open.

Skills consume this surface and executable help/introspection; they cannot own
parallel resolvers, argument grammar or Workflow truth. Help/version remain
usable without docs; actual executable syntax/authority takes priority over prose.
Discovery performs no network, writes, extraction, install/repair or credential
access. Document contents are data and cannot expand an agent's permissions.

Generate root/public operational command coverage, with a traceable map for
leaves covered by a parent/shared page; exclude internal test commands. Use an
installation-local `man1` hierarchy with explicit per-session selection and
local-page reading where supported. Each page carries release/source identity
and a verifiable installed-generation binding, adjacent when destination details
cannot be in release bytes. Diagnose the actual selected page versus the actual
binary: plain `man` can select B while PATH selects A. Show explicit supported
selection of A. No automatic shell profile edits, persistent MANPATH, global
links or system cache updates. Native macOS/Linux reader behavior needs proof;
without a reader, text/help remain usable. No new library is adopted.

### Security and proof boundary

Preserve [ADR-0005](0005-bounded-local-filesystem-threat-model.md): exact roots,
anchored identities, traversal/link/type/count checks, ownership/permissions,
bounded sanitized input/output, expected revisions, concurrency and truthful
logical publication/recovery. No automatic permission repair or privilege
expansion. Local hashes detect observed drift; changing both manifest and files
proves no authenticity. Trusted release acquisition remains separate. No defense
against arbitrary hostile same-UID interleavings, power-loss or media-durability
guarantee is added.

Supported hosts remain macOS arm64 and Linux amd64/arm64. Preserve
[ADR-0021's Windows-withdrawal note](0021-windows-server-amd64-native-acceptance.md#status);
no Windows support/acceptance row is restored.

[Q6](../specifications/010-installed-operational-documentation/clarify.md#q6--reproducible-native-proof-protocol)
retains AC-01–AC-08, all currently NOT_RUN. Future proof needs offline installation
without checkout; separate fresh native Codex/Claude sessions; effective source
denial and independent OS file/network observation including descendants;
positive/negative controls and the A/B/mismatch matrix. Evidence uses exact
PASS / FAIL / NOT_RUN / UNVERIFIED results with sanitized reproducible artifacts.
Inference/auth transport is declared separately from documentary sources. No
Runtime air-gap, absence of model prior knowledge or opaque vendor-side retrieval
claim is made. Infeasible controls require UNVERIFIED and a new human decision
before any AC change. These experiments do not expand product threat guarantees.

Axiom != Lingo; Project != Repository; Role != Model; Execution != Agent;
Provider != Transport; Integration != MCP; Skill != workflow source of truth;
Evidence != chat history.

## Alternatives considered

| Alternative | Trade-offs and disposition |
| --- | --- |
| Verified external generation and local binding (proposed) | Readable without extraction; supports independent installations and explicit return. Cross-layer format migration, binding validation and retention cost remain; local single-user scope keeps operations bounded |
| Bounded embedded corpus, extracted only during authorized installation | Closer compiled identity; larger binaries and extraction/security logic, plus two materialized copies and man publication. Viable but Q3 selected external assets; lazy extraction during discovery violates read-only consultation |
| Conventional shared share/XDG root plus installation discriminator | Familiar layout and portable tools, but discriminator/bootstrap integrity and stale registry resolution remain; introduces shared coupling without removing generation proof. Q2 selects private installation namespace |
| In-place files and skill-owned locators | Less initial coordination/disk cost; duplicate authority and mixed generations undermine accepted identity/reader contract. Automatic rollback/cleanup additionally exceeds approved authority |

## Consequences

### Positive

One verifiable binary-to-generation contract supports offline consultation,
A/B coexistence, observable mismatch and explicit compatible return. Consumers
share resolution instead of duplicating it. Failure and confirmed effects remain
truthful. Historical receipts, accepted ADRs and publication authority are preserved.

### Negative / trade-offs

Strict installer layers need coordinated versioned changes; public discovery and
published links add compatibility obligations. Generation retention consumes disk;
live-use proof may prevent cleanup indefinitely. Man readers and native Runtime
observation require host-specific qualification, with residual vendor opacity.
No additional distributed service, database or third-party dependency is adopted.

Harder to change after shipment: installation binding, published layout/links,
package migration expectations, discovery JSON and man selection. Exact encodings,
quotas, generator, commit/recovery algorithm and effective Q6 controls remain
future design/validation obligations, requiring separate Plan authority.

No semantic supersession of ADR-0005/0016/0018 or restoration of Windows is
proposed. Accepted ADR files remain unchanged. If future detail contradicts an
accepted ADR, stop for a distinct human decision and follow
[ADR-0018](0018-adr-evolution-and-supersession-governance.md); only after a new
ADR is Accepted may reciprocal supersession annotations/index updates take effect.

## Revisit when

Revisit if safe executing-binary binding or prepublication ordering cannot be
proved on a supported host; strict-format migration cannot deliver coherent
assets; retained-generation cost becomes unacceptable; a shared/privileged or
Windows deployment is proposed; independent return cannot satisfy AC-04; or
native Q6 controls cannot substantiate the approved source-access claim.
Those findings invalidate or constrain this recommendation and need explicit
human disposition; implementation shortcuts cannot silently weaken the Spec.
