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

## AXM-21 — Capability Evidence Selection

Child Work Item of AXM-8 (#276) under accepted I276-T16/T17; one branch
(`feat/axm-21-capability-evidence-selection`) and one Draft PR. Governance and
acceptance: [Plan record](plan-tasks-276-stage-agent-execution.md#maintainer-acceptance-and-delivery-governance--recorded-2026-10-13).

`capabilityproof.Selector` (`internal/capabilityproof/selection.go`) receives an
independently observed `Binding` and optional `ModelBinding`; an invalid or
partial `Binding` blocks before any enumeration, so persisted Evidence is never
a source of current identity. It re-verifies each candidate's content address
and admits a record only when `ValidEvidence` holds, it is not invalidated, it
is fresh under `MaxAge` (not future-dated), and both structures are exactly
equal (all seven Binding and seven ModelBinding fields; a Runtime-level request
never matches Profile-level Evidence or the reverse). Outcomes, all with blocker
`capability_unknown` and wrapping `ErrBlocked`:

| Candidates | Result |
|---|---|
| exactly one admissible | its content-addressed `EvidenceRef` |
| none | `ErrEvidenceMissing` |
| more than one | `ErrEvidenceAmbiguous`; no timestamp, name or digest precedence |
| any unreadable, unsafe, unverifiable or duplicate entry | generic `ErrBlocked`; no storage detail exposed |

Storage boundary: `local.CapabilityStore.Scan` enumerates `capability-evidence/v1`
read-only under the shared lock, bounded by the existing local directory limit.
It opens existing private roots only (an absent store yields no records and
creates nothing), refuses pending publication state (`ErrRecoveryRequired`), and
aborts with `ErrUnsafe` on any unexpected name, symlink, non-regular or
over-permissive entry, decoding failure or content-address mismatch, so a
partial view cannot yield a false unique selection. It writes, invalidates,
migrates and indexes nothing; Evidence schema, encoding, digests, Binding and
ModelBinding are unchanged.

Synthetic Evidence (Go 1.26.0 pinned toolchain, `GOTOOLCHAIN=local GOWORK=off`):

- `go test ./internal/capabilityproof -run '^TestEvidenceSelection' -count=1` — exit 0
  (11 tests, 35 including subtests): unique Runtime-/Profile-level, per variant,
  order independence, every Binding/ModelBinding field drift, none, two matches,
  invalidated/expired/future, content-address, corruption, unsupported schema,
  duplicate and unreadable candidates, missing current identity.
- `go test ./internal/local -run '^TestCapabilityEvidenceScan' -count=1` — exit 0
  (4 tests, 12 including subtests): real store selection with byte/mode/mtime
  snapshots proving no mutation, absent store not created, ambiguity, explicit
  invalidation preserved, corrupted/misaddressed/unsupported/unknown/symlink/
  directory/over-permissive entries and pending publication fail closed.
- Regression: `go test ./internal/capabilityproof -count=1` and
  `go test ./internal/local -run '^TestCapability' -count=1` — exit 0;
  `go vet` and `go build` for both packages — exit 0.

These are synthetic tests only: no Runtime executable, probe, inference or
Provider effect was run, and they are not operational Evidence.

Remaining: production BindingSource (current identity observation), wiring the
selector into it and `capabilityproof.Service`, T34 composition, T44/B6 CLI,
retry-reset authority/fencing, and a decision on multiple matching Evidence.
AXM-8 remains In Progress.

## AXM-22 BLD-001A — Process mode identity and approved binding contracts

Child Work Item AXM-22 (Production BindingSource) of AXM-8 (#276) under
accepted I276-T16/T17; branch `feat/axm-22-production-binding-source`, base
`ca105cc`, one Draft PR. BLD-001A is the first bounded block; BLD-001B follows.

Approved decisions (Rafael Gomides, 2026-10-14), recorded in the
[AXM-22 binding identity amendment](amendment-276-axm22-binding-identity.md):

- **BI-1** — `Process bool` in `capabilityproof.Target` is part of the
  invocation-mode identity, reusing `runtimeadapter.ModeDigest`.
- **BI-2** — a narrow effort-source inspector, independent of capability proof,
  using a versioned configured `EffortDeclaration` validated against current
  executable identity and the canonical `EffortSourceDigest`; no vendor-version
  inference. Approved, **not implemented**.

Process propagation (BI-1, partial):

| Consumer | Change |
|---|---|
| `capabilityproof.Target` | `Process` field added; `Binding`, `ModelBinding`, `Current`, `BindingSource` signatures unchanged |
| `Service.Check` (preview and every dispatch) | requested `process` contradicting `Target.Process` blocks `capability_stale`; the `process` capability is still required from Evidence |
| `Observer` | Targets mixing Process modes block, as mixed variants already did |
| `stageexecution` dispatch preview | Target Process from the child's `process` effect, the source the supervisor applies to `StageInvocationRequest.Process` |
| `cmd/lingo` stage policy observer | Target Process from the requested `process` capability, mirroring its variant derivation |
| Probe preparation / Evidence selection | receive the Target / current Binding unchanged; mode distinction comes from the canonical ModeDigest in `Binding` |

Pending consumer: no production `BindingSource` exists, so no code computes
`Binding.ModeDigest` from a Target yet; production reads remain blocked.

Synthetic Evidence (Go 1.26.0, `GOTOOLCHAIN=local GOWORK=off GOFLAGS=`), all exit 0:

- `go test ./internal/capabilityproof -run '^(TestCheckRejectsContradictoryProcessMode|TestObserverRejectsMixedProcessModes|TestClaudeProcessModesHaveDistinctCanonicalIdentity|TestEvidenceSelectionNeverCrossesProcessModes|TestRevalidationRejectsProcessModeDrift)$' -count=1`
  — Claude Process modes have distinct canonical ModeDigests (Codex 0.159.1
  identical by adapter construction); identical requests give identical
  digests; Evidence for either mode never satisfies the other; dispatch-time
  drift is `capability_stale`; contradictory Process blocks; unproven `process`
  still blocks; no Evidence mutation.
- `go test ./internal/stageexecution -run '^TestDispatchPreviewBindsProcessMode$' -count=1`
  — a `process` effect selects the Process=true Target; Process=false
  Evidence and unproven `process` are refused.
- Regression: `go test ./internal/capabilityproof -count=1` (incl. AXM-21
  zero/multiple selection and cancellation), `go test ./internal/runtimeadapter -run '^TestModeDigest' -count=1`,
  `go test ./internal/stageexecution -run '^TestDispatchPreview' -count=1`,
  `go test ./cmd/lingo -run '^TestProductionCapabilityComposition' -count=1`;
  `go build`/`go vet` for the four packages; `gofmt -l` on modified files.

Synthetic only: no Runtime executable, capability probe, inference, stage
dispatch or Provider effect ran; this is not operational capability proof.

Status: Process contract partially implemented; effort inspector not yet
implemented; production BindingSource not yet implemented; T34 remains pending;
T16/T17 are not complete. Remaining (BLD-001B): effort inspector with
canonical-digest equivalence tests, configured `EffortDeclaration` integration
(blocked on a schema decision if no existing configured carrier exists — see
BI-2), production BindingSource and AXM-21 selector composition.
