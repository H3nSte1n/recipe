import { afterEach, describe, expect, it, vi } from 'vitest';
import { listAllPlannedMeals, PlannedMealApiError } from './plannedMealService';
import type { PlannedMeal } from '../types/plannedMeal';

const meal = (id: string): PlannedMeal => ({ id, recipe_id: 'recipe', cook_date: '2026-10-03', meal_type: null, servings: null, notes: null, recipe: { id: 'recipe', title: id, status: 'published' }, created_at: '', updated_at: '' });

afterEach(() => vi.unstubAllGlobals());

describe('planned meal service', () => {
  it('fetches all pages and passes the abort signal', async () => {
    const signal = new AbortController().signal;
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ items: [meal('a')], total: 2, page: 1, page_size: 100 }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ items: [meal('b')], total: 2, page: 2, page_size: 100 }), { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);
    expect((await listAllPlannedMeals('2026-10-01', '2026-10-31', signal)).map((item) => item.id)).toEqual(['a', 'b']);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(String(fetchMock.mock.calls[1][0])).toContain('page=2');
    expect(fetchMock.mock.calls[0][1].signal).toBe(signal);
  });

  it('provides status and safe server message on failure', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: 'No such meal' }), { status: 404 })));
    await expect(listAllPlannedMeals('2026-10-01', '2026-10-31')).rejects.toEqual(new PlannedMealApiError(404, 'No such meal'));
  });
});
