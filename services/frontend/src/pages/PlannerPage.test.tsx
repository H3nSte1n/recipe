import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import PlannerPage from './PlannerPage';
import type { PlannedMeal } from '../types/plannedMeal';
import type { Recipe } from '../types/recipe';
import { calendarDate, shiftMonth } from '../utils/plannerDates';
import { getMyRecipes } from '../services/recipeService';
import { createPlannedMeal, deletePlannedMeal, listAllPlannedMeals, updatePlannedMeal } from '../services/plannedMealService';

vi.mock('../services/recipeService', () => ({ getMyRecipes: vi.fn() }));
vi.mock('../services/plannedMealService', async (importOriginal) => {
  const original = await importOriginal<typeof import('../services/plannedMealService')>();
  return { ...original, listAllPlannedMeals: vi.fn(), createPlannedMeal: vi.fn(), updatePlannedMeal: vi.fn(), deletePlannedMeal: vi.fn() };
});

const recipe = { id: 'recipe-1', title: 'Pasta', status: 'published' } as Recipe;
const meal = (id: string, date: string): PlannedMeal => ({ id, recipe_id: recipe.id, cook_date: date, meal_type: null, servings: null, notes: null, recipe: { id: recipe.id, title: recipe.title, status: 'published' }, created_at: '', updated_at: '' });

beforeEach(() => {
  vi.mocked(getMyRecipes).mockResolvedValue([recipe]);
  vi.mocked(listAllPlannedMeals).mockResolvedValue([]);
});
afterEach(() => vi.clearAllMocks());

describe('planner', () => {
  it('defaults Add meal to the visible month and restores focus after Escape', async () => {
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    const next = screen.getByRole('button', { name: 'Next month' });
    fireEvent.click(next);
    const add = screen.getByRole('button', { name: 'Add meal' });
    add.focus();
    fireEvent.click(add);
    const expected = calendarDate(shiftMonth(new Date(), 1));
    expect(screen.getByLabelText('Cook date')).toHaveValue(expected);
    expect(screen.getByLabelText('Recipe')).toHaveFocus();
    await screen.findByRole('option', { name: 'Pasta' });
    fireEvent.change(screen.getByLabelText('Recipe'), { target: { value: recipe.id } });
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Tab', shiftKey: true });
    expect(screen.getByRole('button', { name: 'Save meal' })).toHaveFocus();
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(add).toHaveFocus();
  });

  it('shows same-day duplicates in both views and supports edit and delete', async () => {
    const date = calendarDate(new Date());
    let entries = [meal('a', date), meal('b', date)];
    vi.mocked(listAllPlannedMeals).mockImplementation(async () => entries);
    vi.mocked(updatePlannedMeal).mockImplementation(async (id, input) => {
      entries = entries.map((item) => item.id === id ? { ...item, ...input } : item);
      return entries.find((item) => item.id === id)!;
    });
    vi.mocked(deletePlannedMeal).mockImplementation(async (id) => { entries = entries.filter((item) => item.id !== id); });
    vi.stubGlobal('confirm', vi.fn(() => true));
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    await waitFor(() => expect(screen.getAllByRole('button', { name: `Edit Pasta on ${date}` })).toHaveLength(2));
    fireEvent.click(screen.getByRole('button', { name: 'List' }));
    expect(screen.getAllByRole('button', { name: `Edit Pasta on ${date}` })).toHaveLength(2);
    fireEvent.click(screen.getAllByRole('button', { name: `Edit Pasta on ${date}` })[0]);
    fireEvent.change(screen.getByLabelText('Meal'), { target: { value: 'dinner' } });
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Save meal' }));
    await waitFor(() => expect(updatePlannedMeal).toHaveBeenCalledWith('a', expect.objectContaining({ meal_type: 'dinner' })));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    fireEvent.click(screen.getAllByRole('button', { name: `Delete Pasta on ${date}` })[0]);
    await waitFor(() => expect(deletePlannedMeal).toHaveBeenCalledWith('a'));
    await waitFor(() => expect(screen.getAllByRole('button', { name: `Edit Pasta on ${date}` })).toHaveLength(1));
    vi.unstubAllGlobals();
  });

  it('keeps the form open during a pending save, then refreshes', async () => {
    let finish!: (value: PlannedMeal) => void;
    let entries: PlannedMeal[] = [];
    vi.mocked(listAllPlannedMeals).mockImplementation(async () => entries);
    vi.mocked(createPlannedMeal).mockImplementation(() => new Promise((resolve) => { finish = resolve; }));
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Add meal' }));
    await screen.findByRole('option', { name: 'Pasta' });
    fireEvent.change(screen.getByLabelText('Recipe'), { target: { value: recipe.id } });
    fireEvent.click(screen.getByRole('button', { name: 'Save meal' }));
    await waitFor(() => expect(createPlannedMeal).toHaveBeenCalledTimes(1));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled();
    entries = [meal('created', calendarDate(new Date()))];
    finish(entries[0]);
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(listAllPlannedMeals).toHaveBeenCalledTimes(2);
    expect(await screen.findByRole('button', { name: `Edit Pasta on ${calendarDate(new Date())}` })).toBeInTheDocument();
  });

  it('shows a save error and retains the draft', async () => {
    vi.mocked(createPlannedMeal).mockRejectedValue(new Error('Please verify your email'));
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Add meal' }));
    await screen.findByRole('option', { name: 'Pasta' });
    fireEvent.change(screen.getByLabelText('Recipe'), { target: { value: recipe.id } });
    fireEvent.click(screen.getByRole('button', { name: 'Save meal' }));
    expect(await within(screen.getByRole('dialog')).findByRole('alert')).toHaveTextContent('Please verify your email');
    expect(screen.getByLabelText('Recipe')).toHaveValue(recipe.id);
  });

  it('ignores a stale month response and can retry a failed load', async () => {
    let finishOld!: (value: PlannedMeal[]) => void;
    const oldDate = calendarDate(new Date());
    const nextDate = calendarDate(shiftMonth(new Date(), 1));
    vi.mocked(listAllPlannedMeals)
      .mockImplementationOnce(() => new Promise((resolve) => { finishOld = resolve; }))
      .mockResolvedValueOnce([meal('new', nextDate)])
      .mockRejectedValueOnce(new Error('Temporary outage'))
      .mockResolvedValueOnce([]);
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'List' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next month' }));
    await screen.findByRole('button', { name: `Edit Pasta on ${nextDate}` });
    finishOld([meal('old', oldDate)]);
    expect(screen.queryByRole('button', { name: `Edit Pasta on ${oldDate}` })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Next month' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Temporary outage');
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(await screen.findByText('No meals planned for this period.')).toBeInTheDocument();
  });
});
