import { afterEach, describe, expect, it, vi } from 'vitest';
import { createPlannedMeal, listAllPlannedMeals, PlannedMealApiError, plannedMealErrorMessage } from './plannedMealService';
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

  it.each([
    [403, 'Please verify your email before changing your meal plan.'],
    [404, 'That planned meal or recipe no longer exists.'],
    [500, 'Something went wrong on our side. Please try again.'],
  ])('keeps the status and shows a friendly message for %i, never the server text for other statuses', async (status, message) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: 'invalid recipe_id' }), { status })));
    await expect(listAllPlannedMeals('2026-10-01', '2026-10-31')).rejects.toEqual(new PlannedMealApiError(status, message));
  });

  it('explains a 400 with the backend message', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: 'only published recipes can be planned' }), { status: 400 })));
    await expect(listAllPlannedMeals('2026-10-01', '2026-10-31')).rejects.toEqual(
      new PlannedMealApiError(400, 'Those meal details are not valid: only published recipes can be planned.'),
    );
  });

  it('falls back to a generic 400 message when the body is not a short error string', async () => {
    const generic = 'Those meal details are not valid. Please check them and try again.';
    for (const body of ['not json', JSON.stringify({ error: 'x'.repeat(300) }), JSON.stringify({ error: { nested: true } })]) {
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(body, { status: 400 })));
      await expect(createPlannedMeal({ recipe_id: 'r', cook_date: '2026-10-03', meal_type: null, servings: null, notes: null })).rejects.toEqual(new PlannedMealApiError(400, generic));
    }
  });

  it('only shows messages the service produced, never browser error text', () => {
    expect(plannedMealErrorMessage(new TypeError('Failed to fetch'), 'fallback')).toBe('fallback');
    expect(plannedMealErrorMessage(new SyntaxError('Unexpected token'), 'fallback')).toBe('fallback');
    expect(plannedMealErrorMessage(new PlannedMealApiError(404, 'gone'), 'fallback')).toBe('gone');
  });
});
