package repository

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/errors"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type PlannedMealRepository interface {
	// LockOwnedRecipe returns the user's recipe and keeps it from being deleted until the
	// surrounding transaction ends. It returns ErrNotFound when the user does not own it.
	LockOwnedRecipe(ctx context.Context, userID, recipeID string) (*domain.PlannedMealRecipe, error)
	GetByID(ctx context.Context, userID, id string) (*domain.PlannedMeal, error)
	// GetByIDForUpdate is GetByID that also locks the meal row until the surrounding transaction ends.
	GetByIDForUpdate(ctx context.Context, userID, id string) (*domain.PlannedMeal, error)
	// CountByUser returns how many meals the user has planned in total.
	CountByUser(ctx context.Context, userID string) (int64, error)
	Create(ctx context.Context, meal *domain.PlannedMeal) error
	Update(ctx context.Context, meal *domain.PlannedMeal) error
	Delete(ctx context.Context, userID, id string) error
	// List returns the user's meals from..to (inclusive, "YYYY-MM-DD") ordered by date, plus the total count.
	List(ctx context.Context, userID, from, to string, limit, offset int) ([]domain.PlannedMeal, int64, error)
	WithTypedTransaction(ctx context.Context, fn func(PlannedMealRepository) error) error
}

type PlannedMealRepositoryImpl struct {
	*BaseRepository
}

func NewPlannedMealRepository(db *gorm.DB) PlannedMealRepository {
	return &PlannedMealRepositoryImpl{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *PlannedMealRepositoryImpl) WithTypedTransaction(ctx context.Context, fn func(PlannedMealRepository) error) error {
	return r.RunInTransaction(ctx, func(tx *gorm.DB) error {
		txRepo := &PlannedMealRepositoryImpl{BaseRepository: NewBaseRepository(tx)}
		return fn(txRepo)
	})
}

// Unlike the other repositories this one uses hand-written SQL rather than GORM models: cook_date
// must stay a date-only string (no time zone shifts) and every read joins the owning recipe.
type plannedMealRow struct {
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

func (row plannedMealRow) toDomain() domain.PlannedMeal {
	return domain.PlannedMeal{
		ID:        row.ID,
		UserID:    row.UserID,
		RecipeID:  row.RecipeID,
		CookDate:  row.CookDate,
		MealType:  row.MealType,
		Servings:  row.Servings,
		Notes:     row.Notes,
		Recipe:    domain.PlannedMealRecipe{ID: row.RecipeID, Title: row.RecipeTitle, Status: row.RecipeStatus},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

// The join on (recipe id, user id) mirrors the composite foreign key, so a meal is only ever
// shown together with a recipe its user owns. cook_date is formatted in SQL to stay date-only.
const selectPlannedMeals = `SELECT p.id, p.user_id, p.recipe_id, TO_CHAR(p.cook_date, 'YYYY-MM-DD') AS cook_date,
    p.meal_type, p.servings, p.notes, r.title AS recipe_title, r.status AS recipe_status,
    p.created_at, p.updated_at
    FROM planned_meals p JOIN recipes r ON r.id = p.recipe_id AND r.user_id = p.user_id`

// plannedMealDBError turns constraint violations into client-safe errors. Anything else is
// returned unchanged so the handler logs it and answers with a generic 500.
func plannedMealDBError(err error) error {
	var pgErr *pgconn.PgError
	if stderrors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // the recipe was deleted or is not owned by the user
			return errors.New("recipe not found", "INVALID")
		case "23514", "23502", "22001", "22007", "22008":
			return errors.New("invalid planned meal", "INVALID")
		}
	}
	return err
}

func (r *PlannedMealRepositoryImpl) LockOwnedRecipe(ctx context.Context, userID, recipeID string) (*domain.PlannedMealRecipe, error) {
	var recipe domain.PlannedMealRecipe
	result := r.DB.WithContext(ctx).
		Raw(`SELECT id, title, status FROM recipes WHERE id = ? AND user_id = ? FOR KEY SHARE`, recipeID, userID).
		Scan(&recipe)
	if result.Error != nil {
		return nil, plannedMealDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, errors.ErrNotFound
	}
	return &recipe, nil
}

func (r *PlannedMealRepositoryImpl) GetByID(ctx context.Context, userID, id string) (*domain.PlannedMeal, error) {
	return r.getByID(ctx, userID, id, "")
}

func (r *PlannedMealRepositoryImpl) GetByIDForUpdate(ctx context.Context, userID, id string) (*domain.PlannedMeal, error) {
	return r.getByID(ctx, userID, id, " FOR UPDATE OF p") // lock the meal only, not the joined recipe
}

func (r *PlannedMealRepositoryImpl) getByID(ctx context.Context, userID, id, lock string) (*domain.PlannedMeal, error) {
	var row plannedMealRow
	result := r.DB.WithContext(ctx).Raw(selectPlannedMeals+` WHERE p.id = ? AND p.user_id = ?`+lock, id, userID).Scan(&row)
	if result.Error != nil {
		return nil, plannedMealDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, errors.ErrNotFound
	}
	meal := row.toDomain()
	return &meal, nil
}

func (r *PlannedMealRepositoryImpl) CountByUser(ctx context.Context, userID string) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).Raw(`SELECT COUNT(*) FROM planned_meals WHERE user_id = ?`, userID).Scan(&total).Error
	return total, plannedMealDBError(err)
}

