# Specification 007 amendment — AXM-22 Process mode and effort-source identity

> **AXM-36 prospective reconciliation — 2026-10-19.**
> [ADR-0032](../../decisions/0032-runtime-admission-without-certification.md#supersedes)
> was [accepted at `b4baf106`](https://github.com/rgomids/axiom/pull/334#issuecomment-6278024276).
> Only PM-5/PM-5a, PM-7, PM.4, PM-13 and the exact-identity declaration portions of BI-2/BI-2a and PM-2
> change within [RA-6](amendment-axm36-runtime-admission.md#ra-6--verified-coupling-and-prospective-reconciliation).
> Historical text and Evidence below are preserved. Profile ownership, safe
> source inspection, identity, policy, authority and supervisor contracts remain.
> The [Proposed Plan](plan-axm36-runtime-admission.md) specifies the bounded
> structural retirement and successor-format design; it grants no implementation.

## Status

**Approved implementation direction, 2026-10-14**, by explicit maintainer
decision (Rafael Gomides), recorded in the AXM-22 BLD-001A execution brief.
Owner: AXM-22 — Production BindingSource, child of AXM-8
([#276](https://github.com/rgomids/axiom/issues/276)), under accepted
I276-T16/T17 ([Plan Part II](plan-tasks-276-stage-agent-execution.md#i276-t16--capability-probe-preview-apply-and-execution)).

Clause **BI-1** (Process mode identity) is partially implemented by BLD-001A.
Clause **BI-2** (effort-source identity inspection) is approved. Its configured
carrier — local configuration format 3 with `EffortDeclaration` — was decided
on 2026-10-14 (D1–D8 below) and its codec/validation is implemented by
BLD-001B.1; the BI-2 effort inspector is implemented by BLD-001B.2a; the
production BindingSource is implemented by BLD-001B.2b
([implementation record](issue-276-implementation.md#axm-22-bld-001b2b--production-bindingsource));
explicit v2→v3 migration, declaration authoring and v3 publication are
implemented by BLD-001B.3b
([implementation record](issue-276-implementation.md#axm-22-bld-001b3b--explicit-v2v3-migration-authoring-and-atomic-publication)).
Clause **BI-2b** (path-operand reliability) is implemented and clause **BI-3**
(explicit v2→v3 migration and minimal authoring contract, approved 2026-10-14)
is specified by BLD-001B.3a and implemented by BLD-001B.3b; composition of the
production BindingSource in `cmd/lingo` remains BLD-001B.3c.

**Pre-MVP canonical format decision, 2026-10-14** (clause **PM** below):
configuration v3 is the only target canonical format before the first stable
release. PM **supersedes prospectively** the BI-3 v2→v3 migration (A1, BI-3.1
`--target-format 3`, BI-3.2) and every obligation to preserve v1/v2
compatibility; the BLD-001B.3b migration is historically implemented and was
removed by BLD-001B.3c-1. The dated records above and below are history and are
not rewritten. v3-only activation of authoring and consumers is pending
(BLD-001B.3c-2, PM-3).

**Pre-stable compatibility and DigestV3 decisions, 2026-10-14 13:30**
(clause [PM.2](#pm2-pre-stable-compatibility-and-canonical-digestv3-bld-001b3c-2a)):
the pre-stable period has no mandatory backward compatibility window and
existing installations reconfigure explicitly (PM-8); the canonical v3
identity is `DigestV3` (PM-9); BLD-001B.3c-2 is divided into 3c-2a/2b/2c
(PM-10).

**Pre-stable reconfiguration and application upgrade classification,
2026-10-14 15:51 BRT** (clause
[PM.3](#pm3-pre-stable-reconfiguration-and-application-upgrade-classification-bld-001b3c-2b3)):
a pre-stable record is reconfigured only by its explicit full retirement and
fresh v3 setup, and application upgrade is separated from Runtime/Profile
operational admission (PM-11). Canonical v3-only activation is delivered by
BLD-001B.3c-2b.3
([implementation record](issue-276-implementation.md#axm-22-bld-001b3c-2b3--coordinated-canonical-v3-only-activation)).

**Authorization epoch and production composition, 2026-10-14** (clause
[PM.4](#pm4-authorization-epoch-and-production-capability-composition-bld-001b3c-3)):
PM-6 is implemented as an internal epoch record under the AXM-22 final
delivery brief, and the production capability Service composes the
BindingSource. The Stage effort-inspection context — stable host/repository
roots, per-attempt child equivalence and operation consent — is the accepted
maintainer decision PM-13 (approved with constraints, 2026-10-14).

This amendment **adds** to the accepted
[#276 amendment](amendment-276-stage-dispatch.md) A2,
[ADR-0024](../../decisions/0024-model-profile-effort-ownership.md) and
[ADR-0025](../../decisions/0025-public-stage-dispatch-and-capability-proof.md).
It rewrites no accepted Plan task, approval receipt, ADR or historical Evidence,
and grants no authority, consent, Provider or Runtime authorization.

## BI-1 — Process is an explicit invocation-mode identity dimension

1. `capabilityproof.Target` carries `Process bool` beside `RuntimeID`,
   `ProfileID` and `Variant`. Target identifies the **requested invocation
   mode**; it never proves a capability.
2. Process feeds the effective `runtimeadapter.StageInvocationRequest.Process`.
   The adapter owns its effect on argv: for Claude 2.1.296 it widens the
   permitted Bash tools from the pinned `axiom` command to `Bash`, so the
   canonical `runtimeadapter.ModeDigest` differs between the two modes. Where an
   adapter emits an identical effective invocation for both modes (Codex
   0.159.1 today), the canonical digest is identical by construction and the
   `process` capability check alone gates the request.
3. Exactly one mode-digest algorithm exists: `runtimeadapter.ModeDigest`
   (`axiom-runtime-mode-v1`). No second digest, encoding change or
   `AdapterVersion` change is introduced.
4. Probe preparation, Evidence selection, preview and dispatch-time
   revalidation use the same requested mode. Preview and dispatch derive
   Process from the child's authorized `process` effect — the same source the
   supervisor applies to the dispatched request. The Runtime policy observer,
   whose request carries capabilities rather than effects, sets Process from
   the requested `process` capability, as it already does for the variant. A
   mode mismatch selects a different binding and is rejected
   (`capability_stale`), never admitted.
5. Fail closed: a check whose requested Process contradicts the Target it read
   blocks as `capability_stale`; an Observer whose Targets mix Process modes
   blocks; Process is never derived from stored Evidence, model identity or
   vendor version, and an unknown Process is never replaced by `false`.
   `Process=false` is a valid explicit mode, not a universal default.
6. `Target.Process` names the mode; Evidence still decides whether the
   `process` capability is proven. Existing `process`, `repository-write`,
   `noninteractive`, `model-invocation` and `axiom-coordination` checks remain.
7. Unchanged: `Binding`, `ModelBinding`, Evidence schema/encoding/digests,
   persisted Runtime/Profile formats, selector cardinality and historical
   Evidence meaning.

Consumer: the production `BindingSource` computes `Binding.ModeDigest` from
the Target's Variant **and** Process through `runtimeadapter.ModeDigest`
(BLD-001B.2b). Production capability reads stay blocked until a v3 local
configuration can be authored and read in composition (BLD-001B.3).

## BI-2 — Effort-source identity inspection independent of capability proof

Approved. Implemented by BLD-001B.2a as the internal
`runtimeadapter.InspectEffortSources`, sharing the PE-05a projection
([implementation record](issue-276-implementation.md#axm-22-bld-001b2a--canonical-effort-source-inspector));
its production BindingSource consumer is BLD-001B.2b. A narrow internal effort
inspector produces the canonical `EffortSourceDigest` for the production
BindingSource:

1. **Same digest.** It reuses the existing canonical effort-source projection
   and digest algorithm (`runtimeadapter.EffortInspection.CheckEffortSources`,
   PE-05a). Equivalence tests against that algorithm are required.
2. **Versioned, configured declaration.** Its input is a versioned
   `EffortDeclaration` obtained through the configured Runtime/Profile,
   consistent with ADR-0024 ownership (the Model Profile owns effort intent; the
   adapter owns recognized sources and precedence). Minimum properties, in
   existing terms: Runtime ID, invocation mode, the executable identity it
   applies to, recognized source scopes/formats, recognized JSON and
   environment keys, and whether precedence is established — plus a
   declaration version.
3. **Applicability.** The declaration is valid only when it matches the
   independently observed current executable identity and invocation mode.
4. **Independence.** It requires no already capability-proven Profile, no
   ProfileProof and no BindingSource output, so no circular dependency arises.
   Source identity is separate from capability proof.
5. **No vendor-version inference.** Codex 0.159.1 and Claude 2.1.296 are
   decision context only; a vendor version string never establishes effort
   support and never becomes a hardcoded declaration.
6. **Fail closed.** A missing, ambiguous, inapplicable or unversioned
   declaration blocks as `effort_configuration_unproven`.
7. No new Provider or Runtime authorization, consent or persisted Evidence
   change.

**Open schema question (historical, as recorded by BLD-001A).**
Local configuration format 2 (`runtimeprofile.Configuration`, `Runtime`,
`ModelProfile`) has no field that carries an `EffortDeclaration`. If BLD-001B
cannot obtain the declaration from an existing configured carrier, the
persisted location and field names require an explicit maintainer schema
decision before implementation; this amendment invents none. *Resolved by
BI-2a below.*

## BI-2a — Configuration v3 carrier for EffortDeclaration (BLD-001B.1)

**Approved maintainer decisions, 2026-10-14** (Rafael Gomides, AXM-22
BLD-001B.1 execution brief). No new ADR (D4); ADR-0024 ownership unchanged.

- **D1.** Local configuration format 3 is the official persisted carrier of
  `EffortDeclaration`.
- **D2.** `EffortDeclaration` belongs to Runtime/executable identity. Effort
  intent remains in `ModelProfile` (ADR-0024).
- **D3.** Format 2 stays readable with its original semantics. A missing
  declaration yields `effort_configuration_unproven`. No implicit migration.
- **D4.** The schema amendment is recorded here; no new ADR.
- **D5.** Codec and validation only. v3 serialization exists only for
  deterministic test fixtures; production v3 write/publication, public CLI
  flags, v2-to-v3 migration and changes to the v1-to-v2 migration are not
  authorized. D5 is an intermediate delivery, not operational acceptance.
- **D6.** Source locations use the closed anchors `user`, `project`,
  `managed` plus a validated relative path. No absolute host path or home
  directory is persisted. The existing local filesystem trust boundary is
  preserved; no new permission model.
- **D7.** `declarationId` is mandatory and maps 1:1 to the existing
  `runtimeadapter.EffortDeclaration.ProofReference`. It identifies the
  configured declaration; it is not capability Evidence and never makes an
  effort capability appear proven.
- **D8.** Production paths that support format 2 only refuse format 3
  explicitly until BLD-001B.2/B.3 implement it; no silent downgrade, ignored
  declaration or reinterpretation as a format-2 record.

Contract (BLD-001B.1):

1. **Ownership and shape.** `runtimeprofile.ConfigurationV3` is the format-2
   shape with `formatVersion: 3` and an optional
   `runtimes[].effortDeclarations[]`. The enclosing Runtime supplies the
   declaration's Runtime identity (no duplicated `runtimeId`). Every format-2
   Runtime/Profile/Preference rule applies unchanged; `ModelProfile` keeps
   effort intent and its Profile digest. Omission is the only spelling of "no
   declaration" (`null`/`[]` refused).
2. **Declaration fields (all required, closed, no defaults).**
   `declarationId` (lowercase token ≤ 64 bytes, safe identifier),
   `declarationVersion` (`1`), `runtimeVersion` and `executableDigest` (the
   applicable executable identity), `invocationMode` (`noninteractive`),
   `precedenceEstablished` (must be `true`; the adapter owns the order),
   `jsonKeys`, `environmentKeys` (safe keys, ≤ 16 each, unique; environment
   keys case-insensitively) and `sources[]` of {`scope`, `anchor`, `path`,
   `format`, `directory`}.
3. **Recognized sources** mirror PE-05a: Codex `user`/`project`/`managed`
   TOML (`config.toml`, `managed_config.toml`) with no JSON keys; Claude adds
   `project-local` and a `managed-drop-in` JSON directory, requires at least one
   JSON key and `CLAUDE_CODE_EFFORT_LEVEL`. Each required scope appears exactly
   once (two sources in a scope are ambiguous precedence); the anchor must own
   the scope (`project-local`→`project`, `managed-drop-in`→`managed`).
4. **Relative paths** are canonical slash-separated paths ≤ 256 bytes over
   `[A-Za-z0-9._ -]`: no absolute/UNC/drive/URI/`~` form, `\`, `:`, empty,
   `.` or `..` segment, control/NUL/non-ASCII character, or segment with
   trailing dot/space. Lexical validity does **not** prove filesystem safety:
   resolution below an anchor must use the existing trusted boundary
   (PE-05a), which refuses symlinked or special components.
5. **Uniqueness and applicability.** `declarationId` is unique within the
   configuration (keeping the ProofReference mapping 1:1); at most 8
   declarations per Runtime, one per (`runtimeVersion`, `executableDigest`,
   `invocationMode`). A declaration applies only to an exact, independently
   observed Runtime/version/executable/mode; decoding never claims that
   observation. A vendor version never creates or selects a declaration.
6. **Codec.** Strict decoding (closed objects, duplicate/case-alias/unknown
   member refusal, size bound) and canonical encoding (set-like collections
   ordered; source order carries no precedence). `DecodeVersioned` dispatches
   1/2 to the unchanged `Decode` and 3 to the v3 decoder; newer formats are
   `unsupported_configuration_format`.
7. **Historical preservation.** Format-1/2 bytes, `Digest` and Profile digests
   are unchanged; history gains no synthetic declaration, is never rewritten,
   and its effort configuration is unproven.
8. **Explicit refusal (D8).** `Decode`/`Validate`/`Encode`/`Digest` return
   `ErrUnsupportedConfigurationFormat` (still an `ErrInvalidConfiguration`)
   for format ≥ 3; the local snapshot reader, `Load`, `Publish` and legacy
   `Save` refuse without mutation; inventory keeps `unsupported_newer`;
   authoring keeps the public `configuration_unavailable` category with an
   actionable next step. No new public error taxonomy.
9. **Declaration ≠ proof.** The declaration has no capability status; a
   `declarationId` equal to a proof reference never satisfies `ProfileProof`.
10. **Writer boundary (D5).** The only v3 encoder is unexported and used by
    package tests; no production path can publish format 3. *(Superseded for
    the BI-3 path only: BLD-001B.3b exports it as `EncodeV3`, used solely by
    the explicit migration and declaration authoring.)*

Delivery boundaries:

- **BLD-001B.2a (implemented):** narrow effort inspector, shared canonical
  `EffortSourceDigest` algorithm with digest-equivalence tests, anchor-root
  resolution.
- **BLD-001B.2b (implemented):** production BindingSource and AXM-21 Evidence
  selection.
- **BLD-001B.3b (implemented):** authorized v3 writing/publication and
  supported local authoring integration (BI-3).
- **BLD-001B.3c (pending):** synthetic end-to-end configuration validation and
  production BindingSource composition.
- New public CLI flags or v2-to-v3 migration require a separate explicit
  maintainer decision before implementation.

## BI-2b — Stage path operands are not effort overrides (BLD-001B.3a)

**Approved maintainer decision, 2026-10-14 10:27 America/Sao_Paulo** (Rafael
Gomides, AXM-22 BLD-001B.3a brief, A5): correct the reliability finding that
refused a workspace merely because its path contains `effort`; preserve real
override detection and canonical `EffortSourceDigest` semantics.

Finding (recorded by BLD-001B.2b): the PE-05a projection scanned every argv
token after `<exe> <mode> --model <model>` with
`strings.Contains(lower, "effort")`. That scan also covered the operands of the
adapter's own filesystem options emitted by `BuildStageInvocation` — Codex
`-C <cwd>` and `--add-dir <inbox>`, Claude `--add-dir <inbox>` and the
`Bash(<axiomPath>:*)` element of `--allowedTools` — so a workspace, inbox or
Axiom path spelling `effort` failed as `effort_override_conflict`.

Contract:

1. Only the operand of one of those adapter path options is exempt from the
   substring scan, and only when it is a clean absolute path (`stagePath`); for
   `--allowedTools` only a `Bash(<clean absolute path>:*)` element is exempt and
   an operand starting with `-` is never exempt. Option names and every other
   token keep the unchanged scan (`effort` substring, `--config`, `-c`,
   `--profile`, `-p`).
2. Relative, unclean or non-path operands of those options are still scanned,
   so `--add-dir model_reasoning_effort=low` or `-C effort/work` still refuse.
3. Environment, recognized source, explicit-override, symlink, grammar and
   precedence checks are unchanged.
4. Argv tokens contribute no projection fact, so the canonical projection,
   field set, encoding and SHA-256 `EffortSourceDigest` are unchanged for every
   previously accepted input, and no path enters the digest. No allowlist of
   user paths and no global suppression of the word `effort` is introduced.

## BI-3 — Explicit v2→v3 migration and minimal EffortDeclaration authoring (BLD-001B.3)

> **Superseded in part by [PM](#pm--pre-mvp-canonical-configuration-format-bld-001b3c).**
> A1, the `--target-format 3` row of BI-3.1, BI-3.2 (migration contract) and
> the `active-v2 → v3` publication transition no longer apply. A2–A4,
> declaration authoring on `binding edit`, preview, exact confirmation,
> revalidation, atomic publication, recovery and BI-3.3 refusals remain in
> force. The text below is preserved as the 2026-10-14 record.

**Approved maintainer decisions, 2026-10-14 10:27 America/Sao_Paulo** (Rafael
Gomides, AXM-22 BLD-001B.3a brief):

- **A1.** Explicit migration from local configuration format 2 to 3; never
  automatic.
- **A2.** Minimal public authoring extension for `EffortDeclaration`, reusing
  existing commands.
- **A3.** Preview, explicit confirmation, atomic publication and recovery.
- **A4.** Refusal of missing, ambiguous or inapplicable declarations, including
  a mismatch against the independently observed executable identity.

These approvals supersede BI-2a **D5** only for the production v3 write path
defined here (`migrate` and declaration authoring through the existing local
store). They authorize no silent migration, no other v3 writer, no
filesystem-safety weakening, no Evidence or digest change, no Runtime probe or
inference and no Provider effect. No new ADR: the contract stays inside
ADR-0024 ownership and the accepted local publication protocol.

BLD-001B.3a recorded this contract; BLD-001B.3b implements BI-3.1–BI-3.3
without changing it (see the implementation record for exact behavior).

### BI-3.1 Public interface (reuse, no new subsystem)

The existing machine-local commands `axiom runtime binding <op>` and their
authority flags `--expected-revision`, `--preview-digest` and
`--authorize-local` are reused unchanged. The extension adds exactly two flags:

| Command | New flag | Meaning |
|---|---|---|
| `axiom runtime binding migrate` | `--target-format 3` | Selects the v2→v3 transition. Required whenever the active state is `active-v2`; refused when the active state is `active-v1` (v1→v2 stays the separate existing step, with its unchanged `--profile-effort` and `--clear-legacy-references`). |
| `axiom runtime binding migrate`, `axiom runtime binding edit` | `--effort-declaration <json-file>` | One complete declaration object in the BI-2a wire form for the Runtime named by `--runtime`. |

Rules:

- `--runtime` is required with `--effort-declaration`; the enclosing Runtime
  supplies the declaration's Runtime identity (BI-2a item 1).
- `--effort-declaration` is read once, bounded by the existing configuration
  size limit, as a regular non-symlinked file, and strictly decoded with the
  BI-2a declaration rules (closed object, no unknown, duplicate or case-alias
  member). Its bytes are not persisted verbatim; only the canonical v3
  encoding is.
- `migrate --target-format 3` requires exactly one declaration, for the named
  Runtime. Other Runtimes and every Model Profile, allowlist and Preference
  are carried over unchanged and without declarations (their effort stays
  `effort_configuration_unproven` until a declaration is authored).
- On an `active-v3` configuration, `binding edit --runtime <id>
  --effort-declaration <json-file>` adds a declaration or replaces the one
  with the same `declarationId`; a different `declarationId` with the same
  (`runtimeVersion`, `executableDigest`, `invocationMode`) tuple is ambiguous
  and refused. Removal of declarations is not part of this delivery.
- `--effort-declaration` on an `active-v2` configuration without `migrate
  --target-format 3` is refused (`migration_required`); no command migrates as
  a side effect.
- No other command, flag, output field or error category is added. Refusals
  use the existing categories (`invalid_input`, `migration_required`,
  `migration_mapping_required`, `candidate_invalid`, `stale_preview`,
  `local_authority_required`, `publication_unconfirmed`,
  `recovery_required`) plus the existing `effort_configuration_unproven` for an
  inapplicable declaration.

### BI-3.2 Migration contract

1. **Initiation.** Only an explicit `axiom runtime binding migrate
   --target-format 3` with `--runtime` and `--effort-declaration` starts it.
   Reads (`list`, `show`, inventory, BindingSource, stage dispatch), CLI
   upgrades and other mutations never migrate. A CLI version alone never
   selects a format.
2. **Source identification.** The source is the current canonical local
   configuration object read through the existing zero-write `Inspect` under
   shared locks. It must be `active-v2`; `fresh`, `retired`, `active-v1`,
   `active-v3`, `pending` and `unsafe` refuse.
3. **Exact source revision.** The preview binds the existing opaque snapshot
   token (filesystem facts, state and exact wire bytes) and the source
   `runtimeprofile.Digest` plus revision. The v2 bytes are decoded only by the
   unchanged v2 codec.
4. **Declaration input.** The supplied declaration must satisfy every BI-2a
   rule (`declarationId` mandatory and 1:1 with ProofReference,
   `declarationVersion` 1, executable identity, `noninteractive`,
   `precedenceEstablished`, recognized scopes, closed anchors, safe relative
   paths, explicit formats). Missing, duplicate, unversioned or unrecognized
   declarations refuse. Axiom never fabricates, completes or defaults a
   declaration, and never derives one from a vendor version.
5. **Independent applicability.** At preview the existing passive observation
   seams used by the BindingSource (`LookPath`, canonical
   `ExecutableIdentity`, the existing version observer keyed by that identity
   with the executable re-read for drift) must yield exactly the declaration's
   `runtimeVersion` and `executableDigest` for the named Runtime. Absent,
   unsupported, unobserved or mismatched identity refuses as
   `effort_configuration_unproven`. This is applicability only: no capability,
   authentication, model or effort proof is performed or implied, and the
   declaration never satisfies `ProfileProof`.
6. **Proposed output.** `formatVersion: 3`, `revision = source + 1`, the
   source Runtimes, Model Profiles (byte-identical effort and Profile digest)
   and Preferences, plus the declaration under its Runtime; encoded with the
   canonical v3 encoder and validated by `ValidateV3`. Every Runtime ID,
   Profile ID, allowlist entry and Preference of the source must be present and
   unchanged, or the candidate refuses.
7. **Preview identity.** Zero writes. The existing preview envelope digest
   (`previewDigest`) binds the snapshot token, the candidate digest, the
   effects (including `migrate local format 2 to 3`, the declaration ID and
   Runtime, and the observed Runtime version and executable digest it applies
   to) and the projected views.
8. **Confirmation.** Apply requires `--authorize-local` with the exact
   `--expected-revision` and `--preview-digest` of that preview, as for every
   existing local mutation.
9. **Revalidation before publication.** Apply re-inspects the snapshot,
   re-reads and re-decodes the declaration file, re-observes the executable
   identity and version, rebuilds the candidate and recomputes the envelope.
   Any difference in source token, declaration, observed identity, candidate
   or envelope is `stale_preview`; nothing is written.
10. **Atomic publication.** The candidate is published through the existing
    `RuntimeProfileStore.Publish` protocol (exclusive locks, compare against the
    exact prior wire, atomic replace, post-commit identity and byte
    verification). BLD-001B.3b extends that store with a format-3-aware
    inspection (`active-v3`) and accepts format 3 only as `active-v2 → v3`
    (migration) or `active-v3 → v3` (declaration authoring), with
    `revision = prior + 1`. It never down-converts v3 to v2, never accepts v3
    over v1, `fresh` or `retired`, and the legacy v2 `Load`/`Save` keep
    refusing v3 (D8). No guard is bypassed.
11. **Recovery and retry.** An interrupted or unverifiable publication is
    `publication_unconfirmed` / `recovery_required` (the existing F6 fault),
    never success. Recovery is the existing scoped `axiom runtime binding
    recover` with its exact authority. There is no automatic retry: after
    recovery the operator prepares a fresh preview and confirmation.
12. **Post-publication version handling.** The active object is now format 3;
    the transition is reported as an explicit effect. Prior v2 bytes are never
    re-encoded or rewritten as v2, the v2 codec, golden bytes and `Digest` are
    unchanged, and no v2 history gains a declaration. Older v2-only readers
    refuse the published format 3 with `unsupported_configuration_format`
    (D8) instead of misreading it. Downgrade from v3 to v2 is not provided and
    requires a separate decision.

### BI-3.3 Fail-closed conditions

Refuse with no write when: the state is not the required one; the declaration
is missing, unreadable, oversized, symlinked, malformed, ambiguous,
unversioned or for another Runtime; the observed executable or version is
absent or differs; the candidate loses or changes any Runtime, Profile or
Preference; `ValidateV3` fails; the preview or snapshot is stale; confirmation
is absent or inexact; or publication cannot be verified.

### BI-3.4 Delivery division

- **BLD-001B.3a (this block):** this contract; BI-2b path reliability fix with
  regression and digest-equivalence tests.
- **BLD-001B.3b:** v2→v3 migration and minimal declaration authoring
  (`runtimeapplication` + `internal/cli` flags), format-3-aware local
  inspection and publication, recovery, synthetic tests.
- **BLD-001B.3c:** composition of the production BindingSource in `cmd/lingo`
  `productionCapabilityService`, synthetic end-to-end validation, final AXM-22
  reconciliation.

## PM — Pre-MVP canonical configuration format (BLD-001B.3c)

**Accepted maintainer decisions, 2026-10-14 11:44 and 12:06
America/Sao_Paulo** (Rafael Gomides, AXM-22 BLD-001B.3c-1 brief), with the
sequencing reconciliation of the same day (Option C):

- **PM-1. Format policy.** Before the first MVP stable release, configuration
  v3 is the only canonical format. There is no v2 reading as a supported
  configuration, no v2→v3 migration, no automatic compatibility fallback and
  no v3→v2 downgrade. There is no obligation to preserve obsolete v1/v2 codecs,
  migration flows or compatibility fixtures. Strict backward compatibility
  starts with the first stable/LTS release.
- **PM-2. Authoring.** v3 authoring and atomic publication remain required:
  `axiom runtime binding edit --runtime <id> --effort-declaration <json-file>`
  with the existing `--expected-revision`, `--preview-digest` and
  `--authorize-local`, strict declaration validation, independent executable
  applicability, zero-write preview, exact confirmation, revalidation before
  publication, atomic publication and F3/F6 recovery. A declaration never
  becomes capability proof.
- **PM-3. Sequencing.** BLD-001B.3c-1 removes the v2→v3 migration but keeps
  the existing v2 authoring path and v2 consumers working for operational
  continuity (existing tests and CI). BLD-001B.3c-2 removes the remaining v2
  paths and switches authoring and consumers to v3 together. This is an
  intermediate boundary, not a reversal of PM-1: no v2 compatibility is
  accepted for the final MVP.
- **PM-4. Consumers.** Runtime policy, Workflow authoring and
  compatibility/upgrade checks become v3-only (BLD-001B.3c-2), with no
  validation weakening.
- **PM-5. Runtime version.** A passive `RuntimeVersionObserver` reads a
  versioned registry file in the repository whose records bind supported
  Runtime versions to exact executable digests. The Runtime is never executed
  to determine its version (BLD-001B.3c-2).
  - **PM-5a. Registry contract (BLD-001B.3c-2c.1).** The registry is
    `internal/runtimeadapter/runtime-versions.json`, embedded with `go:embed`
    into the Axiom build; production reads only those bytes
    (`EmbeddedVersionObserver`), never a filesystem path, environment value,
    flag or download. Wire version `axiom-runtime-version-registry/v1`:
    `{"schema","entries":[{"runtimeId","version","executableDigest"}]}`,
    decoded with exact member names (no unknown, duplicate or
    case-aliased members), at most 64 KiB and 256 entries; `runtimeId` is an
    adapter Runtime (`codex`, `claude`), `version` satisfies
    `SafeRuntimeVersion`, `executableDigest` is 64 lowercase hex characters,
    entries are strictly ordered by (`runtimeId`, `executableDigest`) and each
    digest appears once, so one Runtime/digest pair maps to exactly one
    version. Any other schema or shape refuses the whole registry. The digest
    is the canonical `ExecutableIdentity` (no second hash algorithm); the
    observer receives it from the caller, which observes and revalidates the
    executable for drift, so lookup reads no file and runs nothing. An
    unmatched pair is `capabilityproof.ErrBlocked` (the callers' existing
    blocked/unproven categories), with no fallback to filename, Runtime ID,
    PATH, EffortDeclaration, Profile, Evidence, vendor version or execution.
    A registered version is identity only: it proves no Runtime, model,
    effort, subscription or Provider capability and is not Capability
    Evidence. The production registry has zero entries until an executable
    identity is independently verified and approved; synthetic entries exist
    only in tests through the unexported decoder.
    *Limitation:* `ExecutableIdentity` binds the resolved path as well as the
    content, so a production entry would be specific to one installation
    path. Populating portable production entries needs a recorded decision
    and is not attempted here.
- **PM-6. Authorization epoch.** Preview authorization includes a persistent,
  monotonic authorization epoch that participates in preview identity. A
  preview issued before recovery must not become valid again after rollback
  (BLD-001B.3c-3). Its persisted representation is not decided here. Until
  then, reuse of an otherwise identical pre-rollback preview token remains a
  recorded pending security defect.
- **PM-7. Invariants.** ADR-0024 effort-intent ownership, the Runtime-owned
  `EffortDeclaration`, v3 canonical semantics, the
  `declarationId`/`ProofReference` relationship, canonical
  `EffortSourceDigest` and `ModeDigest`, existing Binding/ModelBinding
  contracts, AXM-21 Evidence selection, capability-proof validation,
  filesystem trust boundaries, explicit local authority and atomic
  publication/recovery safety remain mandatory. No Evidence-derived current
  identity; no declaration promoted to capability proof.

No new ADR: PM narrows the accepted local configuration lifecycle without
conflicting with an accepted invariant.

### PM.1 Delivery division

- **BLD-001B.3c-1:** contract reconciliation and migration removal. The local
  store accepts format 3 over fresh, retired or active-v3 state and never over
  a v1/v2 record. *Delivered; v3-only activation not claimed.*
- **BLD-001B.3c-2:** v3 consumers, v3-only authoring and the passive Runtime
  registry; removal of the remaining v2 authoring, v1→v2 migration and v2
  store paths. Divided on 2026-10-14 (PM-10) into 3c-2a, 3c-2b and 3c-2c.
  *Delivered by 3c-2a, 3c-2b.1–3 and 3c-2c.1 (PM.3).*
- **BLD-001B.3c-3:** persistent authorization epoch and `cmd/lingo`
  BindingSource composition. *Pending.*
- **BLD-001B.3c-4:** synthetic end-to-end validation and final AXM-22
  reconciliation. *Pending.*

### PM.2 Pre-stable compatibility and canonical DigestV3 (BLD-001B.3c-2a)

**Accepted maintainer decisions C1–C3, 2026-10-14 13:30 America/Sao_Paulo**
(Rafael Gomides, AXM-22 BLD-001B.3c-2a brief). Prospective: earlier clauses,
Evidence and acceptance receipts stay as history and are not rewritten.

- **PM-8. Pre-stable compatibility (C1).** Before the first stable/LTS release
  there is no mandatory backward compatibility window for local Runtime/Profile
  configuration. Previously published pre-stable formats (local format 1 from
  releases such as v0.4.0 and v0.10.0, and format 2) are outside the mandatory
  support window. Configuration format 3 is the only target canonical format;
  there is no direct v1→v3 upgrade, no v1→v2 or v2→v3 migration requirement, no
  automatic conversion and no compatibility fallback. An existing installation
  on an unsupported pre-stable format requires explicit reconfiguration; its
  active state refuses safely and is never silently deleted or converted.
  Strict backward compatibility begins at the first stable/LTS release.
  The frozen historical compatibility corpus (`internal/compatibility/testdata/stable-v1`)
  remains intact and append-only: its release snapshots and original bytes are
  historical Evidence, not disposable fixtures, and are never edited,
  relabelled or deleted. Amended prospectively:
  [ADR-0024](../../decisions/0024-model-profile-effort-ownership.md) (Status
  note), [Specification 004 FR-026](../004-mvp-v1-baseline/spec.md) and the
  local configuration rows and first-release window of the
  [#306 amendment §3](amendment-306-profile-effort-onboarding.md#3-versioning-and-explicit-migration-strategy).
  Workflow, Execution, graph and Provider compatibility are unchanged.
- **PM-9. Canonical v3 digest (C2).**
  `DigestV3(configuration) = SHA-256(EncodeV3(configuration))`, lowercase
  hexadecimal, over the complete canonical v3 configuration: `formatVersion`,
  `revision`, `runtimes` with their Runtime-owned `effortDeclarations`,
  `modelProfiles` and `preferences`. EncodeV3 is the only canonicalization; no
  secondary algorithm exists. An invalid configuration has no digest. Every
  valid change to a declaration field (`declarationId`, `runtimeVersion`,
  `executableDigest`, source path, scope/anchor, recognized keys, membership)
  changes the digest; declaration fields are included although they are not
  capability proof. DigestV3 hashes no source contents (EffortSourceDigest),
  host paths, executable bytes, Evidence, registry data, credentials or time.
  It does not redefine the historical format-1/2 `Digest`, and
  `ProfileDigest`, `EffortSourceDigest`, `ModeDigest`, `ExecutableDigest` and
  Evidence digests stay independent. Intended use from BLD-001B.3c-2b: active
  v3 identity is `DigestV3` of the current configuration, and a v3 retirement
  binds `priorConfigurationDigest` to `DigestV3` of the exact prior active v3
  configuration. The existing tombstone contract is unchanged until then, and
  no authorization epoch is introduced here (PM-6).
- **PM-10. Delivery division (C3).**
  - **BLD-001B.3c-2a:** PM-8 contract reconciliation and `DigestV3`.
    *Delivered by this block; consumers unchanged.*
  - **BLD-001B.3c-2b:** coordinated canonical v3 activation — authoring,
    consumers, admission, readiness, `profile validate`, Runtime retirement and
    the v3 compatibility/upgrade inventory. *Delivered by BLD-001B.3c-2b.3
    with PM-11 (PM.3).*
  - **BLD-001B.3c-2c:** passive `RuntimeVersionObserver`, versioned repository
    registry, synthetic v3 dogfood and remaining relevant fixture conversion.
    *BLD-001B.3c-2c.1 delivered the passive embedded registry (PM-5a) and
    isolated synthetic v3 dogfood fixtures, unwired; the dogfood cutover and
    fixture conversion move to BLD-001B.3c-2b.3, and `cmd/lingo` composition
    to BLD-001B.3c-3.*
  - BLD-001B.3c-3 and 3c-4 are unchanged (PM.1).

**Compatibility corpus test owed by BLD-001B.3c-2b.** The current inventory
still classifies local format 1 as `valid_v1`, so
`TestStableCorpusResolvesToDirectUpgrade` passes unchanged and is not modified
before the v3 inventory exists. When BLD-001B.3c-2b makes the inventory v3-only,
that test must keep iterating every frozen snapshot with unchanged bytes and
labels and keep its kind-coverage assertion, but split its expectation: every
non-Runtime/Profile v1 kind still resolves `direct`/`compatible`, while each
pre-stable `runtime_profile_configuration` record is reported as historical pre-stable state
whose current support classification is explicit reconfiguration (refused
safely, never converted or deleted), not a mandatory stable/LTS upgrade.
The expectation is changed, never weakened to suppress a failure, and no
snapshot is removed.

No new ADR: PM-8 is recorded as a dated ADR-0024 Status note, the repository
precedent for a human decision without a superseding ADR (ADR-0010, ADR-0015,
ADR-0019, ADR-0021), and changes no accepted invariant of PM-7.

### PM.3 Pre-stable reconfiguration and application upgrade classification (BLD-001B.3c-2b.3)

**Accepted maintainer decisions A1 and B1, 2026-10-14 15:51 BRT** (Rafael
Gomides, AXM-22 BLD-001B.3c-2b.3 continuation brief). Prospective: PM-1–PM-10,
earlier clauses, Evidence and acceptance receipts stay as history and are not
rewritten.

- **PM-11. Pre-stable reconfiguration and application upgrade
  classification.**
  - **PM-11a. Explicit full retirement of a pre-stable record (A1).** A valid
    active local configuration of format 1 or 2 is reconfigured only by the
    existing command `axiom runtime binding remove --runtime <id> --cascade`.
    On such a record that operation retires the ENTIRE local Runtime/Profile
    configuration, not only the named Runtime; the named Runtime must exist in
    the strictly validated record, and `--cascade` is required. The zero-write
    preview states that meaning and lists every retired Runtime (with its
    enablement and allowlist), every Profile and every Preference, the
    historical configuration digest and that nothing is created or migrated.
    Preview authority binds the exact prior state, revision, canonical source
    identity (the snapshot token over the exact bytes and filesystem facts),
    candidate digest and complete effects; it is confirmed only with the
    existing `--expected-revision`, `--preview-digest` and
    `--authorize-local`, and the source is re-inspected under the exclusive
    lock before publication. Publication writes the existing tombstone
    (`runtime-profile-tombstone`, format 1, revision+1) whose
    `priorConfigurationDigest` is the historical `runtimeprofile.Digest` of the
    exact prior record, through the existing atomic publication and F3/F6
    recovery; historical Digest semantics and the tombstone format are
    unchanged. No EffortDeclaration is synthesized and no legacy Profile is
    transformed. After retirement a fresh canonical v3 configuration is
    created only by separately authorized operations (`runtime binding
    create`); retirement never creates v3 and never resurrects legacy state.
    Every other operation on a pre-stable record (create, edit, enable,
    disable, Profile removal, candidate validation, declaration authoring) is
    refused (`unsupported_configuration_format`, or
    `configuration_unavailable` for a declaration) without writing. There is
    no new command or flag, no partial v1/v2 mutation, no direct v1/v2 → v3
    overwrite (the store still refuses it), no migration and no automatic
    conversion. `axiom compatibility backup` may keep a copy of the source
    under its existing contract; it is not authorization, tombstone, atomicity
    or recovery.
  - **PM-11b. Canonical v3 retirement.** On an active canonical v3 record,
    removing one of several Runtimes publishes a new validated v3 revision
    (dangling allowlists and Preferences are refused or removed with the
    Profile, as before); removing the final Runtime publishes the same
    existing tombstone with `priorConfigurationDigest = DigestV3` of the exact
    prior record (PM-9). No empty v3 configuration exists and no v2 tombstone
    path remains; preview, authority and recovery requirements are unchanged.
  - **PM-11c. Application upgrade separated from operational admission
    (B1).** The state inventory classifies the local Runtime/Profile object as:
    valid active v3 → `runtime_profile_configuration` (supported current
    configuration); valid tombstone → `runtime_profile_configuration`
    (recognized explicit retired state, not an active configuration); valid
    recognized format 1/2 → `runtime_profile_prestable_configuration`
    (historical pre-stable configuration, eligible for the application upgrade
    gate, NOT current configuration); malformed or ambiguous format 1/2 or 3 →
    `malformed`; a future format → `unsupported_newer`, never reinterpreted as
    v3; pending publication or recovery state → `recovery_state`. A valid
    pre-stable record therefore lets the existing upgrade journey proceed
    (`direct`/`compatible` when every other condition holds) with its bytes
    preserved, no migration, no silent retirement and no automatic v3
    initialization, and the compatibility report adds explicit reconfiguration
    guidance. Runtime policy, Workflow authoring, Stage admission, readiness
    and `runtime profile validate` read only canonical v3 and refuse a
    pre-stable record until it is retired and v3 is created. No direct
    Runtime/Profile upgrade from v1/v2 to v3 is promised; this narrow
    historical recognition is not backward-compatible Runtime/Profile execution
    support. Malformed, unsafe, future and recovery states keep their strict
    refusals.
  - **PM-11d. Historical corpus.** `internal/compatibility/testdata/stable-v1`
    stays byte-for-byte intact. `TestStableCorpusResolvesToDirectUpgrade` keeps
    iterating every snapshot with unchanged labels and its kind coverage; its
    expectation is split (owed by PM.2): every snapshot still upgrades
    `direct`/`compatible`, each frozen Runtime/Profile record is the pre-stable
    kind with reconfiguration guidance, and the frozen corpus covers the
    Runtime/Profile family through that historical kind while the canonical v3
    kind stays writer-tested until the first stable release freezes it.
    Execution, graph, Workflow and Provider compatibility contracts are
    unchanged, and no historical record is bound to current v3 Profile intent.
  - **Remaining narrow historical readers.** Format 1/2 decoding remains only
    in the store classification (to refuse and to bind retirement), the
    retirement digest, the compatibility inventory and immutable-artifact or
    corpus tests; the s9 acceptance runner keeps a script-local format-1
    capability model for graph planning while publishing canonical v3.

No new ADR: PM-11 applies the accepted PM-8 pre-stable decision and the existing
tombstone/authority/recovery contracts without new public authority, persisted
fields or formats, and changes no accepted invariant of PM-7.

### PM.4 Authorization epoch and production capability composition (BLD-001B.3c-3)

Implemented under the AXM-22 final delivery brief of 2026-10-14 (Rafael
Gomides), which directs PM-6 to an internal durable record protected by the
existing local filesystem boundary, with no public command, flag,
configuration field, tombstone format, Evidence field or digest algorithm.
This clause records the resulting internal contract; it records no other
maintainer approval. PM-1–PM-11 and the dated history above are unchanged.

- **PM-12. Authorization epoch persistence (PM-6).**
  - *Record.* `runtime-profiles/v1/authorization-epoch.json`, a private
    regular file beside the configuration object, exactly
    `{"schema":"axiom-runtime-authorization-epoch/v1","epoch":N}` + LF with
    `N` a positive `uint64`. Only these canonical bytes decode (strict
    members, ≤ 256 bytes); any other spelling, a symlink, a foreign owner or
    group/other permission is unsafe. It is read through the store's
    descriptor chain like the configuration object. It is not configuration,
    not covered by `DigestV3`, and never copied into a preview or Evidence;
    only its value enters the opaque snapshot token.
  - *Identity.* The Runtime/Profile snapshot token (the existing
    `--expected-revision` value, bound into every authoring preview digest)
    and the scoped Runtime recovery plan identity both include the committed
    epoch. Preview stays zero-write; confirmation requires the exact token,
    hence the exact epoch.
  - *Monotonic advance.* Under the store's exclusive locks, after the exact
    token and candidate checks pass and **before** the configuration can reach
    its commit point, `Publish` re-reads the epoch from the locked
    descriptor and publishes `epoch+1` (staged as
    `.axiom-stage-epoch-<random>`, fsync, rename, directory fsync, read back).
    `ApplyRecovery` on `runtime-profiles/v1` likewise advances the epoch after
    re-verifying the authorized plan and before restoring or finalizing any
    generation. A rejected, unauthorized or invalid request never advances
    it. The counter never wraps (`MaxUint64` refuses), decreases or resets;
    retirement, recreation and an interrupted first publication keep it.
  - *Consequence.* Whatever an authorized attempt's outcome — commit, F3
    pre-commit cleanup, F3–F5 rollback recovery restoring byte-identical
    configuration, or F6+ finalization — the previously reviewed token can
    never match again, and the next operation needs a fresh preview.
  - *Absence and recovery.* An absent record is epoch 0 only for states that
    can predate PM-6 (fresh, retired tombstone, pre-stable format 1/2); an
    active canonical v3 record without it is unsafe. A leftover epoch stage is
    recovery state that every reader refuses; scoped recovery classifies it
    (`restore_prior`, role `epoch_stage`, alone or beside a publication
    marker), advances the committed epoch and removes it. Corrupt or
    ambiguous epoch state fails closed and is never repaired.
  - *Inventory.* The valid record is classified with its family
    (`runtime_profile_configuration`); an invalid one is `malformed`; its
    stage is `recovery_state`.
  - *Threat-model limit.* Deleting the record of a retired or pre-stable state
    requires the principal that owns the private state root, which can already
    rewrite configuration; such a deletion resets nothing for active v3, which
    refuses.
- **Production capability composition.** `cmd/lingo` composes
  `capabilityproof.Service{Store, Source}` with the existing
  `runtimeadapter.BindingSource`: canonical v3 configuration read on every
  observation through the zero-write `RuntimeProfileStore.Inspect` (active v3
  only; fresh, retired, pre-stable, future, corrupt, unsafe and pending states
  blocked, no fallback), the established PATH lookup, `EmbeddedVersionObserver`
  (no other registry source), the pinned Axiom executable (`os.Executable`),
  and the AXM-21 `Selector` over the canonical `CapabilityStore`. Probe
  dependencies stay absent. The Stage policy's version observation now reads
  the same passive registry for the freshly observed executable identity
  instead of a capability observation. A present Source proves nothing. The
  Stage context, effort roots and consent follow PM-13.
- **Known limitations.** The embedded registry has zero entries (PM-5a), so
  every production observation blocks at the version step.
- **Delivery.** BLD-001B.3c-3 (epoch and composition) and BLD-001B.3c-4
  (synthetic end-to-end validation and reconciliation) are delivered together
  with PM-13 ([implementation record](issue-276-implementation.md#axm-22-bld-001b3c-3--authorization-epoch-production-composition-and-stage-effort-context)).

**PM-13. Stage effort-inspection context (accepted maintainer decision,
approved with constraints, 2026-10-14, Rafael Gomides).** Contract R: stable
host/repository effort roots with per-attempt child equivalence. The
canonical `EffortSourceDigest` and `ModeDigest` algorithms, the Evidence
schema and the public CLI are unchanged; no digest version, flag or probe is
added.

1. *Roots* (`runtimeadapter.StageEffortRoots`, resolved per operation, never
   persisted). `user` is the `HOME` of the effective child environment — the
   adapter's filtered environment of the attempt — exactly once, clean and
   absolute; a case alias, a duplicate, or a present `CODEX_HOME` /
   `CLAUDE_CONFIG_DIR` (which relocate the user source to a path the
   declaration does not name) refuses; no OS-account home is substituted.
   `managed` is the single directory of the Runtime's existing
   `DefaultManagedConfiguration` locations. `project` is the registered
   repository root of the Stage.
2. *Missing managed root.* An absence fact only when its nearest existing
   ancestor is anchor-grade (current user or root, no group/other write;
   sticky is no excuse), every higher ancestor is safe and no component is a
   symlink; the absent binding enters the canonical projection unchanged and
   is revalidated after inspection. User and project roots must exist.
   Nothing is created or repaired.
3. *Variant.* Only repository-write Targets and children are admitted.
   Read-only children (working root = private inbox) stay blocked
   (`effort_configuration_unproven`) until independently established
   Runtime-version evidence proves how that working root resolves project
   configuration.
4. *Child equivalence before each attempt* (`VerifyChildEquivalence`, called
   by the supervisor's `StageEffortInspector` on every dispatch-time check).
   The repository-root view and the child-worktree view (project anchor =
   the child's working directory) are each inspected canonically; their
   projections must be equal in every fact except the location binding of
   project-anchored source facts, and every project-anchored declared source
   must have the same presence and exact bytes. User and managed anchors —
   including the managed drop-in inventory — are the same roots in both views
   and are re-inspected. The returned digest is the repository-root view's,
   compared with the frozen tuple. Any difference, unreviewed working
   directory, executable or registry-version drift, configuration
   unavailability or inspection uncertainty fails closed; a selector is the
   existing `effort_override_conflict`.
5. *Stage context.* Operation-scoped: the operation's workspace and inbox
   namespaces and the inherited parent environment (the detached supervisor
   inherits it). ModeDigest and argv path operands carry only path roles
   (BI-1, BI-2b), so these identify every child of a shared Target.
6. *Consent.* `--authorize-local` on a `stage run`/`stage retry` call is
   explicit read-only inspection consent for that operation only, including
   its zero-write preview; without it inspection is unproven. At dispatch the
   committed Stage authority is the PE-05a authorized boundary. Consent grants
   no dispatch, external or capability authority, and exact preview
   confirmation is unchanged.
7. *Limitations.* On macOS `/etc` is a symlink, so the Codex managed root
   `/etc/codex` fails closed under the existing no-symlink boundary. Every
   production observation still blocks at the empty registry (PM-5a).

No new ADR: PM-13 binds caller-supplied roots that BI-2a D6 left to the
caller, relaxes only the missing-managed-root refusal under stricter ancestor
rules, and changes no accepted digest, Evidence or authority invariant (PM-7).
