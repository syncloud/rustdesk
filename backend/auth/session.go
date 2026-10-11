package auth

type session struct {
	Sub   string `json:"sub"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
}
