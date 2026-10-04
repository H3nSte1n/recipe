import type { PlannedMeal } from '../types/plannedMeal';
import { parseCalendarDate } from '../utils/plannerDates';

interface Props {
  dates: string[];
  month: Date;
  items: PlannedMeal[];
  onAdd: (date: string) => void;
  onEdit: (item: PlannedMeal) => void;
}

const weekdays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

export default function PlannerCalendar({ dates, month, items, onAdd, onEdit }: Props) {
  const byDate = new Map<string, PlannedMeal[]>();
  for (const item of items) byDate.set(item.cook_date, [...(byDate.get(item.cook_date) ?? []), item]);

  return (
    <div className="planner-calendar" aria-label="Meal calendar">
      {weekdays.map((day) => <div key={day} className="planner-calendar__weekday">{day}</div>)}
      {dates.map((date) => {
        const day = parseCalendarDate(date);
        return (
          <div key={date} className={`planner-calendar__day${day.getMonth() !== month.getMonth() ? ' planner-calendar__day--outside' : ''}`}>
            <div className="planner-calendar__day-header">
              <span aria-label={day.toLocaleDateString(undefined, { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}>{day.getDate()}</span>
              <button type="button" aria-label={`Add meal on ${date}`} onClick={() => onAdd(date)}>+</button>
            </div>
            {(byDate.get(date) ?? []).map((item) => (
              <button key={item.id} type="button" className="planner-calendar__meal" onClick={() => onEdit(item)} aria-label={`Edit ${item.recipe.title} on ${date}`}>
                {item.meal_type && <span>{item.meal_type} · </span>}{item.recipe.title}
              </button>
            ))}
          </div>
        );
      })}
    </div>
  );
}
