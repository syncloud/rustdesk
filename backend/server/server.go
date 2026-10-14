package server

import (
	"errors"
	"net"
	"os"
	"strings"
	"time"
)

type Server struct {
	Address     string
	KeyFile     string
	IdSocket    string
	RelaySocket string
	DialTimeout time.Duration
}

func (s *Server) Status() (Status, error) {
	key, err := s.key()
	if err != nil {
		return Status{}, err
	}

	status := Status{
		Address:      s.Address,
		Key:          key,
		IdRunning:    s.listening(s.IdSocket),
		RelayRunning: s.listening(s.RelaySocket),
	}
	if key == "" {
		return status, nil
	}

	status.ClientConfig, err = ClientConfig{Host: s.Address, Api: "https://" + s.Address, Key: key}.Encode()
	return status, err
}

func (s *Server) key() (string, error) {
	content, err := os.ReadFile(s.KeyFile)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func (s *Server) listening(socket string) bool {
	conn, err := net.DialTimeout("unix", socket, s.DialTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
