# Database Inspection Quick Start

## Inspect All Databases with Full DDL Output

### Command
```bash
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --all \
 --format ddl \
 --output-dir ./schema-exports
```

This will:
1. Inspect all database connections defined in `config.yaml`
2. Generate DDL (CREATE TABLE, CREATE VIEW, etc.) for each database
3. Save files to `./schema-exports/` directory
4. Cache schemas in `.drift-cache/database-schemas/`

### Output Files

For each connection, generates:
- `<connection-name>-schema.sql` - Complete DDL statements

## Available Output Formats

### 1. Summary (Default - Console Only)
```bash
./drift-analysis-cli gcp sql db --config config.yaml --all
```
Shows quick stats in console

### 2. Full Report (Detailed Text)
```bash
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --all \
 --format full \
 --output-dir ./reports
```

Generates: `<connection>-full-report.txt` with:
- Database information
- All roles with privileges
- All extensions
- All tables with:
 - Columns (name, type, nullable, defaults)
 - Indexes
 - Constraints
 - Owners
- All views with definitions

### 3. DDL (SQL Statements)
```bash
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --all \
 --format ddl \
 --output-dir ./ddl
```

Generates: `<connection>-schema.sql` with:
- CREATE TABLE statements
- CREATE VIEW statements
- CREATE INDEX statements
- ALTER TABLE constraints
- Comments on tables/columns

### 4. JSON (Structured Data)
```bash
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --all \
 --format json \
 --output-dir ./json
```

Generates: `<connection>-schema.json`

### 5. YAML (Readable Structured)
```bash
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --all \
 --format yaml \
 --output-dir ./yaml
```

Generates: `<connection>-schema.yaml`

## Single Connection Examples

### Inspect Specific Connection with DDL
```bash
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection "zpe-cloud-test-environment:us-west1:test-c3ac43e6:postgres" \
 --format ddl \
 --output-dir ./ddl
```

### Generate Full Report for One Database
```bash
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection "zpe-cloud-test-environment:us-west1:test-c3ac43e6:zpebackend" \
 --format full \
 --output-dir ./reports
```

## What Gets Exported

### Database Metadata
- Database name, owner, encoding, collation

### Roles & Users
- Role name
- Privileges (superuser, login, create DB, create role)
- Group memberships

### Extensions
- Extension name, version, schema

### Tables
- Full CREATE TABLE DDL
- Column definitions (name, type, nullability, defaults)
- Primary keys
- Foreign keys
- Unique constraints
- Check constraints
- Indexes (with definitions)
- Table owner
- Row count estimates
- Table size

### Views
- Full CREATE VIEW DDL
- View definitions
- View owner

### Functions & Procedures
(If available in schema - depends on inspector implementation)

## Typical Workflows

### Workflow 1: Document All Databases
```bash
# Generate comprehensive documentation
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --all \
 --format full \
 --output-dir ./documentation

# Generate DDL for backup/migration
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --all \
 --format ddl \
 --output-dir ./ddl-backup
```

### Workflow 2: Compare Environments
```bash
# Export test environment
./drift-analysis-cli gcp sql db \
 --config config-test.yaml \
 --all \
 --format ddl \
 --output-dir ./test-ddl

# Export production environment 
./drift-analysis-cli gcp sql db \
 --config config-prod.yaml \
 --all \
 --format ddl \
 --output-dir ./prod-ddl

# Compare with diff
diff -r ./test-ddl ./prod-ddl
```

### Workflow 3: Schema Migration Planning
```bash
# Before migration - snapshot current state
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection my-db \
 --format ddl \
 --output-dir ./before-migration

# Run migrations
# ...

# After migration - snapshot new state
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection my-db \
 --format ddl \
 --output-dir ./after-migration

# Or use --compare flag
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection my-db \
 --compare
```

### Workflow 4: Compliance Checking
```bash
# Export all schemas to JSON for automated checking
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --all \
 --format json \
 --output-dir ./compliance-check

# Then use jq or custom scripts to verify:
# - All tables have required audit columns
# - Sensitive data columns are encrypted
# - Naming conventions are followed
# - Proper indexing exists
```

## Command Summary

```bash
# List connections
./drift-analysis-cli gcp sql db --config config.yaml --list

# Inspect all (summary)
./drift-analysis-cli gcp sql db --config config.yaml --all

# Inspect all with DDL
./drift-analysis-cli gcp sql db --config config.yaml --all --format ddl --output-dir ./ddl

# Inspect all with full reports
./drift-analysis-cli gcp sql db --config config.yaml --all --format full --output-dir ./reports

# Inspect single connection
./drift-analysis-cli gcp sql db --config config.yaml --connection <name> --format ddl

# Compare with cached baseline
./drift-analysis-cli gcp sql db --config config.yaml --connection <name> --compare
```

## Output Directory Structure

After running with `--all --format ddl --output-dir ./exports`:

```
exports/
├── zpe-cloud-test-environment_us-west1_test-c3ac43e6_postgres-schema.sql
├── zpe-cloud-test-environment_us-west1_test-c3ac43e6_zpebackend-schema.sql
└── zpe-cloud-test-environment_us-west1_test-c3ac43e6_test_zpebackend-schema.sql
```

## Notes

- Files are named with connection name prefix to avoid collisions
- Colons (`:`) in connection names are replaced with underscores (`_`)
- DDL output includes CREATE statements for easy recreation
- Full reports include all metadata in readable text format
- JSON/YAML outputs are suitable for programmatic processing
- Cache is always updated regardless of format chosen

## Integration with CI/CD

```yaml
# .gitlab-ci.yml example
schema-export:
 script:
 - ./drift-analysis-cli gcp sql db --config config.yaml --all --format ddl --output-dir ./schema-ddl
 artifacts:
 paths:
 - schema-ddl/
 expire_in: 30 days

schema-check:
 script:
 - ./drift-analysis-cli gcp sql db --config config.yaml --connection prod-db --compare
 only:
 - main
```
