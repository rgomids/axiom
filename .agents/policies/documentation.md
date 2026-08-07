# Documentation Policy

Documentation is part of the product state.

Update only what the change actually affects.

## Documentation categories

- product;
- architecture;
- ADRs/decisions;
- operations;
- security;
- development;
- agent contracts;
- research.

## Reconciliation examples

| Change | Expected documentation |
|---|---|
| internal bug fix, no contract impact | task/evidence; durable docs only if needed |
| business-rule change | product/specification |
| new integration/boundary | architecture; possibly ADR |
| public API incompatibility | spec + architecture + migration/ADR as applicable |
| infrastructure behavior | operational/architecture docs |
| agent contract change | agent/renderer docs and evals |

Never create documentation churn merely to satisfy a checklist.
