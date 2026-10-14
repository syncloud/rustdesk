package api

import "backend/account"

type PeerPayload struct {
	ID       string   `json:"id"`
	User     string   `json:"user"`
	UserName string   `json:"user_name"`
	Status   int      `json:"status"`
	Note     string   `json:"note"`
	Info     PeerInfo `json:"info"`
}

func newPeerPayload(device account.Device) PeerPayload {
	return PeerPayload{
		ID:       device.ID,
		User:     device.Owner,
		UserName: device.Owner,
		Status:   1,
		Info:     PeerInfo{Username: device.Username, OS: device.OS, DeviceName: device.Hostname},
	}
}
