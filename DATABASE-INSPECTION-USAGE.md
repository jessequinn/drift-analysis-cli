# Database Schema Inspection Guide

## Quick Start

### 1. List Available Connections
```bash
./drift-analysis-cli gcp sql db --config config.yaml --list
```

Output shows all `database_connections` from your config:
```
Database connections in config (3):

 • zpe-cloud-test-environment:us-west1:test-c3ac43e6:postgres
 Instance: zpe-cloud-test-environment:us-west1:test-c3ac43e6
 Database: postgres
 Username: postgres
 Private IP: true
 ...
```

### 2. Inspect a Database (First Time)
```bash
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection zpe-cloud-test-environment:us-west1:test-c3ac43e6:postgres
```

This will:
1. Start Cloud SQL Proxy automatically (for private IP)
2. Connect to the database
3. Inspect schema (tables, views, functions, roles, extensions)
4. Cache the schema to `.drift-cache/database-schemas/`

**Output**:
```
Inspecting database connection: zpe-cloud-test-environment:us-west1:test-c3ac43e6:postgres
 Instance: zpe-cloud-test-environment:us-west1:test-c3ac43e6
 Database: postgres
 Private IP: true

Connecting and inspecting schema...
Starting Cloud SQL Proxy for zpe-cloud-test-environment:us-west1:test-c3ac43e6...
Started cloud-sql-proxy (PID: 12345), waiting for it to be ready...
 Proxy process is running and ready
 Proxy started successfully
Stopping Cloud SQL Proxy...

 Inspection complete!
 Tables: 15
 Views: 3
 Roles: 8
 Extensions: 5

 Cached schema to: /path/to/.drift-cache/database-schemas/zpe-cloud-test-environment_us-west1_test-c3ac43e6_postgres.json
 Initial baseline cached
```

### 3. Compare Schema Changes
After making changes to your database (migrations, DDL changes, etc.), run:

```bash
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection zpe-cloud-test-environment:us-west1:test-c3ac43e6:postgres \
 --compare
```

This will:
1. Load the cached baseline
2. Inspect the current schema
3. Show differences

**Output Example**:
```
 Cached schema exists (age: 2h15m)

Connecting and inspecting schema...
[... connection output ...]

 Inspection complete!
 Tables: 17
 Views: 4
 Roles: 8
 Extensions: 5

Comparing with cached baseline...

 Schema changes detected:

Added Tables (2):
 + public.orders (8 columns)
 + public.order_items (6 columns)

Modified Tables (1):
 ~ public.users

Added Views (1):
 + public.user_orders

Update cached baseline? (yes/no)
```

## Commands Reference

### List Connections
```bash
./drift-analysis-cli gcp sql db --config config.yaml --list
```

### Inspect Database (Create/Update Cache)
```bash
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection <name>
```

### Compare with Cached Baseline
```bash
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection <name> \
 --compare
```

### Use Custom Cache Directory
```bash
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection <name> \
 --cache-dir /custom/path
```

## Configuration

Add database connections to your `config.yaml`:

```yaml
database_connections:
 - name: "my-db-connection"
 instance_connection_name: "project:region:instance"
 database: "postgres"
 username: "postgres"
 password: "password" # or leave empty for IAM
 use_private_ip: true
```

## Use Cases

### Use Case 1: Track Production Schema Changes
```bash
# Daily: Check for unexpected schema changes
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection prod-db \
 --compare
```

### Use Case 2: Verify Migration Applied Correctly
```bash
# Before migration
./drift-analysis-cli gcp sql db --config config.yaml --connection staging-db

# Run migration
# ...

# After migration - check changes
./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection staging-db \
 --compare
```

### Use Case 3: Document Database Schema
```bash
# Inspect and cache schema
./drift-analysis-cli gcp sql db --config config.yaml --connection prod-db

# Cache is saved as JSON in .drift-cache/
# Can be committed to git for documentation
```

