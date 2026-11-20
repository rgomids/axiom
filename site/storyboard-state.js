import { continuityLayout } from './continuity.mjs?v=e773a15fef6c';
import { frames } from './story-frames.js?v=e773a15fef6c';
export const clamp = (n, min = 0, max = 1) => Math.max(min, Math.min(max, n));
export const ease = (n) => {
  const p = clamp(n);
  return p * p * (3 - 2 * p);
};
const mix = (a, b, p) => a + (b - a) * p;
const node = (x = 50, y = 50, w = 140, h = 80, opacity = 0, rotation = 0) => ({
  x,
  y,
  w,
  h,
  opacity,
  rotation,
  scale: 1,
});
const orbit = (n) =>
  n
    ? [
        [26, 17],
        [74, 17],
        [17, 49],
        [83, 49],
        [27, 82],
        [73, 82],
      ]
    : [
        [30, 18],
        [70, 18],
        [20, 50],
        [80, 50],
        [34, 82],
        [66, 82],
      ];
export function frameState(index, narrow = false, width = narrow ? 360 : 1000) {
  index = Math.round(clamp(index, 0, frames.length - 1));
  const layout = frames[index].layout;
  const s = {
    index,
    layout,
    narrow,
    wash: 0,
    idea: node(50, 50, narrow ? 76 : 96, narrow ? 76 : 96),
    hub: node(50, 50, narrow ? 90 : 150, narrow ? 96 : 156),
    works: orbit(narrow).map(([x, y]) => node(x, y, narrow ? 96 : 158, narrow ? 66 : 88)),
    tools: [
      [21, 13],
      [79, 13],
      [12, 43],
      [88, 43],
      [18, 73],
      [82, 73],
      [36, 91],
      [65, 91],
      [93, 88],
    ].map(([x, y]) => node(x, y, narrow ? 88 : 120, narrow ? 44 : 58)),
    ghosts: [
      [18, 15],
      [43, 8],
      [76, 15],
      [8, 48],
      [92, 46],
      [17, 80],
      [43, 94],
      [79, 84],
    ].map(([x, y], i) => node(x, y, 22 + (i % 3) * 10, 26 + (i % 3) * 10)),
    contexts: [node(82, 25, narrow ? 82 : 132, 65), node(15, 58, narrow ? 82 : 132, 65)],
    energy: 0,
    toolGrouping: 0,
    connections: 0,
    grid: 0,
    ground: 0,
    platform: 0,
    beam: 0,
    orbRing: 0,
    execution: 0,
    decision: 0,
    continuity: 0,
    continuityCard: node(72, 50, narrow ? 140 : 220, narrow ? 140 : 170),
    chips: 0,
    panel: 0,
    hubRound: 0,
    ideaCard: 0,
    flowPorts: 0,
    morph: 0,
  };
  const showWorks = (a = 1) => s.works.forEach((n) => (n.opacity = a)),
    showGhosts = (a = 0.45) => s.ghosts.forEach((n) => (n.opacity = a));
  if (layout === 'opening') s.wash = 1;
  if (
    [
      'intention',
      'idea',
      'unfold',
      'fronts',
      'tools',
      'fragmentation',
      'tension',
      'scattered',
    ].includes(layout)
  ) {
    s.idea.opacity = 1;
    s.energy = 1;
    s.orbRing = layout === 'idea' ? 1 : 0.45;
  }
  if (layout === 'unfold') showGhosts(0.65);
  // Cards keep their identity and remain visible as the field fills with work.
  if (['fronts', 'tools', 'fragmentation', 'tension', 'emerge', 'connect'].includes(layout)) {
    showWorks();
    s.energy = 1;
  }
  if (['tools', 'fragmentation', 'tension', 'emerge', 'connect'].includes(layout)) {
    s.tools.forEach((n, i) => {
      n.opacity = i < (layout === 'tools' ? 4 : layout === 'fragmentation' ? 7 : 9) ? 1 : 0;
      if (narrow) {
        const pos = [
          [16, 0],
          [84, 0],
          [50, 4],
          [15, 32],
          [85, 32],
          [15, 66],
          [85, 66],
          [16, 100],
          [84, 100],
        ][i];
        n.x = pos[0];
        n.y = pos[1];
        n.w = 90;
        n.h = 54;
      }
      n.scale = 0.78 + (i % 3) * 0.08;
      n.rotation = i % 2 ? 3 : -3;
    });
  }
  if (['fragmentation', 'tension'].includes(layout)) {
    s.works.forEach((n, i) => {
      n.x += [-5, 5, -1, 1, -5, 5][i];
      n.y += [-2, 3, -2, 2, 2, -3][i];
      n.rotation = [-4, 3, -2, 3, 2, -3][i];
      n.scale = [0.9, 1.06, 1.04, 0.94, 1.02, 0.9][i] * (layout === 'fragmentation' ? 0.96 : 1);
    });
    s.idea.w = s.idea.h = narrow ? 47 : 62;
  }
  if (['emerge', 'connect'].includes(layout)) {
    s.hub.opacity = 1;
    s.energy = 0.8;
    s.hubRound = layout === 'emerge' ? 1 : 0;
    // Organization begins only after the mascot has fully arrived.
    s.connections = layout === 'connect' ? 1 : 0;
    s.toolGrouping = 1;
    s.works.forEach((n, i) => {
      n.x = i % 2 ? (narrow ? 80 : 75) : narrow ? 20 : 25;
    });
    const owners = [0, 1, 2, 3, 0, 1, 2, 4, 5];
    const rows = [0, 0, 0, 0, 1, 1, 1, 0, 0];
    s.tools.forEach((n, i) =>
      Object.assign(
        n,
        node(
          s.works[owners[i]].x,
          s.works[owners[i]].y + (narrow ? 10 : 12) + rows[i] * (narrow ? 5 : 6),
          narrow ? 90 : 130,
          narrow ? 22 : 27,
          0.65,
        ),
      ),
    );
  }
  if (layout === 'lanes') {
    s.works.forEach((n, i) =>
      Object.assign(n, node(52, 10 + i * 19, narrow ? 288 : 510, narrow ? 62 : 72, i < 5 ? 1 : 0)),
    );
    s.connections = 1;
    s.morph = 1;
    s.flowPorts = 1;
    const spine = flowSpineX(s, width);
    s.idea = node(spine, 10, 12, 12, 1);
    s.hub = node(spine, 10, 12, 12, 0);
    s.hubRound = 1;
  }
  if (layout === 'dependencies') {
    const pos = [
      [50, 8],
      [38, 29],
      [50, 51],
      [83, 64],
      [50, 73],
      [50, 94],
    ];
    s.works.forEach((n, i) =>
      Object.assign(n, node(...pos[i], narrow ? 120 : 168, narrow ? 44 : 56, 1)),
    );
    s.contexts.forEach((n) => (n.opacity = 1));
    s.connections = 1;
    s.grid = 0.5;
  }
  if (layout === 'human-gate') {
    s.works.forEach((n) => (n.opacity = 0));
    for (const [i, y, h] of [
      [2, 10, 52],
      [3, 26, 52],
      [4, 58, narrow ? 177 : 190],
      [5, 93, 50],
    ])
      Object.assign(s.works[i], node(50, y, narrow ? 295 : 510, h, 1));
    s.connections = 0.5;
  }
  if (layout === 'execution') {
    s.execution = 1;
    s.panel = 1;
    s.works.forEach((n, i) =>
      Object.assign(n, node(50, 20 + i * 10.6, narrow ? 284 : 500, narrow ? 38 : 43, 1)),
    );
  }
  if (layout === 'convergence') {
    s.works.forEach((n, i) =>
      Object.assign(
        n,
        node(
          i < 5 ? 22 : 73,
          i < 5 ? 14 + i * 18 : 50,
          i < 5 ? (narrow ? 132 : 225) : narrow ? 104 : 170,
          i < 5 ? (narrow ? 48 : 58) : narrow ? 116 : 180,
          1,
        ),
      ),
    );
    s.connections = 1;
    s.energy = 0.7;
    showGhosts(0.15);
  }
  if (layout === 'evidence')
    s.works.forEach((n, i) =>
      Object.assign(n, node(50, 11 + i * 19, narrow ? 300 : 510, narrow ? 65 : 76, i < 5 ? 1 : 0)),
    );
  if (layout === 'decision') s.decision = 1;
  if (layout === 'delivered') {
    s.works[5] = node(50, 45, narrow ? 168 : 240, narrow ? 155 : 210, 1);
    s.ground = 1;
    s.chips = 1;
    showGhosts(0.3);
    s.beam = 0.4;
  }
  if (layout === 'continuity') {
    const pair = continuityLayout(width);
    s.idea = node(pair.left, 50, pair.size, pair.size, 1);
    s.continuityCard = node(pair.right, 50, pair.size, pair.size, 1);
    s.ideaCard = 1;
    s.continuity = 1;
    s.connections = 0.6;
  }
  return s;
}
const interpolate = (a, b, t) => {
  if (typeof a === 'number') return mix(a, b, t);
  if (Array.isArray(a)) return a.map((v, i) => interpolate(v, b[i], t));
  if (a && typeof a === 'object')
    return Object.fromEntries(Object.keys(a).map((k) => [k, interpolate(a[k], b[k], t)]));
  return t < 0.5 ? a : b;
};
export function storyState(progress, narrow = false, width = narrow ? 360 : 1000) {
  const p = clamp(progress, 0, 4),
    q = (p * (frames.length - 1)) / 4,
    from = Math.min(frames.length - 2, Math.floor(q)),
    // Hold the complete composition before a long, continuous transition.
    travel = clamp((q - from - 0.28) / 0.72),
    t = travel * travel * travel * (travel * (travel * 6 - 15) + 10),
    s = interpolate(frameState(from, narrow, width), frameState(from + 1, narrow, width), t),
    index = from + (t >= 0.5 ? 1 : 0);
  s.progress = p;
  s.beatProgress = q;
  s.from = from;
  s.transition = t;
  s.narrow = narrow;
  s.width = width;
  s.index = index;
  s.scene = frames[index].scene - 1;
  s.beat = frames[index].beat;
  s.contentOpacity = Math.abs(1 - 2 * t);
  s.copy = frames.map((_, i) =>
    i === from ? clamp(1 - 2 * t) : i === from + 1 ? clamp(2 * t - 1) : 0,
  );
  // A traveling impulse arrives before the card opens; stagger all six fronts.
  s.reveals = s.works.map((_, i) => {
    const arrival = 0.5 + i * 0.018;
    const growth = ease((travel - arrival) / 0.41);
    return {
      line: ease((travel - 0.1 - i * 0.018) / 0.38),
      card: growth,
      bloom: Math.sin(Math.PI * growth),
    };
  });
  s.novaOpacity = ease((q - 0.54) / 0.15) * (1 - ease((q - 0.9) / 0.35));
  s.novaScale = 0.08 + ease((q - 0.54) / 0.65) * 3.1;
  if (q < 1) {
    s.wash = 1 - ease((q - 0.08) / 0.42);
    s.idea.opacity = ease((q - 0.72) / 0.28);
  }
  if (frames[from].layout === 'connect') {
    // Hide the relocation, then reveal the nucleus already attached to its port.
    s.idea.opacity = ease((t - 0.62) / 0.38);
    s.hub.opacity = 1 - ease(t / 0.65);
    if (t >= 0.5) {
      s.idea.x = flowSpineX(s, width);
      s.idea.y = s.works[0].y;
    }
  }
  s.routingFade = 1 - ease(t / 0.5) + ease((t - 0.5) / 0.5);
  s.flowPorts =
    frameState(from, narrow, width).flowPorts * (1 - ease(t / 0.5)) +
    frameState(from + 1, narrow, width).flowPorts * ease((t - 0.5) / 0.5);
  return s;
}

