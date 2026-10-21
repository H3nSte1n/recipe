import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook, waitFor } from '@testing-library/react';
import { usePlannedMeals } from './usePlannedMeals';
import { listAllPlannedMeals } from '../services/plannedMealService';
import type { PlannedMeal } from '../types/plannedMeal';

vi.mock('../services/plannedMealService', () => ({ listAllPlannedMeals: vi.fn() }));

const meal = (id: string, cook_date: string) => ({ id, cook_date }) as PlannedMeal;

beforeEach(() => vi.mocked(listAllPlannedMeals).mockResolvedValue([]));
afterEach(() => vi.clearAllMocks());

describe('usePlannedMeals', () => {
  it('fetches only weeks it has not loaded yet and keeps the old ones', async () => {
    vi.mocked(listAllPlannedMeals).mockImplementation(async (from) => (from === '2026-10-05' ? [meal('a', '2026-10-07')] : [meal('b', '2026-10-14')]));
    const { result, rerender } = renderHook(({ from, to }) => usePlannedMeals(from, to), { initialProps: { from: '2026-10-05', to: '2026-10-11' } });
    expect(result.current.loading).toBe(true);
    await waitFor(() => expect(result.current.items.map((item) => item.id)).toEqual(['a']));

    rerender({ from: '2026-10-05', to: '2026-10-18' });
    await waitFor(() => expect(result.current.items.map((item) => item.id)).toEqual(['a', 'b']));
    expect(listAllPlannedMeals).toHaveBeenCalledTimes(2);
    expect(listAllPlannedMeals).toHaveBeenLastCalledWith('2026-10-12', '2026-10-18');

    rerender({ from: '2026-10-12', to: '2026-10-18' });
    expect(result.current.items.map((item) => item.id)).toEqual(['b']);
    expect(listAllPlannedMeals).toHaveBeenCalledTimes(2);
  });

  it('loads a run of missing weeks in one request', async () => {
    renderHook(() => usePlannedMeals('2026-10-05', '2026-10-25'));
    await waitFor(() => expect(listAllPlannedMeals).toHaveBeenCalledWith('2026-10-05', '2026-10-25'));
    expect(listAllPlannedMeals).toHaveBeenCalledTimes(1);
  });

  it('splits a long window into requests of at most 13 weeks', async () => {
    // 15 weeks: Monday 2026-09-21 to Sunday 2027-01-03 (105 days, over the API's 93-day limit).
    renderHook(() => usePlannedMeals('2026-09-21', '2027-01-03'));
    await waitFor(() => expect(listAllPlannedMeals).toHaveBeenCalledTimes(2));
    expect(listAllPlannedMeals).toHaveBeenCalledWith('2026-09-21', '2026-12-20');
    expect(listAllPlannedMeals).toHaveBeenCalledWith('2026-12-21', '2027-01-03');
  });

  it('ignores a response that arrives after a refresh and reloads', async () => {
    let finishOld!: (value: PlannedMeal[]) => void;
    vi.mocked(listAllPlannedMeals)
      .mockImplementationOnce(() => new Promise((resolve) => { finishOld = resolve; }))
      .mockResolvedValueOnce([meal('new', '2026-10-06')]);
    const { result } = renderHook(() => usePlannedMeals('2026-10-05', '2026-10-11'));
    act(() => result.current.refresh());
    await waitFor(() => expect(result.current.items.map((item) => item.id)).toEqual(['new']));
    await act(async () => finishOld([meal('old', '2026-10-05')]));
    expect(result.current.items.map((item) => item.id)).toEqual(['new']);
  });

  it('reports a failure and retries on refresh', async () => {
    vi.mocked(listAllPlannedMeals).mockRejectedValueOnce(new Error('Temporary outage')).mockResolvedValueOnce([meal('a', '2026-10-06')]);
    const { result } = renderHook(() => usePlannedMeals('2026-10-05', '2026-10-11'));
    await waitFor(() => expect(result.current.error).toBe('Temporary outage'));
    expect(result.current.loading).toBe(false);
    act(() => result.current.refresh());
    await waitFor(() => expect(result.current.items).toHaveLength(1));
    expect(result.current.error).toBeNull();
  });
});
