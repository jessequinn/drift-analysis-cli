package sql

import (
	"testing"
)

func TestValidateSchemaAgainstBaseline_RequiredTables(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "postgres"},
		},
	}

	baseline := &SchemaBaseline{
		RequiredTables: []string{"public.users", "public.orders"},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected for missing required table")
	}

	if len(result.MissingObjects) != 1 {
		t.Fatalf("Expected 1 missing object, got %d", len(result.MissingObjects))
	}

	missing := result.MissingObjects[0]
	if missing.Name != "public.orders" {
		t.Errorf("Expected missing table 'public.orders', got '%s'", missing.Name)
	}
}

func TestValidateSchemaAgainstBaseline_RequiredViews(t *testing.T) {
	schema := &DatabaseSchema{
		Views: []ViewInfo{},
	}

	baseline := &SchemaBaseline{
		RequiredViews: []string{"public.user_summary"},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected for missing required view")
	}

	if len(result.MissingObjects) != 1 {
		t.Fatalf("Expected 1 missing object, got %d", len(result.MissingObjects))
	}
}

func TestValidateSchemaAgainstBaseline_ViewOwnerExceptions(t *testing.T) {
	schema := &DatabaseSchema{
		Views: []ViewInfo{
			{Schema: "public", Name: "user_summary", Owner: "postgres"},
			{Schema: "public", Name: "admin_view", Owner: "admin"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedViewOwner: "postgres",
		ViewOwnerExceptions: map[string]string{
			"public.admin_view": "admin",
		},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if result.HasDrift {
		t.Error("Expected no drift when view ownership exceptions are applied")
	}
}

func TestValidateSchemaAgainstBaseline_SequenceOwnerExceptions(t *testing.T) {
	schema := &DatabaseSchema{
		Sequences: []SequenceInfo{
			{Schema: "public", Name: "users_id_seq", Owner: "postgres"},
			{Schema: "public", Name: "special_seq", Owner: "admin"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedSequenceOwner: "postgres",
		SequenceOwnerExceptions: map[string]string{
			"public.special_seq": "admin",
		},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if result.HasDrift {
		t.Error("Expected no drift when sequence ownership exceptions are applied")
	}
}

func TestValidateSchemaAgainstBaseline_FunctionOwnerExceptions(t *testing.T) {
	schema := &DatabaseSchema{
		Functions: []FunctionInfo{
			{Schema: "public", Name: "calc", Owner: "postgres", Arguments: "integer"},
			{Schema: "public", Name: "admin_func", Owner: "admin", Arguments: "text"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedFunctionOwner: "postgres",
		FunctionOwnerExceptions: map[string]string{
			"public.admin_func(text)": "admin",
		},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if result.HasDrift {
		t.Error("Expected no drift when function ownership exceptions are applied")
	}
}

func TestValidateSchemaAgainstBaseline_ProcedureOwnerExceptions(t *testing.T) {
	schema := &DatabaseSchema{
		Procedures: []ProcedureInfo{
			{Schema: "public", Name: "update_stats", Owner: "postgres", Arguments: ""},
			{Schema: "public", Name: "admin_proc", Owner: "admin", Arguments: ""},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedProcedureOwner: "postgres",
		ProcedureOwnerExceptions: map[string]string{
			"public.admin_proc()": "admin",
		},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if result.HasDrift {
		t.Error("Expected no drift when procedure ownership exceptions are applied")
	}
}

func TestValidateSchemaAgainstBaseline_MultipleViolationTypes(t *testing.T) {
	schema := &DatabaseSchema{
		DatabaseName: "testdb",
		Owner:        "wrong_owner",
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "wrong_owner"},
			{Schema: "public", Name: "temp", Owner: "postgres"},
		},
		Extensions: []Extension{
			{Name: "uuid-ossp", Version: "1.1"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedTables:        intPtr(3),
		ExpectedDatabaseOwner: "cloudsqlsuperuser",
		ExpectedTableOwner:    "postgres",
		RequiredExtensions:    []string{"uuid-ossp", "pg_trgm"},
		ForbiddenTables:       []string{"public.temp"},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift to be detected")
	}

	if len(result.CountMismatches) != 1 {
		t.Errorf("Expected 1 count mismatch, got %d", len(result.CountMismatches))
	}

	if len(result.MissingObjects) != 1 {
		t.Errorf("Expected 1 missing object, got %d", len(result.MissingObjects))
	}

	if len(result.ForbiddenObjects) != 1 {
		t.Errorf("Expected 1 forbidden object, got %d", len(result.ForbiddenObjects))
	}

	if len(result.OwnershipViolations) != 2 {
		t.Errorf("Expected 2 ownership violations, got %d", len(result.OwnershipViolations))
	}
}

func TestValidateSchemaAgainstBaseline_AllCountsMatch(t *testing.T) {
	schema := &DatabaseSchema{
		Tables:     []TableInfo{{Schema: "public", Name: "users", Owner: "postgres"}},
		Views:      []ViewInfo{{Schema: "public", Name: "summary", Owner: "postgres"}},
		Sequences:  []SequenceInfo{{Schema: "public", Name: "seq1", Owner: "postgres"}},
		Functions:  []FunctionInfo{{Schema: "public", Name: "func1", Owner: "postgres"}},
		Procedures: []ProcedureInfo{{Schema: "public", Name: "proc1", Owner: "postgres"}},
		Extensions: []Extension{{Name: "uuid-ossp"}},
		Roles:      []Role{{Name: "postgres"}},
	}

	baseline := &SchemaBaseline{
		ExpectedTables:     intPtr(1),
		ExpectedViews:      intPtr(1),
		ExpectedSequences:  intPtr(1),
		ExpectedFunctions:  intPtr(1),
		ExpectedProcedures: intPtr(1),
		ExpectedExtensions: intPtr(1),
		ExpectedRoles:      intPtr(1),
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if result.HasDrift {
		t.Errorf("Expected no drift when all counts match")
	}

	if len(result.CountMismatches) != 0 {
		t.Errorf("Expected 0 count mismatches, got %d", len(result.CountMismatches))
	}
}

func TestFormatValidationResult_NoDrift(t *testing.T) {
	result := &SchemaValidationResult{
		HasDrift: false,
	}

	output := FormatValidationResult(result)

	expectedOutput := "No schema drift detected - database matches baseline expectations"
	if output != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, output)
	}
}

func TestValidateSchemaAgainstBaseline_EmptySchema(t *testing.T) {
	schema := &DatabaseSchema{
		Tables:     []TableInfo{},
		Views:      []ViewInfo{},
		Sequences:  []SequenceInfo{},
		Functions:  []FunctionInfo{},
		Procedures: []ProcedureInfo{},
		Extensions: []Extension{},
		Roles:      []Role{},
	}

	baseline := &SchemaBaseline{
		ExpectedTables:     intPtr(0),
		ExpectedViews:      intPtr(0),
		ExpectedSequences:  intPtr(0),
		ExpectedFunctions:  intPtr(0),
		ExpectedProcedures: intPtr(0),
		ExpectedExtensions: intPtr(0),
		ExpectedRoles:      intPtr(0),
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if result.HasDrift {
		t.Error("Expected no drift for empty schema matching empty baseline")
	}
}

func TestValidateSchemaAgainstBaseline_OnlyCountsSpecified(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "random_owner"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedTables: intPtr(1),
		// No ownership rules specified
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if result.HasDrift {
		t.Error("Expected no drift when only counts are specified and they match")
	}
}

func TestValidateSchemaAgainstBaseline_ForbiddenOwnerTakesPrecedence(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "forbidden_user"},
		},
	}

	baseline := &SchemaBaseline{
		ExpectedTableOwner: "postgres",
		ForbiddenOwners:    []string{"forbidden_user"},
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

func TestValidateSchemaAgainstBaseline_AllowedOwnersValidation(t *testing.T) {
	schema := &DatabaseSchema{
		Tables: []TableInfo{
			{Schema: "public", Name: "users", Owner: "unauthorized"},
		},
	}

	baseline := &SchemaBaseline{
		AllowedOwners: []string{"postgres", "cloudsqlsuperuser"},
	}

	result := ValidateSchemaAgainstBaseline(schema, baseline)

	if !result.HasDrift {
		t.Error("Expected drift when owner is not in allowed list")
	}

	if len(result.OwnershipViolations) != 1 {
		t.Fatalf("Expected 1 ownership violation, got %d", len(result.OwnershipViolations))
	}
}
