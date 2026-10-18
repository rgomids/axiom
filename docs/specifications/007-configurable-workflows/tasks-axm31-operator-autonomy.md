# AXM-31 — Tasks da experiência e autonomia da IA operadora

**Proposed — tasks, 2026-10-17.** Oito Tasks futuras, nenhuma iniciada ou aceita
por este artifact. Sem código de produto/teste nesta etapa. Implementação exige
aceite humano destas Tasks e autorização delimitada própria.

**Accepted — anotação factual, 2026-10-17.** O operador aceitou T01–T08
na revisão `10cb320df7ebbf727590acee9f091c51b54a2853`, conforme
[decisão humana no Linear](https://linear.app/rgomids/issue/AXM-31/revisar-autonomia-escopo-e-responsabilidade-da-ia-operadora-no-axiom#comment-56d8406a)
e [registro no PR #325](https://github.com/rgomids/axiom/pull/325#issuecomment-6243413855).
É aceite das Tasks, não review nativo do GitHub ou aceite da implementação.
O pedido humano IMPLEMENT T01–T08 autoriza código/testes, Evidence, commits e
atualização da mesma branch/PR, respeitando S1→S2→S3 e os limites aprovados.
Texto histórico Proposed e requisitos abaixo preservados; sem autorização de
inferência/Provider de produto, merge, release ou fechamento de Work Item.

## Entrada, autoridade e preflight

- [AXM-31](https://linear.app/rgomids/issue/AXM-31): o pedido humano desta etapa
  autoriza somente Tasks, anotação factual de aceite do Plan, verificações
  documentais/segurança, commit, push e atualização do mesmo
  [Draft PR #325](https://github.com/rgomids/axiom/pull/325).
- [Plan aprovado exato](https://github.com/rgomids/axiom/blob/315ea2c422a9b7eba941f6e41a66019e1ff4e392/docs/specifications/007-configurable-workflows/plan-axm31-operator-autonomy.md):
  SHA `315ea2c422a9b7eba941f6e41a66019e1ff4e392`, blob
  `96b5fb383a74197baa58f2fae44c5521431ed2f7`.
  [Decisão registrada no Linear](https://linear.app/rgomids/issue/AXM-31/revisar-autonomia-escopo-e-responsabilidade-da-ia-operadora-no-axiom#comment-23f4e188):
  aprovação humana em conversa, não review/aprovação nativa do GitHub.
  CI e self-review não são aceite. A única alteração do Plan nesta etapa é sua
  anotação factual Accepted; D1–D4 e o restante dos bytes são preservados.
- [Spec aceita](amendment-axm31-operator-autonomy.md), revisão
  `2e9295293c84a51523df50b28258149debb6badf`, blob
  `b4a9605cdb4886193795ad8089c35fe4be41e62b`, por
  [aceite humano explícito](https://github.com/rgomids/axiom/pull/325#issuecomment-6241938642).
  Preservar seus bytes históricos Proposed. A cobertura integral é AO-01–13 e
  P1–P8/N1–N10, inclusive o adendo conversacional, conforme Plan aceito.
- Contratos reutilizados: [Constituição](../../product/constitution.md),
  [Spec 007](spec.md), [ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md),
  [ADR-0025](../../decisions/0025-public-stage-dispatch-and-capability-proof.md),
  [política Linear/GitHub](../../development/linear-github-issue-tracking.md) e
  [SDD](../../../.agents/skills/axiom-sdd/SKILL.md).
- Leitura autenticada revalidou head local/remoto/PR no SHA do Plan, árvore
  limpa, branch `docs/axm-31-operator-autonomy-specification`, PR aberto Draft.
  Lista completa de PRs abertos: #325 e #317 (landing page, alheio ao escopo).
  Não há segundo PR AXM-31 nos resultados. Nenhuma mudança substantiva de Plan
  foi observada antes das edições.
- AXM-31 relido: Todo, sem blockedBy, blocks AXM-12, related AXM-8/AXM-9.
  Comentários retornaram sem paginação restante: gate READY_FOR_tasks registra
  etapa nova/ainda não iniciada; prompt Tasks não é Execution. Não há registro
  acessível indicando execução ativa concorrente de Tasks. Busca local achou
  Spec/Plan AXM-31, sem Tasks anteriores; Tasks #276/#277/#303 são de outros
  owners e foram consultadas para evitar duplicação.
- `axiom` não encontrado no PATH; raiz efetiva de estado, resolvida por inspeção
  de `stateRoot`/`NativeStateRoot`, ausente neste host. Nenhum store criado,
  comando de produto executado ou estado reconstruído. Execution canônica
  **desconhecida**, inclusive em outros hosts; gate/current/target stage não
  inferidos de Todo/PR. Antes de futura implementação, revalidar Execution,
  concorrência, heads, aceite das Tasks e autoridade da unidade.
- Shell/Git, GitHub e Linear de leitura disponíveis. Sem delegação: não há
  comprovação de modelo estritamente inferior ao Orchestrator; inspeções
  determinísticas bounded são suficientes. Nenhuma notificação/automação de
  Runtime foi disparada.

## Classificação, ownership e limites comuns

| Classe | Fato ou pendência | Tratamento |
|---|---|---|
| confirmed | Completion, renderização, Project edit, autoria/seleção, retry/reconcile, três skills e testes existem nos paths abaixo | Reusar contratos e testes; alterar apenas diagnóstico/orientação e assertions dos gaps |
| confirmed | Mapper de falha carece de opções estruturadas; skills não explicitam todos os budgets/checkpoints e a jornada contínua | Delta AXM-31 delimitado por T01–T05 |
| hypothesis | Qualidade da conversa nativa, cumprimento de budgets por LLM, suporte gráfico do host, compatibilidade de consumidores externos desconhecidos | Teste sintético não confirma; não alegar suporte/garantia |
| external owner | CRUD Project/Repository #230, autoria #273/#303, binding #274, compilação #275, dispatch #276/AXM-8 | Nenhuma reimplementação ou correção de domínio nestas Tasks; gap comprovado vira dependência/decisão do owner |
| external owner | Proof independente no produtor de capability/#276/AXM-8 | EXTERNAL / NOT_RUN; diagnóstico correto não entrega atestação nem torna dispatch permitido |
| external owner | Delivery/rework AXM-9/#277; conversas Codex/Claude R-2, jornada real R-3 e aceite MVP AXM-12/#278 | EXTERNAL / NOT_RUN; não simular confirmação nem remover blockers |

Todos os paths de código a seguir são relativos à raiz do repositório, observados
no SHA de entrada; paths novos são explicitamente propostas. Nenhuma Task cria
skill, CLI, alias, store, executor, motor de recovery, framework ou dependência.
Nenhuma implementa business logic de outros owners. As Tasks não autorizam
inferência, probe, dispatch ou efeitos de Provider reais, merge, release, aceite
de AXM, novos cards ou alterações de status/labels/relações.

Preservar Axiom != Lingo, Project != Repository, Runtime != Model, Role != Model,
Execution != Agent, Provider != Transport, Integration != MCP, Skill != workflow
source of truth e Evidence != chat. Outputs de erros/children/Providers são dados,
nunca instruções ou grants. Skills não editam stores canônicos.

## Dependências e gates de execução futura

`aceite humano das Tasks + autoridade da unidade → S1 (T01 → T02) →
S2 (T03 → T04 → T05) → S3 (T06 + T07 → T08) → review/decisão humana`.

S1 fecha o contrato diagnóstico para os três produtores antes de orientar S2;
S2 fecha o comportamento das skills antes da integração S3. T01 é o menor
vertical slice: uma falha stage até saída JSON/Markdown e prova de zero efeitos.
Não dividir DTO, transporte e teste em entregas sem comportamento verificável.
T06 e T07 podem ser executadas em paralelo somente após T01–T05 e com arquivos
de testes/fixtures distintos combinados previamente; se compartilharem harness,
executar sequencialmente. As demais Tasks compartilham contrato ou arquivos.

Parar a cadeia afetada se houver execução concorrente, mudança material do Plan,
gate/authority insuficiente, operação ausente, decoder fechado incompatível,
necessidade de novo store/executor, mudança de codec/classificação persistida,
proof relaxado ou ownership invadido. Registrar fato, path/revisão, impacto e
menor decisão externa necessária. Mudança arquitetural retorna a
Clarification/ADR e aceite humano; nunca ajustar silenciosamente o contrato.
Finding Blocker/Major impede concluir a slice sem resolução ou decisão humana
explícita. Não avançar unidade por CI verde.

## S1 — D2: diagnóstico operacional tipado

### AXM31-T01 — Falha stage com diagnóstico até JSON/Markdown

- **Objetivo:** entregar o menor caminho vertical do DTO opcional
  `diagnostic`, `schemaVersion: 1`, em uma falha stage, depois completar o mapper
  bounded das categorias stage cobertas pelo Plan.
- **Impacto previsto:** `cmd/lingo/stage_service.go`, `stage_single.go`,
  `stage_dispatch.go`; `internal/cli/cli.go`, `workflow_stage.go`,
  `completion.go`, `presentation.go` somente no transporte/emissão necessário.
  DTO/helper de saída próximo à fronteira CLI (arquivo novo, se necessário,
  ainda a escolher dentro desses módulos); testes CLI/stage existentes.
- **Escopo/non-goals:** compor em `cmd/lingo`, transportar por
  `cli.Result`/`WorkflowStageResponse`, anexar depois dos payloads existentes
  sem reenvelopar single/Graph/preview. Manter nomes, ordem relativa, status,
  categoria, provenance/references/next, exit codes e `details` scalar limitado;
  não alterar `completion.Result`, `completion.Status`, reports/codec persistido
  ou classificação. Sem chamada de leitura adicional, mutação, probe, inferência,
  dispatch ou atestação para construir diagnóstico: somente observações já
  obtidas e erros tipados. Ausência de enriquecimento preserva recusa.
- **Contrato:** incluir operation/category/target/observedRevision conhecidos;
  observed/notObserved/hypotheses/missingPrerequisites separados;
  capability/authority/confinement independentes; confirmedEffects/uncertainEffects;
  verification por leitura suportada e fato esperado. Não inventar refs.
  Até três opções ordenadas por menor efeito, com ID, disponibilidade,
  pré-requisitos, efeitos, authority requerida, operação suportada, args
  conhecidos/faltantes, finalidade e verificação. Opção indisponível não recebe
  comando. Coleções/textos bounded cabem no evento de 1 MiB; não perder recusa
  quando diagnóstico faltar. Unknown permanece unknown.
- **Gate:** aceite destas Tasks + autorização T01; confirmar consumidores/goldens
  antes de estabilizar DTO. Decoder fechado afetado exige decisão explícita de
  contrato/versão, sem fallback permissivo.
- **Done verificável:** mesma falha em JSON/Markdown conserva cada valor canônico;
  fixtures com ausente/expirada/drift/indisponível não promovem capability.
  Orphan vivo/incerto oferece esperar/inspecionar/handoff de stop sob identidade
  e autoridade próprias; não cria `kill` nem stop universal. Pre-effect,
  partial e unknown mantêm seus efeitos conhecidos/incertos.
- **Validação/Evidence futura:** ampliar `internal/cli/completion_structure_test.go`,
  `presentation_test.go`, `workflow_stage_test.go`, goldens em
  `internal/cli/testdata/presentation` e testes stage de `cmd/lingo`.
  Rodar `go test ./internal/cli ./cmd/lingo -count=1` em harness sintético;
  registrar diff de JSON/Markdown, limites/ordem, snapshots/digests antes/depois,
  spies de probe/dispatch/mutação com zero chamadas na montagem. Unknown/partial
  não podem ser apresentados como zero efeitos anteriores.
- **Cobertura:** AO-03, AO-04, AO-05, AO-08; P3/N3, P5/N5, N7 (diagnóstico;
  integração de recovery em T07).

### AXM31-T02 — Reusar diagnóstico em Project edit e Workflow authoring

- **Objetivo:** aplicar o mesmo DTO às falhas delimitadas de J1, sem criar
  novo erro de domínio para upsert válido.
- **Impacto previsto:** `cmd/lingo/main.go:editFailure`,
  `cmd/lingo/workflow_authoring.go:workflowAuthoringCompletion`,
  `internal/cli/cli.go`, `workflow_authoring.go` e composição de
  `cli.Result` se necessária; testes configure/edit e autoria existentes.
- **Escopo/non-goals:** mapear erro tipado e observações disponíveis em falhas
  de edição/autoria/seleção; não acrescentar DTO a reports/codec de domínio,
  validação, CRUD, binding ou seleção. Se categoria não existir, identidade
  estável pelo erro tipado, sem fabricar status. Colisão de intenção sem erro
  CLI é decisão da skill T03, não recusa inventada.
- **Gate:** T01 com contrato/compatibilidade validados + autorização T02.
- **Done verificável:** falhas pre-effect e partial/unknown mantêm preview,
  validações, referências e efeitos confirmados/incertos; argumentos ausentes
  explícitos. Sem diagnóstico universal de toda CLI nem efeito de enriquecimento.
- **Validação/Evidence futura:** ampliar `cmd/lingo/configure_edit_test.go`,
  `configure_edit_publish_test.go`, `workflow_authoring_test.go` e equivalentes
  em `internal/cli`; reutilizar `internal/projectapp/edit_test.go`,
  `edit_publish_test.go`, `workflow_authoring_test.go` como regressão do domínio.
  `go test ./cmd/lingo ./internal/cli ./internal/projectapp -count=1`;
  JSON/Markdown comparados, snapshots de stores e contadores de efeitos extras
  iguais a zero. Incluir preview obsoleto e read-back divergente sem replay.
- **Cobertura:** AO-02, AO-03, AO-05, AO-12; P2/N2, P8/N10, N7.

## S2 — D1/D3/D4: orientação e continuidade nas skills existentes

### AXM31-T03 — Entrada dupla e jornada Project → Repository → Workflow

- **Objetivo:** orientar J1/J2 com revalidação de referências e autoridade
  idêntica para intenção natural e operação explícita.
- **Impacto previsto:** `internal/codexruntime/skills/axiom-project/SKILL.md`,
  `axiom-workflow/SKILL.md`, `axiom-work-item/SKILL.md` sob o mesmo diretório;
  `internal/cli/domain_skill_routing_test.go`, `skill_catalog_contract_test.go`,
  testes de conteúdo de `internal/codexruntime`. Catálogo/parser são contratos
  de comparação; sem nova rota/flag, alias ou skill.
- **Escopo/non-goals:** descoberta real; preservar token explícito; `skill inspect`
  encerra o turno em leitura, sem coleta de args/execução posterior. Referência
  observada na conversa é candidata, resolvida inequivocamente e relida antes
  de agir; chat/CWD/remote não criam binding/default/grant. Perguntar só dado
  material ausente; leitura inequívoca sem nova approval.
- **Jornada:** list/show/validate Project e associações; colisão de key/slug no
  escopo correto interrompe intenção de adicionar antes de upsert. Alternativa
  é proposta, nunca rename/replacement escolhido pela IA. Distinguir atualização
  legítima. Validar/preview → consentimento exato → apply → read-back; drift
  exige novo preview; leitura divergente bloqueia, não reaplica. Consultar
  activeWorkflow none/unresolvable sem fallback; definição exata; built-in
  imutável exige cópia `--from-default` com chave escolhida, preview/authority
  próprios. Draft regular no scratch autorizado, validar/edit/read-back;
  seleção é intenção/preview/authority separados. Não criar pré-validação attach.
- **Gate:** S1/T02 concluída + autorização T03; operação ausente vai ao owner.
- **Done verificável:** roteamento natural/explícito usa mesmos args e controles;
  inspect-only e skill/operação/argumento ausentes recusam sem efeito. Contratos
  de autoria, Project selection e Work Item execution não se confundem.
- **Validação/Evidence futura:** ampliar testes de roteamento acima e
  `internal/codexruntime/workflow_skill_content_test.go`; rodar
  `go test ./internal/cli ./internal/codexruntime -count=1`.
  Evidence: tabelas comparadas com inspect/parser, traces sintéticos de turnos,
  decisão material e leitura de revisão. Prova de jornada completa fica em T06.
- **Cobertura:** AO-01, AO-02, AO-05, AO-10, AO-11, AO-12;
  P1/N1, P2/N2, P7/N9, P8/N10, N8.

### AXM31-T04 — Budgets, remediação autorizada e handoff cumulativo

- **Objetivo:** codificar D3 nas três skills, com foco de Execution em
  `axiom-work-item`, sem implementar enforcement ou recovery paralelo.
- **Impacto previsto:** as três skills de T03 e testes de conteúdo/contrato
  existentes em `internal/codexruntime` e `internal/cli`.
- **Escopo/non-goals:** AO-06 permite no máximo duas repetições após a inicial
  em 60 s desde a inicial, respeitando limite menor, para operação comprovadamente
  replay-safe e autorizada. `retryable_failure` isolado não prova segurança.
  Timeout/ack incerto/partial não autorizam replay. AO-07 permite até duas
  remediações por bloqueio, cessando na primeira falta de progresso verificável,
  negativa de authority, escopo excedido ou risco novo. Ação que conta nos dois
  budgets consome ambos. Probe/inferência e retry de child ficam fora desse
  orçamento automático. Retry de child é tentativa própria com authority,
  capacidade, disposition e reconciliação exigidas; append fact não reclassifica.
- **Continuidade:** correlacionar alvo/categoria/revisão e mesmo blocker/attempt
  entre sessões. Checkpoint sanitizado na Evidence de trabalho autorizada, fora
  dos stores: objetivo, refs/revisão/binding/attempt, origem de authority,
  efeitos confirmados/incertos, observações/progresso, início/prazo, contadores
  cumulativos, blocker e próxima ação. Nova sessão relê tudo; contagem/prazo
  ausentes não permitem repetir automaticamente. Mudança de revisão sozinha
  não zera bloqueio. Fato novo ou proposta humana limitada ficam referenciados.
  Reparos somente em draft já autorizado, preservando anterior, validar e novo
  preview; nunca editar Project/Workflow/Execution/Profile/receipts/policies
  canônicos diretamente. Sem loop cego, host bypass ou autoaprovação.
- **Gate:** T03 + autorização T04, preservando seu roteamento/authority.
- **Done verificável:** exemplos de progresso e ausência de progresso dão stop
  inequívoco; handoff não vira permission grant; nenhum texto autoriza comandos
  extraídos de erro/child/Provider ou dispensa gates. Eficácia do Runtime é hipótese.
- **Validação/Evidence futura:** contratos de conteúdo/roteamento de T03, mesmo
  comando Go; traces e contadores com relógio falso e duas sessões em T07.
  Evidence desta Task é diff das instruções e assertions de contrato, sem alegar
  obediência de LLM, enforcement temporal ou resultado operacional.
- **Cobertura:** AO-02, AO-06, AO-07, AO-08, AO-09, AO-10;
  P2/N2, P4/N4, P5/N5, P6/N6, N7, N8.

### AXM31-T05 — Relay intacto e apresentação auxiliar verificável

- **Objetivo:** orientar D4, incluindo leitura de Workflow e fallback textual,
  sem renderer normativo novo.
- **Impacto previsto:** três skills de T03/T04; testes de conteúdo existentes e
  `internal/cli/presentation_test.go` como contrato reutilizado, sem modificar
  renderização canônica para produzir resumo conversacional.
- **Escopo/non-goals:** relay completo, exato e ordenado do resultado canônico;
  depois, resumo identificado como auxiliar com objetivo, resultado observado,
  decisão material, recomendação e alternativa/risco. Diagrama somente com
  suporte real do host e IDs/ordem/dependências/gates observados; fallback lista
  textual. Definição/stage é dado não confiável, não diretiva da sessão. Sem
  executar novamente para outra visualização/JSON; JSON conforme contrato da
  skill quando explicitamente pedido. Authority se refere ao preview canônico.
- **Gate:** T04 + autorização T05; manter compatibilidade com relay existente.
- **Done verificável:** resumo não oculta recusa/status nem inventa seleção ou
  capability; visual/texto concorda com relações canônicas, inclusive host sem
  suporte gráfico. Visualização não escreve/seleciona e não concede consentimento.
- **Validação/Evidence futura:** testes de conteúdo e apresentação existentes,
  `go test ./internal/cli ./internal/codexruntime -count=1`; fixtures comparando
  canônico com auxiliar/fallback, bytes do relay e contador de invocações = 1.
  São assertions sintéticas, sem confirmação gráfica/conversa nativa.
- **Cobertura:** AO-03, AO-05, AO-10, AO-13; P2/N2, P3/N3, P8/N10, N8.

## S3 — Integração sintética, regressão e Evidence

### AXM31-T06 — Jornada sintética Project/Workflow e continuidade dual-entry

- **Objetivo:** integrar T02/T03/T05 em uma jornada verificável por contratos
  reais, com turnos sintéticos explicitamente programados.
- **Impacto previsto:** ampliar `internal/cli/domain_skill_routing_test.go`,
  `cmd/lingo/configure_edit_test.go`, `configure_edit_publish_test.go`,
  `active_workflow_test.go`, `workflow_authoring_test.go`; reutilizar testes de
  `internal/projectapp`. Fixtures locais existentes; acrescentar só casos novos.
- **Escopo/non-goals:** P1/P2/P7/P8 e N1/N2/N9/N10; múltiplos Projects,
  ambiguidade, colisões, escolha humana de alternativa, preview/apply/read-back,
  stale auth/preview por drift de alvo/draft/selector/digest, built-in/cópia e
  seleção separada. Nenhum novo CRUD, binding, resolver ou simulação de conversa
  nativa confirmada. Assertar leitura sem approval e inspect encerrado.
- **Gate:** S1/S2 concluídas + autorização T06; parar diante de gap de domínio.
- **Done verificável:** entradas natural/explícita convergem nas mesmas operações,
  efeitos e recusas. Zero mutações/dispatch/probe nas negativas; zero rename/upsert
  unilateral; built-in/seleção e revisões anteriores preservados. Reobservação
  entre turnos e após apply demonstrada; read-back divergente nunca declara êxito.
- **Validação/Evidence futura:** `go test ./internal/cli ./internal/projectapp
  ./cmd/lingo -count=1` (em uma linha); traces sintéticos com operações/args,
  snapshots/digests, previews/authority e efeitos contados. Comparar JSON/Markdown
  do mesmo caso e auxiliar/fallback; não repetir operação real para obter ambos.
- **Cobertura:** AO-01, AO-02, AO-03, AO-05, AO-10, AO-11, AO-12, AO-13;
  P1/N1, P2/N2, P7/N9, P8/N10, N8 (conteúdo de definição adversarial).

### AXM31-T07 — Diagnóstico/recovery single e Graph, budgets e segurança

- **Objetivo:** integrar T01/T04 com falhas/recovery existentes e demonstrar os
  limites num harness sintético, sem motor novo ou inferência real.
- **Impacto previsto:** ampliar `cmd/lingo/stage_journey_blackbox_test.go`,
  `stage_adversarial_blackbox_test.go`, `stage_dispatch_test.go`,
  `stage_fake_runtime_test.go` e fixtures de teste; reusar testes de
  `internal/stageexecution`, `internal/capabilityproof`, `internal/local`.
  Traces/relógio falso de operador são fixtures/test helpers, não runtime de produto.
- **Escopo/non-goals:** P3–P6/N3–N8 nas topologias single e Graph; proof
  ausente/expirada/drift/produtor indisponível, lease alheia, liveness incerta,
  partial/unknown, timeout/ack perdido, replay de receipt e retry/reconcile.
  Preservar classificação/attempt/facts append-only e contratos de owners.
  Fake child/probe é fixture rotulada, nunca prova operacional independente.
- **Budgets:** relógio falso na inicial, nas duas repetições permitidas, no
  prazo vencido e no limite menor; terceira repetição proibida; duas remediações
  com progresso e terceira proibida; primeira sem progresso encerra. Revalidar
  authority antes de ação; ação com ambos budgets incrementa ambos. Duas sessões
  acumulam contadores/prazo do mesmo blocker/attempt; checkpoint ausente/obsoleto
  ou revisão nova sem fato novo não reinicia loop. Handoff referencia nova proposta
  humana/fato quando existir, sem fabricar grants.
- **Segurança:** canários sintéticos de injeção/secrets em erro, diagnóstico,
  definição, child e Provider; nenhum comando arbitrário, segredo público,
  autoaprovação de gate/Evidence/proof ou host bypass. Testar recusa sem efeitos,
  unknown/partial com efeitos anteriores preservados, sem replay cego/kill.
- **Gate:** S1/S2 concluídas + autorização T07; T06 paralela somente sob o
  isolamento descrito. Nenhum efeito real de Provider/inferência autorizado.
- **Done verificável:** diagnósticos equivalentes JSON/Markdown e opções bounded;
  contadores mostram zero efeitos indevidos (não zero efeitos prévios em partial).
  Retry somente sob authority/disposition/capacidade; append fact conserva status.
  Snapshot e histórico/receipt não duplicam dispatch. Teste de sequência demonstra
  contrato do harness, não garante que um LLM seguirá as instruções.
- **Validação/Evidence futura:** `go test ./cmd/lingo ./internal/stageexecution
  ./internal/capabilityproof ./internal/local -count=1` (em uma linha).
  Registrar topologia/caso, relógio/contadores, snapshots/digests, IDs de facts,
  bytes de saída sanitizados e contagem de chamadas/mutações; jamais stdout de auth,
  paths privados ou transcripts. Regredir contratos de single/Graph existentes.
- **Cobertura:** AO-03, AO-04, AO-05, AO-06, AO-07, AO-08, AO-09, AO-10;
  P3/N3, P4/N4, P5/N5, P6/N6, N7, N8.

### AXM31-T08 — Install/upgrade das skills e pacote integrado de Evidence

- **Objetivo:** verificar entrega conjunta código/skills sem mudar inventário,
  receipts históricos ou políticas; consolidar a matriz sem converter sintético
  em aceite operacional.
- **Impacto previsto:** testes `internal/codexruntime/domain_skills_upgrade_test.go`,
  `workflow_skill_content_test.go`, `shared_history_test.go`,
  `internal/install/domain_skills_upgrade_test.go`, `skill_receipt_test.go`.
  Metadados/digests em `internal/codexruntime/manifest.go` somente se exigidos
  pelo mecanismo vigente de atualização, preservando histórico publicado.
  Evidence futura em documento irmão proposto
  `evidence-axm31-operator-autonomy.md`; nenhum script durável novo previsto.
- **Escopo/non-goals:** fresh install, reinstall/no-op, upgrade das três skills
  nos roots isolados Codex/Claude pelos serviços existentes; conferir bytes,
  digests, ownership, foreign/modified content fail-closed e recibos. Isso não é
  conversa nativa nem ampliação de renderer/Runtime. Reconciliar somente docs
  afetadas pelo comportamento efetivamente entregue sob futura autoridade.
- **Gate:** T06 e T07 concluídas + autorização T08; todos os AO/cenários com
  Evidence ou gap explícito, sem Blocker/Major não resolvido/decidido.
- **Done verificável:** as três skills atuais são as instaladas; nenhuma nova
  rota/alias; histórico/receipts preservados; regressões locais e CI do SHA exato
  reportados. Publicação/release continua gate externo, não ação desta Task.
- **Validação/Evidence futura:** `go test ./internal/codexruntime ./internal/install
  ./internal/cli ./cmd/lingo -count=1` (em uma linha), validator de repositório,
  staged sensitive-files e scanner dedicado se disponível. Reusar suites completas
  quando requerido pelo impacto/CI; evitar repetir checks já PASS sem nova mudança.
  Pacote integrado registra SHA/binário/skill-digest, caso, classificação synthetic,
  comando realmente executado, PASS/FAIL/SKIP/NOT_RUN, efeitos, snapshots/digests,
  authority utilizada/pendente, riscos e próximo gate. CI real por URL/SHA.
  Conversas nativas, proof independente e aceite MVP: **EXTERNAL / NOT_RUN**.
- **Cobertura:** AO-01–13 pela integração/receipts e consolidação da matriz
  inteira P1–P8/N1–N10, sem duplicar implementações T01–T07.

## Matriz de rastreabilidade e testes futuros

T08 consolida todas as linhas; não substitui o teste primário abaixo. Todos os
cenários comportamentais estão **NOT_RUN nesta etapa Tasks**. IDs remetem à
Spec aceita; as Tasks detalham as assertions, sem redefinir P/N.

| AO | Decisão/slice | Tasks primárias | Cenários |
|---|---|---|---|
| AO-01 | D1/S2–S3 | T03, T06 | P1/N1, P7/N9 |
| AO-02 | D1/D3/S1–S3 | T02, T03, T04, T06 | P1/N1, P2/N2, P8/N10 |
| AO-03 | D2/D4/S1–S3 | T01, T02, T05, T06, T07 | P3/N3, N7, P8/N10 |
| AO-04 | D2/S1/S3 | T01, T07 | P3/N3 |
| AO-05 | D2/D4/S1–S3 | T01, T02, T03, T05, T06, T07 | P2/N2, P3/N3, P8/N10 |
| AO-06 | D3/S2–S3 | T04, T07 | P4/N4 |
| AO-07 | D3/S2–S3 | T04, T07 | P4/N4 |
| AO-08 | D2/D3/S1–S3 | T01, T04, T07 | P5/N5, N7 |
| AO-09 | D3/S2–S3 | T04, T07 | P6/N6, N8 |
| AO-10 | D1/D3/S2–S3 | T03, T04, T05, T06, T07 | N8 |
| AO-11 | D1/S2–S3 | T03, T06 | P7/N9 |
| AO-12 | D1/S1–S3 | T02, T03, T06 | P8/N10 |
| AO-13 | D4/S2–S3 | T05, T06 | P8/N10 |

| Cenário | Tasks / base reutilizada | Assertion e Evidence mínima |
|---|---|---|
| P1 / N1 | T03/T06; routing/Project tests | Read sem approval quando inequívoco; múltiplos alvos pedem somente ID faltante; args e zero mutações |
| P2 / N2 | T02–T06; edit/publish/autoria | Draft autorizado/anterior preservado, validar/novo preview; scope/authority/stale preview recusam sem escrita; snapshots/digest/read-back |
| P3 / N3 | T01/T05/T07; stage/proof/presentation | Ausente/expirada/drift/indisponível separados; login/consentimento/child claim/registry vazio não provam capability; zero probe/dispatch nas negativas e na montagem; JSON/Markdown |
| P4 / N4 | T04/T07; fixtures de sequência | Relógio falso, ≤2 repetições/60 s, menor limite, ≤2 remediações, stop sem progresso; timeout/ack incerto impede replay; contadores cumulativos |
| P5 / N5 | T01/T04/T07; stage journey/adversarial | Single/Graph: reconcile append-only/status intacto, retry próprio com disposition/authority/capacidade; vivo/incerto/lease/drift sem sinalização ou dispatch indevido; facts/snapshots |
| P6 / N6 | T04/T07; sequência sintética multi-sessão | Releitura de refs/binding/attempt/authority, checkpoint e budgets acumulados; ausente/obsoleto não reinicia; handoff sanitizado |
| N7 | T01/T02/T04/T07; partial/recovery tests | Efeitos confirmados/incertos preservados, zero replay cego/duplicação; publicação/rework encaminhados AXM-9/AXM-12; receipt/digests |
| N8 | T03–T07; adversarial/presentation | Canários em erro/definição/child/Provider; zero comando host, leak público, grant/proof/gate/aceite fabricados; fixture sanitizada e contador de efeitos |
| P7 / N9 | T03/T06; routing/catalog/inspect | Natural/explícito equivalentes, nova intenção no próximo turno; inspect termina; skill/operação/args ausentes sem efeitos; trace |
| P8 / N10 | T02/T03/T05/T06; edit/autoria/activeWorkflow | Colisão/ambiguidade sem rename/upsert unilateral, read-back/stale preview, built-in intacto/cópia/seleção separada; relay e diagrama/fallback concordam com canônico; digests |

## Validação desta etapa e próximo gate

Checks documentais/segurança e CI do commit Tasks são registrados no mesmo PR;
não são execução dos testes futuros acima. Revisão semântica do Orchestrator deve
conferir cobertura integral, granularidade, dependências, paths, limites/owners e
ausência de alteração de produto. Não é review independente nem aceite humano.
Nenhuma nova automação durável; se futuramente indispensável, registrar em
`scripts/automation-registry.json` sob política própria antes da validação.

Antes de `implement`: humano revisa/aceita a revisão exata destas Tasks e autoriza
a unidade; executor revalida gate/Execution/concorrência/authority. Esta entrega
não aceita Tasks, não conclui AXM-31 e não remove AXM-31 → AXM-12. Proof independente,
AXM-9 e AXM-12 mantêm seus gates. Parar ao concluir `tasks`.
