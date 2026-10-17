# AXM-9 / #277 — Plan mínimo de entrega revisada

Status: **Proposed — aguardando review/aceite humano do Plan**.
Escopo desta revisão: somente PLAN; não contém Tasks nem autoriza implementação.

## Contrato e baseline verificada

Uma jornada: stage integrado → packet com Evidence/review → preview e autoridade
exatos → publicação de PR/artefato → decisão humana → nova Execution de rework.
Fontes: [Issue #277](https://github.com/rgomids/axiom/issues/277),
[WF-011 / AC-011](spec.md#delivery-rejection-and-parent-completion-wf-011) e
[ADR-0020 / HD-003](../../decisions/0020-workflow-definition-revision-binding.md).
Este Plan aplica o contrato aceito, sem alterar Specification ou ADR.

Inspeção de 2026-10-16 (America/Sao_Paulo), main
`731c93e5793b22dae83628670670f71e7bd28977` / v0.17.0, após `git fetch origin`:

- Working tree limpa; `HEAD == origin/main`. Lista completa dos PRs abertos
  (`gh pr list --state open --limit 100`, um resultado) e branches remotas
  (`git ls-remote --heads origin`) sem candidato AXM-9/#277; nenhum Plan #277
  no repositório. Criar um único Draft para este documento; reutilizá-lo depois.
- Resolução de estado em `cmd/lingo/main.go:stateRoot` e
  `internal/local/state_root.go`: o diretório efetivo padrão desta instalação
  não existe. Nenhuma Execution local disponível para continuar aqui. Isso não
  prova ausência em outras máquinas; não reconstruir workflow a partir do GitHub.
- #276 entregue por #320 / v0.17.0; smoke real T40/AC-6 transferido por decisão
  humana para #278/AXM-12. Sua ausência não bloqueia este Plan e não é Evidence
  de execução real. #274 está encerrada e seu binding está presente em main.
- Linear governa operação; GitHub governa contratos/versionamento; Axiom governa
  Execution/Evidence/aceite. Nenhum status/relation do Linear é alterado.

## Matriz dos cinco aceites

Os números seguem a ordem exata dos cinco critérios da Issue. Os testes citados
são evidência de contratos disponíveis por inspeção, não testes executados neste
PLAN nem Evidence operacional da entrega futura.

| AC | Reutilizar / Evidence de código | Lacuna comprovada e delta | Não fazer |
|---|---|---|---|
| 1. Integração produz packet canônico; review/validação ausente bloqueia readiness | `stageexecution.IntegrateService` em `internal/stageexecution/integrate.go`; `WorkflowStore.LoadStageOutcomes` em `internal/local/stage_outcome.go`; ledgers de `workflow/configured.go`. `integrate_test.go` cobre integração, falha de validator, unknown e reparo de projeção. | Receipts/outcomes não agregam packet final, findings e decisão. Compor um packet persistido por digest, validando lineage, revisão e requisitos do workflow retido; readiness derivada de Evidence verificada. | Outro scheduler/store de Evidence; chamar rollup técnico ou `executiongraph.BuildEvidence` (contrato histórico T36) de packet final. |
| 2. PR/push só no envelope aprovado; drift/denial/unknown preservados | Preview/autoridade de integração; `workitem` create-attempt/reconciliation; `githubissues.PrepareMetadataPreview`/`MetadataAuthority.Covers` e testes. | `internal/githubissues/adapter.go` implementa Issue create/read/projection; busca em `internal` não encontrou publicação de PR/push. Adicionar caminho de aplicação de publicação e port mínimo de efeitos, com Git e adapter GitHub concretos, fence durável e reconciliação. | Confundir metadata com PR criado; usar runner histórico de acceptance como implementação; push implícito, force push ou blind retry. |
| 3. CLI/Codex/Claude mostram mesmo artefato/status | `internal/cli/workflow_stage.go`, `cmd/lingo/stage_service.go`, completion canônico; `internal/codexruntime/integration.go` instala o mesmo conjunto de thin skills nos dois Runtimes. | Nenhuma superfície entrega packet/publicação/decisão. Acrescentar operações no domínio Work Item/workflow existente, um DTO de delivery e thin skill `axiom-work-item` compartilhada. | CLI paralela, regra de workflow em prompt ou nova implementação específica por vendor. |
| 4. Rejeição preserva resultado/motivo e permite continuação autorizada | Binding imutável de #274, `workflow.Service.Start`, CAS e `sameExecutionBinding` em `local/workflow_store.go`. | Não há `reworkOf` nem decisão de rejeição. Store usa slot por Repository/Work Item e impede trocar ExecutionID: adicionar retenção endereçável por Execution e troca confirmada do vínculo ativo, com proposta limitada de rework. | Rewind, editar attempts concluídos, herdar autoridade ou escolher R2 silenciosamente. |
| 5. Aceite apenas com autoridade humana explícita; closure/release separados | `workflow.Service.RecordFact` recusa aceite configurado com `delivery_packet_required`; `configuredFactAllowed` e `NextGateAction` mantêm esse gate fechado; lifecycle legado já distingue reviewed/accepted. | Implementar decisão vinculada ao packet exato e leitura/projeção do resultado efetivo; hoje `configuredLifecycle` termina em reviewed e `Terminal.Success` representa transição técnica. | Liberar o fact genérico como atalho; inferir aceite por CI/PR/merge/Issue fechada; fechar Issue/publicar release. |

## Menor mudança vertical proposta

As três partes abaixo expressam dependências causais para implementação futura,
não um catálogo de Tasks. Expor cada operação com JSON/human rendering/help e
skill no mesmo incremento; manter uma única jornada e um único PR aberto.

### 1. Packet e readiness sobre registros existentes

Compor no serviço de workflow uma leitura correlacionada de Project, Repository,
Work Item, Execution/binding, stage ledgers, graph/children/attempts, receipts de
integração e resultados single-stage quando aplicáveis. Usar fontes canônicas
existentes (`WorkflowStore`, `GraphStore`, `SingleStageStore`, receipts), sem
reexecutar trabalho. Incluir árvore/diff integrado, artifacts e validation refs
com digest, findings de review, blockers, decisões e proveniência.

Adicionar apenas o registro companion versionado de delivery que falta, no
`WorkflowStore`, seguindo o precedente content-addressed de `stage_outcome.go` e
publicação protegida de `internal/local/publication.go`. Limitar tamanho/refs,
validar codecs estritos e integrar classificação/inventory/retenção. Packet é
snapshot imutável; mudança de inputs gera outro digest e invalida autoridade
anterior. A criação idempotente do mesmo snapshot retorna a mesma referência.

Readiness exige todos os stages/checkpoints/outputs/validators/reviews requeridos
pelo binding retido. Referência ou digest isolado não prova resultado: verificar
conteúdo, identidade, outcome e vínculo ao stage/attempt correto. Findings
Blocker/Major pendentes bloqueiam, salvo waiver humano com escopo e Evidence
exatos. Review sem resultado interpretável falha fechado; não inventar aprovação.
Packet incompleto permanece inspecionável com blockers/next action, sem publicação.
O packet pode ser preparado ao integrar; a readiness final depende dos gates
restantes, sem autoavançá-los.

### 2. Publicação exata, recuperável e independente do packet

Criar um caso de uso limitado de preview/apply/inspect/reconcile para delivery,
composto em `cmd/lingo`, com regras na aplicação/domínio. O port novo se justifica
pela ausência comprovada de PR publication; não generalizar providers/transports.
Reutilizar política de metadata `PullRequestSurface`; GitHub concreto fica no
adapter `githubissues`, Git local/remoto na fronteira `gitworkspace` existente.

Preview liga Project/Repository/Work Item/Execution/revisão, packet digest, remote
e repository de destino, base/head refs **e commits**, árvore/diff/artifact,
conteúdo do PR e metadata resolvida, efeitos exatos e observação remota. Se a
árvore integrada ainda não tem commit/head publicável, preparar esse artefato
local sob autoridade local e gerar novo preview antes de qualquer efeito remoto.
Revalidar readiness, política, origem, alvo e observações no apply. Autoridade
local não cobre push; push não cobre PR; nenhuma cobre merge/release/aceite.
Um envelope pode autorizar push+PR explicitamente, discriminando ambos os efeitos.
O caminho de artefato local usa referência/digest confirmado sem chamar Provider;
não substitui a implementação do caminho GitHub exigida pelo AC-2.

Persistir fence/identidade de tentativa antes do efeito. Confirmar push pelo ref
remoto exato e PR por repository/base/head/correlação e conteúdo autorizados;
guardar URL/ID/commits/efeitos confirmados no registro de publicação referindo o
packet, sem alterar seus bytes. Read-back confirmado permite no-op/replay do mesmo
pedido. Drift ou negação: zero novo efeito. Ack perdido/timeout após possível
efeito: preservar unknown, `failure`/`recovery_required`; inspect/reconcile por
identidade remota, nunca novo create/push automático. Efeito primário confirmado
com falha secundária: `partial`, refs e próxima ação. Ausência só libera nova
tentativa quando conclusiva e novamente autorizada; múltiplos candidatos exigem
reconciliação. Não adicionar `unknown` a `completion.Status`.

### 3. Decisão e rework preservando história

Persistir decisão companion imutável com actor humano/authority ref, accept ou
reject, motivo/referência, packet digest, Execution/revisão/binding e Evidence.
Solicitação de agente sem autoridade humana verificável não satisfaz o gate.
Accept exige packet final pronto e artefato entregue exato; reject preserva
resultado técnico e efeitos já confirmados, produz requisitos corretivos
acionáveis e termina a entrega como rejected. Replay da mesma decisão é no-op;
decisão contraditória ou packet/revisão stale é recusada.

Manter `State.Terminal` histórico como resultado técnico, sem reinterpretar seus
bytes como aceite. Integrar a decisão canônica na derivação de lifecycle/status e
completion de delivery: pending/rejected nunca projetam accepted ou sucesso final
do workflow. `success` de leitura/registro de rejeição descreve só essa operação.
O fact genérico continua recusando substitutos de packet. Issue closure, release
e projeção Provider não concedem autoridade nem são efeitos destas operações.

Rework preview escolhe explicitamente R1 retida ou R2 selecionada e mostra novo
ExecutionID, `reworkOf` (Execution original + decisão), escopo/Plan corretivo e
inputs anteriores validados por digest. Admissão reutiliza o start/binding de
#274, com nova autoridade de Execution e dos gates posteriores, começando no
primeiro stage exigido. Preparar proposta não inicia Execution.

Resolver a limitação real do slot atual com extensão limitada do `WorkflowStore`:
reter snapshot original por ExecutionID antes de substituir o vínculo ativo;
publicar nova Execution e vínculo sob locks/CAS/protocolo protegido existente,
com read-back. Seletores explícitos continuam lendo original e nova; list/status,
inventory e retenção reconhecem ambas sem duplicação. Interrupção entre retenção,
novo registro e troca de vínculo exige recovery; nenhum start é confirmado até
novo binding e link estarem confirmados. Repetição correlacionada não cria outra
Execution. Não sobrescrever original, transitions, attempts, receipts ou decisão.

## Interfaces, compatibilidade e riscos

Impacto restrito a workflow/delivery, extensão de `WorkflowStore`, fronteiras
Git/GitHub e composição/DTO/help/skill do Work Item. `axiom-workflow` permanece
authoring de definição. Novos nomes/flags devem seguir o catálogo público
existente; este Plan não anuncia comandos ainda inexistentes.

Preservar codecs e corpus execution1/2/3, graphs históricos e resume existentes;
novos companions têm formato próprio e readers estritos. Sem migração inferida,
rebind de R1, mudança de defaults ou limpeza de histórico. A integração com #274
é a admissão e resolução do binding exato, não uma segunda implementação.
Prever retenção de snapshots/companions no downgrade: binário antigo não pode
mutar delivery/rework que não entende; usar envelope de vínculo versionado que
readers antigos recusem nesses alvos, mantendo os snapshots legados intactos. Rollback é
voltar o código preservando registros, sem desfazer efeitos remotos confirmados.

Riscos concretos: mistura de lineage ou review stale (verificação de digest e
origem), concorrência na decisão/link (CAS), PR/push ambíguo (fence/reconcile),
perda de histórico no slot ativo (retenção antes da troca), conteúdo sensível em
packet/PR (refs limitadas e dados sanitizados; sem raw chat/credentials).
Não há nova ADR necessária para aplicar HD-003 e as fronteiras aceitas. Se a
implementação exigir contradizê-las, parar esse delta para decisão explícita.

Preservar Axiom != Lingo, Project != Repository, Role != Model, Execution !=
Agent, Provider != Transport, Integration != MCP, Skill != workflow truth e
Evidence != raw chat. Sem frameworks AXM-29/AXM-31, novo control plane, novo Work
Item, refactor lateral ou delegação recursiva.

## Validação futura e Evidence proporcional

Uma jornada integrada FAKE com branches de falha nos limites alterados basta;
reutilizar `cmd/lingo/stage_journey_blackbox_test.go` e contratos de integração,
publicação local, metadata e skills, sem repetir toda a matriz de #276:

| Grupo | Verificação / Evidence esperada |
|---|---|
| Packet/readiness (AC-1) | Integração confirmada → packet estável; falta de review/validator e finding Major sem waiver bloqueiam. Tamper/lineage stale recusados. Refs/digests/diagnósticos sanitizados ligados ao source SHA. |
| Publicação (AC-2) | Fake Git/Provider registra efeitos exatos; denial/drift = zero efeitos; replay = zero duplicatas; ack perdido → unknown → reconcile do mesmo PR; falha local após efeito = partial. Artefato local não chama Provider. |
| Decisão/rework (AC-4/5) | Aceite exato autorizado; agente/packet stale recusados; rejeição com motivo preserva bytes originais; rework só sob nova autoridade, R1/R2 explícita, nova lineage e gates. Fault no link bloqueia start, replay não duplica. |
| Jornada/superfícies (AC-3 e integração) | CLI JSON/human e mesma thin skill instalada em Codex/Claude retornam mesmos packet/artifact/status/next action; nenhuma regra no prompt. Testar original inspecionável após rework e corpus legado sem alteração. |

Na implementação, executar testes dos pacotes alterados e checks obrigatórios de
repositório/CI; registrar comandos, source SHA, resultados e limites no PR. Ajustar
referência de comandos/skill e documentação de formato/compatibilidade afetadas,
sem churn de roadmap. Nenhum FAKE é Evidence operacional real. PR real, vendor
E2E Codex+Claude e rejeição/continuação reais pertencem a #278/AXM-12 sob envelope
e autorização próprios. Não executá-los nesta etapa.

## Conclusão e gate seguinte

Implementação futura estará tecnicamente concluída quando os cinco ACs tiverem
Evidence determinística correlacionada, jornada pública inspecionável, nenhum
efeito além da autoridade, histórico preservado e CI/review pertinentes aprovados.
Isso não registra aceite humano, fechamento de Issue ou release.

Este PLAN termina com apenas este arquivo revisado, checks de documentação e
segurança pertinentes, commit/push e um único PR Draft com `Related-Issues: #277`,
`Completes-Issues: none` e `Relates to AXM-9`. URL/HEAD/estado remoto e validações
executadas ficam no PR e na devolutiva; não são Evidence funcional.
Próxima etapa: **review/aceite do Plan → Tasks mínimas → implementação futura
separadamente autorizada**. Nenhuma execução dessas etapas é autorizada aqui.