### Use Case 4: Multi-Database Inspection
```bash
# Loop through multiple databases
for db in postgres zpebackend test_zpebackend; do
 ./drift-analysis-cli gcp sql db \
 --config config.yaml \
 --connection "zpe-cloud-test-environment:us-west1:test-c3ac43e6:$db"
done
```

## What Gets Inspected?

The schema inspection captures:

### Database Metadata
- Database name, owner, encoding, collation

### Tables
- Schema and table name
- Owner
- Row count estimate
- Size in bytes
- Columns (name, type, nullable, default, identity)
- Constraints (primary key, foreign key, unique, check)
- Indexes (name, columns, unique, primary)

### Views
- Schema and view name
- Owner
- View definition (SQL)

### Roles
- Role name
- Superuser flag
- Can login flag
- Can create database flag
- Can create role flag
- Member of (groups)

### Extensions
- Extension name
- Version
- Schema

## Cache Structure

Cache files are stored as JSON in `.drift-cache/database-schemas/`:

```
.drift-cache/
└── database-schemas/
 ├── project_region_instance_database1.json
 ├── project_region_instance_database2.json
 └── ...
```

Each cache file contains:
```json
{
 "connection_name": "project:region:instance",
 "database": "postgres",
 "timestamp": "2025-12-29T22:00:00Z",
 "schema": {
 "database_name": "postgres",
 "owner": "postgres",
 "encoding": "UTF8",
 "tables": [...],
 "views": [...],
 "roles": [...],
 "extensions": [...]
 }
}
```

## Best Practices

### 1. Use Read-Only Users
Create a dedicated read-only user for inspection:
```sql
CREATE USER schema_inspector WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE postgres TO schema_inspector;
GRANT USAGE ON SCHEMA public TO schema_inspector;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO schema_inspector;
```

### 2. Regular Snapshots
Schedule daily/weekly inspections to track changes over time:
```bash
# Cron job example (daily at 2 AM)
0 2 * * * cd /path/to/cli && ./drift-analysis-cli gcp sql db --config config.yaml --connection prod-db
```

### 3. Commit Baselines to Git
Consider committing initial baseline to git for team sharing:
```bash
git add .drift-cache/database-schemas/prod-db.json
git commit -m "Add production database schema baseline"
```

### 4. Use IAM Authentication
Leave password empty and use IAM authentication:
```yaml
database_connections:
 - name: "prod-db"
 username: "inspector@project.iam"
 password: "" # Empty for IAM auth
```

### 5. Compare Before Merging
In CI/CD, compare schema after migrations:
```yaml
# .gitlab-ci.yml example
schema-check:
 script:
 - ./drift-analysis-cli gcp sql db --config config.yaml --connection staging-db --compare
```

## Troubleshooting

### "Connection refused"
- Ensure VPC connectivity for private IP instances
- Check that Cloud SQL Proxy can start
- Verify firewall rules

### "Authentication failed"
- Check username/password
- For IAM auth, ensure: `gcloud auth application-default login`
- Verify user exists in database

### "Cache not found"
- Run without `--compare` first to create initial cache
- Check cache directory exists

### "Proxy failed to start"
- Ensure `cloud-sql-proxy` is installed
- Try: `gcloud components install cloud-sql-proxy`

## Related Commands

### Infrastructure Drift (Separate from Schema)
```bash
# Check Cloud SQL instance configuration
./drift-analysis-cli gcp sql --config config.yaml
```

### Old Inspect Command (Using Flags)
```bash
# Direct connection with flags (not using config)
./drift-analysis-cli gcp sql inspect \
 --instance project:region:instance \
 --user postgres \
 --password password \
 --database postgres
```

## Next Steps

After inspecting schemas, you might want to:
1. **Document Schema**: Use cached JSON as documentation
2. **Track Changes**: Set up alerts when schema drifts
3. **Compliance Checks**: Ensure all tables have required columns
4. **Generate DDL**: Export schema as DDL for backups

## Summary

The `gcp sql db` command provides:
- Easy database schema inspection using config
- Automatic proxy management for private IP
- Local caching for fast comparison
- Schema drift detection
- Track changes over time
- Support for multiple databases per instance
