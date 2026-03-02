package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/handlers"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// mockAuthService already defined in handlers_test.go

func TestAuthHandler_Register(t *testing.T) {
	log := zerolog.Nop()
	tests := []struct {
		name         string
		reqBody      interface{}
		mockRegister func(ctx context.Context, login, password string) (*domain.User, error)
		wantStatus   int
	}{
		{
			name:    "success",
			reqBody: map[string]string{"login": "alice", "password": "secret"},
			mockRegister: func(ctx context.Context, login, password string) (*domain.User, error) {
				return &domain.User{ID: "1", Login: login}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "duplicate login",
			reqBody: map[string]string{"login": "alice", "password": "secret"},
			mockRegister: func(ctx context.Context, login, password string) (*domain.User, error) {
				return nil, domain.ErrDuplicateLogin
			},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "validation error",
			reqBody:    map[string]string{"login": "bob", "password": "123"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockAuthService{registerFunc: tt.mockRegister}
			h := &handlers.AuthHandler{
				AuthService: mockSvc,
				AuthSecret:  "testsecret",
				Logger:      log,
			}

			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Post("/register", h.Register)
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}
func TestAuthHandler_Login(t *testing.T) {
	log := zerolog.Nop()
	tests := []struct {
		name       string
		reqBody    interface{}
		mockLogin  func(ctx context.Context, login, password string) (*domain.User, error)
		wantStatus int
	}{
		{
			name:    "success",
			reqBody: map[string]string{"login": "alice", "password": "secret"},
			mockLogin: func(ctx context.Context, login, password string) (*domain.User, error) {
				return &domain.User{ID: "1", Login: login}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "invalid credentials",
			reqBody: map[string]string{"login": "alice", "password": "wrongpass"}, // length 9
			mockLogin: func(ctx context.Context, login, password string) (*domain.User, error) {
				return nil, service.ErrInvalidCredentials
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockAuthService{loginFunc: tt.mockLogin}
			h := &handlers.AuthHandler{
				AuthService: mockSvc,
				AuthSecret:  "testsecret",
				Logger:      log,
			}

			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Post("/login", h.Login)
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}
