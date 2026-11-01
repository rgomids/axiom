# Project Commands

## Run and inspect a canonical Execution

`workflow run` resolves an already-linked Work Item and its Project Repository,
then starts or continues one format-4 Execution. With `--workflow minimal`, it
uses one work stage and a required `result` output. Without that override it
uses the selected Project Workflow, or the minimal default if none is selected.

```bash
axiom --json workflow run --project my-project --repository main \
  --work-item github:owner/repository#123 --workflow minimal \
  --select agent.runtime=codex --select agent.model=<chosen-model> \
  --select agent.effort=runtime-default --authorize-external
axiom workflow stage status --project my-project --execution <execution-id>
axiom workflow stage cancel --project my-project --execution <execution-id> --authorize-local
```

Per-agent overrides use `[<stage-id>/]<agent-id>.<field>=<value>`; fields are
`runtime`, `model`, `effort`, `instructions`, and optional `profile`. Explicit
values override the corresponding optional Profile defaults. Missing/conflicting
inputs refuse. Explicitly configure reusable machine-local defaults with a bounded
JSON array in `LINGO_INVOCATION_DEFAULTS`, for example:

```bash
export LINGO_INVOCATION_DEFAULTS='["runtime=codex","model=my-model","effort=runtime-default"]'
axiom workflow run --project my-project --repository main \
  --work-item github:owner/repository#123 --authorize-external
```

Each entry is `runtime|model|effort|instructions|profile=value`. The optional
`profile` supplies defaults at that level. Per field, run overrides win over
Workflow agent values, which win over local defaults. No candidate is ranked;
conflicting entries, malformed JSON and missing selectors refuse. The array is
limited to 16 entries and 64 KiB. Keep secrets out of these inputs.

Configured `artifact-schema` and `evidence-check` validators using
`builtin-sdd-v1` accept bounded JSON output references (`kind`, `id`, `digest`)
to retained typed artifacts. Validation checks schema, category, successful
outcome, content digest and exact definition/stage/output/attempt/repository
correlation; Evidence checks additionally require evidence retention. The
launcher packet supplies the definition digest and output checks. References
use an unprefixed SHA-256 digest; the Execution records the retained artifact ID
and prefixed content digest. Required metadata references are `workflow-stage`,
`workflow-definition` (unprefixed digest), `workflow-output`, `workflow-attempt`
and `workflow-repository`, each exactly once.

Required outputs must be present; declared optional outputs are validated when
present and may be absent. Work Item inputs come from `target`; earlier-stage
outputs require a completed successful predecessor in the same Repository.
`project-context` with source `project` freezes portable Project identity,
revision and Repository identities, without local paths. Artifact inputs freeze
retained successful artifacts with exactly one matching Repository, source stage
and source attempt reference, plus the source Execution identity, selected by
`--input <declared-source>=<retained-artifact-id>`. Source stage and attempt use
`workflow-stage` and `workflow-attempt`; missing or duplicate provenance refuses.
Earlier-stage input packets retain their producing Execution/stage/attempt IDs.
Undeclared bindings, missing
required inputs and unsupported policies refuse rather than disappearing.

A `human-review` validator waits for the exact successful stage result. Inspect
`resultDigest`, then after actual human review continue with
`workflow run --execution <id> --review-stage <stage> --result-digest
<sha256:...> --evidence <retained-id> --authorize-human --authorize-external`.
The retained Evidence must be successful, category/retention `evidence`, belong
to this Execution and correlate exactly once with `workflow-stage`,
`workflow-definition` (prefixed definition digest), `workflow-result` (the exact
result digest) and `workflow-repository`. Agent messages cannot grant approval.
Result review is distinct from before/after human gates: gates use
`--approve-gate <stage> --authorize-human`, and a before-gate starts no attempt.

Schema-2 `command` validators under `builtin-sdd-v1` declare bounded `argv`,
`timeoutSeconds` and `effectCeilings`. Running them additionally requires
`--authorize-commands` on run or retry, authorizing those exact configured
arguments and effects. No shell expansion, fallback command or implicit authority
is supplied. The frozen attempt retains selections and authority; ordinary
retries preserve the invocation and require current effect authorization.
Eligible work progresses serially; parallel scheduling and cross-Runtime
collaboration remain S2. Cancellation, failed required validation and unknown
effects stop successors. See the [operational guide](operational/workflow.md).

Retry preserves the frozen invocation unless `--new-selection` explicitly permits
fresh values. Previous attempt snapshots remain immutable. If the prior attempt
was allowed workspace writes, inspect that workspace and supply
`--disposition preserve`. No automatic reset is supported.

```bash
axiom workflow stage retry --project my-project --execution <execution-id> \
  --stage work --agent agent --disposition preserve --authorize-external
axiom workflow stage retry --project my-project --execution <execution-id> \
  --stage work --agent agent --new-selection --select agent.model=<chosen-model> \
  --disposition preserve --authorize-external
axiom workflow stage reconcile --project my-project --execution <execution-id> --authorize-local
```

Reconcile observes lost-runner attempts; it never signals or replays them. An
unknown attempt requires an explicit human disposition (`--abandon <attempt-id>
--authorize-human`) before any eligible retry. Cancellation records intent;
status distinguishes that request from the runner's observed process stop.
Exit zero without every declared output is failure. Neither technical completion,
a process stop nor abandonment proves human acceptance or reversal of effects.

`workflow stage plan`, `run` and `integrate`, and the old lifecycle
`start`, `advance`, `fact`, `resume` and Provider projection `reconcile` operations
are retired. Use `workflow run` and the canonical stage controls. Historical
Execution/Graph/Single bytes are inert: there is no legacy resume or conversion.
Public child publication remains unavailable; S1's private engine ingress does
not expose the S2 collaboration transport. Delivery packet readiness is read-only;
publication, acceptance and rework effects remain S3 work. No real installed-state
operation is implied by these examples. The separately authorized stop-or-block
procedure is in the [operational guide](operational/workflow.md).

## Author and select Project workflows

Project selects, Workflow configures, Work Item runs, Execution performs.
`axiom-workflow` owns definition list/show/validate/create/edit/remove/migrate/recover
and read-only configuration readiness. `axiom-project` keeps `workflow.select`;
`axiom-work-item` keeps `run`, `status`. Explicit removed Project
`workflow.*` authoring skill operations are unsupported; canonical CLI commands
remain unchanged.

Ask which workflow is active through `axiom project show --selector <slug-or-id>`
or `axiom project list`. Their additive, read-only `activeWorkflow` field reports
`selected`, `none` or `unresolvable`; a resolved selection includes its name.
These reads never select, repair or fall back. Changing selection requires a
separate exact preview and approval.

