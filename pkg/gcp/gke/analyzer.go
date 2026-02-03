package gke

import (
	"context"
	"fmt"

	"time"

	"github.com/jessequinn/drift-analysis-cli/pkg/analyzer"
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

	// Advanced Networking
	PrivateClusterConfig   *PrivateClusterConfig `yaml:"private_cluster_config,omitempty" json:"private_cluster_config,omitempty"`
	DefaultMaxPodsPerNode  *int64                `yaml:"default_max_pods_per_node,omitempty" json:"default_max_pods_per_node,omitempty"`
	DNSConfig              *DNSConfig            `yaml:"dns_config,omitempty" json:"dns_config,omitempty"`
	GatewayAPIConfig       *GatewayAPIConfig     `yaml:"gateway_api_config,omitempty" json:"gateway_api_config,omitempty"`

	// Cost & Resource Management
	Autopilot                 *bool                        `yaml:"autopilot,omitempty" json:"autopilot,omitempty"`
	VerticalPodAutoscaling    *bool                        `yaml:"vertical_pod_autoscaling,omitempty" json:"vertical_pod_autoscaling,omitempty"`
	ResourceUsageExportConfig *ResourceUsageExportConfig   `yaml:"resource_usage_export_config,omitempty" json:"resource_usage_export_config,omitempty"`

	// Advanced Security
	PodSecurityPolicy        *bool                      `yaml:"pod_security_policy,omitempty" json:"pod_security_policy,omitempty"`
	AuthenticatorGroupsConfig *AuthenticatorGroupsConfig `yaml:"authenticator_groups_config,omitempty" json:"authenticator_groups_config,omitempty"`

	// Observability
	NotificationConfig       *NotificationConfig `yaml:"notification_config,omitempty" json:"notification_config,omitempty"`
	ManagedPrometheus        *bool               `yaml:"managed_prometheus,omitempty" json:"managed_prometheus,omitempty"`

	// Cluster Lifecycle
	EnableKubernetesAlpha *bool `yaml:"enable_kubernetes_alpha,omitempty" json:"enable_kubernetes_alpha,omitempty"`
	EnableTPU             *bool `yaml:"enable_tpu,omitempty" json:"enable_tpu,omitempty"`
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

	// Cost Optimization
	Preemptible *bool `yaml:"preemptible,omitempty" json:"preemptible,omitempty"`
	Spot        *bool `yaml:"spot,omitempty" json:"spot,omitempty"`

	// Advanced Configuration
	ManagementConfig       *ManagementConfig       `yaml:"management,omitempty" json:"management,omitempty"`
	NetworkConfig          *NodePoolNetworkConfig  `yaml:"network_config,omitempty" json:"network_config,omitempty"`
	BootDiskKMSKey         string                  `yaml:"boot_disk_kms_key,omitempty" json:"boot_disk_kms_key,omitempty"`
	ShieldedInstanceConfig *ShieldedInstanceConfig `yaml:"shielded_instance_config,omitempty" json:"shielded_instance_config,omitempty"`
	LinuxNodeConfig        *LinuxNodeConfig        `yaml:"linux_node_config,omitempty" json:"linux_node_config,omitempty"`
	SandboxConfig          *SandboxConfig          `yaml:"sandbox_config,omitempty" json:"sandbox_config,omitempty"`
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

// PrivateClusterConfig holds private cluster configuration
type PrivateClusterConfig struct {
	EnablePrivateEndpoint bool   `yaml:"enable_private_endpoint,omitempty" json:"enable_private_endpoint,omitempty"`
	MasterIPv4CIDRBlock   string `yaml:"master_ipv4_cidr_block,omitempty" json:"master_ipv4_cidr_block,omitempty"`
}

// DNSConfig holds DNS configuration
type DNSConfig struct {
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"` // CLOUD_DNS or KUBE_DNS
}

// GatewayAPIConfig holds Gateway API configuration
type GatewayAPIConfig struct {
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

// ResourceUsageExportConfig holds resource usage export configuration
type ResourceUsageExportConfig struct {
	Enabled          bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	BigQueryDataset  string `yaml:"bigquery_dataset,omitempty" json:"bigquery_dataset,omitempty"`
}

// AuthenticatorGroupsConfig holds RBAC authenticator groups configuration
type AuthenticatorGroupsConfig struct {
	Enabled       bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	SecurityGroup string `yaml:"security_group,omitempty" json:"security_group,omitempty"`
}

// NotificationConfig holds cluster notification configuration
type NotificationConfig struct {
	Enabled   bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	PubSubTopic string `yaml:"pubsub_topic,omitempty" json:"pubsub_topic,omitempty"`
}

// ManagementConfig holds node pool management configuration
type ManagementConfig struct {
	AutoUpgradeSchedule string              `yaml:"auto_upgrade_schedule,omitempty" json:"auto_upgrade_schedule,omitempty"`
	UpgradeSettings     *UpgradeSettings    `yaml:"upgrade_settings,omitempty" json:"upgrade_settings,omitempty"`
}

// UpgradeSettings holds upgrade surge configuration
type UpgradeSettings struct {
	MaxSurge       *int64 `yaml:"max_surge,omitempty" json:"max_surge,omitempty"`
	MaxUnavailable *int64 `yaml:"max_unavailable,omitempty" json:"max_unavailable,omitempty"`
}

// NodePoolNetworkConfig holds node pool network configuration
type NodePoolNetworkConfig struct {
	PodRange string `yaml:"pod_range,omitempty" json:"pod_range,omitempty"`
}

// ShieldedInstanceConfig holds shielded VM configuration for node pool
type ShieldedInstanceConfig struct {
	EnableSecureBoot          *bool `yaml:"enable_secure_boot,omitempty" json:"enable_secure_boot,omitempty"`
	EnableIntegrityMonitoring *bool `yaml:"enable_integrity_monitoring,omitempty" json:"enable_integrity_monitoring,omitempty"`
}

// LinuxNodeConfig holds Linux-specific node configuration
type LinuxNodeConfig struct {
	Sysctls map[string]string `yaml:"sysctls,omitempty" json:"sysctls,omitempty"`
}

// SandboxConfig holds gVisor sandbox configuration
type SandboxConfig struct {
	Type string `yaml:"type,omitempty" json:"type,omitempty"` // "gvisor"
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

// Compile-time interface implementation check
var _ analyzer.ResourceAnalyzer = (*Analyzer)(nil)

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

	// Extract advanced networking
	config.PrivateClusterConfig = extractPrivateClusterConfigAdvanced(cluster)
	if cluster.DefaultMaxPodsConstraint != nil {
		config.DefaultMaxPodsPerNode = int64Ptr(cluster.DefaultMaxPodsConstraint.MaxPodsPerNode)
	}
	config.DNSConfig = extractDNSConfig(cluster)
	config.GatewayAPIConfig = extractGatewayAPIConfig(cluster)

	// Extract cost & resource management
	if cluster.Autopilot != nil {
		config.Autopilot = boolPtr(cluster.Autopilot.Enabled)
	}
	if cluster.VerticalPodAutoscaling != nil {
		config.VerticalPodAutoscaling = boolPtr(cluster.VerticalPodAutoscaling.Enabled)
	}
	config.ResourceUsageExportConfig = extractResourceUsageExportConfig(cluster)

	// Extract advanced security
	// Note: PodSecurityPolicyConfig may not be available in all GKE API versions
	// if cluster.PodSecurityPolicyConfig != nil {
	// 	config.PodSecurityPolicy = boolPtr(cluster.PodSecurityPolicyConfig.Enabled)
	// }
	config.AuthenticatorGroupsConfig = extractAuthenticatorGroupsConfig(cluster)

	// Extract observability
	config.NotificationConfig = extractNotificationConfig(cluster)
	if cluster.MonitoringConfig != nil && cluster.MonitoringConfig.ManagedPrometheusConfig != nil {
		config.ManagedPrometheus = boolPtr(cluster.MonitoringConfig.ManagedPrometheusConfig.Enabled)
	}

	// Extract cluster lifecycle settings
	if cluster.EnableKubernetesAlpha {
		config.EnableKubernetesAlpha = boolPtr(true)
	}
	if cluster.EnableTpu {
		config.EnableTPU = boolPtr(true)
	}

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

			// Cost optimization
			pool.Preemptible = boolPtr(np.Config.Preemptible)
			pool.Spot = boolPtr(np.Config.Spot)

			// Boot disk KMS key
			if np.Config.BootDiskKmsKey != "" {
				pool.BootDiskKMSKey = np.Config.BootDiskKmsKey
			}

			// Shielded instance config
			if np.Config.ShieldedInstanceConfig != nil {
				pool.ShieldedInstanceConfig = &ShieldedInstanceConfig{
					EnableSecureBoot:          boolPtr(np.Config.ShieldedInstanceConfig.EnableSecureBoot),
					EnableIntegrityMonitoring: boolPtr(np.Config.ShieldedInstanceConfig.EnableIntegrityMonitoring),
				}
			}

			// Linux node config
			if np.Config.LinuxNodeConfig != nil && len(np.Config.LinuxNodeConfig.Sysctls) > 0 {
				pool.LinuxNodeConfig = &LinuxNodeConfig{
					Sysctls: np.Config.LinuxNodeConfig.Sysctls,
				}
			}

			// Sandbox config (gVisor)
			if np.Config.SandboxConfig != nil {
				pool.SandboxConfig = &SandboxConfig{
					Type: np.Config.SandboxConfig.Type,
				}
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

			// Management config with upgrade settings
			if np.Management.UpgradeOptions != nil {
				pool.ManagementConfig = &ManagementConfig{}
				if np.UpgradeSettings != nil {
					pool.ManagementConfig.UpgradeSettings = &UpgradeSettings{
						MaxSurge:       int64Ptr(np.UpgradeSettings.MaxSurge),
						MaxUnavailable: int64Ptr(np.UpgradeSettings.MaxUnavailable),
					}
				}
			}
		}

		// Network config
		if np.NetworkConfig != nil && np.NetworkConfig.PodRange != "" {
			pool.NetworkConfig = &NodePoolNetworkConfig{
				PodRange: np.NetworkConfig.PodRange,
			}
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

	// Addons
	a.compareAddons(actual, baseline, drift)

	// Maintenance Window
	a.compareMaintenanceWindow(actual, baseline, drift)

	// Advanced Networking
	a.comparePrivateClusterConfigAdvanced(actual, baseline, drift)
	a.compareDefaultMaxPodsPerNode(actual, baseline, drift)
	a.compareDNSConfig(actual, baseline, drift)
	a.compareGatewayAPIConfig(actual, baseline, drift)

	// Cost & Resource Management
	a.compareAutopilot(actual, baseline, drift)
	a.compareVerticalPodAutoscaling(actual, baseline, drift)
	a.compareResourceUsageExportConfig(actual, baseline, drift)

	// Advanced Security
	a.comparePodSecurityPolicy(actual, baseline, drift)
	a.compareAuthenticatorGroupsConfig(actual, baseline, drift)

	// Observability
	a.compareNotificationConfig(actual, baseline, drift)
	a.compareManagedPrometheus(actual, baseline, drift)

	// Cluster Lifecycle
	a.compareKubernetesAlpha(actual, baseline, drift)
	a.compareTPU(actual, baseline, drift)

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
	if baseline.Network != "" && actual.Network != baseline.Network {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.network",
			Expected: baseline.Network,
			Actual:   actual.Network,
			Severity: "high",
		})
	}

	if baseline.Subnetwork != "" && actual.Subnetwork != baseline.Subnetwork {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "cluster.subnetwork",
			Expected: baseline.Subnetwork,
			Actual:   actual.Subnetwork,
			Severity: "high",
		})
	}

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
		if actual.IPAllocationPolicy.UseIPAliases != baseline.IPAllocationPolicy.UseIPAliases {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.ip_allocation_policy.use_ip_aliases",
				Expected: fmt.Sprintf("%v", baseline.IPAllocationPolicy.UseIPAliases),
				Actual:   fmt.Sprintf("%v", actual.IPAllocationPolicy.UseIPAliases),
				Severity: "critical",
			})
		}

		if baseline.IPAllocationPolicy.ClusterIPv4CIDR != "" &&
			actual.IPAllocationPolicy.ClusterIPv4CIDR != baseline.IPAllocationPolicy.ClusterIPv4CIDR {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.ip_allocation_policy.cluster_ipv4_cidr",
				Expected: baseline.IPAllocationPolicy.ClusterIPv4CIDR,
				Actual:   actual.IPAllocationPolicy.ClusterIPv4CIDR,
				Severity: "high",
			})
		}

		if baseline.IPAllocationPolicy.ServicesIPv4CIDR != "" &&
			actual.IPAllocationPolicy.ServicesIPv4CIDR != baseline.IPAllocationPolicy.ServicesIPv4CIDR {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.ip_allocation_policy.services_ipv4_cidr",
				Expected: baseline.IPAllocationPolicy.ServicesIPv4CIDR,
				Actual:   actual.IPAllocationPolicy.ServicesIPv4CIDR,
				Severity: "high",
			})
		}

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
		if actual.MonitoringConfig.EnableControllerMetrics != baseline.MonitoringConfig.EnableControllerMetrics {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.monitoring_config.enable_controller_metrics",
				Expected: fmt.Sprintf("%v", baseline.MonitoringConfig.EnableControllerMetrics),
				Actual:   fmt.Sprintf("%v", actual.MonitoringConfig.EnableControllerMetrics),
				Severity: "low",
			})
		}
		if actual.MonitoringConfig.EnableSchedulerMetrics != baseline.MonitoringConfig.EnableSchedulerMetrics {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.monitoring_config.enable_scheduler_metrics",
				Expected: fmt.Sprintf("%v", baseline.MonitoringConfig.EnableSchedulerMetrics),
				Actual:   fmt.Sprintf("%v", actual.MonitoringConfig.EnableSchedulerMetrics),
				Severity: "low",
			})
		}
	}
}

