# Copilot Instructions - Recipe App Frontend

## 🍳 Project Overview

**Recipe App** - A full-stack recipe management application with React frontend and Go backend.

**Frontend**: React 18 + TypeScript with Vite
- Development: `npm run dev` → http://localhost:5173
- Docker: Runs Node.js with Vite dev server
- Hot reload: Changes appear instantly
- API proxy: /api/* → backend at http://app:8080 (Docker) or http://localhost:8080 (local)

**Backend**: Go REST API
- Development: `make dev` → http://localhost:8080
- Docker: `docker-compose up app db`

---

## 🛠️ Technology Stack

### Frontend
- **React 18.3.1** - UI library
- **TypeScript 5.2.2** - Static typing (strict mode)
- **Vite 5.1.0** - Build tool and dev server
- **Node.js 24 LTS** - Runtime
- **ESLint 8.57.0** - Code quality

### Backend
- **Go 1.24** - API server
- **PostgreSQL 18** - Database
- **Docker** - Containerization

### Versions
See `.tool-versions`:
- golang 1.24.0
- nodejs 24.0.0

---

## 📁 Project Structure

```
services/frontend/
├── src/
│   ├── components/         # Reusable React components
│   ├── pages/              # Page components (HomePage, etc.)
│   ├── hooks/              # Custom React hooks
│   ├── services/           # API service functions
│   ├── types/              # TypeScript type definitions
│   ├── styles/             # CSS stylesheets
│   │   ├── index.css       # Global styles
│   │   └── App.css         # Layout styles
│   ├── utils/              # Helper functions
│   ├── App.tsx             # Root component
│   └── main.tsx            # React entry point
├── package.json            # Dependencies
├── vite.config.ts          # Vite configuration with API proxy
├── tsconfig.json           # TypeScript configuration
├── .eslintrc.cjs           # ESLint rules
├── index.html              # HTML template
├── .dockerignore
├── Dockerfile              # Docker configuration (Vite dev server)
└── README.md               # Frontend documentation
```

---

## 💻 Development Setup

### Local Development (Recommended)

```bash
# Terminal 1: Start backend
cd services/backend
make dev
# Backend at http://localhost:8080

# Terminal 2: Start frontend
cd services/frontend
npm install  # First time only
npm run dev
# Frontend at http://localhost:5173
```

**Vite automatically proxies `/api/*` to `http://localhost:8080`**

### Docker Development

```bash
# Terminal 1: Start backend + database
docker-compose up db app

# Terminal 2: Start frontend
cd services/frontend
npm install
npm run dev
# Frontend at http://localhost:5173
# Vite proxies to http://app:8080 (Docker service name)
```

---

## 🚀 NPM Scripts

```bash
npm run dev          # Start Vite dev server with hot reload
npm run build        # Build for production (creates dist/)
npm run preview      # Preview production build locally
npm run lint         # Run ESLint code quality checks
npm run type-check   # TypeScript type checking
```

---

## 📋 Coding Standards

### TypeScript
- **Strict mode enabled** - no implicit any
- **Always use type annotations** for function parameters and return types
- **Interface over type** for object shapes
- **Export types** from services for use in components

Example:
```typescript
// ✅ Good
interface User {
  id: string
  name: string
}

const getUser = async (id: string): Promise<User> => {
  const response = await fetch(`/api/users/${id}`)
  return response.json()
}

// ❌ Bad
const getUser = async (id) => {
  const response = await fetch(`/api/users/${id}`)
  return response.json()
}
```

### Component Naming
- **Files**: PascalCase (RecipeCard.tsx, HomePage.tsx)
- **Functions**: PascalCase (RecipeCard, HomePage)
- **Props interfaces**: ComponentNameProps (RecipeCardProps)

### Variable/Function Naming
- **camelCase** for variables and functions
- **Descriptive names** - isLoading, handleClick, fetchRecipes
- **No single letters** except in loops (const [recipes, setRecipes])

### React Components
- **Functional components** only - no class components
- **Hooks for state** - useState, useEffect, useContext
- **Custom hooks** for shared logic
- **PropTypes or TypeScript interfaces** for props

Example:
```typescript
import { useState, useEffect } from 'react'

interface RecipeCardProps {
  id: string
  name: string
  servings: number
}

export default function RecipeCard({ id, name, servings }: RecipeCardProps) {
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    // Side effects here
  }, [id])

  const handleClick = () => {
    // Handle click
  }

  return (
    <div className="recipe-card">
      <h2>{name}</h2>
      <p>Servings: {servings}</p>
      <button onClick={handleClick}>View Recipe</button>
    </div>
  )
}
```

### Error Handling
- **Always use try-catch** for API calls
- **Console.error** for debugging
- **User-friendly error messages**

Example:
```typescript
const fetchRecipes = async () => {
  try {
    setIsLoading(true)
    const response = await fetch('/api/v1/recipes')
    if (!response.ok) throw new Error('Failed to fetch recipes')
    const data = await response.json()
    setRecipes(data)
  } catch (error) {
    console.error('Error fetching recipes:', error)
    setError('Failed to load recipes. Please try again.')
  } finally {
    setIsLoading(false)
  }
}
```

### Styling
- **CSS files in src/styles/** - index.css for global, componentName.css for component-specific
- **Global styles** - resets, variables, base styles
- **BEM convention** for class names: block__element--modifier

Example:
```css
/* src/styles/index.css - Global */
:root {
  --color-primary: #646cff;
  --spacing-lg: 2rem;
}

