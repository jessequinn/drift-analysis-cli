package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jessequinn/drift-analysis-cli/pkg/gcp/sql"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	dbConnectionName string
	compareWithCache bool
	listConnections  bool
	cacheDir         string
	inspectAll       bool
	outputFormat     string
	outputDir        string
)

// sqlDbCmd represents the database schema inspection command using config
var sqlDbCmd = &cobra.Command{
	Use:   "db",
	Short: "Inspect database schemas using configured connections",
	Long: `Inspect database schemas using connections defined in the config file.
	
This command:
- Uses database_connections from config.yaml
- Inspects tables, views, functions, roles, extensions, etc.
- Caches schemas locally in .drift-cache/database-schemas/
- Compares current schema with cached baseline (with --compare)

Examples:
  # Inspect a database connection (creates/updates cache)
  drift-analysis-cli sql db -config config.yaml -connection cfssl-test

  # Compare current schema with cached baseline
  drift-analysis-cli sql db -config config.yaml -connection cfssl-test --compare

  # List all database connections in config
  drift-analysis-cli sql db -config config.yaml --list`,
	RunE: runSQLDb,
}

func init() {
	sqlCmd.AddCommand(sqlDbCmd)
	
	sqlDbCmd.Flags().StringVarP(&dbConnectionName, "connection", "c", "", "database connection name from config")
	sqlDbCmd.Flags().BoolVar(&compareWithCache, "compare", false, "compare current schema with cached baseline")
	sqlDbCmd.Flags().BoolVar(&listConnections, "list", false, "list all database connections in config")
	sqlDbCmd.Flags().StringVar(&cacheDir, "cache-dir", "", "cache directory (default: .drift-cache/database-schemas)")
	sqlDbCmd.Flags().BoolVar(&inspectAll, "all", false, "inspect all database connections in config")
	sqlDbCmd.Flags().StringVarP(&outputFormat, "format", "f", "summary", "output format: summary|full|ddl|json|yaml")
	sqlDbCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "output directory for generated files (default: current directory)")
}

