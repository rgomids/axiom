import { frames } from './story-frames.js';
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
  index = Math.round(clamp(index, 0, 24));
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
    contexts: [node(82, 28, narrow ? 92 : 132, 65), node(17, 52, narrow ? 92 : 132, 65)],
    energy: 0,
    connections: 0,
    grid: 0,
    ground: 0,
    platform: 0,
    beam: 0,
    orbRing: 0,
    execution: 0,
    decision: 0,
    continuity: 0,
    chips: 0,
    panel: 0,
    hubRound: 0,
    ideaCard: 0,
    ideaLink: 0,
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
  if (['fronts', 'scattered'].includes(layout)) {
    showWorks();
    showGhosts();
    s.connections = 1;
  }
  if (layout === 'tools') {
    showWorks(0.12);
    showGhosts();
    s.tools.forEach((n) => (n.opacity = 1));
    s.connections = 0.6;
  }
  if (['fragmentation', 'tension'].includes(layout)) {
    showWorks();
    showGhosts(0.6);
    s.connections = 0.65;
    s.tools.forEach((n, i) => {
      n.opacity = layout === 'tension' ? 0.4 : 0.17;
      n.rotation = i % 2 ? 5 : -5;
    });
    s.works.forEach((n, i) => {
      n.x += [-4, 5, -2, 2, -3, 5][i];
      n.rotation = [-6, 5, -4, 6, 3, -5][i];
      n.w = narrow ? 133 : 190;
      n.h = narrow ? 80 : 108;
    });
    s.idea.w = s.idea.h = narrow ? 47 : 62;
  }
  if (['emerge', 'connect', 'unified', 'platform', 'map'].includes(layout)) {
    showWorks();
    showGhosts(0.35);
    s.connections = 1;
    s.hub.opacity = 1;
    s.energy = 0.65;
    if (layout === 'emerge') {
      s.hubRound = 1;
      s.idea.opacity = 0.1;
      s.orbRing = 1;
    }
    if (['unified', 'platform', 'map'].includes(layout)) s.grid = 1;
    if (layout === 'platform') {
      s.platform = 1;
      s.ground = 1;
      s.beam = 1;
      s.hub.y = 44;
    }
    if (layout === 'map') {
      s.ideaLink = 1;
      s.idea = node(10, 50, narrow ? 36 : 50, narrow ? 36 : 50, 1);
      s.hub.x = 45;
      const mapPositions = narrow
        ? [
            [35, 14],
            [73, 14],
            [83, 40],
            [83, 69],
            [25, 80],
            [62, 89],
          ]
        : [
            [40, 15],
            [70, 15],
            [83, 42],
            [83, 70],
            [28, 80],
            [63, 86],
          ];
      s.works.forEach((n, i) => {
        n.x = mapPositions[i][0];
        n.y = mapPositions[i][1];
      });
    }
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
      [42, 29],
      [50, 51],
      [83, 64],
      [50, 73],
      [50, 94],
    ];
    s.works.forEach((n, i) =>
      Object.assign(n, node(...pos[i], narrow ? 105 : 168, narrow ? 44 : 56, 1)),
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
    s.idea = node(23, 50, narrow ? 90 : 120, narrow ? 94 : 130, 1);
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
    q = p * 6,
    from = Math.min(23, Math.floor(q)),
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
  if (from === 15) {
    // Hide the relocation, then reveal the nucleus already attached to its port.
    s.idea.opacity = 1 - ease(t / 0.45) + ease((t - 0.62) / 0.38);
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

// Cubic geometry stays compatible across radial, linear and dependency layouts.
function relationCurve(s, i, width, height) {
  const n = s.works[i];
  let from = s.hub.opacity > 0.3 ? [s.hub.x, s.hub.y] : [s.idea.x, s.idea.y],
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
    to = i === 3 ? [n.x - (n.w / width) * 50 - 1, n.y] : [n.x, n.y - (n.h / height) * 50 - 1];
  }
  if (s.layout === 'human-gate') {
    const review = s.works[4];
    from = [review.x - (review.w / width) * 50, review.y];
    to = [n.x - (n.w / width) * 50, n.y];
    opacity = i === 5 ? 0.6 : 0;
  }
  if (s.layout === 'continuity') {
    from = [s.idea.x, s.idea.y];
    to = [72, 50];
    opacity = i === 0 ? 0.7 : 0;
  }
  const d =
    s.layout === 'human-gate'
      ? `M ${from[0]} ${from[1]} C ${from[0] - 4} ${from[1]}, ${to[0] - 4} ${to[1]}, ${to[0]} ${to[1]}`
      : ['dependencies', 'lanes'].includes(s.layout)
        ? dependencyPath(from, to, i === 3)
        : connectionPath(from, to, i, s.grid);
  return { points: d.match(/-?\d+(?:\.\d+)?(?:e[+-]?\d+)?/g).map(Number), opacity };
}
export const curvePath = (v) =>
  `M ${v[0]} ${v[1]} C ${v[2]} ${v[3]}, ${v[4]} ${v[5]}, ${v[6]} ${v[7]}`;
export function frameConnections(s, width, height) {
  return s.works.map((_, i) => {
    const c = relationCurve(s, i, width, height);
    return { d: curvePath(c.points), opacity: c.opacity };
  });
}
export function originConnection(s, width) {
  return {
    d: connectionPath(
      [s.idea.x + (s.idea.w / width) * 50, s.idea.y],
      [s.hub.x - (s.hub.w / width) * 50, s.hub.y],
      2,
      0.5,
    ),
    opacity: s.ideaLink * Math.min(s.idea.opacity, s.hub.opacity),
  };
}
export function curvePrefix(v, progress) {
  const p = clamp(progress),
    a = [v[0], v[1]],
    b = [v[2], v[3]],
    c = [v[4], v[5]],
    d = [v[6], v[7]],
    point = (u, w) => u.map((n, i) => mix(n, w[i], p)),
    ab = point(a, b),
    bc = point(b, c),
    cd = point(c, d),
    abc = point(ab, bc),
    bcd = point(bc, cd),
    end = point(abc, bcd);
  return { d: curvePath([...a, ...ab, ...abc, ...end]), end };
}
export function storyConnections(s, width, height) {
  const a = frameState(s.from, s.narrow, width),
    b = frameState(s.from + 1, s.narrow, width);
  return s.works.map((_, i) => {
    const before = relationCurve(a, i, width, height),
      after = relationCurve(b, i, width, height),
      points = before.points.map((n, j) => mix(n, after.points[j], s.transition)),
      revealing = before.opacity < 0.05 && after.opacity > 0.1,
      prefix = curvePrefix(points, revealing ? s.reveals[i].line : 1),
      impulse = curvePrefix(points, s.reveals[i].line);
    return {
      ...prefix,
      opacity:
        mix(before.opacity, after.opacity, s.transition) *
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
export function connectionPath(from, to, index = 0, coordination = 1) {
  const [x, y] = from,
    [ex, ey] = to,
    dx = ex - x,
    dy = ey - y,
    bend = (index % 2 ? 1 : -1) * (8 + (1 - coordination) * 8);
  if (index === 0 || index === 5) {
    const length = Math.hypot(dx, dy) || 1,
      nx = -dy / length,
      ny = dx / length;
    return `M ${x} ${y} C ${x + dx * 0.3 + nx * bend} ${y + dy * 0.3 + ny * bend}, ${x + dx * 0.7 + nx * bend} ${y + dy * 0.7 + ny * bend}, ${ex} ${ey}`;
  }
  return `M ${x} ${y} C ${x + dx * 0.35} ${y - dy * 0.2 + bend}, ${ex - dx * 0.3} ${ey + dy * 0.2 - bend}, ${ex} ${ey}`;
}
export function dependencyPath(from, to, side = false) {
  const [x, y] = from,
    [ex, ey] = to;
  return side
    ? `M ${x} ${y} C ${x + (ex - x) * 0.5} ${y}, ${ex - (ex - x) * 0.2} ${ey}, ${ex} ${ey}`
    : `M ${x} ${y} C ${x} ${y + (ey - y) * 0.5}, ${ex} ${ey - (ey - y) * 0.5}, ${ex} ${ey}`;
}
