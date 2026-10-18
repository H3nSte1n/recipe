import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { PlannedMeal } from '../types/plannedMeal';
import { listAllPlannedMeals } from '../services/plannedMealService';
import { addDays, weekStarts } from '../utils/plannerDates';

// Loads planned meals for whole Monday–Sunday weeks (from = a Monday, to = a Sunday).
// Weeks already loaded are kept, so moving around only fetches what is new.
export function usePlannedMeals(from: string, to: string) {
  const [weeks, setWeeks] = useState<Record<string, PlannedMeal[]>>({});
  const [error, setError] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);
  const weeksRef = useRef(weeks);
  weeksRef.current = weeks;
  const inFlight = useRef(new Set<string>());
  const revisionRef = useRef(revision);
  revisionRef.current = revision;

  useEffect(() => {
    const missing = weekStarts(from, to).filter((monday) => !(monday in weeksRef.current) && !inFlight.current.has(monday));
    if (missing.length === 0) return;
    setError(null);
    // One request per run of neighbouring missing weeks.
    const runs: string[][] = [];
    for (const monday of missing) {
      const run = runs[runs.length - 1];
      if (run && addDays(run[run.length - 1], 7) === monday) run.push(monday);
      else runs.push([monday]);
    }
    const startedAt = revision;
    for (const run of runs) {
      run.forEach((monday) => inFlight.current.add(monday));
      void listAllPlannedMeals(run[0], addDays(run[run.length - 1], 6))
        .then((result) => {
          if (revisionRef.current !== startedAt) return;
          const loaded: Record<string, PlannedMeal[]> = Object.fromEntries(run.map((monday) => [monday, []]));
          for (const item of result) loaded[weekStarts(item.cook_date, item.cook_date)[0]]?.push(item);
          setWeeks((previous) => ({ ...previous, ...loaded }));
        })
        .catch((failure: unknown) => {
          if (revisionRef.current === startedAt) setError(failure instanceof Error ? failure.message : 'Could not load planned meals');
        })
        .finally(() => run.forEach((monday) => inFlight.current.delete(monday)));
    }
  }, [from, to, revision]);

  const items = useMemo(() => weekStarts(from, to).flatMap((monday) => weeks[monday] ?? []), [weeks, from, to]);
  const loading = !error && weekStarts(from, to).some((monday) => !(monday in weeks));

  // Forget everything loaded and fetch the visible weeks again.
  const refresh = useCallback(() => {
    inFlight.current.clear();
    setWeeks({});
    setError(null);
    setRevision((value) => value + 1);
  }, []);

  return { items, loading, error, refresh };
}
