package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jessequinn/drift-analysis-cli/pkg/gcp/sql"
	"github.com/spf13/cobra"
)

var (
	// Direct connection
	inspectHost     string
	inspectPort     int
	
	// Cloud SQL connection
	inspectInstance string
	
	// Common fields
	inspectUser     string
	inspectPassword string
	inspectDatabase string
	inspectOutput   string
	inspectFormat   string
)

// sqlInspectCmd represents the sql inspect command
var sqlInspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect PostgreSQL database schema, roles, and DDL",
	Long: `Connect to a PostgreSQL database and extract detailed schema information including:
- Database metadata (owner, encoding, collation)
- Roles and their privileges
- Tables with columns, constraints, indexes, and ownership
- Views and their definitions
- Extensions
- Generated DDL statements

Supports two connection methods:
1. Cloud SQL connector (recommended): --instance project:region:instance-name
2. Direct connection: --host IP --port 5432

This command requires database connection credentials.`,
	RunE: runSQLInspect,
}

func init() {
	sqlCmd.AddCommand(sqlInspectCmd)
	
	// Cloud SQL connection
	sqlInspectCmd.Flags().StringVarP(&inspectInstance, "instance", "i", "", "Cloud SQL instance connection name (project:region:instance)")
	
	// Direct connection
	sqlInspectCmd.Flags().StringVarP(&inspectHost, "host", "H", "", "database host (for direct connection)")
	sqlInspectCmd.Flags().IntVarP(&inspectPort, "port", "P", 5432, "database port (for direct connection)")
	
	// Common flags
	sqlInspectCmd.Flags().StringVarP(&inspectUser, "user", "u", "", "database user (required)")
	sqlInspectCmd.Flags().StringVarP(&inspectPassword, "password", "p", "", "database password (required)")
	sqlInspectCmd.Flags().StringVarP(&inspectDatabase, "database", "d", "postgres", "database name")
	sqlInspectCmd.Flags().StringVarP(&inspectOutput, "output-file", "o", "", "output file (default: stdout)")
	sqlInspectCmd.Flags().StringVarP(&inspectFormat, "format", "f", "report", "output format (report|ddl)")
	
	sqlInspectCmd.MarkFlagRequired("user")
	sqlInspectCmd.MarkFlagRequired("password")
}

func runSQLInspect(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate: either instance or host must be provided
	if inspectInstance == "" && inspectHost == "" {
		return fmt.Errorf("either --instance (Cloud SQL) or --host (direct) must be specified")
	}
	if inspectInstance != "" && inspectHost != "" {
		return fmt.Errorf("cannot specify both --instance and --host, choose one connection method")
	}

	// Create inspector
	var inspector *sql.DatabaseInspector
	if inspectInstance != "" {
		fmt.Fprintf(os.Stderr, "Connecting to Cloud SQL instance %s as %s...\n", inspectInstance, inspectUser)
		inspector = sql.NewCloudSQLInspector(inspectInstance, inspectUser, inspectPassword, inspectDatabase)
	} else {
		fmt.Fprintf(os.Stderr, "Connecting to %s:%d as %s...\n", inspectHost, inspectPort, inspectUser)
		inspector = sql.NewDatabaseInspector(inspectHost, inspectUser, inspectPassword, inspectDatabase, inspectPort)
	}

	// Inspect database
	schema, err := inspector.InspectDatabase(ctx)
	if err != nil {
		return fmt.Errorf("failed to inspect database: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully extracted schema for database: %s\n\n", schema.DatabaseName)

	// Generate output
	var output string
	switch inspectFormat {
	case "ddl":
		output = schema.GenerateDDL()
	case "report":
		output = schema.FormatSchemaReport()
	default:
		return fmt.Errorf("unsupported format: %s (use 'report' or 'ddl')", inspectFormat)
	}

	// Write output
	if inspectOutput != "" {
		if err := os.WriteFile(inspectOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Output written to: %s\n", inspectOutput)
	} else {
		fmt.Println(output)
	}

	return nil
}
