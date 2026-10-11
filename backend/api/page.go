package api

type Page[T any] struct {
	Total int `json:"total"`
	Data  []T `json:"data"`
}
