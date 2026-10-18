import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import PlannerPage from './PlannerPage';
import type { PlannedMeal } from '../types/plannedMeal';
import type { Recipe } from '../types/recipe';
import { addDays, calendarDate, shiftWeek, weekOf, weekRangeLabel, weekStart } from '../utils/plannerDates';
import { getMyRecipes } from '../services/recipeService';
import { createPlannedMeal, deletePlannedMeal, listAllPlannedMeals, updatePlannedMeal } from '../services/plannedMealService';

vi.mock('../services/recipeService', () => ({ getMyRecipes: vi.fn() }));
vi.mock('../services/plannedMealService', async (importOriginal) => {
  const original = await importOriginal<typeof import('../services/plannedMealService')>();
  return { ...original, listAllPlannedMeals: vi.fn(), createPlannedMeal: vi.fn(), updatePlannedMeal: vi.fn(), deletePlannedMeal: vi.fn() };
});

const recipe = { id: 'recipe-1', title: 'Pasta', status: 'published' } as Recipe;
const dayLabel = (date: Date) => date.toLocaleDateString(undefined, { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' });
const meal = (id: string, date: string): PlannedMeal => ({ id, recipe_id: recipe.id, cook_date: date, meal_type: null, servings: null, notes: null, recipe: { id: recipe.id, title: recipe.title, status: 'published' }, created_at: '', updated_at: '' });

beforeEach(() => {
  vi.mocked(getMyRecipes).mockResolvedValue([recipe]);
  vi.mocked(listAllPlannedMeals).mockResolvedValue([]);
});
afterEach(() => vi.clearAllMocks());

describe('planner', () => {
  it('defaults Add meal to today and restores focus after Escape', async () => {
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    const add = await screen.findByRole('button', { name: 'Add meal' });
    add.focus();
    fireEvent.click(add);
    expect(screen.getByLabelText('Cook date')).toHaveValue(calendarDate(new Date()));
    expect(screen.getByLabelText('Recipe')).toHaveFocus();
    await screen.findByRole('option', { name: 'Pasta' });
    fireEvent.change(screen.getByLabelText('Recipe'), { target: { value: recipe.id } });
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Tab', shiftKey: true });
    expect(screen.getByRole('button', { name: 'Save meal' })).toHaveFocus();
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(add).toHaveFocus();
  });

  it('shows same-day duplicates in the day list and supports edit and delete', async () => {
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
    const edits = () => within(screen.getByRole('group', { name: 'Day list' })).getAllByRole('button', { name: `Edit Pasta on ${date}` });
    await waitFor(() => expect(edits()).toHaveLength(2));
    fireEvent.click(edits()[0]);
    fireEvent.change(screen.getByLabelText('Meal'), { target: { value: 'dinner' } });
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Save meal' }));
    await waitFor(() => expect(updatePlannedMeal).toHaveBeenCalledWith('a', expect.objectContaining({ meal_type: 'dinner' })));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    fireEvent.click(edits()[0]);
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete' }));
    await waitFor(() => expect(deletePlannedMeal).toHaveBeenCalledWith('a'));
    await waitFor(() => expect(edits()).toHaveLength(1));
    vi.unstubAllGlobals();
  });

  it('keeps the form open during a pending save, then refreshes', async () => {
    let finish!: (value: PlannedMeal) => void;
    let entries: PlannedMeal[] = [];
    vi.mocked(listAllPlannedMeals).mockImplementation(async () => entries);
    vi.mocked(createPlannedMeal).mockImplementation(() => new Promise((resolve) => { finish = resolve; }));
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    fireEvent.click(await screen.findByRole('button', { name: 'Add meal' }));
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
    expect(await screen.findAllByRole('button', { name: `Edit Pasta on ${calendarDate(new Date())}` })).not.toHaveLength(0);
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

  it('retries a failed load', async () => {
    vi.mocked(listAllPlannedMeals).mockRejectedValueOnce(new Error('Temporary outage')).mockResolvedValueOnce([meal('a', calendarDate(new Date()))]);
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    expect(await screen.findByRole('alert')).toHaveTextContent('Temporary outage');
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(await screen.findByRole('button', { name: `Edit Pasta on ${calendarDate(new Date())}` })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument();
  });

  it('adds a meal on the day picked in the week strip', async () => {
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    const strip = await screen.findByRole('group', { name: 'Week' });
    const day = new Date(`${weekOf(calendarDate(new Date()))[0]}T12:00:00`);
    fireEvent.click(within(strip).getByRole('button', { name: dayLabel(day) }));
    fireEvent.click(screen.getByRole('button', { name: 'Add meal' }));
    expect(screen.getByLabelText('Cook date')).toHaveValue(calendarDate(day));
  });
});

describe('planner day list (mobile)', () => {
  const today = calendarDate(new Date());
  const sectionDates = () => Array.from(document.querySelectorAll<HTMLElement>('[data-date]')).map((section) => section.dataset.date);
  const stripDay = (date: string) => screen.getByRole('button', { name: dayLabel(new Date(`${date}T12:00:00`)) });
  const scrollPage = () => fireEvent.scroll(window);

  afterEach(() => vi.restoreAllMocks());

  it('opens on today with two past weeks and four future weeks around it', async () => {
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    await screen.findByRole('region', { name: today });
    const dates = sectionDates();
    expect(dates).toHaveLength(49);
    expect(dates[0]).toBe(shiftWeek(weekStart(today), -2));
    expect(dates).toContain(today);
    expect(stripDay(today)).toHaveAttribute('aria-pressed', 'true');
    expect(window.scrollTo).toHaveBeenCalled();
  });

  it('adds weeks above and loads them when scrolled near the top', async () => {
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    await screen.findByRole('region', { name: today });
    const firstBefore = sectionDates()[0]!;
    scrollPage();
    await waitFor(() => expect(sectionDates()).toHaveLength(77));
    expect(sectionDates()[0]).toBe(shiftWeek(firstBefore, -4));
    await waitFor(() => expect(listAllPlannedMeals).toHaveBeenCalledWith(shiftWeek(firstBefore, -4), addDays(firstBefore, -1)));
  });

  it('adds weeks below when scrolled near the bottom', async () => {
    vi.spyOn(window, 'scrollY', 'get').mockReturnValue(5000);
    vi.spyOn(document.documentElement, 'scrollHeight', 'get').mockReturnValue(5600);
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    await screen.findByRole('region', { name: today });
    const lastBefore = sectionDates()[48]!;
    scrollPage();
    await waitFor(() => expect(sectionDates()).toHaveLength(77));
    expect(sectionDates()[76]).toBe(addDays(lastBefore, 28));
  });

  it('moves the week strip to the day at the top of the list while scrolling', async () => {
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    await screen.findByRole('region', { name: today });
    const target = addDays(today, 17);
    const targetIndex = sectionDates().indexOf(target);
    // Lay the days out 100px apart so that `target` is the first one below the sticky header (bottom 0).
    vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
      const date = (this as HTMLElement).dataset?.date;
      const bottom = date ? (sectionDates().indexOf(date) - targetIndex) * 100 + 50 : 0;
      return { top: bottom - 50, bottom, left: 0, right: 0, width: 0, height: 50, x: 0, y: bottom - 50, toJSON: () => ({}) };
    });
    vi.spyOn(window, 'scrollY', 'get').mockReturnValue(2000);
    vi.spyOn(document.documentElement, 'scrollHeight', 'get').mockReturnValue(9000);
    scrollPage();
    await waitFor(() => expect(stripDay(target)).toHaveAttribute('aria-pressed', 'true'));
    expect(screen.queryByRole('button', { name: dayLabel(new Date(`${today}T12:00:00`)) })).not.toBeInTheDocument();
  });

  it('scrolls the list to a day tapped in the week strip, and back to today', async () => {
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    await screen.findByRole('region', { name: today });
    const monday = weekStart(today);
    const scrollTo = vi.mocked(window.scrollTo);
    scrollTo.mockClear();
    fireEvent.click(stripDay(monday));
    expect(scrollTo).toHaveBeenCalledTimes(1);
    expect(stripDay(monday)).toHaveAttribute('aria-pressed', 'true');
    fireEvent.click(screen.getByRole('button', { name: 'Today' }));
    expect(scrollTo).toHaveBeenCalledTimes(2);
    expect(stripDay(today)).toHaveAttribute('aria-pressed', 'true');
  });
});

describe('planner week view (desktop)', () => {
  const today = calendarDate(new Date());
  const week = weekOf(today);
  const typedMeal = (id: string, date: string, mealType: PlannedMeal['meal_type']): PlannedMeal => ({ ...meal(id, date), meal_type: mealType });

  beforeEach(() => {
    vi.stubGlobal('matchMedia', (query: string) => ({ matches: true, media: query, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
  });
  afterEach(() => vi.unstubAllGlobals());

  it('shows the current week and loads only that Monday-to-Sunday range', async () => {
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    expect(await screen.findByRole('heading', { name: weekRangeLabel(today) })).toBeInTheDocument();
    expect(screen.queryByRole('group', { name: 'Meal calendar' })).not.toBeInTheDocument();
    await waitFor(() => expect(listAllPlannedMeals).toHaveBeenCalledWith(week[0], week[6]));
  });

  it('moves between weeks and back to today', async () => {
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    fireEvent.click(await screen.findByRole('button', { name: 'Next week' }));
    expect(screen.getByRole('heading', { name: weekRangeLabel(shiftWeek(today, 1)) })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Previous week' }));
    fireEvent.click(screen.getByRole('button', { name: 'Previous week' }));
    expect(screen.getByRole('heading', { name: weekRangeLabel(shiftWeek(today, -1)) })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Today' }));
    expect(screen.getByRole('heading', { name: weekRangeLabel(today) })).toBeInTheDocument();
  });

  it('shows the Snacks row only when the week has a snack or untyped meal', async () => {
    vi.mocked(listAllPlannedMeals).mockResolvedValue([typedMeal('a', week[0], 'lunch')]);
    const { unmount } = render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    await screen.findByRole('button', { name: `Edit Pasta on ${week[0]}` });
    expect(screen.getByRole('region', { name: 'Lunch' })).toBeInTheDocument();
    expect(screen.queryByRole('region', { name: 'Snacks' })).not.toBeInTheDocument();
    unmount();

    vi.mocked(listAllPlannedMeals).mockResolvedValue([typedMeal('b', week[2], null), typedMeal('c', week[3], 'snack')]);
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    const snacks = await screen.findByRole('region', { name: 'Snacks' });
    expect(within(snacks).getAllByRole('button', { name: /^Edit Pasta/ })).toHaveLength(2);
  });

  it('opens the add form prefilled from an empty cell and the edit form from a card', async () => {
    vi.mocked(listAllPlannedMeals).mockResolvedValue([typedMeal('a', week[1], 'dinner')]);
    render(<PlannerPage onRecipes={vi.fn()} onLogout={vi.fn()} />);
    fireEvent.click(await screen.findByRole('button', { name: `Add Lunch on ${week[4]}` }));
    expect(screen.getByLabelText('Cook date')).toHaveValue(week[4]);
    expect(screen.getByLabelText('Meal')).toHaveValue('lunch');
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });

    fireEvent.click(screen.getByRole('button', { name: `Edit Pasta on ${week[1]}` }));
    expect(screen.getByRole('heading', { name: 'Edit planned meal' })).toBeInTheDocument();
    expect(screen.getByLabelText('Meal')).toHaveValue('dinner');
  });
});
