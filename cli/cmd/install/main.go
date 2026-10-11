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
		Use:          "install",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return installer.New(log.Logger(zap.DebugLevel)).Install()
		},
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
