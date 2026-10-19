# Tasks 009 — Jornada guiada de configuração Project/Runtime

## Status, provenance e autoridade

**Proposed — tasks, revisão 1, 2026-10-18.** Nenhuma task aceita, iniciada ou
concluída nesta etapa. Este documento decompõe, sem alterar contratos, o
[Plan rev.1 aceito](https://github.com/rgomids/axiom/blob/2a9bdd0e62d6af127a3cce0f3d18141088a2aaa9/docs/specifications/009-guided-project-runtime-configuration/plan.md)
no SHA `2a9bdd0e62d6af127a3cce0f3d18141088a2aaa9`, por
[decisão humana](https://github.com/rgomids/axiom/pull/329#issuecomment-6254539185)
e [registro Linear](https://linear.app/rgomids/issue/AXM-33/guiar-configuracao-de-project-e-runtime-com-preflight-de-viabilidade#comment-3c387a40).
Consome a [Spec rev.1](spec.md) aceita em
`a99ba8d3afe49f47ab508422ed7ca70e2e553569`; os bytes aceitos de Spec/Plan,
inclusive seus cabeçalhos históricos Proposed, permanecem intactos.

O [gate observado READY_FOR_tasks](https://linear.app/rgomids/issue/AXM-33/guiar-configuracao-de-project-e-runtime-com-preflight-de-viabilidade#comment-6f9f1aea)
autoriza esta proposta documental, não aceite de Tasks ou implementação.
Reutilizar o único [Draft PR #329](https://github.com/rgomids/axiom/pull/329),
branch `docs/axm-33-guided-configuration-specification`. AXM-33 permanece
`In Review`; Execution, revisão, gates e estágio derivados canônicos **unknown**.
Não reconstruir Execution a partir do backlog, PR ou CI. [Evidence](evidence.md)
registra checks desta etapa separadamente dos testes futuros **NOT_RUN**.

## Fronteiras e gates de execução futura

- **G0 — autorização:** review e aceite humano destas Tasks no SHA exato, mais
  autoridade explícita da unidade de implementação. Nenhuma dependência abaixo
  concede essa autoridade. Checks/merge/publicação não substituem aceite.
- **G1 — contratos:** T01 fixa fontes e limitações observadas. Falta de fato
  material interrompe a parte dependente; Clarification identifica cláusula,
  lacuna, dono e alternativas. Contradição arquitetural durável requer proposta
  e aceite humano sob ADR-0018 antes da parte afetada, sem supersessão tácita.
- **G2 — apresentação:** T06 só ativa integração dependente após entrega
  verificável e par binário/skills compatível do AXM-32/#328/ADR-0030. Até lá,
  orientação/fixtures independentes podem ser preparadas; não remover relay
  literal/Auxiliary, produzir renderer ou alterar JSON/wire por AXM-33.
- **G3 — observações nativas:** T07 exige ambientes Codex/Claude e autorização
  delimitada para observações/instalação de teste. Não instala no operador como
  efeito implícito. Ausência mantém paridade NOT_RUN e bloqueia sua conclusão.
- **G4 — C12 real:** T08 exige G-1 realmente exposto na versão instalada, par
  compatível, alvo autorizado, dependências, auth/proof exatos e válidos, produtor
  disponível quando necessário e autorização futura específica de cada efeito.
  Sem isso, C12 real permanece NOT_RUN; fixture não satisfaz o gate.

AXM-34/G-1/#305 é dono da edição da policy e do seu aceite; G-2 permanece com seu
dono de autoria local. AXM-32/#328 é dono da apresentação/migração e seus controles.
AXM-31 define AO-01–13, autonomia e authority; não reabrir seu aceite. AXM-12/#278
é dono do aceite E2E amplo, R-2/R-3 e processo correspondente: referenciar, sem
absorver esse trabalho nem declarar seu aceite com C12. Nenhuma task implementa
G-1/G-2, engine de proof, Doctor, router, registry, agregador de readiness,
novo store, enum, comando ou bypass/manual edit.

Project != Repository; Runtime != Model; Role != Model; Execution != Agent;
Provider != Transport; Integration != MCP. Policy/intenção é portátil;
binding/Profile/effort, auth e proof são locais. Workflow é Project-owned;
Work Item/Execution têm autoridade própria. Supported, installed, authenticated
e proven são fatos independentes. Aplicam-se os ADRs/contratos ligados no
[Plan](plan.md#decisão-técnica-proposta-e-contratos-preservados), incluindo
ADR-0004/0020/0022/0023/0024/0025 e ADR-0030 no revisionamento do AXM-32.

## Precedência e unidades propostas

IDs T01–T09 são estáveis dentro de Specification 009. P1–P6 são passos do Plan,
sem correspondência a novos estados de produto. E01–E09 abaixo são rótulos de
Evidence a anexar ao registro existente, nunca novos stores/formatos canônicos.
Todos os escopos/arquivos de implementação são **prováveis**, sujeitos à inspeção
T01; isso não autoriza mudanças em contratos fora do Plan.

| Task | Slice do Plan | Dependências causais / gates | Estado nesta revisão |
| --- | --- | --- | --- |
| T01 | Contratos e compatibilidade transversais | G0 | Proposed; NOT_RUN |
| T02 | P1/P2: identidade, inventário e parada cedo | T01 | Proposed; NOT_RUN |
| T03 | P3/P4: escolhas e efeitos Project/Runtime | T02 | Proposed; NOT_RUN |
| T04 | P4/P5/P6: Workflow, continuidade e recuperação | T03 | Proposed; NOT_RUN |
| T05 | Contratos e C01–C16 sintéticos/adversariais | T04 | Proposed; NOT_RUN |
| T06 | Integração da apresentação entregue | T05 + G2 | Proposed; NOT_RUN |
| T07 | Compatibilidade, regressão e paridade nativa | T06 + G3 | Proposed; NOT_RUN |
| T08 | C12 positivo real delimitado | T07 + G4 | Proposed; NOT_RUN |
| T09 | Evidence consolidada, reconciliação e review | T07; resultado ou blocker de T08 | Proposed; NOT_RUN |

A cadeia T01→T02→T03→T04→T05→T06→T07 é sequencial: skills e oráculos
compartilham contratos. Após T07, a consolidação documental de T09 pode ocorrer
em paralelo às observações T08, em artefatos distintos; sua revisão final espera
o resultado ou blocker registrado de T08. Não paralelizar writes nas mesmas
skills/Evidence. T09 pode entregar handoff de bloqueio com C12 NOT_RUN, mas isso
não conclui T08 nem habilita aceite comportamental final. Nenhuma outra task tem
paralelismo causal presumido. A ordem de applies na jornada é derivada das
dependências reais do alvo, não desta ordem de implementação.

### T01 — Fixar contratos observados e dependências de integração

- **Objetivo:** preparar um mapa verificável para executar P1–P6 sem inventar
  capacidades ou assumir que main/release/host representam a instalação efetiva.
- **Escopo provável:** inspeção das quatro skills em
  `internal/codexruntime/skills/`, catálogo `internal/cli/skill_inspect.go`,
  readiness, G-1/G-2, Workflow e fontes de suporte/auth/proof citadas no Plan;
  registro incremental em `evidence.md`. Nenhuma mudança de operação/schema.
- **Dependências:** G0. Revalidar #328 e seus contratos entregues, sem copiar
  implementação incompleta. Identificar o par efetivo/skill-set e distinguir
  inspeção de fonte, fixture e observação instalada autorizada.
- **Resultado observável:** lacuna → dono/operação/formas/authority → pré-requisitos
  → resultado/read-back, com fonte, versão e limitações; mapa de cobertura
  dos testes existentes e das duas contradições de orientação G-1/G-2 do Plan.
- **Conclusão:** fatos materiais cobertos ou bloqueio localizado e encaminhado;
  nenhum minimum version/comando/modelo hardcoded. Metadata insuficiente admite
  só diagnóstico suportado read-only. Observação passiva não executa fornecedor;
  auth preflight e effort-source inspection podem executar processos e precisam
  authority própria. Não preparar/probar/invalidar proof para descobrir suporte.
- **Validação futura:** revisão do mapa contra catálogos/ajuda e produtores reais;
  verificar fronteiras de cada leitura e ausência de fato inventado. Não usar
  build/go run do checkout como instalação do usuário.
- **Evidence E01:** refs/SHA dos donos, par observado, operações e campos exigidos,
  lacunas, fonte dos catálogos e gates G2–G4 ainda pendentes; sem dados privados.

### T02 — Orientar inventário e preflight antes de coletar escolhas

- **Objetivo:** compor P1/P2 na entrada `axiom-project`, interrompendo caminhos
  impossíveis antes de perguntar modelo/effort/ID/template.
- **Escopo provável:** `internal/codexruntime/skills/axiom-project/SKILL.md`;
  instruções de handoff para os donos existentes. Corrigir só orientação G-1
  contraditória pertinente, conforme E01 e sem mudar semântica da operação.
- **Dependências:** T01; fatos materiais desconhecidos bloqueiam seu caminho.
- **Resultado observável:** inspect explícito encerra em inspect; objetivo natural
  resolve UUID/source/selector, Repository por chave/remote declarado/binding,
  policy ausente/unconfigured/vazia/configurada, Profiles locais elegíveis e
  effort, Workflow none/ref/digest/Stage/Agent, readiness por operação e
  Integration/Provider necessários. Acesso ausente consta como não observado.
- **Conclusão:** caminho completo conhecido inspecionado antes de coleta técnica;
  CWD/remote/host não escolhem alvo; sem escolher origin ou fabricar path.
  G-1 instalado ausente/unknown, snapshot incompatível, recovery, negativa ou
  proof/produtor exigido indisponível param imediatamente o caminho dependente.
  Não criar Profile como contorno. Alternativa parcial explica benefício/perdas,
  execução blocked e operação comprovada, pedindo opt-in **antes** de sua coleta.
- **Validação futura:** ordem de eventos C02/03/04/06/11/14; zero perguntas
  técnicas após blocker conhecido, zero descoberta mutante/upgrade automático;
  mudança material refaz preflight. Contrato textual não prova decisão da IA.
- **Evidence E02:** inventário referenciado, lacuna/menor próxima ação, sequência
  preflight/pergunta/opt-in ou parada e contadores zero de efeitos/dispatch.

### T03 — Compor escolhas progressivas e preview exato Project/Runtime

- **Objetivo:** realizar P3/P4 somente após caminho viável ou parcial escolhido.
- **Escopo provável:** skills `axiom-project/SKILL.md` e
  `axiom-runtime/SKILL.md` no diretório acima. Consumir G-1 exclusivo para policy,
  G-2 para binding/Profile; não misturar drafts de policy e Repository/Provider.
- **Dependências:** T02; manter controles dos donos e par de E01.
- **Resultado observável:** reaproveitar fatos/decisões válidos; uma decisão
  material pendente por pergunta, com finalidade, opções observadas, consequências
  e recomendação com fonte. Campo aberto não vira catálogo. ID novo é candidato
  confirmado após validação de forma/colisão. Template/tuning/multiagente opcionais;
  runtime-default só se suportado e escolhido. Sem fonte de modelo/effort,
  bloquear uso; salvar intenção só se contrato permitir e parcial tiver opt-in.
- **Conclusão:** installed/authenticated/supported/proven nunca se implicam;
  esforço pertence ao Profile. Cada efeito tem validação/preview do dono,
  antes/depois, identidade, write set, alcance portátil/local compartilhado,
  dependências, snapshots/digests, impacto Workflow/Stage/Agent e limites de
  readiness. Authority vincula operação/alvo/conteúdo/revisão/alcance exatos;
  escolha geral ou outro apply não a transfere. Negativa cancela sem insistência
  ou substituição; confirmação ainda válida não é solicitada novamente.
- **Validação futura:** C01/05/06/07/12/13/16 com fontes das opções, uma decisão
  por pergunta e contadores por efeito; zero default/proof/consentimento inventado.
- **Evidence E03:** alternativas/fonte, decisões/opt-in, preview e authority por
  apply, inclusive negativas, escopo e efeitos anteriores preservados.

### T04 — Encadear Workflow, stale, read-back e handoff bounded

- **Objetivo:** completar P4/P5/P6 entre os quatro donos sem iniciar Execution.
- **Escopo provável:** quatro skills; especialmente `axiom-workflow/SKILL.md`
  e `axiom-work-item/SKILL.md`. Reconciliar só frases G-1/G-2/G-5/readiness e
  dispatch/delivery afetadas, sem limpeza geral nem mudança G-1/G-2/G-5.
- **Dependências:** T03, que estabelece escolhas e autoridade por efeito.
- **Resultado observável:** Workflow definition list/show/validate e readiness
  alimentam Project workflow.select, com preview/authority próprios. Built-in,
  refs inválidas ou necessidade de autoria/migração fora do objetivo vão ao dono.
  Se policy conflitar com Workflow ativo, ordenar reparação/seleção compatível
  antes da policy ou parar; não oferecer sequência fixa impossível.
- **Conclusão:** reobservar snapshots do dono e fatos materiais Project/source/
  selector, policy, local, executável/auth/proof e Workflow antes de cada apply;
  drift invalida preview/authority e exige confirmação nova. Sem lock/transação
  global ou rebase silencioso. Após cada write, comparar identidade/revisão/
  conteúdo no read-back e reavaliar readiness pelo evaluator existente. Ack
  perdido/leitura divergente para cadeia, separa efeito incerto e impede replay.
  Aplicar AO-06/07/09: ≤2 repetições após inicial em 60 s e ≤2 remediações por
  bloqueio, respeitando teto menor e parando antes sem progresso, na negativa,
  risco novo ou efeito incerto; limites não concedem mutação. Retomada preserva
  correlação/attempt, deadline original e contadores; checkpoint incompleto não
  autoriza repetição. Probes/login/inferência/kill/redispatch/child retry excluídos.
- **Validação futura:** C02/07/08/09/10/13/14/15. Ready/partial/blocked são sínteses
  delimitadas, separadas de result/readiness/certeza/aceite; Work Item handoff
  conserva refs e unknown, sem create/link/start/resume/run/stage dispatch/rebind.
  Preservar policy portátil sem candidato local, paths/links/permissões e dados
  não confiáveis. Sem credential resolution/manual edit/canário real.
- **Evidence E04:** snapshots e recusas stale por dimensão, read-back/readiness
  antes/depois, efeitos confirmados/incertos, relógio/contadores cumulativos,
  autoridade usada/pendente, checkpoint sanitizado e próxima ação pelo dono.

### T05 — Verificar contratos e cenários sintéticos/adversariais

- **Objetivo:** verificar decisões/ordem/efeitos de P1–P6 com oráculos mensuráveis,
  distinguindo instrução textual de comportamento observado.
- **Escopo provável:** ampliar testes existentes
  `internal/codexruntime/workflow_skill_content_test.go`,
  `internal/cli/domain_skill_routing_test.go`,
  `internal/cli/skill_catalog_contract_test.go` e fixtures sanitizadas junto às
  suites existentes; Evidence. Não criar engine/runner de jornada de produto.
- **Dependências:** T04. Fixtures refletem formas/limites reais de E01;
  não antecipar wire/renderer AXM-32. Testes unitários novos só para risco/contrato
  alterado não coberto, reutilizando doubles de stores/observadores existentes.
- **Resultado observável:** C01–C16 sintéticos parametrizados conforme matriz
  abaixo, entradas natural/explícita aplicáveis, relógio controlado e contadores
  de writes/dispatch. Verificar ownership/inspect-stop/preflight/opt-in/read-back
  nas quatro skills, e executar observações conversacionais sintéticas separadas
  das assertions de strings. Native parity permanece obrigação T07.
- **Conclusão:** cada cenário tem esperado/observado e fonte; zero write sem
  authority, default/capacidade inventados, blocker omitido, dispatch ou excesso
  de budget. Denial parametrizado por tipo de apply; stale por cada dimensão;
  C15 com injeção draft/erro/Provider e canário sintético ausente em saída/Evidence.
  Ausência de execução é NOT_RUN, nunca aprovação pelo conteúdo da skill.
- **Validação futura:** suites focadas acima e matriz completa; justificar filtros
  concretos após inspecionar testes. Não alterar testes dos donos para acomodar
  desvio; falha de contrato retorna ao dono/Clarification.
- **Evidence E05:** por caso, fixture/forma/fonte, entrada, eventos e decisões,
  versões/SHA, resultados/oráculos/contadores e recortes mínimos sanitizados;
  C12 synthetic separado de C12 real, independentemente do resultado.

### T06 — Integrar consumo da apresentação sob ownership AXM-32

- **Objetivo:** ativar a jornada sobre o contrato de apresentação entregue e
  compatível, preservando fatos e autoridade definidos nas tasks anteriores.
- **Escopo provável:** pontos de consumo/handoff das quatro skills e testes de
  contrato existentes; `evidence.md`. Nenhum renderer, modelo de formatação,
  envelope JSON ou migração de relay é implementado por esta task.
- **Dependências:** T05 + G2. Revalidar entrega de #328/ADR-0030; se incompleta,
  registrar blocker só desta ativação, sem substituir contrato parcial.
- **Resultado observável:** jornada consome resultados canônicos preservando
  status, readiness, efeito, identidade, refs/digests, consentimento e aceite.
- **Conclusão:** par binário/skills demonstrado compatível e consumo reconciliado
  com dono; sucesso estrutural não oculta execução blocked. Sem deduzir JSON por
  pipe/TTY ou trocar relay histórico unilateralmente antes da entrega AXM-32.
- **Validação futura:** contratos de integração focados e repetição dos casos
  afetados C01/04/07/09/12/13; verificar nenhuma omissão material ou autoridade nova.
- **Evidence E06:** SHA/contrato entregue pelo dono, par compatível, casos
  revalidados e limitações; se gate falhar, blocker/next, sem PASS de integração.

### T07 — Validar compatibilidade, regressão proporcional e paridade Codex/Claude

- **Objetivo:** demonstrar a mesma orientação e restrições nos dois hosts, inclusive
  Runtime alvo diferente da operadora, sem amarrar a jornada a um fornecedor.
- **Escopo provável:** suites existentes de skills/instalação/reinstalação/upgrade,
  observações nativas sanitizadas e Evidence. Policy edit, runtime authoring,
  Workflow binding e compatibility são regressões dos donos, sem reimplementação.
- **Dependências:** T06 + G3; fixtures/conjunto de decisões de T05.
- **Resultado observável:** executar C01–C16 aplicáveis em Codex e Claude pelas
  duas entradas, incluindo alvos cruzados; comparar operação, fonte/opções, ordem,
  consentimento, blockers e efeitos, sem exigir prosa idêntica. Documentar qualquer
  inaplicabilidade objetiva; não reduzir cobertura silenciosamente.
- **Conclusão:** paridade semântica demonstrada e regressão focada verde; validar
  orientação instalada, reinstall/upgrade pelo mecanismo existente, recusa de par
  incompatível e rollback para pacote conhecido compatível. Preservar receipts,
  snapshots/corpus históricos; rollback da orientação não desfaz configuração.
  Inversão de configuração só por operação suportada + nova authority/read-back;
  recuperação/ack incerto fica com dono. Não ampliar suites por rotina.
- **Validação futura:** checks documentais/contratuais pertinentes e regressões
  dos componentes afetados; testes nativos são Evidence distinta dos unitários.
  Sem ambiente/authority, relatar NOT_RUN e pendência, sem declarar paridade.
- **Evidence E07:** operador/alvo/host, versões/par/SHA, pacote/compatibilidade,
  casos por host, matriz semântica, comandos reais/saídas sanitizadas, regressões
  executadas e omissões. C12 real e aceite E2E continuam separados.

### T08 — Observar C12 real somente com pré-requisitos e autoridade específicos

- **Objetivo:** comprovar o positivo G-1 delimitado sem absorver AXM-34 ou AXM-12.
- **Escopo provável:** Evidence de observação autorizada; nenhuma implementação
  de policy, auth/proof, instalador ou processo E2E. Referenciar Evidence canônica.
- **Dependências:** T07 + G4. Operação G-1 instalada e formas/authority reais,
  Repository/Profile/Workflow compatíveis, auth/proof frescos ligados exatamente a
  Profile/model/effort/executável/versão/variante/digests. Não gerar proof/probe
  como descoberta; necessidade segue produtor/dono e autorização independentes.
- **Resultado observável:** preflight → decisão mínima → preview exato → authority
  específica → apply → read-back/readiness sem blocker material no objetivo.
- **Conclusão:** efeito autorizado confirmado e resultado delimitado; zero dispatch
  de Execution. Não confundir policy salva com proven, ready com run/aceite, nem
  C12 synthetic com real. Pré-requisito ausente mantém task incompleta e C12 real
  NOT_RUN com motivo/menor próxima ação; consentimento não cria produtor de proof.
- **Validação futura:** observar sequência real, snapshots/freshness e contadores;
  novo drift/negativa retorna aos controles do dono. Não executar AXM-12/#278 aqui.
- **Evidence E08:** instalação/par e fontes, alvo lógico autorizado, authority por
  efeito, refs de auth/proof sanitizadas, esperado/observado, read-back/readiness,
  limites e próximo gate; sem paths privados, secrets, stdout de auth ou chat bruto.

### T09 — Consolidar Evidence, documentação e review proporcional

- **Objetivo:** permitir decisão humana sobre a entrega exata e retomada segura,
  sem substituir aceite por checks ou encobrir gates externos.
- **Escopo provável:** `evidence.md`, índice e referência de operações apenas se
  afetada; Spec/Plan/ADRs aceitos e histórico permanecem preservados.
- **Dependências:** T07; incorporar resultado ou blocker de T08 antes do handoff
  final. Paralelismo documental limitado descrito acima não autoriza publicação.
- **Resultado observável:** E01–E08 consolidados por GC/caso/camada, comandos e
  resultados reais, lacunas/risco/donos/next, diff mínimo e revisão do SHA exato.
- **Conclusão:** checks documentais/links/governança/security executados e review
  engenharia/segurança proporcional com findings resolvidos ou decisão explícita;
  revisão independente e aceite humano continuam identificados como pendentes
  quando ausentes. C12 NOT_RUN impede alegar validação real completa. Nenhum
  fechamento/status/label/gate, merge/release ou aceite E2E é derivado daqui.
- **Validação futura:** cobertura/dependências, Markdown/links, diff/bytes aceitos,
  hygiene e scanner disponível; não publicar dados não classificados. Checkpoint
  mantém objetivo, concluído/pendente, refs/versões, autoridade, efeitos incertos,
  budgets e próxima unidade sem depender de IDs internos da Runtime.
- **Evidence E09:** revisão e findings, checks e limites, commit/head remoto/PR
  único, CI observado no SHA exato, pedido de decisão humana e handoff bounded.

## Matriz completa GC → Plan → Tasks → validação/Evidence → cenários

| GC | Plan / slice | Tasks | Validação e Evidence futuras | C01–C16 pertinentes |
| --- | --- | --- | --- | --- |
| GC-01 | P1; reobservação P4/P5 | T01/T02/T04/T05/T07 | E01/E02/E04/E05/E07: identidade, inventário, unknown e snapshots por dono | C01/02/03/08/11/14/16 |
| GC-02 | P2 antes de P3; drift retorna P2 | T01/T02/T03/T04/T05/T07/T08 | E01–E05/E07/E08: blocker antes de coleta, caminho conhecido/opt-in, suporte e proof independentes | C03/04/05/06/10/11/12/13/14 |
| GC-03 | P3 condicionado a P2 | T03/T05/T07 | E03/E05/E07: fonte das opções, uma decisão/pergunta, reuso sem default | C01/04/05/06/16 |
| GC-04 | P4 antes de cada write | T03/T04/T05/T06/T07/T08 | E03–E08: tuple exata, negativa/stale zero write, impacto e consentimento separados | C02/03/07/08/09/11/12/13/14/15/16 |
| GC-05 | P5 após cada write; P6 confirmado | T04/T05/T06/T07/T08 | E04–E08: read-back/readiness, certeza e síntese com escopo, zero replay/dispatch | C01/03/04/06/07/09/12/13 |
| GC-06 | Transversal; P1/P2 na retomada | T02/T04/T05/T07/T09 | E02/E04/E05/E07/E09: AO-06/07/09, relógio/budgets, injeção/confinamento/handoff | C02/09/10/15 |

T06 preserva todos os GC ao integrar apresentação; T09 reconcilia a matriz inteira.
Os cruzamentos incluem a cobertura transversal do Plan, sem mudar os requisitos
primários de cada caso da Spec. Todos os cenários têm execução futura em T05/T07;
C12 real pertence exclusivamente à observação gated T08.

| Caso | Task produtora / verificação | Oráculo mínimo e Evidence | Estado agora |
| --- | --- | --- | --- |
| C01 | T02/T03/T04; T05/T07 | E02–E05/E07: estado compatível, zero pergunta conhecida/write/dispatch; ready delimitado | NOT_RUN |
| C02 | T02/T04; T05/T07 | E02/E04/E05/E07: UUID ambíguo/remote divergente; uma decisão, zero origin/Git edit | NOT_RUN |
| C03 | T02/T03/T04; T05/T07 | E02–E05/E07: binding ausente, operação presente/ausente; preflight antes de tuning; write exato ou blocked | NOT_RUN |
| C04 | T02/T03/T04; T05/T07 | E02–E05/E07: sem G-1 instalado/policy, zero coleta modelo/effort/ID/template e Profile/apply; main não supre | NOT_RUN |
| C05 | T02/T03; T05/T07 | E02/E03/E05/E07: capability ambígua, opções/fontes verificadas e uma decisão; zero token/proof inventado | NOT_RUN |
| C06 | T02/T03/T04; T05/T07 | E02–E05/E07: installed/authenticated separados de supported/proven; intenção só com contrato e parcial opt-in | NOT_RUN |
| C07 | T03/T04; T05/T07 | E03–E05/E07: negar separadamente binding/Profile/policy/selection; zero efeito negado/substituição | NOT_RUN |
| C08 | T04; T05/T07 | E04/E05/E07: drift UUID/source/selector, policy, local e Workflow; cada dimensão zero write, preview/authority novos | NOT_RUN |
| C09 | T04/T06; T05/T07 | E04–E07: ack perdido/read-back ausente/writer pós-commit; zero replay/ready indevido, certeza separada | NOT_RUN |
| C10 | T02/T04; T05/T07 | E02/E04/E05/E07: transitório/persistente/duas sessões; ≤2 repetições/60 s, ≤2 remediações e parada anterior sem progresso | NOT_RUN |
| C11 | T01/T02/T04; T05/T07 | E01/E02/E04/E05/E07: inspect encerra; ausente/metadata contraditória bloqueiam, zero substituto mutante | NOT_RUN |
| C12 | T02/T03/T04/T06; T05/T07; T08 real | E02–E08: ordem completa, G-1/dependências/auth/proof válidos; synthetic não substitui real | NOT_RUN synthetic; NOT_RUN real |
| C13 | T02/T03/T04/T06; T05/T07 | E02–E07: G-1 presente/proof-auth ausente ou mismatch; parcial escolhido/confirmado, execução blocked, policy preservada | NOT_RUN |
| C14 | T02/T04; T05/T07 | E02/E04/E05/E07: Workflow none/incompatível/built-in/ref inválida; selection separada ou handoff, zero edição built-in/Execution | NOT_RUN |
| C15 | T04; T05/T07 | E04/E05/E07: injeção draft/erro/Provider e canário sintético; zero comando/authority promovida, canário ausente saída/Evidence | NOT_RUN |
| C16 | T02/T03; T05/T07 | E02/E03/E05/E07: colisão ID validada, rótulo/candidato confirmado, template opcional | NOT_RUN |

Unitário/contrato, conversacional synthetic, paridade nativa Codex/Claude, C12 real
e E2E amplo AXM-12 são camadas distintas, todas **NOT_RUN** aqui. Cada resultado
futuro registra operador/alvo, versão binário/skills/SHA, fonte de catálogo,
refs/revisões/digests/freshness, esperado/observado, decisões/negativas/authority,
eventos/perguntas, writes/dispatch, read-back/readiness e budgets. Evidência
canônica é referenciada; recortes mínimos sanitizados não exportam caminhos
privados, credentials ou conversa bruta. Zero efeitos inesperados é obrigação
transversal, não nova autoridade para testar ambientes reais.

## Riscos, decisões pendentes e saída de tasks

Não há mudança arquitetural/contratual proposta. A decomposição preserva a
Constituição: intenção/SDD e fontes aceitas (I/II/III), Evidence honesta e checks
determinísticos (IV/V), least privilege/authority (VI), unidade mínima (VII) e
Project separado de Repository (VIII). Qualquer colisão material exige
Clarification/ADR conforme G1, sem reescrever os artifacts históricos.

| Risco ou pendência | Gate / menor próxima ação |
| --- | --- |
| Instalação G-1/auth/proof/produtor reais não observados | T01/T02 limitam viabilidade; G4 bloqueia T08, não redação/review de Tasks |
| AXM-32 incompleto ou par binário/skills incompatível | G2/T06: aguardar entrega do dono; não assumir merged, instalado ou apresentação migrada |
| Readiness sem fato material ou metadata inconclusiva | Parar parte dependente, registrar unknown e Clarification ao dono; sem Doctor/store paralelo |
| Drift/ack incerto/efeitos parciais e retomada | T04/T05: read-back, sem replay, budgets persistidos na Evidence autorizada, recovery do dono |
| Fixtures/strings/CI parecerem prova nativa/real | T05/T07/T08: resultados por camada, NOT_RUN explícito; C12 e AXM-12 independentes |
| Linear confundido com adapter ou Execution | Revalidar suporte e binding autorizados; sem espelho GitHub, gate inferido ou edição de estado |

Review desta revisão deve conferir cobertura integral GC/C, ordem/dependências,
pequeno escopo e verificabilidade de cada task, ownership externo, compatibilidade,
rollback e segurança/authority, além de links/Markdown/governança/bytes aceitos.
Recomendação: manter a cadeia mínima e ativação sob G2–G4. Alternativa de integrar
antes da entrega dos donos exigiria duplicação e risco de contrato; não adotada.
Criar estado/autoridade central seria difícil de reverter; texto e organização
das unidades são reversíveis. Não há nova decisão arquitetural a aceitar aqui.

Saída atual: somente Tasks **Proposed**, Evidence incremental e índice; checks
realmente executados em Evidence, commit/push/head e CI do SHA exato no PR/Linear.
Decisão pendente: review/Clarification e **aceite humano da revisão 1 de Tasks no
SHA exato publicado**, seguido de autoridade própria para unidade futura.
Nenhuma task, cenário ou aceite de produto é marcado concluído nesta etapa;
sem código/testes de produto/Runtime, implementação, merge, release ou transição.
