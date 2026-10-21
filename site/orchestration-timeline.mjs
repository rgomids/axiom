// Only the two redesigned intervals receive more reading time.
export const intervals = Array.from({ length: 17 }, (_, i) => (i === 9 ? 4 : i === 10 ? 9 : 1));
export const offsets = [0];
for (const duration of intervals) offsets.push(offsets.at(-1) + duration);
export const totalUnits = offsets.at(-1);
const clamp = (n) => Math.max(0, Math.min(1, n));
export function orchestrationProgress(fraction) {
  const position = clamp(fraction) * totalUnits;
  const index = intervals.findIndex((duration, i) => position < offsets[i] + duration);
  const beat = index < 0 ? 17 : index + (position - offsets[index]) / intervals[index];
  return (beat * 4) / 17;
}
export function orchestrationFraction(beat) {
  const index = Math.max(0, Math.min(16, Math.floor(beat)));
  return (offsets[index] + (beat - index) * intervals[index]) / totalUnits;
}
