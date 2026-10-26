# Plan 013 — Diagnósticos de Runtime/Profile e setup guiado

**Proposed — revisão 3, 2026-10-26** (revisão 2: CR-356-01, CR-356-02; revisão 3: Model Advisor, ampliação de escopo do mantenedor). Implementa a [Spec 013](spec.md).

## Decisões técnicas locais

| Decisão | Escolha | Justificativa |
|---|---|---|
| Forma do diagnóstico | Lista aditiva `runtimeAuthoring.diagnostics[]` | Preserva categorias/contratos (RD-04); várias violações visíveis de uma vez |
| Fonte da explicação | `runtimeprofile.ProfileViolations` ao lado das regras | Mantém regras e explicação no mesmo pacote; `Validate` intocado; teste de concordância (RD-08) |
| Recusas de `editCandidate` | Mapeamento código → diagnóstico em `runtimeapplication` | Sem nova lógica de domínio; somente explicação |
| `next` do completion | Primeiro diagnóstico + escopo declarativo | Terminal mostra `next` no topo; paridade via JSON único |
| Wizard | Orientação na skill `axiom-project` + nota em `axiom-runtime` | Requisito de não criar mecanismo paralelo; operações canônicas já cobrem toda a jornada |
| Ordem da jornada (CR-356-01) | Identidade → inspeção de Workflow → Profiles → política G-1 → seleção → validate | Única ordem viável com contratos atuais: `project workflow list/show` exigem Project; G-1 recusa Project com Workflow selecionado; readiness só é observada após a seleção |
| `--enabled` em Profile (CR-356-02) | Diagnóstico em `inputDiagnostics`, parser intocado | Menor mudança; categoria `invalid_input` e JSON preservados; erro específico em JSON e humano |

Nenhuma decisão arquitetural durável nova: ADR não requerida. ADR-0023/0024/0030
e ADR-0032 preservadas.

## technical_decisions

1. **CR-356-01 — bootstrap parcial em vez de nova superfície.** Alternativa
   avaliada: inspecionar built-ins sem Project (`project workflow list/show`
   sem `--project`). Rejeitada nesta correção por adicionar contrato CLI/skill
   metadata; o bootstrap usa somente operações publicadas, sem nova fonte de
   definições nem resolução duplicada.
2. **Derivação de requisitos** apenas do resultado público de `project
   workflow show`: união de `capabilities`, complexidades e um par
   role/complexity por Agent para preferências; nenhum valor fixo no teste.
3. **Pin de skill-set não publicado substituído.** O manifesto intermediário
   `295fbd…` existiu só nesta branch; o pin passa ao novo embutido, sem entrada
   em `sharedSkillHistory`.
4. **Demais rejeições de `safeAuthoring`** (`--cascade` fora de remove,
   campos de Profile ou `--enabled` em list/show/remove/…) são inalcançáveis
   pelo parser, que só registra essas flags nas operações válidas; mantêm o
   fallback.

5. **Model Advisor como orientação de skill (revisão 3).** Os contratos atuais
   já aceitam um modelo como intenção (`modelSupport: unconfirmed`) e separam
   autoria de autenticação. A pesquisa é uma capacidade do host conversacional,
   e nenhuma necessidade observável exige serviço, port, comando ou estado
   persistente. Por isso o Advisor vive em `axiom-runtime` (dono dos Profiles),
   e `axiom-project` o oferece na etapa 4. Sem ADR: não há decisão arquitetural
   durável; reverter é editar a skill.
6. **Fixtures sintéticas** com fornecedores e modelos fictícios (`va-*`,
   `vb-*`) tornam os casos MA independentes de dados comerciais voláteis. Um
   teste de guarda impede nomes comerciais e preços nas skills e fixtures.
7. **Linhas pré-existentes reconciliadas:** “never recommend a concrete
   model/effort” sem fonte autoritativa passou a reconhecer documentação
   oficial registrada pelo Advisor como fonte de recomendação, nunca de
   suporte.

## deferred_findings

- **DF-013-01** Inspeção de definições built-in sem Project permitiria
  configurar Profiles antes de publicar qualquer Project; exige decisão de
  contrato CLI.
- **DF-013-02** Project com Workflow selecionado e sem política não tem
  caminho de edição G-1 (“requires schema 2 or 3”); exige operação aprovada
  própria.

- **DF-013-03** Medir consumo real por Profile (tokens, quotas) para basear
  recomendações de economia em dados observados; exige Evidence de execução
  e decisão própria.
- **DF-013-04** Interface oficial de descoberta de modelos por Runtime, se um
  fornecedor documentar uma sem autenticação; avaliar adapter próprio.

## Arquivos

- `internal/runtimeprofile/violations.go` — `Violation`, `ProfileViolations`.
- `internal/runtimeapplication/authoring.go` — `AuthoringDiagnostic`,
  `inputDiagnostics`, `candidateDiagnostics`, `refusalDiagnostics`, `next`.
- `internal/codexruntime/skills/axiom-{project,runtime}/SKILL.md`,
  `manifest.go` (histórico v0.22.0), `testdata/pre-guided-setup/`.
- Documentação: `docs/commands.md`, `docs/operational/{runtime,project,troubleshooting}.md`.

## Testes

- Unit: `internal/runtimeapplication/authoring_diagnostics_test.go`.
- E2E binário isolado: `cmd/lingo/guided_setup_journey_test.go` (executáveis
  sintéticos que falham se invocados).
- Skills: `internal/codexruntime/guided_setup_skill_test.go`.

## Rollout e rollback

Mudança aditiva no JSON e nas skills. Rollback = reverter o PR; skills
anteriores continuam reconhecidas pelo histórico compartilhado. Sem migração
de dados.
