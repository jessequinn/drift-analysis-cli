package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DriftItem represents a generic drift item for TUI display
type DriftItem struct {
	ResourceType string
	Project      string
	Name         string
	Location     string
	State        string
	Labels       map[string]string
	Drifts       []DriftDetail
}

// DriftDetail represents a single drift
type DriftDetail struct {
	Field    string
	Expected string
	Actual   string
	Severity string
}

// ReportData holds the complete report data for TUI
type ReportData struct {
	Title            string
	Timestamp        time.Time
	TotalResources   int
	DriftedResources int
	Items            []DriftItem
}

// Run starts the TUI with the provided report data
func Run(data ReportData) error {
	tabs := buildTabs(data)
	model := NewModel(tabs)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// buildTabs creates tabs from report data
func buildTabs(data ReportData) []Tab {
	tabs := []Tab{
		{
			Title:   "Overview",
			Content: buildOverviewTab(data),
		},
		{
			Title:   "Critical",
			Content: buildSeverityTab(data, "critical"),
		},
		{
			Title:   "High",
			Content: buildSeverityTab(data, "high"),
		},
		{
			Title:   "Medium",
			Content: buildSeverityTab(data, "medium"),
		},
		{
			Title:   "Low",
			Content: buildSeverityTab(data, "low"),
		},
		{
			Title:   "All Drifts",
			Content: buildAllDriftsTab(data),
		},
	}
	return tabs
}

// buildOverviewTab creates the overview tab content
func buildOverviewTab(data ReportData) string {
	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("cyan")).
		Underline(true).
		MarginTop(1).
		MarginBottom(1)

	sb.WriteString(titleStyle.Render(data.Title) + "\n\n")

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(25)

	sb.WriteString(labelStyle.Render("Generated:") + infoStyle.Render(data.Timestamp.Format(time.RFC3339)) + "\n")
	sb.WriteString(labelStyle.Render("Total Resources:") + infoStyle.Render(fmt.Sprintf("%d", data.TotalResources)) + "\n")
	sb.WriteString(labelStyle.Render("Resources with Drift:") + infoStyle.Render(fmt.Sprintf("%d", data.DriftedResources)) + "\n")

	if data.TotalResources > 0 {
		complianceRate := float64(data.TotalResources-data.DriftedResources) / float64(data.TotalResources) * 100
		complianceStyle := lipgloss.NewStyle().Bold(true)
		if complianceRate >= 90 {
			complianceStyle = complianceStyle.Foreground(lipgloss.Color("46"))
		} else if complianceRate >= 70 {
			complianceStyle = complianceStyle.Foreground(lipgloss.Color("220"))
		} else {
			complianceStyle = complianceStyle.Foreground(lipgloss.Color("196"))
		}
		sb.WriteString(labelStyle.Render("Compliance Rate:") + complianceStyle.Render(fmt.Sprintf("%.1f%%", complianceRate)) + "\n\n")
	}

	// Count drifts by severity
	critical, high, medium, low := countBySeverity(data.Items)

	sb.WriteString(titleStyle.Render("Drift Summary") + "\n\n")

	if critical > 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Render(fmt.Sprintf("  ✗ CRITICAL: %d", critical)) + "\n")
	}
	if high > 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true).
			Render(fmt.Sprintf("  [WARNING] HIGH:     %d", high)) + "\n")
	}
	if medium > 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Render(fmt.Sprintf("  ● MEDIUM:   %d", medium)) + "\n")
	}
	if low > 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render(fmt.Sprintf("  ○ LOW:      %d", low)) + "\n")
	}

	if critical+high+medium+low == 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true).
			Render("  [OK] No drifts detected - all resources comply with baseline") + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("Resources by Status") + "\n\n")

	// Group resources by drift status
	compliantCount := 0
	driftedCount := 0

	for _, item := range data.Items {
		if len(item.Drifts) == 0 {
			compliantCount++
		} else {
			driftedCount++
		}
	}

	sb.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")).
		Render(fmt.Sprintf("  [OK] Compliant: %d", compliantCount)) + "\n")

	sb.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Render(fmt.Sprintf("  ✗ Drifted:   %d", driftedCount)) + "\n")

	return sb.String()
}

