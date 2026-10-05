import PlannerDayNumber from './PlannerDayNumber';
import type { PlannedMeal } from '../types/plannedMeal';
import { parseCalendarDate, weekdayLetters } from '../utils/plannerDates';

interface PlannerCalendarProps {
  dates: string[];
  month: Date;
  items: PlannedMeal[];
  selectedDate: string;
  today: string;
  onSelectDate: (date: string) => void;
  onEdit: (item: PlannedMeal) => void;
}

export default function PlannerCalendar({ dates, month, items, selectedDate, today, onSelectDate, onEdit }: PlannerCalendarProps) {
  const byDate = new Map<string, PlannedMeal[]>();
  for (const item of items) byDate.set(item.cook_date, [...(byDate.get(item.cook_date) ?? []), item]);

  return (
    <div className="planner-calendar" aria-label="Meal calendar" role="group">
      {weekdayLetters.map((letter, index) => (
        <div key={index} className="planner-calendar__weekday" aria-hidden="true">{letter}</div>
      ))}
      {dates.map((date) => {
        const outside = parseCalendarDate(date).getMonth() !== month.getMonth();
        const classes = ['planner-calendar__day'];
        if (outside) classes.push('planner-calendar__day--outside');
        if (date === selectedDate) classes.push('planner-calendar__day--selected');
        return (
          <div key={date} className={classes.join(' ')}>
            <PlannerDayNumber date={date} today={today} selected={date === selectedDate} outside={outside} onSelect={onSelectDate} />
            {(byDate.get(date) ?? []).map((item) => (
              <button
                key={item.id}
                type="button"
                className="planner-calendar__meal"
                onClick={() => onEdit(item)}
                aria-label={`Edit ${item.recipe.title} on ${date}`}
              >
                {item.recipe.title}
              </button>
            ))}
          </div>
        );
      })}
    </div>
  );
}
