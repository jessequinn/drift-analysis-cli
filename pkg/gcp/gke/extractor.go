package gke

import "google.golang.org/api/container/v1"

// extractNetworkConfig extracts network configuration from cluster
func extractNetworkConfig(cluster *container.Cluster) (network, subnetwork, datapathProvider string) {
	if cluster.Network != "" {
		network = cluster.Network
	}
	if cluster.Subnetwork != "" {
		subnetwork = cluster.Subnetwork
	}
	if cluster.NetworkConfig != nil {
		datapathProvider = cluster.NetworkConfig.DatapathProvider
	}
	return
}

// extractPrivateClusterConfig extracts private cluster configuration
func extractPrivateClusterConfig(cluster *container.Cluster) (privateCluster, masterGlobalAccess bool) {
	if cluster.PrivateClusterConfig != nil {
		privateCluster = cluster.PrivateClusterConfig.EnablePrivateNodes
		if cluster.PrivateClusterConfig.MasterGlobalAccessConfig != nil {
			masterGlobalAccess = cluster.PrivateClusterConfig.MasterGlobalAccessConfig.Enabled
		}
	}
	return
}

// extractIPAllocationPolicy extracts IP allocation policy from cluster
func extractIPAllocationPolicy(cluster *container.Cluster) *IPAllocationPolicy {
	if cluster.IpAllocationPolicy != nil {
		return &IPAllocationPolicy{
			UseIPAliases:     cluster.IpAllocationPolicy.UseIpAliases,
			ClusterIPv4CIDR:  cluster.IpAllocationPolicy.ClusterIpv4CidrBlock,
			ServicesIPv4CIDR: cluster.IpAllocationPolicy.ServicesIpv4CidrBlock,
			StackType:        cluster.IpAllocationPolicy.StackType,
		}
	}
	return nil
}

// extractSecurityFeatures extracts security features from cluster
func extractSecurityFeatures(cluster *container.Cluster) (workloadIdentity, shieldedNodes, databaseEncryption, binaryAuth bool, securityPosture string) {
	if cluster.WorkloadIdentityConfig != nil {
		workloadIdentity = cluster.WorkloadIdentityConfig.WorkloadPool != ""
	}
	if cluster.ShieldedNodes != nil {
		shieldedNodes = cluster.ShieldedNodes.Enabled
	}
	if cluster.DatabaseEncryption != nil {
		databaseEncryption = cluster.DatabaseEncryption.State == "ENCRYPTED"
	}
	if cluster.SecurityPostureConfig != nil {
		securityPosture = cluster.SecurityPostureConfig.Mode
	}
	if cluster.BinaryAuthorization != nil {
		binaryAuth = cluster.BinaryAuthorization.Enabled
	}
	return
}

// extractAddonsConfig extracts addons configuration from cluster
func extractAddonsConfig(cluster *container.Cluster) *AddonsConfig {
	if cluster.AddonsConfig != nil {
		return &AddonsConfig{
			HTTPLoadBalancing:        cluster.AddonsConfig.HttpLoadBalancing == nil || !cluster.AddonsConfig.HttpLoadBalancing.Disabled,
			HorizontalPodAutoscaling: cluster.AddonsConfig.HorizontalPodAutoscaling == nil || !cluster.AddonsConfig.HorizontalPodAutoscaling.Disabled,
			NetworkPolicy:            cluster.AddonsConfig.NetworkPolicyConfig != nil && !cluster.AddonsConfig.NetworkPolicyConfig.Disabled,
		}
	}
	return nil
}

// extractLoggingConfig extracts logging configuration from cluster
func extractLoggingConfig(cluster *container.Cluster) *LoggingConfig {
	if cluster.LoggingConfig != nil && cluster.LoggingConfig.ComponentConfig != nil {
		config := &LoggingConfig{}
		for _, component := range cluster.LoggingConfig.ComponentConfig.EnableComponents {
			if component == "SYSTEM_COMPONENTS" {
				config.EnableSystemLogs = true
			}
			if component == "WORKLOADS" {
				config.EnableWorkloadLogs = true
			}
		}
		return config
	}
	return nil
}

// extractMonitoringConfig extracts monitoring configuration from cluster
func extractMonitoringConfig(cluster *container.Cluster) *MonitoringConfig {
	if cluster.MonitoringConfig != nil && cluster.MonitoringConfig.ComponentConfig != nil {
		config := &MonitoringConfig{}
		for _, component := range cluster.MonitoringConfig.ComponentConfig.EnableComponents {
			switch component {
			case "SYSTEM_COMPONENTS":
				config.EnableSystemMetrics = true
			case "APISERVER":
				config.EnableAPIServerMetrics = true
			case "CONTROLLER_MANAGER":
				config.EnableControllerMetrics = true
			case "SCHEDULER":
				config.EnableSchedulerMetrics = true
			}
		}
		return config
	}
	return nil
}

// extractMaintenanceWindow extracts maintenance window from cluster
func extractMaintenanceWindow(cluster *container.Cluster) *MaintenanceWindow {
	if cluster.MaintenancePolicy != nil && cluster.MaintenancePolicy.Window != nil {
		if cluster.MaintenancePolicy.Window.DailyMaintenanceWindow != nil {
			return &MaintenanceWindow{
				StartTime: cluster.MaintenancePolicy.Window.DailyMaintenanceWindow.StartTime,
				Duration:  cluster.MaintenancePolicy.Window.DailyMaintenanceWindow.Duration,
			}
		}
	}
	return nil
}

// extractMasterAuthorizedNets extracts master authorized networks from cluster
func extractMasterAuthorizedNets(cluster *container.Cluster) []string {
	var nets []string
	if cluster.MasterAuthorizedNetworksConfig != nil && cluster.MasterAuthorizedNetworksConfig.Enabled {
		for _, cidr := range cluster.MasterAuthorizedNetworksConfig.CidrBlocks {
			nets = append(nets, cidr.CidrBlock)
		}
	}
	return nets
}
