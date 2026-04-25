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
		errReload          = errors.New("reload error")
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
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(nil, errGetShoppingList).Once()
			},
		},
		{
			name:        "returns ErrUnauthorized when userID doesnt match with shopping list",
			userID:      "wrong one",
			expectedErr: internalErr.ErrUnauthorized,
			mockShoppingListRepo: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
			},
		},
		{
			name:        "returns error when Update fails",
			userID:      userID,
			expectedErr: errUpdate,
			mockShoppingListRepo: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: userID}, nil).Once()
				m.On("Update", mock.Anything, mock.MatchedBy(func(list *domain.ShoppingList) bool {
					return list.ID == shoppingList.ID
				})).Return(errUpdate).Once()
			},
		},
		{
			name:        "returns error when final GetByID reload fails",
			userID:      userID,
			expectedErr: errReload,
			mockShoppingListRepo: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: userID}, nil).Once()
				m.On("Update", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(nil, errReload).Once()
			},
		},
		{
			name:   "returns reloaded list with changed Name, Description and SortType when update was successfully",
			userID: userID,
			expectedReturn: domain.ShoppingList{
				ID:          shoppingList.ID,
				UserID:      userID,
				Name:        req.Name,
				Description: req.Description,
				SortType:    req.SortType,
			},
			mockShoppingListRepo: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: userID}, nil).Once()
				m.On("Update", mock.Anything, mock.MatchedBy(func(list *domain.ShoppingList) bool {
					return list.Name == req.Name && list.Description == req.Description && list.SortType == req.SortType
				})).Return(nil).Once()
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{
					ID: shoppingList.ID, UserID: userID,
					Name: req.Name, Description: req.Description, SortType: req.SortType,
				}, nil).Once()
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
		{
			name:           "falls back to name sort and logs warning when sortBy is unknown",
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

			sortBy := "name"
			if tt.name == "falls back to name sort and logs warning when sortBy is unknown" {
				sortBy = "invalid_field"
			}

			srv := NewShoppingListService(m, new(mockShoppingListRecipeRepository), new(mockStoreChainService), new(mockAIModel), zap.NewNop())
			v, err := srv.GetSorted(context.Background(), tt.userID, shoppingList.ID, sortBy, "asc")

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
				m.On("OrganizeShoppingList", mock.Anything, mock.AnythingOfType("*domain.ShoppingList"), storeChain.ID).
					Run(func(args mock.Arguments) {
						list := args.Get(1).(*domain.ShoppingList)
						list.Items = sortedItems
					}).Return(nil).Once()
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
				m.On("OrganizeShoppingList", mock.Anything, mock.AnythingOfType("*domain.ShoppingList"), storeChain.ID).
					Run(func(args mock.Arguments) {
						list := args.Get(1).(*domain.ShoppingList)
						list.Items = sortedItems
					}).Return(nil).Once()
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
			v, err := srv.GetSortedByStoreName(context.Background(), tt.userID, shoppingList.ID, storeChain.Name, "", tt.sortDirection)

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

func TestShoppingListService_AddItem(t *testing.T) {
	var (
		errGetByID  = errors.New("getByID error")
		errAddItems = errors.New("addItems error")
	)
	shoppingList := domain.ShoppingList{ID: "1_foo", UserID: "123"}
	req := domain.ShoppingListItemRequest{Name: "foo", Amount: 1, Unit: "kg", Notes: "foobar"}

	tests := []struct {
		name                     string
		userID                   string
		req                      domain.ShoppingListItemRequest
		expectedErr              error
		mockShoppingListRepoFunc func(m *mockShoppingListRepository)
		mockAiModelFunc          func(m *mockAIModel)
	}{
		{
			name:        "returns error when GetByID fails",
			userID:      shoppingList.UserID,
			expectedErr: errGetByID,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(nil, errGetByID).Once()
			},
		},
		{
			name:        "returns error when User is not authorized",
			userID:      "wrong-user",
			expectedErr: internalErr.ErrUnauthorized,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
			},
		},
		{
			name:   "falls back to CategoryOther when categorization fails",
			userID: shoppingList.UserID,
			req:    req,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
				m.On("AddItems", mock.Anything, mock.MatchedBy(func(items []domain.ShoppingListItem) bool {
					return len(items) == 1 && items[0].Category == domain.CategoryOther
				})).Return(nil).Once()
			},
			mockAiModelFunc: func(m *mockAIModel) {
				m.On("CategorizeItems", mock.Anything, mock.Anything).Return(nil, errors.New("categorization error")).Once()
			},
		},
		{
			name:   "falls back to CategoryOther when item name is not in categorization result",
			userID: shoppingList.UserID,
			req:    req,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
				m.On("AddItems", mock.Anything, mock.MatchedBy(func(items []domain.ShoppingListItem) bool {
					return len(items) == 1 && items[0].Category == domain.CategoryOther
				})).Return(nil).Once()
			},
			mockAiModelFunc: func(m *mockAIModel) {
				m.On("CategorizeItems", mock.Anything, mock.Anything).Return(map[string]string{"different-item": string(domain.CategoryDairy)}, nil).Once()
			},
		},
		{
			name:        "returns error when AddItems fails",
			userID:      shoppingList.UserID,
			expectedErr: errAddItems,
			req:         req,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
				m.On("AddItems", mock.Anything, mock.Anything).Return(errAddItems).Once()
			},
			mockAiModelFunc: func(m *mockAIModel) {
				m.On("CategorizeItems", mock.Anything, mock.Anything).Return(map[string]string{"foo": string(domain.CategoryDairy)}, nil).Once()
			},
		},
		{
			name:   "returns nil and applies AI category when creation was successfully",
			userID: shoppingList.UserID,
			req:    req,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
				m.On("AddItems", mock.Anything, mock.MatchedBy(func(items []domain.ShoppingListItem) bool {
					return len(items) == 1 && items[0].Category == domain.CategoryDairy
				})).Return(nil).Once()
			},
			mockAiModelFunc: func(m *mockAIModel) {
				m.On("CategorizeItems", mock.Anything, mock.Anything).Return(map[string]string{"foo": string(domain.CategoryDairy)}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mShoppingListRepo := new(mockShoppingListRepository)
			mAIModel := new(mockAIModel)

			if tt.mockShoppingListRepoFunc != nil {
				tt.mockShoppingListRepoFunc(mShoppingListRepo)
			}

			if tt.mockAiModelFunc != nil {
				tt.mockAiModelFunc(mAIModel)
			}

			srv := NewShoppingListService(mShoppingListRepo, new(mockShoppingListRecipeRepository), new(mockStoreChainService), mAIModel, zap.NewNop())
			err := srv.AddItem(context.Background(), tt.userID, shoppingList.ID, &tt.req)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.Nil(t, err)
			}
			mShoppingListRepo.AssertExpectations(t)
			mAIModel.AssertExpectations(t)
		})
	}
}

