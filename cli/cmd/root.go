package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	telemetryFlag bool
)

var rootCmd = &cobra.Command{
	Use:   "rw",
	Short: "The RedwoodJS CLI",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&telemetryFlag, "telemetry", true, "Send telemetry events, see: https://telemetry.redwoodjs.com")
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
