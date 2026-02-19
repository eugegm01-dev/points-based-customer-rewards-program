// Package auth provides JWT creation and verification for session cookies.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const defaultExpiry = 24 * time.Hour

// Claims holds JWT claims we put in the cookie.
type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"uid"`
	Login  string `json:"login"`
}

// CreateToken signs a new JWT for the user. Used after login/register.
func CreateToken(secret, userID, login string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(defaultExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: userID,
		Login:  login,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// ErrInvalidToken is returned when the token is missing or invalid.
var ErrInvalidToken = errors.New("invalid token")

// ParseToken verifies the token and returns user ID and login.
func ParseToken(secret, tokenString string) (userID, login string, err error) {
	t, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !t.Valid {
		return "", "", ErrInvalidToken
	}
	claims, ok := t.Claims.(*Claims)
	if !ok {
		return "", "", ErrInvalidToken
	}
	return claims.UserID, claims.Login, nil
}
