# Troubleshooting and recovery

Start with the command's sanitized diagnostic, status and named next action.
Distinguish a confirmed cause from a hypothesis and a confirmed effect from an
unknown outcome. Do not repeat an external or local mutation when its result is
uncertain.

For a pending local publication, inspect before applying recovery:

```sh
axiom --json recovery inspect
axiom --json recovery apply --preview-digest <plan-digest> --authorize-local
```

Apply only the fresh exact plan digest returned by inspection. Supported
recovery may restore a prior state before commit or finalize a committed state;
a committed generation is never rolled back. Ambiguous, contradictory, corrupt
or unknown state must remain preserved for review.

For an interrupted owned installation upgrade, preserve the marker, receipt and
exact candidate archive. Resume only through the reported supported path with
that same archive and fresh exact authority. Do not delete markers, change
candidates or manually restore files. Confirmed binary publication remains
confirmed if later bookkeeping fails; there is no automatic rollback.

For a refused Runtime binding or Profile, read
`runtimeAuthoring.diagnostics` first: it names the field, rule and next action.
A missing or malformed field, `binding_missing` or `runtime_absent` is not an
authentication failure; investigate authentication only when an auth
observation reports one.

For a failed Runtime skill convergence, inspect the relevant Runtime status and
preserve modified or foreign files. Run `axiom first-run` only when the
diagnostic calls for it. Authentication, credentials and third-party Runtime
configuration remain under their owners.

If documentation is absent, unknown or invalid, retain the installed binary and
data and use normal `axiom help`/CLI documentation. Do not repair, extract or
fetch documentation from the network on the basis of an inspection result.
See [Notices](notices.md) and the [command reference](cli.md).
