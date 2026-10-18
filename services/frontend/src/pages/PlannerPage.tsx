import { useEffect, useMemo, useRef, useState } from 'react';
import PlannerList from '../components/PlannerList';
import PlannerWeekGrid from '../components/PlannerWeekGrid';
import PlannerWeekStrip from '../components/PlannerWeekStrip';
import PlannedMealForm from '../components/PlannedMealForm';
import { useMediaQuery } from '../hooks/useMediaQuery';
import { usePlannedMeals } from '../hooks/usePlannedMeals';
import { getMyRecipes } from '../services/recipeService';
import { createPlannedMeal, deletePlannedMeal, PlannedMealApiError, updatePlannedMeal } from '../services/plannedMealService';
import type { MealType, PlannedMeal, PlannedMealInput } from '../types/plannedMeal';
import type { Recipe } from '../types/recipe';
import { addDays, calendarDate, shiftWeek, weekOf, weekRangeLabel, weekStart } from '../utils/plannerDates';
import '../styles/PlannerPage.css';

const LIST_DAYS = 14;
const DESKTOP_QUERY = '(min-width: 1024px)';

interface Props {
  onRecipes: () => void;
  onLogout: () => void;
}

export default function PlannerPage({ onRecipes, onLogout }: Props) {
  const [selectedDate, setSelectedDate] = useState(() => calendarDate(new Date()));
  const [form, setForm] = useState<{ date: string; item: PlannedMeal | null; mealType: MealType | null } | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const busyRef = useRef(false);
  const [recipes, setRecipes] = useState<Recipe[]>([]);
  const [recipeError, setRecipeError] = useState<string | null>(null);
  const isDesktop = useMediaQuery(DESKTOP_QUERY);
  // Desktop shows the selected week; smaller screens list two weeks from the selected week's Monday.
  const listDates = useMemo(() => {
    const first = weekStart(selectedDate);
    return Array.from({ length: LIST_DAYS }, (_, index) => addDays(first, index));
  }, [selectedDate]);
  const visibleDates = useMemo(() => (isDesktop ? weekOf(selectedDate) : listDates), [isDesktop, selectedDate, listDates]);
  const today = calendarDate(new Date());
  const recipesById = useMemo(() => new Map(recipes.map((recipe) => [recipe.id, recipe])), [recipes]);
  const { items, loading, error, refresh } = usePlannedMeals(visibleDates[0], visibleDates[visibleDates.length - 1]);

  useEffect(() => {
    let cancelled = false;
    void getMyRecipes()
      .then((result) => { if (!cancelled) setRecipes(result.filter((recipe) => recipe.status === 'published')); })
      .catch(() => { if (!cancelled) setRecipeError('Could not load recipes for planning.'); });
    return () => { cancelled = true; };
  }, []);

  function selectDate(date: string) {
    setSelectedDate(date);
  }

  function openForm(date: string, item: PlannedMeal | null = null, mealType: MealType | null = null) {
    setActionError(null);
    setFormError(null);
    setForm({ date, item, mealType });
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
    <div className={`planner${isDesktop ? ' planner--week' : ''}`}>
      <header className="planner__header">
        <div className="planner__topbar">
          <button type="button" className="planner__text-btn" onClick={onRecipes}>← Recipes</button>
          <button type="button" className="planner__text-btn" onClick={onLogout}>Sign out</button>
        </div>
      </header>
      <main className="planner__main">
        {recipeError && <p className="planner__error" role="alert">{recipeError}</p>}
        {actionError && <p className="planner__error" role="alert">{actionError}</p>}
        {loading && <p className="planner__status" role="status">Loading planned meals…</p>}
        {error && !loading && <div className="planner__status" role="alert"><p>{error}</p><button type="button" onClick={refresh}>Retry</button></div>}
        {isDesktop ? (
          <section className="planner__week-pane" aria-label="Week">
            <div className="planner__week-toolbar">
              <h1 className="planner__week-title">{weekRangeLabel(selectedDate)}</h1>
              <div className="planner__nav">
                <button type="button" className="planner__round-btn" aria-label="Previous week" onClick={() => selectDate(shiftWeek(selectedDate, -1))}>‹</button>
                <button type="button" className="planner__round-btn" aria-label="Next week" onClick={() => selectDate(shiftWeek(selectedDate, 1))}>›</button>
                <button type="button" className="planner__pill-btn" onClick={() => selectDate(today)}>Today</button>
              </div>
            </div>
            <PlannerWeekGrid dates={visibleDates} items={items} recipesById={recipesById} today={today} onEdit={(item) => openForm(item.cook_date, item)} onAdd={(date, mealType) => openForm(date, null, mealType)} />
          </section>
        ) : (
          <section className="planner__list-pane" aria-label="Day">
            <div className="planner__week-strip">
              <PlannerWeekStrip selectedDate={selectedDate} today={today} onSelect={selectDate} />
            </div>
            <PlannerList dates={listDates} items={items} recipesById={recipesById} onEdit={(item) => openForm(item.cook_date, item)} onSelectDate={selectDate} />
          </section>
        )}
      </main>
      <button type="button" className="planner__fab" aria-label="Add meal" onClick={() => openForm(selectedDate)}>+</button>
      {form && <PlannedMealForm key={form.item?.id ?? `new-${form.date}-${form.mealType}`} initial={form.item} date={form.date} defaultMealType={form.mealType} recipes={recipes} saving={saving} error={formError} onSave={(input) => void save(input)} onCancel={() => { if (!busyRef.current) setForm(null); }} onDelete={(item) => void remove(item)} />}
    </div>
  );
}
