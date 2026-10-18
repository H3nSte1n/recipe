// Date-only API values are calendar dates, never UTC timestamps.
export function calendarDate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

export function parseCalendarDate(value: string): Date {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) throw new Error('Invalid calendar date');
  const date = new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]), 12);
  if (calendarDate(date) !== value) throw new Error('Invalid calendar date');
  return date;
}

export function addCalendarDays(date: Date, days: number): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() + days, 12);
}

export function monthGrid(month: Date): string[] {
  const first = new Date(month.getFullYear(), month.getMonth(), 1, 12);
  const mondayOffset = (first.getDay() + 6) % 7;
  const start = addCalendarDays(first, -mondayOffset);
  return Array.from({ length: 42 }, (_, index) => calendarDate(addCalendarDays(start, index)));
}

export function shiftMonth(month: Date, offset: number): Date {
  return new Date(month.getFullYear(), month.getMonth() + offset, 1, 12);
}

export const weekdayLetters = ['M', 'T', 'W', 'T', 'F', 'S', 'S'];

// Monday-to-Sunday dates of the week that contains the given calendar date.
export function weekOf(value: string): string[] {
  const date = parseCalendarDate(value);
  const mondayOffset = (date.getDay() + 6) % 7;
  const monday = addCalendarDays(date, -mondayOffset);
  return Array.from({ length: 7 }, (_, index) => calendarDate(addCalendarDays(monday, index)));
}

// Monday of the week that contains the given calendar date.
export function weekStart(value: string): string {
  return weekOf(value)[0];
}

// Same weekday, `offset` weeks later (negative = earlier).
export function shiftWeek(value: string, offset: number): string {
  return calendarDate(addCalendarDays(parseCalendarDate(value), offset * 7));
}

// "5–11 October 2026", "28 September – 4 October 2026" or "28 December 2026 – 3 January 2027".
export function weekRangeLabel(value: string): string {
  const days = weekOf(value);
  const first = parseCalendarDate(days[0]);
  const last = parseCalendarDate(days[6]);
  const format = (date: Date, options: Intl.DateTimeFormatOptions) => date.toLocaleDateString('en-GB', options);
  if (first.getFullYear() !== last.getFullYear()) {
    return `${format(first, { day: 'numeric', month: 'long', year: 'numeric' })} – ${format(last, { day: 'numeric', month: 'long', year: 'numeric' })}`;
  }
  if (first.getMonth() !== last.getMonth()) {
    return `${format(first, { day: 'numeric', month: 'long' })} – ${format(last, { day: 'numeric', month: 'long', year: 'numeric' })}`;
  }
  return `${first.getDate()}–${format(last, { day: 'numeric', month: 'long', year: 'numeric' })}`;
}

// Saturday and Sunday render muted in the design.
export function isWeekend(value: string): boolean {
  const day = parseCalendarDate(value).getDay();
  return day === 0 || day === 6;
}
