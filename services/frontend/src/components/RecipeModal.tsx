import { useEffect, useRef, useState } from 'react';
import { Recipe } from '../types/recipe';
import { metaOf, ingLine } from '../utils/formatters';
import { getRecipeById } from '../services/recipeService';
import '../styles/RecipeModal.css';

interface RecipeModalProps {
  recipe: Recipe;
  serves: number;
  onInc: () => void;
  onDec: () => void;
  onClose: () => void;
  usedIn?: Record<string, Recipe[]>;
}

interface ModalSection {
  name: string;
  ingredients: Recipe['ingredients'];
  instructions: Recipe['instructions'];
  childId?: string;
}

export default function RecipeModal({ recipe, serves, onInc, onDec, onClose, usedIn }: RecipeModalProps) {
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

  const [navStack, setNavStack] = useState<Recipe[]>([recipe]);
  const [navLoading, setNavLoading] = useState(false);

  const currentRecipe = navStack[navStack.length - 1];

  const navigateToSub = async (childId: string) => {
    if (navLoading) return;
    const existingIndex = navStack.findIndex((r) => r.id === childId);
    if (existingIndex !== -1) {
      setNavStack((prev) => prev.slice(0, existingIndex + 1));
      return;
    }
    setNavLoading(true);
    try {
      const full = await getRecipeById(childId);
      setNavStack((prev) => [...prev, full]);
    } finally {
      setNavLoading(false);
    }
  };

  const sections: ModalSection[] = [
    {
      name: currentRecipe.title,
      ingredients: currentRecipe.ingredients ?? [],
      instructions: currentRecipe.instructions ?? [],
    },
  ];

  if (currentRecipe.sub_recipes) {
    for (const sub of currentRecipe.sub_recipes) {
      if (sub.child) {
        sections.push({
          name: sub.child.title,
          ingredients: sub.child.ingredients ?? [],
          instructions: sub.child.instructions ?? [],
          childId: sub.child.id,
        });
      }
    }
  }

  return (
    <div className="recipe-modal" onClick={onClose}>
      <div className="recipe-modal__card" onClick={(e) => e.stopPropagation()}>
        <div className="recipe-modal__hero">
          {currentRecipe.image_url ? (
            <img src={currentRecipe.image_url} alt={currentRecipe.title} />
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
          {navStack.length > 1 && (
            <div className="recipe-modal__breadcrumb">
              {navStack.map((r, i) => (
                <span key={r.id} style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                  {i > 0 && <span className="recipe-modal__breadcrumb-sep type-caption">›</span>}
                  {i < navStack.length - 1 ? (
                    <button
                      className="recipe-modal__breadcrumb-item type-body-sm"
                      type="button"
                      onClick={() => setNavStack((prev) => prev.slice(0, i + 1))}
                    >
                      {r.title}
                    </button>
                  ) : (
                    <span className="recipe-modal__breadcrumb-item type-body-sm recipe-modal__breadcrumb-item--active">
                      {r.title}
                    </span>
                  )}
                </span>
              ))}
            </div>
          )}
        </div>

        <div className="recipe-modal__content">
          <h1 className="recipe-modal__title type-h1">{currentRecipe.title}</h1>
          <div className="recipe-modal__meta type-body">
            {metaOf(currentRecipe.prep_time, currentRecipe.cook_time, 0)}
          </div>

          <div className="recipe-modal__serves">
            <span className="recipe-modal__serves-label type-label">Serves</span>
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
              {section.childId ? (
                <button
                  className="recipe-modal__section-name type-h3 recipe-modal__section-name--link"
                  type="button"
                  onClick={() => void navigateToSub(section.childId!)}
                  disabled={navLoading}
                >
                  {section.name}
                  <span className="recipe-modal__section-chevron">›</span>
                </button>
              ) : (
                <div className="recipe-modal__section-name type-h3">{section.name}</div>
              )}
              <div className="recipe-modal__columns">
                <div className="recipe-modal__ingredients">
                  {(section.ingredients ?? []).map((ing) => (
                    <div key={ing.id} className="recipe-modal__ingredient type-body">
                      {ingLine(ing.amount, ing.unit, ing.name, serves / (recipe.servings || 1))}
                    </div>
                  ))}
                </div>
                <div className="recipe-modal__steps">
                  {[...(section.instructions ?? [])]
                    .sort((a, b) => a.step_number - b.step_number)
                    .map((inst) => (
                      <div key={inst.id} className="recipe-modal__step type-body">
                        {inst.instruction}
                      </div>
                    ))}
                </div>
              </div>
            </div>
          ))}
          {usedIn?.[currentRecipe.id]?.length ? (
            <div className="recipe-modal__used-in">
              <div className="recipe-modal__used-in-label type-label">Used in</div>
              <div className="recipe-modal__used-in-strip">
                {usedIn[currentRecipe.id].map((parent) => (
                  <button
                    key={parent.id}
                    className="recipe-modal__used-in-card"
                    type="button"
                    disabled={navLoading}
                    onClick={() => void navigateToSub(parent.id)}
                  >
                    {parent.image_url ? (
                      <img
                        className="recipe-modal__used-in-card-image"
                        src={parent.image_url}
                        alt={parent.title}
                      />
                    ) : (
                      <div className="recipe-modal__used-in-card-image" />
                    )}
                    <div className="recipe-modal__used-in-card-title">{parent.title}</div>
                  </button>
                ))}
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
