import { useEffect, useRef, useState, type FormEvent, type KeyboardEvent } from 'react';
import type { Recipe } from '../types/recipe';
import type { MealType, PlannedMeal, PlannedMealInput } from '../types/plannedMeal';

interface Props {
  initial: PlannedMeal | null;
  date: string;
  recipes: Recipe[];
  saving: boolean;
  error: string | null;
  onSave: (input: PlannedMealInput) => void;
  onCancel: () => void;
  onDelete: (item: PlannedMeal) => void;
}

export default function PlannedMealForm({ initial, date, recipes, saving, error, onSave, onCancel, onDelete }: Props) {
  const [recipeId, setRecipeId] = useState(initial?.recipe_id ?? '');
  const [cookDate, setCookDate] = useState(initial?.cook_date ?? date);
  const [mealType, setMealType] = useState(initial?.meal_type ?? '');
  const [servings, setServings] = useState(initial?.servings?.toString() ?? '');
  const [notes, setNotes] = useState(initial?.notes ?? '');
  const dialogRef = useRef<HTMLElement>(null);
  const recipeRef = useRef<HTMLSelectElement>(null);
  const selectedRecipeAvailable = recipes.some((recipe) => recipe.id === initial?.recipe_id);

  useEffect(() => {
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    recipeRef.current?.focus();
    return () => previousFocus?.focus();
  }, []);

  function handleKeyDown(event: KeyboardEvent<HTMLElement>) {
    if (event.key === 'Escape' && !saving) {
      event.preventDefault();
      onCancel();
      return;
    }
    if (event.key !== 'Tab' || !dialogRef.current) return;
    const focusable = Array.from(dialogRef.current.querySelectorAll<HTMLElement>('button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled)'));
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last?.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first?.focus();
    }
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSave({
      recipe_id: recipeId,
      cook_date: cookDate,
      meal_type: (mealType || null) as MealType | null,
      servings: servings === '' ? null : Number(servings),
      notes: notes.trim() || null,
    });
  }

  return (
    <div className="planner-form-backdrop" onMouseDown={(event) => { if (!saving && event.target === event.currentTarget) onCancel(); }}>
      <section ref={dialogRef} className="planner-form" role="dialog" aria-modal="true" aria-labelledby="planner-form-title" onKeyDown={handleKeyDown}>
        <h2 id="planner-form-title">{initial ? 'Edit planned meal' : 'Plan a meal'}</h2>
        <form onSubmit={submit}>
          <label htmlFor="planner-recipe">Recipe</label>
          <select ref={recipeRef} id="planner-recipe" required value={recipeId} onChange={(event) => setRecipeId(event.target.value)}>
            <option value="">Select a recipe</option>
            {initial && !selectedRecipeAvailable && <option value={initial.recipe_id}>{initial.recipe.title}</option>}
            {recipes.map((recipe) => <option key={recipe.id} value={recipe.id}>{recipe.title}</option>)}
          </select>
          <label htmlFor="planner-date">Cook date</label>
          <input id="planner-date" type="date" required value={cookDate} onChange={(event) => setCookDate(event.target.value)} />
          <label htmlFor="planner-meal-type">Meal</label>
          <select id="planner-meal-type" value={mealType} onChange={(event) => setMealType(event.target.value)}>
            <option value="">Any meal</option>
            <option value="breakfast">Breakfast</option>
            <option value="lunch">Lunch</option>
            <option value="dinner">Dinner</option>
            <option value="snack">Snack</option>
          </select>
          <label htmlFor="planner-servings">Servings override</label>
          <input id="planner-servings" type="number" min="1" max="100" step="1" value={servings} onChange={(event) => setServings(event.target.value)} placeholder="Use recipe servings" />
          <label htmlFor="planner-notes">Notes</label>
          <textarea id="planner-notes" maxLength={1000} value={notes} onChange={(event) => setNotes(event.target.value)} />
          {error && <p className="planner__error" role="alert">{error}</p>}
          <div className="planner-form__actions">
            {initial && <button type="button" className="planner-form__delete" disabled={saving} onClick={() => onDelete(initial)}>Delete</button>}
            <button type="button" disabled={saving} onClick={onCancel}>Cancel</button>
            <button type="submit" disabled={saving || !recipeId}>{saving ? 'Saving…' : 'Save meal'}</button>
          </div>
        </form>
      </section>
    </div>
  );
}
