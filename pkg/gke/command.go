package gke

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Command handles GKE drift analysis operations
type Command struct {
	Projects       string
	ProjectList    []string
	Baselines      []GKEBaseline
	OutputFile     string
	Format         string
	FilterRole     string
	GenerateConfig bool
}

// Config represents the YAML configuration file structure for GKE
type Config struct {
	Projects  []string           `yaml:"projects"`
	Baselines []GKEBaseline      `yaml:"baselines,omitempty"`
	
	// Legacy single baseline support
	ClusterBaseline  *ClusterConfig  `yaml:"cluster_baseline,omitempty"`
	NodePoolBaseline *NodePoolConfig `yaml:"nodepool_baseline,omitempty"`
	FilterLabels     map[string]string `yaml:"filter_labels,omitempty"`
}

// GKEBaseline represents a GKE configuration baseline with optional filters
type GKEBaseline struct {
	Name             string            `yaml:"name,omitempty"`
	FilterLabels     map[string]string `yaml:"filter_labels,omitempty"`
	ClusterConfig    *ClusterConfig    `yaml:"cluster_config"`
	NodePoolConfig   *NodePoolConfig   `yaml:"nodepool_config,omitempty"`
}

// Execute runs the GKE drift analysis command
func (c *Command) Execute(ctx context.Context) error {
	// Use provided baselines and projects from main
	var projectList []string
	var baselines []GKEBaseline
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
		filterLabels["cluster-role"] = c.FilterRole
	}

	if len(projectList) == 0 {
		return fmt.Errorf("no projects specified")
	}

	// Initialize analyzer
	analyzer, err := NewAnalyzer(ctx)
	if err != nil {
		return fmt.Errorf("failed to create analyzer: %w", err)
	}
	defer analyzer.Close()

	// Discover all GKE clusters
	clusters, err := analyzer.DiscoverClusters(ctx, projectList)
	if err != nil {
		return fmt.Errorf("failed to discover clusters: %w", err)
	}

	if len(clusters) == 0 {
		fmt.Println("No GKE clusters found in specified projects")
		return nil
	}

	// Generate baseline config if requested
	if c.GenerateConfig {
		return generateBaselineConfig(clusters, c.OutputFile)
	}

	// Perform drift analysis with multiple baselines
	var report *DriftReport
	
	if len(baselines) > 0 {
		// Multi-baseline mode
		report = analyzeMultipleBaselines(analyzer, clusters, baselines)
	} else {
		// Legacy single baseline or no baseline mode
		if len(filterLabels) > 0 {
			clusters = filterClustersByLabels(clusters, filterLabels)
		}
		report = analyzer.AnalyzeDrift(clusters, nil, nil)
	}

	// Output report
	return outputReport(report, c.Format, c.OutputFile)
}

// loadConfig loads configuration from a YAML file
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// generateBaselineConfig generates a baseline configuration from discovered clusters
func generateBaselineConfig(clusters []*ClusterInstance, outputPath string) error {
	if len(clusters) == 0 {
		return fmt.Errorf("no clusters to generate config from")
	}

	// Use first cluster as baseline
	cluster := clusters[0]
	
	var nodePoolBaseline *NodePoolConfig
	if len(cluster.NodePools) > 0 {
		nodePoolBaseline = cluster.NodePools[0]
	}

	config := Config{
		Projects:         []string{cluster.Project},
		ClusterBaseline:  cluster.Config,
		NodePoolBaseline: nodePoolBaseline,
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	if outputPath != "" {
		return os.WriteFile(outputPath, data, 0644)
	}

	fmt.Println(string(data))
	fmt.Printf("\nGenerated baseline config with %d clusters\n", len(clusters))
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

// analyzeMultipleBaselines analyzes clusters against multiple baselines with different filters
func analyzeMultipleBaselines(analyzer *Analyzer, allClusters []*ClusterInstance, baselines []GKEBaseline) *DriftReport {
	combinedReport := &DriftReport{
		Timestamp:     time.Now(),
		TotalClusters: len(allClusters),
		Instances:     make([]*ClusterDrift, 0),
	}

	// Track which clusters have been analyzed
	analyzedClusters := make(map[string]bool)

	// Analyze each baseline with its filters
	for _, baseline := range baselines {
		// Filter clusters for this baseline
		filteredClusters := allClusters
		if len(baseline.FilterLabels) > 0 {
			filteredClusters = filterClustersByLabels(allClusters, baseline.FilterLabels)
		}

		// Analyze with this baseline
		for _, cluster := range filteredClusters {
			clusterKey := fmt.Sprintf("%s/%s/%s", cluster.Project, cluster.Location, cluster.Name)
			if analyzedClusters[clusterKey] {
				continue // Skip already analyzed clusters
			}

			drift := analyzer.analyzeCluster(cluster, baseline.ClusterConfig, baseline.NodePoolConfig)
			combinedReport.Instances = append(combinedReport.Instances, drift)
			
			if len(drift.Drifts) > 0 {
				combinedReport.DriftedClusters++
			}
			
			analyzedClusters[clusterKey] = true
		}
	}

	return combinedReport
}

// filterClustersByLabels filters clusters that match all specified labels
func filterClustersByLabels(clusters []*ClusterInstance, labels map[string]string) []*ClusterInstance {
	if len(labels) == 0 {
		return clusters
	}

	filtered := make([]*ClusterInstance, 0)
	for _, cluster := range clusters {
		if matchesLabels(cluster, labels) {
			filtered = append(filtered, cluster)
		}
	}
	return filtered
}

// matchesLabels checks if a cluster has all the specified labels
func matchesLabels(cluster *ClusterInstance, labels map[string]string) bool {
	if cluster.Labels == nil {
		return false
	}

	for key, value := range labels {
		clusterValue, exists := cluster.Labels[key]
		if !exists || clusterValue != value {
			return false
		}
	}
	return true
}
