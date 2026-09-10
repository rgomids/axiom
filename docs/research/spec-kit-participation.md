# Spec-Kit Participation Analysis

## Status

**Pre-experiment research / historical hypothesis.** The formal comparison is
now documented in [Axiom and GitHub Spec-Kit Evaluation](axiom-speckit-evaluation.md).
No Spec-Kit dependency, wrapper, compatibility contract, or copied artifact is
approved by this document.

## Verified reference

Current official Spec-Kit documentation describes a configurable SDD harness with a core `specify -> plan -> tasks -> implement` flow, optional clarification and analysis gates, project constitution support, multiple agent integrations, presets, extensions and workflows. Its upgrade model manages installed integration/template infrastructure while preserving project specifications and constitution. Sources: [overview](https://github.com/github/spec-kit/blob/main/docs/index.md), [Agentic SDD](https://github.github.io/spec-kit/reference/agentic-sdd.html), [workflows](https://github.com/github/spec-kit/blob/main/docs/reference/workflows.md), and [upgrade guide](https://github.github.com/spec-kit/upgrade.html).

These are upstream capabilities as observed on 2026-08-08, not Axiom decisions.

## Options

| Option | Advantages | Disadvantages | Lock-in and updates | User experience |
|---|---|---|---|---|
| A. Direct dependency | Fast access to mature templates, commands and current features; smaller initial SDD implementation. | Axiom lifecycle and file layout become coupled to upstream behavior; multi-repository/project semantics still require separate design. | Highest runtime/version coupling; upstream upgrades can help but require compatibility management. | Familiar Spec-Kit workflow, but two product identities and command surfaces may be visible. |
| B. Encapsulate/orchestrate | Reuses upstream while Axiom owns higher-level Project, provider, execution and evidence flows. | Wrapper can become leaky; failures, versions and generated files must be reconciled; risk of duplicating orchestration already upstream. | Medium-high; adapter allows replacement but ongoing upstream test matrix is required. | Potentially unified Axiom flow if boundaries stay clear; upgrade errors can be confusing. |
| C. Implement compatible concepts | Axiom can preserve intent/spec/plan/tasks semantics while shaping artifacts around its own domain and multi-repository needs. | More design and validation work; compatibility may drift or become superficial. | Medium conceptual coupling, low runtime coupling; updates require deliberate comparison rather than automatic adoption. | One Axiom experience; import/export compatibility could reduce migration cost if explicitly specified. |
| D. Reference only | Lowest coupling and full freedom to learn from proven patterns. | Duplicated effort and risk of missing upstream improvements; no interoperability promise. | Lowest; maintenance entirely owned by Axiom. | Coherent Axiom surface, but existing Spec-Kit users gain no direct bridge. |

## Fit with current evidence

- Axiom needs Project-level, multi-repository, provider, execution and evidence concepts beyond the demonstrated Spec-Kit feature artifact flow.
- Existing Axiom harness already implements a lightweight intent/specification/plan/review model without an installed Spec-Kit dependency.
- Spec-Kit now supports Codex and richer extensibility than the historical discovery summary alone implied; adopting or wrapping it therefore requires a current experiment, not assumptions based on an older version.
- The first Axiom slice is agent-harness generation, not application-feature implementation. Direct dependency offers no proven benefit for this slice yet.

## Historical research recommendation

This analysis recommended using **C (conceptual compatibility)** as the
comparison baseline while keeping **B (encapsulation)** viable. The required
formal experiment was completed on 2026-08-11 against Spec-Kit `v0.16.2`:

1. current Axiom harness;
2. current Spec-Kit Codex integration;
3. an explicit scorecard covering artifact quality, deterministic validation, upgrade behavior, token/context cost, multi-repository fit, traceability and user friction.

See the [durable evaluation](axiom-speckit-evaluation.md) and
[ADR-0002](../decisions/0002-axiom-speckit-relationship.md). Human review later
accepted an independent Axiom implementation informed by Spec-Kit as a
strategic upstream reference. This pre-experiment analysis remains historical;
no adoption or implementation follows automatically.
