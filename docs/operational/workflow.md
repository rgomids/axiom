# Workflow

A Workflow definition describes stages and their configuration. A Project may
select a definition; creating or editing a definition does not select it.
Inspect available definitions and current selection before planning work:

```sh
axiom --json project workflow list --project <slug-or-id>
axiom --json project show --selector <slug-or-id>
axiom --json project workflow create --project <slug-or-id> \
  --from-default --workflow team-sdd
```

Use `axiom project workflow ... --help` for the installed command's exact
inputs. Validate a complete definition before proposing create/edit. Review the
preview, effects, revision and digest. Apply by repeating the same operation
with its exact returned authority values and `--authorize-local`. Selection is
a separate explicit operation with its own preview and authorization.

Work Item planning binds to the selected Workflow revision and explicit
Execution context. A plan is not dispatch authority. Dispatch, retries and
external Provider effects require their own command-specific preview and
authority. Inspect status and recorded results after an attempt. For interrupted
publication, use the owning recovery path described in
[Troubleshooting](troubleshooting.md); do not retry blindly.

See [Work Item](work-item.md) and the [command reference](cli.md).
