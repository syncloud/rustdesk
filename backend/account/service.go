package account

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

const emptyAddressBook = "{}"

var (
	ErrDeviceRequired     = errors.New("id and uuid are required")
	ErrInvalidAddressBook = errors.New("address book is not valid json")
)

type Service struct {
	appURL  string
	store   Store
	tokens  *Tokens
	logins  *Logins
	devices *Devices
}

func NewService(appURL string, store Store, tokens *Tokens, logins *Logins, devices *Devices) *Service {
	return &Service{
		appURL:  strings.TrimRight(appURL, "/"),
		store:   store,
		tokens:  tokens,
		logins:  logins,
		devices: devices,
	}
}

func (s *Service) StartLogin(login Login) (StartedLogin, error) {
	if login.DeviceID == "" || login.DeviceUUID == "" {
		return StartedLogin{}, ErrDeviceRequired
	}
	code, err := s.logins.Start(login)
	if err != nil {
		return StartedLogin{}, err
	}
	return StartedLogin{Code: code, URL: s.appURL + "/device?code=" + url.QueryEscape(code)}, nil
}

func (s *Service) PendingLogin(code string) (Login, bool) {
	return s.logins.Find(code)
}

func (s *Service) ApproveLogin(code string, user User) bool {
	return s.logins.Approve(code, user)
}

func (s *Service) ClaimLogin(code, deviceID, deviceUUID string) (SignedIn, bool, error) {
	user, approved := s.logins.Claim(code, deviceID, deviceUUID)
	if !approved {
		return SignedIn{}, false, nil
	}
	token, err := s.tokens.Issue(user, deviceID, deviceUUID)
	if err != nil {
		return SignedIn{}, false, err
	}
	if err := s.devices.Assign(deviceID, user.Name); err != nil {
		return SignedIn{}, false, err
	}
	return SignedIn{Token: token, User: user}, true, nil
}

func (s *Service) Authenticate(raw string) (Token, bool, error) {
	return s.tokens.Find(raw)
}

func (s *Service) SignOut(raw string, token Token) error {
	if err := s.tokens.Revoke(raw); err != nil {
		return err
	}
	return s.devices.Assign(token.DeviceID, "")
}

func (s *Service) Heartbeat(deviceID, deviceUUID string) (bool, error) {
	return s.devices.Seen(deviceID, deviceUUID)
}

func (s *Service) Report(device Device) (bool, error) {
	if device.ID == "" || device.UUID == "" {
		return false, nil
	}
	return s.devices.Report(device)
}

func (s *Service) Devices() ([]Device, error) {
	return s.devices.List()
}

func (s *Service) Peers(user string, page, size int) ([]Device, int, error) {
	owned, err := s.devices.OwnedBy(user)
	if err != nil {
		return nil, 0, err
	}
	start := min((page-1)*size, len(owned))
	end := min(start+size, len(owned))
	return owned[start:end], len(owned), nil
}

func (s *Service) AddressBook(user string) (string, error) {
	data, err := s.store.AddressBook(user)
	if err != nil {
		return "", err
	}
	if data == "" {
		return emptyAddressBook, nil
	}
	return data, nil
}

func (s *Service) SaveAddressBook(user, data string) error {
	if !json.Valid([]byte(data)) {
		return ErrInvalidAddressBook
	}
	return s.store.SaveAddressBook(user, data)
}
