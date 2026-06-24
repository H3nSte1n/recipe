import { useState, useMemo } from 'react';
import { Recipe } from '../types/recipe';
import { useRecipes } from '../hooks/useRecipes';
import RecipeCard from '../components/RecipeCard';
import RecipeModal from '../components/RecipeModal';
import AddRecipeModal from '../components/AddRecipeModal';
import SearchBar from '../components/SearchBar';
import { getRecipeById } from '../services/recipeService';
import '../styles/HomePage.css';

interface HomePageProps {
  onLogout: () => void;
}

export default function HomePage({ onLogout }: HomePageProps) {
  const { isLoading, error, filterRecipes, recipes, refresh } = useRecipes();
  const [query, setQuery] = useState('');
  const [selectedRecipe, setSelectedRecipe] = useState<Recipe | null>(null);
  const [serves, setServes] = useState(2);
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingRecipe, setEditingRecipe] = useState<Recipe | null>(null);

  const filtered = filterRecipes(query);

  const usedIn = useMemo(() => {
    const map: Record<string, Recipe[]> = {};
    for (const r of recipes) {
      for (const sr of r.sub_recipes ?? []) {
        if (!map[sr.child_id]) map[sr.child_id] = [];
        map[sr.child_id].push(r);
      }
    }
    return map;
  }, [recipes]);

  const handleInc = () => setServes((s) => Math.min(20, s + 1));
  const handleDec = () => setServes((s) => Math.max(1, s - 1));

  function handleEditRecipe() {
    setEditingRecipe(selectedRecipe);
    setSelectedRecipe(null);
  }

  return (
    <div className="home-page">
      <header className="home-page__header">
        <div className="home-page__header-content">
          <SearchBar value={query} onChange={setQuery} />
          <button
            className="home-page__add-btn"
            type="button"
            aria-label="Add recipe"
            onClick={() => setShowAddModal(true)}
          >
            <svg width={22} height={22} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round">
              <line x1={5} y1={12} x2={19} y2={12} />
              <line x1={12} y1={5} x2={12} y2={19} />
            </svg>
          </button>
        </div>
      </header>
      <main className="home-page__main">
        {isLoading && <div className="home-page__loading">Loading…</div>}
        {error && !isLoading && <div className="home-page__error">Failed to load recipes.</div>}
        {!isLoading && !error && filtered.length === 0 && (
          <div className="home-page__empty">
            <span>Nothing here 😋</span>
            <button
              className="home-page__empty-add-btn"
              type="button"
              onClick={() => setShowAddModal(true)}
            >
              Add recipe
            </button>
          </div>
        )}
        {!isLoading && !error && filtered.length > 0 && (
          <div className="home-page__grid">
            {filtered.map((r) => (
              <RecipeCard
                key={r.id}
                recipe={r}
                onClick={() => {
                  void (async () => {
                    try {
                      const full = await getRecipeById(r.id);
                      setSelectedRecipe(full);
                      setServes(full.servings ?? 2);
                    } catch {
                      setSelectedRecipe(r);
                      setServes(r.servings ?? 2);
                    }
                  })();
                }}
              />
            ))}
          </div>
        )}
      </main>
      <button className="home-page__signout" type="button" onClick={onLogout}>
        Sign out
      </button>
      {selectedRecipe && (
        <RecipeModal
          recipe={selectedRecipe}
          serves={serves}
          onInc={handleInc}
          onDec={handleDec}
          onClose={() => setSelectedRecipe(null)}
          onEdit={handleEditRecipe}
          onSubRecipeClick={(sub) => {
            void (async () => {
              try {
                const full = await getRecipeById(sub.id);
                setSelectedRecipe(full);
                setServes(full.servings ?? 2);
              } catch { /* ignore */ }
            })();
          }}
          onParentRecipeClick={(parent) => {
            void (async () => {
              try {
                const full = await getRecipeById(parent.id);
                setSelectedRecipe(full);
                setServes(full.servings ?? 2);
              } catch { /* ignore */ }
            })();
          }}
          usedIn={usedIn}
        />
      )}
      {showAddModal && (
        <AddRecipeModal
          onClose={() => setShowAddModal(false)}
          onSaved={refresh}
        />
      )}
      {editingRecipe && (
        <AddRecipeModal
          initialRecipe={editingRecipe}
          onClose={() => {
            setSelectedRecipe(editingRecipe);
            setEditingRecipe(null);
          }}
          onSaved={() => {
            const recipeId = editingRecipe.id;
            refresh();
            setEditingRecipe(null);
            void (async () => {
              try {
                const updated = await getRecipeById(recipeId);
                setSelectedRecipe(updated);
                setServes(updated.servings ?? 2);
              } catch {
                // refresh failed — stay at overview
              }
            })();
          }}
          onDeleted={() => {
            refresh();
            setEditingRecipe(null);
          }}
        />
      )}
    </div>
  );
}
