package analyzer

import (
	"context"
	"testing"
)

// MockAnalyzer implements ResourceAnalyzer for testing
type MockAnalyzer struct {
	driftCount int
	report     string
	analyzeErr error
}

func (m *MockAnalyzer) Analyze(ctx context.Context, projects []string) error {
	return m.analyzeErr
}

func (m *MockAnalyzer) GenerateReport() (string, error) {
	return m.report, nil
}

func (m *MockAnalyzer) GetDriftCount() int {
	return m.driftCount
}

// MockBaseline implements Baseline for testing
type MockBaseline struct {
	name        string
	validateErr error
}

func (m *MockBaseline) GetName() string {
	return m.name
}

func (m *MockBaseline) Validate() error {
	return m.validateErr
}

func TestResourceAnalyzer_Interface(t *testing.T) {
	mock := &MockAnalyzer{
		driftCount: 5,
		report:     "Test Report",
	}

	ctx := context.Background()
	err := mock.Analyze(ctx, []string{"project1"})
	if err != nil {
		t.Errorf("Analyze() failed: %v", err)
	}

	report, err := mock.GenerateReport()
	if err != nil {
		t.Errorf("GenerateReport() failed: %v", err)
	}
	if report != "Test Report" {
		t.Errorf("GenerateReport() = %q, want %q", report, "Test Report")
	}

	count := mock.GetDriftCount()
	if count != 5 {
		t.Errorf("GetDriftCount() = %d, want %d", count, 5)
	}
}

func TestBaseline_Interface(t *testing.T) {
	mock := &MockBaseline{
		name: "test-baseline",
	}

	name := mock.GetName()
	if name != "test-baseline" {
		t.Errorf("GetName() = %q, want %q", name, "test-baseline")
	}

	err := mock.Validate()
	if err != nil {
		t.Errorf("Validate() failed: %v", err)
	}
}
