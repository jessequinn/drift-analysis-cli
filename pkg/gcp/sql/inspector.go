package sql

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"sort"
	"strings"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
)

// DatabaseInspector connects to PostgreSQL instances and extracts detailed information
type DatabaseInspector struct {
	useCloudSQLConnector bool
	instanceConnectionName string // project:region:instance for Cloud SQL
	user                 string
	password             string
	database             string
	usePrivateIP         bool   // whether to use private IP for Cloud SQL
	proxyManager         *ProxyManager // manages Cloud SQL Proxy process
	sshTunnel            *SSHTunnelManager // manages SSH tunnel through bastion
	
	// Direct connection fields
	connectionString string
}

// InspectorConfig holds configuration for creating an inspector
type InspectorConfig struct {
	// Cloud SQL connection (recommended)
	UseCloudSQL            bool
	InstanceConnectionName string // format: project:region:instance
	UsePrivateIP           bool
	UseProxy               bool // if true, starts Cloud SQL Proxy in background
	UseGcloudProxy         bool // if true, uses gcloud instead of cloud-sql-proxy binary
	ProxyPort              int  // local port for proxy (default: 5432)
	
	// Direct connection (alternative)
	Host     string
	Port     int
	
	// Common fields
	User     string
	Password string
	Database string
}

// DatabaseSchema contains detailed schema information
type DatabaseSchema struct {
	DatabaseName string
	Owner        string
	Encoding     string
	Collation    string
	Roles        []Role
	Tables       []TableInfo
	Views        []ViewInfo
	Sequences    []SequenceInfo
	Functions    []FunctionInfo
	Procedures   []ProcedureInfo
	Extensions   []Extension
}

// Role represents a PostgreSQL role/user
type Role struct {
	Name       string
	IsSuperuser bool
	CanLogin    bool
	CanCreateDB bool
	CanCreateRole bool
	MemberOf    []string
}

// TableInfo contains table metadata
type TableInfo struct {
	Schema      string
	Name        string
	Owner       string
	RowCount    int64
	SizeBytes   int64
	Columns     []ColumnInfo
	Constraints []ConstraintInfo
	Indexes     []IndexInfo
}

// ColumnInfo contains column metadata
type ColumnInfo struct {
	Name         string
	DataType     string
	IsNullable   bool
	DefaultValue *string
	IsIdentity   bool
}

// ConstraintInfo contains constraint metadata
type ConstraintInfo struct {
	Name       string
	Type       string // PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK
	Definition string
}

// IndexInfo contains index metadata
type IndexInfo struct {
	Name       string
	Columns    []string
	IsUnique   bool
	IsPrimary  bool
	Definition string
}

// ViewInfo contains view metadata
type ViewInfo struct {
	Schema     string
	Name       string
	Owner      string
	Definition string
}

// SequenceInfo contains sequence metadata
type SequenceInfo struct {
	Schema    string
	Name      string
	Owner     string
	DataType  string
	StartValue int64
	MinValue  *int64
	MaxValue  *int64
	Increment int64
}

// FunctionInfo contains function metadata
type FunctionInfo struct {
	Schema     string
	Name       string
	Owner      string
	Language   string
	ReturnType string
	Arguments  string
	Definition string
}

// ProcedureInfo contains procedure metadata
type ProcedureInfo struct {
	Schema     string
	Name       string
	Owner      string
	Language   string
	Arguments  string
	Definition string
}

// Extension represents a PostgreSQL extension
type Extension struct {
	Name    string
	Version string
	Schema  string
}

// NewDatabaseInspector creates a new database inspector with direct connection
func NewDatabaseInspector(host, user, password, database string, port int) *DatabaseInspector {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		host, port, user, password, database)
	return &DatabaseInspector{
		connectionString: connStr,
		useCloudSQLConnector: false,
	}
}

