# Decisions — Documented Command Deprecation Process

## Human-approved policy inputs

Source: `.experiment/operator-answers.md`.

- **D-001:** Lifecycle is
  `Proposed -> Approved -> Deprecated -> Removed`; `Proposed` changes no command
  status; maintainer approves `Approved`; command owners execute later
  transitions.
- **D-002:** Records use `docs/deprecations/NNNN-short-name.md`, an index at
  `docs/deprecations/README.md`, and a reusable template at
  `docs/deprecations/template.md`.
- **D-003:** Required content is status, affected command, replacement or
  explicit absence, rationale, affected users, migration steps, notice or
  release target, rollback plan, owner, approval evidence, and relevant links.
- **D-004:** Normal removal follows at least one published release containing
  notice; urgent security removal may bypass this only with recorded rationale,
  impact, owner, approval, and compensating migration guidance.
- **D-005:** Enforcement remains manual in this slice, supported by a reviewer
  checklist and deterministic repository checks; no executable validator or CI.

## ADR assessment

### Decision

**No ADR warranted.** Record the repository-local documentation convention and
its evidence in the specification, process, and this execution record.

### Constraints

- Documentation-only, provider-neutral, local-checkout workflow.
- No runtime, persistence, integration, CI, dependency, or command behavior.
- Small sample repository with no existing deprecation records or migration.

### Alternatives considered

| Alternative | Complexity | Coupling and portability | Reversibility | Migration impact |
|---|---|---|---|---|
| Create a new ADR for the Markdown process | Higher documentary overhead | Risks presenting sample convention as architecture | ADR history makes a routine choice appear durable | None now; future process changes require ADR lifecycle |
| Record decision in specification/process/evidence | Low | Repository-local and provider-neutral | Easy to revise through later specification change | No existing records to migrate |
| Design a general release-management ADR | High and speculative | Couples command deprecation to an unvalidated broader model | Expensive and misleading | Invents future migration concerns |

### Trade-off and recommendation

Second alternative selected. It preserves traceability without promoting a
routine, reversible documentation convention into application architecture.
No persistence technology, public runtime contract, trust boundary, deployment
topology, project/repository model, provider abstraction, data ownership, or
irreversible dependency changes.

### Revisit when

- A runtime or executable enforcement mechanism is proposed.
- Deprecation state becomes shared across repositories or providers.
- Release orchestration, persistence, approval identity, or synchronization
  becomes an Axiom architecture concern.
- Existing records require a costly schema migration.

## Bounded interpretation decisions

- **D-006:** Implement only trigger categories explicit in the frozen scenario
  and lifecycle: rename, mark deprecated, and remove. Other categories remain
  open.
- **D-007:** Because urgent exception approval is required and the lifecycle
  defines the repository maintainer as the approval authority, maintainer
  approval to `Approved` must explicitly cover the exception.
- **D-008:** Because migration steps and rollback plan are mandatory and no
  omission rule was approved, missing either blocks approval.

These interpretations minimize scope. They do not close the residual questions
recorded in `clarifications.md` and `specification.md`.
