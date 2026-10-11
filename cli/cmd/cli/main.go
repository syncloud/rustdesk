package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"cli/installer"
	"cli/log"
)

func main() {
	cmd := &cobra.Command{
		Use:          "cli",
		SilenceUsage: true,
	}

	add := func(use string, run func(*installer.Installer) error) {
		cmd.AddCommand(&cobra.Command{
			Use: use,
			RunE: func(_ *cobra.Command, _ []string) error {
				logger := log.Logger(zap.DebugLevel)
				logger.Info(use)
				return run(installer.New(logger))
			},
		})
	}

	add("storage-change", (*installer.Installer).StorageChange)
	add("access-change", (*installer.Installer).AccessChange)
	add("backup-pre-stop", (*installer.Installer).BackupPreStop)
	add("restore-pre-start", (*installer.Installer).RestorePreStart)
	add("restore-post-start", (*installer.Installer).RestorePostStart)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
