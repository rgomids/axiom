# Configuration

Axiom keeps portable Project content separate from machine-local preferences,
Runtime bindings, Profiles, credentials and observations. A Project is not a
Repository, and local Runtime configuration does not become Project policy
unless an explicit supported operation records that relationship.

Inspect the current Project context and Runtime/Profile readiness before
authoring configuration. Configuration readiness, Runtime executable presence
and authentication readiness are separate observations. None proves successful
Model Execution.

For every mutating configuration command, review the complete preview and
listed effects. Apply only the same candidate with its exact expected revision
and preview digest and the required explicit authority. A changed input,
dependency or state invalidates that preview. Removal, cascade and migration
are separate operations with their own preview. Historical formats are not
silently rewritten or downgraded; use the supported command's explicit
migration/recovery path.

Use `axiom --help` and `axiom runtime --help` for the installed command tree.
See [Project](project.md), [Runtime](runtime.md) and the
[command reference](cli.md).
