# ADR-0022 — Dedicated workflow conversational surface

## Status

Proposed on 2026-10-10 for human decision under
[Issue #303](https://github.com/rgomids/axiom/issues/303), recording the
maintainer direction stated there ("adopt a dedicated `axiom-workflow`
conversational skill"). The direction is not acceptance of this text: the ADR
stays `Proposed` until the maintainer accepts its exact reviewed revision
together with the
[Specification 007 amendment for #303](../specifications/007-configurable-workflows/amendment-303-workflow-skill.md).
Acceptance would not authorize implementation, merge, release, Runtime
inference or any Provider effect; those keep their own gates.

## Context

[ADR-0020](0020-workflow-definition-revision-binding.md) and Specification 007
HD-002 accepted on 2026-10-09 that workflow *authoring* is exposed through the
Project surface (`axiom project workflow *`, `axiom-project` `workflow.*`) and
workflow *execution* through the Work Item surface (`axiom workflow *`,
`axiom-work-item` `run`/`status`/`plan`). Specification 007 explicitly says
"do not create an `axiom-workflow` skill" and rejected a dedicated workflow
skill because it would duplicate discovery and selection.

Delivery of #273–#275 (v0.13.0–v0.15.0) showed the cost of that choice for a
conversational operator: one workflow journey (author a definition, configure
stages and agents, select it, start an Execution, prepare a stage Plan, inspect
the proposal) crosses two skills whose primary purposes are Project lifecycle
and Provider Work Item lifecycle. The operator must know which skill owns which
half, and the skills require hand-written definition and Plan JSON files.
On 2026-10-10 the maintainer decided that workflow management must be a
first-class conversational capability in both Codex and Claude (#303, Linear
AXM-7).

The accepted decisions mix two layers that this change must keep apart:

1. **Domain and application ownership** — who owns workflow definitions,
   selection, Execution binding, validation, authority and state.
2. **Conversational interaction surface** — which installed Runtime skill a
   user's natural-language workflow intent is routed through.

Only layer 2 is changing.

## Decision

Introduce `axiom-workflow` as a third canonical, embedded, installed domain
skill for Codex and Claude. It is the primary conversational entrypoint for
workflow intent: definitions and revisions, stage and agent configuration,
Runtime/Profile references and readiness inspection, Execution workflow
interaction, stage planning, and workflow acceptance procedures.

The skill is a thin orchestration surface over the **existing** canonical
operations. In particular:

- Workflow definitions and selection remain Project-owned (ADR-0020,
  HD-001). The CLI command `axiom project workflow <operation>` keeps its
  name, arguments, result payload and authority.
- Executions keep their Work Item target and immutable `WorkflowBinding`
  (ADR-0008 for format 1, ADR-0020 for format 2). The `axiom workflow`
  command family keeps its names, arguments, results and authority.
- Lingo and the application services remain the only owners of validation,
  digests, preview/apply, authority, state and result rendering (ADR-0003).
  The skill adds no rule, registry, scheduler, store, approval mechanism or
  result protocol. Its operation metadata comes from `axiom skill inspect
  axiom-workflow`, built from the same command flag sets as every other skill.
- The Runtime may prepare bounded draft input files for those commands on the
  user's behalf. A draft is never an approved Plan or an authority fact.

Existing `axiom-project` `workflow.*` and `axiom-work-item` `plan`/workflow
routes stay installed and behave identically as compatibility routes; their
removal requires a separate versioned decision. Their primary workflow guidance
moves to `axiom-workflow`.

## Alternatives considered

| Option | Complexity / coupling | UX / operability | Reversibility |
|---|---|---|---|
| A. Keep HD-002 (Project + Work Item skills only) | No new artifact | Workflow journey stays split across two overloaded skills; already judged unacceptable by the maintainer | Trivial |
| **B. Dedicated thin `axiom-workflow` skill over existing commands (recommended)** | One more embedded skill, catalog entry, receipt/history pin and install/upgrade path; overlap with compatibility routes | One discoverable workflow entrypoint in both Runtimes; domain and CLI unchanged | Additive; compatibility routes keep older callers; the skill can be retired by a later decision without data migration |
| C. Dedicated skill **and** a new top-level workflow CLI tree for authoring | Moves authoring out of `axiom project`; duplicates selectors and changes public commands | Marginal CLI symmetry; breaks existing scripted callers | Requires command aliases and a CLI deprecation window |
| D. Move workflow domain ownership into a new workflow service | Second source of workflow truth; contradicts ADR-0020 HD-001 | None beyond B | Hard: persisted ownership and selection would migrate |

B changes only the interaction surface. C and D change public CLI or domain
contracts without a validated need.

## Consequences

### Positive

- One conversational surface for the whole workflow journey in Codex and
  Claude, with the same commands, authority and results as the CLI.
- Project and Work Item skills return to their primary lifecycle purposes.
- The #275 acceptance procedure can be driven by one natural-language request.

### Negative / trade-offs

- A third installed skill must be added to the embedded inventory, receipts,
  skill-set history, `skill inspect`, help/routing catalogs and
  install/upgrade/retirement tests in both Runtimes.
- During the compatibility window the same canonical operation is reachable
  from two skills; routing tests must prove identical commands and semantics.
- Runtime-prepared drafts add a presentation risk: the skill must never treat a
  draft, an `approved` field it wrote, or its own summary as human approval.

### What becomes harder to change later

Once installed, the skill name and its operation names become public
conversational contracts covered by skill-set history. Removing or renaming them
requires the same retirement/upgrade protocol used for the v0.6.0 skill split.

## Proposed partial supersession

Applied only when this ADR is accepted, following
[ADR-0018](0018-adr-evolution-and-supersession-governance.md): this ADR would
then gain a `Supersedes` section, and ADR-0020 would keep `Accepted` with the
clause "execution stays under the Work Item domain surface" (Decision section)
struck through and annotated as superseded **only for the conversational
skill surface**. "Authoring stays a Project operation" and every other ADR-0020
decision remain in force unchanged. The Specification 007 fragments are listed
in the amendment.

## Revisit when

Workflow definitions become independently managed across Projects, the CLI
command families must change, or compatibility routes are proposed for removal.
