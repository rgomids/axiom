# Specification 002 amendment — G-1: existing Project Runtime/Profile policy

Status: **Proposed**, revision 1, 2026-10-18. SDD stage: **specify**.
Technical owner: [#305](https://github.com/rgomids/axiom/issues/305), parent
[#303](https://github.com/rgomids/axiom/issues/303).
Operational Work Item: [AXM-34](https://linear.app/rgomids/issue/AXM-34).
These proposed bytes require explicit human review/approval before planning or
implementation authority is inferred. This document records no acceptance.

## 1. Contract lineage and bounded amendment

This amendment specializes [Spec 002 FR-018/019 and AC-15/16](spec.md)
for editing the portable Runtime/Profile policy of an installed, existing
Project. It extends the supported authoring journey beyond CREATE, preserving:

- [#140 portable policy v2](runtime-policy-v2.md): closed shape, presence forms,
  bounds, canonical encoding and exact matching;
- [#231 bootstrap v3/readiness](project-bootstrap-v3.md): v3 retains the v2
  policy; structural validity, operation readiness and authority remain distinct;
- [HD-006 / accepted Spec 007 amendment](../007-configurable-workflows/amendment-303-workflow-skill.md)
  and [ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md),
  accepted at #304 head `7db80ac95b5e83cd8cc9aaaadc785633c3ee3fa9`;
- [ADR-0020](../../decisions/0020-workflow-definition-revision-binding.md):
  Project-owned selection and immutable Execution Workflow bindings;
- [ADR-0024](../../decisions/0024-model-profile-effort-ownership.md): local Model
  Profile owns effort; G-1 adds no portable effort or Stage/Agent override;
- [#230 lifecycle matrix](../004-mvp-v1-baseline/issue-230-resource-lifecycle-matrix.md):
  portable mutation plus coherent installation metadata, preserving operational
  state and existing archive/eligibility admission rules.

Only the historical authoring limitation (policy authored at CREATE, existing
EDIT rejecting those inputs, no supported re-publication in #140) is extended
prospectively for this policy. Historical documents, decisions and Evidence
are preserved. This is neither a claim that G-1 already exists nor an amendment
of Workflow, local Profile or Execution storage formats.

The actor is an operator, using the canonical Project operation directly or
through `axiom-project`. Lingo/application owns identity, validation, readiness,
preview, digest, authorization, persistence and read-back. Conversation proposes
intent and explains results; it does not patch the manifest or approve itself.
This specifies behavior without adding command names, flags, draft wire formats,
parallel registries or a new resolver. Command/payload design belongs to an
approved later Plan, within existing Project configure/edit ownership.

Out of scope: G-2/#306 local Profile authoring; installation/authentication,
credentials, Provider calls, Runtime invocation, Workflow definition or selection
changes, stage dispatch, existing Execution mutation, global Doctor, Git effects,
release, R-2/R-3 and acceptance of other Work Items.

## 2. Observable journey

| Step | Required observable result | Permitted effects |
| --- | --- | --- |
| Read | Explicit selector resolves one installed Project; return immutable UUID, slug/name, portable schema, current policy including presence forms, revision and snapshot digest | Read only |
| Partial draft | Operator identifies only the intended additions/replacements/removals or collection presence changes; retain everything else; return complete proposed policy and bounded diff | Working draft only, no authoritative state change |
| Validate / dry-run | Validate the complete proposed Project; distinguish structural errors, Workflow incompatibility and machine readiness; report safe subjects and remedies | Read only; no publication or authority |
| Preview | Canonical identity, before/after, exact snapshot bindings, effects, blockers, readiness and Workflow/Execution impact (§5) plus application-produced preview digest | Read only |
| Authorize | Explicit local confirmation for that exact Project, proposal digest, expected revision/snapshots and write set | Authority only for the reviewed local policy edit |
| Apply | Re-read and compare all bindings; admit only the exact validated proposal; commit complete state coherently or refuse drift | Selected Project portable policy and required installation bookkeeping only |
| Read-back | Reopen through canonical Project read and `project validate`; report observed identity, schema, policy, revision/digest, readiness and actual effect outcome | Read only |

No CWD, nearest Repository, remote, initiating Runtime, chat memory, duplicate
display name or prior selection identifies the target. Slug is installation-local
lookup ergonomics; resolve it to the immutable UUID and bind that UUID before
drafting. Missing, ambiguous, colliding or inconsistent identity refuses mutation.
The recorded portable source is authoritative; never derive another source from
slug/CWD. Missing, unsafe, malformed, externally modified/unadopted source or an
installation requiring recovery is not repaired or re-adopted by this operation.

Cancellation, read, draft, dry-run and preview do not change Project portable or
installation state. A logically identical update is a reported no-op: no revision
increment, rewritten artifacts or refreshed bindings. Readiness is never authority.

## 3. Portable declarations and partial update semantics

G-1 supports policy edits in **schema 2 and schema 3**, without changing their
version. It reads legacy schema 1 under its historical contract; multi-Runtime
policy editing of v1 refuses with guidance that an explicitly approved migration
is separate. Other supported v1 read/edit behavior is unchanged. Unsupported
schemas refuse before effects. No upgrade, down-conversion or lossy rewrite.

Use only the existing portable fields and exact closed shapes:

| Collection | Members / invariants | Bound |
| --- | --- | --- |
| `runtimes` (`runtimes[]`) | `{id}`; unique opaque nonblank Runtime ID | 8 |
| `modelProfiles` (`modelProfiles[]`) | Unique `key`; configured `{key, runtimeRef, model}` or `{key, state: unconfigured}` alone; configured `runtimeRef` names a declared Runtime | 32 |
| `runtimePreferences` | `{role, complexity, modelProfileRef}`; unique `(role, complexity)`; reference names a configured Project Profile | 32 |

Singular portable `runtime` is forbidden. Bounds apply to the **result**, not
only changed entries. Each collection preserves the distinctions `absent`,
`unconfigured`, `[]`, nonempty sequence, as in #140. Omitting an edit retains the
old collection; requesting absence removes that declaration; explicitly requesting
`unconfigured` or `[]` means those states, never retain/default. A draft must
distinguish these intentions unambiguously without adding portable fields.

Member updates use exact logical identities: Runtime `id`, Profile `key`, and
preference `(role, complexity)`. Add requires absence; replace/remove requires an
existing target. A missing member, duplicate operation, simultaneous add/remove
or multiple replacements of one target refuses rather than guessing precedence.
Intent to replace an entire collection must be explicit. Changing a key is an
explicit removal/addition with all affected references reconciled in the same
draft. No implicit cascade, trimming, case-folding, wildcard, ranking or default.
Canonical ordering remains Runtime ID, Profile key, preference role/complexity.
Ordering never chooses a Runtime or Profile.

Reject duplicate YAML fields/collection identities/preferences, unknown fields,
unsupported forms, invalid tokens, model values violating existing safety rules,
over-limit results, dangling `runtimeRef`/`modelProfileRef`, configured/unconfigured
shape mixtures and preferences pointing at unconfigured Profiles. Runtime removal
with retained associated configured Profiles fails; Profile removal with retained
preferences fails. Report the precise safe field/key and necessary explicit
reconciliation; never silently delete dependent entries or select replacements.

Materialize and validate the **complete proposed Project** before persistence
(FR-018). Partial intent is not a direct file patch or partial persisted state.
Only the three requested policy collections may change. Identity, schema, name,
slug, Repository/Provider/Integration declarations, context, documents, policies
and unrelated presence forms/values remain unchanged. Complete manifest encoding
may use its existing canonical serializer; it must not re-author/default unrelated
fields. Separate unchanged artifacts retain their bytes. Existing machine-local
Repository/document/credential bindings and archive/disabled state are retained;
only bookkeeping required to recognize the new portable revision may advance.

Multiple named Profiles for one Runtime or model are valid; their distinct keys
express distinct permission. The same key cannot mean several associations.
Portable permission does not select a Stage, Agent, Workflow or local Profile.
It neither creates nor enables a local Profile. A chosen local candidate may
propose the existing portable triple, but local state must not overwrite retained
portable declarations. A valid explicit portable triple may be saved without a
matching local candidate. It remains declared on a second machine or after local
retirement/disablement; no local availability requirement is imposed on storage.

## 4. Validation, readiness and safe persistence

Reuse canonical domain/schema validation and the #231 evaluator behind
`project validate`. Dry-run evaluates the proposed snapshot through the same
rules used for persisted state; apply revalidates, and read-back evaluates what
was actually persisted. No duplicate Doctor algorithm in CLI or skills.

Keep distinct outcomes:

1. **Invalid portable state or ambiguous edit/identity:** reject apply before
   effects, even if a local candidate would make execution possible.
2. **Active Workflow portable incompatibility or unresolvable selection (§5):**
   reject apply; explicitly repair/select a compatible Workflow through its own
   separately authorized journey, then obtain a fresh policy preview.
3. **Valid portable state with unavailable/unconfigured local readiness:** allow
   saving under exact local authority, while reporting execution blockers before
   approval and after read-back. Never label that Project execution-ready.

Case 3 includes absent/empty/unconfigured policy where no active Workflow requires
it, missing/disabled/retired local Profile or Runtime, model/association mismatch,
unsupported/unproven capability, unavailable executable, missing authentication
proof, or unresolved resolver multiplicity. Report the canonical readiness reason
and logical subject, retaining declarations. Ambiguity in **execution selection**
always blocks that selection; it never licenses choosing the first candidate.
An unreadable/corrupt required local snapshot that cannot support an exact preview
is a preview/apply blocker, distinct from an observed missing local candidate.

Candidate resolution remains the existing resolver: configured Project policy ∩
exact local Runtime/Profile/model match ∩ Stage/Agent constraints ∩ supported
complexity/capabilities and authoritative observations. No model-name inference,
global preference inventing Project intent, alternative Runtime/model/Profile,
or silent fallback. Effort/authentication/proof remain local/versioned contracts,
not new portable declarations or proof obtained by invoking a Runtime in G-1.

Preserve #231 operation-scoped readiness: a Runtime blocker does not by itself
block Work Item capability operations whose requirements do not include it.
Retain existing `runtime_policy_unavailable` versus `runtime_resolution_blocked`
classification and inner reason where applicable. Diagnostics added for G-1 must
identify validation vs publication vs readiness scope; exact new codes/output
encoding await the Plan, without changing existing categories silently.

## 5. Active Workflow compatibility and impact disclosure

Read the Project's exact active Workflow selection and resolve its recorded
`workflowId`, revision, digest and source. Bind the resolved definition/schema;
do not substitute latest/built-in/repaired revisions. No selection is reported
explicitly as none; an existing missing, mismatched, unreadable or unresolvable
selection blocks apply because compatibility cannot be established.

For each Stage/Agent of that exact definition, inspect actual `profileRef` and
`runtimeConstraints`, role, complexity and capability requirements. Portable
compatibility requires its exact configured Project Profile, a declared associated
Runtime and satisfaction of the Agent's Runtime constraints. Check retained and
changed references, including effects of preference changes using existing
selection precedence; preferences never override an explicit Agent Profile.
Unavailable local evidence is readiness, not a fabricated portable mismatch.
Preserve historical Workflow schema meanings; do not migrate Workflow v1 or
introduce effort into Workflow v2.

Before authorization, report a deterministic, bounded impact set keyed by exact
Workflow reference, Stage ID, Agent ID, Profile key and associated Runtime, with
before/after compatibility and reason. Include affected retained references, not
only deleted members. Distinguish a newly introduced incompatibility, a pre-existing
incompatibility left unresolved (also blocks apply), and local unavailability.
If the set cannot be fully evaluated within supported bounds, block rather than
truncate away blocking references. No Workflow write or automatic selection.

Preview must explain effects in these separate scopes:

- **New Executions**, including those for existing linked Work Items without an
  Execution: use the edited Project policy and exact active Workflow at their own
  admission; remain blocked by current readiness/authority requirements.
- **Existing Executions and Stage Plans/bindings:** no rewrite, rebinding,
  retroactive Profile substitution, cancellation or auto-resume. Their Workflow,
  Runtime/model/Profile/effort and capability snapshots remain immutable facts.
  Future dispatch/resume still performs its already required freshness and
  permission checks; policy revocation may block it and never widens permission.
- **Work Items:** no Provider write, status/relation change, new Execution or
  permission to run; Project policy change alone is not a delivery transition.
- **Other Projects and local configuration:** unchanged. Removing Claude from
  Project A does not uninstall/disable it locally or remove it from Project B.

Impact is derived from exact referenced definitions and policy semantics. Do not
claim a complete inventory of affected existing Executions/Work Items unless it
was actually read; label unknown/uninspected inventory explicitly. If individual
records/counts inform the reviewed effects, bind their consulted snapshot too.

## 6. Identity, authorization, drift and recovery

Preview binds at minimum: operation/edit scope; Project UUID and resolved source
identity; current portable schema/revision and digest of the actual snapshot;
installation revision/digest and relevant operational state; complete proposed
Project digest and policy diff; exact active selection/definition (or explicit
none); consulted local configuration/observation revisions **and digests**;
diagnostic/impact snapshot; allowed write set and effects. Do not put revision,
approval or digest bookkeeping into portable policy fields. Equal counters alone
are insufficient when observed content differs. Identical normalized intent and
snapshots produce identical canonical preview bindings.

Human/system local authorization must name that exact application-produced
preview digest and expected revision, following existing `--authorize-local`
authority semantics. Guided confirmation and non-interactive use obey the same
contract. Conversation, readiness, previously accepted preview, drafted approval
booleans and external authority cannot stand in for this confirmation.

Immediately before commit re-read all bound state and enforce expected revision,
digest, source/object identity and write confinement. Portable edit, relocation,
slug/identity change, installation refresh, active Workflow reselection (even a
compatible one), local configuration or consulted observation drift invalidates
authority. Refuse with zero policy effects and require fresh preview/confirmation;
never rebase the patch, refresh a digest or choose another candidate under old
authority. Conflicting Axiom writers from the same snapshot cannot both commit.

Keep FR-013/018 logical atomicity and the
[ADR-0005 threat boundary](../../decisions/0005-bounded-local-filesystem-threat-model.md)
and [ADR-0007 publication/recovery protocol](../../decisions/0007-local-publication-and-recovery-protocol.md).
No mixed/partially accepted Project may become authoritative. Before commit,
failure preserves prior valid state; after commit, report actual committed state,
including installation bookkeeping incomplete/recovery required if applicable.
Never claim rollback or retry a blind mutation after uncertain acknowledgment.
Read current state/recovery first; recovery is a separate exactly authorized
operation. No deletion of unknown leftovers or automatic restoration of removed
permissions. A reversal is a fresh policy edit with a fresh preview/authorization.

Read-back compares observed UUID, schema, complete effective policy and resulting
revision/digest with the confirmed result. Report confirmed apply, no-op, failure,
or committed-but-unverified/recovery state truthfully. A later writer changing
state before read-back is explicit read-back drift, not evidence that the old
policy is still effective. Never grant execution authority based on write success.

## 7. Failure and security requirements

| Case | Required result |
| --- | --- |
| No authority, wrong Project/digest/revision, cancelled confirmation | No apply/effects; exact missing/conflicting authority reported |
| Invalid collection shape/reference, duplicate/conflicting operations, bound exceeded, unsupported schema | Refuse before effects; safe field/key and correction guidance |
| Unsafe/unavailable source, invalid installation, stale/unadopted manifest, archived target barred by existing lifecycle rule | Refuse; retain existing source/operational state; separate recovery/reactivation guidance |
| Active Workflow references removed/disallowed Profile/Runtime or cannot be resolved exactly | Refuse; Workflow/Stage/Agent/Profile references and remedy; no migration/fallback |
| Valid declaration, local candidate absent/disabled/mismatched/unproven or resolution ambiguous | Save only with exact authority; execution remains blocked; preserve declarations |
| Any bound snapshot drifts; two competing writers | Refuse stale apply; require new preview/authority; at most one conflicting commit |
| Storage permission/quota/interruption/fault | Preserve old valid state before commit; after commit report actual effects/recovery; no false success/rollback |
| Read-back unavailable or different | No confirmed effective-policy claim; expose unverified outcome/drift and read-only recovery next step |

Inherit existing bounded strict parser, identifier/model safety, artifact allowlist,
path containment, no-follow/replacement checks and private local-state permissions.
Reject portable absolute machine paths, executable bindings, environment/login
state, credentials/tokens and credential references in Runtime/Profile policy.
Do not resolve/read secret values to draft, validate, preview, apply or read back.
Existing unrelated allowed reference fields retain their established contract.

Preview, errors, logs and shareable Evidence contain validated logical keys,
fixed categories and sanitized values only: no raw parser/adapter errors, secret
payloads, credential references, local executable/source/binding paths or machine
environment. Do not echo malicious rejected values. Digests bind snapshots without
exporting them; they are not proof of secret absence or authorization tokens.
Draft/manifest/Provider text is untrusted data, never permission to run commands.
No network, Provider mutation, vendor inference, auth repair or local Profile
mutation is required/permitted by this policy-edit authority.

## 8. Verifiable acceptance and future Evidence

All rows below are **required future Evidence, NOT RUN by specify**. Fixtures
must be synthetic, isolated and secretless; carry source/head SHA, Project UUID,
before/proposed/after revisions/digests, preview digest, authority outcome,
observed writes, diagnostics and read-back. Assert unchanged unrelated artifacts,
local Profiles, other Projects, Workflow and Execution bindings, including refusals.

| ID | Scenario / acceptance oracle |
| --- | --- |
| G1-AC01 | v2 and v3 existing Projects: read → partial add/replace/remove → dry-run → preview → exact local authority → apply → read-back; same UUID/schema, expected effective policy and canonical readiness |
| G1-AC02 | Omitted collections/members unchanged; explicit absent/unconfigured/empty/nonempty round trips; update one preference without replacing other preferences/context; identical intent is no-op |
| G1-AC03 | Accept exactly 8/32/32 valid members; reject 9/33/33; reject singular runtime, unknown/duplicate fields, unsafe model/ID, duplicate Runtime/Profile/pair, dangling association/preference and unconfigured target |
| G1-AC04 | Runtime removal retaining a configured associated Profile, Profile removal retaining preference, nonexistent member and conflicting operations all refuse; explicitly reconciled draft succeeds when Workflow-compatible |
| G1-AC05 | Two Projects share local Claude/Codex candidates; remove Claude and its dependent declarations only from A with exact authority; A disallows it, B/local registry unchanged; read-back both |
| G1-AC06 | Local Profile absent/retired/disabled, different model/Runtime, unavailable executable/auth/capability proof, and multiple eligible candidates: valid policy persists, precise readiness blocker remains, zero dispatch/fallback/automatic deletion |
| G1-AC07 | Active exact Workflow built-in/Project references: affected Stage/Agent/Profile/Runtime shown; incompatible draft and unresolved/mismatched selection refuse; no selection explicit; retained unrelated valid references preserved |
| G1-AC08 | Compare new Execution admission semantics against immutable existing bindings; no new Execution/Work Item mutation, rebind or automatic migration; current required permission/freshness refusals preserved |
| G1-AC09 | Missing/wrong authority, UUID, digest, expected revision and cancelled confirmation: zero authoritative writes; readiness/draft boolean cannot authorize |
| G1-AC10 | Drift each bound snapshot separately, including same counter/different content, source relocation, active Workflow change, local configuration/observations; stale apply refuses; two conflicting writers cannot both commit |
| G1-AC11 | Deterministic failures before/at/after publication and installation bookkeeping, permission/quota/interruption and failed/concurrently changed read-back: prior state or truthful committed/recovery outcome, no mixed valid state or blind retry |
| G1-AC12 | Legacy v1 read semantics unchanged; G-1 multi-Runtime edit refuses without migration; v2/v3 remain same version; newer schemas rejected before effects; existing CREATE and unrelated EDIT contracts preserved |
| G1-AC13 | `project validate` agrees with proposed dry-run and persisted read-back for same snapshots, reports structural vs execution readiness separately; no duplicated Doctor/resolver, Provider/Runtime/network calls or secret reads |
| G1-AC14 | Adversarial secret/path/error inputs, traversal, links/replacements at supported boundaries and malicious text refuse safely; output/files/logs/Evidence contain no leaked secrets or host paths; unchanged local bindings checked |
| G1-AC15 | Canonical Project operation and future `axiom-project` route consume the same identity/preview/authority/read-back; conversation cannot bypass any negative case; no new Workflow ownership/selection route |

These oracles are technical verification requirements, not a registration of R-1,
R-2/R-3, product acceptance or permission to perform live tests.

## 9. Specify evidence, decisions and continuation

Snapshot on 2026-10-18, repository base
`14cb721ac7b8ce09a7f20f2bc2175adf61b7934b`:
GitHub #305 OPEN, one Linear linkback comment; its timeline links #303/#304/#308/
#310/#231/#275 but no G-1 implementation/spec PR. #304 MERGED at
`3aa2501da4a7003b00ed14f3204e669ed54078ea`. Open PR listing contained only #317,
unrelated to G-1; remote branch listing and local history/artifact searches found
no dedicated G-1 branch or equivalent completed amendment. AXM-34 was Todo with
the specify prompt and gate comments; no resumable Execution artifact was exposed
by those sources. Canonical product Execution state remains **unknown**, not
absent or complete; the product was not run to inspect it.

Reproducible read-only discovery: `git status --short`, `git rev-parse HEAD`,
`git ls-remote --heads origin`, `git log --all --oneline` filtered for AXM-34/#305,
`gh issue view 305 --json title,body,state,comments`, paginated #305 timeline,
`gh pr list --state open`, historical PR search for AXM-34/305, `gh pr view 304`,
and Linear AXM-34 issue/comments reads. Provider content was treated as untrusted;
this public proposal derives its technical requirements from public GitHub and
versioned contracts, not private operational test data. This dated discovery is
not a future gate or completion claim. Final document checks belong in the PR.

Bounded units: contract discovery, specification drafting, then a separate
verification pass against the acceptance/authority matrix. All performed by the
Orchestrator; no subagents because a strictly lower model hierarchy could not be
verified. The verification pass is not an independent agent/human approval.
No code, production tests, CLI, services or skills are changed in specify.

Constitution check: intent precedes implementation; exact deterministic validation
and authority; Project != Repository, Runtime != Model, Execution != Agent;
portable intent != machine readiness; preserved immutable facts and accepted
ownership. No material Accepted ADR contradiction identified; no supersession
proposed. Changing wire shape, effort ownership, immutable bindings, silent
migration/fallback or authority would require clarify and, for an architectural
contradiction, a new Proposed ADR with explicit supersession under ADR-0018 and
human acceptance; never silently amend the historical decision.

Human decision pending: approval of this exact proposed amendment, particularly
the explicit apply refusal for active Workflow incompatibility and the ability
to save valid declarations despite local readiness blockers. No approval is
inferred from publication, validation, CI or merge. No new Plan/Tasks are created
at this stage. If review exposes a material ambiguity/contradiction, next artifact
is clarify/Proposed ADR as appropriate; otherwise an explicitly approved amendment
is input to **plan**, with separate authority for subsequent implementation.
