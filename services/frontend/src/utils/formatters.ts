const FRACTIONS: Array<[number, string]> = [
  [1 / 4, '¼'],
  [1 / 3, '⅓'],
  [1 / 2, '½'],
  [2 / 3, '⅔'],
  [3 / 4, '¾'],
];

export function fmtQty(n: number | null): string {
  if (n === null || n === 0) return '';
  const whole = Math.floor(n);
  const remainder = n - whole;
  if (remainder === 0) return String(whole);

  let closest = FRACTIONS[0];
  let minDiff = Math.abs(remainder - FRACTIONS[0][0]);
  for (const frac of FRACTIONS) {
    const diff = Math.abs(remainder - frac[0]);
    if (diff < minDiff) {
      minDiff = diff;
      closest = frac;
    }
  }

  return whole > 0 ? `${whole}${closest[1]}` : closest[1];
}

export function ingLine(
  amount: number | null,
  unit: string,
  name: string,
  scale: number
): string {
  const scaledAmount = amount !== null ? amount * scale : null;
  const qty = fmtQty(scaledAmount);
  const parts: string[] = [];
  if (qty) parts.push(qty);
  if (unit) parts.push(unit);
  parts.push(name);
  return parts.join(' ');
}

export function metaOf(
  prepTime: number,
  cookTime: number,
  servings: number
): string {
  const parts: string[] = [];
  const totalTime = prepTime + cookTime;
  if (totalTime > 0) parts.push(`${totalTime}min`);
  if (servings > 0) parts.push(`${servings} servings`);
  return parts.join(' · ');
}
