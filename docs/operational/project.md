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

## Guided setup order

Ask `axiom-project` to configure a Project ready to execute; it coordinates
the owners in this order, each with its own preview, exact authority and
read-back:

1. publish the Project identity, Repositories, Work Item Provider and context
   without Runtime policy (a bootstrap: execution stays blocked);
2. list Workflow definitions, choose one explicitly and read its Agents'
   capabilities, role/complexity and Runtime constraints with
   `project workflow show`;
3. create or reuse Runtime bindings and Profiles (`axiom-runtime`,
   machine-local and shared by every Project) with those capabilities;
4. declare the portable Runtime policy with `--policy-draft`, before selection:
   policy editing is refused once a Workflow is selected;
5. select the Workflow, then run `project validate`.

The final summary keeps configuration validity, Runtime readiness and
authentication (not verified unless observed) separate; nothing executes
during setup. Without a selected Workflow, Runtime readiness is reported as
`inventory_unavailable`, not ready.

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
