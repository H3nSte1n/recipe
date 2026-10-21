import type { PlannedMeal } from '../types/plannedMeal';
import type { Recipe } from '../types/recipe';

interface PlannerWeekMealCardProps {
  item: PlannedMeal;
  recipe: Recipe | undefined;
  onEdit: (item: PlannedMeal) => void;
}

export default function PlannerWeekMealCard({ item, recipe, onEdit }: PlannerWeekMealCardProps) {
  const totalMinutes = recipe ? recipe.prep_time + (recipe.cook_time ?? 0) : 0;
  return (
    <button type="button" className="planner-week-card" onClick={() => onEdit(item)} aria-label={`Edit ${item.recipe.title} on ${item.cook_date}`}>
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
