import {
  storyState,
  storyConnections,
  flowSpineX,
  dependencySideConnections,
  frameState,
  clamp,
} from './story-state.js?v=5d511770b198';
import {
  frames,
  workContent,
  frameProgress,
  sceneStarts,
  sceneRepresentatives,
} from './story-frames.js?v=5d511770b198';
const story = document.querySelector('.storytelling'),
  world = story.querySelector('.live-world');
const entities = {
  idea: story.querySelector('[data-story-idea]'),
  hub: story.querySelector('[data-story-axiom]'),
  works: [...story.querySelectorAll('[data-story-work]')],
  tools: [...story.querySelectorAll('[data-story-tool]')],
  ghosts: [...story.querySelectorAll('[data-story-ghost]')],
  contexts: [...story.querySelectorAll('[data-story-context]')],
};
const paths = [...story.querySelectorAll('[data-story-path]')],
  signals = [...world.querySelectorAll('.connection-signal')],
  trails = [...world.querySelectorAll('.signal-trail')],
  copies = [...story.querySelectorAll('[data-story-copy]')],
  steps = [...story.querySelectorAll('.story-navigation a')];
const params = new URLSearchParams(location.search),
  fixedFrame = Number(params.get('frame')),
  fixedScene = Number(params.get('scene'));
const previewIndex =
  fixedFrame >= 1 && fixedFrame <= frames.length
    ? fixedFrame - 1
    : fixedScene >= 1 && fixedScene <= 5
      ? sceneRepresentatives[fixedScene - 1]
      : null;
let frame = 0,
  top = 0,
  range = 1,
  width = 1,
  height = 1,
  renderedProgress = null,
  lastPaint = 0;
