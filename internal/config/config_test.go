package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	os.Setenv("DATABASE_URI", "postgres://localhost/test")
	defer os.Unsetenv("DATABASE_URI")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, ":8080", cfg.RunAddress)
	assert.Equal(t, "postgres://localhost/test", cfg.DatabaseURI)
	assert.Equal(t, "http://localhost:8080", cfg.AccrualAddress)
	assert.Equal(t, "autotest-secret-key-change-in-production", cfg.AuthSecret)
}

func TestLoad_MissingDatabase(t *testing.T) {
	os.Unsetenv("DATABASE_URI")
	_, err := Load()
	assert.ErrorIs(t, err, ErrMissingConfig)
}
