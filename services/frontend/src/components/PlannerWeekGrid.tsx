import PlannerWeekMealCard from './PlannerWeekMealCard';
import type { MealType, PlannedMeal } from '../types/plannedMeal';
import type { Recipe } from '../types/recipe';
import { groupMealsBySlot, mealSlots, slotOf } from '../utils/mealSlots';
import { formatDayHeading, parseCalendarDate } from '../utils/plannerDates';

interface PlannerWeekGridProps {
  dates: string[];
  items: PlannedMeal[];
  recipesById: Map<string, Recipe>;
  today: string;
  onEdit: (item: PlannedMeal) => void;
  onAdd: (date: string, mealType: MealType) => void;
}

function formatColumnHeading(date: string): string {
  return parseCalendarDate(date).toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric' });
}

export default function PlannerWeekGrid({ dates, items, recipesById, today, onEdit, onAdd }: PlannerWeekGridProps) {
  const mealsByDay = groupMealsBySlot(items);
  const hasSnacks = items.some((item) => slotOf(item.meal_type) === 'snacks');
  const visibleSlots = mealSlots.filter((row) => row.slot !== 'snacks' || hasSnacks);

  return (
    <div className="planner-week-grid" role="group" aria-label="Week plan">
      <div className="planner-week-grid__body">
        {/* One set of column lines behind the header and every row, so they run unbroken. */}
        <div className="planner-week-grid__lines" aria-hidden="true">
          {dates.map((date) => <span key={date} />)}
        </div>
        <div className="planner-week-grid__heads">
          {dates.map((date) => (
            <span key={date} className={`planner-week-grid__head${date === today ? ' planner-week-grid__head--today' : ''}`}>{formatColumnHeading(date)}</span>
          ))}
        </div>
        {visibleSlots.map((row) => (
          <section key={row.slot} className="planner-week-grid__row" aria-label={row.label}>
            <h2 className="planner-week-grid__slot">{row.label}</h2>
            <div className="planner-week-grid__cells">
              {dates.map((date) => (
                <div key={date} className="planner-week-grid__cell">
                  {(mealsByDay.get(date)?.[row.slot] ?? []).map((item) => (
                    <PlannerWeekMealCard key={item.id} item={item} recipe={recipesById.get(item.recipe_id)} onEdit={onEdit} />
                  ))}
                  <button type="button" className="planner-week-grid__add" aria-label={`Add ${row.label} on ${formatDayHeading(date)}`} onClick={() => onAdd(date, row.mealType)}>
                    <span aria-hidden="true">+</span>
                  </button>
                </div>
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}
