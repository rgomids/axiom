# Axiom and GitHub Spec-Kit Comparison

## Result

Both workflows completed the same frozen documentation scenario. Spec-Kit
provided stronger built-in cross-artifact analysis and convergence; Axiom
provided clearer domain classification, leaner Axiom-specific governance, and a
better fit for Project-level traceability beyond a feature directory. Neither
workflow alone solves Axiom's multi-repository, Execution, Evidence, Release,
provider, or living-document ambitions.

The experiment supports **C — conceptual compatibility** as the leading
strategy. This is a Proposed recommendation, not an accepted decision or an
implementation authorization.

## Controlled comparison

| Criterion | Axiom | Spec-Kit |
|---|---|---|
| Intake | Dedicated durable intake classified facts, requirements, assumptions, unknowns, and non-goals before specification. [Evidence](../axiom/workflow/intake.md) | No separate core intake phase; the brief fed constitution and specify. `spec.md` captured assumptions/non-goals, but repository-state classification was less explicit. [Evidence](../speckit/official/specs/001-command-deprecation-process/spec.md) |
| Specification | Behavior-first spec with scenarios, invariants, numbered requirements, acceptance criteria, edge cases, and constitution check. Tailored to Axiom concepts but custom-authored. [Evidence](../axiom/workflow/specification.md) | Official template produced prioritized user stories, acceptance scenarios, entities, functional requirements, measurable outcomes, assumptions, and unresolved decisions. It also introduced unmeasured 10-minute and two-reviewer goals, later labeled unverified. [Evidence](../speckit/official/specs/001-command-deprecation-process/spec.md) |
| Clarification | Five material questions, one batch, hard stop before plan. No recommended answer anchoring. [Evidence](../axiom/workflow/clarifications.md) | Five structured questions with rationale, recommended options, and concise answer form; hard stop before plan. Recommendations improve usability but can anchor decisions. Frozen answers were accepted when different. [Evidence](../speckit/phase-1-final.md) |
| Planning | Lean path/action/responsibility map, contract treatment, architecture, rollout/rollback, validation, security, docs, risks, and constitution check. [Evidence](../axiom/workflow/plan.md) | Standard plan plus research, information model, quickstart, and constitution gate. More comprehensive and reusable; heavier for a five-file documentation change. [Plan](../speckit/official/specs/001-command-deprecation-process/plan.md), [research](../speckit/official/specs/001-command-deprecation-process/research.md) |
| Task decomposition | Seven execution/evidence tasks with objective, scope, dependencies, evidence, and repository impact. Strong execution trace, weak user-story mapping. [Evidence](../axiom/workflow/tasks.md) | Eighteen ordered, path-specific, story-mapped tasks with dependencies, checkpoints, and parallel markers. Stronger feature-delivery mapping, more ceremony. [Evidence](../speckit/official/specs/001-command-deprecation-process/tasks.md) |
| Architecture decisions | Dedicated ADR-threshold skill; alternatives, trade-offs, no-ADR decision, and revisit conditions persisted. [Evidence](../axiom/workflow/decisions.md) | Plan/research surfaced four decisions and no-ADR rationale, but core flow has no dedicated ADR artifact or acceptance lifecycle. [Evidence](../speckit/official/specs/001-command-deprecation-process/research.md) |
| Traceability | Custom IDs link requirements, acceptance, tasks, validation, review, and completion matrix. Links depend on disciplined authoring; no deterministic schema/handoff. [Evidence](../axiom/workflow/execution.md) | Standard feature directory plus FR/SC/story/task links; analyze reported 18/18 buildable-requirement coverage and converge checked artifacts against implementation. Stronger observed traceability. [Evidence](../speckit/official/specs/001-command-deprecation-process/quickstart.md) |
| Documentation lifecycle | Explicit reconcile skill classified affected and unaffected durable docs; changelog and contributor entry points updated. [Evidence](../axiom/workflow/execution.md) | Implementation and convergence updated README, contribution guide, process docs, and changelog. Durable lifecycle beyond the feature was achieved through tasks, not a dedicated reconcile phase. [Evidence](../speckit/official/specs/001-command-deprecation-process/quickstart.md) |
| Validation | Exact commands, exit outcomes, hashes, rejected commands, sensitive scan, closure suite, and severity review. Strong command evidence. Semantic review missed unsupported identifier/trigger rules. [Validation](../axiom/workflow/validation.md), [review](../axiom/workflow/review.md) | Checklist, two analyze passes, implementation checks, and converge. First analysis found one high and three medium issues—including the unsupported `NNNN` sequence inference—then verified corrections. External scanners still unavailable. [Evidence](../speckit/official/specs/001-command-deprecation-process/quickstart.md) |
| Human decision points | One batched clarification. Phase 2 recorded residual unknowns, but final implementation still made bounded rules beyond the answer. | One batched clarification. Residual trigger, allocation, and release-proof questions remained visible in all downstream artifacts. |
| Codex integration | Native repository harness, concise `AGENTS.md` router, focused Axiom skills, policies, and progressive disclosure. No setup for existing Axiom repo. | Official native Codex integration; ten `$speckit-*` skills installed under `.agents/skills`, with manifests and upgrade path. Initialization took 0.63s. [Evidence](../evidence/spec-kit-upstream.md) |
| Context management | Intended progressive disclosure and durable routed context. In this run, broad Axiom context and custom review loops used 4,271,455 input tokens. | Phase artifacts and active-feature state provide durable handoffs. More phases/scaffold and repeated analysis used 7,919,796 input tokens. [Metrics](../evidence/metrics.md) |
| Multi-repository potential | `Project != Repository` is accepted, and the conceptual model explicitly anticipates multiple independent repositories. No executable orchestration exists and this scenario did not test it. | `SPECIFY_INIT_DIR` can target a member project and monorepo use, but observed artifacts remain one initialized project/feature. No logical cross-repository Project/Execution/Evidence aggregation was demonstrated. |
| Extensibility | Skills/policies/provider boundaries are easy to edit, but schemas, renderer, workflow engine, catalog, compatibility, and upgrade contracts are emerging or absent. | Mature integrations, extensions, presets, workflows, steps, bundles, catalogs, manifests, project overrides, and update/remove paths. Clear advantage. [Evidence](../evidence/spec-kit-upstream.md) |
| Deterministic behavior | Existing shell validators and exact repository checks are strong. Workflow phase contracts, requirement coverage, and convergence remain model-authored/model-reviewed. | CLI scripts deterministically resolve feature paths/templates/prerequisites; manifests track installed files. Analyze/converge are still agentic judgment, and incomplete checklists did not block explicit continuation. |
| Operational friction | No extra install in Axiom; fewer phases and lower measured time/tokens. More custom artifact design and manual validation/review work per task. | One-shot init fast, but 31 scaffold files/about 296 KiB, nine full-flow phases, more artifacts, 25m23s, and higher tokens for this small change. |
| Token/context efficiency | 4,271,455 input; 4,068,608 cached; 60,008 output. Lower in this run, still very high. | 7,919,796 input; 7,670,272 cached; 71,325 output. Higher in this run; richer gates partly explain cost. [Metrics](../evidence/metrics.md) |
| Vendor coupling | Current Axiom workflow is Codex-first but domain concepts and provider boundaries aim to remain vendor-neutral. | Direct use couples project layout, commands, templates, manifests, upgrade behavior, and feature lifecycle to GitHub Spec-Kit, even though Spec-Kit itself supports many agents. |

