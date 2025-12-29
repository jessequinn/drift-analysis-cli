package sql

import (
	"fmt"
	"strings"
)

// SchemaValidationResult contains the results of schema baseline validation
type SchemaValidationResult struct {
	HasDrift         bool
	CountMismatches  []CountMismatch
	MissingObjects   []MissingObject
	ForbiddenObjects []ForbiddenObject
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
		CountMismatches:  []CountMismatch{},
		MissingObjects:   []MissingObject{},
		ForbiddenObjects: []ForbiddenObject{},
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

	// Determine if there's drift
	result.HasDrift = len(result.CountMismatches) > 0 ||
		len(result.MissingObjects) > 0 ||
		len(result.ForbiddenObjects) > 0

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

	return sb.String()
}
