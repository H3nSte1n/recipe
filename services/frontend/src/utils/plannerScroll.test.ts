import { describe, expect, it } from 'vitest';
import { topmostDate } from './plannerScroll';

describe('topmostDate', () => {
  const sections = [
    { date: '2026-10-12', bottom: -40 },
    { date: '2026-10-13', bottom: 30 },
    { date: '2026-10-14', bottom: 300 },
  ];

  it('skips days that have scrolled fully under the sticky header', () => {
    expect(topmostDate(sections, 0)).toBe('2026-10-13');
    expect(topmostDate(sections, 100)).toBe('2026-10-14');
  });

  it('returns null when nothing reaches below the header', () => {
    expect(topmostDate(sections, 500)).toBeNull();
    expect(topmostDate([], 0)).toBeNull();
  });
});
