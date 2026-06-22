import { useRef, useState } from 'react';
import { createRecipe } from '../services/recipeService';
import {
  CreateRecipeIngredientPayload,
  CreateRecipeInstructionPayload,
  CreateRecipeNutritionPayload,
  Recipe,
} from '../types/recipe';
import '../styles/AddRecipeModal.css';

interface AddRecipeModalProps {
  onClose: () => void;
  onSaved: () => void;
}

interface Section {
  id: string;
  name: string;
  servings: number;
  ingredients: string;
  instructions: string;
  notes: string;
}

function parseIngredients(text: string): CreateRecipeIngredientPayload[] {
  return text
    .split('\n')
    .filter((l) => l.trim())
    .map((line) => ({ name: line.trim(), description: line.trim(), amount: 0, unit: '', notes: '' }));
}

function parseInstructions(text: string): CreateRecipeInstructionPayload[] {
  return text
    .split('\n')
    .filter((l) => l.trim())
    .map((line, idx) => ({ step_number: idx + 1, instruction: line.trim() }));
}

function AddRecipeModal({ onClose, onSaved }: AddRecipeModalProps) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState('');
  const [prepTime, setPrepTime] = useState('');
  const [calories, setCalories] = useState('');
  const [protein, setProtein] = useState('');
  const [fat, setFat] = useState('');
  const [cookTime, setCookTime] = useState('');
  const [shelfLife, setShelfLife] = useState('');
  const [servings, setServings] = useState('1');
  const [status, setStatus] = useState('published');
  const [carbs, setCarbs] = useState('');
  const [sections, setSections] = useState<Section[]>([
    { id: String(Date.now()), name: '', servings: 1, ingredients: '', instructions: '', notes: '' },
  ]);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState('');

  const fileInputRef = useRef<HTMLInputElement>(null);

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

  function handleSectionChange(id: string, field: keyof Section, value: string) {
    setSections((prev) =>
      prev.map((s) => (s.id === id ? { ...s, [field]: value } : s)),
    );
  }

  function handleServingsChange(id: string, delta: number) {
    setSections((prev) =>
      prev.map((s) => {
        if (s.id !== id) return s;
        const next = Math.min(99, Math.max(1, s.servings + delta));
        return { ...s, servings: next };
      }),
    );
  }

  const handleSave = async () => {
    if (!title.trim() || isSaving) return;
    setIsSaving(true);
    setSaveError('');

    const parsedServings = parseInt(servings);
    if (isNaN(parsedServings) || parsedServings < 1) {
      setSaveError('Servings must be at least 1');
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
        await createRecipe(
          {
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
            ingredients: parseIngredients(sections[0].ingredients),
            instructions: parseInstructions(sections[0].instructions),
            nutrition: nutritionPayload,
          },
          imageFile,
        );
      } else {
        const subRecipes: Recipe[] = [];
        for (const section of sections) {
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
          subRecipes.push(created);
        }
        await createRecipe(
          {
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
            sub_recipes: subRecipes.map((r) => ({ recipe_id: r.id, serving_factor: 1 })),
          },
          imageFile,
        );
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
          Save
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

        {/* D: Stats card */}
        <div className="add-recipe-modal__stats">
          <div className="add-recipe-modal__stats-grid">
            {/* Prep time */}
            <div className="add-recipe-modal__stat-cell">
              <span className="add-recipe-modal__stat-icon">
                <svg width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="10" />
                  <polyline points="12 6 12 12 16 14" />
                </svg>
              </span>
              <span className="add-recipe-modal__stat-label">Prep</span>
              <input
                className="add-recipe-modal__stat-input"
                type="number"
                min={0}
                placeholder="Add"
                value={prepTime}
                onChange={(e) => setPrepTime(e.target.value)}
              />
              <span className="add-recipe-modal__stat-unit">min</span>
            </div>

            {/* Cook time */}
            <div className="add-recipe-modal__stat-cell">
              <span className="add-recipe-modal__stat-icon">
                <svg width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                  <path d="M12 2c0 6-6 8-6 14a6 6 0 0 0 12 0c0-6-6-8-6-14z" />
                </svg>
              </span>
              <span className="add-recipe-modal__stat-label">Cook</span>
              <input
                className="add-recipe-modal__stat-input"
                type="number"
                min={0}
                placeholder="Add"
                value={cookTime}
                onChange={(e) => setCookTime(e.target.value)}
              />
              <span className="add-recipe-modal__stat-unit">min</span>
            </div>

            {/* Servings */}
            <div className="add-recipe-modal__stat-cell">
              <span className="add-recipe-modal__stat-icon">
                <svg width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="5" />
                </svg>
              </span>
              <span className="add-recipe-modal__stat-label">Servings</span>
              <input
                className="add-recipe-modal__stat-input"
                type="number"
                min={1}
                placeholder="1"
                value={servings}
                onChange={(e) => setServings(e.target.value)}
              />
            </div>

            {/* Shelf life */}
            <div className="add-recipe-modal__stat-cell">
              <span className="add-recipe-modal__stat-icon">
                <svg width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                  <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
                  <line x1="16" y1="2" x2="16" y2="6" />
                  <line x1="8" y1="2" x2="8" y2="6" />
                  <line x1="3" y1="10" x2="21" y2="10" />
                </svg>
              </span>
              <span className="add-recipe-modal__stat-label">Shelf life</span>
              <input
                className="add-recipe-modal__stat-input"
                type="number"
                min={0}
                placeholder="Add"
                value={shelfLife}
                onChange={(e) => setShelfLife(e.target.value)}
              />
              <span className="add-recipe-modal__stat-unit">days</span>
            </div>

            {/* Calories */}
            <div className="add-recipe-modal__stat-cell">
              <span className="add-recipe-modal__stat-icon">
                <svg width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                  <path d="M12 2c0 6-6 8-6 14a6 6 0 0 0 12 0c0-6-6-8-6-14z" />
                </svg>
              </span>
              <span className="add-recipe-modal__stat-label">Calories</span>
              <input
                className="add-recipe-modal__stat-input"
                type="number"
                min={0}
                placeholder="Add"
                value={calories}
                onChange={(e) => setCalories(e.target.value)}
              />
              <span className="add-recipe-modal__stat-unit">kcal</span>
            </div>

            {/* Carbs */}
            <div className="add-recipe-modal__stat-cell">
              <span className="add-recipe-modal__stat-icon">
                <svg width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                  <path d="M12 2L6 12a6 6 0 1 0 12 0L12 2z" />
                </svg>
              </span>
              <span className="add-recipe-modal__stat-label">Carbs</span>
              <input
                className="add-recipe-modal__stat-input"
                type="number"
                min={0}
                placeholder="Add"
                value={carbs}
                onChange={(e) => setCarbs(e.target.value)}
              />
              <span className="add-recipe-modal__stat-unit">g</span>
            </div>

            {/* Protein */}
            <div className="add-recipe-modal__stat-cell">
              <span className="add-recipe-modal__stat-icon">
                <svg width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="5" />
                </svg>
              </span>
              <span className="add-recipe-modal__stat-label">Protein</span>
              <input
                className="add-recipe-modal__stat-input"
                type="number"
                min={0}
                placeholder="Add"
                value={protein}
                onChange={(e) => setProtein(e.target.value)}
              />
              <span className="add-recipe-modal__stat-unit">g</span>
            </div>

            {/* Fat */}
            <div className="add-recipe-modal__stat-cell">
              <span className="add-recipe-modal__stat-icon">
                <svg width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
                  <path d="M12 2L6 12a6 6 0 1 0 12 0L12 2z" />
                </svg>
              </span>
              <span className="add-recipe-modal__stat-label">Fat</span>
              <input
                className="add-recipe-modal__stat-input"
                type="number"
                min={0}
                placeholder="Add"
                value={fat}
                onChange={(e) => setFat(e.target.value)}
              />
              <span className="add-recipe-modal__stat-unit">g</span>
            </div>
          </div>

          {/* Status selector */}
          <div className="add-recipe-modal__status-row">
            <span className="add-recipe-modal__stat-label">Status</span>
            <select
              className="add-recipe-modal__status-select"
              value={status}
              onChange={(e) => setStatus(e.target.value)}
            >
              <option value="published">Published</option>
              <option value="draft">Draft</option>
              <option value="archived">Archived</option>
            </select>
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
            sections.map((section) => (
              <div key={section.id} className="add-recipe-modal__section-card">
                <div className="add-recipe-modal__section-header">
                  <span className="add-recipe-modal__section-handle">↓</span>
                  <input
                    className="add-recipe-modal__section-name"
                    placeholder="Give me a name"
                    value={section.name}
                    onChange={(e) => handleSectionChange(section.id, 'name', e.target.value)}
                  />
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
                  >
                    −
                  </button>
                  <span className="add-recipe-modal__servings-count">{section.servings}</span>
                  <button
                    className="add-recipe-modal__servings-btn"
                    type="button"
                    onClick={() => handleServingsChange(section.id, 1)}
                  >
                    +
                  </button>
                </div>
                <div className="add-recipe-modal__section-content">
                  {/* Ingredients field card */}
                  <div className="add-recipe-modal__field-card">
                    <span className="add-recipe-modal__field-label">INGREDIENTS</span>
                    <textarea
                      className="add-recipe-modal__field-textarea"
                      placeholder={'e.g. 1 cup flour\ne.g. 2 eggs'}
                      value={section.ingredients}
                      onChange={(e) => handleSectionChange(section.id, 'ingredients', e.target.value)}
                    />
                  </div>
                  {/* Instructions field card */}
                  <div className="add-recipe-modal__field-card">
                    <span className="add-recipe-modal__field-label">INSTRUCTIONS</span>
                    <textarea
                      className="add-recipe-modal__field-textarea"
                      placeholder="e.g. Combine dry ingredients"
                      value={section.instructions}
                      onChange={(e) => handleSectionChange(section.id, 'instructions', e.target.value)}
                    />
                  </div>
                  {/* Notes field card */}
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
            ))
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
