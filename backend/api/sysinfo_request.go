package api

type SysinfoRequest struct {
	ID       string `json:"id"`
	UUID     string `json:"uuid"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Username string `json:"username"`
	Version  string `json:"version"`
}
