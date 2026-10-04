import type { PlannedMeal } from '../types/plannedMeal';
import { parseCalendarDate } from '../utils/plannerDates';

interface Props {
  items: PlannedMeal[];
  onEdit: (item: PlannedMeal) => void;
  onDelete: (item: PlannedMeal) => void;
}

export default function PlannerList({ items, onEdit, onDelete }: Props) {
  const dates = [...new Set(items.map((item) => item.cook_date))].sort();
  if (dates.length === 0) return <p className="planner__empty">No meals planned for this period.</p>;

  return (
    <div className="planner-list">
      {dates.map((date) => (
        <section key={date} className="planner-list__group" aria-label={date}>
          <h2>{parseCalendarDate(date).toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' })}</h2>
          {items.filter((item) => item.cook_date === date).map((item) => (
            <div key={item.id} className="planner-list__item">
              <div>
                <strong>{item.recipe.title}</strong>
                <span>{[item.meal_type, item.servings ? `${item.servings} servings` : null].filter(Boolean).join(' · ')}</span>
                {item.notes && <p>{item.notes}</p>}
              </div>
              <div className="planner-list__actions">
                <button type="button" onClick={() => onEdit(item)} aria-label={`Edit ${item.recipe.title} on ${date}`}>Edit</button>
                <button type="button" onClick={() => onDelete(item)} aria-label={`Delete ${item.recipe.title} on ${date}`}>Delete</button>
              </div>
            </div>
          ))}
        </section>
      ))}
    </div>
  );
}
