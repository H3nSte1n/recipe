import { useCallback, useMemo, useState } from 'react';
import { addDays, shiftWeek, weekStart } from '../utils/plannerDates';

const WEEKS_BEFORE_TODAY = 2;
const INITIAL_WEEKS = 7;
const WEEKS_PER_EXTENSION = 4;

// The run of whole weeks shown in the mobile day list. It starts around today and only ever grows.
export function usePlannerWindow(today: string) {
  const [range, setRange] = useState(() => ({ start: shiftWeek(weekStart(today), -WEEKS_BEFORE_TODAY), weeks: INITIAL_WEEKS }));
  const from = range.start;
  const to = addDays(range.start, range.weeks * 7 - 1);
  const dates = useMemo(() => Array.from({ length: range.weeks * 7 }, (_, index) => addDays(range.start, index)), [range]);

  const extendTop = useCallback(() => setRange((current) => ({ start: shiftWeek(current.start, -WEEKS_PER_EXTENSION), weeks: current.weeks + WEEKS_PER_EXTENSION })), []);
  const extendBottom = useCallback(() => setRange((current) => ({ ...current, weeks: current.weeks + WEEKS_PER_EXTENSION })), []);

  return { from, to, dates, extendTop, extendBottom };
}
