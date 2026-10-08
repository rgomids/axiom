# Issue #265 — Windows default onboarding

## Status and authority

Implementation of a working default onboarding flow was requested on
2026-10-08. The requester selected consent-gated repair of necessary directories.
[ADR-0019](../../decisions/0019-windows-default-onboarding-and-permission-repair.md)
records this bounded decision. PR #266 is being expanded from documentation
to executable default onboarding; completion requires the Evidence below.

## Problem and outcome

The documented default bootstrap can fail on an eligible Windows client because
AppData ancestors or existing Runtime skill roots fail the filesystem boundary.
Selecting an isolated skill root can install the binary but does not configure
Runtime discovery. The default command must prepare an installation that remains
usable from a new shell without manual root overrides, and configure discoverable
skills for installed supported Runtimes through an explicit onboarding step.

## Required behavior

1. Before publishing a binary, resolve and inspect binary, receipt, state,
   Project and detected Runtime integration roots. Report the actual failed root
   and rule; do not report complete onboarding while a required root is unusable.
2. Use a deterministic private user-local destination that avoids a rejected
   AppData ancestor, without treating an unsafe explicit destination as permission
   to silently redirect. The bootstrap, CLI and subsequent shells must resolve
   the same roots. Never abandon or overwrite an existing installation or state.
3. Remove the need for users to supply environment overrides for the supported
   default flow. Preserve explicit overrides for deliberate isolated tests.
4. Configure Runtime skills in discoverable standard locations. An alternate
   arbitrary skill directory is not successful Runtime integration.
5. Preserve non-Axiom skills and configuration, credentials, unrelated ACEs and
   installation provenance. No elevation, policy bypass or trust of sandbox
   principals by assumption.
6. Treat modifying existing directory security as a separately specified repair
   operation, with verified identities, bounded targets, backup, recovery and
   postcondition checks. Existing ADR-0010 preservation remains authoritative
   until the repair decision is accepted.
7. Present usable PATH configuration and the exact `axiom version` command.
   Clearly report what works immediately and what requires a new shell.
8. Refuse genuinely unsupported or policy-blocked environments with actionable
   diagnostics. Universal success on arbitrary Windows machines is not claimed.

## Permission-repair decision

Option A (selected, 2026-10-08): preflight and request authorization for a bounded ACL
repair when existing Runtime roots fail. Ordinary safe profiles install without
the prompt. This preserves control of other tools' storage but adds one decision
on affected profiles.

Option B (rejected): installer invocation authorizes bounded ACL repair, with saved ACLs
and recovery information. This removes the prompt but changes the accepted
existing-ACL policy and can affect sandbox access to Runtime configuration.

Neither option permits changing AppData or the entire user profile. Repair must
refuse an unsafe ancestor outside the authorized Runtime integration scope.

## Validation and acceptance

Current executable results and remaining acceptance are recorded in
[issue-265 Evidence](evidence-issue-265.md).

- Fresh eligible Windows account: default bootstrap, version, first-run,
  discoverable Codex/Claude skills, and repeat installation.
- Affected profile: unsafe AppData ancestor and unsafe existing skill root;
  verify bounded recovery without manual environment overrides.
- New shell: identical resolved state and installed binary location.
- Existing installations and state: no silent migration, loss or orphaning.
- Negative matrix: explicit unsafe paths, other owners, reparse points, foreign
  content, policy-denied writes, changed identities, interrupted repairs and
  concurrent invocations preserve data and fail with the concrete cause.
- Existing unrelated skills, profile/AppData ACLs and credentials remain intact.
- Linux/macOS behavior and release provenance remain unchanged.

## Implementation plan

1. Preserve ADR-0010's boundary and record the fresh default-root change and
   explicit repair exception through ADR-0019 and this amendment.
2. Implement one default-root resolution contract shared by bootstrap and CLI,
   including legacy-install detection. Deterministic defaults must work from
   subsequent shells without persisted root overrides. The requester separately
   authorized adding the binary directory to persistent user PATH; preserve raw
   entries and registry type, avoid duplicates and provide `-SessionOnly`.
3. Add complete preflight and Runtime repair preview/recovery as selected above.
4. Test native default onboarding and reinstall with synthetic ACL fixtures,
   then validate the affected host with the selected authority.
5. Replace the isolated workaround as the primary remediation in public docs
   only after executable Evidence establishes the normal supported flow.

No new dependency or multi-agent orchestration is required by this plan.
