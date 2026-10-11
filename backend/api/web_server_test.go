package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"backend/server"
)

func serve(t *testing.T, keyFile string) *httptest.ResponseRecorder {
	web := NewWeb("", fixedAuth{}, &server.Server{
		Address:     "rustdesk.example.com",
		KeyFile:     keyFile,
		IdSocket:    path.Join(path.Dir(keyFile), "hbbs.sock"),
		RelaySocket: path.Join(path.Dir(keyFile), "hbbr.sock"),
		DialTimeout: time.Second,
	}, nil, zap.NewNop())

	recorder := httptest.NewRecorder()
	web.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/server", nil))
	return recorder
}

func TestServer_ReturnsStatus(t *testing.T) {
	keyFile := path.Join(t.TempDir(), "id_ed25519.pub")
	require.NoError(t, os.WriteFile(keyFile, []byte("public-key"), 0600))

	recorder := serve(t, keyFile)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var status server.Status
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &status))
	assert.Equal(t, "public-key", status.Key)
	assert.Equal(t, "rustdesk.example.com", status.Address)
	assert.NotEmpty(t, status.ClientConfig)
}

func TestServer_UnreadableKeyIs500(t *testing.T) {
	keyFile := path.Join(t.TempDir(), "id_ed25519.pub")
	require.NoError(t, os.Mkdir(keyFile, 0700))

	recorder := serve(t, keyFile)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}
