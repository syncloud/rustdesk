package server

import (
	"encoding/base64"
	"encoding/json"
	"slices"
)

type ClientConfig struct {
	Host  string `json:"host"`
	Relay string `json:"relay"`
	Api   string `json:"api"`
	Key   string `json:"key"`
}

func (c ClientConfig) Encode() (string, error) {
	body, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	encoded := []byte(base64.URLEncoding.EncodeToString(body))
	slices.Reverse(encoded)
	return string(encoded), nil
}
