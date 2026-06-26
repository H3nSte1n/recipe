import { useState, useEffect, useRef, useMemo } from 'react';
import { Recipe, RecipeIngredient, RecipeInstruction } from '../types/recipe';
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
  onSubRecipeClick?: (recipe: Recipe) => void;
  onParentRecipeClick?: (recipe: Recipe) => void;
  usedIn?: Record<string, Recipe[]>;
}

const STOPWORDS = new Set(['or', 'and', 'the', 'a', 'an', 'of', 'with', 'in', 'to', 'for']);

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function getMatchPatterns(name: string): string[] {
  const cleaned = name
    .replace(/\s*\(.*?\)/g, '')   // strip parentheticals
    .split(/ or /i)[0]            // take part before " or "
    .replace(/[^\w\s]/g, ' ')     // strip punctuation (e.g. trailing commas in "onion, finely diced")
    .replace(/\s+/g, ' ')
    .trim();

  const words = cleaned.split(' ');
  const patterns: string[] = [];

  for (let start = 0; start < words.length; start++) {
    for (let end = words.length; end > start; end--) {
      const phrase = words.slice(start, end).join(' ');
      if (end - start === 1 && STOPWORDS.has(phrase.toLowerCase())) continue;
      patterns.push(phrase);
    }
  }

  return patterns;
}

interface IngredientPattern {
  pattern: string;
  ingredientId: string;
}

function buildIngredientPatterns(ingredients: RecipeIngredient[]): IngredientPattern[] {
  const seen = new Map<string, string>(); // lowercase pattern → first ingredient id

  for (const ing of ingredients) {
    for (const p of getMatchPatterns(ing.name)) {
      const key = p.toLowerCase();
      if (!seen.has(key)) seen.set(key, ing.id);
    }
  }

  return [...seen.entries()]
    .sort((a, b) => b[0].length - a[0].length)
    .map(([pattern, ingredientId]) => ({ pattern, ingredientId }));
}

function parseInstructionText(
  text: string,
  patterns: IngredientPattern[],
  onEnter: (id: string) => void,
  onLeave: () => void
): React.ReactNode {
  if (patterns.length === 0) return text;

  const regex = new RegExp(`\\b(${patterns.map(p => escapeRegex(p.pattern)).join('|')})\\b`, 'gi');

  const parts: React.ReactNode[] = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;
  let keyIndex = 0;

  while ((match = regex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      parts.push(text.slice(lastIndex, match.index));
    }

    const matched = match[0];
    const entry = patterns.find(p => p.pattern.toLowerCase() === matched.toLowerCase());

    if (entry) {
      parts.push(
        <span
          key={keyIndex++}
          className="recipe-modal__ingredient-ref"
          onMouseEnter={() => onEnter(entry.ingredientId)}
          onMouseLeave={onLeave}
        >
          {matched}
        </span>
      );
    } else {
      parts.push(matched);
    }

    lastIndex = regex.lastIndex;
  }

  if (lastIndex < text.length) {
    parts.push(text.slice(lastIndex));
  }

  return parts;
}

interface RecipeColumnsProps {
  ingredients: RecipeIngredient[];
  instructions: RecipeInstruction[];
  scale: number;
}

function RecipeColumns({ ingredients, instructions, scale }: RecipeColumnsProps) {
  const [hoveredIngredientId, setHoveredIngredientId] = useState<string | null>(null);

  const patterns = useMemo(() => buildIngredientPatterns(ingredients), [ingredients]);
  const sortedInstructions = useMemo(
    () => [...instructions].sort((a, b) => a.step_number - b.step_number),
    [instructions]
  );

  return (
    <div className="recipe-modal__columns">
      <div className="recipe-modal__ingredients">
        {ingredients.map(ing => (
          <div
            key={ing.id}
            className="recipe-modal__ingredient-item"
            style={{
              opacity: hoveredIngredientId !== null && hoveredIngredientId !== ing.id ? 0.25 : 1,
              transition: 'opacity 0.15s',
            }}
          >
            {ingLine(ing.amount, ing.unit, ing.name, scale)}
          </div>
        ))}
      </div>
      <div className="recipe-modal__instructions">
        {sortedInstructions.map(inst => (
          <div key={inst.id} className="recipe-modal__instruction-item">
            {parseInstructionText(
              inst.instruction,
              patterns,
              setHoveredIngredientId,
              () => setHoveredIngredientId(null)
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

export default function RecipeModal({ recipe, serves, onInc, onDec, onClose, onEdit, onSubRecipeClick, onParentRecipeClick, usedIn }: RecipeModalProps) {
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
        <div className="recipe-modal__floating-actions">
          {onEdit && (
            <button className="recipe-modal__floating-btn" type="button" aria-label="Edit recipe" onClick={onEdit}>
              <svg width={18} height={18} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                <path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z" />
              </svg>
            </button>
          )}
          <button className="recipe-modal__floating-btn" type="button" aria-label="Close" onClick={onClose}>
            <svg width={18} height={18} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
              <line x1={6} y1={6} x2={18} y2={18} />
              <line x1={18} y1={6} x2={6} y2={18} />
            </svg>
          </button>
        </div>
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
          <RecipeColumns
            ingredients={recipe.ingredients ?? []}
            instructions={recipe.instructions ?? []}
            scale={scale}
          />

          {/* Sub-recipes inline */}
          {(recipe.sub_recipes ?? []).map((sub) => {
            if (!sub.child) return null;
            const subScale = scale * (sub.serving_factor || 1);
            return (
              <div key={sub.child.id} className="recipe-modal__sub-section">
                {sub.child.status === 'published' && onSubRecipeClick ? (
                  <button
                    type="button"
                    className="recipe-modal__sub-title recipe-modal__sub-title--link"
                    onClick={() => onSubRecipeClick(sub.child!)}
                  >
                    {sub.child.title}
                    <svg className="recipe-modal__sub-chevron" width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                      <polyline points="9 18 15 12 9 6" />
                    </svg>
                  </button>
                ) : (
                  <p className="recipe-modal__sub-title">
                    {sub.child.title}
                  </p>
                )}
                <RecipeColumns
                  ingredients={sub.child.ingredients ?? []}
                  instructions={sub.child.instructions ?? []}
                  scale={subScale}
                />
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
              <p className="recipe-modal__used-in-title">Used for</p>
              <div className="recipe-modal__used-in-grid">
                {parentRecipes.map((parent) => (
                  <RecipeCard
                    key={parent.id}
                    recipe={parent}
                    onClick={onParentRecipeClick ? () => onParentRecipeClick(parent) : undefined}
                  />
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
