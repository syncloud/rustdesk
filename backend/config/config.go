package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	AppDomain        string `env:"APP_DOMAIN"`
	AppUrl           string `env:"APP_URL"`
	OIDCClientID     string `env:"OIDC_CLIENT_ID"`
	OIDCClientSecret string `env:"OIDC_CLIENT_SECRET"`
	OIDCAuthBaseURL  string `env:"OIDC_AUTH_BASE_URL"`
	OIDCRedirectURI  string `env:"OIDC_REDIRECT_URI"`
	SessionSecret    string `env:"SESSION_SECRET"`
	Socket           string `env:"SOCKET"`
	KeyFile          string `env:"KEY_FILE"`
	IdSocket         string `env:"ID_SOCKET"`
	RelaySocket      string `env:"RELAY_SOCKET"`
	AuthSocket       string `env:"AUTH_LOCAL_SOCKET"`
	AccountDir       string `env:"ACCOUNT_DIR"`
}

func Load(path string) (*Config, error) {
	values, err := godotenv.Read(path)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	if err := env.ParseWithOptions(config, env.Options{Environment: values, RequiredIfNoDef: true}); err != nil {
		return nil, err
	}
	return config, nil
}