export function flowSpineX(s, width) {
  const first = s.works[0];
  return Math.max(0, first.x - ((first.w / 2 + (s.narrow ? 12 : 28)) / width) * 100);
}

// Orthogonal geometry: all routing changes while invisible, never bends in flight.
function relationCurve(s, i, width, height) {
  const n = s.works[i];
  let from = ['emerge', 'connect'].includes(s.layout) ? [s.hub.x, s.hub.y] : [s.idea.x, s.idea.y],
    to = [n.x, n.y],
    opacity = s.connections * n.opacity;
  if (s.layout === 'convergence') {
    from = [n.x, n.y];
    to = [s.works[5].x, s.works[5].y];
    if (i === 5) opacity = 0;
  }
  if (s.layout === 'lanes') {
    from = [flowSpineX(s, width), i ? s.works[i - 1].y : s.works[0].y];
    to = [flowSpineX(s, width), n.y];
    if (i === 5) opacity = 0;
  }
  if (s.layout === 'dependencies') {
    const prev = { 1: 0, 2: 1, 4: 2, 5: 4, 3: 2 }[i];
    if (prev === undefined) opacity = 0;
    else from = [s.works[prev].x, s.works[prev].y];
    to =
      i === 3
        ? [n.x - (n.w / width) * 50 - 1, n.y]
        : [n.x - (i === 2 ? (n.w / width) * 25 : 0), n.y - (n.h / height) * 50 - 1];
  }
  if (s.layout === 'human-gate') {
    const review = s.works[4];
    from = [review.x - (review.w / width) * 50, review.y];
    to = [n.x - (n.w / width) * 50, n.y];
    opacity = i === 5 ? 0.6 : 0;
  }
  if (s.layout === 'continuity') {
    from = [s.idea.x + (s.idea.w / width) * 50, s.idea.y];
    to = [s.continuityCard.x - (s.continuityCard.w / width) * 50, s.continuityCard.y];
    opacity = i === 0 ? 0.7 : 0;
  }
  const radial = ['emerge', 'connect'].includes(s.layout);
  const left = n.x < s.hub.x;
  const bus = s.hub.x + (left ? -1 : 1) * ((s.hub.w / 2 + (s.narrow ? 5 : 14)) / width) * 100;
  const d = radial
    ? `M ${s.hub.x} ${s.hub.y} L ${bus} ${s.hub.y} L ${bus} ${n.y} L ${n.x + (left ? 1 : -1) * (n.w / width) * 50} ${n.y}`
    : s.layout === 'human-gate'
      ? `M ${from[0]} ${from[1]} L ${from[0] - 4} ${from[1]} L ${from[0] - 4} ${to[1]} L ${to[0]} ${to[1]}`
      : ['dependencies', 'lanes'].includes(s.layout)
        ? dependencyPath(from, to, i === 3)
        : connectionPath(from, to);
  return { points: d.match(/-?\d+(?:\.\d+)?(?:e[+-]?\d+)?/g).map(Number), opacity };
}
export const curvePath = (v) =>
  `M ${v[0]} ${v[1]} L ${v[2]} ${v[3]} L ${v[4]} ${v[5]} L ${v[6]} ${v[7]}`;
