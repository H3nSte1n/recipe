package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/H3nSte1n/recipe/internal/mealplan/application"
	"github.com/H3nSte1n/recipe/internal/mealplan/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type row struct {
	ID           string    `gorm:"column:id"`
	UserID       string    `gorm:"column:user_id"`
	RecipeID     string    `gorm:"column:recipe_id"`
	CookDate     string    `gorm:"column:cook_date"`
	MealType     *string   `gorm:"column:meal_type"`
	Servings     *int      `gorm:"column:servings"`
	Notes        *string   `gorm:"column:notes"`
	RecipeTitle  string    `gorm:"column:recipe_title"`
	RecipeStatus string    `gorm:"column:recipe_status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

const selectMeals = `SELECT p.id, p.user_id, p.recipe_id, TO_CHAR(p.cook_date, 'YYYY-MM-DD') AS cook_date,
    p.meal_type, p.servings, p.notes, r.title AS recipe_title, r.status AS recipe_status,
    p.created_at, p.updated_at
    FROM planned_meals p JOIN recipes r ON r.id = p.recipe_id AND r.user_id = p.user_id`

func mapped(r row) (domain.PlannedMeal, error) {
	id, err := uuid.Parse(r.ID)
	if err != nil {
		return domain.PlannedMeal{}, err
	}
	user, err := uuid.Parse(r.UserID)
	if err != nil {
		return domain.PlannedMeal{}, err
	}
	recipe, err := uuid.Parse(r.RecipeID)
	if err != nil {
		return domain.PlannedMeal{}, err
	}
	date, err := domain.ParseDate(r.CookDate)
	if err != nil {
		return domain.PlannedMeal{}, err
	}
	return domain.PlannedMeal{ID: id, UserID: user, RecipeID: recipe, CookDate: date,
		MealType: r.MealType, Servings: r.Servings, Notes: r.Notes,
		Recipe:    domain.RecipeSummary{ID: recipe, Title: r.RecipeTitle, Status: r.RecipeStatus},
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}

func dbError(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23503":
			return application.ErrNotFound // deleted or inaccessible referenced row
		case "23514", "23502", "22001", "22007", "22008":
			return application.ErrInvalid
		}
	}
	return err
}

func (r *Repository) ownedRecipe(ctx context.Context, tx *gorm.DB, user, recipe uuid.UUID) error {
	var found struct{ ID string }
	result := tx.WithContext(ctx).Raw(`SELECT id FROM recipes WHERE id = ? AND user_id = ? FOR KEY SHARE`, recipe, user).Scan(&found)
	if result.Error != nil {
		return dbError(result.Error)
	}
	if result.RowsAffected == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateWithOwnedRecipe(ctx context.Context, user uuid.UUID, f domain.Fields) (domain.PlannedMeal, error) {
	id := uuid.New()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.ownedRecipe(ctx, tx, user, f.RecipeID); err != nil {
			return err
		}
		return dbError(tx.Exec(`INSERT INTO planned_meals (id, user_id, recipe_id, cook_date, meal_type, servings, notes)
			VALUES (?, ?, ?, CAST(? AS DATE), ?, ?, ?)`, id, user, f.RecipeID, f.CookDate.String(), f.MealType, f.Servings, f.Notes).Error)
	})
	if err != nil {
		return domain.PlannedMeal{}, dbError(err)
	}
	return r.Get(ctx, user, id)
}

func (r *Repository) ReplaceWithOwnedRecipe(ctx context.Context, user, id uuid.UUID, f domain.Fields) (domain.PlannedMeal, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.ownedRecipe(ctx, tx, user, f.RecipeID); err != nil {
			return err
		}
		result := tx.Exec(`UPDATE planned_meals SET recipe_id = ?, cook_date = CAST(? AS DATE), meal_type = ?,
			servings = ?, notes = ? WHERE id = ? AND user_id = ?`, f.RecipeID, f.CookDate.String(), f.MealType, f.Servings, f.Notes, id, user)
		if result.Error != nil {
			return dbError(result.Error)
		}
		if result.RowsAffected == 0 {
			return application.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return domain.PlannedMeal{}, dbError(err)
	}
	return r.Get(ctx, user, id)
}

func (r *Repository) Get(ctx context.Context, user, id uuid.UUID) (domain.PlannedMeal, error) {
	var record row
	result := r.db.WithContext(ctx).Raw(selectMeals+` WHERE p.id = ? AND p.user_id = ?`, id, user).Scan(&record)
	if result.Error != nil {
		return domain.PlannedMeal{}, dbError(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.PlannedMeal{}, application.ErrNotFound
	}
	return mapped(record)
}

func (r *Repository) List(ctx context.Context, user uuid.UUID, from, to domain.Date, page, pageSize int) ([]domain.PlannedMeal, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM planned_meals WHERE user_id = ? AND cook_date >= CAST(? AS DATE) AND cook_date <= CAST(? AS DATE)`,
		user, from.String(), to.String()).Scan(&total).Error; err != nil {
		return nil, 0, dbError(err)
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(selectMeals+` WHERE p.user_id = ? AND p.cook_date >= CAST(? AS DATE) AND p.cook_date <= CAST(? AS DATE)
		ORDER BY p.cook_date ASC, p.id ASC LIMIT ? OFFSET ?`, user, from.String(), to.String(), pageSize, (page-1)*pageSize).Scan(&rows).Error
	if err != nil {
		return nil, 0, dbError(err)
	}
	items := make([]domain.PlannedMeal, 0, len(rows))
	for _, record := range rows {
		item, err := mapped(record)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (r *Repository) Delete(ctx context.Context, user, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Exec(`DELETE FROM planned_meals WHERE id = ? AND user_id = ?`, id, user)
	if result.Error != nil {
		return dbError(result.Error)
	}
	if result.RowsAffected == 0 {
		return application.ErrNotFound
	}
	return nil
}