// NewCloudSQLInspector creates a new database inspector using Cloud SQL connector
func NewCloudSQLInspector(instanceConnectionName, user, password, database string) *DatabaseInspector {
	return &DatabaseInspector{
		useCloudSQLConnector: true,
		instanceConnectionName: instanceConnectionName,
		user:     user,
		password: password,
		database: database,
	}
}

// NewInspectorFromConnectionConfig creates a new database inspector from ConnectionConfig
func NewInspectorFromConnectionConfig(config *ConnectionConfig) (*DatabaseInspector, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid connection config: %w", err)
	}
	
	connName := config.GetConnectionName()
	
	// For private IP, we need to use the proxy approach
	if config.UsePrivateIP {
		return NewInspectorWithProxy(connName, config.Username, config.Password, config.Database, config.UsePrivateIP)
	}
	
	return &DatabaseInspector{
		useCloudSQLConnector:   true,
		instanceConnectionName: connName,
		user:                   config.Username,
		password:               config.Password,
		database:               config.Database,
		usePrivateIP:           config.UsePrivateIP,
	}, nil
}

// NewInspectorFromDatabaseConnection creates a new database inspector from DatabaseConnection
func NewInspectorFromDatabaseConnection(conn *DatabaseConnection) (*DatabaseInspector, error) {
	if err := conn.Validate(); err != nil {
		return nil, fmt.Errorf("invalid connection config: %w", err)
	}
	
	// Check if SSH tunnel is configured
	if conn.SSHTunnel != nil && conn.SSHTunnel.Enabled {
		return NewInspectorWithSSHTunnel(conn)
	}
	
	// Otherwise use the standard connection config path
	return NewInspectorFromConnectionConfig(conn.ToConnectionConfig())
}

// NewInspectorWithSSHTunnel creates a new inspector that uses SSH tunnel through bastion
func NewInspectorWithSSHTunnel(conn *DatabaseConnection) (*DatabaseInspector, error) {
	// Create SSH tunnel manager
	sshTunnel, err := NewSSHTunnelManager(conn.SSHTunnel)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH tunnel manager: %w", err)
	}
	
	// Connection will go through the SSH tunnel
	// The tunnel manager will provide the connection string
	return &DatabaseInspector{
		useCloudSQLConnector:   false,
		instanceConnectionName: conn.GetConnectionName(),
		user:                   conn.Username,
		password:               conn.Password,
		database:               conn.Database,
		usePrivateIP:           true,
		sshTunnel:              sshTunnel,
		connectionString:       "", // Will be set when tunnel is established
	}, nil
}

// NewInspectorWithProxy creates a new inspector that manages a proxy process
func NewInspectorWithProxy(instanceConnectionName, user, password, database string, usePrivateIP bool) (*DatabaseInspector, error) {
	// Create proxy manager - use cloud-sql-proxy binary instead of gcloud
	proxyConfig := ProxyConfig{
		InstanceConnectionName: instanceConnectionName,
		LocalPort:              5432,
		UsePrivateIP:           usePrivateIP,
		UseGcloud:              false, // Use cloud-sql-proxy binary
	}
	
	proxyManager := NewProxyManager(proxyConfig)
	
	// Create direct connection string to localhost (proxy will handle the tunnel)
	// Increase timeouts for Cloud SQL proxy connections
	connStr := fmt.Sprintf("host=localhost port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=60 statement_timeout=60000",
		proxyConfig.LocalPort, user, password, database)
	
	return &DatabaseInspector{
		useCloudSQLConnector:   false, // Use direct connection to proxy
		instanceConnectionName: instanceConnectionName,
		user:                   user,
		password:               password,
		database:               database,
		usePrivateIP:           usePrivateIP,
		proxyManager:           proxyManager,
		connectionString:       connStr,
	}, nil
}

