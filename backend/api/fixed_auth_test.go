package api

import (
	"net/http"

	"backend/account"
)

type fixedAuth struct {
	user account.User
}

func (f fixedAuth) Start() {}

func (f fixedAuth) Login(_ http.ResponseWriter, _ *http.Request) {}

func (f fixedAuth) Callback(_ http.ResponseWriter, _ *http.Request) {}

func (f fixedAuth) Logout(_ http.ResponseWriter, _ *http.Request) {}

func (f fixedAuth) Middleware(next http.Handler) http.Handler {
	return next
}

func (f fixedAuth) User(_ *http.Request) (account.User, bool) {
	return f.user, f.user.Name != ""
}
