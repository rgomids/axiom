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
BLD-001B.1; the effort inspector and production BindingSource are **not
implemented** (BLD-001B.2), nor is v3 publication (BLD-001B.3).

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

Pending consumer: the production `BindingSource` must compute
`Binding.ModeDigest` from the Target's Variant **and** Process through
`runtimeadapter.ModeDigest`. Until it exists, production capability reads stay
blocked (BLD-001B).

## BI-2 — Effort-source identity inspection independent of capability proof

Approved, not yet implemented. A narrow internal effort inspector produces the
canonical `EffortSourceDigest` for the production BindingSource:

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

- **BLD-001B.2 (pending):** narrow effort inspector, shared canonical
  `EffortSourceDigest` algorithm with digest-equivalence tests, anchor-root
  resolution, production BindingSource and AXM-21 Evidence selection.
- **BLD-001B.3 (pending):** authorized v3 writing/publication, supported local
  authoring integration, synthetic end-to-end configuration validation.
- New public CLI flags or v2-to-v3 migration require a separate explicit
  maintainer decision before implementation.
