package server

type Status struct {
	Address      string `json:"address"`
	Key          string `json:"key"`
	ClientConfig string `json:"client_config"`
	IdRunning    bool   `json:"id_running"`
	RelayRunning bool   `json:"relay_running"`
}
