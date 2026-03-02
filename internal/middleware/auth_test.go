package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/auth"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_NoToken(t *testing.T) {
	handler := middleware.Auth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_ValidToken(t *testing.T) {
	secret := "testsecret"
	userID := "123"
	login := "alice"
	token, err := auth.CreateToken(secret, userID, login)
	require.NoError(t, err)

	handler := middleware.Auth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID := middleware.GetUserID(r.Context())
		gotLogin := middleware.GetLogin(r.Context())
		assert.Equal(t, userID, gotUserID)
		assert.Equal(t, login, gotLogin)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: middleware.CookieName, Value: token})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	handler := middleware.Auth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: middleware.CookieName, Value: "invalid.token.string"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_BearerToken(t *testing.T) {
	secret := "testsecret"
	userID := "123"
	login := "alice"
	token, err := auth.CreateToken(secret, userID, login)
	require.NoError(t, err)

	handler := middleware.Auth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID := middleware.GetUserID(r.Context())
		gotLogin := middleware.GetLogin(r.Context())
		assert.Equal(t, userID, gotUserID)
		assert.Equal(t, login, gotLogin)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
