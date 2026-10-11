# Issue #306 — Mandatory local Runtime/Profile security review

## Status and scope

**Proposed design review; Awaiting Human Approval, reconciled 2026-10-11.**
[Human D1–D5/onboarding direction](https://github.com/rgomids/axiom/issues/306#issuecomment-6108025323)
is confirmed; exact security/format/migration artifacts are not accepted.
This supersedes the prior #306 D3 proposal, not accepted production contracts.
This review does not certify future code. Planning only; no credentials, vendor
login files, environment values or authenticated vendor probes were imported.
The reviewed baseline is main `5d52327b0367ba08d872b50e804df5c7d63a3e89`.
See [Plan](plan-306-local-runtime-profiles.md),
[contract proposal](amendment-306-local-runtime-authoring.md), and
[Spec 007 sole-effort/migration/onboarding proposal](../007-configurable-workflows/amendment-306-profile-effort-onboarding.md).
This reconciliation changes no code/schema/fixture and runs no authenticated probes.

## Assets, effects and trust boundaries

Assets: machine-local configuration and exact generation identity; private root
ownership; immutable Project/Workflow/Execution records; secretless review output;
operator authority; truthful capability/authentication observations.

Untrusted inputs: conversational content/draft JSON, local configuration edited
outside Axiom, replaced filesystem objects, stale previews, unknown protocol
markers, inherited vendor settings/environment and raw status output. None grants
authority. Permitted #306 mutations stop at the exact local configuration and
reviewed Axiom protocol records. No vendor token/session/login-file/API-key handling, login/logout/refresh, Provider write,
process inference, skill/Runtime uninstall or portable configuration modification.

## Verified current controls and gaps

| Boundary | Source / existing control | Gap and proposed control |
| --- | --- | --- |
| Domain wire | `runtimeprofile.Validate/Encode/Decode/Digest`: bounded closed v1, cross-references and deterministic digest | `validReference` permits arbitrary bounded single-line text; model `validText` similarly weak. Separate safe authoring policy before output; no new nonempty credential refs |
| Concurrent publication | `RuntimeProfileStore.Save`: revision + 1 under locks; `publishFile` compares bytes | Store's current bytes need not equal human-reviewed bytes; conditional application/store port compares reviewed digest + identities inside lock |
| Root confinement | `safe_fs.go`: anchored private roots, bounded no-follow reads, ownership/permissions/hardlink checks | Store currently discards anchor identity; keep `AnchoredDirectory` and check `StillAtPath` before commit and after read-back |
| Crash safety | `publication.go`: staged verified bytes, digest markers, rename/fsync, blocked pending state | `recoveryDirectories` omits `runtime-profiles/v1`; integrate exact discovery and inventory. Final removal requires recoverable protocol proof |
| Auth/proof | `AuthReport` safe enums/version/digest; fresh `subscriptionInvocation` preflight rejects credential refs | Saving reference/declared capability cannot imply authentication or usable model/effort; default report unknown |
| Evidence | Existing safe completion/report patterns, binding credential field excluded from JSON | Never serialize configuration wholesale or expose legacy field/error values; capture only approved projection |

`RuntimeProfileStore` confines operations to opened roots, but that alone does
not prove the configured pathname still exposes those roots. Both claims must
be tested independently. The existing
`TestRuntimeProfileStorePreservesRevisionAndCompletePriorGeneration` covers an
interruption retaining complete prior state; it does not certify the proposed
authoring/removal authority protocol.

## Required design controls and verification

| Threat | Required refusal/control | Required future Evidence |
| --- | --- | --- |
| Secret in profile/model/reference or draft | Bounded typed grammar + existing pure safety policy; no credentials accepted; no raw parse error/input echo | Inline token/assignment/URL/escaped path/unknown-key canaries absent from stdout/stderr, completion, preview, logs and Evidence |
| Credential leakage from old valid config | Preserve unchanged historical v1 only for inspection; masked safe report; migration to local v2 explicitly clears the legacy reference under new authority or refuses | Legacy secret-shaped reference never printed; no credentialReference field in authored v2; subscription guard refuses nonempty legacy ref before resolver |
| Malicious custom commands/env | No executable path, argv, credential helper or environment fields in authoring; Runtime/adapter identity closed | Reject shell payloads/custom adapter/provider/environment; zero subprocess/network effects |
| Preview-to-apply drift | Compare target identity/existence/revision/content/candidate/effects under ordered locks | Same revision different bytes; root/file replacement; supported create/retire/recreate ABA; candidate reorder vs actual changes; wrong digest/target/authority |
| CREATE race | Fresh absence + no-replace publication | Concurrent creation conflict and zero overwritten state |
| Partial update/removal | Atomic complete candidate or strict same-file tombstone; monotonic generation through retirement/recreation | Fault injection before/after stage, marker, replacement rename, directory sync and read-back; complete prior or recovery-blocked state |
| Symlink/hardlink/ACL escape | Existing root/file protection plus pathname identity fences; no arbitrary destination | POSIX symlink/hardlink/chmod/root swap and native Windows ACL/reparse/ancestor tests; platform gaps reported honestly |
| Uncertain publication mistaken for success | Return uncertainty/recovery; preserve all unknown objects; require fresh scoped recovery authority | Canonical root replaced after commit; incomplete marker; restart/new process; no blind retry/cleanup |
| Removal harms dependent Projects | Explicit local cascade and disclosed known references; no portable/Execution writes | Two Projects share Profile; before/after byte snapshots; future resolution blocked; no credential uninstall/logout |
| Fabricated capability/effort/auth | Declared != proven; exact model/executable/version proof; fresh auth action/dispatch | Stale/unknown/unsupported proof and negative auth block; vendor effort tokens never translated |
| Reused auth authority grants inference | Separate status observation consent; no inference or Provider mutation | Status-runner spy and separate dispatch gate; report status-only Evidence as usability unproven |
| Competing Workflow/effort overrides | Required Profile v2 effort; Workflow v2 fields/capabilities/WorkUnit/env/argv cannot select another effort; versioned immutable snapshot | Same Runtime/model with different Profile efforts, reserved capability injection, snapshot versus controls mismatch and inherited setting conflicts refuse |
| Silent legacy conversion | v1 inspection remains exact; explicit local/Workflow migration; no inferred defaults or new snapshot on old Execution | Original bytes/digests retained, per-Agent legacy effort comparison, old approval refusal, legacy new-admission/retry blocker |
| Onboarding manufactures readiness | Passive detection only; no vendor calls without consent; optional templates with no fixed model/effort; separate auth/model/effort evidence | None/Codex/Claude/both, valid prior state unchanged, one/three Profiles, zero writes/probes, unproven facts block, CLI noninteractive |
| default-sdd chooses initiating vendor | policy-default retains exact Project intersection; explicit candidate assignment or portable preference | Codex-only/Claude-only unique choices; both ambiguous until chosen; no local preference widening, no hidden workflow selection |
| Hidden registry/inventory divergence | Existing store/resolver only; extend recovery/compatibility guards and catalog | Every writer recognized; supported record loads canonically; append-only stable corpus; skill/CLI parity |

A digest does not redact a secret; reject secret-bearing authoring input before
generating public review data. Safety matching reduces accidental exposure but
cannot prove every arbitrary token is nonsecret; no credential value input is
supported. The safe projection must omit unsafe historical model/reference
values and return a logical repair blocker rather than asserting display safety.

## Vendor-managed authentication review (confirmed D2, proposed exact design)

MVP uses only already installed/authenticated Codex/Claude CLIs and supported
native subscriptions. No Axiom credential backend, token/session/API-key capture,
login-file copy/import or login/logout/refresh operation. Authentication observation
and native model/effort probes require specific authority; none occurs here.

Local active format 2 forbids credentialReference authoring and does not carry
that field. Old format 1 remains canonically readable: nonempty legacy references
are masked/unresolved and continue subscription-blocked. A v1→v2 migration explicitly
clears the legacy field as a reviewed local effect or refuses; it never copies it
into drafts/Evidence or manipulates the referenced vendor state. Existing generic
ResolveEnvironment support is not adopted as a credential mechanism for this scope.

Existing preflight may inspect bounded vendor *configuration* solely to reject
unsafe invocation overrides under its accepted authority; this is not authority
to manage vendor login files/sessions. New guidance must route each probe through
specific consent, and report only safe categories/digests/approved version tokens.

## Sole effort owner and immutable snapshot review (D3)

Local Profile v2 is the only mutable effort owner. Workflow v2 references it;
capability-token, WorkUnit-control, argv/env and vendor-setting overrides cannot
create a second selection channel. Profile default mode is an explicit no-Axiom-
override choice, not proof of a specific vendor level. External explicit effort selectors must block rather than supersede the snapshot.
Native model defaults with no selector remain vendor-internal when the Profile
explicitly requests no override; no concrete level is asserted. PE-05a requires
new bounded effort-source inspection: recognized argv/env/config scopes and
version-proven precedence, unreadable/unknown refusal, no login-file access,
setting mutation or raw output. Current auth preflight is not such a proof source.
Include default-mode Claude effort env/Codex configured effort and duplicate/
conflicting selectors in future acceptance, not just explicit-mode adapter tests.

Versioned resolution binds Profile effort/model, profile/configuration digest,
Runtime/executable/version/mode and authoritative proof. Freeze the tuple in the
new Execution/stage/graph publication; derive controls from it. Each dispatch
rechecks exact tuple and fresh authentication before any attempt/effect. Profile
changes invalidate previews; they never update already frozen controls or legacy
bindings. Supported explicit effort needs per-model/per-mode proof; Runtime-wide,
auth-only, template defaults and declared-only facts are insufficient.

Legacy v1 configs/Workflow requests remain readable with their original meaning.
No automatic effort inference or old-record rewriting. New admissions require
explicit migration; historical continuation without a v2 snapshot blocks with
reconciliation guidance. This legacy continuation policy and version/window need
independent exact acceptance. G-3 proof acquisition and #276 dispatch composition
remain separate owners; #306 planning cannot supply their operational Evidence.

## Residual risks and blockers

Current production observation proves executable presence/Axiom integration,
not positive model/effort usability. G-3 remains independent. Auth output is
vendor-version dependent, so unsupported/unknown states block and any real probe
needs separate consent and scoped Evidence. Dependent Project inventory may be
incomplete; disclose limits rather than claim no users of a Profile.

Tombstone retirement/recreation protocol, anchor fences and recovery discovery need executable
proof during implementation. If existing primitives cannot implement the approved
removal semantics safely, stop that work and amend Plan/contract; do not weaken
the boundary. Human acceptance of this review approves a design baseline only;
final code still requires independent security review and exact-head tests.

## Planning validation

Read-only source inspection established the controls/gaps above; two bounded
read-only investigations covered contracts and security, then the Orchestrator
spot-checked core functions. No subagent accepted the design.

Prior baseline Planning checks executed on 2026-10-11 at the proposal culminating
in commit `2096509` (historical validation; **not rerun or new-behavior proof** in
this reconciliation):

- `./scripts/validate-repository.sh .` — PASS; maintainer behavioral vendor
  scenarios explicitly SKIPPED/UNVERIFIED by that validator.
- `go test ./internal/runtimeprofile ./internal/runtimeapplication ./internal/runtimeadapter ./internal/local ./internal/projectapp ./internal/cli ./cmd/lingo` — PASS
  for all seven packages, against the unchanged production baseline.
- Local Markdown target check — PASS for 19 links in the four new/proposed
  artifacts; no external URL reachability claim.
- `git diff --check` / `git diff --cached --check` — PASS.
- `./scripts/check-sensitive-files.sh --staged .` — PASS.
- `gitleaks dir docs/specifications/002-lingo-project-initialization --no-banner --redact`
  and scoped ADR scan — PASS, no detected leaks.

Current reconciliation checks and source-baseline hashes are in
[documentary Evidence](evidence-306-planning-reconciliation.md); the PR records
exact revision/blob identities. Validation covers documentation/unchanged contract
files, not model behavior, vendor authentication, new schemas or migration code. Real vendor auth, model/effort execution, Linux/Windows native runs and
new removal/concurrency behavior are **not verified in Planning**.
