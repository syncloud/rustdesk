package account

import "time"

type Store interface {
	SaveToken(hash string, token Token) error
	Token(hash string) (Token, bool, error)
	DeleteToken(hash string) error
	SaveDevice(device Device, limit int) (bool, error)
	Device(id string) (Device, bool, error)
	SetLastSeen(id string, at time.Time) error
	SetOwner(id, owner string) error
	Devices() ([]Device, error)
	DevicesOwnedBy(owner string) ([]Device, error)
	AddressBook(user string) (string, error)
	SaveAddressBook(user, data string) error
}
