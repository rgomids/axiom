# AXM-32 — Tasks de separação de saída

**Proposed — tasks, 2026-10-18.** Oito Tasks futuras, nenhuma iniciada,
concluída ou aceita por este documento. Seu aceite humano no SHA exato e
uma autorização delimitada de implementação são gates separados.

## Entrada e autoridade

- [Plan aprovado exato](https://github.com/rgomids/axiom/blob/962adfdcd22384c681613810b78022469ee813cb/docs/specifications/008-output-separation/plan.md),
  SHA `962adfdcd22384c681613810b78022469ee813cb`, aceito no
  [GitHub](https://github.com/rgomids/axiom/pull/328#issuecomment-6253809891) e
  [Linear](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia#comment-42cacc85),
  após [review rodada 4](https://github.com/rgomids/axiom/pull/328#pullrequestreview-5564396763).
  O texto histórico Proposed do [Plan](plan.md) é preservado; o aceite posterior
  autoriza somente esta elaboração de Tasks, sem alterar o contrato aprovado.
- [Specification 008](spec.md), R1–R4/AC1–AC9, aceita no SHA `18d29a6`;
  [ADR-0030 Accepted](../../decisions/0030-canonical-results-terminal-and-host-presentation.md);
  [Clarify](clarify.md) e decisão posterior R1 são entradas obrigatórias.
  Inventário suficiente para planejar não prova cobertura universal/dinâmica.
- Escopo documental atual: Tasks, Evidence e reconciliação mínima do índice,
  validação, commit, push na branch `docs/axm-32-output-separation-specification`
  e atualização do mesmo [PR #328](https://github.com/rgomids/axiom/pull/328),
  OPEN/Draft. Sem código, testes de produto, skills de produção ou Runtime real.
  [Evidence desta etapa](evidence.md#tasks-preparation--2026-10-18) registra preflight
  e limites. Linear/PR/CI não são Execution canônica; esta permanece desconhecida.
- Antes de futura implementação: revalidar head/concorrência, aceite das Tasks,
  autoridade da unidade e, se houver operação de workflow, verdade local canônica,
  revisão, gates e prerequisites. Não criar/reconstruir Execution de histórico
  remoto nem avançar gate por inferência. Ausência ou contradição exige handoff.

IDs `AXM32-T01`–`AXM32-T08` são estáveis e locais a este artifact; não criam
Work Items de Provider. Os caminhos de código abaixo existem no baseline do
Plan, salvo adições de fixtures/oráculos explicitamente propostas. Responsabilidade
é por capacidade (CLI, skills, distribuição, validação), sem fixar pessoa, modelo,
ferramenta ou nova Runtime. Não há delegação nesta elaboração: hierarquia de
modelos estritamente inferior não foi comprovada; não se alega review independente.

## Contratos e gates comuns

D1: sem flag e com `--human`, CLI humana nativa em TTY **e** pipe/redirecionamento;
`--json` explícito para automação/skills. Sem auto-JSON em não-TTY. Começar com
texto estático simples; ANSI somente com suporte positivo, nunca em pipes ou
capacidade desconhecida. `NO_COLOR` presente, inclusive vazio, suprime cor.
Sem spinner, movimento de cursor, significado só por cor ou truncamento de IDs.

D2: substituir Markdown/fallback cercado da CLI e relay literal/Auxiliary nas
quatro skills `axiom-project`, `axiom-workflow`, `axiom-work-item`, `axiom-runtime`
no mesmo conjunto coordenado para Codex e Claude. Não publicar fase isolada com
par incompatível, nem adiar a substituição para limpeza posterior. Não reabrir
aceite histórico #232/AXM-11/AXM-31.

D3: preservar DTOs JSON tipados, nomes/tipos/omissões/enums/status/limites,
identidade por invocação e targets tipados, diagnóstico `schemaVersion: 1`,
refs/revisions/digests/provenance. Sem envelope/campo/schema novo, migração de
estado/domínio, extração tipada de `result`/`next` ou árvore autoritativa paralela.
O renderer projeta o mesmo evento; o host consome JSON, nunca texto do terminal.
Zero segunda invocação de use case ou inferência dedicada a formatar.

Completion não é readiness, efeito nem aceite: `success` → 0; `interrupted` → 2;
`failure`, `validation_failure`, `denied_authority`, `partial`,
`retryable_failure` → 1. No-op é resultado da operação; recovery é condição,
sem novos enums. Escrita falha pode retornar 1 após efeito confirmado, sem rollback
implícito. Preservar todos os blockers/warnings materiais e seus escopos, efeitos
confirmados/incertos, autoridade, provenance qualificada e detalhes recuperáveis.
Dirty/unknown/revisão divergente material à confiança deve aparecer.

Preview propõe efeitos; consentimento vincula target/preview/digest/revision exatos,
com read-back canônico e nova autorização diante de drift. Resumo, assentimento
conversacional ou exit code não substituem autoridade/observação. JSON ausente,
malformado, truncado, incompatível, sem status, com target errado ou contraditório
para ações dependentes: explicitar incerteza, refs conhecidas e menor inspeção
segura. Não repetir mutação para obter outra apresentação, ampliar retry/budgets,
invocar Provider/Runtime ou inventar operação disponível, inclusive #305.

G1 (`provider_create_ambiguous` perdido em caminho sem payload) e G2 (ledger
limitado a 64 efeitos) permanecem riscos condicionais em observação, sem Task
corretiva preventiva, schema, migração adicional ou novos Work Items. Não declarar
ledger truncado completo nem fazer replay cego de efeito incerto. Se impacto real
impedir o contrato, parar a cadeia afetada e registrar cenário/reprodução bounded,
revisão, fato perdido, consumers, efeitos/risco e menor decisão específica requerida.
Solicitar decisão humana antes de correção/novo item; não dispensar R1/ACs nem
transformar a observação em gate preventivo. Incidentes não foram reproduzidos aqui.

Conflito material com Plan/Spec/ADR, redaction insuficiente, consumidor incompatível,
root insegura ou autoridade ausente também exige Evidence/handoff, sem ajuste
silencioso da decisão. Achado blocker/major não resolvido impede concluir a unidade.
Sem #305, AXM-12/#278 E2E, novo executor/framework/dependência, merge ou release.

## Sequência e paralelismo

`aceite humano das Tasks + autoridade delimitada → T01 → T02 →
(T03 + T04) → T05 → T06 → T07 → [autorização nativa própria] → T08 → review humano`.

| Fase do Plan | Tasks / dependências reais | Gate de saída futura |
|---|---|---|
| 1 — baseline e entradas | T01; só após gate de entrada | Consumers nomeados, fatos/oráculos e fronteira externa desconhecida |
| 2 — slice vertical | T02 depende de T01 | `project validate` JSON/CLI/skill Project prova R4 e canais |
| 3 — substituição completa | T03 e T04 dependem de T02; T05 depende de ambas | Emissores + quatro skills e equivalência/adversarial fechados |
| 4 — distribuição/rollback | T06 depende de T05 | Pares compatíveis, proteção de upgrade e restauração provados |
| 5 — reconciliação/Evidence | T07 depende de T06; T08 depende de T07 e autorização nativa | Smoke isolado e observações nativas distintos; review/decisão pendentes |

T03/T04 podem trabalhar em paralelo apenas com ownership de arquivos acordado:
CLI/snapshots em T03, skills/content em T04; routing/harness compartilhado deve ser
integrado sequencialmente. T05 integra seus resultados; planejar casos de teste
pode antecipar trabalho, mas não aprova gate sem executar contra o conjunto integrado.
Não há outro paralelismo de entrega entre fases: cada uma consome Evidence da
anterior. Nenhuma Task autoriza publicação parcial do binário/skills.

Cada Task futura registra em [Evidence](evidence.md): ID, revisão binária e de
fixtures/skills, digests quando aplicáveis, invocação/target, comando/teste realmente
executado, exit/status, obrigações esperadas/observadas, efeitos/autoridade,
PASS/FAIL/NOT_RUN e limite. Não persistir raw chat/log, segredo ou dado privado.
Nomes de novos testes são escolhidos na implementação e vinculados ao ID/AC;
os testes existentes abaixo são pontos de extensão, não prova já executada.

## AXM32-T01 — Congelar consumers e obrigações verificáveis

- **Objetivo:** fechar a entrada concreta da migração e o oracle tipado antes de
  trocar defaults; fase 1, R1–R4 e preparação de AC1–AC9.
- **Dependência:** gate de entrada; nenhum código de apresentação antes do baseline.
- **Escopo/impacto:** estender somente variantes tocadas do inventário Clarify;
  mapear produtores → DTO/emissor → consumers, invocation/target, fatos disponíveis,
  missing/unknown, limites e tratamento de compatibilidade. Incluir common/parser,
  Project/list/bootstrap/install/validate/configure/edit/operational/Integration,
  Work Item/lifecycle, Execution/projection/list, workflow authoring/stage/plan/
  delivery, Runtime authoring/resolution/auth/capability, context e maintenance
  transfer/cleanup/recovery/retirement/upgrade. Classificar separadamente help,
  version, skill-inspect e POC legada, sem forçar schema uniforme.
- **Impacto de repositório previsto:** inventário/Evidence e entradas de fixtures
  propostas próximas a `internal/cli/testdata/presentation`; ler goldens JSON,
  `presentation_test.go`, `completion_test.go`, `domain_skill_routing_test.go`,
  `cmd/lingo/skill_protocol_test.go`, installed-skill checks, exemplos de comandos,
  scripts e release packaging. Preservar snapshots JSON/estado e pins publicados.
- **Conclusão verificável:** cada consumer encontrado tem seleção atual, dependência
  de texto humano e tratamento previsto; automação usa/passa a usar JSON explícito
  na unidade que a altera. Registrar consumers externos desconhecidos sem alegar
  exaustividade. Separar o limite próprio de 64KiB de
  `scripts/smoke-prepared-artifact.py` dos budgets de cada emissor e de G2.
  Congelar entradas para todas as linhas da matriz semântica da Spec e AC5/AC6,
  com obrigações por campo tipado/invocação (não por prose matching de `result`).
- **Evidence/risco/gate:** tabela nomeada de consumers/fixtures, hashes baseline,
  fatos covered/gap/unknown e plano de comparação. Oracle ainda é test-only,
  sem schema de produto. Gaps materiais retornam à decisão; não inferir cobertura
  universal da simples presença de um payload `any`.

## AXM32-T02 — Provar Project validate de ponta a ponta

- **Objetivo:** menor slice read-only, do resultado existente a JSON, CLI nativa e
  consumo da skill Project em roots isoladas de ambos hosts; fase 2, R1–R4,
  AC1–AC6, sem publicar esse subconjunto.
- **Dependência:** T01. Responsabilidade: fronteira CLI com integração Project.
- **Escopo/impacto:** `internal/cli/{presentation,readiness,cli}.go`,
  `cmd/lingo/{main,project_lifecycle}.go` apenas na fronteira necessária;
  `internal/codexruntime/skills/axiom-project/SKILL.md` e fixtures/testes de
  `presentation_test.go`, `bootstrap_readiness_test.go`, `operator_journey_test.go`.
  Introduzir só helpers locais de apresentação e contexto de capacidade injetável;
  domínio/readiness não detectam terminal nem mudam algoritmo/status.
- **Conclusão verificável:** evento `success`/0, structure valid, effective partial,
  Work Item ready, Execution bloqueada por `runtime_policy_unavailable` conserva
  **todos** esses fatos em CLI e conversa. Frase de política só com suporte tipado;
  `runtime_resolution_blocked` sozinho não explica causa. Cobrir também valid/ready,
  valid/blocked, estrutura inválida, readiness ausente, blocker não-policy e múltiplos
  blockers, sem chamar todo Project inválido/pronto ou toda operação bloqueada.
  Fixtures default/`--human`/`--json`, TTY/pipe e `NO_COLOR` provam D1; JSON parseável
  único/status/exits inalterados e contadores falsos provam uma chamada de serviço.
  Skill consome JSON com entrada natural/explicit skill, sem pedido de raw JSON.
- **Evidence/risco/gate:** snapshots nativos, JSON baseline e comparações tipadas
  de conversa, refs/digests exatos, traces sintéticos sem efeito real; resultado
  é integração isolada, não observação nativa ou reprodução do incidente pessoal.
  T02 só sai com exemplo R4 e variantes demonstrados, sem recodificar domínio.

## AXM32-T03 — Completar a CLI nativa e os canais

- **Objetivo:** fase 3 CLI, R1/R2/R4, AC1/AC2/AC5/AC6; toda variante canônica do
  inventário apresenta fatos materiais sem Markdown gerado/fallback JSON cercado.
- **Dependência:** T02; pode coexistir com T04 nos limites acima.
- **Escopo/impacto:** `internal/cli/presentation.go`, `completion.go`, `cli.go` e
  emissores específicos inventariados; `cmd/lingo/main.go` para capability/channel,
  fixtures `testdata/presentation`, `completion_structure_test.go`,
  `completion_test.go`, parser/help e `cmd/lingo/{blackbox,hierarchical_help_blackbox}_test.go`.
  Projetar membros tipados do mesmo evento, sem serviço de formatação genérico,
  refactor de domínio ou árvore paralela de resultados autoritativos.
- **Conclusão verificável:** cada variante tem snapshot humano determinístico,
  obrigações materiais completas e JSON compatível; unknown payload/fato requerido
  é explícito. Sem headings/emphasis/fences/tables gerados pelo renderer; distinguir
  dados hostis literais de sintaxe gerada. Repetir entrada/capacidade conserva bytes.
  Cobrir matriz semântica inteira da Spec: sucesso/efeito/no-op, partial com efeito
  confirmado, failure/validation_failure, denied_authority, interrupted/incerto,
  retryable_failure, recovery, readiness bloqueada e transporte indisponível.
  Não suprimir provenance material, blocker secundário ou efeito após erro de escrita.
- **Canais/compatibilidade:** default e human equivalentes em TTY/pipe/redirect;
  JSON só explícito, stdout com documento único sem banner/prompt/ANSI, stderr
  separado; flags globais leading, recusas conflitantes/repetidas/mal posicionadas
  e help humano determinístico preservados. `Run` programático continua prependendo
  JSON, distinto de `RunInteractive`/executável. Testar capacidade ANSI ausente ou
  desconhecida, `NO_COLOR` inclusive vazio, largura estreita sem cortar tokens,
  budgets, short/broken writes com failure e zero reexecução. Não generalizar
  garantia MVP a writer POC sem classificação/Evidence específica.
- **Evidence/risco/gate:** goldens JSON preservados, snapshots nativos novos,
  contadores e matriz de canais/limites com status/exits. Automação inventariada que
  depende do default migra para JSON explícito; consumers externos seguem risco.
  Failures de output não significam que efeitos foram desfeitos.

## AXM32-T04 — Completar as quatro skills JSON → conversa

- **Objetivo:** fase 3 host, R1/R3/R4, AC3/AC4/AC7; síntese normal do host fiel,
  sem literal relay/Auxiliary obrigatório ou chamada extra de formatação.
- **Dependência:** T02; sincronizar arquivos compartilhados com T03.
- **Escopo/impacto:** quatro fontes compartilhadas em
  `internal/codexruntime/skills/{axiom-project,axiom-workflow,axiom-work-item,axiom-runtime}/SKILL.md`;
  `internal/codexruntime/{runtime_test,workflow_skill_content_test}.go`,
  `internal/cli/domain_skill_routing_test.go`, `cmd/lingo/skill_protocol_test.go`.
  Não criar fonte Claude separada; preservar domain ownership e limites AXM-31.
- **Conclusão verificável:** todas solicitam leading `--json` internamente nas
  entradas natural e explícita, sem pedido humano de raw JSON. Pedido explícito
  de JSON recebe canônico sem reescrita. Resposta normal segue objetivo → resultado
  efetivo → blockers/risco/decisão materiais → próxima ação, sem headings vazios,
  dump obrigatório, payload tree ou footer genérico de provenance. Não é byte relay.
  Cada blocker, warning material, efeito confirmado/incerto e qualificação de
  confiança permanece; agrupar warnings só preservando escopo/consequência.
  Refs/revisions/digests acionáveis copiados exatamente dos dados, nunca abreviados
  ou reconstruídos de prosa; detalhes/provenance canônicos continuam recuperáveis.
- **Autoridade:** preview/consent/digest/revision/read-back inalterados; distinguir
  observed/notObserved/hypothesis, completion/readiness/aceite, no-op/partial/recovery,
  budgets e opções disponíveis. JSON inválido/ausente/truncado/unsupported, status
  ausente, target divergente ou observações contraditórias interrompem ações
  dependentes com handoff bounded, sem replay/invenção de capacidade ou de causa.
  Sem segunda chamada LLM, formatter agent, Runtime hop ou autoridade nova.
- **Evidence/risco/gate:** assertions de conteúdo e jornadas isoladas das quatro
  skills nos dois roots com fake tool results e traces; amostras concisas positivas
  e negativas. Strings corretas não provam fidelidade nativa; T05/T08 permanecem.

## AXM32-T05 — Integrar equivalência, consentimento e segurança

- **Objetivo:** saída integrada da fase 3, AC1–AC8/R1–R4: verificar que a troca de
  apresentação preserva fatos/autoridade e não causa efeito adicional.
- **Dependência:** T03 e T04 integradas; owner de validação da fronteira.
- **Escopo/impacto:** estender oráculos/fixtures de T01 e suites existentes de
  CLI/completion/readiness, work-item lifecycle/digest,
  `cmd/lingo/{operator_diagnostic,operator_budget,stage_adversarial_blackbox}_test.go`
  e harness isolado de skill-tool. Novas fixtures/claim ledger são apenas de teste.
- **Conclusão verificável:** para **cada linha** da matriz semântica, comparar
  JSON tipado+invocação com linhas/tokens nativos e conversa candidata. Obrigações:
  completion, readiness por operação, todos os blockers/warnings/efeitos materiais
  com certeza, autoridade, refs/revisions/digests e provenance qualificada; lista
  explícita de unsupported/unknown, que nunca concede permissão.
  Claim ledger test-only liga alegações a fatos fonte; checks também rejeitam
  prosa contraditória, não apenas ledger correto. Negativos obrigatórios falham
  ao omitir segundo blocker, inventar sucesso, atribuir policy sem suporte, alterar
  digest ou esconder efeito partial. Não exigir bytes naturais idênticos nem usar
  julgamento LLM como único oracle: a garantia é bounded aos cenários conhecidos.
- **Consentimento/efeitos:** missing authority, assentimento a resumo, drift de
  digest/revision, read-back, no-op, denied, partial, interrupted/recovery e
  acknowledgement incerto; counters demonstram zero grant/replay/efeito adicional
  por apresentação e nenhum retry além do orçamento/autoridade existentes.
- **Segurança:** canários sintéticos de segredo são redigidos antes da serialização
  JSON/detail/Evidence em todas as superfícies, indicando dado retido sem recuperá-lo.
  ANSI/control/bidi escapados; hostile Markdown, workflow/Provider instructions e
  labels/refs maliciosos são dados, sem execução/seguimento ou mudança de escopo.
  Conservar IDs/digests autorizados exatos; nunca confundir credencial com token
  acionável. Assert zero canário vazado/efeito involuntário, confinamento/retenção
  de detail artifacts e Evidence sanitizada. Gap real de redaction exige handoff.
- **Evidence/risco/gate:** testes nomeados/negativos, status/exits, snapshots,
  expected/observed obligations e traces sem Provider/Runtime real. G1/G2 não
  recebem reparo preventivo nem são chamados resolvidos; aplica-se gate condicional.
  Só seguir a distribuição com equivalência integrada e achados materiais resolvidos.

## AXM32-T06 — Proteger upgrade coordenado e rollback

- **Objetivo:** fase 4, AC9 e compatibilidade AC1/AC3/AC6: manter conjunto binário
  + quatro skills compatível em Codex/Claude, sem perda de ownership/estado.
- **Dependência:** T05; não liberar subconjuntos ainda que testes locais passem.
- **Escopo/impacto:** `internal/codexruntime/{manifest,runtime,integration,upgrade}.go`,
  `manifest.go::sharedSkillHistory`, `integration.go::NewClaude`, fixtures e testes
  `domain_skills_upgrade_test.go`, `shared_history_test.go`,
  `published_receipts_test.go`, `upgrade_test.go`; snapshots de estado existentes
  apenas como entrada imutável, sem migração. Reusar inspect/install protegidos.
- **Conclusão verificável:** registrar revisão substituída e reconhecer bytes
  publicados exatos por fixtures/receipts append-only, sem reescrever pins antigos.
  Conjunto novo distinguível por versionamento existente; avaliar necessidade de
  bump de `BinaryCompatibility` do protocolo de apresentação com Evidence da
  matriz, sem bump JSON/schema. Alocação numérica pertence à review de implementação.
  Testar old/old e new/new; new binário + old skills exige upgrade reconhecido
  antes de depender do novo contrato; old binário + new skills recusa/aguarda par
  compatível, mesmo que JSON parseie. Investigar enforcement antes de alegar
  fail-closed. Validar quatro digests/receipts nos dois roots, no-op reinstall,
  upgrade parcial/interrompido e preservação de modified/foreign/unsafe roots.
- **Rollback verificável:** restaurar par anterior explicitamente compatível,
  binário e quatro skills/receipts em cada host, preservando estado operacional
  e modificações do usuário. Exercitar restauração isolada; se installer antigo
  não reconhecer bytes novos, parar e comprovar procedimento autorizado de
  restauração, sem overwrite forçado, receipt forjado, apagar estado ou replay.
  Manter caminho de automação JSON compatível conhecido durante upgrade/rollback.
- **Evidence/risco/gate:** matriz old/new por host e skill-set/digests, snapshots
  antes/depois, proteção de conflitos, upgrade e restauração. Pins/recibos históricos
  e corpus `internal/compatibility/testdata/stable-v1/snapshots` inalterados;
  somente novas fixtures de histórico quando necessárias. Mixed revisions não
  podem ser chamados compatíveis sem prova; impedimento material retorna à decisão.

## AXM32-T07 — Reconciliar referências e verificar artefato preparado

- **Objetivo:** primeira parte da fase 5, AC1–AC9 consolidadas: documentação e
  smoke isolado representam o conjunto preparado, sem alegar release/aceite.
- **Dependência:** T06. Reconciliar somente docs afetadas sob autoridade futura.
- **Escopo/impacto:** `docs/commands.md`, exemplos/help, referências prospectivas
  #232/AXM-31 e Evidence; `scripts/test-codex-skills.sh`,
  `scripts/smoke-prepared-artifact.py` e consumidores inventariados quando necessário.
  Preservar história datada e decisões aceitas; sem atualizar ADR por conveniência.
  Automação durável alterada deve manter registro em `scripts/automation-registry.json`.
- **Conclusão verificável:** comandos/examples refletem humano nativo versus JSON
  explícito; referências prospectivas conciliam o contrato sem reabrir aceites.
  Rodar suites determinísticas apropriadas de CLI, `cmd/lingo`, codexruntime e
  compatibilidade; prepared-artifact smoke em roots isoladas dos dois hosts,
  com Providers/Runtimes falsos, canais/pipelines e conjunto instalado verificado.
  Não usar smoke como prova de conversa nativa. Checar limite próprio de 64KiB,
  sem afrouxar budgets para ocultar perda. Checks documentais, links, diff e secrets
  reais, com comandos/exits e escopo; CI sempre associado ao SHA observado.
- **Evidence/risco/gate:** mapa AC→testes nomeados/resultados, SHA binário, quatro
  digests por host, revisão do artefato/fixtures, estado preservado e riscos externos.
  Separar completion técnica, readiness observada, compatibilidade e aceite humano.
  Só conjunto compatível comprovado é candidato a futura review de release;
  nenhuma publicação, merge ou aceite AXM-12 segue desta Task.

## AXM32-T08 — Observar conversa nativa e entregar para review

- **Objetivo:** segunda parte da fase 5, AC3/AC4/AC5/AC7/AC8: complementar o oracle
  bounded com observações nativas Codex **e** Claude, sem inferir suporte de outro host.
- **Dependência:** T07 e autorização humana própria para observações nativas;
  esta etapa Tasks não a concede. Verificar interfaces nativas instaladas nesse
  gate; não inventar comando/flag/API ou fixar modelo novo para formatação.
- **Escopo/impacto:** somente fixtures isoladas e Evidence sanitizada, Projects/roots
  sintéticos, fake Provider/Runtime effects; sem mutação externa/dispatch de produto.
  Reusar o binário e os quatro digests comprovados, sem trocar conjunto entre provas.
- **Conclusão verificável:** em cada host observar sucesso ordinário, validação
  bloqueada, consentimento negado e recovery interrompido/partial; registrar
  expected/observed fatos, tokens e tool-call trace. Demonstrar JSON interno,
  síntese normal do próprio host, todos os efeitos/blockers materiais e nenhuma
  chamada dedicada de formatação, Runtime hop ou ação dependente sem autoridade.
  Incluir distinção success/0 vs Execution bloqueada e ausência de causa inventada.
- **Evidence/risco/gate:** comparação bounded sanitizada por caso/host, comando
  realmente usado, SHA/digests, status/exits, fatos e limites, sem raw chat.
  Falta de host/interface/authority = NOT_RUN, nunca PASS sintético atribuído a
  nativo. Divergência material exige correção autorizada e nova validação afetada.
  Entregar review técnico/segurança com findings e riscos; solicitar decisão humana
  sobre entrega, sem declarar readiness de produto/aceite, merge ou release.

## Matriz de rastreabilidade e conclusão

| Requisito / AC | Tasks responsáveis | Evidência de conclusão futura |
|---|---|---|
| R1 — canônico estável | T01–T06; T07 consolida | Inventário, JSON baseline, uma observação/chamada, typed oracle e limits |
| R2 — CLI nativa | T02/T03/T05/T07 | Snapshots determinísticos, canais, zero Markdown gerado/efeito extra |
| R3 — host Markdown | T02/T04/T05/T06/T08 | Quatro skills/ambos hosts, JSON interno, claims exatos e traces |
| R4 — completion/readiness/efeitos | T02–T05/T08 | Status/exits conservados, matriz readiness e nenhuma promoção de aceite |
| AC1 | T01/T02/T03/T05/T07 | Documento JSON único compatível, contadores, limites/escrita, status/exits |
| AC2 | T01/T02/T03/T05 | Cada linha semântica, snapshots/negativos e fatos tipados materiais |
| AC3 | T02/T04/T06/T07/T08 | Natural/explícita, quatro skills nos dois roots; nativo distinto de sintético |
| AC4 | T01/T02/T04/T05/T08 | Oracle/claim ledger + prosa, negativos e observações sem formatter extra |
| AC5 | T02/T03/T05/T08 | Sete variantes readiness, causa policy condicional e success/0 bloqueado |
| AC6 | T01/T02/T03/T06/T07 | Flags/help/canais/NO_COLOR/width/writes, pipelines e conjunto compatível |
| AC7 | T04/T05/T08 | Preview/consent/drift/read-back e counters de ausência de replay/grant |
| AC8 | T03/T04/T05/T08 | Redaction, ANSI/bidi/injection/refs e canários sem efeito/vazamento |
| AC9 | T01/T06/T07 | Bytes publicados, matriz old/new, upgrade/rollback e snapshots imutáveis |

Na etapa documental atual, **toda validação comportamental acima é NOT_RUN**.
Conclusão técnica futura exige gates/evidências das oito Tasks, sem esconder
NOT_RUN nativo, gaps materiais ou consumidores desconhecidos; não equivale a
readiness de todas as operações, aceite humano, AXM-12 E2E ou publicação.
Constituição I–VIII: intent/Plan aceitos precedem ação; dependências e Evidence
rastreáveis; checks determinísticos antes de julgamento; autoridade/redaction
preservadas; sem stack/contrato especulativo ou mudança de Project/Repository.
Não se identificou conflito material com Plan/ADR; self-review documental não é
review independente nem aceite das Tasks.

**Próximo gate:** review e aprovação humana específica deste `tasks.md` no SHA
publicado, seguidas de autorização delimitada antes de `implement`. Parar aqui.
