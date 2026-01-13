package gce

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Drift represents a single configuration drift
type Drift struct {
	Field       string      `json:"field"`
	Expected    interface{} `json:"expected"`
	Actual      interface{} `json:"actual"`
	Severity    string      `json:"severity"`
	Description string      `json:"description,omitempty"`
}

// compareVMConfigs compares actual VM config against baseline and returns list of drifts
func compareVMConfigs(actual, baseline *VMConfig) []Drift {
	var drifts []Drift

	if actual == nil || baseline == nil {
		return drifts
	}

	// CRITICAL: Shielded VM configuration
	if baseline.ShieldedVM != nil {
		drifts = append(drifts, compareShieldedVM(actual.ShieldedVM, baseline.ShieldedVM)...)
	}

	// CRITICAL: Network configuration (external IP exposure)
	if baseline.NetworkConfig != nil && actual.NetworkConfig != nil {
		drifts = append(drifts, compareNetworkConfig(actual.NetworkConfig, baseline.NetworkConfig)...)
	}

	// CRITICAL: Service Account configuration
	if baseline.ServiceAccount != nil {
		drifts = append(drifts, compareServiceAccount(actual.ServiceAccount, baseline.ServiceAccount)...)
	}

	// CRITICAL: Deletion protection
	if baseline.DeletionProtection != actual.DeletionProtection {
		drifts = append(drifts, Drift{
			Field:       "deletion_protection",
			Expected:    baseline.DeletionProtection,
			Actual:      actual.DeletionProtection,
			Severity:    "CRITICAL",
			Description: "Deletion protection mismatch - critical instances should be protected",
		})
	}

	// HIGH: Preemptibility
	if baseline.Preemptible != actual.Preemptible {
		drifts = append(drifts, Drift{
			Field:       "preemptible",
			Expected:    baseline.Preemptible,
			Actual:      actual.Preemptible,
			Severity:    "HIGH",
			Description: "Preemptible setting mismatch - affects instance reliability",
		})
	}

	// HIGH: Automatic restart
	if baseline.AutomaticRestart != actual.AutomaticRestart {
		drifts = append(drifts, Drift{
			Field:       "automatic_restart",
			Expected:    baseline.AutomaticRestart,
			Actual:      actual.AutomaticRestart,
			Severity:    "HIGH",
			Description: "Automatic restart configuration mismatch - affects availability",
		})
	}

	// HIGH: Host maintenance behavior
	if baseline.OnHostMaintenance != "" && baseline.OnHostMaintenance != actual.OnHostMaintenance {
		drifts = append(drifts, Drift{
			Field:       "on_host_maintenance",
			Expected:    baseline.OnHostMaintenance,
			Actual:      actual.OnHostMaintenance,
			Severity:    "HIGH",
			Description: "Host maintenance behavior mismatch - affects availability during maintenance",
		})
	}

	// MEDIUM: Machine type
	if baseline.MachineType != "" && baseline.MachineType != actual.MachineType {
		drifts = append(drifts, Drift{
			Field:       "machine_type",
			Expected:    baseline.MachineType,
			Actual:      actual.MachineType,
			Severity:    "MEDIUM",
			Description: "Machine type mismatch - affects performance and cost",
		})
	}

	// MEDIUM: Disk configuration
	if baseline.DiskSizeGB > 0 && baseline.DiskSizeGB != actual.DiskSizeGB {
		drifts = append(drifts, Drift{
			Field:       "disk_size_gb",
			Expected:    baseline.DiskSizeGB,
			Actual:      actual.DiskSizeGB,
			Severity:    "MEDIUM",
			Description: "Boot disk size mismatch",
		})
	}

	if baseline.DiskType != "" && baseline.DiskType != actual.DiskType {
		drifts = append(drifts, Drift{
			Field:       "disk_type",
			Expected:    baseline.DiskType,
			Actual:      actual.DiskType,
			Severity:    "MEDIUM",
			Description: "Disk type mismatch - affects I/O performance and cost",
		})
	}

	// MEDIUM: Image family
	if baseline.ImageFamily != "" && baseline.ImageFamily != actual.ImageFamily {
		drifts = append(drifts, Drift{
			Field:       "image_family",
			Expected:    baseline.ImageFamily,
			Actual:      actual.ImageFamily,
			Severity:    "MEDIUM",
			Description: "Image family mismatch - may affect compatibility and security patches",
		})
	}

	// LOW: Tags
	if len(baseline.Tags) > 0 {
		drifts = append(drifts, compareTags(actual.Tags, baseline.Tags)...)
	}

	// LOW: Labels
	if len(baseline.Labels) > 0 {
		drifts = append(drifts, compareLabels(actual.Labels, baseline.Labels)...)
	}

	// LOW: Metadata
	if len(baseline.Metadata) > 0 {
		drifts = append(drifts, compareMetadata(actual.Metadata, baseline.Metadata)...)
	}

	return drifts
}

