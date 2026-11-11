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

## Untrusted ancestor owner

`untrusted_ancestor_owner` is an environment trust refusal, not a finding of
corrupt state. Project show/validate/list, Runtime Profile list/validate and
Work Item/Workflow list preserve this diagnostic. It reports only the path class
(`filesystem_root` or `ancestor`), observed owner UID and UID-map observation;
private path components are omitted. Inspection does not mutate state.

On Linux, `uid_mapping=unmapped` means the bounded `/proc/self/uid_map`
observation excludes that UID: a user namespace hides the host owner identity.
An overflow UID such as 65534 does not prove host root ownership. Unknown map
evidence remains unknown. Preserve state; do not delete, repair or reconfigure
it in response to this refusal.

Adding writable roots or moving state beneath one does not remove the ancestor
trust check. Restricted Linux sandbox compatibility remains decision pending
([#387](https://github.com/rgomids/axiom/issues/387)); no overflow-owner exception,
shortened trust chain or external broker is implemented. Keep agent isolation;
`danger-full-access` is not an acceptable mitigation. A reviewed solution must
preserve the filesystem trust boundary before retrying the refused operation.
