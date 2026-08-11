# Axiom execution — phase 2

The human clarification batch is now durably available at
`.experiment/operator-answers.md`. Consume it without changing it.

Continue exclusively with the current Axiom workflow from the existing phase-1
artifacts:

```text
clarify -> plan -> tasks -> implement -> review -> reconcile
```

Requirements:

- First update the specification and clarification record to incorporate the
  answers while preserving what was originally asked.
- Create `experiment-output/plan.md`, `experiment-output/tasks.md`,
  `experiment-output/decisions.md`, `experiment-output/validation.md`,
  `experiment-output/review.md`, and `experiment-output/execution.md`.
- Implement the smallest documentation-only change in the sample repository.
- Apply Axiom architecture-decision, implementation, review, and documentation
  skills where routed. Do not improve or rewrite the Axiom workflow itself.
- Preserve both existing commands and their current descriptions.
- Add no application code, executable validator, dependency, CI, provider,
  credential, or network requirement.
- Run deterministic checks for acceptance, links, required headings,
  whitespace, scope, and secret-sensitive content where tools exist.
- Record exact commands, exit outcomes, difficulties, ambiguities, approximate
  elapsed time only if directly measurable, human interaction count, and any
  unverified claim.
- Decide explicitly whether this sample change warrants an ADR. Do not create a
  speculative ADR merely to fill the workflow.
- Do not commit, push, or access a provider.

Finish only after comparing the result against every shared completion
criterion. Report blockers honestly.
