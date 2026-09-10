# Artifact Inventory

## Frozen inputs

- `../scenario/brief.md`
- `../scenario/operator-answers.md`
- `../scenario/control.md`
- `../scenario/completion-criteria.md`
- `../scenario/checksums.sha256`
- `../scenario/seed/`

## Axiom

- Three phase-1 artifacts: intake, specification, clarifications
- Six phase-2 evidence artifacts: plan, tasks, decisions, validation, review,
  execution
- Five changed implementation files, plus the unchanged command-reference
  baseline, copied under `../axiom/implementation/`
- Exact prompts and final phase responses

## Spec-Kit

- Constitution
- Feature `spec.md`
- Requirements checklist and focused checklist
- `plan.md`, `research.md`, `data-model.md`, `quickstart.md`, `tasks.md`
- Five changed implementation files, plus the unchanged command-reference
  baseline, copied under `../speckit/implementation/`
- Initialization manifests, exact prompts, and final phase responses

## Failures preserved

- Axiom: one rejected temporary-file command, two failed final count commands,
  unavailable `gitleaks`, and one minor semantic finding corrected.
- Spec-Kit: first `analyze` pass found one high and three medium issues;
  unavailable external scanners; incomplete generated checklists remained
  explicit; three ambiguities remained unresolved.

Raw Codex JSONL and provider/tool diagnostics are not retained. They contained
environment-owned memory and were not needed to reproduce the public artifact
comparison. Token totals and final execution results were extracted before
removal.

The first cleanup command using `rm` was rejected by the execution guard and did
not run. Exact scratch workspaces, upstream clone, and raw logs were then moved
to `/Users/rgomids/.Trash/axiom-speckit-eval-scratch-20260811/`; they remain
locally recoverable and are not part of the PR.
