import { useMemo } from 'react';
import type { PlannedMeal } from '../types/plannedMeal';
import type { Recipe } from '../types/recipe';
import { groupMealsBySlot, mealSlots } from '../utils/mealSlots';
import { recipeMinutes } from '../utils/recipeTime';
import { formatDayHeading } from '../utils/plannerDates';

interface PlannerListProps {
  dates: string[];
  items: PlannedMeal[];
  recipesById: Map<string, Recipe>;
  onEdit: (item: PlannedMeal) => void;
  onSelectDate: (date: string) => void;
}

// Every date each recipe is planned on, so a meal can point at its other days.
function datesByRecipe(items: PlannedMeal[]): Map<string, string[]> {
  const byRecipe = new Map<string, Set<string>>();
  for (const item of items) {
    const dates = byRecipe.get(item.recipe_id) ?? new Set<string>();
    dates.add(item.cook_date);
    byRecipe.set(item.recipe_id, dates);
  }
  return new Map([...byRecipe].map(([recipeId, dates]) => [recipeId, [...dates].sort()]));
}

export default function PlannerList({ dates, items, recipesById, onEdit, onSelectDate }: PlannerListProps) {
  const mealsByDay = useMemo(() => groupMealsBySlot(items), [items]);
  const plannedDates = useMemo(() => datesByRecipe(items), [items]);

  return (
    <div className="planner-list" aria-label="Day list" role="group">
      {dates.map((date) => {
        const day = mealsByDay.get(date);
        return (
          <section key={date} className="planner-list__day" aria-labelledby={`planner-day-${date}`} data-date={date}>
            <h2 id={`planner-day-${date}`} className="planner-list__day-heading">{formatDayHeading(date)}</h2>
            {mealSlots.map((group) => {
              const groupItems = day?.[group.slot] ?? [];
              if (groupItems.length === 0) return null;
              return (
                <div key={group.slot} className="planner-list__group">
                  <h3 className="planner-list__group-label">{group.label}</h3>
                  {groupItems.map((item) => {
                    const recipe = recipesById.get(item.recipe_id);
                    const others = (plannedDates.get(item.recipe_id) ?? []).filter((other) => other !== item.cook_date);
                    return (
                      <div key={item.id} className="planner-list__item">
                        <button
                          type="button"
                          className="planner-list__meal"
                          onClick={() => onEdit(item)}
                          aria-label={`Edit ${item.recipe.title} on ${formatDayHeading(date)}`}
                        >
                          {recipe?.image_url
                            ? <img className="planner-list__image" src={recipe.image_url} alt="" />
                            : <span className="planner-list__image planner-list__image--placeholder" aria-hidden="true" />}
                          <span className="planner-list__text">
                            <span className="planner-list__title">{item.recipe.title}</span>
                            {recipeMinutes(recipe) > 0 && <span className="planner-list__time">{recipeMinutes(recipe)}min</span>}
                          </span>
                        </button>
                        {others.length > 0 && (
                          <p className="planner-list__also">
                            Also planned{' '}
                            {others.map((other, index) => (
                              <span key={other}>
                                {index > 0 && ', '}
                                <button type="button" className="planner-list__also-link" onClick={() => onSelectDate(other)}>
                                  {formatDayHeading(other)}
                                </button>
                              </span>
                            ))}
                          </p>
                        )}
                      </div>
                    );
                  })}
                </div>
              );
            })}
          </section>
        );
      })}
    </div>
  );
}