// InspectDatabase connects and extracts detailed schema information
func (di *DatabaseInspector) InspectDatabase(ctx context.Context) (*DatabaseSchema, error) {
	// Start SSH tunnel if configured
	if di.sshTunnel != nil {
		fmt.Printf("Starting SSH tunnel for %s...\n", di.instanceConnectionName)
		if err := di.sshTunnel.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start SSH tunnel: %w", err)
		}
		defer func() {
			fmt.Println("Stopping SSH tunnel...")
			if err := di.sshTunnel.Stop(); err != nil {
				fmt.Printf("Warning: failed to stop SSH tunnel: %v\n", err)
			}
		}()
		fmt.Println("SSH tunnel established successfully")
		
		// Set connection string to use the tunnel
		di.connectionString = di.sshTunnel.GetConnectionString(di.user, di.password, di.database)
	}
	
	// Start proxy if configured
	if di.proxyManager != nil {
		fmt.Printf("Starting Cloud SQL Proxy for %s...\n", di.instanceConnectionName)
		if err := di.proxyManager.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start proxy: %w", err)
		}
		defer func() {
			fmt.Println("Stopping Cloud SQL Proxy...")
			if err := di.proxyManager.Stop(); err != nil {
				fmt.Printf("Warning: failed to stop proxy: %v\n", err)
			}
		}()
		fmt.Println("Proxy started successfully")
	}
	
	var db *sql.DB
	var cleanup func() error
	var err error

	if di.useCloudSQLConnector {
		db, cleanup, err = di.connectWithCloudSQL(ctx)
	} else {
		db, cleanup, err = di.connectDirect(ctx)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer cleanup()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	schema := &DatabaseSchema{}

	// Get database info
	if err := di.getDatabaseInfo(ctx, db, schema); err != nil {
		return nil, fmt.Errorf("failed to get database info: %w", err)
	}

	// Get roles
	if err := di.getRoles(ctx, db, schema); err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}

	// Get extensions
	if err := di.getExtensions(ctx, db, schema); err != nil {
		return nil, fmt.Errorf("failed to get extensions: %w", err)
	}

	// Get tables
	if err := di.getTables(ctx, db, schema); err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	// Get views
	if err := di.getViews(ctx, db, schema); err != nil {
		return nil, fmt.Errorf("failed to get views: %w", err)
	}

	// Get sequences
	if err := di.getSequences(ctx, db, schema); err != nil {
		return nil, fmt.Errorf("failed to get sequences: %w", err)
	}

	// Get functions
	if err := di.getFunctions(ctx, db, schema); err != nil {
		return nil, fmt.Errorf("failed to get functions: %w", err)
	}

	// Get procedures
	if err := di.getProcedures(ctx, db, schema); err != nil {
		return nil, fmt.Errorf("failed to get procedures: %w", err)
	}

	return schema, nil
}

// connectWithCloudSQL establishes connection using Cloud SQL connector
func (di *DatabaseInspector) connectWithCloudSQL(ctx context.Context) (*sql.DB, func() error, error) {
	// Create dialer with optional private IP support
	var dialerOpts []cloudsqlconn.Option
	if di.usePrivateIP {
		dialerOpts = append(dialerOpts, cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPrivateIP()))
	}
	
	d, err := cloudsqlconn.NewDialer(ctx, dialerOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dialer: %w", err)
	}

	// Cleanup function
	cleanup := func() error {
		return d.Close()
	}

	// Create pgx connection config
	connConfig, err := pgx.ParseConfig(fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
		di.user, di.password, di.database))
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set up Cloud SQL dialer
	connConfig.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return d.Dial(ctx, di.instanceConnectionName)
	}

	// Register config and get connection string
	connStr := stdlib.RegisterConnConfig(connConfig)
	
	// Open database
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	fullCleanup := func() error {
		dbErr := db.Close()
		dialerErr := cleanup()
		if dbErr != nil {
			return dbErr
		}
		return dialerErr
	}

	return db, fullCleanup, nil
}

