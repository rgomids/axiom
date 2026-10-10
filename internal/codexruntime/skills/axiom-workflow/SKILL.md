---
name: axiom-workflow
description: Configure Project workflow definitions through canonical Lingo operations.
---

# Axiom Workflow

To inspect supported operations and accepted arguments before execution, run
`axiom skill inspect axiom-workflow` and present its `skill` payload.

Supported domain operations are `definition.list`, `definition.show`, `definition.create`, `definition.edit`, `definition.validate`, `definition.remove`, `definition.recover`, and `configuration.readiness`.

## Operation routing

| Operation | Mode | Lingo command | Effect | Authority | Semantic resolution |
|---|---|---|---|---|---|
| `definition.list` | - | `axiom project workflow list --project <uuid-or-slug>` | read-only | none | allowed |
| `definition.show` | - | `axiom project workflow show --project <uuid-or-slug>` | read-only | none | allowed |
| `definition.create` | - | `axiom project workflow create --project <uuid-or-slug>` | local mutation | preview first; exact `--expected-revision`, `--preview-digest` plus `--authorize-local` | only for unambiguous workflow intent |
| `definition.edit` | - | `axiom project workflow edit --project <uuid-or-slug>` | local mutation | preview first; exact `--expected-revision`, `--preview-digest` plus `--authorize-local` | only for unambiguous workflow intent |
| `definition.validate` | - | `axiom project workflow validate --project <uuid-or-slug>` | read-only | none | allowed |
| `definition.remove` | - | `axiom project workflow remove --project <uuid-or-slug>` | local mutation | preview first; exact `--expected-revision`, `--preview-digest` plus `--authorize-local` | only for unambiguous workflow intent |
| `definition.recover` | - | `axiom project workflow recover --project <uuid-or-slug>` | local mutation | preview first; exact `--expected-revision`, `--preview-digest` plus `--authorize-local` | only for unambiguous workflow intent |
| `configuration.readiness` | configuration | `axiom runtime profile validate`, `axiom runtime profile preview` | read-only | none | allowed |
| `configuration.readiness` | runtime | `axiom runtime codex status`, `axiom runtime claude status` | read-only | none | allowed |
| `configuration.readiness` | authentication | `axiom runtime codex auth`, `axiom runtime claude auth` | read-only | none | allowed |
