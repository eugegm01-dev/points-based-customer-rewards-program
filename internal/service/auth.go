// Package service contains application business logic (use cases).
package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
)

const bcryptCost = 12

// AuthService handles registration and login.
type AuthService struct {
	userRepo domain.UserRepository
}

// NewAuthService returns a new AuthService.
func NewAuthService(userRepo domain.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

// Register creates a new user with hashed password. Returns the created user or domain.ErrDuplicateLogin.
func (s *AuthService) Register(ctx context.Context, login, password string) (*domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, err
	}
	u := &domain.User{
		Login:        login,
		PasswordHash: string(hash),
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Login authenticates by login and password. Returns the user or ErrInvalidCredentials.
func (s *AuthService) Login(ctx context.Context, login, password string) (*domain.User, error) {
	u, err := s.userRepo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

// ErrInvalidCredentials is returned when login or password is wrong.
var ErrInvalidCredentials = errors.New("invalid credentials")
