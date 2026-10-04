export type MealType = 'breakfast' | 'lunch' | 'dinner' | 'snack';

export interface PlannedMealInput {
  recipe_id: string;
  cook_date: string;
  meal_type: MealType | null;
  servings: number | null;
  notes: string | null;
}

export interface PlannedMeal extends PlannedMealInput {
  id: string;
  recipe: { id: string; title: string; status: string };
  created_at: string;
  updated_at: string;
}

export interface PlannedMealPage {
  items: PlannedMeal[];
  total: number;
  page: number;
  page_size: number;
}
