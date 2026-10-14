# Specification 007 amendment — AXM-22 Process mode and effort-source identity

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
v3 authoring/publication is **not implemented** (BLD-001B.3). Clause **BI-2b**
(path-operand reliability) is implemented and clause **BI-3** (explicit v2→v3
migration and minimal authoring contract, approved 2026-10-14) is specified,
not implemented, by BLD-001B.3a.

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
    package tests; no production path can publish format 3.

Delivery boundaries:

- **BLD-001B.2a (implemented):** narrow effort inspector, shared canonical
  `EffortSourceDigest` algorithm with digest-equivalence tests, anchor-root
  resolution.
- **BLD-001B.2b (implemented):** production BindingSource and AXM-21 Evidence
  selection.
- **BLD-001B.3 (pending):** authorized v3 writing/publication, supported local
  authoring integration, synthetic end-to-end configuration validation.
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

BLD-001B.3a records this contract only; nothing below is implemented yet.

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
