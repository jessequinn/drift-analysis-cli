package gke

import (
	"context"
	"fmt"

	"time"

	container "google.golang.org/api/container/v1"
)

// ClusterInstance represents a GKE cluster with its configuration
type ClusterInstance struct {
	Project   string
	Name      string
	Location  string
	Status    string
	Config    *ClusterConfig
	NodePools []*NodePoolConfig
	Labels    map[string]string
}

// ClusterConfig holds the cluster-level configuration
type ClusterConfig struct {
	MasterVersion  string `yaml:"master_version" json:"master_version"`
	ReleaseChannel string `yaml:"release_channel" json:"release_channel"`

	// Networking
	Network              string              `yaml:"network,omitempty" json:"network,omitempty"`
	Subnetwork           string              `yaml:"subnetwork,omitempty" json:"subnetwork,omitempty"`
	PrivateCluster       bool                `yaml:"private_cluster" json:"private_cluster"`
	MasterGlobalAccess   bool                `yaml:"master_global_access,omitempty" json:"master_global_access,omitempty"`
	MasterAuthorizedNets []string            `yaml:"master_authorized_networks,omitempty" json:"master_authorized_networks,omitempty"`
	DatapathProvider     string              `yaml:"datapath_provider,omitempty" json:"datapath_provider,omitempty"`
	IPAllocationPolicy   *IPAllocationPolicy `yaml:"ip_allocation_policy,omitempty" json:"ip_allocation_policy,omitempty"`

	// Security
	WorkloadIdentity    bool   `yaml:"workload_identity" json:"workload_identity"`
	NetworkPolicy       bool   `yaml:"network_policy" json:"network_policy"`
	BinaryAuthorization bool   `yaml:"binary_authorization" json:"binary_authorization"`
	ShieldedNodes       bool   `yaml:"shielded_nodes" json:"shielded_nodes"`
	DatabaseEncryption  bool   `yaml:"database_encryption,omitempty" json:"database_encryption,omitempty"`
	SecurityPosture     string `yaml:"security_posture,omitempty" json:"security_posture,omitempty"`

	// Features
	MaintenanceWindow *MaintenanceWindow `yaml:"maintenance_window,omitempty" json:"maintenance_window,omitempty"`
	Addons            *AddonsConfig      `yaml:"addons,omitempty" json:"addons,omitempty"`
	LoggingConfig     *LoggingConfig     `yaml:"logging_config,omitempty" json:"logging_config,omitempty"`
	MonitoringConfig  *MonitoringConfig  `yaml:"monitoring_config,omitempty" json:"monitoring_config,omitempty"`
}

