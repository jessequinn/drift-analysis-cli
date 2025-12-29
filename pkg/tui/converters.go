package tui

import (
	"github.com/jessequinn/drift-analysis-cli/pkg/gcp/gke"
	"github.com/jessequinn/drift-analysis-cli/pkg/gcp/sql"
)

// FromSQLReport converts a SQL drift report to TUI format
func FromSQLReport(report *sql.DriftReport) ReportData {
	items := make([]DriftItem, 0, len(report.Instances))

	for _, inst := range report.Instances {
		drifts := make([]DriftDetail, 0, len(inst.Drifts))
		for _, d := range inst.Drifts {
			drifts = append(drifts, DriftDetail{
				Field:    d.Field,
				Expected: d.Expected,
				Actual:   d.Actual,
				Severity: d.Severity,
			})
		}

		items = append(items, DriftItem{
			ResourceType: "Cloud SQL",
			Project:      inst.Project,
			Name:         inst.Name,
			Location:     inst.Region,
			State:        inst.State,
			Labels:       inst.Labels,
			Drifts:       drifts,
		})
	}

	return ReportData{
		Title:            "GCP PostgreSQL Drift Analysis Report",
		Timestamp:        report.Timestamp,
		TotalResources:   report.TotalInstances,
		DriftedResources: report.DriftedInstances,
		Items:            items,
	}
}

// FromGKEReport converts a GKE drift report to TUI format
func FromGKEReport(report *gke.DriftReport) ReportData {
	items := make([]DriftItem, 0, len(report.Instances))

	for _, cluster := range report.Instances {
		drifts := make([]DriftDetail, 0, len(cluster.Drifts))
		for _, d := range cluster.Drifts {
			drifts = append(drifts, DriftDetail{
				Field:    d.Field,
				Expected: d.Expected,
				Actual:   d.Actual,
				Severity: d.Severity,
			})
		}

		items = append(items, DriftItem{
			ResourceType: "GKE Cluster",
			Project:      cluster.Project,
			Name:         cluster.Name,
			Location:     cluster.Location,
			State:        cluster.Status,
			Labels:       cluster.Labels,
			Drifts:       drifts,
		})
	}

	return ReportData{
		Title:            "GCP GKE Drift Analysis Report",
		Timestamp:        report.Timestamp,
		TotalResources:   report.TotalClusters,
		DriftedResources: report.DriftedClusters,
		Items:            items,
	}
}
