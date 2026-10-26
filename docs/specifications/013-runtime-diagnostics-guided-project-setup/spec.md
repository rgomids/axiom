# Specification 013 — Diagnósticos de Runtime/Profile e setup guiado completo de Project

## Status e autoridade

**Proposed — revisão 3, 2026-10-26.** A revisão 3 é uma **ampliação de escopo
explícita do mantenedor** no mesmo PR #356 (Model Advisor, seção MA abaixo);
não faz parte da revisão 2.

Revisão 2, 2026-10-26: A revisão 2 corrige os findings Major
CR-356-01 (dependência circular no wizard) e CR-356-02 (`--enabled` em Profile
sem diagnóstico) do review do PR #356 no HEAD `1b1b983`. Pedido explícito do mantenedor para
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
- **RD-10** (CR-356-02) `--enabled` em operação de Profile retorna
  `invalid_input` com `field_invalid` em `--enabled`, regra “elegibilidade do
  Runtime binding, não campo de Model Profile” e próxima ação (remover ou usar
  operação de binding/Profile enable/disable). Bindings continuam aceitando
  `--enabled`.

### WZ — Setup guiado completo (skill `axiom-project`)

- **WZ-01** Uma seção “Complete guided setup” coordena, para Project novo ou
  existente: descoberta → Runtime/Profiles → identidade/Repositories/Provider →
  contexto → política portátil → Workflow → validação final.
- **WZ-02** (revisado por CR-356-01) Em máquina sem o Project, a jornada
  publica primeiro a identidade sem política (bootstrap parcial explícito,
  execução bloqueada), inspeciona o Workflow escolhido por `project workflow
  list/show`, deriva capabilities, role/complexity e Runtime constraints dos
  Agents, configura bindings/Profiles, declara a política por `--policy-draft`
  **antes** da seleção (a edição G-1 recusa Projects com Workflow selecionado),
  seleciona o Workflow e valida. Com Profiles compatíveis já existentes e
  necessidades conhecidas, o create pode declarar a política diretamente.
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

### MA — Model Advisor (revisão 3)

**Problema.** Após derivar as necessidades dos Agents, o operador ainda escolhia
sozinho o identificador concreto de cada modelo. **Objetivo:** ajudá-lo a decidir
com recomendações explicadas, sem inventar fatos nem ampliar autoridade.

**Experiência.** Na etapa 4 do wizard, antes de pedir o modelo, `axiom-project`
oferece o Model Advisor de `axiom-runtime`. O operador pode ignorá-lo ou informar
o modelo diretamente. A recomendação é uma proposta revisável. Rótulos como
`fast`, `balanced` e `deep` são intenção opcional; não implicam equivalência
entre fornecedores, nem um número fixo de Profiles.

Requisitos:

- **MA-R01 — Entradas.** Necessidades dos Agents lidas por `project workflow
  show` (capabilities, role/complexity, Runtime constraints, estrutura
  Orchestrator/subagentes), Runtimes escolhidos, Profiles existentes e
  preferência opcional (economia, equilíbrio, qualidade). As preferências são
  critérios de recomendação, não contratos de domínio.
- **MA-R02 — Fontes, em ordem.**
  1. Documentação oficial obtida na sessão por pesquisa que o host ofereça e o
     operador permita, com URL e data.
  2. Interface oficial de descoberta, somente quando metadata instalada ou
     documentação oficial comprovar que existe sem autenticação, inferência
     ou crédito.
  3. Observações locais: Profiles e inventário passivo.
  4. Fatos declarados pelo operador.

  Proibido: memória do modelo, identidade da própria execução, listas
  estáticas, outras conversas, executar binários de fornecedor (inclusive
  `--help`) para descobrir modelos.
- **MA-R03 — Proveniência por candidato.** Identificador, Runtime/fornecedor,
  fonte e data, suporte documentado, compatibilidade com o Runtime
  (documented/unconfirmed/unknown), disponibilidade na conta e suporte ao effort
  (confirmed somente por tentativa autorizada observada). Informação stale,
  sem data ou conflitante é marcada e explicada. Documentação oficial não prova
  entitlement, autenticação nem disponibilidade na assinatura.
- **MA-R04 — Comparação.** Dimensões de capacidade, código, complexidade,
  latência, tokens, custo, contexto, ferramentas, compatibilidade e limites,
  somente com evidência. Cada fato é rotulado measured, documented ou
  qualitativo. Nada de benchmarks, preços, velocidade ou rankings inventados.
