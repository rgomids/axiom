# Repository Automation Policy

## Purpose

Define which repository automation may be versioned, who owns it, and how its
lifecycle is governed, so that every retained script or workflow has an
explicit, durable reason to exist (Issue #168).

This policy applies to every contributor, human or agent. It does not redefine
security, release or installation contracts; those remain in the
[security policy](security.md), Specifications and ADRs.

## Durable versus ephemeral

**Durable repository automation** may be versioned when it supports a
recurring or reproducible operation:

- CI/CD and release workflows;
- installation and bootstrap;
- deterministic validation;
- reproducible Evidence generation;
- tests, fixtures and test support;
- a documented recurring developer or operator workflow;
- historical replay that a retained contract explicitly needs (for example a
  pinned compatibility fixture or a recovery path).

Manual execution is valid when documented. CI membership alone does not prove
necessity, and an existing reference alone does not justify retention.

**Ephemeral automation.** Agent-created one-off investigation, migration,
repair, transformation, collection or intermediate-generation helpers are
ephemeral by default. They are not durable repository artifacts:

```text
execute → preserve the required sanitized result, provenance and Evidence → discard the helper
```

Run ephemeral helpers outside the tracked tree (a scratch or temporary
directory). Do not commit raw logs, unclassified output or temporary state as
Evidence. A script hard-coded for one incident is not a capability; a reusable
protocol with tests (for example release recovery) is.

Retain an implementation of a past one-off only when replay or audit
demonstrably requires it, registered as `historical-replay` with its replay
reason, owner and removal condition.

## Ownership

Ownership names a capability or responsibility, never an inferred person. The
closed set of capabilities lives in the registry (`capabilities`) and is
changed only by reviewed edits:

`repository-validation`, `security`, `maintainer-harness`, `agent-factory`,
`architecture-verification`, `release`, `delivery`, `installation`,
`compatibility`, `cli-e2e`, `maintainer-acceptance`, `site`, `ci`.

Human reviewers remain those in `.github/CODEOWNERS`. Assigning named
maintainers to a capability is a human decision, not registry metadata.

## Automation registry

[`scripts/automation-registry.json`](../../scripts/automation-registry.json) is
the machine-readable inventory of durable automation. It is metadata only:
nothing reads it to execute commands, and declared values are never evaluated.

### Governed surfaces

A tracked or untracked non-ignored file is a governed automation surface when
it is not a symlink, is not covered by a declared excluded scope, and any of
the following holds:

- it is a workflow under `.github/workflows/` or a composite action under
  `.github/actions/`;
- it has a script or program extension (`.sh`, `.bash`, `.zsh`, `.py`, `.ps1`,
  `.psm1`, `.go`, `.rb`, `.pl`, `.js`, `.mjs`, `.cjs`, `.ts`);
- it is executable or starts with `#!`.

Product and site code are kept out through explicit `excluded_scopes`, each a
directory plus the extensions it excludes and a reason (for example Go under
`cmd/` and `internal/`, browser JavaScript under `site/`). An excluded scope
never covers `scripts/` or `.github/`, and never excludes by extension alone.
A shell or Python helper placed under a product directory is still governed.

Embedded heredocs and generated fake executables belong to the file that
contains or generates them; they are not separate entries.

### Entry contract

Every entry declares exactly these fields. Not-applicable text fields are
`null`; not-applicable lists are `[]`; effects that do not exist are `"none"`.

| Field | Meaning |
|---|---|
| `path` | repository-relative stable entrypoint |
| `kind` | `script` or `workflow` |
| `owner` | one capability from `capabilities` |
| `purpose` | what the automation does |
| `lifecycle` | `durable` or `historical-replay` |
| `disposition` | `keep`, `move`, `merge`, `replace` or `delete` |
| `disposition_ref` | approved work item for a non-`keep` disposition |
| `runtime` | interpreters/runtimes: `bash`, `posix-sh`, `python3`, `powershell`, `go`, `github-actions` |
| `scope` | supported scope: `public`, `maintainer`, `ci`, `test`, `historical` |
| `trigger` | recurring event or operation that runs it (required for `durable`) |
| `replay_reason` | past result it reproduces and why it is kept (required for `historical-replay`, `null` for `durable`) |
| `callers` | tracked files that invoke, source, import or package it |
| `inputs` / `outputs` | arguments, environment, files read; output and exit contract, files written |
| `local_effects` / `network_effects` | effects outside temporary directories; network access |
| `authority` | authorization required before running it |
| `tests` | tracked tests or packages exercising it |
| `evidence` | Evidence produced or consumed, or `null` |
| `compatibility` | pins: `{relation, revision}` (full 40-hex SHA) and/or `{relation, path}` |
| `removal_condition` | when the path may be removed (required for non-`keep` and `historical-replay`) |

`ephemeral` is not a registrable lifecycle. Unknown fields are rejected.
Entries are sorted by `path`.

### Contract expectations

- **Determinism:** validation and Evidence automation produce the same result
  for the same inputs; nondeterministic inputs (time, network, provider state)
  are declared.
- **Side effects and authority:** external, destructive or provider effects
  require the authority declared in the entry and enforced by the automation
  itself; a filename or registry value never grants authority.
- **Safety and idempotency:** effectful automation validates paths, refuses
  unsafe symlinks and traversal, never `eval`s untrusted data, and is safe to
  re-run or refuses explicitly.
- **Tests:** behavioral tests live beside the automation they exercise
  (`scripts/test-*` or the owning package) and run offline without
  credentials.
- **Evidence:** automation emits bounded, sanitized Evidence; temporary
  execution output is not Evidence by itself.
- **Compatibility:** release, installation and recovery entrypoints with
  public, archived or pinned callers keep their paths or a documented
  compatibility facade; historical source revisions stay pinned.

## Lifecycle, moves and removal

A move, merge, replacement or removal must enumerate live callers, packaging and
raw URLs, relative imports, workflow trust revisions and historical
references, and preserve behavior tests and explicit replay paths. Historical
Specifications and Evidence keep their original paths; they are not rewritten
to imply a new location existed historically.

Remove an entry together with its file once its removal condition holds.

## Promotion

Promote behavior out of a script into an owning package, internal repository
tool, application component or CLI when it owns product state or invariants,
durable schemas or authority rules; repeats logic that an existing capability
owns; needs composable typed contracts; or cannot maintain the required
filesystem/process safety in its current form.

Prefer the existing owning package. A maintainer tool may remain a repository
tool. CLI exposure requires a product use case and contract. There is no
language quota, line-count threshold or mandatory Go rewrite.

## Enforcement

`scripts/check-automation-registry.py`, run by
`scripts/validate-repository.sh`, enforces only objective structure: governed
surfaces without an entry, missing or unsafe registered paths, duplicates,
missing or unknown fields, invalid lifecycle, disposition, owner, scope or
runtime values, `durable`/`historical-replay` field consistency, dangling
callers or tests, and compatibility pin shape. It runs offline and never
executes registered automation.

It does not decide future reuse, semantic equivalence, correct ownership,
language choice, invocation frequency, architectural correctness or whether an
effect is authorized. Those belong to engineering and security review.

## Boundary with Issue #154

```text
#168 → automation governance; durable vs ephemeral; ownership;
       lifecycle/location; structural repository enforcement
#154 → test taxonomy; pipeline placement; required checks; blocking gates;
       platform matrix; flakiness/cost; release quality strategy
```

This policy adds no pipeline stage, required status or release gate.
