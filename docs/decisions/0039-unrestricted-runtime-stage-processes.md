# ADR-0039 — Unrestricted Runtime Stage processes

## Status

Accepted on 2026-11-18 at 12:11 BRT by Rafael's
[formal ADR acceptance](https://linear.app/rgomids/issue/AXM-12#comment-49d0899e).
The separate [material policy approval](https://linear.app/rgomids/issue/AXM-12#comment-343815d4)
at 12:04 BRT covers Codex and Claude, writing and read-only Stages, all Projects,
with awareness of host-user filesystem, network and credential access outside
worktrees. This is prospective authority, not PR approval, merge, release,
external delivery or AXM-12/#278/MVP acceptance.

CORRECT-403-01 found no original approval: ORIGINAL_APPROVAL_NOT_FOUND remains
historical fact. The [2026-11-17 drafting authority](https://linear.app/rgomids/issue/AXM-12#comment-4ce47d42)
only authorized proposals. CORRECT-403-02 prepared the bounded amendment below;
CORRECT-403-03 makes it effective under the two decisions above. No retroactive
decision.approved is claimed. See the
[authority investigation](../specifications/015-minimum-viable-operational-workflow/evidence-axm-12-e2e-v0242.md#correct-403-01--authority-reconciliation).

Historical status recorded in commit `204740d` (**UNVERIFIED**, not proof of approval):
> Accepted 2026-11-17. The maintainer explicitly selected this default for every Stage in
> every Project during the AXM-12 follow-up.

The decision below is effective prospectively within the stated scope.

## Context

Codex and Claude run non-interactive Stage processes. Vendor permission
policies can deny tools used during a run. An operator session that starts an
Axiom Execution is launched separately and uses its own vendor permissions.

## Decision

The Stage launcher requests unrestricted process permissions with Codex
`--dangerously-bypass-approvals-and-sandbox` and Claude Code
`--dangerously-skip-permissions`. This applies to both writing and read-only
Stages in all Projects. The Attempt's working directory, inputs, outputs,
worktree allocation, and Axiom's authorization checks remain bound to the
Execution. An Axiom operation that requires a human approval or delivery
authority still requires it.

Runtime processes can access files, tools, network and credentials available
to their operating-system user, including outside the assigned workspace.
The Stage effect envelope records the intended scope and governs actions that
Axiom itself performs. It does not confine a Runtime process or its subprocesses.
The launcher still filters inherited environment variables; this filter and
Axiom permissions are not an OS security barrier. Host and vendor
managed restrictions can still prevent a launch or operation.

## Alternatives considered

- Vendor prompts disabled with tool and workspace restrictions. Confinement
  is stronger and some supported commands can be denied.
- Unrestricted mode for AXM-12 only. This limits exposure to the lab and
  leaves a different default for other Projects.

## Consequences

`RepositoryWrite=false`, read-only labels, an Effect envelope and worktree
selection do not impose OS isolation with these vendor flags. An unrestricted
process can read credentials from files or other host resources despite env
filtering, write other directories and use the host user's network access.
`danger-full-access` is not a permanent mitigation.

Where a boundary is required, use a dedicated host/OS user with access limited
to the authorized resources, or a real external sandbox enforcing that boundary.
A dedicated identity alone does not isolate network or shared readable files;
residual access must be assessed operationally.
Tool permission settings in an independently started operator session are
configured when launching that session.

## Supersedes

Effective on acceptance, only these fragments are partially superseded:

- [Spec 015](../specifications/015-minimum-viable-operational-workflow/spec.md#runtime-process-boundary--accepted-amendment-correct-403-03):
  Non-goals target-boundary bullet; Architecture target effect-bound enforcement
  paragraph; Normal operation items 4 and 6 and permission/validation-command
  paragraphs; Parallel work and integration; Safety proof obligations bullets 2
  and 6; Acceptance negative-test denial clause. Limit their target/effect
  enforcement guarantee to Axiom-mediated actions, without granting direct
  process effects or claiming process confinement.
- [ADR-0037](0037-one-canonical-execution-for-new-runs.md#decision):
  Context's exact target/effect authority and Decision 5's real permission,
  target and effect checks, with Decision 6's message authority boundary retained.
  Axiom-mediated checks remain mandatory; they are not OS process isolation.
- [ADR-0009](0009-parent-child-execution-graph.md#child-scope-authority-and-resolution):
  decisions 6–7 and 9 only insofar as effect/credential ceilings imply direct
  Codex/Claude process confinement. Credential non-copying, exact mediated
  authority and validation remain mandatory.

Frozen invocation/target identity, Execution ownership, cancellation limits,
single-writer fencing, integration, unknown-effect recovery/no blind replay,
exact grants, messages and external delivery checks remain unchanged.
The two human decisions above supply prospective material and formal authority.
Structural validation does not establish implementation or product acceptance.
