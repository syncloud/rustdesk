package installer

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/syncloud/golib/config"
	"github.com/syncloud/golib/linux"
	"github.com/syncloud/golib/platform"
	"go.uber.org/zap"
)

const App = "rustdesk"

type Installer struct {
	appDir             string
	dataDir            string
	commonDir          string
	configDir          string
	installFile        string
	newVersionFile     string
	currentVersionFile string
	platformClient     *platform.Client
	logger             *zap.Logger
}

func New(logger *zap.Logger) *Installer {
	appDir := path.Join("/snap", App, "current")
	dataDir := path.Join("/var/snap", App, "current")
	commonDir := path.Join("/var/snap", App, "common")

	return &Installer{
		appDir:             appDir,
		dataDir:            dataDir,
		commonDir:          commonDir,
		configDir:          path.Join(dataDir, "config"),
		installFile:        path.Join(dataDir, "installed"),
		newVersionFile:     path.Join(appDir, "version"),
		currentVersionFile: path.Join(dataDir, "version"),
		platformClient:     platform.New(),
		logger:             logger,
	}
}

func (i *Installer) Install() error {
	if err := linux.CreateUser(App); err != nil {
		return err
	}
	return i.UpdateConfigs()
}

func (i *Installer) Configure() error {
	if !i.IsInstalled() {
		if err := os.WriteFile(i.installFile, []byte("installed"), 0644); err != nil {
			return err
		}
	}
	if err := i.FixPermissions(); err != nil {
		return err
	}
	return i.UpdateVersion()
}

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.installFile)
	return err == nil
}

func (i *Installer) PreRefresh() error {
	return nil
}

func (i *Installer) PostRefresh() error {
	if err := i.UpdateConfigs(); err != nil {
		return err
	}
	return i.ClearVersion()
}

func (i *Installer) StorageChange() error {
	return i.FixPermissions()
}

func (i *Installer) AccessChange() error {
	return i.UpdateConfigs()
}

func (i *Installer) BackupPreStop() error {
	return i.PreRefresh()
}

func (i *Installer) RestorePreStart() error {
	return i.PostRefresh()
}

func (i *Installer) RestorePostStart() error {
	return i.Configure()
}

func (i *Installer) ClearVersion() error {
	return os.RemoveAll(i.currentVersionFile)
}

func (i *Installer) UpdateVersion() error {
	data, err := os.ReadFile(i.newVersionFile)
	if err != nil {
		return err
	}
	return os.WriteFile(i.currentVersionFile, data, 0644)
}

func (i *Installer) UpdateConfigs() error {
	err := linux.CreateMissingDirs(
		i.dataDir,
		i.commonDir,
		i.configDir,
		path.Join(i.dataDir, "nginx"),
		path.Join(i.dataDir, "server"),
	)
	if err != nil {
		return err
	}

	appUrl, err := i.platformClient.GetAppUrl(App)
	if err != nil {
		return err
	}
	appDomain, err := i.platformClient.GetAppDomainName(App)
	if err != nil {
		return err
	}
	authUrl, err := i.platformClient.GetAppUrl("auth")
	if err != nil {
		return err
	}

	secret, err := getOrCreateSecret(path.Join(i.dataDir, ".secret"))
	if err != nil {
		return err
	}

	redirectPath := "/auth/callback"
	oidcSecret, err := i.platformClient.RegisterOIDCClient(
		App,
		[]string{redirectPath},
		true,
		"client_secret_basic",
	)
	if err != nil {
		return fmt.Errorf("register oidc client: %w", err)
	}

	variables := Variables{
		App:       App,
		AppDir:    i.appDir,
		DataDir:   i.dataDir,
		CommonDir: i.commonDir,
		AppUrl:    appUrl,
		AppDomain: appDomain,
		AuthUrl:   authUrl,
		AuthSocket: strings.TrimSuffix(
			strings.TrimPrefix(i.platformClient.GetAuthLocalSocket(), "http://unix:"), ":"),
		Secret:           secret,
		OIDCClientID:     App,
		OIDCClientSecret: oidcSecret,
		OIDCRedirectURI:  strings.TrimRight(appUrl, "/") + redirectPath,
	}

	if err := config.Generate(path.Join(i.appDir, "config"), i.configDir, variables); err != nil {
		return err
	}
	if err := os.Chmod(path.Join(i.configDir, "backend.env"), 0600); err != nil {
		return err
	}
	if err := os.Chmod(path.Join(i.dataDir, "server"), 0700); err != nil {
		return err
	}
	return i.FixPermissions()
}

func (i *Installer) FixPermissions() error {
	for _, dir := range []string{i.dataDir, i.commonDir} {
		if err := linux.Chown(dir, App); err != nil {
			return err
		}
	}
	return nil
}