// connectDirect establishes direct PostgreSQL connection
func (di *DatabaseInspector) connectDirect(ctx context.Context) (*sql.DB, func() error, error) {
	db, err := sql.Open("postgres", di.connectionString)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open connection: %w", err)
	}

	cleanup := func() error {
		return db.Close()
	}

	return db, cleanup, nil
}

// getDatabaseInfo retrieves basic database information
func (di *DatabaseInspector) getDatabaseInfo(ctx context.Context, db *sql.DB, schema *DatabaseSchema) error {
	query := `
		SELECT 
			current_database(),
			pg_catalog.pg_get_userbyid(d.datdba) as owner,
			pg_encoding_to_char(d.encoding) as encoding,
			d.datcollate as collation
		FROM pg_catalog.pg_database d
		WHERE d.datname = current_database()
	`
	return db.QueryRowContext(ctx, query).Scan(
		&schema.DatabaseName,
		&schema.Owner,
		&schema.Encoding,
		&schema.Collation,
	)
}

// getRoles retrieves all roles and their properties
func (di *DatabaseInspector) getRoles(ctx context.Context, db *sql.DB, schema *DatabaseSchema) error {
	query := `
		SELECT 
			r.rolname,
			r.rolsuper,
			r.rolcanlogin,
			r.rolcreatedb,
			r.rolcreaterole,
			COALESCE(
				ARRAY_AGG(m.rolname) FILTER (WHERE m.rolname IS NOT NULL),
				ARRAY[]::text[]
			) as member_of
		FROM pg_catalog.pg_roles r
		LEFT JOIN pg_catalog.pg_auth_members am ON r.oid = am.member
		LEFT JOIN pg_catalog.pg_roles m ON am.roleid = m.oid
		WHERE r.rolname NOT LIKE 'pg_%'
		  AND r.rolname NOT LIKE 'cloudsql%'
		GROUP BY r.rolname, r.rolsuper, r.rolcanlogin, r.rolcreatedb, r.rolcreaterole
		ORDER BY r.rolname
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var role Role
		var memberOf []string
		err := rows.Scan(
			&role.Name,
			&role.IsSuperuser,
			&role.CanLogin,
			&role.CanCreateDB,
			&role.CanCreateRole,
			(*StringArray)(&memberOf),
		)
		if err != nil {
			return err
		}
		role.MemberOf = memberOf
		schema.Roles = append(schema.Roles, role)
	}

	return rows.Err()
}

// getExtensions retrieves installed extensions
func (di *DatabaseInspector) getExtensions(ctx context.Context, db *sql.DB, schema *DatabaseSchema) error {
	query := `
		SELECT 
			extname,
			extversion,
			n.nspname as schema
		FROM pg_catalog.pg_extension e
		JOIN pg_catalog.pg_namespace n ON e.extnamespace = n.oid
		ORDER BY extname
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var ext Extension
		if err := rows.Scan(&ext.Name, &ext.Version, &ext.Schema); err != nil {
			return err
		}
		schema.Extensions = append(schema.Extensions, ext)
	}

	return rows.Err()
}

