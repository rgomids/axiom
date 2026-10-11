# Issue #306 — Proposed local Runtime/Profile authoring contract

## Status and acceptance boundary

**Proposed / Awaiting Human Approval, reconciled 2026-10-11.**
[Maintainer direction D1–D5/onboarding](https://github.com/rgomids/axiom/issues/306#issuecomment-6108025323)
confirms conceptual choices and replaces the previous D3; it does not accept the
exact artifacts. Prior proposal at `2096509` is historical. This revised proposal,
[Plan](plan-306-local-runtime-profiles.md), [Security](security-review-306-local-runtime-profiles.md),
[ADR-0023](../../decisions/0023-local-runtime-configuration-surface.md),
[Spec 007 effort/onboarding amendment](../007-configurable-workflows/amendment-306-profile-effort-onboarding.md)
and [ADR-0024](../../decisions/0024-model-profile-effort-ownership.md) need exact
independent review. No Tasks, production/schema/fixture changes or vendor probes.

## Preserved accepted contract and proposed delta

Project != Repository; Runtime != Model; Role != Model; Execution != Agent;
Provider != Transport; Skill != workflow source of truth; Evidence != raw chat.
Portable Project v1/v2/v3 policy/bounds remain unchanged; exact Profile key,
Runtime association and model matching remain required. Workflow adds requirements;
enabled/allowlisted local candidates and authoritative observations further
constrain permission. Local preferences never invent portable Project intent.
No initiating-Runtime preference, model ranking or silent fallback.

Current local v1 has no effort; accepted Workflow v1 Agent owns it. D3 now proposes
local **format 2** with required effort and Workflow **schema 2** with no effort
field/override. Historical accepted files and meanings stay intact. The companion
Spec 007 proposal is canonical for migration/snapshot semantics; this document
owns local authoring, security, lifecycle and onboarding operations.

## Proposed requirements

**LA-01 — single authority.** Canonical application use cases own machine-local
read/list/show/validate/create/edit/remove through RuntimeProfileStore. Keep one
codec/store/selector/resolver, not a skill registry or competing version path.

**LA-02 — bounded named Profiles.** IDs stay globally unique/operator-defined;
Runtime must be supported and model explicit. Retain eight Runtime/32 total local
Profile limits and independent portable policy bounds. More than three per Runtime
is supported. Format 2 Profile includes required operator-chosen effort under
PE-01; same model with distinct effort uses distinct IDs. Complexity remains a
separate supported-task constraint; names fast/balanced/deep imply no effort.

**LA-03 — lifecycle/generation (D1).** Initial valid format-2 creation is a bounded
compound candidate, revision 1, with at least one Profile per included installed
Runtime and valid nonempty allowlists. Updates/migration/retirement/recreation
increment generation exactly once; refuse overflow. Runtime Enabled and Profile
allowlist membership retain eligibility semantics. Invalid empty allowlists/dangling
local references require explicit valid compound removal or block. Final removal
atomically replaces the canonical file with a closed metadata record:
`kind: runtime-profile-tombstone`, integer `formatVersion: 1`, positive `revision`
(prior + 1), `priorConfigurationDigest` (64 lowercase SHA-256 hex). No configuration
or credential payload. Recreate by conditional replacement of the exact tombstone
with active v2 at tombstone + 1; never reset to 1. No directory/receipt/binary,
Project/Workflow/Execution or vendor authentication deletion.

**LA-04 — authority/concurrency (D5).** Bind state fresh/active-v1/active-v2/retired,
local target identity, format, prior content/revision/file/directory identities,
candidate digest and all effects. Snapshot token + preview digest + explicit local
authority required. Missing suffix binds nearest existing protected ancestor and
absence; distinguish authorized directory creation from external appearance.
Re-read and compare under ordered exclusive locks. CREATE collision, manual same-
revision rewrite, ABA, stale/changed candidate or unsafe/pending state refuses.
Existing incomplete directories are not fresh absence. Supported retirement keeps
ABA history; no promise to detect arbitrary erased history outside protected state.
No-op is unchanged/no increment. Successful replay is stale, requiring fresh preview.

**LA-05 — safe inputs/projections.** Bounded typed authoring rejects secret values,
host paths, login files, arbitrary commands/argv/env, unknown fields and unsafe
payloads before drafts/output. Reuse applicable pure portableconfig safety checks;
never echo rejected input or serialize raw config. A scanner does not prove every
arbitrary token nonsecret. Legacy unsafe model/reference fields are only safely
projected/redacted; inspect historical state without normalizing/writing it.

**LA-06 — vendor-managed authentication (D2).** Codex/Claude must already be
installed/authenticated/operational through supported native CLI subscriptions.
Axiom does not capture/store/copy/manage tokens, passwords, API keys, sessions,
login files or Provider credentials. No credential backend/login/refresh/logout.
Local v2 authoring forbids credentialReference inputs; the Runtime-level field is
not carried into its new wire. Legacy v1 remains read-compatible and nonempty
references stay redacted/unresolved and subscription-blocked. Migration explicitly
removes the legacy reference as an approved local effect or refuses; no copying
it to drafts or automatic secret-bearing migration. Vendor login state is untouched.

**LA-07 — sole Profile effort (D3).** Local v2 effort is explicitly selected using
PE-01; missing effort never defaults implicitly. Workflow v2 references Profile,
never overrides effort through fields, capabilities, argv/environment or generic
WorkUnit controls. Compiler derives frozen controls from the selected Profile,
with Profile/configuration/proof digests and exact Runtime/model/version/mode.
Declared effort/model capability may be saved as unproven; use requires authoritative
support and fresh dispatch checks. No substitution to another effort/Profile or
cross-vendor equivalence. The Spec 007 proposal owns exact snapshots and migration.

**LA-08 — observation authority (D2).** Passive installed/integration observation,
authentication status, model availability and effort support are separate facts.
Default reads/validation/preview/apply/onboarding inventory run no vendor process,
read no credential values and produce no Provider effects. Auth/status/model probes
require specific separate consent and classification; auth success is neither model
support nor inference permission. Reuse existing preflight/guard ownership, excluding
vendor login-file management. Current observer cannot prove G-3; unknown blocks use.

**LA-09 — dependencies (D5).** Preview local removal/edit impacts and known dependent
Project/Profile references with inventory bounds/incompleteness. No arbitrary
repository scan or claims of universal inventory. Never rewrite portable Project,
Workflow or immutable Execution; future incompatible resolutions block precisely.
Reuse #231 readiness and sole resolver; no global Doctor or automated migration.
Project policy editing is #305, Workflow selection a separate Project operation.

**LA-10 — filesystem/recovery/compatibility (D1).** Reuse protected anchored roots,
permissions/ownership/ACLs, locks, bounded no-follow reads, atomic publication and
scoped recovery. Before/after commit/read-back prove canonical pathname identity;
uncertainty preserves state and blocks blind retry. Extend recovery discovery and
inventory for runtime-profiles/v1 active formats 1/2/tombstone 1, all at the same
canonical location. Restore exact uncommitted prior state only when provable;
committed rollback uses a new monotonic authorized generation. New readers read
old supported formats without refusal/rewrite; new admission may require explicit
migration. Incompatible old readers fail safe. No universal downgrade guarantee,
automatic down-conversion or historical corpus edit. FR-026 matrix/forward path
is specified in the companion amendment and needs exact acceptance.

**LA-11 — skill owner (D4).** Thin axiom-runtime handles natural-language local
configuration/inspection/onboarding, explains options/effects, prepares safe input,
obtains explicit confirmation and invokes canonical CLI only. Application owns
validation, persistence, preview/apply, authority, diagnostics and recovery.
Project defines portable permissions/selects Workflow; Workflow defines stages and
Profile references; Work Item executes. No Project/Workflow authorship forwarding
or duplicate store. ADR-0023 proposes the explicit inventory extension; historical
skill receipts and retired aliases remain intact.

**LA-12 — first experience.** Noninteractive CLI installation passively detects
supported absolute executables; read-only discovery does not invoke bootstrap Run's
integration installation. First configuration-dependent use directs missing/insufficient
state to axiom-runtime guidance. Preserve valid existing state; legacy effort requires
explicit migration, not automatic overwrite. Optional fast/balanced/deep templates
contain no fixed model/effort/ranking. Start with one valid Profile per detected
Runtime; no absent binding. Recommend only verified options with provenance;
unknown model/effort yields explicit blocker/consent handoff. User can save intent
without readiness, but no ready/dispatch claim follows. Each local apply receives
exact confirmation/read-back; derived Workflow and portable policy have separate
owners/approvals. See ON-01–05 and AC-ON1–4 for Codex-only/Claude-only/both cases.

## Proposed reconciliation of related accepted specifications

Accept this local v2 authoring addition independently from Plan approval. Portable
Spec 002 Runtime Policy v2 remains unchanged. Spec 007/ADR-0022's accepted legacy
Agent effort requires the proposed versioned amendment/ADR-0024; annotate affected
historical clauses only after exact acceptance, not as if already active. G-3 proof,
G-4 Provider linkage and #278 real acceptance remain independent.

LA-11/ADR-0023 propose the fourth active configuration skill, preserving historical
Specification 004 consolidation and #303 receipts. No retired generic wrapper
returns. Planning alters no installed catalog/skill/schema or accepted ADR content.

## Pending exact artifact gates

D1–D5/onboarding direction is confirmed; do not solicit it again as unknown.
Remaining review concerns exact local/Workflow/Execution version families, migration
and legacy-continuation window, Profile runtime-default semantics, cross-cutting
implementation ownership and independent Plan/Spec/ADR/Security acceptance.
Any material alternative requires revised artifacts and renewed exact review.
