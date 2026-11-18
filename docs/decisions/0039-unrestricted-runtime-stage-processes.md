# ADR-0039 — Unrestricted Runtime Stage processes

## Status

Proposed — acceptance unverified (CORRECT-403-01, 2026-11-18 UTC).
Primary human authority for the global permission change was not located;
CR-403-F2 remains EVIDENCE_GAP and CR-403-F1 remains blocked. See the
[bounded investigation and decision gate](../specifications/015-minimum-viable-operational-workflow/evidence-axm-12-e2e-v0242.md#correct-403-01--authority-reconciliation).

Historical status recorded in commit `204740d` (not proof of approval):
> Accepted 2026-11-17. The maintainer explicitly selected this default for every Stage in
> every Project during the AXM-12 follow-up.

The text below describes the PR implementation and proposed decision. It does
not amend accepted contracts or authorize operation pending the material gate.

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
The launcher still filters inherited environment variables. Host and vendor
managed restrictions can still prevent a launch or operation.

## Alternatives considered

- Vendor prompts disabled with tool and workspace restrictions. Confinement
  is stronger and some supported commands can be denied.
- Unrestricted mode for AXM-12 only. This limits exposure to the lab and
  leaves a different default for other Projects.

## Consequences

Use a dedicated operating-system identity or externally isolated host where
Runtime processes must be prevented from reaching unrelated resources.
Tool permission settings in an independently started operator session are
configured when launching that session.

## Proposed supersession (not effective)

- [ADR-0009 — Parent/child Execution Graph](0009-parent-child-execution-graph.md#child-scope-authority-and-resolution)
  only the effect and credential ceiling as confinement of Codex and Claude
  Stage processes. Axiom's own target, authority, and validation checks remain.

Spec 015 target/write/message boundaries and ADR-0037 decisions 5–6 also
require explicit reconciliation if this proposal is accepted. No supersession
is effective without a dated primary human decision and reciprocal annotations.
