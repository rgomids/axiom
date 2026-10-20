# ADR-0034 — Release-content documentation association

## Status

Proposed — 2026-10-20, AXM-37 reconciliation R1 in Draft
[PR #335](https://github.com/rgomids/axiom/pull/335). Not accepted or in force.
No implementation or supersession annotation is authorized by this proposal.

## Context

[ADR-0033](0033-installation-bound-operational-documentation.md) binds external
operational assets to the executing binary and requires surviving processes to
prove their own generation or refuse. Its Plan/D2 interpretation demanded proof
of the original whole executable file. The native macOS
[C2 counterexample](../specifications/010-installed-operational-documentation/gate-g1-resume-3.md#c2--executing-file-mutability-before-binary-pin)
changed bytes before file pinning, preserved device/inode and made active A
consume synthetic B content. Positive bounded native suites do not reverse FAIL.
No product resolver exists; neither universal infeasibility nor a product exploit
was established. The original value is correct offline documentation for installed
operation, not forensic identification of every byte loaded into a process.

## Decision

Propose association of exact documentary release content with an observed valid
installation, using a small descriptor compiled into the running binary. Preserve
external Markdown/man, immutable generations, private namespace and all publication
and read-only access constraints. Do not require OS mapped-region introspection,
private ABI or cgo merely to certify original full-file image identity.

The acyclic descriptor commits to full source/build target, documentary inventory/
coverage/transform identity and expected embedded skills. It excludes its own
binary/archive hash. Build the final executable with that descriptor, then hash
binary and package externally. At discovery compare the running descriptor with
validated generation/binding metadata and current installed binary facts. Derive
location from the physical executable and anchored local namespace, never global
search or untrusted arbitrary path pointers. Missing, crossed, changed, unsafe or
ambiguous facts refuse verified guidance; hashing the replacement path alone
never proves what A originally executed.

C2 must be retained as a supported controlled regression: compiled A plus B docs/
binding refuses on descriptor inequality, including same release labels and same
inode with altered bytes. A running A after replacement may serve only identical
exact documentary content when the complete current association is valid; otherwise
refuse. Consumers revalidate/pin actual bytes. This is **not** a proof of original
whole-file launch bytes or mapped memory identity; those remain unknown. A descriptor
is not cryptographic authenticity or an ownership grant. Altering all mutable local
metadata consistently may defeat local drift checks; existing trusted acquisition
and ADR-0005's bounded same-UID model remain the trust boundary. Controlled observed
invalid state is still refused. Do not hide C2 by relabelling it an excluded attack.

No relaxation of installation target confinement, permission/link checks, stale
publisher protection, complete-state publication or truthful recovery is proposed.
ADR-0016 candidate-running proof and Runtime receipt ownership are independent and
unchanged; the documentary descriptor cannot authorize a skill receipt. No change
to ADR-0018 lifecycle or ADR-0021's Windows withdrawal.

## Alternatives considered

| Alternative | Assessment |
| --- | --- |
| Compiled descriptor + validated installation/content association — recommended | Reuses Go build identity, hashes and anchored local readers; detects crossed content without full-image syscall research. Explicitly relinquishes original whole-file image proof; requires new descriptor integration and native refusal tests |
| Preserve exact old guarantee, strengthen OS image mechanism | Stronger identity if demonstrated; C2 and startup/snapshot proof unresolved. More native research and build/ABI maintenance; no evidence yet establishes a qualified portable mechanism |
| Version/path/hash-only association | Fewer build changes, but cannot distinguish active A from later B or same-label content. Rejected: would repeat the demonstrated inference error |

## Consequences

### Positive

Offline corpus, generated help/man parity, coherent upgrades and A/B diagnosis
remain. Descriptor/content comparison replaces process-image proof research on
the delivery critical path; no new service, registry, library or privileged helper.

### Negative / trade-offs

The original whole-executable-at-launch guarantee is lost, explicitly. This cannot
support forensic assertions about loaded code or coordinated hostile local
substitution. Descriptor prebuild generation must avoid hash cycles and generated
reference changes must update it. Public identity semantics and installation
binding remain lasting compatibility obligations; native qualification still costs
three host rows plus real Codex/Claude observation.

## Proposed partial supersession

On explicit human acceptance, partially supersede
[ADR-0033](0033-installation-bound-operational-documentation.md#corpus-and-generation-identity)
only where “actual executing binary”, “proof of the executing binary” and “old
process … prove its own generation” require original whole-file/mapped-image
identity. Apply the same bounded replacement to its Decision opening and concurrent/
surviving-reader proof wording. New guarantee is the association defined above;
all corpus, namespace, external package, publication, retention, return, discovery,
security, platform and Q6 requirements remain in force. ADR-0033 stays Accepted.

No supersession is active. Once accepted, add reciprocal annotations immediately
after affected fragments, rename this section to `Supersedes` with exact links,
and reconcile the index per ADR-0018. The current governance validator forbids an
actual `Supersedes` section on a Proposed ADR; proposed scope is recorded here
without falsifying acceptance or editing the older ADR.

## Revisit when

The descriptor cannot reject C2's crossed content; a supported native host cannot
establish safe local association/refusal; original image identity becomes a product
requirement; coordinated same-UID tampering, privilege separation, shared hosting
or source-install qualification becomes authorized scope. Stop dependent delivery
rather than silently broadening the guarantee.
