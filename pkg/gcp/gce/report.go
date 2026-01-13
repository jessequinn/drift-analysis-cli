package gce

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DriftReport contains the complete drift analysis results
type DriftReport struct {
	Timestamp        time.Time             `json:"timestamp"`
	TotalInstances   int                   `json:"total_instances"`
	DriftedInstances int                   `json:"drifted_instances"`
	CriticalCount    int                   `json:"critical_count"`
	HighCount        int                   `json:"high_count"`
	MediumCount      int                   `json:"medium_count"`
	LowCount         int                   `json:"low_count"`
	Instances        []InstanceDriftReport `json:"instances"`
}

// InstanceDriftReport contains drift information for a single instance
type InstanceDriftReport struct {
	Project         string            `json:"project"`
	Name            string            `json:"name"`
	Zone            string            `json:"zone"`
	Status          string            `json:"status"`
	Labels          map[string]string `json:"labels,omitempty"`
	Drifts          []Drift           `json:"drifts"`
	Recommendations []string          `json:"recommendations,omitempty"`
}

// FormatText generates a human-readable text report
func (r *DriftReport) FormatText() string {
	var sb strings.Builder

	// Header
	sb.WriteString("===============================================================================\n")
	sb.WriteString(" GCP Compute Engine Drift Analysis Report\n")
	sb.WriteString("===============================================================================\n\n")

	// Summary
	sb.WriteString(fmt.Sprintf("Generated: %s\n", r.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Total Instances: %d\n", r.TotalInstances))
	sb.WriteString(fmt.Sprintf("Instances with Drift: %d\n", r.DriftedInstances))

	complianceRate := 0.0
	if r.TotalInstances > 0 {
		complianceRate = float64(r.TotalInstances-r.DriftedInstances) / float64(r.TotalInstances) * 100
	}
	sb.WriteString(fmt.Sprintf("Compliance Rate: %.1f%%\n\n", complianceRate))

	// Drift summary by severity
	sb.WriteString("Drift Summary:\n")
	if r.CriticalCount > 0 {
		sb.WriteString(fmt.Sprintf(" [!] CRITICAL: %d\n", r.CriticalCount))
	}
	if r.HighCount > 0 {
		sb.WriteString(fmt.Sprintf(" [!] HIGH: %d\n", r.HighCount))
	}
	if r.MediumCount > 0 {
		sb.WriteString(fmt.Sprintf(" [*] MEDIUM: %d\n", r.MediumCount))
	}
	if r.LowCount > 0 {
		sb.WriteString(fmt.Sprintf(" [-] LOW: %d\n", r.LowCount))
	}
	sb.WriteString("\n")

	// Sort instances: drifted first, then by name
	instances := make([]InstanceDriftReport, len(r.Instances))
	copy(instances, r.Instances)
	sort.Slice(instances, func(i, j int) bool {
		if len(instances[i].Drifts) != len(instances[j].Drifts) {
			return len(instances[i].Drifts) > len(instances[j].Drifts)
		}
		return instances[i].Name < instances[j].Name
	})

	// Instance details
	for _, instance := range instances {
		if len(instance.Drifts) == 0 {
			continue // Skip instances with no drift
		}

		sb.WriteString("-------------------------------------------------------------------------------\n")
		sb.WriteString(fmt.Sprintf("Instance: %s\n", instance.Name))
		sb.WriteString(fmt.Sprintf("Project: %s\n", instance.Project))
		sb.WriteString(fmt.Sprintf("Zone: %s\n", instance.Zone))
		sb.WriteString(fmt.Sprintf("Status: %s\n", instance.Status))

		if len(instance.Labels) > 0 {
			var labelPairs []string
			for k, v := range instance.Labels {
				labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
			}
			sort.Strings(labelPairs)
			sb.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(labelPairs, ", ")))
		}

		sb.WriteString(fmt.Sprintf("\nDetected Drifts: %d\n\n", len(instance.Drifts)))

		// Sort drifts by severity
		drifts := make([]Drift, len(instance.Drifts))
		copy(drifts, instance.Drifts)
		sort.Slice(drifts, func(i, j int) bool {
			severityOrder := map[string]int{"CRITICAL": 0, "HIGH": 1, "MEDIUM": 2, "LOW": 3}
			return severityOrder[drifts[i].Severity] < severityOrder[drifts[j].Severity]
		})

		for _, drift := range drifts {
			icon := getSeverityIcon(drift.Severity)
			sb.WriteString(fmt.Sprintf(" %s [%s] %s\n", icon, drift.Severity, drift.Field))
			sb.WriteString(fmt.Sprintf("    Expected: %v\n", formatValue(drift.Expected)))
			sb.WriteString(fmt.Sprintf("    Actual:   %v\n", formatValue(drift.Actual)))
			if drift.Description != "" {
				sb.WriteString(fmt.Sprintf("    %s\n", drift.Description))
			}
			sb.WriteString("\n")
		}

		// Recommendations
		if len(instance.Recommendations) > 0 {
			sb.WriteString("Recommendations:\n")
			for _, rec := range instance.Recommendations {
				sb.WriteString(fmt.Sprintf(" • %s\n", rec))
			}
			sb.WriteString("\n")
		}
	}

	// Summary of compliant instances
	compliantCount := r.TotalInstances - r.DriftedInstances
	if compliantCount > 0 {
		sb.WriteString("-------------------------------------------------------------------------------\n")
		sb.WriteString(fmt.Sprintf("%d instance(s) are compliant with baseline configuration\n", compliantCount))
	}

	sb.WriteString("===============================================================================\n")

	return sb.String()
}

// FormatJSON generates a JSON report
func (r *DriftReport) FormatJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// FormatYAML generates a YAML report
func (r *DriftReport) FormatYAML() (string, error) {
	data, err := yaml.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return string(data), nil
}

// getSeverityIcon returns an icon for the severity level
func getSeverityIcon(severity string) string {
	switch severity {
	case "CRITICAL":
		return "[!]"
	case "HIGH":
		return "[!]"
	case "MEDIUM":
		return "[*]"
	case "LOW":
		return "[-]"
	default:
		return "[ ]"
	}
}

// formatValue formats a value for display
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case []string:
		if len(val) == 0 {
			return "[]"
		}
		return "[" + strings.Join(val, ", ") + "]"
	case map[string]string:
		if len(val) == 0 {
			return "{}"
		}
		var pairs []string
		for k, v := range val {
			pairs = append(pairs, fmt.Sprintf("%s: %s", k, v))
		}
		sort.Strings(pairs)
		return "{" + strings.Join(pairs, ", ") + "}"
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "<not set>"
	default:
		return fmt.Sprintf("%v", v)
	}
}
