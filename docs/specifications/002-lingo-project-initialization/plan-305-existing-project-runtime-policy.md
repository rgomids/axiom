# AXM-34 / G-1 — Existing Project Runtime/Profile policy Plan

Status: **Proposed — Awaiting Human Approval — Plan**, revision 1, 2026-10-18.
SDD stage: **plan**. Technical owner: [#305](https://github.com/rgomids/axiom/issues/305),
parent [#303](https://github.com/rgomids/axiom/issues/303); operational record:
[AXM-34](https://linear.app/rgomids/issue/AXM-34).
Canonical delivery: [Draft PR #327](https://github.com/rgomids/axiom/pull/327), branch
`docs/axm-34-project-runtime-policy-specification`. This Plan proposes future
implementation; it creates no executable Tasks and grants no implementation,
Execution, Provider, merge, release, MVP/R-2/R-3 or product acceptance authority.

## 1. Contract, discovery and continuity

Input: [approved G-1 amendment](amendment-305-existing-project-runtime-policy.md),
G1-AC01–G1-AC15. [Maintainer approval](https://github.com/rgomids/axiom/pull/327#issuecomment-6245855021)
applies to specify bytes at exact SHA `6a5b00e43e7a575bfde4e25d2a3110602d9a414c`;
[technical review](https://github.com/rgomids/axiom/pull/327#pullrequestreview-5561476866)
is favorable, with no findings, and is distinct from human acceptance.
This change reconciles status/provenance only in that amendment, including
labeling its former pending gate as historical; behavioral requirements remain
unchanged. Historical CREATE-only contracts are not rewritten.

Discovery on 2026-10-18, before writing:

- Clean local branch, HEAD and `git ls-remote` branch head all equal the approved
  SHA. PR #327 is open/Draft; its complete diff is the amendment plus the
  Specifications index. Remote `PR Checks` and `delivery-metadata`: SUCCESS.
- Read #305 body/comments, #327 body/diff/reviews/approval and open PR inventory:
  #327 is the sole open AXM-34 PR; #317/#328 concern different work. Read Linear
  issue/relations/comments: operational state In Review and plan handoff refer
  back to #327. No Provider state was changed. Private Linear narrative was not
  copied into this public Plan; requirements derive from public GitHub/contracts.
- Repository search, branch history, Spec 002's original Plan/Tasks/Evidence and
  #306's separate Plan/Tasks/Evidence found no G-1 Plan to resume. Reuse #327;
  do not combine G-2 or recreate its delivered local authoring work.
- Canonical Axiom Execution/current gate/revision: **unknown**. No canonical
  endpoint is exposed in this session and no local `.axiom` work ledger exists
  in this checkout; this is not evidence that no Execution exists. No product
  invocation, installation discovery or Execution reconstruction was performed.
  Provider plan notification is a handoff, not canonical Execution evidence.
  Before any future workflow transition, consult exact canonical state; reconcile
  a competing Plan/Execution/PR rather than proceeding concurrently.

Preserved contracts: [Spec 002 FR-018/019, AC-15/16](spec.md),
[policy v2/#140](runtime-policy-v2.md), [bootstrap v3/#231](project-bootstrap-v3.md),
[Spec 007](../007-configurable-workflows/spec.md),
[HD-006 amendment](../007-configurable-workflows/amendment-303-workflow-skill.md),
[ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md),
[ADR-0020](../../decisions/0020-workflow-definition-revision-binding.md),
[ADR-0024](../../decisions/0024-model-profile-effort-ownership.md),
[ADR-0005](../../decisions/0005-bounded-local-filesystem-threat-model.md),
[ADR-0007](../../decisions/0007-local-publication-and-recovery-protocol.md) and
[#230 lifecycle matrix, including delivered reconciliation](../004-mvp-v1-baseline/issue-230-resource-lifecycle-matrix.md).
Governance: [AGENTS](../../../AGENTS.md), [constitution](../../product/constitution.md),
[tracking](../../development/linear-github-issue-tracking.md),
[documentation](../../documentation.md), [security](../../../.agents/policies/security.md).

## 2. Existing ownership and smallest extension

Paths/symbols below were inspected at the approved SHA; proposed changes are
future design, not claims that G-1 interfaces already exist.

| Owner / existing extension point | Current behavior and planned extension |
| --- | --- |
| Domain: `internal/project/project.go`, `runtime_policy.go` | `Declaration`, `Change`, `Intent`, `Project.Propose`, `Equivalent` and full validation already carry policy fields. Merge member intent in application, then submit complete declarations to `Propose`; retain sole domain validator and identity/schema invariants. |
| Strict manifest: `internal/manifest/{codec,schema_v2,mapping,output,security}.go` | Decode/encode complete Projects, enforce closed shapes and canonical ordering. Reuse unchanged portable shape and parser limits (256 KiB, depth 32, 16,384 nodes); no raw YAML patch or alternate serializer. |
| Application: `internal/projectapp/edit.go`, `edit_publish.go`, `snapshot.go`, `ports.go` | `EditIntent`, `PreviewEdit`, `EditProposal`, `ApplyEdit`, `EditAuthority`, `EditPublisher` already merge partial intent and rebuild on replay. Extend these with policy intent, safe policy projection, read dependencies, compatibility/readiness and read-back. Existing publication request only compares manifest/local bytes: G-1 must add the consulted dependencies at commit. |
| Lingo composition: `cmd/lingo/main.go` | `configureEdit`, `projectEdit`, `editSource.SelectForEdit` resolve installed UUID/slug, read recorded source, apply `AdmitProjectEdit` and compose adapters. Extend this route; never use effective CWD selection for G-1. |
| CLI: `internal/cli/cli.go`; conversation: embedded `internal/codexruntime/skills/axiom-project/SKILL.md`, `internal/cli/skill_inspect.go` | `project configure` already has CREATE/EDIT; current policy flags are CREATE-only. Extend EDIT input translation and inspection/routing, keeping application authority. New draft transport is proposed in §3, not an existing flag/API. No second command family or runtime-specific maintainer policy. |
| Installation/publication: `internal/local/{edit_adapters,edit_publication,edit_recovery,publication,safe_fs}.go` | Recorded-source read, local codec, `ProjectEditPublisher`, portable/local CAS, owned recovery state and confirmed-effect classification exist. Reuse these with additional guarded dependencies; keep source confinement and read-only planning. |
| Operational state: `internal/projectapp/{admission,project_state}.go`, `internal/local/operational_store.go` | Read/bind existing archive/disabled state and recheck `AdmitProjectEdit`; keep administrative classification. Do not invent an execution-readiness admission requirement or silently reactivate a Project. |
| Workflow: `internal/projectapp/{active_workflow,workflow_authoring,workflow_profiles_v3}.go`, `internal/local/workflow_definitions.go`, `cmd/lingo/active_workflow.go` | `ResolveActiveWorkflow`, strict catalog/index and exact definitions provide identity resolution. Extract reusable portable compatibility checks from authoring. `ValidateWorkflowProfilesV3` depends on local Profiles and cannot be the sole portable-save gate. |
| Readiness/resolution: `internal/projectapp/readiness.go`, `cmd/lingo/{readiness,runtime_policy}.go`, `internal/runtimeapplication/{policy,policy_v3}.go`, `internal/runtimeprofile` | Reuse `ProjectReadiness`, `EvaluateReadiness`, `EvaluatePolicyV3` and existing resolver/preference precedence; add proposed-snapshot input composition, not a Doctor. Keep local format-3 consumer and historical format behavior. |
| Immutable consumers: `internal/workflowcompiler/compiler.go`, `internal/graphapplication/runtime_policy.go` | Read adjacent selection/admission semantics for impact and regression assertions; G-1 does not edit compiler, Execution/graph formats or dispatch logic. |
| Compatibility: `internal/compatibility/testdata/stable-v1`, Project/manifest/local/CLI test suites | Extend synthetic forward coverage without rewriting frozen historical fixtures or weakening existing CREATE/unrelated EDIT contracts. |

Code also recognizes portable schema 4 (`MultiRuntimeSchema`). The G-1 contract
is explicitly v2/v3: gate G-1 policy intent to exactly those two schemas, refuse
v1 and any other version before effects. Do not narrow the shared helper or
break unrelated supported schema-4 operations. Extending G-1 to v4 requires a
separate reviewed contract; schema-4 support elsewhere is not such authority.

## 3. Partial draft and canonical data strategy

Recommended design: a bounded, in-memory policy edit added to `EditIntent`,
with one optional operation group per collection. No permanent draft registry,
new manifest fields, migration, local Profile authoring or stored approval.

Proposed adapter transport: one strict, bounded JSON draft document supplied to
the existing configure/edit surface. This is a future input contract, not an
existing CLI option. Reuse a repository strict JSON primitive after reviewing
its duplicate-field/depth/size behavior; cap the draft at manifest byte/depth/node
limits and operations at 16 Runtime / 64 Profile / 64 preference member operations
(two operations per bounded current/result identity set). Reject excess rather
than truncate. Final option spelling/help is a later adapter detail; Tasks must
record it and the inspection metadata together, without inventing a command here.

Closed draft shape: only `runtimes`, `modelProfiles`, `runtimePreferences` groups;
each supplied group has exactly one mode, `absent`, `unconfigured`, `replace`,
or `members`. `replace` carries `values` in the existing portable member shape
(including `[]`); `members` carries `add`, `replace`, `remove` lists. For Runtime
and Profile removals use exact `id`/`key`; preferences use exact role/complexity
pairs. Added/replaced members use existing complete member shapes. No authority,
paths, approval booleans, schema or Project identity fields in this draft.
Omitted group retains the declaration. Group mode and list payloads are mutually
exclusive except the three member lists of `members`; duplicate keys and unknown
fields refuse. Reject mixing policy intent with unrelated EDIT changes in one
G-1 request; perform those through their existing separately reviewed journey.

Algorithm, pure application transformation:

1. Read one coherent installed snapshot; clone complete domain state including
   declaration forms. Never seed from available local candidates.
2. Validate operation identities/conflicts before merge: add needs absence;
   replace/remove needs existence; reject repeated targets across or within lists.
   Key rename is explicit remove + add and explicit related reference changes.
3. Apply only supplied groups. `members` on absent/unconfigured collection may
   add explicit members starting from no entries; replace/remove still require an
   existing target. Last member removal results in explicit configured `[]`,
   never absence/unconfigured. Collection-form changes require their own mode.
4. Canonically sort Runtime ID, Profile key and preference role/complexity, with
   exact case/bytes and no trimming/defaulting/ranking. Validate the whole result
   using `Project.Propose`: 8/32/32 final bounds, unique IDs/keys/pairs, associations,
   safe tokens/models, configured vs unconfigured shape. Runtime removal retaining
   configured Profiles and Profile removal retaining preferences refuse. Explain
   explicit dependent edits; no cascade or alternate association.
5. Compare canonical before/after policy plus complete domain state. Diff is a
   semantic list of collection presence/member changes from the recorded manifest,
   not local registry differences. Include removed/replaced values after safety
   projection. Show separate write effects. Encode the complete candidate through
   the canonical manifest codec; decode and validate it again. Assert all unrelated
   fields/forms equal, including schema, UUID, slug/name, context and selection.
6. Preserve companion documents/Workflow bytes. `PreviewEdit` currently reconstructs
   a changed snapshot with `ReadSnapshot(..., nil)`; carry the observed documents
   instead for G-1 so declared documents and their digests survive. Verify existing
   artifact-digest semantics before changing bookkeeping. Never synthesize context.
7. An equal canonical intent is no-op even if original manifest formatting differs:
   preserve original bytes, installation and revision; do not use encode differences
   to justify a rewrite. No refresh of local observations/bindings on no-op.

Schema stays unchanged. No singular portable `runtime`, effort, executable,
credential reference or machine path is introduced. Multiple named Profiles for
the same Runtime/model remain valid. All declaration forms round-trip:
`absent`, `unconfigured`, configured empty sequence and configured nonempty sequence.

## 4. Canonical journey, authority and safe output

| Phase | Application behavior / allowed effects |
| --- | --- |
| Read | Explicit selector → one installed UUID; verify coherent source/local record and revision/artifact digest. Return schema, policy forms and opaque revisions/digests. Missing/ambiguous/stale/unadopted/unsafe/recovery state refuses. Recorded source remains authoritative; publisher's current Lingo-root confinement remains. |
| Draft | §3 pure merge; complete candidate and semantic diff; no storage writes, refresh, lock-file creation or implicit source adoption. |
| Validate / dry-run | Full portable validation, §5 compatibility and §6 readiness on bound snapshots. Return invalid, incompatible/unresolvable, or valid with local blockers. This is the existing preview's read-only path, not a second mutation mode. |
| Preview | Deterministic application-produced envelope binds operation/G-1 scope, UUID/source identity, actual manifest/artifact snapshot, installation wire/revision, operational state, complete candidate, diff, exact Workflow ref/index/definition or explicit none, local configuration/observation tokens and digests, diagnostics/impact, write set and outcome expectations. |
| Local authorize | Reuse `EditAuthority` replay tuple and `--project-id`, `--preview-digest`, `--authorize-local`. Expected portable/local revisions are inside the sealed envelope and CAS request; digest cannot be rebound to newer revisions. No new approval boolean or implicit authority from a draft/readiness/initiating Runtime. |
| Apply | Rebuild same intent from fresh observations using `ApplyEdit`; require equal UUID/digest; revalidate full candidate and all bound dependencies under coordinated publication protection (§7), then publish complete manifest + required installation bookkeeping only. Cancellation before commit refuses; later cancellation cannot imply rollback. |
| Read-back | Reopen canonical selected Project and evaluate `project validate` semantics on observed state; compare UUID/schema/full policy/revision/artifact digest with committed candidate and report readiness. Report no-op, confirmed, refused, committed-but-unverified, recovery or read-back drift accurately. |

Canonicalize envelope serialization and list order; exclude newly generated
wall-clock/correlation metadata from deterministic preview identity. Persisted
observations still bind their exact revision/content digest; newly collected
read-only facts bind stable identity/content evidence, not an invented fresh
revision on every replay. Any material fact change invalidates authority.

Keep full manifest/local bytes private in the envelope. Current `EditPreview`
exposes portable/local destinations and full manifest; **do not serialize that
legacy view unchanged for G-1**. Build a safe G-1 projection carrying policy,
logical subjects, schema and opaque digests only. Unrelated portable credential
reference declarations are preserved internally, not echoed. No raw parser,
adapter, local binding/source/executable path, environment or rejected value in
preview/error/completion/log/Evidence. Separate diagnostic scopes: validation,
workflow compatibility, publication and readiness. Preserve existing stable codes
and inner resolver reason; new bounded reason constants may be added within
these scopes without reinterpreting `runtime_policy_unavailable` or
`runtime_resolution_blocked`. Remedies name the responsible separate journey.

Rollback is a new inverse policy intent under a fresh preview and authority;
never restore a removed permission automatically. Before commit preserve prior
valid state. After confirmed portable commit keep the new generation authoritative,
classify local bookkeeping failure as partial/recovery, and inspect before retry.
Recovery reuses existing inspect/apply and positively owned generations; it is
separately authorized, cannot reconstruct from Provider/chat or delete unknown
leftovers, and must not replay obsolete permissions over a later writer.

## 5. Exact active Workflow compatibility and immutable impact

Resolve actual `workflowId`, revision, digest, source and schema from the
recorded Project selection and strict catalog. Recompute definition digest as
`ResolveActiveWorkflow` does; retain immutable built-in revision lookup and exact
Project revision/index state. No selection is explicit `none`; missing/tampered/
unresolvable selected definition blocks preview/apply. Do not replace it with
latest, default, repaired or migrated Workflow.

Extract a small pure Project/Workflow compatibility helper in `projectapp`,
reused by authoring where applicable, without changing historical authoring gates.
Evaluate before and after for every exact Stage/Agent, including retained
references, ordered by Workflow reference, Stage ID, Agent ID, Profile key/Runtime:

- Explicit `profileRef`: must name exactly one configured portable Profile with
  declared Runtime satisfying `runtimeConstraints`; preferences cannot override it.
  Portable match does not require a locally present Profile. Local model/Runtime/
  effort mismatch is a readiness blocker, not fabricated portable incompatibility.
- Historical `policy-default`: follow existing compiler/request semantics
  (`profileID` maps it to no explicit Profile); evaluate exact role/complexity
  Project preference first, otherwise the portable admissible set subject to
  Runtime constraints. Empty portable set or disallowed referenced preference is
  portable incompatibility. Do not choose the first of several Profiles: expose
  the admissible keys and defer exact executable choice to the existing resolver;
  local multiplicity blocks execution readiness, not portable storage.
- Check role/complexity/capability requirements via existing Workflow validation
  and request mapping. Where support is only declared/observable locally, absence
  or unproven capability remains readiness; never infer capability from model ID.
- Preserve Workflow schema-1 meaning and schema-2 Profile-owned effort rules.
  `policy-default` is honored only where the exact historical schema permits it;
  do not reinterpret schema-2 explicit Profile semantics or add Agent effort.

Report complete before/after compatibility and reasons, distinguishing newly
introduced and pre-existing unresolved incompatibilities; **both block apply**.
Diagnostics include exact Workflow/Stage/Agent/Profile/Runtime logical references.
Use existing Workflow parser/size/cardinality bounds; bound the aggregate report
by input traversal and output byte limits; if full analysis cannot fit, fail
closed with a fixed diagnostic, never omit blocking rows.

Preview discloses four scopes:

1. New Executions (also linked Work Items without Execution): policy and exact
   active Workflow are evaluated at future admission with fresh readiness and
   authority. Save does not admit or dispatch.
2. Existing Execution/Stage Plan/bindings: immutable Workflow, Runtime/model/
   Profile/effort/proof snapshots stay unchanged. Future resume/dispatch still
   uses its current permission/freshness checks; revocation may block it and
   cannot widen permission or rewrite historical facts.
3. Work Items: zero create/status/relation/Provider effects or automatic resume.
4. Other Projects and local registry: unchanged. A's removal does not affect B's
   permission or shared local Claude/Codex installation/Profiles.

Keep the implementation bounded: disclose semantic impact and mark existing
Execution/Work Item inventory **unknown/uninspected** rather than add an inventory
engine. No counts or completeness claim. If a later approved design consults
individual records to assert impacts/counts, those exact snapshots must enter the
envelope and commit dependencies too.

## 6. Portable validity versus machine readiness

Do not reuse CREATE's locally configured candidate requirement as a save gate.
A complete safe portable triple is sufficient for storage when Workflow-compatible
and exactly authorized. Missing/disabled/retired Profile/Runtime, wrong local model
or association, unavailable executable/auth proof, unsupported/unproven capability
and ambiguous local candidates remain execution blockers while declarations persist.
Corrupt/unreadable required snapshots, unlike observed absence, block exact preview.

Refactor composition narrowly so the proposed Project and retained installation
bindings can feed the same `EvaluateReadiness`/Runtime policy source logic used by
`project validate`, without first writing the candidate or loading old persisted
policy by UUID. Use existing resolver/`EvaluatePolicyV3` for execution candidacy;
the pure compatibility helper is portable reference validation, not a second
Runtime selector. Keep Project preference vs explicit Profile precedence identical
across preview/read-back/compiler regression cases. Operation-scoped Work Item
readiness must not inherit unrelated execution blockers.

Observe only safe existing local metadata/proofs. Inspect `runtime_policy.go`
and observation wiring to prevent active auth probes, executable launch, Provider
or network access in G-1. Unknown proof remains unknown. Reuse/factor the existing
read-only evaluator; do not call a side-effecting observer merely to improve a
preview. Do not resolve credential values or copy credential references into
outputs. Format-3 local Profile owns effort (ADR-0024); G-1 neither writes it nor
performs legacy local migration/reconfiguration.

## 7. Commit protocol, concurrency and recovery

Extend `EditPublicationRequest`/adapter with a sealed read-dependency bundle,
not a universal transaction framework. Current manifest/local CAS and replay
alone are insufficient to protect dependencies between preview and publication.
Recommended adapter mechanism:

1. Open/verify existing protected root/object identities. Acquire the existing
   broad-to-narrow state-root → projects directory → Project installation chain
   before portable namespace/source locks. Reuse the same root coordination as
   Runtime Profile/operational writers and Workflow publication. Hold exclusion
   through portable commit; no opposite order, nested self-locking `Inspect`, or
   missing-store creation. Factor lock-held readers where necessary; common root
   locking is deliberately coarse and simplest for this local administrative edit.
2. Under that protection, re-read exact installation wire, operational snapshot,
   portable manifest/companion digests, Workflow index + selected definition,
   Runtime configuration token + digest and consulted proof/observation snapshots.
   Recheck source/ancestor/leaf identity, no-follow/link count, admission, UUID,
   schema, revisions and envelope. Missing snapshots have explicit absence tokens;
   absent→present is drift. Equal counter/different content always refuses.
3. Revalidate the complete candidate and dependency bundle at protected publication
   boundaries, using existing before-commit hooks. Axiom observation writers must
   share coordination or provide a guarded revision/CAS check; if a consulted
   source cannot be guarded through commit, refuse G-1 publication rather than
   claim a freshness guarantee. Do not collect extra volatile observations that
   the read-only evaluator does not require. A lock/check coverage table is an
   explicit passage criterion of unit C below.
4. Prepare bounded complete manifest/local candidates privately. Reuse owned
   `.axiom-edit-*` state and portable publication protocol. Check full source
   generation including companions, not only `axiom.yaml`; policy edits cannot
   overwrite concurrent document/Workflow publications. No cross-filesystem
   extension: unsupported source remains refused, with separate recovery guidance.
5. Commit point remains confirmed publication of the complete portable generation;
   installation portable revision/artifact digests then advance through existing
   local CAS, preserving all unrelated bindings and operational bytes. Retire
   owned recovery state after classification. Readers see prior/new complete
   authority or `recovery_required`, never a mixed accepted installation.
6. A second writer using the same preview loses under CAS. Source relocation,
   compatible Workflow reselection, slug/UUID changes, Profile retirement/config
   changes and observation changes require fresh preview/confirmation. Do not
   rebase or update tokens under old authority. No-op still validates authority
   bindings but publishes nothing and creates no recovery artifacts.

Safe reconstruction uses only owned recovery metadata, recognized prior/new
complete generations and canonical readers. Keep uncertain post-ack/read-back
outcomes separate from known pre-commit failures; no blind apply retry. Later
writer before read-back produces explicit drift; cannot prove prior policy remains
effective. A rollback/recovery operation is outside G-1 edit authority.

ADR-0005 proof scope: supported native Linux/macOS filesystem primitives,
process concurrency, traversal/link/controlled replacement and deterministic
injected faults. No arbitrary hostile same-UID interleaving, physical power-loss
or media-durability guarantee. Unsupported filesystem/platform assumptions fail
closed; later Evidence must name OS/architecture/filesystem and primitives.

## 8. Bounded future delivery units and acceptance matrix

These are dependency/verification envelopes for a later **tasks** phase, not
executable Tasks, assignments, approval or implementation authority. Five units,
ending at G-1 read-back and routing; no automatic expansion into Doctor/G-2/G-3.

| Unit / dependency | Bounded scope | Passage criterion and future validation |
| --- | --- | --- |
| A — domain/application draft | §3 merge, strict adapter draft decoding, full manifest candidate, semantic diff/no-op and preservation; extends current EDIT only | Unit/table/property cases cover all forms, limits, conflicts, explicit references and schema gates; encode/decode complete candidate retains all unrelated state and companions. Negative draft parsing bounded and safe. |
| B — A | Pure exact Workflow compatibility + proposed-snapshot readiness composition; safe bound preview/diagnostics and immutable impact | Unit/integration cases cover built-in/Project exact revisions, schemas, explicit/default Profile precedence, Runtime constraints, retained/new incompatibilities and valid-portable/local-unavailable. Same snapshot gives same dry-run/validate result and digest; no network/Runtime/secret reads. |
| C — A+B | Guarded replay/publication and existing recovery adapter extension, read-back classification | Produce dependency→writer→lock/check coverage table including absent local config/operational state. Integration/multiprocess/fault tests prove CAS, drift at each boundary, old/new/recovery and actual commit truth; no unsupported lock order or unguarded consulted source. |
| D — A+B+C | Existing Lingo/CLI configure EDIT, strict draft transport, JSON/guided preview/confirmation/read-back; thin `axiom-project` and inspect/reference reconciliation | Black-box guided/noninteractive parity, partial replay refusal, policy-only write set, safe output, cancellation/no-op; canonical operation and conversation share same authority. Existing CREATE/unrelated EDIT remain compatible. Native conversation remains separately evidenced, not inferred from parser tests. |
| E — A+B+C+D | Compatibility corpus, security/fault closure, consolidated synthetic Evidence and documentation | All 15 rows below have linked future test/Evidence; frozen corpus unchanged; A/B/local registry/Workflow/Execution bytes compared; appropriate Go/CLI/repository gates plus native Linux/macOS storage Evidence pass, with unexecuted environments reported. No product acceptance implied. |

All test/Evidence entries below are **future required oracles — NOT RUN in plan**.
References to units identify ownership layers, not Task IDs.

| Requirement | Component / layer | Strategy | Future test / Evidence |
| --- | --- | --- | --- |
| G1-AC01 | A–D: Project domain, EDIT application, Lingo/local | Full v2/v3 canonical journey, unchanged UUID/schema | Integration + CLI black-box before/draft/preview/authorize/after/read-back and validate report, exact digests |
| G1-AC02 | A: merge/codec/snapshot | Omission vs explicit presence, retained members, canonical no-op | Table/property round trips across four forms; byte comparisons on no-op and unrelated artifacts; one-preference edit |
| G1-AC03 | A: strict parser/domain | Result bounds 8/32/32 and closed safe policy | Unit negatives 9/33/33, duplicate fields/IDs/pairs, singular runtime, shapes, dangling/unconfigured refs and unsafe values; no effects |
| G1-AC04 | A+B: merge/reference validation | Require explicit dependent removals/replacements, no cascade | Negative missing/conflicting targets and retained dependencies; reconciled compatible draft success |
| G1-AC05 | C–E: local confinement/read-back | Only A portable policy + required installation metadata | Two synthetic Projects share candidates; remove Claude/dependents from A; read-back A+B and byte/digest-compare B/local registry |
| G1-AC06 | B+D: readiness/resolver | Save portable-valid under authority; block execution with exact reason | Matrix absent/retired/disabled/mismatch/executable/auth/capability/ambiguity; persisted declarations, no fallback/dispatch/delete |
| G1-AC07 | B+C: Workflow catalog/application | Exact definition, complete Stage/Agent/Profile/Runtime compatibility | Built-in/Project revisions, none/unresolved/tampered selection, constraints, preferences, retained/new incompatibilities; no Workflow write |
| G1-AC08 | B+E: immutable consumers/impact | Future admission semantics vs frozen bindings | Synthetic retained Execution/Stage Plan bytes before/after, new-admission evaluator and existing permission/freshness refusals; zero Work Item writes |
| G1-AC09 | C+D: authority/completion | Exact UUID/digest/revision replay only | Negative missing/wrong/partial authority and cancellation; fake draft approval/readiness cannot publish; authoritative write count zero |
| G1-AC10 | C: adapters/coordination | Bind every consulted snapshot/content/object identity | Each dependency drift separately, same counter/new content, relocation/reselection, pre-commit hooks and competing processes; at most one commit |
| G1-AC11 | C+E: publication/recovery/read-back | Existing truthful old/new/partial/recovery semantics | Faults at private staging, recovery record, pre-publication, portable commit, installation commit, ack/sync/cleanup and read-back; permission/quota/short-write/interruption; deterministic recovery inspect, no blind retry |
| G1-AC12 | A+D+E: schema/compatibility | v1 read preserved; policy-edit gate v2/v3; no conversion | Frozen legacy corpus, same-version v2/v3 round trip, G-1 refusal for v1/v4/newer schema; existing CREATE/unrelated EDIT regressions |
| G1-AC13 | B+D: canonical readiness composition | Evaluate proposed and persisted state through same rules | Snapshot-equivalence tests; structural vs operation readiness; spies deny Provider/network/Runtime invocation/secret readers |
| G1-AC14 | A–E: strict input/safe projection/filesystem | No-follow/confinement, sanitized fixed diagnostics, private state | Adversarial synthetic sentinels/paths/errors/links/replacements; scan outputs/logs/files/Evidence; local bindings unchanged; native boundary tests |
| G1-AC15 | D+E: CLI/skill inspection | Same configure/edit domain journey and authorization | Black-box canonical vs routed payload parity plus separately scoped native conversation Evidence; no raw file patch/hidden approval/G-2 routing |

Native acceptance follows the [existing support matrix](../004-mvp-v1-baseline/plan.md#10-persistence-filesystem-concurrency-and-recovery),
including its per-architecture filesystem obligations; generic CI is insufficient.
Evidence must record source/head SHA, platform/filesystem, synthetic UUIDs,
before/proposed/observed revisions and digests, preview digest, authority outcome,
actual write set and sanitized diagnostics/read-back. Compare complete unrelated
artifacts/bindings, Project B, registry, Workflow definitions/selection and existing
Executions; refusals/dry-run/cancellation/no-op require zero authoritative effects.
Synthetic secret sentinels stay in test memory and are checked absent from exported
Evidence; never import real provider payloads or credentials. Reuse existing fault
hooks/corpus/testing conventions, not a new permanent test engine.

## 9. Decisions, alternatives, risks and gates

| Proposed choice | Concrete simpler alternative / trade-off | Revisit / human gate |
| --- | --- | --- |
| Extend existing EDIT + ephemeral typed draft | Whole-collection explicit replacement through existing policy inputs reduces member transport code, but makes safe one-member editing/error detection harder. Keep replacement as one mode, normalize all intent in application; no registry/engine. | Adapter spelling is reversible; changing absence/member semantics requires clarify and new contract approval. |
| Small pure portable Workflow helper + existing resolver/readiness | Calling `ValidateWorkflowProfilesV3` directly is shorter but wrongly makes local presence a save prerequisite. Extract portable portion and keep local checks in readiness. | Any unavoidable contradiction with accepted schema/Profile ownership needs clarify/Proposed ADR before Tasks. |
| Extend current CAS/recovery with dependency guards | Coarse shared state-root lock, existing publication/recovery and lock-held readers are simpler than per-snapshot lock registry or new transaction engine. Accept administrative serialization cost; optimize only with Evidence of need. | New storage engine, commit point, durable read-set journal or incompatible recovery format requires explicit architecture decision; do not quietly alter ADR-0007. |
| Semantic impact with unknown execution inventory | A complete cross-Execution inventory is more expensive and creates drift dependencies. Exact active definition plus immutable-scope disclosure satisfies bounded G-1 without invented counts. | Inventory claims would require separately bounded read/snapshot design; no hidden completeness promise. |
| G-1-safe preview projection | Reusing full legacy preview is shorter but leaks host paths/unrelated references. Keep internal envelope sealed and expose only safe policy facts; avoid rewriting unrelated preview contracts. | Any required public wire break is a clarify/compatibility decision before Tasks. |
| Explicit v2/v3 gate | Broad `MultiRuntimeSchema` reuse is less code but silently extends approved scope to v4. Preserve existing v4 elsewhere, refuse G-1 policy edits there. | Separate Spec approval to extend G-1 schemas. |

Principal risks: conflating local blockers with portable invalidity; default-profile
precedence drift; dropping companion digests; incomplete dependency protection or
self-deadlock under existing common-root locks; no-op rewriting formatting/local
metadata; false success after portable commit; raw preview/completion leakage.
Units A–E have finite passage criteria targeting those risks. No material accepted
ADR contradiction is identified in this design; no ADR is created or accepted.
If the lock/source/format review cannot preserve these semantics, stop affected
work for clarify and, where architectural, a Proposed ADR and human decision
before Tasks; do not weaken the approved Specification to fit current code.

Rollout is additive to EDIT and bounded to policy intent; preserve CREATE,
unrelated EDIT, v1 reads, schema-4 behavior elsewhere and frozen historical
formats. Documentation at implementation covers actual configure/help/inspection,
readiness and recovery behavior. Rollback of code follows the existing release/
compatibility policy; rollback of user policy is always a new authorized edit.
No migration or new dependency is proposed.

## 10. Plan verification and handoff

This section is documentary Evidence only, not G1 acceptance or product tests.
Discovery commands: `git status --short`, `git rev-parse HEAD`, `git ls-remote`,
`gh pr view 327`, `gh pr list --state open`, `gh pr diff 327`, `gh issue view 305`,
approval/review reads via `gh api`, Linear issue/comments read, and bounded
`rg`/source reads. Baseline above was exact before edits.

Orchestrator units completed: interface/contract inspection, plan design, separate
cross-check of security/acceptance/ADR alignment. No delegation: tools expose
model overrides but no verifiable strictly-lower hierarchy relative to this
session. Review is an Orchestrator pass, not independent or human approval.
Documentary validation executed on this revision:

- `./scripts/validate-repository.sh .` — PASS; includes ADR governance structure,
  maintainer adapters/harness, automation registry, sensitive-file checks and
  validator regression tests. Optional runtime behavioral scenarios were skipped
  by the script and remain UNVERIFIED.
- Ephemeral Python assertions — PASS: 131 local Markdown link targets (after the
  support-matrix link), 15 unique ordered acceptance rows, balanced fences and
  unchanged approved contract text before the historical gate annotation.
- `git diff --check` — PASS. Final staged paths/content and staged sensitive-file
  validation are recorded in the PR with the resulting commit SHA.
- Separate Orchestrator security/contract review: no blocking finding or material
  accepted-ADR contradiction identified. Approval remains human and pending.
- `gitleaks` is unavailable; not installed. No product/Go tests, Runtime probes,
  native filesystem acceptance or G1 acceptance oracles were run in plan.

Only three Markdown artifacts are changed: this Plan, the minimum status/provenance
reconciliation and the Specifications index. No code, production tests, CLI,
skills, ADR, installation or Execution changes.

Handoff: same PR #327/branch above; approved specify input SHA remains fixed.
Completed output is this Proposed Plan plus status provenance and necessary index
links. Canonical Execution state stays unknown; no lifecycle transition or Linear
status/assignee/relation mutation. Remaining work: human technical review and
**explicit approval of the exact Plan revision before preparing tasks**. Then
separately authorize tasks; implementation stays gated. Re-read remote HEAD,
conflicts, accepted contracts and canonical workflow state at continuation.
