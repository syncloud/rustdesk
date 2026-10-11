package account

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"
)

type Tokens struct {
	Store Store
}

func (t *Tokens) Issue(user User, deviceID, deviceUUID string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	raw := base64.RawURLEncoding.EncodeToString(buf)
	token := Token{User: user, DeviceID: deviceID, DeviceUUID: deviceUUID, Created: time.Now()}
	if err := t.Store.SaveToken(hashToken(raw), token); err != nil {
		return "", err
	}
	return raw, nil
}

func (t *Tokens) Find(raw string) (Token, bool, error) {
	return t.Store.Token(hashToken(raw))
}

func (t *Tokens) Revoke(raw string) error {
	return t.Store.DeleteToken(hashToken(raw))
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
