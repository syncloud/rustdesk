package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"backend/account"
	"backend/api"
	"backend/auth"
	"backend/config"
	"backend/server"
	"backend/store"
)

const configFile = "/var/snap/rustdesk/current/config/backend.env"

func main() {
	cmd := &cobra.Command{
		Use:          "backend",
		Short:        "RustDesk server connection details API",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(logger())
		},
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(log *zap.Logger) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load %s: %w", configFile, err)
	}

	oidc := &auth.OIDC{
		IssuerURL:    cfg.OIDCAuthBaseURL,
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURI,
		AppURL:       cfg.AppUrl,
		CookieSecret: []byte(cfg.SessionSecret),
		AuthSocket:   cfg.AuthSocket,
		Logger:       log,
	}
	db, err := store.Open(cfg.AccountDir)
	if err != nil {
		return fmt.Errorf("open account database: %w", err)
	}
	defer db.Close()
	accounts := account.NewService(
		cfg.AppUrl,
		db,
		&account.Tokens{Store: db},
		&account.Logins{TTL: 5 * time.Minute, Limit: 100},
		&account.Devices{Store: db, Limit: 500, SeenEvery: time.Minute},
	)
	status := &server.Server{
		Address:     cfg.AppDomain,
		KeyFile:     cfg.KeyFile,
		IdSocket:    cfg.IdSocket,
		RelaySocket: cfg.RelaySocket,
		DialTimeout: 2 * time.Second,
	}
	web := api.NewWeb(cfg.Socket, oidc, status, accounts, log)
	return web.Start()
}

func logger() *zap.Logger {
	c := zap.NewProductionConfig()
	c.Encoding = "console"
	c.EncoderConfig.TimeKey = ""
	c.OutputPaths = []string{"stdout"}
	c.ErrorOutputPaths = []string{"stderr"}
	log, err := c.Build()
	if err != nil {
		panic(err)
	}
	return log
}
