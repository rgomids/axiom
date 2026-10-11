# Issue #306 — Proposed local Runtime/Profile authoring contract

## Status and acceptance boundary

**Proposed, 2026-10-11.** This is a reviewable addition to Specification 002,
not an accepted rewrite of [Runtime Policy v2](runtime-policy-v2.md), Spec 007
or ADR-0022. Independent exact-revision contract acceptance is required before
implementation. [Plan](plan-306-local-runtime-profiles.md) and
[security review](security-review-306-local-runtime-profiles.md) remain separate
approval artifacts. Planning authority does not authorize Tasks or product code.

## Preserved accepted contract

Project != Repository; Runtime != Model; Role != Model; Execution != Agent;
Provider != Transport; Skill != workflow source of truth; Evidence != raw chat.
Local configuration owns machine bindings; Project owns portable permission;
Workflow records requirements; Work Item/Execution owns concrete execution.

Portable v1/v2/v3 and active local v1 configuration wire remain unchanged.
LA-03 introduces a distinct versioned retirement record at the canonical location,
with explicitly reviewed compatibility rather than an empty configuration.
Exact Profile ID, Runtime and model match is required. Enabled/allowlisted local
Profiles, authoritative Runtime observations and Stage/Agent requirements further
constrain Project permission. Local preference never invents Project intent.
Unknown/unsupported/ambiguous selection fails closed, with no fallback or
preference for the Runtime hosting this chat.

## Proposed requirements

**LA-01 — single authority.** Read/list/show/validate/create/edit/remove operate
on `RuntimeProfileStore` through canonical application use cases. No second
registry, resolver, skill-owned store or Provider inference. Configuration is
machine-local and Project-independent.

**LA-02 — names and bounded state.** Local Profile IDs stay globally unique and
operator-defined, bound explicitly to `codex` or `claude` and an exact model.
Retain current eight Runtime / 32 total local Profile limits and separate portable
policy bounds. At least four named Profiles per Runtime must be authorable;
example presets carry no vendor quality semantics. Capabilities/complexities are
declared constraints, not proof.

**LA-03 — lifecycle without schema relaxation (D1).** Creation from absence takes
a complete valid bounded candidate: each Runtime has a nonempty allowlist and
at least one matching Profile. Later creates/edits preserve all unrelated entries.
Runtime disable toggles `Enabled`; Profile eligibility uses allowlist membership.
Do not add a duplicate Profile enabled field. A compound removal explicitly names
dependent local Profiles, allowlist memberships and local preferences. Dangling
references or invalid empty allowlists refuse unless the reviewed compound change
removes the owning Runtime. Final removal atomically replaces only the canonical configuration
file with a closed tombstone: `kind: runtime-profile-tombstone`, integer
`formatVersion: 1`, positive monotonic `revision` (prior + 1) and exact
`priorConfigurationDigest` (64 lowercase hex SHA-256). No other fields or
configuration payload are retained in this record. Snapshot reads classify it as
retired with no executable configuration. Recreation conditionally replaces the
exact reviewed tombstone with valid active v1 at tombstone revision + 1; it never
resets generation or uses a fresh CREATE. Fresh absence starts at revision 1;
existing incomplete version directories are blocked, not treated as fresh. No credential/binary/receipt,
portable Project, Workflow or immutable Execution is deleted or rewritten.

**LA-04 — exact mutation envelope.** Bind existence, local target identity,
format, prior revision/content digest/file and directory identity, candidate digest,
operation and every effect. Explicit confirmation requires exact snapshot token,
preview digest and local authorization. Revalidate under exclusive locks at apply.
CREATE collision, same-revision content changes, ABA recreation, unsafe roots,
pending publication, overflow or stale candidate refuse before effects where
possible. After uncertain publication, report recovery required; never retry
using old authority. No-op does not increment revision; successful replay is not
recognized by guessing from numeric revision/current candidate alone.

**LA-05 — safe presentation/input.** Read/list/show and previews use allowlisted
logical fields and safe diagnostic enums, never raw decoded configuration.
Authoring rejects inline secrets, host executable paths, login files, arbitrary
argv/environment, URL/assignment payloads and unknown keys before producing
reviewable output. Reuse applicable pure `portableconfig.SafeValue` checks for
logical/model fields without claiming that a pattern scanner proves arbitrary
text contains no secret. Never echo rejected values. Legacy unsafe fields remain
readable by the canonical store for compatibility but are redacted/refused as
new authored inputs; explicit repair/clearing requires a fresh preview.

