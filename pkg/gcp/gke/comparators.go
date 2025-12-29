package gke

// compareNetworkConfig compares network configuration
func compareNetworkConfig(baseline *GKEBaseline, actual *ClusterConfig, drifts *[]Drift) {
	if baseline.ClusterConfig.Network != "" && baseline.ClusterConfig.Network != actual.Network {
		*drifts = append(*drifts, Drift{
			Field:    "network",
			Expected: baseline.ClusterConfig.Network,
			Actual:   actual.Network,
		})
	}
	if baseline.ClusterConfig.Subnetwork != "" && baseline.ClusterConfig.Subnetwork != actual.Subnetwork {
		*drifts = append(*drifts, Drift{
			Field:    "subnetwork",
			Expected: baseline.ClusterConfig.Subnetwork,
			Actual:   actual.Subnetwork,
		})
	}
	if baseline.ClusterConfig.DatapathProvider != "" && baseline.ClusterConfig.DatapathProvider != actual.DatapathProvider {
		*drifts = append(*drifts, Drift{
			Field:    "datapath_provider",
			Expected: baseline.ClusterConfig.DatapathProvider,
			Actual:   actual.DatapathProvider,
		})
	}
}

// compareSecurityFeatures compares security features
func compareSecurityFeatures(baseline *GKEBaseline, actual *ClusterConfig, drifts *[]Drift) {
	if baseline.ClusterConfig.WorkloadIdentity && !actual.WorkloadIdentity {
		*drifts = append(*drifts, Drift{
			Field:    "workload_identity",
			Expected: "true",
			Actual:   "false",
		})
	}
	if baseline.ClusterConfig.ShieldedNodes && !actual.ShieldedNodes {
		*drifts = append(*drifts, Drift{
			Field:    "shielded_nodes",
			Expected: "true",
			Actual:   "false",
		})
	}
	if baseline.ClusterConfig.DatabaseEncryption && !actual.DatabaseEncryption {
		*drifts = append(*drifts, Drift{
			Field:    "database_encryption",
			Expected: "true",
			Actual:   "false",
		})
	}
	if baseline.ClusterConfig.SecurityPosture != "" && baseline.ClusterConfig.SecurityPosture != actual.SecurityPosture {
		*drifts = append(*drifts, Drift{
			Field:    "security_posture",
			Expected: baseline.ClusterConfig.SecurityPosture,
			Actual:   actual.SecurityPosture,
		})
	}
	if baseline.ClusterConfig.BinaryAuthorization && !actual.BinaryAuthorization {
		*drifts = append(*drifts, Drift{
			Field:    "binary_authorization",
			Expected: "true",
			Actual:   "false",
		})
	}
}

// compareAddons compares addon configuration
func compareAddons(baseline *GKEBaseline, actual *ClusterConfig, drifts *[]Drift) {
	if baseline.ClusterConfig.Addons == nil || actual.Addons == nil {
		return
	}
	if baseline.ClusterConfig.Addons.HTTPLoadBalancing && !actual.Addons.HTTPLoadBalancing {
		*drifts = append(*drifts, Drift{
			Field:    "addons.http_load_balancing",
			Expected: "true",
			Actual:   "false",
		})
	}
	if baseline.ClusterConfig.Addons.HorizontalPodAutoscaling && !actual.Addons.HorizontalPodAutoscaling {
		*drifts = append(*drifts, Drift{
			Field:    "addons.horizontal_pod_autoscaling",
			Expected: "true",
			Actual:   "false",
		})
	}
	if baseline.ClusterConfig.Addons.NetworkPolicy && !actual.Addons.NetworkPolicy {
		*drifts = append(*drifts, Drift{
			Field:    "addons.network_policy",
			Expected: "true",
			Actual:   "false",
		})
	}
}

// compareLogging compares logging configuration
func compareLogging(baseline *GKEBaseline, actual *ClusterConfig, drifts *[]Drift) {
	if baseline.ClusterConfig.LoggingConfig == nil || actual.LoggingConfig == nil {
		return
	}
	if baseline.ClusterConfig.LoggingConfig.EnableSystemLogs && !actual.LoggingConfig.EnableSystemLogs {
		*drifts = append(*drifts, Drift{
			Field:    "logging.system_logs",
			Expected: "true",
			Actual:   "false",
		})
	}
	if baseline.ClusterConfig.LoggingConfig.EnableWorkloadLogs && !actual.LoggingConfig.EnableWorkloadLogs {
		*drifts = append(*drifts, Drift{
			Field:    "logging.workload_logs",
			Expected: "true",
			Actual:   "false",
		})
	}
}

// compareMonitoring compares monitoring configuration
func compareMonitoring(baseline *GKEBaseline, actual *ClusterConfig, drifts *[]Drift) {
	if baseline.ClusterConfig.MonitoringConfig == nil || actual.MonitoringConfig == nil {
		return
	}
	if baseline.ClusterConfig.MonitoringConfig.EnableSystemMetrics && !actual.MonitoringConfig.EnableSystemMetrics {
		*drifts = append(*drifts, Drift{
			Field:    "monitoring.system_metrics",
			Expected: "true",
			Actual:   "false",
		})
	}
	if baseline.ClusterConfig.MonitoringConfig.EnableAPIServerMetrics && !actual.MonitoringConfig.EnableAPIServerMetrics {
		*drifts = append(*drifts, Drift{
			Field:    "monitoring.apiserver_metrics",
			Expected: "true",
			Actual:   "false",
		})
	}
	if baseline.ClusterConfig.MonitoringConfig.EnableControllerMetrics && !actual.MonitoringConfig.EnableControllerMetrics {
		*drifts = append(*drifts, Drift{
			Field:    "monitoring.controller_metrics",
			Expected: "true",
			Actual:   "false",
		})
	}
	if baseline.ClusterConfig.MonitoringConfig.EnableSchedulerMetrics && !actual.MonitoringConfig.EnableSchedulerMetrics {
		*drifts = append(*drifts, Drift{
			Field:    "monitoring.scheduler_metrics",
			Expected: "true",
			Actual:   "false",
		})
	}
}
