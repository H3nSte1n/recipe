import SearchBar from './SearchBar';

interface HomeHeaderProps {
  view: 'grid' | 'graph';
  query: string;
  onQueryChange: (q: string) => void;
  onToggleView: () => void;
  onAddRecipe: () => void;
  onLogout: () => void;
}

export default function HomeHeader({ view, query, onQueryChange, onToggleView, onAddRecipe, onLogout }: HomeHeaderProps) {
  return (
    <header className="home-page__header">
      <div className="home-page__header-content">
        <SearchBar value={query} onChange={onQueryChange} />
        <div className="home-page__header-actions">
          <button
            className={`home-page__view-btn${view === 'graph' ? ' home-page__view-btn--active' : ''}`}
            type="button"
            aria-label="Toggle graph view"
            onClick={onToggleView}
          >
            {view === 'grid' ? (
              <svg width={20} height={20} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round">
                <circle cx={5} cy={12} r={2} />
                <circle cx={19} cy={5} r={2} />
                <circle cx={19} cy={19} r={2} />
                <line x1={7} y1={11} x2={17} y2={6} />
                <line x1={7} y1={13} x2={17} y2={18} />
              </svg>
            ) : (
              <svg width={20} height={20} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round">
                <rect x={3} y={3} width={7} height={7} rx={1} />
                <rect x={14} y={3} width={7} height={7} rx={1} />
                <rect x={3} y={14} width={7} height={7} rx={1} />
                <rect x={14} y={14} width={7} height={7} rx={1} />
              </svg>
            )}
          </button>
          <button
            className="home-page__add-btn"
            type="button"
            aria-label="Add recipe"
            onClick={onAddRecipe}
          >
            <svg width={22} height={22} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round">
              <line x1={5} y1={12} x2={19} y2={12} />
              <line x1={12} y1={5} x2={12} y2={19} />
            </svg>
          </button>
          <button
            className="home-page__logout-btn"
            type="button"
            aria-label="Sign out"
            onClick={onLogout}
          >
            <svg width={22} height={22} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
              <polyline points="16 17 21 12 16 7" />
              <line x1={21} y1={12} x2={9} y2={12} />
            </svg>
          </button>
        </div>
      </div>
    </header>
  );
}
