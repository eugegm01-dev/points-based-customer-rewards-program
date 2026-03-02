package service

import (
	"context"
	"testing"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"golang.org/x/crypto/bcrypt"

	"github.com/stretchr/testify/assert"
)

type mockUserRepo struct {
	createFunc     func(ctx context.Context, user *domain.User) error
	getByLoginFunc func(ctx context.Context, login string) (*domain.User, error)
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	return m.createFunc(ctx, user)
}

func (m *mockUserRepo) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	return m.getByLoginFunc(ctx, login)
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name       string
		login      string
		password   string
		mockCreate func(ctx context.Context, user *domain.User) error
		wantErr    error
	}{
		{
			name:     "success",
			login:    "newuser",
			password: "secret",
			mockCreate: func(ctx context.Context, user *domain.User) error {
				user.ID = "123"
				return nil
			},
			wantErr: nil,
		},
		{
			name:     "duplicate login",
			login:    "existing",
			password: "secret",
			mockCreate: func(ctx context.Context, user *domain.User) error {
				return domain.ErrDuplicateLogin
			},
			wantErr: domain.ErrDuplicateLogin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepo{createFunc: tt.mockCreate}
			svc := NewAuthService(repo)
			user, err := svc.Register(context.Background(), tt.login, tt.password)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, user.ID)
				assert.Equal(t, tt.login, user.Login)
			}
		})
	}
}
func TestAuthService_Login_InvalidPassword(t *testing.T) {
	repo := &mockUserRepo{
		getByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
			return &domain.User{PasswordHash: "correct_hash"}, nil
		},
	}
	svc := NewAuthService(repo)
	// We need to mock bcrypt.CompareHashAndPassword, but that's internal. We can't easily mock it.
	// Instead, we rely on the actual bcrypt. For coverage, we can test with a wrong password.
	// This will produce ErrInvalidCredentials.
	_, err := svc.Login(context.Background(), "any", "wrong")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}
func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name           string
		login          string
		password       string
		mockGetByLogin func(ctx context.Context, login string) (*domain.User, error)
		wantErr        error
	}{
		{
			name:     "success",
			login:    "alice",
			password: "secret",
			mockGetByLogin: func(ctx context.Context, login string) (*domain.User, error) {
				hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcryptCost)
				return &domain.User{Login: login, PasswordHash: string(hash)}, nil
			},
			wantErr: nil,
		},
		{
			name:     "user not found",
			login:    "bob",
			password: "secret",
			mockGetByLogin: func(ctx context.Context, login string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name:     "wrong password",
			login:    "alice",
			password: "wrong",
			mockGetByLogin: func(ctx context.Context, login string) (*domain.User, error) {
				hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcryptCost)
				return &domain.User{Login: login, PasswordHash: string(hash)}, nil
			},
			wantErr: ErrInvalidCredentials,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepo{getByLoginFunc: tt.mockGetByLogin}
			svc := NewAuthService(repo)
			user, err := svc.Login(context.Background(), tt.login, tt.password)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.login, user.Login)
			}
		})
	}
}
