package gce

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jessequinn/drift-analysis-cli/pkg/analyzer"
	"gopkg.in/yaml.v3"
)

// Command handles GCE drift analysis operations
type Command struct {
	Projects       string
	ProjectList    []string
	Baselines      []GCEBaseline
	OutputFile     string
	Format         string
	FilterRole     string
	GenerateConfig bool
}

// Config represents the YAML configuration file structure for GCE
type Config struct {
	Projects  []string      `yaml:"projects"`
	Baselines []GCEBaseline `yaml:"baselines,omitempty"`

	// Legacy single baseline support
	VMBaseline   *VMConfig         `yaml:"vm_baseline,omitempty"`
	FilterLabels map[string]string `yaml:"filter_labels,omitempty"`
}

// Execute runs the GCE drift analysis command
func (c *Command) Execute(ctx context.Context) error {
	// Use provided baselines and projects from main
	var projectList []string
	var baselines []GCEBaseline
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

	if len(baselines) == 0 {
		return fmt.Errorf("no GCE baselines defined")
	}

	// Apply role filter if specified
	if c.FilterRole != "" {
		filterLabels = map[string]string{"instance-role": c.FilterRole}
	}

	// Create analyzer
	gceAnalyzer, err := NewAnalyzer(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GCE analyzer: %w", err)
	}
	defer func() {
		if err := gceAnalyzer.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close GCE analyzer: %v\n", err)
		}
	}()

	// Discover instances
	log.Printf("Discovering GCE instances across %d project(s)...", len(projectList))
	instances, err := gceAnalyzer.DiscoverInstances(ctx, projectList)
	if err != nil {
		return fmt.Errorf("failed to discover instances: %w", err)
	}

	if len(instances) == 0 {
		log.Println("No GCE instances found")
		return nil
	}

	log.Printf("Found %d instance(s)", len(instances))

	// Generate baseline config if requested
	if c.GenerateConfig {
		return c.generateBaselineConfig(instances)
	}

	// Run analysis for each baseline
	for _, baseline := range baselines {
		log.Printf("Analyzing instances against baseline: %s", baseline.Name)

		// Filter instances by baseline labels and name pattern
		filteredInstances := c.filterInstancesByPattern(instances, baseline, filterLabels)

		if len(filteredInstances) == 0 {
			log.Printf("No instances match baseline filter for: %s", baseline.Name)
			continue
		}

		// Analyze drift
		report := gceAnalyzer.AnalyzeDrift(filteredInstances, baseline.VMConfig)

		// Output report
		if err := c.outputReport(report, baseline.Name); err != nil {
			return err
		}
	}

	return nil
}

// filterInstances filters instances based on labels
func (c *Command) filterInstances(instances []*InstanceConfig, baselineLabels, additionalFilters map[string]string) []*InstanceConfig {
	var filtered []*InstanceConfig

	for _, instance := range instances {
		matches := true

		// Check baseline labels
		for key, value := range baselineLabels {
			if instance.Labels[key] != value {
				matches = false
				break
			}
		}

		// Check additional filters
		if matches {
			for key, value := range additionalFilters {
				if instance.Labels[key] != value {
					matches = false
					break
				}
			}
		}

		if matches {
			filtered = append(filtered, instance)
		}
	}

	return filtered
}

// filterInstancesByPattern filters instances by name pattern and labels
func (c *Command) filterInstancesByPattern(instances []*InstanceConfig, baseline GCEBaseline, additionalFilters map[string]string) []*InstanceConfig {
	var filtered []*InstanceConfig

	for _, instance := range instances {
		matches := true

		// Check baseline labels first
		for key, value := range baseline.FilterLabels {
			if instance.Labels[key] != value {
				matches = false
				break
			}
		}

		if !matches {
			continue
		}

		// Check exclude labels - if instance has any of these, skip it
		for key, value := range baseline.ExcludeLabels {
			if value == "" {
				// Empty value means "exclude if label exists (regardless of value)"
				if _, exists := instance.Labels[key]; exists {
					matches = false
					break
				}
			} else {
				// Non-empty value means "exclude if label matches this specific value"
				if instance.Labels[key] == value {
					matches = false
					break
				}
			}
		}

		if !matches {
			continue
		}

		// Check additional filters
		for key, value := range additionalFilters {
			if instance.Labels[key] != value {
				matches = false
				break
			}
		}

		if !matches {
			continue
		}

		// If exclude pattern is specified, skip instances matching it
		if baseline.ExcludeNamePattern != "" {
			if strings.Contains(instance.Name, baseline.ExcludeNamePattern) {
				continue
			}
		}

		// If name pattern is specified, ONLY include instances matching the pattern
		if baseline.NamePattern != "" {
			if !strings.Contains(instance.Name, baseline.NamePattern) {
				continue
			}
		}

		filtered = append(filtered, instance)
	}

	return filtered
}

// outputReport outputs the drift report in the specified format
func (c *Command) outputReport(report *DriftReport, baselineName string) error {
	var output string
	var err error

	switch c.Format {
	case "json":
		output, err = report.FormatJSON()
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
	case "yaml":
		output, err = report.FormatYAML()
		if err != nil {
			return fmt.Errorf("failed to format YAML: %w", err)
		}
	default:
		output = report.FormatText()
	}

	// Write to file or stdout
	if c.OutputFile != "" {
		if err := os.WriteFile(c.OutputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		log.Printf("Report written to: %s", c.OutputFile)
	} else {
		fmt.Println(output)
	}

	return nil
}

// generateBaselineConfig generates a baseline configuration from existing instances
func (c *Command) generateBaselineConfig(instances []*InstanceConfig) error {
	if len(instances) == 0 {
		return fmt.Errorf("no instances found to generate baseline")
	}

	// Group instances by labels to create multiple baselines
	baselineGroups := make(map[string][]*InstanceConfig)

	for _, instance := range instances {
		role := instance.Labels["instance-role"]
		if role == "" {
			role = "default"
		}
		baselineGroups[role] = append(baselineGroups[role], instance)
	}

	var generatedBaselines []GCEBaseline

	for role, roleInstances := range baselineGroups {
		// Use the first instance as the template
		template := roleInstances[0]

		baseline := GCEBaseline{
			Name:         role,
			FilterLabels: map[string]string{"instance-role": role},
			VMConfig:     template.Config,
		}

		generatedBaselines = append(generatedBaselines, baseline)
	}

	config := struct {
		Projects     []string      `yaml:"projects"`
		GCEBaselines []GCEBaseline `yaml:"gce_baselines"`
		Generated    time.Time     `yaml:"generated"`
		Comment      string        `yaml:"_comment"`
	}{
		Projects:     c.ProjectList,
		GCEBaselines: generatedBaselines,
		Generated:    time.Now(),
		Comment:      "Auto-generated baseline configuration. Review and customize as needed.",
	}

	var output string

	switch c.Format {
	case "json":
		data, marshalErr := json.MarshalIndent(config, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal JSON: %w", marshalErr)
		}
		output = string(data)
	default:
		data, marshalErr := yaml.Marshal(config)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal YAML: %w", marshalErr)
		}
		output = string(data)
	}

	// Write to file or stdout
	if c.OutputFile != "" {
		if writeErr := os.WriteFile(c.OutputFile, []byte(output), 0644); writeErr != nil {
			return fmt.Errorf("failed to write output file: %w", writeErr)
		}
		log.Printf("Baseline configuration written to: %s", c.OutputFile)
	} else {
		fmt.Println(output)
	}

	return nil
}

// Compile-time interface implementation check
var _ analyzer.ResourceAnalyzer = (*Analyzer)(nil)
