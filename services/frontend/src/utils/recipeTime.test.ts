import { describe, expect, it } from 'vitest';
import { recipeMinutes } from './recipeTime';
import type { Recipe } from '../types/recipe';

const recipe = (prep_time: number, cook_time?: number) => ({ prep_time, cook_time }) as Recipe;

describe('recipeMinutes', () => {
  it('adds prep and cook time', () => expect(recipeMinutes(recipe(10, 20))).toBe(30));
  it('treats a missing cook time as zero', () => expect(recipeMinutes(recipe(10))).toBe(10));
  it('is zero for an unknown recipe', () => expect(recipeMinutes(undefined)).toBe(0));
});
