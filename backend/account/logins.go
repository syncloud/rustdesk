package account

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

var ErrTooManyLogins = errors.New("too many sign-ins are waiting for approval")

type Logins struct {
	TTL   time.Duration
	Limit int

	mu      sync.Mutex
	pending map[string]Login
}

func (l *Logins) Start(login Login) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	code := base64.RawURLEncoding.EncodeToString(buf)

	l.mu.Lock()
	defer l.mu.Unlock()
	l.expire()
	if len(l.pending) >= l.Limit {
		return "", ErrTooManyLogins
	}
	if l.pending == nil {
		l.pending = map[string]Login{}
	}
	login.Approved = nil
	login.Expires = time.Now().Add(l.TTL)
	l.pending[code] = login
	return code, nil
}

func (l *Logins) Find(code string) (Login, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.expire()
	login, found := l.pending[code]
	return login, found
}

func (l *Logins) Approve(code string, user User) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.expire()
	login, found := l.pending[code]
	if !found {
		return false
	}
	login.Approved = &user
	l.pending[code] = login
	return true
}

func (l *Logins) Claim(code, deviceID, deviceUUID string) (User, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.expire()
	login, found := l.pending[code]
	if !found || login.Approved == nil {
		return User{}, false
	}
	if login.DeviceID != deviceID || login.DeviceUUID != deviceUUID {
		return User{}, false
	}
	delete(l.pending, code)
	return *login.Approved, true
}

func (l *Logins) expire() {
	now := time.Now()
	for code, login := range l.pending {
		if now.After(login.Expires) {
			delete(l.pending, code)
		}
	}
}
