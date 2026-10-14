package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"backend/account"
)

const (
	loginProvider    = "syncloud"
	maxRequestBody   = 64 << 10
	maxAddressBook   = 2 << 20
	loginPending     = "No authed oidc is found"
	loginExpired     = "This sign-in request has expired. Start again from the RustDesk client."
	sysinfoUpdated   = "SYSINFO_UPDATED"
	sysinfoIDUnknown = "ID_NOT_FOUND"
)

type Web struct {
	socket   string
	auth     Auth
	status   ServerStatus
	accounts Accounts
	logger   *zap.Logger
}

func NewWeb(socket string, auth Auth, status ServerStatus, accounts Accounts, logger *zap.Logger) *Web {
	return &Web{socket: socket, auth: auth, status: status, accounts: accounts, logger: logger}
}

func (w *Web) Start() error {
	w.auth.Start()
	_ = os.Remove(w.socket)
	listener, err := net.Listen("unix", w.socket)
	if err != nil {
		return fmt.Errorf("listen %s: %w", w.socket, err)
	}
	w.logger.Info("backend listening", zap.String("socket", w.socket))
	return (&http.Server{Handler: w.Handler()}).Serve(listener)
}

func (w *Web) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/login", w.auth.Login)
	mux.HandleFunc("GET /auth/callback", w.auth.Callback)
	mux.HandleFunc("GET /auth/logout", w.auth.Logout)
	mux.Handle("GET /api/server", w.auth.Middleware(http.HandlerFunc(w.server)))
	mux.Handle("GET /api/devices", w.auth.Middleware(http.HandlerFunc(w.devices)))
	mux.Handle("GET /api/device-login", w.auth.Middleware(http.HandlerFunc(w.deviceLogin)))
	mux.Handle("POST /api/device-login", w.auth.Middleware(http.HandlerFunc(w.approveDeviceLogin)))
	mux.HandleFunc("GET /api/login-options", w.loginOptions)
	mux.HandleFunc("POST /api/login", w.passwordLogin)
	mux.HandleFunc("POST /api/oidc/auth", w.startLogin)
	mux.HandleFunc("GET /api/oidc/auth-query", w.queryLogin)
	mux.HandleFunc("POST /api/heartbeat", w.heartbeat)
	mux.HandleFunc("POST /api/sysinfo", w.sysinfo)
	mux.HandleFunc("POST /api/currentUser", w.authorized(w.currentUser))
	mux.HandleFunc("POST /api/logout", w.authorized(w.logout))
	mux.HandleFunc("GET /api/ab", w.authorized(w.addressBook))
	mux.HandleFunc("POST /api/ab", w.authorized(w.saveAddressBook))
	mux.HandleFunc("GET /api/users", w.authorized(w.users))
	mux.HandleFunc("GET /api/peers", w.authorized(w.peers))
	mux.HandleFunc("/api/", notFound)
	return mux
}

