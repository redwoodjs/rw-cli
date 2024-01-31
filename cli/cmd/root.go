package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	telemetryFlag bool

	BuildVersion = "unknown"
	BuildCommit  = "unknown"
	BuildDate    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:     "rw",
	Short:   "The RedwoodJS CLI",
	Version: "-",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&telemetryFlag, "telemetry", true, "Send telemetry events, see: https://telemetry.redwoodjs.com")
}

func Execute() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("Version: %s\nCommit:\t %s\nDate:\t %s\n", BuildVersion, BuildCommit, BuildDate))
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
