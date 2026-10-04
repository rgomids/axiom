# ADR-0017 — RecognizedPOC preservation archive and transition protocol

## Status

Proposed on 2026-10-04 for
[Issue #153](https://github.com/rgomids/axiom/issues/153) (I153-T02/I153-T03).
Becomes Accepted when the human reviewer merges the PR that introduces it.
It implements, and does not change, the human decisions recorded on
2026-10-01 (Issue #153, Specification 004 FR-026–FR-028, Plan §21). ADR-0005
and ADR-0007 continue to govern filesystem safety, authority and recovery.

## Context

The approved contract requires RecognizedPOC state to resolve through
preserve -> clean rebuild -> supported reconfiguration inside the owned
upgrade: complete verifiable preservation in a dedicated Axiom-owned
machine-local archive, retirement only after source inventory <-> final
manifest correspondence, historical POC workflow truth never promoted, and
recoverable interruption. Specification 004 asks for an ADR reassessment if
the solution introduces a new durable compatibility store; the preservation
archive is such a store, so its concrete shape is recorded here.

## Decision

1. **Location.** The archive namespace is `archive` beside the installation
   receipt directory, or the absolute `AXIOM_ARCHIVE_ROOT`. It must be a
   private directory owned by the user (or absent and created so), outside
   and not containing any Projects, State or Skills root. One operation's
   archive is `recognized-poc-<source inventory digest>`.
2. **Layout.** Objects are content-addressed (`objects/<sha256>`), so a
   relative path from the source is only data in the manifest and can never
   address the archive filesystem. `manifest.json` is written last, through a
   private stage published without replacing any name.
3. **Policy `recognized-poc-preservation/v1`.** The preserved set is every
   inventoried object of the Projects and State roots. Skill files in a
   Runtime root are not preserved or retired here: they are published Axiom
   content converged by the owned skill upgrade under its own ownership rules.
4. **Retirement set.** Exactly the historical workflow material is retired
   from active state: `workflows/` records and the Work Item links the POC
   workflow created. Installation records and portable manifests stay in
   place because the current contract already validated them while
   inventorying; that is the supported reconfiguration. Work Item links are
   retired rather than kept because they were produced by, and only
   meaningful to, the POC workflow; they remain inspectable in the archive.
5. **Correspondence.** Retirement requires: manifest objects equal the
   revalidated inventory (category, relative path, kind, digest, bytes);
   every listed object re-verifies from the destination; the archive holds no
   other object or stage; no active object lies outside the manifest. The
   verified manifest digest is then recorded in the operation marker before
   the first retirement, and a resume accepts only that manifest.
6. **Activation.** After the last retirement the active roots must resolve
   `direct` under the unchanged policy before any installation effect runs.
7. **Retention.** The upgrade never deletes the archive. No automatic or
   age-based cleanup exists; removal is a manual user action until a
   separate explicit, reference-aware cleanup capability is specified.

## Alternatives

- **Mirror the source tree in the archive.** Human-readable paths, but every
  relative path becomes a filesystem address in the archive, adding
  traversal/collision surface that content addressing removes.
- **Keep POC Work Item links active.** Fewer re-selections after upgrade, but
  activates records the POC workflow created, against "reconstruct only
  explicitly supported information".
- **Reuse `compatibility backup`.** It requires the same filesystem and has
  no retirement or correspondence proof.

## Consequences

- An upgrade over POC state completes without manual compatibility commands,
  leaves the Project configured, and reports the archive path.
- A user who wants a POC Work Item again selects it with the current
  `work-item` commands; the historical record stays in the archive.
- Harder to change later: the archive name, `objects/<sha256>` layout,
  manifest schema and policy name, and the `transitionArchive` /
  `transitionManifest` marker fields are durable once released.
