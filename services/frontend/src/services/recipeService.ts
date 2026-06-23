import { Recipe, CreateRecipePayload } from '../types/recipe';
import { getAuthHeaders } from './authService';
import { apiFetch } from '../api/apiClient';

export async function createRecipe(payload: CreateRecipePayload, imageFile?: File | null): Promise<Recipe> {
  try {
    let response: Response;
    if (imageFile) {
      const formData = new FormData();
      formData.append('recipe', JSON.stringify(payload));
      formData.append('image', imageFile);
      response = await apiFetch('/api/v1/recipes', {
        method: 'POST',
        headers: getAuthHeaders(),
        body: formData,
      });
    } else {
      response = await apiFetch('/api/v1/recipes', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders(),
        },
        body: JSON.stringify(payload),
      });
    }
    if (!response.ok) {
      throw new Error(`Failed to create recipe: ${response.status} ${response.statusText}`);
    }
    const data: Recipe = await response.json();
    return data;
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error('An unexpected error occurred while creating the recipe');
  }
}

export async function updateRecipe(id: string, payload: CreateRecipePayload, imageFile?: File | null): Promise<Recipe> {
  try {
    let response: Response;
    if (imageFile) {
      const formData = new FormData();
      formData.append('recipe', JSON.stringify(payload));
      formData.append('image', imageFile);
      response = await apiFetch(`/api/v1/recipes/${id}`, {
        method: 'PUT',
        headers: getAuthHeaders(),
        body: formData,
      });
    } else {
      response = await apiFetch(`/api/v1/recipes/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          ...getAuthHeaders(),
        },
        body: JSON.stringify(payload),
      });
    }
    if (!response.ok) {
      throw new Error(`Failed to update recipe: ${response.status} ${response.statusText}`);
    }
    const data: Recipe = await response.json();
    return data;
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error('An unexpected error occurred while updating the recipe');
  }
}

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
