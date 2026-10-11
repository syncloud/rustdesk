package api

type PeerInfo struct {
	Username   string `json:"username"`
	OS         string `json:"os"`
	DeviceName string `json:"device_name"`
}
