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

// Known cooking units for ingredient text parsing
const KNOWN_UNITS = new Set([
  'g', 'kg', 'mg', 'ml', 'l', 'cl', 'dl',
  'tsp', 'tbsp', 'cup', 'cups', 'oz', 'lb', 'lbs', 'fl',
  'pinch', 'clove', 'cloves', 'piece', 'pieces', 'slice', 'slices',
  'can', 'cans', 'bunch', 'bunches', 'handful', 'handfuls',
  'sprig', 'sprigs', 'sheet', 'sheets', 'bag', 'bags',
  'stick', 'sticks', 'drop', 'drops',
]);

const UNICODE_FRACS: Record<string, number> = {
  '½': 0.5, '¼': 0.25, '¾': 0.75, '⅓': 1 / 3, '⅔': 2 / 3,
  '⅛': 0.125, '⅜': 0.375, '⅝': 0.625, '⅞': 0.875,
};

export function parseIngText(text: string): { amount: number; unit: string; name: string } | null {
  const s = text.trim();
  let amount: number;
  let rest: string;

  // Mixed number + slash fraction: "1 1/2 cups flour"
  const mixedM = s.match(/^(\d+)\s+(\d+)\/(\d+)\s*([\s\S]*)/);
  if (mixedM) {
    amount = parseInt(mixedM[1]) + parseInt(mixedM[2]) / parseInt(mixedM[3]);
    rest = mixedM[4];
  } else {
    // Slash fraction: "1/2 cup flour"
    const slashM = s.match(/^(\d+)\/(\d+)\s*([\s\S]*)/);
    if (slashM) {
      amount = parseInt(slashM[1]) / parseInt(slashM[2]);
      rest = slashM[3];
      if (rest.startsWith('/')) return null;
    } else {
      // Unicode fraction with optional leading whole number: "1½ cups" or "½ cup"
      const unicodeM = s.match(/^(\d*)(½|¼|¾|⅓|⅔|⅛|⅜|⅝|⅞)\s*([\s\S]*)/);
      if (unicodeM) {
        amount = (unicodeM[1] ? parseInt(unicodeM[1]) : 0) + (UNICODE_FRACS[unicodeM[2]] ?? 0);
        rest = unicodeM[3];
      } else {
        // Plain integer or decimal: "20g butter", "1 tbsp salt", "1.5 kg"
        const numM = s.match(/^(\d+(?:[.,]\d+)?)\s*/);
        if (!numM) return null;
        amount = parseFloat(numM[1].replace(',', '.'));
        rest = s.slice(numM[0].length);
      }
    }
  }

  if (!amount || !isFinite(amount) || !rest) return null;

  // Single word that is a known unit with no ingredient name — not useful
  if (/^[a-zA-Z]+$/.test(rest) && KNOWN_UNITS.has(rest.toLowerCase())) return null;

  const tokenM = rest.match(/^([a-zA-Z]+)\s+([\s\S]+)$/);
  if (tokenM && KNOWN_UNITS.has(tokenM[1].toLowerCase())) {
    return { amount, unit: tokenM[1], name: tokenM[2] };
  }
  return { amount, unit: '', name: rest };
}

export function ingLine(
  amount: number | null,
  unit: string,
  name: string,
  scale: number
): string {
  let useAmount = amount;
  let useUnit = unit;
  let useName = name;

  if (useAmount === null || useAmount === 0) {
    const parsed = parseIngText(name);
    if (parsed) {
      useAmount = parsed.amount;
      useUnit = parsed.unit || unit; // prefer parsed unit, fall back to caller's
      useName = parsed.name;
    }
  }

  const scaledAmount = useAmount !== null && useAmount > 0 ? useAmount * scale : useAmount;
  const qty = fmtQty(scaledAmount);
  const parts: string[] = [];
  if (qty) parts.push(qty);
  if (useUnit) parts.push(useUnit);
  parts.push(useName);
  return parts.join(' ');
}

export function metaOf(
  prepTime: number,
  cookTime: number,
  shelfLife?: number
): string {
  const parts: string[] = [];
  const totalTime = prepTime + cookTime;
  if (totalTime > 0) parts.push(`${totalTime}min`);
  if (shelfLife && shelfLife > 0) parts.push(`${shelfLife}d shelf life`);
  return parts.join(' · ');
}