func runSQLDb(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load config
	if cfgFile == "" {
		return fmt.Errorf("config file is required (use -config flag)")
	}

	configData, err := os.ReadFile(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg sql.Config
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Handle list command
	if listConnections {
		return listDatabaseConnections(&cfg)
	}

	// Handle inspect all connections
	if inspectAll {
		return inspectAllConnections(ctx, &cfg)
	}

	// Validate connection name
	if dbConnectionName == "" {
		return fmt.Errorf("connection name is required (use -connection flag, --all for all connections, or --list to see available)")
	}

	// Find the connection
	var conn *sql.DatabaseConnection
	for i := range cfg.DatabaseConnections {
		if cfg.DatabaseConnections[i].Name == dbConnectionName {
			conn = &cfg.DatabaseConnections[i]
			break
		}
	}

	if conn == nil {
		return fmt.Errorf("connection '%s' not found in config (use --list to see available connections)", dbConnectionName)
	}

	// Validate connection
	if err := conn.Validate(); err != nil {
		return fmt.Errorf("invalid connection config: %w", err)
	}

	// Create cache manager
	cache, err := sql.NewSchemaCache(cacheDir)
	if err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}

	fmt.Printf("Inspecting database connection: %s\n", conn.Name)
	fmt.Printf("  Instance: %s\n", conn.GetConnectionName())
	fmt.Printf("  Database: %s\n", conn.Database)
	fmt.Printf("  Private IP: %v\n\n", conn.UsePrivateIP)

	// Check if cached schema exists
	cacheExists := cache.Exists(conn.GetConnectionName(), conn.Database)
	if cacheExists {
		age, _ := cache.GetAge(conn.GetConnectionName(), conn.Database)
		fmt.Printf("INFO: Cached schema exists (age: %v)\n\n", age.Round(1))
	}

	// Create inspector
	inspector, err := sql.NewInspectorFromDatabaseConnection(conn)
	if err != nil {
		return fmt.Errorf("failed to create inspector: %w", err)
	}

	// Inspect current schema
	fmt.Println("Connecting and inspecting schema...")
	currentSchema, err := inspector.InspectDatabase(ctx)
	if err != nil {
		return fmt.Errorf("failed to inspect database: %w", err)
	}

	fmt.Printf("\nInspection complete!\n")
	fmt.Printf("  Tables: %d\n", len(currentSchema.Tables))
	fmt.Printf("  Views: %d\n", len(currentSchema.Views))
	fmt.Printf("  Roles: %d\n", len(currentSchema.Roles))
	fmt.Printf("  Extensions: %d\n\n", len(currentSchema.Extensions))

	// Validate against baseline if configured
	if conn.SchemaBaseline != nil {
		fmt.Println("Validating against schema baseline...")
		validationResult := sql.ValidateSchemaAgainstBaseline(currentSchema, conn.SchemaBaseline)
		
		if validationResult.HasDrift {
			fmt.Println("\n[WARNING] Schema drift detected!\n")
			fmt.Println(sql.FormatValidationResult(validationResult))
		} else {
			fmt.Println("[OK] Database matches baseline expectations\n")
		}
	}

	// Generate output based on format
	if err := generateOutput(currentSchema, conn.Name, outputFormat, outputDir); err != nil {
		return fmt.Errorf("failed to generate output: %w", err)
	}

	// Compare with cached baseline if requested
	if compareWithCache {
		if !cacheExists {
			fmt.Println("WARNING: No cached baseline found. Creating initial cache...")
			if err := cache.Save(conn.GetConnectionName(), conn.Database, currentSchema); err != nil {
				return fmt.Errorf("failed to save cache: %w", err)
			}
			fmt.Printf("Initial baseline cached to: %s\n", cache.GetCacheDir())
			return nil
		}

		fmt.Println("Comparing with cached baseline...")
		cachedSchema, err := cache.Load(conn.GetConnectionName(), conn.Database)
		if err != nil {
			return fmt.Errorf("failed to load cached schema: %w", err)
		}

		diff := sql.CompareSchemas(cachedSchema.Schema, currentSchema)
		
		if !diff.HasChanges() {
			fmt.Println("\nNo schema changes detected!")
			return nil
		}

		fmt.Println("\nWARNING: Schema changes detected:\n")
		printSchemaDiff(diff)

		// Ask if user wants to update cache
		fmt.Println("\nUpdate cached baseline? (yes/no)")
		var response string
		fmt.Scanln(&response)
		if response == "yes" || response == "y" {
			if err := cache.Save(conn.GetConnectionName(), conn.Database, currentSchema); err != nil {
				return fmt.Errorf("failed to update cache: %w", err)
			}
			fmt.Println("Cache updated")
		}
	} else {
		// Save to cache
		if err := cache.Save(conn.GetConnectionName(), conn.Database, currentSchema); err != nil {
			return fmt.Errorf("failed to save cache: %w", err)
		}
		
		if cacheExists {
			fmt.Println("Cache updated")
		} else {
			fmt.Println("Initial baseline cached")
		}
	}

	return nil
}

func listDatabaseConnections(cfg *sql.Config) error {
	if len(cfg.DatabaseConnections) == 0 {
		fmt.Println("No database connections defined in config")
		return nil
	}

	fmt.Printf("Database connections in config (%d):\n\n", len(cfg.DatabaseConnections))
	for _, conn := range cfg.DatabaseConnections {
		fmt.Printf("  • %s\n", conn.Name)
		fmt.Printf("    Instance: %s\n", conn.GetConnectionName())
		fmt.Printf("    Database: %s\n", conn.Database)
		fmt.Printf("    Username: %s\n", conn.Username)
		fmt.Printf("    Private IP: %v\n", conn.UsePrivateIP)
		fmt.Println()
	}

	return nil
}

