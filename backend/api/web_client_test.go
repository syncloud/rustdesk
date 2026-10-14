package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"backend/account"
	"backend/store"
)

func newMux(t *testing.T, dir string, signedIn account.User) http.Handler {
	db, err := store.Open(dir)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	accounts := account.NewService(
		"https://rustdesk.example.com/",
		db,
		&account.Tokens{Store: db},
		&account.Logins{TTL: time.Minute, Limit: 10},
		&account.Devices{Store: db, Limit: 10, SeenEvery: time.Hour},
	)

	web := NewWeb("", fixedAuth{user: signedIn}, nil, accounts, zap.NewNop())
	return web.Handler()
}

func call(mux http.Handler, method, target, token string, body any) *httptest.ResponseRecorder {
	payload := []byte{}
	if raw, isRaw := body.(string); isRaw {
		payload = []byte(raw)
	} else if body != nil {
		payload, _ = json.Marshal(body)
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(payload))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	return recorder
}

func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	var body T
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body), recorder.Body.String())
	return body
}

func signIn(t *testing.T, mux http.Handler, id, uuid string) string {
	started := decode[LoginStarted](t, call(mux, http.MethodPost, "/api/oidc/auth", "", LoginRequest{
		ID: id, UUID: uuid, DeviceInfo: DeviceInfo{OS: "linux", Name: "laptop"},
	}))
	code := started.Code
	approved := call(mux, http.MethodPost, "/api/device-login", "", ApprovalRequest{Code: code})
	require.Equal(t, http.StatusOK, approved.Code, approved.Body.String())
	query := "/api/oidc/auth-query?code=" + url.QueryEscape(code) + "&id=" + id + "&uuid=" + uuid
	return decode[LoginResult](t, call(mux, http.MethodGet, query, "", nil)).AccessToken
}

func TestClient_LoginOptionsOfferSyncloud(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{})

	recorder := call(mux, http.MethodGet, "/api/login-options", "", nil)

	assert.JSONEq(t, `["oidc/syncloud"]`, recorder.Body.String())
}

func TestClient_PasswordLoginIsRefused(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{})

	recorder := call(mux, http.MethodPost, "/api/login", "", `{"username":"a"}`)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.NotEmpty(t, decode[ErrorResponse](t, recorder).Error)
}

func TestClient_LoginStaysPendingUntilApproved(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{Name: "boris"})
	started := decode[LoginStarted](t, call(mux, http.MethodPost, "/api/oidc/auth", "", LoginRequest{
		ID: "123", UUID: "abc",
	}))
	code := started.Code
	assert.Equal(t, "https://rustdesk.example.com/device?code="+url.QueryEscape(code), started.URL)
	query := "/api/oidc/auth-query?code=" + url.QueryEscape(code) + "&id=123&uuid=abc"

	pending := decode[ErrorResponse](t, call(mux, http.MethodGet, query, "", nil))
	assert.Equal(t, "No authed oidc is found", pending.Error)

	shown := decode[DeviceLogin](t, call(mux, http.MethodGet, "/api/device-login?code="+url.QueryEscape(code), "", nil))
	assert.Equal(t, "123", shown.DeviceID)
	assert.Equal(t, "boris", shown.User)

	require.Equal(t, http.StatusOK,
		call(mux, http.MethodPost, "/api/device-login", "", ApprovalRequest{Code: code}).Code)

	claimed := call(mux, http.MethodGet, query, "", nil)
	done := decode[LoginResult](t, claimed)
	assert.Equal(t, "access_token", done.Type)
	assert.NotEmpty(t, done.AccessToken)
	assert.Equal(t, "boris", done.User.Name)
	assert.Contains(t, claimed.Body.String(), `"info":{}`)

	again := decode[ErrorResponse](t, call(mux, http.MethodGet, query, "", nil))
	assert.Equal(t, "No authed oidc is found", again.Error)
}