export function frameConnections(s, width, height) {
  return s.works.map((_, i) => {
    const c = relationCurve(s, i, width, height);
    return { d: curvePath(c.points), opacity: c.opacity };
  });
}
// Reveal a polyline by traveled length, including degenerate straight segments.
export function curvePrefix(v, progress) {
  const vertices = [
    [v[0], v[1]],
    [v[2], v[3]],
    [v[4], v[5]],
    [v[6], v[7]],
  ];
  const lengths = vertices
    .slice(1)
    .map((n, i) => Math.hypot(n[0] - vertices[i][0], n[1] - vertices[i][1]));
  let remaining = lengths.reduce((a, b) => a + b, 0) * clamp(progress);
  let end = vertices[0];
  let d = `M ${end[0]} ${end[1]}`;
  for (let i = 0; i < lengths.length; i++) {
    const t = lengths[i] ? clamp(remaining / lengths[i]) : 1;
    end = vertices[i].map((n, axis) => mix(n, vertices[i + 1][axis], t));
    d += ` L ${end[0]} ${end[1]}`;
    remaining -= lengths[i];
    if (remaining <= 0 && lengths[i]) break;
  }
  return { d, end };
}
export function storyConnections(s, width, height) {
  const a = frameState(s.from, s.narrow, width),
    b = frameState(s.from + 1, s.narrow, width);
  return s.works.map((_, i) => {
    const before = relationCurve(a, i, width, height),
      after = relationCurve(b, i, width, height),
      points = relationCurve(s, i, width, height).points,
      revealing = before.opacity < 0.05 && after.opacity > 0.1,
      prefix = curvePrefix(points, revealing ? s.reveals[i].line : 1),
      impulse = curvePrefix(points, s.reveals[i].line);
    return {
      ...prefix,
      opacity:
        relationCurve(s, i, width, height).opacity *
        (a.layout !== b.layout &&
        ['lanes', 'dependencies', 'human-gate', 'convergence', 'continuity'].some((layout) =>
          [a.layout, b.layout].includes(layout),
        )
          ? s.routingFade
          : 1),
      signalEnd: impulse.end,
      trail: impulse.d,
      signal: Math.sin(Math.PI * s.reveals[i].line) * (after.opacity > 0.1 ? 1 : 0),
    };
  });
}
export function connectionPath(from, to) {
  const [x, y] = from,
    [ex, ey] = to,
    mid = (x + ex) / 2;
  return `M ${x} ${y} L ${mid} ${y} L ${mid} ${ey} L ${ex} ${ey}`;
}
export function dependencyPath(from, to, side = false) {
  if (side) return connectionPath(from, to);
  const [x, y] = from,
    [ex, ey] = to,
    mid = (y + ey) / 2;
  return `M ${x} ${y} L ${x} ${mid} L ${ex} ${mid} L ${ex} ${ey}`;
}

export function dependencySideConnections(s, width, height) {
  const implementation = s.works[2],
    specification = s.works[0];
  return [
    ...s.contexts.map((n) =>
      connectionPath(
        [n.x + (n.x < implementation.x ? 1 : -1) * (n.w / width) * 50, n.y],
        [
          implementation.x +
            (n.x < implementation.x ? -1 : 1) * ((implementation.w / width) * 50 + 1),
          implementation.y,
        ],
      ),
    ),
    dependencyPath(
      [
        specification.x + (specification.w / width) * 25,
        specification.y + (specification.h / height) * 50,
      ],
      [
        implementation.x + (implementation.w / width) * 25,
        implementation.y - (implementation.h / height) * 50 - 1,
      ],
    ),
  ];
}
