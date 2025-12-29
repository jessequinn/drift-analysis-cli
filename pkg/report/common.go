package report

import (
	"fmt"
	"strings"
)

// Drift represents a single configuration difference from the baseline
type Drift struct {
	Field    string `json:"field" yaml:"field"`
	Expected string `json:"expected" yaml:"expected"`
	Actual   string `json:"actual" yaml:"actual"`
	Severity string `json:"severity" yaml:"severity"`
}

// GetIconForSeverity returns an appropriate text marker for the severity level
func GetIconForSeverity(severity string) string {
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
		sb.WriteString("Drift Summary:\n")
		if critical > 0 {
			sb.WriteString(fmt.Sprintf("  [!] CRITICAL: %d\n", critical))
		}
		if high > 0 {
			sb.WriteString(fmt.Sprintf("  [!] HIGH:     %d\n", high))
		}
		if medium > 0 {
			sb.WriteString(fmt.Sprintf("  [*] MEDIUM:   %d\n", medium))
		}
		if low > 0 {
			sb.WriteString(fmt.Sprintf("  [-] LOW:      %d\n", low))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatDrifts generates formatted text for a list of drifts
func FormatDrifts(drifts []Drift) string {
	var sb strings.Builder
	if len(drifts) == 0 {
		sb.WriteString("[OK] No drift detected\n")
	} else {
		sb.WriteString(fmt.Sprintf("Detected Drifts: %d\n\n", len(drifts)))
		for _, drift := range drifts {
			icon := GetIconForSeverity(drift.Severity)
			sb.WriteString(fmt.Sprintf("  %s [%s] %s\n", icon, strings.ToUpper(drift.Severity), drift.Field))
			sb.WriteString(fmt.Sprintf("     Expected: %s\n", drift.Expected))
			sb.WriteString(fmt.Sprintf("     Actual:   %s\n", drift.Actual))
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
