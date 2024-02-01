package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/charmbracelet/lipgloss"
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
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// We will handle displaying errors ourselves and don't want usage information
		// to be printed when an error occurs
		// See: https://github.com/spf13/cobra/issues/340
		// This does not impact the help command or subcommands
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&telemetryFlag, "telemetry", true, "Send telemetry events, see: https://telemetry.redwoodjs.com")
}

func Execute() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("Version: %s\nCommit:\t %s\nDate:\t %s\n", BuildVersion, BuildCommit, BuildDate))
	err := rootCmd.Execute()
	if err != nil {
		slog.Error("command failed with an error", slog.String("error", err.Error()))

		// TODO(jgmw): improve error output styling
		width, _ := getTerminalSize()
		errStyle := lipgloss.NewStyle().
			Bold(true).
			Border(lipgloss.DoubleBorder(), true, false, true).
			BorderForeground(lipgloss.Color("#FF0000")).
			Foreground(lipgloss.Color("#FF0000")).
			Width(width)
		errMsg := fmt.Sprintf("An error occurred:\n%s", err.Error())
		fmt.Println()
		fmt.Println(errStyle.Render(errMsg))
		fmt.Println()

		os.Exit(1)
	}
}