// compareShieldedVM compares Shielded VM configurations (CRITICAL)
func compareShieldedVM(actual, baseline *ShieldedVMConfig) []Drift {
	var drifts []Drift

	if baseline == nil {
		return drifts
	}

	if actual == nil {
		actual = &ShieldedVMConfig{}
	}

	if baseline.EnableSecureBoot && !actual.EnableSecureBoot {
		drifts = append(drifts, Drift{
			Field:       "shielded_vm.enable_secure_boot",
			Expected:    true,
			Actual:      false,
			Severity:    "CRITICAL",
			Description: "Secure Boot disabled - reduces protection against rootkits and bootkits",
		})
	}

	if baseline.EnableVTPM && !actual.EnableVTPM {
		drifts = append(drifts, Drift{
			Field:       "shielded_vm.enable_vtpm",
			Expected:    true,
			Actual:      false,
			Severity:    "CRITICAL",
			Description: "vTPM disabled - required for measured boot and integrity monitoring",
		})
	}

	if baseline.EnableIntegrityMonitoring && !actual.EnableIntegrityMonitoring {
		drifts = append(drifts, Drift{
			Field:       "shielded_vm.enable_integrity_monitoring",
			Expected:    true,
			Actual:      false,
			Severity:    "CRITICAL",
			Description: "Integrity monitoring disabled - cannot detect boot-level modifications",
		})
	}

	return drifts
}

// compareNetworkConfig compares network configurations (CRITICAL for external IP)
func compareNetworkConfig(actual, baseline *NetworkConfig) []Drift {
	var drifts []Drift

	if baseline.ExternalIP != actual.ExternalIP {
		severity := "MEDIUM"
		description := "External IP configuration mismatch"

		// Critical if baseline expects no external IP but instance has one (security risk)
		if !baseline.ExternalIP && actual.ExternalIP {
			severity = "CRITICAL"
			description = "Instance has external IP but baseline expects private IP only - security risk"
		}

		drifts = append(drifts, Drift{
			Field:       "network_config.external_ip",
			Expected:    baseline.ExternalIP,
			Actual:      actual.ExternalIP,
			Severity:    severity,
			Description: description,
		})
	}

	if baseline.Network != "" && baseline.Network != actual.Network {
		drifts = append(drifts, Drift{
			Field:       "network_config.network",
			Expected:    baseline.Network,
			Actual:      actual.Network,
			Severity:    "MEDIUM",
			Description: "VPC network mismatch",
		})
	}

	if baseline.Subnetwork != "" && baseline.Subnetwork != actual.Subnetwork {
		drifts = append(drifts, Drift{
			Field:       "network_config.subnetwork",
			Expected:    baseline.Subnetwork,
			Actual:      actual.Subnetwork,
			Severity:    "MEDIUM",
			Description: "Subnetwork mismatch",
		})
	}

	if baseline.NetworkTier != "" && baseline.NetworkTier != actual.NetworkTier {
		drifts = append(drifts, Drift{
			Field:       "network_config.network_tier",
			Expected:    baseline.NetworkTier,
			Actual:      actual.NetworkTier,
			Severity:    "LOW",
			Description: "Network tier mismatch - affects network performance and cost",
		})
	}

	return drifts
}

// compareServiceAccount compares service account configurations (CRITICAL)
func compareServiceAccount(actual, baseline *ServiceAccountConfig) []Drift {
	var drifts []Drift

	if baseline == nil {
		return drifts
	}

	if actual == nil {
		drifts = append(drifts, Drift{
			Field:       "service_account",
			Expected:    baseline.Email,
			Actual:      "none",
			Severity:    "CRITICAL",
			Description: "No service account configured",
		})
		return drifts
	}

	if baseline.Email != "" && baseline.Email != actual.Email {
		// Using default compute service account is a security risk
		severity := "HIGH"
		if strings.Contains(actual.Email, "-compute@developer.gserviceaccount.com") {
			severity = "CRITICAL"
		}

		drifts = append(drifts, Drift{
			Field:       "service_account.email",
			Expected:    baseline.Email,
			Actual:      actual.Email,
			Severity:    severity,
			Description: "Service account mismatch - affects permissions and security",
		})
	}

	if len(baseline.Scopes) > 0 {
		scopeDrifts := compareScopes(actual.Scopes, baseline.Scopes)
		drifts = append(drifts, scopeDrifts...)
	}

	return drifts
}

