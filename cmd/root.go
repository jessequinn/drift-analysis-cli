package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "drift-analysis-cli",
	Short: "A CLI tool for detecting drift in cloud infrastructure",
	Long: `Drift Analysis CLI is a comprehensive tool for detecting configuration drift
in cloud infrastructure resources. It supports multiple cloud providers and resource types,
comparing actual resource configurations against defined baselines.`,
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.yaml", "config file path")
}