// compareAddons compares addon configuration
func (a *Analyzer) compareAddons(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.Addons != nil && actual.Addons != nil {
		if actual.Addons.HTTPLoadBalancing != baseline.Addons.HTTPLoadBalancing {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.addons.http_load_balancing",
				Expected: fmt.Sprintf("%v", baseline.Addons.HTTPLoadBalancing),
				Actual:   fmt.Sprintf("%v", actual.Addons.HTTPLoadBalancing),
				Severity: "low",
			})
		}
		if actual.Addons.HorizontalPodAutoscaling != baseline.Addons.HorizontalPodAutoscaling {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.addons.horizontal_pod_autoscaling",
				Expected: fmt.Sprintf("%v", baseline.Addons.HorizontalPodAutoscaling),
				Actual:   fmt.Sprintf("%v", actual.Addons.HorizontalPodAutoscaling),
				Severity: "low",
			})
		}
		if actual.Addons.NetworkPolicy != baseline.Addons.NetworkPolicy {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.addons.network_policy",
				Expected: fmt.Sprintf("%v", baseline.Addons.NetworkPolicy),
				Actual:   fmt.Sprintf("%v", actual.Addons.NetworkPolicy),
				Severity: "medium",
			})
		}
	}
}

// compareMaintenanceWindow compares maintenance window configuration
func (a *Analyzer) compareMaintenanceWindow(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.MaintenanceWindow != nil && actual.MaintenanceWindow != nil {
		if baseline.MaintenanceWindow.StartTime != "" &&
			actual.MaintenanceWindow.StartTime != baseline.MaintenanceWindow.StartTime {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.maintenance_window.start_time",
				Expected: baseline.MaintenanceWindow.StartTime,
				Actual:   actual.MaintenanceWindow.StartTime,
				Severity: "low",
			})
		}
		if baseline.MaintenanceWindow.Duration != "" &&
			actual.MaintenanceWindow.Duration != baseline.MaintenanceWindow.Duration {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.maintenance_window.duration",
				Expected: baseline.MaintenanceWindow.Duration,
				Actual:   actual.MaintenanceWindow.Duration,
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

		// Node pool version
		if baseline.Version != "" {
			actualMinor := extractMinorVersion(pool.Version)
			baselineMinor := extractMinorVersion(baseline.Version)
			if actualMinor != baselineMinor {
				drift.Drifts = append(drift.Drifts, Drift{
					Field:    fmt.Sprintf("%s.version", poolPrefix),
					Expected: baseline.Version,
					Actual:   pool.Version,
					Severity: "high",
				})
			}
		}

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

		// Disk type
		if baseline.DiskType != "" && pool.DiskType != baseline.DiskType {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    fmt.Sprintf("%s.disk_type", poolPrefix),
				Expected: baseline.DiskType,
				Actual:   pool.DiskType,
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

		// Service account
		if baseline.ServiceAccount != "" && pool.ServiceAccount != baseline.ServiceAccount {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    fmt.Sprintf("%s.service_account", poolPrefix),
				Expected: baseline.ServiceAccount,
				Actual:   pool.ServiceAccount,
				Severity: "high",
			})
		}

		// Initial node count
		if baseline.InitialNodeCount > 0 && pool.InitialNodeCount != baseline.InitialNodeCount {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    fmt.Sprintf("%s.initial_node_count", poolPrefix),
				Expected: fmt.Sprintf("%d", baseline.InitialNodeCount),
				Actual:   fmt.Sprintf("%d", pool.InitialNodeCount),
				Severity: "low",
			})
		}

		// Autoscaling
		if baseline.Autoscaling != nil && pool.Autoscaling != nil {
			if pool.Autoscaling.Enabled != baseline.Autoscaling.Enabled {
				drift.Drifts = append(drift.Drifts, Drift{
					Field:    fmt.Sprintf("%s.autoscaling.enabled", poolPrefix),
					Expected: fmt.Sprintf("%v", baseline.Autoscaling.Enabled),
					Actual:   fmt.Sprintf("%v", pool.Autoscaling.Enabled),
					Severity: "high",
				})
			}
			if baseline.Autoscaling.Enabled && pool.Autoscaling.Enabled {
				if pool.Autoscaling.MinNodeCount != baseline.Autoscaling.MinNodeCount {
					drift.Drifts = append(drift.Drifts, Drift{
						Field:    fmt.Sprintf("%s.autoscaling.min_node_count", poolPrefix),
						Expected: fmt.Sprintf("%d", baseline.Autoscaling.MinNodeCount),
						Actual:   fmt.Sprintf("%d", pool.Autoscaling.MinNodeCount),
						Severity: "medium",
					})
				}
				if pool.Autoscaling.MaxNodeCount != baseline.Autoscaling.MaxNodeCount {
					drift.Drifts = append(drift.Drifts, Drift{
						Field:    fmt.Sprintf("%s.autoscaling.max_node_count", poolPrefix),
						Expected: fmt.Sprintf("%d", baseline.Autoscaling.MaxNodeCount),
						Actual:   fmt.Sprintf("%d", pool.Autoscaling.MaxNodeCount),
						Severity: "medium",
					})
				}
			}
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

		// Preemptible (cost optimization)
		if baseline.Preemptible != nil {
			actualPreempt := false
			if pool.Preemptible != nil {
				actualPreempt = *pool.Preemptible
			}
			if actualPreempt != *baseline.Preemptible {
				drift.Drifts = append(drift.Drifts, Drift{
					Field:    fmt.Sprintf("%s.preemptible", poolPrefix),
					Expected: fmt.Sprintf("%v", *baseline.Preemptible),
					Actual:   fmt.Sprintf("%v", actualPreempt),
					Severity: "high",
				})
			}
		}

		// Spot instances (cost optimization)
		if baseline.Spot != nil {
			actualSpot := false
			if pool.Spot != nil {
				actualSpot = *pool.Spot
			}
			if actualSpot != *baseline.Spot {
				drift.Drifts = append(drift.Drifts, Drift{
					Field:    fmt.Sprintf("%s.spot", poolPrefix),
					Expected: fmt.Sprintf("%v", *baseline.Spot),
					Actual:   fmt.Sprintf("%v", actualSpot),
					Severity: "high",
				})
			}
		}

		// Boot disk KMS key
		if baseline.BootDiskKMSKey != "" && pool.BootDiskKMSKey != baseline.BootDiskKMSKey {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    fmt.Sprintf("%s.boot_disk_kms_key", poolPrefix),
				Expected: baseline.BootDiskKMSKey,
				Actual:   pool.BootDiskKMSKey,
				Severity: "medium",
			})
		}

		// Shielded instance config
		if baseline.ShieldedInstanceConfig != nil && pool.ShieldedInstanceConfig != nil {
			if baseline.ShieldedInstanceConfig.EnableSecureBoot != nil && pool.ShieldedInstanceConfig.EnableSecureBoot != nil {
				if *pool.ShieldedInstanceConfig.EnableSecureBoot != *baseline.ShieldedInstanceConfig.EnableSecureBoot {
					drift.Drifts = append(drift.Drifts, Drift{
						Field:    fmt.Sprintf("%s.shielded_instance_config.enable_secure_boot", poolPrefix),
						Expected: fmt.Sprintf("%v", *baseline.ShieldedInstanceConfig.EnableSecureBoot),
						Actual:   fmt.Sprintf("%v", *pool.ShieldedInstanceConfig.EnableSecureBoot),
						Severity: "medium",
					})
				}
			}
			if baseline.ShieldedInstanceConfig.EnableIntegrityMonitoring != nil && pool.ShieldedInstanceConfig.EnableIntegrityMonitoring != nil {
				if *pool.ShieldedInstanceConfig.EnableIntegrityMonitoring != *baseline.ShieldedInstanceConfig.EnableIntegrityMonitoring {
					drift.Drifts = append(drift.Drifts, Drift{
						Field:    fmt.Sprintf("%s.shielded_instance_config.enable_integrity_monitoring", poolPrefix),
						Expected: fmt.Sprintf("%v", *baseline.ShieldedInstanceConfig.EnableIntegrityMonitoring),
						Actual:   fmt.Sprintf("%v", *pool.ShieldedInstanceConfig.EnableIntegrityMonitoring),
						Severity: "medium",
					})
				}
			}
		}

		// Management config - upgrade settings
		if baseline.ManagementConfig != nil && baseline.ManagementConfig.UpgradeSettings != nil {
			if pool.ManagementConfig != nil && pool.ManagementConfig.UpgradeSettings != nil {
				if baseline.ManagementConfig.UpgradeSettings.MaxSurge != nil && pool.ManagementConfig.UpgradeSettings.MaxSurge != nil {
					if *pool.ManagementConfig.UpgradeSettings.MaxSurge != *baseline.ManagementConfig.UpgradeSettings.MaxSurge {
						drift.Drifts = append(drift.Drifts, Drift{
							Field:    fmt.Sprintf("%s.management.upgrade_settings.max_surge", poolPrefix),
							Expected: fmt.Sprintf("%d", *baseline.ManagementConfig.UpgradeSettings.MaxSurge),
							Actual:   fmt.Sprintf("%d", *pool.ManagementConfig.UpgradeSettings.MaxSurge),
							Severity: "medium",
						})
					}
				}
				if baseline.ManagementConfig.UpgradeSettings.MaxUnavailable != nil && pool.ManagementConfig.UpgradeSettings.MaxUnavailable != nil {
					if *pool.ManagementConfig.UpgradeSettings.MaxUnavailable != *baseline.ManagementConfig.UpgradeSettings.MaxUnavailable {
						drift.Drifts = append(drift.Drifts, Drift{
							Field:    fmt.Sprintf("%s.management.upgrade_settings.max_unavailable", poolPrefix),
							Expected: fmt.Sprintf("%d", *baseline.ManagementConfig.UpgradeSettings.MaxUnavailable),
							Actual:   fmt.Sprintf("%d", *pool.ManagementConfig.UpgradeSettings.MaxUnavailable),
							Severity: "medium",
						})
					}
				}
			}
		}

		// Network config - pod range
		if baseline.NetworkConfig != nil && baseline.NetworkConfig.PodRange != "" {
			actualPodRange := ""
			if pool.NetworkConfig != nil {
				actualPodRange = pool.NetworkConfig.PodRange
			}
			if actualPodRange != baseline.NetworkConfig.PodRange {
				drift.Drifts = append(drift.Drifts, Drift{
					Field:    fmt.Sprintf("%s.network_config.pod_range", poolPrefix),
					Expected: baseline.NetworkConfig.PodRange,
					Actual:   actualPodRange,
					Severity: "medium",
				})
			}
		}

		// Sandbox config (gVisor)
		if baseline.SandboxConfig != nil && baseline.SandboxConfig.Type != "" {
			actualSandbox := "none"
			if pool.SandboxConfig != nil {
				actualSandbox = pool.SandboxConfig.Type
			}
			if actualSandbox != baseline.SandboxConfig.Type {
				drift.Drifts = append(drift.Drifts, Drift{
					Field:    fmt.Sprintf("%s.sandbox_config.type", poolPrefix),
					Expected: baseline.SandboxConfig.Type,
					Actual:   actualSandbox,
					Severity: "low",
				})
			}
		}
	}
}

