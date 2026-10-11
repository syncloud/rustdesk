package account_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"backend/account"
)

func TestTokens_AreStoredHashedAndSurviveARestart(t *testing.T) {
	dir := t.TempDir()
	db := open(t, dir)
	raw, err := (&account.Tokens{Store: db}).Issue(account.User{Name: "boris"}, "1", "a")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	files, err := filepath.Glob(filepath.Join(dir, "*"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		content, err := os.ReadFile(file)
		require.NoError(t, err)
		assert.NotContains(t, string(content), raw)
	}

	token, found, err := (&account.Tokens{Store: open(t, dir)}).Find(raw)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "boris", token.User.Name)
}

func TestTokens_SigningInAgainOnADeviceReplacesItsToken(t *testing.T) {
	tokens := &account.Tokens{Store: open(t, t.TempDir())}
	first, err := tokens.Issue(account.User{Name: "boris"}, "1", "a")
	require.NoError(t, err)
	second, err := tokens.Issue(account.User{Name: "boris"}, "1", "a")
	require.NoError(t, err)

	_, firstFound, err := tokens.Find(first)
	require.NoError(t, err)
	_, secondFound, err := tokens.Find(second)
	require.NoError(t, err)

	assert.False(t, firstFound)
	assert.True(t, secondFound)
}

func TestTokens_ARevokedTokenIsGone(t *testing.T) {
	tokens := &account.Tokens{Store: open(t, t.TempDir())}
	raw, err := tokens.Issue(account.User{Name: "boris"}, "1", "a")
	require.NoError(t, err)

	require.NoError(t, tokens.Revoke(raw))

	_, found, err := tokens.Find(raw)
	require.NoError(t, err)
	assert.False(t, found)
}