func (r *PlannedMealRepositoryImpl) Create(ctx context.Context, meal *domain.PlannedMeal) error {
	err := r.DB.WithContext(ctx).Raw(`INSERT INTO planned_meals (user_id, recipe_id, cook_date, meal_type, servings, notes)
		VALUES (?, ?, CAST(? AS DATE), ?, ?, ?) RETURNING id`,
		meal.UserID, meal.RecipeID, meal.CookDate, meal.MealType, meal.Servings, meal.Notes).Scan(&meal.ID).Error
	return plannedMealDBError(err)
}

func (r *PlannedMealRepositoryImpl) Update(ctx context.Context, meal *domain.PlannedMeal) error {
	result := r.DB.WithContext(ctx).Exec(`UPDATE planned_meals SET recipe_id = ?, cook_date = CAST(? AS DATE), meal_type = ?,
		servings = ?, notes = ? WHERE id = ? AND user_id = ?`,
		meal.RecipeID, meal.CookDate, meal.MealType, meal.Servings, meal.Notes, meal.ID, meal.UserID)
	if result.Error != nil {
		return plannedMealDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

func (r *PlannedMealRepositoryImpl) Delete(ctx context.Context, userID, id string) error {
	result := r.DB.WithContext(ctx).Exec(`DELETE FROM planned_meals WHERE id = ? AND user_id = ?`, id, userID)
	if result.Error != nil {
		return plannedMealDBError(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

func (r *PlannedMealRepositoryImpl) List(ctx context.Context, userID, from, to string, limit, offset int) ([]domain.PlannedMeal, int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).Raw(`SELECT COUNT(*) FROM planned_meals WHERE user_id = ? AND cook_date >= CAST(? AS DATE) AND cook_date <= CAST(? AS DATE)`,
		userID, from, to).Scan(&total).Error
	if err != nil {
		return nil, 0, plannedMealDBError(err)
	}

	var rows []plannedMealRow
	err = r.DB.WithContext(ctx).Raw(selectPlannedMeals+` WHERE p.user_id = ? AND p.cook_date >= CAST(? AS DATE) AND p.cook_date <= CAST(? AS DATE)
		ORDER BY p.cook_date ASC, p.id ASC LIMIT ? OFFSET ?`, userID, from, to, limit, offset).Scan(&rows).Error
	if err != nil {
		return nil, 0, plannedMealDBError(err)
	}

	meals := make([]domain.PlannedMeal, 0, len(rows))
	for _, row := range rows {
		meals = append(meals, row.toDomain())
	}
	return meals, total, nil
}
