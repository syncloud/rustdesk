package config

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	file := path.Join(t.TempDir(), "backend.env")
	require.NoError(t, os.WriteFile(file, []byte(
		"# generated\n"+
			"APP_DOMAIN=rustdesk.example.com\n"+
			"APP_URL=https://rustdesk.example.com\n"+
			"OIDC_CLIENT_ID=rustdesk\n"+
			"OIDC_CLIENT_SECRET=\"quoted secret\"\n"+
			"OIDC_AUTH_BASE_URL=https://auth.example.com\n"+
			"AUTH_LOCAL_SOCKET=/var/snap/platform/current/authelia.socket\n"+
			"OIDC_REDIRECT_URI=https://rustdesk.example.com/auth/callback\n"+
			"SESSION_SECRET=abc123\n"+
			"SOCKET=/var/snap/rustdesk/current/backend.sock\n"+
			"KEY_FILE=/var/snap/rustdesk/current/server/id_ed25519.pub\n"+
			"ID_SOCKET=/var/snap/rustdesk/current/hbbs.sock\n"+
			"RELAY_SOCKET=/var/snap/rustdesk/current/hbbr.sock\n"+
			"ACCOUNT_DIR=/var/snap/rustdesk/current/account\n"+
			"\n"), 0644))

	cfg, err := Load(file)
	require.NoError(t, err)

	assert.Equal(t, "rustdesk.example.com", cfg.AppDomain)
	assert.Equal(t, "quoted secret", cfg.OIDCClientSecret)
	assert.Equal(t, "https://auth.example.com", cfg.OIDCAuthBaseURL)
	assert.Equal(t, "/var/snap/platform/current/authelia.socket", cfg.AuthSocket)
	assert.Equal(t, "/var/snap/rustdesk/current/account", cfg.AccountDir)
	assert.Equal(t, "abc123", cfg.SessionSecret)
	assert.Equal(t, "/var/snap/rustdesk/current/hbbs.sock", cfg.IdSocket)
	assert.Equal(t, "/var/snap/rustdesk/current/hbbr.sock", cfg.RelaySocket)
	assert.Equal(t, "/var/snap/rustdesk/current/server/id_ed25519.pub", cfg.KeyFile)
}

func TestLoad_MissingKeyIsAnError(t *testing.T) {
	file := path.Join(t.TempDir(), "backend.env")
	require.NoError(t, os.WriteFile(file, []byte("APP_DOMAIN=rustdesk.example.com\n"), 0644))

	_, err := Load(file)

	assert.ErrorContains(t, err, "APP_URL")
}

func TestLoad_Missing(t *testing.T) {
	_, err := Load(path.Join(t.TempDir(), "absent.env"))
	assert.Error(t, err)
}
