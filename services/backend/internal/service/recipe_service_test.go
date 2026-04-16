package service

import (
	"context"
	"errors"
	"mime/multipart"
	"testing"

	"github.com/H3nSte1n/recipe/internal/domain"
	apperrors "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/pkg/ai"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockRecipeRepo struct {
	mock.Mock
}

func (m *mockRecipeRepo) Create(ctx context.Context, recipe *domain.Recipe) error {
	args := m.Called(ctx, recipe)
	return args.Error(0)
}

func (m *mockRecipeRepo) Update(ctx context.Context, recipe *domain.Recipe) error {
	args := m.Called(ctx, recipe)
	return args.Error(0)
}

func (m *mockRecipeRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockRecipeRepo) GetByID(ctx context.Context, id string, nutritionLevel domain.NutritionDetailLevel) (*domain.Recipe, error) {
	args := m.Called(ctx, id, nutritionLevel)
	v, _ := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

func (m *mockRecipeRepo) ListByUserID(ctx context.Context, userID string, includePrivate bool) ([]domain.Recipe, error) {
	args := m.Called(ctx, userID, includePrivate)
	v, _ := args.Get(0).([]domain.Recipe)
	return v, args.Error(1)
}

func (m *mockRecipeRepo) ListPublic(ctx context.Context, page, pageSize int) ([]domain.Recipe, int64, error) {
	args := m.Called(ctx, page, pageSize)
	v, _ := args.Get(0).([]domain.Recipe)
	return v, args.Get(1).(int64), args.Error(2)
}

func (m *mockRecipeRepo) RunTx(ctx context.Context, fn func() error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) == nil {
		return fn()
	}
	return args.Error(0)
}

type mockRecipeUserRepo struct {
	mock.Mock
}

func (m *mockRecipeUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	v, _ := args.Get(0).(*domain.User)
	return v, args.Error(1)
}

type mockRecipeAIConfigRepo struct {
	mock.Mock
}

func (m *mockRecipeAIConfigRepo) GetDefaultConfig(ctx context.Context, userID string) (*domain.UserAIConfig, error) {
	args := m.Called(ctx, userID)
	v, _ := args.Get(0).(*domain.UserAIConfig)
	return v, args.Error(1)
}

type mockFileStore struct {
	mock.Mock
}

func (m *mockFileStore) UploadFile(ctx context.Context, file *multipart.FileHeader) (string, error) {
	args := m.Called(ctx, file)
	return args.String(0), args.Error(1)
}

func (m *mockFileStore) DeleteFile(ctx context.Context, fileURL string) error {
	args := m.Called(ctx, fileURL)
	return args.Error(0)
}

type mockAIModel struct {
	mock.Mock
}

func (m *mockAIModel) Parse(ctx context.Context, content string, contentType string) (*domain.Recipe, error) {
	args := m.Called(ctx, content, contentType)
	v, _ := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

func (m *mockAIModel) ParseInstructions(ctx context.Context, content string) (*[]domain.RecipeInstruction, error) {
	args := m.Called(ctx, content)
	v, _ := args.Get(0).(*[]domain.RecipeInstruction)
	return v, args.Error(1)
}

func (m *mockAIModel) CategorizeItems(ctx context.Context, items []string) (map[string]string, error) {
	args := m.Called(ctx, items)
	v, _ := args.Get(0).(map[string]string)
	return v, args.Error(1)
}

type mockURLParser struct {
	mock.Mock
}

func (m *mockURLParser) Parse(ctx context.Context, url string, aiModel ai.AIModel) (*domain.Recipe, error) {
	args := m.Called(ctx, url, aiModel)
	v, _ := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

type mockPDFParser struct {
	mock.Mock
}

func (m *mockPDFParser) Parse(ctx context.Context, pdfData []byte, aiModel ai.AIModel) (*domain.Recipe, error) {
	args := m.Called(ctx, pdfData, aiModel)
	v, _ := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

func newTestRecipeService(
	recipeRepo *mockRecipeRepo,
	userRepo *mockRecipeUserRepo,
	aiConfigRepo *mockRecipeAIConfigRepo,
	fileStore *mockFileStore,
	urlParser *mockURLParser,
	pdfParser *mockPDFParser,
) RecipeService {
	return NewRecipeService(recipeRepo, userRepo, aiConfigRepo, fileStore, zap.NewNop(), nil, urlParser, pdfParser)
}

func TestRecipeService_GetByID_Success(t *testing.T) {
	userID := "user-1"
	recipeID := "recipe-1"
	recipe := &domain.Recipe{ID: recipeID, UserID: userID, IsPrivate: false}

	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("GetByID", mock.Anything, recipeID, domain.NutritionDetailBase).Return(recipe, nil).Once()

	srv := newTestRecipeService(recipeRepo, new(mockRecipeUserRepo), new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	result, err := srv.GetByID(context.Background(), userID, recipeID, domain.NutritionDetailBase)

	require.NoError(t, err)
	require.Equal(t, recipe, result)
	recipeRepo.AssertExpectations(t)
}

func TestRecipeService_GetByID_NotFound(t *testing.T) {
	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("GetByID", mock.Anything, "recipe-1", domain.NutritionDetailBase).Return(nil, errors.New("not found")).Once()

	srv := newTestRecipeService(recipeRepo, new(mockRecipeUserRepo), new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	result, err := srv.GetByID(context.Background(), "user-1", "recipe-1", domain.NutritionDetailBase)

	require.Nil(t, result)
	require.Error(t, err)
	recipeRepo.AssertExpectations(t)
}

func TestRecipeService_GetByID_PrivateUnauthorized(t *testing.T) {
	recipeID := "recipe-1"
	recipe := &domain.Recipe{ID: recipeID, UserID: "other-user", IsPrivate: true}

	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("GetByID", mock.Anything, recipeID, domain.NutritionDetailBase).Return(recipe, nil).Once()

	srv := newTestRecipeService(recipeRepo, new(mockRecipeUserRepo), new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	result, err := srv.GetByID(context.Background(), "user-1", recipeID, domain.NutritionDetailBase)

	require.Nil(t, result)
	require.Error(t, err)
	recipeRepo.AssertExpectations(t)
}

func TestRecipeService_Create_Success(t *testing.T) {
	userID := "user-1"
	recipeID := "recipe-1"
	req := &domain.CreateRecipeRequest{
		Title:      "Test Recipe",
		SourceType: "MANUAL",
		Servings:   2,
	}
	expectedRecipe := &domain.Recipe{ID: recipeID, UserID: userID, Title: "Test Recipe"}

	recipeRepo := new(mockRecipeRepo)
	userRepo := new(mockRecipeUserRepo)
	userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID}, nil).Once()
	recipeRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *domain.Recipe) bool {
		return r.UserID == userID && r.Title == "Test Recipe"
	})).Return(nil).Once()
	recipeRepo.On("GetByID", mock.Anything, mock.AnythingOfType("string"), domain.NutritionDetailBase).Return(expectedRecipe, nil).Once()

	srv := newTestRecipeService(recipeRepo, userRepo, new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	result, err := srv.Create(context.Background(), userID, req)

	require.NoError(t, err)
	require.Equal(t, expectedRecipe, result)
	recipeRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestRecipeService_Create_UserNotFound(t *testing.T) {
	userRepo := new(mockRecipeUserRepo)
	userRepo.On("GetByID", mock.Anything, "user-1").Return(nil, errors.New("not found")).Once()

	srv := newTestRecipeService(new(mockRecipeRepo), userRepo, new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	result, err := srv.Create(context.Background(), "user-1", &domain.CreateRecipeRequest{})

	require.Nil(t, result)
	require.Error(t, err)
	userRepo.AssertExpectations(t)
}

func TestRecipeService_Create_RepoError(t *testing.T) {
	userID := "user-1"
	userRepo := new(mockRecipeUserRepo)
	userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID}, nil).Once()

	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

	srv := newTestRecipeService(recipeRepo, userRepo, new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	result, err := srv.Create(context.Background(), userID, &domain.CreateRecipeRequest{})

	require.Nil(t, result)
	require.Error(t, err)
	recipeRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestRecipeService_Update_NotFound(t *testing.T) {
	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("GetByID", mock.Anything, "recipe-1", domain.NutritionDetailBase).Return(nil, errors.New("not found")).Once()

	srv := newTestRecipeService(recipeRepo, new(mockRecipeUserRepo), new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	result, err := srv.Update(context.Background(), "user-1", "recipe-1", &domain.CreateRecipeRequest{})

	require.Nil(t, result)
	require.ErrorIs(t, err, apperrors.ErrNotFound)
	recipeRepo.AssertExpectations(t)
}

func TestRecipeService_Update_Unauthorized(t *testing.T) {
	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("GetByID", mock.Anything, "recipe-1", domain.NutritionDetailBase).
		Return(&domain.Recipe{ID: "recipe-1", UserID: "other-user"}, nil).Once()

	srv := newTestRecipeService(recipeRepo, new(mockRecipeUserRepo), new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	result, err := srv.Update(context.Background(), "user-1", "recipe-1", &domain.CreateRecipeRequest{})

	require.Nil(t, result)
	require.ErrorIs(t, err, apperrors.ErrUnauthorized)
	recipeRepo.AssertExpectations(t)
}

func TestRecipeService_Delete_Success(t *testing.T) {
	userID := "user-1"
	recipeID := "recipe-1"
	recipe := &domain.Recipe{ID: recipeID, UserID: userID}

	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("GetByID", mock.Anything, recipeID, domain.NutritionDetailBase).Return(recipe, nil).Once()
	recipeRepo.On("RunTx", mock.Anything, mock.Anything).Return(nil).Once()
	recipeRepo.On("Delete", mock.Anything, recipeID).Return(nil).Once()

	srv := newTestRecipeService(recipeRepo, new(mockRecipeUserRepo), new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	err := srv.Delete(context.Background(), userID, recipeID)

	require.NoError(t, err)
	recipeRepo.AssertExpectations(t)
}

func TestRecipeService_Delete_NotFound(t *testing.T) {
	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("GetByID", mock.Anything, "recipe-1", domain.NutritionDetailBase).Return(nil, errors.New("not found")).Once()

	srv := newTestRecipeService(recipeRepo, new(mockRecipeUserRepo), new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	err := srv.Delete(context.Background(), "user-1", "recipe-1")

	require.ErrorIs(t, err, apperrors.ErrNotFound)
	recipeRepo.AssertExpectations(t)
}

func TestRecipeService_Delete_Unauthorized(t *testing.T) {
	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("GetByID", mock.Anything, "recipe-1", domain.NutritionDetailBase).
		Return(&domain.Recipe{ID: "recipe-1", UserID: "other-user"}, nil).Once()

	srv := newTestRecipeService(recipeRepo, new(mockRecipeUserRepo), new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	err := srv.Delete(context.Background(), "user-1", "recipe-1")

	require.ErrorIs(t, err, apperrors.ErrUnauthorized)
	recipeRepo.AssertExpectations(t)
}

func TestRecipeService_ListUserRecipes_Success(t *testing.T) {
	userID := "user-1"
	recipes := []domain.Recipe{{ID: "recipe-1", UserID: userID}, {ID: "recipe-2", UserID: userID}}

	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("ListByUserID", mock.Anything, userID, true).Return(recipes, nil).Once()

	srv := newTestRecipeService(recipeRepo, new(mockRecipeUserRepo), new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	result, err := srv.ListUserRecipes(context.Background(), userID)

	require.NoError(t, err)
	require.Equal(t, recipes, result)
	recipeRepo.AssertExpectations(t)
}

func TestRecipeService_ListUserRecipes_Error(t *testing.T) {
	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("ListByUserID", mock.Anything, "user-1", true).Return(nil, errors.New("db error")).Once()

	srv := newTestRecipeService(recipeRepo, new(mockRecipeUserRepo), new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	result, err := srv.ListUserRecipes(context.Background(), "user-1")

	require.Error(t, err)
	require.Nil(t, result)
	recipeRepo.AssertExpectations(t)
}

func TestRecipeService_ListPublicRecipes_Success(t *testing.T) {
	recipes := []domain.Recipe{{ID: "recipe-1"}, {ID: "recipe-2"}}
	var total int64 = 2

	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("ListPublic", mock.Anything, 1, 10).Return(recipes, total, nil).Once()

	srv := newTestRecipeService(recipeRepo, new(mockRecipeUserRepo), new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	result, count, err := srv.ListPublicRecipes(context.Background(), 1, 10)

	require.NoError(t, err)
	require.Equal(t, recipes, result)
	require.Equal(t, total, count)
	recipeRepo.AssertExpectations(t)
}

func TestRecipeService_ListPublicRecipes_Error(t *testing.T) {
	recipeRepo := new(mockRecipeRepo)
	recipeRepo.On("ListPublic", mock.Anything, 1, 10).Return(nil, int64(0), errors.New("db error")).Once()

	srv := newTestRecipeService(recipeRepo, new(mockRecipeUserRepo), new(mockRecipeAIConfigRepo), new(mockFileStore), new(mockURLParser), new(mockPDFParser))
	result, count, err := srv.ListPublicRecipes(context.Background(), 1, 10)

	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, int64(0), count)
	recipeRepo.AssertExpectations(t)
}
