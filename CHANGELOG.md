# Changelog

All notable changes to Axiom's repository-level product and architecture baseline are documented here.

## Unreleased

- Evaluated Axiom's architectural relationship with GitHub Spec-Kit through controlled Scenario 001 and Scenario 002 evidence.
- Corrected Scenario 002 scoring methodology so unexpected findings are not automatically treated as false positives.
- Accepted ADR-0002: Axiom maintains an independent SDD harness, domain model, and lifecycle informed by Spec-Kit as a strategic upstream reference, without runtime or compatibility dependency.
- Accepted ADR-0003: Lingo is Axiom's executable local control-plane direction; implementation remains gated by an approved Specification.
- Refined the conceptual model around Project/Repository separation, Runtime, Capability, Integration, Transport, Agent/Profile concepts, orchestration, Execution, Evidence, and portable versus local configuration.
- Added architectural sequencing and `lingo project init` as the candidate first executable vertical slice, without implementing it.
