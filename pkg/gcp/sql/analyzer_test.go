package sql

import (
	"context"
	"testing"
)

func TestDatabaseConfig(t *testing.T) {
	config := DatabaseConfig{
		DatabaseVersion: "POSTGRES_15",
		Tier:            "db-custom-2-7680",
		DiskSize:        100,
		DiskType:        "PD_SSD",
	}

	if config.DatabaseVersion != "POSTGRES_15" {
		t.Errorf("DatabaseVersion = %v, want POSTGRES_15", config.DatabaseVersion)
	}
	if config.Tier != "db-custom-2-7680" {
		t.Errorf("Tier = %v, want db-custom-2-7680", config.Tier)
	}
}

func TestSettingsConfig(t *testing.T) {
	settings := Settings{
		AvailabilityType:            "REGIONAL",
		BackupEnabled:               true,
		BackupRetentionDays:         7,
		PointInTimeRecovery:         true,
		TransactionLogRetentionDays: 7,
	}

	if settings.AvailabilityType != "REGIONAL" {
		t.Errorf("AvailabilityType = %v, want REGIONAL", settings.AvailabilityType)
	}
	if !settings.BackupEnabled {
		t.Error("BackupEnabled = false, want true")
	}
}

func TestMatchesLabels(t *testing.T) {
	tests := []struct {
		name   string
		inst   *DatabaseInstance
		labels map[string]string
		want   bool
	}{
		{
			name: "exact match",
			inst: &DatabaseInstance{
				Name:   "test-instance",
				Labels: map[string]string{"env": "prod", "team": "backend"},
			},
			labels: map[string]string{"env": "prod", "team": "backend"},
			want:   true,
		},
		{
			name: "subset match",
			inst: &DatabaseInstance{
				Name:   "test-instance",
				Labels: map[string]string{"env": "prod", "team": "backend", "region": "us"},
			},
			labels: map[string]string{"env": "prod"},
			want:   true,
		},
		{
			name: "no match",
			inst: &DatabaseInstance{
				Name:   "test-instance",
				Labels: map[string]string{"env": "dev", "team": "backend"},
			},
			labels: map[string]string{"env": "prod"},
			want:   false,
		},
		{
			name: "empty filter matches all",
			inst: &DatabaseInstance{
				Name:   "test-instance",
				Labels: map[string]string{"env": "prod"},
			},
			labels: map[string]string{},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesLabels(tt.inst, tt.labels); got != tt.want {
				t.Errorf("matchesLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPostgreSQL(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "postgres 15",
			version: "POSTGRES_15",
			want:    true,
		},
		{
			name:    "postgres 14",
			version: "POSTGRES_14",
			want:    true,
		},
		{
			name:    "mysql",
			version: "MYSQL_8_0",
			want:    false,
		},
		{
			name:    "empty",
			version: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPostgreSQL(tt.version)
			if got != tt.want {
				t.Errorf("isPostgreSQL(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestNewAnalyzer(t *testing.T) {
	ctx := context.Background()

	analyzer, err := NewAnalyzer(ctx)
	if err != nil {
		t.Fatalf("NewAnalyzer() error = %v", err)
	}

	if analyzer == nil {
		t.Fatal("Expected non-nil analyzer")
	}
}

func TestAnalyzeDrift(t *testing.T) {
	ctx := context.Background()
	analyzer, err := NewAnalyzer(ctx)
	if err != nil {
		t.Fatalf("NewAnalyzer() error = %v", err)
	}
	defer analyzer.Close()

	instances := []*DatabaseInstance{
		{
			Project:   "test-project",
			Name:      "test-instance",
			Region:    "us-central1",
			State:     "RUNNABLE",
			Databases: []string{"postgres"},
			Config: &DatabaseConfig{
				DatabaseVersion: "POSTGRES_15",
				Tier:            "db-f1-micro",
				DiskSize:        10,
			},
			Labels: map[string]string{"env": "test"},
		},
	}

	baseline := &DatabaseConfig{
		DatabaseVersion: "POSTGRES_15",
		Tier:            "db-f1-micro",
		DiskSize:        10,
	}

	report := analyzer.AnalyzeDrift(instances, baseline)
	if report == nil {
		t.Fatal("Expected non-nil report")
	}

	if len(report.Instances) != 1 {
		t.Errorf("Expected 1 instance in report, got %d", len(report.Instances))
	}
}

func TestAnalyzeInstance(t *testing.T) {
	ctx := context.Background()
	analyzer, err := NewAnalyzer(ctx)
	if err != nil {
		t.Fatalf("NewAnalyzer() error = %v", err)
	}
	defer analyzer.Close()

	inst := &DatabaseInstance{
		Project:   "test-project",
		Name:      "test-instance",
		Region:    "us-central1",
		State:     "RUNNABLE",
		Databases: []string{"postgres"},
		Config: &DatabaseConfig{
			DatabaseVersion: "POSTGRES_15",
			Tier:            "db-f1-micro",
			DiskSize:        10,
		},
		Labels: map[string]string{"env": "test"},
	}

	baseline := &DatabaseConfig{
		DatabaseVersion: "POSTGRES_15",
		Tier:            "db-f1-micro",
		DiskSize:        10,
	}

	drift := analyzer.AnalyzeInstance(inst, baseline)
	if drift == nil {
		t.Fatal("Expected non-nil drift")
	}

	if drift.Name != inst.Name {
		t.Errorf("Name = %v, want %v", drift.Name, inst.Name)
	}
}
