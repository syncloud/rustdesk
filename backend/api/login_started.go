package api

type LoginStarted struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}
