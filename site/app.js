/* Axiom immersive landing. Native scrolling, no scroll interception or autoplay. */
const root = document.documentElement;
const i18n = [...document.querySelectorAll('[data-en]')];
i18n.forEach((el) => (el.dataset.pt = el.textContent));
const ariaI18n = [...document.querySelectorAll('[data-en-aria]')];
ariaI18n.forEach((el) => (el.dataset.ptAria = el.getAttribute('aria-label')));
const opening = document.querySelector('.opening'),
  motionButton = document.querySelector('#motion'),
  camera = document.querySelector('.logo-camera svg');
const setupTabs = [...document.querySelectorAll('[data-setup]')];
const reduced = matchMedia('(prefers-reduced-motion: reduce)');
let language = 'pt',
  manualStatic = new URLSearchParams(location.search).has('preview'),
  staticMode = reduced.matches,
  selectedSetup = 0,
  frame = 0;
let bounds = { openingTop: 0, openingRange: 1, scrollRange: 1 };
const installCommand =
  'curl -fsSL https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.sh | sh';
const setupCommands = [
  '',
  'axiom version\naxiom help\naxiom first-run',
  'axiom project configure\naxiom project list\naxiom project show --selector my-project',
  'axiom work-item create --project my-project --repository main --provider-repository owner/repository --classification type:task --classification area:cli\n\naxiom workflow start --project my-project --repository main --number 123\n\naxiom workflow status --project my-project --repository main --number 123',
];
const comments = {
  pt: [
    '# instale a release estável mais recente',
    '# verifique o binário e prepare as skills',
    '# configure e inspecione seu Project',
    '# exemplo: crie e acompanhe o Work Item',
  ],
  en: [
    '# install the latest stable release',
    '# verify the binary and prepare skills',
    '# configure and inspect your Project',
    '# example: create and follow a Work Item',
  ],
};
function clamp(n, min = 0, max = 1) {
  return Math.min(max, Math.max(min, n));
}
function smooth(n) {
  n = clamp(n);
  return n * n * (3 - 2 * n);
}
function updateTerminal() {
  renderCommands();
  document.querySelector('.terminal-comment').textContent = comments[language][selectedSetup];
  document.querySelector('.terminal-platform').textContent =
    selectedSetup === 0
      ? language === 'pt'
        ? 'Hosts suportados: macOS arm64 e Linux amd64/arm64.'
        : 'Supported hosts: macOS arm64 and Linux amd64/arm64.'
      : language === 'pt'
        ? 'Valores de exemplo · adapte ao seu ambiente'
        : 'Example values · adapt to your environment';
  document.querySelector('.os-tabs').hidden = selectedSetup !== 0;
  document.querySelector('#copy-status').textContent = '';
  document
    .querySelector('#terminal-panel')
    .setAttribute('aria-labelledby', `setup-${selectedSetup}`);
  setupTabs.forEach((t, i) => {
    t.setAttribute('aria-selected', String(i === selectedSetup));
    t.tabIndex = i === selectedSetup ? 0 : -1;
  });
}
function chooseSetup(index, focus = false) {
  selectedSetup = index;
  updateTerminal();
  if (focus) setupTabs[index].focus();
}
setupTabs.forEach((tab, i) => {
  tab.addEventListener('click', () => chooseSetup(i));
  tab.addEventListener('keydown', (event) => {
    let n;
    if (event.key === 'ArrowDown' || event.key === 'ArrowRight') n = (i + 1) % 4;
    else if (event.key === 'ArrowUp' || event.key === 'ArrowLeft') n = (i + 3) % 4;
    else if (event.key === 'Home') n = 0;
    else if (event.key === 'End') n = 3;
    else return;
    event.preventDefault();
    chooseSetup(n, true);
  });
});
function renderCommands() {
  const list = document.querySelector('#commands');
  const commands = (selectedSetup === 0 ? installCommand : setupCommands[selectedSetup])
    .split(/\n+/)
    .filter((command) => command.trim());
  list.replaceChildren(
    ...commands.map((command, index) => {
      const row = document.createElement('div');
      row.className = 'command-row';
      const pre = document.createElement('pre'),
        code = document.createElement('code');
      code.textContent = command;
      pre.append(code);
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'copy-command';
      button.textContent = language === 'pt' ? 'Copiar ⧉' : 'Copy ⧉';
      button.setAttribute(
        'aria-label',
        `${language === 'pt' ? 'Copiar comando' : 'Copy command'} ${index + 1}: ${command}`,
      );
      button.addEventListener('click', async () => {
        const status = document.querySelector('#copy-status');
        try {
          await navigator.clipboard.writeText(command);
          if (!button.isConnected) return;
          button.textContent = language === 'pt' ? 'Copiado ✓' : 'Copied ✓';
          status.textContent =
            language === 'pt' ? `Comando ${index + 1} copiado.` : `Command ${index + 1} copied.`;
        } catch {
          if (!button.isConnected) return;
          const selection = getSelection(),
            range = document.createRange();
          range.selectNodeContents(code);
          selection.removeAllRanges();
          selection.addRange(range);
          status.textContent =
            language === 'pt'
              ? 'Comando selecionado. Copie com Ctrl+C ou ⌘C.'
              : 'Command selected. Copy with Ctrl+C or ⌘C.';
        }
      });
      row.append(pre, button);
      return row;
    }),
  );
}
function measure() {
  bounds.width = innerWidth;
  bounds.height = innerHeight;
  bounds.logoWidth =
    innerWidth <= 760 ? Math.min(innerWidth * 0.96, 400) : Math.min(innerWidth * 0.58, 520);
  if (innerHeight <= 720) bounds.logoWidth = innerWidth <= 760 ? 300 : 400;
  bounds.openingTop = opening.getBoundingClientRect().top + scrollY;
  bounds.openingRange = Math.max(1, opening.offsetHeight - innerHeight);
  bounds.scrollRange = Math.max(1, document.body.scrollHeight - innerHeight);
  schedule();
}
function paint() {
  frame = 0;
  root.style.setProperty('--page-progress', clamp(scrollY / bounds.scrollRange));
  const p = staticMode ? 1 : clamp((scrollY - bounds.openingTop) / bounds.openingRange),
    reveal = smooth(p / 0.78);
  const worldWidth = Math.exp(
      Math.log(20) * (1 - reveal) + Math.log((600 * bounds.width) / bounds.logoWidth) * reveal,
    ),
    worldHeight = (worldWidth * bounds.height) / bounds.width;
  camera.setAttribute(
    'viewBox',
    `${300 - worldWidth * 0.5} ${203 - worldHeight * (0.5 - 0.21 * reveal)} ${worldWidth} ${worldHeight}`,
  );
  root.style.setProperty('--wash', 1 - smooth(p / 0.16));
  root.style.setProperty('--intro-copy', 1 - smooth(p / 0.22));
  root.style.setProperty('--final-copy', smooth((p - 0.68) / 0.23));
  document.body.classList.toggle('gold-start', !staticMode && p < 0.13);
}
function schedule() {
  if (!frame) frame = requestAnimationFrame(paint);
}
function syncMotion() {
  staticMode = manualStatic || reduced.matches;
  document.body.classList.toggle('static-motion', staticMode);
  root.style.scrollBehavior = staticMode ? 'auto' : '';
  motionButton.setAttribute('aria-pressed', String(staticMode));
  motionButton.textContent = staticMode ? '▷' : 'Ⅱ';
  const label = reduced.matches
    ? language === 'pt'
      ? 'Movimento reduzido conforme preferência do sistema'
      : 'Reduced motion follows system preference'
    : staticMode
      ? language === 'pt'
        ? 'Ativar movimento'
        : 'Enable motion'
      : language === 'pt'
        ? 'Desativar movimento'
        : 'Disable motion';
  motionButton.setAttribute('aria-label', label);
  motionButton.title = label;
  if (reduced.matches) motionButton.setAttribute('aria-disabled', 'true');
  else motionButton.removeAttribute('aria-disabled');
  measure();
  dispatchEvent(new Event('axiom:settings'));
}
motionButton.addEventListener('click', () => {
  if (reduced.matches) return;
  manualStatic = !manualStatic;
  syncMotion();
});
reduced.addEventListener('change', syncMotion);
function setLanguage(next) {
  language = next;
  root.lang = next === 'pt' ? 'pt-BR' : 'en';
  i18n.forEach((el) => (el.textContent = el.dataset[next]));
  ariaI18n.forEach((el) => el.setAttribute('aria-label', el.dataset[next + 'Aria']));
  const button = document.querySelector('#language');
  button.textContent = next === 'pt' ? 'EN' : 'PT';
  button.setAttribute('aria-label', next === 'pt' ? 'Switch to English' : 'Mudar para português');
  document.title =
    next === 'pt'
      ? 'Axiom — Toda intenção merece um caminho.'
      : 'Axiom — Every intent deserves a path.';
  updateTerminal();
  syncMotion();
  try {
    localStorage.setItem('axiom-language', next);
  } catch {
    // Storage is optional; settings still work for this page.
  }
}
document
  .querySelector('#language')
  .addEventListener('click', () => setLanguage(language === 'pt' ? 'en' : 'pt'));
addEventListener('scroll', schedule, { passive: true });
addEventListener('resize', measure, { passive: true });
addEventListener('load', measure, { once: true });
new ResizeObserver(measure).observe(document.body);
try {
  if (localStorage.getItem('axiom-language') === 'en') setLanguage('en');
} catch {
  // Storage is optional; settings still work for this page.
}
const revealObserver = new IntersectionObserver(
  (entries) =>
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        entry.target.classList.add('is-visible');
        revealObserver.unobserve(entry.target);
      }
    }),
  { threshold: 0.1 },
);
document
  .querySelectorAll(
    '.manifesto-title,.manifesto-bottom,.human-layout,.install-layout,.reality-grid,.closing-actions',
  )
  .forEach((el) => {
    el.classList.add('reveal-ready');
    revealObserver.observe(el);
  });
updateTerminal();
syncMotion();
