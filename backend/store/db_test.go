package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_CreatesAPrivateDatabase(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "account")

	db, err := Open(dir)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.SaveAddressBook("boris", "{}"))

	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, file := range files {
		info, err := file.Info()
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), file.Name())
	}
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

func TestOpen_ReopeningKeepsTheDataAndTheSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	first, err := Open(dir)
	require.NoError(t, err)
	require.NoError(t, first.SaveAddressBook("boris", "{}"))
	require.NoError(t, first.Close())

	second, err := Open(dir)
	require.NoError(t, err)
	defer second.Close()

	version := 0
	require.NoError(t, second.db.QueryRow("PRAGMA user_version").Scan(&version))
	data, err := second.AddressBook("boris")
	require.NoError(t, err)
	assert.Equal(t, len(schema), version)
	assert.Equal(t, "{}", data)
}

func TestOpen_RefusesADatabaseFromANewerVersion(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	require.NoError(t, err)
	_, err = db.db.Exec("PRAGMA user_version = 999")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	_, err = Open(dir)

	assert.ErrorContains(t, err, "newer")
}
