# Axiom and GitHub Spec-Kit Evaluation

## Status

Two controlled scenarios completed on 2026-08-11. After review of both scenarios
and the methodology correction, the human architectural decision was recorded
as [ADR-0002](../decisions/0002-axiom-speckit-relationship.md) with status
**Accepted** on 2026-09-10: Axiom will implement its own SDD harness, domain, and
lifecycle, informed by Spec-Kit as a strategic upstream research reference. No
Spec-Kit dependency, production wrapper, compatibility promise, copied
template, Lingo implementation, upstream-watch automation, or new harness
implementation is authorized by this research.

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

## Research trajectory

```text
Initial hypothesis
-> A / B / C / D
-> Scenario 001
-> C leading
-> Scenario 002
-> B vs C
-> C empirically stronger
-> human architectural decision
-> independent Axiom implementation informed by Spec-Kit
```

“C leading” records the experiment-stage interpretation. The final decision is
not a compatibility promise: it preserves independent Axiom ownership while
treating Spec-Kit as a strategic upstream research reference.

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

## Historical strategy assessment

The following A/B/C/D assessment is preserved as the interpretation used during
the experiments, before the final human decision.

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

| Run | Expected | Detected | Missed | Seeded recall | Unexpected | Validated false positives | Input tokens | Wall time |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Current Axiom baseline | 5 | 4 | 1 | 80% | 4 | 0 | 94,595 | 107.35s |
| C — experimental Axiom-native | 5 | 4 | 1 | 80% | 1 | 0 | 77,514 | 89.82s |
| B — encapsulated Spec-Kit | 5 | 3 | 2 | 60% | 3 | 0 | 193,169 | 177.83s + 1.30s preparation |

All approaches found rollback coverage, identifier contradiction, and
unverifiable performance. Both Axiom runs found the missing decision. B could
not because official analyze/converge does not consume the translated decision
artifact. Every approach missed Redis as unsupported behavior. C's experimental
fixture-boundary rule reduced unexpected findings from four to one but did not
improve seeded-finding recall over current Axiom.

### Methodology correction

The original scorer incorrectly treated every unexpected finding as a false
positive even though the oracle covers only the five seeded faults. Post-run
methodology review classified all eight unexpected findings individually. Each
is a production-completeness observation outside the explicitly non-executable,
deliberately partial fixture: baseline 4, C 1, B 3. Valid additional findings,
validated false positives, duplicates, and unclassified findings are all zero.
No model was re-executed; original results, events, timing, and seeded detection
remain unchanged.

Expected matches were also revalidated. Exact neutral category plus an exact
stable-reference token is sufficient for these five unique oracle pairs; every
current match is semantically correct. The scorer now rejects substring-only
matches and records the actual finding ID used for each detected expected item.

### Additional adapter and upgrade evidence

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

B demonstrated immediate mature analyze/converge behavior but had lower seeded
recall after translation. It lost an Axiom-relevant Decision input, doubled
runtime, used 2.49 times C's input tokens, added mutable disposable state, and
introduced version/failure/cleanup responsibilities.

C directly represented every neutral input, produced fewer unexpected
out-of-scope observations, and used lower context and runtime. Both B and C
produced zero validated false positives, so false-positive count does not
distinguish them. C's quality is still insufficient: it missed one deliberate
unsupported behavior, remained model-driven, and has no production Project-level
implementation. Scenario 002 supports conceptual compatibility, not immediate
Axiom-native product implementation.

For a Project spanning independent backend, frontend, and infra repositories,
B would need one synthetic aggregate Spec-Kit project, three projected
projects, or one run per repository plus Axiom-owned correlation. C could
operate on a Project-level matrix without translation, but Axiom lacks the
minimum Project/repository source and evidence contracts needed to prove it.

### Scenario 002 interpretation before human decision

Keep the ranking:

1. C — conceptual compatibility;
2. B — encapsulation/orchestration;
3. D — reference only;
4. A — direct dependency.

C remains the leading hypothesis because it detected one more expected major
finding than B, avoided B's Decision-input loss, and materially reduced
translation, runtime coupling, context, runtime, failure handling, and
multi-repository impedance. C also produced two fewer unexpected observations,
but the recommendation does not treat them as false positives. B remains
plausible only as an optional bounded adapter when a specific approved workflow
benefits enough to justify its maintenance surface.

This evidence made C the empirically stronger experiment-stage hypothesis. It
did not itself authorize implementation or establish a compatibility contract.

### Limitations

- one synthetic, single-repository scenario and one run per approach;
- no variance estimate;
- the seeded oracle is not complete ground truth; unexpected-finding
  classification remains explicit experiment-review judgment;
- no newer stable Spec-Kit version existed for upgrade migration testing;
- multi-repository behavior is reasoned, not executable;
- neither C prototype nor B adapter is production architecture.

### Redis unsupported-behavior gap

Every path missed Redis introduced by plan, task, and implementation without a
supporting requirement. This is a candidate for a future Axiom capability:
detect introduced behavior or implementation scope that has no supporting
requirement, decision, or approved-plan rationale. This research conclusion does
not authorize implementation in Axiom.

## Revisit evidence that could materially change the accepted decision

Human acceptance of ADR-0002 required no additional scenario. The following
questions can trigger reconsideration when their prerequisites exist.

| Question | Why it matters | Result favoring B | Result favoring C |
|---|---|---|---|
| Can a bounded B adapter preserve Axiom Decision input and match C's seeded recall without adding Axiom-native semantic analysis? | Decision translation loss caused B's lower Scenario 002 recall. | Equal or better recall with small isolated translation. | Persistent loss, or parity only by duplicating native analysis. |
| How do B and C operate after minimum Project/repository input and evidence contracts exist? | Multi-repository delivery is central to Axiom and unproven for both. | Lower total aggregation complexity with equivalent provenance. | Synthetic projections or provenance loss for B while C consumes the Project contract directly. |
| What is B's migration cost when a newer stable Spec-Kit release exists? | No real upgrade was available in Scenario 002. | Controlled compatible upgrade cheaper than native maintenance. | Material adapter/schema/failure regression and migration cost. |

These are revisit conditions, not a reason to create Scenario 003 now.
Agent-harness generation does not test the observed analyze/converge Decision
gap.

## Decision

Human review accepted **Independent Axiom implementation informed by Spec-Kit**.
Axiom owns its SDD harness, domain, and lifecycle. Spec-Kit is a strategic
upstream research reference, not a runtime dependency, technological
foundation, mandatory adapter, file-format contract, behavioral compatibility
promise, or architectural dependency.

Axiom may learn from Spec-Kit patterns only after asking whether they solve an
Axiom problem, validating them against Axiom requirements, experimenting when
necessary, and recording an ADR when the consequence is architectural.
Divergence is allowed where Axiom's domain requires different behavior.

## Future upstream watch

An approximately weekly Spec-Kit Upstream Watch is a **Planned** direction. It
will compare the last evaluated upstream state with the current state across
releases, documentation, workflows, commands, extensions, presets, agent
integrations, templates, architecture, and concepts. Relevant findings should
flow through research, experiments when needed, ADR review when architectural,
and Axiom-native implementation after approval.

No watch mechanism is selected or implemented here. See the
[strategic upstream register](spec-kit-strategic-upstream.md).

## Next work, not implemented

The next product activity is to **Design the first Axiom-native SDD harness
evolution**. Intake, specify, clarify, plan, tasks, implement, review, analyze,
converge, and reconcile are candidate inputs, not an approved final workflow.
**Design Spec-Kit Upstream Watch** remains a separate future activity.
