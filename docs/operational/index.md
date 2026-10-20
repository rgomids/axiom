# Axiom operational guide

This English guide ships with an Axiom installation for use without a source
checkout or network connection. Start with [Concepts](concepts.md), then
[Installation](installation.md). Use the [command reference](cli.md) for the
installed command tree and exact flags. This file and the linked guides explain
the operational boundaries; they do not grant authority to perform an action.

| Need | Guide |
|---|---|
| Understand Axiom and Lingo terms | [Concepts](concepts.md) |
| Install and verify Axiom | [Installation](installation.md) |
| Select and inspect a Project | [Project](project.md) |
| Configure Runtime and Model Profile | [Runtime](runtime.md) |
| Preview and run a Workflow | [Workflow](workflow.md) |
| Create, inspect, and run Work Items | [Work Item](work-item.md) |
| Review local configuration | [Configuration](configuration.md) |
| Diagnose interruption or uncertainty | [Troubleshooting and recovery](troubleshooting.md) |
| Read scope and support notices | [Notices](notices.md) |

The intended installed documentation discovery command is `axiom --json docs
inspect` (human form: `axiom --human docs inspect`). It reports installation-bound
documentation facts. It does not certify that a man reader selected a particular
page or that a Runtime loaded particular skill bytes. See [man selection](runtime.md#man-pages).

## Source mapping

Authored pages are drawn from public user-facing command help and operational
documentation. The source inventory is explicit in `allowlist.json`; the
generator records source revisions and hashes in the installed provenance.
Generated `cli.md` is derived from the public CLI command and flag tree, not
copied from a historical specification or maintainer guide.

Continue to [Notices](notices.md) for limitations and compatibility boundaries.
