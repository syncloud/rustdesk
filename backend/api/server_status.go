package api

import "backend/server"

type ServerStatus interface {
	Status() (server.Status, error)
}
