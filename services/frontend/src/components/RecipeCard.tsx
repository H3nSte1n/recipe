import { Recipe } from '../types/recipe';
import { metaOf } from '../utils/formatters';
import '../styles/RecipeCard.css';

interface RecipeCardProps {
  recipe: Recipe;
  onClick: () => void;
}

export default function RecipeCard({ recipe, onClick }: RecipeCardProps) {
  const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      onClick();
    }
  };

  return (
    <div
      className="recipe-card"
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={handleKeyDown}
    >
      <div className="recipe-card__image">
        {recipe.image_url ? (
          <img src={recipe.image_url} alt={recipe.title} />
        ) : (
          <div className="recipe-card__image-placeholder" />
        )}
      </div>
      <div className="recipe-card__title type-h3">{recipe.title}</div>
      <div className="recipe-card__meta type-body-sm">
        {metaOf(recipe.prep_time, recipe.cook_time, recipe.servings)}
      </div>
    </div>
  );
}
