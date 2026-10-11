package api

type DeviceLogin struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	DeviceOS   string `json:"device_os"`
	User       string `json:"user"`
}
