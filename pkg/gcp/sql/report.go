package sql

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jessequinn/drift-analysis-cli/pkg/report"
	"gopkg.in/yaml.v3"
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
type Drift = report.Drift

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
	sb.WriteString(report.FormatDriftSummary(criticalCount, highCount, mediumCount, lowCount))

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

	// Define styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("───────────────────────────────────────────────────────────────────────────────")

	sb.WriteString(divider + "\n")
	sb.WriteString(headerStyle.Render(fmt.Sprintf("Cloud SQL Instance: %s", id.Name)) + "\n\n")
	sb.WriteString(labelStyle.Render("Project:  ") + valueStyle.Render(id.Project) + "\n")
	sb.WriteString(labelStyle.Render("Region:   ") + valueStyle.Render(id.Region) + "\n")
	sb.WriteString(labelStyle.Render("State:    ") + valueStyle.Render(id.State) + "\n")

	if len(id.Labels) > 0 {
		if role, exists := id.Labels["database-role"]; exists {
			sb.WriteString(labelStyle.Render("Role:     ") + valueStyle.Render(role) + "\n")
		}
	}

	if id.MaintenanceWindow != nil {
		sb.WriteString(labelStyle.Render("Maintenance Window: ") +
			valueStyle.Render(fmt.Sprintf("Day %d, Hour %d UTC (%s)",
				id.MaintenanceWindow.Day, id.MaintenanceWindow.Hour, id.MaintenanceWindow.UpdateTrack)) + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(report.FormatDrifts(id.Drifts))

	if len(id.Recommendations) > 0 {
		recStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true)
		sb.WriteString(recStyle.Render("💡 Recommendations:") + "\n")
		for _, rec := range id.Recommendations {
			sb.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				Render(fmt.Sprintf("  • %s", rec)) + "\n")
		}
	}

	return sb.String()
}

// FormatJSON generates JSON output of the drift report
func (r *DriftReport) FormatJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// FormatYAML generates YAML output of the drift report
func (r *DriftReport) FormatYAML() (string, error) {
	data, err := yaml.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return string(data), nil
}
