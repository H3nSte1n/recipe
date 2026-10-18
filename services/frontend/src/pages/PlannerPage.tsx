import { useEffect, useMemo, useRef, useState } from 'react';
import PlannerCalendar from '../components/PlannerCalendar';
import PlannerList from '../components/PlannerList';
import PlannerWeekStrip from '../components/PlannerWeekStrip';
import PlannedMealForm from '../components/PlannedMealForm';
import { useMediaQuery } from '../hooks/useMediaQuery';
import { usePlannedMeals } from '../hooks/usePlannedMeals';
import { getMyRecipes } from '../services/recipeService';
import { createPlannedMeal, deletePlannedMeal, PlannedMealApiError, updatePlannedMeal } from '../services/plannedMealService';
import type { PlannedMeal, PlannedMealInput } from '../types/plannedMeal';
import type { Recipe } from '../types/recipe';
import { calendarDate, monthGrid, parseCalendarDate, shiftMonth, weekOf } from '../utils/plannerDates';
import '../styles/PlannerPage.css';

const LIST_DAYS = 14;
const DESKTOP_QUERY = '(min-width: 1024px)';

function firstOfMonth(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), 1, 12);
}

interface Props {
  onRecipes: () => void;
  onLogout: () => void;
}

export default function PlannerPage({ onRecipes, onLogout }: Props) {
  const [month, setMonth] = useState(() => new Date(new Date().getFullYear(), new Date().getMonth(), 1, 12));
  const [selectedDate, setSelectedDate] = useState(() => calendarDate(new Date()));
  // Only matters below 1024px; wider screens show calendar and list together.
  const [mobileView, setMobileView] = useState<'calendar' | 'list'>('list');
  const [form, setForm] = useState<{ date: string; item: PlannedMeal | null } | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const busyRef = useRef(false);
  const [recipes, setRecipes] = useState<Recipe[]>([]);
  const [recipeError, setRecipeError] = useState<string | null>(null);
  const isDesktop = useMediaQuery(DESKTOP_QUERY);
  const dates = useMemo(() => monthGrid(month), [month]);
  // Desktop shows one week; smaller screens show the whole month grid.
  const visibleDates = useMemo(() => (isDesktop ? weekOf(selectedDate) : dates), [isDesktop, selectedDate, dates]);
  const today = calendarDate(new Date());
  const recipesById = useMemo(() => new Map(recipes.map((recipe) => [recipe.id, recipe])), [recipes]);
  const listDates = useMemo(() => dates.filter((date) => date >= selectedDate).slice(0, LIST_DAYS), [dates, selectedDate]);
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
    // Picking a day from a neighbouring month moves the grid to that month.
    const picked = parseCalendarDate(date);
    if (picked.getMonth() !== month.getMonth() || picked.getFullYear() !== month.getFullYear()) setMonth(firstOfMonth(picked));
  }

  function openDay(date: string) {
    selectDate(date);
    setMobileView('list');
  }

  function showMonth(target: Date) {
    setMonth(target);
    const isCurrentMonth = target.getMonth() === new Date().getMonth() && target.getFullYear() === new Date().getFullYear();
    setSelectedDate(isCurrentMonth ? calendarDate(new Date()) : calendarDate(target));
  }

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
    <div className={`planner planner--${mobileView}`}>
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
        <div className="planner__layout">
          <section className="planner__calendar-pane" aria-label="Calendar">
            <div className="planner__toolbar">
              <h1 className="planner__month-title">{month.toLocaleDateString(undefined, { month: 'long', year: 'numeric' })}</h1>
              <div className="planner__month-nav">
                <button type="button" className="planner__round-btn" aria-label="Previous month" onClick={() => showMonth(shiftMonth(month, -1))}>‹</button>
                <button type="button" className="planner__round-btn" aria-label="Next month" onClick={() => showMonth(shiftMonth(month, 1))}>›</button>
                <button type="button" className="planner__pill-btn" onClick={() => showMonth(firstOfMonth(new Date()))}>Today</button>
              </div>
            </div>
            <PlannerCalendar dates={dates} month={month} items={items} selectedDate={selectedDate} today={today} onSelectDate={openDay} onEdit={(item) => openForm(item.cook_date, item)} />
          </section>
          <section className="planner__list-pane" aria-label="Day">
            <div className="planner__week-strip">
              <PlannerWeekStrip selectedDate={selectedDate} today={today} onSelect={selectDate} />
              <button type="button" className="planner__chevron" aria-label="Show month calendar" aria-expanded={false} onClick={() => setMobileView('calendar')}>⌄</button>
            </div>
            <PlannerList dates={listDates} items={items} recipesById={recipesById} onEdit={(item) => openForm(item.cook_date, item)} onSelectDate={openDay} />
          </section>
        </div>
      </main>
      <button type="button" className="planner__fab" aria-label="Add meal" onClick={() => openForm(selectedDate)}>+</button>
      {form && <PlannedMealForm key={form.item?.id ?? `new-${form.date}`} initial={form.item} date={form.date} recipes={recipes} saving={saving} error={formError} onSave={(input) => void save(input)} onCancel={() => { if (!busyRef.current) setForm(null); }} onDelete={(item) => void remove(item)} />}
    </div>
  );
}
