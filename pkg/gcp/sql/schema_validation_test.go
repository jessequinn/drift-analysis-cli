package sql

import (
	"testing"
)

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

func TestValidateSchemaAgainstBaseline_TableCount(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "postgres"},
			{Schema: "public", Name: "products", Owner: "postgres"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedTables: intPtr(3),
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected")
	}

	if len(result.CountMismatches) != 1 {
		t.Fatalf("Expected 1 count mismatch, got %d", len(result.CountMismatches))
	}

	mismatch := result.CountMismatches[0]
	if mismatch.ObjectType != "Tables" {
		t.Errorf("Expected ObjectType 'Tables', got '%s'", mismatch.ObjectType)
	}
	if mismatch.Expected != 3 {
		t.Errorf("Expected count 3, got %d", mismatch.Expected)
	}
	if mismatch.Actual != 2 {
		t.Errorf("Expected actual count 2, got %d", mismatch.Actual)
	}
}

func TestValidateSchemaAgainstBaseline_ViewCount(t *testing.T) {
	schema := &DatabaseSchema{
		Views: []ViewInfo{
			{Schema: "public", Name: "user_summary", Owner: "postgres"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedViews: intPtr(2),
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected")
	}

	if len(result.CountMismatches) != 1 {
		t.Fatalf("Expected 1 count mismatch, got %d", len(result.CountMismatches))
	}
}

func TestValidateSchemaAgainstBaseline_SequenceCount(t *testing.T) {
	schema := &DatabaseSchema{
		Sequences: []SequenceInfo{
			{Schema: "public", Name: "users_id_seq", Owner: "postgres"},
			{Schema: "public", Name: "products_id_seq", Owner: "postgres"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedSequences: intPtr(3),
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected")
	}

	if len(result.CountMismatches) != 1 {
		t.Fatalf("Expected 1 count mismatch, got %d", len(result.CountMismatches))
	}
}

func TestValidateSchemaAgainstBaseline_FunctionCount(t *testing.T) {
	schema := &DatabaseSchema{
		Functions: []FunctionInfo{
			{Schema: "public", Name: "calculate_total", Owner: "postgres"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedFunctions: intPtr(0),
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected")
	}

	if len(result.CountMismatches) != 1 {
		t.Fatalf("Expected 1 count mismatch, got %d", len(result.CountMismatches))
	}
}

func TestValidateSchemaAgainstBaseline_ProcedureCount(t *testing.T) {
	schema := &DatabaseSchema{
		Procedures: []ProcedureInfo{
			{Schema: "public", Name: "update_stats", Owner: "postgres"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedProcedures: intPtr(2),
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected")
	}

	if len(result.CountMismatches) != 1 {
		t.Fatalf("Expected 1 count mismatch, got %d", len(result.CountMismatches))
	}
}

func TestValidateSchemaAgainstBaseline_RequiredExtensions(t *testing.T) {
	schema := &DatabaseSchema{
		Extensions: []Extension{
			{Name: "uuid-ossp", Version: "1.1"},
		},
	}

	baseline := &SchemaBaseline{
		RequiredExtensions: []string{"uuid-ossp", "pg_trgm"},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected for missing extension")
	}

	if len(result.MissingObjects) != 1 {
		t.Fatalf("Expected 1 missing object, got %d", len(result.MissingObjects))
	}

	missing := result.MissingObjects[0]
	if missing.ObjectType != "Extension" {
		t.Errorf("Expected ObjectType 'Extension', got '%s'", missing.ObjectType)
	}
	if missing.Name != "pg_trgm" {
		t.Errorf("Expected missing extension 'pg_trgm', got '%s'", missing.Name)
	}
}

func TestValidateSchemaAgainstBaseline_ForbiddenTables(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "postgres"},
			{Schema: "public", Name: "temp_debug", Owner: "postgres"},
		},
	}

	baseline := &SchemaBaseline{
		ForbiddenTables: []string{"public.temp_debug"},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected for forbidden table")
	}

	if len(result.ForbiddenObjects) != 1 {
		t.Fatalf("Expected 1 forbidden object, got %d", len(result.ForbiddenObjects))
	}

	forbidden := result.ForbiddenObjects[0]
	if forbidden.ObjectType != "Table" {
		t.Errorf("Expected ObjectType 'Table', got '%s'", forbidden.ObjectType)
	}
	if forbidden.Name != "public.temp_debug" {
		t.Errorf("Expected forbidden table 'public.temp_debug', got '%s'", forbidden.Name)
	}
}

func TestValidateSchemaAgainstBaseline_DatabaseOwnership(t *testing.T) {
	schema := &DatabaseSchema{
		DatabaseName: "testdb",
		Owner:        "postgres",
	}

	baseline := &SchemaBaseline{
		ExpectedDatabaseOwner: "cloudsqlsuperuser",
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected for database ownership")
	}

	if len(result.OwnershipViolations) != 1 {
		t.Fatalf("Expected 1 ownership violation, got %d", len(result.OwnershipViolations))
	}

	violation := result.OwnershipViolations[0]
	if violation.ObjectType != "Database" {
		t.Errorf("Expected ObjectType 'Database', got '%s'", violation.ObjectType)
	}
	if violation.ActualOwner != "postgres" {
		t.Errorf("Expected ActualOwner 'postgres', got '%s'", violation.ActualOwner)
	}
	if violation.ExpectedOwner != "cloudsqlsuperuser" {
		t.Errorf("Expected ExpectedOwner 'cloudsqlsuperuser', got '%s'", violation.ExpectedOwner)
	}
}

func TestValidateSchemaAgainstBaseline_TableOwnership(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "app_user"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedTableOwner: "postgres",
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected for table ownership")
	}

	if len(result.OwnershipViolations) != 1 {
		t.Fatalf("Expected 1 ownership violation, got %d", len(result.OwnershipViolations))
	}

	violation := result.OwnershipViolations[0]
	if violation.ObjectType != "Table" {
		t.Errorf("Expected ObjectType 'Table', got '%s'", violation.ObjectType)
	}
	if violation.ActualOwner != "app_user" {
		t.Errorf("Expected ActualOwner 'app_user', got '%s'", violation.ActualOwner)
	}
}

func TestValidateSchemaAgainstBaseline_ViewOwnership(t *testing.T) {
	schema := &DatabaseSchema{
		Views: []ViewInfo{
			{Schema: "public", Name: "user_summary", Owner: "app_user"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedViewOwner: "postgres",
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected for view ownership")
	}

	if len(result.OwnershipViolations) != 1 {
		t.Fatalf("Expected 1 ownership violation, got %d", len(result.OwnershipViolations))
	}
}

func TestValidateSchemaAgainstBaseline_SequenceOwnership(t *testing.T) {
	schema := &DatabaseSchema{
		Sequences: []SequenceInfo{
			{Schema: "public", Name: "users_id_seq", Owner: "app_user"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedSequenceOwner: "postgres",
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected for sequence ownership")
	}

	if len(result.OwnershipViolations) != 1 {
		t.Fatalf("Expected 1 ownership violation, got %d", len(result.OwnershipViolations))
	}
}

func TestValidateSchemaAgainstBaseline_FunctionOwnership(t *testing.T) {
	schema := &DatabaseSchema{
		Functions: []FunctionInfo{
			{Schema: "public", Name: "calculate_total", Owner: "app_user", Arguments: "integer"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedFunctionOwner: "postgres",
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected for function ownership")
	}

	if len(result.OwnershipViolations) != 1 {
		t.Fatalf("Expected 1 ownership violation, got %d", len(result.OwnershipViolations))
	}
}

func TestValidateSchemaAgainstBaseline_ProcedureOwnership(t *testing.T) {
	schema := &DatabaseSchema{
		Procedures: []ProcedureInfo{
			{Schema: "public", Name: "update_stats", Owner: "app_user", Arguments: ""},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedProcedureOwner: "postgres",
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected for procedure ownership")
	}

	if len(result.OwnershipViolations) != 1 {
		t.Fatalf("Expected 1 ownership violation, got %d", len(result.OwnershipViolations))
	}
}

func TestValidateSchemaAgainstBaseline_AllowedOwners(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "postgres"},
		},
	}

	baseline := &SchemaBaseline{
		AllowedOwners: []string{"postgres", "cloudsqlsuperuser"},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if result.HasDrift {
		t.Error("Expected no drift when owner is in allowed list")
	}
}

func TestValidateSchemaAgainstBaseline_ForbiddenOwners(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "suspicious_user"},
		},
	}

	baseline := &SchemaBaseline{
		ForbiddenOwners: []string{"suspicious_user"},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected for forbidden owner")
	}

	if len(result.OwnershipViolations) != 1 {
		t.Fatalf("Expected 1 ownership violation, got %d", len(result.OwnershipViolations))
	}

	violation := result.OwnershipViolations[0]
	if violation.ViolationType != "forbidden_owner" {
		t.Errorf("Expected ViolationType 'forbidden_owner', got '%s'", violation.ViolationType)
	}
}

func TestValidateSchemaAgainstBaseline_OwnershipExceptions(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "postgres"},
			{Schema: "public", Name: "audit_log", Owner: "admin"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedTableOwner: "postgres",
		TableOwnerExceptions: map[string]string{
			"public.audit_log": "admin",
		},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if result.HasDrift {
		t.Error("Expected no drift when ownership exceptions are applied")
	}
}

func TestValidateSchemaAgainstBaseline_NoDrift(t *testing.T) {
	schema := &DatabaseSchema{
		DatabaseName: "testdb",
		Owner:        "cloudsqlsuperuser",
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "postgres"},
		},
		Views: []ViewInfo{
			{Schema: "public", Name: "user_summary", Owner: "postgres"},
		},
		Sequences: []SequenceInfo{
			{Schema: "public", Name: "users_id_seq", Owner: "postgres"},
		},
		Functions:  []FunctionInfo{},
		Procedures: []ProcedureInfo{},
		Extensions: []Extension{
			{Name: "uuid-ossp", Version: "1.1"},
		},
		Roles: []Role{
			{Name: "postgres"},
			{Name: "app_user"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedTables:        intPtr(1),
		ExpectedViews:         intPtr(1),
		ExpectedSequences:     intPtr(1),
		ExpectedFunctions:     intPtr(0),
		ExpectedProcedures:    intPtr(0),
		ExpectedExtensions:    intPtr(1),
		ExpectedRoles:         intPtr(2),
		ExpectedDatabaseOwner: "cloudsqlsuperuser",
		ExpectedTableOwner:    "postgres",
		ExpectedViewOwner:     "postgres",
		ExpectedSequenceOwner: "postgres",
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if result.HasDrift {
		t.Errorf("Expected no drift, but got: CountMismatches=%d, MissingObjects=%d, ForbiddenObjects=%d, OwnershipViolations=%d",
			len(result.CountMismatches), len(result.MissingObjects), len(result.ForbiddenObjects), len(result.OwnershipViolations))
	}
}

func TestFormatValidationResult(t *testing.T) {
	result := &SchemaValidationResult{
		HasDrift: true,
		CountMismatches: []CountMismatch{
			{ObjectType: "Tables", Expected: 10, Actual: 12},
		},
		MissingObjects: []MissingObject{
			{ObjectType: "Extension", Name: "pg_trgm"},
		},
		ForbiddenObjects: []ForbiddenObject{
			{ObjectType: "Extension", Name: "dblink"},
		},
		OwnershipViolations: []OwnershipViolation{
			{ObjectType: "Table", ObjectName: "public.users", ActualOwner: "app_user", ExpectedOwner: "postgres"},
		},
	}

	output := FormatValidationResult(result)

	if output == "" {
		t.Error("Expected non-empty formatted output")
	}

	// Check that output contains expected content
	expectedStrings := []string{"Count Mismatches", "Missing Required Objects", "Forbidden Objects Found", "Ownership Violations", "Tables", "pg_trgm", "dblink"}
	for _, expected := range expectedStrings {
		found := false
		for i := 0; i <= len(output)-len(expected); i++ {
			if output[i:i+len(expected)] == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected output to contain '%s'", expected)
		}
	}
}

func TestValidateSchemaAgainstBaseline_NilBaseline(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "postgres"},
		},
	}

	result := ValidateSchemaAgainstBaseline(schema, nil)

	if result.HasDrift {
		t.Error("Expected no drift when baseline is nil")
	}
}

func TestValidateSchemaAgainstBaseline_MultipleCountMismatches(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "postgres"},
		},
		Views: []ViewInfo{
			{Schema: "public", Name: "summary", Owner: "postgres"},
		},
		Sequences: []SequenceInfo{
			{Schema: "public", Name: "seq1", Owner: "postgres"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedTables:    intPtr(2),
		ExpectedViews:     intPtr(3),
		ExpectedSequences: intPtr(5),
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected")
	}

	if len(result.CountMismatches) != 3 {
		t.Fatalf("Expected 3 count mismatches, got %d", len(result.CountMismatches))
	}
}
