package report

import (
	"strings"
	"testing"
)

func TestGetIconForSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		want     string
	}{
		{"critical", "critical", "✗"},
		{"high", "high", "[WARNING]"},
		{"medium", "medium", "●"},
		{"low", "low", "○"},
		{"unknown", "unknown", " "},
		{"empty", "", " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetIconForSeverity(tt.severity); got != tt.want {
				t.Errorf("GetIconForSeverity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCountBySeverity(t *testing.T) {
	tests := []struct {
		name     string
		drifts   []Drift
		wantCrit int
		wantHigh int
		wantMed  int
		wantLow  int
	}{
		{
			name:     "empty",
			drifts:   []Drift{},
			wantCrit: 0,
			wantHigh: 0,
			wantMed:  0,
			wantLow:  0,
		},
		{
			name: "mixed severities",
			drifts: []Drift{
				{Severity: "critical"},
				{Severity: "critical"},
				{Severity: "high"},
				{Severity: "medium"},
				{Severity: "low"},
			},
			wantCrit: 2,
			wantHigh: 1,
			wantMed:  1,
			wantLow:  1,
		},
		{
			name: "all critical",
			drifts: []Drift{
				{Severity: "critical"},
				{Severity: "critical"},
			},
			wantCrit: 2,
			wantHigh: 0,
			wantMed:  0,
			wantLow:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCrit, gotHigh, gotMed, gotLow := CountBySeverity(tt.drifts)
			if gotCrit != tt.wantCrit || gotHigh != tt.wantHigh || gotMed != tt.wantMed || gotLow != tt.wantLow {
				t.Errorf("CountBySeverity() = (%d, %d, %d, %d), want (%d, %d, %d, %d)",
					gotCrit, gotHigh, gotMed, gotLow, tt.wantCrit, tt.wantHigh, tt.wantMed, tt.wantLow)
			}
		})
	}
}

func TestFormatDriftSummary(t *testing.T) {
	tests := []struct {
		name     string
		critical int
		high     int
		medium   int
		low      int
		want     []string
	}{
		{
			name:     "no drifts",
			critical: 0,
			high:     0,
			medium:   0,
			low:      0,
			want:     []string{},
		},
		{
			name:     "all severities",
			critical: 2,
			high:     1,
			medium:   3,
			low:      4,
			want:     []string{"Drift Summary", "CRITICAL: 2", "HIGH:     1", "MEDIUM:   3", "LOW:      4"},
		},
		{
			name:     "only critical",
			critical: 5,
			high:     0,
			medium:   0,
			low:      0,
			want:     []string{"Drift Summary", "CRITICAL: 5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDriftSummary(tt.critical, tt.high, tt.medium, tt.low)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("FormatDriftSummary() missing %q in output:\n%s", want, got)
				}
			}
		})
	}
}

func TestFormatDrifts(t *testing.T) {
	tests := []struct {
		name   string
		drifts []Drift
		want   []string
	}{
		{
			name:   "no drifts",
			drifts: []Drift{},
			want:   []string{"No drift detected"},
		},
		{
			name: "single drift",
			drifts: []Drift{
				{
					Field:    "version",
					Expected: "1.0",
					Actual:   "2.0",
					Severity: "high",
				},
			},
			want: []string{"Detected Drifts: 1", "HIGH", "version", "Expected: 1.0", "Actual:   2.0"},
		},
		{
			name: "multiple drifts",
			drifts: []Drift{
				{Field: "tier", Expected: "db-n1-standard-1", Actual: "db-n1-standard-2", Severity: "critical"},
				{Field: "backup", Expected: "enabled", Actual: "disabled", Severity: "high"},
			},
			want: []string{"Detected Drifts: 2", "CRITICAL", "tier", "HIGH", "backup"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDrifts(tt.drifts)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("FormatDrifts() missing %q in output:\n%s", want, got)
				}
			}
		})
	}
}
