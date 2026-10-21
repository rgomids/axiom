import { orchestrationProgress, orchestrationFraction } from './orchestration-timeline.mjs?v=e773a15fef6c';
export { orchestrationProgress };
const story = document.querySelector('.storytelling');
const architecture = document.querySelector('[data-orchestration="architecture"]');
const workflow = document.querySelector('[data-orchestration="workflow"]');
const params = new URLSearchParams(location.search);
const clamp = (n) => Math.max(0, Math.min(1, n));
const smooth = (n) => {
  n = clamp(n);
  return n * n * n * (n * (n * 6 - 15) + 10);
};
const boundaries = [0, 0.3, 0.46, 0.66, 0.84, 1];
const panels = [...workflow.querySelectorAll('[data-workflow-moment]')];
const mobilePanels = [...architecture.querySelectorAll('[data-arch-focus]')];
const archParts = [...architecture.querySelectorAll('[data-reveal]')];
let currentArchitecture = -1,
  currentWorkflow = -1;
let routingDirty = true;
function routeAgents() {
  const root = workflow.querySelector('.orch-parallel');
  const rect = root.getBoundingClientRect();
  if (!rect.width || !rect.height) return;
  const box = (selector) => {
    const b = root.querySelector(selector).getBoundingClientRect();
    return {
      x: ((b.left + b.width / 2 - rect.left) / rect.width) * 100,
      top: ((b.top - rect.top) / rect.height) * 100,
      bottom: ((b.bottom - rect.top) / rect.height) * 100,
    };
  };
  const source = box('.orch-dispatch'),
    backend = box('[data-agent="backend"]'),
    integration = box('[data-agent="integration"]'),
    doc = box('[data-agent="documentation"]'),
    tests = box('[data-agent="tests"]'),
    output = box('.orch-agent-output');
  const fork = (source.bottom + Math.min(backend.top, integration.top, doc.top)) / 2;
  root
    .querySelector('.orch-parallel-dispatch')
    .setAttribute(
      'd',
      [backend, integration, doc]
        .map((n) => `M${source.x} ${source.bottom} V${fork} H${n.x} V${n.top}`)
        .join(' '),
    );
  const join = (Math.max(backend.bottom, integration.bottom) + tests.top) / 2;
  root
    .querySelector('.orch-test-dependencies')
    .setAttribute(
      'd',
      [backend, integration]
        .map((n) => `M${n.x} ${n.bottom} V${join} H${tests.x} V${tests.top}`)
        .join(' '),
    );
  const merge = (Math.max(tests.bottom, doc.bottom) + output.top) / 2;
  root
    .querySelector('.orch-merge-dependencies')
    .setAttribute(
      'd',
      [tests, doc]
        .map((n) => `M${n.x} ${n.bottom} V${merge} H${output.x} V${output.top}`)
        .join(' '),
    );
  routingDirty = false;
}
new ResizeObserver(() => {
  routingDirty = true;
}).observe(workflow.querySelector('.orch-parallel'));
document.fonts.ready.then(() => {
  routingDirty = true;
});
function show(el, alpha, translate = true) {
  el.style.opacity = alpha;
  el.style.visibility = alpha > 0.001 ? 'visible' : 'hidden';
  if (translate) el.style.transform = `translateY(${(1 - alpha) * 9}px)`;
}
function announcePanel(el, active) {
  el.inert = !active;
  el.setAttribute('aria-hidden', String(!active));
}
function setControls(root, index) {
  root.querySelectorAll('[data-orch-go]').forEach((button, i) => {
    if (i === index) button.setAttribute('aria-current', 'step');
    else button.removeAttribute('aria-current');
  });
}
export function paintOrchestration(progress, staticMode, previewIndex) {
  if (staticMode && previewIndex === null) {
    story.style.setProperty('--orchestration-base', '1');
    story.style.setProperty('--orchestration-filter', 'none');
    for (const root of [architecture, workflow]) {
      show(root, 0, false);
      announcePanel(root, false);
    }
    return;
  }
  const beat = (progress * 17) / 4;
  let a = smooth((beat - 8.55) / 0.45) * (1 - smooth((beat - 9.82) / 0.28));
  let w = smooth((beat - 9.82) / 0.28) * (1 - smooth((beat - 10.8) / 0.2));
  let ap = clamp((beat - 9) / 0.8),
    wp = clamp((beat - 10) / 0.78);
  if (previewIndex === 9) {
    a = 1;
    w = 0;
    ap = params.has('detail') ? clamp((Number(params.get('detail')) - 1) / 4) * 0.92 : 1;
  }
  if (previewIndex === 10) {
    a = 0;
    w = 1;
    const detail = Math.min(4, Math.max(0, Number(params.get('detail') || 3) - 1));
    const fraction = params.has('sub')
      ? Math.min(0.92, Math.max(0.08, (Number(params.get('sub')) - 0.5) / 3))
      : detail === 2
        ? 0.85
        : detail === 0
          ? 0.12
          : 0.5;
    wp = boundaries[detail] + (boundaries[detail + 1] - boundaries[detail]) * fraction;
  }
  const intensity = Math.min(1, a + w);
  // Finish the outgoing caption before the new typography arrives.
  // The diagrams still crossfade, but two differently sized titles never overlap.
  let caption = beat < 9 ? 1 - smooth((beat - 8.3) / 0.25) : smooth((beat - 11) / 0.1);
  if (previewIndex !== null) caption = [9, 10].includes(previewIndex) ? 0 : 1;
  story.style.setProperty('--orchestration-base', String(1 - intensity));
  story.style.setProperty('--orchestration-filter', caption < 1 ? `opacity(${caption})` : 'none');
  show(architecture, a, false);
  show(workflow, w, false);
  // Diagrams may crossfade; their headings and controls must never compete.
  for (const [root, alpha] of [
    [architecture, a],
    [workflow, w],
  ]) {
    const textAlpha = smooth((alpha - 0.55) / 0.45);
    root
      .querySelectorAll(
        '.orch-topline,.orch-step-nav,.orch-phase-rail,.orch-scene-copy,.orch-disclosure',
      )
      .forEach((element) => {
        element.style.opacity = String(textAlpha);
      });
  }
  announcePanel(architecture, a > 0.5);
  announcePanel(workflow, w > 0.5);
  architecture.dataset.active = String(a > 0.01 && !staticMode);
  workflow.dataset.active = String(w > 0.01 && !staticMode);
  const ai = Math.min(4, Math.floor(ap * 5));
  if (ai !== currentArchitecture) {
    setControls(architecture, ai);
    currentArchitecture = ai;
  }
  for (const element of archParts) {
    const stage = Number(element.dataset.reveal);
    const reveal = stage === 0 ? 1 : smooth((ap - (stage * 0.18 - 0.04)) / 0.13);
    element.style.setProperty('--reveal', String(reveal));
  }
  mobilePanels.forEach((element, i) => {
    const x = ap * 5;
    const alpha =
      (i === 0 ? 1 : smooth((x - i + 0.18) / 0.34)) *
      (i === 4 ? 1 : 1 - smooth((x - i - 0.82) / 0.34));
    show(element, alpha);
    announcePanel(element, i === ai);
  });
  const wi =
    wp >= 1
      ? 4
      : Math.min(
          4,
          Math.max(
            0,
            boundaries.findIndex((v, i) => i < 5 && wp >= v && wp < boundaries[i + 1]),
          ),
        );
  if (wi !== currentWorkflow) {
    setControls(workflow, wi);
    currentWorkflow = wi;
  }
  panels.forEach((panel, i) => {
    const alpha =
      (i === 0 ? 1 : smooth((wp - boundaries[i] + 0.02) / 0.04)) *
      (i === 4 ? 1 : 1 - smooth((wp - boundaries[i + 1] + 0.02) / 0.04));
    show(panel, alpha);
    announcePanel(panel, wi === i);
  });
  const intakeProgress = Math.min(2.999, (wp / 0.3) * 3);
  workflow.querySelectorAll('[data-intake-column]').forEach((column, i) => {
    column.style.setProperty(
      '--column-alpha',
      String(
        smooth((intakeProgress - i + 0.15) / 0.3) * (1 - smooth((intakeProgress - i - 0.85) / 0.3)),
      ),
    );
  });
  const verifyProgress = Math.min(
    2.999,
    clamp((wp - boundaries[3]) / (boundaries[4] - boundaries[3])) * 3,
  );
  workflow.querySelectorAll('.orch-verify > section').forEach((column, i) => {
    column.style.setProperty(
      '--column-alpha',
      String(
        (i === 0 ? 1 : smooth((verifyProgress - i + 0.15) / 0.3)) *
          (i === 2 ? 1 : 1 - smooth((verifyProgress - i - 0.85) / 0.3)),
      ),
    );
  });
  workflow.querySelectorAll('[data-workflow-phase]').forEach((label, i) => {
    const active =
      wi === 0
        ? i === Math.floor(intakeProgress)
        : wi < 3
          ? i === 3
          : wi === 3
            ? i === (verifyProgress < 1 ? 3 : 4)
            : i === 5;
    label.dataset.current = String(active);
    label.dataset.complete = String(
      wi === 0 ? i < Math.floor(intakeProgress) : wi < 3 ? i < 3 : wi === 3 ? i < 4 : i < 5,
    );
  });
  const agentProgress = clamp((wp - boundaries[2]) / (boundaries[3] - boundaries[2]));
  workflow
    .querySelector('.orch-parallel')
    .style.setProperty('--tests-ready', String(smooth((agentProgress - 0.35) / 0.18)));
  workflow
    .querySelector('.orch-parallel')
    .style.setProperty('--merge-ready', String(smooth((agentProgress - 0.7) / 0.15)));
  workflow.dataset.agentPhase =
    agentProgress < 0.35 ? 'independent' : agentProgress < 0.7 ? 'tests' : 'merge';
  story.dataset.orchestrationMoment = String(w > 0.5 ? wi + 1 : a > 0.5 ? ai + 1 : 0);
  if (routingDirty) routeAgents();
}
document.querySelectorAll('[data-orch-go]').forEach((button) =>
  button.addEventListener('click', () => {
    const [type, step] = button.dataset.orchGo.split(':');
    const i = Number(step);
    if (params.has('frame')) {
      params.set('frame', type === 'architecture' ? '10' : '11');
      params.set('detail', String(i + 1));
      location.search = params.toString();
      return;
    }
    const local =
      type === 'architecture'
        ? (i + 0.4) / 5
        : i === 0
          ? 0.03
          : (boundaries[i] + boundaries[i + 1]) * 0.5;
    const beat = type === 'architecture' ? 9 + local * 0.8 : 10 + local * 0.78;
    const range = story.offsetHeight - innerHeight;
    scrollTo({
      top: story.getBoundingClientRect().top + scrollY + orchestrationFraction(beat) * range,
      behavior: matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth',
    });
  }),
);
