# Axiom and GitHub Spec-Kit Evaluation

## Status

Two controlled scenarios completed on 2026-08-11. Architectural recommendation
is recorded as [ADR-0002](../decisions/0002-axiom-speckit-relationship.md) with
status **Proposed**. The PR remains an active evaluation surface pending human
review. No Spec-Kit dependency, production wrapper, compatibility contract,
copied template, or Lingo implementation is authorized.

Temporary reproducibility evidence is committed under
[`experiments/speckit-evaluation/`](../../experiments/speckit-evaluation/).
This document is self-contained because that directory may be removed after
human review and preservation of durable findings.

## Question

Which relationship should Axiom have with GitHub Spec-Kit?

- A — direct dependency;
- B — encapsulation/orchestration;
- C — conceptual compatibility without mandatory dependency;
- D — reference only.

The research did not select an answer before collecting comparative evidence.

## Evaluated upstream

- Official repository: <https://github.com/github/spec-kit>
- Stable release: `v0.16.2`
- Evaluated commit: `4871b485f97c7fa452ec58eba325d87536c55c34`
- Release date: 2026-08-10
- License: MIT
- Native Codex integration: skills under `.agents/skills`, invoked as
  `$speckit-<command>`
- Official full flow evaluated:
  `constitution -> specify -> clarify -> plan -> checklist -> tasks -> analyze -> implement -> converge`

Spec-Kit also provides integrations, extensions, presets, workflows, bundles,
catalogs, project-local overrides, manifest-aware updates, and optional Git
workflow support. The experiment used pinned one-shot `uvx`; it did not add a
persistent dependency.

## Scenario 001 — Full SDD comparison

### Method

A neutral documentation-only scenario introduced a repository-local process for
future command deprecations. It exercised requirements, clarification,
planning, tasks, architecture treatment, validation, documentation lifecycle,
and human approval without application code.

Controls:

- same frozen brief, seed repository, operator answers, completion criteria,
  Codex model (`gpt-5.6-sol`), reasoning effort, host, and autonomy;
- one batched human clarification response per approach;
- Axiom executed first with current Axiom harness unchanged;
- Spec-Kit executed second from a clean seed with official `v0.16.2` Codex
  integration;
- no output was improved after observing the other approach;
- no provider mutation, permanent dependency, Lingo, or application code.

### Evidence

Both workflows completed the scenario with:

- durable specification, clarification, plan, tasks, implementation, and
  validation evidence;
- five implementation Markdown files;
- unchanged existing command names and descriptions;
- one human clarification batch;
- no application code, executable validator, dependency, CI, provider, secret,
  or network requirement;
- an explicit conclusion that the sample change did not warrant an ADR.

Measured execution:

| Metric | Axiom | Spec-Kit |
|---|---:|---:|
| Wall time | 15m55s | 25m23s |
| Input tokens | 4,271,455 | 7,919,796 |
| Cached input tokens | 4,068,608 | 7,670,272 |
| Output tokens | 60,008 | 71,325 |

These are real Codex CLI metrics for one scenario, not general benchmarks.

### Axiom strengths

- Explicit intake classification: fact, requirement, assumption, unknown, and
  non-goal.
- Natural alignment with Axiom's Project, Decision, Execution, Evidence,
  provider, and documentation-reconciliation concepts.
- Lean, tailored plan and severity-based engineering/security review.
- Strong command-level validation evidence, including failed/rejected commands
  and scanner limitations.
- Lower measured time and token use in this run.
- No extra workflow installation in an existing Axiom repository.

### Axiom gaps

- Phase handoffs, schemas, requirement coverage, and convergence are not
  deterministic or standardized.
- Same agent authored and reviewed custom artifacts; review missed a material
  unsupported inference.
- No built-in equivalent of Spec-Kit analyze/converge.
- No executable multi-repository Project orchestration, Execution/Evidence
  model, or living-doc reconciliation engine exists yet; current strength is
  conceptual rather than implemented.
- Extensibility, installation, upgrade, compatibility, and catalog contracts
  are immature compared with Spec-Kit.

### Spec-Kit strengths

- Mature, consistent artifact and command structure.
- Structured clarification with bounded questions and recommended options.
- Prioritized user stories, measurable outcomes, story-mapped tasks, and
  concrete file paths.
- Read-only cross-artifact analysis before implementation.
- Convergence check after implementation.
- Native Codex integration with manifests and an update path.
- Mature extension, preset, workflow, bundle, integration, and catalog model.
- Strong durable handoff through feature artifacts.

