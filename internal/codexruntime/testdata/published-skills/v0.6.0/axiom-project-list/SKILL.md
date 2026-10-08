---
name: axiom-project-list
description: List configured Axiom Projects through Lingo from any working directory.
---

# List Axiom Projects

To inspect accepted arguments before starting this workflow, run
`axiom --json skill inspect axiom-project-list` and report its
`skill` payload. Inspection requests stop there: do not collect inputs or execute
any command below. The binary owns argument names, descriptions, required and
conditional inputs, and accepted forms; do not maintain a separate argument list.
For workflow invocation, preserve the guided behavior below.

Run `axiom --json project list` and report the returned `projects` collection.
Never enumerate filesystem state, infer Project identity from the Runtime current
working directory, or resolve Projects independently.

Canonical completion fields: `status`, `result`, `references`, `next`, `details`, `provenance`
Operation-specific payloads preserved separately: `projects`

Copy canonical completion fields only from Lingo's top-level JSON object. Omit
canonical fields absent from that object. Never derive, synthesize, or reinterpret
a canonical field from `projects` or another operation-specific payload. Preserve the `projects` array exactly
as the operation-specific payload, including an empty array and an empty `name`
when portable display metadata is unavailable. Never derive or reinterpret
canonical fields from a Project entry.
