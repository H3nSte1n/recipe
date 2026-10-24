import type { MealType, PlannedMeal } from '../types/plannedMeal';

export type MealSlot = 'breakfast' | 'lunch' | 'dinner' | 'snacks';
export type MealsBySlot = Record<MealSlot, PlannedMeal[]>;

export const mealSlots: { slot: MealSlot; label: string; mealType: MealType }[] = [
  { slot: 'breakfast', label: 'Breakfast', mealType: 'breakfast' },
  { slot: 'lunch', label: 'Lunch', mealType: 'lunch' },
  { slot: 'dinner', label: 'Dinner', mealType: 'dinner' },
  { slot: 'snacks', label: 'Snacks', mealType: 'snack' },
];

// Snack meals and meals without a type share the Snacks row.
export function slotOf(mealType: MealType | null): MealSlot {
  return mealType === 'breakfast' || mealType === 'lunch' || mealType === 'dinner' ? mealType : 'snacks';
}

export function groupMealsBySlot(items: PlannedMeal[]): Map<string, MealsBySlot> {
  const days = new Map<string, MealsBySlot>();
  for (const item of items) {
    let day = days.get(item.cook_date);
    if (!day) {
      day = { breakfast: [], lunch: [], dinner: [], snacks: [] };
      days.set(item.cook_date, day);
    }
    day[slotOf(item.meal_type)].push(item);
  }
  return days;
}