// IPAllocationPolicy holds IP allocation configuration
type IPAllocationPolicy struct {
	UseIPAliases     bool   `yaml:"use_ip_aliases" json:"use_ip_aliases"`
	ClusterIPv4CIDR  string `yaml:"cluster_ipv4_cidr,omitempty" json:"cluster_ipv4_cidr,omitempty"`
	ServicesIPv4CIDR string `yaml:"services_ipv4_cidr,omitempty" json:"services_ipv4_cidr,omitempty"`
	StackType        string `yaml:"stack_type,omitempty" json:"stack_type,omitempty"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	EnableSystemLogs   bool `yaml:"enable_system_logs" json:"enable_system_logs"`
	EnableWorkloadLogs bool `yaml:"enable_workload_logs" json:"enable_workload_logs"`
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	EnableSystemMetrics     bool `yaml:"enable_system_metrics" json:"enable_system_metrics"`
	EnableAPIServerMetrics  bool `yaml:"enable_apiserver_metrics" json:"enable_apiserver_metrics"`
	EnableControllerMetrics bool `yaml:"enable_controller_metrics" json:"enable_controller_metrics"`
	EnableSchedulerMetrics  bool `yaml:"enable_scheduler_metrics" json:"enable_scheduler_metrics"`
}

// NodePoolConfig holds node pool configuration
type NodePoolConfig struct {
	Name             string             `yaml:"name" json:"name"`
	Version          string             `yaml:"version" json:"version"`
	MachineType      string             `yaml:"machine_type" json:"machine_type"`
	DiskSizeGB       int64              `yaml:"disk_size_gb" json:"disk_size_gb"`
	DiskType         string             `yaml:"disk_type,omitempty" json:"disk_type,omitempty"`
	ImageType        string             `yaml:"image_type" json:"image_type"`
	InitialNodeCount int64              `yaml:"initial_node_count" json:"initial_node_count"`
	Autoscaling      *AutoscalingConfig `yaml:"autoscaling,omitempty" json:"autoscaling,omitempty"`
	AutoUpgrade      bool               `yaml:"auto_upgrade" json:"auto_upgrade"`
	AutoRepair       bool               `yaml:"auto_repair" json:"auto_repair"`
	ServiceAccount   string             `yaml:"service_account,omitempty" json:"service_account,omitempty"`
	Labels           map[string]string  `yaml:"labels,omitempty" json:"labels,omitempty"`
	Taints           []string           `yaml:"taints,omitempty" json:"taints,omitempty"`
}

// AutoscalingConfig holds autoscaling settings
type AutoscalingConfig struct {
	Enabled      bool  `yaml:"enabled" json:"enabled"`
	MinNodeCount int64 `yaml:"min_node_count" json:"min_node_count"`
	MaxNodeCount int64 `yaml:"max_node_count" json:"max_node_count"`
}

// MaintenanceWindow defines cluster maintenance window
type MaintenanceWindow struct {
	StartTime string `yaml:"start_time" json:"start_time"`
	Duration  string `yaml:"duration" json:"duration"`
}

// AddonsConfig holds cluster addon configuration
type AddonsConfig struct {
	HTTPLoadBalancing        bool `yaml:"http_load_balancing" json:"http_load_balancing"`
	HorizontalPodAutoscaling bool `yaml:"horizontal_pod_autoscaling" json:"horizontal_pod_autoscaling"`
	NetworkPolicy            bool `yaml:"network_policy" json:"network_policy"`
}

// Analyzer performs drift analysis on GKE clusters
type Analyzer struct {
	service    *container.Service
	lastReport *DriftReport
	projects   []string
}

// NewAnalyzer creates a new GKE Analyzer instance
func NewAnalyzer(ctx context.Context) (*Analyzer, error) {
	service, err := container.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GKE client: %w", err)
	}

	return &Analyzer{service: service}, nil
}

// Close releases resources held by the Analyzer
func (a *Analyzer) Close() error {
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
	return a.lastReport.DriftedClusters
}

// DiscoverClusters finds all GKE clusters across the specified GCP projects
func (a *Analyzer) DiscoverClusters(ctx context.Context, projects []string) ([]*ClusterInstance, error) {
	var clusters []*ClusterInstance

	for _, project := range projects {
		projectClusters, err := a.discoverProjectClusters(ctx, project)
		if err != nil {
			return nil, fmt.Errorf("failed to discover clusters in project %s: %w", project, err)
		}
		clusters = append(clusters, projectClusters...)
	}

	return clusters, nil
}

// discoverProjectClusters lists all GKE clusters in a single GCP project
func (a *Analyzer) discoverProjectClusters(ctx context.Context, project string) ([]*ClusterInstance, error) {
	parent := fmt.Sprintf("projects/%s/locations/-", project)
	resp, err := a.service.Projects.Locations.Clusters.List(parent).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	var clusters []*ClusterInstance
	for _, cluster := range resp.Clusters {
		clusterInstance := &ClusterInstance{
			Project:   project,
			Name:      cluster.Name,
			Location:  cluster.Location,
			Status:    cluster.Status,
			Config:    extractClusterConfig(cluster),
			NodePools: extractNodePools(cluster),
			Labels:    cluster.ResourceLabels,
		}

		clusters = append(clusters, clusterInstance)
	}

	return clusters, nil
}

// extractClusterConfig extracts cluster-level configuration
func extractClusterConfig(cluster *container.Cluster) *ClusterConfig {
	config := &ClusterConfig{
		MasterVersion: cluster.CurrentMasterVersion,
		NetworkPolicy: cluster.NetworkPolicy != nil && cluster.NetworkPolicy.Enabled,
	}

	// Release channel
	if cluster.ReleaseChannel != nil {
		config.ReleaseChannel = cluster.ReleaseChannel.Channel
	}

	// Extract network configuration
	config.Network, config.Subnetwork, config.DatapathProvider = extractNetworkConfig(cluster)

	// Extract private cluster configuration
	config.PrivateCluster, config.MasterGlobalAccess = extractPrivateClusterConfig(cluster)

	// Extract master authorized networks
	config.MasterAuthorizedNets = extractMasterAuthorizedNets(cluster)

	// Extract IP allocation policy
	config.IPAllocationPolicy = extractIPAllocationPolicy(cluster)

	// Extract security features
	config.WorkloadIdentity, config.ShieldedNodes, config.DatabaseEncryption,
		config.BinaryAuthorization, config.SecurityPosture = extractSecurityFeatures(cluster)

	// Extract addons
	config.Addons = extractAddonsConfig(cluster)

	// Extract logging and monitoring
	config.LoggingConfig = extractLoggingConfig(cluster)
	config.MonitoringConfig = extractMonitoringConfig(cluster)

	// Extract maintenance window
	config.MaintenanceWindow = extractMaintenanceWindow(cluster)

	return config
}

// extractNodePools extracts node pool configurations from a cluster
func extractNodePools(cluster *container.Cluster) []*NodePoolConfig {
	nodePools := make([]*NodePoolConfig, 0)

	for _, np := range cluster.NodePools {
		pool := &NodePoolConfig{
			Name:             np.Name,
			Version:          np.Version,
			InitialNodeCount: np.InitialNodeCount,
		}

		// Node config
		if np.Config != nil {
			pool.MachineType = np.Config.MachineType
			pool.DiskSizeGB = np.Config.DiskSizeGb
			pool.DiskType = np.Config.DiskType
			pool.ImageType = np.Config.ImageType
			pool.ServiceAccount = np.Config.ServiceAccount
			pool.Labels = np.Config.Labels

			// Extract taints
			for _, taint := range np.Config.Taints {
				pool.Taints = append(pool.Taints, fmt.Sprintf("%s=%s:%s", taint.Key, taint.Value, taint.Effect))
			}
		}

		// Autoscaling
		if np.Autoscaling != nil && np.Autoscaling.Enabled {
			pool.Autoscaling = &AutoscalingConfig{
				Enabled:      true,
				MinNodeCount: np.Autoscaling.MinNodeCount,
				MaxNodeCount: np.Autoscaling.MaxNodeCount,
			}
		}

		// Management
		if np.Management != nil {
			pool.AutoUpgrade = np.Management.AutoUpgrade
			pool.AutoRepair = np.Management.AutoRepair
		}

		nodePools = append(nodePools, pool)
	}

	return nodePools
}

// AnalyzeDrift compares discovered clusters against a baseline and generates a drift report
func (a *Analyzer) AnalyzeDrift(clusters []*ClusterInstance, baseline *ClusterConfig, nodePoolBaseline *NodePoolConfig) *DriftReport {
	report := &DriftReport{
		Timestamp:     time.Now(),
		TotalClusters: len(clusters),
		Instances:     make([]*ClusterDrift, 0),
	}

	for _, cluster := range clusters {
		drift := a.analyzeCluster(cluster, baseline, nodePoolBaseline)
		report.Instances = append(report.Instances, drift)

		if len(drift.Drifts) > 0 {
			report.DriftedClusters++
		}
	}

	a.lastReport = report
	return report
}

// analyzeCluster compares a single cluster against the baseline configuration
func (a *Analyzer) analyzeCluster(cluster *ClusterInstance, baseline *ClusterConfig, nodePoolBaseline *NodePoolConfig) *ClusterDrift {
	drift := &ClusterDrift{
		Project:   cluster.Project,
		Name:      cluster.Name,
		Location:  cluster.Location,
		Status:    cluster.Status,
		Labels:    cluster.Labels,
		NodePools: cluster.NodePools,
		Drifts:    make([]Drift, 0),
	}

	if baseline == nil {
		return drift
	}

	// Compare cluster config
	a.compareClusterConfig(cluster.Config, baseline, drift)

	// Compare node pools
	if nodePoolBaseline != nil {
		a.compareNodePools(cluster.NodePools, nodePoolBaseline, drift)
	}

	return drift
}

// compareClusterConfig compares cluster configuration against baseline
func (a *Analyzer) compareClusterConfig(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	// Version and channel
	a.compareVersion(actual, baseline, drift)
	a.compareReleaseChannel(actual, baseline, drift)

	// Core cluster features
	a.compareCoreFeaturesCluster(actual, baseline, drift)

	// Networking
	a.compareNetworking(actual, baseline, drift)

	// IP Allocation Policy
	a.compareIPAllocation(actual, baseline, drift)

	// Security features
	a.compareSecurityCluster(actual, baseline, drift)

	// Logging and Monitoring
	a.compareLoggingCluster(actual, baseline, drift)
	a.compareMonitoringCluster(actual, baseline, drift)

	// Compare master authorized networks if specified in baseline
	if len(baseline.MasterAuthorizedNets) > 0 {
		a.compareMasterAuthorizedNetworks(baseline, actual, drift)
	}
}

// compareVersion compares master version
func (a *Analyzer) compareVersion(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.MasterVersion != "" {
		actualMinor := extractMinorVersion(actual.MasterVersion)
		baselineMinor := extractMinorVersion(baseline.MasterVersion)
		if actualMinor != baselineMinor {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.master_version",
				Expected: baseline.MasterVersion,
				Actual:   actual.MasterVersion,
				Severity: "high",
			})
		}
	}
}

// compareReleaseChannel compares release channel
func (a *Analyzer) compareReleaseChannel(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.ReleaseChannel != "" && actual.ReleaseChannel != baseline.ReleaseChannel {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.release_channel",
			Expected: baseline.ReleaseChannel,
			Actual:   actual.ReleaseChannel,
			Severity: "medium",
		})
	}
}

// compareCoreFeaturesCluster compares core cluster features
func (a *Analyzer) compareCoreFeaturesCluster(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if actual.PrivateCluster != baseline.PrivateCluster {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.private_cluster",
			Expected: fmt.Sprintf("%v", baseline.PrivateCluster),
			Actual:   fmt.Sprintf("%v", actual.PrivateCluster),
			Severity: "critical",
		})
	}

	if actual.WorkloadIdentity != baseline.WorkloadIdentity {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.workload_identity",
			Expected: fmt.Sprintf("%v", baseline.WorkloadIdentity),
			Actual:   fmt.Sprintf("%v", actual.WorkloadIdentity),
			Severity: "high",
		})
	}

	if actual.NetworkPolicy != baseline.NetworkPolicy {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.network_policy",
			Expected: fmt.Sprintf("%v", baseline.NetworkPolicy),
			Actual:   fmt.Sprintf("%v", actual.NetworkPolicy),
			Severity: "high",
		})
	}

	if actual.BinaryAuthorization != baseline.BinaryAuthorization {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.binary_authorization",
			Expected: fmt.Sprintf("%v", baseline.BinaryAuthorization),
			Actual:   fmt.Sprintf("%v", actual.BinaryAuthorization),
			Severity: "high",
		})
	}
}

// compareNetworking compares networking configuration
func (a *Analyzer) compareNetworking(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.DatapathProvider != "" && actual.DatapathProvider != baseline.DatapathProvider {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.datapath_provider",
			Expected: baseline.DatapathProvider,
			Actual:   actual.DatapathProvider,
			Severity: "medium",
		})
	}

	if actual.MasterGlobalAccess != baseline.MasterGlobalAccess {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.master_global_access",
			Expected: fmt.Sprintf("%v", baseline.MasterGlobalAccess),
			Actual:   fmt.Sprintf("%v", actual.MasterGlobalAccess),
			Severity: "medium",
		})
	}
}

// compareIPAllocation compares IP allocation policy
func (a *Analyzer) compareIPAllocation(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.IPAllocationPolicy != nil && actual.IPAllocationPolicy != nil {
		if baseline.IPAllocationPolicy.StackType != "" &&
			actual.IPAllocationPolicy.StackType != baseline.IPAllocationPolicy.StackType {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.ip_allocation_policy.stack_type",
				Expected: baseline.IPAllocationPolicy.StackType,
				Actual:   actual.IPAllocationPolicy.StackType,
				Severity: "high",
			})
		}
	}
}

// compareSecurityCluster compares security features
func (a *Analyzer) compareSecurityCluster(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if actual.ShieldedNodes != baseline.ShieldedNodes {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.shielded_nodes",
			Expected: fmt.Sprintf("%v", baseline.ShieldedNodes),
			Actual:   fmt.Sprintf("%v", actual.ShieldedNodes),
			Severity: "high",
		})
	}

	if actual.DatabaseEncryption != baseline.DatabaseEncryption {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.database_encryption",
			Expected: fmt.Sprintf("%v", baseline.DatabaseEncryption),
			Actual:   fmt.Sprintf("%v", actual.DatabaseEncryption),
			Severity: "critical",
		})
	}

	if baseline.SecurityPosture != "" && actual.SecurityPosture != baseline.SecurityPosture {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.security_posture",
			Expected: baseline.SecurityPosture,
			Actual:   actual.SecurityPosture,
			Severity: "high",
		})
	}
}

// compareLoggingCluster compares logging configuration
func (a *Analyzer) compareLoggingCluster(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.LoggingConfig != nil && actual.LoggingConfig != nil {
		if actual.LoggingConfig.EnableSystemLogs != baseline.LoggingConfig.EnableSystemLogs {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.logging_config.enable_system_logs",
				Expected: fmt.Sprintf("%v", baseline.LoggingConfig.EnableSystemLogs),
				Actual:   fmt.Sprintf("%v", actual.LoggingConfig.EnableSystemLogs),
				Severity: "medium",
			})
		}
		if actual.LoggingConfig.EnableWorkloadLogs != baseline.LoggingConfig.EnableWorkloadLogs {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.logging_config.enable_workload_logs",
				Expected: fmt.Sprintf("%v", baseline.LoggingConfig.EnableWorkloadLogs),
				Actual:   fmt.Sprintf("%v", actual.LoggingConfig.EnableWorkloadLogs),
				Severity: "low",
			})
		}
	}
}

// compareMonitoringCluster compares monitoring configuration
func (a *Analyzer) compareMonitoringCluster(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.MonitoringConfig != nil && actual.MonitoringConfig != nil {
		if actual.MonitoringConfig.EnableSystemMetrics != baseline.MonitoringConfig.EnableSystemMetrics {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.monitoring_config.enable_system_metrics",
				Expected: fmt.Sprintf("%v", baseline.MonitoringConfig.EnableSystemMetrics),
				Actual:   fmt.Sprintf("%v", actual.MonitoringConfig.EnableSystemMetrics),
				Severity: "medium",
			})
		}
		if actual.MonitoringConfig.EnableAPIServerMetrics != baseline.MonitoringConfig.EnableAPIServerMetrics {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.monitoring_config.enable_apiserver_metrics",
				Expected: fmt.Sprintf("%v", baseline.MonitoringConfig.EnableAPIServerMetrics),
				Actual:   fmt.Sprintf("%v", actual.MonitoringConfig.EnableAPIServerMetrics),
				Severity: "low",
			})
		}
	}
}

// compareMasterAuthorizedNetworks compares master authorized network lists between baseline and actual
func (a *Analyzer) compareMasterAuthorizedNetworks(baseline, actual *ClusterConfig, drift *ClusterDrift) {
	// Create sets for comparison
	baselineNets := make(map[string]bool)
	for _, net := range baseline.MasterAuthorizedNets {
		baselineNets[net] = true
	}

	actualNets := make(map[string]bool)
	for _, net := range actual.MasterAuthorizedNets {
		actualNets[net] = true
	}

	// Find required networks (in baseline but not in actual)
	requiredNets := make([]string, 0)
	for net := range baselineNets {
		if !actualNets[net] {
			requiredNets = append(requiredNets, net)
		}
	}

	// Find extra networks (in actual but not in baseline)
	extraNets := make([]string, 0)
	for net := range actualNets {
		if !baselineNets[net] {
			extraNets = append(extraNets, net)
		}
	}

	// Report required networks as high severity
	if len(requiredNets) > 0 {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.master_authorized_networks",
			Expected: fmt.Sprintf("Required: %v", requiredNets),
			Actual:   fmt.Sprintf("%v", actual.MasterAuthorizedNets),
			Severity: "high",
		})
	}

	// Report extra networks as medium severity
	if len(extraNets) > 0 {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.master_authorized_networks",
			Expected: fmt.Sprintf("%v", baseline.MasterAuthorizedNets),
			Actual:   fmt.Sprintf("Extra: %v", extraNets),
			Severity: "medium",
		})
	}
}

// compareNodePools compares node pools against baseline
func (a *Analyzer) compareNodePools(actualPools []*NodePoolConfig, baseline *NodePoolConfig, drift *ClusterDrift) {
	for _, pool := range actualPools {
		poolPrefix := fmt.Sprintf("nodepool[%s]", pool.Name)

		// Machine type
		if baseline.MachineType != "" && pool.MachineType != baseline.MachineType {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    fmt.Sprintf("%s.machine_type", poolPrefix),
				Expected: baseline.MachineType,
				Actual:   pool.MachineType,
				Severity: "high",
			})
		}

		// Disk size
		if baseline.DiskSizeGB > 0 && pool.DiskSizeGB != baseline.DiskSizeGB {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    fmt.Sprintf("%s.disk_size_gb", poolPrefix),
				Expected: fmt.Sprintf("%d", baseline.DiskSizeGB),
				Actual:   fmt.Sprintf("%d", pool.DiskSizeGB),
				Severity: "medium",
			})
		}

		// Image type
		if baseline.ImageType != "" && pool.ImageType != baseline.ImageType {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    fmt.Sprintf("%s.image_type", poolPrefix),
				Expected: baseline.ImageType,
				Actual:   pool.ImageType,
				Severity: "medium",
			})
		}

		// Auto upgrade
		if pool.AutoUpgrade != baseline.AutoUpgrade {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    fmt.Sprintf("%s.auto_upgrade", poolPrefix),
				Expected: fmt.Sprintf("%v", baseline.AutoUpgrade),
				Actual:   fmt.Sprintf("%v", pool.AutoUpgrade),
				Severity: "high",
			})
		}

		// Auto repair
		if pool.AutoRepair != baseline.AutoRepair {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    fmt.Sprintf("%s.auto_repair", poolPrefix),
				Expected: fmt.Sprintf("%v", baseline.AutoRepair),
				Actual:   fmt.Sprintf("%v", pool.AutoRepair),
				Severity: "high",
			})
		}
	}
}

// extractMinorVersion extracts minor version from full version string
func extractMinorVersion(version string) string {
	// Example: "1.33.5-gke.1308000" -> "1.33"
	if len(version) < 4 {
		return version
	}
	for i, c := range version[2:] {
		if c == '.' {
			return version[:2+i]
		}
	}
	return version
}
