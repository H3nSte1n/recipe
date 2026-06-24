import { useEffect, useRef, useState } from 'react';
import { createRecipe, updateRecipe } from '../services/recipeService';
import {
  CreateRecipeIngredientPayload,
  CreateRecipeInstructionPayload,
  CreateRecipeNutritionPayload,
  Recipe,
} from '../types/recipe';
import { parseIngText } from '../utils/formatters';
import '../styles/AddRecipeModal.css';

interface AddRecipeModalProps {
  onClose: () => void;
  onSaved: () => void;
  initialRecipe?: Recipe;
}

interface AutoResizeTextareaProps {
  className?: string;
  placeholder?: string;
  value: string;
  onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => void;
}

function AutoResizeTextarea({ className, placeholder, value, onChange }: AutoResizeTextareaProps) {
  const ref = useRef<HTMLTextAreaElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = `${el.scrollHeight}px`;
  }, [value]);
  return (
    <textarea
      ref={ref}
      className={className}
      placeholder={placeholder}
      value={value}
      onChange={onChange}
    />
  );
}

function formatIngredients(recipe: Recipe): string {
  return (recipe.ingredients ?? [])
    .map((i) => {
      if (i.amount > 0) {
        const scaled = Math.round(i.amount * 100) / 100;
        const amountStr = Number.isInteger(scaled) ? String(scaled) : scaled.toFixed(2).replace(/\.?0+$/, '');
        return i.unit ? `${amountStr}${i.unit} ${i.name}` : `${amountStr} ${i.name}`;
      }
      return i.description || i.name;
    })
    .join('\n');
}

function parseIngredients(text: string): CreateRecipeIngredientPayload[] {
  return text
    .split('\n')
    .filter((l) => l.trim())
    .map((line) => {
      const parsed = parseIngText(line.trim());
      return parsed
        ? { name: parsed.name, description: line.trim(), amount: parsed.amount, unit: parsed.unit, notes: '' }
        : { name: line.trim(), description: line.trim(), amount: 0, unit: '', notes: '' };
    });
}

function parseInstructions(text: string): CreateRecipeInstructionPayload[] {
  return text
    .split('\n')
    .filter((l) => l.trim())
    .map((line, idx) => ({ step_number: idx + 1, instruction: line.trim() }));
}

