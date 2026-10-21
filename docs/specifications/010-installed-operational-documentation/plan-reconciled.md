# Plan 010 — Reconciled implementation proposal

**PROPOSED — R1, 2026-10-20.** Implements [Spec R1](spec-reconciled.md) only after
[H1](reconciliation.md#grouped-human-decision-h1) accepts the exact baseline and
ADR-0034. No production files change in RECONCILE. Later implementation requires
its own complete prompt/authority; no new Execution or duplicate PR is inferred.
Historical [Plan](plan.md), [Tasks](tasks.md), D1/D3 and C1 remain unchanged.

## Minimal design and existing owners

Use existing Go CLI tree, release tar/gzip/checksum pipeline, shell/Go archive
readers, anchored filesystem helpers, preview/expected revision mechanisms and
existing receipt/skill owners. No framework migration or generic transaction layer.
The current repository has no installed documentation resolver; new responsibilities
below are proposed work, not capability claims.

| Responsibility / owner | Bounded change after approval |
| --- | --- |
| `docs/operational/` proposed authored source | Explicit English allowlist: index/concepts, install/verification, operating guides, configuration and recovery; required local links, source mapping and notices. Do not copy all current mixed development docs |
| `internal/cli/commands.go`, `help.go`, existing build tooling | Standard-library rendering from real command/flag tree; shared root/group man pages and complete coverage map. No Cobra or new host prerequisite; generation happens before packaging |
| `internal/provenance` / `cmd/lingo` build identity | Add separately scoped compiled documentary descriptor. Preserve abbreviated public provenance; descriptor carries full commit/build target and exact docs/command-map/transform/expected-skill identity |
| `scripts/build-release-archives.sh`, verifier, `internal/install/archive.go`, `scripts/install-release.sh` | Accepted closed v2 format evolution, v1 unchanged. Check paths/types/counts/hashes before extraction/use and before candidate execution. Extend only documentary fields required for the descriptor; no schema redesign |
| `internal/install/fresh_posix.go`, `upgrade.go`, `internal/local` | Confined generation/binding preparation, installation-wide writer coordination, binary-publication confirmation and truthful partial/recovery. Reuse existing expected revisions and anchored sessions |
| Proposed small operational-doc reader, CLI adapter | One resolver/content validator shared by discovery consumers, install validation where useful and CLI; install owns writes, CLI presentation. No abstraction for arbitrary asset types, process registry or skill-owned resolver |
| Four active `internal/codexruntime/skills/*/SKILL.md` | Use canonical inspect/help, revalidate selected inventory/content, preserve all historical ownership sets. Report expected/installed/loaded separately; loaded stays unknown without independent observation |
| Existing native acceptance facilities | Add exact-artifact docs/man/offline rows and native Runtime observation. Reuse declared host facilities; authorize environments separately, no new hosting platform or permanent experiment workflow by default |

## Contracts retained and one bounded identity amendment

Retain the accepted D1-a/b/c/d/e and D3-a/b/c values in
[contracts](contracts-proposed.md), with qualifications in
[acceptance-resume-2](acceptance-resume-2.md) and [C1](acceptance-resume-3.md).
That includes independent versions, strict closed encodings, paths, numeric
budgets, coverage, discovery states/reasons/exits, session man selection, private
`.axiom-docs/axiom/`, immutable generations and bindings. Do not infer that their
historical Proposed headers erase later acceptance. Never weaken v1 decoders.

Only image-proof-dependent semantics change. The following is a **proposed textual
amendment of the unshipped contracts**, not an executable schema edit:

| Accepted surface | R1 exact amendment |
| --- | --- |
| Generation key / package / binding / receipt versions | Keep existing generation formula, fields, versions and paths. Add no package record: derive association identity from already verified generation/source/platform/architecture and exact expected embedded skills. Binding device/inode describe the current installed target, not mapped image proof |
| Compiled documentary identity | Build both generation `G` and association identity `D` into the binary. `D` is SHA-256 of ASCII `axiom-docs-association-v1\n<G>\n<C>\n<platform>\n<architecture>\n<S>\n`, substituting existing lowercase digests/full commit and canonical release row tokens. `S` is SHA-256 of sorted `H  skills/<active-name>/SKILL.md\n` lines for the four exact active embedded files; it is not the archive skill-format constants or a Runtime receipt |
| Binary discovery object, docs schema 1 | Keep current members; add required nullable `documentaryIdentity:H?` and required `associationProof:enum(compiled_descriptor,unavailable)`. `imageProof` remains present and is always `unavailable` in this implementation; never emit a false OS proof label. `binary.sha256` is the verified current on-disk digest |
| Verified state | `associationProof=compiled_descriptor`, non-null matching D/G and complete safe target/binding/inventory validation; all six paths only then. This replaces original-image proof as a verified prerequisite. Documentary identity may be emitted only after independent comparison with installed metadata |
| Unknown reason | Add `association_unavailable` for unavailable association. Preserve `image_proof_unavailable` in accepted enum for historical compatibility, but do not emit it in R1. Existing identity/hash/binding/unsafe/concurrent reasons and all exits/priorities otherwise unchanged |

Closed unshipped schema 1 gains these exact fields/reason by H1 approval; no
already published consumer exists to promise backward compatibility. T01 records
the explicit vector delta; future consumers reject unknown keys as before. The
binary compares its compiled G/D and its embedded skill bytes to verified external
inputs, never derives its own expected G/D from mutable local metadata. The current
disk binary must match the verified binding digest and release metadata, whose
source/target/content inputs must recompute compiled D/G. Do not execute another
binary during discovery. This association relies on the verified publication
chain and does not authenticate coordinated replacement of mutable local records.
No OS image identity follows from these checks. No executable-format parser added.

Build DAG: approved authored text + CLI descriptor + transform + embedded skill
inventory → generated docs/man/coverage and generation descriptor → final platform
binary compiled with descriptor → binary digest and external manifest/binding inputs
→ archive → external archive checksum. Descriptor excludes hashes of binary/archive
that contain it. Generated text does not embed its own final hash. Preserve full
source/output provenance without circular hashing or repeated runtime generation.
The pre-binary CLI descriptor must come from the same command tree as the final
binary (including the new inspect command); help parity tests catch drift.

The read path safely resolves the executable locator, anchors the private target
namespace, checks descriptor/current binding and full inventory, then reads a
complete revalidated snapshot. No OS image API adoption. Same-label divergent
content, stale/crossed binding, C2 synthetic B, root/leaf changes or missing proof
refuse. Current-path hash is never reported as original process-image proof.
Aliases grant read location only, never installation write authority. A new public
JSON path is a locator requiring revalidation, not a pinned file descriptor.

## Publication, coordination and recovery

1. Inspect exact independent binary/receipt/Runtime roots and expected revisions;
   include every new documentary effect in preview/digest authority.
2. Serialize publishers by **physical binary parent identity + exact target leaf**,
   independent of receipt root. Reuse existing lock primitives if sufficient;
   otherwise add one bounded installation lock, not a lock framework. Revalidate
   lock identity/content, target and effect revisions before commit; replacement
   locks/stale previews refuse. Never unlink/recreate a busy or unknown lock.
3. Stage complete generation under the anchored generation parent; validate
   inventory and descriptor, then publish immutable generation and binding before
   binary rename. Equivalent recognized bytes are a no-op; foreign/conflicting
   bytes are preserved. No mutable `current` pointer or activation registry.
4. Binary publication/read-back is the logical activation boundary. Confirmed
   binary remains confirmed across receipt/skills/ack/cleanup failures. Preserve
   existing ADR-0016 candidate-running proof; descriptor grants no receipt authority.
5. Readers pin/revalidate complete content or refuse on detected change. Unknown
   binary effect or marker state requires bounded inspection and fresh exact
   recovery authority through existing supported paths. Do not add a general
   recovery engine, automatic retry, rollback or generation GC. Retain complete
   unactivated generations, user notes and foreign residues as appropriate conflicts.

Historical D4's 21-transition fixture and lock layout are useful prior Evidence,
not adopted production algorithms. Reconcile only boundaries changed by the
association model. Fault tests cover incomplete/short write, quota/space, permissions,
precommit, uncertain rename, confirmed commit/preack and cleanup; inspect actual
committed effects, not merely fixture counters. No power-loss durability promise.

## Compatibility and qualification

| Producer/consumer or route | R1 requirement |
| --- | --- |
| Historical v1 / existing and updated readers | Preserve exact historical format, ownership and known receipt variants; docs absent, never manufactured |
| New v2 / old shell and Go | Refuse unknown format/extra entries before candidate execution; pinned consumers and execution sentinels |
| New v2 / updated shell and Go | Same closed validation, complete manifest and descriptor; shell has no new parser prerequisite |
| Owned v1→v2 | Use matching verified v2 facade/candidate, not assumption old binaries parse v2. Retain v0.1.0 floor objective UNVERIFIED until actual historical artifact/receipt/native routes qualify |
| v2 reinstall / v2→v2 | No-op equivalent delivery; otherwise exact preview, complete generation-before-binary and confirmed effects |
| Source install / raw copied binary | Explicitly unqualified docs or safe refusal/absence; no borrowing another release, no silent receipt conversion |
| Runtime skills / independent return | ADR-0016 unchanged; Claude convergence through authorized first-run. A/B shared-root conflicts visible; compatible return tested, no state downgrade |
| Hosts | macOS arm64 and Linux amd64/arm64 native; Windows withdrawal unchanged. Declare tested filesystem assumptions and observer coverage |

No floor, platform or Q6 requirement is reduced here. If qualification cannot be
completed, preserve UNVERIFIED and present exact gap for human disposition before
full delivery claims. Local hashes prove observed integrity, not origin; existing
trusted release acquisition is separate. No signing/attestation service added.

## One implementation with internal checkpoints

[Tasks R1](tasks-reconciled.md) preserves IDs but removes approval-per-task staging.
After baseline acceptance and a separate Implement prompt, execute one cohesive
change on PR #335 with internal checks:

- **Content/contract checkpoint:** T01/T05/T06 produce full corpus/reference,
  measurements and exact descriptor semantics; coverage/quotas freeze before T07.
  This preserves C1 and stops material overflow without preventing authorship.
- **Safety integration checkpoint:** T02/T03/T04 establish descriptor refusal,
  bounded confinement and publication boundaries before T08/T09 integrate writes.
  T07 codec/build work needs measured outputs/contracts, not process-image research.
  T10 reader can be developed against validated fixtures before T09 exists.
- **Qualification/readiness checkpoint:** T13–T16 integrate artifact/native/man/
  offline/Runtime evidence and review. Full AC-01–08/V1–V9 remain required. Green
  CI is not product acceptance, release authority or a Linear transition.

These checkpoints are executable verification within one Implement, not sixteen
human microgates. Material contract change, quota excess/incomplete coverage,
unsafe/unprovable association or missing required native controls still stops the
affected work. Report remaining decisions together; independent safe work may
continue only within that later prompt's authority. Disposable hosts, real Runtime
sessions or privilege changes are never authorized merely by this Plan.

## Versioned strict verifier amendment (2026-10-21)

Rafael's approved Option A is defined by the normative
[distribution/integrity/bootstrap v1 amendment](distribution-helper-v1.md).
It adds three independently authenticated helper release assets and explicit
remote/offline trust handoff, without changing archive metadata/member formats,
legacy multistream semantics or default generation. New facade requires the
trusted helper before execution; previous Python-only v2 framing prerequisite
is superseded for this facade. Historical Evidence and accepted identity,
publication and Q6 boundaries remain intact.
