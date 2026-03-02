package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndParseToken(t *testing.T) {
	secret := "my-secret"
	userID := "123"
	login := "alice"

	token, err := CreateToken(secret, userID, login)
	require.NoError(t, err)

	parsedUserID, parsedLogin, err := ParseToken(secret, token)
	require.NoError(t, err)
	assert.Equal(t, userID, parsedUserID)
	assert.Equal(t, login, parsedLogin)
}

func TestParseToken_Invalid(t *testing.T) {
	secret := "my-secret"
	_, _, err := ParseToken(secret, "invalid.token.string")
	assert.ErrorIs(t, err, ErrInvalidToken)
}
