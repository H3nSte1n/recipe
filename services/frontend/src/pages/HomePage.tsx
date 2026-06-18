import { useState } from 'react';
import { Recipe } from '../types/recipe';
import { useRecipes } from '../hooks/useRecipes';
import RecipeCard from '../components/RecipeCard';
import RecipeModal from '../components/RecipeModal';
import SearchBar from '../components/SearchBar';
import { getRecipeById } from '../services/recipeService';
import '../styles/HomePage.css';

interface HomePageProps {
  onLogout: () => void;
}

export default function HomePage({ onLogout }: HomePageProps) {
  const { isLoading, error, filterRecipes } = useRecipes();
  const [query, setQuery] = useState('');
  const [selectedRecipe, setSelectedRecipe] = useState<Recipe | null>(null);
  const [serves, setServes] = useState(2);

  const filtered = filterRecipes(query);

  const handleInc = () => setServes((s) => Math.min(20, s + 1));
  const handleDec = () => setServes((s) => Math.max(1, s - 1));

  return (
    <div>
      <header className="home-page__header">
        <SearchBar value={query} onSearch={setQuery} />
      </header>
      <main className="home-page__main">
        {isLoading && <div className="home-page__loading">Loading…</div>}
        {error && !isLoading && <div className="home-page__error">Failed to load recipes.</div>}
        {!isLoading && !error && filtered.length === 0 && (
          <div className="home-page__empty">Nothing here.</div>
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
                      setServes(2);
                    } catch {
                      setSelectedRecipe(r);
                      setServes(2);
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
      {selectedRecipe && (
        <RecipeModal
          recipe={selectedRecipe}
          serves={serves}
          onInc={handleInc}
          onDec={handleDec}
          onClose={() => setSelectedRecipe(null)}
        />
      )}
    </div>
  );
}
