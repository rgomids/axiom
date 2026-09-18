# Getting Started

## Clone the repository

Em um diretório de sua escolha:

```bash
git clone https://github.com/rgomids/axiom.git
cd axiom
```

Os comandos seguintes assumem a raiz do repositório. Se já houver um checkout,
use-o sem mover, apagar ou sobrescrever outros diretórios. SSH não é requisito.

## Minimum requirements

- Git; acesso HTTPS ao GitHub para clonar o repositório público;
- Bash;
- utilitários POSIX usados pelos scripts (`find`, `grep` e `awk`);
- Codex apenas para operar o harness de agentes, não para executar as validações;
- GitHub CLI apenas para administração do repositório remoto.

A fundação Go da Specification 002 contém domínio, contratos de aplicação e codecs
portátil e local JSON. Go 1.26 ou posterior é requisito para os checks Go; o codec
portátil depende de `go.yaml.in/yaml/v3@v3.0.5`.
Consulte [preparação do cache e validação offline](../commands.md#t01t04-validation)
e [Evidence T04](../specifications/002-lingo-project-initialization/evidence-t04.md).
O POC Lingo oferece lifecycle portátil mínimo e `lingo project install`, que
persiste `installation.json` em `<state-root>/projects/<project-id>/installation.json`.
`reopen` reconhece o estado local quando disponível, separado da configuração
portátil. Bindings, resolução de credenciais, Runtime observations, reconciliação
avançada, hardening completo de filesystem, crash recovery e proteção contra
races de symlink, canonical-path e ancestor continuam fora do POC; o escopo
restante é #19–#21. Consulte [comandos](../commands.md#lingo-poc-minimal-lifecycle).
Consulte o [lifecycle atual](../specifications/README.md#002--lingo-project-initialization)
para distinguir implementação mergeada, aceitação humana e autorização de novas Tasks.

## Validate the harness

Na raiz do repositório:

```bash
./scripts/validate-repository.sh .
./scripts/validate-agent-package.sh .
```

Execute também os checks locais de segurança:

```bash
./scripts/test-validate-agent-package.sh
./scripts/test-check-sensitive-files.sh
./scripts/check-sensitive-files.sh .
```

## Work with Codex

1. Abra a raiz do seu checkout do repositório.
2. Leia [../../AGENTS.md](../../AGENTS.md) antes de alterar arquivos.
3. Consulte somente o contexto e as políticas relevantes em `.agents/`.
4. Execute mudanças através da menor skill aplicável em `.agents/skills/`.
5. Preserve a separação entre discovery, hipótese, specification, decisão, implementação e evidência.
6. Use somente os comandos POC documentados; não invente comandos futuros.

## Before a commit

Siga [../security/repository-security.md](../security/repository-security.md). No mínimo, revise todo o diff staged, execute o checker com `--staged`, valide o harness e confirme que nenhum dado privado de providers foi incorporado.

Lista consolidada de comandos: [../commands.md](../commands.md).
