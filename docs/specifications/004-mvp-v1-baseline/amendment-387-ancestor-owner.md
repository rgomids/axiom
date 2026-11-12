# Specification 004 amendment — Issue #387 ancestor owner refusal

Revision 1, 2026-11-11. Authorized bounded diagnostic correction; compatibility
**DECISION_PENDING (Rafael)**. Base: `e5b9d10880cfc210356b8c49d639b1718b494e01`.

## Specification

An ancestor ownership refusal is an environment trust failure, not evidence of
corrupt Project state. Preserve the existing fail-closed ownership, permission,
link, identity and publication checks (ADR-0005 / ADR-0007). Inspection must not
repair, reconfigure, create or remove state in response to this refusal.

Expose `untrusted_ancestor_owner` with bounded observations: path class
`filesystem_root` or `ancestor` (never personal path components), observed numeric
UID and Linux UID-map observation. A restricted UID map with owners unmapped must
be described as such, without claiming the observed owner was host root. Missing,
malformed or unavailable map evidence remains unknown. Never accept overflow UID
as root. JSON and terminal output must retain the category, observations and a
reference to troubleshooting guidance across InstallationStore Inspect/selection,
Project show/validate/list, Runtime Profile list/validate and Work Item/Workflow
list. Existing corrupt state keeps its existing diagnostic.

## Plan and tasks

1. Add a typed unsafe ancestor-owner error at the shared POSIX check. Carry its
   bounded category/detail through existing string-category adapter boundaries;
   retain `errors.Is(ErrUnsafe)` and use the existing CLI diagnostic envelope.
2. Add deterministic injected ownership/stat and UID-map tests, store propagation
   and command JSON/text regressions. No real state, vendor inference or probes.
3. Reconcile command reference, installed operational troubleshooting and shipped
   Runtime skills; refresh installed-doc metadata using existing tooling.
4. Run required targeted/full checks, review, record evidence, commit/push this
   branch, open one normal PR and observe CI. No merge/release/status transitions.

## Compatibility investigation and decision proposal

**DECISION_PENDING (Rafael)** — no compatibility relaxation is authorized by the
accepted contracts. ADR-0007 consumes ADR-0005's controlled ancestor-replacement
protection. ADR-0028 bounds process supervision; it grants no filesystem exception.
Issue #265 used explicitly approved Windows defaults and bounded consented Runtime
ACL repair (Spec 005 / ADR-0019); it did not waive POSIX ancestor ownership.

| Option | Threat analysis and trade-off | Disposition |
| --- | --- | --- |
| Place roots under sandbox writable root | Full chain still starts at `/`; writable permission alone provides no trustworthy owner evidence | Does not resolve this failure |
| Treat overflow UID as root with restricted `uid_map`, above owned anchor, no group/other write | Mapping proves IDs are hidden, not that their host identity is root; a foreign owner can chmod/replace its own directory despite current mode | Relaxes ownership rule; approval required |
| Start at euid-owned root | Open handles confine operations, but named roots can be replaced by owners of unchecked ancestors | Relaxes replacement boundary; approval required |
| Narrow broker outside namespace | Keeps owner checks and agent confinement, but adds IPC authentication, request/object authority, replay and lifecycle boundaries | New product/security contract; approval required |

Recommendation: ship truthful refusal now; design a narrow broker separately if
restricted-sandbox operation remains required. Preserve exact-root authority,
no ambient filesystem access and no vendor inference in that design. Do not use
`danger-full-access` as mitigation. A broker contract and persisted authority
would become harder to change after deployment; no broker is implemented here.
Use `Refs #387`: compatibility remains unresolved pending the human decision.