func printSchemaDiff(diff *sql.SchemaDiff) {
	if len(diff.AddedTables) > 0 {
		fmt.Printf("Added Tables (%d):\n", len(diff.AddedTables))
		for _, t := range diff.AddedTables {
			fmt.Printf("  + %s.%s (%d columns)\n", t.Schema, t.Name, len(t.Columns))
		}
		fmt.Println()
	}

	if len(diff.DeletedTables) > 0 {
		fmt.Printf("Deleted Tables (%d):\n", len(diff.DeletedTables))
		for _, t := range diff.DeletedTables {
			fmt.Printf("  - %s.%s\n", t.Schema, t.Name)
		}
		fmt.Println()
	}

	if len(diff.ModifiedTables) > 0 {
		fmt.Printf("Modified Tables (%d):\n", len(diff.ModifiedTables))
		for _, t := range diff.ModifiedTables {
			fmt.Printf("  ~ %s.%s\n", t.Schema, t.Name)
		}
		fmt.Println()
	}

	if len(diff.AddedViews) > 0 {
		fmt.Printf("Added Views (%d):\n", len(diff.AddedViews))
		for _, v := range diff.AddedViews {
			fmt.Printf("  + %s.%s\n", v.Schema, v.Name)
		}
		fmt.Println()
	}

	if len(diff.DeletedViews) > 0 {
		fmt.Printf("Deleted Views (%d):\n", len(diff.DeletedViews))
		for _, v := range diff.DeletedViews {
			fmt.Printf("  - %s.%s\n", v.Schema, v.Name)
		}
		fmt.Println()
	}

	if len(diff.AddedRoles) > 0 {
		fmt.Printf("Added Roles (%d):\n", len(diff.AddedRoles))
		for _, r := range diff.AddedRoles {
			fmt.Printf("  + %s\n", r)
		}
		fmt.Println()
	}

	if len(diff.DeletedRoles) > 0 {
		fmt.Printf("Deleted Roles (%d):\n", len(diff.DeletedRoles))
		for _, r := range diff.DeletedRoles {
			fmt.Printf("  - %s\n", r)
		}
		fmt.Println()
	}

	if len(diff.AddedExtensions) > 0 {
		fmt.Printf("Added Extensions (%d):\n", len(diff.AddedExtensions))
		for _, e := range diff.AddedExtensions {
			fmt.Printf("  + %s (%s)\n", e.Name, e.Version)
		}
		fmt.Println()
	}

	if len(diff.DeletedExtensions) > 0 {
		fmt.Printf("Deleted Extensions (%d):\n", len(diff.DeletedExtensions))
		for _, e := range diff.DeletedExtensions {
			fmt.Printf("  - %s\n", e.Name)
		}
		fmt.Println()
	}
}

// inspectAllConnections inspects all configured database connections
func inspectAllConnections(ctx context.Context, cfg *sql.Config) error {
	if len(cfg.DatabaseConnections) == 0 {
		fmt.Println("No database connections defined in config")
		return nil
	}

	fmt.Printf("Inspecting %d database connection(s)...\n\n", len(cfg.DatabaseConnections))

	// Create cache manager
	cache, err := sql.NewSchemaCache(cacheDir)
	if err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}

	for i, conn := range cfg.DatabaseConnections {
		fmt.Printf("[%d/%d] Inspecting: %s\n", i+1, len(cfg.DatabaseConnections), conn.Name)
		fmt.Printf("  Instance: %s\n", conn.GetConnectionName())
		fmt.Printf("  Database: %s\n\n", conn.Database)

		// Validate connection
		if err := conn.Validate(); err != nil {
			fmt.Printf("  ERROR: Invalid connection config: %v\n\n", err)
			continue
		}

		// Create inspector
		inspector, err := sql.NewInspectorFromDatabaseConnection(&conn)
		if err != nil {
			fmt.Printf("  ERROR: Failed to create inspector: %v\n\n", err)
			continue
		}

		// Inspect database
		schema, err := inspector.InspectDatabase(ctx)
		if err != nil {
			fmt.Printf("  ERROR: Failed to inspect database: %v\n\n", err)
			continue
		}

		fmt.Printf("  Inspection complete!\n")
		fmt.Printf("    Tables: %d\n", len(schema.Tables))
		fmt.Printf("    Views: %d\n", len(schema.Views))
		fmt.Printf("    Roles: %d\n", len(schema.Roles))
		fmt.Printf("    Extensions: %d\n", len(schema.Extensions))

		// Validate against baseline if configured
		if conn.SchemaBaseline != nil {
			validationResult := sql.ValidateSchemaAgainstBaseline(schema, conn.SchemaBaseline)
			
			if validationResult.HasDrift {
				fmt.Printf("    [WARNING] Schema drift detected!\n")
				// Print summary only
				if len(validationResult.CountMismatches) > 0 {
					fmt.Printf("      Count mismatches: %d\n", len(validationResult.CountMismatches))
				}
				if len(validationResult.MissingObjects) > 0 {
					fmt.Printf("      Missing objects: %d\n", len(validationResult.MissingObjects))
				}
				if len(validationResult.ForbiddenObjects) > 0 {
					fmt.Printf("      Forbidden objects: %d\n", len(validationResult.ForbiddenObjects))
				}
				if len(validationResult.OwnershipViolations) > 0 {
					fmt.Printf("      Ownership violations: %d\n", len(validationResult.OwnershipViolations))
				}
			} else {
				fmt.Printf("    [OK] Matches baseline\n")
			}
		}

		// Save to cache
		if err := cache.Save(conn.GetConnectionName(), conn.Database, schema); err != nil {
			fmt.Printf("  WARNING: Failed to save cache: %v\n", err)
		}

		// Generate output
		if err := generateOutput(schema, conn.Name, outputFormat, outputDir); err != nil {
			fmt.Printf("  WARNING: Failed to generate output: %v\n", err)
		}

		fmt.Println()
	}

	fmt.Printf("Completed inspecting %d connection(s)\n", len(cfg.DatabaseConnections))
	return nil
}

