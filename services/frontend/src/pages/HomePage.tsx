import { useState, useMemo } from 'react';
import { Recipe } from '../types/recipe';
import { useRecipes } from '../hooks/useRecipes';
import RecipeCard from '../components/RecipeCard';
import RecipeModal from '../components/RecipeModal';
import AddRecipeModal from '../components/AddRecipeModal';
import RecipeGraph from '../components/RecipeGraph';
import HomeHeader from '../components/HomeHeader';
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
  const [view, setView] = useState<'grid' | 'graph'>('grid');

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

  async function handleGraphNodeClick(recipe: Recipe) {
    try {
      const full = await getRecipeById(recipe.id);
      setSelectedRecipe(full);
      setServes(full.servings ?? 2);
    } catch {
      setSelectedRecipe(recipe);
      setServes(recipe.servings ?? 2);
    }
  }

  return (
    <div className={`home-page${view === 'graph' ? ' home-page--graph' : ''}`}>
      <HomeHeader
        view={view}
        query={query}
        onQueryChange={setQuery}
        onToggleView={() => setView(v => v === 'grid' ? 'graph' : 'grid')}
        onAddRecipe={() => setShowAddModal(true)}
        onLogout={onLogout}
      />
      <main className={`home-page__main${view === 'graph' ? ' home-page__main--graph' : ''}`}>
        {view === 'graph' ? (
          <RecipeGraph recipes={recipes} usedIn={usedIn} onRecipeClick={(r) => void handleGraphNodeClick(r)} />
        ) : (
          <>
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
          </>
        )}
      </main>
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
