import type { PlannedMeal } from '../types/plannedMeal';
import { formatDayHeading } from '../utils/plannerDates';
import type { Recipe } from '../types/recipe';
import { recipeMinutes } from '../utils/recipeTime';

interface PlannerWeekMealCardProps {
  item: PlannedMeal;
  recipe: Recipe | undefined;
  onEdit: (item: PlannedMeal) => void;
}

export default function PlannerWeekMealCard({ item, recipe, onEdit }: PlannerWeekMealCardProps) {
  const totalMinutes = recipeMinutes(recipe);
  return (
    <button type="button" className="planner-week-card" onClick={() => onEdit(item)} aria-label={`Edit ${item.recipe.title} on ${formatDayHeading(item.cook_date)}`}>
      {recipe?.image_url
        ? <img className="planner-week-card__image" src={recipe.image_url} alt="" />
        : <span className="planner-week-card__image planner-week-card__image--placeholder" aria-hidden="true" />}
      <span className="planner-week-card__text">
        <span className="planner-week-card__title">{item.recipe.title}</span>
        {totalMinutes > 0 && <span className="planner-week-card__time">{totalMinutes}min</span>}
      </span>
    </button>
  );
}