func (w *Web) server(rw http.ResponseWriter, _ *http.Request) {
	status, err := w.status.Status()
	if err != nil {
		w.logger.Error("server status", zap.Error(err))
		writeError(rw, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(rw, http.StatusOK, status)
}

func (w *Web) devices(rw http.ResponseWriter, _ *http.Request) {
	devices, err := w.accounts.Devices()
	if err != nil {
		w.logger.Error("devices", zap.Error(err))
		writeError(rw, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(rw, http.StatusOK, devices)
}

func (w *Web) deviceLogin(rw http.ResponseWriter, r *http.Request) {
	user, signedIn := w.auth.User(r)
	if !signedIn {
		writeError(rw, http.StatusUnauthorized, "unauthenticated")
		return
	}
	login, found := w.accounts.PendingLogin(r.URL.Query().Get("code"))
	if !found {
		writeError(rw, http.StatusNotFound, loginExpired)
		return
	}
	writeJSON(rw, http.StatusOK, DeviceLogin{
		DeviceID:   login.DeviceID,
		DeviceName: login.DeviceName,
		DeviceOS:   login.DeviceOS,
		User:       user.Name,
	})
}

func (w *Web) approveDeviceLogin(rw http.ResponseWriter, r *http.Request) {
	user, signedIn := w.auth.User(r)
	if !signedIn {
		writeError(rw, http.StatusUnauthorized, "unauthenticated")
		return
	}
	request := ApprovalRequest{}
	if !readJSON(rw, r, maxRequestBody, &request) {
		return
	}
	if !w.accounts.ApproveLogin(request.Code, user) {
		writeError(rw, http.StatusNotFound, loginExpired)
		return
	}
	writeJSON(rw, http.StatusOK, Empty{})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func (w *Web) loginOptions(rw http.ResponseWriter, _ *http.Request) {
	writeJSON(rw, http.StatusOK, []string{"oidc/" + loginProvider})
}

func (w *Web) passwordLogin(rw http.ResponseWriter, _ *http.Request) {
	writeError(rw, http.StatusBadRequest, "Sign in with the Syncloud button below")
}

func (w *Web) startLogin(rw http.ResponseWriter, r *http.Request) {
	request := LoginRequest{}
	if !readJSON(rw, r, maxRequestBody, &request) {
		return
	}
	started, err := w.accounts.StartLogin(account.Login{
		DeviceID:   request.ID,
		DeviceUUID: request.UUID,
		DeviceName: request.DeviceInfo.Name,
		DeviceOS:   request.DeviceInfo.OS,
	})
	if errors.Is(err, account.ErrDeviceRequired) {
		writeError(rw, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, account.ErrTooManyLogins) {
		writeError(rw, http.StatusTooManyRequests, err.Error())
		return
	}
	if err != nil {
		w.fail(rw, "start login", err)
		return
	}
	writeJSON(rw, http.StatusOK, LoginStarted{Code: started.Code, URL: started.URL})
}

func (w *Web) queryLogin(rw http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	signedIn, approved, err := w.accounts.ClaimLogin(query.Get("code"), query.Get("id"), query.Get("uuid"))
	if err != nil {
		w.fail(rw, "claim login", err)
		return
	}
	if !approved {
		writeError(rw, http.StatusOK, loginPending)
		return
	}
	writeJSON(rw, http.StatusOK, LoginResult{
		AccessToken: signedIn.Token,
		Type:        "access_token",
		User:        newUserPayload(signedIn.User),
	})
}

func (w *Web) heartbeat(rw http.ResponseWriter, r *http.Request) {
	request := HeartbeatRequest{}
	if !readJSON(rw, r, maxRequestBody, &request) {
		return
	}
	known, err := w.accounts.Heartbeat(request.ID, request.UUID)
	if err != nil {
		w.fail(rw, "heartbeat", err)
		return
	}
	writeJSON(rw, http.StatusOK, HeartbeatResponse{Sysinfo: !known})
}

func (w *Web) sysinfo(rw http.ResponseWriter, r *http.Request) {
	request := SysinfoRequest{}
	if !readJSON(rw, r, maxRequestBody, &request) {
		return
	}
	accepted, err := w.accounts.Report(account.Device{
		ID:       request.ID,
		UUID:     request.UUID,
		Hostname: request.Hostname,
		OS:       request.OS,
		Username: request.Username,
		Version:  request.Version,
	})
	if err != nil {
		w.fail(rw, "sysinfo", err)
		return
	}
	if !accepted {
		writeText(rw, sysinfoIDUnknown)
		return
	}
	writeText(rw, sysinfoUpdated)
}

func (w *Web) currentUser(rw http.ResponseWriter, _ *http.Request, token account.Token) {
	writeJSON(rw, http.StatusOK, newUserPayload(token.User))
}

func (w *Web) logout(rw http.ResponseWriter, r *http.Request, token account.Token) {
	if err := w.accounts.SignOut(bearer(r), token); err != nil {
		w.fail(rw, "sign out", err)
		return
	}
	writeJSON(rw, http.StatusOK, Empty{})
}

func (w *Web) addressBook(rw http.ResponseWriter, _ *http.Request, token account.Token) {
	data, err := w.accounts.AddressBook(token.User.Name)
	if err != nil {
		w.fail(rw, "read address book", err)
		return
	}
	writeJSON(rw, http.StatusOK, AddressBookPayload{Data: data})
}

func (w *Web) saveAddressBook(rw http.ResponseWriter, r *http.Request, token account.Token) {
	request := AddressBookPayload{}
	if !readJSON(rw, r, maxAddressBook, &request) {
		return
	}
	err := w.accounts.SaveAddressBook(token.User.Name, request.Data)
	if errors.Is(err, account.ErrInvalidAddressBook) {
		writeError(rw, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		w.fail(rw, "save address book", err)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func (w *Web) users(rw http.ResponseWriter, r *http.Request, token account.Token) {
	users := []UserPayload{}
	if page(r) == 1 {
		users = append(users, newUserPayload(token.User))
	}
	writeJSON(rw, http.StatusOK, Page[UserPayload]{Total: 1, Data: users})
}

func (w *Web) peers(rw http.ResponseWriter, r *http.Request, token account.Token) {
	devices, total, err := w.accounts.Peers(token.User.Name, page(r), pageSize(r))
	if err != nil {
		w.fail(rw, "peers", err)
		return
	}
	peers := []PeerPayload{}
	for _, device := range devices {
		peers = append(peers, newPeerPayload(device))
	}
	writeJSON(rw, http.StatusOK, Page[PeerPayload]{Total: total, Data: peers})
}

func (w *Web) authorized(next func(http.ResponseWriter, *http.Request, account.Token)) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		token, found, err := w.accounts.Authenticate(bearer(r))
		if err != nil {
			w.fail(rw, "token", err)
			return
		}
		if !found {
			writeError(rw, http.StatusUnauthorized, "Sign in again")
			return
		}
		next(rw, r, token)
	}
}

func (w *Web) fail(rw http.ResponseWriter, what string, err error) {
	w.logger.Error(what, zap.Error(err))
	writeError(rw, http.StatusInternalServerError, err.Error())
}

func bearer(r *http.Request) string {
	value, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !found {
		return ""
	}
	return strings.TrimSpace(value)
}

func page(r *http.Request) int {
	return positive(r.URL.Query().Get("current"), 1)
}

func pageSize(r *http.Request) int {
	return min(positive(r.URL.Query().Get("pageSize"), 100), 100)
}

func positive(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func readJSON(w http.ResponseWriter, r *http.Request, limit int64, value any) bool {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "request is too large")
		return false
	}
	if err := json.Unmarshal(body, value); err != nil {
		writeError(w, http.StatusBadRequest, "request is not valid json")
		return false
	}
	return true
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, "not found")
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, ErrorResponse{Error: message})
}

func writeText(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = io.WriteString(w, body)
}
