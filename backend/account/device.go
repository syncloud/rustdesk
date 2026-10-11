package account

import "time"

type Device struct {
	ID       string    `json:"id"`
	UUID     string    `json:"uuid"`
	Hostname string    `json:"hostname"`
	OS       string    `json:"os"`
	Username string    `json:"username"`
	Version  string    `json:"version"`
	Owner    string    `json:"owner"`
	LastSeen time.Time `json:"last_seen"`
}
