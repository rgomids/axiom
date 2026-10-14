# Specification 007 amendment — AXM-22 Process mode and effort-source identity

## Status

**Approved implementation direction, 2026-10-14**, by explicit maintainer
decision (Rafael Gomides), recorded in the AXM-22 BLD-001A execution brief.
Owner: AXM-22 — Production BindingSource, child of AXM-8
([#276](https://github.com/rgomids/axiom/issues/276)), under accepted
I276-T16/T17 ([Plan Part II](plan-tasks-276-stage-agent-execution.md#i276-t16--capability-probe-preview-apply-and-execution)).

Clause **BI-1** (Process mode identity) is partially implemented by BLD-001A.
Clause **BI-2** (effort-source identity inspection) is approved but **not
implemented**; it belongs to BLD-001B.

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

**Open schema question (blocks BLD-001B implementation, not this amendment).**
Local configuration format 2 (`runtimeprofile.Configuration`, `Runtime`,
`ModelProfile`) has no field that carries an `EffortDeclaration`. If BLD-001B
cannot obtain the declaration from an existing configured carrier, the
persisted location and field names require an explicit maintainer schema
decision before implementation; this amendment invents none.