// getTables retrieves all user tables with detailed information
func (di *DatabaseInspector) getTables(ctx context.Context, db *sql.DB, schema *DatabaseSchema) error {
	query := `
		SELECT 
			schemaname,
			tablename,
			tableowner
		FROM pg_catalog.pg_tables
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY schemaname, tablename
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var table TableInfo
		if err := rows.Scan(&table.Schema, &table.Name, &table.Owner); err != nil {
			return err
		}

		// Get row count and size
		if err := di.getTableStats(ctx, db, &table); err != nil {
			// Log but don't fail - stats might not be available
			table.RowCount = -1
			table.SizeBytes = -1
		}

		// Get columns
		if err := di.getTableColumns(ctx, db, &table); err != nil {
			return fmt.Errorf("failed to get columns for %s.%s: %w", table.Schema, table.Name, err)
		}

		// Get constraints
		if err := di.getTableConstraints(ctx, db, &table); err != nil {
			return fmt.Errorf("failed to get constraints for %s.%s: %w", table.Schema, table.Name, err)
		}

		// Get indexes
		if err := di.getTableIndexes(ctx, db, &table); err != nil {
			return fmt.Errorf("failed to get indexes for %s.%s: %w", table.Schema, table.Name, err)
		}

		schema.Tables = append(schema.Tables, table)
	}

	return rows.Err()
}

// getTableStats retrieves row count and size
func (di *DatabaseInspector) getTableStats(ctx context.Context, db *sql.DB, table *TableInfo) error {
	query := `
		SELECT 
			COALESCE(n_live_tup, 0) as row_count,
			pg_total_relation_size(quote_ident($1) || '.' || quote_ident($2))::bigint as size_bytes
		FROM pg_stat_user_tables
		WHERE schemaname = $1 AND relname = $2
	`
	return db.QueryRowContext(ctx, query, table.Schema, table.Name).Scan(
		&table.RowCount,
		&table.SizeBytes,
	)
}

// getTableColumns retrieves column information
func (di *DatabaseInspector) getTableColumns(ctx context.Context, db *sql.DB, table *TableInfo) error {
	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable = 'YES' as is_nullable,
			column_default,
			is_identity = 'YES' as is_identity
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := db.QueryContext(ctx, query, table.Schema, table.Name)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		if err := rows.Scan(&col.Name, &col.DataType, &col.IsNullable, &col.DefaultValue, &col.IsIdentity); err != nil {
			return err
		}
		table.Columns = append(table.Columns, col)
	}

	return rows.Err()
}

// getTableConstraints retrieves constraint information
func (di *DatabaseInspector) getTableConstraints(ctx context.Context, db *sql.DB, table *TableInfo) error {
	query := `
		SELECT 
			con.conname as constraint_name,
			CASE con.contype
				WHEN 'p' THEN 'PRIMARY KEY'
				WHEN 'f' THEN 'FOREIGN KEY'
				WHEN 'u' THEN 'UNIQUE'
				WHEN 'c' THEN 'CHECK'
			END as constraint_type,
			pg_get_constraintdef(con.oid) as definition
		FROM pg_catalog.pg_constraint con
		JOIN pg_catalog.pg_class rel ON con.conrelid = rel.oid
		JOIN pg_catalog.pg_namespace nsp ON rel.relnamespace = nsp.oid
		WHERE nsp.nspname = $1 AND rel.relname = $2
		ORDER BY con.conname
	`

	rows, err := db.QueryContext(ctx, query, table.Schema, table.Name)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var constraint ConstraintInfo
		if err := rows.Scan(&constraint.Name, &constraint.Type, &constraint.Definition); err != nil {
			return err
		}
		table.Constraints = append(table.Constraints, constraint)
	}

	return rows.Err()
}

// getTableIndexes retrieves index information
func (di *DatabaseInspector) getTableIndexes(ctx context.Context, db *sql.DB, table *TableInfo) error {
	query := `
		SELECT 
			i.relname as index_name,
			ix.indisunique as is_unique,
			ix.indisprimary as is_primary,
			pg_get_indexdef(ix.indexrelid) as definition,
			ARRAY_AGG(a.attname ORDER BY array_position(ix.indkey, a.attnum)) as columns
		FROM pg_catalog.pg_index ix
		JOIN pg_catalog.pg_class i ON ix.indexrelid = i.oid
		JOIN pg_catalog.pg_class t ON ix.indrelid = t.oid
		JOIN pg_catalog.pg_namespace n ON t.relnamespace = n.oid
		JOIN pg_catalog.pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = $1 AND t.relname = $2
		GROUP BY i.relname, ix.indisunique, ix.indisprimary, ix.indexrelid
		ORDER BY i.relname
	`

	rows, err := db.QueryContext(ctx, query, table.Schema, table.Name)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var index IndexInfo
		var columns []string
		if err := rows.Scan(&index.Name, &index.IsUnique, &index.IsPrimary, &index.Definition, (*StringArray)(&columns)); err != nil {
			return err
		}
		index.Columns = columns
		table.Indexes = append(table.Indexes, index)
	}

	return rows.Err()
}

// getViews retrieves view information
func (di *DatabaseInspector) getViews(ctx context.Context, db *sql.DB, schema *DatabaseSchema) error {
	query := `
		SELECT 
			schemaname,
			viewname,
			viewowner,
			definition
		FROM pg_catalog.pg_views
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY schemaname, viewname
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var view ViewInfo
		if err := rows.Scan(&view.Schema, &view.Name, &view.Owner, &view.Definition); err != nil {
			return err
		}
		schema.Views = append(schema.Views, view)
	}

	return rows.Err()
}

