// A centered pair with equal squares and a modest, responsive gap.
export function continuityLayout(width) {
  const size = width >= 900 ? 240 : width >= 600 ? 200 : Math.min(156, (width - 30) / 2);
  const gap = Math.max(24, Math.min(64, width * 0.05));
  const offset = ((size + gap) / 2 / width) * 100;
  return { size, left: 50 - offset, right: 50 + offset };
}
