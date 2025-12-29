# Refactoring Summary: Separation of Concerns

## What Was Done

Successfully refactored the configuration structure to separate:
1. **Cloud SQL Infrastructure Baselines** (`sql_baselines`)
2. **Database Connection Configurations** (`database_connections`)

## Key Changes

### 1. Configuration Structure

**Before**:
```yaml
sql_baselines:
 - name: "application"
 connection: # Mixed with baseline 
 instance_connection_name: "..."
 database: "postgres"
 username: "postgres"
 password: "..."
 use_private_ip: true
 config:
 database_version: POSTGRES_14
 # ... infrastructure settings ...
```

**After**:
```yaml
# Infrastructure configuration (Cloud SQL instances)
sql_baselines:
 - name: "application"
 filter_labels:
 database-role: "application"
 config:
 database_version: POSTGRES_14
 tier: db-custom-4-16384
 # ... infrastructure settings ...

# Database connections (schema inspection)
database_connections:
 - name: "cfssl-test"
 instance_connection_name: "zpe-cloud-test-environment:us-west1:test-c3ac43e6"
 database: "postgres"
 username: "postgres"
 password: "kXJPFKWpH0"
 use_private_ip: true
```

### 2. New Components

#### `DatabaseConnection` Type
```go
type DatabaseConnection struct {
 Name string // Friendly name
 InstanceConnectionName string // project:region:instance
 Database string // Database name
 Username string // DB user
 Password string // Password (or IAM)
 UsePrivateIP bool // Private IP flag
 Project, Region, InstanceName string // Optional parts
}
```

#### `SchemaCache` Manager
- Stores inspected schemas locally in `.drift-cache/database-schemas/`
- JSON format for fast read/write
- Includes metadata: timestamp, connection info
- Enables offline schema comparison

#### `CompareSchemas()` Function
- Compares two DatabaseSchema objects
- Returns `SchemaDiff` with:
 - Added/deleted/modified tables
 - Added/deleted views
 - Added/deleted roles
 - Added/deleted extensions

### 3. File Structure

```
pkg/gcp/sql/
├── command.go # Updated with DatabaseConnection type
├── analyzer.go # Infrastructure drift analysis (unchanged)
├── inspector.go # Database schema inspection (unchanged)
├── proxy.go # Cloud SQL Proxy management (existing)
└── cache.go # NEW: Schema caching system
```

## Why This Design?

### Separation of Concerns

| Concern | Component | Purpose |
|---------|-----------|---------|
| **Infrastructure** | `sql_baselines` | Monitor Cloud SQL instance config drift |
| **Schema** | `database_connections` + cache | Track database schema changes |

### Different Use Cases

**Infrastructure Monitoring**:
- "Are all instances using the right tier?"
- "Do all instances have IAM auth enabled?"
- "Are backups configured correctly?"
- Check frequency: Hourly/daily
- Method: GCP API (no database connection needed)

**Schema Monitoring**:
- "What tables exist in this database?"
- "Has the schema changed since yesterday?"
- "Which tables were added/removed?"
- Check frequency: On-demand or per deployment
- Method: Direct database connection (requires proxy for private IP)

### Benefits

1. **Clarity**: Clear what each configuration section does
2. **Flexibility**: One infrastructure baseline, many database connections
3. **Performance**: Cache schemas locally, avoid repeated connections
4. **Independence**: Can check infrastructure without connecting to databases
5. **Security**: Connection credentials separate from baseline definitions

## Local Schema Cache

### Purpose
Store inspected database schemas locally for:
- **Fast comparison**: No need to reconnect every time
- **Offline analysis**: Analyze changes without live connection
- **History tracking**: See how schema evolved
- **Drift detection**: Compare current vs baseline

### Storage
```
.drift-cache/
└── database-schemas/
 ├── zpe-cloud-test-environment_us-west1_test-c3ac43e6_postgres.json
 └── zpe-cloud-test-environment_us-west1_test-c3ac43e6_cfssl.json
```

