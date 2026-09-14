# Project Commands

Execute estes comandos na raiz de `/Users/rgomids/Projects/axiom`.

## Runtime

Não há aplicação ou CLI para executar neste bootstrap. Nenhum comando futuro do Axiom é definido aqui.

## T01 domain validation

Go 1.26 instalado. Biblioteca padrão somente; sem serviços/dependências locais.
Build valida pacote de domínio; não produz CLI.

```bash
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
go test -cover ./...
go test -race -shuffle=on -count=10 ./internal/project
go vet ./...
go build ./...
go run ./scripts/check-project-domain.go
bash scripts/test-check-project-domain.sh
```

O checker AST inspeciona imports/símbolos puros e helpers dos testes fora da suíte
de domínio. O runner Go e ferramentas de verificação fazem I/O de compilação e
relatório; comportamento de domínio/testes não faz I/O de aplicação.
[Evidence e limites de T01](specifications/002-lingo-project-initialization/evidence-t01.md).

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

Não use Makefile como interface principal. Este repositório não possui Makefile nem runtime de aplicação neste estágio.
