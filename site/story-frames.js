// Five supplied storyboards, read left-to-right, five beats each.
export const text = (pt, en) => ({ pt, en });
export const sceneNames = [
  text('O nascimento da ideia', 'The birth of an idea'),
  text('A complexidade', 'Complexity'),
  text('A entrada do Axiom', 'Enter Axiom'),
  text('Orquestração e controle', 'Orchestration and control'),
  text('A entrega verificável', 'Verifiable delivery'),
];
export const fronts = [
  text('Requisitos', 'Requirements'),
  text('Arquitetura', 'Architecture'),
  text('Código', 'Code'),
  text('Testes', 'Tests'),
  text('Revisão', 'Review'),
  text('Entrega', 'Delivery'),
];
export const stages = [
  text('Specification', 'Specification'),
  text('Arquitetura', 'Architecture'),
  text('Implementação', 'Implementation'),
  text('Validação e testes', 'Validation and tests'),
  text('Revisão humana', 'Human review'),
  text('Entrega', 'Delivery'),
];
export const tools = [
  'Notion',
  'Claude',
  'Codex',
  'GitHub',
  'Docs',
  'Decisões',
  'CLI',
  'Issues',
  '…',
];
const f = (scene, beat, layout, pt, en, lead, leadEn) => ({
  scene,
  beat,
  layout,
  title: text(pt, en),
  lead: text(lead, leadEn),
});
export const frames = [
  f(
    1,
    1,
    'opening',
    'Uma ideia.\nInfinitas possibilidades.',
    'One idea.\nInfinite possibilities.',
    'Dê uma direção à sua intenção.',
    'Give your intent a direction.',
  ),
  f(
    1,
    2,
    'intention',
    'Uma intenção.',
    'An intention.',
    'O ponto de partida de tudo.',
    'The starting point of everything.',
  ),
  f(
    1,
    3,
    'idea',
    'Uma ideia.',
    'An idea.',
    'Uma necessidade, um problema a resolver ou uma oportunidade de criar algo novo.',
    'A need, a problem to solve, or an opportunity to create something new.',
  ),
  f(
    1,
    4,
    'unfold',
    'Cada ideia se desdobra\nem muitas partes.',
    'Every idea unfolds\ninto many parts.',
    'É aqui que o desafio começa.',
    'This is where the challenge begins.',
  ),
  f(
    1,
    5,
    'fronts',
    'Mais elementos. Mais decisões.\nMais contextos.',
    'More elements. More decisions.\nMore contexts.',
    'Como manter tudo conectado?',
    'How do you keep it all connected?',
  ),
  f(
    2,
    1,
    'idea',
    'Uma ideia.',
    'An idea.',
    'Tudo começa com uma intenção.',
    'Everything starts with an intention.',
  ),
  f(
    2,
    2,
    'fronts',
    'Uma ideia.',
    'An idea.',
    'Rapidamente surgem várias frentes.',
    'Many workstreams quickly emerge.',
  ),
  f(
    2,
    3,
    'tools',
    'Múltiplas ferramentas.\nMúltiplos agentes.',
    'Multiple tools.\nMultiple agents.',
    'O contexto começa a se fragmentar.',
    'Context begins to fragment.',
  ),
  f(
    2,
    4,
    'fragmentation',
    'O trabalho se dispersa.',
    'Work becomes scattered.',
    'Requisitos, código, decisões, revisões e entregas vivem em lugares diferentes.',
    'Requirements, code, decisions, reviews and deliveries live in different places.',
  ),
  f(
    2,
    5,
    'tension',
    'Agentes aceleram tarefas.\nMas quem mantém tudo conectado?',
    'Agents accelerate tasks.\nBut who keeps everything connected?',
    'Uma camada de orquestração é necessária.',
    'An orchestration layer is needed.',
  ),
  f(
    3,
    1,
    'scattered',
    'O que estava disperso\ncomeça a se conectar.',
    'What was scattered\nbegins to connect.',
    'A mesma intenção. Os mesmos elementos.',
    'The same intent. The same elements.',
  ),
  f(
    3,
    2,
    'emerge',
    'O Axiom emerge.',
    'Axiom emerges.',
    'Um elemento de organização para intenção, decisões, código, agentes e evidências.',
    'An organizing element for intent, decisions, code, agents and evidence.',
  ),
  f(
    3,
    3,
    'connect',
    'O Axiom conecta\no que estava disperso.',
    'Axiom connects\nwhat was scattered.',
    'Coordenação, contexto compartilhado e limites.',
    'Coordination, shared context and boundaries.',
  ),
  f(
    3,
    4,
    'unified',
    'Um fluxo coerente.',
    'A coherent flow.',
    'Intenção, decisões, código e evidências conectados, com controle humano.',
    'Intent, decisions, code and evidence connected, with human control.',
  ),
  f(
    3,
    5,
    'platform',
    'Pronto para orquestração.',
    'Ready for orchestration.',
    'Os elementos conectados evoluem para um fluxo configurável.',
    'Connected elements evolve into a configurable workflow.',
  ),
  f(
    4,
    1,
    'map',
    'Da intenção ao fluxo.',
    'From intent to workflow.',
    'Etapas, agentes e limites explícitos.',
    'Explicit stages, agents and boundaries.',
  ),
  f(
    4,
    2,
    'lanes',
    'Um processo configurável.',
    'A configurable process.',
    'Você define agentes, modelos e responsabilidades por etapa.',
    'You define agents, models and responsibilities for each stage.',
  ),
  f(
    4,
    3,
    'dependencies',
    'Dependências e contexto.',
    'Dependencies and context.',
    'O Axiom gerencia o fluxo e as relações. O resultado de cada etapa alimenta a próxima.',
    'Axiom manages the flow and its relationships. Each stage feeds the next.',
  ),
  f(
    4,
    4,
    'human-gate',
    'Com controle humano.',
    'With human control.',
    'Você revisa os resultados e decide quando avançar.',
    'You review the results and decide when to proceed.',
  ),
  f(
    4,
    5,
    'execution',
    'Uma visão da execução.',
    'One execution view.',
    'Etapas, agentes e progresso conectados em um só lugar.',
    'Stages, agents and progress connected in one place.',
  ),
  f(
    5,
    1,
    'convergence',
    'O fluxo converge.',
    'The flow converges.',
    'O que foi construído se une em um resultado verificável.',
    'What was built comes together as a verifiable result.',
  ),
  f(
    5,
    2,
    'evidence',
    'Não apenas trabalho concluído.\nTrabalho verificável.',
    'Not just completed work.\nVerifiable work.',
    'Código, testes, diff, artefatos e Evidence: resultados inspecionáveis e rastreáveis.',
    'Code, tests, diff, artifacts and Evidence: inspectable, traceable results.',
  ),
  f(
    5,
    3,
    'decision',
    'Você decide o próximo passo.',
    'You decide the next step.',
    'Revisão, aceite ou retrabalho. A autoridade continua humana.',
    'Review, acceptance or rework. Authority remains human.',
  ),
  f(
    5,
    4,
    'delivered',
    'Entregue. Com evidências.',
    'Delivered. With evidence.',
    'Da intenção ao resultado, preservando decisões e contexto.',
    'From intent to outcome, preserving decisions and context.',
  ),
  f(
    5,
    5,
    'continuity',
    'O contexto permanece.\nVocê segue em frente.',
    'Context remains.\nYou move forward.',
    'Sua próxima ideia. Conectada desde o início.',
    'Your next idea. Connected from the start.',
  ),
];
export function workContent(index, i) {
  const layout = frames[index].layout;
  const c = {
    title: fronts[i],
    detail: text('', ''),
    metric: text('', ''),
    icon: ['spec', 'tree', 'code', 'check', 'review', 'cube'][i],
    kind: 'front',
    agent: '',
  };
  if (index === 4 && i === 0) c.title = stages[0];
  if (['fragmentation', 'tension'].includes(layout)) {
    c.title = [
      text('Specification v1 → v2', 'Specification v1 → v2'),
      text('Arquitetura · ADR', 'Architecture · ADR'),
      text('Discussão em chat', 'Chat discussion'),
      text('Código em construção', 'Code in progress'),
      text('Testes', 'Tests'),
      text('Deploy · pendente', 'Deploy · pending'),
    ][i];
    c.detail = [
      text('Requisitos e mudanças', 'Requirements and changes'),
      text('Decisões em outro contexto', 'Decisions in another context'),
      text('O contexto se fragmenta', 'Context fragments'),
      text('function deliver() { … }', 'function deliver() { … }'),
      text('Validações separadas', 'Separate validations'),
      text('Entrega ainda desconectada', 'Delivery still disconnected'),
    ][i];
    c.kind = 'artifact';
  }
  if (['lanes', 'dependencies', 'human-gate', 'execution'].includes(layout)) {
    c.title = stages[i];
    c.kind = layout;
    if (layout === 'lanes') {
      if (i === 4) c.title = text('Revisão e entrega', 'Review and delivery');
      c.detail = [
        text('Definir o que construir e por quê.', 'Define what to build and why.'),
        text('Decisões e trade-offs.', 'Decisions and trade-offs.'),
        text('Execução com limites.', 'Execution with boundaries.'),
        text('Verificar antes de concluir.', 'Verify before concluding.'),
        text('Evidence e aceite humano.', 'Evidence and human acceptance.'),
        text('', ''),
      ][i];
      c.agent = i === 2 || i === 3 ? 'Codex · o4-mini' : 'Claude · Sonnet 4';
    }
    if (layout === 'dependencies' && i === 3) c.title = text('Testes', 'Tests');
    if (layout === 'human-gate') {
      c.detail =
        i === 4
          ? text('Analise o diff e os resultados.', 'Inspect the diff and results.')
          : i === 5
            ? text('Aguardando aprovação.', 'Awaiting approval.')
            : text('Codex · o4-mini', 'Codex · o4-mini');
      c.metric =
        i === 5
          ? text('Pendente', 'Pending')
          : i === 4
            ? text('Revisar', 'Review')
            : text('Concluído ✓', 'Completed ✓');
    }
    if (layout === 'execution')
      c.metric =
        i < 2
          ? text('✓', '✓')
          : i === 2
            ? text('Em execução · 12m', 'Running · 12m')
            : text('Pendente', 'Pending');
  }
  if (layout === 'convergence') {
    c.title =
      i === 0
        ? stages[0]
        : i === 1
          ? stages[1]
          : i === 2
            ? stages[2]
            : i === 3
              ? fronts[3]
              : i === 4
                ? fronts[4]
                : text('Resultado', 'Outcome');
    c.kind = i === 5 ? 'cube' : 'contributor';
  }
  if (layout === 'evidence') {
    c.title = [
      text('Código', 'Code'),
      text('Testes', 'Tests'),
      text('Diff', 'Diff'),
      text('Artefatos', 'Artifacts'),
      text('Evidence', 'Evidence'),
      text('', ''),
    ][i];
    c.detail = [
      text('Pull Request #123', 'Pull Request #123'),
      text('12/12 passaram', '12/12 passed'),
      text('Revisão concluída', 'Review completed'),
      text('Build e packages', 'Build and packages'),
      text('Logs, resultados e contexto', 'Logs, results and context'),
      text('', ''),
    ][i];
    c.metric = i === 0 ? text('+247 −12 ↗', '+247 −12 ↗') : text('✓', '✓');
    c.icon = ['folder', 'check', 'spec', 'cube', 'database', 'cube'][i];
    c.kind = 'evidence';
  }
  if (layout === 'delivered' && i === 5) {
    c.title = text('Entregue', 'Delivered');
    c.detail = text('Com evidências.', 'With evidence.');
    c.icon = 'check';
    c.kind = 'delivered';
  }
  return c;
}