func (di *DatabaseInspector) getSequences(ctx context.Context, db *sql.DB, schema *DatabaseSchema) error {
	query := `
		SELECT 
			schemaname,
			sequencename,
			sequenceowner
		FROM pg_catalog.pg_sequences
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY schemaname, sequencename
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var seq SequenceInfo
		if err := rows.Scan(&seq.Schema, &seq.Name, &seq.Owner); err != nil {
			return err
		}
		schema.Sequences = append(schema.Sequences, seq)
	}

	return rows.Err()
}

func (di *DatabaseInspector) getFunctions(ctx context.Context, db *sql.DB, schema *DatabaseSchema) error {
	query := `
		SELECT 
			n.nspname as schema,
			p.proname as name,
			pg_catalog.pg_get_userbyid(p.proowner) as owner,
			l.lanname as language,
			pg_catalog.pg_get_function_result(p.oid) as return_type,
			pg_catalog.pg_get_function_arguments(p.oid) as arguments
		FROM pg_catalog.pg_proc p
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		LEFT JOIN pg_catalog.pg_language l ON l.oid = p.prolang
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		  AND p.prokind = 'f'  -- functions only (not procedures)
		ORDER BY n.nspname, p.proname
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var fn FunctionInfo
		if err := rows.Scan(&fn.Schema, &fn.Name, &fn.Owner, &fn.Language, &fn.ReturnType, &fn.Arguments); err != nil {
			return err
		}
		schema.Functions = append(schema.Functions, fn)
	}

	return rows.Err()
}

