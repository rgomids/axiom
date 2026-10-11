# Issue #306 — Mandatory local Runtime/Profile security review

## Status and scope

**Proposed design review; awaiting independent human acceptance, 2026-10-11.**
This review does not certify future code. Planning only; no credentials, vendor
login files, environment values or authenticated vendor probes were imported.
The reviewed baseline is main `5d52327b0367ba08d872b50e804df5c7d63a3e89`.
See [Plan](plan-306-local-runtime-profiles.md) and
[contract proposal](amendment-306-local-runtime-authoring.md).

## Assets, effects and trust boundaries

Assets: machine-local configuration and exact generation identity; private root
ownership; immutable Project/Workflow/Execution records; secretless review output;
operator authority; truthful capability/authentication observations.

Untrusted inputs: conversational content/draft JSON, local configuration edited
outside Axiom, replaced filesystem objects, stale previews, unknown protocol
markers, inherited vendor settings/environment and raw status output. None grants
authority. Permitted #306 mutations stop at the exact local configuration and
reviewed Axiom protocol records. No vendor login/logout/refresh, Provider write,
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
| Credential leakage from old valid config | Read internally only; masked safe report; unchanged legacy refs may survive unrelated edit without serialization into draft; explicit clear uses new authority | Legacy secret-shaped reference never printed; empty authored ref; subscription guard refuses nonempty ref before resolver |
| Malicious custom commands/env | No executable path, argv, credential helper or environment fields in authoring; Runtime/adapter identity closed | Reject shell payloads/custom adapter/provider/environment; zero subprocess/network effects |
| Preview-to-apply drift | Compare target identity/existence/revision/content/candidate/effects under ordered locks | Same revision different bytes; root/file replacement; supported create/retire/recreate ABA; candidate reorder vs actual changes; wrong digest/target/authority |
| CREATE race | Fresh absence + no-replace publication | Concurrent creation conflict and zero overwritten state |
| Partial update/removal | Atomic complete candidate or strict same-file tombstone; monotonic generation through retirement/recreation | Fault injection before/after stage, marker, replacement rename, directory sync and read-back; complete prior or recovery-blocked state |
| Symlink/hardlink/ACL escape | Existing root/file protection plus pathname identity fences; no arbitrary destination | POSIX symlink/hardlink/chmod/root swap and native Windows ACL/reparse/ancestor tests; platform gaps reported honestly |
| Uncertain publication mistaken for success | Return uncertainty/recovery; preserve all unknown objects; require fresh scoped recovery authority | Canonical root replaced after commit; incomplete marker; restart/new process; no blind retry/cleanup |
| Removal harms dependent Projects | Explicit local cascade and disclosed known references; no portable/Execution writes | Two Projects share Profile; before/after byte snapshots; future resolution blocked; no credential uninstall/logout |
| Fabricated capability/effort/auth | Declared != proven; exact model/executable/version proof; fresh auth action/dispatch | Stale/unknown/unsupported proof and negative auth block; vendor effort tokens never translated |
| Reused auth authority grants inference | Separate status observation consent; no inference or Provider mutation | Status-runner spy and separate dispatch gate; report status-only Evidence as usability unproven |
| Hidden registry/inventory divergence | Existing store/resolver only; extend recovery/compatibility guards and catalog | Every writer recognized; supported record loads canonically; append-only stable corpus; skill/CLI parity |

A digest does not redact a secret; reject secret-bearing authoring input before
generating public review data. Safety matching reduces accidental exposure but
cannot prove every arbitrary token is nonsecret; no credential value input is
supported. The safe projection must omit unsafe historical model/reference
values and return a logical repair blocker rather than asserting display safety.

## Credential-reference review outcome (proposed D2)

Recommend no new reference backend in #306. Vendor-managed subscription state
stays vendor-owned. Local authoring binds supported Runtime/Profiles with empty
`CredentialReference`. Existing `runtimeadapter.ResolveEnvironment` support for
opaque references is not sufficient security approval for authoring a source.
Subscription dispatch explicitly refuses such references before resolution.
This choice avoids inventing a keychain/env/file-secret authority mechanism.

Alternative nonempty references require a precise source-qualified grammar,
reviewed backend and storage/effect/permission policy, redacted locator handling,
revocation behavior and separate dispatch compatibility. No such mechanism is
approved here. D2 must be accepted explicitly, not inferred from this review.

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

Planning checks executed locally on 2026-10-11:

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

Final exact revision and artifact blob identities are recorded in the PR/Issue
progress comment. These checks validate documentation/baseline, not new production
behavior. Real vendor auth, model/effort execution, Linux/Windows native runs and
new removal/concurrency behavior are **not verified in Planning**.