// generateOutput generates output in the specified format
func generateOutput(schema *sql.DatabaseSchema, connectionName string, format string, outputDir string) error {
	switch format {
	case "summary":
		// Just console output, already done
		return nil

	case "full":
		// Full detailed report
		output := generateFullReport(schema)
		return writeOutput(connectionName, "full-report.txt", output, outputDir)

	case "ddl":
		// DDL statements
		output := schema.GenerateDDL()
		return writeOutput(connectionName, "schema.sql", output, outputDir)

	case "json":
		// JSON format
		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal to JSON: %w", err)
		}
		return writeOutput(connectionName, "schema.json", string(data), outputDir)

	case "yaml":
		// YAML format
		data, err := yaml.Marshal(schema)
		if err != nil {
			return fmt.Errorf("failed to marshal to YAML: %w", err)
		}
		return writeOutput(connectionName, "schema.yaml", string(data), outputDir)

	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// generateFullReport generates a comprehensive text report
func generateFullReport(schema *sql.DatabaseSchema) string {
	var sb strings.Builder

	sb.WriteString(strings.Repeat("=", 80) + "\n")
	sb.WriteString(fmt.Sprintf("DATABASE SCHEMA REPORT: %s\n", schema.DatabaseName))
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")

	// Database info
	sb.WriteString("DATABASE INFORMATION\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString(fmt.Sprintf("Name:      %s\n", schema.DatabaseName))
	sb.WriteString(fmt.Sprintf("Owner:     %s\n", schema.Owner))
	sb.WriteString(fmt.Sprintf("Encoding:  %s\n", schema.Encoding))
	sb.WriteString(fmt.Sprintf("Collation: %s\n", schema.Collation))
	sb.WriteString("\n")

	// Roles
	if len(schema.Roles) > 0 {
		sb.WriteString(fmt.Sprintf("ROLES (%d)\n", len(schema.Roles)))
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		for _, role := range schema.Roles {
			sb.WriteString(fmt.Sprintf("\nRole: %s\n", role.Name))
			sb.WriteString(fmt.Sprintf("  Superuser:      %v\n", role.IsSuperuser))
			sb.WriteString(fmt.Sprintf("  Can Login:      %v\n", role.CanLogin))
			sb.WriteString(fmt.Sprintf("  Can Create DB:  %v\n", role.CanCreateDB))
			sb.WriteString(fmt.Sprintf("  Can Create Role:%v\n", role.CanCreateRole))
			if len(role.MemberOf) > 0 {
				sb.WriteString(fmt.Sprintf("  Member Of:      %v\n", strings.Join(role.MemberOf, ", ")))
			}
		}
		sb.WriteString("\n")
	}

	// Extensions
	if len(schema.Extensions) > 0 {
		sb.WriteString(fmt.Sprintf("EXTENSIONS (%d)\n", len(schema.Extensions)))
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		for _, ext := range schema.Extensions {
			sb.WriteString(fmt.Sprintf("  %-30s Version: %-10s Schema: %s\n", ext.Name, ext.Version, ext.Schema))
		}
		sb.WriteString("\n")
	}

	// Tables
	if len(schema.Tables) > 0 {
		sb.WriteString(fmt.Sprintf("TABLES (%d)\n", len(schema.Tables)))
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		for _, table := range schema.Tables {
			sb.WriteString(fmt.Sprintf("\nTable: %s.%s\n", table.Schema, table.Name))
			sb.WriteString(fmt.Sprintf("  Owner:      %s\n", table.Owner))
			sb.WriteString(fmt.Sprintf("  Rows:       %d (estimated)\n", table.RowCount))
			sb.WriteString(fmt.Sprintf("  Size:       %d bytes\n", table.SizeBytes))
			sb.WriteString(fmt.Sprintf("  Columns:    %d\n", len(table.Columns)))
			
			// Columns
			if len(table.Columns) > 0 {
				sb.WriteString("\n  Columns:\n")
				for _, col := range table.Columns {
					nullable := "NOT NULL"
					if col.IsNullable {
						nullable = "NULL"
					}
					defaultVal := ""
					if col.DefaultValue != nil {
						defaultVal = fmt.Sprintf(" DEFAULT %s", *col.DefaultValue)
					}
					sb.WriteString(fmt.Sprintf("    %-30s %-20s %s%s\n", col.Name, col.DataType, nullable, defaultVal))
				}
			}

			// Indexes
			if len(table.Indexes) > 0 {
				sb.WriteString(fmt.Sprintf("\n  Indexes: %d\n", len(table.Indexes)))
				for _, idx := range table.Indexes {
					idxType := ""
					if idx.IsPrimary {
						idxType = " (PRIMARY KEY)"
					} else if idx.IsUnique {
						idxType = " (UNIQUE)"
					}
					sb.WriteString(fmt.Sprintf("    %s%s on (%s)\n", idx.Name, idxType, strings.Join(idx.Columns, ", ")))
				}
			}

			// Constraints
			if len(table.Constraints) > 0 {
				sb.WriteString(fmt.Sprintf("\n  Constraints: %d\n", len(table.Constraints)))
				for _, cons := range table.Constraints {
					sb.WriteString(fmt.Sprintf("    %s (%s)\n", cons.Name, cons.Type))
				}
			}
		}
		sb.WriteString("\n")
	}

	// Views
	if len(schema.Views) > 0 {
		sb.WriteString(fmt.Sprintf("VIEWS (%d)\n", len(schema.Views)))
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		for _, view := range schema.Views {
			sb.WriteString(fmt.Sprintf("\nView: %s.%s\n", view.Schema, view.Name))
			sb.WriteString(fmt.Sprintf("  Owner: %s\n", view.Owner))
			sb.WriteString(fmt.Sprintf("  Definition:\n%s\n", view.Definition))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(strings.Repeat("=", 80) + "\n")
	sb.WriteString("END OF REPORT\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	return sb.String()
}

// writeOutput writes output to a file
func writeOutput(connectionName string, filename string, content string, outputDir string) error {
	// Sanitize connection name for filename
	safeName := strings.ReplaceAll(connectionName, ":", "_")
	safeName = strings.ReplaceAll(safeName, "/", "_")
	
	// Construct filename with connection name prefix
	baseFilename := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := filepath.Ext(filename)
	fullFilename := fmt.Sprintf("%s-%s%s", safeName, baseFilename, ext)

	// Determine output path
	outputPath := fullFilename
	if outputDir != "" {
		// Create output directory if it doesn't exist
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		outputPath = filepath.Join(outputDir, fullFilename)
	}

	// Write file
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("  Output written to: %s\n", outputPath)
	return nil
}
