package gke

import (
	"strings"
	"testing"
	"time"
)

func TestDriftReport_FormatText(t *testing.T) {
	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	
	tests := []struct {
		name   string
		report *DriftReport
		want   []string
	}{
		{
			name: "no drift",
			report: &DriftReport{
				Timestamp:       timestamp,
				TotalClusters:   2,
				DriftedClusters: 0,
				Instances: []*ClusterDrift{
					{
						Project:  "test-project",
						Name:     "test-cluster",
						Location: "us-central1",
						Status:   "RUNNING",
						Drifts:   []Drift{},
					},
				},
			},
			want: []string{
				"GKE Drift Analysis Report",
				"Total Clusters: 2",
				"Clusters with Drift: 0",
				"Compliance Rate: 100.0%",
				"No drift detected",
			},
		},
		{
			name: "with drifts",
			report: &DriftReport{
				Timestamp:       timestamp,
				TotalClusters:   3,
				DriftedClusters: 1,
				Instances: []*ClusterDrift{
					{
						Project:  "test-project",
						Name:     "test-cluster",
						Location: "us-central1",
						Status:   "RUNNING",
						Drifts: []Drift{
							{Field: "version", Expected: "1.27", Actual: "1.26", Severity: "high"},
							{Field: "network_policy", Expected: "enabled", Actual: "disabled", Severity: "critical"},
						},
					},
				},
			},
			want: []string{
				"GKE Drift Analysis Report",
				"Total Clusters: 3",
				"Clusters with Drift: 1",
				"Compliance Rate: 66.7%",
				"Drift Summary:",
				"CRITICAL: 1",
				"HIGH:     1",
				"Detected Drifts: 2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.report.FormatText()
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("FormatText() missing %q in output:\n%s", want, got)
				}
			}
		})
	}
}

func TestClusterDrift_FormatText(t *testing.T) {
	tests := []struct {
		name    string
		cluster *ClusterDrift
		want    []string
	}{
		{
			name: "basic cluster no drift",
			cluster: &ClusterDrift{
				Project:  "test-project",
				Name:     "test-cluster",
				Location: "us-central1",
				Status:   "RUNNING",
				Drifts:   []Drift{},
			},
			want: []string{
				"Cluster:  test-cluster",
				"Project:  test-project",
				"Location: us-central1",
				"Status:   RUNNING",
				"No drift detected",
			},
		},
		{
			name: "cluster with drifts and node pools",
			cluster: &ClusterDrift{
				Project:  "test-project",
				Name:     "prod-cluster",
				Location: "us-east1",
				Status:   "RUNNING",
				Labels:   map[string]string{"cluster-role": "production"},
				NodePools: []*NodePoolConfig{
					{Name: "default-pool", MachineType: "n1-standard-4", InitialNodeCount: 3},
					{Name: "highmem-pool", MachineType: "n1-highmem-8", InitialNodeCount: 2},
				},
				Drifts: []Drift{
					{Field: "version", Expected: "1.27", Actual: "1.26", Severity: "high"},
				},
			},
			want: []string{
				"Cluster:  prod-cluster",
				"Project:  test-project",
				"Location: us-east1",
				"Role:     production",
				"Node Pools: 2",
				"default-pool: n1-standard-4 (3 nodes)",
				"highmem-pool: n1-highmem-8 (2 nodes)",
				"Detected Drifts: 1",
				"HIGH",
				"version",
				"Expected: 1.27",
				"Actual:   1.26",
			},
		},
		{
			name: "cluster without role label",
			cluster: &ClusterDrift{
				Project:  "test-project",
				Name:     "test-cluster",
				Location: "us-central1",
				Status:   "RUNNING",
				Labels:   map[string]string{"environment": "staging"},
				Drifts:   []Drift{},
			},
			want: []string{
				"Cluster:  test-cluster",
				"Project:  test-project",
				"Location: us-central1",
				"Status:   RUNNING",
				"No drift detected",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cluster.FormatText()
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("FormatText() missing %q in output:\n%s", want, got)
				}
			}
		})
	}
}

func TestDriftReport_countBySeverity(t *testing.T) {
	tests := []struct {
		name     string
		report   *DriftReport
		wantCrit int
		wantHigh int
		wantMed  int
		wantLow  int
	}{
		{
			name: "no drifts",
			report: &DriftReport{
				Instances: []*ClusterDrift{
					{Drifts: []Drift{}},
				},
			},
			wantCrit: 0,
			wantHigh: 0,
			wantMed:  0,
			wantLow:  0,
		},
		{
			name: "mixed severities across clusters",
			report: &DriftReport{
				Instances: []*ClusterDrift{
					{
						Drifts: []Drift{
							{Severity: "critical"},
							{Severity: "high"},
						},
					},
					{
						Drifts: []Drift{
							{Severity: "critical"},
							{Severity: "medium"},
							{Severity: "low"},
						},
					},
				},
			},
			wantCrit: 2,
			wantHigh: 1,
			wantMed:  1,
			wantLow:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCrit, gotHigh, gotMed, gotLow := tt.report.countBySeverity()
			if gotCrit != tt.wantCrit || gotHigh != tt.wantHigh || gotMed != tt.wantMed || gotLow != tt.wantLow {
				t.Errorf("countBySeverity() = (%d, %d, %d, %d), want (%d, %d, %d, %d)",
					gotCrit, gotHigh, gotMed, gotLow, tt.wantCrit, tt.wantHigh, tt.wantMed, tt.wantLow)
			}
		})
	}
}
