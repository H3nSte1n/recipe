import React, { useEffect, useState } from 'react';
import { authApi, recipeApi, Recipe } from "../services/api";
import '../styles/Home.scss';

function formatMeta(recipe: Recipe): string {
  const parts: string[] = [];
  const totalTime = recipe.prep_time + recipe.cook_time;
  if (totalTime > 0) parts.push(`${totalTime}min`);
  if (recipe.nutrition?.calories) parts.push(`${Math.round(recipe.nutrition.calories)}kcal`);
  if (recipe.nutrition?.protein) parts.push(`${Math.round(recipe.nutrition.protein)}g Protein`);
  return parts.join(' · ');
}

type HomeProps = {
  onLogout: () => void;
};

export const Home: React.FC<HomeProps> = ({ onLogout }) => {
  const [recipes, setRecipes] = useState<Recipe[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchRecipes = async () => {
      try {
        setLoading(true);
        const data = await recipeApi.getRecipes();
        setRecipes(data);
      } catch (err) {
        console.error(err);
        if (!authApi.isAuthenticated()) {
          onLogout();
          return;
        }
        setError('Failed to load recipes. Please try again.');
      } finally {
        setLoading(false);
      }
    };

    fetchRecipes();
  }, []);

  const handleLogout = () => {
    authApi.logout();
    onLogout();
  };

  if (loading) {
    return <div className="home__loading">Loading recipes...</div>;
  }

  if (error) {
    return <div className="home__error">{error}</div>;
  }

  return (
    <div className="home">
      <button type="button" className="home__logout" onClick={handleLogout}>
        Logout
      </button>

      {recipes.length === 0 && (
        <p className="home__empty">No recipes found for this account.</p>
      )}

      <div className="recipe-grid">
        {recipes.map((recipe) => (
          <div key={recipe.id} className="recipe-card">
            <div className="recipe-card__image">
              <img
                src={recipe.image_url || 'https://images.unsplash.com/photo-1546069901-ba9599a7e63c?w=800'}
                alt={recipe.title}
              />
            </div>
            <div className="recipe-card__content">
              <h2 className="recipe-card__title">{recipe.title}</h2>
              <p className="recipe-card__meta">{formatMeta(recipe)}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};