# Effective Project context — Issue #233

Implementation scope authorized by the user's request to implement Issue #233.
The dependency #231 was delivered by PR #257 and released in v0.8.0. This
artifact records the behavior and plan; it does not claim human acceptance,
merge, release publication or an architectural change to Project identity.

## Contract

Resolve one configured Project in this order: explicit operation selector,
session override, persistent machine/user-local default, unresolved error.
The winning selector is validated with the existing installed Project and
recorded portable-source contract. A missing/deleted/ambiguous/stale winner
fails; CWD, Git remotes, chat history and a different tier never repair it.
An explicit selector bypasses lower preferences, including stale/corrupt ones.
Inspection of all tiers deliberately validates and exposes stale preferences.

Preferences persist canonical Project UUIDs, never slugs or local paths, in
the protected native state root at `project-context/v1/`. The portable schema
does not change. `default.json` holds one UUID; `session-<sha256>.json` holds
the override of one opaque caller-provided session ID. Format v1 has exactly
`formatVersion` and `projectId`; an empty `projectId` means cleared. Unknown
versions/keys, duplicates, nulls, unsafe filesystem objects and pending
publication/recovery state fail closed. Writes reuse protected-root locking
and the existing atomic publication protocol. Inventory recognizes the new
kind for compatibility/upgrade inspection.

The caller passes a unique session identifier using `--session <id>` before
the command, after the output-format flag. Use a fresh identifier for each
new Runtime session; never use the Runtime name alone as session identity.
The identifier is 1–128 ASCII letters, digits, hyphens or underscores. No
session identifier means default-only context. `session-clear`/`session-end`
clear that session's override; a new identifier naturally inherits the default.
These commands do not end a Codex/Claude conversation. Runtime shutdown is
caller-owned; no Runtime lifecycle hook or chat-history inference is added.

Local preference mutations require `--authorize-local`. Explicit operation
selection never writes preferences. Existing Work Item and Execution authority
and readiness checks remain independent. CREATE/EDIT Project configuration
keeps its explicit mode selector and never inherits context.
Mutation completion reports the committed preference update separately from
effective resolution; `context.issue` exposes any remaining stale/unresolved
context without misreporting a successful write as an uncommitted failure.

Project show/resolve, Work Item commands, Runtime profile previews and workflow
commands may omit the Project selector. CLI and application entrypoints share
the effective resolver. Workflow start receives the canonical UUID before
preview/start and stores it in the existing Execution record. A context change
between preview and start changes the preview digest and cannot start the
unreviewed target. Stored Executions, transitions, Evidence and provenance
remain ID-addressed and unchanged. To inspect/resume the original Execution
after changing context, name its original Project explicitly.

## Implementation plan

1. Add the application precedence/inspection/set/clear use cases.
2. Persist versioned local preferences with protected storage and inventory.
3. Expose context operations and explicit session selection through the CLI.
4. Resolve omitted Project selectors before interactive prompts and operation
   preflight; preserve every existing explicit-selector path.
5. Verify precedence, persistence, isolation, stale selectors, no effects,
   Execution binding, compatibility and repository regression.

No new external dependency, synchronized state, credential contract or
Project identity semantics is introduced.

See [validation Evidence](evidence-issue-233.md).
