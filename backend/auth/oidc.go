package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"backend/account"
)

const (
	stateCookie    = "rustdesk_oidc_state"
	verifierCookie = "rustdesk_oidc_verifier"
	sessionCookie  = "rustdesk_session"
	nextCookie     = "rustdesk_next"
	sessionTTL     = 12 * time.Hour
	initRetry      = 30 * time.Second
	flowTTL        = 10 * time.Minute
)

type OIDC struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AppURL       string
	CookieSecret []byte
	AuthSocket   string
	Logger       *zap.Logger

	client       *http.Client
	mu           sync.RWMutex
	ready        bool
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
}

func (o *OIDC) Init(ctx context.Context) error {
	o.client = &http.Client{Timeout: 30 * time.Second, Transport: newAuthTransport(o.AuthSocket)}
	provider, err := oidc.NewProvider(o.context(ctx), o.IssuerURL)
	if err != nil {
		return fmt.Errorf("oidc provider discovery: %w", err)
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.provider = provider
	o.verifier = provider.Verifier(&oidc.Config{ClientID: o.ClientID})
	o.oauth2Config = oauth2.Config{
		ClientID:     o.ClientID,
		ClientSecret: o.ClientSecret,
		RedirectURL:  o.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	o.ready = true
	return nil
}

func (o *OIDC) Start() {
	o.InitWithRetry(context.Background(), initRetry)
	if !o.Ready() {
		o.Logger.Warn("oidc not ready yet, serving anyway and retrying in the background")
	}
}

func (o *OIDC) InitWithRetry(ctx context.Context, every time.Duration) {
	if err := o.Init(ctx); err == nil {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(every):
			}
			if err := o.Init(ctx); err != nil {
				o.Logger.Warn("oidc discovery failed, retrying", zap.Error(err))
				continue
			}
			o.Logger.Info("oidc discovery succeeded")
			return
		}
	}()
}

func (o *OIDC) Ready() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.ready
}

func (o *OIDC) Login(w http.ResponseWriter, r *http.Request) {
	if !o.Ready() {
		http.Error(w, "sign-in is unavailable: the device cannot reach its auth service yet",
			http.StatusServiceUnavailable)
		return
	}
	state, err := randBase64(16)
	if err != nil {
		http.Error(w, "state", http.StatusInternalServerError)
		return
	}
	verifier, err := randBase64(32)
	if err != nil {
		http.Error(w, "verifier", http.StatusInternalServerError)
		return
	}

	setCookie(w, stateCookie, state, flowTTL)
	setCookie(w, verifierCookie, verifier, flowTTL)
	setCookie(w, nextCookie, localPath(r.URL.Query().Get("next")), flowTTL)

	http.Redirect(w, r, o.oauth2Config.AuthCodeURL(
		state,
		oauth2.AccessTypeOnline,
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge(verifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	), http.StatusFound)
}

func (o *OIDC) Callback(w http.ResponseWriter, r *http.Request) {
	if !o.Ready() {
		http.Error(w, "sign-in is unavailable: the device cannot reach its auth service yet",
			http.StatusServiceUnavailable)
		return
	}
	state, err := r.Cookie(stateCookie)
	if err != nil {
		http.Error(w, "state cookie missing", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != state.Value {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	verifier, err := r.Cookie(verifierCookie)
	if err != nil {
		http.Error(w, "verifier cookie missing", http.StatusBadRequest)
		return
	}

	ctx := o.context(r.Context())
	token, err := o.oauth2Config.Exchange(ctx, r.URL.Query().Get("code"),
		oauth2.SetAuthURLParam("code_verifier", verifier.Value))
	if err != nil {
		o.Logger.Error("oauth2 exchange", zap.Error(err))
		http.Error(w, "exchange failed", http.StatusBadGateway)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "id_token missing", http.StatusBadGateway)
		return
	}
	idToken, err := o.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		o.Logger.Error("id token verify", zap.Error(err))
		http.Error(w, "id token invalid", http.StatusUnauthorized)
		return
	}

	claims := claims{}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "claims", http.StatusBadGateway)
		return
	}

	name := claims.Username
	if name == "" {
		name = claims.Sub
	}
	cookie, err := o.encodeSession(session{
		Sub:   claims.Sub,
		Name:  name,
		Email: claims.Email,
		Exp:   time.Now().Add(sessionTTL).Unix(),
	})
	if err != nil {
		http.Error(w, "encode session", http.StatusInternalServerError)
		return
	}

	next := "/"
	if stored, err := r.Cookie(nextCookie); err == nil {
		next = localPath(stored.Value)
	}

	clearCookie(w, stateCookie)
	clearCookie(w, verifierCookie)
	clearCookie(w, nextCookie)
	setCookie(w, sessionCookie, cookie, sessionTTL)
	http.Redirect(w, r, next, http.StatusFound)
}

func (o *OIDC) Logout(w http.ResponseWriter, r *http.Request) {
	clearCookie(w, sessionCookie)
	http.Redirect(w, r,
		strings.TrimRight(o.IssuerURL, "/")+"/logout?rd="+url.QueryEscape(strings.TrimRight(o.AppURL, "/")+"/"),
		http.StatusFound)
}

func (o *OIDC) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil || !o.ValidSession(cookie.Value) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, "unauthenticated", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (o *OIDC) encodeSession(s session) (string, error) {
	body, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(body) + "." +
		base64.URLEncoding.EncodeToString(o.sign(body)), nil
}

func (o *OIDC) ValidSession(raw string) bool {
	_, valid := o.decodeSession(raw)
	return valid
}

func (o *OIDC) User(r *http.Request) (account.User, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return account.User{}, false
	}
	s, valid := o.decodeSession(cookie.Value)
	if !valid || s.Name == "" {
		return account.User{}, false
	}
	return account.User{Name: s.Name, Email: s.Email}, true
}

func (o *OIDC) decodeSession(raw string) (session, bool) {
	encodedBody, encodedSig, found := strings.Cut(raw, ".")
	if !found {
		return session{}, false
	}
	body, err := base64.URLEncoding.DecodeString(encodedBody)
	if err != nil {
		return session{}, false
	}
	sig, err := base64.URLEncoding.DecodeString(encodedSig)
	if err != nil {
		return session{}, false
	}
	if !hmac.Equal(sig, o.sign(body)) {
		return session{}, false
	}
	var s session
	if err := json.Unmarshal(body, &s); err != nil {
		return session{}, false
	}
	return s, time.Now().Unix() < s.Exp
}

func localPath(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.Contains(next, "\\") {
		return "/"
	}
	return next
}

func (o *OIDC) sign(body []byte) []byte {
	mac := hmac.New(sha256.New, o.CookieSecret)
	mac.Write(body)
	return mac.Sum(nil)
}

func setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
	})
}

func (o *OIDC) context(ctx context.Context) context.Context {
	if o.client == nil {
		return ctx
	}
	return oidc.ClientContext(ctx, o.client)
}

func randBase64(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
