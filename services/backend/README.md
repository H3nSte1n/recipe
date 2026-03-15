# Recipe App - Backend API

A feature-rich RESTful API for managing recipes, shopping lists, and AI-powered recipe parsing. Built with Go, PostgreSQL, and modern best practices.

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-blue.svg)](https://www.postgresql.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

---

## 🌟 Features

### 🔐 Authentication & User Management
- **JWT-based authentication** with secure token handling
- User registration and login
- Password reset functionality with email tokens
- User profile management
- Account deletion with cascade cleanup

### 🍳 Recipe Management
- **CRUD operations** for recipes (Create, Read, Update, Delete)
- Public and private recipe visibility
- Recipe ratings and reviews
- Nutrition information tracking (calories, protein, carbs, fats, fiber)
- **Multi-level nutrition detail** (base, detailed, full)
- Ingredient management with amounts and units
- Step-by-step cooking instructions
- Recipe images with flexible storage (local or S3)
- Sub-recipe support for complex dishes

### 🤖 AI-Powered Features
- **Recipe import from URLs** using intelligent parsing
- **PDF recipe parsing** with OCR capabilities
- **Plain text instruction parsing** into structured steps
- **AI categorization** of shopping list items
- Support for multiple AI providers:
  - **Claude** (3.5 Sonnet, 3 Opus, 3 Sonnet, 3 Haiku)
  - **GPT** (GPT-4 Turbo, GPT-4, GPT-3.5 Turbo)
- Configurable AI models per user
- Custom AI settings and preferences

### 🛒 Shopping List Management
- Create and manage multiple shopping lists
- Add items manually or from recipes
- **AI-powered item categorization** (9 categories)
- Automatic ingredient scaling based on servings
- Check off items while shopping
- Item notes and quantity tracking
- Link items back to source recipes

### 🏪 Smart Store Sorting
- **Sort shopping lists by store layout** for optimal shopping routes
- Support for major store chains:
  - 🇳🇱 **Netherlands**: Albert Heijn, Jumbo, Lidl, Aldi
  - 🇩🇪 **Germany**: Rewe, Edeka
- **Flexible sorting options**:
  - By store layout (entrance to exit or reverse)
  - By category (PRODUCE, MEAT, DAIRY, BAKERY, etc.)
  - By name (A-Z or Z-A)
  - By amount
  - By checked status
  - By creation date
- Store name lookup (no UUID required)
- Ascending/descending direction control

### 📦 Additional Features
- **File storage** with local and S3 support
- **Database migrations** with version control
- **Email notifications** via SMTP
- **URL scraping** for recipe imports
- **PDF parsing** for recipe extraction
- **Comprehensive error handling**
- **Structured logging** with Zap
- **Request validation**
- **CORS support**
- **Hot reload** in development with Air

---

## 🏗️ Architecture

### Tech Stack
- **Language**: Go 1.25
- **Framework**: Gin (HTTP router)
- **Database**: PostgreSQL 18
- **ORM**: GORM
- **Authentication**: JWT (golang-jwt)
- **AI Integration**: Claude API, OpenAI API
- **Storage**: Local filesystem or AWS S3
- **Email**: SMTP
- **Logging**: Zap
- **Migrations**: golang-migrate
- **Dev Tools**: Air (hot reload)

### Project Structure
```
backend/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── domain/                  # Domain models
│   │   ├── user.go
│   │   ├── profile.go
│   │   ├── recipe.go
│   │   ├── shopping_list.go
│   │   ├── store_chain.go
│   │   └── ai_config.go
│   ├── handler/                 # HTTP handlers (controllers)
│   │   ├── user_handler.go
│   │   ├── recipe_handler.go
│   │   ├── shopping_list_handler.go
│   │   ├── store_chain_handler.go
│   │   └── ai_config_handler.go
│   ├── service/                 # Business logic
│   │   ├── user_service.go
│   │   ├── recipe_service.go
│   │   ├── shopping_list_handler.go
│   │   └── store_chain_service.go
│   ├── repository/              # Data access layer
│   │   ├── user_repository.go
│   │   ├── recipe_repository.go
│   │   ├── shopping_list_repository.go
│   │   └── store_chain_repository.go
│   ├── middleware/              # HTTP middleware
│   │   ├── auth.go
│   │   └── context.go
│   ├── router/                  # Route definitions
│   │   └── router.go
│   └── errors/                  # Custom error types
│       └── errors.go
├── pkg/                         # Reusable packages
│   ├── ai/                      # AI model integrations
│   ├── config/                  # Configuration management
│   ├── database/                # Database utilities
│   ├── email/                   # Email service
│   ├── pdfparser/               # PDF parsing
│   ├── storage/                 # File storage
│   ├── urlparser/               # URL scraping
│   └── validator/               # Input validation
├── migrations/                  # Database migrations
│   ├── 000001_init_schema.up.sql
│   ├── 000002_create_user.up.sql
│   ├── ...
│   └── 000012_seed_ai_models.up.sql
├── make/                        # Makefile includes
│   ├── app.mk
│   ├── db.mk
│   ├── dev.mk
│   └── test.mk
├── Dockerfile                   # Docker configuration
├── Makefile                     # Build automation
├── go.mod                       # Go dependencies
├── go.sum                       # Dependency checksums
└── env.development.yaml         # Environment config
```

### Design Patterns
- **Clean Architecture** (Handler → Service → Repository)
- **Dependency Injection** for testability
- **Repository Pattern** for data access
- **Factory Pattern** for storage providers
- **Strategy Pattern** for AI models
- **Middleware Pattern** for cross-cutting concerns

---

## 🚀 Getting Started

### Prerequisites
- **Go 1.25+**
- **Docker & Docker Compose**
- **PostgreSQL 18** (if running locally)
- **Make** (optional, for build commands)

### Installation

#### 1. Clone the Repository
```bash
git clone https://github.com/H3nSte1n/recipe.git
cd recipe/services/backend
```

#### 2. Configure Environment
```bash
# Copy the sample configuration
cp env.development.yaml.sample env.development.yaml

# Edit the configuration with your values
nano env.development.yaml
```

**Key configurations to update:**
```yaml
jwt:
  secret: your-super-secret-key-here  # Change this!

smtp:
  user: your-email@gmail.com
  password: your-app-specific-password

ai:
  openai_api_key: your_openai_api_key
  anthropic_api_key: your_anthropic_api_key

storage:
  type: local  # or 's3'
  aws:
    access_key_id: your-aws-key
    secret_access_key: your-aws-secret
```

#### 3. Start with Docker Compose
```bash
# From the project root
docker-compose up -d

# Check if services are running
docker-compose ps

# View logs
docker-compose logs -f app
```

The API will be available at `http://localhost:8080`

---

## 🛠️ Development

### Using Make Commands

```bash
# Start the application
make run

# Run migrations
make migrate-up

# Rollback migrations
make migrate-down

# Create a new migration
make migrate-create NAME=add_new_feature

# Run tests
make test

# Build the binary
make build

# Clean build artifacts
make clean
```

### Database Management

```bash
# Connect to database
docker-compose exec db psql -U postgres -d recipe_db

# Run a specific migration
make migrate-up VERSION=5

# Check migration status
migrate -path migrations -database "postgresql://..." version

# Seed store chains
docker-compose exec -T db psql -U postgres -d recipe_db < migrations/000010_seed_store_chains.up.sql

# Seed AI models
docker-compose exec -T db psql -U postgres -d recipe_db < migrations/000012_seed_ai_models.up.sql
```

### Hot Reload Configuration

The project uses Air for hot reload during development. Configuration in `.air.toml`:
- Watches: `*.go`, `*.yaml` files
- Excludes: `tmp/`, `vendor/`, `*_test.go`
- Auto-rebuild on file changes

---

## 📜 License

This project is licensed under the MIT License - see the LICENSE file for details.

---

## 👨‍💻 Author

**Henry Steinhauer**

- Email: hello@steinhauer.dev
- Portfolio: https://steinhauer.dev

---

## 🎯 Future Enhancements

### Planned Features
- [ ] Recipe sharing between users
- [ ] Meal planning calendar
- [ ] Nutrition goal tracking
- [ ] Recipe collections/cookbooks
- [ ] Social features (likes, comments)
- [ ] Recipe recommendations
- [ ] Grocery price tracking
- [ ] Multi-language support
- [ ] Mobile app integration

### Technical Improvements
- [ ] Rate limiting
- [ ] Caching layer (Redis)
- [ ] Full-text search (Elasticsearch)
- [ ] WebSocket support for real-time updates
- [ ] Prometheus metrics
- [ ] OpenAPI/Swagger documentation
- [ ] CI/CD pipeline
