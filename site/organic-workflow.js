/* Approved organic control plane, integrated with the landing settings. */
(() => {
  const story = document.querySelector('.organic-workflow'),
    map = document.querySelector('.organic-map'),
    paths = [...document.querySelectorAll('[data-path]')],
    glows = [...document.querySelectorAll('[data-glow]')],
    nodes = [...document.querySelectorAll('[data-node]')],
    dots = [...document.querySelectorAll('.step-dots a')];

  const desktopPaths = [
    'M450 265 C420 208 365 82 268 82 C199 82 135 74 143 120',
    'M426 296 C368 256 337 326 251 306 C183 290 123 275 140 294',
    'M448 341 C427 417 350 442 282 480 C236 506 177 539 190 480',
    'M550 265 C580 208 635 82 732 82 C801 82 865 74 857 120',
    'M574 296 C632 256 663 326 749 306 C817 290 877 275 860 294',
    'M552 341 C573 417 650 442 718 480 C764 506 823 539 810 480',
  ];
  const mobilePaths = [
    'M455 190 C408 225 494 270 392 286 C320 298 223 260 250 320',
    'M545 190 C592 225 506 270 608 286 C680 298 777 260 750 320',
    'M465 210 C455 328 544 398 415 449 C330 482 201 446 250 512',
    'M535 210 C545 328 456 398 585 449 C670 482 799 446 750 512',
    'M480 213 C477 393 570 606 414 650 C327 674 216 643 250 704',
    'M520 213 C523 393 430 606 586 650 C673 674 784 643 750 704',
  ];
  const copyPT = [
    [
      'Antes dos caminhos, uma intenção.',
      'O Axiom mantém o contexto no centro. Ao rolar, descubra como cada parte se conecta.',
    ],
    [
      'A intenção ganha forma.',
      'Resultados, restrições e critérios de aceitação tornam a ideia uma referência compartilhada.',
    ],
    [
      'As decisões encontram um caminho.',
      'Planos e ADRs preservam o raciocínio. A próxima etapa recebe o contexto de que precisa.',
    ],
    [
      'O trabalho avança com limites.',
      'Agentes em Runtimes como Codex e Claude executam trabalho delimitado, com autoridade explícita.',
    ],
    [
      'O resultado passa por verificação.',
      'Testes e revisão tornam a implementação inspecionável antes de uma conclusão.',
    ],
    [
      'A entrega deixa evidências.',
      'Evidence conecta o que foi solicitado ao que pode ser observado e reproduzido.',
    ],
    [
      'O aceite continua humano.',
      'Você avalia as evidências e decide entre aceitar ou pedir retrabalho. A intenção permanece conectada.',
    ],
  ];
  const copyEN = [
    [
      'Before the paths, an intent.',
      'Axiom keeps context at the center. Scroll to discover how each part connects.',
    ],
    [
      'Intent takes shape.',
      'Outcomes, constraints, and acceptance criteria turn the idea into a shared reference.',
    ],
    [
      'Decisions find a path.',
      'Plans and ADRs preserve the rationale. Each next step gets the context it needs.',
    ],
    [
      'Work moves forward within boundaries.',
      'Agents in Runtimes such as Codex and Claude perform bounded work under explicit authority.',
    ],
    [
      'The result is verified.',
      'Tests and review make the implementation inspectable before a conclusion.',
    ],
    [
      'Delivery leaves evidence.',
      'Evidence connects what was requested to what can be observed and reproduced.',
    ],
    [
      'Acceptance stays human.',
      'You assess the evidence and decide whether to accept or request rework. Intent stays connected.',
    ],
  ];
  let staticMode = document.body.classList.contains('static-motion'),
    frame = 0,
    previousTime = 0,
    visual = 0,
    current = -1,
    swap = 0,
    animation;
  let geometry = { top: 0, range: 1 };
  function clamp(v, min = 0, max = 1) {
    return Math.min(max, Math.max(min, v));
  }
  function smooth(v) {
    v = clamp(v);
    return v * v * (3 - 2 * v);
  }
  function setNarrative(index, immediate = false) {
    current = index;
    const token = ++swap,
      body = document.querySelector('.narrative-text');
    if (animation) animation.cancel();
    const replace = () => {
      const copy = document.documentElement.lang.startsWith('en') ? copyEN : copyPT;
      body.querySelector('h2').textContent = copy[index][0];
      body.querySelector('p').textContent = copy[index][1];
      const total = document.createElement('span');
      total.textContent = '/ 06';
      document
        .querySelector('.narrative-count')
        .replaceChildren(document.createTextNode(`0${index}`), total);
    };
    dots.forEach((a, i) => {
      a.classList.toggle('active', i + 1 === index);
      if (i + 1 === index) a.setAttribute('aria-current', 'step');
      else a.removeAttribute('aria-current');
    });
    if (immediate || staticMode) {
      replace();
      return;
    }
    animation = body.animate(
      [
        { opacity: 1, transform: 'translateY(0)' },
        { opacity: 0, transform: 'translateY(-4px)' },
      ],
      { duration: 160, easing: 'ease-out', fill: 'forwards' },
    );
    animation.finished
      .then(() => {
        if (token !== swap) return;
        replace();
        animation.cancel();
        animation = body.animate(
          [
            { opacity: 0, transform: 'translateY(5px)' },
            { opacity: 1, transform: 'translateY(0)' },
          ],
          { duration: 420, easing: 'cubic-bezier(.22,.61,.36,1)' },
        );
        return animation.finished;
      })
      .catch(() => {});
  }
  function measure() {
    geometry.top = story.getBoundingClientRect().top + scrollY;
    geometry.range = Math.max(1, story.offsetHeight - innerHeight);
    const mobile = innerWidth <= 760,
      collection = mobile ? mobilePaths : desktopPaths;
    document
      .querySelector('.branches')
      .setAttribute('viewBox', mobile ? '0 0 1000 800' : '0 0 1000 600');
    collection.forEach((d, i) => {
      paths[i].setAttribute('d', d);
      glows[i].setAttribute('d', d);
      glows[i].setAttribute('pathLength', '1');
    });
    schedule();
  }
  function paint(now = performance.now()) {
    frame = 0;
    const elapsed = Math.min(64, Math.max(1, now - (previousTime || now - 16)));
    previousTime = now;
    const target = staticMode ? 1 : clamp((scrollY - geometry.top) / geometry.range);
    visual = staticMode ? 1 : visual + (target - visual) * (1 - Math.exp(-elapsed / 190));
    if (Math.abs(target - visual) < 0.00015) visual = target;
    map.style.setProperty('--growth', visual);
    const cursor = visual * 6.8 - 0.4;
    paths.forEach((path, i) => {
      const draw = smooth((cursor - i) / 0.85),
        node = smooth((draw - 0.67) / 0.33),
        focus = smooth(1 - Math.abs(cursor - i - 0.75));
      path.style.setProperty('--draw', staticMode ? 1 : draw);
      glows[i].style.setProperty('--draw', staticMode ? 1 : draw);
      nodes[i].style.setProperty('--node', staticMode ? 1 : node);
      nodes[i].style.setProperty('--focus', staticMode ? 0 : focus);
    });
    const index = staticMode ? 6 : clamp(Math.floor(cursor + 0.6), 0, 6);
    if (index !== current) setNarrative(index);
    if (visual !== target) schedule();
  }
  function schedule() {
    if (!frame) frame = requestAnimationFrame(paint);
  }
  function syncSettings() {
    staticMode = document.body.classList.contains('static-motion');
    const en = document.documentElement.lang.startsWith('en');
    document.querySelector('.organic-workflow .intro-line').textContent = staticMode
      ? en
        ? 'Full map, without animation.'
        : 'Mapa completo, sem animação.'
      : en
        ? 'Scroll to watch the work take shape.'
        : 'Role para ver o trabalho ganhar forma.';
    setNarrative(staticMode ? 6 : Math.max(0, current), true);
    measure();
  }
  addEventListener('axiom:settings', syncSettings);
  addEventListener('scroll', schedule, { passive: true });
  addEventListener('resize', measure, { passive: true });
  addEventListener('load', measure, { once: true });
  new ResizeObserver(measure).observe(story);
  setNarrative(0, true);
  syncSettings();
})();
