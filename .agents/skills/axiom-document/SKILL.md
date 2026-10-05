---
name: axiom-document
description: Reconcile Axiom durable documentation after a change without producing unnecessary documentation churn.
---

# Documentation Reconciliation

## Procedure

1. Identify what materially changed.
2. Map the change to documentation categories.
3. Update only affected durable documents.
4. Ensure approved decisions and implementation do not conflict.
5. Preserve historical ADRs; never rewrite them. Evolve per `docs/decisions/0018-adr-evolution-and-supersession-governance.md`: annotate a superseded fragment in place, use `Superseded` status only for full replacement, add the `## Supersedes` backlink in the new ADR, and reconcile the `docs/decisions/README.md` index.
6. Separate research hypotheses from accepted architecture.
7. Add links between related docs when it improves discoverability.

## Check

Ask:

- Did product behavior change?
- Did a contract change?
- Did an architectural boundary change?
- Was a meaningful decision made?
- Did an accepted ADR become partially or fully superseded?
- Did operations/deployment change?
- Did security behavior change?
- Did agent generation/runtime contracts change?

If all are no, durable documentation may not require modification.