## Most important quality finding

Frozen answers required the `NNNN-short-name.md` shape but did not define number
allocation. They also did not answer full trigger coverage.

- Axiom recorded these gaps, then its implementation said `NNNN` was the “next
  unused” identifier and covered rename/explicit deprecation/removal. The final
  review passed the bounded interpretation. [Axiom process](../axiom/implementation/docs/deprecations/README.md),
  [Axiom decision record](../axiom/workflow/decisions.md)
- Spec-Kit initially made a similar sequence inference. `$speckit-analyze`
  classified it High, sent the correction back to owning artifacts, and the
  final implementation explicitly refused to infer an allocator. [Spec-Kit
  research](../speckit/official/specs/001-command-deprecation-process/research.md),
  [Spec-Kit process](../speckit/implementation/docs/deprecations/README.md)

This is direct evidence that Spec-Kit's analyze/converge loop currently catches
a cross-artifact consistency failure the Axiom self-review missed.

## Overlap analysis

### What Axiom already does that Spec-Kit does

- Intent and outcome before implementation.
- Durable specification, clarification, plan, tasks, implementation, and
  validation artifacts.
- Constitution/governance checks.
- Human gates for material unknowns.
- Codex skills and repository-local Markdown.
- Traceability through named requirements, tasks, evidence, and final review.
- Security/scope constraints and honest unverified-check reporting.

