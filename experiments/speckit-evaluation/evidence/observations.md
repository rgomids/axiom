# Controlled Observations

## Shared result

Both approaches completed the same documentation-only scenario with one human
clarification batch, five implementation files, unchanged command
descriptions, no dependency, no provider mutation, no application code, and an
explicit no-ADR conclusion.

## Axiom observations

- Intake explicitly classified facts, requirements, assumptions, unknowns, and
  non-goals before specification.
- Clarification stopped the flow on five material questions.
- Plan, task, decision, validation, severity-based review, and documentation
  reconciliation evidence were direct and tailored to Axiom concepts.
- Validation was strong and unusually transparent about rejected or failed
  commands.
- Review found and corrected one minor overreach.
- Review missed a more material consistency problem: the implementation said
  `NNNN` is the "next unused" identifier and treated rename as a covered trigger
  even though the frozen human answer did not decide allocation or full trigger
  coverage. Axiom's own decision file labeled these as bounded interpretations,
  but the final 11/11 pass overstated conformance.
- The workflow had no built-in cross-artifact analyzer or convergence command;
  correctness depended on the same agent authoring and reviewing custom files.

## Spec-Kit observations

- Official templates created a constitution, prioritized user stories,
  measurable outcomes, design research, information model, quickstart, and
  ordered tasks without custom workflow-file design.
- Clarify presented five structured questions with recommended choices and
  accepted the frozen answers even when they differed.
- The first read-only analyze pass found one high and three medium issues,
  including the exact unsupported `NNNN` sequence inference that survived the
  Axiom review. Corrections returned to the owning artifacts; second analysis
  reported no material inconsistency.
- Converge checked specification, plan, constitution, tasks, and implementation
  and appended no work.
- Generated artifacts were more numerous and the implementation process was
  more verbose for a small documentation change.
- The generated requirements checklists remained incomplete. Explicit user
  instruction to run the full sequence allowed implementation to proceed; the
  gate was procedural, not an unconditionally deterministic blocker.
- Three residual ambiguities remained honest: trigger coverage beyond explicit
  deprecation, `NNNN` allocation/collision authority, and proof of a published
  notice-bearing release.

## Fairness limits

- One small documentation scenario cannot prove fit for application code,
  security-critical changes, multi-repository Projects, upgrades, or long-lived
  documentation drift.
- Both executions were sequential and the active operator observed Axiom first.
  Frozen inputs and prompts prevented deliberate post-observation improvement,
  but model-level independence is not provable.
- The Axiom workspace included current Axiom product context; the Spec-Kit
  workspace included generated Spec-Kit infrastructure. This is the intended
  treatment difference, not identical prompt size.
- Same user-level Codex memory loaded in both runs and may have influenced both.
