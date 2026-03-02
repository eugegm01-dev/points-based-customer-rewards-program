package migrate

import (
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

func TestMigrationsEmbedded(t *testing.T) {
	// Just check that the embed directive worked.
	entries, err := migrations.ReadDir("migrations")
	require.NoError(t, err)
	require.NotEmpty(t, entries)
}
