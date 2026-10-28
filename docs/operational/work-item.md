# Work Item

A Work Item is a unit of planned or executed delivery work. It may relate to a
Project, Repository, Workflow and Execution, but those associations are
explicit state. A Work Item does not imply that a Provider or Runtime is
available.

Inspect the selected Project and Work Item before planning or resuming:

```sh
axiom --json project context show
axiom work-item create --help
```

Use the installed help for the exact `work-item` subcommand and required
selectors. Planning records proposed work and dependencies. Execution or
dispatch is a separate step with its own preview and explicit authority.
Provider actions, remote writes and publication are not implied by a plan,
green validation or a skill invocation.

For a GitHub-backed create, supply the Project, Repository key and provider
repository. The command begins with a preview; inspect it before the guided
confirmation:

```sh
axiom work-item create --project <slug-or-id> --repository <repository-key> \
  --provider-repository <owner/repository> --type task
```

Discover only the create contract, including when each section is gathered,
with `axiom --json skill inspect axiom-work-item --operation create`. A refusal
names its cause in `details`, for example `unknown_input:--title`,
`draft_incomplete` (answer only the returned `questions`) or a Provider code such
as `provider_unauthenticated`; correct only that input instead of repeating the
command unchanged.

GitHub operations require an authenticated GitHub CLI session and configured
Project integration. This external effect is separate from offline
installation and local Project inspection.

After an attempt, inspect its persisted status and evidence. If the result is
unknown or partial, follow the reported reconciliation/recovery command. Do not
repeat an external effect while its prior outcome is uncertain. See
[Workflow](workflow.md), [Runtime](runtime.md) and
[Troubleshooting](troubleshooting.md).
