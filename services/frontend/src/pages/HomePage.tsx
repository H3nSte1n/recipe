import { useState } from 'react';
import { Recipe } from '../types/recipe';
import { useRecipes } from '../hooks/useRecipes';
import RecipeCard from '../components/RecipeCard';
import SearchBar from '../components/SearchBar';
import '../styles/HomePage.css';

interface HomePageProps {
  onLogout: () => void;
}

export default function HomePage({ onLogout: _onLogout }: HomePageProps) {
  const { isLoading, error, filterRecipes } = useRecipes();
  const [query, setQuery] = useState('');
  const [_selectedRecipe, setSelectedRecipe] = useState<Recipe | null>(null);
  const [_serves, setServes] = useState(2);

  const filtered = filterRecipes(query);

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
                  setSelectedRecipe(r);
                  setServes(2);
                }}
              />
            ))}
          </div>
        )}
      </main>
      <button className="home-page__profile" type="button" aria-label="Profile">
        J
      </button>
    </div>
  );
}
