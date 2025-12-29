package analyzer

import (
	"context"
)

// ResourceAnalyzer defines the interface for analyzing cloud resources for drift
type ResourceAnalyzer interface {
	// Analyze performs drift analysis on resources across specified projects
	Analyze(ctx context.Context, projects []string) error

	// GenerateReport generates a formatted report of the drift analysis
	GenerateReport() (string, error)

	// GetDriftCount returns the number of drifts detected
	GetDriftCount() int
}

// Baseline defines the interface for baseline configurations
type Baseline interface {
	// GetName returns the name/identifier of the baseline
	GetName() string

	// Validate checks if the baseline configuration is valid
	Validate() error
}
