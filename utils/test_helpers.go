package utils

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func getMigrations(t *testing.T) []string {
	// Read directory entries
	migrationDir := filepath.Join("../", "migrations")
	entries, err := os.ReadDir(migrationDir)
	require.NoError(t, err)

	// Sort entries by name to ensure correct order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// Build a slice of strings with full paths
	var scripts []string
	for _, entry := range entries {
		if !entry.IsDir() {
			scripts = append(scripts, filepath.Join(migrationDir, entry.Name()))
		}
	}

	return scripts
}

func StartTestPostgres(ctx context.Context, t *testing.T, dbname, user, password string) *postgres.PostgresContainer {
	migrations := getMigrations(t)

	ctr, err := postgres.Run(ctx,
		"postgres:16",
		postgres.WithDatabase(dbname),
		postgres.WithUsername(user),
		postgres.WithPassword(password),
		postgres.BasicWaitStrategies(),
		postgres.WithInitScripts(migrations...),
	)
	require.NoError(t, err)

	// Clean up the container after the test is complete
	t.Cleanup(func() {
		if err := ctr.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	return ctr
}
