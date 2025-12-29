package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jessequinn/drift-analysis-cli/pkg/gcp/gke"
	"github.com/jessequinn/drift-analysis-cli/pkg/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var gkeOutputFormat string

// gkeCmd represents the gke command
var gkeCmd = &cobra.Command{
	Use:   "gke",
	Short: "Analyze GKE clusters for configuration drift",
	Long: `Analyze Google Kubernetes Engine clusters against baseline configurations.
Compares cluster settings, node pool configurations, networking, and security settings.`,
	RunE: runGKEAnalysis,
}

func init() {
	gcpCmd.AddCommand(gkeCmd)
	gkeCmd.Flags().StringVarP(&gkeOutputFormat, "output", "o", "text", "output format (text|json|yaml|tui)")
}

func runGKEAnalysis(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Read config file
	configData, err := os.ReadFile(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		Projects     []string          `yaml:"projects"`
		GKEBaselines []gke.GKEBaseline `yaml:"gke_baselines"`
	}

	if err := yaml.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if len(config.GKEBaselines) == 0 {
		return fmt.Errorf("no GKE baselines defined in config")
	}

	// Create analyzer
	analyzer, err := gke.NewAnalyzer(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GKE analyzer: %w", err)
	}
	defer analyzer.Close()

	// Run analysis for each baseline
	for _, baseline := range config.GKEBaselines {
		fmt.Printf("Analyzing GKE clusters: %s\n", baseline.Name)
		fmt.Println("================================================================================")

		// Discover clusters
		clusters, err := analyzer.DiscoverClusters(ctx, config.Projects)
		if err != nil {
			return fmt.Errorf("failed to discover clusters: %w", err)
		}

		// Filter by labels if specified
		if len(baseline.FilterLabels) > 0 {
			filtered := make([]*gke.ClusterInstance, 0)
			for _, cluster := range clusters {
				matches := true
				for key, value := range baseline.FilterLabels {
					if cluster.Labels[key] != value {
						matches = false
						break
					}
				}
				if matches {
					filtered = append(filtered, cluster)
				}
			}
			clusters = filtered
		}

		// Analyze drift
		report := analyzer.AnalyzeDrift(clusters, baseline.ClusterConfig, baseline.NodePoolConfig)

		// Output report
		switch gkeOutputFormat {
		case "tui":
			// Convert to TUI format and run interactive display
			tuiData := tui.FromGKEReport(report)
			return tui.Run(tuiData)
		case "json":
			output, err := report.FormatJSON()
			if err != nil {
				return fmt.Errorf("failed to format JSON: %w", err)
			}
			fmt.Println(output)
		case "yaml":
			output, err := report.FormatYAML()
			if err != nil {
				return fmt.Errorf("failed to format YAML: %w", err)
			}
			fmt.Println(output)
		default:
			fmt.Println(report.FormatText())
		}

		fmt.Println()
	}

	return nil
}
