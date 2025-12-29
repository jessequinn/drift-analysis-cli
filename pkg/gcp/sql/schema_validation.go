package sql

import (
	"fmt"
	"strings"
)

// SchemaValidationResult contains the results of schema baseline validation
type SchemaValidationResult struct {
	HasDrift            bool
	CountMismatches     []CountMismatch
	MissingObjects      []MissingObject
	ForbiddenObjects    []ForbiddenObject
	OwnershipViolations []OwnershipViolation
}

// OwnershipViolation represents an object with incorrect ownership
type OwnershipViolation struct {
	ObjectType     string
	ObjectName     string
	ActualOwner    string
	ExpectedOwner  string
	ViolationType  string // "wrong_owner", "forbidden_owner", "database_owner"
}

// CountMismatch represents a mismatch in expected vs actual counts
type CountMismatch struct {
	ObjectType string
	Expected   int
	Actual     int
}

// MissingObject represents a required object that doesn't exist
type MissingObject struct {
	ObjectType string
	Name       string
}

// ForbiddenObject represents an object that shouldn't exist but does
type ForbiddenObject struct {
	ObjectType string
	Name       string
}

// ValidateSchemaAgainstBaseline validates a database schema against baseline expectations
func ValidateSchemaAgainstBaseline(schema *DatabaseSchema, baseline *SchemaBaseline) *SchemaValidationResult {
	if baseline == nil {
		return &SchemaValidationResult{HasDrift: false}
	}

	result := &SchemaValidationResult{
		CountMismatches:     []CountMismatch{},
		MissingObjects:      []MissingObject{},
		ForbiddenObjects:    []ForbiddenObject{},
		OwnershipViolations: []OwnershipViolation{},
	}

	// Check expected counts
	if baseline.ExpectedTables != nil && *baseline.ExpectedTables != len(schema.Tables) {
		result.CountMismatches = append(result.CountMismatches, CountMismatch{
			ObjectType: "Tables",
			Expected:   *baseline.ExpectedTables,
			Actual:     len(schema.Tables),
		})
	}

	if baseline.ExpectedViews != nil && *baseline.ExpectedViews != len(schema.Views) {
		result.CountMismatches = append(result.CountMismatches, CountMismatch{
			ObjectType: "Views",
			Expected:   *baseline.ExpectedViews,
			Actual:     len(schema.Views),
		})
	}

	if baseline.ExpectedRoles != nil && *baseline.ExpectedRoles != len(schema.Roles) {
		result.CountMismatches = append(result.CountMismatches, CountMismatch{
			ObjectType: "Roles",
			Expected:   *baseline.ExpectedRoles,
			Actual:     len(schema.Roles),
		})
	}

	if baseline.ExpectedExtensions != nil && *baseline.ExpectedExtensions != len(schema.Extensions) {
		result.CountMismatches = append(result.CountMismatches, CountMismatch{
			ObjectType: "Extensions",
			Expected:   *baseline.ExpectedExtensions,
			Actual:     len(schema.Extensions),
		})
	}

	if baseline.ExpectedSequences != nil && *baseline.ExpectedSequences != len(schema.Sequences) {
		result.CountMismatches = append(result.CountMismatches, CountMismatch{
			ObjectType: "Sequences",
			Expected:   *baseline.ExpectedSequences,
			Actual:     len(schema.Sequences),
		})
	}

	if baseline.ExpectedFunctions != nil && *baseline.ExpectedFunctions != len(schema.Functions) {
		result.CountMismatches = append(result.CountMismatches, CountMismatch{
			ObjectType: "Functions",
			Expected:   *baseline.ExpectedFunctions,
			Actual:     len(schema.Functions),
		})
	}

	if baseline.ExpectedProcedures != nil && *baseline.ExpectedProcedures != len(schema.Procedures) {
		result.CountMismatches = append(result.CountMismatches, CountMismatch{
			ObjectType: "Procedures",
			Expected:   *baseline.ExpectedProcedures,
			Actual:     len(schema.Procedures),
		})
	}

	// Check required tables
	tableMap := make(map[string]bool)
	for _, table := range schema.Tables {
		key := fmt.Sprintf("%s.%s", table.Schema, table.Name)
		tableMap[key] = true
		tableMap[table.Name] = true // Also check without schema
	}

	for _, requiredTable := range baseline.RequiredTables {
		if !tableMap[requiredTable] {
			result.MissingObjects = append(result.MissingObjects, MissingObject{
				ObjectType: "Table",
				Name:       requiredTable,
			})
		}
	}

	// Check required views
	viewMap := make(map[string]bool)
	for _, view := range schema.Views {
		key := fmt.Sprintf("%s.%s", view.Schema, view.Name)
		viewMap[key] = true
		viewMap[view.Name] = true
	}

	for _, requiredView := range baseline.RequiredViews {
		if !viewMap[requiredView] {
			result.MissingObjects = append(result.MissingObjects, MissingObject{
				ObjectType: "View",
				Name:       requiredView,
			})
		}
	}

	// Check required extensions
	extMap := make(map[string]bool)
	for _, ext := range schema.Extensions {
		extMap[ext.Name] = true
	}

	for _, requiredExt := range baseline.RequiredExtensions {
		if !extMap[requiredExt] {
			result.MissingObjects = append(result.MissingObjects, MissingObject{
				ObjectType: "Extension",
				Name:       requiredExt,
			})
		}
	}

	// Check forbidden tables
	for _, forbiddenTable := range baseline.ForbiddenTables {
		if tableMap[forbiddenTable] {
			result.ForbiddenObjects = append(result.ForbiddenObjects, ForbiddenObject{
				ObjectType: "Table",
				Name:       forbiddenTable,
			})
		}
	}

	// Check database ownership
	if baseline.ExpectedDatabaseOwner != "" && schema.Owner != baseline.ExpectedDatabaseOwner {
		result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
			ObjectType:     "Database",
			ObjectName:     schema.DatabaseName,
			ActualOwner:    schema.Owner,
			ExpectedOwner:  baseline.ExpectedDatabaseOwner,
			ViolationType:  "database_owner",
		})
	}

	// Check table ownership
	allowedOwnersMap := make(map[string]bool)
	for _, owner := range baseline.AllowedOwners {
		allowedOwnersMap[owner] = true
	}
	
	forbiddenOwnersMap := make(map[string]bool)
	for _, owner := range baseline.ForbiddenOwners {
		forbiddenOwnersMap[owner] = true
	}

	for _, table := range schema.Tables {
		tableName := fmt.Sprintf("%s.%s", table.Schema, table.Name)
		
		// Check for forbidden owners
		if forbiddenOwnersMap[table.Owner] {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "Table",
				ObjectName:     tableName,
				ActualOwner:    table.Owner,
				ExpectedOwner:  "(any non-forbidden owner)",
				ViolationType:  "forbidden_owner",
			})
			continue
		}
		
		// Check specific exception first
		if baseline.TableOwnerExceptions != nil {
			if expectedOwner, hasException := baseline.TableOwnerExceptions[tableName]; hasException {
				if table.Owner != expectedOwner {
					result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
						ObjectType:     "Table",
						ObjectName:     tableName,
						ActualOwner:    table.Owner,
						ExpectedOwner:  expectedOwner,
						ViolationType:  "wrong_owner",
					})
				}
				continue
			}
			// Also check without schema prefix
			if expectedOwner, hasException := baseline.TableOwnerExceptions[table.Name]; hasException {
				if table.Owner != expectedOwner {
					result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
						ObjectType:     "Table",
						ObjectName:     tableName,
						ActualOwner:    table.Owner,
						ExpectedOwner:  expectedOwner,
						ViolationType:  "wrong_owner",
					})
				}
				continue
			}
		}
		
		// Check against expected table owner
		if baseline.ExpectedTableOwner != "" && table.Owner != baseline.ExpectedTableOwner {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "Table",
				ObjectName:     tableName,
				ActualOwner:    table.Owner,
				ExpectedOwner:  baseline.ExpectedTableOwner,
				ViolationType:  "wrong_owner",
			})
		}
		
		// Check against allowed owners (if specified)
		if len(baseline.AllowedOwners) > 0 && !allowedOwnersMap[table.Owner] {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "Table",
				ObjectName:     tableName,
				ActualOwner:    table.Owner,
				ExpectedOwner:  fmt.Sprintf("one of: %v", baseline.AllowedOwners),
				ViolationType:  "wrong_owner",
			})
		}
	}

	// Check view ownership
	for _, view := range schema.Views {
		viewName := fmt.Sprintf("%s.%s", view.Schema, view.Name)
		
		// Check for forbidden owners
		if forbiddenOwnersMap[view.Owner] {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "View",
				ObjectName:     viewName,
				ActualOwner:    view.Owner,
				ExpectedOwner:  "(any non-forbidden owner)",
				ViolationType:  "forbidden_owner",
			})
			continue
		}
		
		// Check specific exception first
		if baseline.ViewOwnerExceptions != nil {
			if expectedOwner, hasException := baseline.ViewOwnerExceptions[viewName]; hasException {
				if view.Owner != expectedOwner {
					result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
						ObjectType:     "View",
						ObjectName:     viewName,
						ActualOwner:    view.Owner,
						ExpectedOwner:  expectedOwner,
						ViolationType:  "wrong_owner",
					})
				}
				continue
			}
			if expectedOwner, hasException := baseline.ViewOwnerExceptions[view.Name]; hasException {
				if view.Owner != expectedOwner {
					result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
						ObjectType:     "View",
						ObjectName:     viewName,
						ActualOwner:    view.Owner,
						ExpectedOwner:  expectedOwner,
						ViolationType:  "wrong_owner",
					})
				}
				continue
			}
		}
		
		// Check against expected view owner
		if baseline.ExpectedViewOwner != "" && view.Owner != baseline.ExpectedViewOwner {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "View",
				ObjectName:     viewName,
				ActualOwner:    view.Owner,
				ExpectedOwner:  baseline.ExpectedViewOwner,
				ViolationType:  "wrong_owner",
			})
		}
		
		// Check against allowed owners (if specified)
		if len(baseline.AllowedOwners) > 0 && !allowedOwnersMap[view.Owner] {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "View",
				ObjectName:     viewName,
				ActualOwner:    view.Owner,
				ExpectedOwner:  fmt.Sprintf("one of: %v", baseline.AllowedOwners),
				ViolationType:  "wrong_owner",
			})
		}
	}

	// Check sequence ownership
	for _, seq := range schema.Sequences {
		seqName := fmt.Sprintf("%s.%s", seq.Schema, seq.Name)
		
		if forbiddenOwnersMap[seq.Owner] {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "Sequence",
				ObjectName:     seqName,
				ActualOwner:    seq.Owner,
				ExpectedOwner:  "(any non-forbidden owner)",
				ViolationType:  "forbidden_owner",
			})
			continue
		}
		
		if baseline.SequenceOwnerExceptions != nil {
			if expectedOwner, hasException := baseline.SequenceOwnerExceptions[seqName]; hasException {
				if seq.Owner != expectedOwner {
					result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
						ObjectType:     "Sequence",
						ObjectName:     seqName,
						ActualOwner:    seq.Owner,
						ExpectedOwner:  expectedOwner,
						ViolationType:  "wrong_owner",
					})
				}
				continue
			}
		}
		
		if baseline.ExpectedSequenceOwner != "" && seq.Owner != baseline.ExpectedSequenceOwner {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "Sequence",
				ObjectName:     seqName,
				ActualOwner:    seq.Owner,
				ExpectedOwner:  baseline.ExpectedSequenceOwner,
				ViolationType:  "wrong_owner",
			})
		}
		
		if len(baseline.AllowedOwners) > 0 && !allowedOwnersMap[seq.Owner] {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "Sequence",
				ObjectName:     seqName,
				ActualOwner:    seq.Owner,
				ExpectedOwner:  fmt.Sprintf("one of: %v", baseline.AllowedOwners),
				ViolationType:  "wrong_owner",
			})
		}
	}

	// Check function ownership
	for _, fn := range schema.Functions {
		fnName := fmt.Sprintf("%s.%s(%s)", fn.Schema, fn.Name, fn.Arguments)
		
		if forbiddenOwnersMap[fn.Owner] {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "Function",
				ObjectName:     fnName,
				ActualOwner:    fn.Owner,
				ExpectedOwner:  "(any non-forbidden owner)",
				ViolationType:  "forbidden_owner",
			})
			continue
		}
		
		if baseline.FunctionOwnerExceptions != nil {
			if expectedOwner, hasException := baseline.FunctionOwnerExceptions[fnName]; hasException {
				if fn.Owner != expectedOwner {
					result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
						ObjectType:     "Function",
						ObjectName:     fnName,
						ActualOwner:    fn.Owner,
						ExpectedOwner:  expectedOwner,
						ViolationType:  "wrong_owner",
					})
				}
				continue
			}
		}
		
		if baseline.ExpectedFunctionOwner != "" && fn.Owner != baseline.ExpectedFunctionOwner {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "Function",
				ObjectName:     fnName,
				ActualOwner:    fn.Owner,
				ExpectedOwner:  baseline.ExpectedFunctionOwner,
				ViolationType:  "wrong_owner",
			})
		}
		
		if len(baseline.AllowedOwners) > 0 && !allowedOwnersMap[fn.Owner] {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "Function",
				ObjectName:     fnName,
				ActualOwner:    fn.Owner,
				ExpectedOwner:  fmt.Sprintf("one of: %v", baseline.AllowedOwners),
				ViolationType:  "wrong_owner",
			})
		}
	}

	// Check procedure ownership
	for _, proc := range schema.Procedures {
		procName := fmt.Sprintf("%s.%s(%s)", proc.Schema, proc.Name, proc.Arguments)
		
		if forbiddenOwnersMap[proc.Owner] {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "Procedure",
				ObjectName:     procName,
				ActualOwner:    proc.Owner,
				ExpectedOwner:  "(any non-forbidden owner)",
				ViolationType:  "forbidden_owner",
			})
			continue
		}
		
		if baseline.ProcedureOwnerExceptions != nil {
			if expectedOwner, hasException := baseline.ProcedureOwnerExceptions[procName]; hasException {
				if proc.Owner != expectedOwner {
					result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
						ObjectType:     "Procedure",
						ObjectName:     procName,
						ActualOwner:    proc.Owner,
						ExpectedOwner:  expectedOwner,
						ViolationType:  "wrong_owner",
					})
				}
				continue
			}
		}
		
		if baseline.ExpectedProcedureOwner != "" && proc.Owner != baseline.ExpectedProcedureOwner {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "Procedure",
				ObjectName:     procName,
				ActualOwner:    proc.Owner,
				ExpectedOwner:  baseline.ExpectedProcedureOwner,
				ViolationType:  "wrong_owner",
			})
		}
		
		if len(baseline.AllowedOwners) > 0 && !allowedOwnersMap[proc.Owner] {
			result.OwnershipViolations = append(result.OwnershipViolations, OwnershipViolation{
				ObjectType:     "Procedure",
				ObjectName:     procName,
				ActualOwner:    proc.Owner,
				ExpectedOwner:  fmt.Sprintf("one of: %v", baseline.AllowedOwners),
				ViolationType:  "wrong_owner",
			})
		}
	}

	// Determine if there's drift
	result.HasDrift = len(result.CountMismatches) > 0 ||
		len(result.MissingObjects) > 0 ||
		len(result.ForbiddenObjects) > 0 ||
		len(result.OwnershipViolations) > 0

	return result
}