/* src/styles/RecipeCard.css - Component-specific */
.recipe-card {
  padding: var(--spacing-lg);
  border: 1px solid var(--color-primary);
}

.recipe-card__title {
  margin: 0;
}

.recipe-card__button--primary {
  background-color: var(--color-primary);
}
```

---

## 🔌 API Integration

### Backend Endpoints
Base URL: `http://localhost:8080/api/v1` (local) or `http://app:8080/api/v1` (Docker)

Common patterns:
```typescript
// GET
const getRecipes = async () => {
  const res = await fetch('/api/v1/recipes')
  if (!res.ok) throw new Error('Failed')
  return res.json()
}

// POST
const createRecipe = async (recipe: CreateRecipePayload) => {
  const res = await fetch('/api/v1/recipes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(recipe),
  })
  if (!res.ok) throw new Error('Failed')
  return res.json()
}

// PUT
const updateRecipe = async (id: string, updates: Partial<Recipe>) => {
  const res = await fetch(`/api/v1/recipes/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error('Failed')
  return res.json()
}

// DELETE
const deleteRecipe = async (id: string) => {
  const res = await fetch(`/api/v1/recipes/${id}`, {
    method: 'DELETE',
  })
  if (!res.ok) throw new Error('Failed')
}
```

### Vite Proxy Configuration
In `vite.config.ts`, API calls to `/api/*` are proxied:
```
/api/v1/recipes → http://localhost:8080/api/v1/recipes (local)
/api/v1/recipes → http://app:8080/api/v1/recipes (Docker)
```

### Response Types
Always create interfaces for API responses:

```typescript
// src/types/recipe.ts
export interface Recipe {
  id: string
  name: string
  description: string
  servings: number
  prepTime: number
  cookTime: number
  public: boolean
  createdAt: string
  updatedAt: string
}

export interface CreateRecipePayload {
  name: string
  description: string
  servings: number
  prepTime: number
  cookTime: number
  public?: boolean
}
```

---

## 🎯 Common Patterns

### Loading State
```typescript
const [recipes, setRecipes] = useState<Recipe[]>([])
const [isLoading, setIsLoading] = useState(false)
const [error, setError] = useState<string | null>(null)

const loadRecipes = async () => {
  try {
    setIsLoading(true)
    setError(null)
    const data = await fetch('/api/v1/recipes').then(r => r.json())
    setRecipes(data)
  } catch (err) {
    setError('Failed to load recipes')
  } finally {
    setIsLoading(false)
  }
}
```

### Custom Hook
```typescript
// src/hooks/useRecipes.ts
export const useRecipes = () => {
  const [recipes, setRecipes] = useState<Recipe[]>([])
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    fetchRecipes()
  }, [])

  const fetchRecipes = async () => {
    try {
      setIsLoading(true)
      const res = await fetch('/api/v1/recipes')
      const data = await res.json()
      setRecipes(data)
    } finally {
      setIsLoading(false)
    }
  }

  return { recipes, isLoading, refetch: fetchRecipes }
}
```

### Service Module
```typescript
// src/services/recipeService.ts
export const recipeService = {
  getAll: async (): Promise<Recipe[]> => {
    const res = await fetch('/api/v1/recipes')
    if (!res.ok) throw new Error('Failed to fetch')
    return res.json()
  },

  getById: async (id: string): Promise<Recipe> => {
    const res = await fetch(`/api/v1/recipes/${id}`)
    if (!res.ok) throw new Error('Failed to fetch')
    return res.json()
  },

  create: async (payload: CreateRecipePayload): Promise<Recipe> => {
    const res = await fetch('/api/v1/recipes', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    if (!res.ok) throw new Error('Failed to create')
    return res.json()
  },
}
```

---

## ⚡ Performance Tips

1. **Lazy load pages** with React.lazy()
2. **Memoize components** with React.memo() when needed
3. **Use useCallback** for event handlers passed as props
4. **Avoid inline function definitions** in JSX
5. **Code split** by routes

Example:
```typescript
import { lazy, Suspense } from 'react'

const RecipePage = lazy(() => import('./pages/RecipePage'))

export function App() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <RecipePage />
    </Suspense>
  )
}
```

---

## 🔧 Vite Configuration

Key settings in `vite.config.ts`:
- Port: 5173
- Host: 0.0.0.0 (accessible from Docker)
- API proxy: /api/* → backend
- Hot reload: Enabled by default

---

## 🚫 Important Notes

- **API proxy is automatic** - just call `/api/*` in code
- **Backend must be running** - frontend can't work without it
- **Strict TypeScript** - no implicit any allowed
- **Hot reload doesn't work** - check if Vite is running

---

## 📚 More Information

- Backend: See `services/backend/README.md`