The most important observed difference: frozen answers specified a four-digit
`NNNN` filename shape but did not define number allocation. Axiom's final
process invented “next unused” and its review passed. Spec-Kit initially made a
similar inference, but analyze classified it High and the corrected output
explicitly left allocation unresolved. This is evidence of practical value in
Spec-Kit's cross-artifact analysis loop.

### Spec-Kit limits

- Higher measured time, token use, artifact count, and operational ceremony for
  a small documentation change.
- Feature/project-local artifact model does not demonstrate Axiom's logical
  Project spanning independent repositories with combined Execution, Evidence,
  Release, provider, and living-document state.
- Generated measurable goals can exceed available evidence; the run introduced
  10-minute and two-reviewer goals and later marked them unverified.
- Generated quality checklists remained incomplete; explicit full-sequence
  instruction allowed implementation, so the gate was procedural rather than
  an unconditional deterministic blocker.
- Deep adoption would couple Axiom to `.specify`, upstream templates, commands,
  manifests, versions, and upgrade behavior.
- Workflow shell steps are not a capability sandbox and require trust review.

## Overlap and boundary

Both products cover intent, constitution, specification, clarification, plan,
tasks, implementation, validation, Codex skills, and durable Markdown.

Axiom requires more than the observed Spec-Kit feature flow:

- Project distinct from Repository and cross-repository delivery;
- Work Item/provider source-of-truth boundaries;
- accountable Decision, Execution, Evidence, Artifact, and Release lifecycles;
- evidence provenance, integrity, retention, sensitivity, and privacy;
- living product/architecture/operations/security documentation reconciliation;
- deterministic state and useful non-model behavior;
- agent blueprint/renderer/package lifecycle.

Rebuilding Spec-Kit's generic templates, clarify UX, checklists,
analyze/converge, Codex rendering, manifests, extension model, and update system
without a domain-specific reason would be duplication. Directly adopting them
would create runtime and lifecycle coupling before Axiom's own contracts exist.

## Strategy assessment

### A — Direct dependency

Maximum immediate reuse and maximum coupling. It leaves Axiom's domain gaps
unsolved, makes upgrades/migrations part of Axiom's runtime, and risks shaping
Lingo around upstream feature state. Not recommended.

### B — Encapsulation/orchestration

Could reuse clarify/analyze/converge behind an Axiom boundary. Requires pinned
versions, compatibility fixtures, failure mapping, provenance, rollback, and
ongoing upstream tests. Viable later for one approved workflow; insufficient
evidence for default architecture now.

### C — Conceptual compatibility

Keeps Axiom authoritative for its domain while adopting proven vocabulary and
quality patterns and preserving optional future import/export or adapter paths.
Main risk is duplicating generic SDD and drifting from a vague compatibility
promise. Best current balance when compatibility is explicitly scoped and no
runtime dependency is implied.

### D — Reference only

Lowest external coupling but highest reinvention and no interoperability path.
Safer than premature dependency, but discards too much relevant evidence from a
mature adjacent workflow.

## Scenario 001 recommendation

1. C — conceptual compatibility.
2. B — encapsulation/orchestration.
3. D — reference only.
4. A — direct dependency.

Recommend C as a Proposed direction:

- define mappings for intent/intake, constitution, specification,
  clarification, plan, tasks, analysis, implementation, convergence/reconcile;
- adopt observed quality patterns without copying templates by default;
- keep Axiom state authoritative for Project, Decision, Execution, Evidence,
  Release, provider, and living documentation;
- introduce no Spec-Kit runtime dependency now;
- promise no artifact import/export compatibility until separately specified
  and tested;
- keep B available for a later narrow adapter experiment.

## Scenario 002 — Analyze/converge capability reuse

### Evidence

A second frozen fixture contained five deliberate semantic inconsistencies
across specification, plan, tasks, implementation, tests, and decisions:

- rollback requirement without task/implementation/verification;
- Redis task and implementation without a requirement;
- service-generated identifier contradicting caller-owned identifier intent;
- unverifiable performance acceptance;
- PostgreSQL introduced without decision evidence.

The expected-finding oracle, neutral capability contract, decision criteria,
scenario, and current-Axiom prompt were frozen before execution. Current Axiom
ran first without skill improvements. Only afterward, the experiment added a
50-line Axiom-native analysis skill and a confined Spec-Kit `v0.16.2` adapter.