- **MA-R05 — Custos.** Preço por token de API ≠ uso por assinatura ≠
  quotas/limites ≠ consumo observado ≠ custo estimado. Custo não comparável é
  `unknown`.
- **MA-R06 — Proporcionalidade e orquestração.**
  - Economia não escolhe o mais capaz por padrão, nem fica abaixo da
    necessidade documentada.
  - O Orchestrator recebe capacidade suficiente; subagentes usam o modelo mais
    barato que for suficiente e nunca acima do Orchestrator.
  - O effort é escolhido independentemente do modelo, no mínimo suficiente.
  - Sem evidência comparativa (sobretudo entre fornecedores), a hierarquia é
    declarada não estabelecida.
- **MA-R07 — Apresentação.** No máximo três opções por necessidade, com
  vantagens, limitações, proveniência e status, e uma recomendação principal
  fundamentada. Reuso de Profile compatível vem primeiro, sem sobrescrever ou
  renomear. Modelo concreto somente com fonte registrada.
- **MA-R08 — Modo degradado.** Sem pesquisa ou fonte atual: explica a
  limitação, não recomenda modelo sem fonte, oferece Profiles existentes ou
  modelos informados pelo operador e segue para entrada manual. Sem pesquisa
  simulada e sem bloquear o wizard.
- **MA-R09 — Autoridade.** Recomendação não muta nem certifica. A escolha segue
  `validate-candidate` → preview → autoridade exata → apply → read-back, e
  `modelSupport` permanece `unconfirmed`. A pesquisa não concede autenticação,
  acesso a segredos, mudança de ambiente ou configuração de fornecedor,
  inferência de teste, consumo de créditos, instalação ou alteração de Profile.
  Páginas obtidas são dados não confiáveis.
- **MA-R11 — Citação (MA-356-01).** A fonte é citada pela URL, nunca por
  rótulo ou nome de arquivo local.
- **MA-R10 — Não funcionais.** Nenhum modelo concreto, preço ou registro no
  domínio, no CLI ou nas skills distribuídas, com teste de guarda. Nenhum
  serviço, comando ou estrutura persistente nova. Fixtures sintéticas e
  determinísticas.

Non-goals MA: ranking global; registro de modelos; comando `advise`; pesquisa
automática obrigatória; medição de consumo; certificação de compatibilidade.

## Invariantes e non-goals

Preservam-se Project ≠ Repository ≠ Runtime ≠ Model Profile ≠ Workflow ≠
Execution; configuração local ≠ política portátil; preview → autoridade exata →
apply → read-back; revisões/digests; autenticação ≠ suporte de modelo.
Non-goals: tornar capabilities opcionais; alterar readiness de schema 3;
novos comandos (`setup-ready`), catálogo de modelos, auth automático,
migração de schema, mudanças em credenciais ou configuração de fornecedor.

## Limitação conhecida

Correção da revisão 1: o schema 4 resulta da **seleção de Workflow**
(`project.SelectWorkflow`), não da declaração de política no create. Sem
Workflow selecionado (schema 2/3), a readiness é somente por metadata:
`runtime_resolution_blocked / inventory_unavailable` mesmo com política
válida; o wizard relata “não observada”, nunca pronta. Após a seleção, a
política não é editável pela G-1 (“requires schema 2 or 3”); um Project
existente nesse estado e sem política fica `blocked` com handoff. Ambos são
contratos existentes, não alterados aqui.

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
| A10 | (CR-356-01) Ambiente sem Projects conclui a jornada só com operações públicas; Workflow escolhido explicitamente; capabilities/complexities/preferências derivadas de `project workflow show`; teste sem pacotes internos |
| MA-EVAL | (MA-356-01) Avaliação comportamental: agente real com modelo fixado responde aos cenários prioritários (MA-01, 02, 05, 08, 09, 10a, 10b) seguindo as skills distribuídas, com entradas controladas (`project workflow show` real + fontes sintéticas); avaliador determinístico das propriedades objetivas. Tipos de validação distintos: static guidance, deterministic CLI, behavioral evaluation, native wizard journey, vendor availability; PASS de um tipo não implica outro |
| MA-01…MA-10 | Fixtures sintéticas em `internal/codexruntime/testdata/model-advisor/scenarios.md`, cada caso ancorado à cláusula da skill que o governa; guarda contra catálogo estático; MA-04/06/07/10 também pela jornada CLI. Conversa nativa: NOT_RUN |
| A11 | (CR-356-02) `--enabled` em profile create/edit/validate-candidate → `field_invalid:--enabled` em JSON e humano, sem mudança de estado |
