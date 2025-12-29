package report

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Drift represents a single configuration difference from the baseline
type Drift struct {
	Field    string `json:"field" yaml:"field"`
	Expected string `json:"expected" yaml:"expected"`
	Actual   string `json:"actual" yaml:"actual"`
	Severity string `json:"severity" yaml:"severity"`
}

// GetIconForSeverity returns an appropriate styled icon for the severity level
func GetIconForSeverity(severity string) string {
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

// CountBySeverity tallies the number of drifts by severity level
func CountBySeverity(drifts []Drift) (critical, high, medium, low int) {
	for _, drift := range drifts {
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
	return
}

// FormatDriftSummary generates a formatted summary of drifts by severity
func FormatDriftSummary(critical, high, medium, low int) string {
	var sb strings.Builder
	if critical+high+medium+low > 0 {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("cyan")).
			Underline(true)

		sb.WriteString(titleStyle.Render("Drift Summary") + "\n")
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
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatDrifts generates formatted text for a list of drifts
func FormatDrifts(drifts []Drift) string {
	var sb strings.Builder
	if len(drifts) == 0 {
		okStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)
		sb.WriteString(okStyle.Render("[OK] No drift detected") + "\n")
	} else {
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("cyan"))

		fieldStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true)

		labelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

		expectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

		actualStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

		sb.WriteString(headerStyle.Render(fmt.Sprintf("Detected Drifts: %d", len(drifts))) + "\n\n")
		for _, drift := range drifts {
			icon := GetIconForSeverity(drift.Severity)
			severityStyle := lipgloss.NewStyle().Bold(true)
			switch drift.Severity {
			case "critical":
				severityStyle = severityStyle.Foreground(lipgloss.Color("196"))
			case "high":
				severityStyle = severityStyle.Foreground(lipgloss.Color("208"))
			case "medium":
				severityStyle = severityStyle.Foreground(lipgloss.Color("220"))
			case "low":
				severityStyle = severityStyle.Foreground(lipgloss.Color("244"))
			}

			sb.WriteString(fmt.Sprintf("  %s %s %s\n",
				icon,
				severityStyle.Render(fmt.Sprintf("[%s]", strings.ToUpper(drift.Severity))),
				fieldStyle.Render(drift.Field)))
			sb.WriteString(labelStyle.Render("     Expected: ") + expectedStyle.Render(drift.Expected) + "\n")
			sb.WriteString(labelStyle.Render("     Actual:   ") + actualStyle.Render(drift.Actual) + "\n")
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
