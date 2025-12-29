package gke

import (
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/drift-analysis-cli/pkg/report"
)

// DriftReport contains the complete analysis results for all clusters
type DriftReport struct {
	Timestamp       time.Time       `json:"timestamp" yaml:"timestamp"`
	TotalClusters   int             `json:"total_clusters" yaml:"total_clusters"`
	DriftedClusters int             `json:"drifted_clusters" yaml:"drifted_clusters"`
	Instances       []*ClusterDrift `json:"instances" yaml:"instances"`
}

// ClusterDrift represents drift analysis results for a single GKE cluster
type ClusterDrift struct {
	Project   string            `json:"project" yaml:"project"`
	Name      string            `json:"name" yaml:"name"`
	Location  string            `json:"location" yaml:"location"`
	Status    string            `json:"status" yaml:"status"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	NodePools []*NodePoolConfig `json:"node_pools,omitempty" yaml:"node_pools,omitempty"`
	Drifts    []Drift           `json:"drifts" yaml:"drifts"`
}

// Drift represents a single configuration difference from the baseline
type Drift = report.Drift

// FormatText generates a human-readable text report
func (r *DriftReport) FormatText() string {
	var sb strings.Builder

	sb.WriteString("═══════════════════════════════════════════════════════════════════════════════\n")
	sb.WriteString("  GCP GKE Drift Analysis Report\n")
	sb.WriteString("═══════════════════════════════════════════════════════════════════════════════\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n", r.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Total Clusters: %d\n", r.TotalClusters))
	sb.WriteString(fmt.Sprintf("Clusters with Drift: %d\n", r.DriftedClusters))

	if r.TotalClusters > 0 {
		sb.WriteString(fmt.Sprintf("Compliance Rate: %.1f%%\n\n",
			float64(r.TotalClusters-r.DriftedClusters)/float64(r.TotalClusters)*100))
	}

	// Summary by severity
	criticalCount, highCount, mediumCount, lowCount := r.countBySeverity()
	sb.WriteString(report.FormatDriftSummary(criticalCount, highCount, mediumCount, lowCount))

	// Detailed cluster reports
	for i, cluster := range r.Instances {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(cluster.FormatText())
	}

	return sb.String()
}

// countBySeverity tallies the number of drifts by severity level across all clusters
func (r *DriftReport) countBySeverity() (critical, high, medium, low int) {
	for _, cluster := range r.Instances {
		for _, drift := range cluster.Drifts {
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

// FormatText generates a formatted text representation of cluster drift details
func (cd *ClusterDrift) FormatText() string {
	var sb strings.Builder

	sb.WriteString("───────────────────────────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("Cluster:  %s\n", cd.Name))
	sb.WriteString(fmt.Sprintf("Project:  %s\n", cd.Project))
	sb.WriteString(fmt.Sprintf("Location: %s\n", cd.Location))
	sb.WriteString(fmt.Sprintf("Status:   %s\n", cd.Status))

	if len(cd.Labels) > 0 {
		if role, exists := cd.Labels["cluster-role"]; exists {
			sb.WriteString(fmt.Sprintf("Role:     %s\n", role))
		}
	}

	// Show node pools summary
	if len(cd.NodePools) > 0 {
		sb.WriteString(fmt.Sprintf("Node Pools: %d\n", len(cd.NodePools)))
		for _, np := range cd.NodePools {
			sb.WriteString(fmt.Sprintf("  - %s: %s (%d nodes)\n", np.Name, np.MachineType, np.InitialNodeCount))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(report.FormatDrifts(cd.Drifts))

	return sb.String()
}
