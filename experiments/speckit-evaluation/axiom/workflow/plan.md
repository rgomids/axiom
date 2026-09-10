# Plan — Documented Command Deprecation Process

## Status

Approved for execution by the phase-2 instruction, constrained by the clarified
specification and frozen operator answers.

## Inputs

- `experiment-output/intake.md`
- `experiment-output/specification.md`
- `experiment-output/clarifications.md`
- `.experiment/operator-answers.md`
- `.experiment/completion-criteria.md`

## Change map

| Path | Action | Responsibility |
|---|---|---|
| `docs/deprecations/README.md` | Create | Process, lifecycle, index, notice, exception, review checklist |
| `docs/deprecations/template.md` | Create | Reusable record structure with required content |
| `README.md` | Update | Make deprecation process discoverable |
| `docs/contributing.md` | Update | Route contributors to the process |
| `CHANGELOG.md` | Update | Record documentation lifecycle change |
| `docs/commands.md` | Preserve | Existing names and descriptions remain byte-for-byte unchanged |
| `experiment-output/*.md` | Create/update | Durable workflow, decision, execution, validation, and review evidence |

## Implementation sequence

1. Create the deprecation process/index and template.
2. Add minimal discoverability links in `README.md` and
   `docs/contributing.md`.
3. Add one `CHANGELOG.md` entry.
4. Confirm `docs/commands.md` retains its pre-change SHA-256.
5. Run narrow deterministic checks, then broader scope and sensitive-content
   checks.
6. Review against every acceptance criterion, architecture policy, security
   policy, and shared completion criterion.
7. Reconcile only affected documentation and record evidence.

## Contract treatment

- New documentation contract: record trigger, lifecycle, required fields,
  authority, notice, exception, and checklist.
- Existing command contract: unchanged.
- Provider contract: none.
- Runtime or executable contract: none.

## Architecture and ADR

No application architecture changes. `experiment-output/decisions.md` records
the ADR threshold analysis. No ADR file is warranted for this local, reversible
documentation convention.

## Data, migration, rollout, and rollback

- Data model or persistence: none.
- Migration: none; no existing deprecation records exist.
- Deployment or operational rollout: none.
- Documentation adoption: contributors use the template for future covered
  changes.
- Rollback: remove new deprecation documents and revert three discoverability
  or lifecycle documentation edits. Existing command documentation remains
  untouched throughout.

## Observability and traceability

No runtime telemetry applies. Durable record fields, lifecycle status, owner,
approval evidence, relevant links, changelog entry, and experiment evidence
provide review traceability.

## Validation strategy

- File presence and exact expected paths.
- Required headings in process and template.
- Internal Markdown-link resolution.
- No trailing whitespace or diff whitespace errors.
- Pre-change versus post-change `docs/commands.md` SHA-256 equality.
- Changed-file scope and Markdown-only implementation.
- No executable bit on new documents.
- Secret-sensitive content scan using available local tools plus heuristic
  checks when a consolidated scanner is unavailable.
- Acceptance-criterion and completion-criterion matrix review.

## Security assessment

- No credentials, provider data, network access, executable content, or
  permission expansion.
- Repository content is treated as public.
- Template uses descriptive placeholders only; it must not request secret
  values.
- Urgent security exception retains rationale, impact, owner, approval, and
  compensating migration guidance without requiring sensitive incident detail.

## Documentation reconciliation

- Product behavior: sample contributor documentation gains a deprecation
  process.
- Contract: documented command deprecation governance added; commands unchanged.
- Architecture, operations, deployment, Axiom product, and agent-runtime docs:
  unaffected.
- `README.md`, `docs/contributing.md`, and `CHANGELOG.md`: update required.
- ADR: not required.

## Risks and controls

- **Residual trigger ambiguity:** limit normative trigger to rename, explicit
  deprecation, and removal; preserve other categories as unresolved.
- **Manual enforcement drift:** keep required fields in one template and mirror
  them in a reviewer checklist; verify both deterministically.
- **Accidental command change:** compare exact file hash before and after.
- **Scope expansion:** inspect all changed and untracked paths before completion.

## Constitution check

- Plan consumes durable intake, specification, and clarification.
- Human answers govern material policy.
- Residual unknowns remain visible.
- Change is smallest coherent documentation-only slice.
- Objective claims have reproducible validation commands.
- No speculative dependency, provider, architecture, or ADR is introduced.
