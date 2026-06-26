import { useEffect, useRef, useState } from 'react';
import { createRecipe, deleteRecipe, getMyRecipes, getRecipeById, updateRecipe } from '../services/recipeService';
import {
  CreateRecipeIngredientPayload,
  CreateRecipeInstructionPayload,
  CreateRecipeNutritionPayload,
  Recipe,
  RecipeIngredient,
  SubRecipePayload,
} from '../types/recipe';
import { parseIngText } from '../utils/formatters';
import '../styles/AddRecipeModal.css';

interface AddRecipeModalProps {
  onClose: () => void;
  onSaved: () => void;
  onDeleted?: () => void;
  initialRecipe?: Recipe;
}

interface SubSection {
  id: string;
  name: string;
  portion: number;
  ingredients: string;
  instructions: string;
  notes: string;
  linkedRecipeId?: string;
  linkedIngredients?: RecipeIngredient[];
}

interface AutoResizeTextareaProps {
  className?: string;
  placeholder?: string;
  value: string;
  onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => void;
  disabled?: boolean;
}

function AutoResizeTextarea({ className, placeholder, value, onChange, disabled }: AutoResizeTextareaProps) {
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
      disabled={disabled}
    />
  );
}

function formatRecipeIngredients(ings: RecipeIngredient[], factor: number): string {
  return ings
    .map((i) => {
      let amt = i.amount;
      let unit = i.unit;
      let name = i.name;
      if (amt === 0) {
        const parsed = parseIngText(i.description || i.name);
        if (parsed) { amt = parsed.amount; unit = parsed.unit; name = parsed.name; }
      }
      if (amt > 0) {
        const scaled = Math.round(amt * factor * 100) / 100;
        const amountStr = Number.isInteger(scaled) ? String(scaled) : scaled.toFixed(2).replace(/\.?0+$/, '');
        return unit ? `${amountStr}${unit} ${name}` : `${amountStr} ${name}`;
      }
      return i.description || i.name;
    })
    .join('\n');
}

