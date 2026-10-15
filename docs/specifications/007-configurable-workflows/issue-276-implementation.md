# #276 implementation record

Latest continuation: [operational checkpoint, Task inventory and T43/T41 Evidence](#continue-2026-10-15--operational-checkpoint-for-pr-320). Earlier PR implementation: [T13/T14/T16 production integration](#implement-2026-10-15--prepared-production-probe-and-canary). Current scope: [maintainer addendum](#implementation-scope-addendum--2026-10-15). Baseline assessment: [2026-10-15 reconciliation and all 44 Tasks](#reconcile-2026-10-15--main-baseline-and-task-matrix). Earlier sections are dated implementation history; later records and the reconciliation qualify their pending/not-started statements.

## Implementation scope addendum — 2026-10-15

The [maintainer scope addendum](plan-tasks-276-stage-agent-execution.md#implementation-scope-addendum--2026-10-15)
qualifies the earlier dated T40/T43 assessments: all real tests are deferred
(**NOT_RUN**); T40 does not block technical delivery, while its own Done and
AC-6 remain unproved; active T43/T13 platform Evidence is Linux/macOS only.
Windows requirements in the historical catalog are excluded prospectively.
The maintainer owns the deferred-proof risk and final operational acceptance.
No Task or AXM-8 state is promoted by these decisions.


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

## AXM-22 BLD-001B.3a — Migration contract and effort path reliability

Authority: Rafael Gomides approved A1–A5 on 2026-10-14 at 10:27
(America/Sao_Paulo): explicit v2→v3 migration (never automatic), minimal public
`EffortDeclaration` authoring reusing existing commands, preview/confirmation/
atomic publication/recovery, refusal of missing/ambiguous/inapplicable
declarations including executable-identity mismatch, and correction of the
BLD-001B.2b path finding. It supersedes BI-2a D5 only for that write path.

Scope: contract and fix only. Recorded in
[amendment BI-2b and BI-3](amendment-276-axm22-binding-identity.md#bi-3--explicit-v2v3-migration-and-minimal-effortdeclaration-authoring-bld-001b3).

Files:

| File | Change |
|---|---|
| `amendment-276-axm22-binding-identity.md` | BI-2b (path-operand reliability) and BI-3 (migration, authoring, preview/confirmation, revalidation, atomic publication, recovery, fail-closed conditions, 3a/3b/3c division) |
| `internal/runtimeadapter/effort_sources.go` | unexported `effortPathOperand`; the PE-05a argv scan skips only the clean absolute operand of the adapter's own path options |
| `internal/runtimeadapter/effort_sources_path_test.go` | path regression, preserved-override and operand-predicate tests |
| `internal/runtimeadapter/binding_source_test.go` | BindingSource fixture stage root now spells `effort` (was a workaround for the finding) |

**Migration/authoring contract (BI-3, not implemented).** Reuse
`axiom runtime binding migrate|edit` and the existing `--expected-revision`,
`--preview-digest`, `--authorize-local` authority. Add only
`--target-format 3` (migrate; required on `active-v2`, refused on `active-v1`)
and `--effort-declaration <json-file>` (one strict BI-2a declaration for
`--runtime`). Preview is zero-write and binds the snapshot token, source digest,
candidate digest, declaration and the independently observed executable digest
and version. Apply re-inspects, re-reads, re-observes and recomputes; any change
is `stale_preview`. Publication goes through the existing
`RuntimeProfileStore.Publish` protocol, extended in 3b to accept format 3 only
as `active-v2 → v3` or `active-v3 → v3`, never as a downgrade; F6 is
`recovery_required` and recovery is the existing scoped `recover` with fresh
authority and no automatic retry. Runtimes, Profiles, Profile digests and
Preferences are carried unchanged; v2 history is never rewritten.

**Path finding root cause.** `effortSourceProjection` scanned every argv token
after `<exe> <mode> --model <model>` with `strings.Contains(lower, "effort")`,
including the operands of `BuildStageInvocation`'s own options: Codex `-C` and
`--add-dir`, Claude `--add-dir` and the `Bash(<axiomPath>:*)` element of
`--allowedTools`. Workspace, inbox and environment scanning, source-root
inspection and override detection were not the cause.

**Correction.** `effortPathOperand(runtime, option, value)` exempts only those
operands, and only when they are `stagePath` (clean absolute, no control
characters); for `--allowedTools` only `Bash(<stagePath>:*)` elements, never a
value starting with `-`. Everything else keeps the unchanged predicate. Argv
tokens add no projection fact, so no field, encoding, hash or ProofReference
changes.

Synthetic Evidence (Go 1.26.0 linux/amd64, `GOTOOLCHAIN=local GOWORK=off GOFLAGS=`):

- Reproduction: with the exemption disabled (loop equal to `ef4f3b6`),
  `go test ./internal/runtimeadapter -run TestEffortSourcesPathSpellingIsNotOverride`
  FAILED: `codex/explicit … nested-mixed-case: err effort_override_conflict`.
  Restored before every recorded run.
- `go test ./internal/runtimeadapter -run 'TestEffortSourcesPath|TestEffortSourcesOverridesStillRefusedUnderEffortPaths' -v`
  — exit 0, 3 tests, 18 subtests. Codex and Claude × explicit/runtime-default
  effort × read-only/repository-write × Process false/true: six path sets
  (component, workspace named `effort`, nested mixed case, selector-spelled
  directories, Axiom path, and the real temporary root carrying the test name)
  give the same digest as the neutral baseline, each through `compareWithPE05a`
  (inspector and `CheckEffortSources` byte-identical projection and digest):
  96 comparisons. Still refused under an `effort` stage path: conflicting
  `--config`/`-c`, `--profile`, non-path and relative/unclean path operands, a
  path option swallowing `--config`, Claude `--effort`, an `effort` tool
  element, `--allowedTools -p`, inherited and declared environment conflicts,
  configured selectors (`effort_override_conflict`), and symlinked sources,
  unsupported TOML and ambiguous JSON (`effort_configuration_unproven`).
- `go test ./internal/runtimeadapter -run 'TestBindingSource|TestEffortSources|TestInspectEffortSources|TestTrustedEffortRoot|TestAdapterVersionCanonical|TestModeDigest|TestFrozenStage' -v`
  — exit 0, 34 tests, including the existing PE-05a equivalence
  (`TestInspectEffortSourcesMatchesPE05aDigest`, `…RefusalsMatchPE05a`), both
  symlink refusal tests and the BindingSource suite now on an `effort` stage
  root.
- `go test ./internal/runtimeprofile -run 'TestV3|TestV2'` (11 tests) and
  `go test ./internal/capabilityproof -run 'Process|Selection|TestCheck|TestShow|TestInvalidat|TestRequired|TestEach|TestObserver|TestConfiguredSource'`
  — exit 0.
- `gofmt -l`, `go build` (runtimeadapter, capabilityproof, runtimeprofile,
  `cmd/lingo`), `go vet` (runtimeadapter, runtimeprofile),
  `git diff --check` — exit 0.

No `t.Skip`. Hosts: executed on Linux amd64 only; macOS arm64 and Linux arm64
are unverified in this block (tests are `unix`-tagged and need native
symlinks only).

Explicitly not done: v3 production migration **not implemented**; v3
production publication **not implemented**; `cmd/lingo` production
composition **not completed**; no operational capability proof, Runtime
executable, probe, inference, stage dispatch or Provider effect.

Status: AXM-22 not complete; T16/T17 not complete. Remaining: BLD-001B.3b
(v2→v3 migration, minimal authoring flags, format-3 local inspection and
atomic publication/recovery) and BLD-001B.3c (BindingSource composition in
`cmd/lingo`, synthetic end-to-end validation, final AXM-22 reconciliation);
T34/T44 and remaining AXM-8 work.

## AXM-22 BLD-001B.3b — Explicit v2→v3 migration, authoring and atomic publication

Authority: A1–A4 (Rafael Gomides, 2026-10-14 10:27 America/Sao_Paulo) as
recorded in
[BI-3](amendment-276-axm22-binding-identity.md#bi-3--explicit-v2v3-migration-and-minimal-effortdeclaration-authoring-bld-001b3).
Implements BI-3.1–BI-3.3 without changing the contract. No ADR, persisted
format, Binding/ModelBinding, Evidence schema, ModeDigest or
EffortSourceDigest change.

Files:

| File | Change |
|---|---|
| `internal/runtimeprofile/v3.go` | `EncodeV3` (the former test-only canonical encoder, now exported for BI-3 only); `DecodeEffortDeclaration` (strict closed object, every member required, Runtime rules of the enclosing Runtime); `MigrateV2ToV3` (revision+1, one declaration, v2 projection digest must equal the source `Digest`); `WithEffortDeclaration` (add or same-`declarationId` replace; ambiguous tuple, foreign identity or missing Runtime refused) |
| `internal/local/runtime_profile_authoring.go` | `InspectVersioned` (zero-write; adds `active-v3`, strictly decoded; v1/v2/tombstone states and tokens unchanged); `Inspect` stays v2-only; `Publish` accepts format 3 only as `active-v2 → v3` or `active-v3 → v3`, canonical bytes, revision+1; v2 or tombstone over v3 refused; post-commit verification v3-aware |
| `internal/local/effort_declaration_input.go` | `ReadEffortDeclarationInput`: one read, regular file, no symlink component except trusted root-owned aliases, `O_NOFOLLOW`, same-file check, configuration size bound |
| `internal/runtimeadapter/effort_applicability.go` | `ObserveEffortApplicability`: LookPath, `ExecutableIdentity`, version observer keyed by that identity, executable re-read for drift; nothing executed |
| `internal/runtimeapplication/authoring.go`, `authoring_declaration.go` | `TargetFormat`/`EffortDeclaration` request fields; declaration branch; shared `confirm` tail (behavior unchanged for existing operations) with revalidation before publication; read-only `v3View` for presentation |
| `internal/cli/runtime_authoring.go` | `--target-format` (binding migrate) and `--effort-declaration` (binding migrate/edit) only |
| `cmd/lingo/runtime_authoring.go` | authoring adapter uses `InspectVersioned`; composes the reader and applicability with the existing `runtimeObservationSources().version` seam |
| `docs/commands.md`, amendment status | public CLI and status reconciliation |

**CLI contract.** `axiom runtime binding migrate --runtime <id> --target-format 3
--effort-declaration <json-file>` and `axiom runtime binding edit --runtime <id>
--effort-declaration <json-file>`, with the unchanged `--expected-revision`,
`--preview-digest`, `--authorize-local`. Other commands, `profile *` and the
v1→v2 flags keep their flag surface. `--target-format` other than 3, without a
declaration, or on another command; `--effort-declaration` without `--runtime`,
with another candidate field or v1 mappings: `invalid_input`.

**State and categories.** Migration requires `active-v2` (`active-v1`:
`migration_required`; `fresh`, `retired`, `active-v3`:
`migration_mapping_required`); declaration edit requires `active-v3`
(otherwise `migration_required`). Unreadable/malformed/oversized/symlinked/
non-strict input: `invalid_input`; absent, unobserved or mismatched executable
digest/version: `effort_configuration_unproven`; ambiguous, foreign or
unconfigured Runtime: `candidate_invalid`; `pending`: `recovery_required`.
Existing categories reused: `revision_overflow`; `unchanged`/`preview_required`
for an identical declaration; on `active-v3` every non-declaration mutation is
the existing `configuration_unavailable` (no downgrade), while `list`/`show`
project it read-only.

**Preview and confirmation.** Zero writes. The existing envelope binds the
snapshot token, source revision, source `Digest` (effect line), candidate
digest, declaration ID and Runtime, observed version, executable digest and
mode, and projected views. Apply re-inspects, re-reads the file once,
re-decodes, re-observes, rebuilds and recomputes the envelope; any difference is
`stale_preview`; the observation is repeated immediately before `Publish` and
drift there is `stale_preview`. A replaced executable between preview and apply
no longer matches the declaration and refuses earlier as
`effort_configuration_unproven`. Neither writes.

**Publication and recovery.** Existing `Publish` protocol: exclusive locks,
exact prior token and wire, atomic replace, post-commit identity and byte
verification. An interrupted or unverifiable publication is
`publication_unconfirmed`, and the following attempt is `recovery_required` (no
automatic retry). Scoped `recover` is unchanged. After roll-forward (F6) the
pre-recovery token is refused. After rollback (F3) the exact pre-attempt
snapshot is restored (same identities and bytes), so a fresh preview is
identical to the earlier one; no new marker was introduced to distinguish them.

**v2 compatibility.** v2 codec, golden bytes, `Digest`, Profile digests and the
v1→v2 migration are unchanged. History gains no declaration. `Inspect`, `Load`,
`Save`, inventory (`unsupported_newer`) and other v2-only consumers keep
refusing format 3 without mutation.

Synthetic Evidence (Go 1.26.0 linux/amd64, `GOTOOLCHAIN=local GOWORK=off
GOFLAGS=`, `set -o pipefail`; every run exit 0, no FAIL, no SKIP):

- `go test ./internal/runtimeprofile -run 'TestDecodeEffortDeclarationIsStrict|TestMigrateV2ToV3|TestWithEffortDeclaration' -v`:
  4 tests (strict input incl. unknown/duplicate/case-alias/missing/oversized/
  trailing/absolute path/`runtimeId`; preservation, Profile digests, canonical
  bytes, other Runtimes without declarations, version alone selects nothing;
  add/replace/ambiguous/foreign/missing Runtime). Historical
  `-run 'TestV3|TestV2'`: 11 tests; whole package: 28.
- `go test ./internal/runtimeapplication -run TestDeclaration -v`: 4 tests,
  29 subtests (preview zero-write and single read, exact apply, revision+1,
  carried Runtimes/Profiles/Preferences, canonical published bytes equal to
  the previewed candidate digest, `EffortSupport`/`ModelSupport` remain
  `unproven`; edit add/replace/unchanged/ambiguous; v2 mutation over v3
  refused; 20 refusals and 9 drift/authority/interruption cases, all with
  zero writes). Whole package: 25 tests, 65 subtests.
- `go test ./internal/local -run 'TestRuntimeProfileV3|TestReadEffortDeclarationInput|TestRuntimeProfileV2OnlyPathsRefuseV3' -v`:
  5 tests, 6 subtests (v2→v3 and v3→v3 publication with exact bytes; legacy
  `Inspect`/`Load`/`Save` refusal; skipped revision, non-canonical bytes, v2
  and tombstone over v3, v1→v3 and retired→v3 refused without mutation; F3/F6
  fault, pending refusal, scoped recovery; regular/symlink/symlinked ancestor/
  missing/directory/oversized/FIFO/NUL input). Existing runtime-profile,
  authoring and recovery suites `-run 'RuntimeProfile|RuntimeAuthoring|RuntimeRecovery'`:
  17 tests, 25 subtests.
- `go test ./internal/runtimeadapter -run TestObserveEffortApplicability`: 1
  test (synthetic executable never run; absent observer, unobserved/unsafe
  version, absent/relative executable, unsupported Runtime, drift).
  `-run 'TestBindingSource|TestInspectEffortSources|TestEffortSources'`: 27
  tests, 74 subtests.
- `go test ./internal/cli`: 118 tests, 1794 subtests, including
  `TestRuntimeDeclarationFlagSurface` and the help/skill parser-consistency
  suites.
- `go test ./cmd/lingo -run 'TestExecutableDeclarationMigrationFailsClosedWithoutVersionObserver|TestExecutableRuntimeAuthoringRestartAndRetirement' -v`:
  2 tests, 4 subtests. Through the built binary and real local store, migration
  refuses as `effort_configuration_unproven`, v2 edit with a declaration as
  `migration_required`, symlinked or untargeted input as `invalid_input`; v2
  bytes unchanged, the synthetic executable is never invoked and no host path
  is reported.
- `gofmt -l`, `go build ./...`, `go vet` (the six touched packages),
  `scripts/check-adr-governance.py`, `scripts/check-sensitive-files.sh`,
  `git diff --check`: exit 0. `go run ./scripts/check-architecture.go
  domain|application` exits 1 on unchanged `internal/project` and
  `internal/projectapp` imports (`encoding/hex`, `strconv`), which this block
  does not touch; it is not fixed here. `gitleaks` is not installed.

No `t.Skip`. Hosts: Linux amd64 only; macOS arm64 and Linux arm64 unverified.

Limits and open risks:

- No production version reader is composed, so in `cmd/lingo` declaration
  authoring always refuses as `effort_configuration_unproven` until
  BLD-001B.3c. This is the intended fail-closed behavior: no probe or Runtime
  execution was added.
- After a successful migration, the v2-only consumers would refuse the record:
  Runtime policy resolution reports `configuration_unsafe`, workflow
  authoring `profile_effort_unconfigured`, and state compatibility (`axiom upgrade`, maintenance
  inventory) classifies it `unsupported_newer`. BLD-001B.3c, or a separate
  decision, must make them v3-aware before migration is operationally usable.

Explicitly not done: BindingSource composition in `cmd/lingo`; no operational
capability proof, probe, inference, stage dispatch, T40 or Provider effect.

Status: AXM-22 not complete; T16/T17 not complete. Remaining: BLD-001B.3c
(production BindingSource and version observation in `cmd/lingo`
`productionCapabilityService`, v3-aware Runtime policy/compatibility consumers
or an explicit decision, synthetic end-to-end validation, final AXM-22
reconciliation); T34/T44 and remaining AXM-8 work.

## AXM-22 BLD-001B.3c-1 — Pre-MVP canonical v3 reconciliation

Decision: Rafael Gomides, 2026-10-14 11:44 and 12:06 America/Sao_Paulo
(pre-MVP canonical format), and the same-day sequencing reconciliation
(Option C), recorded as clause
[PM](amendment-276-axm22-binding-identity.md#pm--pre-mvp-canonical-configuration-format-bld-001b3c).
Before the first stable release configuration v3 is the only target canonical
format: no v2 reading, no v2→v3 migration, no compatibility fallback, no v3→v2
downgrade. PM supersedes prospectively the BI-3 v2→v3 migration (A1, BI-3.1
`--target-format 3`, BI-3.2) and the obligation to preserve v1/v2
compatibility. The BLD-001B.3b record above stays as the historical record of
what was implemented in commit `6660196`; it is not rewritten.

Sequencing (PM-3). A first attempt in this block made the existing authoring
commands write format 3 directly and refused v1/v2 records. It stopped under
the two-attempt rule: `TestRuntimeImpactsSharedProjectsPreservedAfterRemoval`
showed that the v2-only Runtime policy consumer then reports
`configuration_unsafe` instead of `no_allowed_match`, and the CI dogfood
script depends on the v1→v2 path. The maintainer chose Option C: this block
removes the migration and keeps the existing v2 authoring path and consumers
working for operational continuity. BLD-001B.3c-2 removes the remaining v2
paths and switches authoring and consumers to v3 together. **Canonical
v3-only activation is not complete.** No v2 compatibility is accepted for the
final MVP.

Removed (v2→v3 migration only):

- `runtimeprofile.MigrateV2ToV3` and its tests (`TestMigrateV2ToV3PreservesSource`,
  `TestMigrateV2ToV3Refusals`); `v3_migration_test.go` becomes
  `v3_declaration_test.go` with v3 fixtures authored directly.
- `AuthoringRequest.TargetFormat`, the `--target-format` flag on
  `runtime binding migrate`, and `--effort-declaration` on `migrate`.
- The migrate branch of `authorDeclaration` (`migrate local format 2 to 3`
  effects, `migration_mapping_required`/`migration_required` declaration
  categories). A declaration over a format 1/2 or fresh state now refuses as
  `configuration_unavailable` with no migration or conversion.
- The local store's `active-v2 → v3` publication transition: format 3 is now
  accepted over fresh, retired or active-v3 state only, in canonical bytes at
  revision+1, and never over a v1/v2 record (no conversion by overwrite).

Retained (PM-2): `binding edit --runtime <id> --effort-declaration <file>`
with `--expected-revision`/`--preview-digest`/`--authorize-local`; strict
`DecodeEffortDeclaration`; independent executable/version applicability;
zero-write preview; exact confirmation; revalidation immediately before
publication; `WithEffortDeclaration` add/same-id replace with ambiguity and
foreign-identity refusal; atomic publication with the existing locks, compare,
post-commit verification and F3/F6 recovery; the bounded no-symlink
declaration reader. No digest, Evidence, Binding, proof or persisted-format
change; no new command or flag.

Temporary v2 dependencies retained until BLD-001B.3c-2:

- `runtimeapplication.Authoring` v2 create/edit/enable/disable/remove/validate
  and v2 tombstone retirement, and the v1→v2 `runtime binding|profile migrate`
  with `--profile-effort`/`--clear-legacy-references` (used by
  `scripts/dogfood-poc.sh` in CI).
- `local.RuntimeProfileStore` v2/tombstone `Publish` branches and the v2-only
  `Inspect`/`Load`/`Save`.
- v2-only consumers: Runtime policy (`cmd/lingo/runtime_policy.go`), Workflow
  authoring (`cmd/lingo/workflow_authoring.go`,
  `internal/local/workflow_definitions.go`), and state compatibility /
  `axiom upgrade` inventory classification.

Validation (Go 1.26.0, `GOTOOLCHAIN=local`, `GOWORK=off`, `GOFLAGS=`,
`pipefail`; synthetic only, no Runtime execution, probe or inference):

- `go test ./internal/runtimeprofile ./internal/runtimeapplication
  ./internal/local ./internal/cli ./internal/runtimeadapter ./cmd/lingo
  -count=1`: exit 0.
- Focused filters, each executing tests: runtimeprofile declaration/v3 (10
  tests, including `TestNoHistoricalFormatBecomesV3`); runtimeapplication
  `TestDeclaration|TestAuthoring` (13 tests, 40 passes with subtests); local
  v3 publication, refusal, recovery and declaration-reader tests (7 tests, 26
  passes); `TestRuntimeDeclarationFlagSurface`; `cmd/lingo`
  `TestExecutableDeclarationAuthoringWithoutMigration`,
  `TestExecutableRuntimeAuthoringRestartAndRetirement`, `TestRuntimeImpacts*`
  (5 tests, 12 passes). All exit 0.
- Positive coverage: v3 decoding; direct v3 initialization at the store over
  fresh and retired state; declaration add/replace on v3; zero-write preview;
  exact confirmation; snapshot and executable revalidation; canonical atomic
  publication; F3/F6 recovery of v3 → v3.
- Negative coverage: `--target-format` and `migrate --effort-declaration`
  rejected by the parser; declaration over v1/v2/fresh refused with no write;
  v3 never published over v1/v2; v2 or tombstone over v3 refused (no
  downgrade); missing, malformed, oversized, unknown/duplicate-field,
  ambiguous, wrong-Runtime, wrong-executable and version-mismatched
  declarations refused; stale preview, inexact digest/revision and missing
  `--authorize-local` refused; symlinked declaration input refused; no
  capability, model or authentication support inferred; a v2 record is never
  decoded as v3.
- `gofmt -l internal cmd` (empty), `go build ./...`, `go vet` (five touched
  packages), `scripts/check-adr-governance.py`, `scripts/check-sensitive-files.sh`,
  `git diff --check`: exit 0. `gitleaks` is not installed.

No `t.Skip`. Hosts: Linux amd64 only; macOS arm64 and Linux arm64 unverified.
`scripts/dogfood-poc.sh` was not run locally.

Pending, recorded and not claimed:

- BLD-001B.3c-2: v3-only authoring and consumers together; removal of the
  temporary v2 dependencies above; passive `RuntimeVersionObserver` over a
  versioned repository registry of exact executable digests (PM-5). Until it
  is composed, `cmd/lingo` declaration authoring refuses as
  `effort_configuration_unproven`. Format 3 has no retirement record; the v3
  removal of the last Runtime is decided there.
- BLD-001B.3c-3: persistent monotonic authorization epoch (PM-6). Until then,
  rollback recovery restores an identical snapshot whose pre-rollback preview
  token is reusable — a recorded pending security defect, not resolved here.
  `cmd/lingo` BindingSource composition.
- BLD-001B.3c-4: synthetic end-to-end validation and final AXM-22
  reconciliation.

Status: AXM-22 not complete; T16/T17 not complete.

## AXM-22 BLD-001B.3c-2a — Pre-MVP compatibility contract and canonical DigestV3

Decisions C1–C3: Rafael Gomides, 2026-10-14 13:30 America/Sao_Paulo, recorded
as clause
[PM.2 (PM-8–PM-10)](amendment-276-axm22-binding-identity.md#pm2-pre-stable-compatibility-and-canonical-digestv3-bld-001b3c-2a).
Starting HEAD `9287fa9`. This block reconciles contracts and adds one pure
function; no production consumer, persisted field or command changes.

Pre-stable compatibility (C1, PM-8), amended prospectively; earlier text,
receipts and Evidence are preserved:

- [ADR-0024](../../decisions/0024-model-profile-effort-ownership.md): dated
  Status note (the ADR-0010/0015/0019/0021 precedent for a human decision
  without a superseding ADR, not an ADR-0018 supersession annotation) and
  [index](../../decisions/README.md) clause. Effort ownership, execution
  binding semantics, historical v1 meaning and the acceptance receipt are
  unchanged.
- [Specification 004 FR-026](../004-mvp-v1-baseline/spec.md): the historical
  release corpus is distinct from the mandatory supported upgrade window, which
  starts at the first stable/LTS release; published pre-stable local
  Runtime/Profile formats 1/2 have no guaranteed direct upgrade to format 3
  and no required migration; explicit reconfiguration, safe refusal, no silent
  deletion or conversion. Other persisted families are unchanged.
- Specification 007: PM.2 in the AXM-22 amendment, and a dated note in the
  [#306 amendment §3](amendment-306-profile-effort-onboarding.md#3-versioning-and-explicit-migration-strategy)
  superseding only its local configuration forward paths and first-release
  local window. The BI-3 and earlier PM records are not deleted.
- Historical corpus: `internal/compatibility/testdata/stable-v1` is untouched
  (every snapshot, including v0.4.0 and v0.10.0, holds local format 1).
  `TestStableCorpusResolvesToDirectUpgrade` still passes unchanged because the
  current inventory classifies format 1 as `valid_v1`; its expectation change
  depends on the v3 inventory and is specified for BLD-001B.3c-2b in PM.2.

Canonical v3 digest (C2, PM-9): `runtimeprofile.DigestV3` in
`internal/runtimeprofile/v3.go` validates with `ValidateV3`, encodes with the
unchanged `EncodeV3` and returns lowercase hex SHA-256 of those exact bytes;
an invalid configuration returns `ErrInvalidConfiguration` and no digest.
`Digest`, `ProfileDigest`, `EffortSourceDigest`, `ModeDigest` and all persisted
fields are unchanged. `internal/runtimeprofile/v3_digest_test.go`:

- `TestDigestV3GoldenVectors`: five vectors, canonical bytes in
  `testdata/configuration-v3.json` (multiple Runtimes and preferences,
  `e5bb920d…681d40`, equal to `sha256sum` of the pre-existing golden file) and
  `testdata/digest-v3/{codex-declaration,claude-declaration,multiple-declarations,preferences}.json`
  (`5765a511…6f15`, `9addda04…7231`, `4d705a21…3da9`, `2b1cb1c6…5ac6`); each
  asserts EncodeV3 bytes = file, DigestV3 = SHA-256(file) = constant,
  repeatability and reversal of every EncodeV3-ordered collection.
- `TestDigestV3SemanticDrift`: 18 valid mutations yield 19 pairwise distinct
  digests (revision, Runtime set, enabled, Profile model, effort value and
  mode, preferences, declarationId, runtimeVersion, executableDigest, source
  path, scope/anchor swap, JSON key, JSON and environment key inventory,
  declaration removed/added); 10 closed-schema violations (Runtime rename,
  declarationVersion, invocationMode, precedence, anchor, missing source,
  absolute path, ambiguous tuple, format, zero revision) and the zero value
  are refused with no digest.
- `TestDigestV3PreservesHistoricalIdentities`: historical `Digest` constant,
  DigestV3 ≠ Digest of the v2 projection, pinned `ProfileDigest` values
  unchanged by declaration drift, no capability proof in v3 bytes.

Validation (Go 1.26.0, `GOTOOLCHAIN=local`, `GOWORK=off`, `GOFLAGS=`,
`pipefail`; synthetic only, no Runtime execution, probe or inference), all
exit 0:

- `go test ./internal/runtimeprofile -run '^TestDigestV3' -count=1 -v` (3 tests).
- `go test ./internal/runtimeprofile -run '^TestV3|^TestDigest|^TestV2|ProfileDigest' -count=1 -v`
  (15 tests); `go test ./internal/runtimeprofile -count=1`.
- `go test ./internal/runtimeadapter -run '^TestModeDigest|^TestEffortSource' -count=1 -v`
  (17 passes) and `-run '^TestBindingSource|^TestEffortInspect|^TestInspectEffort'`
  (14 passes).
- `go test ./internal/compatibility -run '^TestStableCorpus|^TestV0100|^TestReleasePreparationEnforcesStableCorpus' -count=1 -v`
  (4 passes).
- `gofmt -l internal/runtimeprofile` (empty), `go build ./...`,
  `go vet ./internal/runtimeprofile`, `scripts/check-adr-governance.py`
  (27 ADRs) and its self-test, `validate_examples.py`,
  `scripts/check-sensitive-files.sh`, `scripts/validate-repository.sh`,
  `git diff --check`. `gitleaks` is not installed.

No `t.Skip`. Host: Linux amd64 only; macOS arm64 and Linux arm64 unverified.
The full test matrix and `scripts/dogfood-poc.sh` were not run locally.

Pending, recorded and not claimed:

- BLD-001B.3c-2b: consumer adoption of DigestV3 and coordinated v3-only
  activation (authoring, Runtime policy, Workflow authoring, stage admission,
  readiness, `profile validate`), Runtime retirement binding
  `priorConfigurationDigest` to DigestV3, v3 compatibility/upgrade inventory
  and the corpus test expectation in PM.2.
- BLD-001B.3c-2c: passive `RuntimeVersionObserver`, versioned repository
  registry, synthetic v3 dogfood and remaining fixture conversion.
- BLD-001B.3c-3: persistent authorization epoch (the pre-rollback preview
  reuse defect remains open) and `cmd/lingo` BindingSource composition.
- BLD-001B.3c-4: synthetic end-to-end validation and final AXM-22
  reconciliation.

Status: canonical v3 activation not complete; AXM-22 not complete; T16/T17
not complete.

## AXM-22 BLD-001B.3c-2b.1 — Canonical v3 consumer preparation

Starting HEAD `5ee8671`. Preparation only: internal primitives and synthetic
fixtures for the coordinated v3-only activation, under the decisions already
recorded in PM.1/PM.2. No production consumer is wired to them, and no
persisted field, digest, golden vector, CLI command or flag changes.

Typed v3 projection boundary (`internal/runtimeprofile/v3_consumer.go`):

- `ConsumerV3`, built only by `NewConsumerV3(RecordStateV3, *ConfigurationV3)`
  or `ConsumeVersioned(VersionedConfiguration)`. It reads `ConfigurationV3`
  directly after `ValidateV3`; there is no v3→v2 projection, every
  EffortDeclaration is retained, and identity is the unchanged `DigestV3`
  (`Digest()`), computed only over validated data. `Configuration()` returns
  an independent canonical copy.
- Explicit record states: `RecordMissingV3` → `ErrConfigurationMissingV3`,
  `RecordRetiredV3` (tombstone after the final Runtime) →
  `ErrConfigurationRetiredV3`, inconsistent state/record or invalid v3 →
  `ErrInvalidConfiguration`; historical formats 1/2 →
  `ErrUnsupportedConfigurationFormat`, never upgraded.
- Exact lookups, no implicit default and no substitution: `EnabledRuntime`
  (`ErrRuntimeAbsent`, `ErrRuntimeDisabled`), `AllowedProfile`
  (`ErrProfileMissing`, `ErrProfileRuntimeMismatch`,
  `ErrProfileNotAllowlisted`; the first two reuse the existing authoring codes),
  `ProfileIntent` (Runtime, adapter, Profile, model, effort intent, unchanged
  `ProfileDigest`, revision and DigestV3; no declaration, observation or
  proof) and `EffortDeclaration` (enabled owning Runtime, then the unchanged
  `ApplicableEffortDeclaration`; missing or inapplicable stays
  `effort_configuration_unproven`).

Synthetic fixtures (`internal/runtimeprofile/runtimeprofiletest`, imported by
tests only, the `internal/testfs` precedent; no new dependency): `CodexOnly`,
`ClaudeOnly`, `MultiRuntime` (different allowlists, an owned but unlisted
Profile), explicit and runtime-default effort, valid Codex/Claude
declarations, `WithoutDeclarations`, `WithRuntimeDisabled`, `WithPreferences`,
`WithRevision`, `WithDeclaration`, `WithAllowlisted` (inconsistent
references), unknown Runtime/Profile identifiers, applicability inputs and
`Consumer`/`Encode`/`Digest` helpers. Runtime versions and executable digests
are synthetic; no credential, production support claim or Runtime execution.

Tests (`internal/runtimeprofile/v3_consumer_test.go`, external package):
`TestV3FixturesAreCanonicalAndRoundTrip`, `TestV3ConsumerRecordStates`,
`TestV3ConsumerRefusesHistoricalFormats`, `TestV3ConsumerRuntimeAndProfileLookup`,
`TestV3ConsumerProfileIntent`, `TestV3ConsumerEffortDeclarationOwnership`,
`TestV3ConsumerNoDowngrade`, `TestV3ConsumerDigestIdentity`
(DigestV3 = SHA-256(EncodeV3) per fixture; declaration drift, removal,
revision and preferences change it; ProfileDigest is independent of
declarations and revision). Existing golden vectors and the frozen corpus are
untouched.

Validation (Go 1.26.0, `GOTOOLCHAIN=local`, `GOWORK=off`, `GOFLAGS=`,
`pipefail`; synthetic only, no Runtime execution, probe or inference), all
exit 0:

- `go test ./internal/runtimeprofile -run 'TestV3Consumer|TestV3Fixtures' -count=1 -v`
  (8 passes); `go test ./internal/runtimeprofile -count=1 -v` (38 passes);
  `-run 'TestDigestV3|TestV3'` (18 passes).
- `go test ./internal/runtimeadapter -run 'ModeDigest|EffortSource|Identity|BindingSource' -count=1 -v`
  (34 passes; ModeDigest and EffortSourceDigest unchanged).
- Existing v2 production smoke: `./internal/runtimeapplication -run 'Authoring|Declaration|Policy'`
  (13), `./internal/stageexecution -run 'Admission|Admit'` (1),
  `./internal/local -run 'RuntimeProfile'` (6),
  `./cmd/lingo -run 'RuntimeProfile|RuntimePolicy|Readiness'` (6).
- `gofmt -l internal/runtimeprofile` (empty); `go build` of runtimeprofile,
  runtimeapplication, stageexecution, local and `cmd/lingo`;
  `go vet ./internal/runtimeprofile/...`; `scripts/check-adr-governance.py`
  (27 ADRs); `scripts/check-sensitive-files.sh`;
  `scripts/validate-repository.sh`; `git diff --cached --check`.

No `t.Skip`. Host: Linux amd64 only. The full test matrix and
`scripts/dogfood-poc.sh` were not run locally.

Still active (v2) and unchanged: Runtime policy, Workflow authoring
(`active-v2` gate), stage admission (format-2 gate), readiness,
`profile validate`, upgrade inventory, authoring lifecycle, Runtime retirement,
the v1→v2 migration used by dogfood and the v2 store paths.

Pending, recorded and not claimed:

- BLD-001B.3c-2b.2: remaining consumer preparation (recorded below).
- BLD-001B.3c-2c.1: passive Runtime registry and synthetic v3 dogfood
  preparation.
- BLD-001B.3c-2b.3: coordinated v3-only activation and retirement (wiring of
  `ConsumerV3`, local state mapping, removal of v2 paths, corpus expectation in
  PM.2).
- BLD-001B.3c-3: persistent authorization epoch and `cmd/lingo` BindingSource
  composition.
- BLD-001B.3c-4: final synthetic validation and reconciliation.

Status: canonical v3 activation not complete; AXM-22 not complete; T16/T17
not complete.

## AXM-22 BLD-001B.3c-2b.2 — v3 policy, Workflow, admission, readiness and validation consumers

Starting HEAD `ae4d192`. Preparation only: isolated format-3 evaluators that
reuse `runtimeprofile.ConsumerV3`, `RecordStateV3`, `ProfileIntentV3`,
`DigestV3` and the unchanged `ProfileDigest`. None is called by a production
path; Runtime policy, Workflow authoring, stage admission, readiness and
`runtime profile validate` keep their current v2 behavior. No v3→v2
projection, persisted field, digest, golden vector, CLI command, flag or
feature flag changes.

Prepared evaluators:

- `internal/runtimeapplication/policy_v3.go`:
  - `ConsumerFromRecordV3(state, *ConfigurationV3)` maps an
    `InspectVersioned` state through `RecordStateV3` and `NewConsumerV3`:
    `fresh` → `configuration_missing`, `retired` → `configuration_retired`,
    invalid v3 → `invalid_configuration`, `active-v1/v2` →
    `unsupported_configuration_format`, `pending`/`unsafe` → the existing
    `localStateCategory` codes. A valid v3 record is never
    `configuration_unsafe` merely because the v2 reader refuses format 3.
  - `EvaluatePolicyV3(project, ConsumerV3, Request)` intersects portable
    Project policy (allowed Runtimes, portable Profile Runtime/model
    relationship, preferences) with `AllowedProfile`/`ProfileIntent`, using the
    existing Preview codes (`invalid_request`, `invalid_project`,
    `policy_unconfigured`, `invalid_configuration`, `no_allowed_match`,
    `ambiguous_match`, `unsafe_configuration`). Several admissible Profiles are
    `ambiguous_match`; nothing is selected implicitly. Requested capabilities
    are matched against Profile-declared capabilities only; the candidate
    carries no observation, `ResolvedProfile` or proof, and an
    EffortDeclaration never contributes. Observer inputs and capability proof
    remain on the existing path until activation.
  - `CheckPolicyV3` accepts only the exact reviewed candidate (DigestV3,
    ProfileDigest, revision) and returns `ErrStale` on drift.
  - `ReadinessCandidatesV3(state, *ConfigurationV3)` discloses enabled
    Runtimes with each allowlisted Profile's model and effort intent; no
    selection, no declaration read, distinct missing/retired/invalid codes.
- `internal/projectapp/workflow_profiles_v3.go`:
  `ValidateWorkflowProfilesV3(Definition, ConsumerV3, project.State)` binds
  every schema-2 agent to the exact `ProfileIntentV3` with `migrateWorkflow`'s
  categories (`workflow_migration_required` for schema 1, `forbidden_effort`
  for any agent effort per ADR-0024, `invalid_profile_mapping`,
  `profile_effort_unconfigured`, `project_policy_update_required`,
  `profile_runtime_mismatch`).
- `internal/stageexecution/admission.go`: the configuration-independent part
  of `Admit` is extracted unchanged into `admitExecution`; `Admit` keeps its
  exact v2 decision order. `AdmitV3(Operation, ContractV3)` reuses it, then
  derives current Profile availability from `ConsumerV3.ProfileIntent` of the
  frozen binding (the `stageContract` rule): missing, retired and historical
  v1/v2 → `profile_effort_unconfigured`; invalid v3 →
  `invalid_stage_contract`; policy → `project_policy_update_required`. It
  returns the current `ProfileIntentV3`, not capability proof, and never
  compares or rewrites frozen Execution history. Historical Execution/Graph
  inspection is decided before any configuration read.
- `cmd/lingo/runtime_profile_v3.go`: unexported
  `runtimeProfileValidityV3(snapshot, err)` returns valid only for an active,
  valid v3 record; v1/v2, invalid, missing, retired, unsafe and
  recovery-required are refused. Not wired to `runtime profile validate`.

Synthetic tests (`runtimeprofiletest` fixtures; no legacy v2 fixture converted,
no corpus file changed): `TestPolicyV3ExactProfileIntentAndIdentity`,
`TestPolicyV3NoImplicitRuntimeSelection`, `TestPolicyV3FailsClosed`,
`TestPolicyV3DeclarationIsNotCapabilityProof`,
`TestPolicyV3IdentityDriftIsStale`, `TestConsumerFromRecordV3States`,
`TestReadinessCandidatesV3`, `TestWorkflowProfilesV3BindsExactProfileIntent`,
`TestWorkflowProfilesV3FailsClosed`, `TestAdmitV3ActiveBindsCurrentIdentity`,
`TestAdmitV3RefusesNonCurrentConfiguration`,
`TestAdmitV3PreservesExecutionAndInspectionSemantics` (parity with `Admit`
for every configuration-independent row and record state),
`TestRuntimeProfileValidityV3`, `TestRuntimeProfileValidityV3FreshStoreZeroWrite`.

Validation (Go 1.26.0, `GOTOOLCHAIN=local`, `GOWORK=off`, `GOFLAGS=`,
`pipefail`; synthetic only, no Runtime execution, probe or inference), all
exit 0:

- New: `./internal/runtimeapplication -run 'TestPolicyV3|TestConsumerFromRecordV3|TestReadinessCandidatesV3'`
  (7), `./internal/projectapp -run TestWorkflowProfilesV3` (2),
  `./internal/stageexecution -run TestAdmitV3` (3),
  `./cmd/lingo -run TestRuntimeProfileValidityV3` (2).
- ConsumerV3/DigestV3: `./internal/runtimeprofile -run 'TestV3Consumer|TestV3Fixtures|TestDigestV3|TestV3'`
  (18); `./internal/runtimeprofile` (38);
  `./internal/runtimeadapter -run 'ModeDigest|EffortSource|Identity|BindingSource'` (34).
- Existing production v2 behavior: `./internal/runtimeapplication` (32),
  `./internal/projectapp -run Workflow` (12) and full package,
  `./internal/stageexecution` (54), `./internal/workflowcompiler`,
  `./internal/local -run RuntimeProfile` (6), and 37 `cmd/lingo` Runtime
  policy, Workflow authoring/start/list, readiness/preflight, Runtime impacts,
  stage plan and `runtime profile validate` tests (Executable* and Runtime auth
  tests excluded).
- `gofmt -l` (empty); `go build` and `go vet` of runtimeapplication,
  projectapp, stageexecution, runtimeprofile and `cmd/lingo`; Staticcheck
  0.8.1 with the CI check set on the same packages;
  `scripts/check-adr-governance.py`; `scripts/check-sensitive-files.sh`;
  `scripts/validate-repository.sh`; `git diff --cached --check`.

No `t.Skip`. Host: Linux amd64 only. The full `go test -race ./...` matrix and
`scripts/dogfood-poc.sh` were not run locally.

Still active (v2) and unchanged: every production consumer listed for
BLD-001B.3c-2b.1, upgrade/compatibility inventory, Runtime retirement and the
v2 store paths.

Pending, recorded and not claimed:

- Remaining fixture conversions: the legacy v2 test fixtures stay until
  coordinated activation replaces them.
- BLD-001B.3c-2c.1: passive Runtime registry and synthetic v3 dogfood
  preparation.
- BLD-001B.3c-2b.3: coordinated v3-only activation (wiring the evaluators
  above, observer/proof composition over `ConsumerV3`, retirement, removal of
  v2 paths).
- BLD-001B.3c-3: persistent authorization epoch and `cmd/lingo` BindingSource
  composition.
- BLD-001B.3c-4: final synthetic validation and reconciliation.

Status: canonical v3-only operation is not available; AXM-22 not complete;
T16/T17 not complete.

## AXM-22 BLD-001B.3c-2c.1 — Passive Runtime version registry and synthetic v3 dogfood preparation

Starting HEAD `27ca9ad`. Preparation only: the registry and observer are not
composed by any production path (`cmd/lingo` keeps a nil version observer, so
declaration applicability and BindingSource remain
`effort_configuration_unproven`/blocked). Runtime policy, Workflow authoring,
stage admission, readiness, `runtime profile validate`, the upgrade inventory,
`scripts/dogfood-poc.sh` and the historical corpus are unchanged. No CLI
command, flag, persisted field, digest or ADR changes.

Registry (amendment PM-5a):

- `internal/runtimeadapter/runtime-versions.json`, wire version
  `axiom-runtime-version-registry/v1`, embedded with `go:embed`. Production
  contents: `{"schema":"axiom-runtime-version-registry/v1","entries":[]}`. No
  executable identity has been independently verified and approved, so no
  production entry is claimed; synthetic Codex/Claude identities are never
  production entries.
- `internal/runtimeadapter/version_registry.go`: unexported
  `decodeVersionRegistry` (`strictjson.Decode`, 64 KiB, 256 entries, exact
  members, one schema, adapter Runtime IDs, `SafeRuntimeVersion`, 64
  lowercase hex digests, strict (`runtimeId`, `executableDigest`) order,
  digest unique). `EmbeddedVersionObserver()` is the only exported
  constructor and reads only the embedded bytes; no path, environment value,
  flag or download selects a registry.
- `RegistryVersionObserver.Version(ctx, runtimeID, executableDigest)` has the
  unchanged BindingSource `Version` / `ObserveEffortApplicability` signature.
  It requires the exact pair, returns the registered version, preserves
  context cancellation and otherwise returns `capabilityproof.ErrBlocked`. It
  reads no file and executes nothing; callers supply the canonical
  `ExecutableIdentity` and revalidate it for drift. It reads no declaration,
  Profile, Evidence or capability proof.
- Limitation recorded: `ExecutableIdentity` binds the resolved path, so a
  production entry would be installation-path specific; portable production
  entries need a recorded decision.

Synthetic fixtures (test-only, `//go:build unix`):

- `internal/runtimeadapter/version_registry_test.go`: synthetic executables
  with fixed harmless bytes that would write a marker if run (asserted
  absent), synthetic registry wire builders, registry and observer tests.
- `internal/runtimeadapter/dogfood_v3_fixture_test.go`: `newDogfoodV3Fixture`
  builds `runtimeprofiletest.MultiRuntime()` (Codex + Claude, explicit
  Profile effort, canonical declarations, user/project/managed anchors,
  TOML/JSON/drop-in sources, revision 1), binds each declaration and
  synthetic registry entry to the actual `ExecutableIdentity` of its
  synthetic executable, and lays out `home`/`project`/`managed`/`bin` under
  the test temp root. Isolated from the active dogfood journey, which keeps
  its v1→v2 setup until BLD-001B.3c-2b.3.

Tests: `TestEmbeddedVersionRegistryIsEmptyAndRefuses`,
`TestRegistryVersionObserverExactLookup`, `TestDecodeVersionRegistryFailsClosed`
(29 refusal shapes: malformed, trailing, unsupported/missing schema,
missing/null entries, unknown and case-aliased members, duplicate member,
duplicate/conflicting/digest-reused/unordered entries, unknown or invalid
Runtime, missing/empty/unsafe/secret-bearing/non-string version, invalid
digests, entry and byte bounds), `TestRegistryVersionIsNotDeclarationOrCapability`,
`TestBindingSourceConsumesRegistryVersionObserver` (Codex, Claude; the
registry observer substituted for `Version` in a test fixture),
`TestDogfoodV3FixtureIsCanonical` (`DigestV3` = SHA-256 of `EncodeV3`, stable
decode/encode, `ConsumerV3` declaration lookup),
`TestDogfoodV3FixtureApplicability` (Codex, Claude: observed applicability,
declaration match, `InspectEffortSources`, no cross-Runtime selection, changed
registry version, unknown digest, missing declaration, symlinked source
refusal, changed executable bytes), `TestDogfoodV3FixtureLeavesProductionRegistry`.

Validation (Go 1.26.0, `GOTOOLCHAIN=local`, `GOWORK=off`, `GOFLAGS=`,
`pipefail`; synthetic only, no Runtime execution, probe, inference or
Provider effect), all exit 0:

- New: `go test ./internal/runtimeadapter -run 'Registry|DogfoodV3|BindingSourceConsumes' -v`
  (8 tests, 4 Codex/Claude subtests).
- Regressions, full packages: `./internal/runtimeadapter` (73),
  `./internal/runtimeprofile/...` (38), `./internal/capabilityproof` (54),
  `./internal/runtimeapplication` (32), `./internal/graphapplication` (30),
  `./internal/stageexecution` (54), `./internal/strictjson`; and
  `./cmd/lingo -run 'Runtime|Workflow|Stage|Readiness|Profile|Effort|Binding|Admission'`
  (161), covering production v2 Runtime policy, Workflow authoring, stage
  admission, readiness and profile validation.
- `go build ./...`; `go vet` of runtimeadapter, runtimeprofile and
  `cmd/lingo`; `scripts/check-go-quality.sh format|tidy|staticcheck`
  (Staticcheck 0.8.1); `scripts/check-adr-governance.py`;
  `scripts/check-sensitive-files.sh`; `validate_examples.py`;
  `scripts/validate-repository.sh .`; `git diff --cached --check`.

Environment note: in a shell exporting `CLAUDE_CODE_EFFORT_LEVEL` (a Claude
Code maintainer session), two `graphapplication` tests
(`TestSubscriptionDispatchWithValidatedEffort/claude`,
`TestClaudeEffortDispatchPreservesLargeInheritedEnvironment`) fail
identically at `27ca9ad` because the inherited effort variable is refused;
the regression runs above used a clean environment (`env -i` with
`HOME`/`PATH`/Go caches) and pass. Not changed here.

No `t.Skip`. Host: Linux amd64 only. The full `go test -race ./...` matrix and
`scripts/dogfood-poc.sh` were not run locally (the script is unchanged).

Pending, recorded and not claimed:

- BLD-001B.3c-2b.3: coordinated v3-only authoring and consumer activation,
  dogfood cutover onto these fixtures, Runtime retirement, legacy fixture
  conversion and removal of v2 paths.
- BLD-001B.3c-3: persistent authorization epoch (PM-6) and `cmd/lingo`
  BindingSource composition, including the embedded registry observer.
- BLD-001B.3c-4: final synthetic end-to-end verification and reconciliation.
- Approved production registry entries (none exist; see the limitation).

Status: canonical v3-only operation is not available; AXM-22 not complete;
T16/T17 not complete.

## AXM-22 BLD-001B.3c-2b.3 — Coordinated canonical v3-only activation

Starting HEAD `27f683c`. Authoring, every current Runtime/Profile consumer,
the compatibility inventory and the active dogfood become canonical-v3-only
together, applying the accepted PM-1/PM-8/PM-9 decisions and maintainer
decisions A1/B1 of 2026-10-14 15:51 BRT, recorded as amendment clause
[PM-11](amendment-276-axm22-binding-identity.md#pm3-pre-stable-reconfiguration-and-application-upgrade-classification-bld-001b3c-2b3)
and in [Specification 004 FR-026](../004-mvp-v1-baseline/spec.md). No new
command, flag, persisted field, tombstone format, digest algorithm, registry
format, `ExecutableIdentity` change, production registry entry or ADR. The
first execution of this block stopped at T1.3 before any change because no
authorized reconfiguration path existed for multi-Runtime pre-stable records;
A1/B1 resolved that.

Local lifecycle (`internal/local`, `internal/runtimeapplication`):

- `RuntimeProfileStore.Inspect` is the single zero-write reader: `fresh`,
  `retired`, `active-v3` (strict `DecodeVersioned`), or a recognized
  `active-v1`/`active-v2` record exposed only as `Legacy`. `InspectVersioned`,
  `Create`, `Save`, `Load` and the format-1 writer are removed.
- `Publish` accepts canonical v3 (revision+1) over fresh, retired or active-v3
  state and the existing tombstone over active-v3 (`priorConfigurationDigest =
  DigestV3`) or active-v1/v2 (historical `Digest`); it refuses every format 1/2
  configuration write and any v3 over a v1/v2 record. Locks, exact token,
  filesystem identity checks, atomic replace, post-commit verification and
  F3/F6 recovery are unchanged.
- `Authoring.Run` edits `ConfigurationV3` directly (create, edit, enable,
  disable, remove, validate, validate-candidate, list, show, recover);
  declarations are retained on structural edits; removing one of several
  Runtimes publishes v3, removing the last publishes the tombstone bound to
  `DigestV3`; fresh/retired state starts a new explicit record. The
  `migrate` operation, `MigrationEfforts`, `ClearLegacyReferences`, the CLI
  `runtime binding|profile migrate` leaves and `--profile-effort` /
  `--clear-legacy-references` are removed; the `axiom-runtime` skill table and
  the #230 lifecycle matrix drop them (replaced skill revision recorded in
  `sharedSkillHistory`).
- `retireLegacy` (A1): on active-v1/v2 only `binding remove --runtime <id>
  --cascade` with an existing Runtime retires the entire record; the preview
  lists the ENTIRE-scope meaning, every Runtime/Profile/Preference and the
  historical digest; confirmation is the existing exact authority and the
  store re-checks the token under lock. Every other operation refuses
  (`unsupported_configuration_format`, `binding_missing`,
  `compound_removal_required`, `configuration_unavailable` for declarations,
  `configuration_invalid` for a malformed record).

Consumers:

- Runtime policy (`runtimeapplication.Service`): `Snapshot.Consumer` is a
  `ConsumerV3`; Preview selects with `EvaluatePolicyV3` (exact selection,
  `ambiguous_match`, Project constraints, allowlists, Preferences) and then
  applies the existing observation/proof gates to the selected Profile through
  the new `ConsumerV3.ObservedChoice`, which reuses the format-2 proof rule
  (`resolveProfileAt`) bound to `DigestV3`; no second resolver and no v3→v2
  projection. `Check` re-previews exactly; `Binding.CredentialReference` is
  always empty. Availability uses `ConsumerV3.ProfileReadiness`.
  `capabilityproof.ConfiguredSource` reads the consumer. Pre-stable states map
  to `unsupported_configuration_format`.
- Workflow authoring: `WorkflowProfilePort` returns a `ConsumerV3`;
  `migrateWorkflow` checks references through `ValidateWorkflowProfilesV3`;
  the publication guard rechecks `active-v3` and `DigestV3`.
- Stage admission: `Contract` carries `Record`/`VersionedConfiguration`
  (`ContractV3` is an alias) and `Admit` is `AdmitV3`; `stageContract` maps
  the inspected state; stage dispatch targets read the consumer.
- Readiness candidates use `ReadinessCandidatesV3`; `runtime profile validate`
  uses `runtimeProfileValidityV3` (pre-stable records get explicit retirement
  guidance).
- Compatibility inventory (B1): v3 and tombstone →
  `runtime_profile_configuration`; valid v1/v2 →
  `runtime_profile_prestable_configuration` (upgrade-eligible, reconfiguration
  guidance in the report); malformed → `malformed`; format >3 →
  `unsupported_newer`; protocol names → `recovery_state`.
- `scripts/acceptance/s9-graph-runner.go` (build-ignored acceptance tool)
  publishes canonical v3 instead of the removed v1 writer.

Remaining format 1/2 readers (narrow, historical): store classification and
retirement digest, compatibility inventory, the s9 runner's script-local
graph-planning model, and tests over immutable artifacts/corpora. No
production v2 consumer or fallback remains.

Fixtures (classified): A (current local state) converted to canonical v3 in
`internal/local`, `runtimeapplication`, `capabilityproof`, `stageexecution`,
`graphapplication`, `workflowcompiler`, `projectapp`, `runtimeadapter` and
`cmd/lingo`; B/C (`internal/compatibility/testdata/stable-v1`,
`internal/workflowcompiler/testdata/issue306`) preserved byte-for-byte and now
verified as immutable historical Evidence (`TestIssue306HistoricalCorpusIsImmutableAndReadable`,
split `TestStableCorpusResolvesToDirectUpgrade`). Format-1-only semantics are
asserted as v3 fail-closed results: credential references (none in v3) are
refused before resolution, effort is Profile-owned, machine availability
never selects between two configured Runtimes (`ambiguous_match`), and the
production observer without injected proof is `effort_unproven` /
`authentication_unproven`. Positive lifecycles use the existing explicit
injected-proof seam (`withInjectedProofs`).

Dogfood (`scripts/dogfood-poc.sh`): authors two Runtimes as canonical v3
through `runtime binding create` under an isolated PATH of never-executed
stubs, validates, proves `runtime profile migrate` is gone, asserts
`ambiguous_match` without `--runtime` and `effort_unproven` for the explicit
start, and runs the synthetic receipts (`cmd/lingo` lifecycle/authoring
tests, `internal/runtimeadapter` `TestDogfoodV3Fixture*`). No v1 file, no
migration and no registry override.

New/changed regression coverage includes `TestRuntimeAuthoringLegacyRetirement`,
`TestRuntimeAuthoringLifecycle`, `TestRuntimeProfileReaderRecognizesV3AndRefusesFuture`,
`TestAuthoringLegacyFullRetirement`, `TestAuthoringLegacyMalformedRefusesRetirement`,
`TestConsumerV3ObservedChoiceAppliesExistingProofGates`,
`TestRuntimeProfileUpgradeClassification`, `TestStableCorpusResolvesToDirectUpgrade`,
`TestExecutableDeclarationAuthoringWithoutMigration` (binary: fresh v3,
declaration refusal, disable/enable, last-Runtime retirement with `DigestV3`,
recreate, A1 legacy full retirement and fresh v3), `TestProfileV3ProtectedRetirementCompilerGraphRestartChain`.

Validation (Go 1.26.0, `GOTOOLCHAIN=local`, `GOWORK=off`, `GOFLAGS=`,
`pipefail`, `env -u CLAUDE_CODE_EFFORT_LEVEL`; synthetic only), all exit 0:

- Focused filters (`-v`, tests executed): local store lifecycle/retirement/
  recovery 45; authoring incl. A1 43; Runtime policy 71; `runtimeprofile` +
  `capabilityproof` 209; Workflow authoring 21; Stage admission 41;
  compatibility + stable corpus 14; inventory/writer 37; workflowcompiler
  chain 3; graphapplication policy 61; `cmd/lingo` readiness/validate/policy
  38; binary authoring/retirement 1; black-box journeys 6; v3 dogfood
  fixtures/registry 12; CLI/skills 527.
- `go test ./... -count=1` and `go test -race ./... -count=1`: exit 0.
- `scripts/dogfood-poc.sh` (isolated install, HOME, PATH and state; fake
  `gh`; stubs never executed): exit 0.
- `go build ./...`, `go vet ./...`, `go vet scripts/acceptance/s9-graph-runner.go`,
  `scripts/check-go-quality.sh format|tidy|staticcheck` (Staticcheck 0.8.1;
  first run flagged the now-unused `capture` observer, removed), plus the
  repository validators recorded in the PR.

Environment note: with `CLAUDE_CODE_EFFORT_LEVEL` exported (a Claude Code
maintainer shell), `TestClaudeEffortDispatchPreservesLargeInheritedEnvironment`
refuses the inherited effort override as designed; runs above unset it.

No `t.Skip`. No Runtime executed, probed or asked for its version; no model
inference, live Stage dispatch or Provider effect. `scripts/test-upgrade-journeys.sh`
needs published release assets and runs in CI only.

Limitations and pending, recorded and not claimed:

- The embedded Runtime version registry has no production entry and
  `ExecutableIdentity` binds the resolved installation path; production
  workflow start therefore fails closed (`effort_unproven`) until
  BLD-001B.3c-3 composes the production BindingSource and decides registry
  portability or accepts the limitation.
- PM-6 authorization epoch (pre-recovery preview-token replay after rollback
  remains an open defect) — BLD-001B.3c-3.
- BLD-001B.3c-4: final synthetic end-to-end verification and AXM-22
  reconciliation.

Status: canonical v3-only local Runtime/Profile configuration is active; real
Runtime capability readiness is not claimed; AXM-22 not complete; T16/T17 not
complete.

## AXM-22 BLD-001B.3c-3 — Authorization epoch, production composition and Stage effort context

Scope: [amendment PM.4](amendment-276-axm22-binding-identity.md#pm4-authorization-epoch-and-production-capability-composition-bld-001b3c-3)
— PM-12 (epoch, under the AXM-22 final delivery brief) and PM-13 (Stage
effort-inspection context, maintainer decision approved with constraints,
2026-10-14); also closes BLD-001B.3c-4. Starting HEAD `fe5c85f` (PR #318).

| File | Change |
| --- | --- |
| `internal/local/runtime_authorization_epoch.go` (new) | Epoch record codec, locked read, monotonic staged advance, no wrap |
| `internal/local/runtime_profile_authoring.go` | Epoch in snapshot token and recovery identity; advance before commit; fresh-at-epoch; active v3 without epoch unsafe |
| `internal/local/recovery.go` | Epoch-stage recovery; recovery advances the epoch before acting |
| `internal/local/inventory.go` | Epoch record recognized in its family; invalid record malformed |
| `internal/runtimeadapter/stage_effort.go` (new) | `StageEffortRoots`, `VerifyChildEquivalence`, dispatch-time `StageEffortInspector` |
| `internal/runtimeadapter/effort_inspector.go` | Declaration-returning inspection; validated absent managed root |
| `internal/runtimeadapter/binding_source.go` | Roots resolved for the Target and the effective child environment |
| `cmd/lingo/capability_composition.go` | Production `BindingSource` and inspector composition; operation scope and consent |
| `cmd/lingo/stage_dispatch.go`, `cmd/lingo/stage_supervisor.go` | Operation-scoped composition; registry version for Stage policy; supervisor inspector |
| `internal/local/runtime_authorization_epoch_test.go`, `internal/runtimeadapter/stage_effort_test.go` (new); `cmd/lingo/capability_composition_test.go`, `internal/runtimeadapter/binding_source_test.go`, `internal/runtimeadapter/effort_inspector_test.go`, `internal/local/runtime_profile_v3_refusal_test.go` | PM-6/PM-13 regressions; composition E2E; signature update; missing-root case split (user/project still refuse, managed absence tested separately); directly placed v3 record needs the epoch |
| `scripts/dogfood-poc.sh` | Runs the composition, PM-13 and epoch receipts |

Decisions: the epoch is separate from configuration bytes and advances
before the commit point and before recovery acts, so no attempt outcome can
revive a reviewed token; it is read from the locked descriptor (no lost
update) and tolerated as absent only for states that predate PM-6. The Stage
policy reads the Runtime version from the registry, no longer from
`Source.Current`. PM-13 roots are host/repository-stable, so preview and
dispatch observe one canonical digest, while each attempt separately proves
that the child worktree's effective configuration equals the inspected one.

Synthetic Evidence (Go 1.26.0, `GOTOOLCHAIN=local GOWORK=off GOFLAGS=`,
`pipefail`, `CLAUDE_CODE_EFFORT_LEVEL` unset); see the PR #318 body for the
final command list and exit codes:

- Epoch (`internal/local -run TestAuthorizationEpoch`): fresh preview, exact
  apply, applied-preview reuse, configuration/epoch change, **F3 rollback
  (cleanup and recovery) to byte- and inode-identical configuration followed
  by stale reuse of the pre-F3 preview**, F6 recovery and reuse of the
  pre-F6 preview and applied recovery plan, recovery plan bound to the epoch,
  restart, retirement, recreation, interrupted first publication, concurrent
  writers, no escalation or retry, overflow, corrupt/missing/symlinked/
  permissive record, interrupted epoch persistence and its recovery,
  zero-write preview, inventory.
- PM-13 (`internal/runtimeadapter -run 'TestStageEffort|TestTrustedEffortAnchorAbsentManaged|TestVerifyChildEquivalence'`):
  root resolution and 12 refusals (read-only, HOME missing/relative/unclean/
  root/duplicate/case alias, `CODEX_HOME`, `CLAUDE_CONFIG_DIR`, unknown
  Runtime, untrusted repository, cancellation); absent managed root stable,
  part of identity, never created, distinct once it appears; sticky- or
  group-writable or symlinked parent and missing user/project refuse; child
  equivalence for Codex and Claude with differing bytes, missing on either
  side, child selector, symlink, unsafe or missing child, Claude
  `project-local` only in the repository and managed drop-in drift; the
  dispatch inspector's digest equals the repository-root digest, uses the
  attempt's environment, catches drift between attempts and refuses 11
  failure modes; preview BindingSource and dispatch inspector agree for both
  Runtimes.
- Composition (`cmd/lingo -run 'CapabilityComposition|ProductionEffortInspector'`):
  repository-write positive selection with independently computed identity
  through the production resolver; unproven capability, Process, read-only
  variant and other Profile not inferred; drift after preview and before
  dispatch; a 35-scenario negative matrix including `CODEX_HOME` and missing
  child HOME; production composition blocked by the empty embedded registry;
  consent only with `--authorize-local`; operation context; read-only,
  unconsented and untrusted-repository refusals; supervisor inspector wiring.
- Full suite, race suite, build, vet, format, tidy, staticcheck, dogfood and
  repository validators: recorded in the PR body.
- `internal/compatibility/testdata/stable-v1` unchanged.

IMPLEMENTED: PM-6 epoch; production composition of Store, Source, canonical
v3 configuration, lookup, embedded registry, pinned Axiom, Selector, PM-13
roots, consent and Stage context; dispatch-time equivalence inspector;
passive Stage policy version. TESTED SYNTHETICALLY: all of the above. NOT
OPERATIONALLY PROVEN: any real Runtime identity, version, effort source,
child resolution behaviour or capability; no Runtime executed, probed or
asked for its version, no model inference, Stage dispatch or Provider effect.

Limitations: the embedded registry is empty, so production capability checks
fail closed at the version step; read-only Stage children are blocked
(PM-13 item 3); on macOS the Codex managed root `/etc/codex` is under a
symlinked `/etc` and fails closed.

Status: AXM-22 implementation complete under these limitations; real
Runtime capability availability is not claimed. T16/T17 remain open for
operational proof; AXM-8 not complete; T34/T44 not started.

## RECONCILE 2026-10-15 — main baseline and Task matrix

### Sources, authority and integrated boundary

Read-only audit and documentation-only reconciliation, performed in one session
without subagents, as required by the founder. No inference, live probe, Runtime
version command, product Provider effect, acceptance, merge or release performed.
The requested repository reconciliation and PR authorize publication of the
minimal technical conclusions below; no raw private provider payload is imported.

Observed local HEAD and fetched `origin/main`:
`8242680cdbfc36aa0fb63a8863129fe2ea65c7b3` (2026-10-15 UTC).
All code paths below refer to this exact baseline, not a floating future main.

| Source | Observation / Evidence |
| --- | --- |
| [AXM-8](https://linear.app/rgomids/issue/AXM-8) | Read description, current relations and both comments (no next page). Latest technical direction: 14/10 reconciliation comment `df8bc25f-d795-4823-8e8f-0e7f62b766c7`; 15/10 prompt comment `333a9c4f-f5ee-4f45-bbf6-81db4aa370cc` reiterates it. In Progress; blocks AXM-9; no blockedBy relation. Old 13/10 description is subordinate to later decisions. |
| [GitHub #276](https://github.com/rgomids/axiom/issues/276) | Open; six AC unchecked. Historical planning/backlog/single-original-PR statements remain history, not current authority or completed delivery. |
| [#311](https://github.com/rgomids/axiom/pull/311) | MERGED 2026-10-13 22:43:06Z; `ee3bbaa20f3171613d680e698a34f4f998682a42`; PR HEAD `129897cf0ff639cf3bebf0f921e13f3fb3598f61`. Despite its docs title it also contains stage/graph/capability implementation: `git log -1 -- internal/executiongraph/successor.go internal/stageexecution/run.go internal/cli/workflow_stage.go` identifies this commit. Do not infer documentation-only contents from the title. |
| [AXM-21 / #315](https://github.com/rgomids/axiom/pull/315) | MERGED 2026-10-14 01:51:44Z; `ca105cc8fd8fbadb52eb74df5747b150f799ecab`; PR HEAD `a782744f9d1e76e1f874cc4c98e9f3b5534fac99`. Exact fresh unique Evidence selection and read-only store scan; contributes to T16/T17, does not complete them. |
| [AXM-22 / #318](https://github.com/rgomids/axiom/pull/318) | MERGED 2026-10-15 02:11:49Z; `8242680cdbfc36aa0fb63a8863129fe2ea65c7b3`; PR HEAD `297a0d64ae0dab7f9e974b4ba002e1ee07d18321`. Independent production BindingSource, canonical v3 activation, epoch, Stage roots/equivalence and composition; synthetic only. AXM-21/22 observed Awaiting Release, not product Done. |
| AXM-23–26 | Individually read: all Canceled, no blocking relations. Remaining scope reconsolidated into AXM-8 by founder decision. No new cards, states or relations written. See approved-ID/alias mapping in the [Plan addendum](plan-tasks-276-stage-agent-execution.md#reconciliation-addendum--2026-10-15). |
| Open-PR discovery before edits | `gh pr list --state open --limit 100 --json number,title,body,headRefName,headRefOid,url` returned only #317, branch `feat/immersive-landing-page`, unrelated to AXM-8. Cross-checks: all-state search AXM-8, open search `276 in:body`, AXM-8 attachment/branch and actual #311/#315/#318 refs; no open AXM-8 work PR. New documentation branch `docs/axm-8-reconcile-plan-tasks`; one work PR, no reopening merged PRs. |

The [Plan addendum](plan-tasks-276-stage-agent-execution.md#reconciliation-addendum--2026-10-15)
reconciles the v3-only configuration overlay, successor Graph boundary, delivery
governance, IDs and T41 optional execution. Accepted historical text, approval
blobs and Graph1/2/corpus bytes are preserved. Existing BI-3 migration and early
BLD records above are history superseded prospectively by PM/PM-11 and final
activation. Earlier “BindingSource absent”, “no production v2 consumer” (outside
the residual probe seam below), “T34 not started” and “AXM-22 incomplete” claims
must be read at their dated block boundary, not as current global conclusions.

### Assessment semantics and reading the matrix

`IMPLEMENTED` means the bounded code/use-case requirement is integrated and
supported by named deterministic tests at this baseline; it is **not** a Task
workflow transition, all-platform certification, operational proof or human
acceptance. `PARTIAL` means integrated components exist but an exact remaining
criterion/composition/verification gap is identified. `PENDING` means the
requested surface is absent. `NOT_PROVEN` means the required end-to-end Evidence
has not been established; component tests or historical receipts cannot replace
it. No Task is marked Done by this reconciliation.

Each row links the **unchanged approved title and full Task contract**, repeats
its acceptance criterion, maps its requirement/decision, code and tests, then
names the residual gap and unresolved dependency scopes. Paths in the code column
are relative to repository root; test basenames are in the indicated component
unless qualified. V1/V2/V5 below are this audit's synthetic execution Evidence;
AXM-21/22 sections above retain their historical focused Evidence. No row has
new operational Evidence or human acceptance. Dependencies listed as residual
are the approved predecessors assessed other than IMPLEMENTED; already
implemented prerequisites are consumed, never repeated. An implemented component
can exist before its predecessors' full public/operational verification.

| Task (approved title) | Status | Requirement → acceptance criterion | Integrated implementation | Tests / Evidence | Exact remaining gap; residual dependencies |
| --- | --- | --- | --- | --- | --- |
| [I276-T01 — Specify #276 successor-Graph codec, field contract and version dispatch](plan-tasks-276-stage-agent-execution.md#i276-t01--specify-276-successor-graph-codec-field-contract-and-version-dispatch) | IMPLEMENTED | ADR-0026 D1–6; I-1–I-6. **AC:** Every field has a type, closed values where applicable, requiredness per graph/attempt state and invariant I-1–I-6 mapping; vectors are accepted by the #276 validators and successor strict reader/writer; Graph1/2 fixtures retain identical bytes and SHA-256 digests; old readers reject the successor; no I306-X05 handoff is outstanding. | [internal/executiongraph/successor.go](../../../internal/executiongraph/successor.go); [internal/local/successor_graph_store.go](../../../internal/local/successor_graph_store.go) | successor_test.go: TestSuccessorCanonicalVectors, TestSuccessorHistoricalBytesAndDigests; successor_graph_store_test.go; V1/V2/V5 where applicable | No codec gap identified; operational dispatch is not proved by vectors. **Residual predecessors:** none internal. |
| [I276-T02 — Never redispatch a running attempt (G8 defect)](plan-tasks-276-stage-agent-execution.md#i276-t02--never-redispatch-a-running-attempt-g8-defect) | IMPLEMENTED | G8; AC-5. **AC:** A `running` last attempt is reported `blocked` with a stable reason and never dispatched. | [internal/executiongraph/scheduler.go](../../../internal/executiongraph/scheduler.go) | scheduler_test.go: TestSchedulerConcurrentRunningRetryNeverDispatches, TestReadyChildrenStatuses; V1/V2/V5 where applicable | No local criterion gap identified; no redispatch of running attempts. **Residual predecessors:** none internal. |
| [I276-T03 — Pure attempt classifier and record ordering (CR-001)](plan-tasks-276-stage-agent-execution.md#i276-t03--pure-attempt-classifier-and-record-ordering-cr-001) | IMPLEMENTED | CR-001; PD-9; AC-3/5. **AC:** Every row of the classifier table maps to exactly one status/reason; ordering is total and deterministic; outcome enum enforced. | [internal/executiongraph/classify.go](../../../internal/executiongraph/classify.go); [internal/coordination/coordination.go](../../../internal/coordination/coordination.go) | classify_test.go: TestClassifyAttemptRules, TestLatestAttemptRecordTotalOrdering; V1/V2/V5 where applicable | Rule/ordering coverage is synthetic; no real exchange asserted. **Residual predecessors:** none internal. |
| [I276-T04 — Scheduler, retry and integration consume persisted classification](plan-tasks-276-stage-agent-execution.md#i276-t04--scheduler-retry-and-integration-consume-persisted-classification) | IMPLEMENTED | PD-9; I-1–I-6; AC-4. **AC:** No consumer calls the classifier; status/dispatch/integration agree for every persisted state; invariants checked. | [internal/executiongraph/consumers.go](../../../internal/executiongraph/consumers.go); [internal/executiongraph/integration.go](../../../internal/executiongraph/integration.go) | consumers_test.go: TestConsumerPersistedStateMatrix, TestConsumerLoadInvariants, TestConsumerBatchConcurrency; V1/V2/V5 where applicable | No consumer criterion gap identified; public journey tracked under T26/T34/T36. **Residual predecessors:** none internal. |
| [I276-T05 — Stdin delivery, per-attempt save and drain hook](plan-tasks-276-stage-agent-execution.md#i276-t05--stdin-delivery-per-attempt-save-and-drain-hook) | IMPLEMENTED | PD-2/9; AC-1/4. **AC:** Stdin delivered byte-exact; each attempt persisted independently; hook order enforced. | [internal/executiongraph/scheduler.go](../../../internal/executiongraph/scheduler.go) | scheduler_test.go: TestSchedulerSavesCompletionBeforeBatchEnds, TestOSProcessRunnerStdinBytes, TestSchedulerCompletionFailuresKeepRunningState; V1/V2/V5 where applicable | Local process tests; not real vendor execution. **Residual predecessors:** none internal. |
| [I276-T06 — Run lease, epoch fencing and stale/orphan detection](plan-tasks-276-stage-agent-execution.md#i276-t06--run-lease-epoch-fencing-and-staleorphan-detection) | IMPLEMENTED | PD-2/10; AC-2/5. **AC:** Two holders never coexist; probes never block; stale/orphan classification deterministic; monotonic epochs and process identity survive encode/decode/restart in successor format. | [internal/local/run_lease.go](../../../internal/local/run_lease.go); [internal/executiongraph/reconciliation.go](../../../internal/executiongraph/reconciliation.go) | run_lease_test.go: TestRunLeaseReplacementAndEpochFencing; successor_graph_store_test.go: TestSuccessorSavePinsLeaseThroughCommit; V1/V2/V5 where applicable | Run-lease fencing exists; destructive retry-reset fencing is a distinct missing T31 consumer, not proof that this foundation is absent. **Residual predecessors:** none internal. |
| [I276-T07 — Dispatch-authority record and protected multi-object commit](plan-tasks-276-stage-agent-execution.md#i276-t07--dispatch-authority-record-and-protected-multi-object-commit) | IMPLEMENTED | PD-3/12; ADR-0007. **AC:** Protected authority and successor Graph (or single-agent ledger) commit atomically with recoverable intermediate states; authority scope, preview digest and graph revision binding enforced; read-back verified. | [internal/executiongraph/dispatch_authority.go](../../../internal/executiongraph/dispatch_authority.go); [internal/local/dispatch_authority.go](../../../internal/local/dispatch_authority.go) | dispatch_authority_test.go: TestDispatchCompanionFaultWindows, TestDispatchGraphStoreCommitAndReadback, TestSingleDispatchStoreCommitAndScope; V1/V2/V5 where applicable | Protected commit tested; does not supply the missing T31 reset guard or complete public single-stage composition. **Residual predecessors:** none internal. |
| [I276-T08 — Append-only reconciliation facts](plan-tasks-276-stage-agent-execution.md#i276-t08--append-only-reconciliation-facts) | IMPLEMENTED | PD-10; AC-5. **AC:** Successor facts bind exact parent/graph revision/attempt, authority and observation digest; facts never mutate terminal status; duplicate/contradictory facts preserved and surfaced; retry eligibility reads only persisted validated facts. | [internal/executiongraph/reconciliation.go](../../../internal/executiongraph/reconciliation.go); [internal/local/successor_graph_store.go](../../../internal/local/successor_graph_store.go) | reconciliation_test.go: TestReconciliationAppendRetryAndConflict; successor_graph_store_test.go: TestSuccessorReconciliationPublicationRecovery; V1/V2/V5 where applicable | Append-only facts are not success promotion; reset recovery still needs T31 fencing. **Residual predecessors:** none internal. |
| [I276-T09 — #276 admission guards for versioned contracts](plan-tasks-276-stage-agent-execution.md#i276-t09--276-admission-guards-for-versioned-contracts) | IMPLEMENTED | CR-006; PD-15; PM-11. **AC:** Each historical §3.10 matrix row and approved successor-format delta has guards/tests; #276 mutation of Graph1/2 is refused, read-only status preserved; historical bytes/digests unchanged. | [internal/stageexecution/admission.go](../../../internal/stageexecution/admission.go) | admission_test.go: TestAdmissionHistoricalMatrix, TestAllHistoricalGraphMutationsRefusedAndInspectionPreservesBytes; admission_v3_test.go; V1/V2/V5 where applicable | Current admission is v3; historical local-v2 positive rows require the addendum overlay, not reimplementation of #306. **Residual predecessors:** none internal. |
| [I276-T10 — Capability domain: vocabulary, states, evidence binding, freshness](plan-tasks-276-stage-agent-execution.md#i276-t10--capability-domain-vocabulary-states-evidence-binding-freshness) | IMPLEMENTED | PD-13; D-1. **AC:** Evidence matches a binding only on exact equality; any element change yields `unknown`/`capability_stale`; no path to `proven` without probe facts. | [internal/capabilityproof/evidence.go](../../../internal/capabilityproof/evidence.go) | evidence_test.go: TestFreshnessAndExactBinding, TestNoPromotionWithoutProbe, TestRedactionAndStrictDecode; V1/V2/V5 where applicable | Evidence vocabulary/encoding is implemented; declarations and observations do not prove capability. **Residual predecessors:** none internal. |
| [I276-T11 — Capability evidence store with redaction and recovery](plan-tasks-276-stage-agent-execution.md#i276-t11--capability-evidence-store-with-redaction-and-recovery) | IMPLEMENTED | PD-13; ADR-0007. **AC:** Records round-trip; unsafe fields rejected at encode; recovery recognizes intermediate states. | [internal/local/capability_store.go](../../../internal/local/capability_store.go) | capability_store_test.go: TestCapabilityStoreRoundTripInvalidate, TestCapabilityRecoveryAndInventory, TestCapabilityStoreRejectsUnsafeBeforeEffects; V1/V2/V5 where applicable | Content-addressed storage and AXM-21 read-only enumeration exist; no real proof stored here. **Residual predecessors:** none internal. |
| [I276-T12 — Interface observation (refute-only)](plan-tasks-276-stage-agent-execution.md#i276-t12--interface-observation-refute-only) | IMPLEMENTED | PD-13; level-1 refutation. **AC:** Missing option → `unsupported`; present option → still `unknown` until a probe proves it. | [internal/capabilityproof/interface.go](../../../internal/capabilityproof/interface.go); [internal/runtimeadapter/interface_observation.go](../../../internal/runtimeadapter/interface_observation.go) | interface_test.go: TestInterfaceRefutation; runtimeadapter/interface_observation_test.go: TestInterfaceExactEnvironmentAndReplacement; testdata help fixtures; V1/V2/V5 where applicable | Parser and bounded observation port only refute; production identity uses the empty passive registry, not help output as proof. **Residual predecessors:** none internal. |
| [I276-T13 — Probe envelope lifecycle](plan-tasks-276-stage-agent-execution.md#i276-t13--probe-envelope-lifecycle) | PARTIAL | CR-005; ADR-0005/7. **AC:** Envelope paths resolve without symlinks; cleanup removes only owned objects; quarantine preserved until explicit cleanup. | [internal/capabilityproof/envelope.go](../../../internal/capabilityproof/envelope.go); [internal/local/probe_envelope.go](../../../internal/local/probe_envelope.go) | probe_envelope_test.go: TestProbeEnvelopeLifecycle, TestProbeEnvelopeRefusesReplacedDirectory, TestProbeEnvelopeInterruptedCleanupRecovery; V1/V2/V5 where applicable | POSIX filesystem lifecycle present; catalog platform Evidence beyond this Linux run unverified; production ProbeEnvelopes port absent in T16 composition. **Residual predecessors:** none internal. |
| [I276-T14 — Deterministic canary entrypoint](plan-tasks-276-stage-agent-execution.md#i276-t14--deterministic-canary-entrypoint) | PARTIAL | CR-005/007; PD-13. **AC:** Canary touches only manifest targets; reports are deterministic and redacted. | [internal/capabilityproof/canary.go](../../../internal/capabilityproof/canary.go); [cmd/lingo/probe_canary.go](../../../cmd/lingo/probe_canary.go); [cmd/lingo/main.go](../../../cmd/lingo/main.go) | canary_test.go: TestCanaryManifestTargetsAndRedaction, TestCanaryRefusesBeforeEffects; V1/V2/V5 where applicable | Hidden entrypoint is called with nil loader in main, always refusing; verified envelope/effects loader not composed. Pure canary is not an operable installed probe. **Residual predecessors:** T13. |
| [I276-T15 — Per-variant scoring, positive control and confinement classification](plan-tasks-276-stage-agent-execution.md#i276-t15--per-variant-scoring-positive-control-and-confinement-classification) | IMPLEMENTED | CR-007/008; PD-13. **AC:** `os-enforced` only with both controls passing, identity unchanged, matching uid/gid, allowed-root writes succeeding and errno in {EACCES, EPERM, EROFS}; otherwise `advisory` or `unknown`. | [internal/capabilityproof/classify.go](../../../internal/capabilityproof/classify.go) | classify_test.go: TestVariantScoring, TestConfinementControlMatrix, TestControlMeasurementNoForeignEffects; V1/V2/V5 where applicable | Fake sandbox scoring/control tests; no OS-enforced confinement claim for a real Runtime. **Residual predecessors:** T13, T14. |
| [I276-T16 — Capability probe preview, apply and execution](plan-tasks-276-stage-agent-execution.md#i276-t16--capability-probe-preview-apply-and-execution) | PARTIAL | PD-12/13; CR-007/008; PM. **AC:** Probe never runs without both authorities; evidence binds the exact variant; inconclusive runs leave `unknown`. | [internal/capabilityproof/service.go](../../../internal/capabilityproof/service.go); [cmd/lingo/capability_composition.go](../../../cmd/lingo/capability_composition.go) | probe_test.go: TestProbeEachVariant, TestProbeRefusesBeforeEffects, TestProbeInconclusive; selection_test.go; capability_composition_test.go; V1/V2/V5 where applicable | ProbeAdapter/ProbeEnvelopes/ProbeWriter/ProbeAuth absent in production; PreviewProbe requires ConfigurationVersion == 2 and synthetic probeFixture uses 2. Needs canonical-v3 probe preparation/composition plus installed synthetic proof. AXM-21 selection and AXM-22 BindingSource alone do not finish this Task; real proof remains separate. **Residual predecessors:** none internal. |
| [I276-T17 — Capability observer integration and dispatch-time revalidation](plan-tasks-276-stage-agent-execution.md#i276-t17--capability-observer-integration-and-dispatch-time-revalidation) | PARTIAL | PD-13; BI-1/2; PM-12/13. **AC:** No dispatch with a non-`proven` required capability; any binding drift blocks. | [internal/capabilityproof/observer.go](../../../internal/capabilityproof/observer.go); [internal/runtimeadapter/binding_source.go](../../../internal/runtimeadapter/binding_source.go); [cmd/lingo/capability_composition.go](../../../cmd/lingo/capability_composition.go) | service_test.go: TestCheckBindingDrift, TestRequiredNonProvenBlocks; binding_source_test.go; capability_composition_test.go; V1/V2/V5 where applicable | AXM-21 exact/fresh/unique selection and AXM-22 independent current binding, v3, effort and dispatch equivalence integrated. Embedded registry empty blocks every production version observation; no real binding/capability Evidence or complete installed dispatch proof. **Residual predecessors:** none internal. |
| [I276-T18 — Argument allowlist and confinement flags](plan-tasks-276-stage-agent-execution.md#i276-t18--argument-allowlist-and-confinement-flags) | IMPLEMENTED | PD-4; G5; AC-1. **AC:** Every emitted argv is in the allowlist for the observed version; bypass options can never be emitted. | [internal/runtimeadapter/stage_invocation.go](../../../internal/runtimeadapter/stage_invocation.go) | stage_invocation_test.go: TestStageArgvGolden, TestStageRejectsConfiguredArguments; identity_test.go: TestModeDigestGolden; V1/V2/V5 where applicable | Golden argv tests target supported fixture versions; present vendor compatibility is not established by fixtures. **Residual predecessors:** none internal. |
| [I276-T19 — Environment allowlist and mandatory subscription authentication](plan-tasks-276-stage-agent-execution.md#i276-t19--environment-allowlist-and-mandatory-subscription-authentication) | IMPLEMENTED | PD-5; G13/14; AC-1/6. **AC:** No variable outside the list reaches a child; preflight inspects the same bytes used at spawn. | [internal/runtimeadapter/stage_invocation.go](../../../internal/runtimeadapter/stage_invocation.go); [internal/runtimeadapter/authpreflight.go](../../../internal/runtimeadapter/authpreflight.go) | stage_invocation_test.go: TestStageEnvironmentClosed; authpreflight_test.go: TestAuthPreflightInspectsTheExactEffectiveInvocation; V1/V2/V5 where applicable | Environment/auth gates implemented; no real subscription-auth verification performed. PM-13 further refuses relocated homes for effort inspection. **Residual predecessors:** none internal. |
| [I276-T20 — Task packet builder and stdin delivery](plan-tasks-276-stage-agent-execution.md#i276-t20--task-packet-builder-and-stdin-delivery) | IMPLEMENTED | PD-11; AC-1. **AC:** Packet is deterministic for identical inputs and bounded. | [internal/runtimeadapter/packet.go](../../../internal/runtimeadapter/packet.go); [internal/graphapplication/task_packet.go](../../../internal/graphapplication/task_packet.go) | packet_test.go: TestTaskPacketGraphAndSingleAgent, TestTaskPacketRejectsUnsafeInputs, TestPreparePacketRecordsBeforeDelivery; V1/V2/V5 where applicable | Packet/stdin component available; public single-agent path still has missing T26 seams. **Residual predecessors:** none internal. |
| [I276-T21 — Consume frozen resolvedProfile and effort-source checks at dispatch](plan-tasks-276-stage-agent-execution.md#i276-t21--consume-frozen-resolvedprofile-and-effort-source-checks-at-dispatch) | PARTIAL | CR-003/006; PE-05a; PM-13. **AC:** Adapter emits effort only from a tuple with matching proof; any drift blocks before spawn. | [internal/runtimeadapter/frozen_stage.go](../../../internal/runtimeadapter/frozen_stage.go); [internal/runtimeadapter/stage_effort.go](../../../internal/runtimeadapter/stage_effort.go); [cmd/lingo/stage_supervisor.go](../../../cmd/lingo/stage_supervisor.go) | frozen_stage_test.go: TestFrozenStageDeliveredEffortInspection; stage_effort_test.go; capability_composition_test.go; V1/V2/V5 where applicable | Tuple/effort checks and per-child equivalence integrated synthetically. Production registry and read-only refusal prevent full operational dispatch; v3 overlay applies, no Workflow effort override. **Residual predecessors:** T17. |
| [I276-T22 — Attempt inbox and child-context publish](plan-tasks-276-stage-agent-execution.md#i276-t22--attempt-inbox-and-child-context-publish) | IMPLEMENTED | PD-4; AC-3; nine kinds. **AC:** Candidates land only in the caller's inbox; all nine kinds validated. | [internal/coordination/inbox.go](../../../internal/coordination/inbox.go); [internal/local/attempt_inbox.go](../../../internal/local/attempt_inbox.go); [cmd/lingo/stage_publish.go](../../../cmd/lingo/stage_publish.go) | attempt_inbox_test.go: TestAttemptInboxEveryKindAndExclusiveCreate, TestAttemptInboxRejectsInvalidPublish; stage_dispatch_test.go: TestPublishRefusesAbsentRunningContext; V1/V2/V5 where applicable | Typed inbox publish exists; CLI live-child success and real Q/A belong to T34/T36/T40. **Residual predecessors:** none internal. |
| [I276-T23 — Ingestion, final drain and epoch fencing](plan-tasks-276-stage-agent-execution.md#i276-t23--ingestion-final-drain-and-epoch-fencing) | IMPLEMENTED | PD-4/9; I-1–I-6; AC-3. **AC:** Only validated candidates become canonical; late records rejected; drained set matches classifier input. | [internal/local/coordination_ingest.go](../../../internal/local/coordination_ingest.go); [internal/stageexecution/supervisor.go](../../../internal/stageexecution/supervisor.go) | coordination_ingest_test.go: TestIngestCrossRuntimeQuestionAnswerAndBlocker, TestIngestCapacityReserveAndCrashReplay, TestFinalDrainRacesPublishWithoutLateCanonicalRecord; V1/V2/V5 where applicable | Cross-Runtime labels in these tests are synthetic; no real agents exchanged messages. **Residual predecessors:** none internal. |
| [I276-T24 — Coordination stream read, list, show and bounded wait](plan-tasks-276-stage-agent-execution.md#i276-t24--coordination-stream-read-list-show-and-bounded-wait) | IMPLEMENTED | PD-1/4; AC-3. **AC:** Reads never return inbox candidates; wait is bounded and cancellable. | [internal/coordination/read.go](../../../internal/coordination/read.go); [internal/local/coordination_store.go](../../../internal/local/coordination_store.go) | coordination_read_test.go: TestCoordinationReadStreamCanonicalOnlyAndTamper, TestReadStreamSupportsCompleteCodecByteWindow; V1/V2/V5 where applicable | Read/list/show/wait component present; canonical records only, no raw-chat replay. **Residual predecessors:** none internal. |
| [I276-T25 — Publication builder and dispatch preview](plan-tasks-276-stage-agent-execution.md#i276-t25--publication-builder-and-dispatch-preview) | PARTIAL | PD-3/12; AC-1/4; PM. **AC:** Preview is deterministic and binds every element listed in Plan §3.3; nothing is written. | [internal/stageexecution/preview.go](../../../internal/stageexecution/preview.go); [cmd/lingo/stage_dispatch.go](../../../cmd/lingo/stage_dispatch.go) | preview_test.go: TestDispatchPreviewGraphAndSingle, TestDispatchPreviewRightsGolden, TestDispatchPreviewRefusals; stage_dispatch_test.go; V1/V2/V5 where applicable | Preview/recompilation and retained-publication tests present; complete public v3 composition across capability, graph/single and retry digest paths not demonstrated by installed blackbox Evidence. **Residual predecessors:** T17, T21. |
| [I276-T26 — Stage run apply and single-agent dispatch](plan-tasks-276-stage-agent-execution.md#i276-t26--stage-run-apply-and-single-agent-dispatch) | PARTIAL | PD-3/11/12; AC-1. **AC:** No process starts before the commit is read back; drift refuses with `stale_preview`. | [internal/stageexecution/run.go](../../../internal/stageexecution/run.go); [internal/stageexecution/start.go](../../../internal/stageexecution/start.go); [internal/stagepublication/publisher.go](../../../internal/stagepublication/publisher.go); [cmd/lingo/stage_dispatch.go](../../../cmd/lingo/stage_dispatch.go) | run_test.go: TestRunGraphAndSingleCommitBeforeStart, TestRunConfirmedCommitFailureIsPartial; publisher_test.go; V1/V2/V5 where applicable | Public stageRun omits LocalPublisher.Single and CommittedStarter.Sequential; graph has wiring, single is only injectable/tested at the seam. Complete production graph/single installed path pending. **Residual predecessors:** T25, T27. |
| [I276-T27 — Detached bounded supervisor](plan-tasks-276-stage-agent-execution.md#i276-t27--detached-bounded-supervisor) | PARTIAL | PD-2; AC-1/2/4. **AC:** Supervisor survives the initiating process; stops on drift or cancel; leaves only persisted truth. | [internal/stageexecution/supervisor.go](../../../internal/stageexecution/supervisor.go); [internal/stageexecution/detach_posix.go](../../../internal/stageexecution/detach_posix.go); [cmd/lingo/stage_supervisor.go](../../../cmd/lingo/stage_supervisor.go) | supervisor_test.go: TestSupervisorFailureInjectionPhases, TestSupervisorInitiatingContextDoesNotOwnRun; supervisor_preparation_test.go; V1/V2/V5 where applicable | Detached supervisor and kill-window logic present; full installed v3 production launch/restart with composed proof not established. No live launch here. **Residual predecessors:** T21, T43. |
| [I276-T28 — Stage status and resume after restart](plan-tasks-276-stage-agent-execution.md#i276-t28--stage-status-and-resume-after-restart) | PARTIAL | PD-2/12; AC-2. **AC:** Status computes nothing beyond persisted fields; identical before and after restart. | [internal/stageexecution/status.go](../../../internal/stageexecution/status.go); [internal/stageexecution/resume.go](../../../internal/stageexecution/resume.go); [cmd/lingo/stage_service.go](../../../cmd/lingo/stage_service.go) | lifecycle_test.go: TestStatusRendersPersistedStateAcrossStatuses, TestResumeAfterSupervisorKillStartsOnlyRemainingFirstAttempts; V1/V2/V5 where applicable | Persisted status/resume tested at service seam; installed restart parity still depends on T26/T34/T36; historical inspection does not authorize mutation. **Residual predecessors:** T26. |
| [I276-T29 — Stage cancel](plan-tasks-276-stage-agent-execution.md#i276-t29--stage-cancel) | PARTIAL | PD-6; AC-5. **AC:** Cancel never reports an unobserved stop. | [internal/stageexecution/cancel.go](../../../internal/stageexecution/cancel.go); [cmd/lingo/stage_service.go](../../../cmd/lingo/stage_service.go) | lifecycle_test.go: TestCancelBeforeDispatchPreventsAttempts, TestCancelWhileRunningRequiresConfirmedStop; V1/V2/V5 where applicable | Cancel logic exists; complete public/platform stop-confirmation Evidence and installed journey still pending, not inferred from a request marker. **Residual predecessors:** T43, T27. |
| [I276-T30 — Stage reconcile](plan-tasks-276-stage-agent-execution.md#i276-t30--stage-reconcile) | PARTIAL | PD-10; AC-5. **AC:** Reconcile never upgrades status; facts recorded with observations digest. | [internal/stageexecution/reconcile.go](../../../internal/stageexecution/reconcile.go); [cmd/lingo/stage_service.go](../../../cmd/lingo/stage_service.go) | reconcile_test.go: TestReconcileStaleAndUnknownAppendsFactsOnly, TestReconcileRefusals; V1/V2/V5 where applicable | Service appends facts without promotion; installed recovery/CLI parity remains unproved; facts alone do not authorize destructive reset. **Residual predecessors:** T28. |
| [I276-T31 — Stage retry with workspace disposition](plan-tasks-276-stage-agent-execution.md#i276-t31--stage-retry-with-workspace-disposition) | PARTIAL | PD-8/12; AC-5. **AC:** Retry only from eligible statuses with all checks; dispositions explicit. | [internal/stageexecution/retry.go](../../../internal/stageexecution/retry.go); [cmd/lingo/stage_dispatch.go](../../../cmd/lingo/stage_dispatch.go); `internal/gitworkspace/retry.go` | retry_test.go; recompiled_proposal_test.go; cmd/lingo/stage_dispatch_test.go: TestStageDisposerResetFailsBeforeWorkspacePreparation; gitworkspace/retry_reset_test.go; V1/V2/V5 where applicable | Preserve path and receipt checks exist; reset always returns retry_reset_authority_fencing_unavailable. Need authority-bound current lease/epoch and worktree ownership across each destructive effect and recovery; retained retry preview/digest path lacks complete installed proof. T06/T07/T08 supply primitives, not this guard. **Residual predecessors:** T26. |
| [I276-T32 — Validator policy artifact](plan-tasks-276-stage-agent-execution.md#i276-t32--validator-policy-artifact) | IMPLEMENTED | PD-7; amendment A5. **AC:** Only reviewed policy commands run; digest drift refuses integration. | [internal/stageexecution/validator_policy.go](../../../internal/stageexecution/validator_policy.go); [internal/local/validator_policy_store.go](../../../internal/local/validator_policy_store.go) | validator_policy_test.go: TestPolicyPreviewApplyRefusesAgentAuthorityAndDrift; validator_policy_store_test.go; V1/V2/V5 where applicable | Reviewed argv-only digest policy available; no agent-record authority inferred. **Residual predecessors:** none internal. |
| [I276-T33 — Stage integrate and parent roll-up](plan-tasks-276-stage-agent-execution.md#i276-t33--stage-integrate-and-parent-roll-up) | IMPLEMENTED | PD-7/9; AC-4. **AC:** Roll-up succeeds only with every required child `result_recorded` and validators passing. | [internal/stageexecution/integrate.go](../../../internal/stageexecution/integrate.go); [internal/executiongraph/stage_outcome.go](../../../internal/executiongraph/stage_outcome.go) | integrate_test.go: TestIntegrateTwoChildrenWithValidators, TestIntegrateProjectionFailureIsPartialAndRepairable, TestIntegrateUnknownEffectBlocksRerun; V1/V2/V5 where applicable | Application integration/roll-up criteria covered synthetically; public and real journey pending under T34/T36/T40, never Work Item acceptance. **Residual predecessors:** T26. |
| [I276-T34 — Stage and coordination CLI commands, help and result metadata](plan-tasks-276-stage-agent-execution.md#i276-t34--stage-and-coordination-cli-commands-help-and-result-metadata) | PARTIAL | PD-1/12; AC-1/2/5; AC-009. **AC:** Every operation reachable with correct authority flags, exit codes and JSON/human parity. | [internal/cli/workflow_stage.go](../../../internal/cli/workflow_stage.go); [cmd/lingo/stage_service.go](../../../cmd/lingo/stage_service.go); [cmd/lingo/stage_dispatch.go](../../../cmd/lingo/stage_dispatch.go); [docs/commands.md](../../../docs/commands.md) | cli/workflow_stage_test.go: TestWorkflowStageCommandsRouteExactInputs, TestWorkflowStageJSONAndHumanParity, TestWorkflowStageExitCodesFollowCanonicalStatus; stage_dispatch_test.go; V1/V2/V5 where applicable | CLI and graph composition already integrated, so not “not started”. Complete installed graph/single run, retry digest, restart and command parity missing; T31 reset disabled. No capability CLI ownership here. **Residual predecessors:** T26, T28, T29, T30, T31. |
| [I276-T35 — axiom-work-item skill routes and skill contract tests](plan-tasks-276-stage-agent-execution.md#i276-t35--axiom-work-item-skill-routes-and-skill-contract-tests) | IMPLEMENTED | PD-14; ADR-0022; AC-2/009. **AC:** Routing tests prove ownership exclusivity and no authority from routing. | [internal/codexruntime/skills/axiom-work-item/SKILL.md](../../../internal/codexruntime/skills/axiom-work-item/SKILL.md); [internal/cli/workflow_stage.go](../../../internal/cli/workflow_stage.go) | cli/domain_skill_routing_test.go: TestCanonicalSkillRoutingTableConvergesWithInspection, TestRoutingAloneNeverGrantsAuthority; codexruntime/domain_skills_upgrade_test.go; V1/V2/V5 where applicable | Thin stage/coordination routes integrated; capability remains excluded. This satisfies routing component checks, not native conversational operational acceptance. **Residual predecessors:** T34. |
| [I276-T36 — Synthetic end-to-end suite with fake Runtime executables](plan-tasks-276-stage-agent-execution.md#i276-t36--synthetic-end-to-end-suite-with-fake-runtime-executables) | NOT_PROVEN | AC-1–5; AC-009. **AC:** All listed scenarios green; Evidence classified `synthetic`. | `internal/stageexecution/*_test.go`; [internal/local/coordination_ingest_test.go](../../../internal/local/coordination_ingest_test.go); [cmd/lingo/stage_dispatch_test.go](../../../cmd/lingo/stage_dispatch_test.go) | Service/fake-adapter and local publication tests exist; no complete installed stage blackbox suite located; V1/V2/V5 where applicable | Need catalog's installed fake-codex/claude graph AND single journey, overlap/conflicts, cross-surface restart and all kill windows together. Component suite success is insufficient. **Residual predecessors:** T34, T44. |
| [I276-T37 — Cross-PR compatibility matrix verification](plan-tasks-276-stage-agent-execution.md#i276-t37--cross-pr-compatibility-matrix-verification) | PARTIAL | CR-006; ADR-0026 D6; PM-11. **AC:** All 15 historical rows and every accepted successor admission/refusal row green against delivered #306 code and #276 T01 codec; six-family historical corpus/hash proof preserved. | [internal/stageexecution/admission_test.go](../../../internal/stageexecution/admission_test.go); [internal/stageexecution/admission_v3_test.go](../../../internal/stageexecution/admission_v3_test.go); [internal/executiongraph/successor_test.go](../../../internal/executiongraph/successor_test.go); [internal/compatibility/stable_corpus_test.go](../../../internal/compatibility/stable_corpus_test.go) | TestAdmissionHistoricalMatrix, TestUnknownWireNeverFallsBack, TestSuccessorHistoricalBytesAndDigests, TestStableCorpusResolvesToDirectUpgrade; V1/V2/V5 where applicable | Historical/current admission and frozen corpora covered in separate suites; consolidated all-15-row plus v3/successor restart/failure matrix across public composition still missing. No historical fixture rewrite. **Residual predecessors:** T21, T26, T31. |
| [I276-T38 — Security regression suite](plan-tasks-276-stage-agent-execution.md#i276-t38--security-regression-suite) | PARTIAL | Plan §7/9.1; CR-005/007/008. **AC:** Every listed regression has a failing-before/passing-after or invariant test. | `internal/capabilityproof/*_test.go`; `internal/runtimeadapter/*_test.go`; [internal/local/coordination_ingest_test.go](../../../internal/local/coordination_ingest_test.go) | TestConfinementControlMatrix, TestExtendedMissingAuthority, TestExtendedResidue, TestStageRejectsConfiguredArguments, TestIngestRejectsCandidatesAndReportsSafeCategories; V1/V2/V5 where applicable | Attack/refusal tests exist; complete consolidated installed security report across pending probe/CLI production adapters still missing. No actual confinement guarantee inferred. **Residual predecessors:** T16, T41, T43. |
| [I276-T39 — Documentation reconciliation and implementation record](plan-tasks-276-stage-agent-execution.md#i276-t39--documentation-reconciliation-and-implementation-record) | PARTIAL | Plan §13; documentation governance. **AC:** Every AC and decision mapped to delivered Evidence or an explicit open gate. | [docs/specifications/007-configurable-workflows/issue-276-implementation.md](../../../docs/specifications/007-configurable-workflows/issue-276-implementation.md); [docs/specifications/README.md](../../../docs/specifications/README.md); [docs/commands.md](../../../docs/commands.md) | This dated reconciliation; repository/ADR checks V3; Task/DAG checks V4; V1/V2/V5 where applicable | Current audit does not complete final T39: T36/T37/T38 and final command/help reconciliation remain open; no AC or human acceptance checked off. **Residual predecessors:** T36, T37, T38. |
| [I276-T40 — Bounded real Codex + Claude smoke (separately authorized)](plan-tasks-276-stage-agent-execution.md#i276-t40--bounded-real-codex--claude-smoke-separately-authorized) | NOT_PROVEN | Plan §9.4; AC-1/3/6. **AC:** Specific human authorization and all Plan §9.4 preconditions are satisfied, including IG-1 and the applicable IG-7 gate; real Codex and Claude execute the bounded smoke. Evidence classified `engineering_smoke_real` records all §9.4 fields and checks: exact envelope, subscription auth mode, versions and identity digests, graph, cross-Runtime question/answer and consumed results, local integration, persisted classifications before/after initiating-session exit, cross-surface start/inspect/integrate, and bounded redacted artifacts (AC-6). Missing proof leaves the corresponding acceptance criteria pending; no R-2/R-3 or #278 acceptance claim. | `No new engineering_smoke_real package`; `see gate record below` | NOT_RUN; no real Runtime, inference, probe or Provider effect executed; V1/V2/V5 where applicable | Separate exact-envelope human authority AND operable public v3 proof/dispatch prerequisites missing. Historical S9 and synthetic fixtures cannot satisfy this Task, R-2/R-3 or #278. **Residual predecessors:** T39, T16. |
| [I276-T41 — Extended temp-root probe (optional, PD-17)](plan-tasks-276-stage-agent-execution.md#i276-t41--extended-temp-root-probe-optional-pd-17) | PARTIAL | PD-17; CR-005. **AC:** Refused without its own authority; residue detected. | [internal/capabilityproof/extended.go](../../../internal/capabilityproof/extended.go) | extended_test.go: TestExtendedAuthorizedFakeRoots, TestExtendedMissingAuthority, TestExtendedResidue, TestExtendedForbiddenRoots; V1/V2/V5 where applicable | Guarded use case and fake-root tests exist; production ExtendedEffects and T44 route absent. Real execution optional, implementation/test dependency retained; no shared temp-root probe run here. **Residual predecessors:** T16. |
| [I276-T42 — Capability show and invalidate use cases](plan-tasks-276-stage-agent-execution.md#i276-t42--capability-show-and-invalidate-use-cases) | IMPLEMENTED | PD-13; amendment A1. **AC:** Show never reports `proven` without matching fresh evidence; invalidation is recorded and makes evidence `unknown`. | [internal/capabilityproof/service.go](../../../internal/capabilityproof/service.go); [internal/local/capability_store.go](../../../internal/local/capability_store.go) | service_test.go: TestShowStatesAndProfileRemoval, TestInvalidateAuthorityAndPreview, TestInvalidationPreviewFreezesProfile; V1/V2/V5 where applicable | Show/invalidate use cases implemented; public CLI unavailable under T44 and production readiness blocked under T17. API does not manufacture proven state. **Residual predecessors:** none internal. |
| [I276-T43 — Process containment and confirmed stop](plan-tasks-276-stage-agent-execution.md#i276-t43--process-containment-and-confirmed-stop) | PARTIAL | PD-2/6; AC-1/5. **AC:** Descendants are stopped with the group; unobserved stop is never reported as stopped. | [internal/executiongraph/process_posix.go](../../../internal/executiongraph/process_posix.go); [internal/executiongraph/process_observe_darwin.go](../../../internal/executiongraph/process_observe_darwin.go) | process_posix_test.go: TestPOSIXProcessContainment, TestPOSIXAlreadyExitedGroup; process_test.go; V1/V2/V5 where applicable | Linux local containment tested; no Windows job-object implementation found and no native macOS run in this audit. Frozen catalog mentions Windows; current supported-host policy is Linux/macOS. Historical requirement is not silently waived or a claim of Windows support; any scope disposition needs human authority. **Residual predecessors:** none internal. |
| [I276-T44 — Capability CLI commands, help and result metadata](plan-tasks-276-stage-agent-execution.md#i276-t44--capability-cli-commands-help-and-result-metadata) | PENDING | PD-1/12/14/17. **AC:** Probe operations unreachable without both authorities; extended probe needs its own authority. | [internal/cli/commands.go](../../../internal/cli/commands.go); `cmd/lingo/capability_composition.go (prerequisites only)` | No capability command/blackbox suite located; application service tests belong to T16/T42; V1/V2/V5 where applicable | runtime <id> capability show/probe/invalidate and distinct extended-probe parser/help/result routes absent. Existing runtime readiness commands are not T44. Depends on production T16/T41 plus existing T42. **Residual predecessors:** T16, T41. |

### Remaining gates and acceptance Evidence

| Gate | Classification | Current Evidence and exact outstanding condition |
| --- | --- | --- |
| Canonical version registry | **Confirmed production refusal** | `internal/runtimeadapter/runtime-versions.json` contains `entries: []`; `TestEmbeddedVersionRegistryIsEmptyAndRefuses` and `TestProductionCapabilityCompositionEmbeddedRegistryBlocks` pass. PM-5a accepts only embedded exact identity lookup, not declarations, help, Evidence or an ambient registry override. Independently established version entries remain absent; no fabricated entries or registry design decision added here. |
| Probe preparation/composition (T16/T14) | **Confirmed implementation gap** | `Service.PreviewProbe` requires `ConfigurationVersion == 2`; `probeFixture` also supplies 2. Current v3 admission/authoring does not satisfy that residual seam. `productionCapabilityService` supplies only Store/Source; all four Probe ports are nil. `main.go` supplies nil to the hidden canary loader. Synthetic probe tests pass with injected ports; they do not prove installed operability. This contradicts a blanket “no v2 consumer remains” reading; v3 is the accepted target, and the probe gap must be fixed within that contract, never by restoring v2 operation. |
| Binding/revalidation (T17/T21) | **Integrated foundation; operational validation future** | AXM-21 Selector plus AXM-22 BindingSource, v3 reader, epoch, PM-13 roots and dispatch equivalence tests pass. Registry refusal currently prevents production readiness. Real executable/version/effort/capability proof has not been produced. T16/T17 remain open; operational proof is a separate gate and is not added retroactively to their synthetic Task criteria. |
| Read-only child effort context | **Confirmed scoped limitation** | PM-13 item 3 and `StageEffortRoots.Resolve` refuse non-repository-write Targets (`effort_configuration_unproven`); working root is the private inbox. Independent Runtime-version evidence of project configuration resolution remains required. This does not imply that all repository-write scenarios are blocked for this reason. |
| macOS Codex `/etc/codex` | **Contract/source-backed platform limitation; not run here** | PM-13 item 7 documents symlinked `/etc`; existing no-symlink managed-root rule refuses that host layout. No native macOS inspection or execution occurred in this Linux audit; do not claim a host-specific live failure was reproduced here. |
| Retry reset / authority fencing (T31, consuming T06/T07/T08) | **Confirmed implementation gap, safe refusal** | `cmd/lingo/stage_dispatch.go` `stageDisposer.Dispose` and `internal/gitworkspace/retry.go` refuse reset before effects. `TestStageDisposerResetFailsBeforeWorkspacePreparation`, `TestExplicitResetFailClosedBeforeEffects`, `TestResetWorkspaceFailClosedWithoutManager` pass. T06 leases/epochs, T07 protected authority and T08 facts exist; no implemented guard binds them to each destructive reset effect/recovery. These tests prove refusal, not fencing availability. Preserve remains available as a non-destructive component path; no manual-reset workaround authorized. |
| Multiple matching Evidence | **Decision only if a selection precedence is wanted; not a present unsafe-path blocker** | Accepted AXM-21 behavior is fail-closed ambiguity. `selection_test.go` covers it. No precedence policy is needed to keep refusing; no new decision or automatic ordering introduced. |
| T41 → T38/T44 | **No dependency conflict requiring change** | T41 delivery/refusal/residue tests and guarded CLI are catalog requirements; actual extended probe execution is optional and separately authorized. Edges retained; no mandatory real temp-root test inferred. |
| T43 platform wording | **Bounded human disposition if changing the frozen criterion** | Catalog requires Windows job-object/native platform Evidence; current code has POSIX containment, supported-host policy is Linux/macOS. No Windows implementation/Evidence found and no native macOS run here. Preserve the accepted criterion and report PARTIAL; do not add Windows support or silently waive its historical wording in this reconciliation. Other work proceeds independently. |
| T40 real smoke | **NOT_RUN: future exact-envelope human authority and technical prerequisites** | No smoke envelope was authorized in this session; user explicitly prohibits live probes/inference. Empty production registry, absent probe ports/v3 preparation, incomplete public graph/single journey and final T36–T39 verification are separately evidenced above. This row is blocking/not-run Evidence only, **not** `engineering_smoke_real` and not Task completion. Failed or inconclusive future runs must preserve real Evidence without closing unproven criteria. |
| External baseline / IG-1–7 | **Delivered producer contracts; local verification and live prerequisites separate** | #306/#310 is integrated at `5e1c995ab57ed3fbd54e5b72288607140d031e7d`; no new I306 work. IG-1 supported authoring exists (now AXM-22 v3); clean installed configuration for T40 still unproved. IG-2 vendor-managed credentials and IG-3 adapter parameters are consumed; no real auth inferred. IG-4 capability is #276-owned, open above. IG-5 effort contract and IG-6 runtime skill delivered, not real proof. IG-7 producer side delivered; #276 admission/compatibility/public-composition verification remains its own responsibility. #272 subscription implementation is not live subscription proof. Provider G-4 applies only to scenarios requiring Provider effects; this audit did not test its availability or establish it as a present blocker for every AXM-8 operation. #277/#278 and R-2/R-3 remain successors. |

| Acceptance | Evidence at observed main | Still unproved |
| --- | --- | --- |
| AC-1 installed configured Codex+Claude start | T05/T18–T20 components, T25–T27 service tests; partial public wiring | Complete installed graph **and single** fake-runtime blackbox journey, then authorized real Codex+Claude start without checkout-only runner/state editing. |
| AC-2 either initiating Runtime, restart/status/resume | T06/T28 state/restart tests and T35 thin routes | Complete installed cross-surface/restart journey and real initiating-session independence. |
| AC-3 real cross-Runtime Q/A consumed; deterministic nine-kind/security cases | T03/T22–T24 synthetic nine-kind, lineage, capacity, final-drain tests | Real bounded Q/A and orchestrator consumption; complete installed negative matrix. Labels in fixtures are not real Runtime exchanges. |
| AC-4 permitted overlap, conflicts, integration/roll-up authority | Scheduler overlap/conflict and T33 validator/partial-effect tests | Consolidated installed scenario and its Evidence, preserving authority under the production composition. |
| AC-5 truthful outcomes; uncertain attempts reconciled before retry | Classifier, reconciliation, supervisor kill-window and retry tests | Installed recovery matrix; destructive reset remains unavailable, never silently treated as success. |
| AC-6 early real smoke with exact bounded metadata | No new operational Evidence; T40 NOT_RUN record above | Exact authorized real smoke and all §9.4 fields, validation and applicable acceptance. |
| Spec 007 WF-009 / AC-009 | Components mapped to AC-2/3/5; T34 parser/routing and T35 skill tests | CLI/Codex/Claude installed parity, restart lineage, public nine-kind/security and recovery journey. Neither partial implementation nor this documentation accepts AC-009. |

### Validation ledger — this reconciliation

All Go checks below used the already installed pinned Go 1.26.0 toolchain at
`/home/box/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.0.linux-amd64/bin/go`,
with `env -u CLAUDE_CODE_EFFORT_LEVEL GOTOOLCHAIN=local GOWORK=off GOFLAGS=`.
The initial attempt through `/usr/bin/go` with `GOTOOLCHAIN=local` refused before
running tests because that host binary is Go 1.24.4; no dependencies were installed
and the existing 1.26.0 binary resolved that check. No production Runtime was
executed, asked for a version, authenticated or probed. Local test processes and
fake adapters are synthetic, including filesystem/Git/process containment tests.

| Ref | Command / deterministic inspection | Result / boundary |
| --- | --- | --- |
| V1 | `go test ./internal/executiongraph ./internal/coordination ./internal/capabilityproof ./internal/runtimeadapter ./internal/stageexecution ./internal/stagepublication ./internal/local ./internal/cli ./internal/codexruntime -count=1` | Exit 0, all nine packages; Linux synthetic/component suite only. |
| V2 | `go test ./cmd/lingo -run 'Test(CapabilityComposition\|ProductionCapabilityComposition\|ProductionEffortInspector\|StageDisposer\|RetainedDispatch\|PublishRefuses\|HiddenSupervisor)' -count=1` | Exit 0; focused production-composition/retained-publication/refusal tests with injected dependencies, not complete installed E2E. |
| V3 | `./scripts/validate-repository.sh .` | Exit 0: maintainer harness/adapters, automation registry (91 surfaces), label policy, ADR governance (27 ADRs), sensitive-file checks and diff whitespace. Re-run after documentation reconciliation; exit 0. |
| V4 | `python3 /tmp/axiom-reconcile-check.py` (ephemeral local inspection, no durable automation added) | 44 unique catalog IDs and matrix rows; 100 edges, acyclic, longest path 13 Tasks; DAG = catalog = dependency/reverse table; batch membership B0:15/B1:6/B2:4/B3:3/B4:8/B5:3/B6:4/B7:1; traceability covers every Task; relative destinations and Markdown heading anchors resolve. Entire Plan body from Part I onward identical to baseline. Matrix: 23 IMPLEMENTED components, 18 PARTIAL, 2 NOT_PROVEN, 1 PENDING; no Done transitions. |
| V5 | `go test ./internal/gitworkspace ./internal/compatibility ./internal/workflowcompiler -count=1` | Exit 0, all three packages; reset refusal, frozen historical corpus and compiler checks. No release-asset upgrade journey or native macOS run. |

V4 derives predecessor sets from each catalog's `Depends on`, compares them with
Mermaid edges and both dependency-table directions, then performs a topological
traversal; it compares batch membership and traceability IDs, the 44 matrix rows,
relative links/anchors and the unchanged accepted Plan/catalog suffix against
`git show 8242680c:.../plan-tasks-276-stage-agent-execution.md`. Test symbols and
concrete implementation paths were checked against the baseline source. These
checks validate documentary consistency, not semantic operational acceptance.
No behavioral tests, production files, schemas, model/configuration artifacts or
historical corpus/receipts were changed. Full Go/race/build suites were not
repeated for a documentation-only diff; focused checks substantiate the matrix.

Dedicated `gitleaks` is unavailable on this host; it was not installed.
Worktree and staged sensitive-file checks and manual classification review
cover this documentation diff; no dedicated-scanner result is claimed.

### Review and next executable unit

Review lanes: contracts (v3 overlay and independent formats), authority/security
(no new effects, secret/provider-payload import or operational claims),
traceability (stable IDs, unchanged acyclic DAG) and evidence (synthetic vs real).
No blocking finding against this **documentation-only reconciliation** remains;
implementation gaps and the bounded T43 historical criterion conflict are exposed
above, not waived. The known probe v2 seam is an implementation discrepancy with
the current v3 contract, not a reason to reopen #306 or change accepted formats.
The next implementation must comply with v3 and retain fail-closed behavior.

**Next bounded executable work within consolidated AXM-8:** finish the existing
T16 production probe-preparation boundary for canonical v3 (including the T14
verified canary loader and missing probe ports), with injected/fake-runtime
positive and refusal Evidence and no live probe. Reuse the independently observed
AXM-21/22 identities/selection; do not recreate their foundations or populate the
registry by assumption. The empty registry is a confirmed operational gate, not
an excuse to infer readiness. Subsequent T25/T26/T31/T34 composition must complete
graph/single publication and retry digest/fencing before claiming installed
parity; T44 capability CLI is still absent. This is residual work under existing
Tasks, not a new card, contract, implementation authorization for real effects or
work started by this reconciliation. T36–T39 final verification then T40 exact
human-authorized smoke remain open.

## IMPLEMENT 2026-10-15 — T16 v3 preparation (partial)

Baseline: PR #319 merge `87f882115667cd1de8ac06397a4d11941a01f7fd`.
No open AXM-8 implementation PR existed when this unit began. The work branch
is `feat/axm-8-stage-agent-execution`; continue the same Draft PR.
The human decision is summarized in the scope addendum above; it is authority,
not execution Evidence.

| Task | Current delta | Remaining boundary |
| --- | --- | --- |
| T16 | **PARTIAL**. `PreviewProbe` and its synthetic fixture now require canonical configuration v3. `BindingSource.PrepareProbe` shares the existing complete observation and dispatch invocation construction, including effort projection, without selecting prior Evidence. v1/v2, identity drift, missing authority, stale preview, incomplete binding and authentication refusals remain fail-closed. | The returned preparation deliberately has no envelope manifest or task packet yet. The four production Probe ports remain absent; no installed probe path or complete fake end-to-end production journey is claimed. |
| T14 | **PARTIAL, unchanged**. Existing hidden entrypoint and pure canary tests pass. | Verified production loader, active-envelope context and native effect integration remain unimplemented; `main.go` still supplies nil. Existing fake loader tests are not native loader Evidence. |

**Open integration question, not a new accepted contract:**
`local.prepareEnvelope` assigns physical directory identities to the ready
manifest; `PreviewProbe` requires that exact ready manifest and digest.
Preparing those objects inside preview would violate its no-effect contract.
Replacing the digest after apply would break reviewed-manifest authority.
The session asked whether composition may require a live T13 envelope already
provisioned under independent authorization. No answer has been assumed.
An alternative planned-manifest lifecycle would require explicit contract
clarification; it was not implemented. No accepted ADR was rewritten or found
to require retroactive supersession by the independent v3 correction.

**Synthetic Evidence executed locally (Linux):** all Go commands used the
installed Go 1.26.0 binary, `GOTOOLCHAIN=local GOWORK=off GOFLAGS=` and an unset
`CLAUDE_CODE_EFFORT_LEVEL`. No vendor binary was launched or inspected for its
version/help. Fixture Runtime versions and executable bytes are synthetic.

| Check | Result / limit |
| --- | --- |
| `go test ./internal/capabilityproof ./internal/runtimeadapter ./internal/local ./internal/executiongraph -count=1` | PASS, four packages. `AXIOM_TEST_PROBE_STATE_ROOT` was an isolated private directory under `/home/box`, outside repository/temp/vendor roots, automatically cleaned after the subprocess. Existing envelope lifecycle, symlink/replacement, cleanup/recovery and POSIX containment tests executed. |
| `go test -race ./internal/capabilityproof ./internal/runtimeadapter -count=1` | PASS after the final test change. `TestProbeEachVariant` exercises both positive application-port fixtures; refusals, interruption, inconclusive/model refusal and cleanup failure remain covered. `TestProbeFakeExecutableFailureNeverProves` executes only a generated `/bin/sh` fixture exiting 7: no capability is promoted, output/host paths are excluded from Evidence. This uses an injected envelope verifier and is not the missing production end-to-end journey. |
| `TestPrepareProbeUsesObservedV3WithoutEvidence` | PASS for Codex/Claude × both variants × both Process modes, using synthetic local files and trusted fixture roots. Empty Evidence selection does not prevent preparation or become proof; executable replacement is refused. `TestPrepareProbeRefusesRetiredConfiguration` covers v1/v2 directly. |
| `go test ./cmd/lingo -run 'Test(CapabilityComposition\|ProductionCapabilityComposition\|ProductionEffortInspector\|HiddenCanary\|HiddenSupervisor)' -count=1` | PASS. Production empty-registry refusal and injected canary tests remain intact. |
| `go vet ./internal/capabilityproof ./internal/runtimeadapter ./cmd/lingo` | PASS. |
| `GOOS=darwin GOARCH=arm64 go build ./cmd/lingo ./internal/capabilityproof ./internal/runtimeadapter ./internal/local ./internal/executiongraph` | PASS cross-compilation only; no native macOS execution claimed. |
| `./scripts/check-go-quality.sh format` | PASS. |
| `./scripts/validate-repository.sh .` | PASS; runtime behavioral scenarios deliberately UNVERIFIED, not requested. |
| Accepted Plan/catalog suffix comparison against baseline | PASS: Part I onward is byte-identical; only the current-scope preamble/addendum changed. |

CI now allocates a private synthetic probe state directory under the hosted
runner's HOME on the existing Linux/macOS verify jobs, so envelope tests do not
skip for a missing fixture root. It introduces no real Runtime test or Windows
job. Remote CI results belong to the PR head, not this local ledger.

Engineering/security self-review used `axiom-review`; documentation reconciliation
used `axiom-document`. Standard library and existing observation/filesystem/process
components suffice: no dependency, cryptography or parallel envelope implementation
was added. The only production comment edit corrects the existing public
`ProbePreparation` contract's configuration version. Staged sensitive-file and
whitespace checks are required before commit; dedicated `gitleaks` is unavailable
on this host, so no dedicated-scanner result is claimed.

**Handoff:** T16/T14 are not Done and this unit does not meet the full requested
production integration outcome. The next existing unit is the remaining T16/T14
composition and verified loader after the envelope lifecycle question is resolved:
connect the four ports, bind the exact packet/manifest and active context, exercise
positive and refusal fake executable journeys, then review Evidence in this same
PR. Do not start T25/T26/T31/T34/T44 here. The embedded registry remains empty;
real tests and T40 are **NOT_RUN** by human decision. AC-6 and operational
acceptance remain unproved; no merge, release, Issue closure or Linear state
transition is authorized or performed.


## IMPLEMENT 2026-10-15 — Prepared production probe and canary

Continuation of Draft [PR #320](https://github.com/rgomids/axiom/pull/320),
branch `feat/axm-8-stage-agent-execution`. Resumed WIP
`077fd12606ef4ab688218d45eec7f83a156e4919` is preserved in history. This record
supersedes the earlier partial-unit handoff and qualifies the dated baseline
matrix above; no historical assessment or approval is rewritten.

The lifecycle question is resolved by the maintainer's **option A**, recorded
in the [Plan addendum](plan-tasks-276-stage-agent-execution.md#envelope-lifecycle-decision--option-a).
The two referenced human decisions were read in full. No material conflict with
ADR-0025/0026 or the accepted binding amendment was identified. The accepted
Plan/catalog suffix from Part I remains byte-identical to baseline `87f8821`.

| Task | Delivered boundary | Remaining boundary |
| --- | --- | --- |
| T13 integration | `prepareCapabilityEnvelope` requires independent local preparation authority, reuses `local.CreateEnvelope` and returns its live handle. `InspectReady` verifies the exact manifest digest, physical identities and absence of cleanup intent without writes. Recovery handles cannot dispatch. Quarantine revokes the live handle and retains objects for exact cleanup. | Public preparation/confirmation UX belongs to the existing T44 unit. Native platform execution is assessed by the PR checks, not inferred from Linux. |
| T14 | The hidden production entrypoint now supplies `local.ProbeCanaryLoader` with its effective CWD. It validates the ready manifest, digest, variant, identities and launch/attempt context; uses owned-directory operations, the existing deterministic canary and T22 inbox publication. Only manifest-relative negative targets and an owned loopback listener are exercised. | Real Runtime sandbox behavior is NOT_RUN. This is technical implementation, not merged/accepted Task Done. |
| T16 | All four production ports are composed: `preparedProbeAdapter`, `local.PreparedProbeEnvelopes`, `local.CapabilityStore`, and `preparedProbeAuthentication` with the existing OS auth preflight. Preparation uses canonical v3 BindingSource/effort inspection and the exact ready manifest, task packet, invocation/environment and variant. Apply revalidates before/after independent auth and before the single process invocation; verification reads effects and correlated inbox records, then cleans or quarantines. | Public capability commands remain T44; no installed public probe journey is claimed. Real proof and operational acceptance remain separate. |

No second envelope implementation, planned manifest, dependency, version-registry
entry, Graph/corpus migration or new Task was introduced. Existing filesystem,
process, identity, effort, packet and inbox components suffice. Added production
comments are public API GoDoc or semantic build constraints; the updated
Quarantine GoDoc states its authority-revocation contract.

**Synthetic tests and security review.** `TestPreparedProductionProbeVariants`
copies the test executable into private fixture `codex`, `claude` and `axiom`
paths. These are FAKE executables, never installed vendors. Both Runtimes and
both variants traverse the production adapters, OS auth-status runner, OS process
runner, real T13 envelope, native canary loader/effects, T22 inbox and canonical
Evidence writer. The passive version observer is fixture-injected; the production
embedded registry remains empty. Fixture auth/status results confer no real
subscription or capability proof.

Snapshots verify preview zero-writes, and successful apply leaves the fixture
repository, user/managed roots and executable directory unchanged. Tests check
packet/manifest/attempt correlation, exact stored Evidence bytes and content
address, variant-bound positive effects, advisory confinement, redaction and
exact cleanup. Fake timeout/interruption markers are harness instrumentation in
the fixture executable directory, outside the envelope; they are not canary
effects or persisted Evidence.

Refusals cover independent local/external authority, a different consented
preview, auth failure, manifest/configuration/Profile/environment/executable/Axiom
drift, retired v1/v2, symlink and physical replacement. Fake raw success claims,
nonzero failure, model refusal and usage-limit output cannot promote capabilities.
Timeout/interruption produce inconclusive Unknown Evidence and retain quarantine;
unsafe cleanup never publishes Evidence. Loader tests cover missing/wrong context,
manifest digest, nonce, external address, child binding and symlink drift;
publication tests reject packet/attempt drift and unknown denial. Existing
classification tests retain pre/post control and errno attribution coverage.

The reported `authentication_blocked` failure reproduced with Linux CI's
`-race -covermode=atomic -coverprofile=...` flags, after non-coverage runs passed.
The copied FAKE executable emitted version/status without a terminating newline;
Go coverage then appended `warning: GOCOVERDIR not set, no coverage data emitted`
on stderr, joining the output into `0.159.1warning: ...` or
`Logged in using ChatGPTwarning: ...`. The existing production preflight
correctly rejected the unrecognized version/status. Fake version/status now emit
complete newline-terminated lines. No auth parser, environment allowlist,
coverage setting or production gate was relaxed. The first publication
`4541ecf` passed native macOS but failed Linux coverage; that result is not
reported as green CI. The final PR check must use the subsequent correction.

Separately, the interruption fixture now explicitly requires the invocation-start
marker and `context.Canceled` from marker-triggered cancellation; a bounded
one-minute outer window permits instrumented preflight. Timeout likewise must
reach the fake invocation. Probe budgets allow instrumented subprocess startup
(15 seconds for normal fixtures, 5 seconds for timeout). The test cannot pass by
treating a preflight failure as interruption. Quarantine authority revocation is
also checked independently of process output.

**Validation ledger (Linux, synthetic).** Go 1.26.0 from the existing cached
toolchain (`GOTOOLCHAIN=auto` initially; then its pinned binary on PATH with
`GOTOOLCHAIN=local GOWORK=off GOFLAGS=`), ambient Claude effort unset.
`AXIOM_TEST_PROBE_STATE_ROOT` is a private directory under `/home/box`, outside
repository, temporary and vendor roots. Fixtures are removed by test cleanup.

| Command | Result |
| --- | --- |
| `go test ./internal/capabilityproof -count=1 -timeout=90s` | PASS |
| `go test ./internal/runtimeadapter -count=1 -timeout=120s` | PASS |
| `go test ./internal/local -count=1 -timeout=180s` | PASS, private envelope/session/loader tests enabled |
| `go test ./cmd/lingo -count=1 -timeout=120s` | PASS, complete package (79.3s) |
| `go test -race ./cmd/lingo -run '^TestPreparedProductionProbe' -count=1 -timeout=120s` | PASS, final matrix including outside-root snapshots and refusal cases (65.7s) |
| `go test -race ./cmd/lingo -run '^TestPreparedProductionProbeRefusals/interrupted$' -count=5 -timeout=90s` | PASS, five consecutive dispatch-confirmed interruptions (17.1s total) |
| `go test -race <package> -count=1 -timeout=120s`, separately for every `go list ./...` package except `cmd/lingo` | PASS for all 36 remaining packages; four have no test files |
| `go test -race -covermode=atomic -coverprofile=<private-output> ./cmd/lingo -run '^TestPreparedProductionProbe' -count=1 -timeout=120s` | PASS after the FAKE line-ending correction; all variants/refusals with Linux CI instrumentation (63.8s) |
| Same race/coverage command with `-run '^TestPreparedProductionProbeRefusals/interrupted$' -count=5 -timeout=90s` | PASS, five consecutive interruptions with Linux coverage instrumentation (17.7s) |
| `go vet ./...`; `go build ./...`; `go mod verify` | PASS |
| `./scripts/check-go-quality.sh format`, `tidy`, `staticcheck` | PASS; Staticcheck 2026.2.1 / v0.8.1 |
| `./scripts/validate-repository.sh .` | PASS; vendor runtime behavioral checks deliberately UNVERIFIED |
| Accepted Plan/catalog suffix comparison against `87f8821` | PASS |

The complete local `cmd/lingo` race attempt with `-timeout=180s` reached that
limit during the pre-existing Work Item projection blackbox, without an
assertion failure. Its full race result remains for CI; the complete non-race
package and bounded probe race matrix passed. No timeout result is called PASS.

Commit identity and remote Linux/macOS CI conclusions are recorded in the same
PR after publication, against its exact head. A native
macOS CI result is required; cross-compilation is not a substitute.

Security/engineering self-review found no remaining blocking finding in this
bounded diff. The dedicated `gitleaks` scanner is unavailable on this host; no
dedicated-scanner result is claimed. Staged sensitive-file and whitespace checks
passed before commit.

**Handoff.** This bounded slice integrates the requested production seams;
T13/T14/T16 are not declared merged, accepted or operationally proven. Continue
in this same Draft PR. The next existing capability unit is T41's production
extended-effects boundary before T44 public capability commands; consolidated
stage dispatch still has its existing T25/T26/T31/T34 work. None was started
here. Real probes, vendor version/help/authentication, inference, installed
smoke, Provider effects and T40 remain **NOT_RUN**. AC-6, real capabilities and
human operational acceptance remain unproved. No merge, release, Issue closure,
Linear lifecycle/blocker/label change or new Work Item is authorized by this
record. No additional human decision is needed for the delivered slice.

### Review corrections 2026-10-15 — PR #320 round 1, CR-001 / CR-002

Scope is exclusively the two Major findings in
[review 5539186839](https://github.com/rgomids/axiom/pull/320#pullrequestreview-5539186839),
against `481f6997f806beef74466678d13c9a9b33568f41`. The earlier implementation
ledger remains historical; its capability-promotion and advisory-confinement
claims are corrected by this entry. No Task, accepted Plan/catalog, Specification,
ADR, receipt or historical corpus is changed.

**CR-001.** Inbox lineage, digest, nonce, process identity and denial reports
are consistency checks on child-controlled data. Neither unchanged targets nor
parent-side positive controls attest that a denied syscall happened. A loopback
nonce exchange also cannot identify the executable responsible. The existing
contract provides no independent attestation of canary execution, model invocation
or effective effort. The smallest safe correction retains those verification
flags as false and the verification as inconclusive. Observed file contents and
validated publications remain diagnostic observations; `Service.Probe` consequently
persists all capability states and confinement as `unknown`. This applies even
when the cooperative synthetic child runs the actual canary. These production
ports cannot currently establish capability proof. No signature, permission
expansion, dependency or alternative probe mechanism is introduced. This follows
ADR-0025's own-observation and unknown rules, without changing that decision.

The adversarial subprocess fixture reads the manifest, workspace nonce and
launch context; writes the expected positive files; and publishes a valid
`Progress` containing a correlated `CanaryReport` with fabricated `EACCES` for
every negative target. It never invokes the canary or attempts a negative write.
Both fake Runtime identities and both variants must reach report verification
with exit zero and accepted negative observations, while `CanaryRan`,
`CanaryExecuted`, `ModelInvoked` and `EffortVerified` remain false. Final Evidence
must be inconclusive, with every capability and confinement `unknown`. Cooperative
positive fixtures still require observed nonce/workspace/publication/canary-file
effects, packet binding, zero-write preview, exact stored Evidence, unchanged
outside roots and exact cleanup; they now expect honest unknown capability states.

**CR-002.** `Service.Probe` cannot quarantine an envelope when its opener fails.
The opener now owns failure recovery from immediately before the first launch-file
write, including partially failed writes. Every unsuccessful return closes its
listener, revokes the live preparation through existing `Envelope.Quarantine`,
retains owned objects, and joins any quarantine verification error with the
original failure. Recovery uses the existing inventory and exact cleanup protocol;
there is no implicit deletion, reuse or retry. A private create callback at the
existing inbox-creation boundary permits deterministic fault injection without
changing the public port or adding a second lifecycle.

Fault tests inject ENOSPC after launch persistence, after a partial attempt inbox
with incomplete binding, and after complete inbox creation. A fourth test starts
cleanup recovery before the final ready inspection. Every case requires a revoked
handle, closed listener, refused reopening for dispatch, preserved objects,
`InventoryRecovery`, recovery-only reopening and successful explicit exact cleanup.

Validation uses the cached Go 1.26.0 binary with
`GOTOOLCHAIN=local GOWORK=off GOFLAGS=`, ambient Claude effort unset and
`AXIOM_TEST_PROBE_STATE_ROOT` under a private `/home/box` root outside repositories,
temporary and vendor roots. The system Go 1.24.4 initially refused the module;
that attempt executed no tests.

| Command | Result |
| --- | --- |
| `go test -race ./internal/local ./cmd/lingo -run 'TestProbeOpenFailure\|TestPreparedProductionProbe(ForgedReport\|Variants)' -count=1 -timeout=180s` | PASS |
| `go test -race ./internal/local ./internal/capabilityproof ./internal/runtimeadapter -count=1 -timeout=180s` | PASS, complete packages |
| `go test -race -covermode=atomic -coverprofile=<private-output> ./cmd/lingo -run 'TestPreparedProductionProbe\|TestPrepareEnvelopeRequiresIndependentLocalAuthority\|TestHiddenCanary' -count=1 -timeout=180s` | PASS, full probe matrix with Linux CI instrumentation |
| `go test -race ./cmd/lingo -run '^TestPreparedProductionProbeVariants$' -count=1 -timeout=120s` | PASS after adding explicit positive-effect assertions |
| `go vet ./...`; `go build ./...`; `go mod verify` | PASS |
| `./scripts/check-go-quality.sh all` | PASS, format/tidy/Staticcheck v0.8.1 |
| `./scripts/validate-repository.sh .` | PASS, Runtime behavioral scenarios deliberately UNVERIFIED |

Security review: child publications cannot elevate canonical capability Evidence;
cleanup remains bound to verified ownership and exact objects. No material ADR
conflict was identified. Dedicated `gitleaks` remains unavailable locally; no dedicated scanner result is claimed. Main-session implementation, review and validation were
used without delegation because strict capability inferiority could not be
verified. Real Runtime/version/help/authentication, inference, live probes,
installed smoke, Provider effects and T40 remain **NOT_RUN**. No acceptance,
merge, release, Issue/Task closure or Linear mutation. Remote exact-head native
Linux/macOS CI and the re-review request are recorded on this same Draft PR.


## CONTINUE 2026-10-15 — Operational checkpoint for PR #320

Snapshot before edits: clean `feat/axm-8-stage-agent-execution` at
`e05e209f7a77d5f56ad67ce02320cd533b8ea9b5`; remote branch matches;
`main@87f882115667cd1de8ac06397a4d11941a01f7fd`. GitHub read-back:
#276 OPEN; #320 OPEN/DRAFT, sole open AXM-8 work PR; unrelated #317 is not
part of this work. CI run 38804515821 and delivery metadata PASS for that
exact baseline HEAD, including native Linux/macOS. Review 5540357018 is
favorable for its slice, native COMMENT/self-review, not native approval.
CR-001/CR-002 remain resolved at baseline. AXM-8 is In Progress, no blockedBy,
blocks AXM-9. No lifecycle, relation, label or acceptance mutation is authorized.

The three maintainer decisions named in the continuation request were read in
full and match the versioned scope addenda. Only technical decision summaries
already authorized for this public repository are retained here. No raw provider
comments are imported. No new Specification, ADR or Work Item is needed for the
approved work. No delegation: a strictly lower model hierarchy cannot be
verified from this runtime's metadata, so the Orchestrator performs the units.

### Current Task inventory

This overlays, rather than replaces, the dated main-baseline matrix. INTEGRADO
means the component and its cited synthetic Evidence exist on main, not that
the whole installed journey or Task Done has human acceptance. IMPLEMENTADO NO
PR is unmerged and qualified by remaining public/operational boundaries. Tests
below identify existing Evidence, not new executions in this checkpoint. Exact
criteria, file links and residual gaps remain in the baseline rows; dependencies
below are the immutable approved catalog, with the later v3/Linux/macOS addenda.

| Task | State | Catalog predecessors | Requirement / AC | Existing Evidence locator |
| --- | --- | --- | --- | --- |
| I276-T01 | INTEGRADO | none (internal) | ADR-0026 D1–6; I-1–I-6. | successor_test.go: TestSuccessorCanonicalVectors, TestSuccessorHistoricalBytesAndDigests; successor_graph_store_test.go |
| I276-T02 | INTEGRADO | none (internal) | G8; AC-5. | scheduler_test.go: TestSchedulerConcurrentRunningRetryNeverDispatches, TestReadyChildrenStatuses |
| I276-T03 | INTEGRADO | none (internal) | CR-001; PD-9; AC-3/5. | classify_test.go: TestClassifyAttemptRules, TestLatestAttemptRecordTotalOrdering |
| I276-T04 | INTEGRADO | T01, T02, T03 | PD-9; I-1–I-6; AC-4. | consumers_test.go: TestConsumerPersistedStateMatrix, TestConsumerLoadInvariants, TestConsumerBatchConcurrency |
| I276-T05 | INTEGRADO | none (internal) | PD-2/9; AC-1/4. | scheduler_test.go: TestSchedulerSavesCompletionBeforeBatchEnds, TestOSProcessRunnerStdinBytes, TestSchedulerCompletionFailuresKeepRunningState |
| I276-T06 | INTEGRADO | T01 | PD-2/10; AC-2/5. | run_lease_test.go: TestRunLeaseReplacementAndEpochFencing; successor_graph_store_test.go: TestSuccessorSavePinsLeaseThroughCommit |
| I276-T07 | INTEGRADO | T01, T06 | PD-3/12; ADR-0007. | dispatch_authority_test.go: TestDispatchCompanionFaultWindows, TestDispatchGraphStoreCommitAndReadback, TestSingleDispatchStoreCommitAndScope |
| I276-T08 | INTEGRADO | T04, T06 | PD-10; AC-5. | reconciliation_test.go: TestReconciliationAppendRetryAndConflict; successor_graph_store_test.go: TestSuccessorReconciliationPublicationRecovery |
| I276-T09 | INTEGRADO | T01 | CR-006; PD-15; PM-11. | admission_test.go: TestAdmissionHistoricalMatrix, TestAllHistoricalGraphMutationsRefusedAndInspectionPreservesBytes; admission_v3_test.go |
| I276-T10 | INTEGRADO | none (internal) | PD-13; D-1. | evidence_test.go: TestFreshnessAndExactBinding, TestNoPromotionWithoutProbe, TestRedactionAndStrictDecode |
| I276-T11 | INTEGRADO | T10 | PD-13; ADR-0007. | capability_store_test.go: TestCapabilityStoreRoundTripInvalidate, TestCapabilityRecoveryAndInventory, TestCapabilityStoreRejectsUnsafeBeforeEffects |
| I276-T12 | INTEGRADO | T10, T19 | PD-13; level-1 refutation. | interface_test.go: TestInterfaceRefutation; runtimeadapter/interface_observation_test.go: TestInterfaceExactEnvironmentAndReplacement; testdata help fixtures |
| I276-T13 | IMPLEMENTADO NO PR | T11 | CR-005; ADR-0005/7. | T13/T14/T16 production integration + review corrections |
| I276-T14 | IMPLEMENTADO NO PR | T13, T22 | CR-005/007; PD-13. | T13/T14/T16 production integration + review corrections |
| I276-T15 | INTEGRADO | T13, T14 | CR-007/008; PD-13. | classify_test.go: TestVariantScoring, TestConfinementControlMatrix, TestControlMeasurementNoForeignEffects |
| I276-T16 | IMPLEMENTADO NO PR | T11, T12, T15, T18, T19, T20 | PD-12/13; CR-007/008; PM. | T13/T14/T16 production integration + review corrections |
| I276-T17 | PARCIAL | T10, T11, T18 | PD-13; BI-1/2; PM-12/13. | service_test.go: TestCheckBindingDrift, TestRequiredNonProvenBlocks; binding_source_test.go; capability_composition_test.go |
| I276-T18 | INTEGRADO | none (internal) | PD-4; G5; AC-1. | stage_invocation_test.go: TestStageArgvGolden, TestStageRejectsConfiguredArguments; identity_test.go: TestModeDigestGolden |
| I276-T19 | INTEGRADO | none (internal) | PD-5; G13/14; AC-1/6. | stage_invocation_test.go: TestStageEnvironmentClosed; authpreflight_test.go: TestAuthPreflightInspectsTheExactEffectiveInvocation |
| I276-T20 | INTEGRADO | T05 | PD-11; AC-1. | packet_test.go: TestTaskPacketGraphAndSingleAgent, TestTaskPacketRejectsUnsafeInputs, TestPreparePacketRecordsBeforeDelivery |
| I276-T21 | PARCIAL | T17, T18 | CR-003/006; PE-05a; PM-13. | frozen_stage_test.go: TestFrozenStageDeliveredEffortInspection; stage_effort_test.go; capability_composition_test.go |
| I276-T22 | INTEGRADO | none (internal) | PD-4; AC-3; nine kinds. | attempt_inbox_test.go: TestAttemptInboxEveryKindAndExclusiveCreate, TestAttemptInboxRejectsInvalidPublish; stage_dispatch_test.go: TestPublishRefusesAbsentRunningContext |
| I276-T23 | INTEGRADO | T03, T05, T06, T22, T24 | PD-4/9; I-1–I-6; AC-3. | coordination_ingest_test.go: TestIngestCrossRuntimeQuestionAnswerAndBlocker, TestIngestCapacityReserveAndCrashReplay, TestFinalDrainRacesPublishWithoutLateCanonicalRecord |
| I276-T24 | INTEGRADO | none (internal) | PD-1/4; AC-3. | coordination_read_test.go: TestCoordinationReadStreamCanonicalOnlyAndTamper, TestReadStreamSupportsCompleteCodecByteWindow |
| I276-T25 | PARCIAL | T09, T17, T18, T32, T19, T21 | PD-3/12; AC-1/4; PM. | preview_test.go: TestDispatchPreviewGraphAndSingle, TestDispatchPreviewRightsGolden, TestDispatchPreviewRefusals; stage_dispatch_test.go |
| I276-T26 | PARCIAL | T07, T25, T27 | PD-3/11/12; AC-1. | run_test.go: TestRunGraphAndSingleCommitBeforeStart, TestRunConfirmedCommitFailureIsPartial; publisher_test.go |
| I276-T27 | PARCIAL | T04, T05, T06, T21, T23, T07, T19, T20, T43 | PD-2; AC-1/2/4. | supervisor_test.go: TestSupervisorFailureInjectionPhases, TestSupervisorInitiatingContextDoesNotOwnRun; supervisor_preparation_test.go |
| I276-T28 | PARCIAL | T04, T06, T26, T07 | PD-2/12; AC-2. | lifecycle_test.go: TestStatusRendersPersistedStateAcrossStatuses, TestResumeAfterSupervisorKillStartsOnlyRemainingFirstAttempts |
| I276-T29 | PARCIAL | T43, T27 | PD-6; AC-5. | lifecycle_test.go: TestCancelBeforeDispatchPreventsAttempts, TestCancelWhileRunningRequiresConfirmedStop |
| I276-T30 | PARCIAL | T08, T28 | PD-10; AC-5. | reconcile_test.go: TestReconcileStaleAndUnknownAppendsFactsOnly, TestReconcileRefusals |
| I276-T31 | PARCIAL | T08, T26 | PD-8/12; AC-5. | retry_test.go; recompiled_proposal_test.go; cmd/lingo/stage_dispatch_test.go: TestStageDisposerResetFailsBeforeWorkspacePreparation; gitworkspace/retry_reset_test.go |
| I276-T32 | INTEGRADO | none (internal) | PD-7; amendment A5. | validator_policy_test.go: TestPolicyPreviewApplyRefusesAgentAuthorityAndDrift; validator_policy_store_test.go |
| I276-T33 | INTEGRADO | T04, T26, T32 | PD-7/9; AC-4. | integrate_test.go: TestIntegrateTwoChildrenWithValidators, TestIntegrateProjectionFailureIsPartialAndRepairable, TestIntegrateUnknownEffectBlocksRerun |
| I276-T34 | PARCIAL | T24, T26, T28, T29, T30, T31, T33 | PD-1/12; AC-1/2/5; AC-009. | cli/workflow_stage_test.go: TestWorkflowStageCommandsRouteExactInputs, TestWorkflowStageJSONAndHumanParity, TestWorkflowStageExitCodesFollowCanonicalStatus; stage_dispatch_test.go |
| I276-T35 | INTEGRADO | T34 | PD-14; ADR-0022; AC-2/009. | cli/domain_skill_routing_test.go: TestCanonicalSkillRoutingTableConvergesWithInspection, TestRoutingAloneNeverGrantsAuthority; codexruntime/domain_skills_upgrade_test.go |
| I276-T36 | NÃO PROVADO | T34, T35, T44 | AC-1–5; AC-009. | Service/fake-adapter and local publication tests exist; no complete installed stage blackbox suite located |
| I276-T37 | PARCIAL | T09, T01, T21, T26, T31 | CR-006; ADR-0026 D6; PM-11. | TestAdmissionHistoricalMatrix, TestUnknownWireNeverFallsBack, TestSuccessorHistoricalBytesAndDigests, TestStableCorpusResolvesToDirectUpgrade |
| I276-T38 | PARCIAL | T15, T16, T18, T19, T23, T41, T43 | Plan §7/9.1; CR-005/007/008. | TestConfinementControlMatrix, TestExtendedMissingAuthority, TestExtendedResidue, TestStageRejectsConfiguredArguments, TestIngestRejectsCandidatesAndReportsSafeCategories |
| I276-T39 | PARCIAL | T36, T37, T38 | Plan §13; documentation governance. | This dated reconciliation; repository/ADR checks V3; Task/DAG checks V4 |
| I276-T40 | GATE HUMANO / NOT_RUN | T39, T16 | Plan §9.4; AC-1/3/6. | NOT_RUN; no real Runtime, inference, probe or Provider effect executed |
| I276-T41 | PARCIAL | T16 | PD-17; CR-005. | extended_test.go: TestExtendedAuthorizedFakeRoots, TestExtendedMissingAuthority, TestExtendedResidue, TestExtendedForbiddenRoots |
| I276-T42 | INTEGRADO | T11, T10 | PD-13; amendment A1. | service_test.go: TestShowStatesAndProfileRemoval, TestInvalidateAuthorityAndPreview, TestInvalidationPreviewFreezesProfile |
| I276-T43 | PARCIAL | T05 | PD-2/6; AC-1/5. | process_posix_test.go: TestPOSIXProcessContainment, TestPOSIXAlreadyExitedGroup; process_test.go |
| I276-T44 | PENDENTE | T16, T41, T42 | PD-1/12/14/17. | No capability command/blackbox suite located; application service tests belong to T16/T42 |

### Recoverable execution sequence

Each row is a checkpoint inside this one PR, not a new Task or a change to the
approved DAG. Integrations and pushes are serialized. Read relevant contracts
only as each unit is entered. Never use child claims, fake fixtures or the empty
passive registry to create operational capability proof.

| Unit / criteria | Prerequisites | Files / interfaces | Validation / output | Risk / stop condition |
| --- | --- | --- | --- | --- |
| T43: descendants and truthful stop | T05 integrated | executiongraph process boundary + OSProcessRunner | native fake-process exit/timeout/cancel/escape tests, race; native Linux/macOS CI | unobservable escape stays unknown; no Windows; no attribution-based kill outside owned group |
| T41 → T44: extended effects and capability CLI | T16 ports in PR, T42 integrated | capabilityproof.ExtendedEffects; local owned file operations; cmd/lingo composition; internal/cli help/result | fake temp roots; exact cleanup/residue/identity; CLI no-authority zero-writes; preparation separate from probe | unsafe root or incomplete identity/attestation refuses; no real vendor call |
| T17/T21/T25 → T27/T26: graph and single composition | T43, T07/T09/T18–24/T32 | stageexecution preview/run/start, stagepublication LocalPublisher.Single, CommittedStarter.Sequential, cmd/lingo supervisor | exact frozen-v3 preview; readback-before-spawn; both topologies; protected failure windows | no spawn before authority commit; missing proof fails closed |
| T28/T29/T30/T31 → T34 | T26/T27; T06/T08 | persisted status/resume/cancel/reconcile/retry, gitworkspace fencing, stage/coordination CLI | restart, confirmed cancel, append-only facts, authority/lease/epoch/worktree checks at every reset effect | reset remains refused unless full safe destructive protocol is proved; no counter reset or automatic retry |
| T36/T37/T38 → T39 | T34/T44 + required catalog predecessors | installed CLI fake suite, immutable corpora, commands/help, this record | graph/single; nine kinds; cross-Runtime Q/A consumed; overlap/dependencies; kill windows; 15 historical + v3 rows; consolidated security report | never weaken gate or rewrite historical fixture to pass; unresolved Major/Blocker prevents readiness claim |
| T40 future operational gate | T39/T16, fresh independent authorization | future engineering_smoke_real package | exact preconditions/envelope checklist only in this implementation | NOT_RUN; no vendor version/help/auth/probe/inference, Provider effect, real smoke, merge, release or Done |

AC-1–5 and WF-009/AC-009 retain component synthetic Evidence only; the installed
end-to-end technical matrix is not yet proved. AC-1–3 real-runtime implications
and AC-6 remain operational gates. T40's deferral does not satisfy its Done or
AC-6. Continue the existing units without requesting generic reapproval.

### T43 checkpoint — natural leader exit does not prove descendant completion

`OSProcessRunner.RunObserved` now observes the process group after a natural
leader exit. If members remain, an escape is observed, or observation fails,
it requests the existing bounded group stop and records unknown/ambiguous
effects. Exit 0 and a pending cancel cannot override those effects. A clean
natural exit retains the prior `Stopped=false` semantics (no forced stop);
`Stopped` is not repurposed as a generic completion flag. Existing timeout,
cancel and correlation-publication-failure paths still require confirmed stop.
An unstarted POSIX group refuses both stop and gone checks before any signal,
preventing PID zero from addressing the caller's own group. Tests now release
registered group handles explicitly.

Reuse decision: stdlib `os/exec`, existing process observation, POSIX groups and
`x/sys` suffice for this bounded correction. No dependency, wrapper protocol,
privileged observer, broader signal authority, vendor invocation or schema
change was introduced. The existing advisory containment boundary is retained.

Regression proof: the new `TestOSProcessRunnerStopsDescendants/completion`
was run against the baseline scheduler from `e05e209` while preserving/restoring
the worktree implementation. `go test ./internal/executiongraph -run
'^TestOSProcessRunnerStopsDescendants$/^completion$' -count=1 -timeout=30s`
returned **exit 1**, showing exit 0, `Stopped=false`, `OutcomeUnknown=false`
while the descendant was alive. Test cleanup stopped only that fixture's
identity-matched descendant. The corrected test is green and separately checks
that cancel cannot promote the ambiguous result.

New native-process tests cover normal-leader exit with a surviving group member,
timeout after observed startup, explicit cancellation, failed correlation
publication, clean natural exit 0/7, and an unstarted group. The Linux-only
`TestOSProcessRunnerOrphanedEscapeIsUnknown` exercises an escaped double-fork
through the actual runner: the escape survives until explicit fixture cleanup,
`Stopped=false`, unknown/ambiguous effects, and unknown classification. These
are test-binary executables, not Codex/Claude or installed Runtime probes.

**T43 remains PARTIAL.** Native macOS validation is required in this PR's CI,
and the existing `TestPOSIXProcessContainment/orphaned escaped descendant`
still skips on macOS: Darwin has no Linux child subreaper, and the current
ancestry snapshot cannot attribute an already reparented escape. The new group
completion fix does not close that observation gap and cannot prove the full
T43 descendant criterion on macOS. This is a material remaining technical gap,
not a request to waive safety or a claim of platform-complete stop. A verified
platform tracking protocol (or a fail-closed unsupported/unknown boundary when
tracking is insufficient) is still needed before T43/T29/T38 can be complete.
Do not authorize signals to unattributable processes or treat group disappearance
alone as proof against arbitrary daemonization. Windows remains outside scope.

Validation on Linux, pinned cached Go 1.26.0, `GOTOOLCHAIN=local GOWORK=off
GOFLAGS=`:

| Command | Actual result |
| --- | --- |
| `go test -race ./internal/executiongraph ./internal/stageexecution ./internal/capabilityproof ./internal/runtimeadapter -count=1 -timeout=180s` | exit 0; 21.103s / 19.426s / 1.807s / 5.941s |
| `go test -race ./internal/executiongraph -run 'TestOSProcessRunnerStopsDescendants\|TestOSProcessRunnerNaturalExitPreservesOutcome\|TestPOSIXUnstartedGroupRefusesStop' -count=1 -timeout=60s` | exit 0; 8.500s; includes final ambiguous-effect/cancel assertion and clean-exit regression |
| `go test -race ./internal/executiongraph -run '^TestOSProcessRunnerOrphanedEscapeIsUnknown$' -count=1 -timeout=30s` | exit 0; 4.125s |
| `go test -race ./cmd/lingo -run '^TestPreparedProductionProbe\|^TestProbeOpenFailure\|^TestStage' -count=1 -timeout=180s` with private `AXIOM_TEST_PROBE_STATE_ROOT` under `/home/box` | exit 0; 79.276s; production probe, forged-report and stage regressions, FAKE executables only |
| `go vet ./...`; `go build ./...`; `go mod verify` | exit 0 |
| `./scripts/check-go-quality.sh format`, `tidy`, `staticcheck` | exit 0; Staticcheck 2026.2.1 (v0.8.1) |
| `./scripts/validate-repository.sh .` | exit 0; native vendor behavioral scenarios SKIP/UNVERIFIED as required |

An initial intermediate implementation marked clean natural exits `Stopped`;
the capability regression correctly failed because that flag means interrupted
probe/quarantine. It was corrected before publication, preserving clean-exit
semantics. That failed intermediate run is not reported as PASS. The final
focused clean-exit and failed-probe regressions passed. Full exact-head native
Linux/macOS CI remains a separate observation after push, never inferred from
these local runs. No real version/help/authentication, probe, inference, smoke
or Provider mutation was executed. T40/AC-6 remain NOT_RUN/unproved.

Source SHA-256 at this checkpoint:

- `internal/executiongraph/scheduler.go`: `cd2cbb4390c664739503735a6f6f3353f1dd8f9a4279d20eb8c2109d544793ed`
- `internal/executiongraph/process_posix.go`: `b60249add3c5121fc0b38e3107979fa457e5c7e1bfcf31b529deeee849426c68`
- `internal/executiongraph/process_posix_test.go`: `09d2873477ab79777122f909967b359e6c85ad3dcd454aa933f20b84422f810c`
- `internal/executiongraph/process_runner_linux_test.go`: `54bfebf7d716c156430b9d9c3701f280d37cf545dd7b3758a9a37219ad7cf3d2`

Engineering/security self-review used `axiom-review` and documentation
reconciliation used `axiom-document`: no blocking finding in this bounded
correction; the material Darwin observation gap remains explicitly open for the
whole delivery. Existing CR-001/CR-002 fixes and their FAKE regression were
preserved. No accepted ADR, historical Plan/catalog suffix, Graph1/2/corpus,
capability facts, authority contract or runtime policy was weakened. No new
narrative code comments, TODO/FIXME, permissions or unrelated changes.
Dedicated gitleaks is unavailable; no dedicated scanner PASS is claimed.

Inventory check: an ephemeral read-only Python check compared each of the 44
unique continuation rows with the approved catalog's exact predecessor list and
performed a DFS over all nodes: PASS, exit 0, acyclic. No durable helper added.

### T41 checkpoint — digest of the read-only extended preview

`ExtendedService.Preview` now returns a SHA-256 digest over the exact probe ID,
ordered roots and their physical snapshots, unique filename, one-file limit and
timeout. The closed projection uses stdlib JSON of `{Encoding, Preview}` with
encoding `axiom-extended-probe-preview-v1` and the preview's Digest empty; the
existing `packetDigest` supplies SHA-256. No raw paths are added to persisted
Evidence; paths remain in the ephemeral local preview. This does not hash or
attest an operational Runtime invocation: production extended adapter composition
is still missing. The standard `ProbeAuthority` cannot substitute for the
separate `ExtendedAuthority` type.

Apply still re-observes the exact preview before the first effect. A missing,
changed or stale digest, root identity, filename, timeout, limit or root set
refuses without controls, canary or cleanup calls. Tests cover 17 mutation
cases, repeat determinism, and independently changed probe/path/inode/root-set
digests, in addition to existing missing authority, forbidden home/vendor/
external roots, control failure and residue cases. All effects are fake; no
shared system temp directory or real Runtime was probed.

Validation (Go 1.26.0, same pinned environment as T43):

| Command | Actual result |
| --- | --- |
| `go test ./internal/capabilityproof -run '^TestExtended' -count=1 -timeout=30s` | exit 0, 0.017s before the additional digest-source tests |
| `go test -race ./internal/capabilityproof -count=1 -timeout=90s` | exit 0, 1.211s, including all final extended tests |
| `go test ./internal/runtimeadapter ./internal/stageexecution -count=1 -timeout=90s` | exit 0, 0.726s / 2.738s |
| `go vet ./internal/capabilityproof`; `go build ./internal/capabilityproof` | exit 0 |
| `./scripts/check-go-quality.sh format`, `tidy`, `staticcheck` | exit 0 |

Source SHA-256: `extended.go`
`1e78f761e822436bcc9798ad465beaa0932c44feca2ab7aad37cd96d63ffb8ef`;
`extended_test.go`
`34c45e1a6d41a5ac188ac7e5b14ce111d7d87a0c58ab17034d153ce8372eb7c9`.
Engineering/security review: no new dependency, effect authority, filesystem
operation or capability promotion. This is an ephemeral preview API extension,
not a persisted schema migration. Existing Graph/corpus/ADRs are untouched.
**T41 remains PARTIAL:** production identity-bound ExtendedEffects with exact
owned-file cleanup, dispatch-identical invocation/auth/binding and independent
measurement, plus T44 routing, still need implementation and installed tests.
Do not wire a child-controlled errno/claim as independent measurement. These
checks neither satisfy operational proof nor make the delivery ready for review.

### T40 preparation only — future authorization and Evidence checklist

Status: **NOT_RUN**. This is a checklist, not an `engineering_smoke_real`
Evidence package and not an authorized command sequence. The maintainer must
grant new specific authority before any real vendor version/help/authentication,
probe, inference or installed smoke is invoked. Preparation consent never
implies probe consent; extended-temp consent is separate from standard consent.

Before proposing a future run, verify all of Plan §9.4's five preconditions:

1. IG-1: supported canonical v3 authoring creates the clean configuration, with
   no hand-edited state. Confirm the installed Axiom binary's exact version,
   commit and SHA-256, and the same reviewed CLI/skill contracts.
2. Independently observed version identities must be admissible through the
   accepted passive registry. Current empty registry does not meet this gate.
   Fresh separately authorized capability probes must produce matching proven
   Evidence for each exact child binding/variant, with inspected Profile/effort
   sources. Current production verification's unknown results do not meet it.
3. Both vendors must already be authenticated using subscription mode. Any
   future `codex login status` / `claude auth status --json` observation needs
   its own granted preflight envelope here; no API keys, environment bypass,
   child claims or fixture auth can supply proof.
4. A human must approve the exact run envelope: scratch lab repository (never
   Axiom's repository), canonical workflow digest/stage, Graph3 and parent/child
   identities, Runtime/Profile/model/effort, permitted read/worktree-write/process
   effects, validator-policy digest, maximum attempts 1, per-child timeout,
   total bound, coordination budget, private Evidence directory, approver/time,
   and exact preview digest/authorities. Name residual confinement/cost risks,
   cleanup/quarantine and uncertain-stop recovery before consent.
5. Complete I2–I6 and the applicable delivered IG-7/#276 v3/Graph3 admission and
   composition checks; T36/T37/T38/T39 technical Evidence and independent review
   must cover the actual installed artifacts. No green component CI replaces
   these missing installed-journey checks.

If any prerequisite is absent, retain NOT_RUN and identify it. For a separately
authorized run, the eventual redacted `engineering_smoke_real` record must include
exact executable/version/mode/capability/Profile/effort/policy digests, subscription
observation, confinement level and scope, exact graph/authority and packet
correlation, bounded streams and attempt metadata, and all Plan §9.4 checks:
Codex question_request, Claude answer with question_reference, both result
records consumed by the orchestrator/integration owner, authorized concurrency,
local integration/validator outputs, persisted status/reason/readiness before
and after initiating-session exit, and start from one supported Runtime's skill
with inspection/integration from the other. Preserve failures, uncertainty,
stop confirmation and cleanup/residue, not only successes. Retain no account
IDs, credentials or raw transcripts. This cannot claim R-2/R-3, #278, release,
AXM-8 Done or operational acceptance. T40 execution and AC-6 remain open.

### Acceptance at this continuation checkpoint

| Criterion | Synthetic Evidence now | Still open |
| --- | --- | --- |
| AC-1 | Existing packet/argv/auth/environment and graph/single service components; production T13/T14/T16 FAKE matrix; new descendant boundary tests | Public installed graph AND single FAKE start, then separately authorized real configured Codex+Claude journey |
| AC-2 | Persisted lease/status/resume and thin skill routes; detached-supervisor component tests | Complete installed cross-surface restart/status/resume and native initiating-session independence |
| AC-3 | Nine-kind inbox/ingestion, lineage/revision/epoch/capacity, synthetic Q/A components | Installed orchestrator consumes cross-Runtime FAKE Q/A; full negative matrix; later separately authorized real exchange |
| AC-4 | Scheduler conflict/overlap and validator-bound integration/roll-up components | Installed concurrency/dependency/effect and integration journey under final composition |
| AC-5 | Classifier/reconciliation/retry and kill-window components; new exit/timeout/cancel/escaped-descendant regressions | Installed recovery; Darwin reparented escape gap; destructive retry-reset fencing remains unavailable/fail-closed |
| AC-6 | No operational Evidence | T40 NOT_RUN; future exact-envelope authorization and all real smoke fields/checks |
| WF-009 / AC-009 | Stage/coordination parser, T35 routing and persisted state components | Complete CLI/Codex/Claude installed parity, graph/single lifecycle, public coordination and recovery journeys |

The Task inventory above remains current: T43 and T41 gained bounded unmerged
implementation/Evidence but stay PARTIAL. T13/T14/T16 are preserved in the same
Draft PR, not reimplemented. T44 is still PENDING; T36 is not proved; T37–T39
are partial; no technical-delivery or human acceptance claim is made.

### HANDOFF — continue the same AXM-8 technical delivery

This is a recoverable session checkpoint, **not Done or technical delivery ready
for review**. Do not restart SDD or request a new prompt per Task. Preserve the
same objective and authorization from the single continuation request. Context
was bounded after reconnaissance and two safe implementation units; the next
production boundary should be entered with its full relevant contracts available.

```yaml
objective: Complete remaining authorized technical I276-T01–T44 in Draft PR 320
state: in_progress
branch: feat/axm-8-stage-agent-execution
source_head: aa92e330d90bff360c55139493fb0ba316f5cd59
source_head_note: This record is in the documentation-only successor; code equals source_head. Read exact published HEAD and CI from PR 320, then verify git status/rev-parse/ls-remote before mutation.
source_commits:
  - dbfe896f80de91e449a42265dbeb5ea165241648: T43 natural-exit descendants correction and 44-Task inventory
  - aa92e330d90bff360c55139493fb0ba316f5cd59: T41 extended preview digest and synthetic refusals
completed_units:
  - Revalidated sole work PR, main, HEAD, AXM-8 relations and three human decisions
  - Compared all 44 Tasks with main and PR; catalog predecessor and DAG check passed
  - T43 bounded correction with regression failure against baseline and race checks
  - T41 digest-bound read-only extended preview with 17 mutation refusals and digest-source tests
  - T39 checkpoint reconciliation and T40 future prerequisite checklist only
remaining_units:
  - T43 Darwin reparented-escape observation or an honest fail-closed boundary; native test cannot be skipped as proof
  - T41 production ExtendedEffects and exact invocation/binding/authentication/cleanup; fake-root positive/negative tests
  - T44 public capability show/probe/invalidate plus independently authorized preparation and distinct extended operation
  - T17/T21/T25 production preview and immediate revalidation; T26 graph AND single wiring with T27 supervision
  - T28/T29/T30/T31 lifecycle and safe reset authority/lease/epoch/worktree fencing
  - T34 installed stage/coordination parity; preserve T35 thin routes and T32/T33 validators
  - T36 installed fake-Runtime graph/single and Q/A consumed by orchestrator, restart, overlap/conflicts and kill windows
  - T37 compatibility matrix and immutable corpus hashes; T38 consolidated security regressions; final T39
  - Final exact-head Linux/macOS CI and independent review, with findings resolved in this same PR
next_action: Implement T41 production extended-effects composition against the existing port, with exact trusted root/file identities and separate invocation/authority preview. Keep native roots unprobed; test on private synthetic roots with FAKE executables. T43 Darwin is a separate material open technical node, not a reason to block independent T41/T44 work.
next_files: internal/capabilityproof/extended.go; internal/local native extended-effects boundary; cmd/lingo capability composition and its tests. New filenames are a plan, not existing implementation.
next_acceptance: Preview zero writes; missing authorities and stale digest/binding/root refuse before effects; one exclusive empty filename per root; cleanup only the exact owned inode; residue fails; child claims never prove invocation or denial; no promotion when independent observation is absent.
authority_granted: Scoped #276 code/docs, deterministic and synthetic tests, cohesive commits, push existing branch, update same PR
pending_authority: All real Runtime version/help/authentication/probe/inference/smoke effects, T40, Provider effects, merge, release, final acceptance
validation_status: Focused race/regressions, vet/build/module/format/tidy/staticcheck/repository checks passed as recorded above; exact remote CI is SHA-specific and must be read from PR 320. Vendor behavioral checks deliberately UNVERIFIED.
review_status: Earlier CR-001/CR-002 resolved; favorable COMMENT at e05e209 covers its slice only. New checkpoint self-review is not independent approval. Whole delivery remains incomplete.
```

Before a substantial process-tracking or extended-probe mechanism, compare the
stdlib and existing `x/sys`/owned-filesystem primitives. Do not invent a new
trust path or adopt another dependency implicitly. Stop only an effect whose
safe authority/identity/measurement cannot be established; preserve all
independent work. No current human action is required to continue implementation.
Future operational tests need the specific independent consent listed above.

### Continuation — Darwin orphan boundary (T43)

The `82e711e` checkpoint is preserved. PR #320 was re-observed open/Draft on
the same branch, with remote HEAD `82e711eaf85fd01f024ea94b4348b48bee58fd50`;
its CI run `38818347537` passed native Linux/macOS. AXM-8 remains In Progress,
with no blockedBy relation. The three specified human decisions were read in
full; no Provider content was imported. No delegation: the required strictly
lower model hierarchy is not established by the available runtime metadata.

T43 now takes a pre-spawn Darwin launchd-adoption baseline using the existing
`x/sys/unix` sysctl boundary. It compares PID **and start identity** after exit
and before stop confirmation. A new live adopted process outside known attempt
groups is un-attributable: stop stays unconfirmed and the runner remains unknown.
This deliberately permits conservative false negatives when an unrelated process
daemonizes concurrently; it grants no sandbox or arbitrary-descendant kill power.
No new dependency or accepted ADR change. Linux retains its subreaper boundary.
The previous Darwin orphan-test skip was removed and the runner's orphan test
is shared by Linux and Darwin. Native macOS execution must be observed in CI;
cross-compilation alone is not native proof.

Local synthetic validation, pinned Go 1.26.0, ambient Claude effort unset:

| Command | Result |
| --- | --- |
| `go test -race ./internal/executiongraph -run 'TestPOSIX\|TestOSProcessRunner' -count=1 -timeout=120s` | exit 0, 22.128s |
| `go test ./internal/executiongraph ./internal/stageexecution ./internal/capabilityproof -count=1 -timeout=120s` | exit 0, 12.690s / 2.886s / 0.089s |
| `GOOS=darwin GOARCH=arm64 go test -c ./internal/executiongraph` | exit 0, compilation only |
| `go vet ./internal/executiongraph`; format / tidy / staticcheck | exit 0; Staticcheck 0.8.1 |

Staticcheck initially refused because its installed binary was absent from PATH;
adding the existing `/home/box/go/bin` resolved it, without installing anything.
No actual vendor executable, authentication, probe or inference was invoked.
T40 and AC-6 remain NOT_RUN. This unit does not close the remaining T41/T44,
stage-composition, retry-reset or installed-validation gaps listed above.

CI `38823438873` at `13412e0` **FAILED** on native macOS: the orphan regression
ran, but the baseline wrongly included only processes already parented by
launchd. Pre-existing unrelated processes adopted later made clean exits and
confirmed group-stop tests inconclusive. The correction snapshots **all** live
pre-spawn process identities, while subsequent observations consider adopted
processes only. No outcome assertion or security gate was weakened. Linux
focused race regression repeated: exit 0, 22.140s; Darwin arm64 compilation:
exit 0. Native CI for the correction is still required. An unrelated process
created and daemonized after spawn can still conservatively prevent confirmation.

### T44 incremental CLI — show and invalidation

Both `runtime codex capability` and `runtime claude capability` now route `show`
and `invalidate` through the canonical T42 services and production BindingSource.
The CLI has no evidence-selection or capability rules. Exact Profile, variant,
process mode and operation roots select the binding; local consent gates effort
inspection. Invalidation previews bind the entire domain preview by SHA-256;
apply checks local authority, exact digest and the domain's fresh CAS preview.
Unknown bindings remain unknown. No executable, auth command or Provider is run.
Commands/help use the same flag registry; human/JSON share the payload. No
capability route was added to a Runtime skill, per PD-14.

Synthetic tests cover both command paths, parity/help, malformed/duplicate and
foreign-authority flags, unknown-state zero-write refusal and an actual canonical
Evidence-store show/invalidate/show round trip under injected version/Evidence
fixtures. That fixture is explicitly not an operational proof. Focused race
checks (`go test -race ./internal/cli ./cmd/lingo -run
'TestRuntimeCapability|TestCapabilityInvalidation|TestCapabilityPublic|TestCapabilityExcluded'
-count=1 -timeout=60s`) passed, exit 0 (1.124s / 1.152s).
T44 remains partial until standard probe/preparation and distinct extended
operation are composed and validated; T41 and installed T36 are still open.

Native macOS CI at `4f841ab` (`38824648854`) also failed: new launchd services
outside the operator identity can appear during probes/timeouts. The observer
now scopes adoption to matching real, saved **or** effective operator UID;
an effective-root supervisor still observes all identities. Setuid execution
retains the real operator UID and remains included. This is advisory process
tracking for the non-elevated Runtime contract, never a privilege/confinement
claim. New unrelated processes with the operator's identity still conservatively
prevent confirmation. No tests or result gates changed. Darwin arm64 test
compilation passed (exit 0); native CI remains required.