The workflow skill prepares complete conversational JSON drafts as regular files
in Runtime scratch outside Project, Repository and Axiom state roots. It validates,
previews, obtains approval of the exact digest/revision, applies, then reads back.
Creation/editing never selects. `--file` symlink/non-regular-file hardening remains
a known limitation owned by [#307](https://github.com/rgomids/axiom/issues/307).
Readiness reports Configuration Valid, Runtime Ready and Authentication Ready as
separate point-in-time facts. Auth is read-only and never blocks authoring.
Project policy editing (#305), local Profile authoring (#306), global Doctor
(#231), dispatch (#276) and delivery publication/rework (#277) are outside the configuration surface. Delivery packet preview/inspection is documented below.
Work Item stage-plan confirmation binds to `planDocumentDigest`, Execution ID and
expected revision, `stageId`, `workflowRef` and `plan.digest`; provisional
`approved: true` never grants authority. Synthetic R-1 evidence does not establish
native operability (R-2) or human MVP acceptance (R-3), deferred to #278.

Workflow authoring uses the same native terminal presentation as other
commands. Use `--json` when machine-readable output is required; both views
preserve the complete `workflowAuthoring` payload and exact preview values.

`axiom project workflow` manages definitions; `axiom workflow list` continues
to list Executions. Start with the ready-to-use default:

```bash
axiom --json project workflow list --project my-project
axiom project workflow select --help
```

Use the exact `workflowId`, `revision`, `digest` and `source` from the listing
with `select --workflow <id> --revision <n> --digest <sha256> --source <source>`.
Selection is explicit. Review `workflowAuthoring.effects`, then repeat the same
command with `--expected-revision <projectRevision> --preview-digest
<previewDigest> --authorize-local` from the preview.

Create a custom copy without manually defining stages:

```bash
axiom --json project workflow create --project my-project --from-default --workflow team-sdd
```

Apply its exact preview with the same authority tuple. Use `show` with an exact
reference to inspect the complete definition; save your edited JSON as a new
revision and use `validate --file <json>` followed by `edit --file <json>
--workflow <id> --prior-revision <n> --prior-digest <sha256>`. `create --file`
imports a complete validated definition under a new workflow key. Existing keys
require `edit` and its exact published prior reference; retired assignments also
reserve the key. An exact published create replay is unchanged, and
`--from-default` always proposes revision 1 rather than allocating an edit.
Creating/editing never changes selection; run `select` explicitly for the new reference. All commands require
`--project <uuid-or-slug>` and have `--help`; their argument/authority metadata
is also exposed by `axiom skill inspect axiom-workflow` for definition configuration
and `axiom skill inspect axiom-project` for selection.

`remove` retires an unselected, unreferenced revision while retaining inspectable
content and its immutable index assignment. Built-ins cannot be retired. Unknown
local reference/recovery formats block retirement. For interruption, follow
`recovery inspect/apply`; a confirmed unindexed definition requires a new reviewed
`project workflow recover --file <exact-json>` preview and explicit apply.

Definitions and selection are available in this slice; configurable Execution
binding, planning and dispatch remain separate. See the
[implementation boundary and verification map](specifications/007-configurable-workflows/issue-273-implementation.md).

## Hierarchical CLI help

```bash
axiom --help
axiom -h
axiom help
axiom project --help
axiom project show -h
axiom runtime profile validate --help
axiom --json workflow run --help
```

Every public command group and leaf supports `--help` and `-h`. Root help lists
public top-level surfaces; group help lists direct children; leaf help shows
usage, required, conditional and optional inputs, repeatable flags and examples.
Examples containing placeholders require real values before execution.

Help is deterministic human text, including when combined with `--json` or
`--human`. It runs before application composition, Project/context resolution,
Provider/Runtime access and mutation. A help-looking flag value (for example
`--name --help`) remains data; tokens after `--` do not request help. Unknown
command paths and supplied invalid leaf syntax remain failures and point to the
closest valid help surface. Help does not require omitted mandatory inputs.
Normal parser failures keep their status/category and include relevant help in
structured guidance (`next`, or `nextAction` on historical event output).

The command tree in `internal/cli/commands.go` serves routing and help. Help
reads the same flag builders as execution and reuses presence requirements;
domain validation, authority and workflow logic stay in their existing layers.
Later command features extend this tree and their parser flag metadata together.
See [GitHub #133 / Linear AXM-10 Evidence](development/evidence-issue-133.md).

## Effective Project context

Project inspection, Work Item, Runtime preview and workflow operations resolve
an omitted Project as session override, then persistent local default. An
explicit selector always wins and never writes preferences. A stale winning
selector fails instead of inferring a Project from CWD or Git.

```bash
axiom --json project context default-set --selector my-project --authorize-local
axiom --json project context show
axiom --json --session codex-session-42 project context session-set --selector other-project --authorize-local
axiom --json --session codex-session-42 project context show
axiom --json --session codex-session-42 project show
axiom --json --session codex-session-42 project show --selector my-project
axiom --json --session codex-session-42 project context session-end --authorize-local
axiom --json project context default-clear --authorize-local
```

Use a different caller-provided ID for every new Runtime session. IDs contain
1–128 ASCII letters, digits, hyphens or underscores. Without `--session`, the
operation uses only the local default or explicit selector. `session-clear`
also resets an override. `session-end` clears Axiom's selection, without ending
the Runtime conversation. New sessions inherit the local default. Preferences
hold UUIDs in native machine/user-local state, outside portable configuration.

`context show` exposes `effective`, `source`, `default` and `session`; use
`--selector` to inspect an explicit effective choice. A workflow's Execution
captures its original Project UUID. After changing context, name that original
Project explicitly when inspecting/resuming the Execution. All existing
readiness, preview and mutation-authority requirements continue to apply.

See the [contract and plan](specifications/002-lingo-project-initialization/effective-project-context.md)
and [validation Evidence](specifications/002-lingo-project-initialization/evidence-issue-233.md).

Execute estes comandos na raiz do repositório.

## Configure machine-local Runtime Profiles

Inspect without starting vendor processes:

```bash
axiom runtime binding list
axiom runtime profile list
axiom skill inspect axiom-runtime
```

`axiom-runtime` manages local bindings and operator-defined Profiles. A binding needs an installed supported Runtime. `runtime binding create --runtime
codex` previews a binding without creating a Profile catalogue. Configure explicit
invocation defaults or per-agent run fields separately. To create an optional
reusable Profile with the binding, choose its model and effort explicitly:

```bash
axiom runtime binding create --runtime codex --profile operator-review \
  --model operator-model --effort-mode runtime-default --effort default \
  --capabilities read --complexities high
```

This prepares a preview. Apply the same candidate only with its returned opaque
`--expected-revision`, `--preview-digest` and `--authorize-local`. Changed inputs,
state, identities or dependencies require another preview. `runtime-default`
means no Axiom override; successful authoring saves intent and proves no vendor
model, effort support or authentication. Inspect read-back after apply.

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

Profile create/edit changes model intent, effort, capabilities and complexity.
Binding edit changes eligibility through `--enabled`; enable/disable use the
same preview protocol. Removing one of several Runtimes publishes a new
revision. Removing the final Runtime requires explicit `--cascade` and retires
the configuration through the existing versioned tombstone; nothing is
resurrected, and a new configuration is created explicitly afterwards. Reads
and recovery never delete vendor settings, authentication, receipts, Projects,
Workflows or historical Executions. Pending state uses scoped
`runtime profile recover` and its separate exact recovery confirmation;
uncertain publication requires inspection before any new attempt.

Local configuration format 4 stores Runtime/Profile intent without effort-source
declarations. Authoring writes only format 4. Historical formats 1–3 remain
readable for inspection and explicit retirement; current use refuses them with
`unsupported_configuration_format`. There is no implicit conversion or downgrade.

To reconfigure, preview `axiom runtime binding remove --runtime <id> --cascade`.
Historical retirement covers the entire configuration and binds its exact digest.
Apply only with the returned `--expected-revision`, `--preview-digest` and
`--authorize-local`, then explicitly create format 4 with `runtime binding create`.
Use `axiom compatibility backup` to preserve source records first.
`--effort-declaration` is retired and cannot read a file, invoke a Runtime or
write configuration.

Workflow migration is a separate `project workflow migrate` operation: map each
Agent using repeated `--agent-profile <stage>:<agent>:<profile>`, inspect the
legacy/Profile effort comparison and confirm the new immutable revision.
Selection remains a separate Project operation. `--from-default
--default-revision 2` derives the neutral schema-2 default; revision 1 remains
readable as definitions. Canonical runs lower supported schema-1/2 definitions
into the frozen per-attempt invocation. Profiles are optional defaults; explicit
agent values own the effective invocation. Historical execution bytes remain inert
and cannot be resumed or converted.

## Runtime operational journey

Inspect bindings and optional Profiles, choose Runtime/model/effort explicitly,
then use `workflow run --authorize-external`. Normal runs perform no mandatory
authentication, compatibility, capability or effort-source preflight. Vendor and
spawn failures are recorded as observed attempt failures. Optional Runtime
inspection commands grant no dispatch authority and prove no successful inference.

Inspect canonical status after dispatch. An acknowledgement, login or exit zero
alone does not establish a successful result. Unknown effects require observation
and explicit disposition before another authorized attempt. Never repair credentials,
expand permissions or substitute Runtime/model automatically after failure.

`runtime <codex|claude> capability show|invalidate|prepare|probe|probe-extended`
is retired. Old probe directories remain inert historical state. No registry or
probe receipt is required.

## Inspect skill arguments

```bash
axiom skill inspect axiom-project
axiom --json skill inspect axiom-work-item
axiom --json skill inspect axiom-work-item --operation create
axiom skill inspect axiom-workflow
axiom skill inspect axiom-runtime
```

Use exactly `axiom-project`, `axiom-work-item`, `axiom-workflow` or `axiom-runtime`, the four product skills
shown by `axiom help`.
Inspection describes the skill embedded in this binary without executing it,
reading Project/Provider/Runtime state, prompting, or granting authority. It also
works before `first-run`. It does not inspect repository maintainer skills.

JSON preserves canonical completion fields and adds `skill.name`,
`skill.operations[]` and `skill.commands[]`. Each operation has a `name`, its
`commands`, and one or more `modes`: the distinct authority paths of that
operation (for example Project `configure` `create` versus `edit`, Project
`integration` `list` versus `disable`, or Work Item `run` `transition`, `fact`
and `reconcile`). A mode states its
`commands`, optional explicit `selector`, `effect` (`read-only`,
`preview-only`, `local-mutation` or `external-mutation`), `authority`,
`authorityInputs`, `rejectedInputs`, `semanticResolution` (`allowed` for
read-only modes, otherwise `unambiguous-only`) and an `example`. The metadata
describes, and never grants, authority. Each command includes its CLI spelling and `arguments[]`:
`name`, `required`, optional `requiredWhen`, `description`, `acceptedForms`,
`repeatable`. `required: true` means required for a noninteractive CLI request;
`requiredWhen` states a conditional requirement; otherwise the input is optional
at that boundary. Domain validation may require additional content before a
mutation, as stated in descriptions. Fully guided skill invocation still accepts
missing inputs and asks for them. No workflow argument belongs on `skill inspect`.

`--operation <name>` (or `--operation=<name>`) restricts the same payload to one
operation name listed by the unrestricted inspection, such as Work Item
`create` or Project `list`: `skill.operations[]` holds only that operation and
`skill.commands[]` only its commands. It is a view of the same registrations,
never a second registry; an unknown operation is a validation failure. A
Runtime uses it to load only the contract it is about to execute.

A refused Work Item or workflow selector command sets the canonical `details`
field to a stable handle: `<code>` or `<code>:--<flag>`, for example
`unknown_input:--title`, `duplicate_input:--project`,
`missing_required_input:--provider-repository` or `conflicting_input:--work-item`.
A refused Work Item application result sets `details` to its category, such as
`draft_incomplete`, `invalid_draft_input`, `repository_not_configured`,
`provider_unauthenticated`, `provider_unavailable` or
`provider_invalid_response`. Successful results keep `details` empty. The
handle explains a refusal; it never relaxes validation or grants authority.

Both discovery and parsing use the same flag registrations and presence rules.
Unknown/duplicate/conflicting invocation inputs retain their validation errors;
inspection neither validates live domain state nor authorizes any effect.
See the [contract and evidence](specifications/004-mvp-v1-baseline/issue-131-skill-arguments.md).

## Inspect installed operational documentation

```sh
axiom --json docs inspect
axiom --human docs inspect
```

`docs inspect` is read-only. It verifies, against the descriptor compiled into the
running binary, the documentation generation and binding published next to the
installed binary (`.axiom-docs/axiom/`), and returns the Markdown and man paths
only when the state is `verified`. Every other state (`absent`, `invalid`,
`unknown`) is a refusal with a `reason`, exit 1 and no paths. Presentation flags
go before the command words. It never repairs, downloads, starts a man reader or
a Runtime, and reports `loaded` as `unknown`: `skills.observed` lists on-disk
skill facts per Runtime, not session state. A release archive carries these
documents only in archive generation 2 (`scripts/build-release-archives.sh
--format 2`); generation 1 installations report no documentation.

`scripts/test-docs-offline.sh` builds a development generation 2 archive,
installs it into a throwaway prefix and checks verification, corpus integrity,
closed links and refusal of tampered, copied or modified installations.

## Lingo Project lifecycle

O lifecycle portátil básico é local. Comandos separados de Runtime/Work Item
podem acessar Codex user-global e GitHub quando explicitamente invocados. Use um
root temporário/isolado enquanto avalia; `LINGO_PROJECTS_ROOT` deve ser absoluto.
Sem override, Lingo usa `~/.axiom/projects` para configuração portátil. Estado
local usa `$XDG_STATE_HOME/lingo` no Linux quando absoluto, caso contrário
`$HOME/.local/state/lingo`; no macOS, usa
`$HOME/Library/Application Support/Lingo`. `LINGO_STATE_ROOT` substitui esse
destino para desenvolvimento/testes e deve ser absoluto, não vazio e separado
do root portátil.

```bash
export LINGO_PROJECTS_ROOT="$(mktemp -d)"
export LINGO_STATE_ROOT="$(mktemp -d)"

go run ./cmd/lingo project init --slug sample --name "Sample"
go run ./cmd/lingo project validate --slug sample
go run ./cmd/lingo project reopen --slug sample
go run ./cmd/lingo project install --source "$LINGO_PROJECTS_ROOT/sample"
go run ./cmd/lingo project update --slug sample --name "Sample renamed"
```

`project update` muda apenas Projects ainda não instalados nesta máquina. Para um
Project instalado, ele recusa sem escrita (`explicit_edit_required`) porque deixaria o
registro de instalação defasado; use `project configure --project <slug> --name`.

Cada operação escreve resumo humano por padrão; prefixe o comando com `--json`
para evento estruturado. `version`, `first-run`, `project configure`,
`project validate`, `project list` e `project show` usam o contrato canônico de completion;
demais comandos POC preservam temporariamente o evento histórico. Exit codes:
`0` sucesso, `1` erro/conflito, `2`
interrupção/cancelamento. `init` é create/no-op/conflito: não
renomeia nem atualiza um Project existente; use `update` para alterar o nome.

`install` grava o record local estrito em `<state-root>/projects/<project-id>/installation.json`;
não altera o manifest de origem. Um manifest que declara Repositories exige um
`--repository <key>=<absolute-path>` exato para cada key declarada. O store portátil básico aceita somente Project
mínimo de um único `axiom.yaml` e recusa symlinks, artefatos extras e roots
relativos. Bindings, Runtime Codex, GitHub Work Items e workflow usam
stores/adapters separados; rename e documentos opcionais completos continuam
fora do POC.

Execute o dogfooding reproduzível deste incremento em roots temporários:

```bash
./scripts/dogfood-poc.sh
```

O script exige Go 1.26, Bash, `find`, `wc`, `tr`, `grep`, `sed`, `awk` e
`shasum`. Ele instala Lingo em um PATH isolado, inicia fora do Project, instala e
verifica as seis skills, configura/resolve Project, instala um Project com
policy Runtime/Profile autorada, prova bloqueios sem fallback para Codex, revisa o
preview Runtime observado pelo próprio Lingo (inclusive drift do executável) e só
então inicia com `--runtime-preview`, usa um Provider GitHub fake
limitado, executa todos os gates, cobre interrupção/retomada e completa o Work
Item somente com authority explícita. A saída final é Evidence JSON versionada
com hashes SHA-256. Instalação usa publicação sem substituição; resíduos de
tentativas interrompidas retornam `recovery_required` e exigem inspeção humana.
No macOS, builds com e sem cgo inspecionam ACLs no objeto aberto. O caminho sem
cgo usa `fgetattrlist` e confirma suporte do volume a extended security antes de
aceitar ausência de ACL; resposta incompleta ou estado indeterminado falha
fechado.
Consulte o [procedimento manual de recovery](specifications/002-lingo-project-initialization/recovery-poc.md)
antes de mover qualquer artefato. A matriz macOS/Linux executa os mesmos checks
em [POC verification](../.github/workflows/poc-verification.yml).

## T01–T04 validation

Go 1.26 instalado. T01/T02/T04 usam biblioteca padrão; T03 usa o parser fixado
em `go.mod`/`go.sum`. O store POC usa `golang.org/x/sys/unix@v0.44.0` em
Linux/macOS. Em cache novo, preparação explícita com rede:

```bash
GOTOOLCHAIN=local go mod download go.yaml.in/yaml/v3@v3.0.5
GOTOOLCHAIN=local go mod download golang.org/x/sys@v0.44.0
```

Depois, os checks abaixo rodam offline. Build valida pacotes e o executável Lingo POC.

```bash
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
go test -cover ./...
go test -race -shuffle=on -count=10 ./...
go vet ./...
go build ./...
go mod verify
go test -fuzz=FuzzRecordRoundTrip -fuzztime=20s -parallel=2 ./internal/local
go test -fuzz=FuzzDecodeSafeRoundTrip -fuzztime=30s -parallel=2 ./internal/manifest
go test -run '^$' -bench=BenchmarkHostileBounds -benchtime=1x -benchmem ./internal/manifest
go run ./scripts/check-architecture.go domain
bash scripts/test-check-project-domain.sh
go run ./scripts/check-architecture.go application
bash scripts/test-check-projectapp.sh
```

`scripts/check-architecture.go` é ferramenta de verificação do repositório, não
capacidade do produto: um único motor AST aplica a política independente do
perfil `domain` (`internal/project`) ou `application` (`internal/projectapp`);
perfil ausente, desconhecido ou extra é recusado com mensagem de uso e código 2
(`go run` relata `exit status 2` e sai com 1). Evidence
histórica cita os checkers anteriores
[`check-project-domain.go`](https://github.com/rgomids/axiom/blob/2e23c9c4655a55f2e6b9e7e94e266173afea3b1c/scripts/check-project-domain.go) e
[`check-projectapp.go`](https://github.com/rgomids/axiom/blob/2e23c9c4655a55f2e6b9e7e94e266173afea3b1c/scripts/check-projectapp.go).

O checker AST inspeciona imports/símbolos puros e helpers dos testes fora da suíte
de domínio. O runner Go e ferramentas de verificação fazem I/O de compilação e
relatório; comportamento de domínio/testes não faz I/O de aplicação.
[Evidence e limites de T01](specifications/002-lingo-project-initialization/evidence-t01.md) e
[T02](specifications/002-lingo-project-initialization/evidence-t02.md). Checker de aplicação
bloqueia imports externos à fronteira e I/O ambiente, inclusive nos testes.
Spies/barreiras em memória provam contratos; não provam protocolo de filesystem.
[T03 Evidence](specifications/002-lingo-project-initialization/evidence-t03.md) detalha
parser, schema, limites, security policy v1 e cobertura. Teste AST do codec verifica
imports de produção; fixtures são sintéticas.
[T04 Evidence](specifications/002-lingo-project-initialization/evidence-t04.md) registra
JSON estrito, metadados locais, ausência versus corrupção e testes de fronteira.
`go test ./internal/local` inclui matrizes, round-trips, inspeção estática do DTO/imports
e integração com fake do port T02. Além do lifecycle portátil, este POC inclui
`axiom project install`, que persiste `installation.json` em
`<state-root>/projects/<project-id>/installation.json`; `reopen` reconhece esse
estado local quando disponível. Configuração portátil e estado local permanecem
separados. Credenciais continuam externas no `gh`; recuperação automatizada e a
matriz completa de falhas de filesystem continuam fora da evidência concluída.
#19/#20 seguem abertos e #21 está pronta para aceite humano.

## Harness validation

Valide o bootstrap completo:

```bash
./scripts/validate-repository.sh .
```

Valide somente a estrutura do harness:

```bash
./scripts/validate-agent-package.sh .
```

Teste o comportamento do validador de pacotes, inclusive rejeição de symlinks e arquivos sensíveis:

```bash
./scripts/test-validate-agent-package.sh
```

## Automation governance

A [política de automação do repositório](../.agents/policies/repository-automation.md)
define automação durável versus helpers efêmeros. Todo script, programa ou
workflow governado precisa de uma entrada em
[`scripts/automation-registry.json`](../scripts/automation-registry.json); o
`validate-repository.sh` executa as duas verificações abaixo.

Valide o registry contra a árvore atual (arquivos versionados e não ignorados):

```bash
python3 scripts/check-automation-registry.py .
```

Teste o comportamento do validador com fixtures offline (registry válido,
helper não registrado, path ausente, owner/purpose ausentes, lifecycle
inválido, registro duplicado, caller pendente, pin de compatibilidade inválido
e declaração durable/ephemeral inválida):

```bash
python3 scripts/test-check-automation-registry.py
```

A verificação é estrutural e offline: não executa automação registrada, não
consulta a rede e não decide reutilização, ownership correto, linguagem ou
frequência real de execução, que pertencem à revisão de engenharia.

## Security validation

Teste o checker local:

```bash
./scripts/test-check-sensitive-files.sh
```

Escaneie arquivos versionados e não ignorados no worktree:

```bash
./scripts/check-sensitive-files.sh .
```

Escaneie exatamente paths e conteúdo staged:

```bash
./scripts/check-sensitive-files.sh --staged .
```

Quando `gitleaks` estiver disponível:

```bash
gitleaks detect --source . --no-git
```

## Git review

```bash
git status --short
git diff
git diff --cached
git diff --check
git remote -v
git branch --show-current
```

Não use Makefile como interface principal. Este repositório não possui Makefile.

## Install the E2E POC locally

Install Axiom from the current checkout. The default user binary destination is
`$HOME/.local/bin`. The installer reports `pathConfigured: false` and prints a
shell-safe export when that directory is absent from `PATH`; it does not mutate
shell profiles:

```bash
./scripts/install-axiom.sh
export PATH="$HOME/.local/bin:$PATH"
command -v axiom
axiom version
axiom --json version
```

For an isolated or custom user destination:

```bash
AXIOM_BIN_DIR=/absolute/path/to/bin \
AXIOM_INSTALL_STATE_ROOT=/absolute/path/to/state \
./scripts/install-axiom.sh
export PATH="/absolute/path/to/bin:$PATH"
```

The installed public executable is `axiom`; its source-install receipt is
`axiom.receipt` in the state root. No shell alias is involved. A `lingo`
executable left by an earlier source install is neither replaced nor removed;
delete it yourself when no longer needed. The installer is safe to rerun. It
replaces only a prior binary whose exact checksum matches its protected receipt;
an unrelated or modified destination is refused. `axiom version` reports `development` for source builds plus short revision
or `unavailable` and source state `clean`, `dirty`, or `unknown`; it never invents
a release version. The receipt retains the full source commit and dirty flag. Test
missing-PATH guidance, dirty metadata,
installation and unrelated-CWD invocation with:

```bash
./scripts/test-install-axiom.sh
```

## First run and Runtime integrations

`axiom first-run` is an idempotent bootstrap (S9/T40). It discovers each
supported Runtime by resolving its executable from the current `PATH`
(`codex`, `claude`); a configuration directory alone does not count, and the
executable is never run. A Runtime whose executable is not on this process's
`PATH` is reported absent (with `configurationWithoutExecutable` when its
configuration directory exists); put it on `PATH` and rerun, or use the
per-Runtime command below. For every Runtime found it installs or upgrades the
three canonical Axiom-owned user-global skills, then reports every supported Runtime:

```bash
axiom first-run
axiom --json first-run
```

| Runtime | Skill root | Receipt |
|---|---|---|
| Codex | `$HOME/.agents/skills` (`AXIOM_CODEX_SKILLS_ROOT` for isolated validation) | `.axiom-skill-set.receipt` (skill set and manifest digest) |
| Claude | `<CLAUDE_CONFIG_DIR or ~/.claude>/skills/<skill>/SKILL.md` | `.axiom-skill-set.receipt` with `runtime=claude`, the skill root and each skill digest |

Each detected Runtime converges independently: absent skills are installed,
current content is a no-op, and only content that is a previous Axiom-owned
revision for that Runtime is upgraded. Every skill set published through the
shared Runtime integration (from this release on) is a known revision for every
Runtime, so after `axiom upgrade` changes the skill text, the next `first-run`
upgrades Claude as well as Codex; older Codex-only revisions are never adopted
in a Claude root. A receipt is recognized when it is byte-for-byte the receipt
this binary or an earlier Axiom revision wrote for that root, serialized with
that revision's own skill set (so receipts from releases that predate a skill
stay recognized). Unknown, foreign, or modified content is preserved and fails
that Runtime (`<runtime>_skill_conflict`); the receipt never authorizes
overwriting changed content, and no skill is replaced beside a receipt that is
not this Runtime's current or earlier Axiom receipt for that root.

Each skill is reported as `missing`, `equivalent` (this binary's text),
`owned_older` (an earlier Axiom revision; replaced), `modified` (unknown
content under a name the root's recognized Axiom receipt says Axiom installed:
an edited Axiom skill), `foreign` (unknown content without that attestation),
or `unsafe` (not a private directory holding exactly one private regular
`SKILL.md`). The receipt is reported as `absent`, `current`, `legacy`,
`unrecognized` or `unsafe`. Every artifact that blocks convergence is listed,
relative to the Runtime skill root, never overwritten:

```text
runtime: claude present=true state=failed reason=claude_skill_conflict
  receipt: state=legacy
  conflict: artifact=axiom-work-item-run/SKILL.md state=modified sha256=<digest> preserved=true
```

JSON carries the same facts as `receipt` and `conflicts[]` (`artifact`,
`state`, `sha256`). Keep the artifact, or move it aside if it is not yours,
then run `axiom first-run` again. The skill root belongs to the Runtime, which commonly creates
it `0755`: it must be a real directory (not a symlink) owned by you, not
writable by group or other, and without extended ACL, otherwise that Runtime
fails (`<runtime>_skill_root_unavailable`) without changes. So `0700`, `0750`
and `0755` are accepted and `0770`, `0775` and `0777` are refused. Every
ancestor directory up to `/` is checked the same way (owned by root or by
you, no group/other write unless the sticky bit protects existing entries),
since a mutable ancestor could otherwise let another principal replace the
root itself between its check and its use; ordinary system directories and a
`0755` home directory pass, and a `/tmp`-style sticky world-writable
directory passes because of the sticky bit. Everything Axiom creates under
the root stays `0700`/`0600`. A missing root is created `0700`. A relative `CLAUDE_CONFIG_DIR` fails Claude the same way.
Axiom only reads `CLAUDE_CONFIG_DIR` from the process environment, not from
Claude settings files. The install keeps its persistent `.axiom-skill-set.lock`
in each skill root.

| Detected | Result | Exit |
|---|---|---|
| none | `success`, no supported Runtime available, no effect | `0` |
| all converge | `success` | `0` |
| some converge, others fail | `partial`; converged results are kept | `1` |
| all fail | `failure` | `1` |

The JSON result carries `firstRun.detected`, `firstRun.failed` and one entry
per Runtime with `present`, `configurationWithoutExecutable` (a stale
configuration directory), `state` (`absent`, `configured`,
`already_configured`, `failed`), `reason`, and the skill states. first-run never
installs a Runtime, authenticates, reads or changes credentials, provisions
secrets or touches subscriptions, and does not infer a Project.

Per-Runtime commands install or inspect one integration regardless of
discovery:

```bash
axiom runtime codex install
axiom runtime codex status
axiom runtime codex auth
axiom runtime claude install
axiom runtime claude status
axiom runtime claude auth
```

`runtime <codex|claude> auth` is the read-only CLI login observation
([#272 contract](specifications/004-mvp-v1-baseline/issue-272-cli-subscription-auth-preflight.md)).
It runs vendor version/status commands with the shell's unchanged environment
and working directory. Environment variable names, including Claude settings
`env` entries, do not determine authentication compatibility. The CLI's reported
login method, executable identity and explicit non-environment configuration
checks determine the result. Only `subscription_observed` succeeds; API-key or
missing login, ambiguous status and observation failures remain failures.
Success is not proof of effective routing, billing, usability or Project readiness.
No login, refresh, inference or global environment change is performed. Stage
dispatch retains its existing environment isolation and repeats the same check
on each child's effective invocation.

Codex standalone skill names accept lowercase letters, digits and hyphens, so
the requested semantic `axiom:<skill>` names are invoked as:

```text
$axiom-project
$axiom-work-item
$axiom-workflow
$axiom-runtime
```

`$axiom-project`, `$axiom-work-item`, `$axiom-workflow` and `$axiom-runtime` are the product Runtime skills.
Explicit operations route directly to Lingo; natural-language intent selects
only supported operations. Ambiguity requires clarification; interpretation
never supplies authority. Inspect the current binary for operation/mode,
argument and effect contracts. Claude invokes `/axiom-project`,
`/axiom-work-item`, `/axiom-workflow` and `/axiom-runtime`, or selects them from their descriptions.

Releases through v0.10.0 distributed operation-specific skills (six before
#229, eight after it). The pre-MVP decision now removes these active entrypoints.
Installation verifies historical ownership before retiring them; modified,
foreign, linked or unrecognized content conflicts and is preserved. Receipts
and digests retained internally are migration proofs, not discoverable skills.
See [consolidation contract](specifications/004-mvp-v1-baseline/issue-230-canonical-consolidation.md).
For isolated validation:

```bash
AXIOM_CODEX_SKILLS_ROOT=/absolute/test/root axiom runtime codex install
CLAUDE_CONFIG_DIR=/absolute/test/claude axiom runtime claude install
./scripts/test-codex-skills.sh
```

## Preview Project Runtime/Profile selection

Predictive Runtime/Profile selection preview is retired. Configure optional
Profiles with `runtime binding` / `runtime profile`, or provide explicit per-agent
Runtime/model/effort values to `workflow run`. A Project Runtime allowlist still
limits actual selected targets; capability/complexity rankings, compatibility
predictions and authentication probes are not run prerequisites.

## Validate the Runtime Profile store

Inspect the isolated S8 Runtime Profile configuration without creating, writing,
or repairing anything:

```bash
axiom runtime profile validate
axiom --json runtime profile validate
```

The command takes no flags; any extra argument is rejected before the store is
read. It loads `local.RuntimeProfileStore` from the absolute `LINGO_STATE_ROOT`
and calls `runtimeprofile.Validate` against the published configuration. It is
strictly read-only: it never creates or repairs state, and it never invokes or
authenticates a Runtime/model adapter. It does not perform capability discovery;
it validates only the shape of the persisted configuration.

A published, well-formed configuration returns the canonical `success`
completion. A missing store or missing published configuration, and a present
but structurally or semantically invalid configuration, both fail closed and
return the canonical `validation_failure` completion with a sanitized category;
neither case creates state or exposes internal error detail. Human output
prints the concise status by default; prefix the command with `--json` for the
structured event, as with other `axiom` commands.

## Build and install exact-version S2 archives

A release build requires a clean checkout, an exact semantic version, and an
absolute output directory. It emits three checksummed archives plus
`SHA256SUMS`:

```bash
./scripts/build-release-archives.sh \
  --version 0.1.0 \
  --output /absolute/release
```

Archive generation 1 remains the effective default in the builder and release
workflows. Explicit `--format 2` adds the installed operational corpus and uses
`macos-arm64`; generation 1 retains `macos-27-arm64`. Both generations use
`linux-amd64` and `linux-arm64`. `--only` accepts a single normalized row
(`macos-arm64`, `linux-amd64` or `linux-arm64`) for local development builds;
a publishable release set always contains all three rows.

Each archive holds one bundle directory with the canonical public executable
`axiom`, `LICENSE`, the release installer `install.sh`, `release-metadata.txt`,
`skills-manifest.txt`, the four canonical Runtime skills, and a complete `MANIFEST.sha256`.
Upgrading a recognized six- or eight-skill installation retires verified
Axiom-owned legacy entries and converges to the four canonical skills. Foreign,
modified, linked or unrecognized content is preserved and reported as conflict.
The installer publishes `<bin-dir>/axiom`. A receipt or binary from a pre-`axiom`
archive (which shipped `lingo`) is not recognized as owned and is preserved;
no migration from such an installation is performed.

Supported hosts: macOS arm64 and Linux amd64/arm64.
Installer eligibility is the OS family and architecture, plus real tool
prerequisites; the numeric OS version is not an installation filter. The
`macos-27-arm64` name is legacy release-row naming, not a version requirement,
and the Linux archives (`linux-amd64`, `linux-arm64`) are static builds for any
Linux distribution. Other operating systems and architectures fail closed.
Eligibility is not a support commitment: maintenance covers vendor-maintained
OS versions, and exact validated environments are recorded as Evidence
([ADR-0015](decisions/0015-installer-host-eligibility-os-family-architecture.md),
[Specification 006](specifications/006-installer-host-eligibility/spec.md)). Install the
archive matching the current host into explicit user-owned destinations:

```bash
./scripts/install-release.sh \
  --archive /absolute/release/axiom-0.1.0-macos-27-arm64.tar.gz \
  --checksums /absolute/release/SHA256SUMS \
  --bin-dir /absolute/user-owned/bin \
  --receipt-dir /absolute/user-owned/state
```

POSIX installation requires a private executable temporary workspace. The
release facade respects `TMPDIR` (or the system default when unset), verifies
the archive and bundle before executing its candidate, and checks execution
before creating installation roots or publishing persistent installation state.
An unsafe workspace or one that does not permit execution fails explicitly;
select a safe executable `TMPDIR` and retry. The facade owns bootstrap integrity
and transport; `internal/install` owns the POSIX installation lifecycle.

The install is checksum-first. An existing receipt root is Axiom-owned state and
must be owned by the current user, mode `0700`, and free of extended ACLs. An
existing binary root may be a shared user directory such as `~/.local/bin`: it
must be a real directory (not a symlink) owned by the current user that group
and other cannot write and that has no extended ACL, so `0700`, `0750` and
`0755` are accepted and `0702`, `0720`, `0770`, `0775` and `0777` are refused.
Every ancestor of the binary and receipt roots, up to `/`, is checked the
same way (owned by root or by you, no group/other write unless the sticky
bit protects existing entries), since a mutable ancestor could otherwise let
another principal replace the root itself; ordinary system directories and a
`0755` home directory pass, and a `/tmp`-style sticky world-writable
directory passes because of the sticky bit. A missing root is created `0700`,
and the published `axiom` stays `0700` and the receipt `0600`. Unsafe roots
are preserved, not repaired. The closed receipt includes an RFC 3339 UTC
`installedAt` value created for the successful installation generation and
preserved on equivalent reinstall. Exact owned reinstall is a no-op. Platform,
ownership, permission, ACL, link, type, schema, and content conflicts fail closed.
The installer never edits shell profiles or `PATH`. Refusals that need no lock
(symlinked or unsafe existing roots, a binary without receipt) are decided
before any directory is created, and a refused concurrent installer never
releases the lock held by another operation. For an existing owned
installation with a different binary, it runs the verified candidate's own
[owned upgrade](#compatibility-cleanup-recovery-and-upgrade): a read-only
preview, then apply under that exact preview digest, printing
`install_status=upgraded`. A newer version upgrades; an older one is refused
(`downgrade_refused`); a divergent same version is refused. An interrupted
owned upgrade resumes only with the same archive; any other interrupted
operation stays `recovery_required`.

Validate archive structure, clean install, no-op, conflicts, interruption, and
recovery markers with:

```bash
./scripts/test-release-archives.sh
```

## Install a published release (S9/T39)

`scripts/install.sh` is the remote bootstrap. It needs no checkout, Go or build;
it requires `sh`, `bash`, `curl`, `tar`, `awk`, `grep`, `mktemp` and `sha256sum`
or `shasum`. It requires a published release, so it fails until one exists:

```bash
curl -fsSL https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.sh | sh
curl -fsSL https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.sh | sh -s -- --channel stable
curl -fsSL https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.sh | sh -s -- --version v0.1.0-rc.2
```

Selection:

- no selector or `--channel stable`: the latest published stable release, read
  only from the `Location` of GitHub's `/releases/latest` redirect. It never
  falls back to a release candidate; with no stable release it fails and names
  `--version`.
- `--version vX.Y.Z` or `--version vX.Y.Z-rc.N`: exactly that published tag.
- release candidates: only by exact version, `--version vX.Y.Z-rc.N`. There is
  no release-candidate channel; `--channel rc`, like any other unsupported
  selector, is an input error with no effect.
- `--channel` and `--version` are mutually exclusive and fail before any effect.

The bootstrap detects the eligible OS/architecture before any download, then
fetches `SHA256SUMS` over HTTPS. Its complete, closed three-archive inventory
selects generation 1 (`macos-27-arm64`) or 2 (`macos-arm64`) for all hosts;
Linux retains `linux-amd64`/`linux-arm64`. Mixed generations, duplicate names,
unknown assets, missing rows and malformed checksums refuse before any archive
request. It requests exactly the selected row with no fallback, and verifies its digest
before reading the archive. It then requires the bundle's `install.sh` and
`release-metadata.txt` to match the bundle manifest and the metadata to be the
resolved version and row of a clean release build, and runs that release
installer. All installation effects, ownership checks, no-op reinstall,
protected owned upgrade, downgrade refusal and recovery belong to that
installer (see above). Defaults are `--bin-dir $HOME/.local/bin` and
`--receipt-dir ${XDG_STATE_HOME:-$HOME/.local/state}/axiom/install`. An existing
binary directory must be yours, not a symlink, not writable by group or other,
and without extended ACLs (a usual `0755` `~/.local/bin` is accepted); an
existing receipt directory must be yours, mode `0700` and without extended
ACLs. Otherwise pass another absolute canonical directory. It prints the resolved
identity (`install_tag`, `install_asset`, `install_asset_sha256`, `install_row`,
`install_revision`, receipt path) on stdout for automation. Human presentation
goes to stderr with the Axiom mascot ASCII logo, progress/status headings, and
actionable diagnostics. On success, stderr ends with a short human summary that
shows the real status (`Installed`, `Upgraded`, or `Unchanged`), installed
version, binary location, documentation URL, and the next command when useful.
If the binary directory is not on `PATH`, that summary includes the export to
run for the current shell. ANSI color is used only for an attached non-dumb
terminal and is disabled by `NO_COLOR`; captured or non-color terminals remain
plain text. It never uses `sudo`, edits shell profiles or `PATH`, installs
Runtimes, or touches credentials. Release acceptance (T24) pins
`--version vX.Y.Z-rc.N`.

Test the matrix with a fake `curl` and local release fixtures (no live GitHub):

```bash
./scripts/test-install-bootstrap.sh
```

Selector, host and input refusals run on any host. Install, reinstall, upgrade,
downgrade, foreign/modified/unsafe state, concurrency, interruption and network
cases need a supported row (any macOS arm64 or Linux x86_64/aarch64 qualifies)
and exit `78` elsewhere. OS version and Linux distribution are not installer
filters.

## Prepare a release artifact set (S9/T38)

Public release tags are `vMAJOR.MINOR.PATCH` (stable) or
`vMAJOR.MINOR.PATCH-rc.N` (release candidate). Release metadata records the
semantic version without the leading `v`:

```bash
./scripts/release-tag-version.sh v0.1.0-rc.2
# tag=v0.1.0-rc.2
# version=0.1.0-rc.2
# channel=rc
```

The manually dispatched `Release artifacts` workflow
(`.github/workflows/release-artifacts.yml`) is the PREPARE phase of the
[Release flow](#release-flow). It takes `tag` and `revision` inputs, runs from
`main`, checks out that exact revision, requires it to be clean and to pass
`release-preflight.sh`, builds the three supported rows with
`build-release-archives.sh`, verifies the complete set, renders the release
notes, and retains `artifacts/`, `release-evidence.txt`, `release-notes.md`,
`preflight.txt` and `prepare-metadata.txt` as the workflow artifact
`axiom-release-<tag>`.
It has a read-only token and never creates tags, GitHub Releases, prereleases,
or `latest`, and never writes to the repository. Cross-built artifacts are not
native target acceptance; publication (see [Release flow](#release-flow)) and
native acceptance stay with T23/T24.

Verify an artifact set locally from a clean checkout at its exact revision:

```bash
./scripts/verify-release-artifacts.sh \
  --dir /absolute/release \
  --version 0.1.0-rc.2 \
  --revision "$(git rev-parse HEAD)"
```

For generation 2, pass `--format 2` to both builder and verifier. The verifier
keeps generation 1 as its default and rejects a set for the other generation.
The bundled installer supports `--verify-only` with the usual absolute archive,
checksum and destination arguments: it validates without executing the candidate,
checking host eligibility or creating destinations. Release verification uses
this mode for all v2 rows before source/provenance checks. The prepared-artifact
smoke also accepts explicit `--format 2` and `--row macos-arm64`; its default
remains generation 1. Smoke is development evidence, not native documentation
or agent qualification.

The verifier requires exactly `SHA256SUMS` plus one archive per supported row,
correct checksums, the closed bundle entry set with a `0700` `axiom` executable,
a complete `MANIFEST.sha256`, exact release metadata, `LICENSE`, `install.sh`
and skills identical to the source, the executable format of each row, and Go
build information naming the exact revision with `vcs.modified=false`. The
executable for the host's own row must report the exact provenance. It prints
closed `key=value` Evidence ending in `publication=none` and `result=pass`.

Test the tag contract, clean/dirty source, the full matrix, rerun equivalence,
fail-closed incomplete or foreign sets, and the workflow's no-publication
boundary (builds run in a clean clone of the committed `HEAD`):

```bash
./scripts/test-release-pipeline.sh
```

A rerun from the same revision yields identical bundle contents and
executables; archive bytes (tar timestamps) may differ, so the published
`SHA256SUMS` is the one produced by the run that is published.

## Release flow

The process, versioning and authority rules are in
[CONTRIBUTING.md](../CONTRIBUTING.md#release-flow). Merges integrate code;
`release.sh start` (`$axiom-release`) starts releases
([ADR-0011](decisions/0011-command-driven-release-start.md)). Workflows:

| Workflow | Trigger | Effect |
|---|---|---|
| `.github/workflows/ci.yml` | every PR and dispatch; not on push to `main` | required checks only; read-only token; `release.sh` resolves a `main` commit's CI from the head of its merged PR when both trees are equal |
| `.github/workflows/release-please.yml` | manual dispatch from `main` by `release.sh start` with `planned_version` and `main` (never a push) | requires `main` to still be the planned SHA at dispatch, re-runs `release-plan.sh` on it, then opens/updates the Release PR (`CHANGELOG.md`, `.release-please-manifest.json`); dispatches CI and `delivery-metadata` on the Release PR branch only when the PR records the planned version and its base is the validated SHA or an ancestor; never tags or releases |
| `.github/workflows/release-artifacts.yml` | manual dispatch from `main` with `tag` and `revision` | PREPARE: preflight, build, verify, notes; retains the exact set as workflow artifact `axiom-release-<tag>`; read-only token; never publishes |
| `.github/workflows/publish-release.yml` | manual dispatch from `main` with `tag`, `revision`, `prepared_run`, `preview_digest` | PUBLISH: re-verifies that prepared artifact, requires its envelope digest to equal `preview_digest`, then draft, upload, read-back, publish and, for a stable release, the envelope's Issue effects; `publish` job gated by the `release` environment; never rebuilds |
| `.github/workflows/delivery-metadata.yml` | PR opened, edited, reopened or synchronized (not Release PRs); dispatch by `release-please.yml` on the Release PR branch | validates the Conventional Commit title and `Related-Issues`/`Completes-Issues`, and refuses closing keywords; on dispatch passes only for the bot-authored Release PR head (`delivery-github.sh release-pr-head`); no token write, no secret |
| `.github/workflows/delivery-sync.yml` | push to `main` | merge-time delivery projection: completing PRs move Issues to `Awaiting Release` (Issues stay open; one closed by GitHub at exactly that merge is reopened); a release commit records `Target Release`; with projection enabled, released Issues are reconciled to `Released` |

Recovery is an exception for a release commit that already exists with
inconsistent history (v0.3.0); new releases are validated before their
Release PR. For an unpublished stable release whose immutable source lacks delivery
metadata, opt-in `--corrections-revision <full-main-sha>` and
`--corrections-digest <committed-file-sha256>` select the additional pinned
input for `status`, `prepare` and `publish`. The same pins are required across
phases; `verify --download` reads and validates published provenance. See
[recovery contract and commands](development/release-recovery.md). No source
revision, archive inventory or publication authority changes.

Discover the state and the next step (read-only apart from `git fetch` of
`main`):

```bash
./scripts/release.sh status
./scripts/release.sh status --tag v0.1.0-rc.1
./scripts/release.sh status --tag v0.1.0
```

`status` prints closed `key=value` facts (repository, branch, HEAD, worktree,
`main`, required CI from `.github/rulesets/main.json`, `release` environment,
open and merged-unpublished Release PRs, the release plan), `state` and
`next_action`:

| `state` | `next_action` | Meaning |
|---|---|---|
| `no_release_in_progress` | `none`, `start_release`, `qualify_main`, `await_run` or `blocked` | nothing releasable; the plan passed and the native suite succeeded at `main`; the native suite never ran at `main` (`release.sh qualify`), is running, or failed; or the plan or the previous-release check refused (`reason`) |
| `release_pr_starting` | `await_run` | a Release Please run is in flight |
| `release_pr_open` | `review_release_pr`, `refresh_release_pr`, `qualify_main`, `await_run` or `blocked` | human review and merge (after a branch update when only validated hidden commits are behind); a releasable commit or another planned version since it was built; or `main` no longer validates, is red, or the planned tag has a release |
| `release_pr_merged` / `release_candidate` | `prepare` or `blocked` | build and verify the exact set of the release commit (or RC revision) |
| `preparing` / `publishing` | `await_run` | a preparation or publication run is in flight (a publication waits for the `release` environment approval) |
| `awaiting_publication_authority` | `authorize_publication` | the envelope of the newest verified prepared run, with `preview_digest` |
| `published` | `verify_published` | read back and verify |

A stable tag resolves to its release commit (the first-parent `main` commit
whose manifest introduced the version); a release candidate defaults to
`origin/main`. Before that commit exists, `status --tag vX.Y.Z` considers only
the open Release PR titled exactly `chore(main): release X.Y.Z` (never
`X.Y.Z0`) and reaches the same decision as `status` for it. Preparation is offered only when required CI on the revision is
green, the `release` environment requires a reviewer and the release range
resolves its Issue set. `status` tries the three newest unexpired prepared
artifacts of the tag from `main` and uses the first that verifies at the
revision, so a later run resumes at the envelope without a run id; rejected
runs are listed as `prepared_run_rejected.<id>=<reason>` and preparation is
offered again.

Start a release (preflight, then Release Please; no tag, release or artifact):

```bash
./scripts/release-plan.sh --main-ref origin/main
./scripts/release.sh start
```

`start_release` and `refresh_release_pr` also require the native suite
(`nightly.yml`: Linux/macOS race suites, release contracts, upgrade journeys)
to have succeeded at the exact `main` SHA ([ADR-0036](decisions/0036-native-qualification-gates-release-start.md)).
`status` reports `main_native` (`success`, `failure`, `pending`, `missing`)
and `main_native_run`; a missing run reports `next_action=qualify_main`:

```bash
./scripts/release.sh qualify
```

`qualify` dispatches `nightly.yml` on `main`, waits and prints `status`
again; it publishes nothing. A failed run blocks the start until `main` is
fixed or a human re-runs it.

`release-plan.sh` is Git-only: for every first-parent commit since the last
release commit it requires a Conventional Commit subject and declared or
reviewed delivery metadata, requires the tag of the previous release at that
commit, refuses an existing or older planned tag, and prints
`planned_version`, `planned_tag`, the bump, one `commit=` line per commit,
`change.N` lines and the delivered Issues. Being Git-only, it does not see
GitHub Releases: `release.sh` reads the previous release with
`publish-release.sh --check` (and the ADR-0010 pins it records) and reports
`previous_release_state`; anything but `published` (a tag without its GitHub
Release, a draft, a release bound to another tag or commit) blocks
`start_release`, `refresh_release_pr` and `review_release_pr`, and makes a
nothing-to-release `status` report `blocked` instead of `none`. `start` requires `next_action` to be
`start_release` or `refresh_release_pr`, dispatches `release-please.yml` with
`planned_version` and the exact `main` SHA, waits, and stops at
`release_pr=<url>` with `next_action=review_release_pr`. It refuses when the
Release PR does not record the planned version. It never approves or merges.
`release-pr-checks.sh dispatch REPO VERSION MAIN` (in the workflow) also
requires the Release PR head to be one commit whose parent is `MAIN` or an
ancestor of it.

Prepare, then review the envelope:

```bash
./scripts/release.sh prepare
./scripts/release.sh prepare --tag v0.1.0-rc.1
./scripts/release.sh status --tag v0.1.0-rc.1 --revision <full-sha> --prepared-run <run_id>
```

`prepare` dispatches `release-artifacts.yml` (no publication), waits for it and
prints the result of `status --prepared-run`: the prepared artifact is
downloaded, re-verified with `verify-prepared-release.sh` in a clean clone at the
revision, and its publication envelope is printed as `preview.*` lines with
`preview_digest`, the SHA-256 of the envelope. Only after explicit human
authorization of that digest:

```bash
./scripts/release.sh publish --preview-digest <preview_digest> --authorize-publication
./scripts/release.sh publish --tag v0.1.0-rc.1 --preview-digest <preview_digest> --authorize-publication
gh run watch <run_id> --repo rgomids/axiom --exit-status
```

The digest binds the tag, revision and prepared run; `--tag`, `--revision`
and `--prepared-run` are optional and, when given, must equal the envelope's.

Without `--authorize-publication`, or when the envelope recomputed now differs
from `--preview-digest`, nothing is dispatched (`preview changed; review and
authorize again`). The publication workflow recomputes the envelope again from
the same prepared bytes and refuses before any effect on a mismatch. A maintainer
then approves the `release` environment in GitHub. Verify a published release
(default tag: the version `main` records). It requires the published
`SHA256SUMS` and notes to equal the prepared set (`prepared_match=pass`, or
`not_checked` once the workflow artifact expired); with `--download`, the
assets are re-verified with `verify-release-artifacts.sh` in a clean clone at
the tagged revision (the verified prepared run must be a successful
`release-artifacts.yml` dispatch from `main`):

```bash
./scripts/release.sh verify --download
./scripts/release.sh verify --tag v0.1.0-rc.1 --download
```

Building blocks, each read-only unless stated:

```bash
./scripts/release-preflight.sh --tag v0.1.0 --revision <full-sha> --main-ref origin/main
./scripts/release-notes.sh --tag v0.1.0 --revision <full-sha> --repo rgomids/axiom
./scripts/verify-prepared-release.sh --tag v0.1.0 --revision <full-sha> --prepared /abs/prepared \
  --repo rgomids/axiom --run <run_id>
./scripts/publish-release.sh --check --repo rgomids/axiom --tag v0.1.0 --revision <full-sha> --make-latest true
./scripts/publish-release.sh --envelope --repo rgomids/axiom --tag v0.1.0 --revision <full-sha> --make-latest true \
  --prepared-run <run_id> --dir /abs/prepared/artifacts --evidence /abs/prepared/release-evidence.txt \
  --notes /abs/prepared/release-notes.md
```

`release-preflight.sh` requires a full revision on the first-parent history of
`main` that already carries `.release-please-manifest.json`; for a stable tag,
the release commit of that version; for an RC, a manifest version not newer
than the RC, no existing stable of that version and an RC number greater than
every existing one; the version must be newer than the latest stable tag, and
an existing tag is accepted only at the same revision. It prints `make_latest`.
`verify-prepared-release.sh` runs in a clean checkout of the revision and
requires the prepared run to be a successful `release-artifacts.yml` dispatch
from `main`, the preflight to pass now, `verify-release-artifacts.sh` to accept
the artifacts and reproduce `release-evidence.txt` (host-only `version_smoke`
ignored), and `release-notes.md` to equal the notes of the revision.
`publish-release.sh --envelope` prints the envelope and `preview_digest`;
without `--check`/`--envelope` it publishes only with `--authorized-digest`
equal to that digest (only the `publish-release.yml` job runs it) and prints
each remote `effect=` line.

Delivery tracking ([CONTRIBUTING.md](../CONTRIBUTING.md#delivery-tracking)).
Each command is read-only unless stated:

```bash
./scripts/delivery-issues.sh parse --file message.txt                 # one commit message or PR body
./scripts/delivery-issues.sh check-pr --title title.txt --body body.txt
./scripts/delivery-issues.sh commits --from <full-sha> --to <full-sha>   # merge-time lines
./scripts/delivery-issues.sh commits --from <full-sha> --to <full-sha> --require-declared # post-merge validation before sync effects
./scripts/delivery-issues.sh release --tag v0.2.0 --revision <release-commit>
./scripts/delivery-github.sh state --repo rgomids/axiom --tag v0.2.0 --revision <release-commit>
./scripts/delivery-github.sh verify --repo rgomids/axiom --tag v0.2.0 --revision <release-commit>
```

`release` prints the Issue set a stable release delivers (`issues=`, one
`issue.N=` line with the completing PRs, `range_base=`, `undeclared_commits=`,
`legacy_boundary=v0.2.0`, `undeclared_policy=legacy_allowed|fail_closed`).
Above the boundary an undeclared commit other than the release's own Release
Please commit fails closed.
For a release candidate it prints `issues=not_applicable`. `state` prints the
delivery lines of the publication envelope. `delivery-github.sh sync` and
`release` change Issues and the Project. They run only in `delivery-sync.yml`
and inside an authorized `publish-release.yml` run. `sync` also reopens an
Issue GitHub closed at exactly the completing merge (premature-closure
fail-safe) and, with projection enabled, reconciles the Project of every stable
release since `v0.2.0` (`reconcile_repair=`, `reconcile_consistent=`,
`reconcile_skipped=` lines).
`./scripts/release.sh verify` also runs `delivery-github.sh verify`.

Test the delivery grammar, PR check and release Issue sets locally (no
GitHub):

```bash
./scripts/test-delivery.sh
```

Test the whole contract with a stateful fake `gh` and local Git fixtures (no
network or GitHub effects). This includes delivery: merge projection, notes,
envelope binding, closure after read-back, RC behavior, partial recovery,
Project failures before effects, the premature-closure fail-safe and Project
reconciliation after a disabled projection. preflight rules, notes, envelope completeness and
determinism, stale authority, publication of exactly the envelope's bytes,
reruns, partial drafts, conflicts, stable versus prerelease and `latest`, the
prepare/publish authority boundary of `release.sh`, the Release Please
configuration, and the workflows' triggers, permissions and pins:

```bash
./scripts/test-release-flow.sh
```

The same suite covers recovered release notes: published notes with Markdown
links and glob characters compare literally against both the prepared notes and
the notes regenerated from the pinned inputs, and changed notes refuse even
when they match a Bash glob pattern.

Published immutable recovery releases may opt in to `--repair-revision <full SHA>`
for `status`, `verify` and `publish`, preserving the original metadata pins and
prepared run. Envelope v4 binds the additional reviewed execution SHA and only
remaining delivery/label effects; new human authority remains required. The
option is forbidden on `prepare`. `start --repair-revision <SHA>` applies only
to read-only validation of the recovered previous release. An explicit repair
SHA that does not apply to a published recovery release refuses; it never falls
back to the normal flow. See the
[published repair guide](development/release-recovery.md#published-release-control-code-repair).

## CLI output and help

Workflow definition inspection in the shared Runtime skill derives a three-level
Workflow/Stage/Agent view from the same canonical `project workflow show` result.
An empirically qualified host should show a rendered diagram with complementary
text; unknown/failed rendering uses complete text with exact identity, Stage
order, scoped dependencies and declared gates. AXM-38 observed local SVG inline
on one exact Claude Desktop Code build; full qualification remains partial
(see [real-host Evidence](specifications/011-rendered-workflow-presentation/evidence-qualification.md)).
Source fences and file links alone do not prove rendering. This changes host
conversation guidance only. Native terminal and JSON contracts remain unchanged.
See [Spec 011 implementation Evidence](specifications/011-rendered-workflow-presentation/evidence-implement.md)
for qualification limits and isolated packaging/rollback checks.

Canonical MVP results use native terminal text by default, including TTY,
pipes and redirection ([Spec 008](specifications/008-output-separation/spec.md),
[ADR-0030](decisions/0030-canonical-results-terminal-and-host-presentation.md)).
The renderer projects the same canonical event once: status label/code, result,
next/references/details, labelled operation payloads and provenance. It generates
no Markdown headings, emphasis, tables or fenced JSON fallback. Deep collections
use compact native delimiters; values and actionable IDs/digests are not truncated.
Control and bidi characters remain visible as escapes. Output is static plain
text for every terminal capability, width and `NO_COLOR` setting (including an
empty value); no color, cursor movement or model call is required.

Scripts, automation and installed skills must request the stable typed JSON
surface with leading `--json`. There is no automatic JSON selection for non-TTY
stdout. `--human` explicitly selects the default; the programmatic Go `Run`
entrypoint retains JSON. Help remains deterministic human text even with JSON
selected. Conflicting/repeated/misplaced output flags remain input errors.
Short/broken writes return failure, even if the underlying operation confirmed
an effect; they do not roll back or repeat the operation.

```bash
axiom project show --selector my-project
axiom --json project show --selector my-project
axiom project list
axiom --json project list
axiom help
```

Strict S5 workflow selectors are explicit and independent of current directory:

```bash
axiom --json workflow run \
  --project <project-uuid-or-slug> \
  --repository <project-scoped-key> \
  --work-item 'github:<owner>/<repository>#<number>' \
  --workflow minimal --select agent.runtime=codex \
  --select agent.model=<chosen-model> --select agent.effort=runtime-default \
  --authorize-external

axiom --json workflow status \
  --project <project-uuid-or-slug> \
  --repository <project-scoped-key> \
  --work-item 'github:<owner>/<repository>#<number>' \
  --execution <execution-id>
```

A complete call asks zero questions. Interactive calls preserve supplied valid
values and ask only missing selectors. Unknown, duplicate, conflicting,
ambiguous, or malformed selectors return canonical `validation_failure` before
effects. A selector never grants local or Provider mutation authority.

Canonical JSON for current MVP surfaces, including workflow selector operations, uses
`status`, `result`, optional `references`, optional `next`, optional `details`, and
mandatory `provenance`. Human output renders the same canonical event. Other POC
operations temporarily retain `operation`, `status`, `category`, and applicable
typed payloads until their authorized MVP Tasks migrate them. Exit codes remain
`0` for success, `1` for failure, and `2` for interruption/cancellation.

| Codex skill | Stable Lingo entrypoint |
|---|---|
| `$axiom-project` | `workflow.select` → `project workflow select`; `configure` → `project configure`, `list` → `project list`, `show` → `project show`; `validate` / `archive` / `reactivate` / `integration` → [resource lifecycle](#maintain-resource-lifecycle-issue-230) |
| `$axiom-workflow` | `definition.list/show/validate/create/edit/remove/migrate/recover` → `project workflow` definition commands; `configuration.readiness` → read-only Runtime/Profile validate/status/auth |
| `$axiom-runtime` | `binding.*` / `profile.*` → canonical `runtime binding` / `runtime profile` authoring; exact preview/apply and scoped recovery |
| `$axiom-work-item` | `create` → `work-item create|select`, `run` → `workflow run`, `status` → `workflow status|evidence|list` (`status` mode `list` → `workflow list`); `list` / `show` / `update` / `comment` / `close` / `reopen` → [resource lifecycle](#maintain-resource-lifecycle-issue-230) |

Skills collect missing selectors conversationally, but Lingo retains validation,
repository resolution, workflow ordering, and external-mutation authority. Skills
request leading `--json` internally on natural and explicit entry paths. The
already operating host synthesizes a concise response from typed data: objective,
effective result, material blockers/risks/decision and supported next action.
It preserves every material exception and exact actionable token without a
mandatory payload dump, literal relay or Auxiliary paragraph. Explicit raw-JSON
requests receive the same original JSON unchanged; no operation is repeated.
Malformed, incomplete, unsupported, wrong-target or contradictory observations
stop dependent actions and require bounded read/handoff. Missing facts are
unknown; neither a summary nor completion success grants authority or acceptance.

For `project validate`, read `readiness.structure`, `effective` and each scoped
operation independently. Completion success/0, valid structure, effective partial,
Work Item ready and Execution blocked can coexist. Missing Runtime policy is a
supported explanation only with typed `runtime_policy_unavailable` in that
operation's blockers; `runtime_resolution_blocked` alone does not identify it.

The embedded presentation contract is skill-set/binary compatibility `3`.
Upgrade the binary and all four skills together in each host root. Version-2
owned bytes/receipts remain recognized for protected upgrade; modified or foreign
content is preserved and requires inspection. Mixed pairs are not ready merely
because their JSON parses. Before upgrading, retain the previous binary and an
exact backup of each owned skill root including its receipt. To roll back, stop
use of the pair, preserve the upgraded root separately, restore the complete
previous root to its original path and use its matching binary. Verify status
and all four digests before resuming; do not force an old installer to overwrite
new or user-modified content. Claude receipts bind the original root path.
Operational state is not removed or replayed by pair restoration. Historical
[#232](https://github.com/rgomids/axiom/issues/232) and AXM-31 acceptance remains
valid; Spec 008 prospectively replaces their presentation requirements.

## Resolve a configured Project globally

List configured Projects from protected local installation records, independent
of current directory and repository availability:

```bash
axiom project list
axiom --json project list
```

JSON always contains a `projects` array with `id`, `slug`, `name`, and the
machine-local `status` (`active`, `archived`, `invalid`, `recovery_required`); an
empty installation returns `"projects": []`. Archived Projects are hidden unless
`--include-archived` is given; an unreadable operational record is shown as
`invalid`, never as `active`. When portable display metadata is
temporarily unavailable, the configured Project remains present with an empty
`name`. The concise listing never includes source or repository paths.

Use a canonical Project UUID or installation-unique slug from any directory:

```bash
axiom project resolve --selector my-project
axiom project show --selector my-project
```

Resolution reads protected machine-local state. It never searches the caller's
current directory and fails explicitly for ambiguous Projects or unavailable
portable/repository locations. `project show` reports each Repository's
`availability` instead of failing on one broken binding, plus the Project's
machine-local state; `project resolve` stays strict.

## Configure a Project

Guided CLI:

```bash
axiom project configure
```

Repeatable non-interactive form:

```bash
axiom --json project configure \
  --slug my-project \
  --name "My Project" \
  --repository main=/absolute/path/to/working-copy \
  --work-item-provider github
```

This first call is read-only and returns `setup.projectId` plus an exact
`setup.digest`. After review, repeat the same facts with:

```bash
axiom --json project configure \
  --project-id <preview-project-id> \
  --slug my-project \
  --name "My Project" \
  --repository main=/absolute/path/to/working-copy \
  --work-item-provider github \
  --preview-digest "$PREVIEW_DIGEST" \
  --authorize-local
```

Repeat `--repository` for multi-repository Projects. Keys and capability intent
enter portable state; absolute paths and observed revisions remain only in
protected machine-local state and the review preview. Replaced bindings or changed
state invalidate authority. Guided mode previews the same normalized proposal and
asks before publication. Codex uses the same command through
`$axiom-project`; neither path infers identity from CWD.

### Guided bootstrap (Issue #231)

CREATE publishes portable schema 3 per the
[bootstrap v3 contract](specifications/002-lingo-project-initialization/project-bootstrap-v3.md).
For each explicit `--repository` (a bare absolute path gets a derived key
proposal), Lingo reads only that location's Git metadata (`.git`, `config`) and
file names: no Git process, fetch, network, CWD or parent walk. Remote names have
no priority (`origin` is not special); aliases of one locator collapse. Two or
more distinct locators, or Git configuration Lingo cannot fully read, block
publication until an explicit choice:

```bash
axiom --json project configure --slug my-project --name "My Project" \
  --repository core=/absolute/core --repository /absolute/web \
  --repository-remote core=https://github.com/acme/core.git \
  --work-item-provider github \
  --runtime claude --model-profile careful \
  --runtime-preference implementation/high=careful \
  --technology cloud.aws=aws --remove-technology package-manager.npm \
  --documentation architecture=repository:core/docs/architecture \
  --documentation notes=local-file:/absolute/notes/product.md \
  --business-context "Bounded product context" --context-source architecture \
  --glossary "work-item=Work Item:A bounded unit of engineering intent"
```

The preview lists Repository remotes and candidates, local Runtime candidates
(from the machine-local Runtime Profile configuration; nothing is defaulted),
detected technology facts with repository-relative Evidence (file names only;
conflicting package managers are flagged), documentation sources, glossary,
authority expectations and `blockers`. Runtime/profile/preference flags author the
#140 policy without editing `axiom.yaml`; omit them and Execution readiness reports
`runtime_policy_unavailable`. A `local-file` source stores only its key in
`axiom.yaml`; its absolute path and file identity are written to the
installation record (format 2), never its content. All CREATE-only flags are
rejected with `--project` (post-create lifecycle is #230).

## Validate Project readiness

```bash
axiom --json project validate --slug my-project
axiom --json project validate --project <uuid-or-slug>
```

`--project` validates an installed Project from the portable source its
installation record names (never `<projects-root>/<slug>`) and adds its
machine-local state.

Structural validation is unchanged (`Project is valid` / `Project state is
invalid`). A valid Project also returns a read-only `readiness` report:
Repository bindings, capability → Integration → Provider mapping, credential
binding presence (never values), the #140 Runtime availability, documentation
resolution (no paths or content), context counts, authority expectations, and
`operations` for `work-item` and `execution` with exact blockers. `effective` is
`ready`, `partial` (some operations ready) or `blocked`. Documentation and context
gaps are warnings and never block. These optional observations grant no run authority. Canonical `workflow run`
validates its actual linked target, Repository and effect authority directly;
Profile/catalogue/authentication observations are not preventive admission gates.

## Edit an existing Project Runtime/Profile policy

Use an explicit installed UUID/slug. `--policy-draft` accepts strict inline JSON,
not a file path or raw YAML. `{}` reads the current policy. Omitting the replay
tuple performs validation, dry-run and canonical preview only, with zero writes.

```bash
axiom --json project configure --project my-project --policy-draft '{}'
axiom --json project configure --project my-project --policy-draft '{"runtimes":{"mode":"members","add":[{"id":"codex"}]},"modelProfiles":{"mode":"members","add":[{"key":"worker","runtimeRef":"codex","model":"approved-model"}]},"runtimePreferences":{"mode":"members","add":[{"role":"implementation","complexity":"high","modelProfileRef":"worker"}]}}'
```

The `policyEdit.preview` contains only portable policy, exact Workflow impact,
scoped readiness, observed revisions, digest and allowed effects. It excludes
full manifests, host paths, credential references and local bindings. Review it,
then repeat the identical intent with all three returned authority inputs:
`--project-id <uuid> --preview-digest <digest> --authorize-local`. A stale or partial
tuple refuses. A policy no-op verifies exact authority and dependencies without
publishing or changing revisions. Guided and structured modes use the same input
and authorization; EDIT never infers consent from prompt text.

Omitted groups preserve their declaration and members. Each supplied group uses
one of `{"mode":"absent"}`, `{"mode":"unconfigured"}`,
`{"mode":"replace","values":[]}`, or `{"mode":"members","add":[],"replace":[],"remove":[]}`.
Runtime removal targets an ID string, Profile removal a key string, and preference
removal an exact `{"role":"implementation","complexity":"high"}` pair. Adds
require absence; replacements/removals require existence. Dependent removals or
replacements must be explicit. Empty configured lists differ from absent and
unconfigured. Bounds remain 8 Runtimes, 32 Profiles and 32 preferences; draft input
is at most 256 KiB, depth 32 and 16,384 nodes, with 16/64/64 member operations.
Do not combine policy intent with other EDIT or CREATE flags.

Policy storage requires portable validity and exact active Workflow compatibility;
missing/disabled/mismatched local Profiles do not delete portable declarations or
prevent an otherwise authorized save. `project validate` and policy read-back use
the same read-only metadata and policy rules for v2/v3. Executable, authentication
and effort sources remain uninspected (`inventory_unavailable`); the explicit
Runtime resolution operation still acquires its own current observations.
Readiness grants no authority. Follow `runtime.next` guidance to `axiom-runtime`
for local setup, under separate authority.

`policyEdit.outcome` preserves publication truth. `readBack.status` distinguishes
`confirmed`, `drift`, `unverified` and `recovery_required`, with expected/observed
revisions and any changed readiness snapshot. A confirmed portable commit followed
by local bookkeeping or read-back failure is partial, never claimed rollback.
Inspect recovery separately; do not reapply blindly. No policy edit rewrites
Workflow/Execution/Stage Plan snapshots, Work Items, another Project or local
Runtime/Profile configuration. Schemas 1/4/newer and unsupported source/layouts
are refused without migration. Source publication remains confined to the recorded
Project under the existing Lingo projects root.

## Maintain resource lifecycle (Issue #230)

Every user-managed MVP resource has its own lifecycle; there is no generic CRUD
and no delete. The frozen
[lifecycle matrix](specifications/004-mvp-v1-baseline/issue-230-resource-lifecycle-matrix.md)
and [Evidence](specifications/004-mvp-v1-baseline/evidence-230.md) are the
contract. Each mutation previews first; repeat the same command with the exact
returned digest and the explicit authority flag to apply it. An equivalent
request is a no-op that needs no authority.

### Project edit and Repository associations

```bash
axiom --json project configure --project my-project --name "New name" \
  --repository docs=/absolute/docs --remove-repository legacy
# after review:
axiom --json project configure --project my-project --name "New name" \
  --repository docs=/absolute/docs --remove-repository legacy \
  --project-id <id> --preview-digest <digest> --authorize-local
```

The preview is the complete resulting configuration; omitted values are
preserved. `--repository` attaches or updates an association and
`--remove-repository` detaches one: detach removes only the Axiom association and
its local binding, never the working copy or a remote, and the preview names
the preserved local Work Item/Execution history (`preserve_repository_history`).
The replay needs all three of `--project-id`, `--preview-digest` and
`--authorize-local` (`incomplete_edit_authority` otherwise); a stale preview or a
concurrent change is denied with zero writes. Publication writes the portable
manifest, then the installation record, behind owned recovery state: if it stops
in between, Project reads fail with `recovery_required` until `axiom recovery
inspect` / `apply` finalizes the confirmed portable change (it never rolls it
back). Edit publication is limited to Projects whose recorded source is the Lingo
projects root.

### Project archive and reactivate

```bash
axiom --json project archive --project my-project
axiom --json project archive --project my-project --preview-digest <digest> --authorize-local
axiom --json project reactivate --project my-project --preview-digest <digest> --authorize-local
```

Archive is machine-local and reversible. It is stored in
`<state-root>/projects/<project-id>/operational.json` (format 1; a missing record
means active) and never changes `axiom.yaml`, another machine, Repositories,
Work Items, Executions, Evidence or Providers. While archived, inspection and
administration work (`project show|validate|configure --project`, `integration
*`, `work-item list|show`, `workflow list|status|evidence`); operational
evolution fails with `project_archived` before readiness or any effect,
previews included: `work-item create|select|update|comment|close|reopen|complete`
and `workflow run`.

### Integrations

```bash
axiom --json integration list --project my-project
axiom --json integration show --project my-project --integration work-items
axiom --json integration validate --project my-project
axiom --json integration disable --project my-project --integration work-items
axiom --json integration enable --project my-project --integration work-items
axiom --json integration remove --project my-project --integration work-items
```

`list`/`show`/`validate` read portable declarations and local eligibility only
(no Provider call; credential references are named, never resolved).
`disable`/`enable` change only whether this machine may use the Integration:
every operation that would use it through a Provider or Transport fails with
`integration_disabled`, while inspection and administration stay available.
`remove` drops the portable declaration through the reviewed Project edit (same
replay tuple); removing `work-items` also removes its paired Provider, and a
removal that would leave a dangling reference fails. None of these revokes
credentials, logs out, uninstalls MCP or Runtime configuration, or deletes
Provider resources. A local disable entry for a key that is no longer declared is
reported as `staleDisabled` and cleared only by `enable`. Updating the Work Item
Provider remains `project configure --project <sel> --work-item-provider <id>`.

### Work Items

```bash
axiom --json work-item list --project my-project [--repository main]
axiom --json work-item update <selectors> --title "Clearer title" [--scope ...]
axiom --json work-item comment <selectors> --message "Note"
axiom --json work-item close <selectors>
axiom --json work-item reopen <selectors>
```

`<selectors>` are `--project`, `--repository` and `--work-item
github:<owner>/<repository>#<number>` (or `--number`). `list` shows Axiom-linked
Work Items only, from local links. `update` changes only the title and the
Axiom-authored sections; type and classification stay create-time. Every Provider
mutation previews the current Issue first and needs the exact `--preview-digest`
plus `--authorize-external`; closing a closed Issue or reopening an open one is a
no-op when the local link agrees. A stale local link is repaired under the
reviewed digest and authority without another Provider mutation. When the Issue
changed state outside Axiom, `work-item select` can also refresh the local link
under reviewed local authority. A comment is not
idempotent: repeating an authorized comment posts it again. A confirmed Provider
effect followed by a local failure is reported as `partial`; Axiom never claims
rollback or repeats the mutation.
`work-item delete` does not exist.

### Executions

```bash
axiom --json workflow list --project my-project [--repository main]
```

`workflow list` discovers the Project's Executions without a known identity
(read-only, deterministic order) and fails closed on damaged state. Canonical
Execution cancellation uses `workflow stage cancel`; it records intent and reports
the observed stop separately. History and Evidence remain immutable.

## GitHub Work Items

Create begins with a read-only, provider-neutral draft preview. Guided mode asks
only missing fields, prints the exact draft/target/effects/digest, and accepts
only the literal `yes` before the external effect:

```bash
axiom work-item create \
  --project my-project \
  --repository main \
  --provider-repository owner/repository
```

Choose `--type story|bug|task`. Guided creation asks for type; legacy
non-interactive callers without it receive an explicit `task` draft. A story also
requires `--beneficiary` (who benefits) and `--value` (concrete user/product
benefit), keeping implementation activity in scope.

Use repeatable `--classification <existing-provider-value>` to propose labels
explicitly. On GitHub these values are existing label names. Without explicit
values, the adapter selects an existing type label or reports
`no_existing_label_for_item_type` with an empty label set. Inspect type, story
value, `draft.providerDocument.metadata.labels`, and notices before authorizing.
Unknown explicit labels fail; changed proposed metadata invalidates the digest.
Dropped or unverified labels produce a partial result referencing the existing
Issue. Inspect that Issue and permissions; do not create another Issue to repair
classification. See the [classification contract](specifications/004-mvp-v1-baseline/work-item-classification.md).

The canonical Runtime skill `axiom-work-item` accepts a short problem statement,
extracts facts already provided, and asks conversationally only for material
missing information. You do not need to know the section schema. The Runtime
proposes safe assumptions explicitly and presents a complete Lingo preview for
review before requesting authority to create the Issue. Corrections require a
new preview; draft approval alone does not authorize external mutation.

The terminal CLI uses plain questions for missing sections; it does not perform
natural-language inference. For conversational extraction and synthesis, invoke
the Runtime skill. Both paths preserve the same seven-section draft contract.

For non-interactive use, provide all seven sections. The first call is read-only:

```bash
axiom --json work-item create \
  --project my-project \
  --repository main \
  --provider-repository owner/repository \
  --problem "Observed behavior blocks delivery" \
  --desired-outcome "Delivery proceeds safely" \
  --context "Observed on supported hosts" \
  --scope "Bounded application change" \
  --constraints "Preserve exact authority" \
  --non-goals "No workflow execution" \
  --acceptance "Deterministic checks pass"
```

After reviewing every preview fact, repeat the exact same fields with the returned
digest and explicit external authority:

```bash
axiom --json work-item create \
  --project my-project \
  --repository main \
  --provider-repository owner/repository \
  --problem "Observed behavior blocks delivery" \
  --desired-outcome "Delivery proceeds safely" \
  --context "Observed on supported hosts" \
  --scope "Bounded application change" \
  --constraints "Preserve exact authority" \
  --non-goals "No workflow execution" \
  --acceptance "Deterministic checks pass" \
  --preview-digest "$PREVIEW_DIGEST" \
  --authorize-external
```

Runtime-authored proposals use repeatable `--elaborated-section name=content`,
where `name` is `problem`, `desired_outcome`, `context`, `scope`, `constraints`,
`non_goals`, or `acceptance_expectations`. For example:

```bash
axiom --json work-item create \
  --project my-project --repository main --provider-repository owner/repository \
  --intent "Export fails when multiple rows are selected" \
  --desired-outcome "Export all selected rows" \
  --elaborated-section "context=No additional environment information supplied" \
  --elaborated-section "scope=Restore export of multiple selected rows" \
  --elaborated-section "constraints=Preserve unrelated behavior; no additional limits supplied" \
  --elaborated-section "non_goals=No unrelated export features" \
  --acceptance "Select two rows, export, and verify both in the saved file"
```

Ordinary section flags preserve verbatim user text as user-authored; elaborated
sections retain Axiom authorship after review. If a section combines extracted
facts with synthesis or paraphrase, transport the combination as elaboration.
Duplicate, unknown, empty, or simultaneously supplied/elaborated sections are
rejected. Incomplete JSON requests return missing questions without creating
anything. Reviewed fields and their authorship are bound to the preview digest.

`--intent` may supply the problem section when `--problem` is absent. Changed
facts invalidate the digest. Authentication comes only from the existing `gh`
CLI session; Axiom does not persist its credential or infer a Git remote.

Selecting an existing Issue is a separate local-publication review. Preview the
exact provider identity/state first, then repeat it with local authority:

```bash
axiom --json work-item select \
  --project my-project \
  --repository main \
  --provider-repository owner/repository \
  --number 123

axiom --json work-item select \
  --project my-project \
  --repository main \
  --provider-repository owner/repository \
  --number 123 \
  --preview-digest <preview-digest> \
  --authorize-local

axiom work-item show --project my-project --repository main --number 123
```

The reviewed effect set includes the local create-attempt fence. Before POST,
Axiom durably reserves the exact target/draft create attempt. An
ambiguous result leaves that fence `pending`; every later process reconciles the
persisted correlation only and cannot emit another POST until the ambiguity is
resolved. Zero matches remain fail-closed without a time heuristic; one exact
match is linked; multiple matches require operator review. Confirmed GitHub
effect plus local failure is reported as canonical `partial` with the Issue
reference. New local link filenames bind provider, resource, and external ID;
existing v1 records remain readable through exact identity validation. `comment`
and `complete` now follow the reviewed external-mutation contract of
[Maintain resource lifecycle](#maintain-resource-lifecycle-issue-230): a single
`--authorize-external` call returns `external_authority_denied` with the preview.
`complete` is a compatibility spelling of `close`; neither is workflow progress or
human acceptance.

## Execute the bounded workflow

Use the [canonical run operation](#run-and-inspect-a-canonical-execution).
The Execution retains the selected definition/digest and immutable attempt
invocation, authority, process observations and validated results. Changing a
Project selection changes future runs; it does not rewrite retained attempts.
`workflow status`, `evidence` and `list` inspect canonical Executions only.
They never activate or convert old state.

### Retired stage dispatch contracts

Stage Plan, Graph/Single dispatch, lifecycle gate/fact writers and Provider
projection are retired. Historical protocol Evidence remains in the
[#275 record](specifications/007-configurable-workflows/issue-275-implementation.md)
and [#274 record](specifications/007-configurable-workflows/issue-274-implementation.md).
Those records describe historical contracts and confer no permission to resume
old attempts. New runs use `executions/v2` exclusively.

## Compatibility, cleanup, recovery, and upgrade

These S7 maintenance commands follow one rule: without `--authorize-local` they
are read-only previews; a mutation requires repeating the command with the exact
`--preview-digest` of a current preview. Any change between review and apply
denies authority. Targets and roots are explicit; the CWD is never used.

Classify the configured roots (`LINGO_PROJECTS_ROOT`, `LINGO_STATE_ROOT`, and the
Codex skill root) without creating, locking, or changing anything:

```bash
axiom --json compatibility inspect
```

The closed classifications are `absent_v1`, `valid_v1`, `recognized_poc`,
`malformed`, `unsupported_older`, `unsupported_newer`, and `recovery_required`.
Recognition of historical `v0.1.0-poc.1` state requires the complete positive
POC workflow signature. That signature beside v1-only state (written by a v1
release that already operated over the root) is `valid_v1` with reason
`v1_state_with_preserved_poc_history`: the POC records stay in place, untouched,
and are never read, migrated, or promoted (an explicit opt-in migration is not
available; see Issue #175). Partial, unknown, unsafe (link, mode,
ownership, ACL, type, size), and uncertain state is preserved for review and is
never reported as absent. Only Axiom-owned skill names in the shared skill root
are inspected. The report contains categories, kinds, owned relative names, and
digests, never file content.

Recognized POC state is never migrated in place. Preserve it with separate
authorities, each writing only new private objects into an absent target on the
same local filesystem as the source, outside every owned root:

```bash
axiom --json compatibility backup --target /absolute/new/poc-backup
axiom --json compatibility export --target /absolute/new/poc-export
# after review, repeat each with --preview-digest <digest> --authorize-local
```

Backup copies every recognized POC object; export copies only validated
portable Project manifests to `<target>/projects`, which a separate clean
`LINGO_STATE_ROOT` can then configure explicitly with `axiom project configure`.
`manifest.json` is written last; a target without it is incomplete and is never
adopted. An equivalent completed target is reported as a no-op.

Explicit, reference-aware artifact cleanup:

```bash
axiom --json artifact cleanup
axiom --json artifact cleanup --preview-digest <digest> --authorize-local
axiom --json artifact retire --artifact <artifact-id>
axiom --json artifact retire --artifact <artifact-id> --preview-digest <digest> --authorize-local
```

Eligible: unreferenced `diagnostic` artifacts at least 30 days old, `evidence`
artifacts at least 365 days after an explicit retirement, and confirmed cleanup
records at least 90 days old. References come from every local Execution record
and artifact under the state-root lock; uncertain reference state fails closed.
`active`, `preserved_review`, and referenced artifacts are always preserved.

`artifact retire` records that Evidence has no live reference; it removes
nothing. It is denied for any referenced, non-Evidence, or already retired
artifact, and writes `artifacts/v1/retirements/<id>.json` (format 1, separate
from metadata format 1) bound to the exact artifact revision. Evidence without a
retirement, or with a corrupt, stale, or other-revision retirement, is preserved.
A new artifact that references a retired artifact supersedes its retirement, so
a later retirement must be recorded explicitly. Each authorized batch removes
at most 128 exact objects and publishes a bounded cleanup record before removal.
Capacity exhaustion never deletes anything.

Guided recovery of interrupted local publication:

```bash
axiom --json recovery inspect
axiom --json recovery apply --preview-digest <plan-digest> --authorize-local
```

Inspection lists each interrupted protocol directory, its marker, generations,
fault stage, and proposed action. Only `restore_prior` (before the commit point)
and `finalize_committed` (after it) are ever applied, under a fresh exact plan
digest and the owning store's lock order; a committed generation is never rolled
back. Ambiguous, contradictory, corrupt, or unknown state and interrupted S2
attempt markers remain `preserved_review`.

Owned upgrade of a release installation made by `install-release.sh`:

```bash
axiom --json upgrade \
  --archive /absolute/release/axiom-0.2.0-macos-27-arm64.tar.gz \
  --checksums /absolute/release/SHA256SUMS \
  --bin-dir /absolute/user-owned/bin \
  --receipt-dir /absolute/user-owned/state
# after review, repeat with --preview-digest <digest> --authorize-local
```

The archive is verified exactly as the installer does before anything else.
Preflight requires the exact approved host row, an unmodified owned binary and
receipt, a newer version (downgrade and divergent same-version replacement are
refused), a supported persisted-state transition, and observed free space.
Ownership and persisted-state compatibility are evaluated separately: the
inspected state resolves through one compatibility policy to exactly one
`transition.strategy` (`direct`, `migrate`, `preserve_rebuild_reconfigure`, or
`refuse`), bound into the preview digest with the state digest. The stable
window is exactly persisted-state v1 (`direct`); absent state is also `direct`,
and so is v1 state beside preserved POC workflow history, which the upgrade
leaves unchanged.
Recognized historical POC state selects `preserve_rebuild_reconfigure`, which
the upgrade executes before any installation effect (Issue #153):

1. **Preserve.** Every inventoried object of the Projects and State roots is
   copied into an Axiom-owned machine-local archive,
   `<archive root>/recognized-poc-<source digest>/`, as
   `objects/<sha256>` (content-addressed, `0600` in `0700` directories), each
   re-read and verified from the destination. The archive root is `archive`
   beside the installation receipt directory (`~/.local/state/axiom/archive`
   for the default installer), or `AXIOM_ARCHIVE_ROOT` when set to an absolute
   path; it must be private, owned by you and outside every Project, State and
   Skills root, and may be on another filesystem (copy and verify, never a
   cross-filesystem rename).
2. **Prove.** `manifest.json` is written last and lists `policy`
   (`recognized-poc-preservation/v1`), `sourceDigest`, and every object's
   `category`, `relative`, `kind`, `sha256` and `bytes`, plus the `retired` and
   `kept` sets. Nothing is retired until the manifest equals the revalidated
   inventory, every listed object verifies, and the archive holds nothing
   else; the verified manifest digest is then bound to the operation marker.
3. **Rebuild and reconfigure.** Only historical workflow material is retired
   from active state: `workflows/` records and the Work Item links the POC
   workflow created. Installation records and portable `axiom.yaml`
   manifests, which the current contract already validates, stay in place, so
   the Project remains configured. The rebuilt root must resolve `direct`
   (`valid_v1`) before the binary is replaced.

The installer prints `install_preserved=<archive>` on success. The archive is
historical material only: it is never read as active state, never promoted to
an Execution, and never deleted by an upgrade; remove it yourself only when
you no longer need it. Every boundary is resumable with the same archive under
fresh exact authority. Transition markers use `formatVersion=2` and record
`transitionArchive`, the verified `transitionManifest` before retirement,
`transitionRetired` (the confirmed prefix of Work Item links followed by
workflow records, each ordered by category/path), and `transitionActivated`.
Every confirmed retirement updates the marker before the next effect. Until
activation is explicitly recorded, kept objects and pending retirements must
remain present and byte-exact; only that confirmed prefix may be absent.
Activation requires complete retirement progress, exact kept objects and a
`direct` rebuilt root. Afterwards normal v1 writes are permitted, while the
recorded archive and complete retirement count still verify. Unknown fields,
duplicate fields, incompatible transition markers (including version 1), and
contradictory progress require recovery. If interruption occurs after deletion
but before its marker confirmation, absence is unproved and automatic resume
refuses; preserve the archive and marker for operator review. Retirement uses
the same state -> family -> Project locks as local writers and revalidates the
exact generation while holding them. Existing manifests must equal the
canonical policy encoding, including POC identity and derived kept/retired
sets; extra JSON is refused. A changed source, an archive that
does not correspond, an unsafe or overlapping archive location, or missing
space stops with no further retirement (`state_changed`,
`preservation_conflict`, `preservation_target_unsafe`, `insufficient_space`).
An equivalent rerun after success is a no-op. A caller without an archive
location, or a resumed upgrade that was not authorized as a transition, stops
before any effect with `state_transition_unavailable`. Other state is refused before any effect as
`state_unsupported` (newer or outside the window), `state_unsafe` (foreign,
modified, ambiguous, corrupt, or unsafe), or `state_recovery_required`
(interrupted operation). Each refusal's next step is a product action, not a
manual compatibility command sequence. The binary
and then the receipt are published and re-read as separate confirmed effects;
`installedAt` is preserved. A later failure is `partial`: the installer's
`.axiom-install-operation` marker records the exact archive, so only the same
archive can resume (through `axiom upgrade` or the release installer), and any
other install is refused in the meantime. The upgrade publishes skill files
only to the Codex root, followed by the Codex skill-set receipt when the
running binary is the candidate release: its build provenance (clean release,
version and revision) equals the candidate's checksum-verified metadata and it
embeds exactly the candidate's skill files, which is always the case for the
release installer. The receipt is replaced only when absent or recognized as
Axiom's, as its own authorized effect after every skill file, and an
interruption resumes like any other effect. An upgrade not run as the
candidate (for example an older binary given a newer archive) cannot derive
the candidate's receipt: it ends `partial` with `skillReceipt=refresh_required`,
and an unrecognized receipt is preserved with `skillReceipt=conflict`; the
diagnostic names the next action. In every case `axiom first-run` with the
upgraded binary converges every other detected Runtime, such as Claude, from an
earlier Axiom-owned revision to the new one.
There is no automatic update, rollback, or cross-root transaction.

Release PR check resolution (read-only):

```bash
./scripts/release-pr-checks.sh resolve rgomids/axiom
python3 scripts/test-release-pr-checks.py
```

After every successful Release Please run (dispatched by `release.sh start`),
the workflow resolves the current open Release PR independently of action
outputs, validates its bot author, repository, main base, expected branch,
pending label and the title `chore(main): release <planned_version>`, checks
its SHA against the remote ref and requires its single parent to be the
validated `main` SHA or an ancestor of it before dispatching both required
workflows (`dispatch rgomids/axiom <planned_version> <main>`). A missing
Release PR, another version, an unvalidated base, ambiguity, API errors and
identity drift fail the run, so such a Release PR gets no required checks. Repeated
dispatches only rerun checks. Existing Release PR heads must contain both workflow
files; this path does not update their branches or bypass the delivery guard.

## Delivery packet (#277)

`axiom workflow delivery packet --project <id> --repository <key> --work-item
github:<owner>/<repository>#<number> --execution <id>` inspects canonical
Execution readiness without writing a companion packet or contacting a Provider.
It reports the retained definition/attempt/result correlation, completion and
blockers. Incomplete, cancelled or unknown results cannot become ready.

### Exact delivery publication

Delivery publication/prepare/apply/inspect/reconcile effects are unavailable on
this S1 path. The readiness response grants no push, PR, merge, release or
Provider authority. Canonical delivery effects are approved S3 work; old packet
and publication bytes remain inert rather than being resumed or converted.

### Human delivery decision and linked rework

Human acceptance and linked rework are S3 work. Technical completion and retained
Evidence do not accept the product or close its Work Item. Historical delivery
contracts remain recorded in [Spec 007](specifications/007-configurable-workflows/spec.md);
they are not an alternative operational engine.
