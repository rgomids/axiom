# Documentation Policy

Documentation is part of the product state.

Before reconciling documents, consult [Documentation Governance v1](../../docs/documentation.md) for canonical sources, categories, lifecycle rules, and update triggers. Update only what the change actually affects.

## Authority and anti-drift

- Notion owns product discovery; Specifications, architecture, and ADRs own technical contracts and decisions. Secondary sources summarize and link.
- Keep operational tracking in Issues/Projects when adopted, or existing Task/PR records until then. Never copy Task/PR status into Product Foundation or Roadmap; link to the Specifications index for consolidated lifecycle state.
- Preserve approved decisions and dated history. Merge does not mean human acceptance; acceptance does not authorize the next Task.
- Agent context must not override canonical artifacts. Hypotheses must remain distinct from accepted decisions.
- Reconcile conflicting secondary sources against canonical sources; expose evidence/contract discrepancies for explicit resolution.
- Do not manually duplicate volatile content or add unverifiable claims or badges.

## Reconciliation examples

| Change | Expected documentation |
|---|---|
| internal bug fix, no contract impact | task/evidence; durable docs only if needed |
| business-rule change | Specification and Evidence; Notion if product discovery changes |
| new integration/boundary | architecture; possibly ADR |
| public API incompatibility | spec + architecture + migration/ADR as applicable |
| infrastructure behavior | operational/architecture docs |
| agent contract change | agent/renderer docs and evals |

Never create documentation churn merely to satisfy a checklist.
