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

  return (
    <div>
      <header className="home-page__header">
        <SearchBar value={query} onSearch={setQuery} />
      </header>
      <main className="home-page__main">
        {isLoading && <div className="home-page__loading type-body">Loading…</div>}
        {error && !isLoading && <div className="home-page__error type-body">Failed to load recipes.</div>}
        {!isLoading && !error && filtered.length === 0 && (
          <div className="home-page__empty type-h2">Nothing here.</div>
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
      <button className="home-page__profile" type="button" aria-label="Profile" onClick={onLogout}>
        J
      </button>
      <button
        className="home-page__add"
        type="button"
        aria-label="Add recipe"
        onClick={() => setShowAddModal(true)}
      >
        +
      </button>
      {selectedRecipe && (
        <RecipeModal
          recipe={selectedRecipe}
          serves={serves}
          onInc={handleInc}
          onDec={handleDec}
          onClose={() => setSelectedRecipe(null)}
          usedIn={usedIn}
        />
      )}
      {showAddModal && (
        <AddRecipeModal
          onClose={() => setShowAddModal(false)}
          onSaved={refresh}
        />
      )}
    </div>
  );
}