func TestShoppingListService_UpdateItem(t *testing.T) {
	var (
		errGetItemByID = errors.New("GetItemByID error")
		errGetByID     = errors.New("GetByID error")
		errUpdateItem  = errors.New("UpdateItem error")
	)
	req := domain.UpdateShoppingListItemRequest{Name: "foo_edited", Amount: 312, Unit: "bar", Category: domain.CategoryProduce, Notes: "foobar edited"}
	list := domain.ShoppingList{ID: "1", UserID: "123"}
	item := domain.ShoppingListItem{ID: "1_foo", ListID: list.ID, Name: "foo", Amount: 213, Unit: "foo", Category: domain.CategoryBakery, Notes: "foobar"}
	tests := []struct {
		name        string
		userID      string
		expectedErr error
		mockFunc    func(m *mockShoppingListRepository)
	}{
		{
			name:        "returns error when GetItemByID fails",
			userID:      list.UserID,
			expectedErr: errGetItemByID,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(nil, errGetItemByID).Once()
			},
		},
		{
			name:        "returns error when GetByID fails",
			userID:      list.UserID,
			expectedErr: errGetByID,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(nil, errGetByID).Once()
			},
		},
		{
			name:        "returns Unauthorized when user is not authorized",
			userID:      "wrong-user",
			expectedErr: internalErr.ErrUnauthorized,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
			},
		},
		{
			name:        "returns error when UpdateItem fails",
			userID:      list.UserID,
			expectedErr: errUpdateItem,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
				m.On("UpdateItem", mock.Anything, mock.AnythingOfType("*domain.ShoppingListItem")).Return(errUpdateItem).Once()
			},
		},
		{
			name:   "calls UpdateItem with Name, Amount, Unit, Category, Notes updated and returns nil when request is successfully",
			userID: list.UserID,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
				m.On("UpdateItem", mock.Anything, mock.MatchedBy(func(r *domain.ShoppingListItem) bool {
					return req.Name == r.Name && req.Amount == r.Amount && req.Unit == r.Unit && req.Category == r.Category && req.Notes == r.Notes
				})).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockShoppingListRepository)
			tt.mockFunc(m)

			srv := NewShoppingListService(m, new(mockShoppingListRecipeRepository), new(mockStoreChainService), new(mockAIModel), zap.NewNop())
			err := srv.UpdateItem(context.Background(), tt.userID, item.ID, &req)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.Nil(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListService_Create(t *testing.T) {
	var (
		errCreate   = errors.New("create error")
		errGetByID  = errors.New("getByID error")
		errAddItems = errors.New("addItems error")
	)
	userID := "123"
	list := domain.ShoppingList{ID: "1_foo", UserID: userID}
	req := domain.CreateShoppingListRequest{Name: "my list", Description: "desc", SortType: domain.SortTypeCategory}
	reqWithItems := domain.CreateShoppingListRequest{
		Name:        "my list",
		Description: "desc",
		SortType:    domain.SortTypeCategory,
		Items:       []domain.ShoppingListItemRequest{{Name: "Milk", Amount: 1, Unit: "L", Category: domain.CategoryDairy}},
	}

	tests := []struct {
		name           string
		req            domain.CreateShoppingListRequest
		expectedReturn *domain.ShoppingList
		expectedErr    error
		expectedErrMsg string
		mockFunc       func(m *mockShoppingListRepository)
	}{
		{
			name:        "returns error when Create fails",
			req:         req,
			expectedErr: errCreate,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*domain.ShoppingList")).Return(errCreate).Once()
			},
		},
		{
			name:           "returns error when list ID is empty after creation",
			req:            req,
			expectedErrMsg: "failed to retrieve generated list ID after creation",
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*domain.ShoppingList")).Return(nil).Once()
			},
		},
		{
			name:        "returns error when AddItems fails",
			req:         reqWithItems,
			expectedErr: errAddItems,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*domain.ShoppingList")).
					Run(func(args mock.Arguments) {
						args.Get(1).(*domain.ShoppingList).ID = list.ID
					}).Return(nil).Once()
				m.On("AddItems", mock.Anything, mock.Anything).Return(errAddItems).Once()
			},
		},
		{
			name:        "returns error when final GetByID fails",
			req:         req,
			expectedErr: errGetByID,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*domain.ShoppingList")).
					Run(func(args mock.Arguments) {
						args.Get(1).(*domain.ShoppingList).ID = list.ID
					}).Return(nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(nil, errGetByID).Once()
			},
		},
		{
			name:           "returns created shopping list when request is successfully",
			req:            req,
			expectedReturn: &domain.ShoppingList{ID: list.ID, UserID: userID, Name: req.Name, Description: req.Description, SortType: req.SortType},
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("Create", mock.Anything, mock.MatchedBy(func(l *domain.ShoppingList) bool {
					return l.Name == req.Name && l.Description == req.Description && l.SortType == req.SortType && l.UserID == userID
				})).Run(func(args mock.Arguments) {
					args.Get(1).(*domain.ShoppingList).ID = list.ID
				}).Return(nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: userID, Name: req.Name, Description: req.Description, SortType: req.SortType}, nil).Once()
			},
		},
		{
			name:           "returns created shopping list with items when request includes items",
			req:            reqWithItems,
			expectedReturn: &domain.ShoppingList{ID: list.ID, UserID: userID, Name: reqWithItems.Name},
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*domain.ShoppingList")).
					Run(func(args mock.Arguments) {
						args.Get(1).(*domain.ShoppingList).ID = list.ID
					}).Return(nil).Once()
				m.On("AddItems", mock.Anything, mock.MatchedBy(func(items []domain.ShoppingListItem) bool {
					return len(items) == 1 && items[0].Name == reqWithItems.Items[0].Name
				})).Return(nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: userID, Name: reqWithItems.Name}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockShoppingListRepository)
			tt.mockFunc(m)

			srv := NewShoppingListService(m, new(mockShoppingListRecipeRepository), new(mockStoreChainService), new(mockAIModel), zap.NewNop())
			v, err := srv.Create(context.Background(), userID, &tt.req)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Nil(t, v)
			} else if tt.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
				require.Nil(t, v)
			} else {
				require.Nil(t, err)
				require.Equal(t, tt.expectedReturn, v)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListService_GetByID(t *testing.T) {
	var errGetByID = errors.New("getByID error")

	shoppingList := domain.ShoppingList{ID: "1_foo", UserID: "123"}

	tests := []struct {
		name           string
		userID         string
		expectedReturn *domain.ShoppingList
		expectedErr    error
		mockFunc       func(m *mockShoppingListRepository)
	}{
		{
			name:        "returns error when GetByID fails",
			userID:      shoppingList.UserID,
			expectedErr: errGetByID,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(nil, errGetByID).Once()
			},
		},
		{
			name:        "returns ErrUnauthorized when user is not authorized",
			userID:      "wrong-user",
			expectedErr: internalErr.ErrUnauthorized,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
			},
		},
		{
			name:           "returns shopping list when request is successfully",
			userID:         shoppingList.UserID,
			expectedReturn: &domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID},
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockShoppingListRepository)
			tt.mockFunc(m)

			srv := NewShoppingListService(m, new(mockShoppingListRecipeRepository), new(mockStoreChainService), new(mockAIModel), zap.NewNop())
			v, err := srv.GetByID(context.Background(), tt.userID, shoppingList.ID)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Nil(t, v)
			} else {
				require.Nil(t, err)
				require.Equal(t, tt.expectedReturn, v)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListService_ListByUserID(t *testing.T) {
	var errListByUserID = errors.New("listByUserID error")

	userID := "123"
	lists := []domain.ShoppingList{{ID: "1_foo", UserID: userID}, {ID: "2_bar", UserID: userID}}

	tests := []struct {
		name           string
		expectedReturn []domain.ShoppingList
		expectedErr    error
		mockFunc       func(m *mockShoppingListRepository)
	}{
		{
			name:        "returns error when ListByUserID fails",
			expectedErr: errListByUserID,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("ListByUserID", mock.Anything, userID).Return(nil, errListByUserID).Once()
			},
		},
		{
			name:           "returns list of shopping lists when request is successfully",
			expectedReturn: lists,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("ListByUserID", mock.Anything, userID).Return(lists, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockShoppingListRepository)
			tt.mockFunc(m)

			srv := NewShoppingListService(m, new(mockShoppingListRecipeRepository), new(mockStoreChainService), new(mockAIModel), zap.NewNop())
			v, err := srv.ListByUserID(context.Background(), userID)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Nil(t, v)
			} else {
				require.Nil(t, err)
				require.Equal(t, tt.expectedReturn, v)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListService_DeleteItem(t *testing.T) {
	var (
		errGetItemByID = errors.New("GetItemByID error")
		errGetByID     = errors.New("GetByID error")
		errDeleteItem  = errors.New("DeleteItem error")
	)
	list := domain.ShoppingList{ID: "1", UserID: "123"}
	item := domain.ShoppingListItem{ID: "1_foo", ListID: list.ID}

	tests := []struct {
		name        string
		userID      string
		expectedErr error
		mockFunc    func(m *mockShoppingListRepository)
	}{
		{
			name:        "returns error when GetItemByID fails",
			userID:      list.UserID,
			expectedErr: errGetItemByID,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(nil, errGetItemByID).Once()
			},
		},
		{
			name:        "returns error when GetByID fails",
			userID:      list.UserID,
			expectedErr: errGetByID,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(nil, errGetByID).Once()
			},
		},
		{
			name:        "returns ErrUnauthorized when user is not authorized",
			userID:      "wrong-user",
			expectedErr: internalErr.ErrUnauthorized,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
			},
		},
		{
			name:        "returns error when DeleteItem fails",
			userID:      list.UserID,
			expectedErr: errDeleteItem,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
				m.On("DeleteItem", mock.Anything, item.ID).Return(errDeleteItem).Once()
			},
		},
		{
			name:   "returns nil when deletion was successfully",
			userID: list.UserID,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
				m.On("DeleteItem", mock.Anything, item.ID).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockShoppingListRepository)
			tt.mockFunc(m)

			srv := NewShoppingListService(m, new(mockShoppingListRecipeRepository), new(mockStoreChainService), new(mockAIModel), zap.NewNop())
			err := srv.DeleteItem(context.Background(), tt.userID, item.ID)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.Nil(t, err)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListService_ToggleItem(t *testing.T) {
	var (
		errGetItemByID = errors.New("GetItemByID error")
		errGetByID     = errors.New("GetByID error")
		errUpdateItem  = errors.New("UpdateItem error")
	)
	list := domain.ShoppingList{ID: "1", UserID: "123"}
	item := domain.ShoppingListItem{ID: "1_foo", ListID: list.ID, IsChecked: false}

	tests := []struct {
		name        string
		userID      string
		checked     bool
		expectedErr error
		mockFunc    func(m *mockShoppingListRepository)
	}{
		{
			name:        "returns error when GetItemByID fails",
			userID:      list.UserID,
			checked:     true,
			expectedErr: errGetItemByID,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(nil, errGetItemByID).Once()
			},
		},
		{
			name:        "returns error when GetByID fails",
			userID:      list.UserID,
			checked:     true,
			expectedErr: errGetByID,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(nil, errGetByID).Once()
			},
		},
		{
			name:        "returns ErrUnauthorized when user is not authorized",
			userID:      "wrong-user",
			checked:     true,
			expectedErr: internalErr.ErrUnauthorized,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
			},
		},
		{
			name:        "returns error when UpdateItem fails",
			userID:      list.UserID,
			checked:     true,
			expectedErr: errUpdateItem,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
				m.On("UpdateItem", mock.Anything, mock.AnythingOfType("*domain.ShoppingListItem")).Return(errUpdateItem).Once()
			},
		},
		{
			name:    "calls UpdateItem with IsChecked set to true and returns nil when request is successfully",
			userID:  list.UserID,
			checked: true,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID, IsChecked: false}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
				m.On("UpdateItem", mock.Anything, mock.MatchedBy(func(i *domain.ShoppingListItem) bool {
					return i.IsChecked == true
				})).Return(nil).Once()
			},
		},
		{
			name:    "calls UpdateItem with IsChecked set to false and returns nil when request is successfully",
			userID:  list.UserID,
			checked: false,
			mockFunc: func(m *mockShoppingListRepository) {
				m.On("GetItemByID", mock.Anything, item.ID).Return(&domain.ShoppingListItem{ID: item.ID, ListID: item.ListID, IsChecked: true}, nil).Once()
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
				m.On("UpdateItem", mock.Anything, mock.MatchedBy(func(i *domain.ShoppingListItem) bool {
					return i.IsChecked == false
				})).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockShoppingListRepository)
			tt.mockFunc(m)

			srv := NewShoppingListService(m, new(mockShoppingListRecipeRepository), new(mockStoreChainService), new(mockAIModel), zap.NewNop())
			err := srv.ToggleItem(context.Background(), tt.userID, item.ID, tt.checked)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.Nil(t, err)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListService_AddRecipeToList(t *testing.T) {
	var (
		errGetByID   = errors.New("getByID error")
		errGetRecipe = errors.New("getRecipe error")
		errAddItems  = errors.New("addItems error")
	)
	list := domain.ShoppingList{ID: "1_foo", UserID: "123"}
	recipe := domain.Recipe{
		ID:       "recipe_1",
		Servings: 2,
		Ingredients: []domain.RecipeIngredient{
			{Name: "Milk", Amount: 200, Unit: "ml"},
			{Name: "Flour", Amount: 100, Unit: "g"},
		},
	}
	req := domain.AddRecipeToListRequest{RecipeID: recipe.ID, Servings: 4}

	tests := []struct {
		name                     string
		userID                   string
		expectedErr              error
		expectedErrMsg           string
		mockShoppingListRepoFunc func(m *mockShoppingListRepository)
		mockRecipeRepoFunc       func(m *mockShoppingListRecipeRepository)
		mockAiModelFunc          func(m *mockAIModel)
	}{
		{
			name:        "returns error when GetByID fails",
			userID:      list.UserID,
			expectedErr: errGetByID,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, list.ID).Return(nil, errGetByID).Once()
			},
		},
		{
			name:        "returns ErrUnauthorized when user is not authorized",
			userID:      "wrong-user",
			expectedErr: internalErr.ErrUnauthorized,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
			},
		},
		{
			name:        "returns error when GetRecipe fails",
			userID:      list.UserID,
			expectedErr: errGetRecipe,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
			},
			mockRecipeRepoFunc: func(m *mockShoppingListRecipeRepository) {
				m.On("GetByID", mock.Anything, recipe.ID, domain.NutritionDetailBase).Return(nil, errGetRecipe).Once()
			},
		},
		{
			name:           "returns error when recipe has zero servings",
			userID:         list.UserID,
			expectedErrMsg: "recipe has no servings defined",
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
			},
			mockRecipeRepoFunc: func(m *mockShoppingListRecipeRepository) {
				zeroServingsRecipe := domain.Recipe{ID: recipe.ID, Servings: 0, Ingredients: recipe.Ingredients}
				m.On("GetByID", mock.Anything, recipe.ID, domain.NutritionDetailBase).Return(&zeroServingsRecipe, nil).Once()
			},
		},
		{
			name:        "returns error when AddItems fails",
			userID:      list.UserID,
			expectedErr: errAddItems,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
				m.On("AddItems", mock.Anything, mock.Anything).Return(errAddItems).Once()
			},
			mockRecipeRepoFunc: func(m *mockShoppingListRecipeRepository) {
				m.On("GetByID", mock.Anything, recipe.ID, domain.NutritionDetailBase).Return(&recipe, nil).Once()
			},
			mockAiModelFunc: func(m *mockAIModel) {
				m.On("CategorizeItems", mock.Anything, mock.Anything).Return(map[string]string{"Milk": string(domain.CategoryDairy), "Flour": string(domain.CategoryBakery)}, nil).Once()
			},
		},
		{
			name:   "returns nil without calling AI when recipe has no ingredients",
			userID: list.UserID,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
			},
			mockRecipeRepoFunc: func(m *mockShoppingListRecipeRepository) {
				m.On("GetByID", mock.Anything, recipe.ID, domain.NutritionDetailBase).Return(&domain.Recipe{ID: recipe.ID, Servings: 2, Ingredients: nil}, nil).Once()
			},
			// mockAiModelFunc intentionally omitted — AssertExpectations will confirm CategorizeItems was never called
		},
		{
			name:   "returns nil and scales items by servings ratio when request is successfully",
			userID: list.UserID,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
				m.On("AddItems", mock.Anything, mock.MatchedBy(func(items []domain.ShoppingListItem) bool {
					// req.Servings(4) / recipe.Servings(2) = scalingFactor 2.0
					return len(items) == 2 &&
						items[0].Amount == recipe.Ingredients[0].Amount*2 &&
						items[1].Amount == recipe.Ingredients[1].Amount*2
				})).Return(nil).Once()
			},
			mockRecipeRepoFunc: func(m *mockShoppingListRecipeRepository) {
				m.On("GetByID", mock.Anything, recipe.ID, domain.NutritionDetailBase).Return(&recipe, nil).Once()
			},
			mockAiModelFunc: func(m *mockAIModel) {
				m.On("CategorizeItems", mock.Anything, mock.Anything).Return(map[string]string{"Milk": string(domain.CategoryDairy), "Flour": string(domain.CategoryBakery)}, nil).Once()
			},
		},
		{
			name:   "falls back to CategoryOther for all items when AI categorization fails",
			userID: list.UserID,
			mockShoppingListRepoFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, list.ID).Return(&domain.ShoppingList{ID: list.ID, UserID: list.UserID}, nil).Once()
				m.On("AddItems", mock.Anything, mock.MatchedBy(func(items []domain.ShoppingListItem) bool {
					return len(items) == 2 && items[0].Category == domain.CategoryOther && items[1].Category == domain.CategoryOther
				})).Return(nil).Once()
			},
			mockRecipeRepoFunc: func(m *mockShoppingListRecipeRepository) {
				m.On("GetByID", mock.Anything, recipe.ID, domain.NutritionDetailBase).Return(&recipe, nil).Once()
			},
			mockAiModelFunc: func(m *mockAIModel) {
				m.On("CategorizeItems", mock.Anything, mock.Anything).Return(nil, errors.New("ai error")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mShoppingListRepo := new(mockShoppingListRepository)
			mRecipeRepo := new(mockShoppingListRecipeRepository)
			mAIModel := new(mockAIModel)

			if tt.mockShoppingListRepoFunc != nil {
				tt.mockShoppingListRepoFunc(mShoppingListRepo)
			}
			if tt.mockRecipeRepoFunc != nil {
				tt.mockRecipeRepoFunc(mRecipeRepo)
			}
			if tt.mockAiModelFunc != nil {
				tt.mockAiModelFunc(mAIModel)
			}

			srv := NewShoppingListService(mShoppingListRepo, mRecipeRepo, new(mockStoreChainService), mAIModel, zap.NewNop())
			err := srv.AddRecipeToList(context.Background(), tt.userID, list.ID, &req)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else if tt.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.Nil(t, err)
			}
			mShoppingListRepo.AssertExpectations(t)
			mRecipeRepo.AssertExpectations(t)
			mAIModel.AssertExpectations(t)
		})
	}
}

