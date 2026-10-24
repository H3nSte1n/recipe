import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { PlannedMeal } from '../types/plannedMeal';
import { listAllPlannedMeals, plannedMealErrorMessage } from '../services/plannedMealService';
import { addDays, weekStart, weekStarts } from '../utils/plannerDates';

// The API rejects ranges over 93 days, so a request covers at most 13 whole weeks (91 days).
const MAX_WEEKS_PER_REQUEST = 13;

// Loads planned meals for whole Monday–Sunday weeks (from = a Monday, to = a Sunday).
// Weeks already loaded are kept, so moving around only fetches what is new.
export function usePlannedMeals(from: string, to: string) {
  const [weeks, setWeeks] = useState<Record<string, PlannedMeal[]>>({});
  const [error, setError] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);
  const weeksRef = useRef(weeks);
  weeksRef.current = weeks;
  const inFlight = useRef(new Set<string>());
  // Weeks whose loaded meals are out of date: still shown, but fetched again when visible.
  const stale = useRef(new Set<string>());
  const revisionRef = useRef(revision);
  revisionRef.current = revision;
  const controller = useRef(new AbortController());

  // Requests outlive range changes (loaded weeks are kept) but not the page itself.
  useEffect(() => {
    const current = new AbortController();
    const requests = inFlight.current;
    controller.current = current;
    return () => {
      current.abort();
      requests.clear(); // aborted requests never finish; StrictMode re-runs effects and must refetch
    };
  }, []);

  useEffect(() => {
    const missing = weekStarts(from, to).filter((monday) => (!(monday in weeksRef.current) || stale.current.has(monday)) && !inFlight.current.has(monday));
    if (missing.length === 0) return;
    setError(null);
    // One request per run of neighbouring missing weeks, capped at MAX_WEEKS_PER_REQUEST.
    const runs: string[][] = [];
    for (const monday of missing) {
      const run = runs[runs.length - 1];
      if (run && run.length < MAX_WEEKS_PER_REQUEST && addDays(run[run.length - 1], 7) === monday) run.push(monday);
      else runs.push([monday]);
    }
    const startedAt = revision;
    const { signal } = controller.current;
    for (const run of runs) {
      run.forEach((monday) => inFlight.current.add(monday));
      void listAllPlannedMeals(run[0], addDays(run[run.length - 1], 6), signal)
        .then((result) => {
          if (signal.aborted || revisionRef.current !== startedAt) return;
          const loaded: Record<string, PlannedMeal[]> = Object.fromEntries(run.map((monday) => [monday, []]));
          for (const item of result) loaded[weekStart(item.cook_date)]?.push(item);
          run.forEach((monday) => stale.current.delete(monday));
          setWeeks((previous) => ({ ...previous, ...loaded }));
        })
        .catch((failure: unknown) => {
          if (signal.aborted || revisionRef.current !== startedAt) return;
          setError(plannedMealErrorMessage(failure, 'Could not load planned meals. Please check your connection and try again.'));
        })
        .finally(() => {
          // A refresh already cleared inFlight; don't drop markers of the requests it started.
          if (!signal.aborted && revisionRef.current === startedAt) run.forEach((monday) => inFlight.current.delete(monday));
        });
    }
  }, [from, to, revision]);

  const items = useMemo(() => weekStarts(from, to).flatMap((monday) => weeks[monday] ?? []), [weeks, from, to]);
  const loading = !error && weekStarts(from, to).some((monday) => !(monday in weeks));

  // Fetch fresh data for the weeks containing `dates` (every loaded week when omitted). The old
  // meals stay on screen until the new ones arrive, so the list does not blank out after an edit.
  const refresh = useCallback((dates?: string[]) => {
    const mondays = dates ? dates.map(weekStart) : Object.keys(weeksRef.current);
    mondays.forEach((monday) => stale.current.add(monday));
    inFlight.current.clear();
    setError(null);
    setRevision((value) => value + 1);
  }, []);

  return { items, loading, error, refresh };
}
