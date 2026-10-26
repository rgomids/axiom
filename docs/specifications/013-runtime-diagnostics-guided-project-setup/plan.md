# Plan 013 — Diagnósticos de Runtime/Profile e setup guiado

**Proposed — revisão 1, 2026-10-25.** Implementa a [Spec 013](spec.md).

## Decisões técnicas locais

| Decisão | Escolha | Justificativa |
|---|---|---|
| Forma do diagnóstico | Lista aditiva `runtimeAuthoring.diagnostics[]` | Preserva categorias/contratos (RD-04); várias violações visíveis de uma vez |
| Fonte da explicação | `runtimeprofile.ProfileViolations` ao lado das regras | Mantém regras e explicação no mesmo pacote; `Validate` intocado; teste de concordância (RD-08) |
| Recusas de `editCandidate` | Mapeamento código → diagnóstico em `runtimeapplication` | Sem nova lógica de domínio; somente explicação |
| `next` do completion | Primeiro diagnóstico + escopo declarativo | Terminal mostra `next` no topo; paridade via JSON único |
| Wizard | Orientação na skill `axiom-project` + nota em `axiom-runtime` | Requisito de não criar mecanismo paralelo; operações canônicas já cobrem toda a jornada |
| Ordem Runtime → Project | Create inclui política | Evita Project schema 3 com readiness apenas por metadata |

Nenhuma decisão arquitetural durável nova: ADR não requerida. ADR-0023/0024/0030
e ADR-0032 preservadas.

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
