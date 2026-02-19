// Package middleware provides HTTP middleware (auth, etc.).
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/auth"
)

type contextKey string

const (
	// CookieName is the name of the auth cookie (JWT).
	CookieName = "token"
	// ContextKeyUserID is the context key for the authenticated user ID.
	ContextKeyUserID contextKey = "user_id"
	// ContextKeyLogin is the context key for the authenticated user login.
	ContextKeyLogin contextKey = "login"
)

// Auth reads the JWT from cookie (or Authorization: Bearer), verifies it, and sets user in context.
// If no valid token, it returns 401 and does not call next.
func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""
			if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
				tokenStr = c.Value
			}
			if tokenStr == "" {
				if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
					tokenStr = strings.TrimPrefix(h, "Bearer ")
				}
			}
			if tokenStr == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			userID, login, err := auth.ParseToken(secret, tokenStr)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
			ctx = context.WithValue(ctx, ContextKeyLogin, login)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID returns the authenticated user ID from context, or "" if not set.
func GetUserID(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyUserID).(string)
	return v
}

// GetLogin returns the authenticated user login from context, or "" if not set.
func GetLogin(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyLogin).(string)
	return v
}
