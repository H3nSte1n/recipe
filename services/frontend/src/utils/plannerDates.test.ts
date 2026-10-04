import { describe, expect, it } from 'vitest';
import { calendarDate, monthGrid, parseCalendarDate, shiftMonth } from './plannerDates';

describe('calendar-only dates', () => {
  it('accepts leap day and rejects invalid dates and instants', () => {
    expect(calendarDate(parseCalendarDate('2024-02-29'))).toBe('2024-02-29');
    expect(() => parseCalendarDate('2025-02-29')).toThrow();
    expect(() => parseCalendarDate('2026-10-03T00:00:00Z')).toThrow();
  });

  it('builds a Monday-first 42-day grid across year boundaries', () => {
    const dates = monthGrid(parseCalendarDate('2027-01-01'));
    expect(dates).toHaveLength(42);
    expect(dates[0]).toBe('2026-12-28');
    expect(dates[41]).toBe('2027-02-07');
    expect(calendarDate(shiftMonth(parseCalendarDate('2027-01-31'), -1))).toBe('2026-12-01');
  });
});
