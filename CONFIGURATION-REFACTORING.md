# Refactored Configuration Structure

## Overview

The configuration has been refactored to separate two distinct concerns:

1. **Cloud SQL Instance Configuration** (`sql_baselines`) - Infrastructure drift detection
2. **Database Schema Inspection** (`database_connections`) - Content/schema analysis

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ config.yaml │
├─────────────────────────────────────────────────────────────┤
│ │
│ sql_baselines: database_connections: │
│ ├─ Cloud SQL instances ├─ Connection credentials │
│ ├─ Infrastructure config ├─ Private IP settings │
│ ├─ Database flags └─ Target databases │
│ ├─ Backup settings │ │
│ └─ IP configuration │ │
│ │ │ │
│ ▼ ▼ │
│ ┌─────────────────┐ ┌─────────────────┐ │
│ │ Instance Drift │ │ Schema Inspect │ │
│ │ Analysis │ │ + Cache │ │
│ └─────────────────┘ └─────────────────┘ │
│ │ │ │
│ ▼ ▼ │
│ Reports infra .drift-cache/ │
│ drift vs baseline database-schemas/ │
│ ├─ connection1.json │
│ ├─ connection2.json │
│ └─ (tables, views, │
│ functions, roles, │
│ DDLs, etc.) │
└─────────────────────────────────────────────────────────────┘
```

## 1. Cloud SQL Instance Baselines (`sql_baselines`)

**Purpose**: Define expected Cloud SQL instance **infrastructure** configuration

**What it checks**:
- Database version (e.g., POSTGRES_14)
- Instance tier/size (e.g., db-custom-2-7680)
- Disk configuration (size, type, autoresize)
- Database flags (e.g., `log_connections`, `cloudsql.iam_authentication`)
- Backup settings
- IP configuration (public/private IP, SSL)
- High availability settings
- Maintenance windows

**Example**:
```yaml
sql_baselines:
 - name: "application"
 filter_labels:
 database-role: "application"
 config:
 database_version: POSTGRES_14
 tier: db-custom-4-16384
 disk_size_gb: 100
 disk_type: PD_SSD
 database_flags:
 cloudsql.iam_authentication: "on"
 log_connections: "on"
 settings:
 availability_type: REGIONAL
 backup_enabled: true
```

**Use case**: "Are all my production databases using the correct instance type, have backups enabled, and use the right security flags?"

## 2. Database Connections (`database_connections`)

**Purpose**: Define how to **connect** to databases for schema inspection

**What it does**:
- Connects to actual databases (via Cloud SQL Proxy for private IP)
- Inspects database schema:
 - Tables (columns, constraints, indexes)
 - Views (definitions)
 - Functions & Procedures
 - Roles & Permissions
 - Extensions
 - Sequences
- Caches schema locally for fast comparison

**Example**:
```yaml
database_connections:
 - name: "cfssl-test"
 instance_connection_name: "project:region:instance"
 database: "postgres"
 username: "postgres"
 password: "..."
 use_private_ip: true
```

**Use case**: "What tables, views, and functions exist in this database? Has the schema changed since last inspection?"

## Local Schema Cache

### Location
```
.drift-cache/
└── database-schemas/
 ├── project_region_instance_postgres.json
 └── project_region_instance_app_db.json
```

### Contents
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
 "tables": [
 {
 "schema": "public",
 "name": "users",
 "owner": "postgres",
 "row_count": 1523,
 "size_bytes": 98304,
 "columns": [
 {"name": "id", "data_type": "integer", "is_nullable": false},
 {"name": "email", "data_type": "text", "is_nullable": false}
 ],
 "indexes": [...]
 }
 ],
 "views": [...],
 "roles": [...],
 "extensions": [...]
 }
}
```

### Benefits
1. **Fast Comparison**: No need to reconnect to database every time
2. **Offline Analysis**: Can analyze schema changes without live connection
3. **History**: Track how schema evolves over time
4. **Drift Detection**: Compare current state vs cached baseline

## Configuration Examples

### Complete Example

