import PlannerDayNumber from './PlannerDayNumber';
import { weekOf, weekdayLetters } from '../utils/plannerDates';

interface PlannerWeekStripProps {
  selectedDate: string;
  today: string;
  onSelect: (date: string) => void;
}

export default function PlannerWeekStrip({ selectedDate, today, onSelect }: PlannerWeekStripProps) {
  return (
    <div className="planner-week" role="group" aria-label="Week">
      {weekOf(selectedDate).map((date, index) => (
        <div key={date} className="planner-week__day">
          <span className="planner-week__letter" aria-hidden="true">{weekdayLetters[index]}</span>
          <PlannerDayNumber date={date} today={today} selected={date === selectedDate} onSelect={onSelect} />
        </div>
      ))}
    </div>
  );
}
