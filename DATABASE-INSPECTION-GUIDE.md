# Database Schema Inspection Guide

The drift-analysis-cli now includes a powerful database inspection feature that connects directly to PostgreSQL databases to extract detailed schema information, including DDL, roles, and table ownership.

## Overview

Unlike the drift analysis which uses the Cloud SQL Admin API (limited to instance-level configuration), the `inspect` command **directly connects to PostgreSQL databases** to retrieve:

- **Database metadata** - owner, encoding, collation
- **Roles/Users** - privileges, superuser status, group membership
- **Tables** - columns, data types, constraints, indexes, owners, row counts, sizes
- **Views** - definitions and ownership
- **Extensions** - installed extensions and versions
- **DDL Generation** - Complete schema recreation scripts

## Requirements

### Database Access
You need direct database connection credentials:
- Database host/IP address
- Port (default: 5432)
- Username with sufficient privileges
- Password
- Database name (default: postgres)

### Required Privileges
The database user should have at least:
- `CONNECT` privilege on the database
- `SELECT` on `pg_catalog` tables
- `SELECT` on `information_schema` views

For complete inspection, `pg_read_all_stats` role is recommended.

## Usage

### Basic Schema Report

```bash
# Connect and generate a human-readable report
drift-analysis-cli gcp sql inspect \
 --host 10.0.0.5 \
 --user myuser \
 --password mypassword \
 --database mydb
```

### Generate DDL

```bash
# Extract complete DDL for schema recreation
drift-analysis-cli gcp sql inspect \
 --host 10.0.0.5 \
 --user myuser \
 --password mypassword \
 --database mydb \
 --format ddl \
 --output-file schema.sql
```

### Short Flags

```bash
# Using short flags
drift-analysis-cli gcp sql inspect \
 -H 10.0.0.5 \
 -u myuser \
 -p mypassword \
 -d mydb \
 -f ddl \
 -o schema.sql
```

## Output Formats

### Report Format (default)
Human-readable summary with:
- Database information
- List of extensions
- Roles with privileges
- Tables with metadata (rows, size, columns, indexes, constraints)
- Views summary

Example output:
```
═══════════════════════════════════════════════════════════════════════════════
 Database Schema Report: mydb
═══════════════════════════════════════════════════════════════════════════════

Owner: postgres
Encoding: UTF8
Collation: en_US.UTF8

Extensions:
 • uuid-ossp (v1.1) in schema public
 • pg_stat_statements (v1.9) in schema public

Roles: 5
 • app_readonly [login]
 Member of: readonly_group
 • app_writer [login, createdb]
 Member of: writer_group
 • postgres [superuser, login, createdb, createrole]

Tables: 12
 • public.users (owner: postgres)
 Rows: 15234, Size: 2.3 MB
 Columns: 8, Indexes: 3, Constraints: 2
 • public.orders (owner: app_writer)
 Rows: 45678, Size: 12.8 MB
 Columns: 12, Indexes: 5, Constraints: 4

Total Rows: 234567, Total Size: 145.6 MB

Views: 3
 • public.user_summary (owner: postgres)
 • public.order_stats (owner: postgres)
```

### DDL Format
Complete SQL DDL statements for recreating the schema:
```sql
-- Database: mydb
-- Owner: postgres
-- Encoding: UTF8
-- Collation: en_US.UTF8

-- Extensions
CREATE EXTENSION IF NOT EXISTS uuid-ossp WITH SCHEMA public;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements WITH SCHEMA public;

-- Roles
CREATE ROLE app_readonly WITH LOGIN;
CREATE ROLE app_writer WITH LOGIN CREATEDB;

-- Table: public.users
-- Owner: postgres
-- Rows: 15234
CREATE TABLE public.users (
 id bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
 username character varying(50) NOT NULL,
 email character varying(100) NOT NULL,
 created_at timestamp with time zone DEFAULT now(),
 CONSTRAINT users_pkey PRIMARY KEY (id),
 CONSTRAINT users_email_key UNIQUE (email)
);
ALTER TABLE public.users OWNER TO postgres;
CREATE INDEX idx_users_username ON public.users USING btree (username);
```

## Use Cases

### 1. Schema Documentation
Generate comprehensive documentation of your database schema:
```bash
drift-analysis-cli gcp sql inspect \
 -H $DB_HOST -u $DB_USER -p $DB_PASS -d $DB_NAME \
 --format report \
 --output-file docs/schema-report.txt
```

### 2. Schema Migration/Backup
Extract DDL for schema recreation or migration:
```bash
drift-analysis-cli gcp sql inspect \
 -H $DB_HOST -u $DB_USER -p $DB_PASS -d $DB_NAME \
 --format ddl \
 --output-file backups/schema-$(date +%Y%m%d).sql
```

### 3. Security Audit
Review roles, privileges, and table ownership:
```bash
drift-analysis-cli gcp sql inspect \
 -H $DB_HOST -u $DB_USER -p $DB_PASS -d $DB_NAME \
 | grep -A 5 "Roles:"
```

### 4. Table Size Analysis
Identify large tables and row counts:
```bash
drift-analysis-cli gcp sql inspect \
 -H $DB_HOST -u $DB_USER -p $DB_PASS -d $DB_NAME \
 | grep -A 3 "Tables:"
```

### 5. Extension Inventory
List all installed PostgreSQL extensions:
```bash
drift-analysis-cli gcp sql inspect \
 -H $DB_HOST -u $DB_USER -p $DB_PASS -d $DB_NAME \
 | grep -A 20 "Extensions:"
```

