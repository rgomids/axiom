# Specification 007 amendment — AXM-31: autonomia responsável da IA operadora

## Status, autoridade e escopo

**Proposed — specify, 2026-10-17.** Requer review e aceite humano desta
Specification; não é Plan, Tasks, implementação ou decisão arquitetural aceita.
Work Item: [AXM-31](https://linear.app/rgomids/issue/AXM-31).
Diretriz de produto: [Usuário → IA operadora → Axiom CLI](https://app.notion.com/p/3fbe01f2262681dabefbde5c7f626104),
aprovada em 16/10/2026 conforme contexto autorizado pelo mantenedor.
Este texto é uma proposta técnica original baseada nesse objetivo autorizado;
não importa transcrições, dados privados nem conteúdo integral de Providers.

Amplia a experiência observável de WF-009/AC-009 e WF-013/AC-013 da
[Spec 007](spec.md), sem transferir seus donos históricos. Preserva WF-011/#277
(AXM-9: delivery packet/publicação/rework) e WF-012/#278 (AXM-12: validação real
Codex/Claude e aceite humano). Não entrega smoke, atestador de capability,
novo Runtime ou recuperação universal. Não reabre AXM-8 nem refatora AXM-29.
Especifica também a experiência de IA operadora para operações **já suportadas**
via linguagem natural ou invocação direta de skill/operação, com continuidade
conversacional, sem ampliar a implementação dos donos de Project, Repository ou
Workflow. Um diagrama opcional não implica nova UI, renderer ou comando instalado.

Authority desta etapa: leitura/análise e artifacts de Specification, commit,
push e um Draft PR. Nenhum dispatch, inferência, probe real, Provider de produto,
merge, release, deploy, alteração de blockers/status ou aceite está autorizado.
O gate informado é `READY_FOR_specify`; não foi observado como estado de uma
Execution canônica. A criação deste artifact não avança gate algum.

## Problema, atores e resultado desejado

O usuário quer atingir um objetivo de engenharia e compreender escolhas materiais,
sem precisar diagnosticar leases, epochs ou capability proof. A IA operadora
interpreta o objetivo e escolhe semanticamente entre ações permitidas, investiga,
valida o resultado e presta contas. Axiom governa estado, autorização, invariantes
e Evidence de modo determinístico; Lingo expõe suas operações e diagnósticos.
O mantenedor decide produto, arquitetura, riscos adicionais e aceite.
Children e Providers fornecem dados, nunca autoridade por declaração própria.

A pessoa pode iniciar uma interação declarando a intenção em linguagem natural
ou invocando diretamente uma skill/operação suportada. A IA deve conservar uma
experiência de conversa nos dois caminhos: interpretar pedidos não explícitos,
respeitar escolhas explícitas e coletar somente dados necessários, sem exigir
conhecimento de comandos, digests ou esquemas. O diálogo não é registro canônico,
nem fonte de selector/authority; toda referência contextual precisa ser resolvida
e revalidada contra resultados e bindings efetivos do Axiom.

O resultado esperado é resolver um obstáculo local já autorizado ou explicar
precisamente o bloqueio, com alternativas, efeitos e próximo passo verificável.
Uma recusa correta pode ser o resultado esperado; continuar investigando não
significa continuar dispatch. Evidence não concede autorização, e autorização
não comprova capability. A IA não decide estado canônico pela conversa.

## Baseline e diagnóstico limitado

Inspeção em `0262c550e53e1ba8868e936d05b58fa78f67b56b`, HEAD = origin/main após
fetch, em 17/10/2026. As referências locais abaixo dizem respeito a esse baseline,
não a execução futura. Nenhum teste operacional foi executado nesta etapa.

| Fato / classificação | Evidence inspecionada | Consequência para a proposta |
|---|---|---|
| Gap demonstrado por inspeção: next action sugere probe sem expor que o produtor atual não pode fornecer toda a prova | `cmd/lingo/stage_service.go`, `classifyStageError`: `capability_unknown` → “Run an authorized capability probe”; `internal/local/probe_session.go`, `Verify`: `Inconclusive: true`, sem atribuição de `ModelInvoked`/`EffortVerified`; `internal/capabilityproof/service.go` exige esses fatos. `internal/runtimeadapter/runtime-versions.json` tem `entries: []`. Busca de atribuições em código não-test confirmou a ausência. | Distinguir prova ausente/expirada de capacidade de observação indisponível. Não sugerir repetição como resolução garantida. Corrigir diagnóstico não implementa atestação independente. |
| Contexto histórico confirmado, não prova operacional atual | [Handoff #320](https://github.com/rgomids/axiom/pull/320#issuecomment-6208213568) descreve o bloqueio; [registro versionado](issue-276-implementation.md) registra qualificação instalada FAKE e T40/AC-6 NOT_RUN; #320 consta MERGED, commit `eed4aace5c3d223f0588d14c21fcfc8ac06659ba`. | O handoff é histórico. A inspeção atual sustenta o gap de orientação; não sustenta que todos os demais gaps daquele comentário ainda existam. |
| Contrato já atendido por código, sem nova execução de testes aqui | `internal/stageexecution/reconcile.go`: preview reobservado, fact append-only, status inalterado; `cmd/lingo/stage_service.go`, `stage_single.go`, `stage_dispatch.go`: retry explícito e disposition; testes `cmd/lingo/stage_adversarial_blackbox_test.go` e `stage_journey_blackbox_test.go`. | Reutilizar essas fronteiras. Não construir outro recovery engine nem resetar tentativas. |
| Gap de orientação por inspeção | `classifyStageError` devolve categoria e uma string `next`; `stop_orphan_then_reconcile` pede parar grupo órfão sem ali discriminar opções, prova de identidade e autoridade de sinalização. | Oferecer opções condicionadas; nunca interpretar a mensagem como permissão para matar processos. |
| Contrato já atendido parcialmente; hipótese de fricção conversacional | `internal/codexruntime/skills/axiom-work-item/SKILL.md` já resolve intenção, lê estado, preserva unknown e exige decisão humana para preview exato. Não foi observada conversa real de usuário nesta etapa. | Reforçar diagnóstico e continuidade sobre esse roteamento, sem alegar falha de Codex/Claude nem dispensar consentimento. |
| Entrada dupla parcialmente prevista nas skills; lacuna de aceite conversacional | `internal/codexruntime/skills/axiom-project/SKILL.md` e `axiom-workflow/SKILL.md` distinguem operações explícitas de intenção natural e expõem roteamento; `axiom-work-item/SKILL.md` também. As skills preservam resultados canônicos e impedem inferência de selector por chat. Não houve teste ponta a ponta de conversa com transição entre skills. | Especificar equivalência de entrada, continuidade por referências verificadas e apresentação humana sem interpretar texto de chat como estado, prova ou authority. A proposta não afirma nova capability instalada nem determina alteração dessas skills sem Plan. |

Revalidação de descoberta: `gh pr list --state all --search AXM-31` retornou
vazio; lista completa de PRs abertos mostrou somente #317. `git ls-remote
--heads origin` não mostrou branch AXM-31; busca de artifacts versionados não
localizou especificação/Evidence anterior desse Work Item. Linear retornou Todo,
um comentário com o prompt de specify e relação AXM-31 blocks AXM-12; AXM-12
também depende de AXM-9. Nenhum desses fatos prova Execution iniciada.
O diretório de estado efetivo desta instalação, resolvido conforme
`cmd/lingo/main.go:stateRoot` e `internal/local/state_root.go`, não existe.
**Execution canônica permanece desconhecida**, sem inferir ausência em outro host
ou reconstruí-la a partir de GitHub/Linear. Não foi criado estado Axiom.

## Journeys delimitadas

1. **Diagnóstico sem interrupção técnica desnecessária.** Ao receber
   `capability_unknown`, a IA lê diagnóstico e referências do alvo exato.
   Se falta prova mas há caminho de observação suportado, apresenta o probe e
   seu envelope para autoridade própria. Se o produtor não suporta a prova,
   explica a limitação, oferece leitura/handoff e não repete inferência.
   Nenhum dos ramos transforma unknown em proven.
2. **Reparo local autorizado.** Uma configuração de entrada inválida impede
   preview. A IA propõe o menor ajuste em artifact de trabalho local abrangido
   pelo escopo, guarda a versão anterior, valida e refaz preview. Apply de
   configuração canônica continua exigindo decisão para seu preview exato.
   Se o ajuste muda binding/efeitos, a autoridade anterior não cobre a mutação.
3. **Interrupção e continuidade.** Outra sessão recebe referências de uma
   tentativa incerta, lê estado atual e observa lease/processo. Não reexecuta
   child nem sinaliza PID por suposição. Faz diagnóstico e, com autoridade
   própria, reconciliação append-only; retry requer nova autoridade, capacidade
   disponível e disposition. Se identidade/liveness segue incerta, entrega
   blocker recuperável com próxima observação, preservando a tentativa.
4. **Alternativa com risco material.** Reset, nova inferência ou efeito externo
   é uma opção separada: mostrar impacto e pré-requisitos; obter consentimento
   aplicável e revalidar preview. Recusa de consentimento leva a alternativa
   segura ou handoff. Uma publicação incerta é encaminhada à reconciliação de
   AXM-9, não repetida nem implementada novamente por AXM-31.

5. **Administração conversacional com referências canônicas.** O usuário pede
   os Projects disponíveis, escolhe inequivocamente um deles e solicita associar
   um Repository com chave já existente. A IA lê os dados do Project e demonstra
   o conflito sem mutar; oferece chave alternativa e só a usa após escolha.
   Reúne informações faltantes, valida, obtém preview/autorização exatos, aplica
   com operações de Project e confirma a associação por nova leitura. Em seguida
   consulta o Workflow selecionado, mostra uma descrição compreensível das
   etapas, opcionalmente diagramada pelo host, e encaminha uma edição da etapa
   escolhida pelo contrato de autoria de Workflow. Quando o selecionado for o
   default/built-in imutável, explica a necessidade de criar uma cópia editável
   no Project antes de modificar a etapa; criar/editar não ativa a nova revisão.
   Selecioná-la depois é uma operação separada, com preview e authority próprios.
   Nenhuma dessas operações é criada por AXM-31.
6. **Entrada explícita de skill seguida de conversa.** O usuário invoca uma
   operação suportada de `axiom-project` ou `axiom-workflow`: a IA preserva a
   operação exata, consulta o resultado e responde. No turno seguinte,
   "agora altere esta etapa" pode ser tratado como nova intenção; a IA revalida
   referência/revisão fornecida pela leitura anterior ou pede identificação
   quando não houver vínculo inequívoco. Invocar somente `skill inspect`
   permanece inspeção, não aplicação. O resultado, exigências de autoridade
   e refusals são idênticos aos da entrada natural.

## Requisitos propostos

Os MUST abaixo só se tornam normativos após aceite. IDs AO são locais a este
amendment; não renumeram WF/AC históricos.

| ID | Comportamento observável |
|---|---|
| AO-01 | A IA MUST usar descoberta de operações/argumentos e resolução de alvo disponíveis no Runtime instalado; não presumir flags, modelos, plugins, Runtime equivalente suportado ou acesso a Provider. Ler estado do alvo exato antes de continuidade. Uma operação explícita de inspeção continua limitada à inspeção. |
| AO-02 | Leitura/diagnóstico permitido MUST prosseguir sem nova confirmação quando alvo/escopo forem inequívocos. Reparo local reversível MUST estar abrangido pela autoridade, preservar conteúdo anterior e ser validado. Mutação canônica mantém preview/apply, decisão humana exata e checks existentes; “reversível” não é exceção a eles. Reutilizar uma decisão ainda válida para o mesmo preview evita nova pergunta; intenção genérica não preenche flags de autoridade. |
| AO-03 | A falha MUST expor os componentes descritos abaixo de forma estruturada e legível por máquina, com representação humana equivalente; preservar completion canônico, categoria, provenance, references, next e details. Não acrescentar `unknown`/`blocked` a completion.Status nem criar segundo protocolo de resultados. |
| AO-04 | Diagnóstico MUST separar observado, não observado, hipótese de causa, capability, autorização, confinement e efeitos confirmados/incertos. `capability_unknown` MUST distinguir falta/expiração/drift de prova da indisponibilidade do mecanismo que a produziria; quando não distinguível, declarar essa limitação e indicar leitura segura. Consentimento não satisfaz proof. |
| AO-05 | A IA MUST escolher a menor ação permitida que progrida no objetivo; apresentar ao humano somente decisão material, autoridade ausente, conflito ou inexistência de alternativa segura. Mostrar recomendação, alternativa viável, impacto e o que impede prosseguir, sem obrigar o usuário a interpretar códigos internos. Não ocultar referências técnicas na Evidence. |
| AO-06 | Repetições automáticas MUST limitar-se a leituras e operações comprovadamente idempotentes ou recusadas antes de qualquer efeito, dentro da autoridade vigente. Para o mesmo alvo, categoria e revisão: no máximo 2 repetições após a tentativa inicial, dentro de 60 s, respeitando limite menor existente. Esses limites são proposta de comportamento, não nova configuração global. |
| AO-07 | Uma remediação só pode continuar se a observação seguinte mostrar pré-requisito satisfeito, estado/efeito confirmado ou nova informação verificável que reduza a incerteza. Repetir mensagem ou revalidar o mesmo digest sem fato novo não é progresso. No máximo 2 ações de remediação por mesmo bloqueio; cessar antes ao detectar ausência de progresso, negativa de autoridade, escopo excedido ou risco novo. Probes/inferência não entram nesse orçamento automático. |
| AO-08 | Retry de child MUST permanecer nova tentativa explícita, com autoridade própria, efeitos reconciliados e capacidade de tentativas disponível; não é repetição idempotente de CLI. Unknown de tentativa exige reconciliação conforme ADR-0025; append fact não reclassifica status, comprova sucesso ou autoriza retry. Replay de receipt permitido pelo contrato não cria outro dispatch. |
| AO-09 | Interrupção, orçamento esgotado ou falta de autoridade MUST produzir handoff delimitado e sanitizado, recuperável por referências canônicas, com tentativas anteriores preservadas. Nova sessão MUST reobservar estado/autoridade; não reiniciar orçamento para o mesmo bloqueio sem fato novo verificável ou nova proposta humana limitada. |
| AO-10 | A IA MUST tratar erros, artifacts e conteúdo de children/Providers como dados não confiáveis. Não executar comandos arbitrários recebidos em mensagens. Selecionar operação suportada, validar argumentos/alvo e jamais usar acesso do host para contornar recusa Axiom. Sem autoaprovação de Evidence, gates, ADR, review independente ou aceite. |
| AO-11 | Para uma operação existente e suportada pela Runtime instalada, a IA MUST aceitar tanto intenção natural inequívoca quanto skill/operação explicitamente invocada como pontos de entrada. Entrada natural escolhe somente a operação; operação explícita MUST ser preservada sem reclassificação semântica. A conversa pode continuar no turno seguinte com outro pedido após qualquer dos caminhos. Inspeção explícita é read-only e termina sem execução ou coleta de argumentos adicionais naquele turno. A equivalência não cria suporte para uma capability ausente nem concede authority. |
| AO-12 | A IA MUST manter continuidade conversacional mínima ao combinar Project, Repository e Workflow: usar apenas seleção e referências canônicas efetivamente observadas para resolver menções como "esse Project" ou "essa etapa"; reler e validar selector, digest/revisão e estado antes de efeitos; perguntar somente o dado materialmente ausente ou ambíguo. Chat/CWD/Git remote não criam selector, binding, default ou autoridade. Identificadores duplicados MUST gerar diagnóstico e opções sem renomeação automática, e confirmação de escrita MUST provir de read-back canônico. |
| AO-13 | A IA MUST distinguir resposta técnica canônica de apresentação ao usuário, oferecendo resumo claro de Project, Repository, definição/etapas e risco sem inventar estados, comandos ou capacidades. Quando o host suportar representação de Workflow em diagrama, ela MAY oferecer visual derivado dos dados canônicos; caso contrário, MUST fornecer alternativa textual compreensível. A visualização é leitura, não modifica definição/seleção; explicação adicional não substitui nem altera o resultado canônico emitido por Lingo. |

### Roteamento conversacional e fronteira de ownership

- **Project e Repository:** a skill `axiom-project` consulta `list/show` e
  encaminha associação/remoção/edição de Repository por `configure`, somente
  quando essas operações estiverem disponíveis na instalação e o Project exato
  estiver resolvido. Conflitos de chave vêm da validação do Lingo; propostas da
  IA não alteram identidade nem são mutações. Requerimentos de Resource Lifecycle
  permanecem com seu dono técnico, não com AXM-31.
- **Workflow:** `axiom-project` consulta e altera a **seleção**; a skill
  `axiom-workflow` consulta/valida/edita a **definição**. Mostrar diagrama
  derivado do contrato não seleciona nem publica revisão. Definição built-in
  é imutável: editar sua etapa exige primeiro cópia editável de Project por
  operação suportada, nunca alteração direta do default. Uma edição deve
  preservar a versão anterior, validar a nova definição e obter preview exato;
  seleção exige operação e consentimento separados.
- **Work Item/Execution:** `axiom-work-item` continua responsável por iniciar,
  consultar e conduzir trabalho nas operações suportadas; AXM-9 retém
  delivery/rework e AXM-12 retém demonstração operacional end-to-end.
- **Entrada e continuidade:** invocar uma operação explícita não perde a
  possibilidade de nova conversa; contexto observado ajuda a identificar uma
  referência **somente após** resolução inequívoca e revalidação canônica.
  Resultados Markdown/JSON canônicos mantêm sua semântica/ordem; texto auxiliar
  ou diagramas não os contradizem. O Plan deve verificar os contratos reais de
  cada skill e Runtime para essa apresentação, sem criar um segundo renderer
  normativo nem prometer diagrama onde não há suporte.

As jornadas acima são **acceptance contracts propostos**, não Evidence de que a
experiência completa já opera. Operações ausentes e limitações de renderização
devem ser registradas e encaminhadas ao owner correto; a skill não poderá
contornar uma recusa por CLI ou editar stores diretamente.

### Contrato observável de falha

Dados semânticos necessários, dentro do completion/payload/details existentes;
nomes de campos e codec final ficam para Plan, sem prometer flags novas:

- **Identidade:** operação recusada, categoria estável, alvo e referências de
  Execution/stage/attempt quando conhecidos, revisão observada; ausência explícita
  quando não houver verdade local suficiente.
- **Causa e estado:** fato observado, hipótese identificada como tal,
  pré-requisito faltante, efeitos confirmados e incertos separadamente.
- **Opções seguras:** lista limitada a 3, ordenada de diagnóstico/menor efeito
  para maior risco; para cada uma, disponibilidade (`disponível`, `condicionada`
  ou `indisponível`), pré-requisitos, efeitos, autoridade requerida e por que
  ela pode resolver ou apenas investigar. Se nenhuma ação de resolução for
  segura, incluir leitura/handoff, com blocker explícito.
- **Próxima ação:** operação/comando realmente suportado com alvo conhecido;
  parâmetros desconhecidos devem aparecer como pendência, nunca exemplo pronto
  para apply. Uma ação indisponível pode ser decisão/encaminhamento, sem comando
  fictício. Mutação mostra preview antes de apply e exige autoridade correspondente.
- **Verificação posterior:** leitura/validação e resultado que confirmariam
  resolução. Falha da verificação mantém a condição e interrompe ou muda para
  diagnóstico com fato novo; sucesso da leitura não é sucesso do workflow.

Exemplo sem executar: capability proof indisponível → observar identidade e
suporte do produtor (leitura); probe condicionado ao envelope autorizado **e** a
observação suportada; handoff para decisão sobre prova independente quando não
suportada. “Rodar o probe novamente” sozinho não atende AO-03/AO-04.

### Matriz de risco e authority

| Nível / ação | Autonomia permitida e pré-requisitos | Fronteira / verificação |
|---|---|---|
| A — leituras, status, diagnóstico, preview sem efeitos | Escopo/alvo claros; sem nova confirmação de leitura permitida; repetição AO-06 | Sem coletar secrets/raw chat; read-back preserva estado |
| A — ajuste de artifact local de trabalho reversível | Autoridade já concedida para esse artifact; cópia/diff anterior preservado; validar resultado | Não editar store/ledger/Evidence canônicos; sem expansão de roots, permissões ou objetivos |
| B — configuração canônica, reconcile, cancel, integração local | Preview atual e decisão exata; grants locais e titularidade exigidos pela operação | Reobservar revision/lease/identidade; cancel solicitado não é parada confirmada; reconciliação não promove status |
| B — stage retry, probe, inferência, reset, push/PR ou Provider | Cada efeito com autoridade própria, alvo/digest atuais e pré-requisitos comprovados; inferência exige autoridade externa; reset exige ownership/fencing/disposition | Nenhum efeito por mera classificação semântica; publication/rework permanecem AXM-9; perda de ack exige reconciliação, não repetição |
| C — bypass ou autoaprovação | Proibido: forjar proof/consentimento/resultado, alterar estado interno para liberar dispatch, ampliar grants, burlar CI/proteções, extrair credenciais, apagar tentativas ou aceitar pelo humano | Nem consentimento genérico, exit 0, registro do child, status Provider ou host access legitimam o atalho |

## Aceite mensurável e Evidence futura

Cada linha é requisito de entrega futura, **NOT_RUN nesta Specification**.
Fixtures são identificadas como synthetic; R-2/R-3 reais continuam AXM-12.

| Cenário / requisitos | Resultado esperado e medida | Evidence futura |
|---|---|---|
| P1/N1: leitura inequívoca vs alvo ambíguo; AO-01/02 | Leitura sem pergunta de aprovação; alvo ambíguo solicita somente dado faltante, zero mutações | Jornada CLI e roteamento das duas skills, eventos canônicos e contagem de efeitos |
| P2/N2: artifact local autorizado vs fora do escopo; AO-02/05 | Ajuste preserva versão anterior, valida e refaz preview; escopo excedido deixa artifact intacto e pede autoridade; nenhuma pergunta repetida para decisão exata ainda válida | Diff sanitizado, autoridade de origem, preview/read-back; fixture de drift recusa apply sem efeitos |
| P3/N3: proof ausente/expirada vs produtor sem suporte; AO-03/04 | Cada falha fornece identidade, estado, opções, autoridade, próxima ação e verificação; suporte indisponível não recomenda probe como resolução certa; consentimento/child claim não libera spawn | Saídas JSON/human equivalentes; registry vazio/prova expirada/drift; contador de dispatch = 0 nas negativas |
| P4/N4: indisponibilidade pre-effect transitória vs persistente; AO-06/07 | Transitória resolve após fato novo; persistente faz ≤2 repetições em ≤60 s, no máximo 2 remediações e handoff; primeira ausência de progresso interrompe remediação | Relógio controlado, sequência de observações/digests, contadores; sem Runtime real |
| P5/N5: tentativa unknown; AO-08 | Com identidade/parada comprovadas e autoridade exata, append fact; tentativa/status anteriores intactos; retry continua separado. Processo vivo, liveness incerta ou lease alheia impedem retry/stop indevido | Snapshot antes/depois graph e single, refs de fact/authority; contadores de sinalização/dispatch e negativas de drift |
| P6/N6: nova sessão; AO-09 | Handoff recupera mesmo alvo/binding/attempt, reobserva revisão e limites; nenhuma tentativa duplicada nem reinício do orçamento sem fato novo | Duas sessões sintéticas, refs estáveis, histórico de decisões/tentativas preservado e orçamento acumulado |
| N7: perda de ack/partial e publicação; AO-03/08 | Efeito confirmado preservado e efeito incerto explicitado; próxima ação aponta à inspeção/reconciliação existente, zero replay cego | Fixtures de resultado partial/unknown; publicação real e rework testados por AXM-9/AXM-12 |
| N8: injeção/segredos/autoaprovação; AO-09/10 | Comando host malicioso não executado; canário sensível ausente do handoff público; nenhum gate/aceite/proof promovido | Fixtures sanitizadas de conteúdo não confiável, assertions de zero efeitos e ausência de canários |
| P7/N9: intenção natural vs operação explícita/inspect-only; AO-01/11 | Para `list/show` suportados, ambas as entradas resultam na mesma operação e nos mesmos resultados canônicos; a skill explícita não é reinterpretada, e uma próxima intenção segue no turno seguinte. `skill inspect` encerra na inspeção, sem mutação ou nova execução. Entrada/skill indisponível não produz comando fictício. | Diálogos sintéticos em Codex/Claude quando disponíveis, tracing de roteamento, CLI args/resultado, nenhuma mutação no modo inspect. |
| P8/N10: contexto Project → Repository → Workflow e visualização; AO-02/05/12/13 | Project e etapa inequívocos são revalidados sem repetir dados conhecidos; chave duplicada apresenta alternativa sem aplicar; ambiguidade pede identificador; preview obsoleto recusa efeito; alteração aprovada é confirmada por read-back. Definição/seleção permanecem separadas; tentativa de editar built-in diretamente
  é recusada, enquanto cópia editável autorizada mantém o default intacto.
  Diagrama somente quando suportado e derivado dos dados; fallback textual sempre disponível. | Cenários sintéticos de múltiplos Projects e chaves duplicadas, diferenças antes/depois, previews digests, contagem de prompts, referências, recusa com zero efeitos, captura de apresentação gráfica ou fallback. |

Evidence de remediação/handoff MUST registrar objetivo delimitado, alvo canônico
ou sua indisponibilidade, baseline/binding/revisão, classificação synthetic ou
operacional, observações referenciadas, opção escolhida e motivo, autoridade
utilizada/pendente, tentativas e contadores acumulados, efeitos confirmados/incertos,
resultado da verificação, blocker e próxima ação. Registrar referências/digests
em vez de transcrições. Não exportar host paths, valores de ambiente, credenciais,
identificadores privados ou stdout bruto de autenticação. Diagnóstico local deve
respeitar as políticas de acesso e sanitização; Evidence de capability mantém
as restrições próprias da ADR-0025. Handoff não é novo store de autoridade.

## Compatibilidade, migração e revisão constitucional

Extensão comportamental aditiva sobre completion/help/skills e operações atuais;
não propõe nova entidade, policy engine, framework, daemon, provider de feedback
ou schema de Execution/Graph/Workflow. Registros históricos permanecem intactos;
sem migração, reconstrução, limpeza automática ou reinterpretação de unknown.
Clientes antigos continuam vendo categoria/next; Plan deve confirmar codecs
estritos antes de adicionar payload e versionar explicitamente se houver quebra.
Ausência de diagnóstico enriquecido mantém a recusa e orienta leitura/handoff,
nunca fallback permissivo. Desativar orientação nova mantém controles vigentes;
não reverte efeitos já confirmados nem remove Evidence.

| Contrato confrontado | Resultado / colisão evitada |
|---|---|
| Constituição I–VII | Intento e Evidence antecedem implementação; IA interpreta, Axiom valida; decisões materiais humanas; escopo mínimo, dados preservados. Nenhuma emenda constitucional proposta. |
| Constituição VIII e Spec 007 | Preservar Axiom != Lingo, Project != Repository, Role != Model, Execution != Agent, Provider != Transport, Integration != MCP, Skill != workflow source of truth e Evidence != raw chat. Não criar inferência de alvo por CWD/chat. |
| Spec 007, failurePolicy e public operations | Retry de child continua explícito e limitado, mutações preview/apply com decisão exata; diagnóstico automático não autoriza gate, apply ou dispatch. Sem mudança material desses contratos. |
| ADR-0025, decisões 2/4/5 | Autoridade antes de dispatch, reconciliação append-only, proof independente no modo exato e separado de consentimento/confinement permanecem. Nenhum novo atestador definido aqui. |
| Política Linear/GitHub | Todo/PR/merge não são Execution nem aceite. Não alterar status, blockers ou terceiros; referências ao PR são relacionais, sem closing keywords. |
| Skills Project/Workflow/Work Item e renderização | As três skills já expõem operação explícita e resolução de intenção; não inferir selectors de chat, não trocar status canônico por paráfrase nem confundir inspeção com execução. AO-11–13 propõem continuidade e apresentação auxiliar; Plan deve verificar a compatibilidade efetiva no Runtime, sem alterar silenciosamente contratos aceitos. |

**Não se identificou necessidade de supersessão para esta proposta restrita.**
Dispensar proof, aceitar declaração do child ou deixar consentimento promover
unknown seria colisão material, excluída desta proposta. Se escolhida depois,
exige nova ADR **Proposed**, seção `Supersedes` nomeando precisamente ADR-0025
decisão 5 (e contratos da Spec 007 afetados), alternativas/trade-offs, aceite
humano e reconciliação conforme ADR-0018 antes de implementação. Não editar a
ADR histórica silenciosamente ou tratar esse caminho como aprovado.

## Decisões abertas, impacto no AXM-12 e próximo gate

- **Aprovar este amendment, incluindo entrada dupla AO-11, referências conversacionais AO-12 e apresentação AO-13, além dos limites AO-06/07.** Recomendação: limites pequenos
  fixos para esta jornada; alternativa: exigir handoff após primeira falha, com
  maior interrupção, ou configurar orçamento global, com complexidade sem gap
  demonstrado. Limites são baratos de rever; autoridade e proof não mudam com eles.
- **Proof operacional segue pendência independente.** Manter recusa e reportar
  mecanismo indisponível é a recomendação neste escopo. Uma fonte de atestação
  independente compatível exigirá diagnóstico/decisão e correção bounded no dono
  técnico correspondente antes do dispatch real de AXM-12. Relaxar proof amplia
  confiança e torna mais difícil preservar as garantias; não é solução aprovada.
  Essa pendência não impede revisar o artifact de specify e não é declarada resolvida.

AXM-31 melhora a preparação de AXM-12 ao permitir diagnóstico, correções já
autorizadas e handoff com menos intervenção técnica; não promete que um Runtime
se tornará dispatchable. Para considerar satisfeita **a dependência AXM-31 →
AXM-12**, verificar entrega técnica integrada/publicada conforme política vigente,
aceite humano explícito de AXM-31 referenciando a revisão aceita e matriz P1–N8
com Evidence dos requisitos aplicáveis; nenhum Blocker/Major desses requisitos
pode permanecer sem resolução ou decisão humana explícita de escopo. O responsável
pela operação deve então reconciliar a relação no Linear sob autoridade própria.
Merge desta Specification isoladamente não satisfaz esse critério.

Mesmo depois disso, AXM-9 continua dependência independente; ausência de capability
proof ou autoridade real continua gate de AXM-12. Provas reais de autenticação,
R-2/R-3, jornada de publicação/rework e aceite do MVP permanecem exclusivamente em
seus donos. **Não remover bloqueio agora.**

Próxima etapa recomendada: **clarify/review e aceite desta Specification**,
concentrados nos limites e fronteiras acima. Plan depende desse aceite e de
autoridade posterior. Esta solicitação não executa plan nem implement.

## Validação desta etapa

- `./scripts/validate-repository.sh .`: PASS; inclui harness, governança ADR,
  registro de automação e checks de arquivos sensíveis. O próprio validator
  registra cenários comportamentais de Runtime como SKIP/UNVERIFIED.
- `git diff --check`: PASS. Verificação Python efêmera: 99 destinos locais de
  links Markdown existentes nos dois artifacts alterados; presença de AO-01–10
  e linhas P1/N1–P6/N6, N7/N8 confirmada. Isso prova estrutura, não comportamento.
- Revisão semântica e de segurança pelo Orchestrator: nenhum achado bloqueante
  identificado na proposta restrita; não é review independente nem aceite.
  Apenas este amendment e sua entrada no índice; sem código/testes modificados.
- `gitleaks` não está disponível neste ambiente; não instalado automaticamente.
  Nenhuma inferência, dispatch, probe real ou teste operacional foi executado.
  A revisão humana da Specification e a decisão de proof continuam pendentes.
- Adendo de revisão conversacional no mesmo PR: AO-11–13, jornadas 5–6 e P7/N9,
  P8/N10 foram adicionados como proposta após reconciliação do AXM-31. A
  verificação estrutural e o CI desse adendo devem ser reportados pelo head
  atualizado; não tratar os PASS anteriores como validação deste novo commit.
