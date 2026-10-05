import type { MealType, PlannedMeal } from '../types/plannedMeal';
import type { Recipe } from '../types/recipe';
import { parseCalendarDate } from '../utils/plannerDates';

interface PlannerListProps {
  dates: string[];
  items: PlannedMeal[];
  recipesById: Map<string, Recipe>;
  onEdit: (item: PlannedMeal) => void;
  onSelectDate: (date: string) => void;
}

const mealGroups: { type: MealType | null; label: string }[] = [
  { type: 'breakfast', label: 'Breakfast' },
  { type: 'lunch', label: 'Lunch' },
  { type: 'dinner', label: 'Dinner' },
  { type: 'snack', label: 'Snack' },
  { type: null, label: 'Other' },
];

function formatDayHeading(date: string): string {
  return parseCalendarDate(date).toLocaleDateString(undefined, { weekday: 'long', day: 'numeric', month: 'long' });
}

export default function PlannerList({ dates, items, recipesById, onEdit, onSelectDate }: PlannerListProps) {
  function otherDatesFor(item: PlannedMeal): string[] {
    const others = items.filter((other) => other.recipe_id === item.recipe_id && other.cook_date !== item.cook_date);
    return [...new Set(others.map((other) => other.cook_date))].sort();
  }

  return (
    <div className="planner-list" aria-label="Day list" role="group">
      {dates.map((date) => {
        const dayItems = items.filter((item) => item.cook_date === date);
        return (
          <section key={date} className="planner-list__day" aria-label={date}>
            <h2 className="planner-list__day-heading">{formatDayHeading(date)}</h2>
            {mealGroups.map((group) => {
              const groupItems = dayItems.filter((item) => item.meal_type === group.type);
              if (groupItems.length === 0) return null;
              return (
                <div key={group.label} className="planner-list__group">
                  <h3 className="planner-list__group-label">{group.label}</h3>
                  {groupItems.map((item) => {
                    const recipe = recipesById.get(item.recipe_id);
                    const others = otherDatesFor(item);
                    return (
                      <div key={item.id} className="planner-list__item">
                        <button
                          type="button"
                          className="planner-list__meal"
                          onClick={() => onEdit(item)}
                          aria-label={`Edit ${item.recipe.title} on ${date}`}
                        >
                          {recipe?.image_url
                            ? <img className="planner-list__image" src={recipe.image_url} alt="" />
                            : <span className="planner-list__image planner-list__image--placeholder" aria-hidden="true" />}
                          <span className="planner-list__text">
                            <span className="planner-list__title">{item.recipe.title}</span>
                            {recipe && recipe.prep_time > 0 && <span className="planner-list__time">{recipe.prep_time}min</span>}
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
