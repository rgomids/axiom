# Command Deprecation Record: Short Name

## Status

Use one lifecycle value: `Proposed`, `Approved`, `Deprecated`, or `Removed`. A new record starts as
`Proposed` and changes no command status.

## Affected Command

Name the documented command exactly as it appears in [the command reference](../commands.md).

## Replacement

Name the replacement and where users find it. If none exists, state `None` explicitly.

## Rationale

Explain why deprecation is proposed and why preserving the current command is insufficient.

## Affected Users

Describe who relies on the command and the user impact of deprecation or removal.

## Migration Steps

Provide ordered, actionable migration guidance. For an urgent-security removal, include compensating
migration guidance.

## Notice / Release Target

Identify the notice-bearing published release and intended removal target. If using the
urgent-security exception, state that the normal one-release minimum is bypassed.

## Rollback Plan

Describe restoration triggers and steps. If no viable rollback exists, state that explicitly and
document compensating safety measures for human review.

## Owner

Name the command owner responsible for later lifecycle transitions.

## Approval Evidence

Record repository-maintainer approval for `Approved`. For an urgent-security exception, also record
the required exception approval. Evidence must remain understandable without provider-only context.

## Relevant Links

Link repository-local command documentation, migration material, release evidence, and related
durable context. Use `None` when no additional link exists.