// comparePrivateClusterConfigAdvanced compares advanced private cluster configuration
func (a *Analyzer) comparePrivateClusterConfigAdvanced(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.PrivateClusterConfig != nil {
		if actual.PrivateClusterConfig == nil {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.private_cluster_config",
				Expected: "configured",
				Actual:   "not configured",
				Severity: "high",
			})
			return
		}

		if actual.PrivateClusterConfig.EnablePrivateEndpoint != baseline.PrivateClusterConfig.EnablePrivateEndpoint {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.private_cluster_config.enable_private_endpoint",
				Expected: fmt.Sprintf("%v", baseline.PrivateClusterConfig.EnablePrivateEndpoint),
				Actual:   fmt.Sprintf("%v", actual.PrivateClusterConfig.EnablePrivateEndpoint),
				Severity: "critical",
			})
		}

		if baseline.PrivateClusterConfig.MasterIPv4CIDRBlock != "" &&
			actual.PrivateClusterConfig.MasterIPv4CIDRBlock != baseline.PrivateClusterConfig.MasterIPv4CIDRBlock {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.private_cluster_config.master_ipv4_cidr_block",
				Expected: baseline.PrivateClusterConfig.MasterIPv4CIDRBlock,
				Actual:   actual.PrivateClusterConfig.MasterIPv4CIDRBlock,
				Severity: "high",
			})
		}
	}
}

