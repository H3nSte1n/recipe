import { describe, expect, it } from 'vitest';
import type { MealType, PlannedMeal } from '../types/plannedMeal';
import { groupMealsBySlot, slotOf } from './mealSlots';

const meal = (id: string, cook_date: string, meal_type: MealType | null) => ({ id, cook_date, meal_type }) as PlannedMeal;

describe('meal slots', () => {
  it('puts snacks and untyped meals in the Snacks slot', () => {
    expect(slotOf('breakfast')).toBe('breakfast');
    expect(slotOf('dinner')).toBe('dinner');
    expect(slotOf('snack')).toBe('snacks');
    expect(slotOf(null)).toBe('snacks');
  });

  it('groups meals by day and slot, keeping same-slot duplicates in order', () => {
    const days = groupMealsBySlot([
      meal('a', '2026-10-05', 'breakfast'),
      meal('b', '2026-10-05', 'breakfast'),
      meal('c', '2026-10-05', null),
      meal('d', '2026-10-06', 'lunch'),
    ]);
    expect(days.get('2026-10-05')?.breakfast.map((item) => item.id)).toEqual(['a', 'b']);
    expect(days.get('2026-10-05')?.snacks.map((item) => item.id)).toEqual(['c']);
    expect(days.get('2026-10-06')?.lunch.map((item) => item.id)).toEqual(['d']);
    expect(days.get('2026-10-07')).toBeUndefined();
  });
});
