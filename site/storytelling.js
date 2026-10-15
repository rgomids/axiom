import { storyState, connectionPath, clamp } from './story-state.js';

const story = document.querySelector('.storytelling');
const idea = story.querySelector('[data-story-idea]');
const nova = story.querySelector('.story-supernova');
const hub = story.querySelector('[data-story-axiom]');
const works = [...story.querySelectorAll('[data-story-work]')];
const paths = [...story.querySelectorAll('[data-story-path]')];
const copies = [...story.querySelectorAll('[data-story-copy]')];
const steps = [...story.querySelectorAll('.story-navigation a')];
const context = story.querySelector('.story-context');
const roles = story.querySelector('.story-roles');
const decision = story.querySelector('[data-story-decision]');
const origin = story.querySelector('.story-origin-link');
const fixedPreview = new URLSearchParams(location.search).get('scene');
const previewScene = /^[1-5]$/.test(fixedPreview ?? '') ? Number(fixedPreview) - 1 : null;
let frame = 0,
  top = 0,
  range = 1,
  width = 1,
  height = 1;

function measure() {
  top = story.getBoundingClientRect().top + scrollY;
  const world = story.querySelector('.story-world').getBoundingClientRect();
  width = world.width;
  height = world.height;
  range = Math.max(1, story.offsetHeight - innerHeight);
  schedule();
}
function place(element, position) {
  element.style.transform = `translate3d(${(position[0] * width) / 100}px,${(position[1] * height) / 100}px,0) translate(-50%,-50%)`;
}
function paint() {
  frame = 0;
  const staticMode = document.body.classList.contains('static-motion');
  document.body.classList.toggle('story-motion', !staticMode || previewScene !== null);
  const p = previewScene ?? clamp((scrollY - top) / range) * 4;
  const state = storyState(p, innerWidth <= 760);
  story.dataset.storyProgress = p.toFixed(6);
  story.dataset.storyScene = String(state.scene + 1);
  story.style.setProperty('--story-wash', state.wash);
  story.style.setProperty('--story-ambient', state.ambient);
  document.body.classList.toggle(
    'gold-start',
    (!staticMode || previewScene !== null) && state.wash > 0.65 && scrollY < top + innerHeight,
  );
  if (staticMode && previewScene === null) return;
  place(nova, innerWidth <= 760 ? [50, 46] : [50, 52]);
  nova.style.opacity = state.novaOpacity;
  nova.style.setProperty('--nova-scale', state.novaScale);
  nova.style.setProperty('--nova-halo', state.novaHalo);
  idea.style.opacity = state.ideaOpacity;
  idea.style.width = state.ideaSize + 'px';
  idea.style.height = state.ideaSize + 'px';
  place(idea, state.idea);
  idea.querySelector('.idea-label').style.opacity = 1 - state.delivery;
  place(hub, state.hub);
  hub.style.opacity = state.hubOpacity;
  hub.style.transform += ` scale(${state.hubScale})`;
  const proofLabels = document.documentElement.lang.startsWith('en')
    ? [
        'Scope / criteria',
        'ADRs / context',
        'Code / pull request',
        'Tests / review',
        'Diff / Evidence',
        'Human decision',
      ]
    : [
        'Escopo / critérios',
        'ADRs / contexto',
        'Código / pull request',
        'Testes / revisão',
        'Diff / Evidence',
        'Decisão humana',
      ];
  works.forEach((work, i) => {
    const node = state.works[i];
    place(work, [node.x, node.y]);
    work.style.opacity = node.opacity;
    work.style.transform += ` rotate(${node.rotation}deg)`;
    work.style.setProperty('--work-focus', node.focus);
    work.querySelector('.work-proof').textContent = proofLabels[i];
    work.querySelector('.work-proof').style.opacity = clamp((state.proof - 0.5) * 2);
    work.querySelector('.work-detail').style.opacity = clamp(1 - state.proof * 2);
    const destination = state.hub.map(
      (value, axis) => value + (state.idea[axis] - value) * state.delivery,
    );
    const from = state.idea.map(
      (value, axis) => value + (destination[axis] - value) * state.coordination,
    );
    const fragmentLength = 0.32 + i * 0.055;
    const coverage = fragmentLength + (1 - fragmentLength) * state.coordination;
    const end = [from[0] + (node.x - from[0]) * coverage, from[1] + (node.y - from[1]) * coverage];
    paths[i].setAttribute('d', connectionPath(from, end, i, state.coordination));
    paths[i].style.opacity = state.links * node.opacity * (0.35 + state.coordination * 0.55);
    paths[i].style.strokeDasharray = 'none';
    paths[i].style.setProperty('--work-focus', node.focus);
  });
  copies.forEach((copy, i) => {
    const alpha =
      i === 0
        ? state.openingCopy
        : clamp((0.5 - Math.abs(p - i)) / 0.15) * (i === 1 ? state.complexityCopy : 1);
    copy.style.opacity = alpha;
    copy.style.visibility = alpha > 0 ? 'visible' : 'hidden';
    copy.style.transform = `translateY(${(i - p) * 12}px)`;
  });
  context.style.opacity = state.context;
  roles.style.opacity = state.roles;
  decision.style.opacity = state.gate;
  origin.style.opacity = state.gate;
  origin.setAttribute('d', connectionPath(state.idea, innerWidth <= 760 ? [50, 84] : [77, 65]));
  steps.forEach((step, i) => {
    if (i === state.scene) step.setAttribute('aria-current', 'step');
    else step.removeAttribute('aria-current');
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
