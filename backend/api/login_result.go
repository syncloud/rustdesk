package api

type LoginResult struct {
	AccessToken string      `json:"access_token"`
	Type        string      `json:"type"`
	User        UserPayload `json:"user"`
}
