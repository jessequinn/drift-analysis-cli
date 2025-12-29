package gke

import (
	"context"
	"testing"
)

func TestClusterConfig(t *testing.T) {
	config := ClusterConfig{
		MasterVersion:  "1.27",
		ReleaseChannel: "REGULAR",
		Network:        "default",
		PrivateCluster: true,
	}

	if config.MasterVersion != "1.27" {
		t.Errorf("MasterVersion = %v, want 1.27", config.MasterVersion)
	}
	if config.ReleaseChannel != "REGULAR" {
		t.Errorf("ReleaseChannel = %v, want REGULAR", config.ReleaseChannel)
	}
}

func TestNodePoolConfig(t *testing.T) {
	nodePool := NodePoolConfig{
		Name:             "default-pool",
		MachineType:      "n1-standard-2",
		DiskSizeGB:       100,
		ImageType:        "COS_CONTAINERD",
		InitialNodeCount: 3,
		AutoUpgrade:      true,
		AutoRepair:       true,
	}

	if nodePool.MachineType != "n1-standard-2" {
		t.Errorf("MachineType = %v, want n1-standard-2", nodePool.MachineType)
	}
	if nodePool.DiskSizeGB != 100 {
		t.Errorf("DiskSizeGB = %v, want 100", nodePool.DiskSizeGB)
	}
}

func TestMatchesLabels(t *testing.T) {
	tests := []struct {
		name    string
		cluster *ClusterInstance
		labels  map[string]string
		want    bool
	}{
		{
			name: "exact match",
			cluster: &ClusterInstance{
				Name:   "test-cluster",
				Labels: map[string]string{"env": "prod", "team": "platform"},
			},
			labels: map[string]string{"env": "prod", "team": "platform"},
			want:   true,
		},
		{
			name: "subset match",
			cluster: &ClusterInstance{
				Name:   "test-cluster",
				Labels: map[string]string{"env": "prod", "team": "platform", "region": "us"},
			},
			labels: map[string]string{"env": "prod"},
			want:   true,
		},
		{
			name: "no match",
			cluster: &ClusterInstance{
				Name:   "test-cluster",
				Labels: map[string]string{"env": "dev", "team": "platform"},
			},
			labels: map[string]string{"env": "prod"},
			want:   false,
		},
		{
			name: "empty filter matches all",
			cluster: &ClusterInstance{
				Name:   "test-cluster",
				Labels: map[string]string{"env": "prod"},
			},
			labels: map[string]string{},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesLabels(tt.cluster, tt.labels); got != tt.want {
				t.Errorf("matchesLabels() = %v, want %v", got, tt.want)
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

	clusters := []*ClusterInstance{
		{
			Project:  "test-project",
			Name:     "test-cluster",
			Location: "us-central1",
			Status:   "RUNNING",
			Config: &ClusterConfig{
				MasterVersion:  "1.27.3-gke.100",
				ReleaseChannel: "REGULAR",
				PrivateCluster: true,
			},
			Labels: map[string]string{"env": "test"},
		},
	}

	baseline := &ClusterConfig{
		MasterVersion:  "1.27.3-gke.100",
		ReleaseChannel: "REGULAR",
		PrivateCluster: true,
	}

	report := analyzer.AnalyzeDrift(clusters, baseline, nil)
	if report == nil {
		t.Fatal("Expected non-nil report")
	}

	if len(report.Instances) != 1 {
		t.Errorf("Expected 1 cluster in report, got %d", len(report.Instances))
	}
}

func TestExtractMinorVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "full version string",
			version: "1.27.3-gke.100",
			want:    "1.27",
		},
		{
			name:    "simple version",
			version: "1.27",
			want:    "1.27",
		},
		{
			name:    "single digit",
			version: "1",
			want:    "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMinorVersion(tt.version)
			if got != tt.want {
				t.Errorf("extractMinorVersion(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}
