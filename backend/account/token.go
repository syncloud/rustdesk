package account

import "time"

type Token struct {
	User       User      `json:"user"`
	DeviceID   string    `json:"device_id"`
	DeviceUUID string    `json:"device_uuid"`
	Created    time.Time `json:"created"`
}
