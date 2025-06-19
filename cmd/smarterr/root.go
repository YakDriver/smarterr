package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "smarterr",
	Short: "smarterr is a tool for diagnosing and validating smarterr configuration.",
	Long: `smarterr is a CLI for exploring, validating, and generating smarterr configuration
used by embedded Go error reporting systems.`,
}

var debugFlag bool

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable smarterr debug output (even if config fails to load)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
