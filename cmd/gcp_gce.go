package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jessequinn/drift-analysis-cli/pkg/gcp/gce"
	"github.com/jessequinn/drift-analysis-cli/pkg/logger"
	"github.com/jessequinn/drift-analysis-cli/pkg/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// gceCmd represents the gce command
var gceCmd = &cobra.Command{
	Use:   "gce",
	Short: "Analyze GCE instances for configuration drift",
	Long: `Analyze Google Compute Engine instances against baseline configurations.
Compares instance settings, machine types, disks, networking, and security settings.`,
	RunE:         runGCEAnalysis,
	SilenceUsage: true, // Don't show usage on runtime errors
}

func init() {
	gcpCmd.AddCommand(gceCmd)
}

func runGCEAnalysis(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	logger.Info("Starting GCE drift analysis", map[string]interface{}{
		"config": cfgFile,
		"output": outputFormat,
	})

	// Read config file
	configData, err := os.ReadFile(cfgFile)
	if err != nil {
		logger.Error("Failed to read config file", err, map[string]interface{}{
			"file": cfgFile,
		})
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		Projects     []string          `yaml:"projects"`
		GCEBaselines []gce.GCEBaseline `yaml:"gce_baselines"`
	}

	if err := yaml.Unmarshal(configData, &config); err != nil {
		logger.Error("Failed to parse config", err)
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if len(config.GCEBaselines) == 0 {
		logger.Warn("No GCE baselines defined in config")
		return fmt.Errorf("no GCE baselines defined in config")
	}

	logger.Info("Configuration loaded", map[string]interface{}{
		"projects":  len(config.Projects),
		"baselines": len(config.GCEBaselines),
	})

	// Create analyzer
	analyzer, err := gce.NewAnalyzer(ctx)
	if err != nil {
		logger.Error("Failed to create GCE analyzer", err)
		return fmt.Errorf("failed to create GCE analyzer: %w", err)
	}
	defer analyzer.Close()

	// Run analysis for each baseline
	for _, baseline := range config.GCEBaselines {
		logger.Info("Analyzing baseline", map[string]interface{}{
			"baseline": baseline.Name,
		})

		fmt.Printf("Analyzing GCE instances: %s\n", baseline.Name)
		fmt.Println("================================================================================")

		// Discover instances
		instances, err := analyzer.DiscoverInstances(ctx, config.Projects)
		if err != nil {
			logger.Error("Failed to discover instances", err)
			return fmt.Errorf("failed to discover instances: %w", err)
		}

		logger.Debug("Instances discovered", map[string]interface{}{
			"count": len(instances),
		})

		// Filter by labels if specified
		if len(baseline.FilterLabels) > 0 {
			filtered := make([]*gce.InstanceConfig, 0)
			for _, instance := range instances {
				matches := true
				for key, value := range baseline.FilterLabels {
					if instance.Labels[key] != value {
						matches = false
						break
					}
				}
				if matches {
					filtered = append(filtered, instance)
				}
			}
			instances = filtered

			logger.Debug("Instances filtered", map[string]interface{}{
				"count":  len(instances),
				"labels": baseline.FilterLabels,
			})
		}

		if len(instances) == 0 {
			logger.Warn("No instances found matching baseline filters")
			fmt.Printf("No instances found matching baseline filters\n\n")
			continue
		}

		// Analyze drift
		report := analyzer.AnalyzeDrift(instances, baseline.VMConfig)

		logger.Info("Drift analysis complete", map[string]interface{}{
			"total":    report.TotalInstances,
			"drifted":  report.DriftedInstances,
			"critical": report.CriticalCount,
			"high":     report.HighCount,
			"medium":   report.MediumCount,
			"low":      report.LowCount,
		})

		// Output report
		switch outputFormat {
		case "tui":
			// Convert to TUI format and run interactive display
			tuiData := tui.FromGCEReport(report)
			return tui.Run(tuiData)
		case "json":
			output, err := report.FormatJSON()
			if err != nil {
				logger.Error("Failed to format JSON", err)
				return fmt.Errorf("failed to format JSON: %w", err)
			}
			fmt.Println(output)
		case "yaml":
			output, err := report.FormatYAML()
			if err != nil {
				logger.Error("Failed to format YAML", err)
				return fmt.Errorf("failed to format YAML: %w", err)
			}
			fmt.Println(output)
		default:
			fmt.Println(report.FormatText())
		}

		fmt.Println()
	}

	logger.Info("GCE drift analysis completed successfully")
	return nil
}
