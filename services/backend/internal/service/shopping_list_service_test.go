package service

import (
	"context"
	"errors"
	"github.com/H3nSte1n/recipe/internal/domain"
	internalErr "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"
)

func itemIDs(items []domain.ShoppingListItem) []string {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	return ids
}

type mockShoppingListRepository struct {
	mock.Mock
}

func (m *mockShoppingListRepository) GetByID(ctx context.Context, listID string) (*domain.ShoppingList, error) {
	args := m.Called(ctx, listID)
	v, _ := args.Get(0).(*domain.ShoppingList)
	return v, args.Error(1)
}

func (m *mockShoppingListRepository) GetItemByID(ctx context.Context, itemID string) (*domain.ShoppingListItem, error) {
	args := m.Called(ctx, itemID)
	v, _ := args.Get(0).(*domain.ShoppingListItem)
	return v, args.Error(1)
}

func (m *mockShoppingListRepository) Create(ctx context.Context, list *domain.ShoppingList) error {
	args := m.Called(ctx, list)
	return args.Error(0)
}

func (m *mockShoppingListRepository) Update(ctx context.Context, list *domain.ShoppingList) error {
	args := m.Called(ctx, list)
	return args.Error(0)
}

func (m *mockShoppingListRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockShoppingListRepository) ListByUserID(ctx context.Context, userID string) ([]domain.ShoppingList, error) {
	args := m.Called(ctx, userID)
	v, _ := args.Get(0).([]domain.ShoppingList)
	return v, args.Error(1)
}

func (m *mockShoppingListRepository) AddItems(ctx context.Context, items []domain.ShoppingListItem) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

