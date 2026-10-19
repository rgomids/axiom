# Plan 009 — Jornada guiada de configuração Project/Runtime

## Status, provenance e limite

**Proposed — plan, revisão 1, 2026-10-18.** Implementa futuramente a
[Specification 009](spec.md) rev.1 aceita **exatamente** em
`a99ba8d3afe49f47ab508422ed7ca70e2e553569`, por
[decisão humana no PR #329](https://github.com/rgomids/axiom/pull/329#issuecomment-6254246774)
e [registro Linear](https://linear.app/rgomids/issue/AXM-33/guiar-configuracao-de-project-e-runtime-com-preflight-de-viabilidade#comment-e55b95ac).
O cabeçalho Proposed da Spec é histórico e seus bytes ficam intactos. O
[gate observado READY_FOR_plan](https://linear.app/rgomids/issue/AXM-33/guiar-configuracao-de-project-e-runtime-com-preflight-de-viabilidade#comment-6cc861da)
autoriza esta proposta documental; não é gate de Execution canônica.

Work Item [AXM-33](https://linear.app/rgomids/issue/AXM-33), `In Review`, sem
blockedBy formal na entrada; Execution/gates/revisão canônicos **unknown**.
Reutilizar o único [Draft PR #329](https://github.com/rgomids/axiom/pull/329),
branch `docs/axm-33-guided-configuration-specification`. Não há aceite deste Plan,
Tasks, código, operação de produto, probe, migração, merge ou release nesta etapa.
[Evidence incremental](evidence.md#plan--checkpoint-e-validação-documental--2026-10-18)
registra inspeções e limitações, sem copiar conversa privada.

## Decisão técnica proposta e contratos preservados

Compor a jornada nas quatro skills atuais, com entrada de objetivo em
`axiom-project`, handoffs explícitos e chamadas aos donos existentes. O Project
continua dono do alvo e da policy; `axiom-runtime` configura a máquina;
`axiom-workflow` inspeciona/configura definições Project-owned; `axiom-work-item`
continua dono de Work Item/Execution. Skill não é fonte canônica do Workflow.
Não criar skill, comando, Doctor, router, registry de modelos, enum de jornada,
store, agregador de readiness ou motor de recuperação. Os passos P1–P6 abaixo
são organização deste Plan, nunca estados persistidos ou APIs propostas.

| Alternativa | Trade-off e recomendação |
| --- | --- |
| Compor skills/donos atuais | Recomendada: menor impacto, usa autoridade/readiness já testadas; exige continuidade e testes semânticos entre skills |
| Nova skill ou serviço central de setup | Concentraria orientação, mas duplicaria routing, snapshots e ownership; rejeitada sem necessidade demonstrada |
| Encerrar tudo ao primeiro blocker | Mais simples, mas elimina benefício independente; preservar a decisão aceita de parcial somente com opt-in informado |

O custo difícil de reverter seria introduzir nova autoridade/persistência ou
alterar schemas históricos. Texto de orientação e organização dos testes são
reversíveis. Nenhuma ADR nova é necessária para composição sem novos contratos.
Se faltar um fato material no resultado atual, parar a parte dependente: registrar
Clarification com cláusula/fato faltante, alternativas e dono; mudança durável
incompatível exige proposta/aceite humano sob ADR-0018 antes de código. Não
preencher lacuna com inferência nem autorizar implicitamente alteração de wire.

Preservar Project != Repository, Runtime != Model, Role != Model,
Execution != Agent, Provider != Transport, Integration != MCP, Evidence != conversa.
Manifesto portátil guarda intenção; instalação, bindings, Profiles, auth e proof
ficam locais. Effort pertence ao Model Profile vigente, nunca à Role ou à policy.
Contratos históricos mantêm versão/semântica; nenhum Workflow/Execution é rebindado.
Aplicam-se [Constituição](../../product/constitution.md),
[ADR-0004](../../decisions/0004-portable-project-manifest.md),
[ADR-0020](../../decisions/0020-workflow-definition-revision-binding.md),
[ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md),
[ADR-0023](../../decisions/0023-local-runtime-configuration-surface.md),
[ADR-0024](../../decisions/0024-model-profile-effort-ownership.md) e
[ADR-0025](../../decisions/0025-public-stage-dispatch-and-capability-proof.md).

## Reconhecimento versionado e donos

Baseline local limpo: `a99ba8d3afe49f47ab508422ed7ca70e2e553569`; main observado
`000936e5de71169ff563b236bcf17c3cfabc146c`. Inspeção de fonte somente leitura,
sem construir/executar Axiom ou Runtime e sem consultar Project do operador.
Nomes abaixo identificam contratos observados, não sintaxe fixa para o executor:
argumentos/authority efetivos sempre vêm da instalação inspecionada.

| Dono / fontes reais | Composição mínima e limite |
| --- | --- |
| [axiom-project](../../../internal/codexruntime/skills/axiom-project/SKILL.md), [catálogo](../../../internal/cli/skill_inspect.go) | `list/show/validate`; `configure` edit para Repository e policy; `workflow.select` para seleção. Não há operação Repository independente. Não usar create para contornar G-1 |
| [G-1 #305](../002-lingo-project-initialization/amendment-305-existing-project-runtime-policy.md), [projectapp](../../../internal/projectapp/policy_apply.go) | Policy read/validate/dry-run/apply/read-back existente; draft parcial exclusivo, sem misturar Repository/Provider. Snapshot/impacto Workflow e persistência continuam neste dono |
| [axiom-runtime](../../../internal/codexruntime/skills/axiom-runtime/SKILL.md), [autoria local](../../../internal/runtimeapplication/authoring.go) | `binding/profile list/show`, `validate-candidate`, create/edit/enable e recover somente quando necessários e autorizados. Reutilizar configuração válida; nenhuma migração/retirada automática |
| [axiom-workflow](../../../internal/codexruntime/skills/axiom-workflow/SKILL.md) | `definition.list/show/validate` e `configuration.readiness`; autoria/migração tem jornada própria. Seleção continua operação Project, execução continua Work Item |
| [axiom-work-item](../../../internal/codexruntime/skills/axiom-work-item/SKILL.md) | Inspeção/status de alvo já autorizado e handoff. Nenhum create/select/start/run/stage dispatch como teste de readiness. Linear no backlog não implica suporte do adapter de produto |
| [Readiness #231](../002-lingo-project-initialization/project-bootstrap-v3.md#4-canonical-project-readiness), [projeção CLI](../../../internal/cli/readiness.go) | Usar evaluator `projectapp`, relatórios por operação e blockers; sucesso estrutural não promove execução. Sem recálculo concorrente na skill |
| [AXM-31 AO-01–13](../007-configurable-workflows/amendment-axm31-operator-autonomy.md), [diagnóstico](../../../internal/cli/diagnostic.go) | Continuidade, opções e budgets existentes. Diagnóstico opcional ausente/inválido preserva recusa; não é operação Doctor ou fonte de autoridade |
| [Proof CLI](../../../internal/cli/runtime_capability.go), [auth](../../../internal/runtimeadapter/authpreflight.go), [observação passiva](../../../internal/runtimeadapter/observation.go) | Separar leitura de Evidence existente, observação de executável e ações que executam processos/inspecionam effort sob consentimento. Não produzir proof neste slice |

Discrepâncias pontuais a reconciliar futuramente nas skills, sem alterar agora:
`axiom-project`, seção Workflow ownership, ainda diz G-1 indisponível apesar de
sua seção policy e catálogo; `axiom-workflow`, Configuration readiness, diz
G-1/G-2/G-5 indisponíveis e contém referências antigas a dispatch/delivery.
Revisar apenas frases que afetam esta jornada contra catálogo efetivo e donos;
não fazer limpeza geral nem transformar presença no checkout em instalação.
A semântica G-1 pertence a AXM-34/#305; a semântica G-2/#306 a seus contratos.
AXM-33 pode ajustar orientação/handoff e testes de contrato, não essas operações.

[AXM-32 / PR #328](https://github.com/rgomids/axiom/pull/328) foi revalidado em
`aba1c4d843e4b44a891531831aec2c79e95aae18`: Spec 008, ADR-0030, Plan e Tasks
aceitos; implementação ainda incompleta, com decisão de binding/modo de entrega
pendente no checkpoint. [ADR-0030 nesse SHA](https://github.com/rgomids/axiom/blob/aba1c4d843e4b44a891531831aec2c79e95aae18/docs/decisions/0030-canonical-results-terminal-and-host-presentation.md)
reserva JSON canônico, terminal nativo e conversa do host a AXM-32. Não copiar seu
renderer, migrar literal relay/Auxiliary ou criar chamada de modelo de formatação.
O baseline ainda possui relay literal: não removê-lo por AXM-33. Na integração,
revalidar o par binário/skills e consumir o contrato entregue por AXM-32; se ainda
não entregue, bloquear somente a ativação dependente da nova apresentação.
Fixtures podem verificar decisões/fatos sem antecipar renderer nem wire novo.

## Sequência proposta, precedências e outputs

### P1 — Resolver alvo e inventariar sem efeitos

Inspecionar a operação/skill efetiva antes de coletar configuração. Pedido
explícito de inspect termina na própria inspeção. Para objetivo natural, resolver
um único Project por list/show e observar UUID, source/selector e revisão;
ambiguidade pede só identificador material. CWD, remote, host e memória não são
selectors. Ler cada Repository por chave, identidade remota declarada e binding
local; divergência não escolhe `origin` nem altera Git.

Ler separadamente: policy portátil (ausente/unconfigured/vazio/configurado),
bindings e Profiles locais habilitados/elegíveis e associação Runtime/model/Profile,
effort declarado, Workflow none ou source/ID/revisão/digest e requisitos Stage/Agent,
readiness por operação, Integration/Provider e observações de auth/proof existentes.
Todo fato leva referência, versão e instante/validade disponível; acesso ausente
é **não observado**, não configuração ausente. Output: inventário derivado com
lacunas e fontes; nenhuma serialização nova de estado/autoridade.

### P2 — Preflight completo antes de escolhas técnicas

Para cada pré-requisito do objetivo delimitado, relacionar lacuna → operação/dono
→ disponibilidade/argumentos → dependências → autoridade futura → output esperado
→ leitura que o confirmará. Ordem lógica: identidade e compatibilidade da
instalação; Repository/binding; policy e candidato local; definição/seleção Workflow;
readiness Work Item/execução e auth/proof exigidos. Inspecionar todas as dependências
conhecidas antes de perguntar modelo, effort, ID ou template. A ordem de writes
não é fixa: se a policy proposta conflita com Workflow ativo, a reparação/seleção
compatível deve vir antes, sob seu próprio preview; se exigir autoria/migração
fora do objetivo, fazer handoff e parar a parte afetada.

| Observação futura necessária | Método permitido e conclusão limitada |
| --- | --- |
| Versão efetiva Axiom/Lingo | Resolver a instalação efetivamente usada e sua operação version/provenance documentada; registrar release/revision/sourceState e skill-set/compatibilidade por status. Não usar `go run`, build do checkout ou release anunciada como substituto; divergência entre executáveis/skill-set bloqueia |
| Operações/argumentos/efeitos | Introspecção do binário instalado (`skill inspect`) e ajuda canônica de comandos, metadata de modes/accepted forms/authority; comparar operações requeridas. Metadata incompleta permite diagnóstico read-only suportado, nunca tentativa de mutação |
| Snapshots/formats | Leituras/validadores do dono para schema portátil, formato local, source/installation, Workflow e referências/digests. Recusa de formato/recovery impede write; não converter ou retirar registro para fazer a jornada caber |
| Installed/integrated | Observação passiva de executável e status de compatibilidade da integração pelo dono. Não executar fornecedor para descobrir instalação; nome/path ou CI não provam suporte |
| Supported | Fonte canônica aplicável à combinação observada; catálogo de tokens só prova forma. [Registry versionado](../../../internal/runtimeadapter/runtime-versions.json) com entries vazio não prova modelo/effort. Sem fonte, não recomendar combinação concreta |
| Authenticated | Apenas observação sanitizada válida no escopo/identidade do dono. Auth preflight atual pode rodar status/version do fornecedor: não é leitura passiva nem login; não acioná-lo automaticamente nesta jornada. Se nova observação for necessária, encaminhar sob authority própria e manter unknown até retorno |
| Proven | Inspeção suportada da Evidence existente e validade do binding exato (Profile/model/effort/executável/versão/variante/digests/freshness). `capability show` não é permissão irrestrita: se exigir effort-source inspection, processo ou consentimento, parar e encaminhar. Não prepare/probe/invalide Evidence para descobrir capacidade |

Installed, authenticated, supported e proven permanecem fatos independentes.
Auth subscription observada não prova usabilidade/billing; Profile salvo não é
proof; consentimento não cria produtor de proof. Readiness não pode certificar
fatos que seu relatório não expõe. Fato relevante desconhecido limita a conclusão.

**Parada imediata:** identidade insegura, operação/argumento necessário ausente ou
unknown, snapshot incompatível, proof/produtor exigido indisponível, recusa,
recovery pendente ou autoridade negada interrompem o caminho dependente antes de
coleta técnica e de qualquer apply. Caso C04: policy precisa de G-1 e a versão
efetiva não a expõe → informar versão, lacuna e próxima reavaliação após processo
próprio de instalação; zero novo Profile, sem upgrade automático ou contorno.
Output: caminho condicionado verificável ou blocker com menor próxima ação.

Alternativa parcial exige benefício independente, operação suportada observada,
perdas e execução ainda bloqueada explicados. Perguntar primeiro se a pessoa quer
esse objetivo reduzido; somente opt-in explícito habilita sua coleta/preview.
Escopo não escolhido termina blocked. Um subconjunto já confirmado pode constar
do relato, mas não autoriza novas ações. Mudança material refaz preflight.

### P3 — Uma decisão material por vez

Reutilizar Profile e Workflow compatíveis e decisões ainda válidas. Explicar
finalidade, alternativas observadas, efeito prático e recomendação com fonte.
Campo aberto de intenção não vira catálogo; capability ambígua pede uma escolha
semântica. Sem fonte de modelo/effort, explicar unknown e bloquear uso; intenção
só pode ser salva se o contrato local permitir e o objetivo parcial for escolhido.
Não exigir template, tuning ou multiagente desnecessários. `runtime-default` só
se suportado e escolhido explicitamente, sem inventar esforço interno.

Novo Profile ID: validar forma/colisão pelo dono, propor apenas candidato e pedir
confirmação de rótulo compreensível. Runtime do host não seleciona alvo nem modelo.
Output: decisão explícita ligada a fatos atuais, ou pendência única; nenhum apply.

### P4 — Draft, preview e autoridade por efeito

Preparar dados tipados apenas no formato efetivamente aceito; validação e preview
são do dono. Policy usa o caminho G-1 exclusivo; binding/Profile local usa G-2;
Workflow selection usa Project. Não combinar seus writes ou confirmações.
Preview deve permitir revisar identidade, antes/depois/diff, alcance portátil ou
local compartilhado, write set, revisões/digests vinculados, dependências,
impacto Stage/Agent/Workflow e readiness esperada/limites. Não alegar inventário de
Work Items/Executions não consultados. Draft não é novo store canônico.

Autorização é **por apply exato**: operação, alvo/id, conteúdo, revisão/digest e
alcance apresentados, conforme tuple exigida por sua metadata. Intenção geral,
escolha de Profile, validação ou confirmação de policy não autorizam o próximo
apply. Negativa cancela sem substituição/insistência; efeitos anteriores ficam
identificados. Confirmação válida do mesmo preview não exige pergunta duplicada.

Antes de apply, reobservar todos os snapshots vinculados pelo dono e os fatos
materiais adicionais usados no preflight: Project/source/selector, policy,
configuração local, executável/proof/auth e Workflow. Dono continua responsável
pela checagem atômica/stale que seu contrato oferece; a skill não promete nova
atomicidade entre operações. Qualquer drift material invalida preview/autoridade,
exige novo preview e confirmação, sem rebase silencioso. Se faltar cobertura
canônica necessária, bloquear/Clarify em vez de implementar lock paralelo.

### P5 — Read-back e readiness após cada apply

Ler estado persistido pelo dono; comparar identidade, revisão/digest e conteúdo
com preview autorizado. Usar read-back G-1 emitido e nova leitura quando necessária,
list/show G-2 e definição/seleção Workflow conforme operação. Reavaliar readiness
das operações afetadas pelo evaluator existente. Depois atualizar lacunas; nunca
continuar uma cadeia dependente com ack incerto ou leitura divergente.

Commit/write sucedido com readiness bloqueada permanece configuração parcial.
Ack perdido, read-back indisponível ou writer concorrente deixam efeitos incertos
explicitados; não repetir mutação. Somente inspeção/recovery suportado com sua
própria autoridade. Sem transação global ou rollback automático entre donos.

### P6 — Síntese e handoff sem iniciar Execution

Separar status/result da operação, readiness por operação, certeza de efeitos e
aceite. `ready/partial/blocked` são só sínteses da jornada, nunca completion enums,
gates, campos ou estado paralelo. Ready exige pré-requisitos do objetivo/Workflow
observados e válidos, sem lacuna relevante; declarar escopo avaliado e que start
precisa preview/authority e auth/proof frescos próprios. Partial exige subconjunto
útil explicitamente escolhido e confirmado, mantendo blockers canônicos de
execução. Blocked informa lacuna e menor próxima ação sem promessa de sucesso.

Handoff para `axiom-work-item` preserva alvo/referências e não cria/linka Work Item,
não inicia/resume Execution, não muda gates nem infere stage do Linear. Se alvo
Execution não foi observado, registrar unknown. Não usar run para provar readiness.

## Rastreabilidade de critérios e validações futuras

| Critério | Etapas / precedência | Operações/donos e output | Validação / cenários |
| --- | --- | --- | --- |
| GC-01 | P1 antes de P2/P3; reobservar P4/P5 | Project list/show/validate, Repository por chave, G-2 list/show, Workflow definition.show; inventário referenciado ou unknown | Identidade inequívoca; nenhum selector por host/remote; C01/02/03/08/11/14/16 |
| GC-02 | P2 antes de P3; mudança material retorna P2 | Introspecção/ajuda, readiness Project, observações do dono; caminho completo ou parada, parcial opt-in | Ordem de eventos: blocker antes de pergunta técnica; nenhuma descoberta mutante; C03/04/05/06/10/11/12/13/14 |
| GC-03 | P3 só após P2 viável/opt-in | Catálogos/validate-candidate G-2 e definições Workflow; decisão única fundamentada | Zero valores/defaults/proof inventados; zero repetição de decisão válida; C01/04/05/06/16 e paridade |
| GC-04 | P4 antes de cada write, independente | G-1 configure/edit, G-2 binding/profile e Project workflow.select; preview/tuple exatos | Negativa/stale têm zero efeitos; snapshots parametrizados; C02/03/07/08/09/11/12/13/14/15/16 |
| GC-05 | P5 após cada write; P6 só com confirmação | Read-back do dono e evaluator readiness; lacunas, efeitos e síntese delimitada | Sucesso não esconde blocker/unknown; sem replay; C01/03/04/06/07/09/12/13 |
| GC-06 | Transversal, retomada preserva P1/P2 e budgets | AO-06/07/09 nos donos; inspeção/recovery/handoff sanitizado | Relógio/contadores, zero bypass/canário; C02/09/10/15 |

## Segurança, continuidade e recuperação bounded

Nenhuma leitura concede escrita; authority local, externa, probe, recuperação e
start são independentes. Não resolver credenciais para preview, coletar stdout
de auth ou publicar paths privados. Usar referências lógicas/digests e relatórios
sanitizados; sem varrer diretórios privados. Preservar confinamento, links,
permissões e regras de publicação dos donos. Repository/Provider/erro/draft são
dados não confiáveis: instruções embutidas não viram comandos/consentimento.
Não editar manifestos, registries, receipts, Git, credenciais ou vendor settings
para contornar recusa. Ausência local não apaga declaração portátil.

Aplicar AO-06/07/09: mesma correlação alvo/categoria/revisão/bloqueio/attempt,
no máximo duas repetições após inicial em 60 s e duas remediações por bloqueio,
respeitando teto menor; parar na primeira falta de progresso verificável,
negativa, risco novo, efeito incerto ou excesso de escopo. Só leituras/replay seguro
comprovado sob autoridade vigente; tetos não concedem mutação. Probes, inferência,
login, child retry, kill e redispatch não entram no orçamento automático.

Checkpoint em Evidence de trabalho já autorizada: objetivo, versão/par compatível,
alvo/refs/revisões, decisões válidas, blocker/attempt, início/deadline original,
contadores cumulativos, autoridade usada/pendente, efeitos confirmados/incertos,
verificação e próxima ação. Retomada reobserva estado/authority sem zerar budgets;
checkpoint incompleto/sem tempos impede repetição automática. Revisão nova sozinha
não zera bloqueio; fato novo/proposta humana delimitada precisa referência e não
concede efeitos. Não serializar segredos nem criar store de autorização.

## Validação proporcional e matriz de Evidence

Testes futuros devem consumir respostas canônicas reais do contrato ou fixtures
sanitizadas que preservem sua forma/limites, com doubles de stores/observadores,
relógio controlado e contadores de writes/dispatch. Não adicionar engine de
jornada só para testar o texto. Testes de strings nas skills provam instruções,
não decisões da IA. Reusar suites de catálogo/routing e controles dos donos;
ampliar unidade só onde mudar comportamento ou existir risco não coberto.

| Caso | Fixture/variante e oráculo mensurável | Gate nesta revisão |
| --- | --- | --- |
| C01 | Estado já compatível: zero perguntas de fatos conhecidos, zero writes/dispatch, ready com escopo | NOT_RUN |
| C02 | UUID ambíguo e remote divergente: uma decisão pendente por pergunta, zero escolha por origin/edição Git | NOT_RUN |
| C03 | Binding ausente com operação presente/ausente: preflight antes de tuning; write só exato e confirmado ou blocked | NOT_RUN |
| C04 | Instalação sem G-1 e policy ausente: zero perguntas de modelo/effort/ID/template, zero Profile novo/apply | NOT_RUN |
| C05 | Capability ambígua/catálogo conhecido: alternativas com fonte, uma escolha sem token/proof inventado | NOT_RUN |
| C06 | Installed/authenticated com supported/proven unknown/incompatível: zero inferência; intenção só com contrato e parcial opt-in | NOT_RUN |
| C07 | Negativa por binding, Profile, policy e selection: zero writes do efeito negado, nenhuma substituição | NOT_RUN |
| C08 | Drift em UUID/source/selector, policy, local e Workflow: cada dimensão recusa stale sem write e exige novo preview/consentimento | NOT_RUN |
| C09 | Ack perdido, read-back faltante e writer pós-commit: zero replay, nenhuma afirmação ready sem confirmação, certeza separada | NOT_RUN |
| C10 | Falha transitória/persistente e duas sessões: relógio verifica ≤2 repetições/60 s, ≤2 remediações; parada anterior sem progresso e sem reset | NOT_RUN |
| C11 | Inspect explícito, operação ausente e metadata/skill contraditória: inspect encerra; zero coleta/apply substituto; unknown bloqueia | NOT_RUN |
| C12 | G-1 presente + dependências/auth/proof válidos: ordem preflight → decisão → preview → consentimento → apply → read-back/readiness; fixture não substitui real | NOT_RUN synthetic; NOT_RUN real |
| C13 | G-1 presente, proof/auth ausente ou mismatch: benefício escolhido e confirmado, partial com execução blocked; policy portátil preservada | NOT_RUN |
| C14 | Workflow none/incompatível/built-in/ref inválida: seleção observada e consentimento separado ou handoff; zero edição de built-in/Execution | NOT_RUN |
| C15 | Injeção em draft/erro/Provider e canário sintético: zero comandos/autoridade promovida; canário ausente em saída/Evidence | NOT_RUN |
| C16 | Colisão de Profile ID e template opcional: validar colisão, candidato só após decisão; zero pergunta obrigatória de template | NOT_RUN |

Executar entradas natural e explícita de cada caso aplicável. Repetir o conjunto
de decisões em Codex e Claude, inclusive host diferente do Runtime alvo.
Comparar semântica (operação, escolhas, ordem, consentimento, blockers, efeitos),
não prosa idêntica. Exigir zero autoridade ampliada, zero capacidade/default
inventado, zero blocker material omitido, zero dispatch e zero loop fora dos
limites em casos sintéticos. Assertar conteúdo/fonte das alternativas e número
máximo de uma decisão material pendente por pergunta.

Contratos das quatro skills devem cobrir ownership, inspect-stop, preflight,
parcial opt-in, budgets e read-back; testes de instalação/reinstalação/upgrade
comparam a mesma orientação em ambos os hosts. Expandir testes existentes em
`internal/codexruntime/workflow_skill_content_test.go`,
`internal/cli/domain_skill_routing_test.go` e
`internal/cli/skill_catalog_contract_test.go`, preservando histórico publicado.
Suites de policy edit, runtime authoring, Workflow binding e compatibility são
regressão dos donos, não autorização para mudar sua implementação. Não ampliar
suite por rotina se nenhum contrato/código relevante mudou.

| Camada de Evidence | Conteúdo exigido / gate de saída futuro | Estado nesta etapa |
| --- | --- | --- |
| Documental | Links/índice, cobertura GC/C, governança ADR, whitespace, escopo e hygiene; revisão semântica/security do Plan | Registrar comandos/resultados em evidence.md; não prova produto |
| Unitária/contrato | Fixtures/refusals/stale, revisão/digest exatos, ausência de efeitos, instalação das skills e compatibilidade | NOT_RUN |
| Conversacional synthetic C01–C16 | Fonte/versão e SHA, eventos inventário/preflight/perguntas/decisões, preview/authority, writes/dispatch, read-back/readiness, budgets e resumo | NOT_RUN |
| Paridade nativa Codex/Claude | Mesmas decisões e restrições, hosts/alvos cruzados, recortes sanitizados mínimos e resultados canônicos referenciados | NOT_RUN |
| C12 E2E real | Instalação efetiva G-1, par binário/skills compatível, alvo autorizado, auth/proof válidos e exatos, preview/apply/read-back/readiness e Evidence observada | NOT_RUN; falta de G-1/produtor/proof válido bloqueia execução real |
| Aceite | Review e decisão humana da revisão exata, separados de CI, merge e publicação | Plan Proposed; sem aceite |

C12 real exige autoridade futura separada para ambiente/alvo e efeitos exatos;
não executar probe/run como parte deste Plan nem construir engine de proof ou
processo E2E. Sem pré-requisitos, manter NOT_RUN com motivo; não trocar por PASS
sintético. Registrar synthetic/real, Runtime operador/alvo, versão/skill-set/SHA,
refs/digests/freshness, resultado esperado/observado e autoridade, inclusive
recusas. Evidence canônica é referenciada; conversa bruta não a substitui.
R-2/R-3, aceite AXM-34 e E2E amplo AXM-12 continuam gates independentes.

## Componentes previstos, dependências e integração

| Componente/arquivo previsto na implementação futura | Mudança mínima / dependência |
| --- | --- |
| `internal/codexruntime/skills/axiom-project/SKILL.md` | Entrada de objetivo, P1/P2 e handoff; corrigir orientação G-1 contraditória com dono, sem duplicar schema/argumentos |
| `internal/codexruntime/skills/axiom-runtime/SKILL.md` | Condicionar coleta ao preflight, opções observadas, decisão única/opt-in e confirmação local; preservar G-2 e effort |
| `internal/codexruntime/skills/axiom-workflow/SKILL.md` | Continuidade/preflight, reconciliar frases afetadas G-1/G-2/readiness; preservar definição/seleção e autoridade separadas |
| `internal/codexruntime/skills/axiom-work-item/SKILL.md` | Receber handoff de configuração, reobservar pré-requisitos; nenhum start implícito ou alteração de lifecycle |
| Testes de skills/routing citados e fixtures junto às suites existentes | Contratos/oráculos negativos e GC/C; nome/local final no futuro Tasks, sem criar runner de produto |
| `docs/specifications/009-guided-project-runtime-configuration/evidence.md`, índice e referência de operações se afetada | Evidence incremental e reconciliação estritamente proporcional; nenhum accepted artifact reescrito |

Não há mudança prevista em domínio, CLI, API, schemas, persistência, dependências,
infra ou processos de execução. Falta de informação canônica bloqueia a parte
afetada e retorna Clarification/dono; este Plan não aprova extensão especulativa.

Sequenciamento futuro, sem constituir Tasks aprovados: (a) fixar par compatível e
contratos observados dos donos; (b) orientar P1/P2 e parada; (c) compor P3–P6,
security e continuidade; (d) validar contratos/fixtures e paridade; (e) integrar
apresentação entregue por AXM-32 e realizar C12 somente sob sua authority futura.
Cada dependência falha interrompe só seu caminho. AXM-31 está Done; não reabrir
aceite. G-1 #327 merged em main e AXM-34 Awaiting Release não provam operação
instalada/proof. G-1 insuficiente bloqueia C12, **não** redação/review deste Plan.
G-2, AXM-32, AXM-12, produtor de proof e migração histórica não são entregas AXM-33.

Compatibilidade/release: orientação deve consultar versão/metadata efetivas e
recusar combinações binário/skills incompatíveis; sem minimum version inventada.
Usar mecanismo existente de empacotamento/upgrade dos donos e suas validações,
sem installer novo ou publicação aqui. Preservar snapshots de skills/receipts e
corpus históricos. Mudança de apresentação é integrada ao conjunto compatível de
AXM-32, nunca cherry-pick de contrato parcial por suposição.

Rollback: retirar orientação futura ou restaurar pacote compatível conhecido pode
cessar a jornada; não desfaz binds, Profiles, policy ou seleção já aplicados.
Desfazer configuração só por operação inversa suportada, novo preview/authority,
read-back e avaliação de dependências. Ack incerto/recovery pendente exige dono;
sem restaurar arquivos/backups por fora nem prometer atomicidade global.

## Riscos, fatos incertos e saída de plan

| Risco/incerteza | Mitigação / decisão necessária |
| --- | --- |
| Instalação/produtor de proof reais não observados | Preflight obrigatório; C12 real NOT_RUN até Evidence exata e authority futura; não impede Plan |
| Skills stale e apresentação AXM-32 ainda não entregue | Reconciliar somente orientação afetada; verificar par na integração e coordenar com donos; não migrar output aqui |
| Readiness sem auth/proof/fato material completo | Manter não observado e restringir ready; lacuna material segue Clarification, sem novo Doctor |
| Drift entre operações, estado parcial e efeito incerto | Snapshot por dono, nova confirmação, read-back após cada write; parar cadeia e preservar efeitos/budgets |
| Teste textual parecer prova comportamental | Separar evidências documentais/contratuais/synthetic/nativas/reais; nenhum PASS por CI documental |
| Backlog Linear confundido com adapter/Execution | Não fabricar binding, Issue espelho ou Execution; suporte efetivo e authority são gates independentes |

Saída desta etapa: Plan Proposed rastreável a GC-01–06/C01–C16, fontes atuais e
fronteiras reconciliadas, somente Plan/índice/Evidence alterados, bytes aceitos da
Spec preservados, checks documentais com resultados registrados, diff/security
revisados e publicação confirmada no mesmo PR/branch com SHA e comentário bounded
no AXM-33. CI do novo SHA é observado, nunca herdado do anterior.

Decisão humana pendente: review/aceite deste **Plan rev.1 no SHA exato publicado**,
ou Clarification pontual. Recomendação: aceitar a composição mínima com ativação
condicionada aos contratos dos donos e validação real posterior. Não há decisão
arquitetural incompatível proposta. Tasks só após aceite humano específico do
Plan e authority própria; nenhuma aceitação, transição canônica, merge ou release
é registrada por este documento.