### What Spec-Kit does better

- Standardized phase artifacts and command surface.
- Structured clarify interaction with bounded question batches.
- Prioritized user stories and task mapping.
- Cross-artifact coverage analysis before implementation.
- Post-implementation convergence against spec/plan/tasks.
- Manifest-aware Codex integration install, status, update, and removal.
- Mature extension, preset, workflow, bundle, catalog, and integration systems.
- Consistent feature-directory handoff without chat continuity.

### What Axiom does differently

- Treats Project as distinct from Repository and aims to preserve one outcome
  across independent repositories/providers.
- Separates Work Item, Specification, Plan, Decision, Execution, Evidence,
  Artifact, Release, Agent, Provider, and Integration concepts.
- Makes reconciliation of living product/architecture/operations/security docs
  an explicit final phase.
- Uses severity-based engineering/security review rather than only feature
  artifact consistency.
- Treats provider boundaries, evidence provenance, approvals, release, and
  operational accountability as product-domain concerns.
- Explicitly classifies fact, requirement, principle, hypothesis, decision, and
  open question during intake/governance.

### What Axiom needs beyond Spec-Kit

- Logical multi-repository Project identity and repository-role mapping.
- Cross-repository plan, execution, validation, release, and rollback evidence.
- Durable Execution and Evidence identities, provenance, integrity, retention,
  sensitivity, and privacy rules.
- Living-document reconciliation across product, architecture, ADRs,
  operations, security, and releases.
- Provider-neutral Work Item/source-of-truth and integration capability models.
- Approval/waiver accountability by action and risk.
- Deterministic schemas/state transitions and useful non-LLM behavior.
- Agent blueprint/renderer/package lifecycle for Axiom's first vertical slice.

### What would be duplicated if Axiom implemented its own version

- Constitution/spec/clarify/plan/tasks templates and phase semantics.
- Feature directory conventions and prerequisite/path scripts.
- Requirement-quality checklists.
- Cross-artifact coverage/consistency analysis.
- Implementation task tracking and convergence.
- Codex skill rendering and installation manifests.
- Integration, preset, extension, workflow, catalog, and upgrade mechanics if
  Axiom later builds equivalents.

Duplication is justified only where Axiom's Project, Evidence, provider, living
documentation, or deterministic control-plane requirements materially change
the contract. Recreating mature generic SDD mechanics without that difference
would be waste.

### What could be reused

- Vocabulary mappings: constitution, specification, clarification, plan, tasks,
  analysis, implementation, convergence.
- Interaction patterns: batched clarify questions and explicit recommended
  options.
- Artifact-quality patterns: story priorities, measurable outcomes, requirement
  checklists, cross-artifact coverage, convergence.