func (m *mockShoppingListRepository) UpdateItem(ctx context.Context, item *domain.ShoppingListItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *mockShoppingListRepository) DeleteItem(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockShoppingListRecipeRepository struct {
	mock.Mock
}

func (m *mockShoppingListRecipeRepository) GetByID(ctx context.Context, id string, nutritionLevel domain.NutritionDetailLevel) (*domain.Recipe, error) {
	args := m.Called(ctx, id, nutritionLevel)
	v, _ := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

type mockStoreChainService struct {
	mock.Mock
}

func (m *mockStoreChainService) GetChain(ctx context.Context, chainID string) (*domain.StoreChain, error) {
	args := m.Called(ctx, chainID)
	v, _ := args.Get(0).(*domain.StoreChain)
	return v, args.Error(1)
}

func (m *mockStoreChainService) GetChainByName(ctx context.Context, name string, country string) (*domain.StoreChain, error) {
	args := m.Called(ctx, name, country)
	v, _ := args.Get(0).(*domain.StoreChain)
	return v, args.Error(1)
}

func (m *mockStoreChainService) ListChains(ctx context.Context, country string) ([]domain.StoreChain, error) {
	args := m.Called(ctx, country)
	v, _ := args.Get(0).([]domain.StoreChain)
	return v, args.Error(1)
}

func (m *mockStoreChainService) OrganizeShoppingList(ctx context.Context, list *domain.ShoppingList, chainID string) error {
	args := m.Called(ctx, list, chainID)
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

func TestShoppingListService_Update(t *testing.T) {
	var (
		errGetShoppingList = errors.New("get shopping list error")
		errUpdate          = errors.New("update error")
	)

	userID := "123rf123123"
	shoppingList := domain.ShoppingList{UserID: userID, ID: "1_foo"}
	req := domain.UpdateShoppingListRequest{Name: "foobarfoo", Description: "New", SortType: domain.SortTypeCategory}
	tests := []struct {
		name                 string
		userID               string
		expectedReturn       domain.ShoppingList
		expectedErr          error
		mockShoppingListRepo func(m *mockShoppingListRepository)
	}{
		{
			name:        "returns error when GetByID returns error",
			userID:      userID,
			expectedErr: errGetShoppingList,
			mockShoppingListRepo: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(nil, errGetShoppingList)
			},
		},
		{
			name:        "returns ErrUnauthorized when userID doesnt match with shopping list",
			userID:      "wrong one",
			expectedErr: internalErr.ErrUnauthorized,
			mockShoppingListRepo: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil)
			},
		},
		{
			name:        "returns error when Update fails",
			userID:      userID,
			expectedErr: errUpdate,
			mockShoppingListRepo: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: userID}, nil)
				m.On("Update", mock.Anything, mock.MatchedBy(func(list *domain.ShoppingList) bool {
					return list.ID == shoppingList.ID
				})).Return(errUpdate)
			},
		},
		{
			name:   "returns list with changed Name, Description and SortType when update was successfully",
			userID: userID,
			expectedReturn: domain.ShoppingList{
				ID:          shoppingList.ID,
				UserID:      userID,
				Name:        req.Name,
				Description: req.Description,
				SortType:    req.SortType,
			},
			mockShoppingListRepo: func(m *mockShoppingListRepository) {
				updatedList := domain.ShoppingList{ID: shoppingList.ID, UserID: userID}
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&updatedList, nil)
				m.On("Update", mock.Anything, mock.MatchedBy(func(list *domain.ShoppingList) bool {
					return list.Name == req.Name && list.Description == req.Description && list.SortType == req.SortType
				})).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockShoppingListRepository)
			tt.mockShoppingListRepo(m)

			srv := NewShoppingListService(m, new(mockShoppingListRecipeRepository), new(mockStoreChainService), new(mockAIModel), zap.NewNop())
			v, err := srv.Update(context.Background(), tt.userID, shoppingList.ID, &req)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Nil(t, v)
			} else {
				require.Nil(t, err)
				require.Equal(t, tt.expectedReturn, *v)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListService_Delete(t *testing.T) {
	var (
		errGetByID  = errors.New("user not found")
		errDeletion = errors.New("deletion failed")
	)
	shoppingList := domain.ShoppingList{ID: "1_foo", UserID: "123"}
	wrongUserID := "foo"

	tests := []struct {
		name        string
		expectedErr error
		mockMethod  func(m *mockShoppingListRepository)
	}{
		{
			name:        "returns error when GetByID returns error",
			expectedErr: errGetByID,
			mockMethod: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(nil, errGetByID)
			},
		},
		{
			name:        "returns ErrUnauthorized when user is not authorized",
			expectedErr: internalErr.ErrUnauthorized,
			mockMethod: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: wrongUserID}, nil)
			},
		},
		{
			name:        "returns error when deletion fails",
			expectedErr: errDeletion,
			mockMethod: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil)
				m.On("Delete", mock.Anything, shoppingList.ID).Return(errDeletion)
			},
		},
		{
			name: "returns nil when deletion was successfully",
			mockMethod: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil)
				m.On("Delete", mock.Anything, shoppingList.ID).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockShoppingListRepository)
			tt.mockMethod(m)

			srv := NewShoppingListService(m, new(mockShoppingListRecipeRepository), new(mockStoreChainService), new(mockAIModel), zap.NewNop())
			err := srv.Delete(context.Background(), shoppingList.UserID, shoppingList.ID)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.Nil(t, err)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListService_GetSorted(t *testing.T) {
	var errGetByID = errors.New("user not found")

	shoppingList := domain.ShoppingList{ID: "1_foo", UserID: "123", Items: []domain.ShoppingListItem{{ID: "2", Name: "b"}, {ID: "1", Name: "a"}, {ID: "3", Name: "c"}}}
	tests := []struct {
		name           string
		userID         string
		expectedErr    error
		expectedReturn domain.ShoppingList
		mockMethod     func(m *mockShoppingListRepository)
	}{
		{
			name:        "returns error when GetByID returns error",
			userID:      shoppingList.UserID,
			expectedErr: errGetByID,
			mockMethod: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(nil, errGetByID)
			},
		},
		{
			name:        "returns Unauthorized error when User is Unauthorized",
			userID:      "321",
			expectedErr: internalErr.ErrUnauthorized,
			mockMethod: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{UserID: shoppingList.UserID, ID: shoppingList.ID}, nil)
			},
		},
		{
			name:           "returns Shopping List with sorted Items according to SortBy and SortDirection when request was successfully",
			userID:         shoppingList.UserID,
			expectedReturn: domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID, Items: []domain.ShoppingListItem{{ID: "1", Name: "a"}, {ID: "2", Name: "b"}, {ID: "3", Name: "c"}}},
			mockMethod: func(m *mockShoppingListRepository) {
				items := make([]domain.ShoppingListItem, len(shoppingList.Items))
				copy(items, shoppingList.Items)
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{UserID: shoppingList.UserID, ID: shoppingList.ID, Items: items}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockShoppingListRepository)
			tt.mockMethod(m)

			srv := NewShoppingListService(m, new(mockShoppingListRecipeRepository), new(mockStoreChainService), new(mockAIModel), zap.NewNop())
			v, err := srv.GetSorted(context.Background(), tt.userID, shoppingList.ID, "name", "asc")

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Nil(t, v)
			} else {
				require.Nil(t, err)
				require.Equal(t, v.ID, tt.expectedReturn.ID)
				require.Equal(t, v.Items, tt.expectedReturn.Items)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListService_GetSortedByStoreName(t *testing.T) {
	var (
		errGetByID              = errors.New("getByID error")
		errGetChainByName       = errors.New("getChainByName error")
		errOrganizeShoppingList = errors.New("organizeShoppingList error")
	)
	shoppingList := domain.ShoppingList{ID: "1_foo", UserID: "123", Items: []domain.ShoppingListItem{{ID: "2", Category: domain.CategoryDairy}, {ID: "1", Category: domain.CategoryBeverages}, {ID: "3", Category: domain.CategoryBakery}}}
	sortedItems := []domain.ShoppingListItem{{ID: "1", Category: domain.CategoryBeverages}, {ID: "2", Category: domain.CategoryDairy}, {ID: "3", Category: domain.CategoryBakery}}
	reversedSortedItems := []domain.ShoppingListItem{{ID: "3", Category: domain.CategoryBakery}, {ID: "2", Category: domain.CategoryDairy}, {ID: "1", Category: domain.CategoryBeverages}}
	storeChain := domain.StoreChain{ID: "1_bar", Name: "foobar", Layout: []domain.StoreSection{{Categories: []domain.Category{domain.CategoryDairy}, Order: 2}, {Categories: []domain.Category{domain.CategoryBeverages}, Order: 1}, {Categories: []domain.Category{domain.CategoryBakery}, Order: 3}}}
	tests := []struct {
		name                           string
		userID                         string
		sortDirection                  string
		expectedErr                    error
		expectedReturn                 domain.ShoppingList
		mockShoppingListRepositoryFunc func(m *mockShoppingListRepository)
		mockStoreChainServiceFunc      func(m *mockStoreChainService)
	}{
		{
			name:          "returns error when GetByID returns an error",
			userID:        shoppingList.UserID,
			sortDirection: "desc",
			expectedErr:   errGetByID,
			mockShoppingListRepositoryFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(nil, errGetByID).Once()
			},
		},
		{
			name:          "returns ErrUnauthorized when User is not authorized",
			userID:        "wrong-user",
			sortDirection: "desc",
			expectedErr:   internalErr.ErrUnauthorized,
			mockShoppingListRepositoryFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
			},
		},
		{
			name:          "returns error when GetChainByName returns error",
			userID:        shoppingList.UserID,
			sortDirection: "desc",
			expectedErr:   errGetChainByName,
			mockShoppingListRepositoryFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
			},
			mockStoreChainServiceFunc: func(m *mockStoreChainService) {
				m.On("GetChainByName", mock.Anything, storeChain.Name, "").Return(nil, errGetChainByName).Once()
			},
		},
		{
			name:          "returns error when OrganizeShoppingList returns error",
			userID:        shoppingList.UserID,
			sortDirection: "desc",
			expectedErr:   errOrganizeShoppingList,
			mockShoppingListRepositoryFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
			},
			mockStoreChainServiceFunc: func(m *mockStoreChainService) {
				m.On("GetChainByName", mock.Anything, storeChain.Name, "").Return(&domain.StoreChain{ID: storeChain.ID, Name: storeChain.Name}, nil).Once()
				m.On("OrganizeShoppingList", mock.Anything, mock.AnythingOfType("*domain.ShoppingList"), storeChain.ID).Return(errOrganizeShoppingList).Once()
			},
		},
		{
			name:           "returns shopping list sorted in descending order when request is successfully with sortDirection eq desc",
			userID:         shoppingList.UserID,
			sortDirection:  "desc",
			expectedReturn: domain.ShoppingList{ID: shoppingList.ID, Name: shoppingList.Name, Items: reversedSortedItems},
			mockShoppingListRepositoryFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID, Items: shoppingList.Items}, nil).Once()
			},
			mockStoreChainServiceFunc: func(m *mockStoreChainService) {
				m.On("GetChainByName", mock.Anything, storeChain.Name, "").Return(&domain.StoreChain{ID: storeChain.ID, Name: storeChain.Name}, nil).Once()
				m.On("OrganizeShoppingList", mock.Anything, mock.AnythingOfType("*domain.ShoppingList"), storeChain.ID).Return(nil).Once()
			},
		},
		{
			name:           "returns shopping list sorted in ascending order when request is successfully without sortDirection is desc",
			userID:         shoppingList.UserID,
			sortDirection:  "asc",
			expectedReturn: domain.ShoppingList{ID: shoppingList.ID, Name: shoppingList.Name, Items: sortedItems},
			mockShoppingListRepositoryFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID, Items: shoppingList.Items}, nil).Once()
			},
			mockStoreChainServiceFunc: func(m *mockStoreChainService) {
				m.On("GetChainByName", mock.Anything, storeChain.Name, "").Return(&domain.StoreChain{ID: storeChain.ID, Name: storeChain.Name}, nil).Once()
				m.On("OrganizeShoppingList", mock.Anything, mock.AnythingOfType("*domain.ShoppingList"), storeChain.ID).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockShoppingListRepo := new(mockShoppingListRepository)
			mockStoreChainSrv := new(mockStoreChainService)

			if tt.mockShoppingListRepositoryFunc != nil {
				tt.mockShoppingListRepositoryFunc(mockShoppingListRepo)
			}

			if tt.mockStoreChainServiceFunc != nil {
				tt.mockStoreChainServiceFunc(mockStoreChainSrv)
			}

			srv := NewShoppingListService(mockShoppingListRepo, new(mockShoppingListRecipeRepository), mockStoreChainSrv, new(mockAIModel), zap.NewNop())
			v, err := srv.GetSortedByStoreName(context.Background(), tt.userID, shoppingList.ID, storeChain.Name, tt.sortDirection)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Nil(t, v)
			} else {
				require.Nil(t, err)
				require.Equal(t, v.ID, tt.expectedReturn.ID)
				require.Equal(t, v.Name, tt.expectedReturn.Name)
				require.Equal(t, itemIDs(v.Items), itemIDs(tt.expectedReturn.Items))
			}
			mockShoppingListRepo.AssertExpectations(t)
			mockStoreChainSrv.AssertExpectations(t)
		})
	}
}
