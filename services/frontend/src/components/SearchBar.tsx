import '../styles/SearchBar.css'

interface SearchBarProps {
  value: string
  onSearch: (q: string) => void
}

export default function SearchBar({ value, onSearch }: SearchBarProps) {
  return (
    <div className="search-bar">
      <input
        type="text"
        value={value}
        onChange={(e) => onSearch(e.target.value)}
        placeholder="Search..."
        aria-label="Search recipes"
        className="search-bar__input"
      />
      <button
        className="search-bar__add-btn"
        type="button"
        aria-label="Add recipe"
      >
        <svg
          width={24}
          height={24}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={1.5}
          strokeLinecap="round"
        >
          <line x1={5} y1={12} x2={19} y2={12} />
          <line x1={12} y1={5} x2={12} y2={19} />
        </svg>
      </button>
    </div>
  )
}