func (di *DatabaseInspector) getProcedures(ctx context.Context, db *sql.DB, schema *DatabaseSchema) error {
	query := `
		SELECT 
			n.nspname as schema,
			p.proname as name,
			pg_catalog.pg_get_userbyid(p.proowner) as owner,
			l.lanname as language,
			pg_catalog.pg_get_function_arguments(p.oid) as arguments
		FROM pg_catalog.pg_proc p
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		LEFT JOIN pg_catalog.pg_language l ON l.oid = p.prolang
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		  AND p.prokind = 'p'  -- procedures only
		ORDER BY n.nspname, p.proname
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var proc ProcedureInfo
		if err := rows.Scan(&proc.Schema, &proc.Name, &proc.Owner, &proc.Language, &proc.Arguments); err != nil {
			return err
		}
		schema.Procedures = append(schema.Procedures, proc)
	}

	return rows.Err()
}

// GenerateDDL generates DDL statements from the schema
func (schema *DatabaseSchema) GenerateDDL() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("-- Database: %s\n", schema.DatabaseName))
	sb.WriteString(fmt.Sprintf("-- Owner: %s\n", schema.Owner))
	sb.WriteString(fmt.Sprintf("-- Encoding: %s\n", schema.Encoding))
	sb.WriteString(fmt.Sprintf("-- Collation: %s\n\n", schema.Collation))

	// Extensions
	if len(schema.Extensions) > 0 {
		sb.WriteString("-- Extensions\n")
		for _, ext := range schema.Extensions {
			sb.WriteString(fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s WITH SCHEMA %s;\n",
				ext.Name, ext.Schema))
		}
		sb.WriteString("\n")
	}

	// Roles
	if len(schema.Roles) > 0 {
		sb.WriteString("-- Roles\n")
		for _, role := range schema.Roles {
			attrs := []string{}
			if role.IsSuperuser {
				attrs = append(attrs, "SUPERUSER")
			}
			if role.CanLogin {
				attrs = append(attrs, "LOGIN")
			}
			if role.CanCreateDB {
				attrs = append(attrs, "CREATEDB")
			}
			if role.CanCreateRole {
				attrs = append(attrs, "CREATEROLE")
			}
			sb.WriteString(fmt.Sprintf("CREATE ROLE %s", role.Name))
			if len(attrs) > 0 {
				sb.WriteString(" WITH " + strings.Join(attrs, " "))
			}
			sb.WriteString(";\n")
		}
		sb.WriteString("\n")
	}

	// Tables
	for _, table := range schema.Tables {
		sb.WriteString(fmt.Sprintf("-- Table: %s.%s\n", table.Schema, table.Name))
		sb.WriteString(fmt.Sprintf("-- Owner: %s\n", table.Owner))
		if table.RowCount >= 0 {
			sb.WriteString(fmt.Sprintf("-- Rows: %d\n", table.RowCount))
		}
		sb.WriteString(fmt.Sprintf("CREATE TABLE %s.%s (\n", table.Schema, table.Name))

		// Columns
		colDefs := []string{}
		for _, col := range table.Columns {
			def := fmt.Sprintf("    %s %s", col.Name, col.DataType)
			if !col.IsNullable {
				def += " NOT NULL"
			}
			if col.DefaultValue != nil {
				def += fmt.Sprintf(" DEFAULT %s", *col.DefaultValue)
			}
			if col.IsIdentity {
				def += " GENERATED ALWAYS AS IDENTITY"
			}
			colDefs = append(colDefs, def)
		}
		sb.WriteString(strings.Join(colDefs, ",\n"))

		// Constraints
		if len(table.Constraints) > 0 {
			sb.WriteString(",\n")
			constraintDefs := []string{}
			for _, con := range table.Constraints {
				constraintDefs = append(constraintDefs, fmt.Sprintf("    CONSTRAINT %s %s", con.Name, con.Definition))
			}
			sb.WriteString(strings.Join(constraintDefs, ",\n"))
		}

		sb.WriteString("\n);\n")
		sb.WriteString(fmt.Sprintf("ALTER TABLE %s.%s OWNER TO %s;\n", table.Schema, table.Name, table.Owner))

		// Indexes (excluding primary key which is already in constraints)
		for _, idx := range table.Indexes {
			if !idx.IsPrimary {
				sb.WriteString(idx.Definition + ";\n")
			}
		}
		sb.WriteString("\n")
	}

	// Views
	for _, view := range schema.Views {
		sb.WriteString(fmt.Sprintf("-- View: %s.%s\n", view.Schema, view.Name))
		sb.WriteString(fmt.Sprintf("-- Owner: %s\n", view.Owner))
		sb.WriteString(view.Definition)
		if !strings.HasSuffix(view.Definition, ";") {
			sb.WriteString(";")
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("ALTER VIEW %s.%s OWNER TO %s;\n\n", view.Schema, view.Name, view.Owner))
	}

	return sb.String()
}

// StringArray is a helper type for scanning PostgreSQL arrays
type StringArray []string

func (a *StringArray) Scan(src interface{}) error {
	if src == nil {
		*a = []string{}
		return nil
	}

	switch v := src.(type) {
	case []byte:
		return a.scanBytes(v)
	case string:
		return a.scanBytes([]byte(v))
	default:
		return fmt.Errorf("unsupported type: %T", src)
	}
}

func (a *StringArray) scanBytes(src []byte) error {
	str := string(src)
	if str == "{}" || str == "" {
		*a = []string{}
		return nil
	}

	// Remove outer braces
	str = strings.Trim(str, "{}")
	
	// Split by comma
	parts := strings.Split(str, ",")
	result := make([]string, len(parts))
	for i, part := range parts {
		result[i] = strings.Trim(part, `"`)
	}
	
	*a = result
	return nil
}

