package api

type LoginRequest struct {
	ID         string     `json:"id"`
	UUID       string     `json:"uuid"`
	DeviceInfo DeviceInfo `json:"deviceInfo"`
}