- Optional future import/export of selected Spec-Kit feature artifacts.
- Optional adapter invoking a pinned Spec-Kit version for a validated workflow
  where reuse has measured benefit.
- Upgrade/manifests/catalog security lessons without copying implementation.

No reuse above is authorized by this experiment. Each executable reuse needs a
specification, boundary, version policy, and validation matrix.

### What would create coupling

- Making `.specify/` or `specs/<feature>/` Axiom's authoritative state.
- Exposing `$speckit-*` as required Axiom commands.
- Depending on upstream templates, scripts, manifests, catalogs, or workflow
  semantics at runtime.
- Mapping Axiom Project/Execution/Evidence lifecycles directly onto Spec-Kit
  feature state.
- Requiring Spec-Kit upgrades before Axiom can evolve its own contracts.
- Wrapping output without owning error, rollback, compatibility, and provenance
  semantics.

## Strategy evaluation

### A — Direct dependency

**Shape:** `Axiom -> Spec-Kit`

- **Benefits:** Fastest reuse of mature templates, Codex integration,
  checklists, analyze/converge, extension ecosystem, and upstream improvements.
- **Limitations:** Spec-Kit does not supply Axiom's Project, cross-repository,
  Execution/Evidence, provider, release, or living-doc contracts. Axiom must
  still build them around an upstream-centered feature model.
- **Lock-in:** Highest. Axiom state, UX, tests, and releases would depend on
  upstream layout/commands/version semantics.
- **Maintenance/update:** Requires pinning, compatibility matrix, migration
  tests, security review, and coordinated project-integration upgrades.
- **User experience:** Mature but exposes two product identities unless Axiom
  simply becomes a Spec-Kit distribution.
- **Codex:** Excellent today; upstream integration is native.
- **Customization:** Strong through presets/extensions, but deep Axiom domain
  differences may require invasive overrides.
- **Future Lingo:** Lingo would inherit `.specify` concepts early, making later
  domain separation costly.
- **Multi-repository:** Additional Axiom coordination still required across
  independently initialized Spec-Kit roots.
- **Living documentation:** Spec-Kit feature artifacts help, but Axiom-specific
  reconcile lifecycle still required.
- **Duplication risk:** Lowest for generic SDD, but bridge/domain duplication
  remains.
- **Future migration cost:** Highest if upstream no longer fits.

**Assessment:** Not supported by current evidence.

### B — Encapsulation/orchestration

**Shape:** Axiom owns UX/domain and calls pinned Spec-Kit behind a boundary.

- **Benefits:** Reuses observed strengths—clarify, templates, analyze,
  converge—while Axiom can own higher-level Project and Evidence flows.
- **Limitations:** Wrapper can leak file paths, prompts, errors, state, and
  upgrade behavior. Axiom must reconcile two workflow models and provenance
  systems.
- **Lock-in:** Medium-high runtime coupling; lower than A only if adapter
  contracts are narrow and replaceable.
- **Maintenance/update:** Ongoing upstream compatibility matrix, golden fixtures,
  migration tests, and version support policy.
- **User experience:** Potentially one Axiom surface, but debugging wrapped
  upstream behavior may be confusing.
- **Codex:** Strong; both are Codex-capable.
- **Customization:** Axiom can add Project/reconcile/evidence phases around
  Spec-Kit; deep changes still fight upstream assumptions.
- **Future Lingo:** Lingo becomes orchestrator plus translation layer before its
  own domain contracts are proven.
- **Multi-repository:** Axiom could coordinate one Spec-Kit run per repository,
  but atomicity and combined evidence remain Axiom responsibilities.
- **Living documentation:** Axiom reconcile can sit after upstream converge.
- **Duplication risk:** Medium; orchestration, state, validation, and artifact
  translation overlap both products.
- **Future migration cost:** Medium-high; adapter helps only if upstream state
  is not authoritative.

**Assessment:** Viable future option after one narrow adapter experiment; not
best default now.