function formatRecipe(recipe: Recipe): string {
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

function buildInitialSubSections(recipe: Recipe): SubSection[] {
  const subs = recipe.sub_recipes ?? [];
  return subs.map((sr, i) => ({
    id: String(Date.now() + i),
    name: sr.child?.title ?? '',
    portion: sr.serving_factor,
    linkedRecipeId: sr.child_id,
    linkedIngredients: sr.child?.ingredients ?? [],
    ingredients: formatRecipeIngredients(sr.child?.ingredients ?? [], sr.serving_factor),
    instructions: (sr.child?.instructions ?? [])
      .sort((a, b) => a.step_number - b.step_number)
      .map((ins) => ins.instruction)
      .join('\n'),
    notes: '',
  }));
}

// ── SubRecipeCard ───────────────────────────────────────────────────────────

interface SubRecipeCardProps {
  sub: SubSection;
  allRecipes: Recipe[];
  onDelete: () => void;
  onChange: (field: keyof Pick<SubSection, 'name' | 'ingredients' | 'instructions' | 'notes'>, value: string) => void;
  onLink: (recipe: Recipe) => void;
  onUnlink: () => void;
  onPortionChange: (delta: number) => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  canMoveUp: boolean;
  canMoveDown: boolean;
}

function SubRecipeCard({ sub, allRecipes, onDelete, onChange, onLink, onUnlink, onPortionChange, onMoveUp, onMoveDown, canMoveUp, canMoveDown }: SubRecipeCardProps) {
  const [dropdownOpen, setDropdownOpen] = useState(false);

  const suggestions = sub.name.length > 0
    ? allRecipes.filter((r) => r.title.toLowerCase().includes(sub.name.toLowerCase())).slice(0, 5)
    : [];

  return (
    <div className="sub-recipe-card">
      {/* Card header */}
      <div className="sub-recipe-card__header">
        <div className="sub-recipe-card__reorder">
          <button className="sub-recipe-card__reorder-btn" type="button" aria-label="Move up" onClick={onMoveUp} disabled={!canMoveUp}>▲</button>
          <button className="sub-recipe-card__reorder-btn" type="button" aria-label="Move down" onClick={onMoveDown} disabled={!canMoveDown}>▼</button>
        </div>

        {sub.linkedRecipeId ? (
          <div className="sub-recipe-card__name-linked">
            <span className="sub-recipe-card__name-text">{sub.name}</span>
            <button className="sub-recipe-card__unlink" type="button" onClick={onUnlink} aria-label="Unlink">×</button>
          </div>
        ) : (
          <div className="sub-recipe-card__name-search">
            <input
              className="sub-recipe-card__name-input"
              placeholder="Give me a name (e.g. Salmon)"
              value={sub.name}
              onChange={(e) => {
                onChange('name', e.target.value);
                setDropdownOpen(true);
              }}
              onFocus={() => { if (sub.name.length > 0) setDropdownOpen(true); }}
              onBlur={() => setTimeout(() => setDropdownOpen(false), 150)}
            />
            {dropdownOpen && suggestions.length > 0 && (
              <div className="sub-recipe-card__dropdown">
                {suggestions.map((r) => (
                  <button
                    key={r.id}
                    className="sub-recipe-card__dropdown-item"
                    type="button"
                    onMouseDown={() => onLink(r)}
                  >
                    {r.title}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        <div className="sub-recipe-card__right">
          {sub.linkedRecipeId && (
            <div className="sub-recipe-card__portion">
              <span className="sub-recipe-card__portion-label">Portion</span>
              <button className="sub-recipe-card__portion-btn" type="button" onClick={() => onPortionChange(-1)}>−</button>
              <span className="sub-recipe-card__portion-count">{sub.portion}</span>
              <button className="sub-recipe-card__portion-btn" type="button" onClick={() => onPortionChange(1)}>+</button>
            </div>
          )}
          <button className="sub-recipe-card__delete" type="button" aria-label="Delete sub-recipe" onClick={onDelete}>
            <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
              <polyline points="3 6 5 6 21 6" />
              <path d="M19 6l-1 14H6L5 6" />
              <path d="M10 11v6M14 11v6M9 6V4h6v2" />
            </svg>
          </button>
        </div>
      </div>

      {/* Panels */}
      <div className="sub-recipe-card__panels">
        <div className="add-recipe-modal__panel">
          <span className="add-recipe-modal__panel-label">Ingredients</span>
          <AutoResizeTextarea
            className="add-recipe-modal__panel-textarea"
            placeholder={'1 cup flour\n2 eggs'}
            value={sub.ingredients}
            disabled={!!sub.linkedRecipeId}
            onChange={(e) => onChange('ingredients', e.target.value)}
          />
        </div>
        <div className="add-recipe-modal__panel">
          <span className="add-recipe-modal__panel-label">Instructions</span>
          <AutoResizeTextarea
            className="add-recipe-modal__panel-textarea"
            placeholder={'Step 1\nStep 2'}
            value={sub.instructions}
            disabled={!!sub.linkedRecipeId}
            onChange={(e) => onChange('instructions', e.target.value)}
          />
        </div>
      </div>
      <div className="add-recipe-modal__notes-panel">
        <span className="add-recipe-modal__panel-label">Notes</span>
        <AutoResizeTextarea
          className="add-recipe-modal__panel-textarea"
          placeholder="Any notes…"
          value={sub.notes}
          onChange={(e) => onChange('notes', e.target.value)}
        />
      </div>
    </div>
  );
}

// ── AddRecipeModal ──────────────────────────────────────────────────────────

export default function AddRecipeModal({ onClose, onSaved, onDeleted, initialRecipe }: AddRecipeModalProps) {
  const [title, setTitle] = useState(initialRecipe?.title ?? '');
  const [description, setDescription] = useState(initialRecipe?.description ?? '');
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState(initialRecipe?.image_url ?? '');
  const [prepTime, setPrepTime] = useState(initialRecipe?.prep_time ? String(initialRecipe.prep_time) : '');
  const [cookTime, setCookTime] = useState(initialRecipe?.cook_time ? String(initialRecipe.cook_time) : '');
  const [shelfLife, setShelfLife] = useState(initialRecipe?.shelf_life ? String(initialRecipe.shelf_life) : '');
  const [servings, setServings] = useState(initialRecipe?.servings ?? 1);
  const initNutrition = initialRecipe?.nutrition;
  const [calories, setCalories] = useState(initNutrition != null ? String(initNutrition.calories) : '');
  const [carbs, setCarbs] = useState(initNutrition != null ? String(initNutrition.carbs) : '');
  const [protein, setProtein] = useState(initNutrition != null ? String(initNutrition.protein) : '');
  const [fat, setFat] = useState(initNutrition != null ? String(initNutrition.fat) : '');
  const hasSubRecipes = initialRecipe != null && (initialRecipe.sub_recipes?.length ?? 0) >= 1;
  const [ingredients, setIngredients] = useState(
    hasSubRecipes ? '' : (initialRecipe ? formatRecipe(initialRecipe) : '')
  );
  const [instructions, setInstructions] = useState(
    hasSubRecipes
      ? ''
      : (initialRecipe?.instructions ?? [])
          .sort((a, b) => a.step_number - b.step_number)
          .map((i) => i.instruction)
          .join('\n')
  );
  const [notes, setNotes] = useState(initialRecipe?.notes ?? '');
  const [subSections, setSubSections] = useState<SubSection[]>(() =>
    hasSubRecipes ? buildInitialSubSections(initialRecipe!) : []
  );
  const subSectionsRef = useRef<SubSection[]>(subSections);
  useEffect(() => { subSectionsRef.current = subSections; }, [subSections]);
  const [allRecipes, setAllRecipes] = useState<Recipe[]>([]);
  const [isSaving, setIsSaving] = useState(false);

  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    getMyRecipes().then(setAllRecipes).catch(() => {});
  }, []);

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

  async function handleDelete() {
    if (!initialRecipe) return;
    if (!window.confirm('Delete this recipe? This cannot be undone.')) return;
    await deleteRecipe(initialRecipe.id);
    onDeleted?.();
    onClose();
  }

  function handleAddSubSection() {
    if (subSections.length === 0) {
      // Transition from single-block to multi-block mode:
      // wrap current top-level content as subSections[0], add empty subSections[1]
      const existingBlock: SubSection = {
        id: crypto.randomUUID(),
        name: '',
        portion: 1,
        ingredients,
        instructions,
        notes,
      };
      const emptyBlock: SubSection = {
        id: crypto.randomUUID(),
        name: '',
        portion: 1,
        ingredients: '',
        instructions: '',
        notes: '',
      };
      setSubSections([existingBlock, emptyBlock]);
      setIngredients('');
      setInstructions('');
      setNotes('');
    } else {
      // Already in multi-block mode: just append a new empty block
      setSubSections((prev) => [
        ...prev,
        { id: crypto.randomUUID(), name: '', portion: 1, ingredients: '', instructions: '', notes: '' },
      ]);
    }
  }

  function handleDeleteSubSection(id: string) {
    const current = subSectionsRef.current;
    const filtered = current.filter((s) => s.id !== id);
    if (filtered.length === 1) {
      // Transition from multi-block back to single-block mode:
      // restore surviving block's content into top-level state
      const survivor = filtered[0];
      setIngredients(survivor.ingredients);
      setInstructions(survivor.instructions);
      setNotes(survivor.notes);
      setSubSections([]);
    } else if (filtered.length === 0) {
      // Deleting the last block: restore its content to top-level state
      const deleted = current.find((s) => s.id === id);
      if (deleted) {
        setIngredients(deleted.ingredients);
        setInstructions(deleted.instructions);
        setNotes(deleted.notes);
      }
      setSubSections([]);
    } else {
      setSubSections(filtered);
    }
  }

  function handleSubChange(id: string, field: keyof Pick<SubSection, 'name' | 'ingredients' | 'instructions' | 'notes'>, value: string) {
    setSubSections((prev) => prev.map((s) => s.id === id ? { ...s, [field]: value } : s));
  }

  async function handleLink(id: string, recipe: Recipe) {
    setSubSections((prev) =>
      prev.map((s) => s.id === id
        ? { ...s, name: recipe.title, linkedRecipeId: recipe.id, ingredients: '', instructions: '' }
        : s)
    );
    try {
      const full = await getRecipeById(recipe.id);
      const linkedIngredients = full.ingredients ?? [];
      setSubSections((prev) =>
        prev.map((s) => {
          if (s.id !== id || s.linkedRecipeId !== recipe.id) return s;
          return {
            ...s,
            linkedIngredients,
            ingredients: formatRecipeIngredients(linkedIngredients, s.portion),
            instructions: (full.instructions ?? [])
              .sort((a, b) => a.step_number - b.step_number)
              .map((i) => i.instruction)
              .join('\n'),
          };
        })
      );
    } catch {
      // keep the name link even if fetching details fails
    }
  }

  function handleUnlink(id: string) {
    setSubSections((prev) =>
      prev.map((s) => s.id === id ? { ...s, linkedRecipeId: undefined, linkedIngredients: undefined } : s)
    );
  }

  function handleMoveSubSection(id: string, dir: 'up' | 'down') {
    setSubSections((prev) => {
      const idx = prev.findIndex((s) => s.id === id);
      if (idx < 0) return prev;
      const next = [...prev];
      const swapIdx = dir === 'up' ? idx - 1 : idx + 1;
      if (swapIdx < 0 || swapIdx >= next.length) return prev;
      [next[idx], next[swapIdx]] = [next[swapIdx], next[idx]];
      return next;
    });
  }

  function handlePortionChange(id: string, delta: number) {
    setSubSections((prev) =>
      prev.map((s) => {
        if (s.id !== id) return s;
        const next = Math.min(99, Math.max(0.5, Math.round((s.portion + delta) * 2) / 2));
        return {
          ...s,
          portion: next,
          ingredients: s.linkedIngredients
            ? formatRecipeIngredients(s.linkedIngredients, next)
            : s.ingredients,
        };
      })
    );
  }

  const handleSave = async () => {
    if (!title.trim() || isSaving) return;
    setIsSaving(true);

    const nutritionPayload: CreateRecipeNutritionPayload | undefined =
      calories !== '' || protein !== '' || fat !== '' || carbs !== ''
        ? { calories: parseFloat(calories) || 0, protein: parseFloat(protein) || 0, fat: parseFloat(fat) || 0, carbs: parseFloat(carbs) || 0 }
        : undefined;

    try {
      // Build sub-recipe payloads (create new ones if not linked)
      const subRecipePayloads: SubRecipePayload[] = [];
      for (const sub of subSections) {
        if (sub.linkedRecipeId) {
          subRecipePayloads.push({ recipe_id: sub.linkedRecipeId, serving_factor: sub.portion });
        } else if (sub.name.trim() || sub.ingredients.trim() || sub.instructions.trim()) {
          const created = await createRecipe({
            title: sub.name.trim() || 'Untitled',
            description: '',
            source_type: 'MANUAL',
            servings,
            prep_time: 0,
            cook_time: parseInt(cookTime) || 0,
            shelf_life: parseInt(shelfLife) || 0,
            notes: sub.notes,
            is_private: false,
            status: 'draft',
            ingredients: parseIngredients(sub.ingredients),
            instructions: parseInstructions(sub.instructions),
          });
          subRecipePayloads.push({ recipe_id: created.id, serving_factor: sub.portion });
        }
      }

      const isMultiBlock = subSections.length >= 1;
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
        // In multi-block mode all content is in sub-recipes; send empty arrays for main recipe
        ingredients: isMultiBlock ? [] : parseIngredients(ingredients),
        instructions: isMultiBlock ? [] : parseInstructions(instructions),
        nutrition: nutritionPayload,
        ...(subRecipePayloads.length > 0 && { sub_recipes: subRecipePayloads }),
      };

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
        <div className="add-recipe-modal__floating-actions">
          <button
            className="add-recipe-modal__publish-btn"
            type="button"
            disabled={!title.trim() || isSaving}
            onClick={handleSave}
          >
            {initialRecipe ? 'Update' : 'Publish'}
          </button>
          {initialRecipe && (
            <button className="add-recipe-modal__floating-btn" type="button" aria-label="Delete recipe" onClick={handleDelete}>
              <svg width={18} height={18} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                <polyline points="3 6 5 6 21 6" />
                <path d="M19 6l-1 14H6L5 6" />
                <path d="M10 11v6M14 11v6M9 6V4h6v2" />
              </svg>
            </button>
          )}
          <button className="add-recipe-modal__floating-btn" type="button" aria-label="Close" onClick={onClose}>
            <svg width={18} height={18} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
              <line x1={6} y1={6} x2={18} y2={18} />
              <line x1={18} y1={6} x2={6} y2={18} />
            </svg>
          </button>
        </div>

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
          {subSections.length === 0 ? (
            /* Single-block mode: flat card with no header */
            <div className="sub-recipe-card sub-recipe-card--flat">
              <div className="sub-recipe-card__panels">
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
          ) : (
            /* Multi-block mode: all blocks as SubRecipeCard */
            subSections.map((sub, idx) => (
              <SubRecipeCard
                key={sub.id}
                sub={sub}
                allRecipes={allRecipes}
                onDelete={() => handleDeleteSubSection(sub.id)}
                onChange={(field, value) => handleSubChange(sub.id, field, value)}
                onLink={(recipe) => void handleLink(sub.id, recipe)}
                onUnlink={() => handleUnlink(sub.id)}
                onPortionChange={(delta) => handlePortionChange(sub.id, delta)}
                onMoveUp={() => handleMoveSubSection(sub.id, 'up')}
                onMoveDown={() => handleMoveSubSection(sub.id, 'down')}
                canMoveUp={idx > 0}
                canMoveDown={idx < subSections.length - 1}
              />
            ))
          )}
        </div>

        {/* ── Footer ─────────────────────────────────────────── */}
        <div className="add-recipe-modal__footer">
          <button className="add-recipe-modal__sub-recipe-btn" type="button" onClick={handleAddSubSection}>
            + Add sub-recipe
          </button>
        </div>

      </div>
    </div>
  );
}
