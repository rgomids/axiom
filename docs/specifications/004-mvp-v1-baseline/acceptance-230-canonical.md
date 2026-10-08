# Issue #230 — Isolated macOS native acceptance

Status: **PENDING**. Instructions are reproducible acceptance steps, not Evidence
that Codex, Claude or GitHub acceptance ran. Local fake/stub tests do not prove
native semantic selection. A real run requires a human-approved disposable
Runtime environment and an exact isolated GitHub target/effect envelope.

## Local binary and roots

Run from the candidate repository on macOS arm64. This block writes only a new
temporary sandbox and builds current source; it does not install into real user
Runtime roots. Keep the sandbox path to inspect results before removing it.

```bash
AXIOM_ACCEPTANCE_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/axiom-canonical.XXXXXX")
chmod 700 "$AXIOM_ACCEPTANCE_ROOT"
mkdir -p "$AXIOM_ACCEPTANCE_ROOT"/{home,projects,state,codex,claude,skills,bin,repo}
export HOME="$AXIOM_ACCEPTANCE_ROOT/home"
export LINGO_PROJECTS_ROOT="$AXIOM_ACCEPTANCE_ROOT/projects"
export LINGO_STATE_ROOT="$AXIOM_ACCEPTANCE_ROOT/state"
export AXIOM_CODEX_SKILLS_ROOT="$HOME/.agents/skills"
export CODEX_HOME="$AXIOM_ACCEPTANCE_ROOT/codex"
export CLAUDE_CONFIG_DIR="$AXIOM_ACCEPTANCE_ROOT/claude"
go build -o "$AXIOM_ACCEPTANCE_ROOT/bin/axiom" ./cmd/lingo
export PATH="$AXIOM_ACCEPTANCE_ROOT/bin:$PATH"
axiom --json runtime codex install
axiom --json runtime claude install
axiom --json runtime codex status
axiom --json runtime claude status
axiom --json skill inspect axiom-project
axiom --json skill inspect axiom-work-item
find "$AXIOM_CODEX_SKILLS_ROOT" "$CLAUDE_CONFIG_DIR/skills" -name SKILL.md -print
axiom --json skill inspect axiom-project-show
```

Expect exactly two Axiom directories per skill root and a current receipt.
Legacy inspect must fail. The source build above proves current local bytes;
a development release archive additionally verifies bundle parity:

```bash
./scripts/build-release-archives.sh --version 9999.0.0-acceptance.1 \
  --development --output "$AXIOM_ACCEPTANCE_ROOT/artifacts"
./scripts/test-release-archives.sh
./scripts/test-codex-skills.sh
```

A clean release-provenance upgrade must use the approved immutable candidate
archive; do not interpret a dirty development build as release acceptance.
Run historical archive journeys with checksum-verified downloaded releases:

```bash
./scripts/test-upgrade-journeys.sh \
  --candidate /absolute/candidate-artifacts \
  --previous /absolute/v0.10.0-artifacts \
  --previous /absolute/v0.6.0-artifacts
```

## Native Runtime protocol

Use a disposable macOS account or VM for the real Runtime session. This prevents
unrelated installed user-global skills, credentials, MCP integrations and
Runtime settings from contaminating selection. Install candidate skills there
only after approval. Launch Codex and Claude through their supported interfaces
in that environment; authenticate explicitly without copying credentials into
this repository. Runtime bootstrap detection does not prove Runtime discovery.

For each turn below, start from explicit target facts and inspect the canonical
skill's argument contract. Record Runtime/version/model, actual selected skill,
argv, exit, canonical JSON, payloads and effects. Compare with direct CLI using
identical arguments and state; normalize only declared provenance/timing fields.
Do not retry mutating commands to compare: compare previews first, and execute
an authorized effect once on the disposable target. Keep sanitized Evidence.

`P` = supplied Project slug/UUID; `R` = Project-scoped repository key;
`W` = exact `github:owner/repository#number`; `E` = exact Execution ID.
Required/conditional arguments remain discoverable through `skill inspect`.
These examples specify concrete requests, not an independent argument schema.

| User request | Canonical operation/mode | CLI route and inputs to collect | Authority |
|---|---|---|---|
| Liste meus projetos. | project/list | `project list`; `--include-archived` only when requested | read-only |
| Mostre o projeto X. | project/show | `project show --selector X` | read-only |
| Valide a configuração desse projeto. | project/validate | `project validate --project P` | read-only; obtain missing P |
| Adicione este repositório ao projeto. | project/configure/edit | `project configure --project P --repository R=/absolute/path` | reviewed edit + returned Project ID/digest + local authority |
| Arquive o projeto X. | project/archive | `project archive --project X` | reviewed operational digest + local authority |
| Reative o projeto X. | project/reactivate | `project reactivate --project X` | reviewed operational digest + local authority |
| Desabilite a integração work-items. | project/integration/disable | `integration disable --project P --integration work-items` | reviewed operational digest + local authority |
| Mostre as integrações configuradas. | project/integration/list | `integration list --project P` | read-only |
| Crie uma issue para este problema. | work-item/create/new | `work-item create --project P --repository R --provider-repository owner/repository` plus inspected draft arguments | reviewed draft digest + explicit external authority |
| Liste os Work Items do projeto. | work-item/list | `work-item list --project P` (optional R) | read-only linked items |
| Atualize o título dessa issue. | work-item/update | `work-item update --project P --repository R --work-item W --title text` | reviewed change digest + external authority |
| Comente na issue. | work-item/comment | `work-item comment --project P --repository R --work-item W --message text` | reviewed digest + external authority; repeated comment is another effect |
| Feche a issue. | work-item/close | `work-item close --project P --repository R --work-item W` | reviewed digest + external authority |
| Reabra a issue. | work-item/reopen | `work-item reopen --project P --repository R --work-item W` | reviewed digest + external authority |
| Liste as execuções. | work-item/status/list | `workflow list --project P` (optional R) | read-only |
| Mostre o status da execução. | work-item/status/default | `workflow status --project P --repository R --work-item W --execution E` | read-only |
| Continue a execução. | work-item/run/transition | inspect status first, then `workflow resume` with persisted P/R/W/E and exact revision | authorized run; Lingo state/gate/revision rules |
| Mostre as evidências. | work-item/status/default | `workflow evidence --project P --repository R --work-item W --execution E` | read-only |

Also test explicit operations, missing/conditional arguments, unsupported
`delete`/sequential `cancel`, and ambiguous "resolva isso", "remova o projeto"
and "limpe as issues". Ambiguity must clarify/fail, with zero mutation.
Inspect failures and payload placement rather than inferring success from prose.

## Real GitHub Provider gate

Before any real GitHub run approve one disposable repository, account, Project,
repository association, Issue selector (or one exact CREATE draft), preview
revision/digest and allowed create/update/comment/close/reopen effects. No
production repository/Issue is valid. Read-only Provider access still requires
an isolated configured integration. External authority for one preview never
authorizes subsequent edits/comments or cleanup/deletion.

Check preview → exact authorization → read-back for each separately approved
effect. Check stale preview refusal, no-op close/reopen, disabled integration and
archived Project denial, unknown/partial failure reporting. Record both Runtime
and CLI equivalence; never grant authority merely because a Runtime selected a
skill. Human acceptance remains a separate decision.
