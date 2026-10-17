# AXM-9 / #277 — Tasks de entrega revisada

Status: **Proposed — aguardando review das Tasks e autorização separada de implement**.
Nenhuma Task abaixo foi executada; os cinco ACs continuam não implementados.

## Baseline e autoridade

Derivado exclusivamente do [Plan aceito](plan-277-reviewed-delivery.md) no commit
`9f23e6ab697d24656f43ad20f35fad294af116ad`, blob
`855484c87289ebe6c5794572eef8249e00325b34`.
O status Proposed dentro daquele documento é histórico: o
[aceite humano posterior](https://github.com/rgomids/axiom/pull/323#issuecomment-6227335396),
após [review técnico](https://github.com/rgomids/axiom/pull/323#pullrequestreview-5555050259),
autoriza TASKS, não implementação, merge ou aceite funcional. O usuário também
identifica o registro Linear `2edff942-b665-41ff-b84f-1a74d8360236` no
[AXM-9](https://linear.app/rgomids/issue/AXM-9); não foi consultado/importado nesta etapa.
Contrato: [Issue #277](https://github.com/rgomids/axiom/issues/277),
[WF-011 / AC-011](spec.md#delivery-rejection-and-parent-completion-wf-011),
[ADR-0020 / HD-003](../../decisions/0020-workflow-definition-revision-binding.md),
[amendment de superfície](amendment-303-workflow-skill.md) e
[ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md).

Na inspeção desta etapa, HEAD local e remoto do único PR AXM-9 aberto,
[#323](https://github.com/rgomids/axiom/pull/323), continuam no SHA aceito;
PR aberto e Draft, branch `docs/axm-9-delivery-plan`. `main` remoto permanece
`731c93e5793b22dae83628670670f71e7bd28977` / v0.17.0, sem divergência material.
Não havia artifact de Tasks #277. Plan/spec/ADRs ficam intactos.
#276/AXM-8 está entregue; AC-6/T40 real pertence a #278/AXM-12 e não bloqueia
estas Tasks. GitHub versiona o contrato, Linear governa backlog/status, Axiom
governa Execution/Evidence; esta sessão não altera Linear nem cria Execution.

## Ordem mínima e contratos inspecionados

**T1 → T2 → T3**, três incrementos verticais no mesmo PR. T2 consome o packet
imutável de T1; T3 consome packet e referência de entrega confirmada de T2 para
accept, preservando também publicação partial/unknown em reject. Não há task
separada de camada, migração, skill, testes ou integração. Cada incremento inclui
CLI/DTO/help/skill, testes e documentação que altera. A sequência compartilha
persistência/DTO, por isso não se justifica fan-out ou delegação nesta etapa curta.

Contratos comprovados no SHA aceito, a reconfirmar pelo implementador antes de
fixar novos nomes/assinaturas:

| Fronteira existente | Reutilização e lacuna relevante |
|---|---|
| `internal/stageexecution/integrate.go`: `IntegrateService.Preview/Apply`, `IntegrateStore`, `StageOutcomeLedger`; `cmd/lingo/stage_service.go`: `stageStores`, `stageContract`, `localIntegrateStore` | Integração, receipts, graph/records e projeção de outcomes existem; não são o packet final. |
| `internal/local/stage_outcome.go`: `WorkflowStore.RecordStageOutcome/LoadStageOutcomes`; `internal/workflow/configured.go`: `StageLedger`, `ValidatorResult`, `WorkflowBinding` | Outcomes endereçados por Execution, ledgers e binding retido alimentam T1; não criar outro store de Evidence. |
| `internal/local/workflow_store.go`: `Create/Save/Load/List`, `sameExecutionBinding`; `internal/local/publication.go`: `publishFile`, `PublicationError` | Um slot por Repository/Work Item; Save exige CAS e impede trocar ExecutionID/binding. Companions usam publicação protegida; T3 precisa retenção e vínculo ativo explícitos. |
| `internal/workflow/service.go`: `Service.Start/RecordFact`; `configured.go`: `configuredLifecycle/configuredFactAllowed`; `gates.go`: `NextGateAction` | Reutilizar admissão #274; Start encontra o slot existente. Fact genérico nega `delivery_packet_required`, lifecycle configurado termina em reviewed: T3 integra decisão sem atalho. |
| `internal/workitem/service.go`: `CreateAttempt`, `resumeCreate/reconcileAttempt`; `internal/githubissues/metadata.go`: `PrepareMetadataPreview`, `MetadataAuthority.Covers`; `internal/githubissues/adapter.go` | Precedentes de fence/reconciliação e política `PullRequestSurface`; adapter cria Issues, não implementa publicação de PR. T2 adiciona só o port de efeito ausente. |
| `internal/gitworkspace/git.go`: `Manager`; `internal/cli/workflow_stage.go`: `WorkflowStageInput/Response`, catálogo; `internal/completion/completion.go`: `Result`, `Facts`, `Classify`; `internal/codexruntime/integration.go` | Reutilizar fronteira Git, composição e completion. Codex/Claude compartilham instalação de thin skills; nenhuma superfície delivery existe ainda. Não acrescentar `unknown` ao enum de completion. |

Os nomes de novas operações, codecs e ports serão definidos na implementação
contra esse código, sem tratar esta decomposição como API já disponível.

## T1 — Produzir e inspecionar packet com readiness verificável

**Objetivo / ACs:** integração confirmada produz snapshot canônico inspecionável,
com blockers/next action quando incompleto (AC-1); CLI e ambas as thin surfaces
expõem as mesmas referências e readiness (parte de AC-3).

**Inputs/dependências:** baseline #274/#276, binding retido, ledgers de stages,
GraphStore/SingleStageStore, children/attempts, receipts de integração e review /
validation refs existentes. Não reexecutar trabalho nem inferir review pelo chat.

**Escopo e resultado observável:** compor no workflow uma correlação de
Project/Repository/Work Item/Execution/revisão/binding, árvore/diff, artifacts,
validators/reviews/findings, decisões/waivers, blockers e proveniência. Verificar
conteúdo, digest, resultado e vínculo ao stage/attempt, inclusive single-stage
quando aplicável. O snapshot é imutável e idempotente; inputs alterados geram outro
digest. Readiness exige todos os stages/checkpoints/outputs/reviews/validators do
binding; review não interpretável falha fechado. Blocker/Major pendente só admite
waiver humano verificável do escopo e Evidence exatos. Packet incompleto continua
legível, sem publicação ou avanço automático de gates.

**Delta de persistência e repositório:** companion versionado no WorkflowStore
porque outcomes/receipts existentes não guardam agregação final/readiness;
reutilizar content addressing e protocolo protegido, com codec estrito, limites de
refs/bytes, inventário/classificação/retenção. Alterações limitadas a workflow,
local, composição em `cmd/lingo`, DTO/rendering/catálogo em `internal/cli` e
`internal/codexruntime/skills/axiom-work-item/SKILL.md`. Documentar apenas comandos,
formato e contratos de routing afetados; atualizar pin/history da skill conforme
os testes existentes. Não copiar Evidence bruta para o companion.

**Conclusão / testes determinísticos / Evidence E1:** estender contratos de
integração e store existentes para confirmar packet estável após integração,
replay sem duplicação, falta de review/validator, finding Major sem waiver,
waiver exato válido e waiver de outro escopo/actor recusado, tamper e binding /
lineage stale recusados. Verificar persistência/read-back e falha protegida no
novo registro; JSON/human/skill mostram o mesmo digest, blockers e next action.
E1 = resultados desses testes, refs/digests sanitizados correlacionados ao source
SHA e inspeção de packet incompleto/pronto. O incremento termina consumível por T2,
com os checks de compatibilidade aplicáveis aos novos companions passando.

**Segurança / antiobjetivos:** nenhuma chamada Provider, push, Runtime real ou
aceite. Refs limitadas e dados sanitizados; Evidence != raw chat. Não criar
scheduler, segundo store de Evidence, rollup paralelo ou prompt com regras.

## T2 — Entregar packet sob envelope exato e reconciliar efeitos

**Objetivo / ACs:** publicar PR ou artifact local com efeitos exatos confirmados;
denial, drift e unknown nunca viram sucesso nem blind retry (AC-2), mantendo a
mesma referência de artifact/status nas três superfícies (parte de AC-3).

**Inputs/dependências:** T1 pronto e digest corrente, política de metadata
resolvida, alvo Git/Provider explícito, base/head e commits observados. Packet
incompleto pode ser inspecionado, mas não autoriza apply.

**Escopo e resultado observável:** caso de uso limitado preview/apply/inspect /
reconcile no domínio/aplicação, composto em `cmd/lingo`. Envelope vincula
Project/Repository/Work Item/Execution/revisão/binding, packet, remote/repository,
base/head refs e commits, árvore/diff/artifact, conteúdo/metadata do PR,
observação remota e cada efeito. Se falta commit publicável, prepará-lo sob
autoridade local e emitir novo preview antes de efeito remoto. Apply revalida
readiness, política, digest e observações antes dos efeitos autorizados.

Persistir fence/correlação antes de push/create; confirmar ref remoto exato e PR
por repository/base/head/correlação/conteúdo. Guardar ID/URL/commits e efeitos
confirmados em companion de publicação ligado ao packet, sem alterar seus bytes.
Replay/read-back confirmado é no-op. Negação ou drift no preflight: zero novo
efeito. Drift/falha após primeiro efeito preserva o confirmado e bloqueia efeitos
restantes não cobertos. Ack perdido/timeout após possível efeito persiste unknown
com `failure`/`recovery_required`, sem retry automático. Reconciliação lê identidade
remota; múltiplos candidatos ou ausência inconclusiva bloqueiam. Só ausência
conclusiva e nova autorização exata permitem outra tentativa. Efeito primário
confirmado com falha secundária resulta `partial`, refs e próxima ação.

**Delta de fronteira/persistência e repositório:** port mínimo de publicação
necessário ao AC-2 porque metadata/Issue create não publicam PR; adapter GitHub
concreto em `githubissues`, observação/push na fronteira `gitworkspace` existente,
sem abstração universal Provider/Transport. Companion necessário porque receipt
de integração não registra tentativa/efeito remoto de delivery; usar WorkflowStore
e proteção de T1. Caminho de artifact local confirma ref/digest sem Provider,
mas não substitui caminho GitHub. Completar CLI/DTO/help/skill compartilhada e
referências afetadas neste incremento; sem novos comandos presumidos aqui.

**Conclusão / testes determinísticos / Evidence E2:** fake de efeitos para
service e contratos sintéticos do adapter concreto, além de Git temporário/local
sem rede, comprovam alvo/conteúdo exatos e discriminam push e PR. Cobrir packet
incompleto, stale digest/binding, denial de push e de PR, drift, replay, timeout /
ack perdido → unknown → read-back/reconcile do mesmo efeito; erro ao persistir
após efeito confirmado → partial/recovery. Testar ausência conclusiva versus
inconclusiva, candidatos múltiplos e nova autoridade para retry. Artefato local
não chama Provider. E2 = registro de chamadas/contagem de efeitos, envelope e
refs sanitizados, testes correlacionados ao SHA; superfícies exibem o mesmo
artifact, estado da publicação e next action, separado do sucesso da operação.

**Segurança / antiobjetivos:** autoridade local não cobre push; push não cobre PR;
envelope conjunto discrimina ambos. Validar destino/conteúdo e preservar guardas
Git contra execução implícita. Nenhuma autoridade cobre merge/release/aceite ou
force push. Sem credenciais/logs sensíveis, chamada Provider real ou retry cego.

## T3 — Registrar decisão humana e continuar em nova Execution vinculada

**Objetivo / ACs:** decisão explícita sobre packet exato governa delivery (AC-5);
reject conserva história e permite apenas continuação nova autorizada (AC-4).
Fechar AC-3 e a jornada integrada nas superfícies já estendidas por T1/T2.

**Inputs/dependências:** packet de T1, publicação/entrega de T2 e autoridade humana
verificável. Accept requer packet final pronto e artifact entregue exato; reject
preserva também efeitos partial/unknown sem ocultá-los nem repeti-los.

**Escopo e resultado observável:** companion imutável de decisão contém actor /
authority ref, accept/reject, motivo/ref, packet digest, Execution/revisão/binding
e Evidence. Replay idêntico é no-op; contradição, autoridade de agente insuficiente
ou packet/revisão stale são recusados. Reject produz requisitos corretivos
acionáveis e delivery terminal rejected; resultado técnico permanece histórico.
Integrar decisão na derivação de lifecycle/status, completion e leitura/projeção:
pending/rejected nunca são accepted ou sucesso final do workflow. `success` de
inspect/registro de reject descreve apenas essa operação. Não reinterpretar
`State.Terminal` antigo nem liberar aceite pelo fact genérico.

Rework preview explicita nova ExecutionID, `reworkOf` (original + decisão),
escopo/Plan corretivo, escolha R1 retida ou R2 selecionada e inputs anteriores
validados por digest. Preparar proposta não inicia Execution. Reutilizar admissão
Start/binding #274, com nova autoridade de Execution e de cada gate posterior,
sem salto do primeiro stage obrigatório ou fallback de Runtime.

**Delta de persistência e repositório:** T3 estende WorkflowStore para retenção
endereçável por ExecutionID e troca confirmada do vínculo ativo; o slot atual
impede satisfazer AC-4 com Save. Reter original antes da troca, publicar novo
registro e link com locks/CAS/protocolo protegido/read-back. Seleção por ID lê
original e novo; list/status, inventory e retenção reconhecem ambos sem duplicar.
Fault entre retenção, novo registro e link bloqueia start até recovery; replay
correlacionado não cria terceira Execution. Preservar transitions/attempts /
receipts/decisão e binding R1 byte a byte. Companions de decisão/lineage faltam no
modelo vigente e são necessários a AC-4/5, não um novo control plane.

Preservar corpus execution1/2/3, graphs e resume. Definir envelope de vínculo
versionado que binário antigo recuse mutar nos alvos novos; snapshots legados
continuam íntegros. Sem migração global/rebind/limpeza. Testar upgrade/leitura
histórica e recusa de writer antigo em delivery/rework; declarar janela/forward
path/corpus do formato novo conforme FR-026 e qualidade. Rollback conserva
registros e efeitos remotos, não tenta desfazê-los. Reconciliar somente docs de
formato/compatibilidade, comandos e skill/catálogos afetados junto com código.

**Conclusão / testes determinísticos / Evidence E3:** aceite humano exato versus
agente/CI/PR/merge/closure sem autoridade; stale binding/digest, replay e decisão
contraditória; reject com motivo, bytes históricos preservados, R1 e R2 explícitas,
falta/nova autoridade, inputs adulterados e gates reiniciados. Fault injection
nos limites retenção/registro/link, concorrência CAS e recovery confirmam link
antes de start e nenhuma Execution duplicada. E3 = decisões/lineage e refs
sanitizadas, hashes antes/depois, testes de compatibilidade e uma jornada FAKE
integrada (estender `cmd/lingo/stage_journey_blackbox_test.go`), da integração ao
packet/publicação/decisão/rework, com original ainda inspecionável. Reutilizar
contratos sintéticos de instalação/routing Codex/Claude e JSON/human para provar
mesmos artifact/status/next action; não duplicar todos os casos de serviço na CLI.

**Segurança / antiobjetivos:** nova Execution nunca herda autoridade; humano
continua dono do accept/waiver e dos efeitos exatos. Não fechar Issue, publicar
release, editar história, alterar defaults ou iniciar Runtime real. Sem AXM-29/31,
framework de recovery, scheduler ou store paralelo; `axiom-workflow` continua
somente configuração e `axiom-work-item` continua superfície de execução.

## Matriz Tasks → ACs → Evidence e parada

A numeração abaixo é a ordem dos **cinco ACs de #277**, não a ordem das frentes.

| Task | ACs cobertos | Evidence mínima esperada na implementação |
|---|---|---|
| T1 | AC-1; AC-3 para packet/readiness | E1: snapshot e blockers verificáveis; store/integração, negativas e paridade de leitura |
| T2 (após T1) | AC-2; AC-3 para publicação/artifact | E2: envelope exato, zero efeitos indevidos, replay/reconcile/partial e adapters sintéticos |
| T3 (após T2) | AC-4 e AC-5; AC-3 integrado | E3: decisão explícita, nova lineage R1/R2, CAS/recovery, história/corpus e jornada FAKE |

Cada incremento entrega os testes dos pacotes alterados e validators obrigatórios;
na integração final executar `go test ./...` e
`./scripts/validate-repository.sh .`, além dos checks de CI aplicáveis. Mudanças de
persistência exercitam os guards de inventory/canonical stores e corpus append-only
da [política de qualidade](../../../.agents/policies/quality.md). Testes locais e CI
ficam separados na Evidence do SHA. FAKE/contratos sintéticos não provam operação
real: smoke Codex/Claude, PR Provider real e aceite/rejeição/continuação E2E são
#278/AXM-12, com autorização própria; não são Tasks bloqueantes ocultas de #277.

Revisão própria da decomposição: três unidades causais, sem task auxiliar de
camada/teste/docs, sem API final inventada, sem dependência AXM-29/31 ou ampliação
de autoridade. Preservar Project != Repository, Runtime != Model, Role != Model,
Execution != Agent, Provider != Transport, Integration != MCP, Skill != fonte da
verdade e Evidence != raw chat. Não foi identificado conflito material com o Plan;
se o código vigente exigir contradizê-lo, parar esse delta para deliberação humana.

**Parada desta etapa:** somente este artifact versionado, validação documental /
segurança, commit/push no branch e Evidence no PR #323 Draft. Nenhuma Task executada,
Issue adicional, mudança Linear ou aceite fabricado. **Próximo gate: review das
Tasks → autorização separada de implementação.**
