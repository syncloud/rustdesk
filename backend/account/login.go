package account

import "time"

type Login struct {
	DeviceID   string
	DeviceUUID string
	DeviceName string
	DeviceOS   string
	Approved   *User
	Expires    time.Time
}
