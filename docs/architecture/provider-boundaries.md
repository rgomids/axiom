# Provider Boundaries

## Status

Architectural constraint and trade-off analysis. No adapter, interface, provider or source-of-truth policy is selected here.

## Boundary rule

Axiom domain language must not require GitHub, GitLab, Bitbucket, Jira, Linear, Notion or another product. External products expose capabilities; configured Integrations bind those capabilities to a Project or Workspace.

```text
Axiom workflow
-> required capability
-> integration boundary when needed
-> provider-specific adapter
-> external provider
```

## Where abstraction is justified

| Capability boundary | Why it is real | Minimum concept needed now |
|---|---|---|
| Source-control identity and change references | Projects may span repositories on different providers or local-only repositories. | Provider-neutral repository identity plus optional external reference. |
| Work tracking references | A Work Item may originate in Jira, Linear, GitHub Issues or no external tracker. | External reference metadata; ownership/sync semantics remain open. |
| Documentation references | Durable knowledge may live in repository docs, Notion, Confluence or elsewhere. | Classified source reference with provenance; no universal document API yet. |
| Agent execution | Codex is current target, but blueprint concepts should not be Codex paths. | Vendor-neutral blueprint and target-specific renderer boundary. |

## Where abstraction would be premature

- One universal CRUD interface across unrelated providers.
- A normalized query language before required cross-provider queries are known.
- Bidirectional synchronization before ownership, conflict and deletion rules exist.
- A single status taxonomy that erases provider semantics before mappings are observed.
- Credential storage, webhooks, retries or rate-limit frameworks before an integration slice requires them.
- Runtime plugin SDK before a second renderer or executor validates the boundary.

## Trade-offs

| Approach | Benefit | Cost/risk |
|---|---|---|
| Core depends directly on first provider | Fast first integration. | Vendor concepts leak into state; replacement and multi-provider Projects become expensive. |
| Universal abstraction up front | Appears portable. | Lowest-common-denominator contract, speculative complexity and untested mappings. |
| Capability boundary with thin provider references | Protects core language while preserving provider-specific detail. | Some workflow-specific ports and translation remain necessary. |

Current recommendation: use the third approach as a design constraint, but introduce an executable port only when a specified workflow invokes an external capability and a realistic replacement or trust boundary exists.

## Decisions still required

- Whether Axiom owns Work Item identity or indexes provider-owned items.
- Source-of-truth and conflict policy per artifact type.
- Integration scope: Workspace, Project, Repository or workflow.
- Credential reference and permission model.
- Offline behavior, caching and evidence retention.
- First provider experiment and success criteria.
