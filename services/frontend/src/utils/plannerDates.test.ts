import { describe, expect, it } from 'vitest';
import { calendarDate, monthGrid, parseCalendarDate, shiftMonth, shiftWeek, weekRangeLabel, weekStart } from './plannerDates';

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

describe('week helpers', () => {
  it('finds the Monday of any weekday, including a Sunday', () => {
    expect(weekStart('2026-10-07')).toBe('2026-10-05');
    expect(weekStart('2026-10-11')).toBe('2026-10-05');
    expect(weekStart('2026-10-05')).toBe('2026-10-05');
  });

  it('shifts by whole weeks across month and year boundaries', () => {
    expect(shiftWeek('2026-10-07', 1)).toBe('2026-10-14');
    expect(shiftWeek('2026-10-07', -1)).toBe('2026-09-30');
    expect(shiftWeek('2026-12-30', 1)).toBe('2027-01-06');
  });

  it('labels the range within a month, across months and across years', () => {
    expect(weekRangeLabel('2026-10-07')).toBe('5–11 October 2026');
    expect(weekRangeLabel('2026-09-30')).toBe('28 September – 4 October 2026');
    expect(weekRangeLabel('2026-12-30')).toBe('28 December 2026 – 3 January 2027');
  });
});
