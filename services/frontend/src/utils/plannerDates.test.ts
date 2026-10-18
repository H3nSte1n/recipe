import { describe, expect, it } from 'vitest';
import { addDays, calendarDate, parseCalendarDate, shiftWeek, weekRangeLabel, weekStart } from './plannerDates';

describe('calendar-only dates', () => {
  it('accepts leap day and rejects invalid dates and instants', () => {
    expect(calendarDate(parseCalendarDate('2024-02-29'))).toBe('2024-02-29');
    expect(() => parseCalendarDate('2025-02-29')).toThrow();
    expect(() => parseCalendarDate('2026-10-03T00:00:00Z')).toThrow();
  });

  it('adds and subtracts days across month and year boundaries', () => {
    expect(addDays('2026-12-30', 3)).toBe('2027-01-02');
    expect(addDays('2027-03-01', -1)).toBe('2027-02-28');
    expect(addDays('2026-10-05', 0)).toBe('2026-10-05');
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