func TestShoppingListService_GetSortedForStore(t *testing.T) {
	var (
		errGetByID              = errors.New("getByID error")
		errOrganizeShoppingList = errors.New("organizeShoppingList error")
	)
	shoppingList := domain.ShoppingList{ID: "1_foo", UserID: "123", Items: []domain.ShoppingListItem{{ID: "2", Category: domain.CategoryDairy}, {ID: "1", Category: domain.CategoryBeverages}}}
	chainID := "chain_1"
	organizedItems := []domain.ShoppingListItem{{ID: "1", Category: domain.CategoryBeverages}, {ID: "2", Category: domain.CategoryDairy}}

	tests := []struct {
		name                           string
		userID                         string
		expectedErr                    error
		expectedReturn                 *domain.ShoppingList
		mockShoppingListRepositoryFunc func(m *mockShoppingListRepository)
		mockStoreChainServiceFunc      func(m *mockStoreChainService)
	}{
		{
			name:        "returns error when GetByID fails",
			userID:      shoppingList.UserID,
			expectedErr: errGetByID,
			mockShoppingListRepositoryFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(nil, errGetByID).Once()
			},
		},
		{
			name:        "returns ErrUnauthorized when user is not authorized",
			userID:      "wrong-user",
			expectedErr: internalErr.ErrUnauthorized,
			mockShoppingListRepositoryFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
			},
		},
		{
			name:        "returns error when OrganizeShoppingList fails",
			userID:      shoppingList.UserID,
			expectedErr: errOrganizeShoppingList,
			mockShoppingListRepositoryFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID}, nil).Once()
			},
			mockStoreChainServiceFunc: func(m *mockStoreChainService) {
				m.On("OrganizeShoppingList", mock.Anything, mock.AnythingOfType("*domain.ShoppingList"), chainID).Return(errOrganizeShoppingList).Once()
			},
		},
		{
			name:           "returns shopping list organized by store layout when request is successfully",
			userID:         shoppingList.UserID,
			expectedReturn: &domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID, Items: organizedItems},
			mockShoppingListRepositoryFunc: func(m *mockShoppingListRepository) {
				m.On("GetByID", mock.Anything, shoppingList.ID).Return(&domain.ShoppingList{ID: shoppingList.ID, UserID: shoppingList.UserID, Items: shoppingList.Items}, nil).Once()
			},
			mockStoreChainServiceFunc: func(m *mockStoreChainService) {
				m.On("OrganizeShoppingList", mock.Anything, mock.AnythingOfType("*domain.ShoppingList"), chainID).
					Run(func(args mock.Arguments) {
						list := args.Get(1).(*domain.ShoppingList)
						list.Items = organizedItems
					}).Return(nil).Once()
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
			v, err := srv.GetSortedForStore(context.Background(), tt.userID, shoppingList.ID, chainID)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Nil(t, v)
			} else {
				require.Nil(t, err)
				require.Equal(t, tt.expectedReturn.ID, v.ID)
				require.Equal(t, itemIDs(tt.expectedReturn.Items), itemIDs(v.Items))
			}
			mockShoppingListRepo.AssertExpectations(t)
			mockStoreChainSrv.AssertExpectations(t)
		})
	}
}