// FormatValidationResult formats the validation result as a human-readable string
func FormatValidationResult(result *SchemaValidationResult) string {
	if !result.HasDrift {
		return "No schema drift detected - database matches baseline expectations"
	}

	var sb strings.Builder
	sb.WriteString("SCHEMA DRIFT DETECTED:\n\n")

	if len(result.CountMismatches) > 0 {
		sb.WriteString("Count Mismatches:\n")
		for _, mismatch := range result.CountMismatches {
			sb.WriteString(fmt.Sprintf("  %s: Expected %d, Found %d (diff: %+d)\n",
				mismatch.ObjectType,
				mismatch.Expected,
				mismatch.Actual,
				mismatch.Actual-mismatch.Expected,
			))
		}
		sb.WriteString("\n")
	}

	if len(result.MissingObjects) > 0 {
		sb.WriteString("Missing Required Objects:\n")
		for _, missing := range result.MissingObjects {
			sb.WriteString(fmt.Sprintf("  [MISSING] %s: %s\n", missing.ObjectType, missing.Name))
		}
		sb.WriteString("\n")
	}

	if len(result.ForbiddenObjects) > 0 {
		sb.WriteString("Forbidden Objects Found:\n")
		for _, forbidden := range result.ForbiddenObjects {
			sb.WriteString(fmt.Sprintf("  [ERROR] %s: %s (should not exist)\n", forbidden.ObjectType, forbidden.Name))
		}
		sb.WriteString("\n")
	}

	if len(result.OwnershipViolations) > 0 {
		sb.WriteString("Ownership Violations:\n")
		for _, violation := range result.OwnershipViolations {
			switch violation.ViolationType {
			case "database_owner":
				sb.WriteString(fmt.Sprintf("  [ERROR] %s: %s - Owner: %s, Expected: %s\n",
					violation.ObjectType,
					violation.ObjectName,
					violation.ActualOwner,
					violation.ExpectedOwner,
				))
			case "forbidden_owner":
				sb.WriteString(fmt.Sprintf("  [ERROR] %s: %s - Forbidden owner: %s\n",
					violation.ObjectType,
					violation.ObjectName,
					violation.ActualOwner,
				))
			case "wrong_owner":
				sb.WriteString(fmt.Sprintf("  [WARNING] %s: %s - Owner: %s, Expected: %s\n",
					violation.ObjectType,
					violation.ObjectName,
					violation.ActualOwner,
					violation.ExpectedOwner,
				))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
