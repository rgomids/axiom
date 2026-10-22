# Project

A Project is selected explicitly by a stable UUID or slug. An explicit selector
wins. When omitted, Axiom uses the session selection and then the persistent
local default; a stale winning selection fails instead of guessing from the
working directory or Git repository.

Inspect and select with the command reference for exact syntax:

```sh
axiom --json project list
axiom --json project context show
axiom project show --selector <slug-or-id>
```

Set or clear a local default only through the corresponding explicit command.
Session selection is temporary and does not change the persistent default.
Project configuration and Repository association are separate facts. Listing
a Repository nearby, finding a `.git` directory or seeing a skill suggestion
does not associate it with a Project.

## Local roots

| Data | Default | Override |
|---|---|---|
| Portable Project content | `$HOME/.axiom/projects` | `LINGO_PROJECTS_ROOT` (absolute path) |
| Machine-local state on Linux | `$XDG_STATE_HOME/lingo` when `XDG_STATE_HOME` is absolute; otherwise `$HOME/.local/state/lingo` | `LINGO_STATE_ROOT` (absolute path) |
| Machine-local state on macOS | `$HOME/Library/Application Support/Lingo` | `LINGO_STATE_ROOT` (absolute path) |

`LINGO_STATE_ROOT`, when set, takes precedence over platform defaults. It must
be absolute and cannot be the filesystem root. `XDG_STATE_HOME` affects only
the Linux default. Keep portable content and machine-local state in distinct
roots.

Set a persistent default Project with
`axiom --json project context default-set --selector <slug-or-id>
--authorize-local`; inspect it with `axiom --json project context show`. If the
command returns a preview digest and expected revision, apply only those exact
values.

When a command proposes changes, inspect its full preview and effects. Apply
only with the exact returned digest and expected revision, plus the required
authorization flag. If the preview is stale or any dependency changes, obtain a
new one. See [Concepts](concepts.md), [Configuration](configuration.md), and the
[command reference](cli.md).
