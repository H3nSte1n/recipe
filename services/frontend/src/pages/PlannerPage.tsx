import { useEffect, useMemo, useRef, useState } from 'react';
import PlannerCalendar from '../components/PlannerCalendar';
import PlannerList from '../components/PlannerList';
import PlannedMealForm from '../components/PlannedMealForm';
import { usePlannedMeals } from '../hooks/usePlannedMeals';
import { getMyRecipes } from '../services/recipeService';
import { createPlannedMeal, deletePlannedMeal, PlannedMealApiError, updatePlannedMeal } from '../services/plannedMealService';
import type { PlannedMeal, PlannedMealInput } from '../types/plannedMeal';
import type { Recipe } from '../types/recipe';
import { calendarDate, monthGrid, shiftMonth } from '../utils/plannerDates';
import '../styles/PlannerPage.css';

interface Props {
  onRecipes: () => void;
  onLogout: () => void;
}

export default function PlannerPage({ onRecipes, onLogout }: Props) {
  const [month, setMonth] = useState(() => new Date(new Date().getFullYear(), new Date().getMonth(), 1, 12));
  const [view, setView] = useState<'calendar' | 'list'>('calendar');
  const [form, setForm] = useState<{ date: string; item: PlannedMeal | null } | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const busyRef = useRef(false);
  const [recipes, setRecipes] = useState<Recipe[]>([]);
  const [recipeError, setRecipeError] = useState<string | null>(null);
  const dates = useMemo(() => monthGrid(month), [month]);
  const { items, loading, error, refresh } = usePlannedMeals(dates[0], dates[dates.length - 1]);

  useEffect(() => {
    let cancelled = false;
    void getMyRecipes()
      .then((result) => { if (!cancelled) setRecipes(result.filter((recipe) => recipe.status === 'published')); })
      .catch(() => { if (!cancelled) setRecipeError('Could not load recipes for planning.'); });
    return () => { cancelled = true; };
  }, []);

  function openForm(date: string, item: PlannedMeal | null = null) {
    setActionError(null);
    setFormError(null);
    setForm({ date, item });
  }

  async function save(input: PlannedMealInput) {
    if (!form || busyRef.current) return;
    busyRef.current = true;
    setSaving(true);
    setFormError(null);
    try {
      if (form.item) await updatePlannedMeal(form.item.id, input);
      else await createPlannedMeal(input);
      setForm(null);
      refresh();
    } catch (failure) {
      if (failure instanceof PlannedMealApiError && failure.status === 404) refresh();
      setFormError(failure instanceof Error ? failure.message : 'Could not save planned meal.');
    } finally {
      busyRef.current = false;
      setSaving(false);
    }
  }

  async function remove(item: PlannedMeal) {
    if (busyRef.current) return;
    if (!window.confirm(`Delete ${item.recipe.title} from ${item.cook_date}?`)) return;
    busyRef.current = true;
    setSaving(true);
    setActionError(null);
    setFormError(null);
    try {
      await deletePlannedMeal(item.id);
      setForm(null);
      refresh();
    } catch (failure) {
      const message = failure instanceof Error ? failure.message : 'Could not delete planned meal.';
      if (form) setFormError(message);
      else setActionError(message);
      if (failure instanceof PlannedMealApiError && failure.status === 404) refresh();
    } finally {
      busyRef.current = false;
      setSaving(false);
    }
  }

  return (
    <div className="planner">
      <header className="planner__header">
        <div className="planner__topbar">
          <button type="button" onClick={onRecipes}>← Recipes</button>
          <h1>Meal planner</h1>
          <button type="button" onClick={onLogout}>Sign out</button>
        </div>
        <div className="planner__toolbar">
          <div className="planner__month-nav">
            <button type="button" aria-label="Previous month" onClick={() => setMonth((current) => shiftMonth(current, -1))}>‹</button>
            <h2>{month.toLocaleDateString(undefined, { month: 'long', year: 'numeric' })}</h2>
            <button type="button" aria-label="Next month" onClick={() => setMonth((current) => shiftMonth(current, 1))}>›</button>
            <button type="button" onClick={() => setMonth(new Date(new Date().getFullYear(), new Date().getMonth(), 1, 12))}>Today</button>
          </div>
          <div className="planner__toolbar-actions">
            <div className="planner__view-switch" role="group" aria-label="Planner view">
              <button type="button" aria-pressed={view === 'calendar'} onClick={() => setView('calendar')}>Calendar</button>
              <button type="button" aria-pressed={view === 'list'} onClick={() => setView('list')}>List</button>
            </div>
            <button type="button" className="planner__add" onClick={() => {
              const today = calendarDate(new Date());
              openForm(today.slice(0, 7) === calendarDate(month).slice(0, 7) ? today : calendarDate(month));
            }}>Add meal</button>
          </div>
        </div>
      </header>
      <main className="planner__main">
        {recipeError && <p className="planner__error" role="alert">{recipeError}</p>}
        {actionError && <p className="planner__error" role="alert">{actionError}</p>}
        {loading && <p className="planner__status" role="status">Loading planned meals…</p>}
        {error && !loading && <div className="planner__status" role="alert"><p>{error}</p><button type="button" onClick={refresh}>Retry</button></div>}
        {!loading && !error && view === 'calendar' && items.length === 0 && <p className="planner__empty">No meals planned for this period. Choose a date to add one.</p>}
        {!loading && !error && (view === 'calendar'
          ? <PlannerCalendar dates={dates} month={month} items={items} onAdd={(date) => openForm(date)} onEdit={(item) => openForm(item.cook_date, item)} />
          : <PlannerList items={items} onEdit={(item) => openForm(item.cook_date, item)} onDelete={(item) => void remove(item)} />)}
      </main>
      {form && <PlannedMealForm key={form.item?.id ?? `new-${form.date}`} initial={form.item} date={form.date} recipes={recipes} saving={saving} error={formError} onSave={(input) => void save(input)} onCancel={() => { if (!busyRef.current) setForm(null); }} onDelete={(item) => void remove(item)} />}
    </div>
  );
}
