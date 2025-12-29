package sql

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Command handles Cloud SQL drift analysis operations
type Command struct {
	Projects       string
	ProjectList    []string
	Baselines      []SQLBaseline
	OutputFile     string
	Format         string
	FilterRole     string
	GenerateConfig bool
}

// Config represents the YAML configuration file structure for SQL
type Config struct {
	Projects  []string      `yaml:"projects"`
	Baselines []SQLBaseline `yaml:"baselines,omitempty"`

	// Legacy single baseline support
	Baseline     *DatabaseConfig   `yaml:"baseline,omitempty"`
	FilterLabels map[string]string `yaml:"filter_labels,omitempty"`
}

// SQLBaseline represents a SQL configuration baseline with optional filters
type SQLBaseline struct {
	Name         string            `yaml:"name,omitempty"`
	FilterLabels map[string]string `yaml:"filter_labels,omitempty"`
	Config       *DatabaseConfig   `yaml:"config"`
}

// Execute runs the SQL drift analysis command
func (c *Command) Execute(ctx context.Context) error {
	// Use provided baselines and projects from main
	var projectList []string
	var baselines []SQLBaseline
	var filterLabels map[string]string

	if len(c.ProjectList) > 0 {
		projectList = c.ProjectList
		baselines = c.Baselines
	} else if c.Projects != "" {
		projectList = strings.Split(c.Projects, ",")
		for i := range projectList {
			projectList[i] = strings.TrimSpace(projectList[i])
		}
	} else {
		return fmt.Errorf("must provide either -projects or -config")
	}

	// Apply command-line filter if specified
	if c.FilterRole != "" {
		if filterLabels == nil {
			filterLabels = make(map[string]string)
		}
		filterLabels["database-role"] = c.FilterRole
	}

	if len(projectList) == 0 {
		return fmt.Errorf("no projects specified")
	}

	// Initialize analyzer
	analyzer, err := NewAnalyzer(ctx)
	if err != nil {
		return fmt.Errorf("failed to create analyzer: %w", err)
	}
	defer func() {
		if err := analyzer.Close(); err != nil {
			log.Printf("Warning: failed to close analyzer: %v", err)
		}
	}()

	// Discover all PostgreSQL instances
	instances, err := analyzer.DiscoverInstances(ctx, projectList)
	if err != nil {
		return fmt.Errorf("failed to discover instances: %w", err)
	}

	if len(instances) == 0 {
		fmt.Println("No PostgreSQL instances found in specified projects")
		return nil
	}

	// Generate baseline config if requested
	if c.GenerateConfig {
		return generateBaselineConfig(instances, c.OutputFile)
	}

	// Perform drift analysis with multiple baselines
	var report *DriftReport

	if len(baselines) > 0 {
		// Multi-baseline mode
		report = analyzeMultipleBaselines(analyzer, instances, baselines)
	} else {
		// Legacy single baseline or no baseline mode
		var singleBaseline *DatabaseConfig
		if len(filterLabels) > 0 {
			instances = filterInstancesByLabels(instances, filterLabels)
		}
		report = analyzer.AnalyzeDrift(instances, singleBaseline)
	}

	// Output report
	return outputReport(report, c.Format, c.OutputFile)
}

// generateBaselineConfig generates a baseline configuration from discovered instances
func generateBaselineConfig(instances []*DatabaseInstance, outputPath string) error {
	if len(instances) == 0 {
		return fmt.Errorf("no instances to generate config from")
	}

	// Use first instance as baseline
	baseline := instances[0].Config

	config := Config{
		Projects: []string{instances[0].Project},
		Baseline: baseline,
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	if outputPath != "" {
		return os.WriteFile(outputPath, data, 0644)
	}

	fmt.Println(string(data))
	fmt.Printf("\nGenerated baseline config with %d instances\n", len(instances))
	return nil
}

// outputReport formats and writes the drift report
func outputReport(report *DriftReport, format, outputPath string) error {
	var output string

	switch format {
	case "json":
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		output = string(data)
	case "yaml":
		data, err := yaml.Marshal(report)
		if err != nil {
			return err
		}
		output = string(data)
	case "text":
		output = report.FormatText()
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if outputPath != "" {
		return os.WriteFile(outputPath, []byte(output), 0644)
	}

	fmt.Println(output)
	return nil
}

// analyzeMultipleBaselines analyzes instances against multiple baselines with different filters
func analyzeMultipleBaselines(analyzer *Analyzer, allInstances []*DatabaseInstance, baselines []SQLBaseline) *DriftReport {
	combinedReport := &DriftReport{
		Timestamp:      analyzer.GetTimestamp(),
		TotalInstances: len(allInstances),
		Instances:      make([]*InstanceDrift, 0),
	}

	// Track which instances have been analyzed
	analyzedInstances := make(map[string]bool)

	// Analyze each baseline with its filters
	for _, baseline := range baselines {
		// Filter instances for this baseline
		filteredInstances := allInstances
		if len(baseline.FilterLabels) > 0 {
			filteredInstances = filterInstancesByLabels(allInstances, baseline.FilterLabels)
		}

		// Analyze with this baseline
		for _, inst := range filteredInstances {
			instanceKey := fmt.Sprintf("%s/%s", inst.Project, inst.Name)
			if analyzedInstances[instanceKey] {
				continue // Skip already analyzed instances
			}

			drift := analyzer.AnalyzeInstance(inst, baseline.Config)
			combinedReport.Instances = append(combinedReport.Instances, drift)

			if len(drift.Drifts) > 0 {
				combinedReport.DriftedInstances++
			}

			analyzedInstances[instanceKey] = true
		}
	}

	return combinedReport
}

// filterInstancesByLabels filters instances that match all specified labels
func filterInstancesByLabels(instances []*DatabaseInstance, labels map[string]string) []*DatabaseInstance {
	if len(labels) == 0 {
		return instances
	}

	filtered := make([]*DatabaseInstance, 0)
	for _, inst := range instances {
		if matchesLabels(inst, labels) {
			filtered = append(filtered, inst)
		}
	}
	return filtered
}

// matchesLabels checks if an instance has all the specified labels
func matchesLabels(inst *DatabaseInstance, labels map[string]string) bool {
	if inst.Labels == nil {
		return false
	}

	for key, value := range labels {
		instValue, exists := inst.Labels[key]
		if !exists || instValue != value {
			return false
		}
	}
	return true
}
