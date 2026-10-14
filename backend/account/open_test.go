package account_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"backend/store"
)

func open(t *testing.T, dir string) *store.DB {
	db, err := store.Open(dir)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}