func TestClient_AnotherDeviceCannotClaimAnApprovedLogin(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{Name: "boris"})
	started := decode[LoginStarted](t, call(mux, http.MethodPost, "/api/oidc/auth", "", LoginRequest{
		ID: "123", UUID: "abc",
	}))
	code := started.Code
	call(mux, http.MethodPost, "/api/device-login", "", ApprovalRequest{Code: code})

	stolen := decode[ErrorResponse](t, call(mux, http.MethodGet,
		"/api/oidc/auth-query?code="+url.QueryEscape(code)+"&id=999&uuid=abc", "", nil))

	assert.Equal(t, "No authed oidc is found", stolen.Error)
}

func TestClient_ApprovalNeedsAWebSession(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{})
	started := decode[LoginStarted](t, call(mux, http.MethodPost, "/api/oidc/auth", "", LoginRequest{
		ID: "123", UUID: "abc",
	}))

	recorder := call(mux, http.MethodPost, "/api/device-login", "",
		ApprovalRequest{Code: started.Code})

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestClient_UnknownApprovalCodeIs404(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{Name: "boris"})

	recorder := call(mux, http.MethodPost, "/api/device-login", "", ApprovalRequest{Code: "nope"})

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestClient_CurrentUserNeedsAToken(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{Name: "boris"})

	assert.Equal(t, http.StatusUnauthorized,
		call(mux, http.MethodPost, "/api/currentUser", "", Empty{}).Code)
	assert.Equal(t, http.StatusUnauthorized,
		call(mux, http.MethodPost, "/api/currentUser", "forged", Empty{}).Code)

	token := signIn(t, mux, "123", "abc")
	user := decode[UserPayload](t, call(mux, http.MethodPost, "/api/currentUser", token, Empty{}))
	assert.Equal(t, "boris", user.Name)
}

func TestClient_LogoutRevokesTheToken(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{Name: "boris"})
	token := signIn(t, mux, "123", "abc")

	require.Equal(t, http.StatusOK, call(mux, http.MethodPost, "/api/logout", token, Empty{}).Code)

	assert.Equal(t, http.StatusUnauthorized,
		call(mux, http.MethodPost, "/api/currentUser", token, Empty{}).Code)
}

func TestClient_AddressBookRoundTripAndSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	mux := newMux(t, dir, account.User{Name: "boris"})
	token := signIn(t, mux, "123", "abc")

	empty := decode[AddressBookPayload](t, call(mux, http.MethodGet, "/api/ab", token, nil))
	assert.Equal(t, "{}", empty.Data)

	book := `{"tags":["home"],"peers":[{"id":"456","alias":"desk"}]}`
	saved := call(mux, http.MethodPost, "/api/ab", token, AddressBookPayload{Data: book})
	require.Equal(t, http.StatusOK, saved.Code)
	assert.Empty(t, saved.Body.String())

	restarted := newMux(t, dir, account.User{Name: "boris"})
	loaded := decode[AddressBookPayload](t, call(restarted, http.MethodGet, "/api/ab", token, nil))
	assert.Equal(t, book, loaded.Data)
}

func TestClient_AddressBookIsPerUser(t *testing.T) {
	dir := t.TempDir()
	boris := signIn(t, newMux(t, dir, account.User{Name: "boris"}), "123", "abc")
	mux := newMux(t, dir, account.User{Name: "anna"})
	anna := signIn(t, mux, "456", "def")
	call(mux, http.MethodPost, "/api/ab", boris, AddressBookPayload{Data: `{"peers":[]}`})

	assert.Equal(t, "{}", decode[AddressBookPayload](t, call(mux, http.MethodGet, "/api/ab", anna, nil)).Data)
}

func TestClient_AddressBookRejectsBrokenData(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{Name: "boris"})
	token := signIn(t, mux, "123", "abc")

	recorder := call(mux, http.MethodPost, "/api/ab", token, AddressBookPayload{Data: "{"})

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestClient_AddressBookNeedsAToken(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{Name: "boris"})

	assert.Equal(t, http.StatusUnauthorized, call(mux, http.MethodGet, "/api/ab", "", nil).Code)
	assert.Equal(t, http.StatusUnauthorized,
		call(mux, http.MethodPost, "/api/ab", "", AddressBookPayload{Data: "{}"}).Code)
}

