package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/jessequinn/drift-analysis-cli/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	logFile     string
	outputFormat string
	jsonLogs    bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "drift-analysis-cli",
	Short: "A CLI tool for detecting drift in cloud infrastructure",
	Long: `Drift Analysis CLI is a comprehensive tool for detecting configuration drift
in cloud infrastructure resources. It supports multiple cloud providers and resource types,
comparing actual resource configurations against defined baselines.`,
	Version: "1.0.0",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logger
		logWriter := io.Writer(os.Stderr)
		
		// If log file specified, write to both file and stderr
		if logFile != "" {
			file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			} else {
				logWriter = io.MultiWriter(os.Stderr, file)
				// Don't defer file.Close() here - let it live for the whole run
			}
		}
		
		logLevel := logger.LevelInfo
		logger.InitLogger(logWriter, jsonLogs, logLevel)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("Command execution failed", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.yaml", "config file path")
	rootCmd.PersistentFlags().StringVarP(&logFile, "log-file", "l", "", "write logs to file (in addition to stderr)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "output format (text|json|yaml|tui)")
	rootCmd.PersistentFlags().BoolVar(&jsonLogs, "json-logs", false, "output logs in JSON format")
}
