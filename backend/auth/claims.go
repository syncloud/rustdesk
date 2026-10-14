package auth

type claims struct {
	Sub      string `json:"sub"`
	Email    string `json:"email"`
	Username string `json:"preferred_username"`
}