## Connection Options

### Cloud SQL Private IP
```bash
# Connect to Cloud SQL via private IP
drift-analysis-cli gcp sql inspect \
 -H 10.128.0.5 \
 -u myuser \
 -p mypassword \
 -d mydb
```

### Cloud SQL Public IP with SSL
The inspector automatically uses `sslmode=require`. For Cloud SQL:
```bash
# Public IP connection (SSL enforced)
drift-analysis-cli gcp sql inspect \
 -H 34.123.45.67 \
 -u myuser \
 -p mypassword \
 -d mydb
```

### Custom Port
```bash
# Non-standard port
drift-analysis-cli gcp sql inspect \
 -H 10.0.0.5 \
 -P 5433 \
 -u myuser \
 -p mypassword \
 -d mydb
```

## Information Extracted

### Database Level
- Name
- Owner
- Character encoding
- Collation (LC_COLLATE, LC_CTYPE)

### Roles
- Role name
- Superuser status
- Login capability
- Create database privilege
- Create role privilege
- Group membership (member of)

### Tables
- Schema name
- Table name
- Owner
- Approximate row count
- Total size (including indexes and TOAST)
- Column definitions:
 - Name
 - Data type
 - Nullable/NOT NULL
 - Default values
 - Identity columns
- Constraints:
 - Primary keys
 - Foreign keys
 - Unique constraints
 - Check constraints
- Indexes:
 - Name
 - Columns
 - Unique/non-unique
 - Full index definition

### Views
- Schema name
- View name
- Owner
- Complete view definition (SELECT statement)

### Extensions
- Extension name
- Version
- Schema location

## Limitations

1. **System Catalogs Only**: Filters out PostgreSQL system schemas (`pg_catalog`, `information_schema`)
2. **Cloud SQL Roles**: Filters out `pg_*` and `cloudsql*` system roles
3. **Row Count Estimates**: Uses `pg_stat_user_tables` which may not be exact
4. **No Data Export**: Only extracts schema/DDL, not actual table data
5. **Connection Required**: Requires direct network access to the database

## Security Considerations

### Credential Management
**Never** store passwords in command history or scripts:

```bash
# Good: Use environment variables
export DB_PASSWORD="secret"
drift-analysis-cli gcp sql inspect -H $HOST -u $USER -p $DB_PASSWORD -d $DB

# Good: Prompt for password (in future version)
drift-analysis-cli gcp sql inspect -H $HOST -u $USER --prompt-password -d $DB

# Bad: Password in command line
drift-analysis-cli gcp sql inspect -H $HOST -u $USER -p secret -d $DB
```

### Read-Only Access
Create a dedicated read-only role for inspection:

```sql
-- Create read-only role
CREATE ROLE schema_inspector LOGIN PASSWORD 'strong_password';

-- Grant necessary privileges
GRANT CONNECT ON DATABASE mydb TO schema_inspector;
GRANT pg_read_all_stats TO schema_inspector;
GRANT SELECT ON ALL TABLES IN SCHEMA information_schema TO schema_inspector;
GRANT SELECT ON ALL TABLES IN SCHEMA pg_catalog TO schema_inspector;
```

Then use:
```bash
drift-analysis-cli gcp sql inspect \
 -H $HOST -u schema_inspector -p $PASSWORD -d mydb
```

## Troubleshooting

### Connection Errors
```
Error: failed to connect: dial tcp ...: i/o timeout
```
**Solution**: Check network connectivity, firewall rules, and Cloud SQL authorized networks.

### Permission Denied
```
Error: failed to get roles: permission denied for table pg_authid
```
**Solution**: Grant `pg_read_all_stats` role or run as a superuser.

### SSL/TLS Errors
```
Error: SSL is not enabled on the server
```
**Solution**: Use `sslmode=disable` (not recommended for production) or enable SSL on the server.

## Examples

### Full Workflow
```bash
#!/bin/bash

# Configuration
DB_HOST="10.128.0.5"
DB_USER="postgres"
DB_PASS="your-password"
DATABASES=("mydb1" "mydb2" "mydb3")

# Create output directory
mkdir -p schema-docs

# Extract schema for each database
for db in "${DATABASES[@]}"; do
 echo "Extracting schema for $db..."

 # Generate report
 drift-analysis-cli gcp sql inspect \
 -H $DB_HOST -u $DB_USER -p $DB_PASS -d $db \
 --format report \
 --output-file schema-docs/${db}-report.txt

 # Generate DDL
 drift-analysis-cli gcp sql inspect \
 -H $DB_HOST -u $DB_USER -p $DB_PASS -d $db \
 --format ddl \
 --output-file schema-docs/${db}-schema.sql
done

echo "Schema extraction complete!"
```

## Integration with Drift Analysis

Use inspection results to:
1. Document current database state before drift analysis
2. Verify table ownership matches security policies
3. Audit role and privilege configurations
4. Generate baseline configurations for specific databases
5. Track schema changes over time by comparing DDL outputs

## Future Enhancements

Planned features:
- [ ] Schema comparison between databases
- [ ] Role/privilege drift detection
- [ ] Table ownership validation against baselines
- [ ] Automatic DDL diff generation
- [ ] Export to JSON/YAML formats
- [ ] Password prompt option (secure input)
- [ ] Support for reading password from file
- [ ] Parallel inspection of multiple databases
