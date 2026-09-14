# Getting Started

## Expected location

O checkout local esperado neste ambiente é:

```text
/Users/rgomids/Projects/axiom
```

O repositório remoto usa SSH. Não mova, apague ou sobrescreva outro diretório para recriar esse checkout.

## Minimum requirements

- Git com acesso SSH autenticado ao GitHub;
- Bash;
- utilitários POSIX usados pelos scripts (`find`, `grep` e `awk`);
- Codex para operar o harness de agentes;
- GitHub CLI apenas para administração do repositório remoto.

T01 da Specification 002 entrega domínio puro em Go. Go 1.26 é requisito para seus testes e build; biblioteca padrão somente, sem instalação de dependências externas. Ainda não existe CLI Lingo. Consulte [comandos de domínio](../commands.md#t01-domain-validation) e [Evidence T01](../specifications/002-lingo-project-initialization/evidence-t01.md).

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

1. Abra o repositório em `/Users/rgomids/Projects/axiom`.
2. Leia [../../AGENTS.md](../../AGENTS.md) antes de alterar arquivos.
3. Consulte somente o contexto e as políticas relevantes em `.agents/`.
4. Execute mudanças através da menor skill aplicável em `.agents/skills/`.
5. Preserve a separação entre discovery, hipótese, specification, decisão, implementação e evidência.
6. Não invente comandos da futura CLI.

## Before a commit

Siga [../security/repository-security.md](../security/repository-security.md). No mínimo, revise todo o diff staged, execute o checker com `--staged`, valide o harness e confirme que nenhum dado privado de providers foi incorporado.

Lista consolidada de comandos: [../commands.md](../commands.md).
