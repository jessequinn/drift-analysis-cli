package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jessequinn/drift-analysis-cli/pkg/gcp/sql"
	"github.com/jessequinn/drift-analysis-cli/pkg/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var sqlOutputFormat string

// sqlCmd represents the sql command
var sqlCmd = &cobra.Command{
	Use:   "sql",
	Short: "Analyze Cloud SQL instances for configuration drift",
	Long: `Analyze Google Cloud SQL instances against baseline configurations.
Compares database flags, settings, backups, and more.`,
	RunE: runSQLAnalysis,
}

func init() {
	gcpCmd.AddCommand(sqlCmd)
	sqlCmd.Flags().StringVarP(&sqlOutputFormat, "output", "o", "text", "output format (text|json|yaml|tui)")
}

func runSQLAnalysis(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Read config file
	configData, err := os.ReadFile(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		Projects     []string          `yaml:"projects"`
		SQLBaselines []sql.SQLBaseline `yaml:"sql_baselines"`
	}

	if err := yaml.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if len(config.SQLBaselines) == 0 {
		return fmt.Errorf("no SQL baselines defined in config")
	}

	// Create analyzer
	analyzer, err := sql.NewAnalyzer(ctx)
	if err != nil {
		return fmt.Errorf("failed to create SQL analyzer: %w", err)
	}
	defer analyzer.Close()

	// Run analysis for each baseline
	for _, baseline := range config.SQLBaselines {
		fmt.Printf("Analyzing SQL instances: %s\n", baseline.Name)
		fmt.Println("================================================================================")

		// Discover instances
		instances, err := analyzer.DiscoverInstances(ctx, config.Projects)
		if err != nil {
			return fmt.Errorf("failed to discover instances: %w", err)
		}

		// Filter by labels if specified
		if len(baseline.FilterLabels) > 0 {
			filtered := make([]*sql.DatabaseInstance, 0)
			for _, inst := range instances {
				matches := true
				for key, value := range baseline.FilterLabels {
					if inst.Labels[key] != value {
						matches = false
						break
					}
				}
				if matches {
					filtered = append(filtered, inst)
				}
			}
			instances = filtered
		}

		// Analyze drift
		report := analyzer.AnalyzeDrift(instances, baseline.Config)

		// Output report
		switch sqlOutputFormat {
		case "tui":
			// Convert to TUI format and run interactive display
			tuiData := tui.FromSQLReport(report)
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
