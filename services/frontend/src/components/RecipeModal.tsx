import { useEffect, useRef } from 'react';
import { Recipe } from '../types/recipe';
import { metaOf, ingLine } from '../utils/formatters';
import RecipeCard from './RecipeCard';
import '../styles/RecipeModal.css';

interface RecipeModalProps {
  recipe: Recipe;
  serves: number;
  onInc: () => void;
  onDec: () => void;
  onClose: () => void;
  onEdit?: () => void;
  usedIn?: Record<string, Recipe[]>;
}

export default function RecipeModal({ recipe, serves, onInc, onDec, onClose, onEdit, usedIn }: RecipeModalProps) {
  const onCloseRef = useRef(onClose);
  useEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCloseRef.current();
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, []);

  const scale = serves / (recipe.servings || 1);

  const parentRecipes = usedIn?.[recipe.id] ?? [];

  return (
    <div className="recipe-modal" onClick={onClose}>
      <div className="recipe-modal__card" onClick={(e) => e.stopPropagation()}>
        {/* ── Header ─────────────────────────────────────────── */}
        <div className="recipe-modal__header">
          <div className="recipe-modal__image">
            {recipe.image_url ? (
              <img src={recipe.image_url} alt={recipe.title} />
            ) : (
              <div className="recipe-modal__image-placeholder" />
            )}
          </div>
          <div className="recipe-modal__info">
            <div className="recipe-modal__actions">
              {onEdit && (
                <button className="recipe-modal__action-btn" type="button" aria-label="Edit recipe" onClick={onEdit}>
                  <svg width={18} height={18} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                    <path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z" />
                  </svg>
                </button>
              )}
              <button className="recipe-modal__action-btn" type="button" aria-label="Close" onClick={onClose}>
                <svg width={18} height={18} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                  <line x1={6} y1={6} x2={18} y2={18} />
                  <line x1={18} y1={6} x2={6} y2={18} />
                </svg>
              </button>
            </div>

            <h1 className="recipe-modal__title type-h1">{recipe.title}</h1>
            <p className="recipe-modal__cook-time">
              {metaOf(recipe.prep_time, recipe.cook_time, recipe.shelf_life)}
            </p>

            {recipe.nutrition && (
              <p className="recipe-modal__nutrition">
                {Math.round(recipe.nutrition.calories * scale)} kcal
                &nbsp;·&nbsp;{Math.round(recipe.nutrition.protein * scale)}g protein
                &nbsp;·&nbsp;{Math.round(recipe.nutrition.fat * scale)}g fat
                &nbsp;·&nbsp;{Math.round(recipe.nutrition.carbs * scale)}g carbs
              </p>
            )}

            <div className="recipe-modal__stepper">
              <span className="recipe-modal__serves-label">Serves</span>
              <button className="recipe-modal__stepper-btn" type="button" aria-label="Decrease servings" onClick={onDec}>−</button>
              <span className="recipe-modal__serves-count">{serves}</span>
              <button className="recipe-modal__stepper-btn" type="button" aria-label="Increase servings" onClick={onInc}>+</button>
            </div>
          </div>
        </div>

        {/* ── Body ───────────────────────────────────────────── */}
        <div className="recipe-modal__body">
          <div className="recipe-modal__columns">
            <div className="recipe-modal__ingredients">
              {(recipe.ingredients ?? []).map((ing) => (
                <div key={ing.id} className="recipe-modal__ingredient-item">
                  {ingLine(ing.amount, ing.unit, ing.name, scale)}
                </div>
              ))}
            </div>
            <div className="recipe-modal__instructions">
              {[...(recipe.instructions ?? [])]
                .sort((a, b) => a.step_number - b.step_number)
                .map((inst) => (
                  <div key={inst.id} className="recipe-modal__instruction-item">
                    {inst.instruction}
                  </div>
                ))}
            </div>
          </div>

          {/* Sub-recipes inline */}
          {(recipe.sub_recipes ?? []).map((sub) => {
            if (!sub.child) return null;
            return (
              <div key={sub.child.id} className="recipe-modal__sub-section">
                <p className="recipe-modal__sub-title">{sub.child.title}</p>
                <div className="recipe-modal__columns">
                  <div className="recipe-modal__ingredients">
                    {(sub.child.ingredients ?? []).map((ing) => (
                      <div key={ing.id} className="recipe-modal__ingredient-item">
                        {ingLine(ing.amount, ing.unit, ing.name, scale)}
                      </div>
                    ))}
                  </div>
                  <div className="recipe-modal__instructions">
                    {[...(sub.child.instructions ?? [])]
                      .sort((a, b) => a.step_number - b.step_number)
                      .map((inst) => (
                        <div key={inst.id} className="recipe-modal__instruction-item">
                          {inst.instruction}
                        </div>
                      ))}
                  </div>
                </div>
              </div>
            );
          })}

          {recipe.notes && (
            <div className="recipe-modal__notes-section">
              <p className="recipe-modal__notes-label">Notes</p>
              <p className="recipe-modal__notes-text">{recipe.notes}</p>
            </div>
          )}

          {parentRecipes.length > 0 && (
            <div className="recipe-modal__used-in">
              <p className="recipe-modal__used-in-title">Also used in</p>
              <div className="recipe-modal__used-in-grid">
                {parentRecipes.map((parent) => (
                  <RecipeCard key={parent.id} recipe={parent} onClick={() => undefined} />
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
