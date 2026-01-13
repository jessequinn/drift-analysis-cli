package gce

import (
	"strings"

	"google.golang.org/api/compute/v1"
)

// extractVMConfig extracts VM configuration from a Compute Engine instance
func extractVMConfig(instance *compute.Instance) *VMConfig {
	config := &VMConfig{
		MachineType:        extractMachineType(instance.MachineType),
		Tags:               instance.Tags.Items,
		Labels:             instance.Labels,
		Preemptible:        instance.Scheduling != nil && instance.Scheduling.Preemptible,
		AutomaticRestart:   instance.Scheduling != nil && *instance.Scheduling.AutomaticRestart,
		OnHostMaintenance:  instance.Scheduling.OnHostMaintenance,
		DeletionProtection: instance.DeletionProtection,
	}

	// Extract boot disk info
	if len(instance.Disks) > 0 {
		bootDisk := instance.Disks[0]
		config.DiskSizeGB = bootDisk.DiskSizeGb
		config.DiskType = extractDiskType(bootDisk.Type)

		if bootDisk.InitializeParams != nil {
			config.ImageFamily = extractImageFamily(bootDisk.InitializeParams.SourceImage)
			config.ImageProject = extractImageProject(bootDisk.InitializeParams.SourceImage)
		}
	}

	// Extract network config
	if len(instance.NetworkInterfaces) > 0 {
		config.NetworkConfig = extractNetworkConfig(instance.NetworkInterfaces[0])
	}

	// Extract service account
	if len(instance.ServiceAccounts) > 0 {
		config.ServiceAccount = &ServiceAccountConfig{
			Email:  instance.ServiceAccounts[0].Email,
			Scopes: instance.ServiceAccounts[0].Scopes,
		}
	}

	// Extract metadata
	config.Metadata = extractMetadata(instance)

	// Extract Shielded VM config
	if instance.ShieldedInstanceConfig != nil {
		config.ShieldedVM = &ShieldedVMConfig{
			EnableSecureBoot:          instance.ShieldedInstanceConfig.EnableSecureBoot,
			EnableVTPM:                instance.ShieldedInstanceConfig.EnableVtpm,
			EnableIntegrityMonitoring: instance.ShieldedInstanceConfig.EnableIntegrityMonitoring,
		}
	}

	return config
}

// extractNetworkConfig extracts network interface configuration
func extractNetworkConfig(networkInterface *compute.NetworkInterface) *NetworkConfig {
	config := &NetworkConfig{
		Network:    extractNetworkName(networkInterface.Network),
		Subnetwork: extractSubnetworkName(networkInterface.Subnetwork),
		InternalIP: networkInterface.NetworkIP,
		ExternalIP: len(networkInterface.AccessConfigs) > 0,
	}

	if len(networkInterface.AccessConfigs) > 0 {
		config.NetworkTier = networkInterface.AccessConfigs[0].NetworkTier
	}

	return config
}

// extractMetadata converts instance metadata to a map
func extractMetadata(instance *compute.Instance) map[string]string {
	metadata := make(map[string]string)
	if instance.Metadata != nil {
		for _, item := range instance.Metadata.Items {
			if item.Value != nil {
				metadata[item.Key] = *item.Value
			}
		}
	}
	return metadata
}

// extractMachineType extracts the machine type name from the full URL
func extractMachineType(machineTypeURL string) string {
	parts := strings.Split(machineTypeURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return machineTypeURL
}

// extractDiskType extracts the disk type from the full URL
func extractDiskType(diskTypeURL string) string {
	parts := strings.Split(diskTypeURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return diskTypeURL
}

// extractImageFamily extracts the image family from the source image URL
func extractImageFamily(sourceImage string) string {
	if sourceImage == "" {
		return ""
	}
	parts := strings.Split(sourceImage, "/")
	if len(parts) > 0 {
		imageName := parts[len(parts)-1]
		// Extract family name (e.g., "ubuntu-2204-jammy-v20231213" -> "ubuntu-2204")
		if idx := strings.LastIndex(imageName, "-v"); idx > 0 {
			return imageName[:idx]
		}
		return imageName
	}
	return sourceImage
}

// extractImageProject extracts the project from the source image URL
func extractImageProject(sourceImage string) string {
	if sourceImage == "" {
		return ""
	}
	parts := strings.Split(sourceImage, "/")
	for i, part := range parts {
		if part == "projects" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// extractNetworkName extracts the network name from the full URL
func extractNetworkName(networkURL string) string {
	parts := strings.Split(networkURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return networkURL
}

// extractSubnetworkName extracts the subnetwork name from the full URL
func extractSubnetworkName(subnetworkURL string) string {
	parts := strings.Split(subnetworkURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return subnetworkURL
}

// extractZoneFromURL extracts the zone name from the zone URL
func extractZoneFromURL(zoneURL string) string {
	parts := strings.Split(zoneURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return zoneURL
}

// generateRecommendations creates actionable recommendations based on detected drifts
func generateRecommendations(drifts []Drift) []string {
	recommendations := make([]string, 0)
	seen := make(map[string]bool)

	for _, drift := range drifts {
		var recommendation string

		switch drift.Field {
		case "shielded_vm.enable_secure_boot":
			if !seen["shielded_vm"] {
				recommendation = "Enable Shielded VM features (Secure Boot, vTPM, Integrity Monitoring) for enhanced security"
				seen["shielded_vm"] = true
			}
		case "shielded_vm.enable_vtpm", "shielded_vm.enable_integrity_monitoring":
			if !seen["shielded_vm"] {
				recommendation = "Complete Shielded VM configuration for full security benefits"
				seen["shielded_vm"] = true
			}
		case "network_config.external_ip":
			recommendation = "Remove external IP and use Cloud NAT or Identity-Aware Proxy for secure access"
		case "service_account.email":
			recommendation = "Use a custom service account with minimal required permissions"
		case "service_account.scopes":
			recommendation = "Restrict service account scopes to minimum required permissions"
		case "preemptible":
			recommendation = "Avoid preemptible VMs for production workloads requiring high availability"
		case "automatic_restart":
			recommendation = "Enable automatic restart for production instances to maintain availability"
		case "on_host_maintenance":
			recommendation = "Set on_host_maintenance to MIGRATE for zero-downtime maintenance"
		case "deletion_protection":
			recommendation = "Enable deletion protection for critical production instances"
		case "machine_type":
			recommendation = "Ensure machine type matches performance requirements and cost expectations"
		case "disk_type":
			recommendation = "Use pd-ssd for production workloads requiring better I/O performance"
		case "disk_size_gb":
			recommendation = "Review disk size to ensure adequate capacity for workload"
		case "metadata.enable-oslogin":
			recommendation = "Enable OS Login for centralized SSH key and access management"
		case "network_config.network_tier":
			recommendation = "Use PREMIUM network tier for production workloads requiring low latency"
		}

		if recommendation != "" && !seen[recommendation] {
			recommendations = append(recommendations, recommendation)
			seen[recommendation] = true
		}
	}

	return recommendations
}
