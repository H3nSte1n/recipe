package service

import (
	"context"
	"testing"

	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/repository"
	"github.com/H3nSte1n/recipe/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeAIConfigRepo is a minimal in-memory implementation of aiConfigRepository.
// It returns copies on read so the service cannot mutate stored state in place,
// mirroring how GORM hands back fresh structs.
type fakeAIConfigRepo struct {
	store  map[string]domain.UserAIConfig
	nextID int
}

func newFakeAIConfigRepo() *fakeAIConfigRepo {
	return &fakeAIConfigRepo{store: map[string]domain.UserAIConfig{}}
}

func (r *fakeAIConfigRepo) Create(_ context.Context, config *domain.UserAIConfig) error {
	r.nextID++
	config.ID = string(rune('a' + r.nextID))
	r.store[config.ID] = *config
	return nil
}

func (r *fakeAIConfigRepo) Update(_ context.Context, config *domain.UserAIConfig) error {
	r.store[config.ID] = *config
	return nil
}

func (r *fakeAIConfigRepo) GetByID(_ context.Context, id string) (*domain.UserAIConfig, error) {
	c := r.store[id]
	return &c, nil
}

func (r *fakeAIConfigRepo) ListByUserID(_ context.Context, userID string) ([]domain.UserAIConfig, error) {
	var out []domain.UserAIConfig
	for _, c := range r.store {
		if c.UserID == userID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *fakeAIConfigRepo) Delete(_ context.Context, id string) error {
	delete(r.store, id)
	return nil
}

func (r *fakeAIConfigRepo) GetAIModels(_ context.Context) ([]domain.AIModel, error) {
	return nil, nil
}

func (r *fakeAIConfigRepo) GetDefaultConfig(_ context.Context, userID string) (*domain.UserAIConfig, error) {
	for _, c := range r.store {
		if c.UserID == userID && c.IsDefault {
			cc := c
			return &cc, nil
		}
	}
	return nil, nil
}

func (r *fakeAIConfigRepo) SetDefault(_ context.Context, _, _ string) error { return nil }

func (r *fakeAIConfigRepo) ClearDefaultByUserID(_ context.Context, _ string, _ ...string) error {
	return nil
}

func (r *fakeAIConfigRepo) WithTypedTransaction(_ context.Context, fn func(repository.AIConfigRepository) error) error {
	return fn(r)
}

func (r *fakeAIConfigRepo) GetByUserAndModel(_ context.Context, _, _ string) (*domain.UserAIConfig, error) {
	return nil, nil
}

// rawStoredKey returns the API key exactly as persisted (still encrypted).
func (r *fakeAIConfigRepo) rawStoredKey(id string) string {
	return r.store[id].APIKey
}

func newTestAIConfigService(t *testing.T) (*fakeAIConfigRepo, AIConfigService, *crypto.Cipher) {
	t.Helper()
	repo := newFakeAIConfigRepo()
	cipher, err := crypto.NewCipher("test-encryption-key")
	require.NoError(t, err)
	svc := NewAIConfigService(repo, cipher, zap.NewNop())
	return repo, svc, cipher
}

func TestAIConfigService_CreateEncryptsAndReadDecrypts(t *testing.T) {
	repo, svc, cipher := newTestAIConfigService(t)
	ctx := context.Background()
	const plaintext = "sk-ant-EXAMPLE"

	created, err := svc.Create(ctx, "user-1", &domain.CreateUserAIConfigRequest{
		AIModelID: "model-1",
		APIKey:    plaintext,
	})
	require.NoError(t, err)

	// Stored value must be ciphertext, not the plaintext key.
	stored := repo.rawStoredKey(created.ID)
	assert.NotEqual(t, plaintext, stored, "stored key must be encrypted")
	decrypted, err := cipher.Decrypt(stored)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted, "stored ciphertext must decrypt to plaintext (single encryption)")

	// Service reads must return plaintext.
	got, err := svc.GetByID(ctx, "user-1", created.ID)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got.APIKey)
}

func TestAIConfigService_UpdateWithoutKeyChangeDoesNotDoubleEncrypt(t *testing.T) {
	repo, svc, cipher := newTestAIConfigService(t)
	ctx := context.Background()
	const plaintext = "sk-ant-EXAMPLE"

	created, err := svc.Create(ctx, "user-1", &domain.CreateUserAIConfigRequest{
		AIModelID: "model-1",
		APIKey:    plaintext,
	})
	require.NoError(t, err)

	// Update an unrelated field; the key is not supplied.
	isDefault := true
	_, err = svc.Update(ctx, "user-1", created.ID, &domain.UpdateUserAIConfigRequest{
		IsDefault: &isDefault,
	})
	require.NoError(t, err)

	// A single decrypt of the stored value must yield the original plaintext.
	// Double-encryption would yield ciphertext instead.
	stored := repo.rawStoredKey(created.ID)
	decrypted, err := cipher.Decrypt(stored)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted, "update must not double-encrypt the key")

	got, err := svc.GetByID(ctx, "user-1", created.ID)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got.APIKey)
}

func TestAIConfigService_LegacyPlaintextReadFallback(t *testing.T) {
	repo, svc, _ := newTestAIConfigService(t)
	ctx := context.Background()

	// Simulate a row written before at-rest encryption: plaintext in the column.
	repo.store["legacy"] = domain.UserAIConfig{
		ID:     "legacy",
		UserID: "user-1",
		APIKey: "legacy-plaintext-key",
	}

	got, err := svc.GetByID(ctx, "user-1", "legacy")
	require.NoError(t, err)
	assert.Equal(t, "legacy-plaintext-key", got.APIKey, "legacy plaintext must be returned as-is")
}