function measure() {
  top = story.getBoundingClientRect().top + scrollY;
  const rect = world.getBoundingClientRect();
  width = rect.width;
  height = rect.height;
  range = Math.max(1, story.offsetHeight - innerHeight);
  schedule();
}
function place(el, n) {
  el.style.transform = `translate3d(${(n.x * width) / 100}px,${(n.y * height) / 100}px,0) translate(-50%,-50%) rotate(${n.rotation}deg) scale(${n.scale ?? 1})`;
  el.style.width = n.w + 'px';
  el.style.height = n.h + 'px';
  el.style.opacity = n.opacity;
  el.style.visibility = n.opacity > 0.001 ? 'visible' : 'hidden';
}
const alpha = (selector, value) => {
  const el = world.querySelector(selector);
  el.style.opacity = value;
  el.style.visibility = value > 0.001 ? 'visible' : 'hidden';
};
function paint(now) {
  frame = 0;
  const staticMode = document.body.classList.contains('static-motion');
  document.body.classList.toggle('story-motion', !staticMode || previewIndex !== null);
  const target =
    previewIndex !== null ? frameProgress(previewIndex) : clamp((scrollY - top) / range) * 4;
  if (renderedProgress === null || staticMode || previewIndex !== null) renderedProgress = target;
  const dt = Math.min(64, now - (lastPaint || now));
  lastPaint = now;
  renderedProgress += (target - renderedProgress) * (1 - Math.exp(-dt / 95));
  const settled = Math.abs(target - renderedProgress) < 0.000005;
  if (settled) renderedProgress = target;
  const p = renderedProgress,
    s = storyState(p, innerWidth <= 760, width),
    lang = document.documentElement.lang.startsWith('en') ? 'en' : 'pt';
  story.dataset.storySettled = String(settled);
  world.dataset.ambient = String(!staticMode && previewIndex === null);
  if (!settled) schedule();
  story.dataset.storyProgress = p.toFixed(6);
  story.dataset.storyScene = String(s.scene + 1);
  story.dataset.storyFrame = String(s.index + 1);
  story.style.setProperty('--story-wash', s.wash);
  document.body.classList.toggle(
    'gold-start',
    (!staticMode || previewIndex !== null) && s.wash > 0.65 && scrollY < top + innerHeight,
  );
  if (staticMode && previewIndex === null) return;
  world.dataset.layout = s.layout;
  world.style.setProperty('--tool-grouping', s.toolGrouping);
  place(entities.idea, s.idea);
  entities.idea.style.setProperty('--idea-card', s.ideaCard);
  entities.idea.querySelector('.idea-label').style.opacity = s.ideaCard;
  entities.idea.querySelector('.orb-ring').style.opacity = s.orbRing * (1 - s.ideaCard);
  place(entities.hub, s.hub);
  entities.hub.style.borderRadius = `${14 + s.hubRound * 36}%`;
  entities.hub.querySelector('.story-mascot').style.opacity = 1 - s.morph;
  const before = frameState(s.from, s.narrow, width),
    after = frameState(s.from + 1, s.narrow, width);
  for (const group of ['works', 'tools', 'ghosts', 'contexts'])
    entities[group].forEach((el, i) => {
      const entering = before[group][i].opacity < 0.05 && after[group][i].opacity > 0.1,
        roleChange =
          group === 'works' && workContent(s.from, i).kind !== workContent(s.from + 1, i).kind,
        growth = s.reveals[i % 6].card,
        fade = entering
          ? growth
          : roleChange
            ? s.transition < 0.5
              ? s.contentOpacity
              : growth
            : 1;
      place(el, {
        ...s[group][i],
        opacity: s[group][i].opacity * fade,
        scale:
          entering || (roleChange && s.transition >= 0.5)
            ? 0.78 + growth * 0.22
            : (s[group][i].scale ?? 1) * (1 + s.reveals[i % 6].bloom * 0.025),
      });
      if (group === 'ghosts') {
        el.style.setProperty('--pull-x', `${((50 - s[group][i].x) * width) / 500}px`);
        el.style.setProperty('--pull-y', `${((50 - s[group][i].y) * height) / 500}px`);
      }
      el.style.setProperty(
        '--card-bloom',
        after[group][i].opacity > 0.1 && (entering || roleChange) ? s.reveals[i % 6].bloom : 0,
      );
    });
  entities.works.forEach((el, i) => {
    const c = workContent(s.index, i);
    el.dataset.kind = c.kind;
    for (const field of ['title', 'detail', 'metric']) {
      const target = el.querySelector('.work-' + field);
      target.dataset.pt = c[field].pt;
      target.dataset.en = c[field].en;
      target.textContent = c[field][lang];
    }
    const agent = el.querySelector('.work-agent');
    agent.hidden = !c.agent;
    agent.querySelector('span').textContent = c.agent;
    const mark = agent.querySelector('img');
    mark.src = `assets/${c.agent.startsWith('Claude') ? 'claude' : 'openai'}-mark.svg`;
    mark.classList.toggle('claude-mark', c.agent.startsWith('Claude'));
    el.querySelector('.story-icon use').setAttribute('href', '#story-icon-' + c.icon);
    const beforeText = workContent(s.from, i),
      afterText = workContent(s.from + 1, i);
    const changed = JSON.stringify(beforeText) !== JSON.stringify(afterText);
    el.querySelector('.work-body').style.opacity = changed ? s.contentOpacity : 1;
  });
  storyConnections(s, width, height).forEach((connection, i) => {
    const path = paths[i];
    path.setAttribute('d', connection.d);
    path.setAttribute('marker-end', s.layout === 'dependencies' ? 'url(#story-flow-arrow)' : '');
    path.style.opacity = connection.opacity;
    signals[i].setAttribute('cx', connection.signalEnd[0]);
    signals[i].setAttribute('cy', connection.signalEnd[1]);
    signals[i].setAttribute('rx', 280 / width);
    signals[i].setAttribute('ry', 280 / height);
    signals[i].style.opacity = connection.opacity * connection.signal;
    trails[i].setAttribute('d', connection.trail);
    trails[i].style.opacity = connection.opacity * connection.signal;
  });
  const dependency = s.layout === 'dependencies' ? s.routingFade : 0;
  const sideConnections = dependencySideConnections(s, width, height);
  world.querySelector('.dependency-shortcut').setAttribute('d', sideConnections[2]);
  world.querySelector('.dependency-shortcut').style.opacity = dependency * 0.65;
  world.querySelectorAll('.context-link').forEach((path, i) => {
    path.setAttribute('d', sideConnections[i]);
    path.style.opacity = s.contexts[i].opacity * dependency;
  });
  alpha('.story-wave', Math.max(s.energy * 0.75, (1 - s.wash) * 0.45) * (1 - s.toolGrouping * 0.7));
  alpha('.story-ground', s.ground);
  alpha('.execution-shell', s.execution);
  alpha('.story-decision', s.decision);
  alpha('.delivery-chips', s.chips);
  alpha('.story-continuity', s.continuity);
  place(world.querySelector('.story-continuity'), {
    x: 72,
    y: 50,
    w: innerWidth <= 760 ? 140 : 220,
    h: innerWidth <= 760 ? 140 : 170,
    opacity: s.continuity,
    rotation: 0,
  });
  alpha('.flow-ports', s.flowPorts);
  world.querySelectorAll('.flow-port').forEach((el, i) => {
    el.setAttribute('cx', flowSpineX(s, width));
    el.setAttribute('cy', s.works[i].y);
    el.setAttribute('rx', (4 / width) * 100);
    el.setAttribute('ry', (4 / height) * 100);
  });
  world.querySelectorAll('.flow-fork').forEach((el, i) => {
    const x = flowSpineX(s, width),
      y = s.works[i].y;
    el.setAttribute('d', `M ${x} ${y} H ${x + 7} H ${s.works[i].x}`);
  });
  alpha('.story-supernova', s.novaOpacity);
  world.querySelector('.story-supernova').style.setProperty('--nova-scale', s.novaScale);
  copies.forEach((el, i) => {
    el.style.opacity = s.copy[i];
    el.style.visibility = s.copy[i] > 0.001 ? 'visible' : 'hidden';
    el.style.transform = `translateY(${(1 - s.copy[i]) * 10}px)`;
  });
  story.querySelector('.story-disclaimer').style.opacity =
    clamp((s.beatProgress - sceneStarts[3]) / 0.6) ** 2 *
    (3 - 2 * clamp((s.beatProgress - sceneStarts[3]) / 0.6));
  story.querySelector('.story-counter').textContent =
    `0${s.scene + 1} — 05 · 0${s.beat} / 0${frames.filter((f) => f.scene === s.scene + 1).length}`;
  steps.forEach((el, i) => {
    if (i === s.scene) el.setAttribute('aria-current', 'step');
    else el.removeAttribute('aria-current');
  });
}
function schedule() {
  if (!frame) frame = requestAnimationFrame(paint);
}
addEventListener('scroll', schedule, { passive: true });
addEventListener('resize', measure, { passive: true });
addEventListener('axiom:settings', measure);
addEventListener(
  'load',
  () => {
    measure();
    if (previewIndex === null && /^#(?:cena-\d|quadro-\d-\d|fluxo|proposta)$/.test(location.hash)) {
      requestAnimationFrame(() =>
        document.getElementById(location.hash.slice(1))?.scrollIntoView({ behavior: 'instant' }),
      );
    }
  },
  { once: true },
);
document.fonts.ready.then(measure);
new ResizeObserver(measure).observe(story);
measure();
