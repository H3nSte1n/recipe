import type { Recipe } from '../types/recipe';

// Total minutes a recipe takes (prep plus cook), shown the same way in every planner view.
export function recipeMinutes(recipe: Recipe | undefined): number {
  return recipe ? recipe.prep_time + (recipe.cook_time ?? 0) : 0;
}
