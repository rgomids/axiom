# Spec-Kit execution — phase 2

The single human clarification batch is now available at
`.experiment/operator-answers.md`. It was frozen before either approach ran.
Consume it exactly; do not replace it with the recommended multiple-choice
answers from the prior question set. Preserve any residual ambiguity honestly.

Continue using only official Spec-Kit `v0.16.2` skills. Run every remaining
command boundary in the official full sequence:

1. Finish `$speckit-clarify` and encode the human answers into `spec.md`.
2. Run `$speckit-plan` for the smallest documentation-only realization.
3. Run `$speckit-checklist` for requirement quality.
4. Run `$speckit-tasks`.
5. Run read-only `$speckit-analyze`; fix material artifact inconsistencies only
   at the owning phase, then re-run analysis if required.
6. Run `$speckit-implement`.
7. Run `$speckit-converge`; if it appends legitimate missing work, implement it
   and converge once more. Do not add scope merely to force convergence.

Controls:

- Preserve official Spec-Kit artifact paths and templates.
- Implement only the frozen documentation scenario; do not improve or customize
  Spec-Kit itself.
- Preserve both existing commands and descriptions.
- Add no application code, executable validator, dependency, CI, provider,
  credential, or network requirement.
- Update `CHANGELOG.md` as required by shared completion criteria.
- Use deterministic local commands for links, required headings, whitespace,
  scope, and sensitive content where available; report unavailable scanners.
- State explicitly what architecture decisions were surfaced and whether any ADR
  exists or is absent in the official output.
- Record human interaction count as one batched response.
- Do not commit, push, create issues, or access a provider.

Finish only after checking every shared completion criterion. Final response
must list command statuses, artifacts, validations, difficulties, ambiguities,
human intervention, and unverified claims.
