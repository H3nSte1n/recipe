import { Recipe } from '../types/recipe';
import { metaOf } from '../utils/formatters';
import '../styles/RecipeCard.css';

interface RecipeCardProps {
  recipe: Recipe;
  onClick?: () => void;
}

export default function RecipeCard({ recipe, onClick }: RecipeCardProps) {
  const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if ((event.key === 'Enter' || event.key === ' ') && onClick) {
      event.preventDefault();
      onClick();
    }
  };

  return (
    <div
      className="recipe-card"
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onClick={onClick}
      onKeyDown={handleKeyDown}
    >
      <div className="recipe-card__image-wrap">
        {recipe.image_url ? (
          <img className="recipe-card__image" src={recipe.image_url} alt={recipe.title} />
        ) : (
          <div className="recipe-card__image-placeholder" />
        )}
      </div>
      <div className="recipe-card__meta">
        <p className="recipe-card__title">{recipe.title}</p>
        <p className="recipe-card__time">{metaOf(recipe.prep_time, recipe.cook_time, recipe.shelf_life)}</p>
      </div>
    </div>
  );
}
