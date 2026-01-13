package tui

import (
	"fmt"

	"github.com/jessequinn/drift-analysis-cli/pkg/gcp/gce"
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

// FromGCEReport converts a GCE drift report to TUI format
func FromGCEReport(report *gce.DriftReport) ReportData {
	items := make([]DriftItem, 0, len(report.Instances))

	for _, instance := range report.Instances {
		drifts := make([]DriftDetail, 0, len(instance.Drifts))
		for _, d := range instance.Drifts {
			drifts = append(drifts, DriftDetail{
				Field:    d.Field,
				Expected: formatValue(d.Expected),
				Actual:   formatValue(d.Actual),
				Severity: d.Severity,
			})
		}

		items = append(items, DriftItem{
			ResourceType: "GCE Instance",
			Project:      instance.Project,
			Name:         instance.Name,
			Location:     instance.Zone,
			State:        instance.Status,
			Labels:       instance.Labels,
			Drifts:       drifts,
		})
	}

	return ReportData{
		Title:            "GCP Compute Engine Drift Analysis Report",
		Timestamp:        report.Timestamp,
		TotalResources:   report.TotalInstances,
		DriftedResources: report.DriftedInstances,
		Items:            items,
	}
}

// formatValue converts an interface{} to a string representation
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
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
