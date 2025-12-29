package csql

import (
	"fmt"
	"strings"
	"time"
)

// DriftReport contains the complete analysis results for all instances
type DriftReport struct {
	Timestamp        time.Time        `json:"timestamp" yaml:"timestamp"`
	TotalInstances   int              `json:"total_instances" yaml:"total_instances"`
	DriftedInstances int              `json:"drifted_instances" yaml:"drifted_instances"`
	Instances        []*InstanceDrift `json:"instances" yaml:"instances"`
}

// InstanceDrift represents drift analysis results for a single database instance
type InstanceDrift struct {
	Project           string             `json:"project" yaml:"project"`
	Name              string             `json:"name" yaml:"name"`
	Region            string             `json:"region" yaml:"region"`
	State             string             `json:"state" yaml:"state"`
	Labels            map[string]string  `json:"labels,omitempty" yaml:"labels,omitempty"`
	Databases         []string           `json:"databases,omitempty" yaml:"databases,omitempty"`
	MaintenanceWindow *MaintenanceWindow `json:"maintenance_window,omitempty" yaml:"maintenance_window,omitempty"`
	Drifts            []Drift            `json:"drifts" yaml:"drifts"`
	Recommendations   []string           `json:"recommendations" yaml:"recommendations"`
}

// Drift represents a single configuration difference from the baseline
type Drift struct {
	Field    string `json:"field" yaml:"field"`
	Expected string `json:"expected" yaml:"expected"`
	Actual   string `json:"actual" yaml:"actual"`
	Severity string `json:"severity" yaml:"severity"`
}

// FormatText generates a human-readable text report with summary and detailed drift information
func (r *DriftReport) FormatText() string {
	var sb strings.Builder

	sb.WriteString("═══════════════════════════════════════════════════════════════════════════════\n")
	sb.WriteString("  GCP PostgreSQL Drift Analysis Report\n")
	sb.WriteString("═══════════════════════════════════════════════════════════════════════════════\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n", r.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Total Instances: %d\n", r.TotalInstances))
	sb.WriteString(fmt.Sprintf("Instances with Drift: %d\n", r.DriftedInstances))
	sb.WriteString(fmt.Sprintf("Compliance Rate: %.1f%%\n\n", 
		float64(r.TotalInstances-r.DriftedInstances)/float64(r.TotalInstances)*100))

	// Summary by severity
	criticalCount, highCount, mediumCount, lowCount := r.countBySeverity()
	if criticalCount+highCount+mediumCount+lowCount > 0 {
		sb.WriteString("Drift Summary:\n")
		if criticalCount > 0 {
			sb.WriteString(fmt.Sprintf("  [!] CRITICAL: %d\n", criticalCount))
		}
		if highCount > 0 {
			sb.WriteString(fmt.Sprintf("  [!] HIGH:     %d\n", highCount))
		}
		if mediumCount > 0 {
			sb.WriteString(fmt.Sprintf("  [*] MEDIUM:   %d\n", mediumCount))
		}
		if lowCount > 0 {
			sb.WriteString(fmt.Sprintf("  [-] LOW:      %d\n", lowCount))
		}
		sb.WriteString("\n")
	}

	// Detailed instance reports
	for i, inst := range r.Instances {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(inst.FormatText())
	}

	return sb.String()
}

// countBySeverity tallies the number of drifts by severity level across all instances
func (r *DriftReport) countBySeverity() (critical, high, medium, low int) {
	for _, inst := range r.Instances {
		for _, drift := range inst.Drifts {
			switch drift.Severity {
			case "critical":
				critical++
			case "high":
				high++
			case "medium":
				medium++
			case "low":
				low++
			}
		}
	}
	return
}

// FormatText generates a formatted text representation of instance drift details
func (id *InstanceDrift) FormatText() string {
	var sb strings.Builder

	sb.WriteString("───────────────────────────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("Instance: %s\n", id.Name))
	sb.WriteString(fmt.Sprintf("Project:  %s\n", id.Project))
	sb.WriteString(fmt.Sprintf("Region:   %s\n", id.Region))
	sb.WriteString(fmt.Sprintf("State:    %s\n", id.State))

	if len(id.Labels) > 0 {
		if role, exists := id.Labels["database-role"]; exists {
			sb.WriteString(fmt.Sprintf("Role:     %s\n", role))
		}
	}

	if id.MaintenanceWindow != nil {
		sb.WriteString(fmt.Sprintf("Maintenance Window: Day %d, Hour %d UTC (%s)\n", 
			id.MaintenanceWindow.Day, id.MaintenanceWindow.Hour, id.MaintenanceWindow.UpdateTrack))
	}

	sb.WriteString("\n")

	if len(id.Drifts) == 0 {
		sb.WriteString("[OK] No drift detected\n")
	} else {
		sb.WriteString(fmt.Sprintf("Detected Drifts: %d\n\n", len(id.Drifts)))
		
		for _, drift := range id.Drifts {
			icon := getIconForSeverity(drift.Severity)
			sb.WriteString(fmt.Sprintf("  %s [%s] %s\n", icon, strings.ToUpper(drift.Severity), drift.Field))
			sb.WriteString(fmt.Sprintf("     Expected: %s\n", drift.Expected))
			sb.WriteString(fmt.Sprintf("     Actual:   %s\n", drift.Actual))
			sb.WriteString("\n")
		}
	}

	if len(id.Recommendations) > 0 {
		sb.WriteString("Recommendations:\n")
		for _, rec := range id.Recommendations {
			sb.WriteString(fmt.Sprintf("  - %s\n", rec))
		}
	}

	return sb.String()
}

// getIconForSeverity returns an appropriate text marker for the severity level
func getIconForSeverity(severity string) string {
	switch severity {
	case "critical":
		return "[!]"
	case "high":
		return "[!]"
	case "medium":
		return "[*]"
	case "low":
		return "[-]"
	default:
		return "[ ]"
	}
}