```yaml
projects:
 - my-project

# Infrastructure baselines
sql_baselines:
 - name: "production"
 filter_labels:
 environment: "prod"
 config:
 database_version: POSTGRES_15
 tier: db-custom-8-32768
 disk_size_gb: 500
 database_flags:
 cloudsql.iam_authentication: "on"
 max_connections: "500"
 settings:
 availability_type: REGIONAL
 backup_enabled: true

# Schema inspection connections
database_connections:
 - name: "prod-app-db"
 instance_connection_name: "my-project:us-central1:prod-db"
 database: "application"
 username: "inspector"
 password: "${DB_PASSWORD}" # Can use env vars
 use_private_ip: true

 - name: "prod-analytics-db"
 instance_connection_name: "my-project:us-central1:analytics-db"
 database: "warehouse"
 username: "readonly"
 use_private_ip: true
```

## Usage Workflows

### Workflow 1: Infrastructure Drift Analysis
```bash
# Check if Cloud SQL instances match baseline config
./drift-analysis-cli sql -config config.yaml

# Output: Which instances have wrong tier, missing flags, etc.
```

### Workflow 2: Schema Inspection (First Time)
```bash
# Inspect and cache database schema
./drift-analysis-cli sql inspect -connection cfssl-test

# Creates: .drift-cache/database-schemas/cfssl-test.json
```

### Workflow 3: Schema Change Detection
```bash
# Re-inspect and compare with cached baseline
./drift-analysis-cli sql inspect -connection cfssl-test --compare

# Output:
# Added tables: orders, products
# Deleted views: old_report_view
# Modified tables: users (2 new columns)
```

### Workflow 4: Combined Analysis
```bash
# Check both infrastructure AND schema
./drift-analysis-cli sql --full-analysis

# Checks:
# 1. Are instances configured correctly? (sql_baselines)
# 2. Has database schema changed? (database_connections + cache)
```

## Why This Separation?

### Before (Mixed)
```yaml
sql_baselines:
 - name: "application"
 connection: # Connection mixed with baseline
 instance_connection_name: "..."
 username: "..."
 config: # Infrastructure config
 database_version: POSTGRES_14
```

**Problems**:
- Confusing: Is this for infrastructure or schema inspection?
- Can't have multiple databases per instance baseline
- Connection credentials tied to infrastructure baseline

### After (Separated)
```yaml
sql_baselines: # Infrastructure ONLY
 - name: "application"
 config:
 database_version: POSTGRES_14

database_connections: # Schema inspection ONLY
 - name: "cfssl-test"
 instance_connection_name: "..."
 username: "..."
```

**Benefits**:
- Clear separation of concerns
- One infrastructure baseline can relate to many database connections
- Connection credentials isolated
- Cache schema locally for fast comparison
- Can inspect multiple databases on same instance

## Migration Path

Existing configs with `connection:` field in `sql_baselines` will continue to work (backward compatible), but it's recommended to migrate:

**Old**:
```yaml
sql_baselines:
 - name: "app"
 connection:
 instance_connection_name: "..."
 config: {...}
```

**New**:
```yaml
sql_baselines:
 - name: "app"
 config: {...}

database_connections:
 - name: "app-db"
 instance_connection_name: "..."
```

## Commands (Future)

```bash
# Infrastructure drift
./drift-analysis-cli sql drift

# Schema inspection
./drift-analysis-cli sql inspect -connection <name>
./drift-analysis-cli sql inspect -all # All connections

# Schema comparison
./drift-analysis-cli sql compare -connection <name>
./drift-analysis-cli sql compare -baseline cached

# Cache management
./drift-analysis-cli sql cache list
./drift-analysis-cli sql cache clear
./drift-analysis-cli sql cache export -connection <name> -format yaml

# Full analysis
./drift-analysis-cli sql --full-analysis
```

## Summary

| Aspect | `sql_baselines` | `database_connections` |
|--------|----------------|------------------------|
| **Purpose** | Infrastructure config | Schema inspection |
| **Checks** | Instance settings, flags, disk | Tables, views, functions, roles |
| **Output** | Drift report | Cached schema + diff |
| **Frequency** | Periodic (hourly/daily) | On-demand or scheduled |
| **Cache** | No | Yes (`.drift-cache/`) |
| **Connection** | API-based (no proxy) | Direct (with proxy) |

**Key Insight**: Infrastructure can drift slowly (someone changes a flag), but schema can change frequently (devs deploying migrations). Separating these allows different monitoring strategies.
