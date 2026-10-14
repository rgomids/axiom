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

## AXM-22 BLD-001B.1 — Configuration v3 EffortDeclaration codec

Decision: Rafael Gomides approved D1–D8 on 2026-10-14 (AXM-22 BLD-001B.1
brief), recorded in [amendment BI-2a](amendment-276-axm22-binding-identity.md#bi-2a--configuration-v3-carrier-for-effortdeclaration-bld-001b1).
This resolves the BLD-001A schema question; it is an intermediate delivery.

Files:

| File | Change |
|---|---|
| `internal/runtimeprofile/v3.go` | `ConfigurationV3`, `RuntimeV3`, `EffortDeclaration`, `EffortSourceDeclaration`; `ValidateV3`, `DecodeVersioned`, `ApplicableEffortDeclaration`, `ValidEffortSourcePath`, `ProofReference()`; unexported canonical test-fixture encoder |
| `internal/runtimeprofile/runtimeprofile.go` | `Validate`/`Decode` refuse format ≥ 3 with `ErrUnsupportedConfigurationFormat`; v1/v2 paths unchanged |
| `internal/local/runtime_profile_authoring.go` | snapshot reader and `Publish` surface the explicit refusal; no mutation |
| `internal/runtimeapplication/authoring.go` | actionable `Next` inside the existing `configuration_unavailable` category |
| `internal/runtimeprofile/testdata/configuration-v3.json` | canonical synthetic v3 fixture |
| tests | `runtimeprofile/v3_test.go`, `local/runtime_profile_v3_refusal_test.go`, `runtimeapplication/authoring_v3_refusal_test.go`, `runtimeadapter/effort_sources_v3_path_test.go` |

Boundaries held: no production v3 writer (encoder unexported, test use
only), no CLI flag, no v2-to-v3 migration, v1-to-v2 migration untouched, no
Binding/ModelBinding/Evidence/digest/`runtimeadapter` production change, no
new public error category.

Synthetic Evidence (Go 1.26.0, `GOTOOLCHAIN=local GOWORK=off GOFLAGS=`), all exit 0:

- `go test ./internal/runtimeprofile -run 'TestV3|TestV2ClosedCodecAndHistoricalBytes' -v`
  — golden format-2 bytes (`08a005ea…`) and `Digest` (`8a7f9a6e…`) recorded
  at 825b88c unchanged; v1/v2 dispatch without synthetic declarations and
  `effort_configuration_unproven`; canonical v3 round trip, permutation
  determinism and golden fixture; Profile effort/digest unchanged; 1:1
  `declarationId`/ProofReference; applicability exact on Runtime, executable,
  version and mode; vendor version creates nothing; declaration never
  satisfies `ProfileProof`; 50+ negative declaration/path/wire cases
  (missing/duplicate id, version, mode, precedence, anchors, absolute/UNC/
  drive/traversal/URI/control/oversized paths, keys, counts, unknown/
  duplicate/case-alias fields); `Decode`/`Validate`/`Encode`/`Digest` refuse v3.
- `go test ./internal/local -run 'TestRuntimeProfileV2OnlyPathsRefuseV3|TestRuntimeProfileInventoryInvalidSupportedV2IsMalformed' -v`
  — `Publish` (fresh and over v2), snapshot reader, `Load`, legacy `Save`
  refuse v3; v2 and v3 bytes unchanged; inventory `unsupported_newer`.
- `go test ./internal/runtimeapplication -run 'TestAuthoringRefusesV3WithoutMigrationOrWrite' -v`
  — create/list/migrate refused, zero writes.
- `go test ./internal/runtimeadapter -run 'TestEffortSources' -v` — includes a
  lexically valid configured path whose component is a symlink escaping the
  anchor: refused by the existing trusted resolution (no skip; setup failure fails).
- Regression: `go test ./internal/runtimeprofile`,
  `go test ./internal/local -run 'RuntimeProfile|RuntimeAuthoring|Runtime.*Recovery'`,
  `go test ./internal/runtimeapplication -run 'Authoring'`; `go build` (four
  packages and `./cmd/...`), `go vet` (four packages), `gofmt -l`.

Distinctions: schema accepted ≠ inspector implemented; schema accepted ≠
operational authoring; declaration present ≠ capability proven. No Runtime
executable, probe, inference, stage dispatch or Provider effect ran.

Status: T16/T17 not complete. Remaining: BLD-001B.2 (effort inspector,
canonical `EffortSourceDigest` equivalence, anchor-root resolution,
production BindingSource, AXM-21 selector composition); BLD-001B.3
(authorized v3 authoring/publication); T34 and remaining AXM-8 work.

## AXM-22 BLD-001B.2a — Canonical effort-source inspector

Scope: the narrow internal effort inspector of
[amendment BI-2](amendment-276-axm22-binding-identity.md#bi-2--effort-source-identity-inspection-independent-of-capability-proof),
fed by the BI-2a v3 declaration. Production BindingSource is BLD-001B.2b.

Files:

| File | Change |
|---|---|
| `internal/runtimeadapter/effort_sources.go` | PE-05a body extracted unchanged into the shared unexported `effortSourceProjection`; `CheckEffortSources` keeps its consent and `ValidateResolvedProfile` gate and hashes that projection with the existing `effortHash` (SHA-256, lowercase hex) |
| `internal/runtimeadapter/effort_inspector.go` | internal `InspectEffortSources(ctx, EffortSourceRequest)` with `EffortSourceRoots`; `trustedEffortRoot` |
| `internal/runtimeadapter/effort_identity_posix.go` | `effortAncestorSafe` (the ADR-0005 container rule of `internal/local`) and the stricter `effortRootSafe` |
| `internal/runtimeadapter/effort_inspector_test.go` | equivalence, refusal, independence, read-only and fail-closed tests |

Behaviour:

- **One algorithm.** The inspector maps the configured declaration onto the
  existing `runtimeadapter.EffortDeclaration` (`ProofReference` =
  `declarationId`, D7) and calls the same projection; no field is added to the
  digest.
- **Applicability.** Selection is `runtimeprofile.ApplicableEffortDeclaration`
  over a `DecodeVersioned` result, matching the observed Runtime, version,
  executable digest and mode exactly; it revalidates the full v3 rules
  (declaration version, uniqueness, scopes, anchors, precedence). The core
  then rechecks the executable identity of `argv[0]`, the argv mode/model and
  the Runtime of the invocation. Model and effort intent come from the
  configured Model Profile (ADR-0024), never from a proof.
- **Anchors.** `user`/`project`/`managed` resolve only to caller-supplied
  current roots. Each root must be absolute, clean, an existing non-symlinked
  directory bound through the PE-05a path boundary, owned by the current user
  or root without group/other write, under ancestors that are safe from
  replacement; its binding is compared again after inspection. Sources are
  then read through the unchanged PE-05a resolver (symlink, special-file,
  drop-in and drift refusals). Nothing is created, repaired or mutated.
- **Fail closed.** Every refusal is `effort_configuration_unproven`, or the
  existing `effort_override_conflict` for a present override; no new error
  category. A missing anchor root refuses; an absent source file under a
  trusted root stays a PE-05a absence fact.
- **No proof.** The request carries no `ProfileProof`, `ResolvedProfile`,
  Observation, capability status or Evidence; the result is a digest only.
  Format-1/2 records and v3 Runtimes without a declaration stay unproven; a
  vendor version selects nothing.

Synthetic Evidence (Go 1.26.0, `GOTOOLCHAIN=local GOWORK=off GOFLAGS=`):

- `go test ./internal/runtimeadapter -run 'TestInspectEffortSources|TestTrustedEffortRoot' -v`
  — exit 0, 8 tests. For Codex and Claude: canonical projection bytes and
  digest equal `CheckEffortSources` for baseline, absent source, different
  non-selector bytes, explicit Profile effort and another `declarationId`;
  content drift, `declarationId` and drop-in membership each change the
  digest; selector/environment/argv overrides and unsupported grammar refuse
  identically; 36 (Codex) / 39 (Claude) fail-closed mutations each start
  from an accepted baseline.
- An uncommitted cross-check against the `dd08cbf` `CheckEffortSources` body
  matched the refactored function and the inspector byte for byte in 10
  Codex/Claude cases (7 digests, 3 identical refusals).
- `go test ./internal/runtimeadapter` and
  `go test ./internal/runtimeprofile -run 'TestV3|TestV2'` — exit 0;
  `go build ./...`, `go vet` (runtimeadapter, runtimeprofile), `gofmt -l`,
  `check-go-quality.sh format|tidy`, `git diff --check` — exit 0.
- Not run: `check-go-quality.sh staticcheck` (staticcheck v0.8.1 not
  installable in this environment).

No `t.Skip` in added tests. No Runtime executable, probe, inference, stage
dispatch or Provider effect ran.

Status: T16/T17 not complete. Remaining: BLD-001B.2b (production
BindingSource composing Binding/ModelBinding, Process-aware ModeDigest, this
inspector, the AXM-21 Selector and `capabilityproof.Service`); BLD-001B.3
(authorized v3 authoring/publication); T34/T44 and remaining AXM-8 work.

## AXM-22 BLD-001B.2b — Production BindingSource

Scope: the production `capabilityproof.BindingSource` of
[amendment BI-1/BI-2/BI-2a](amendment-276-axm22-binding-identity.md),
consuming the BLD-001B.2a inspector and the AXM-21 Selector. The interface,
`Binding`, `ModelBinding`, Evidence, `ModeDigest` and `EffortSourceDigest`
are unchanged.

Files:

| File | Change |
|---|---|
| `internal/runtimeadapter/binding_source.go` | `BindingSource` (implements `capabilityproof.BindingSource`) and `StageContext`; lives in `runtimeadapter` because that package already imports `capabilityproof` |
| `internal/runtimeadapter/frozen_stage.go` | the explicit-effort argv/environment encoding extracted unchanged into the unexported `withStageEffort`, shared by `BuildFrozenStageInvocation` and the BindingSource |
| `internal/runtimeadapter/binding_source_test.go` | synthetic identity, drift, selection and Service tests |
| `internal/runtimeadapter/effort_sources_test.go` | both historical `t.Skip("native symlink unavailable")` calls replaced by `t.Fatalf` with the setup error |

Behaviour — each `Current` observes afresh, in order:

1. **Configuration** through an injected version-aware read
   (`runtimeprofile.VersionedConfiguration`, i.e. `DecodeVersioned`). Only a
   valid format 3 proceeds; format 1/2 is `effort_configuration_unproven`
   (D3). Legacy v2 `Load`/`Publish`/`Save` are untouched and still refuse v3.
2. **Runtime/Profile**: an enabled Runtime whose ID and adapter equal the
   Target Runtime, allowlisting the Target Profile, bound to the same Runtime;
   otherwise `ErrBlocked`. `ProfileDigest` is the existing
   `runtimeprofile.ProfileDigest`; revision is the configuration revision.
3. **Executable** via injected `LookPath` and the canonical
   `ExecutableIdentity`; **version** via an injected observer keyed by that
   identity (the existing `runtimeVersionObserver` signature). No vendor binary
   is executed by the BindingSource.
4. **Pinned Axiom** identity via `ExecutableIdentity(AxiomPath)`.
5. **Mode**: one effective `StageInvocationRequest` from the Target (Variant →
   `RepositoryWrite`, `Process`), the observed executable/version, the Profile
   model and a caller-supplied `StageContext` (workspace, inbox, parent
   environment; required, never defaulted). `ModeDigest` and `AdapterVersion`
   are the canonical functions: Claude's Process modes differ, Codex's are
   identical by construction.
6. **Effort**: the same request through `BuildStageInvocation` plus
   `withStageEffort`, then `InspectEffortSources` with the observed
   Runtime/version/executable, `noninteractive`, caller-supplied trusted roots
   and the caller's consent. Its refusals propagate unchanged
   (`effort_configuration_unproven`, `effort_override_conflict`).
7. Executable and Axiom identities are re-read; replacement during
   observation is `ErrBlocked`.
8. **Selection**: AXM-21 `Selector.Select` with the complete Binding and
   ModelBinding; `Current` carries its unique reference and
   `ProfileActive=true`. Zero, several or unverifiable records fail closed as
   `ErrEvidenceMissing`, `ErrEvidenceAmbiguous`, `ErrBlocked`; no precedence,
   invalidation or write.

No step reads Evidence, `ProfileProof` or a `ResolvedProfile`; Selection runs
only after the identity is complete. `context.Canceled` and
`context.DeadlineExceeded` keep their identity at every stage.
`ConfiguredSource` and `Service` consume it unchanged; with the same
configuration `ConfiguredSource` recomputes identical Profile fields.

Observed limitation (unchanged PE-05a behaviour): the projection refuses any
stage argument containing `effort`, so a workspace or inbox path with that
substring fails closed as `effort_override_conflict`. Test fixtures therefore
place stage paths under a neutral temporary root.

Composition boundary: `cmd/lingo` still composes `capabilityproof.Service`
without a Source. The local snapshot reader refuses format 3 (D8), so wiring
this BindingSource into production commands depends on BLD-001B.3.

Synthetic Evidence (Go 1.26.0, `GOTOOLCHAIN=local GOWORK=off GOFLAGS=`):

- `go test ./internal/runtimeadapter -run TestBindingSource -v` — exit 0;
  7 tests, 38 subtests. Codex and Claude × read-only/repository-write ×
  Process false/true return the complete Binding and ModelBinding equal to
  independent oracles (`ModeDigest`, `AdapterVersion`, `ProfileDigest`, and
  PE-05a `CheckEffortSources` over `BuildFrozenStageInvocation`) and the unique
  EvidenceRef; explicit and runtime-default effort. 34 fail-closed cases:
  missing Runtime/Profile/ProfileID, unknown variant, disabled Runtime,
  executable unavailable/drift, invalid executable applicability, unsupported,
  unobserved and declaration-drifted version, Process and variant mismatch,
  Axiom drift/unavailable, Profile digest/effort drift, revision drift,
  effort-source drift, override conflict, missing declaration, format 2,
  unreadable configuration, no consent, group-writable and symlinked roots,
  missing stage context, adapter and ModeDigest mismatch, missing, multiple
  and corrupt Evidence, cancellation and deadline during selection. Separate
  tests prove no Evidence-derived identity, that a `declarationId` spelled as
  an EvidenceRef proves nothing (`Service.Check` → `capability_unknown`), and
  `ConfiguredSource`/`Service.Check` compatibility including Process
  contradiction (`capability_stale`) and a disabled Runtime.
- `go test ./internal/runtimeadapter -run 'TestBindingSource|TestEffortSources|TestInspectEffortSources|TestTrustedEffortRoot|TestAdapterVersionCanonical|TestModeDigest|TestFrozenStage'`
  and `go test ./internal/capabilityproof -run 'Process|Selection|TestCheck|TestShow|TestInvalidat|TestRequired|TestEach|TestObserver|TestConfiguredSource'`
  — exit 0. Both symlink refusal tests
  (`TestEffortSourcesMalformedAndSymlinkRefusal`,
  `TestEffortSourcesDropInUnknownAndSymlinkRefusal/symlink`) pass on Linux
  amd64 with no skip.
- `go build` (runtimeadapter, capabilityproof, cmd/lingo), `go vet`
  (runtimeadapter, capabilityproof), `check-go-quality.sh format` and
  `staticcheck`, `git diff --check` — exit 0. `check-architecture.go` covers
  only `internal/project`/`internal/projectapp`, which this block does not
  touch.

No `t.Skip` in added or modified tests. No Runtime executable, probe,
inference, stage dispatch, T40 or Provider effect ran. Nothing here is
capability proof.

Status: T16/T17 not complete. Remaining: BLD-001B.3 (authorized v3
authoring/publication and composition wiring); T34/T44; retry-reset
authority/fencing; remaining AXM-8 work.