| Run | Expected | Detected | Missed | Unexpected | Input tokens | Wall time |
|---|---:|---:|---:|---:|---:|---:|
| Current Axiom baseline | 5 | 4 | 1 | 4 | 94,595 | 107.35s |
| C — experimental Axiom-native | 5 | 4 | 1 | 1 | 77,514 | 89.82s |
| B — encapsulated Spec-Kit | 5 | 3 | 2 | 3 | 193,169 | 177.83s + 1.30s preparation |

All approaches found rollback coverage, identifier contradiction, and
unverifiable performance. Both Axiom runs found the missing decision. B could
not because official analyze/converge does not consume the translated decision
artifact. Every approach missed Redis as unsupported behavior. C's experimental
fixture-boundary rule reduced unexpected findings from four to one but did not
improve expected-finding recall over current Axiom.

B successfully ran official analyze then converge. Converge appended six tasks
and nine lines to the disposable task file. The adapter preserved raw events,
normalized findings, path/category/severity mappings, and exposed failure
states. It required 226 shell lines plus a 37-line prompt and generated 31
Spec-Kit files (about 292 KiB) in the temporary workspace. C required a 50-line
skill plus a 19-line prompt, excluding shared neutral validation/scoring.

Failure tests showed:

- missing launcher: fail closed, exit 69;
- unavailable pinned ref: raw upstream exit 1, no result, partial temp cleanup;
- incompatible output: fail closed, exit 65, no normalized output;
- capability nonzero exit: fail closed through wrapper exit 70;
- actual converge write: confined to disposable `tasks.md`.

The latest stable Spec-Kit release still equaled the pin (`v0.16.2`), so a real
two-version upgrade comparison was not possible.

### Interpretation

B demonstrated immediate mature analyze/converge behavior but did not improve
accuracy after translation. It lost an Axiom-relevant Decision input, doubled
runtime, used 2.49 times C's input tokens, added mutable disposable state, and
introduced version/failure/cleanup responsibilities.

C directly represented every neutral input and produced better precision with
lower context and runtime. Its quality is still insufficient: it missed one
deliberate unsupported behavior, remained model-driven, and has no production
Project-level implementation. Scenario 002 supports conceptual compatibility,
not immediate Axiom-native product implementation.

For a Project spanning independent backend, frontend, and infra repositories,
B would need one synthetic aggregate Spec-Kit project, three projected
projects, or one run per repository plus Axiom-owned correlation. C could
operate on a Project-level matrix without translation, but Axiom lacks the
minimum Project/repository source and evidence contracts needed to prove it.

### Recommendation

Keep the ranking:

1. C — conceptual compatibility;
2. B — encapsulation/orchestration;
3. D — reference only;
4. A — direct dependency.

C remains the leading hypothesis because it detected one more expected major
finding than B, produced two fewer unexpected findings, and materially reduced
translation, coupling, context, and runtime. B remains plausible only as an
optional bounded adapter when a specific approved workflow benefits enough to
justify its maintenance surface.

ADR-0002 remains Proposed. No result authorizes implementation or adoption.

### Limitations

- one synthetic, single-repository scenario and one run per approach;
- no variance estimate;
- non-executable fixture makes unexpected-finding classification oracle-bound;
- no newer stable Spec-Kit version existed for upgrade migration testing;
- multi-repository behavior is reasoned, not executable;
- neither C prototype nor B adapter is production architecture.

## Remaining evidence

- a newer stable Spec-Kit release allows a real pinned-vs-upgrade compatibility
  test;
- Axiom defines minimum Project/repository source and evidence contracts, making
  an executable multi-repository comparison non-speculative;
- user demand justifies an import/export contract;
- Spec-Kit adds domain capabilities that materially overlap Axiom's Project,
  Execution, Evidence, Release, or living-document contracts;
- Axiom's own implementation cost for generic SDD exceeds measured adapter
  maintenance cost.

No Scenario 003 was added. Multi-repository coordination could change the
decision later, but executing it before the minimum Project contract exists
would compare invented scaffolds. Upgrade compatibility can change B's
maintenance assessment only after a newer stable release exists. Agent-harness
generation does not test the observed analyze/converge gap.

## Remaining human decisions

- Accept, reject, or revise ADR-0002's Proposed recommendation after reviewing
  both scenarios.
- If accepted later, choose whether initial compatibility means vocabulary and
  lifecycle mapping only or a separately specified import/export contract.
