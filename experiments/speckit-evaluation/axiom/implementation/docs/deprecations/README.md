# Command deprecations

Use this repository-local process before a command documented in
[`docs/commands.md`](../commands.md) is renamed, marked deprecated, or removed.
This slice does not decide whether behavior, argument, output, or
documentation-only changes that do not deprecate a command require a record.

Creating a `Proposed` record changes no command status.

## Record convention

Create records in this directory from [`template.md`](template.md). Name each
record `NNNN-short-name.md`, where `NNNN` is the next unused four-digit
identifier and `short-name` is a concise lowercase hyphenated label. Add every
record to the index below.

## Lifecycle and authority

```text
Proposed -> Approved -> Deprecated -> Removed
```

| Status | Meaning | Authority |
|---|---|---|
| `Proposed` | Record is under review; command remains supported and unchanged. | Contributor creates the record. |
| `Approved` | Record content and intended deprecation are approved. | Repository maintainer approves this transition. |
| `Deprecated` | Users have received a deprecation notice and migration guidance. | Command owner executes this transition. |
| `Removed` | Command has been removed after the applicable notice rule. | Command owner executes this transition. |

## Create a record

1. Copy `template.md` to the next unused `NNNN-short-name.md` path.
2. Keep initial status `Proposed` and complete every required section.
3. Add the record to the index.
4. Request repository-maintainer review without changing command status.
5. Record approval evidence before changing status to `Approved`.
6. Let the command owner perform later lifecycle transitions and update the
   record as evidence changes.

The process requires no issue tracker, hosted source-control feature,
credential, or network access.

## Notice and urgent security exception

Normal removal follows at least one published release containing the
deprecation notice.

Urgent security removal may bypass that minimum only when the record states:

- exception rationale;
- user and operational impact;
- owner;
- repository-maintainer approval explicitly covering the exception;
- compensating migration guidance.

The command owner records this evidence before transition to `Removed`.

## Reviewer checklist

- [ ] Change is a covered rename, explicit deprecation, or removal.
- [ ] Filename follows `NNNN-short-name.md` and the index contains the record.
- [ ] Status is one of `Proposed`, `Approved`, `Deprecated`, or `Removed`.
- [ ] Affected command is identified exactly.
- [ ] Replacement is identified or its absence is explicit.
- [ ] Rationale and affected users explain user impact.
- [ ] Migration steps are complete.
- [ ] Notice or release target is explicit.
- [ ] Rollback plan is complete.
- [ ] Owner and approval evidence are present.
- [ ] Relevant links are present and resolve locally when repository-relative.
- [ ] `Proposed` record has not changed command status.
- [ ] Normal removal follows one published release containing notice, or the
      urgent security exception contains every required item above.
- [ ] Record requires no provider, credential, or network access to understand.

Missing required migration steps or rollback plan blocks approval.

## Record index

No command deprecation records exist.

| ID | Command | Status | Owner | Notice or release target |
|---|---|---|---|---|