// compareDefaultMaxPodsPerNode compares default max pods per node
func (a *Analyzer) compareDefaultMaxPodsPerNode(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.DefaultMaxPodsPerNode != nil {
		if actual.DefaultMaxPodsPerNode == nil {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.default_max_pods_per_node",
				Expected: fmt.Sprintf("%d", *baseline.DefaultMaxPodsPerNode),
				Actual:   "not set",
				Severity: "medium",
			})
		} else if *actual.DefaultMaxPodsPerNode != *baseline.DefaultMaxPodsPerNode {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.default_max_pods_per_node",
				Expected: fmt.Sprintf("%d", *baseline.DefaultMaxPodsPerNode),
				Actual:   fmt.Sprintf("%d", *actual.DefaultMaxPodsPerNode),
				Severity: "medium",
			})
		}
	}
}

// compareDNSConfig compares DNS configuration
func (a *Analyzer) compareDNSConfig(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.DNSConfig != nil && baseline.DNSConfig.Provider != "" {
		if actual.DNSConfig == nil || actual.DNSConfig.Provider != baseline.DNSConfig.Provider {
			actualProvider := "not set"
			if actual.DNSConfig != nil {
				actualProvider = actual.DNSConfig.Provider
			}
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.dns_config.provider",
				Expected: baseline.DNSConfig.Provider,
				Actual:   actualProvider,
				Severity: "medium",
			})
		}
	}
}

