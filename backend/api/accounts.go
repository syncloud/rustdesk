package api

import "backend/account"

type Accounts interface {
	StartLogin(login account.Login) (account.StartedLogin, error)
	PendingLogin(code string) (account.Login, bool)
	ApproveLogin(code string, user account.User) bool
	ClaimLogin(code, deviceID, deviceUUID string) (account.SignedIn, bool, error)
	Authenticate(raw string) (account.Token, bool, error)
	SignOut(raw string, token account.Token) error
	Heartbeat(deviceID, deviceUUID string) (bool, error)
	Report(device account.Device) (bool, error)
	Devices() ([]account.Device, error)
	Peers(user string, page, size int) ([]account.Device, int, error)
	AddressBook(user string) (string, error)
	SaveAddressBook(user, data string) error
}