// FormatSchemaReport generates a human-readable report
func (schema *DatabaseSchema) FormatSchemaReport() string {
	var sb strings.Builder

	sb.WriteString("═══════════════════════════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("  Database Schema Report: %s\n", schema.DatabaseName))
	sb.WriteString("═══════════════════════════════════════════════════════════════════════════════\n\n")

	sb.WriteString(fmt.Sprintf("Owner:     %s\n", schema.Owner))
	sb.WriteString(fmt.Sprintf("Encoding:  %s\n", schema.Encoding))
	sb.WriteString(fmt.Sprintf("Collation: %s\n\n", schema.Collation))

	// Extensions
	if len(schema.Extensions) > 0 {
		sb.WriteString("Extensions:\n")
		for _, ext := range schema.Extensions {
			sb.WriteString(fmt.Sprintf("  • %s (v%s) in schema %s\n", ext.Name, ext.Version, ext.Schema))
		}
		sb.WriteString("\n")
	}

	// Roles
	if len(schema.Roles) > 0 {
		sb.WriteString(fmt.Sprintf("Roles: %d\n", len(schema.Roles)))
		sort.Slice(schema.Roles, func(i, j int) bool {
			return schema.Roles[i].Name < schema.Roles[j].Name
		})
		for _, role := range schema.Roles {
			attrs := []string{}
			if role.IsSuperuser {
				attrs = append(attrs, "superuser")
			}
			if role.CanLogin {
				attrs = append(attrs, "login")
			}
			if role.CanCreateDB {
				attrs = append(attrs, "createdb")
			}
			if role.CanCreateRole {
				attrs = append(attrs, "createrole")
			}
			attrStr := ""
			if len(attrs) > 0 {
				attrStr = " [" + strings.Join(attrs, ", ") + "]"
			}
			sb.WriteString(fmt.Sprintf("  • %s%s\n", role.Name, attrStr))
			if len(role.MemberOf) > 0 {
				sb.WriteString(fmt.Sprintf("    Member of: %s\n", strings.Join(role.MemberOf, ", ")))
			}
		}
		sb.WriteString("\n")
	}

	// Tables summary
	if len(schema.Tables) > 0 {
		sb.WriteString(fmt.Sprintf("Tables: %d\n", len(schema.Tables)))
		totalRows := int64(0)
		totalSize := int64(0)
		for _, table := range schema.Tables {
			if table.RowCount >= 0 {
				totalRows += table.RowCount
			}
			if table.SizeBytes >= 0 {
				totalSize += table.SizeBytes
			}
			sb.WriteString(fmt.Sprintf("  • %s.%s (owner: %s)\n", table.Schema, table.Name, table.Owner))
			if table.RowCount >= 0 {
				sb.WriteString(fmt.Sprintf("    Rows: %d, Size: %s\n", table.RowCount, formatBytes(table.SizeBytes)))
			}
			sb.WriteString(fmt.Sprintf("    Columns: %d, Indexes: %d, Constraints: %d\n",
				len(table.Columns), len(table.Indexes), len(table.Constraints)))
		}
		sb.WriteString(fmt.Sprintf("\nTotal Rows: %d, Total Size: %s\n\n", totalRows, formatBytes(totalSize)))
	}

	// Views
	if len(schema.Views) > 0 {
		sb.WriteString(fmt.Sprintf("Views: %d\n", len(schema.Views)))
		for _, view := range schema.Views {
			sb.WriteString(fmt.Sprintf("  • %s.%s (owner: %s)\n", view.Schema, view.Name, view.Owner))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
