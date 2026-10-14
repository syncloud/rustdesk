package api

import (
	"net/http"

	"backend/account"
)

type Auth interface {
	Start()
	Login(w http.ResponseWriter, r *http.Request)
	Callback(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	Middleware(next http.Handler) http.Handler
	User(r *http.Request) (account.User, bool)
}
