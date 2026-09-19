# Project Commands

Execute estes comandos na raiz do repositório.

## Lingo POC minimal lifecycle

O POC é local. Ele não acessa rede, Git, providers, runtimes ou credenciais. Use
um root temporário/isolado enquanto avalia; `LINGO_PROJECTS_ROOT` deve ser absoluto.
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

Cada operação escreve um evento JSON seguro em stdout e retorna `0` em sucesso,
`1` em erro/conflito e `2` em cancelamento. `init` é create/no-op/conflito: não
renomeia nem atualiza um Project existente; use `update` para alterar o nome.

`install` grava o record local estrito em `<state-root>/projects/<project-id>/installation.json`;
não altera o manifest de origem. O adaptador atual aceita somente Project mínimo de um único `axiom.yaml`. Ele
recusa symlinks, artefatos extras e roots relativos em vez de tentar preservar
documentos opcionais sem um protocolo completo. Binding,
rename, documentos, Git, Runtime, Provider e rede permanecem fora deste incremento.

Execute o dogfooding reproduzível deste incremento em roots temporários:

```bash
./scripts/dogfood-poc.sh
```

O script exige Go 1.26, Bash, `find`, `wc`, `tr`, `cut`, `shasum` e `cmp`. A saída contém eventos
JSON seguros e um resultado estruturado com hashes SHA-256. Instalação usa publicação sem substituição; resíduos
de tentativas interrompidas retornam `recovery_required` e exigem inspeção humana.
No macOS, o adaptador exige build com cgo para inspecionar ACLs; sem cgo,
operações de filesystem falham fechadas.
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
go run ./scripts/check-project-domain.go
bash scripts/test-check-project-domain.sh
go run ./scripts/check-projectapp.go
bash scripts/test-check-projectapp.sh
```

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
`lingo project install`, que persiste `installation.json` em
`<state-root>/projects/<project-id>/installation.json`; `reopen` reconhece esse
estado local quando disponível. Configuração portátil e estado local permanecem
separados. Bindings, resolução de credenciais, Runtime observations,
reconciliação avançada, recuperação automatizada e a matriz completa de falhas
de filesystem continuam fora da evidência concluída; #19–#21 seguem abertos.

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

## Temporary Spec-Kit evaluation

Validate the frozen Scenario 002 inputs, prototype hashes, scripts, and failure
behavior without installing a persistent dependency:

```bash
cd experiments/speckit-evaluation/scenarios/002-analyze-converge-reuse
shasum -a 256 -c scenario/checksums.sha256
shasum -a 256 -c protocol/freeze.sha256
shasum -a 256 -c protocol/prototype-freeze.sha256
bash -n scripts/*.sh adapters/speckit/*.sh
scripts/test-tools.sh
adapters/speckit/test-failures.sh
```

`scripts/test-tools.sh` also validates the unexpected-finding review and proves
that seeded reference matching rejects substring-only matches.

Não use Makefile como interface principal. Este repositório não possui Makefile.

## Install the E2E POC locally

Install Axiom from the current checkout. The default user binary destination is
`$HOME/.local/bin`; it must already be included in `PATH` for bare `lingo`
invocation:

```bash
./scripts/install-axiom.sh
lingo version
```

For an isolated or custom user destination:

```bash
AXIOM_BIN_DIR=/absolute/path/to/bin \
AXIOM_INSTALL_STATE_ROOT=/absolute/path/to/state \
./scripts/install-axiom.sh
```

The installer is safe to rerun. It replaces only a prior binary whose exact
checksum matches its protected receipt; an unrelated or modified destination is
refused. Test installation and unrelated-CWD invocation with:

```bash
./scripts/test-install-axiom.sh
```

## Configure Codex Runtime

Install and inspect the global thin Axiom skills:

```bash
lingo runtime codex install
lingo runtime codex status
```

Codex standalone skill names accept lowercase letters, digits and hyphens, so
the requested semantic `axiom:<skill>` names are invoked as:

```text
$axiom-project-configure
$axiom-project-show
$axiom-work-item-create
$axiom-work-item-run
$axiom-work-item-status
```

The default user-global root is `$HOME/.agents/skills`. For isolated validation:

```bash
AXIOM_CODEX_SKILLS_ROOT=/absolute/test/root lingo runtime codex install
./scripts/test-codex-skills.sh
```

## Resolve a configured Project globally

Use a canonical Project UUID or installation-unique slug from any directory:

```bash
lingo project resolve --selector my-project
```

Resolution reads protected machine-local state. It never searches the caller's
current directory and fails explicitly for ambiguous Projects or unavailable
portable/repository locations.

## Configure a Project

Guided CLI:

```bash
lingo project configure
```

Repeatable non-interactive form:

```bash
lingo project configure \
  --slug my-project \
  --name "My Project" \
  --repository main=/absolute/path/to/working-copy
```

Repeat `--repository` for multi-repository Projects. Keys enter portable intent;
absolute paths remain only in protected machine-local state. Codex uses the same
command through `$axiom-project-configure`.

## GitHub Work Items

Create or select one GitHub Issue linked to a configured Project repository:

```bash
lingo work-item create \
  --project my-project \
  --repository main \
  --title "Bounded change" \
  --body "Scope and acceptance criteria" \
  --authorize-external

lingo work-item select --project my-project --repository main --number 123
lingo work-item show --project my-project --repository main --number 123
```

Add an Evidence/status reference or complete the Issue:

```bash
lingo work-item comment --project my-project --repository main --number 123 \
  --message "Evidence: ..." --authorize-external
lingo work-item complete --project my-project --repository main --number 123 \
  --authorize-external
```

Create, comment and complete refuse execution without explicit external mutation
authority. Authentication comes from the existing `gh` CLI session; Axiom does
not persist its credential.
