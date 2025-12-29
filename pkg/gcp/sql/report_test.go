package sql

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
				Timestamp:        timestamp,
				TotalInstances:   2,
				DriftedInstances: 0,
				Instances: []*InstanceDrift{
					{
						Project: "test-project",
						Name:    "test-instance",
						Region:  "us-central1",
						State:   "RUNNABLE",
						Drifts:  []Drift{},
					},
				},
			},
			want: []string{
				"PostgreSQL Drift Analysis Report",
				"Total Instances: 2",
				"Instances with Drift: 0",
				"Compliance Rate: 100.0%",
				"No drift detected",
			},
		},
		{
			name: "with drifts",
			report: &DriftReport{
				Timestamp:        timestamp,
				TotalInstances:   3,
				DriftedInstances: 1,
				Instances: []*InstanceDrift{
					{
						Project: "test-project",
						Name:    "test-instance",
						Region:  "us-central1",
						State:   "RUNNABLE",
						Drifts: []Drift{
							{Field: "tier", Expected: "db-n1-standard-1", Actual: "db-n1-standard-2", Severity: "high"},
							{Field: "backup", Expected: "enabled", Actual: "disabled", Severity: "critical"},
						},
					},
				},
			},
			want: []string{
				"PostgreSQL Drift Analysis Report",
				"Total Instances: 3",
				"Instances with Drift: 1",
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

func TestInstanceDrift_FormatText(t *testing.T) {
	tests := []struct {
		name     string
		instance *InstanceDrift
		want     []string
	}{
		{
			name: "basic instance no drift",
			instance: &InstanceDrift{
				Project: "test-project",
				Name:    "test-instance",
				Region:  "us-central1",
				State:   "RUNNABLE",
				Drifts:  []Drift{},
			},
			want: []string{
				"Instance: test-instance",
				"Project:  test-project",
				"Region:   us-central1",
				"State:    RUNNABLE",
				"No drift detected",
			},
		},
		{
			name: "instance with drifts and role",
			instance: &InstanceDrift{
				Project: "test-project",
				Name:    "app-instance",
				Region:  "us-east1",
				State:   "RUNNABLE",
				Labels:  map[string]string{"database-role": "application"},
				Drifts: []Drift{
					{Field: "tier", Expected: "db-n1-standard-1", Actual: "db-n1-standard-2", Severity: "high"},
				},
				Recommendations: []string{"Resize instance to match baseline"},
			},
			want: []string{
				"Instance: app-instance",
				"Project:  test-project",
				"Region:   us-east1",
				"Role:     application",
				"Detected Drifts: 1",
				"HIGH",
				"tier",
				"Expected: db-n1-standard-1",
				"Actual:   db-n1-standard-2",
				"Recommendations:",
				"Resize instance to match baseline",
			},
		},
		{
			name: "instance with maintenance window",
			instance: &InstanceDrift{
				Project: "test-project",
				Name:    "test-instance",
				Region:  "us-central1",
				State:   "RUNNABLE",
				MaintenanceWindow: &MaintenanceWindow{
					Day:         3,
					Hour:        4,
					UpdateTrack: "stable",
				},
				Drifts: []Drift{},
			},
			want: []string{
				"Instance: test-instance",
				"Maintenance Window: Day 3, Hour 4 UTC (stable)",
				"No drift detected",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.instance.FormatText()
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
				Instances: []*InstanceDrift{
					{Drifts: []Drift{}},
				},
			},
			wantCrit: 0,
			wantHigh: 0,
			wantMed:  0,
			wantLow:  0,
		},
		{
			name: "mixed severities across instances",
			report: &DriftReport{
				Instances: []*InstanceDrift{
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
