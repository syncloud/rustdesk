package api

import "backend/account"

type UserPayload struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	Note        string   `json:"note"`
	Status      int      `json:"status"`
	IsAdmin     bool     `json:"is_admin"`
	Info        UserInfo `json:"info"`
}

func newUserPayload(user account.User) UserPayload {
	return UserPayload{Name: user.Name, Email: user.Email, Status: 1}
}