// compareGatewayAPIConfig compares Gateway API configuration
func (a *Analyzer) compareGatewayAPIConfig(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.GatewayAPIConfig != nil {
		actualEnabled := false
		if actual.GatewayAPIConfig != nil {
			actualEnabled = actual.GatewayAPIConfig.Enabled
		}
		if actualEnabled != baseline.GatewayAPIConfig.Enabled {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.gateway_api_config.enabled",
				Expected: fmt.Sprintf("%v", baseline.GatewayAPIConfig.Enabled),
				Actual:   fmt.Sprintf("%v", actualEnabled),
				Severity: "medium",
			})
		}
	}
}

// compareAutopilot compares Autopilot mode
func (a *Analyzer) compareAutopilot(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.Autopilot != nil {
		actualAutopilot := false
		if actual.Autopilot != nil {
			actualAutopilot = *actual.Autopilot
		}
		if actualAutopilot != *baseline.Autopilot {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.autopilot",
				Expected: fmt.Sprintf("%v", *baseline.Autopilot),
				Actual:   fmt.Sprintf("%v", actualAutopilot),
				Severity: "critical",
			})
		}
	}
}

// compareVerticalPodAutoscaling compares VPA configuration
func (a *Analyzer) compareVerticalPodAutoscaling(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.VerticalPodAutoscaling != nil {
		actualVPA := false
		if actual.VerticalPodAutoscaling != nil {
			actualVPA = *actual.VerticalPodAutoscaling
		}
		if actualVPA != *baseline.VerticalPodAutoscaling {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.vertical_pod_autoscaling",
				Expected: fmt.Sprintf("%v", *baseline.VerticalPodAutoscaling),
				Actual:   fmt.Sprintf("%v", actualVPA),
				Severity: "medium",
			})
		}
	}
}

