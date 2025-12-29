package csql

import (
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