export default function AddRecipeModal({ onClose, onSaved, initialRecipe }: AddRecipeModalProps) {
  const [title, setTitle] = useState(initialRecipe?.title ?? '');
  const [description, setDescription] = useState(initialRecipe?.description ?? '');
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState(initialRecipe?.image_url ?? '');
  const [prepTime, setPrepTime] = useState(initialRecipe?.prep_time ? String(initialRecipe.prep_time) : '');
  const [cookTime, setCookTime] = useState(initialRecipe?.cook_time ? String(initialRecipe.cook_time) : '');
  const [shelfLife, setShelfLife] = useState(initialRecipe?.shelf_life ? String(initialRecipe.shelf_life) : '');
  const [servings, setServings] = useState(initialRecipe?.servings ?? 1);
  const [calories, setCalories] = useState(initialRecipe?.nutrition?.calories ? String(initialRecipe.nutrition.calories) : '');
  const [carbs, setCarbs] = useState(initialRecipe?.nutrition?.carbs ? String(initialRecipe.nutrition.carbs) : '');
  const [protein, setProtein] = useState(initialRecipe?.nutrition?.protein ? String(initialRecipe.nutrition.protein) : '');
  const [fat, setFat] = useState(initialRecipe?.nutrition?.fat ? String(initialRecipe.nutrition.fat) : '');
  const [ingredients, setIngredients] = useState(initialRecipe ? formatIngredients(initialRecipe) : '');
  const [instructions, setInstructions] = useState(
    (initialRecipe?.instructions ?? [])
      .sort((a, b) => a.step_number - b.step_number)
      .map((i) => i.instruction)
      .join('\n')
  );
  const [notes, setNotes] = useState(initialRecipe?.notes ?? '');
  const [isSaving, setIsSaving] = useState(false);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const onCloseRef = useRef(onClose);
  useEffect(() => { onCloseRef.current = onClose; }, [onClose]);
  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onCloseRef.current(); };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, []);

  function handleImageSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setImageFile(file);
    setImagePreview(URL.createObjectURL(file));
  }

  const handleSave = async () => {
    if (!title.trim() || isSaving) return;
    setIsSaving(true);

    const nutritionPayload: CreateRecipeNutritionPayload | undefined =
      calories !== '' || protein !== '' || fat !== '' || carbs !== ''
        ? { calories: parseFloat(calories) || 0, protein: parseFloat(protein) || 0, fat: parseFloat(fat) || 0, carbs: parseFloat(carbs) || 0 }
        : undefined;

    const payload = {
      title: title.trim(),
      description,
      source_type: 'MANUAL',
      servings,
      prep_time: parseInt(prepTime) || 0,
      cook_time: parseInt(cookTime) || 0,
      shelf_life: parseInt(shelfLife) || 0,
      notes,
      is_private: false,
      status: 'published',
      ingredients: parseIngredients(ingredients),
      instructions: parseInstructions(instructions),
      nutrition: nutritionPayload,
    };

    try {
      if (initialRecipe) {
        await updateRecipe(initialRecipe.id, payload, imageFile);
      } else {
        await createRecipe(payload, imageFile);
      }
      onSaved();
      onClose();
    } catch {
      setIsSaving(false);
    }
  };

  return (
    <div className="add-recipe-modal" onClick={onClose}>
      <div className="add-recipe-modal__card" onClick={(e) => e.stopPropagation()}>

        {/* ── Header ─────────────────────────────────────────── */}
        <div className="add-recipe-modal__header">
          <div
            className="add-recipe-modal__image-area"
            onClick={() => fileInputRef.current?.click()}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') fileInputRef.current?.click(); }}
            aria-label="Upload image"
          >
            {imagePreview ? (
              <img src={imagePreview} alt="preview" />
            ) : (
              <div className="add-recipe-modal__image-placeholder">
                <svg width={32} height={32} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                  <circle cx="8.5" cy="8.5" r="1.5" />
                  <polyline points="21 15 16 10 5 21" />
                </svg>
              </div>
            )}
          </div>
          <input ref={fileInputRef} type="file" accept="image/*" style={{ display: 'none' }} onChange={handleImageSelect} />

          <div className="add-recipe-modal__form-side">
            <div className="add-recipe-modal__header-actions">
              <button
                className="add-recipe-modal__publish-btn"
                type="button"
                disabled={!title.trim() || isSaving}
                onClick={handleSave}
              >
                {initialRecipe ? 'Update' : 'Publish'}
              </button>
              <button className="add-recipe-modal__action-btn" type="button" aria-label="Close" onClick={onClose}>
                <svg width={18} height={18} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                  <line x1={6} y1={6} x2={18} y2={18} />
                  <line x1={18} y1={6} x2={6} y2={18} />
                </svg>
              </button>
            </div>

            <input
              className="add-recipe-modal__title-input"
              placeholder="Recipe name"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
            <input
              className="add-recipe-modal__desc-input"
              placeholder="Add a description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />

            <div className="add-recipe-modal__nutrition-table">
              <span className="add-recipe-modal__nutrition-label">Prep time</span>
              <input className="add-recipe-modal__nutrition-input" type="number" min={0} placeholder="0" value={prepTime} onChange={(e) => setPrepTime(e.target.value)} />
              <span className="add-recipe-modal__nutrition-unit">min</span>

              <span className="add-recipe-modal__nutrition-label">Cook time</span>
              <input className="add-recipe-modal__nutrition-input" type="number" min={0} placeholder="0" value={cookTime} onChange={(e) => setCookTime(e.target.value)} />
              <span className="add-recipe-modal__nutrition-unit">min</span>

              <span className="add-recipe-modal__nutrition-label">Shelf life</span>
              <input className="add-recipe-modal__nutrition-input" type="number" min={0} placeholder="0" value={shelfLife} onChange={(e) => setShelfLife(e.target.value)} />
              <span className="add-recipe-modal__nutrition-unit">days</span>

              <span className="add-recipe-modal__nutrition-label">Calories</span>
              <input className="add-recipe-modal__nutrition-input" type="number" min={0} placeholder="—" value={calories} onChange={(e) => setCalories(e.target.value)} />
              <span className="add-recipe-modal__nutrition-unit">kcal</span>

              <span className="add-recipe-modal__nutrition-label">Carbs</span>
              <input className="add-recipe-modal__nutrition-input" type="number" min={0} placeholder="—" value={carbs} onChange={(e) => setCarbs(e.target.value)} />
              <span className="add-recipe-modal__nutrition-unit">g</span>

              <span className="add-recipe-modal__nutrition-label">Protein</span>
              <input className="add-recipe-modal__nutrition-input" type="number" min={0} placeholder="—" value={protein} onChange={(e) => setProtein(e.target.value)} />
              <span className="add-recipe-modal__nutrition-unit">g</span>

              <span className="add-recipe-modal__nutrition-label">Fat</span>
              <input className="add-recipe-modal__nutrition-input" type="number" min={0} placeholder="—" value={fat} onChange={(e) => setFat(e.target.value)} />
              <span className="add-recipe-modal__nutrition-unit">g</span>
            </div>

            <div className="add-recipe-modal__stepper">
              <span className="add-recipe-modal__serves-label">Serves</span>
              <button className="add-recipe-modal__stepper-btn" type="button" onClick={() => setServings((s) => Math.max(1, s - 1))}>−</button>
              <span className="add-recipe-modal__serves-count">{servings}</span>
              <button className="add-recipe-modal__stepper-btn" type="button" onClick={() => setServings((s) => Math.min(20, s + 1))}>+</button>
            </div>
          </div>
        </div>

        {/* ── Body ───────────────────────────────────────────── */}
        <div className="add-recipe-modal__body">
          <div className="add-recipe-modal__form-container">
            <div className="add-recipe-modal__panels">
              <div className="add-recipe-modal__panel">
                <span className="add-recipe-modal__panel-label">Ingredients</span>
                <AutoResizeTextarea
                  className="add-recipe-modal__panel-textarea"
                  placeholder={'1 cup flour\n2 eggs'}
                  value={ingredients}
                  onChange={(e) => setIngredients(e.target.value)}
                />
              </div>
              <div className="add-recipe-modal__panel">
                <span className="add-recipe-modal__panel-label">Instructions</span>
                <AutoResizeTextarea
                  className="add-recipe-modal__panel-textarea"
                  placeholder={'Combine dry ingredients\nAdd wet ingredients and mix'}
                  value={instructions}
                  onChange={(e) => setInstructions(e.target.value)}
                />
              </div>
            </div>
            <div className="add-recipe-modal__notes-panel">
              <span className="add-recipe-modal__panel-label">Notes</span>
              <AutoResizeTextarea
                className="add-recipe-modal__panel-textarea"
                placeholder="Any notes for this recipe…"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
              />
            </div>
          </div>
        </div>

        {/* ── Footer ─────────────────────────────────────────── */}
        <div className="add-recipe-modal__footer">
          <button className="add-recipe-modal__sub-recipe-btn" type="button" disabled>
            + Add sub-recipe
          </button>
        </div>

      </div>
    </div>
  );
}
