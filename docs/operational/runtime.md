# Runtime

Codex and Claude are installed separately from Axiom. `axiom first-run`
passively detects their executables and installs or updates Axiom's canonical
skills. It does not launch the Runtime, sign in, modify credentials or prove
that a skill is loaded. A missing Runtime is not an Axiom installation failure.

Runtime bindings and Model Profiles are machine-local configuration. Inspect
them before use. Creating or editing a binding/Profile requires a complete
preview and apply bound to its exact expected revision and preview digest, with
`--authorize-local`. A successful configuration write records intent only; it
does not establish that a vendor supports the model or effort choice.

Authentication readiness is a point-in-time login observation. It does not
prove inference, billing or future availability. A Model Execution requires
its own applicable preview and authority. Inspect persisted status and result
records; an acknowledgement or exit code alone does not prove successful work.

## Man pages

Use the installed Markdown/help when no `man` reader is available. The intended
`axiom --json docs inspect` result does not identify the page actually selected
by a reader; `readerSelection` is unknown until independently observed. Where
the discovery result verifies a man root, select it for this shell session only:

```sh
MANPATH="$verified_man_root:" man axiom
```

`verified_man_root` must be the actual verified path from the current
inspection. Compare the selected page's actual path and content digest with
the source inventory. If PATH resolves Axiom installation A while `man` selects
installation B, use A's verified man root explicitly and compare again. Do not
persist a MANPATH change based only on a locator or assume that a page was
selected because it exists.

## Configure a Runtime

Inspect existing bindings and Profiles first:

```sh
axiom --json runtime binding list
axiom --json runtime profile list
```

After selecting an installed Runtime and model intent, propose a binding:

```sh
axiom runtime binding create --runtime codex --profile operator-review \
  --model operator-model --effort-mode runtime-default --effort default \
  --capabilities read --complexities high
```

This is a preview. Apply the same candidate with its returned
`--expected-revision`, `--preview-digest` and `--authorize-local`. Re-preview
after any input or state changes. Example identifiers are local choices;
successful authoring does not prove vendor support or authentication.

A refusal carries `runtimeAuthoring.diagnostics`: each entry names the `code`
(`field_required`, `field_invalid`, `binding_missing`, `profile_missing`,
`runtime_absent`, `configuration_conflict`, `configuration_invalid`), the CLI
`field`, the `rule` and the `next` action, also shown in terminal output. For
example, omitting `--capabilities` returns `candidate_invalid` with
`field_required` on `--capabilities`; `--complexities` stays optional. These
are local declaration checks: authentication is not evaluated, so do not change
credentials or environment variables to resolve them. `validate-candidate`
confirms declaration only; `modelSupport` and `effortSupport` stay
`unconfirmed`.

During guided setup, `axiom-runtime` can act as a Model Advisor: it
recommends models for each Profile from the Workflow's Agent needs and an
optional preference (economy, balanced, quality), citing official vendor
documentation the host retrieves with your permission. Recommendations carry
their source and date and keep account availability and effort support
unconfirmed; API prices are not subscription cost. Without sources it says so
and lets you name the model. Nothing is created until you approve the exact
preview.

Continue to [Workflow](workflow.md) or [Troubleshooting](troubleshooting.md).
