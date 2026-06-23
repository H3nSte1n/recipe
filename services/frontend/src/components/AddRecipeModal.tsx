import { useEffect, useRef, useState } from 'react';
import { createRecipe, getMyRecipes, getRecipeById, updateRecipe } from '../services/recipeService';
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
  initialRecipe?: Recipe;
}

interface Section {
  id: string;
  name: string;
  servings: number;
  ingredients: string;
  instructions: string;
  notes: string;
  linkedRecipeId?: string;
  linkedIngredients?: RecipeIngredient[];
}

function formatIngredients(ingredients: RecipeIngredient[], factor: number): string {
  return ingredients
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

function buildInitialSections(recipe?: Recipe): Section[] {
  if (!recipe) {
    return [{ id: String(Date.now()), name: '', servings: 1, ingredients: '', instructions: '', notes: '' }];
  }
  const subs = recipe.sub_recipes ?? [];
  if (subs.length > 0) {
    return subs.map((sr, i) => ({
      id: String(Date.now() + i),
      name: sr.child?.title ?? '',
      servings: sr.serving_factor,
      linkedRecipeId: sr.child_id,
      linkedIngredients: sr.child?.ingredients ?? [],
      ingredients: formatIngredients(sr.child?.ingredients ?? [], sr.serving_factor),
      instructions: (sr.child?.instructions ?? [])
        .sort((a, b) => a.step_number - b.step_number)
        .map((ins) => ins.instruction)
        .join('\n'),
      notes: '',
    }));
  }
  return [{
    id: String(Date.now()),
    name: '',
    servings: recipe.servings,
    ingredients: formatIngredients(recipe.ingredients ?? [], 1),
    instructions: (recipe.instructions ?? [])
      .sort((a, b) => a.step_number - b.step_number)
      .map((ins) => ins.instruction)
      .join('\n'),
    notes: recipe.notes ?? '',
  }];
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

function AddRecipeModal({ onClose, onSaved, initialRecipe }: AddRecipeModalProps) {
  const [title, setTitle] = useState(initialRecipe?.title ?? '');
  const [description, setDescription] = useState(initialRecipe?.description ?? '');
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState(initialRecipe?.image_url ?? '');
  const [prepTime, setPrepTime] = useState(initialRecipe?.prep_time ? String(initialRecipe.prep_time) : '');
  const [calories, setCalories] = useState(initialRecipe?.nutrition?.calories ? String(initialRecipe.nutrition.calories) : '');
  const [protein, setProtein] = useState(initialRecipe?.nutrition?.protein ? String(initialRecipe.nutrition.protein) : '');
  const [fat, setFat] = useState(initialRecipe?.nutrition?.fat ? String(initialRecipe.nutrition.fat) : '');
  const [cookTime, setCookTime] = useState(initialRecipe?.cook_time ? String(initialRecipe.cook_time) : '');
  const [shelfLife, setShelfLife] = useState(initialRecipe?.shelf_life ? String(initialRecipe.shelf_life) : '');
  const [servings, setServings] = useState(initialRecipe?.servings ? String(initialRecipe.servings) : '1');
  const [status, setStatus] = useState(initialRecipe?.status ?? 'published');
  const [carbs, setCarbs] = useState(initialRecipe?.nutrition?.carbs ? String(initialRecipe.nutrition.carbs) : '');
  const [sections, setSections] = useState<Section[]>(() => buildInitialSections(initialRecipe));
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [allRecipes, setAllRecipes] = useState<Recipe[]>([]);
  const [openDropdownId, setOpenDropdownId] = useState<string | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    getMyRecipes().then(setAllRecipes).catch(() => {});
  }, []);

  function handleImageSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setImageFile(file);
    setImagePreview(URL.createObjectURL(file));
  }

  function handleAddSection() {
    setSections((prev) => [
      ...prev,
      { id: String(Date.now()), name: '', servings: 1, ingredients: '', instructions: '', notes: '' },
    ]);
  }

  function handleDeleteSection(id: string) {
    setSections((prev) => {
      if (prev.length <= 1) return prev;
      return prev.filter((s) => s.id !== id);
    });
  }

  function handleSectionChange(id: string, field: 'name' | 'ingredients' | 'instructions' | 'notes', value: string) {
    setSections((prev) =>
      prev.map((s) => (s.id === id ? { ...s, [field]: value } : s)),
    );
  }

  function handleServingsChange(id: string, delta: number) {
    setSections((prev) =>
      prev.map((s) => {
        if (s.id !== id) return s;
        const next = Math.min(99, Math.max(1, s.servings + delta));
        if (s.linkedIngredients) {
          return { ...s, servings: next, ingredients: formatIngredients(s.linkedIngredients, next) };
        }
        return { ...s, servings: next };
      }),
    );
  }

  async function handleLinkRecipe(sectionId: string, recipe: Recipe) {
    setOpenDropdownId(null);
    setSections((prev) =>
      prev.map((s) =>
        s.id === sectionId
          ? { ...s, name: recipe.title, linkedRecipeId: recipe.id, ingredients: '', instructions: '' }
          : s,
      ),
    );
    try {
      const full = await getRecipeById(recipe.id);
      const linkedIngredients = full.ingredients ?? [];
      const instructionsText = (full.instructions ?? []).map((i) => i.instruction).join('\n');
      setSections((prev) =>
        prev.map((s) => {
          if (s.id !== sectionId || s.linkedRecipeId !== recipe.id) return s;
          return {
            ...s,
            linkedIngredients,
            ingredients: formatIngredients(linkedIngredients, s.servings),
            instructions: instructionsText,
          };
        }),
      );
    } catch {
      setSaveError('Could not load recipe details. Please try again.');
    }
  }

  function handleUnlinkSection(sectionId: string) {
    setSections((prev) =>
      prev.map((s) =>
        s.id === sectionId ? { ...s, linkedRecipeId: undefined } : s,
      ),
    );
  }

  const handleSave = async () => {
    if (!title.trim() || isSaving) return;
    setIsSaving(true);
    setSaveError('');
    const isEditing = !!initialRecipe;

    const parsedServings = parseInt(servings);
    if (isNaN(parsedServings) || parsedServings < 1) {
      setSaveError('Servings must be at least 1');
      setIsSaving(false);
      return;
    }
    if (parseFloat(servings) !== parsedServings) {
      setSaveError('Servings must be a whole number');
      setIsSaving(false);
      return;
    }

    const nutritionPayload: CreateRecipeNutritionPayload | undefined =
      calories !== '' || protein !== '' || fat !== '' || carbs !== ''
        ? {
            calories: parseFloat(calories) || 0,
            protein: parseFloat(protein) || 0,
            fat: parseFloat(fat) || 0,
            carbs: parseFloat(carbs) || 0,
          }
        : undefined;

    try {
      if (sections.length === 1) {
        const singlePayload = {
          title: title.trim(),
          description,
          source_type: 'MANUAL',
          servings: parsedServings,
          prep_time: parseInt(prepTime) || 0,
          cook_time: parseInt(cookTime) || 0,
          shelf_life: parseInt(shelfLife) || 0,
          notes: sections[0].notes,
          is_private: false,
          status,
          ingredients: parseIngredients(sections[0].ingredients),
          instructions: parseInstructions(sections[0].instructions),
          nutrition: nutritionPayload,
        };
        if (isEditing) {
          await updateRecipe(initialRecipe!.id, singlePayload, imageFile);
        } else {
          await createRecipe(singlePayload, imageFile);
        }
      } else {
        const subRecipePayloads: SubRecipePayload[] = [];
        for (const section of sections) {
          if (section.linkedRecipeId) {
            subRecipePayloads.push({ recipe_id: section.linkedRecipeId, serving_factor: section.servings });
          } else {
            const created = await createRecipe({
              title: section.name.trim() || 'Untitled',
              description: '',
              source_type: 'MANUAL',
              servings: section.servings,
              prep_time: 0,
              cook_time: parseInt(cookTime) || 0,
              shelf_life: parseInt(shelfLife) || 0,
              notes: section.notes,
              is_private: false,
              status,
              ingredients: parseIngredients(section.ingredients),
              instructions: parseInstructions(section.instructions),
            });
            subRecipePayloads.push({ recipe_id: created.id, serving_factor: section.servings });
          }
        }
        const multiPayload = {
          title: title.trim(),
          description,
          source_type: 'MANUAL',
          servings: parsedServings,
          prep_time: parseInt(prepTime) || 0,
          cook_time: parseInt(cookTime) || 0,
          shelf_life: parseInt(shelfLife) || 0,
          notes: '',
          is_private: false,
          status,
          ingredients: [],
          instructions: [],
          nutrition: nutritionPayload,
          sub_recipes: subRecipePayloads,
        };
        if (isEditing) {
          await updateRecipe(initialRecipe!.id, multiPayload, imageFile);
        } else {
          await createRecipe(multiPayload, imageFile);
        }
      }
      onSaved();
      onClose();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save recipe');
      setIsSaving(false);
    }
  };

  return (
    <div className="add-recipe-modal">
      {/* Header */}
      <div className="add-recipe-modal__header">
        <button className="add-recipe-modal__back" type="button" onClick={onClose} aria-label="Go back">
          <svg width={20} height={20} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
            <polyline points="15 18 9 12 15 6" />
          </svg>
        </button>
        <button
          className="add-recipe-modal__save"
          type="button"
          disabled={!title.trim() || isSaving}
          onClick={handleSave}
        >
          {initialRecipe ? 'Update' : 'Save'}
        </button>
      </div>

      {/* Scrollable content */}
      <div className="add-recipe-modal__content">

        {/* A: Image upload */}
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
            <>
              <svg width={32} height={32} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                <circle cx="8.5" cy="8.5" r="1.5" />
                <polyline points="21 15 16 10 5 21" />
              </svg>
              <span className="add-recipe-modal__image-label">Add image</span>
            </>
          )}
        </div>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          style={{ display: 'none' }}
          onChange={handleImageSelect}
        />

        {/* B: Title */}
        <input
          className="add-recipe-modal__title-input"
          placeholder="Give me a name"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />

        {/* C: Description */}
        <input
          className="add-recipe-modal__desc-input"
          placeholder="Add a description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />

        {/* D: Status card */}
        <div className="add-recipe-modal__status-card">
          <span className="add-recipe-modal__status-card-label">Status</span>
          <div className="add-recipe-modal__status-card-control">
            <select
              className="add-recipe-modal__status-card-select"
              value={status}
              onChange={(e) => setStatus(e.target.value)}
            >
              <option value="published">Published</option>
              <option value="draft">Draft</option>
              <option value="archived">Archived</option>
            </select>
            <svg className="add-recipe-modal__status-card-chevron" width={18} height={18} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
              <polyline points="6 9 12 15 18 9" />
            </svg>
          </div>
        </div>

        {/* D2: Meta grid — same card style as nutrition */}
        <div className="add-recipe-modal__nutrition-grid add-recipe-modal__meta-grid">
          <div className="add-recipe-modal__nutrition-card">
            <span className="add-recipe-modal__nutrition-card-label">Prep Time</span>
            <input className="add-recipe-modal__nutrition-card-input" type="number" min={0} placeholder="0" value={prepTime} onChange={(e) => setPrepTime(e.target.value)} />
            <span className="add-recipe-modal__nutrition-card-unit">min</span>
          </div>
          <div className="add-recipe-modal__nutrition-card">
            <span className="add-recipe-modal__nutrition-card-label">Cook Time</span>
            <input className="add-recipe-modal__nutrition-card-input" type="number" min={0} placeholder="0" value={cookTime} onChange={(e) => setCookTime(e.target.value)} />
            <span className="add-recipe-modal__nutrition-card-unit">min</span>
          </div>
          <div className="add-recipe-modal__nutrition-card">
            <span className="add-recipe-modal__nutrition-card-label">Servings</span>
            <input className="add-recipe-modal__nutrition-card-input" type="number" min={1} step={1} placeholder="1" value={servings} onChange={(e) => setServings(e.target.value)} />
            <span className="add-recipe-modal__nutrition-card-unit">&nbsp;</span>
          </div>
          <div className="add-recipe-modal__nutrition-card">
            <span className="add-recipe-modal__nutrition-card-label">Shelf Life</span>
            <input className="add-recipe-modal__nutrition-card-input" type="number" min={0} placeholder="0" value={shelfLife} onChange={(e) => setShelfLife(e.target.value)} />
            <span className="add-recipe-modal__nutrition-card-unit">days</span>
          </div>
        </div>

        {/* D3: Nutrition grid */}
        <div className="add-recipe-modal__nutrition-grid">
          <div className="add-recipe-modal__nutrition-card">
            <span className="add-recipe-modal__nutrition-card-label">Calories</span>
            <input className="add-recipe-modal__nutrition-card-input" type="number" min={0} placeholder="—" value={calories} onChange={(e) => setCalories(e.target.value)} />
            <span className="add-recipe-modal__nutrition-card-unit">kcal</span>
          </div>
          <div className="add-recipe-modal__nutrition-card">
            <span className="add-recipe-modal__nutrition-card-label">Carbs</span>
            <input className="add-recipe-modal__nutrition-card-input" type="number" min={0} placeholder="—" value={carbs} onChange={(e) => setCarbs(e.target.value)} />
            <span className="add-recipe-modal__nutrition-card-unit">g</span>
          </div>
          <div className="add-recipe-modal__nutrition-card">
            <span className="add-recipe-modal__nutrition-card-label">Protein</span>
            <input className="add-recipe-modal__nutrition-card-input" type="number" min={0} placeholder="—" value={protein} onChange={(e) => setProtein(e.target.value)} />
            <span className="add-recipe-modal__nutrition-card-unit">g</span>
          </div>
          <div className="add-recipe-modal__nutrition-card">
            <span className="add-recipe-modal__nutrition-card-label">Fat</span>
            <input className="add-recipe-modal__nutrition-card-input" type="number" min={0} placeholder="—" value={fat} onChange={(e) => setFat(e.target.value)} />
            <span className="add-recipe-modal__nutrition-card-unit">g</span>
          </div>
        </div>

        {/* E: Sections */}
        <div className="add-recipe-modal__sections">
          {sections.length === 1 ? (
            <div className="add-recipe-modal__section-card">
              <div className="add-recipe-modal__section-content">
                {/* Ingredients field card */}
                <div className="add-recipe-modal__field-card">
                  <span className="add-recipe-modal__field-label">INGREDIENTS</span>
                  <textarea
                    className="add-recipe-modal__field-textarea"
                    placeholder={'e.g. 1 cup flour\ne.g. 2 eggs'}
                    value={sections[0].ingredients}
                    onChange={(e) => handleSectionChange(sections[0].id, 'ingredients', e.target.value)}
                  />
                </div>
                {/* Instructions field card */}
                <div className="add-recipe-modal__field-card">
                  <span className="add-recipe-modal__field-label">INSTRUCTIONS</span>
                  <textarea
                    className="add-recipe-modal__field-textarea"
                    placeholder={'e.g. Combine dry ingredients\ne.g. Add wet ingredients and mix'}
                    value={sections[0].instructions}
                    onChange={(e) => handleSectionChange(sections[0].id, 'instructions', e.target.value)}
                  />
                </div>
              </div>
            </div>
          ) : (
            sections.map((section) => {
              const filtered = allRecipes.filter(
                (r) => section.name.length > 0 && r.title.toLowerCase().includes(section.name.toLowerCase()),
              );
              return (
                <div key={section.id} className="add-recipe-modal__section-card">
                  <div className="add-recipe-modal__section-header">
                    <span className="add-recipe-modal__section-handle">↓</span>
                    {section.linkedRecipeId ? (
                      <>
                        <span className="add-recipe-modal__section-linked-name">{section.name}</span>
                        <button
                          className="add-recipe-modal__section-unlink"
                          type="button"
                          onClick={() => handleUnlinkSection(section.id)}
                          aria-label="Unlink recipe"
                        >×</button>
                      </>
                    ) : (
                      <div className="add-recipe-modal__section-search">
                        <input
                          className="add-recipe-modal__section-name"
                          placeholder="Give me a name"
                          value={section.name}
                          onChange={(e) => {
                            handleSectionChange(section.id, 'name', e.target.value);
                            setOpenDropdownId(section.id);
                          }}
                          onFocus={() => { if (section.name.length > 0) setOpenDropdownId(section.id); }}
                          onBlur={() => setTimeout(() => setOpenDropdownId(null), 150)}
                        />
                        {openDropdownId === section.id && filtered.length > 0 && (
                          <div className="add-recipe-modal__section-dropdown">
                            {filtered.slice(0, 5).map((r) => (
                              <button
                                key={r.id}
                                className="add-recipe-modal__section-dropdown-item"
                                type="button"
                                onMouseDown={() => handleLinkRecipe(section.id, r)}
                              >
                                {r.title}
                              </button>
                            ))}
                          </div>
                        )}
                      </div>
                    )}
                    <button
                      className="add-recipe-modal__section-delete"
                      onClick={() => handleDeleteSection(section.id)}
                      type="button"
                      aria-label="Delete section"
                    >
                      <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                        <polyline points="3 6 5 6 21 6" />
                        <path d="M19 6l-1 14H6L5 6" />
                        <path d="M10 11v6M14 11v6" />
                        <path d="M9 6V4h6v2" />
                      </svg>
                    </button>
                  </div>
                  <div className="add-recipe-modal__servings-row">
                    <span>Servings:</span>
                    <button
                      className="add-recipe-modal__servings-btn"
                      type="button"
                      onClick={() => handleServingsChange(section.id, -1)}
                    >−</button>
                    <span className="add-recipe-modal__servings-count">{section.servings}</span>
                    <button
                      className="add-recipe-modal__servings-btn"
                      type="button"
                      onClick={() => handleServingsChange(section.id, 1)}
                    >+</button>
                  </div>
                  <div className="add-recipe-modal__section-content">
                    <div className="add-recipe-modal__field-card">
                      <span className="add-recipe-modal__field-label">INGREDIENTS</span>
                      <textarea
                        className="add-recipe-modal__field-textarea"
                        placeholder={'e.g. 1 cup flour\ne.g. 2 eggs'}
                        value={section.ingredients}
                        disabled={!!section.linkedRecipeId}
                        onChange={(e) => handleSectionChange(section.id, 'ingredients', e.target.value)}
                      />
                    </div>
                    <div className="add-recipe-modal__field-card">
                      <span className="add-recipe-modal__field-label">INSTRUCTIONS</span>
                      <textarea
                        className="add-recipe-modal__field-textarea"
                        placeholder="e.g. Combine dry ingredients"
                        value={section.instructions}
                        disabled={!!section.linkedRecipeId}
                        onChange={(e) => handleSectionChange(section.id, 'instructions', e.target.value)}
                      />
                    </div>
                    <div className="add-recipe-modal__field-card">
                      <span className="add-recipe-modal__field-label">NOTES</span>
                      <textarea
                        className="add-recipe-modal__field-textarea"
                        placeholder="e.g. Leave out thyme — too overpowering here"
                        value={section.notes}
                        onChange={(e) => handleSectionChange(section.id, 'notes', e.target.value)}
                      />
                    </div>
                  </div>
                </div>
              );
            })
          )}
        </div>

        {/* F: Add section button */}
        <button className="add-recipe-modal__add-btn" type="button" onClick={handleAddSection}>
          + Add
        </button>

        {/* G: Save error */}
        {saveError && <p className="add-recipe-modal__save-error">{saveError}</p>}
      </div>
    </div>
  );
}

export default AddRecipeModal;
