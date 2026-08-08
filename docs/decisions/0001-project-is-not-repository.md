# ADR-0001 — Project is not Repository

## Status

Accepted on 2026-08-08.

## Context

Axiom must preserve product intent, work, decisions, execution and delivery across systems that may be split among backend, client, infrastructure and other repositories. Treating a Git repository as the product boundary would fragment this state and make multi-repository support an infrastructure convention rather than a domain capability.

Current repository guidance and discovery consistently state this constraint. Workspace ownership, persistence and repository-sharing semantics remain open and are not decided here.

## Decision

Project is a logical boundary distinct from Repository. A Project may associate multiple independent Repositories, and the domain must not require physical proximity, a common provider, a monorepo or Git submodules.

This decision does not establish Project as Axiom's primary aggregate, ownership boundary or persistence root. Those choices remain open with the Workspace and state models.

## Alternatives considered

### Repository as Project

Simpler discovery and storage, but fragments one product across repositories and leaks VCS structure into product identity.

### Workspace as the only aggregate

Provides a global view, but leaves product ownership and isolation unclear before Workspace semantics are known.

### Project as logical boundary with repository associations

Preserves product coherence, permits multi-repository work and keeps Workspace as a separately decidable operational context.

## Consequences

### Positive

- Work Items, Specifications, Decisions, Executions, Evidence and Releases can span repositories.
- Repository providers and local layouts can vary.
- Git submodules remain an optional repository technique, not an Axiom requirement.

### Negative / trade-offs

- Axiom needs explicit Project and Repository identities and association rules.
- Validation and release evidence may need aggregation across revisions.
- Repository sharing, availability and partial failure require later decisions.

## Revisit when

- Evidence shows the Project boundary cannot own traceability without a higher-level domain aggregate.
- Cross-Project repository sharing creates incompatible ownership rules.
- The product scope changes from development control plane to repository-only tooling.
