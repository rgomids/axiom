# Documented Command Deprecations

This directory contains the repository-local process and records for documented command
deprecations. It remains usable from a local checkout without chat history, credentials, network
access, or a specific provider.

Use the [record template](template.md) when this process requires a record.

## When a Record Is Required

Create a record before proposing that a documented command enter the deprecation lifecycle. A
command cannot reach `Removed` without an earlier record and lifecycle transitions.

An editorial correction that preserves command name, description, behavior, and lifecycle does not
require a record. The current policy does not decide whether a rename, replacement, or incompatible
behavior change independently triggers this process. Obtain an explicit human decision before
treating any such proposal as covered or exempt.

## Lifecycle and Authority

Every record moves only through this documented order:

```text
Proposed -> Approved -> Deprecated -> Removed
```

- `Proposed`: The record exists. Creating it changes no command status.
- `Approved`: A repository maintainer has approved transition to this state.
- `Deprecated`: The command owner executes this later transition.
- `Removed`: The command owner executes this later transition after satisfying the notice policy or
  its urgent-security exception.

The process defines no reverse transition. A record's rollback plan describes how to restore safe
user operation; it does not silently change lifecycle state.

## Notice and Urgent-Security Exception

Removal normally follows at least one published release containing the deprecation notice.

An urgent security removal may bypass that minimum only when the record states all of:

- rationale;
- impact;
- owner;
- approval; and
- compensating migration guidance.

The repository does not define a provider-specific proof of publication. Preserve relevant release
evidence in the record and leave any unresolved sufficiency question for human review.

## Create and Name a Record

1. Copy [the reusable template](template.md).
2. Store the new record in this directory as `NNNN-short-name.md`.
3. Use four digits for `NNNN` and a concise hyphenated label for `short-name`.
4. Complete every required section. Use an explicit absence only where the template permits it.
5. Add the record to the index below before requesting review.

The frozen decision does not define how contributors allocate `NNNN` or resolve competing choices.
Do not infer a numbering authority or sequence rule. Any collision must remain unresolved until a
human chooses the record identifier.

## Deprecation Records

No command is currently deprecated. No deprecation records exist.

| Record | Status | Affected command |
|--------|--------|------------------|
| None | — | — |

## Reviewer Checklist

Do not approve the record until every applicable item is satisfied:

- [ ] File is under `docs/deprecations/`, matches `NNNN-short-name.md`, and appears in the index.
- [ ] Status is one of `Proposed`, `Approved`, `Deprecated`, or `Removed` and matches lifecycle order.
- [ ] Affected command matches the command reference exactly.
- [ ] Replacement is named or its absence is explicit.
- [ ] Rationale and affected users make user impact reviewable.
- [ ] Migration steps are ordered and actionable.
- [ ] Notice/release target identifies the notice-bearing release and intended removal target.
- [ ] Rollback plan contains triggers and steps, or explicitly explains infeasibility and safeguards.
- [ ] Owner is named and approval evidence shows repository-maintainer approval before `Approved`.
- [ ] Relevant repository-local links resolve and provider-only context is unnecessary.
- [ ] Any urgent-security exception states rationale, impact, owner, approval, and compensating
  migration guidance.
- [ ] No required migration, rollback, authority, or approval information remains unresolved.
- [ ] Local validation commands below pass and scope remains documentation-only.

## Local Validation Commands

Run from repository root. These are one-off review commands, not an executable validator.

```sh
test -f docs/deprecations/README.md
test -f docs/deprecations/template.md

for heading in \
  'Status' 'Affected Command' 'Replacement' 'Rationale' 'Affected Users' \
  'Migration Steps' 'Notice / Release Target' 'Rollback Plan' 'Owner' \
  'Approval Evidence' 'Relevant Links'; do
  rg -Fqx "## $heading" docs/deprecations/template.md
done

test -f docs/commands.md
test -f docs/contributing.md
test -f docs/deprecations/README.md
test -f docs/deprecations/template.md
test -f CHANGELOG.md

! rg -n '[[:blank:]]+$' README.md CHANGELOG.md docs

test "$(rg -c '^## `example (validate|package)`$' docs/commands.md)" -eq 2
rg -Fqx 'Validates repository documentation before review.' docs/commands.md
rg -Fqx 'Packages validated repository documentation for distribution.' docs/commands.md

git status --short --untracked-files=all
```

Review `git status` and reject any application code, executable validator, dependency, CI,
provider-integration, credential, network requirement, or command-description change.