// compareScopes compares service account scopes (CRITICAL)
func compareScopes(actual, baseline []string) []Drift {
	var drifts []Drift

	actualSet := make(map[string]bool)
	for _, scope := range actual {
		actualSet[scope] = true
	}

	baselineSet := make(map[string]bool)
	for _, scope := range baseline {
		baselineSet[scope] = true
	}

	// Check for missing scopes
	var missing []string
	for scope := range baselineSet {
		if !actualSet[scope] {
			missing = append(missing, scope)
		}
	}

	// Check for extra scopes (especially dangerous ones)
	var extra []string
	for scope := range actualSet {
		if !baselineSet[scope] {
			extra = append(extra, scope)
		}
	}

	if len(missing) > 0 {
		drifts = append(drifts, Drift{
			Field:       "service_account.scopes",
			Expected:    baseline,
			Actual:      actual,
			Severity:    "HIGH",
			Description: fmt.Sprintf("Missing required scopes: %s", strings.Join(missing, ", ")),
		})
	}

	// Extra scopes, especially cloud-platform, are critical security issues
	if len(extra) > 0 {
		severity := "HIGH"
		for _, scope := range extra {
			if strings.Contains(scope, "cloud-platform") {
				severity = "CRITICAL"
				break
			}
		}

		drifts = append(drifts, Drift{
			Field:       "service_account.scopes",
			Expected:    baseline,
			Actual:      actual,
			Severity:    severity,
			Description: fmt.Sprintf("Extra scopes detected (over-permissioned): %s", strings.Join(extra, ", ")),
		})
	}

	return drifts
}

// compareTags compares network tags (LOW)
func compareTags(actual, baseline []string) []Drift {
	var drifts []Drift

	if !stringSlicesEqual(actual, baseline) {
		drifts = append(drifts, Drift{
			Field:       "tags",
			Expected:    baseline,
			Actual:      actual,
			Severity:    "LOW",
			Description: "Network tags mismatch - may affect firewall rules",
		})
	}

	return drifts
}

// compareLabels compares resource labels (LOW)
func compareLabels(actual, baseline map[string]string) []Drift {
	var drifts []Drift

	for key, expectedValue := range baseline {
		if actualValue, exists := actual[key]; !exists || actualValue != expectedValue {
			drifts = append(drifts, Drift{
				Field:       fmt.Sprintf("labels.%s", key),
				Expected:    expectedValue,
				Actual:      actualValue,
				Severity:    "LOW",
				Description: "Label mismatch - may affect organization and billing",
			})
		}
	}

	return drifts
}

// compareMetadata compares instance metadata (LOW to MEDIUM)
func compareMetadata(actual, baseline map[string]string) []Drift {
	var drifts []Drift

	for key, expectedValue := range baseline {
		actualValue, exists := actual[key]

		severity := "LOW"
		description := "Metadata mismatch"

		// Some metadata keys are more critical
		if key == "enable-oslogin" || key == "enable-os-login" {
			severity = "MEDIUM"
			description = "OS Login configuration mismatch - affects access management"
		} else if strings.HasPrefix(key, "startup-script") {
			severity = "MEDIUM"
			description = "Startup script mismatch - affects instance initialization"
		} else if strings.Contains(key, "ssh") {
			severity = "MEDIUM"
			description = "SSH configuration mismatch - affects access control"
		}

		if !exists || actualValue != expectedValue {
			drifts = append(drifts, Drift{
				Field:       fmt.Sprintf("metadata.%s", key),
				Expected:    expectedValue,
				Actual:      actualValue,
				Severity:    severity,
				Description: description,
			})
		}
	}

	return drifts
}

// stringSlicesEqual checks if two string slices are equal (order independent)
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	sort.Strings(aCopy)
	sort.Strings(bCopy)

	return reflect.DeepEqual(aCopy, bCopy)
}
