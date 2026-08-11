# Research: Documented Command Deprecation Process

No external research, provider access, or network access was required. Decisions derive from the
frozen brief, one frozen human-answer batch, the constitution, and current repository files.

## Decision 1: Repository-local artifact set

**Decision**: Use `docs/deprecations/README.md` for process, lifecycle, index, and reviewer checklist;
use `docs/deprecations/template.md` for reusable record content; use individual
`docs/deprecations/NNNN-short-name.md` files only when actual deprecations exist.

**Rationale**: Matches frozen location and naming while avoiding a fictional deprecation record.

**Alternatives considered**: One monolithic registry; provider issues or pull-request templates.
Rejected because the frozen answer requires an index and template, and provider dependence is
prohibited.

## Decision 2: Lifecycle authority

**Decision**: Document `Proposed -> Approved -> Deprecated -> Removed`. A proposed record changes no
command status. A repository maintainer approves `Approved`; command owners execute later
transitions.

**Rationale**: Exact frozen human answer; preserves explicit human approval.

**Alternatives considered**: Prior recommended three-stage lifecycle and two-maintainer approval.
Rejected because neither was selected by the frozen human answer.

## Decision 3: Notice and urgent exception

**Decision**: Normal removal follows at least one published release containing the deprecation
notice. Urgent security removal may bypass that minimum only when its record states rationale,
impact, owner, approval, and compensating migration guidance.

**Rationale**: Exact frozen human answer; provides deterministic review criteria without provider
automation.

**Alternatives considered**: Fixed calendar periods and provider-specific release checks. Rejected
because they replace the human answer or violate provider neutrality.

## Decision 4: Validation boundary

**Decision**: Publish a reviewer checklist and runnable one-off local commands. Do not add an
executable validator, dependency, CI workflow, credential, or network requirement.

**Rationale**: Exact frozen enforcement answer and constitution constraint.

**Alternatives considered**: Validation script, CI job, hosting-provider gate. Rejected as explicitly
out of scope.

## Decision 5: Residual ambiguity treatment

**Decision**: Require a record for an explicit documented-command deprecation, including later
removal. State that independent triggers for rename, replacement, or incompatible behavior remain
unresolved. Require the `NNNN-short-name.md` shape but do not invent a numbering allocator. Require
release evidence in relevant links without defining a provider or publication mechanism.

**Rationale**: Enables the frozen core scenario while preserving unanswered boundaries honestly.

**Alternatives considered**: Adopt prior multiple-choice recommendations or silently infer lifecycle
policy. Rejected because operator answers must be consumed exactly.

## ADR Assessment

No ADR. Decisions affect documentation information architecture and review policy only; no runtime,
data store, executable interface, dependency, or provider boundary is introduced. Long-term policy
choices are already explicit in the specification and this research artifact.
