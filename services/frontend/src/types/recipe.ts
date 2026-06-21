export interface CreateRecipeIngredientPayload {
  name: string;
  description: string;
  amount: number;
  unit: string;
  notes: string;
}

export interface CreateRecipeInstructionPayload {
  step_number: number;
  instruction: string;
}

export interface CreateRecipeNutritionPayload {
  calories: number;
  protein: number;
  fat: number;
}

export interface SubRecipePayload {
  recipe_id: string;
  serving_factor: number;
}

export interface CreateRecipePayload {
  title: string;
  description: string;
  source_type: string;
  servings: number;
  prep_time: number;
  notes: string;
  is_private: boolean;
  status: string;
  ingredients: CreateRecipeIngredientPayload[];
  instructions: CreateRecipeInstructionPayload[];
  nutrition?: CreateRecipeNutritionPayload;
  sub_recipes?: SubRecipePayload[];
}

export interface RecipeIngredient {
  id: string;
  recipe_id: string;
  name: string;
  description: string;
  amount: number;
  unit: string;
  notes: string;
}

export interface RecipeInstruction {
  id: string;
  recipe_id: string;
  step_number: number;
  instruction: string;
}

export interface SubRecipe {
  id: string;
  parent_id: string;
  child_id: string;
  serving_factor: number;
  child?: Recipe;
}

export interface Recipe {
  id: string;
  user_id: string;
  title: string;
  description: string;
  notes: string;
  rating: number;
  image_url?: string;
  source_type: string;
  source?: string;
  is_private: boolean;
  servings: number;
  prep_time: number;
  cook_time: number;
  shelf_life: number;
  status: string;
  created_at: string;
  updated_at: string;
  ingredients?: RecipeIngredient[];
  instructions?: RecipeInstruction[];
  sub_recipes?: SubRecipe[];
}
