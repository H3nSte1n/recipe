import { useEffect, useRef } from 'react';
import { Recipe } from '../types/recipe';
import { metaOf, ingLine } from '../utils/formatters';
import '../styles/RecipeModal.css';

interface RecipeModalProps {
  recipe: Recipe;
  serves: number;
  onInc: () => void;
  onDec: () => void;
  onClose: () => void;
}

interface ModalSection {
  name: string;
  ingredients: Recipe['ingredients'];
  instructions: Recipe['instructions'];
}

export default function RecipeModal({ recipe, serves, onInc, onDec, onClose }: RecipeModalProps) {
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

  const sections: ModalSection[] = [
    {
      name: recipe.title,
      ingredients: recipe.ingredients ?? [],
      instructions: recipe.instructions ?? [],
    },
  ];

  if (recipe.sub_recipes) {
    for (const sub of recipe.sub_recipes) {
      if (sub.child) {
        sections.push({
          name: sub.child.title,
          ingredients: sub.child.ingredients ?? [],
          instructions: sub.child.instructions ?? [],
        });
      }
    }
  }

  return (
    <div className="recipe-modal" onClick={onClose}>
      <div className="recipe-modal__card" onClick={(e) => e.stopPropagation()}>
        <div className="recipe-modal__hero">
          {recipe.image_url ? (
            <img src={recipe.image_url} alt={recipe.title} />
          ) : (
            <div className="recipe-modal__hero-placeholder" />
          )}
          <div className="recipe-modal__controls">
            <button className="recipe-modal__control-btn" type="button" aria-label="Edit recipe">
              <svg
                width={18}
                height={18}
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth={1.5}
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z" />
              </svg>
            </button>
            <button
              className="recipe-modal__control-btn"
              type="button"
              aria-label="Close"
              onClick={onClose}
            >
              <svg
                width={18}
                height={18}
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth={1.5}
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <line x1={6} y1={6} x2={18} y2={18} />
                <line x1={18} y1={6} x2={6} y2={18} />
              </svg>
            </button>
          </div>
        </div>

        <div className="recipe-modal__content">
          <h1 className="recipe-modal__title">{recipe.title}</h1>
          <div className="recipe-modal__meta">
            {metaOf(recipe.prep_time, recipe.cook_time, recipe.servings)}
          </div>

          <div className="recipe-modal__serves">
            <span className="recipe-modal__serves-label">Serves</span>
            <button
              className="recipe-modal__serves-btn"
              type="button"
              aria-label="Decrease servings"
              onClick={onDec}
            >
              <svg
                width={18}
                height={18}
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth={1.5}
                strokeLinecap="round"
              >
                <line x1={5} y1={12} x2={19} y2={12} />
              </svg>
            </button>
            <span className="recipe-modal__serves-count">{serves}</span>
            <button
              className="recipe-modal__serves-btn"
              type="button"
              aria-label="Increase servings"
              onClick={onInc}
            >
              <svg
                width={18}
                height={18}
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth={1.5}
                strokeLinecap="round"
              >
                <line x1={5} y1={12} x2={19} y2={12} />
                <line x1={12} y1={5} x2={12} y2={19} />
              </svg>
            </button>
          </div>

          {sections.map((section, i) => (
            <div key={i} className="recipe-modal__section">
              <div className="recipe-modal__section-name">{section.name}</div>
              <div className="recipe-modal__columns">
                <div className="recipe-modal__ingredients">
                  {(section.ingredients ?? []).map((ing) => (
                    <div key={ing.id} className="recipe-modal__ingredient">
                      {ingLine(ing.amount, ing.unit, ing.name, serves / 2)}
                    </div>
                  ))}
                </div>
                <div className="recipe-modal__steps">
                  {[...(section.instructions ?? [])]
                    .sort((a, b) => a.step_number - b.step_number)
                    .map((inst) => (
                      <div key={inst.id} className="recipe-modal__step">
                        <span className="recipe-modal__step-num">{inst.step_number}.</span>
                        <span>{inst.instruction}</span>
                      </div>
                    ))}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
