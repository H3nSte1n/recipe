# Recipe App 🍳

A modern recipe management platform with AI-powered features, smart shopping lists, and intelligent store navigation.

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-blue.svg)](https://www.postgresql.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Backend CI](https://github.com/H3nSte1n/recipe/actions/workflows/backend-ci.yml/badge.svg?branch=main)](https://github.com/H3nSte1n/recipe/actions/workflows/backend-ci.yml)

---

## 📋 Overview

Recipe App is a full-stack application that helps users manage their recipes, create smart shopping lists, and optimize their grocery shopping experience. It features AI-powered recipe parsing, automatic ingredient categorization, and store-specific shopping list sorting.

### Key Features

- 🤖 **AI-Powered Recipe Import** - Import recipes from URLs or PDFs using Claude/GPT
- 🛒 **Smart Shopping Lists** - Create lists from recipes with automatic ingredient categorization
- 🏪 **Store Navigation** - Sort shopping lists by store layout for efficient shopping
- 📱 **Recipe Management** - Full CRUD operations with nutrition tracking
- 🔐 **Secure Authentication** - JWT-based auth with password reset
- 📊 **Multi-Store Support** - Albert Heijn, Jumbo, Lidl, Aldi, Rewe, Edeka

---

## 🏗️ Project Structure

```
recipe/
├── services/
│   ├── backend/          # Go REST API
│   │   ├── cmd/          # Application entry point
│   │   ├── internal/     # Core business logic
│   │   ├── pkg/          # Reusable packages
│   │   └── migrations/   # Database migrations
│   └── frontend/         # React + TypeScript Frontend
│       └── src/     # React components and styles
├── docker-compose.yml    # Docker orchestration
└── LICENSE              # MIT License
```

---

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.25+ (for local backend development)
- Node.js 24 LTS (for local frontend development)
- PostgreSQL 18 (managed by Docker)

### Start the Application

```bash
# Clone the repository
git clone https://github.com/H3nSte1n/recipe.git
cd recipe

# Start all services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f app
```

The API will be available at `http://localhost:8080`

---

## 📚 Documentation

### Backend API
The backend is a fully-featured REST API built with Go.

**📖 [Complete Backend Documentation](services/backend/README.md)**

## 🎯 Features in Detail

### 1. Recipe Management
- Create, read, update, and delete recipes
- Track nutrition information (calories, macros)
- Upload recipe images
- Import from websites or PDFs
- Public/private recipe visibility

### 2. AI-Powered Features
- **URL Import**: Parse recipes from any cooking website
- **PDF Import**: Extract recipes from PDF cookbooks
- **Smart Categorization**: Automatically categorize ingredients
- Support for Claude 3.5 Sonnet, GPT-4, and more

### 3. Shopping Lists
- Create lists manually or from recipes
- Auto-scale ingredients based on servings
- Check off items while shopping
- Add notes to items
- Link items to source recipes

### 4. Smart Store Sorting
Sort your shopping list to match your store's layout:

```bash
# Sort for Albert Heijn (entrance to exit)
GET /api/v1/shopping-lists/:id?sort_by=store&store_name=Albert%20Heijn

# Sort in reverse (exit to entrance)
GET /api/v1/shopping-lists/:id?sort_by=store&store_name=Jumbo&sort_direction=desc

# Sort by category, name, or checked status
GET /api/v1/shopping-lists/:id?sort_by=category
```

**Supported Stores:**
- 🇳🇱 Albert Heijn, Jumbo, Lidl, Aldi
- 🇩🇪 Rewe, Edeka

---

## 📊 Tech Stack

### Backend
- **Language**: Go 1.25
- **Framework**: Gin
- **Database**: PostgreSQL 18
- **ORM**: GORM
- **Authentication**: JWT
- **AI**: Claude API, OpenAI API
- **Storage**: Local/S3
- **Logging**: Zap

### Infrastructure
- **Containerization**: Docker
- **Orchestration**: Docker Compose
- **Database Migrations**: golang-migrate
- **Hot Reload**: Air

---

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 👨‍💻 Author

**Henry Steinhauer**

- Email: hello@steinhauer.dev
- Portfolio: https://steinhauer.dev

---

## 🎯 Roadmap

### Current Status
- ✅ Backend API
- ✅ Authentication & User Management
- ✅ Recipe Management
- ✅ Shopping Lists
- ✅ AI Integration
- ✅ Store Sorting
- ✅ Frontend (In Progress)
- ✅ Docker Containerization

### Upcoming Features
- [ ] User dashboard and recipe library UI
- [ ] Recipe search and filtering
- [ ] User authentication interface
- [ ] Recipe sharing between users
- [ ] Meal planning calendar
- [ ] Nutrition goal tracking
- [ ] Recipe recommendations
- [ ] Mobile app
- [ ] Multi-language support

---

## 📞 Support

For questions, issues, or feature requests:
- 📧 Email: hello@steinhauer.dev
- 📖 Documentation: [Backend README](services/backend/README.md)
- 🐛 Issues: Create an issue on GitHub