// compareResourceUsageExportConfig compares resource usage export configuration
func (a *Analyzer) compareResourceUsageExportConfig(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.ResourceUsageExportConfig != nil {
		if actual.ResourceUsageExportConfig == nil {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.resource_usage_export_config",
				Expected: "configured",
				Actual:   "not configured",
				Severity: "low",
			})
			return
		}

		if actual.ResourceUsageExportConfig.Enabled != baseline.ResourceUsageExportConfig.Enabled {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.resource_usage_export_config.enabled",
				Expected: fmt.Sprintf("%v", baseline.ResourceUsageExportConfig.Enabled),
				Actual:   fmt.Sprintf("%v", actual.ResourceUsageExportConfig.Enabled),
				Severity: "low",
			})
		}

		if baseline.ResourceUsageExportConfig.BigQueryDataset != "" &&
			actual.ResourceUsageExportConfig.BigQueryDataset != baseline.ResourceUsageExportConfig.BigQueryDataset {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.resource_usage_export_config.bigquery_dataset",
				Expected: baseline.ResourceUsageExportConfig.BigQueryDataset,
				Actual:   actual.ResourceUsageExportConfig.BigQueryDataset,
				Severity: "low",
			})
		}
	}
}

// comparePodSecurityPolicy compares Pod Security Policy configuration
func (a *Analyzer) comparePodSecurityPolicy(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.PodSecurityPolicy != nil {
		actualPSP := false
		if actual.PodSecurityPolicy != nil {
			actualPSP = *actual.PodSecurityPolicy
		}
		if actualPSP != *baseline.PodSecurityPolicy {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.pod_security_policy",
				Expected: fmt.Sprintf("%v", *baseline.PodSecurityPolicy),
				Actual:   fmt.Sprintf("%v", actualPSP),
				Severity: "high",
			})
		}
	}
}