func TestClient_SharedAddressBookEndpointsAre404SoTheClientUsesTheLegacyOne(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{Name: "boris"})
	token := signIn(t, mux, "123", "abc")

	assert.Equal(t, http.StatusNotFound, call(mux, http.MethodPost, "/api/ab/personal", token, nil).Code)
	assert.Equal(t, http.StatusNotFound, call(mux, http.MethodPost, "/api/ab/settings", token, nil).Code)
}

func TestClient_HeartbeatAsksAnUnknownDeviceForSysinfo(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{})

	unknown := decode[HeartbeatResponse](t, call(mux, http.MethodPost, "/api/heartbeat", "",
		HeartbeatRequest{ID: "123", UUID: "abc"}))
	assert.True(t, unknown.Sysinfo)

	reported := call(mux, http.MethodPost, "/api/sysinfo", "", SysinfoRequest{
		ID: "123", UUID: "abc", Hostname: "desk", OS: "linux / Ubuntu", Version: "1.5.0",
	})
	assert.Equal(t, "SYSINFO_UPDATED", reported.Body.String())

	known := call(mux, http.MethodPost, "/api/heartbeat", "", HeartbeatRequest{ID: "123", UUID: "abc"})
	assert.JSONEq(t, `{}`, known.Body.String())
}

func TestClient_SysinfoFromAnotherUuidDoesNotOverwriteADevice(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{Name: "boris"})
	call(mux, http.MethodPost, "/api/sysinfo", "", SysinfoRequest{ID: "123", UUID: "abc", Hostname: "desk"})

	forged := call(mux, http.MethodPost, "/api/sysinfo", "",
		SysinfoRequest{ID: "123", UUID: "other", Hostname: "evil"})

	assert.Equal(t, "ID_NOT_FOUND", forged.Body.String())
	devices := []account.Device{}
	require.NoError(t, json.Unmarshal(call(mux, http.MethodGet, "/api/devices", "", nil).Body.Bytes(), &devices))
	require.Len(t, devices, 1)
	assert.Equal(t, "desk", devices[0].Hostname)
}

func TestClient_PeersListsDevicesSignedInAsTheUser(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{Name: "boris"})
	call(mux, http.MethodPost, "/api/sysinfo", "", SysinfoRequest{
		ID: "123", UUID: "abc", Hostname: "desk", OS: "linux / Ubuntu", Username: "boris",
	})
	call(mux, http.MethodPost, "/api/sysinfo", "", SysinfoRequest{ID: "777", UUID: "zzz", Hostname: "other"})
	token := signIn(t, mux, "123", "abc")

	peers := decode[Page[PeerPayload]](t, call(mux, http.MethodGet, "/api/peers?current=1&pageSize=100", token, nil))
	assert.Equal(t, 1, peers.Total)
	require.Len(t, peers.Data, 1)
	assert.Equal(t, "123", peers.Data[0].ID)
	assert.Equal(t, "desk", peers.Data[0].Info.DeviceName)

	users := decode[Page[UserPayload]](t, call(mux, http.MethodGet, "/api/users?current=1&pageSize=100", token, nil))
	assert.Equal(t, 1, users.Total)

	second := decode[Page[PeerPayload]](t, call(mux, http.MethodGet, "/api/peers?current=2&pageSize=100", token, nil))
	assert.Empty(t, second.Data)
}

func TestClient_OversizedRequestIsRefused(t *testing.T) {
	mux := newMux(t, t.TempDir(), account.User{})

	recorder := call(mux, http.MethodPost, "/api/sysinfo", "", `{"id":"`+string(bytes.Repeat([]byte("a"), 70<<10))+`"}`)

	assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
}
