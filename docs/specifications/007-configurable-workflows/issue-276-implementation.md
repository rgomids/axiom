# #276 implementation record

## Historical gap report and approved decision

I276-T16/T17 — Missing production capability BindingSource

Status: Implementation gap discovered during I276-T34 production wiring.

Problem: capabilityproof.Service requires a production BindingSource, but no complete implementation is currently wired. ConfiguredSource is only a decorator, and the available observers do not supply the complete current binding.

Required correction:

- Implement deterministic, local observation of executable, adapter, Axiom binary, invocation mode, variant, Profile and effort-source identities.
- Reuse canonical Runtime/Profile resolution and existing identity observers.
- Ensure full binding consistency with evidence creation and dispatch-time revalidation.
- Fail closed on missing, stale or ambiguous identities.
- Never use persisted Evidence as the source of current identity.
- Never promote observations alone to proven.
- Add positive and negative tests for binding equality, drift, Profile changes and missing observations.

Ownership: T16 production composition + T17 observer/revalidation contract.

Blocked consumer: T34 production wiring.

Non-goals: Real probes, inference, T40, Provider effects, new authority or changes to historical Graph formats.

Completion Evidence: Focused synthetic tests for BindingSource, correct integration with capabilityproof.Service, and no regression of T17 fail-closed behavior.

Approved technical decision — Canonical AdapterVersion and ModeDigest

Record:
- Decision approved through Maestro on behalf of Rafael.
- Option 1 selected.
- AdapterVersion is canonical and adapter-owned.
- ModeDigest uses one shared, deterministic, versioned encoding.
- Probe, Evidence and dispatch must use identical identity computation.
- Binding identity drift invalidates operational use of previously matching Evidence.
- Historical Evidence remains immutable.
- No automatic Evidence migration or digest rewriting.
- Incompatible Evidence evaluates as unknown/stale, subject to existing evaluation contracts.
- Fresh proof requires separate authority.
- No new ADR or Specification amendment.
- Remaining implementation: production BindingSource and T34 composition.

## BLD-005 — canonical identity primitives

Implemented in `internal/runtimeadapter/identity.go`. `AdapterVersion(runtimeID)`
returns `codex-adapter-v1` or `claude-adapter-v1`, with no caller override;
unsupported IDs fail closed. Bump the affected adapter version whenever
adapter-controlled invocation or confinement semantics materially change,
including fixed safety flags, tool permissions, path roles or environment policy.
Vendor/executable versions, Profile, model and effort do not determine this version.

`ModeDigest(StageInvocationRequest)` validates through the pure
`BuildStageInvocation` builder, then hashes a closed semantic projection with
SHA-256 (64 lowercase hexadecimal characters). It does not run an executable.
Encoding `axiom-runtime-mode-v1` is compact Go JSON with fields in this order:
`encoding`, `runtime`, `adapter`, `variant`, `workingRoot`, `environment`, `options`.
All fields are strings except the ordered string array `options`; no omitted fields
or trailing newline. An encoding change requires a new encoding version.

The projection covers the canonical adapter identity, read-only/repository-write
variant, working root role, exact stage environment allowlist policy
(`stage-allowlist-v1-exact`) and adapter-emitted mode options in invocation order:

- Codex: `exec`, `--ignore-user-config`, `--ephemeral`, stdin `-`,
  `--sandbox workspace-write`, `-C` root role, and repository-write `--add-dir inbox`.
- Claude: `--print`, `--input-format text`, `--permission-prompts none`,
  `--permission-mode dontAsk`, the exact allowed tools and `--add-dir inbox`.
  Read-only omits Edit; Process replaces restricted `Bash(pinned-axiom:*)` with Bash.

Absolute paths become `workspace`, `inbox` or `pinned-axiom` roles, preserving
their security distinctions and validated disjoint workspace/inbox roots. The
pinned Axiom binary is identified separately by Binding.AxiomDigest. Codex Process
does not change its effective argv, so both Process values share a digest.
The environment policy is bound, but raw values, correlation IDs, executable
paths, model/Profile selection and operational output bounds are not serialized.
Model/effort and executable/vendor identities remain in the existing separate
Binding/ModelBinding fields. Unknown vendor versions, caller argv/environment/
credential overrides and invalid paths fail closed. Future unknown emitted
options also fail closed until explicitly covered by the encoding contract.

Golden vectors are in `internal/runtimeadapter/identity_test.go`,
`TestModeDigestGolden`: eight Runtime/variant/Process cases (six distinct digests),
derived from the initial encoding contract, never historical Evidence. Synthetic
tests also cover determinism, equivalent requests across operational locations,
safety policy differences, override rejection, Binding validation and exclusion
of credentials and temporary paths from the canonical representation.

Reuse boundary: probe preparation, Evidence binding creation and dispatch-time
revalidation must call these same primitives using current effective mode inputs.
Historical configured invocations are outside this closed stage-mode contract.
No consumers are wired by BLD-005; production BindingSource and T34 composition
remain incomplete. Evidence encoding, historical bytes/digests, exact binding
equality, capability states/freshness, Graph1/2/3 and authority contracts are unchanged.