// compareAuthenticatorGroupsConfig compares authenticator groups configuration
func (a *Analyzer) compareAuthenticatorGroupsConfig(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.AuthenticatorGroupsConfig != nil {
		if actual.AuthenticatorGroupsConfig == nil {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.authenticator_groups_config",
				Expected: "configured",
				Actual:   "not configured",
				Severity: "medium",
			})
			return
		}

		if actual.AuthenticatorGroupsConfig.Enabled != baseline.AuthenticatorGroupsConfig.Enabled {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.authenticator_groups_config.enabled",
				Expected: fmt.Sprintf("%v", baseline.AuthenticatorGroupsConfig.Enabled),
				Actual:   fmt.Sprintf("%v", actual.AuthenticatorGroupsConfig.Enabled),
				Severity: "medium",
			})
		}
	}
}

// compareNotificationConfig compares notification configuration
func (a *Analyzer) compareNotificationConfig(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.NotificationConfig != nil {
		if actual.NotificationConfig == nil {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.notification_config",
				Expected: "configured",
				Actual:   "not configured",
				Severity: "low",
			})
			return
		}

		if actual.NotificationConfig.Enabled != baseline.NotificationConfig.Enabled {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.notification_config.enabled",
				Expected: fmt.Sprintf("%v", baseline.NotificationConfig.Enabled),
				Actual:   fmt.Sprintf("%v", actual.NotificationConfig.Enabled),
				Severity: "low",
			})
		}
	}
}

