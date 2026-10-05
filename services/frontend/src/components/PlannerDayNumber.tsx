import { isWeekend, parseCalendarDate } from '../utils/plannerDates';

interface PlannerDayNumberProps {
  date: string;
  today: string;
  selected: boolean;
  outside?: boolean;
  onSelect: (date: string) => void;
}

export default function PlannerDayNumber({ date, today, selected, outside = false, onSelect }: PlannerDayNumberProps) {
  const day = parseCalendarDate(date);
  const classes = ['planner-day'];
  if (isWeekend(date)) classes.push('planner-day--weekend');
  if (outside) classes.push('planner-day--outside');
  if (date === today) classes.push('planner-day--today');
  if (selected) classes.push('planner-day--selected');

  return (
    <button
      type="button"
      className={classes.join(' ')}
      aria-pressed={selected}
      aria-label={day.toLocaleDateString(undefined, { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
      onClick={() => onSelect(date)}
    >
      {day.getDate()}
    </button>
  );
}