**LA-06 — credential-reference decision (D2).** Recommended #306 authoring
supports vendor-managed subscription authentication only: `CredentialReference`
must be empty for newly authored bindings. No secure-secret backend is invented.
Legacy nonempty references are not displayed or resolved and stay blocked under
the existing subscription contract. An edit can preserve a legacy field internally
for unrelated changes or explicitly clear it; it cannot copy it into a proposal.
Supporting newly authored nonempty references requires independently accepted
source grammar, backend/security review and non-subscription effect contract.

**LA-07 — invocation and effort (D3).** No arbitrary commands or persisted effort
defaults are added to local v1. Model/capabilities/complexities use existing fields;
Stage/Agent owns effort choice. A structurally valid effort capability declaration can be saved as a declared
constraint. That save supplies no proof. Without authoritative proof for the exact
model, executable digest, version and noninteractive mode, readiness reports
`unproven` and resolver/dispatch refuse its use. #306 does not obtain G-3 proof,
infer vendor-wide support or translate tokens. Inherited declarations
do not become proof; resolver and dispatch continue their existing checks.

**LA-08 — consent and observation.** Default reads/validation/preview/apply never
invoke vendors, resolve secrets or infer model support. Existing auth status
subprocesses require separate explicit consent; report version/auth categories
and provenance only. Saving or enabling a Profile never asserts readiness.
Installed, integrated, authenticated and model/effort usable are distinct facts.
Auth observation does not authorize inference or Provider effects; dispatch must
revalidate according to its existing boundary.

**LA-09 — diagnostics/removal.** Reuse #231 Project readiness and existing resolver.
Report known dependent local/Project references and bounded inventory limitations
before removal, without scanning arbitrary repositories or editing declarations.
Missing local Profiles invalidate future dependent resolutions; immutable
Executions retain their original choices. No global Doctor. Requirement-specific
resolution remains separate from coarse Project availability.

**LA-10 — filesystem/recovery.** Reuse protected roots, ACL/mode/ownership checks,
ordered locks, atomic publication and recovery. Preserve canonical pathname
identity across commit/read-back; unknown or replaced state refuses or reports
uncertainty. Extend recovery discovery and inventory for Runtime/Profile protocol
records and the tombstone. The configuration store is the single decoder/selector
for active and retired generations. New readers support active v1 unchanged; old
readers reject the tombstone fail-closed. The FR-026 compatibility window is
forward upgrade to the new reader; downgrade of retired state requires separately
authorized active-v1 restoration first. Freeze new encoding fixtures without
editing old stable snapshots. No naive final-file unlink or automatic destructive repair. Persisted
format changes discovered during implementation require a new reviewed contract
and compatibility window before work resumes.

**LA-11 — conversational ownership (D4).** Proposed ADR-0023 adds `axiom-runtime`
as a thin local configuration surface for both maintained Runtimes. It prepares
safe drafts, validates, previews, obtains exact confirmation, applies and inspects
through canonical application/CLI operations. Human need not author JSON or derive
digests. It cannot edit Project policy, Workflow selection/definitions or execute
Work Items. Existing Workflow readiness guidance remains read-only. No new skill
or catalog is accepted/installed by this document.

## Proposed reconciliation of related accepted specifications

Spec 002 gains LA-01–LA-10 only after explicit acceptance; portable matching
remains unchanged. Spec 007's accepted HD-006/G-2 boundary is preserved; its G-3
observation and G-4 Provider linkage remain independent. Add a reference to the
accepted local authoring contract when accepted, rather than relabeling historical
G-2 blockers as delivered before Evidence.

LA-11/ADR-0023 require a bounded additive exception to Specification 004's closed
active skill inventory as already amended by #303. The 2026-10-08 two-skill
record and #303's three-skill receipts remain historical provenance. Proposed
current inventory would become Project, Work Item, Workflow and Runtime; no
resurrection of the retired Execution/Repository/Artifact/Integration wrappers.
This proposal does not modify accepted inventory text or receipts in place.

## Human decision record still required

Review D1–D5 in the Plan. Accept/reject the exact contract blob independently
from Plan approval; security acceptance and ADR-0023 remain separate. If D1
empty-state migration, D2 credential backend or D3 persisted effort defaults are
selected instead, revise this contract and Plan for renewed review before Tasks
or implementation of affected behavior. No decisions are marked accepted here.