### Content Example
```json
{
 "connection_name": "zpe-cloud-test-environment:us-west1:test-c3ac43e6",
 "database": "postgres",
 "timestamp": "2025-12-29T22:00:00Z",
 "schema": {
 "database_name": "postgres",
 "owner": "postgres",
 "encoding": "UTF8",
 "tables": [
 {
 "schema": "public",
 "name": "users",
 "owner": "postgres",
 "columns": [...],
 "indexes": [...],
 "constraints": [...]
 }
 ],
 "views": [...],
 "roles": [...],
 "extensions": [...]
 }
}
```

## API Changes

### New Methods

#### DatabaseConnection
```go
func (dc *DatabaseConnection) GetConnectionName() string
func (dc *DatabaseConnection) Validate() error
func (dc *DatabaseConnection) ToConnectionConfig() *ConnectionConfig
```

#### SchemaCache
```go
func NewSchemaCache(cacheDir string) (*SchemaCache, error)
func (sc *SchemaCache) Save(connectionName, database string, schema *DatabaseSchema) error
func (sc *SchemaCache) Load(connectionName, database string) (*CachedSchema, error)
func (sc *SchemaCache) Exists(connectionName, database string) bool
func (sc *SchemaCache) GetAge(connectionName, database string) (time.Duration, error)
func (sc *SchemaCache) List() ([]CachedSchema, error)
func (sc *SchemaCache) Delete(connectionName, database string) error
func (sc *SchemaCache) Clear() error
func (sc *SchemaCache) ExportYAML(connectionName, database, outputPath string) error
```

#### Schema Comparison
```go
func CompareSchemas(old, new *DatabaseSchema) *SchemaDiff
func (sd *SchemaDiff) HasChanges() bool
```

## Backward Compatibility

The old `connection` field within `sql_baselines` is kept for backward compatibility through the `ConnectionConfig` type. However, it's recommended to migrate to the new structure.

## Future Commands (Not Yet Implemented)

```bash
# Schema inspection
./drift-analysis-cli sql inspect -connection cfssl-test
./drift-analysis-cli sql inspect -all

# Schema comparison
./drift-analysis-cli sql compare -connection cfssl-test
./drift-analysis-cli sql diff -connection cfssl-test --baseline cached

# Cache management
./drift-analysis-cli sql cache list
./drift-analysis-cli sql cache show -connection cfssl-test
./drift-analysis-cli sql cache clear
./drift-analysis-cli sql cache export -connection cfssl-test -format yaml

# Full analysis (both infrastructure and schema)
./drift-analysis-cli sql --full-analysis
```

## Configuration Example

See `config.yaml` for the complete example with:
- 4 infrastructure baselines (`sql_baselines`)
- 1 database connection (`database_connections`)
- Separated concerns
- Clear documentation

## Testing

All existing tests pass:
```bash
$ go test ./pkg/gcp/sql/... -v
=== RUN TestDatabaseConfig
--- PASS: TestDatabaseConfig (0.00s)
=== RUN TestSettingsConfig
--- PASS: TestSettingsConfig (0.00s)
[... 20+ more tests ...]
PASS
```

## Documentation

New documentation files:
1. **CONFIGURATION-REFACTORING.md**: Detailed explanation of the new structure
2. **config.yaml**: Updated with separated configuration
3. **config.yaml.example**: Updated example

Existing documentation updated:
- Removed old connection examples from `sql_baselines`
- Added new `database_connections` section

## Summary

This refactoring provides:
- Clear separation: infrastructure vs schema inspection
- Local caching for faster schema analysis
- Schema comparison/diff capabilities
- Better organized configuration
- Foundation for future schema monitoring features
- Backward compatible with existing configs
- All tests passing

The codebase is now better structured for future features like:
- Automated schema migration tracking
- Schema drift alerts
- Database documentation generation
- Compliance checking (e.g., "all tables must have audit columns")
