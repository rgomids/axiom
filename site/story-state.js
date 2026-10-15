/* Scroll is the only clock. Coordinates are percentages of the shared stage. */
export const clamp = (value, min = 0, max = 1) => Math.max(min, Math.min(max, value));
const ease = (value) => {
  const p = clamp(value);
  return p * p * (3 - 2 * p);
};
const mix = (a, b, p) => a + (b - a) * ease(p);
const desktop = [
  [
    [50, 52],
    [50, 52],
    [50, 52],
    [50, 52],
    [50, 52],
    [50, 52],
  ],
  [
    [17, 16],
    [78, 12],
    [12, 52],
    [86, 49],
    [27, 85],
    [73, 83],
  ],
  [
    [22, 17],
    [20, 50],
    [24, 83],
    [78, 17],
    [80, 50],
    [76, 83],
  ],
  [
    [21, 16],
    [19, 50],
    [23, 84],
    [79, 16],
    [81, 50],
    [77, 84],
  ],
  [
    [16, 22],
    [40, 22],
    [16, 51],
    [40, 51],
    [16, 80],
    [40, 80],
  ],
];
const mobile = [
  [
    [50, 46],
    [50, 46],
    [50, 46],
    [50, 46],
    [50, 46],
    [50, 46],
  ],
  [
    [26, 24],
    [76, 17],
    [24, 49],
    [77, 52],
    [26, 79],
    [74, 83],
  ],
  [
    [25, 39],
    [75, 39],
    [25, 64],
    [75, 64],
    [25, 88],
    [75, 88],
  ],
  [
    [25, 40],
    [75, 40],
    [25, 64],
    [75, 64],
    [25, 88],
    [75, 88],
  ],
  [
    [25, 13],
    [75, 13],
    [25, 32],
    [75, 32],
    [25, 51],
    [75, 51],
  ],
];

export function storyState(progress, narrow = false) {
  const p = clamp(progress, 0, 4),
    part = Math.min(3, Math.floor(p)),
    t = p - part;
  const points = narrow ? mobile : desktop;
  const coordination = ease((p - 1.2) / 0.8),
    delivery = ease(p - 3);
  const ideaStart = narrow ? [50, 46] : [50, 52];
  const ideaOrganized = narrow ? [50, 3] : [50, 10];
  const ideaEnd = narrow ? [50, 68] : [77, 30];
  const idea = ideaStart.map((value, axis) =>
    mix(mix(value, ideaOrganized[axis], coordination), ideaEnd[axis], delivery),
  );
  const hub = narrow ? [50, mix(19, 66, delivery)] : [mix(50, 77, delivery), mix(50, 15, delivery)];
  const works = points[part].map((point, i) => ({
    x: mix(point[0], points[part + 1][i][0], t),
    y: mix(point[1], points[part + 1][i][1], t),
    opacity: ease((p - 0.48 - i * 0.065) / 0.22),
    rotation: mix([-5, 4, -3, 5, 3, -4][i] * ease((p - 0.5) * 2), 0, coordination),
    focus: p >= 2 && p < 3 ? Math.max(0, 1 - Math.abs((p - 2) * 6 - i - 0.35)) : 0,
  }));
  return {
    progress: p,
    scene: Math.min(4, Math.round(p)),
    wash: 1 - ease((p - 0.12) / 0.55),
    idea,
    ideaOpacity: ease((p - 0.08) / 0.3),
    ideaSize: mix(74, 26, coordination),
    hub,
    hubOpacity: ease((p - 1.25) / 0.45) * (1 - delivery),
    hubScale: mix(1, 0.48, delivery),
    works,
    coordination,
    delivery,
    links: ease((p - 0.55) / 0.35),
    proof: ease((p - 3.1) / 0.6),
    context: ease((p - 0.7) / 0.3) * (1 - ease((p - 1.75) / 0.3)),
    roles: ease((p - 2.1) / 0.3) * (1 - ease((p - 3) / 0.25)),
    gate: ease((p - 3.55) / 0.35),
  };
}

export function connectionPath(from, to, index = 0, coordinated = 1) {
  const [x, y] = from,
    [endX, endY] = to;
  const dx = endX - x,
    dy = endY - y;
  const bend = (index % 2 ? 1 : -1) * (8 + (1 - coordinated) * 4);
  return `M ${x} ${y} C ${x + dx * 0.4} ${y - dy * 0.25 + bend}, ${endX - dx * 0.3} ${endY + dy * 0.25 - bend}, ${endX} ${endY}`;
}
