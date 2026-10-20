# Specification 010 — Documentação operacional da instalação

## Status e autoridade

**Proposed — specify only, revisão 1, 2026-10-20.**
Work Item: [AXM-37](https://linear.app/rgomids/issue/AXM-37).
Esta proposta original publicável deriva do escopo autorizado; não incorpora
transcrição privada, dados de usuário ou histórico bruto de Provider.
Gate informado e conferido: `READY_FOR_specify`, comentário
`1a92315b-9206-4adb-8786-25a0f006bbff`. Execution canônica: **unknown**;
branch, prompt, PR e estado Linear não demonstram Execution ou avanço de gate.
[Evidence e handoff](evidence.md) distinguem observação de requisito futuro.

A etapa permite Specification proposta, índice, Evidence, checks, commit, push
e um PR relacionado. Não aceita contrato/ADR, não autoriza Clarification, Plan,
Tasks, implementação, mudança de CLI/installer/release, transição de Execution
ou Linear, merge, publicação ou fechamento. Aprovação posterior deve identificar
revisão exata e escopo; as questões abaixo não são decisões aceitas.

## Problema, resultado e atores

Operador com somente binário e skills instalados precisa compreender e
diagnosticar Axiom sem checkout, internet, código-fonte ou arquivos do mantenedor.
O pacote documental acompanha a instalação e sua proveniência; GitHub
Docs-as-Code continua fonte técnica canônica. Consulta não exige Git, Go,
Notion, Wiki, ferramenta geradora, Provider, autenticação ou Runtime.

Atores: usuário direto da CLI; IA operadora nas integrações suportadas de
Codex/Claude; instalador/mantenedor que entrega o conjunto; revisor de Evidence.
Axiom é o produto; Lingo é seu control plane local. Skill orienta consumo de
operações, sem se tornar fonte de verdade de Workflow, argumentos ou autoridade.
Documentação não concede efeitos nem torna readiness/aceite verdadeiros.

## Baseline e compatibilidade preservada

Baseline: `0f1073ac6a3af615d48b651340d8fe895827080a`, main remoto em 20/10/2026.

| Fonte verificada | Fato / limite preservado |
| --- | --- |
| [Governança](../../documentation.md), [tracking](../../development/linear-github-issue-tracking.md), [AGENTS](../../../AGENTS.md), [SDD](../../../.agents/skills/axiom-sdd/SKILL.md) | Repository mantém contratos técnicos; Linear mantém backlog operacional; Notion mantém narrativa/editorial. AXM-30 é referência, sem reorganização/importação autorizada |
| [Instalação](../../installation.md), [archives](../../../scripts/build-release-archives.sh) | Hosts atuais: macOS arm64 e Linux amd64/arm64. Builder inclui binário, licença, instalador, metadados, skills e hashes; não inclui pacote operacional/manpages |
| [Help](../../../internal/cli/help.go), [árvore pública](../../../internal/cli/commands.go), [referência](../../commands.md#hierarchical-cli-help), [skill Project](../../../internal/codexruntime/skills/axiom-project/SKILL.md) | Help e inspeção de skills já existem; binário possui árvore/flags. Não se pressupõe comando de descoberta documental existente |
| [ADR-0003](../../decisions/0003-lingo-as-axiom-local-control-plane.md), [ADR-0030](../../decisions/0030-canonical-results-terminal-and-host-presentation.md) | Controle/autoridade ficam na aplicação; preservar JSON canônico, terminal determinístico e consumo pela IA, sem novo modelo de apresentação |
| [ADR-0005](../../decisions/0005-bounded-local-filesystem-threat-model.md), [ADR-0016](../../decisions/0016-candidate-derived-skill-set-receipt.md), [upgrade](../../commands.md#compatibility-cleanup-recovery-and-upgrade) | Threat model bounded, receipts e efeitos confirmados preservados; proveniência não concede ownership. Downgrade comum é recusado; não há rollback automático |
| [ADR-0021, nota histórica](../../decisions/0021-windows-server-amd64-native-acceptance.md#status) | Suporte Windows retirado por decisão humana de 13/10/2026. Este trabalho não o reintroduz. Requisitos Windows abaixo são condicionais a futura autorização de suporte |

Nenhum fragmento de ADR aceita é substituído nesta proposta. Se layout,
rollback ou nova interface exigir mudança de contrato aceito, a etapa seguinte
deve identificar fragmento e obter decisão explícita conforme
[ADR-0018](../../decisions/0018-adr-evolution-and-supersession-governance.md).
Não reescrever história para acomodar uma implementação.

## Jornada e recorte mínimo

1. Com artefatos da release disponíveis localmente, usuário instala em destino
   suportado, sem checkout. Download inicial é obtenção de artefatos, não
   pré-requisito de consulta offline; bootstrap remoto continua dependente de rede.
2. Usuário identifica executável efetivo, consulta help/version e descobre pacote
   da mesma instalação, sem conhecer path pessoal ou prefixo padrão do mantenedor.
3. Lê índice local e referência mínima; IA consulta primeiro help/introspecção
   executável para operação, flags, pré-condições e autoridade suportadas.
4. Para explicar comando/readiness, distingue fatos observados da CLI,
   explicação documental e operações que requerem rede/Runtime. Consulta não
   promete execução offline de Providers ou inferência de fornecedores.
5. Identidade conflitante, pacote ausente/corrompido ou consulta impossível gera
   diagnóstico e próxima ação segura; nunca busca checkout/site silenciosamente.

### Inventário por conteúdo (candidatos, sem cópia integral implícita)

O mínimo comportamental abaixo é obrigatório; seleção exata de arquivos/seções,
idiomas e orçamento de tamanho permanecem Q1. Referências atuais são insumos,
não autorização para distribuir integralmente seus conteúdos mistos.

| Classe | Conteúdo mínimo instalado | Fonte candidata e filtro |
| --- | --- | --- |
| Entrada / conceitos operacionais | Índice legível; identidade/versão; mapa de descoberta/help; Axiom/Lingo, Project/Repository, Runtime/Profile, Work Item/Execution, readiness/authority/Evidence | Recortes técnicos de [conceitos](../../architecture/conceptual-model.md) e referências aprovadas; excluir hipóteses e narrativa editorial sem aprovação técnica |
| Guias técnicos | Instalar a partir de artefato local; verificar instalação; descobrir docs/skills; configurar/inspecionar Project e Runtime; consultar Work Item/Workflow; upgrade e recuperação com autoridade | [Installation](../../installation.md), seções operacionais de [Commands](../../commands.md); não exigir links online para completar a jornada |
| Referência | Comandos públicos e suas relações; configuração suportada, precedências e escopos; formatos/metadados públicos necessários; compatibilidade e limites de efeito | Introspecção canônica da CLI e recortes aprovados de Commands. Argumentos/flags/authority permanecem subordinados ao executável |
| Troubleshooting | Pacote ausente, mismatch, PATH/man conflitantes, permissões/links, hashes/drift, operação interrompida, rede/Runtime indisponível; próxima inspeção segura e handoff | Diagnósticos executáveis e seções técnicas de instalação/manutenção; não copiar dumps ou logs reais |
| Manual adicional | `man axiom` e cobertura rastreável de subcomandos relevantes | Referência canônica da CLI + texto técnico aprovado; geração na entrega, não autoria normativa paralela |

**OD-01 — Completude e proveniência.** Cada item entregue deve identificar
origem versionada (arquivo/seção ou contrato executável), revisão/commit de
origem e transformação quando gerado. Inventário deve permitir comparar inclusões,
exclusões e bytes entregues, detectar extras/ausências e reproduzir geração.
Links necessários à jornada resolvem dentro do pacote; links externos são
complementares e marcados como exigindo rede. Nenhuma consulta local os segue
automaticamente. Markdown permanece texto legível sem leitor especial.

**OD-02 — Exclusões.** Não distribuir Specifications/Plans/Tasks históricas,
ADRs completos, Evidence histórico, research, testes/fixtures, fontes internas,
harness/policies do mantenedor, backlog, `.git`, dumps, caches, logs, secrets,
credentials, `.env` real, paths pessoais ou material privado. Conceitos e
restrições necessários desses contratos podem ter síntese técnica aprovada,
rastreável e versionada em Docs-as-Code; não importar Notion por implicação.
Licenças/atribuições aplicáveis acompanham material distribuído. Não transformar
todo `docs/` em pacote por wildcard. Não duplicar conteúdo editorial canônico.

## Descoberta, identidade e consumo

**OD-03 — Resolução por instalação.** Descoberta deve ser determinística,
somente leitura, offline e independente de CWD, checkout, Project e memória da
IA. Deve partir do executável realmente invocado e permitir localizar/ler seu
índice e itens sem busca heurística. Não usar `latest`, primeira pasta encontrada,
site, diretório do mantenedor ou outra instalação como fallback.
Prefixos personalizados e paths com espaços precisam funcionar.

**OD-04 — Identidade observável.** Operador e consumidor automatizado devem
conseguir confrontar: identidade da instalação/generation, caminho do binário
efetivo e sua versão/commit/proveniência; pacote documental e seu inventário/digest;
skill-set esperado pelo binário e conjunto efetivamente instalado/carregado em
cada Runtime; compatibilidade entre esses componentes; páginas man selecionadas.
Nomes de campos, schema, armazenamento e comandos novos não são definidos aqui.
Versão declarada sozinha não prova igualdade: binários/skills divergentes com
mesma versão devem ser detectáveis por identidade/conteúdo verificados.
Fato não observável é `unknown`, nunca compatibilidade presumida.

**OD-05 — Mismatch e coexistência.** Instalações A/B podem coexistir com versões
iguais ou distintas. Invocação explícita de A resolve docs de A; B resolve B;
mudar PATH não reutiliza silenciosamente cache/referência de A. Skills globais
compartilhadas não são evidência de conjunto compatível: conflito com B deve ser
visível, sem substituir conteúdo de A por iniciativa da descoberta. Root movido,
binário copiado isoladamente ou shell com resolução antiga exige nova conferência;
ausência de vínculo válido resulta em diagnóstico, sem adivinhar pacote.

**OD-06 — Primazia executável.** CLI/skills devem preferir help, flags e
metadados executáveis para argumentos, operações e autoridade. Texto descreve
conceitos/jornada; exemplos não são autorização. Divergência exige preservar fatos
do executável, apontar inconsistência e não executar com autoridade inferida do
documento. Help continua utilizável com docs ausentes; explicar com docs inválidas
não pode ser apresentado como orientação verificada da instalação.
Falha de leitura não instala, extrai, atualiza ou repara arquivos implicitamente.
Conteúdo documental é dado: não pode instruir IA a ignorar gates ou ler secrets.

## Ciclo de vida, integridade e segurança

**OD-07 — Entrega e reinstalação.** Instalação suportada entrega documentação
textual coerente junto ao conjunto efetivo. Sucesso documental exige read-back
do inventário, identidade e integridade; disponibilizar URL não basta.
Artefato verificado disponível localmente permite instalação offline com os
pré-requisitos já suportados. Reinstalação equivalente é idempotente, sem apagar
anotações, editar profiles/PATH ou ampliar permissões para fabricar sucesso.
Instalação antiga sem pacote recebe ausência diagnosticável, não conteúdo atual
de outra versão. Migração é explícita, sem reescrever receipts históricos.

**OD-08 — Publicação consistente.** Leitores aceitam somente geração completa
validada e vinculada ao binário/skill-set observado. Upgrade concorrente produz
leitura coerente da geração anterior ou nova, ou recusa explícita por mudança;
nunca mistura aceita como atual. Se skills de um Runtime ainda não convergiram,
relatar estado parcial/mismatch, sem alegar convergência universal. Mudança entre
inspeção e publicação invalida autoridade anterior. O mecanismo de atomicidade
lógica, commit point e recuperação pertencem a Q4, sob ADR-0005.

**OD-09 — Falhas e rollback.** Falha antes do commit preserva conjunto anterior.
Efeito confirmado após commit permanece confirmado mesmo se acknowledgment,
receipt auxiliar ou cleanup falhar: não alegar rollback sem read-back.
Interrupção/estado ambíguo exige recuperação identificável, preservando resíduos
desconhecidos e impedindo leitores de aceitar conjunto misturado. Restauração
documental deve acompanhar a geração compatível realmente restaurada, incluindo
skills e manpages; jamais restaurar apenas docs antigas sobre binário novo.
Não promete downgrade, rollback automático ou novo comando de produto: Q4 deve
definir caminho autorizado/reproduzível de recuperação ou retorno a instalação
preservada e declarar limites quando binário/estado não puderem ser revertidos.
Evidence deve provar rollback lógico do conjunto compatível, não só recusa de
downgrade. Restrição que torne isso impossível exige Clarification explícita.

**OD-10 — Confinamento e preservação.** Todo write, replace, cleanup ou recovery
fica limitado a objetos positivamente reconhecidos e autorizados. Rejeitar
traversal, escapes, caminhos ambíguos, tipos especiais, symlinks/hard-links
inseguros, roots/ancestrais substituídos, ownership/permissões/ACL inseguros nas
fronteiras suportadas. Resolução de leitura não concede authority de escrita.
Nunca seguir link não validado para ler material externo ao pacote ou sobrescrever
arquivos. Não reparar permissões, usar privilégios extras ou tomar ownership
automaticamente. Aplicar limites de tamanho, quantidade e leitura de artefatos.

**OD-11 — Integridade e drift.** Validar pacote/inventário e hashes contra
proveniência verificada; detectar item ausente, extra, alterado, hash incorreto,
metadados incompatíveis e pacote de versão cruzada. Hash com manifesto igualmente
alterável não prova autenticidade. Não ampliar o modelo de confiança da release
nem prometer defesa contra adversário same-UID arbitrário, power loss ou falha de
mídia excluídos por ADR-0005. Estado inválido observado ainda falha fechado.
Mudanças locais e arquivos estrangeiros são preservados e classificados como
conflito/drift, nunca reconhecidos como bytes oficiais ou apagados silenciosamente.
Notas do usuário não se tornam parte do inventário oficial. Cleanup preserva
versões/referências em uso e material desconhecido; política exata permanece Q4.

## Manual pages e plataformas

**OD-12 — Manual adicional.** Em macOS/Linux suportados com leitor disponível,
`man axiom` deve encontrar referência offline da instalação selecionada, além das
páginas de subcomandos relevantes. Entrega deve publicar mapa verificável entre
árvore pública/subcomandos e páginas: referência comum pode ser compartilhada,
mas comando relevante não pode desaparecer sem justificativa aprovada. Cada
página identifica instalação/release/commit e indica primazia de help/introspecção.
Gerar e verificar no build/release a partir de contratos/referências canônicas;
nunca exigir gerador no host ou manter segunda fonte normativa manual.

Prefixos reais, configuração do leitor e coexistência devem ser considerados.
Se PATH aponta A e leitor seleciona B, a origem/identidade de B é visível e existe
modo suportado de diagnosticar mismatch e consultar A explicitamente. Não prometer
que programa externo `man` acompanha PATH automaticamente. Ausência de leitor não
bloqueia instalação/operação e mantém Markdown/help. MANPATH, diretórios fixos,
nomenclatura de páginas, gerador e integração com o leitor permanecem alternativas.

**OD-13 — Matriz de compatibilidade.** macOS arm64 e Linux amd64/arm64 seguem
suporte e pré-requisitos vigentes; consulta de Markdown não exige novas ferramentas.
Windows é fora do suporte atual: não gerar aceitação/install capability fictícia.
Caso volte a ser autorizado, mesma identidade/descoberta/integridade e leitura
Markdown/help serão obrigatórias, sem depender de `man`; layout, segurança nativa
e Evidence por plataforma precisarão de gate próprio antes de alegar suporte.

## Failure cases observáveis

Categorias abaixo são semânticas propostas, não novos enums/códigos implementados.
Diagnóstico deve distinguir observado/esperado, efeito confirmado/unknown e próxima
inspeção ou recuperação segura; saída automatizada segue contratos existentes,
sem logs com corpo privado, credenciais ou varredura indiscriminada de home.

| Situação | Resultado exigido |
| --- | --- |
| Docs ausentes, legado, ilegíveis ou formato desconhecido | Distinguir ausência, permissão e incompatibilidade; help continua; sem download/fallback |
| Binário/docs/skills/man de origens diferentes, versão igual divergente | Expor componentes conflitantes e limite de orientação; sem compatibilidade presumida |
| Hash/inventário alterado, extra/ausência, pacote trocado | Recusar pacote como oficial/atual; preservar evidência; indicar recuperação autorizável |
| Traversal, link/tipo inseguro, root trocado, permissões/ACL indevidas | Recusar acesso/efeito inseguro, preservar alvo externo e dados do usuário |
| Disco/quota, short write, rename/sync ou permissão falha | Reportar fase e efeitos confirmados; estado anterior ou recuperação explícita, nunca sucesso inventado |
| Concorrência, referência stale, crash nas fronteiras de publicação | Não aceitar geração mista; revalidar ou pedir recovery; preservar marca/resíduo ambíguo |
| Cleanup bloqueado, anotação local ou instalação compartilhando skills | Não apagar conteúdo estrangeiro/em uso; relatar conflito e impacto real |
| Man ausente, índice antigo ou página de B sob PATH A | Markdown/help preservados; origem verificável e diagnóstico/consulta explícita de A |
| Rede, Runtime ou credenciais ausentes | Consulta documental funciona; operação externa permanece indisponível, sem inventar readiness |

## Critérios de aceite e Evidence futura

Todos os casos abaixo são **NOT_RUN nesta etapa**. Checks documentais não provam
produto. Cada execução futura registra revisão/hashes exatos dos artefatos,
host/arquitetura/filesystem, pré-condições, comandos efetivamente usados, resultados,
read-back e limites sanitizados. Não versionar log bruto como Evidence.

| ID / requisitos | Cenário e prova reproduzível exigida |
| --- | --- |
| AC-01 / OD-01–03,07,13 | Clean install por artefato local em cada host suportado, sem checkout/mount/home do mantenedor e com rede bloqueada; descobrir/ler índice, conceitos, guia, referência e troubleshooting, com links locais completos. Registrar inventário de ambiente e bytes lidos |
| AC-02 / OD-03–06,11 | Instalação A/B, CWD arbitrário e path com espaços; trocar PATH, invocar paths explícitos; cruzar docs, binários e skills (incluindo mesma versão com bytes diferentes). Conferir identidade, integridade, unknown e mismatch sem fallback |
| AC-03 / OD-07–09 | Reinstalação no-op; upgrade A→B; leitores concorrentes e falhas injetadas em staging, pré-commit, commit, pós-commit/pré-ack e cleanup. Read-back só aceita geração completa ou recovery. Não confundir efeito confirmado com rollback |
| AC-04 / OD-08–11 | Restaurar conjunto compatível A a partir de B por caminho autorizado e suportado, com limites de estado explícitos; provar docs/skills/man/binary coerentes. Manter instalação B separada e preservar dados. Recusa comum de downgrade também deve continuar, sem substituir prova de restauração |
| AC-05 / OD-10–11 | Matriz de traversal, symlinks/hard-links, ancestrais/leaf trocados nos boundaries, permissões/ACL, tampering/hashes/manifesto, conteúdo extra, truncamento, quota/disco e instalação repetida; snapshots demonstram confinamento e preservação de notas/estrangeiros |
| AC-06 / OD-01–02,11–12 | Build/release reproduz inventário/proveniência e manpages a partir de fontes aprovadas; compara referência de comandos/flags ao help da CLI exata; verifica exclusões/secrets, links locais e cobertura de subcomandos, sem gerador no host de instalação |
| AC-07 / OD-04–05,12–13 | macOS/Linux: `man axiom`, páginas relevantes e origem selecionada; upgrade/restauração e A/B com PATH e seleção man divergentes; diagnóstico e consulta correta. Repetir sem leitor para comprovar Markdown/help. Windows apenas após suporte autorizado |
| AC-08 / OD-03–06 | Sessões nativas reais, novas e separadas de Codex e Claude, usando skills da instalação efetiva para explicar comando e readiness observado em fixture local controlada. Negar acesso a checkout, memórias/config do mantenedor, search/site/MCP documental; capturar política efetiva e acessos/fontes observáveis, não só declaração da IA. Trocar A/B e testar pacote ausente: sem fallback, sem autoridade inventada |

AC-08 exige observar identidade/versionamento real dos Runtimes e seus limites de
isolamento. Fake Runtime, teste de strings, transcript ou alegação da IA não prova
sessão nativa nem ausência de acesso. Se fornecedor exigir rede para inferência,
separar esse transporte autorizado de acesso a fontes documentais remotas e declarar
limite: não é prova de Runtime air-gapped. AC-01 prova consulta documental realmente
offline sem Runtime. Protocolo e monitoramento suficientes são Q6; nenhuma
capability de confinamento é presumida. Evidence incompleta permanece unverified.

## Questões para Clarification / Plan (não resolvidas)

Estas são as seis perguntas do card, com alternativas ainda abertas. A etapa
seguinte deve reconciliá-las com seu requisito de resolução antes do Plan; análises
de implementação podem fundamentar escolhas no Plan somente após gate apropriado.
Não há decisão humana bloqueando a redação desta proposta. Se escolha mudar o
contrato acima, submeter Clarification antes de tratá-lo como aprovado.

| ID / pergunta | Opções e trade-offs / risco | Próximo responsável e custo de mudança |
| --- | --- | --- |
| Q1 — Qual recorte mínimo de docs? | Recortes de fontes existentes reduzem autoria, mas podem carregar links/contexto do checkout; corpus operacional dedicado em Docs-as-Code melhora jornada, mas exige reconciliação; referência gerada reduz drift de flags, mas não ensina conceitos/diagnóstico. Nenhuma opção autoriza copiar histórico/Notion | Clarification valida suficiência/idiomas e exclusões; Plan mapeia fonte→item, tamanho e cobertura. Ampliar recorte aumenta manutenção e superfície pública |
| Q2 — Qual layout macOS/Linux/Windows e resolução por instalação? | Prefixo da instalação facilita coexistência, mas precisa lidar com roots movidos; diretórios convencionais por plataforma/XDG facilitam descoberta, mas podem colidir entre instalações; localização explícita vinculada a metadados evita heurística, mas exige integração. Windows condicionado ao suporte, sem default adotado | Plan confronta segurança e identidade; Clarification se alterar portabilidade/compatibilidade. Paths/identidade publicados tornam migração e coexistência mais custosas |
| Q3 — Como empacotar e alinhar docs/binário/skill-set/commit? | Arquivos externos no bundle são legíveis sem executar, mas exigem publicação/verificação conjunta; embed + extração simplifica vínculo com binário, mas aumenta artefato e riscos de extração/permissão. Compilar toda documentação no binário continua non-goal. Igualdade de versão não basta em nenhuma opção | Plan analisa formato, proveniência, receipts e builds; mudança de formato/ADR exige revisão explícita. Formato distribuído passa a ter compatibilidade histórica |
| Q4 — Como publicar, restaurar e limpar com atomicidade? | Gerações preservadas com seleção coerente facilitam retorno, mas consomem espaço; staging/substituição com recovery exige protocolo de falha preciso; retenção explícita conserva rollback, limpeza automática pode apagar versão em uso. Rollback não pode contornar downgrade ou corromper estado existente | Plan fundamenta commit point, falhas, retention, autoridade e read-back; Clarification resolve qualquer mudança do contrato de rollback. Bytes/receipts publicados e dados locais tornam reversão cara |
| Q5 — Qual interface expõe localização e versão? | Superfície CLI somente leitura favorece vínculo ao executável; metadados locais legíveis dispensam execução, mas precisam de bootstrap e validação; contrato das skills pode guiar consulta, mas não duplicar resolver/argumentos. MANPATH/prefixos explícitos são alternativas de integração man, não garantia de seleção automática | Plan compara contratos existentes e define interface sem segunda fonte de authority; Clarification se alterar consumo público. Nova interface/schema exige compatibilidade; Cobra/outros geradores só após análise da CLI real |
| Q6 — Como demonstrar rigorosamente ausência de checkout/fallback? | Host/VM/container limpo com observação de acessos oferece isolamento verificável, mas aumenta custo; allowlist/negação de paths e rede em ambiente controlado pode cobrir Runtimes nativos, mas cobertura depende de controles reais; transcript/fixture sozinho é insuficiente. Não presumir acesso a sandbox de fornecedor | Clarification fixa evidência mínima se limites mudarem; Plan detalha protocolo reproduzível para ambos os Runtimes. Sem observação suficiente, aceite permanece pendente, mesmo com testes sintéticos verdes |

## Non-goals e próximo gate

Sem portal novo, nova fonte normativa, reorganização Notion/Wiki, compilação de
toda documentação no binário, distribuição de Specifications históricas,
dependência/gerador/framework adotado, comando/schema/layout implementado,
reintrodução de Windows, migração de dados de Project/Execution, bypass de
authority, aceite do MVP ou liberação automática de AXM-39/AXM-12.

Preservar Axiom != Lingo; Project != Repository; Role != Model; Execution != Agent;
Provider != Transport; Integration != MCP; Skill != workflow source of truth;
Evidence != raw chat history. Aprovação da revisão exata e gate próprio podem
autorizar Clarification/Plan; esta entrega termina na Specification proposta.
