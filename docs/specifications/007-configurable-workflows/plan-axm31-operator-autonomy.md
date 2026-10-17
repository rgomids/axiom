# AXM-31 — Plan da experiência e autonomia da IA operadora

**Proposed — plan, 2026-10-17.** Somente desenho técnico; sem Tasks executáveis,
implementação, dispatch, probe, Provider de produto ou transição de Execution.

## Gate, fontes e checkpoint

O mantenedor @rgomids [aceitou explicitamente a Specification](https://github.com/rgomids/axiom/pull/325#issuecomment-6241938642)
em 2026-10-17, referindo-se ao SHA `2e92952`, resolvido pelo head remoto como
`2e9295293c84a51523df50b28258149debb6badf`. O aceite inclui AO-01–13, jornadas
1–6, P1–P8/N1–N10 e os limites AO-06/07; preserva proof independente,
authority exata, preview/apply e ownership de AXM-9/AXM-12. Libera Plan,
sem autorizar implementação, merge ou release.
O [amendment aceito nessa revisão](https://github.com/rgomids/axiom/blob/2e9295293c84a51523df50b28258149debb6badf/docs/specifications/007-configurable-workflows/amendment-axm31-operator-autonomy.md)
tem blob `b4a9605cdb4886193795ad8089c35fe4be41e62b`; seus bytes Proposed são
preservados como revisão histórica. Este Plan não se autoaceita.

Fontes normativas: [Spec 007](spec.md), [Constituição](../../product/constitution.md),
[ADR-0025](../../decisions/0025-public-stage-dispatch-and-capability-proof.md),
[ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md) e
[política Linear/GitHub](../../development/linear-github-issue-tracking.md).
A [matriz de aceite](amendment-axm31-operator-autonomy.md#aceite-mensurável-e-evidence-futura)
permanece o contrato; este Plan não reduz a cobertura a P1–N8 mencionada no
parágrafo histórico sobre dependência.

Preflight por leitura remota e inspeção local em 2026-10-17:

- HEAD local e branch remota canônica no SHA completo acima; árvore limpa antes
  das edições. Único PR aberto de AXM-31: [Draft #325](https://github.com/rgomids/axiom/pull/325),
  branch `docs/axm-31-operator-autonomy-specification`. O outro PR aberto, #317,
  é alheio ao escopo. Não criar outro PR para AXM-31.
- AXM-31: Todo, sem blockedBy, blocks AXM-12, related AXM-8/AXM-9;
  cinco comentários Linear, sem paginação restante. O comentário de prompt
  condicional não é o aceite; a decisão vinculante está no comentário GitHub acima.
- `PR Checks` e `delivery-metadata` PASS **no SHA da Specification**,
  [CI](https://github.com/rgomids/axiom/actions/runs/39071420903) e
  [metadata](https://github.com/rgomids/axiom/actions/runs/39071438064).
  Não constituem Evidence comportamental nem CI deste futuro commit de Plan.
- Inspeção de código/skills no mesmo head, sem executar produto. `axiom` não
  foi encontrado no PATH desta sessão; os arquivos embarcados não provam uma
  instalação operacional no host. Execution canônica permanece desconhecida.
- Shell/Git e leituras autenticadas GitHub/Linear disponíveis. Sem spawn:
  inferioridade estrita de modelo não comprovada; inspeção determinística
  limitada atende o gate e o desenho com menor custo de coordenação.

## Problema e inventário comprovado

A operação suportada deve continuar por intenção natural ou token explícito,
perguntar somente o dado material ausente e explicar falhas/decisões ao humano.
O problema confirmado é orientação incompleta sobre contratos existentes;
a qualidade de conversa real entre turnos ainda é hipótese a validar.
As referências de código desta seção são relativas à raiz do repositório e ao
head auditado, não promessa de capacidade de qualquer Runtime instalado.

| Capacidade / classificação | Evidence por inspeção | Delta mínimo / dono |
|---|---|---|
| Project/Repository: coberto | `internal/codexruntime/skills/axiom-project/SKILL.md`, `internal/cli/skill_inspect.go`, `internal/projectapp/edit.go`: list/show/validate, configure create/edit, preview/apply/read-back | Reutilizar; Project mantém associações. Sem novo CRUD, comando Repository ou gravação direta. |
| Colisão ao adicionar: gap de orientação confirmado | `edit.go:proposePortable/proposeLocal` permite upsert por chave; chave existente pode ser atualização legítima. `internal/project/validation.go` recusa chaves/locators duplicados nas declarações | Skill deve distinguir intenção de adicionar de atualizar antes de propor substituição. Não alegar que todo upsert existente é erro CLI. Regras e recusas canônicas continuam no domínio. |
| Workflow: coberto | `internal/projectapp/workflow_authoring.go`, `cmd/lingo/workflow_authoring.go`, skill `axiom-workflow`: list/show/validate, create `--from-default`, edit, seleção separada; `builtin_read_only` | Reutilizar cópia/edit e `axiom-project workflow.select`; sem nova definição/seleção engine. |
| Work Item/Execution: coberto | Skill `axiom-work-item`, `internal/cli/skill_inspect.go`, `internal/cli/workflow_stage.go`: catálogo e authority por operação, stage status/retry/reconcile; delivery já roteado | Preservar catálogo, binding e receipts; AXM-9 mantém delivery, sem reproduzir sua semântica aqui. |
| Entrada dupla e inspect-only: coberto parcialmente | As três skills já preservam token explícito, resolvem intenção e encerram `skill inspect`; `internal/cli/domain_skill_routing_test.go` verifica contratos de roteamento | Não reescrever resolver na CLI. Reforçar continuidade e negativos nas skills/tests existentes. |
| Continuidade: gap textual confirmado / eficácia hipótese | Skills proíbem inferência por chat/CWD/remote, mas não explicitam a sequência completa de referência observada → releitura → próximo turno | Explicitar referência candidata observada, sem criar binding ou selector implícito; conversa nativa continua não comprovada. |
| Completion/metadados: coberto | `internal/completion/completion.go`, `internal/cli/completion.go`, `presentation.go`, `workflow_stage.go`: status fechado, categoria e payload, Markdown/JSON ordenados; skill inspect descreve args/efeitos/authority | Extensão delimitada do payload de falha; manter status, campos, ordem relativa e exit codes. `details` é string limitada a 256 bytes, não objeto de diagnóstico. |
| Diagnóstico de stage: gap confirmado | `cmd/lingo/stage_service.go:stageError`: `capability_unknown` recomenda probe; `stop_orphan_then_reconcile` não apresenta alternativas/authority de stop | Enriquecer fatos e opções no mapeamento, sem ampliar autorização nem sinalizar processos pela skill. |
| Falhas Project/Workflow: gap estruturado confirmado | `cmd/lingo/main.go:editFailure` e `cmd/lingo/workflow_authoring.go:workflowAuthoringCompletion` mapeiam fatos para texto/next e report; não expõem o conjunto estruturado de opções/authority/verificação AO-03 | Reusar o DTO de diagnóstico nas falhas dessas jornadas, sem mudar a validação/CRUD. |
| Proof: owner externo / bloqueio real de produção | `internal/local/probe_session.go:Verify` inicia `Inconclusive: true`, não atesta `ModelInvoked/EffortVerified`; `internal/capabilityproof/service.go` exige observação independente; `internal/runtimeadapter/runtime-versions.json` vazio | Corrigir orientação, sem implementar atestador nem popular registry. Registry vazio sozinho não prova impossibilidade de todos os Runtimes. Dono técnico da capability/#276/AXM-8 precisa solução bounded própria antes de dispatch real AXM-12. |
| Retry/reconcile: coberto | `internal/stageexecution/reconcile.go`, `cmd/lingo/stage_service.go`, `stage_single.go`, `stage_dispatch.go`; testes `stage_adversarial_blackbox_test.go`/`stage_journey_blackbox_test.go` | Reusar checks, disposition, append fact, lease/epoch e histórico; nenhum novo recovery engine. |
| Limites/handoff: gap de orientação confirmado | Skills atuais não codificam os budgets AO-06/07 nem checkpoint acumulado AO-09 | Orientação e Evidence sintética delimitadas; o Runtime continua responsável por seguir o contrato. Sem scheduler, store ou policy global novos. |
| Apresentação auxiliar/diagramas: parcial / limite do host | Skills exigem relay exato do Markdown; Workflow já admite resumo ao lado do preview. Não foi provado renderer gráfico nesta sessão | Delimitar resumo auxiliar após resultado intacto; diagrama opcional derivado, fallback textual obrigatório, sem dependência/UI nova. |

## Decisões mínimas, interfaces e boundaries

### D1 — Skills conduzem a conversa; aplicação mantém domínio e authority

Alterar somente as três skills embarcadas em `internal/codexruntime/skills` e
seus contratos de instalação/upgrade quando necessário. Não adicionar skill,
alias, interpretador natural na CLI ou segundo catálogo de argumentos.
O catálogo do binário continua canônico; não se extrai comando executável de
`next`, texto de erro, conteúdo de child ou Provider.
Descoberta interna e pedido explícito `skill inspect` são distintos: este último
termina na inspeção, sem perguntas adicionais ou execução no mesmo turno.
Operação explícita inválida recebe recusa, sem fallback semântico.

Uma referência de conversa é somente candidata: guardar origem da leitura,
Project ID, chave Repository e identidade/revisão/digest Workflow, ou os
selectors persistidos de Work Item/Execution/stage/attempt quando disponíveis.
Revalidar pelo domínio antes do próximo efeito; ausência/ambiguidade pede apenas
o identificador faltante. Nunca fabricar default, session selector ou binding.
Selectors já omitidos em start/resolution não podem ser trocados por explícitos
no apply reaproveitando preview, mesmo que apontem ao mesmo UUID.

Não alterar registros canônicos para guardar contexto. O contexto do Runtime é
descartável; handoff é Evidence sanitizada e requer releitura. Sem referências
confiáveis, fazer descoberta/pergunta bounded, jamais reconstruir Execution.

### D2 — Diagnóstico tipado dentro do payload operacional existente

Acrescentar o payload operacional opcional `diagnostic`, com `schemaVersion: 1`,
ao mesmo evento de completion das falhas cobertas. Compor na fronteira de aplicação
de `cmd/lingo`, transportar em `cli.Result`/`WorkflowStageResponse` e serializar
por `internal/cli`, ao lado dos payloads `stage`, `workflowAuthoring` ou `edit`
quando presentes. Acrescentar depois dos payloads operacionais existentes; manter
sua ordem relativa. Não envelopar novamente Graph/single/preview nem mudar seus
nomes/caminhos. Falha sem outro payload pode emitir somente `diagnostic` junto ao
completion. Usar o mesmo DTO de saída para stage e falhas Project/Workflow de J1,
sem acrescentá-lo aos reports de domínio ou seus codecs persistidos.
Não alterar `completion.Result`, `completion.Status` ou `details` para isso.
O DTO é dado de saída da operação, não protocolo alternativo nem estado persistido.

| Campo proposto | Conteúdo e restrição |
|---|---|
| `operation`, `category`, `target`, `observedRevision` | Operação/categoria existentes, refs exatas conhecidas e revisão. Ausência explícita para o que não foi resolvido; nunca inventar UUID/revisão. |
| `observed`, `notObserved`, `hypotheses`, `missingPrerequisites` | Fatos com referências sanitizadas; hipótese sempre separada. Nenhuma saída bruta de autenticação/child. |
| `capability`, `authority`, `confinement` | Observações independentes; indisponível/desconhecido não é negado/provado. Authority requerida não é grant. |
| `confirmedEffects`, `uncertainEffects` | Somente efeitos sustentados por observação; não presumir ausência de efeito em timeout/perda de ack. |
| `options` | No máximo 3, ordenadas por menor efeito; ID, disponibilidade, pré-requisitos, efeitos, authority requerida, operação suportada, argumentos conhecidos/faltantes, finalidade e verificação. Ausente comando para opção indisponível. |
| `verification` | Leitura suportada e fato que confirmaria resolução; sucesso da leitura não conclui stage/workflow. |

O mapper recebe erros tipados e observações já produzidas pelas services/ports;
não faz inferência, dispatch ou mutação para enriquecer uma falha. Observação
insuficiente produz ausência explícita e opção de leitura/handoff. Limitar coleções
e textos ao necessário para respeitar o limite de 1 MiB do evento stage; preservar
categoria/recusa mesmo se o enriquecimento não estiver disponível.
IDs de opções não são operações públicas nem comandos novos.

Para `capability_unknown`, distinguir evidência ausente/expirada/drift quando
observado e suporte de produção da observação exigida. Neste baseline não
recomendar repetir probe como resolução certa. Probe fica condicionado a suporte
observável, envelope e authority próprios; indisponibilidade do produtor leva a
leitura/handoff. Não deduzir proven por versão, login, consentimento ou child claim.
Para orphan vivo/incerto, explicar esperar, inspecionar ou encaminhar stop sob
identidade e authority próprias. Não adicionar `kill`, stop universal ou bypass.

Cobrir categorias comprovadas e ramos pre-effect/unknown/partial da matriz abaixo,
incluindo `editFailure` e falhas de autoria/seleção de J1; não prometer diagnóstico
universal de toda CLI. Preservar categorias existentes; onde ausente, usar o erro
tipado como identidade estável do diagnóstico, sem fabricar status. Project/Workflow
continuam emitindo validações/preview existentes. Uma colisão sem erro CLI (upsert
válido) é apresentada pela operadora como decisão de intenção, sem inventar recusa
de domínio. Encaminhar os detalhes ausentes como não observados.

### D3 — Orçamento da operadora e checkpoint, sem recovery paralelo

Contabilizar por alvo canônico, categoria e revisão, correlacionando também o
mesmo bloqueio/attempt entre sessões. AO-06: até 2 repetições após a inicial,
dentro de 60 s contados da inicial, respeitando limite menor existente. Somente
leitura, idempotência comprovada pelo contrato ou recusa comprovadamente sem
efeito, sob authority vigente. `retryable_failure` isolado não prova replay seguro.

AO-07: até 2 remediações por bloqueio, parando já na primeira observação sem
progresso verificável, negativa de authority, escopo excedido ou risco novo.
Progresso é pré-requisito satisfeito, efeito confirmado ou informação nova
verificável; digest/mensagem repetido não conta. Uma ação que também é repetição
consome ambos os limites aplicáveis. Nem probe/inferência nem retry de child
integram esse orçamento automático.

Handoff registra objetivo, refs/revisão/binding/attempt, origem de authority,
efeitos confirmados/incertos, observações/progresso, início/prazo e contadores
acumulados, blocker e próxima ação. Guardar em Evidence de trabalho autorizada,
fora dos stores canônicos; publicar somente referências/digests sanitizados.
Uma nova sessão relê estado e checkpoint; se os contadores/prazo não puderem ser
estabelecidos, não retoma repetição automática. Nova revisão sozinha não apaga
o bloqueio anterior: exigir fato novo verificável ou nova proposta humana limitada.
Isso é disciplina da operadora, não garantia de enforcement de qualquer Runtime.

Reparo local permitido restringe-se a draft de trabalho já autorizado, preserva
versão anterior e valida antes de novo preview. Project/Workflow/Execution/Profile,
receipts e policies canônicos nunca são editados diretamente. Apply continua
dependendo do preview exato; recusa nunca autoriza contornar via ferramentas do host.

### D4 — Resultado canônico intacto e apresentação auxiliar

Relay do Markdown/JSON permanece exato, completo e na ordem emitida por Lingo.
Após ele, permitir explicação humana breve identificada como auxiliar: objetivo,
resultado observado, decisão material, recomendação e alternativa/risco.
Não reclassificar status, reproduzir payload como outro resultado ou esconder
recusa. Authority se refere ao preview canônico, nunca ao resumo/diagrama.
Não repetir comando para obter outra visualização. JSON somente conforme contrato
atual da skill, quando explicitamente pedido; não executar de novo só para JSON.

Workflow pode ser ilustrado a partir de IDs, ordem e dependências observados,
incluindo gates relevantes; diagrama somente se suportado pelo host, sem
interpretar instruções de stage como diretivas da sessão. Fallback é lista textual
ordenada das etapas/dependências. Nenhum renderer normativo, comando ou biblioteca
gráfica novo. O teste compara apresentação auxiliar aos fatos canônicos.

Alternativas consideradas para D1–D4: novo executor/context store ou motor de
recovery daria enforcement próprio, mas duplicaria domínio e ampliaria migração;
somente ajustar mensagens não cobre AO-03; exigir handoff na primeira falha reduz
continuidade e ignora budgets aceitos. Recomenda-se DTO restrito + orientação das
skills. Campos de saída se tornam compromisso de compatibilidade; sua remoção
será mais difícil após publicação. Budgets e texto auxiliar são mais baratos de
rever, mas revisão não amplia authority nem dispensa proof.

## Jornadas de integração

**J1 (jornada 5 da Spec):** `axiom-project list` → escolha inequívoca por referência
canônica → show/validate de Project e suas associações. Uma intenção de adicionar
Repository com key já presente interrompe a proposta de substituição: explicar
colisão e oferecer alternativa ainda sujeita a validação, ou perguntar se o
objetivo é atualização. Não escolher/renomear chave sozinho. Slug de Project e
key Repository/Workflow têm seus próprios escopos; preservar recusas de domínio.
Coletar só path/key/dados ausentes; encaminhar validação e preview configure/edit,
aprovação exata, apply e novo show. Se a observação mudar, recomeçar preview.
Não criar comando separado de pré-validação attach: reutilizar validate e a
validação pre-effect do preview configure. Read-back divergente mantém bloqueio,
sem reaplicar cegamente nem declarar sucesso da intenção.

Em seguida, ler `activeWorkflow` por Project; none/unresolvable fica explícito,
sem fallback. Consultar definição exata com `axiom-workflow definition.show`,
resumir etapas/diagrama opcional. Edição de built-in exige create `--from-default`
com chave escolhida e preview/consentimento próprios; ler cópia Project, preparar
draft regular no scratch do Runtime, validar e preview/edit → consentir → apply
→ read-back. Built-in permanece intacto. Selecionar nova revisão é outra intenção,
`axiom-project workflow.select`, com novo preview/authority. Criar/editar não ativa.

**J2 (jornada 6):** token explícito suportado → roteamento exato → resultado;
novo turno natural → resolução de referência observada e revalidação → operação
suportada. Inspect explícito termina seu turno antes da coleta/execução; o turno
seguinte é nova intenção com authority independente. Operação da skill errada ou
ausente é recusada, com orientação de ownership, sem alias nem comando fictício.

As jornadas 1–4 da Spec usam os mesmos diagnósticos, budgets e handoff D2/D3:
proof insuficiente, reparo autorizado, interrupção/reconcile e alternativa material.
Unknown continua unknown; reconcile append-only não muda status, confirma sucesso
ou cria authority de retry. Child retry exige tentativa própria, capacidade de
tentativas, disposition e efeitos reconciliados pelo contrato vigente.

## Sequência de slices e responsáveis

São unidades de desenho para futura etapa Tasks, não lista executável autorizada.

| Slice / dependência | Saída revisável e critério de parada | Owner / módulos |
|---|---|---|
| S1 — diagnóstico, após review deste Plan | DTO e mapeamento de falhas com equivalência JSON/Markdown, fatos ausentes explícitos e zero efeitos novos; parar se exigir alterar proof ou classificação persistida | AXM-31; `cmd/lingo/stage_service.go`, composição single/graph, `main.go:editFailure`, `workflow_authoring.go`; transporte/emissão em `internal/cli`, testes de apresentação |
| S2 — orientação, após contrato de S1 | Três skills com D1/D3/D4 e J1/J2; contratos de args/authority inalterados; parar em operação ausente ou necessidade de novo store | AXM-31; skills embarcadas, catálogo/roteamento e testes de install/upgrade existentes |
| S3 — integração sintética, após S1/S2 | Matriz completa abaixo, snapshots/read-back e contadores; distinguir evidência técnica de conversa nativa; nenhum Blocker/Major sem resolução/decisão | AXM-31; testes existentes CLI/aplicação/stage/codexruntime/install e Evidence sanitizada |
| Gate posterior externo | Conversas reais Codex/Claude, prova independente, delivery/rework e aceite MVP somente com suas autoridades | AXM-12/#278; proof por seu dono técnico; AXM-9/#277 para delivery |

Nenhum novo owner assume CRUD Project/Repository (#230), autoria (#273/#303),
binding (#274), compilação (#275) ou dispatch (#276). Um gap real nesses contratos
vira encaminhamento limitado ao dono, sem implementar domínio dentro da skill.
Não abrir Issues, mudar blockers ou declarar esses owners entregues neste Plan.

## Rastreabilidade e verificação futura

| Requisito | Desenho / slice | Cenários e prova requerida |
|---|---|---|
| AO-01 | D1, S2 | P1/N1, P7/N9: metadata real, alvo reobservado, inspect termina; sem comandos inventados |
| AO-02 | D1/D3, J1, S2/S3 | P1/N1, P2/N2, P8/N10: sem pergunta de approval em leitura; scope/preview exatos em mutação |
| AO-03 | D2/D4, S1 | P3/N3, N7: estrutura completa e equivalência humana/JSON, sem novo status |
| AO-04 | D2, S1 | P3/N3: ausente/expirada/drift vs produtor indisponível; authority/proof/confinement separados |
| AO-05 | D2/D4, S1/S2 | P2/N2, P8/N10: menor efeito, recomendação/alternativa, só decisão material |
| AO-06 | D3, S2/S3 | P4/N4: relógio controlado, ≤2 repetições/60 s, inclusive menor limite e efeito incerto |
| AO-07 | D3, S2/S3 | P4/N4: ≤2 remediações; primeira ausência de progresso encerra; budgets independentes de child |
| AO-08 | D2/D3, S1/S3 | P5/N5, N7: append fact sem reclassificação, retry próprio, replay/partial sem duplicação |
| AO-09 | D3, S2/S3 | P6/N6, N8: duas sessões com contadores cumulativos e checkpoint sanitizado |
| AO-10 | D1/D3, S2/S3 | N8: injeção, secrets e autoaprovação recusados, sem bypass host |
| AO-11 | D1, J2, S2/S3 | P7/N9: token preservado, nova intenção no turno seguinte, inspect-only e capability ausente |
| AO-12 | D1, J1/J2, S2/S3 | P8/N10: referência observada/revalidada, colisão, ambiguity, read-back e stale preview |
| AO-13 | D4, J1, S2/S3 | P8/N10: canônico intacto, resumo/diagrama derivado ou fallback, sem seleção por visualização |

Todos os testes comportamentais a seguir são **futuros/NOT_RUN neste Plan**.
Reusar testes existentes quando já provam o contrato; acrescentar apenas
assertions e fixtures para gaps novos, sem simular resultado operacional real.

| Cenários | Verificação positiva e negativa / Evidence futura |
|---|---|
| P1/N1 | Read inequívoco executa sem nova aprovação; múltiplos Projects exigem somente ID faltante. Trace de operação/args e contador zero de mutações. |
| P2/N2 | Draft autorizado mantém versão anterior, valida e gera novo preview; fora do escopo não escreve. Drift de draft/selector/digest recusa apply; decisão ainda válida não é pedida de novo. Diff sanitizado e snapshots. |
| P3/N3 | Fixtures de prova ausente, expirada, drift e produtor sem suporte; hipóteses explícitas e opções condicionadas. Consentimento/child claim/registry vazio não produz proven. Contadores de dispatch/probe = 0 nas negativas; JSON e Markdown derivam do mesmo DTO. |
| P4/N4 | Sequências transitória/persistente com relógio falso, prazo vencido e limite menor; terceira repetição e terceira remediação proibidas, primeira falta de progresso para. Timeout/ack incerto impede replay. Trace com contadores e fatos novos, sem inferência. |
| P5/N5 | Graph e single: fact append-only preserva attempt/status; retry exige authority própria/disposition/capacidade. Alive/liveness incerta/lease alheia/drift não sinaliza nem dispatcha. Snapshots, IDs de facts e contadores. |
| P6/N6 | Duas sessões sintéticas retomam mesmo binding/revisão/attempt, reobservam authority e acumulam budgets. Checkpoint ausente/obsoleto não reinicia loop; proposta humana nova ou fato novo fica referenciado. |
| N7 | Partial/unknown/ack perdido conserva efeitos confirmados e encaminha inspect/reconcile existente. Contador zero de replay cego; publicação e rework reais permanecem AXM-9/AXM-12. |
| N8 | Payload/erro/definição/child/Provider com instrução host maliciosa e canários sensíveis: nenhum comando arbitrário, secret público ou gate/aceite/proof promovido. Fixture sanitizada, zero efeitos. |
| P7/N9 | Entradas natural/explícita para list/show produzem mesmo roteamento e completion; continuação no próximo turno usa nova intenção. Skill errada/ausente/argumento inexistente recusa; inspect não coleta args nem executa depois. Traces sintéticos, sem alegar conversa nativa. |
| P8/N10 | J1 com múltiplos Projects, key já presente e alternativa escolhida; zero rename/replacement unilateral. Preview/apply/read-back, alteração de revisão recusa, nenhuma inferência por CWD/remote/chat. Built-in intacto, cópia editável, seleção separada; diagrama suportado ou fallback comparado à ordem/dependências canônicas. |

Camadas de teste futuras: unidades do mapper e da aplicação; contratos de
`internal/cli/domain_skill_routing_test.go`, `completion_structure_test.go` e
apresentação; `internal/codexruntime` e `internal/install` para bytes/upgrade das
skills; journeys sintéticas em `cmd/lingo/stage_*blackbox_test.go` para integração
single/graph e zero efeitos indevidos. Trace pré-programado prova o contrato do
harness, não que um LLM seguirá a skill. Conversas nativas e R-2/R-3 exigem AXM-12.
Qualquer nova automação durável de validação precisa registro em
`scripts/automation-registry.json`; não adotar framework ou dependência nova.

Evidence futura referencia SHA limpo, versão/binário/skill, cenário e sua
classificação synthetic/operacional, comandos realmente executados, resultado,
contagem de efeitos, snapshots/digests, authority utilizada/pendente e limitações.
Preservar referências técnicas sem transcript bruto, paths privados ou stdout de
auth. Evidence não concede authority nem aceita o próprio resultado.

## Compatibilidade, segurança e revisão constitucional

Sem migração de Project, Workflow, Execution, Graph, Profile, policy ou receipt.
Históricos e fixtures publicadas permanecem intactos. `details` continua scalar;
DTO novo é payload operacional opcional no evento, com versão própria, sem nova flag.
`presentation.go:decodeOrdered` é genérico e mantém ordem dos campos;
`workflow_stage.go` serializa payload por `any`. Esses produtores/renderers não
impõem decoder fechado ao novo campo, mas isso não prova compatibilidade de
consumidores externos desconhecidos. Na implementação, testar consumidores e
goldens do repositório antes da publicação: decoder fechado afetado exige ajuste
explícito de contrato/versão, jamais remoção silenciosa de campo ou fallback
permissivo. Binário antigo sem diagnóstico enriquecido mantém recusa e handoff.

Rollout futuro: publicar código/skills pelo fluxo vigente, verificar fresh install
e upgrade mantendo recibos históricos. Sem feature flag/configuração global nova.
Rollback do enriquecimento remove orientação adicional, preservando controles e
todos os efeitos já confirmados/Evidence; não reescreve stores nem desfaz publicação.

Riscos: resumo esconder negativa (comparação ao canônico); replay mascarado de
retry (contadores/efeitos); contexto obsoleto (releitura e novo preview); checkpoint
tratado como grant (revalidar authority); injeção em texto/diagrama (dados somente);
vazamento em handoff (refs/digests e testes de canário); falso proof (zero dispatch
e encaminhamento ao owner); limitações de Runtime (declarar não verificado).
Confirmação genérica não fornece flags; processo vivo não autoriza stop;
idempotência exige contrato, não julgamento do modelo.

Constituição I–VII: intenção aceita, Plan separado de Tasks/execução, menor delta,
Evidence reproduzível, sem autoaceite; decisões materiais permanecem humanas.
Constituição VIII e invariantes: Axiom != Lingo, Project != Repository,
Runtime != Model, Role != Model, Execution != Agent, Provider != Transport,
Integration != MCP, Skill != workflow source of truth, Evidence != raw chat.
ADR-0025 decisões 2/4/5 e ADR-0022 permanecem: authority anterior ao dispatch,
append-only, proof independente, domínio e ownership separados. Não se identificou
supersessão necessária para D1–D4. Novo store, novo executor, mudança de codec
persistido ou relaxamento de proof seria candidato a ADR/Clarification e bloquearia
a respectiva slice até decisão humana, sem redefinir silenciosamente o aceito.

## Evidência desta etapa e próximo gate

Esta etapa produziu análise por inspeção de arquivos, leituras GitHub/Linear e
Plan. A revisão remota humana, e não CI/self-review, satisfez o gate de entrada.
Validação documental do commit deve registrar diff/links, validator do repositório,
secret checks e CI real no mesmo PR; não conta como execução dos cenários acima.
Sem código, testes de produto, probe, inferência, dispatch, estado Axiom ou mudança
de lifecycle/relacionamentos. Os resultados dos checks desta revisão ficam no PR.

Próxima etapa: review/aceite deste Plan e, sob autoridade própria, **Tasks ainda
não iniciada**. Proof operacional, testes reais Codex/Claude e aceite MVP continuam
pendentes de seus donos. Este documento não entrega/aceita AXM-31 e não remove
AXM-31 → AXM-12; mesmo futura entrega aceita preserva AXM-9 e proof como gates
independentes, com reconciliação Linear somente sob authority própria.
