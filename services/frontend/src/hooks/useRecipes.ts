import { useState, useEffect, useCallback } from 'react';
import { Recipe } from '../types/recipe';
import { getMyRecipes } from '../services/recipeService';

interface UseRecipesReturn {
  recipes: Recipe[];
  isLoading: boolean;
  error: string | null;
  filterRecipes: (query: string) => Recipe[];
  refresh: () => void;
}

export function useRecipes(): UseRecipesReturn {
  const [recipes, setRecipes] = useState<Recipe[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;

    const fetchRecipes = async (): Promise<void> => {
      try {
        setIsLoading(true);
        setError(null);
        const data = await getMyRecipes();
        if (!cancelled) {
          setRecipes(data.filter((r) => r.status === 'published'));
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load recipes');
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    fetchRecipes();

    return () => {
      cancelled = true;
    };
  }, [refreshKey]);

  const filterRecipes = useCallback((query: string): Recipe[] => {
    if (!query.trim()) {
      return recipes;
    }
    const lowerQuery = query.toLowerCase();
    return recipes.filter((recipe) => recipe.title.toLowerCase().includes(lowerQuery));
  }, [recipes]);

  const refresh = useCallback(() => setRefreshKey((k) => k + 1), []);

  return { recipes, isLoading, error, filterRecipes, refresh };
}
