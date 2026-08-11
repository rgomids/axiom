# Data Model: Documented Command Deprecation Process

This is a documentation information model, not an application data model.

## Documented Command

- **Identity**: Exact documented command name.
- **Current instances**: `example validate`; `example package`.
- **Invariant for this slice**: Name, description, and behavior remain unchanged.
- **Relationship**: One command may have zero or more historical deprecation records; none exist now.

## Deprecation Record

- **File identity**: `docs/deprecations/NNNN-short-name.md`.
- **Index relationship**: Every actual record is listed in `docs/deprecations/README.md`.
- **Required fields**:
  - Status
  - Affected Command
  - Replacement (or explicit absence)
  - Rationale
  - Affected Users
  - Migration Steps
  - Notice / Release Target
  - Rollback Plan
  - Owner
  - Approval Evidence
  - Relevant Links
- **Validation rules**: Every required heading exists; absence is stated explicitly where permitted;
  local links resolve; no provider-only context is required.

## Lifecycle Stage

```text
Proposed -> Approved -> Deprecated -> Removed
```

- `Proposed`: Record exists; command status does not change.
- `Approved`: Repository maintainer has approved the record.
- `Deprecated`: Command owner has executed the later transition; the record carries its
  notice/release target.
- `Removed`: Command owner has executed removal after the normal notice minimum or a documented
  urgent-security exception.
- Reverse transitions are not defined. Rollback guidance describes restoration actions, not a
  lifecycle transition invented by this slice.

## Approval Evidence

- Records human approval for transition to `Approved` and any urgent-security exception.
- Remains provider-neutral; a durable repository-local statement or relevant link may carry evidence.
- Exact evidence syntax is not prescribed by the frozen answer.

## Deprecation Index

- **Path**: `docs/deprecations/README.md`.
- **Contents**: Process trigger, lifecycle, authority, notice policy, exception policy, record naming,
  template link, reviewer checklist, and list of actual records.
- **Empty state**: Explicitly states that no command is currently deprecated and lists no fictional
  record.

## Residual Ambiguities

- Trigger coverage beyond explicit command deprecation.
- Allocation and collision handling for the four-digit `NNNN` identifier.
- Provider-neutral proof that a notice-bearing release was published.
