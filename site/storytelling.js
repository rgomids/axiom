import { storyState, connectionPath, dependencyPath, clamp } from './story-state.js';
import { frames, workContent } from './story-frames.js';
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
  copies = [...story.querySelectorAll('[data-story-copy]')],
  steps = [...story.querySelectorAll('.story-navigation a')];
const params = new URLSearchParams(location.search),
  fixedFrame = Number(params.get('frame')),
  fixedScene = Number(params.get('scene'));
const previewIndex =
  fixedFrame >= 1 && fixedFrame <= 25
    ? fixedFrame - 1
    : fixedScene >= 1 && fixedScene <= 5
      ? [0, 9, 12, 19, 24][fixedScene - 1]
      : null;
let frame = 0,
  top = 0,
  range = 1,
  width = 1,
  height = 1;
function measure() {
  top = story.getBoundingClientRect().top + scrollY;
  const rect = world.getBoundingClientRect();
  width = rect.width;
  height = rect.height;
  range = Math.max(1, story.offsetHeight - innerHeight);
  schedule();
}
function place(el, n) {
  el.style.transform = `translate3d(${(n.x * width) / 100}px,${(n.y * height) / 100}px,0) translate(-50%,-50%) rotate(${n.rotation}deg)`;
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
function paint() {
  frame = 0;
  const staticMode = document.body.classList.contains('static-motion');
  document.body.classList.toggle('story-motion', !staticMode || previewIndex !== null);
  const p = previewIndex !== null ? previewIndex / 6 : clamp((scrollY - top) / range) * 4,
    s = storyState(p, innerWidth <= 760),
    lang = document.documentElement.lang.startsWith('en') ? 'en' : 'pt';
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
  place(entities.idea, s.idea);
  entities.idea.dataset.card = String(Math.round(s.ideaCard));
  entities.idea.querySelector('.idea-label').style.opacity = s.ideaCard;
  entities.idea.querySelector('.orb-ring').style.opacity = s.orbRing;
  place(entities.hub, s.hub);
  entities.hub.style.borderRadius = `${14 + s.hubRound * 36}%`;
  entities.hub.querySelector('.story-mascot').style.opacity = 1 - s.morph;
  entities.hub.querySelector('span').style.opacity = 1 - s.morph;
  for (const group of ['works', 'tools', 'ghosts', 'contexts'])
    entities[group].forEach((el, i) => place(el, s[group][i]));
  const roleChange =
    frames[Math.floor(s.beatProgress)].layout !==
    frames[Math.min(24, Math.ceil(s.beatProgress))].layout;
  entities.works.forEach((el, i) => {
    const c = workContent(s.index, i);
    el.dataset.kind = c.kind;
    for (const field of ['title', 'detail', 'metric']) {
      const target = el.querySelector('.work-' + field);
      target.dataset.pt = c[field].pt;
      target.dataset.en = c[field].en;
      target.textContent = c[field][lang];
    }
    el.querySelector('.work-agent').textContent = c.agent;
    el.querySelector('.story-icon use').setAttribute('href', '#story-icon-' + c.icon);
    el.querySelector('.work-body').style.opacity = roleChange ? s.contentOpacity : 1;
  });
  paths.forEach((path, i) => {
    const n = s.works[i],
      center = s.hub.opacity > 0.3 ? [s.hub.x, s.hub.y] : [s.idea.x, s.idea.y];
    let from = center,
      to = [n.x, n.y],
      opacity = s.connections * n.opacity;
    if (s.layout === 'convergence') {
      from = [n.x, n.y];
      to = [s.works[5].x, s.works[5].y];
      if (i === 5) opacity = 0;
    }
    if (s.layout === 'lanes') {
      from = [s.works[0].x - 29, i ? s.works[i - 1].y : s.works[0].y];
      to = [n.x - 29, n.y];
      if (i === 5) opacity = 0;
    }
    if (s.layout === 'dependencies') {
      const prev = { 1: 0, 2: 1, 4: 2, 5: 4, 3: 2 }[i];
      if (prev === undefined) opacity = 0;
      else from = [s.works[prev].x, s.works[prev].y];
    }
    if (s.layout === 'human-gate') {
      from = [s.works[4].x - 44, s.works[4].y];
      to = [n.x - 44, n.y];
      opacity = i === 5 ? 0.6 : 0;
    }
    if (s.layout === 'continuity') {
      from = [s.idea.x, s.idea.y];
      to = [72, 50];
      opacity = i === 0 ? 0.7 : 0;
    }
    if (s.layout === 'dependencies')
      to = i === 3 ? [n.x - (n.w / width) * 50 - 1, n.y] : [n.x, n.y - (n.h / height) * 50 - 1];
    path.setAttribute(
      'd',
      s.layout === 'dependencies'
        ? dependencyPath(from, to, i === 3)
        : connectionPath(from, to, i, s.grid),
    );
    path.setAttribute('marker-end', s.layout === 'dependencies' ? 'url(#story-flow-arrow)' : '');
    path.style.opacity = opacity;
  });
  world.querySelectorAll('.context-link').forEach((path, i) => {
    const origin = s.works[i === 0 ? 1 : 2];
    path.setAttribute(
      'd',
      connectionPath([origin.x, origin.y], [s.contexts[i].x, s.contexts[i].y], i),
    );
    path.style.opacity = s.contexts[i].opacity;
  });
  alpha('.story-energy', s.energy);
  alpha('.story-grid', s.grid);
  alpha('.story-ground', s.ground);
  alpha('.story-beam', s.beam);
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
  alpha('.flow-ports', s.layout === 'lanes' ? 1 : 0);
  world.querySelectorAll('.flow-port').forEach((el, i) => {
    el.setAttribute('cx', s.works[0].x - 29);
    el.setAttribute('cy', s.works[i].y);
    el.setAttribute('rx', (4 / width) * 100);
    el.setAttribute('ry', (4 / height) * 100);
  });
  world.querySelectorAll('.flow-fork').forEach((el, i) => {
    const x = s.works[0].x - 29,
      y = s.works[i].y;
    el.setAttribute('d', `M ${x} ${y} H ${x + 7} H ${s.works[i].x}`);
  });
  alpha('.story-particles', Math.max(s.energy, s.connections, s.ground) * 0.35);
  alpha('.story-supernova', s.novaOpacity);
  world.querySelector('.story-supernova').style.setProperty('--nova-scale', s.novaScale);
  copies.forEach((el, i) => {
    el.style.opacity = s.copy[i];
    el.style.visibility = s.copy[i] > 0.001 ? 'visible' : 'hidden';
    el.style.transform = `translateY(${(1 - s.copy[i]) * 10}px)`;
  });
  story.querySelector('.story-disclaimer').style.opacity = s.beatProgress >= 15.5 ? 1 : 0;
  story.querySelector('.story-counter').textContent = `0${s.scene + 1} — 05 · 0${s.beat} / 05`;
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
addEventListener('load', measure, { once: true });
document.fonts.ready.then(measure);
new ResizeObserver(measure).observe(story);
measure();