// buildSeverityTab creates a tab filtered by severity
func buildSeverityTab(data ReportData, severity string) string {
	var sb strings.Builder

	filteredItems := filterBySeverity(data.Items, severity)

	if len(filteredItems) == 0 {
		okStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true).
			MarginTop(2)
		sb.WriteString(okStyle.Render(fmt.Sprintf("[OK] No %s severity drifts detected", strings.ToUpper(severity))) + "\n")
		return sb.String()
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("cyan")).
		MarginTop(1).
		MarginBottom(1)

	sb.WriteString(headerStyle.Render(fmt.Sprintf("%s Severity Drifts (%d)", strings.ToUpper(severity), len(filteredItems))) + "\n\n")

	for _, item := range filteredItems {
		sb.WriteString(formatDriftItem(item, severity))
		sb.WriteString("\n")
	}

	return sb.String()
}

// buildAllDriftsTab creates a tab with all drifts
func buildAllDriftsTab(data ReportData) string {
	var sb strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("cyan")).
		MarginTop(1).
		MarginBottom(1)

	sb.WriteString(headerStyle.Render(fmt.Sprintf("All Resources (%d)", len(data.Items))) + "\n\n")

	for _, item := range data.Items {
		sb.WriteString(formatDriftItem(item, ""))
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatDriftItem formats a single drift item
func formatDriftItem(item DriftItem, filterSeverity string) string {
	var sb strings.Builder

	// Resource header
	resourceStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("cyan"))

	sb.WriteString(resourceStyle.Render(fmt.Sprintf("● %s: %s/%s", item.ResourceType, item.Project, item.Name)) + "\n")

	locationStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	sb.WriteString(locationStyle.Render(fmt.Sprintf("  Location: %s | State: %s", item.Location, item.State)) + "\n")

	// Show labels if any
	if len(item.Labels) > 0 && item.Labels["database-role"] != "" {
		sb.WriteString(locationStyle.Render(fmt.Sprintf("  Role: %s", item.Labels["database-role"])) + "\n")
	}

	// Drifts
	if len(item.Drifts) == 0 {
		okStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)
		sb.WriteString(okStyle.Render("  [OK] No drift detected") + "\n")
	} else {
		filteredDrifts := item.Drifts
		if filterSeverity != "" {
			filteredDrifts = filterDriftsBySeverity(item.Drifts, filterSeverity)
		}

		for _, drift := range filteredDrifts {
			icon := getIconForSeverity(drift.Severity)
			severityStyle := getSeverityStyle(drift.Severity)

			fieldStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true)

			labelStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

			expectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("46"))

			actualStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

			sb.WriteString(fmt.Sprintf("    %s %s %s\n",
				icon,
				severityStyle.Render(fmt.Sprintf("[%s]", strings.ToUpper(drift.Severity))),
				fieldStyle.Render(drift.Field)))
			sb.WriteString(labelStyle.Render("       Expected: ") + expectedStyle.Render(drift.Expected) + "\n")
			sb.WriteString(labelStyle.Render("       Actual:   ") + actualStyle.Render(drift.Actual) + "\n")
		}
	}

	return sb.String()
}

// Helper functions

func countBySeverity(items []DriftItem) (critical, high, medium, low int) {
	for _, item := range items {
		for _, drift := range item.Drifts {
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

func filterBySeverity(items []DriftItem, severity string) []DriftItem {
	var filtered []DriftItem
	for _, item := range items {
		hasSeverity := false
		for _, drift := range item.Drifts {
			if drift.Severity == severity {
				hasSeverity = true
				break
			}
		}
		if hasSeverity {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterDriftsBySeverity(drifts []DriftDetail, severity string) []DriftDetail {
	var filtered []DriftDetail
	for _, drift := range drifts {
		if drift.Severity == severity {
			filtered = append(filtered, drift)
		}
	}
	return filtered
}

func getIconForSeverity(severity string) string {
	switch severity {
	case "critical":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Render("✗")
	case "high":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true).
			Render("[WARNING]")
	case "medium":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Render("●")
	case "low":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("○")
	default:
		return " "
	}
}

func getSeverityStyle(severity string) lipgloss.Style {
	style := lipgloss.NewStyle().Bold(true)
	switch severity {
	case "critical":
		return style.Foreground(lipgloss.Color("196"))
	case "high":
		return style.Foreground(lipgloss.Color("208"))
	case "medium":
		return style.Foreground(lipgloss.Color("220"))
	case "low":
		return style.Foreground(lipgloss.Color("244"))
	default:
		return style
	}
}
