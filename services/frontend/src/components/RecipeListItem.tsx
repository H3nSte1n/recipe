import { Recipe } from '../types/recipe';
import { metaOf } from '../utils/formatters';
import '../styles/RecipeListItem.css';

interface RecipeListItemProps {
  recipe: Recipe;
  onClick?: () => void;
}

export default function RecipeListItem({ recipe, onClick }: RecipeListItemProps) {
  const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if ((event.key === 'Enter' || event.key === ' ') && onClick) {
      event.preventDefault();
      onClick();
    }
  };

  return (
    <div
      className="recipe-list-item"
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onClick={onClick}
      onKeyDown={handleKeyDown}
    >
      <div className="recipe-list-item__image-wrap">
        {recipe.image_url ? (
          <img className="recipe-list-item__image" src={recipe.image_url} alt={recipe.title} />
        ) : (
          <div className="recipe-list-item__image-placeholder" />
        )}
      </div>
      <div className="recipe-list-item__meta">
        <p className="recipe-list-item__title">{recipe.title}</p>
        <p className="recipe-list-item__time">{metaOf(recipe.prep_time, recipe.cook_time, recipe.shelf_life)}</p>
      </div>
    </div>
  );
}
