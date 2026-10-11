package installer

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
)

func getOrCreateSecret(file string) (string, error) {
	content, err := os.ReadFile(file)
	if err == nil {
		return strings.TrimSpace(string(content)), nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(buf)
	if err := os.WriteFile(file, []byte(secret), 0600); err != nil {
		return "", err
	}
	return secret, nil
}
