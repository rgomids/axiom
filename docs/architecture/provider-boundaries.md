# Provider Boundaries

## Status

Architectural constraint and trade-off analysis. No adapter, interface,
provider, transport, installer or source-of-truth policy is selected here.

## Boundary rule

Axiom domain language must not require GitHub, GitLab, Bitbucket, Jira, Linear,
Notion or another product. External Providers expose Capabilities; configured
Integrations bind those Capabilities to a Project or Workspace and adapters use
a Transport when needed.

```text
Provider
-> offers Capability
-> configured through Integration
-> reached through Transport

Axiom workflow
-> required Capability
-> configured Integration
-> Transport and provider-specific adapter
-> Provider
```

The distinctions are mandatory:

- `Provider != Capability`;
- `Provider != Integration`;
- `Integration != Transport`;
- `Integration != MCP`;
- a Transport is infrastructure, not domain ownership.

## Where abstraction is justified

| Capability boundary | Why it is real | Minimum concept needed now |
|---|---|---|
| Source-control identity and change references | Projects may span repositories on different providers or local-only repositories. | Provider-neutral repository identity plus optional external reference. |
| Work tracking references | A Work Item may originate in Jira, Linear, GitHub Issues or no external tracker. | External reference metadata; ownership/sync semantics remain open. |
| Documentation references | Durable knowledge may live in repository docs, Notion, Confluence or elsewhere. | Classified source reference with provenance; no universal document API yet. |
| Agent execution | Codex is current target, but blueprint and role concepts should not be Codex paths or models. | Runtime boundary, vendor-neutral profiles and target-specific adapter/renderer. |
| Capability negotiation | A workflow must know whether selected runtimes and Integrations can satisfy required operations before execution. | Provider-neutral capability names and explicit supported/missing results; no universal provider API. |

## Where abstraction would be premature

- One universal CRUD interface across unrelated providers.
- A normalized query language before required cross-provider queries are known.
- Bidirectional synchronization before ownership, conflict and deletion rules exist.
- A single status taxonomy that erases provider semantics before mappings are observed.
- Credential storage, webhooks, retries or rate-limit frameworks before an integration slice requires them.
- Runtime plugin SDK before a second renderer or executor validates the boundary.
- Hardcoded model catalogs before runtime capability discovery is specified.
- Automatic Integration installation before authority, rollback and credential-reference behavior are specified.

## Trade-offs

| Approach | Benefit | Cost/risk |
|---|---|---|
| Core depends directly on first provider | Fast first integration. | Vendor concepts leak into state; replacement and multi-provider Projects become expensive. |
| Universal abstraction up front | Appears portable. | Lowest-common-denominator contract, speculative complexity and untested mappings. |
| Capability boundary with thin provider references | Protects core language while preserving provider-specific detail. | Some workflow-specific ports and translation remain necessary. |

Current recommendation: use the third approach as a design constraint, but introduce an executable port only when a specified workflow invokes an external capability and a realistic replacement or trust boundary exists.

## Capability negotiation direction

A future Lingo workflow should derive required capabilities from the Plan and
Project configuration, then ask runtime adapters and configured Integrations
which capabilities they can satisfy.

```text
Project configuration + Plan
-> required capabilities
-> Runtime/Integration capability discovery
-> supported and missing capabilities
-> validation or user decision
```

For example, selecting Codex, Jira, Notion, and GitHub may require
`work-item.read`, `work-item.write`, `documentation.read`,
`documentation.write`, `repository.read`, `repository.write`, and
`pull-request.create`. A missing capability is an explicit validation result,
not permission to install or configure anything automatically.

## Integration bootstrap direction

When a required capability needs an external Integration, Lingo may eventually
detect existing configuration, propose a Transport, guide setup, validate
connectivity, and register the Integration. Installation, mutation, permission
expansion, and rollback require a separate Specification and explicit authority.

MCP may be one proposed Transport. Native APIs, CLIs, HTTP, or future mechanisms
remain possible. Selecting a Provider never implies selecting MCP.

## Credential boundary

Portable Project configuration stores credential references only, never secret
values. Possible reference types include environment-variable names,
operating-system credential-store identifiers, and runtime-managed credential
identifiers.

No credential store is selected. Future adapters must use least privilege,
avoid placing secrets in logs or Evidence, and keep machine-local credentials
outside versionable Project configuration.

## Decisions still required

- Whether Axiom owns Work Item identity or indexes provider-owned items.
- Source-of-truth and conflict policy per artifact type.
- Integration scope: Workspace, Project, Repository or workflow.
- Credential reference and permission model.
- Capability vocabulary, discovery, negotiation and degradation semantics.
- Transport selection and adapter/runtime responsibility.
- Integration bootstrap authority, failure and rollback behavior.
- Offline behavior, caching and evidence retention.
- First provider experiment and success criteria.