// compareManagedPrometheus compares managed Prometheus configuration
func (a *Analyzer) compareManagedPrometheus(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.ManagedPrometheus != nil {
		actualMP := false
		if actual.ManagedPrometheus != nil {
			actualMP = *actual.ManagedPrometheus
		}
		if actualMP != *baseline.ManagedPrometheus {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.managed_prometheus",
				Expected: fmt.Sprintf("%v", *baseline.ManagedPrometheus),
				Actual:   fmt.Sprintf("%v", actualMP),
				Severity: "low",
			})
		}
	}
}

// compareKubernetesAlpha compares Kubernetes alpha features flag
func (a *Analyzer) compareKubernetesAlpha(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.EnableKubernetesAlpha != nil {
		actualAlpha := false
		if actual.EnableKubernetesAlpha != nil {
			actualAlpha = *actual.EnableKubernetesAlpha
		}
		if actualAlpha != *baseline.EnableKubernetesAlpha {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.enable_kubernetes_alpha",
				Expected: fmt.Sprintf("%v", *baseline.EnableKubernetesAlpha),
				Actual:   fmt.Sprintf("%v", actualAlpha),
				Severity: "low",
			})
		}
	}
}

// compareTPU compares TPU enablement
func (a *Analyzer) compareTPU(actual, baseline *ClusterConfig, drift *ClusterDrift) {
	if baseline.EnableTPU != nil {
		actualTPU := false
		if actual.EnableTPU != nil {
			actualTPU = *actual.EnableTPU
		}
		if actualTPU != *baseline.EnableTPU {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "cluster.enable_tpu",
				Expected: fmt.Sprintf("%v", *baseline.EnableTPU),
				Actual:   fmt.Sprintf("%v", actualTPU),
				Severity: "low",
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
