package cmd

import (
	"github.com/spf13/cobra"
)

// gcpCmd represents the gcp command
var gcpCmd = &cobra.Command{
	Use:   "gcp",
	Short: "Analyze GCP resources for configuration drift",
	Long: `Analyze Google Cloud Platform resources for configuration drift.
Supports Cloud SQL, GKE clusters, and GCE instances.`,
}

func init() {
	rootCmd.AddCommand(gcpCmd)
}
