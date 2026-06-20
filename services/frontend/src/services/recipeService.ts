import { Recipe } from '../types/recipe';
import { getAuthHeaders } from './authService';
import { apiFetch } from '../api/apiClient';

export async function getMyRecipes(): Promise<Recipe[]> {
  try {
    const response = await apiFetch('/api/v1/recipes', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch recipes: ${response.status} ${response.statusText}`);
    }

    const data: Recipe[] = await response.json();
    return data;
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error('An unexpected error occurred while fetching recipes');
  }
}

export async function getRecipeById(id: string): Promise<Recipe> {
  try {
    const response = await apiFetch(`/api/v1/recipes/${id}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch recipe: ${response.status} ${response.statusText}`);
    }

    const data: Recipe = await response.json();
    return data;
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error('An unexpected error occurred while fetching the recipe');
  }
}
