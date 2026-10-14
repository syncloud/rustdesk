package server

import (
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newServer(t *testing.T) *Server {
	dir := t.TempDir()
	return &Server{
		Address:     "rustdesk.example.com",
		KeyFile:     path.Join(dir, "id_ed25519.pub"),
		IdSocket:    path.Join(dir, "hbbs.sock"),
		RelaySocket: path.Join(dir, "hbbr.sock"),
		DialTimeout: time.Second,
	}
}

func TestStatus_KeyNotGeneratedYet(t *testing.T) {
	status, err := newServer(t).Status()

	require.NoError(t, err)
	assert.Empty(t, status.Key)
	assert.Empty(t, status.ClientConfig)
	assert.Equal(t, "rustdesk.example.com", status.Address)
}

func TestStatus_ClientConfigPointsAtHttpsSoTheClientUsesSecureWebSocket(t *testing.T) {
	server := newServer(t)
	require.NoError(t, os.WriteFile(server.KeyFile, []byte("public-key\n"), 0600))

	status, err := server.Status()

	require.NoError(t, err)
	assert.Equal(t, "public-key", status.Key)
	expected, err := ClientConfig{
		Host: "rustdesk.example.com",
		Api:  "https://rustdesk.example.com",
		Key:  "public-key",
	}.Encode()
	require.NoError(t, err)
	assert.Equal(t, expected, status.ClientConfig)
}

func TestStatus_UnreadableKeyIsAnError(t *testing.T) {
	server := newServer(t)
	require.NoError(t, os.Mkdir(server.KeyFile, 0700))

	_, err := server.Status()

	assert.Error(t, err)
}

func TestStatus_ReportsEachDaemonBySocket(t *testing.T) {
	server := newServer(t)
	listener, err := net.Listen("unix", server.IdSocket)
	require.NoError(t, err)

	status, err := server.Status()
	require.NoError(t, err)
	assert.True(t, status.IdRunning)
	assert.False(t, status.RelayRunning)

	require.NoError(t, listener.Close())
	status, err = server.Status()
	require.NoError(t, err)
	assert.False(t, status.IdRunning)
}
