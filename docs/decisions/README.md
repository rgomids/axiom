# Architecture Decision Records

Este diretório registra decisões técnicas duráveis e difíceis de reverter.

## Index

- [ADR-0021 — Windows Server AMD64 native acceptance](0021-windows-server-amd64-native-acceptance.md) — Accepted, 2026-10-09; explicit maintainer authorization for member Server AMD64, preserving local NTFS protections and separate client coverage.

- [ADR-0020 — Workflow definition ownership and revision binding](0020-workflow-definition-revision-binding.md) — **Accepted, 2026-10-09** (human decision in [Issue #271](https://github.com/rgomids/axiom/issues/271#issuecomment-6074127314)); Project-owned portable definitions and selection, JCS/SHA-256 revisions, immutable local snapshots and new linked Executions for rework; historical sequential and graph decisions remain intact. Implementation requires separate authority.

- [ADR-0001 — Project is not Repository](0001-project-is-not-repository.md) — Accepted, 2026-08-08.
- [ADR-0002 — Axiom Relationship with GitHub Spec-Kit](0002-axiom-speckit-relationship.md) — Accepted, 2026-09-10.
- [ADR-0003 — Lingo as Axiom Local Control Plane](0003-lingo-as-axiom-local-control-plane.md) — Accepted, 2026-09-10; implementation requires an approved Specification.
- [ADR-0004 — Portable Project Manifest](0004-portable-project-manifest.md) — Accepted, 2026-09-12; versioned portable intent, identity/evolution and local-state separation, with explicit Git authority boundaries.
- [ADR-0005 — Bounded local filesystem threat model](0005-bounded-local-filesystem-threat-model.md) — Accepted, 2026-09-20; exact supported local filesystem properties and explicit same-UID/power-loss/media exclusions from HD-3.
- [ADR-0006 — Machine-local detail artifacts](0006-machine-local-detail-artifacts.md) — Accepted, 2026-09-20; central local ownership, Execution-first correlation, Evidence references, retention and safe cleanup from HD-2.
- [ADR-0007 — Local publication and recovery protocol](0007-local-publication-and-recovery-protocol.md) — **Accepted, 2026-09-20**; shared logical publication states, coordination, commit truth, prior/new generations and fail-closed recovery under ADR-0005.
- [ADR-0008 — Minimal machine-local Execution record](0008-minimal-machine-local-execution-record.md) — **Accepted, 2026-09-20**; bounded sequential workflow authority, revisioned transitions, resume and cross-boundary correlation without defining a general Execution graph.
- [ADR-0009 — Parent/child Execution Graph](0009-parent-child-execution-graph.md) — **Accepted, 2026-09-26**; revisioned DAG, lineage, authority, isolation, integration, structured coordination and cross-Runtime Evidence. S8/T30–T36 implementation was authorized on 2026-09-27 (Issue #97 comment #5852650410); the real T36 Runtime run remains separately gated.

- [ADR-0010 — Windows native filesystem boundary](0010-windows-native-filesystem-boundary.md) — **Accepted for the requested implementation, 2026-09-30**; native Windows security boundary. Upstream acceptance remains subject to PR review; release publication requires separate authority. Partially superseded by [ADR-0015](0015-installer-host-eligibility-os-family-architecture.md) and [ADR-0021](0021-windows-server-amd64-native-acceptance.md): its OS-version bound ("Windows 10 (1809+) and Windows 11") is preserved struck through and annotated in place; the rest remains in force.
- [ADR-0010 — Pinned metadata recovery for an unpublished release](0010-pinned-release-corrections.md) — Accepted direction, 2026-10-02; opt-in pinned release metadata recovery, preserving original source provenance and human publication gates. Implementation review, merge and publication remain separately gated.
- [ADR-0011 — Command-driven release start](0011-command-driven-release-start.md) — Proposed for human review, 2026-10-02 (its Status records that merging it with its implementation accepts it); merges integrate code, `$axiom-release` starts releases after a preflight of the whole release range.
- [ADR-0012 — Environment-scoped release publication credential](0012-release-publication-credential.md) — Proposed for human review, 2026-10-03 (its Status records that merge accepts the repository change); the environment-scoped publication credential required for preserved historical release sources.
- [ADR-0013 — Explicit control-code repair of a published recovery release](0013-published-release-control-repair.md) — Proposed for human review, 2026-10-03 (its Status records that merge accepts the repository change); explicit published-release control-code repair, preserving original metadata pins and binding a separate execution SHA.

- [ADR-0014 — Canonical maintainer skills with runtime-native discovery](0014-canonical-maintainer-skills-runtime-discovery.md) — **Accepted, 2026-10-03** (human decision in Issue #174); `.agents/skills/*` is the canonical runtime-neutral maintainer-skill source, `.claude/skills/*` are closed-allowlist discovery symlinks; supersedes only the repository bootstrap-policy fragment that rejected every `.claude/` entry, not an ADR.

- [ADR-0015 — Installer host eligibility is OS-family and architecture based](0015-installer-host-eligibility-os-family-architecture.md) — **Accepted, 2026-10-04** (human decision in Issue #183); installer eligibility is OS family, architecture and real prerequisites, never the numeric OS version; installation eligibility, support commitment (vendor-maintained OS versions) and acceptance Evidence are separate; `macos-27-*` is legacy row naming. Supersedes only the OS-version fragment of ADR-0010. Partially superseded by [ADR-0021](0021-windows-server-amd64-native-acceptance.md).

- [ADR-0016 — The owned upgrade publishes the candidate's Codex skill-set receipt when it runs as the candidate](0016-candidate-derived-skill-set-receipt.md) — **Accepted, 2026-10-04** (human decision; Issue #186); the upgrade derives the Codex skill-set receipt from the running binary only when its clean release provenance and embedded skills equal the verified candidate, as a separate authorized effect over an absent or Axiom-recognized receipt; replaces the interim S7/T20 `refresh_required` limitation without rewriting its Evidence.

- [ADR-0017 — RecognizedPOC preservation archive and transition protocol](0017-recognized-poc-preservation-archive.md) — **Accepted, 2026-10-04** (human decision; Issue #153); content-addressed Axiom-owned archive beside the receipt directory (or `AXIOM_ARCHIVE_ROOT`), manifest written last, complete inventory correspondence bound to the operation marker before retirement, only POC workflow records and their Work Item links retired, never deleted by an upgrade.

- [ADR-0018 — ADR evolution and supersession governance](0018-adr-evolution-and-supersession-governance.md) — **Accepted, 2026-10-05** (human decision in Issue #169); partial supersession keeps `Accepted` with a canonical in-place annotation and a `Partially superseded by` index note (not a lifecycle status); full supersession uses `Superseded`; superseding ADRs link back in `## Supersedes`; `scripts/check-adr-governance.py` validates structure, review owns semantic conflicts.

- [ADR-0019 — Windows default onboarding and consent-gated permission repair](0019-windows-default-onboarding-and-permission-repair.md) — **Accepted for the requested implementation, 2026-10-08**; private profile storage for fresh installations, preserved legacy locations, and exact-authority Runtime directory repair with backup and recovery. Extends the Windows boundary without trusting extra principals; merge and release remain separate.

## Candidate assessment

| Candidate | Current classification | ADR now? |
|---|---|---|
| Codex-first harness | Accepted bounded scope for the Agent Factory renderer (Codex only); maintainers may use Codex or Claude. | No for the renderer. Maintainer skill discovery across runtimes is Accepted in ADR-0014. |
| Go for future CLI | Go foundations implemented; CLI framework and distribution remain undecided. | Later, with an approved CLI specification and alternatives evidence. |
| Provider abstraction | Accepted boundary principle; concrete ports and adapters remain open. | Later, when a specified integration creates a durable contract. |
| Local versus remote control-plane topology | Local Lingo direction accepted; remote and hybrid alternatives remain revisit paths. | Accepted in ADR-0003; detailed behavior still requires Specification evidence. |
| Portable Project Manifest | Versioned shareable Project intent, currently axiom.yaml, distinct from local state. | Accepted in ADR-0004; concrete v1 contract belongs to Specification 002/Plan. Identity/location, incremental update and optional Git backing authority extend this boundary; concrete storage/sync engines and aggregate ownership remain open. |
| Spec-Kit relationship | Independent Axiom implementation informed by Spec-Kit as a strategic upstream reference. | Accepted in ADR-0002; future adapters or compatibility contracts require separate evidence and approval. |
| Execution and Evidence model | ADR-0006 resolves detail-artifact ownership/correlation; ADR-0008 remains the accepted sequential/historical contract. Issue #97 requires parent/child DAG semantics accepted in ADR-0009. | ADR-0009 is Accepted because graph lineage, authority, dependencies, integration, cancellation/retry, coordination and Evidence are durable/cross-cutting. Its acceptance does not authorize S8 implementation. |
| Local filesystem threat model | Bounded local MVP model with required confinement, process concurrency, deterministic faults, complete canonical state and guided recovery; arbitrary malicious same-UID interleavings and physical/media durability excluded. | Accepted in ADR-0005; concrete mechanisms remain Plan/implementation work. |
| Local publication and recovery protocol | Specification 004 Plan defines shared coordination, private preparation, protected publication, commit-point, prior/new generation and reader/recovery semantics across mutable local stores. | ADR-0007 is Accepted. It consumes ADR-0005 without turning syscalls, filenames, layouts, libraries or Go packages into architecture. |
| Lingo local control plane | Accepted architectural direction separating executable workflows from the Axiom domain and thin runtime skills. | Accepted in ADR-0003; do not implement before an approved Specification. |

ADRs futuros devem registrar, no mínimo:

- status;
- contexto;
- decisão;
- alternativas consideradas;
- consequências e trade-offs;
- evidências ou specifications relacionadas.
- `## Supersedes`, quando substituir total ou parcialmente outro ADR, conforme [ADR-0018](0018-adr-evolution-and-supersession-governance.md).

Hipóteses de pesquisa não são decisões. Não crie um ADR apenas para preencher a árvore documental.
