# Recipe App - Frontend

A modern React + TypeScript frontend for the Recipe App. Provides a user-friendly interface for managing recipes, shopping lists, and AI-powered features.

## 🛠️ Development

### Prerequisites
- Node.js 24 LTS (recommended) or 18+
- npm (included with Node.js)

### Setup

```bash
npm install

cp .env.example .env.local
```

### Development Server

```bash
npm run dev

# The app will be available at http://localhost:5173
```

### Linting & Type Checking

```bash
npm run lint
npm run type-check
```

## 📁 Project Structure

```
src/
├── main.tsx           # React entry point
├── App.tsx            # Main app component
├── App.css            # App styles
├── index.css          # Global styles
└── vite-env.d.ts      # Vite type definitions
```

## 🔧 Configuration

### Environment Variables

Create a `.env.local` file:

```env
VITE_API_URL=http://localhost:8080
VITE_ENV=development
```

### In Docker

The frontend is containerized and:
- Runs on port 5173
- Proxies API requests to the backend service (`http://app:8080`)
