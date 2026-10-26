# Specification 013 — Diagnósticos de Runtime/Profile e setup guiado completo de Project

## Status e autoridade

**Proposed — revisão 1, 2026-10-25.** Pedido explícito do mantenedor para
especificar, implementar e entregar ambas as correções em uma branch e um PR,
sem merge, release ou deploy. Aceite humano permanece pendente e não é
substituído por checks. [Plan](plan.md) · [Tasks](tasks.md) · [Evidence](evidence.md).

Baseline: `c00156f` (main, pós-v0.22.0). Constrói sobre a
[Spec 009](../009-guided-project-runtime-configuration/spec.md) (jornada de
readiness de Project existente, GC-01–06) sem reabrir seus contratos.

## Problema

Durante o setup do Project sintético `govsm` (v0.22.0, revision `215f83db930a`),
`runtime binding create --runtime codex --profile codex-default --model
gpt-6-luna --effort-mode explicit --effort low` retornou
`validation_failure / candidate_invalid`, sem nomear o campo. A IA operadora
investigou autenticação e variáveis de ambiente sem relação causal. Causa
revalidada no HEAD: `runtimeprofile.Validate` exige ao menos uma capability
(`validTokens(profile.Capabilities, false)`) e colapsa toda violação de Profile
em `ErrInvalidConfiguration`; `safeAuthoring` faz o mesmo com `invalid_input`.
Outras recusas (`binding_missing`, `runtime_absent`, colisões) também não
nomeavam campo nem próxima ação.

Separadamente, `axiom-project` permitia criar um Project com identidade,
Repository e Provider e terminar com Runtime/Profile ausentes, sem conduzir
política portátil e Workflow — o operador precisava descobrir `axiom-runtime`,
`--policy-draft` e `workflow.select` sozinho.

## Requisitos

### RD — Diagnósticos de autoria local

- **RD-01** Toda recusa de `runtime binding|profile` create/edit/
  validate-candidate (e demais operações com causa local conhecida) MUST
  incluir `runtimeAuthoring.diagnostics[]` com `code`, `field` (flag CLI),
  `rule` e `next`, sem ecoar o valor rejeitado.
- **RD-02** Códigos distinguem: `field_required`, `field_invalid`,
  `binding_missing`, `profile_missing`, `runtime_absent`,
  `configuration_conflict`, `configuration_invalid` (fallback de invariante
  cruzada).
- **RD-03** `--capabilities` omitido → `field_required` em `--capabilities`,
  com sintaxe `--capabilities <comma-list>` e orientação de derivá-las dos
  Agents do Workflow. `--capabilities` continua obrigatório; `--complexities`
  continua opcional.
- **RD-04** Categorias públicas existentes (`candidate_invalid`,
  `invalid_input`, `binding_missing`, …) e status de completion permanecem
  inalterados; o campo é aditivo (`omitempty`), sem alterar digests de preview
  bem-sucedidos.
- **RD-05** Recusas declarativas declaram que autenticação não foi avaliada e
  não orientam alterar credenciais/ambiente. `authentication: unknown`,
  `modelSupport/effortSupport: unconfirmed` permanecem.
- **RD-06** `candidate_valid` informa que não é certificação de fornecedor.
  Modelo desconhecido pode ser salvo como intenção; nada é afirmado sobre
  suporte. `binding_missing` informa que não diz nada sobre modelo.
- **RD-07** Saída humana e JSON derivam do mesmo documento canônico (paridade).
- **RD-08** A explicação nunca relaxa regras: `runtimeprofile.Validate` segue
  como única autoridade; `ProfileViolations` concorda com ela (teste).
- **RD-09** Nenhum auth preflight, processo de fornecedor ou inferência é
  executado pela autoria declarativa.

### WZ — Setup guiado completo (skill `axiom-project`)

- **WZ-01** Uma seção “Complete guided setup” coordena, para Project novo ou
  existente: descoberta → Runtime/Profiles → identidade/Repositories/Provider →
  contexto → política portátil → Workflow → validação final.
- **WZ-02** Runtime/Profiles são configurados antes de publicar Project novo
  destinado à execução, e o create preview inclui `--runtime`,
  `--model-profile` e `--runtime-preference` necessários.
- **WZ-03** Escopo machine-local e compartilhado dos Profiles é explicitado
  antes de qualquer mutação; capabilities derivam dos Agents do Workflow.
- **WZ-04** `default-sdd` é opção explícita, nunca default silencioso;
  seleção usa `workflow.select` com autoridade própria e read-back.
- **WZ-05** Resumo final com três dimensões independentes (Configuration valid,
  Runtime ready, Authentication ready) e status separado de Work Item e
  Execution; autenticação não observada limita o objetivo a `partial`.
- **WZ-06** Sem novo validador, store, autoridade, resolver ou transação;
  efeitos confirmados são preservados em falhas posteriores; drift/stale exige
  novo preview; efeito incerto vai para inspeção/recovery do dono.
- **WZ-07** `axiom-runtime` instrui leitura de `diagnostics` e proíbe
  remediação de autenticação para recusas declarativas.
- **WZ-08** Codex e Claude recebem o mesmo conjunto de skills; a revisão
  publicada v0.22.0 é registrada para upgrade de propriedade.

## Invariantes e non-goals

Preservam-se Project ≠ Repository ≠ Runtime ≠ Model Profile ≠ Workflow ≠
Execution; configuração local ≠ política portátil; preview → autoridade exata →
apply → read-back; revisões/digests; autenticação ≠ suporte de modelo.
Non-goals: tornar capabilities opcionais; alterar readiness de schema 3;
novos comandos (`setup-ready`), catálogo de modelos, auth automático,
migração de schema, mudanças em credenciais ou configuração de fornecedor.

## Limitação conhecida

Projects schema 3 (criados sem política de Runtime) mantêm readiness somente
por metadata após edição G-1: `runtime_resolution_blocked /
inventory_unavailable`. Isso é contrato existente; o wizard o relata como
“Runtime readiness não observada”, nunca como pronta. Unificar a semântica com
schema 4 exigiria decisão própria.

## Critérios de aceite

| ID | Verificação determinística |
|---|---|
| A1 | Comando original sem `--capabilities` → `candidate_invalid` + `field_required:--capabilities`, JSON e humano |
| A2 | Mesmo comando + `--capabilities read,repository-write` → `candidate_valid`, preview, apply exato, read-back |
| A3 | Profile sem binding → `binding_missing` com `next` de criação; `modelSupport: unconfirmed` |
| A4 | Modelo desconhecido salvo como intenção; nenhuma alegação de suporte |
| A5 | Nenhuma saída sugere variáveis de autenticação; nenhum processo de fornecedor é invocado |
| A6 | Jornada `govsm` Codex+Claude do zero até `project validate` com structure valid, runtime ready, execution e work-item ready |
| A7 | Project existente incompleto completado via G-1; bloqueios remanescentes relatados com causa correta |
| A8 | Preview obsoleto/autoridade inválida recusados sem efeitos; replay é `stale_preview`; reuso detectado (`binding_exists`/`profile_exists`) |
| A9 | Upgrade da revisão v0.22.0 para Codex e Claude; conteúdo do wizard presente em ambos |
