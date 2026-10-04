import { useCallback, useEffect, useState } from 'react';
import type { PlannedMeal } from '../types/plannedMeal';
import { listAllPlannedMeals } from '../services/plannedMealService';

export function usePlannedMeals(from: string, to: string) {
  const [items, setItems] = useState<PlannedMeal[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError(null);
    // Clear old month while its replacement is in flight.
    setItems([]);
    void listAllPlannedMeals(from, to, controller.signal)
      .then((result) => { if (!controller.signal.aborted) setItems(result); })
      .catch((failure: unknown) => {
        if (!controller.signal.aborted) setError(failure instanceof Error ? failure.message : 'Could not load planned meals');
      })
      .finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, [from, to, revision]);

  const refresh = useCallback(() => setRevision((value) => value + 1), []);
  return { items, loading, error, refresh };
}
