package gce

import (
	"context"
	"fmt"
	"time"

	"github.com/jessequinn/drift-analysis-cli/pkg/analyzer"
	"google.golang.org/api/compute/v1"
)

// InstanceConfig represents a GCE instance with its configuration
type InstanceConfig struct {
	Project  string
	Name     string
	Zone     string
	Status   string
	Config   *VMConfig
	Labels   map[string]string
	Metadata map[string]string
}

// VMConfig holds the VM configuration parameters
type VMConfig struct {
	MachineType        string                `yaml:"machine_type" json:"machine_type"`
	DiskSizeGB         int64                 `yaml:"disk_size_gb" json:"disk_size_gb"`
	DiskType           string                `yaml:"disk_type" json:"disk_type"`
	ImageFamily        string                `yaml:"image_family,omitempty" json:"image_family,omitempty"`
	ImageProject       string                `yaml:"image_project,omitempty" json:"image_project,omitempty"`
	NetworkConfig      *NetworkConfig        `yaml:"network_config,omitempty" json:"network_config,omitempty"`
	ServiceAccount     *ServiceAccountConfig `yaml:"service_account,omitempty" json:"service_account,omitempty"`
	Metadata           map[string]string     `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Tags               []string              `yaml:"tags,omitempty" json:"tags,omitempty"`
	Labels             map[string]string     `yaml:"labels,omitempty" json:"labels,omitempty"`
	Preemptible        bool                  `yaml:"preemptible" json:"preemptible"`
	AutomaticRestart   bool                  `yaml:"automatic_restart" json:"automatic_restart"`
	OnHostMaintenance  string                `yaml:"on_host_maintenance" json:"on_host_maintenance"`
	ShieldedVM         *ShieldedVMConfig     `yaml:"shielded_vm,omitempty" json:"shielded_vm,omitempty"`
	DeletionProtection bool                  `yaml:"deletion_protection" json:"deletion_protection"`
}

// NetworkConfig holds network interface configuration
type NetworkConfig struct {
	Network     string `yaml:"network" json:"network"`
	Subnetwork  string `yaml:"subnetwork" json:"subnetwork"`
	ExternalIP  bool   `yaml:"external_ip" json:"external_ip"`
	InternalIP  string `yaml:"internal_ip,omitempty" json:"internal_ip,omitempty"`
	NetworkTier string `yaml:"network_tier,omitempty" json:"network_tier,omitempty"`
}

// ServiceAccountConfig holds service account configuration
type ServiceAccountConfig struct {
	Email  string   `yaml:"email" json:"email"`
	Scopes []string `yaml:"scopes" json:"scopes"`
}

// ShieldedVMConfig holds Shielded VM security configuration
type ShieldedVMConfig struct {
	EnableSecureBoot          bool `yaml:"enable_secure_boot" json:"enable_secure_boot"`
	EnableVTPM                bool `yaml:"enable_vtpm" json:"enable_vtpm"`
	EnableIntegrityMonitoring bool `yaml:"enable_integrity_monitoring" json:"enable_integrity_monitoring"`
}

// GCEBaseline represents a GCE configuration baseline with optional filters
type GCEBaseline struct {
	Name         string            `yaml:"name,omitempty"`
	FilterLabels map[string]string `yaml:"filter_labels,omitempty"`
	VMConfig     *VMConfig         `yaml:"vm_config"`
}

// Analyzer performs drift analysis on GCE instances
type Analyzer struct {
	service    *compute.Service
	lastReport *DriftReport
	projects   []string
}

// NewAnalyzer creates a new GCE Analyzer instance
func NewAnalyzer(ctx context.Context) (*Analyzer, error) {
	service, err := compute.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Compute Engine client: %w", err)
	}

	return &Analyzer{service: service}, nil
}

// Close releases resources held by the Analyzer
func (a *Analyzer) Close() error {
	return nil
}

// Compile-time interface implementation checks
var _ analyzer.ResourceAnalyzer = (*Analyzer)(nil)
var _ analyzer.Baseline = (*GCEBaseline)(nil)

// GetName returns the baseline name
func (b GCEBaseline) GetName() string {
	return b.Name
}

// Validate checks if the baseline is valid
func (b GCEBaseline) Validate() error {
	if b.Name == "" {
		return fmt.Errorf("baseline name is required")
	}
	if b.VMConfig == nil {
		return fmt.Errorf("vm_config is required")
	}
	return nil
}

// Analyze performs drift analysis implementing analyzer.ResourceAnalyzer interface
func (a *Analyzer) Analyze(ctx context.Context, projects []string) error {
	a.projects = projects
	return nil
}

// GenerateReport generates a formatted report implementing analyzer.ResourceAnalyzer interface
func (a *Analyzer) GenerateReport() (string, error) {
	if a.lastReport == nil {
		return "", fmt.Errorf("no analysis has been performed yet")
	}
	return a.lastReport.FormatText(), nil
}

// GetDriftCount returns the number of drifts detected implementing analyzer.ResourceAnalyzer interface
func (a *Analyzer) GetDriftCount() int {
	if a.lastReport == nil {
		return 0
	}
	return a.lastReport.DriftedInstances
}

// DiscoverInstances finds all GCE instances across the specified GCP projects
func (a *Analyzer) DiscoverInstances(ctx context.Context, projects []string) ([]*InstanceConfig, error) {
	var instances []*InstanceConfig

	for _, project := range projects {
		projectInstances, err := a.discoverProjectInstances(ctx, project)
		if err != nil {
			return nil, fmt.Errorf("failed to discover instances in project %s: %w", project, err)
		}
		instances = append(instances, projectInstances...)
	}

	return instances, nil
}

// discoverProjectInstances lists all GCE instances in a single GCP project across all zones
func (a *Analyzer) discoverProjectInstances(ctx context.Context, project string) ([]*InstanceConfig, error) {
	var instances []*InstanceConfig

	req := a.service.Instances.AggregatedList(project)
	if err := req.Pages(ctx, func(page *compute.InstanceAggregatedList) error {
		for _, instancesScopedList := range page.Items {
			if instancesScopedList.Instances == nil {
				continue
			}

			for _, instance := range instancesScopedList.Instances {
				instanceConfig := &InstanceConfig{
					Project:  project,
					Name:     instance.Name,
					Zone:     extractZoneFromURL(instance.Zone),
					Status:   instance.Status,
					Config:   extractVMConfig(instance),
					Labels:   instance.Labels,
					Metadata: extractMetadata(instance),
				}

				instances = append(instances, instanceConfig)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return instances, nil
}

// AnalyzeDrift compares discovered instances against the baseline configuration
func (a *Analyzer) AnalyzeDrift(instances []*InstanceConfig, baseline *VMConfig) *DriftReport {
	report := &DriftReport{
		Timestamp:        time.Now(),
		TotalInstances:   len(instances),
		DriftedInstances: 0,
		Instances:        make([]InstanceDriftReport, 0),
	}

	for _, instance := range instances {
		instanceReport := a.analyzeInstanceDrift(instance, baseline)
		if len(instanceReport.Drifts) > 0 {
			report.DriftedInstances++
		}
		report.Instances = append(report.Instances, instanceReport)
	}

	// Calculate severity counts
	for _, instanceReport := range report.Instances {
		for _, drift := range instanceReport.Drifts {
			switch drift.Severity {
			case "CRITICAL":
				report.CriticalCount++
			case "HIGH":
				report.HighCount++
			case "MEDIUM":
				report.MediumCount++
			case "LOW":
				report.LowCount++
			}
		}
	}

	a.lastReport = report
	return report
}

// analyzeInstanceDrift compares a single instance against baseline
func (a *Analyzer) analyzeInstanceDrift(instance *InstanceConfig, baseline *VMConfig) InstanceDriftReport {
	report := InstanceDriftReport{
		Project:         instance.Project,
		Name:            instance.Name,
		Zone:            instance.Zone,
		Status:          instance.Status,
		Labels:          instance.Labels,
		Drifts:          make([]Drift, 0),
		Recommendations: make([]string, 0),
	}

	// Compare configurations
	drifts := compareVMConfigs(instance.Config, baseline)
	report.Drifts = append(report.Drifts, drifts...)

	// Generate recommendations
	report.Recommendations = generateRecommendations(drifts)

	return report
}