### C — Conceptual compatibility

**Shape:** `Axiom concepts ≈ Spec-Kit concepts`, with no mandatory runtime
dependency.

- **Benefits:** Preserves Axiom's domain and multi-repository direction while
  adopting proven phase vocabulary, clarify patterns, coverage analysis, and
  convergence expectations. Keeps optional interoperability possible.
- **Limitations:** Axiom must implement deterministic contracts and can drift or
  create superficial compatibility.
- **Lock-in:** Low runtime coupling, medium conceptual coupling.
- **Maintenance/update:** Periodic upstream comparison, explicit compatibility
  mapping/version, and selective adoption rather than automatic upgrades.
- **User experience:** One Axiom surface and state model.
- **Codex:** Axiom keeps native harness; Spec-Kit artifacts could be mapped or
  imported only when specified.
- **Customization:** Highest fit for Project, Execution, Evidence, providers,
  living docs, and Lingo.
- **Future Lingo:** Lingo can implement Axiom contracts first and expose
  optional compatibility at boundaries.
- **Multi-repository:** Native Axiom Project model remains unconstrained by one
  Spec-Kit root.
- **Living documentation:** Axiom reconcile remains first-class.
- **Duplication risk:** High for generic SDD if Axiom rebuilds templates and
  analyzers without measured domain need. Control through explicit reuse-or-own
  decisions per capability.
- **Future migration cost:** Low-medium; optional adapters can be added without
  migrating authoritative Axiom state.

**Assessment:** Best current fit.

### D — Reference only

- **Benefits:** Lowest coupling and maximum freedom.
- **Limitations:** Gives up interoperability and makes Axiom own every generic
  SDD improvement, validation, Codex integration, and upgrade mechanism.
- **Lock-in:** Lowest external lock-in; highest lock-in to Axiom's own bespoke
  implementation.
- **Maintenance/update:** Manual research only; no compatibility obligation.
- **User experience:** One coherent Axiom surface, but no bridge for Spec-Kit
  users/artifacts.
- **Codex:** Entirely Axiom-owned.
- **Customization:** Maximum.
- **Future Lingo:** Fully independent, but more functionality must be built.
- **Multi-repository/living docs:** Unconstrained and fully Axiom-owned.
- **Duplication risk:** Highest.
- **Future migration cost:** Medium if later interoperability becomes valuable.

**Assessment:** Safer than premature dependency, but discards too much proven
value from a mature adjacent workflow.

## Ranking

1. **C — Conceptual compatibility.** Best balance of Axiom domain freedom,
   Spec-Kit learning/reuse potential, Codex fit, and reversibility.
2. **B — Encapsulation/orchestration.** Strong reuse path if a narrow future
   workflow proves adapter value and upgrade operability.
3. **D — Reference only.** Low coupling but unnecessarily accepts maximum
   duplication and no interoperability.
4. **A — Direct dependency.** Reuse is real, but current domain mismatch and
   migration/upgrade coupling are too large.

## Recommendation

Propose C: define an explicit conceptual mapping between Axiom and Spec-Kit
phases, adopt the observed quality patterns, and keep Axiom authoritative for
Project, Work Item, Decision, Execution, Evidence, Release, providers, and
living documentation. Do not add Spec-Kit as a dependency, copy its templates,
promise artifact compatibility, or implement an adapter in this change.

Keep B as a testable future option. Reconsider when one approved Axiom workflow
can compare an optional pinned Spec-Kit adapter against an Axiom-native path,
including upgrades, failures, multi-repository aggregation, evidence
provenance, and rollback.

## Open questions for human decision

1. Accept, reject, or revise ADR-0002's Proposed recommendation for C?
2. If C is accepted later, should initial compatibility mean vocabulary and
   lifecycle mapping only, or also a future import/export contract? The latter
   needs a separate specification and evidence.

No other question blocks review of this experiment.
