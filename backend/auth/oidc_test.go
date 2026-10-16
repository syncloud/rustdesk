package auth

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newOIDC() *OIDC {
	return &OIDC{CookieSecret: []byte("test-secret")}
}

func TestSession_RoundTrip(t *testing.T) {
	o := newOIDC()

	cookie, err := o.encodeSession(session{Sub: "user", Exp: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)

	assert.True(t, o.ValidSession(cookie))
}

func TestSession_Expired(t *testing.T) {
	o := newOIDC()

	cookie, err := o.encodeSession(session{Sub: "user", Exp: time.Now().Add(-time.Hour).Unix()})
	require.NoError(t, err)

	assert.False(t, o.ValidSession(cookie))
}

func TestSession_TamperedBody(t *testing.T) {
	o := newOIDC()
	cookie, err := o.encodeSession(session{Sub: "user", Exp: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)

	forged, err := (&OIDC{CookieSecret: []byte("other-secret")}).encodeSession(
		session{Sub: "attacker", Exp: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)

	assert.False(t, o.ValidSession(forged))
	assert.NotEqual(t, cookie, forged)
}

func TestSession_Malformed(t *testing.T) {
	o := newOIDC()

	for _, raw := range []string{"", "nodot", "!!!.???", "."} {
		assert.False(t, o.ValidSession(raw), raw)
	}
}

func TestMiddleware_ApiUnauthenticatedIs401(t *testing.T) {
	o := newOIDC()
	handler := o.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/server", nil))

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestMiddleware_PageUnauthenticatedRedirects(t *testing.T) {
	o := newOIDC()
	handler := o.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, "/auth/login", recorder.Header().Get("Location"))
}

func TestMiddleware_ValidSessionPassesThrough(t *testing.T) {
	o := newOIDC()
	cookie, err := o.encodeSession(session{Sub: "user", Exp: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)

	called := false
	handler := o.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/server", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
	handler.ServeHTTP(httptest.NewRecorder(), request)

	assert.True(t, called)
}

func TestPkceChallenge_IsDeterministicAndUrlSafe(t *testing.T) {
	challenge := pkceChallenge("verifier")

	assert.Equal(t, challenge, pkceChallenge("verifier"))
	assert.NotContains(t, challenge, "+")
	assert.NotContains(t, challenge, "/")
	assert.NotContains(t, challenge, "=")
}

func TestInitWithRetry_DoesNotBlockServingWhenAuthUnreachable(t *testing.T) {
	o := &OIDC{
		IssuerURL:    "https://auth.unresolvable.invalid",
		CookieSecret: []byte("s"),
		AuthSocket:   path.Join(t.TempDir(), "absent.socket"),
		Logger:       zap.NewNop(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o.InitWithRetry(ctx, time.Hour)

	assert.False(t, o.Ready(), "discovery cannot succeed against an unresolvable host")

	recorder := httptest.NewRecorder()
	o.Login(recorder, httptest.NewRequest(http.MethodGet, "/auth/login", nil))
	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code,
		"login must report unavailable, not panic on a nil oauth2 config")

	cb := httptest.NewRecorder()
	o.Callback(cb, httptest.NewRequest(http.MethodGet, "/auth/callback", nil))
	assert.Equal(t, http.StatusServiceUnavailable, cb.Code)
}

func TestSessionsKeepWorkingWhileOidcIsDown(t *testing.T) {
	o := &OIDC{CookieSecret: []byte("secret"), Logger: zap.NewNop()}
	cookie, err := o.encodeSession(session{Sub: "u", Exp: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)

	called := false
	handler := o.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodGet, "/api/server", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	assert.False(t, o.Ready())
	assert.True(t, called, "an existing session is validated from the cookie HMAC, not from Authelia")
}

func TestAuthTransport_OverSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "a")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	socket := path.Join(dir, "s")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	defer listener.Close()

	go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"served":"over-socket","host":"` + r.Host +
			`","proto":"` + r.Header.Get("X-Forwarded-Proto") +
			`","fwdhost":"` + r.Header.Get("X-Forwarded-Host") + `"}`))
	}))

	rt := newAuthTransport(socket)
	client := &http.Client{Transport: rt, Timeout: 10 * time.Second}

	resp, err := client.Get("https://auth.example.com/.well-known/openid-configuration")
	require.NoError(t, err, "must not need DNS for the auth host")
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), "over-socket")
	assert.Contains(t, string(body), `"host":"auth.example.com"`,
		"the public host must be preserved so token issuer validation still matches")
	assert.Contains(t, string(body), `"proto":"https"`,
		"without forwarded headers Authelia advertises http://localhost as the issuer")
	assert.Contains(t, string(body), `"fwdhost":"auth.example.com"`)

	resp, err = client.Get("https://auth.example.com:10443/.well-known/openid-configuration")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(body), `"host":"auth.example.com:10443"`)
	assert.Contains(t, string(body), `"fwdhost":"auth.example.com:10443"`)
}

func TestLogout_EndsTheAutheliaSessionToo(t *testing.T) {
	o := &OIDC{
		IssuerURL:    "https://auth.example.com",
		AppURL:       "https://rustdesk.example.com",
		CookieSecret: []byte("secret"),
	}

	recorder := httptest.NewRecorder()
	o.Logout(recorder, httptest.NewRequest(http.MethodGet, "/auth/logout", nil))

	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t,
		"https://auth.example.com/logout?rd=https%3A%2F%2Frustdesk.example.com%2F",
		recorder.Header().Get("Location"),
		"clearing only the app cookie signs the user straight back in through the live SSO session")
	assert.Contains(t, recorder.Header().Get("Set-Cookie"), sessionCookie+"=;")
}

func TestUser_ComesFromAValidSessionCookie(t *testing.T) {
	o := newOIDC()
	cookie, err := o.encodeSession(session{
		Sub: "id", Name: "boris", Email: "b@example.com", Exp: time.Now().Add(time.Hour).Unix(),
	})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})

	user, signedIn := o.User(request)

	assert.True(t, signedIn)
	assert.Equal(t, "boris", user.Name)

	_, anonymous := o.User(httptest.NewRequest(http.MethodGet, "/", nil))
	assert.False(t, anonymous)
}

func TestLocalPath_OnlyAllowsPathsOnThisSite(t *testing.T) {
	assert.Equal(t, "/device?code=abc", localPath("/device?code=abc"))
	assert.Equal(t, "/", localPath(""))
	assert.Equal(t, "/", localPath("https://evil.example.com/"))
	assert.Equal(t, "/", localPath("//evil.example.com/"))
	assert.Equal(t, "/", localPath(`/\evil.example.com`))
}
