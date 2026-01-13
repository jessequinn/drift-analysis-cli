# GCP Drift Analysis CLI

[![CI](https://github.com/jessequinn/drift-analysis-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/jessequinn/drift-analysis-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jessequinn/drift-analysis-cli)](https://goreportcard.com/report/github.com/jessequinn/drift-analysis-cli)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A comprehensive CLI tool for detecting configuration drift across Google Cloud Platform resources including Cloud SQL PostgreSQL instances, GKE clusters, and GCE instances.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Commands](#commands)
- [Database Schema Inspection](#database-schema-inspection)
- [Drift Detection Details](#drift-detection-details)
- [Authentication](#authentication)
- [Advanced Usage](#advanced-usage)
- [Development](#development)
- [Error Handling](#error-handling)
- [License](#license)

## Features

- **Multi-Resource Drift Detection**: Cloud SQL, GKE clusters, and GCE instances
- **Severity-Based Analysis**: CRITICAL, HIGH, MEDIUM, LOW classifications
- **Multi-Project Support**: Analyze resources across multiple GCP projects
- **Database Schema Inspection**: Direct PostgreSQL connection for DDL extraction
- **Flexible Output**: Text, JSON, YAML, and interactive TUI formats
- **Structured Logging**: JSON and text logging with file output support
- **Baseline Generation**: Auto-generate configs from existing infrastructure
- **Label-Based Filtering**: Target specific resource roles or environments
- **Security Focus**: Identifies security gaps and misconfigurations

## Installation

```bash
# Clone and build
git clone https://github.com/jessequinn/drift-analysis-cli
cd drift-analysis-cli
go mod download
go build -o drift-analysis-cli
```

## Quick Start

### Cloud SQL Drift Analysis

```bash
# Basic drift analysis
./drift-analysis-cli gcp sql --config config.yaml

# With JSON logging to file
./drift-analysis-cli gcp sql --config config.yaml --json-logs --log-file sql-analysis.log

# JSON output
./drift-analysis-cli gcp sql --config config.yaml --output json

# Interactive TUI
./drift-analysis-cli gcp sql --config config.yaml --output tui
```

### GKE Drift Analysis

```bash
# Basic analysis
./drift-analysis-cli gcp gke --config config.yaml

# Generate baseline from current infrastructure
./drift-analysis-cli gcp gke --generate-baseline --output gke-baseline.yaml

# JSON output with logging
./drift-analysis-cli gcp gke --config config.yaml --output json --log-file gke.log
```

### GCE Drift Analysis

```bash
# Basic analysis
./drift-analysis-cli gcp gce --config config.yaml

# With severity filtering (JSON output)
./drift-analysis-cli gcp gce --config config.yaml --output json | jq '.drifts[] | select(.severity=="CRITICAL")'

# Generate baseline
./drift-analysis-cli gcp gce --generate-baseline --output gce-baseline.yaml
```

### Database Schema Inspection

```bash
# List configured database connections
./drift-analysis-cli gcp sql db --config config.yaml --list

# Inspect a database (creates cache)
./drift-analysis-cli gcp sql db --config config.yaml --connection mydb

# Compare with cached baseline
./drift-analysis-cli gcp sql db --config config.yaml --connection mydb --compare

# Export DDL
./drift-analysis-cli gcp sql db --config config.yaml --connection mydb --format ddl --output-dir ./schemas
```

## Configuration

Create a `config.yaml` file with your baselines:

```yaml
# Projects to scan
projects:
  - my-project-1
  - my-project-2

# Cloud SQL baselines
sql_baselines:
  - name: "application"
    filter_labels:
      database-role: "application"
    config:
      database_version: POSTGRES_15
      tier: db-custom-4-16384
      disk_size_gb: 100
      disk_type: PD_SSD
      
      required_databases:
        - app_db
        - postgres
      
      database_flags:
        cloudsql.iam_authentication: "on"
        max_connections: "200"
        log_min_duration_statement: "1000"
      
      settings:
        availability_type: REGIONAL
        backup_enabled: true
        backup_retention_days: 7
        point_in_time_recovery: true
        transaction_log_retention_days: 7
      
      ip_configuration:
        ipv4_enabled: false
        require_ssl: true
        authorized_networks:
          - "10.0.0.0/24"
      
      insights_config:
        query_insights_enabled: true

# GKE baselines
gke_baselines:
  - name: "production"
    filter_labels:
      cluster-role: "production"
    cluster_config:
      master_version: "1.33"
      release_channel: REGULAR
      private_cluster: true
      master_global_access: true
      datapath_provider: ADVANCED_DATAPATH
      
      master_authorized_networks:
        - "10.0.0.0/24"
      
      ip_allocation_policy:
        stack_type: IPV4_IPV6
      
      shielded_nodes: true
      security_posture: BASIC
      workload_identity: true
      
      logging_config:
        enable_system_logs: true
        enable_workload_logs: true
      
      monitoring_config:
        enable_system_metrics: true
        enable_apiserver_metrics: true
      
      nodepool_config:
        machine_type: n2-standard-4
        disk_size_gb: 100
        disk_type: pd-ssd
        image_type: COS_CONTAINERD
        auto_upgrade: true
        auto_repair: true

# GCE baselines
gce_baselines:
  - name: "production-vms"
    filter_labels:
      env: "production"
      instance-role: "app-server"
    vm_config:
      machine_type: n2-standard-4
      disk_size_gb: 100
      disk_type: pd-ssd
      image_family: ubuntu-2204-jammy
      image_project: ubuntu-os-cloud
      
      network_config:
        network: default
        subnetwork: default
        external_ip: false
        network_tier: PREMIUM
      
      service_account:
        email: "app-sa@project.iam.gserviceaccount.com"
        scopes:
          - "https://www.googleapis.com/auth/logging.write"
          - "https://www.googleapis.com/auth/monitoring.write"
      
      metadata:
        enable-oslogin: "TRUE"
      
      tags:
        - "http-server"
        - "https-server"
      
      preemptible: false
      automatic_restart: true
      on_host_maintenance: "MIGRATE"
      deletion_protection: true
      
      shielded_vm:
        enable_secure_boot: true
        enable_vtpm: true
        enable_integrity_monitoring: true

# Database connections for schema inspection
database_connections:
  - name: "production-database"
    instance_connection_name: "project:region:instance"
    database: "mydb"
    username: "postgres"
    password: "${DB_PASSWORD}"  # Use env var
    use_private_ip: true
    
    # Optional: SSH tunnel through bastion
    ssh_tunnel:
      enabled: true
      bastion_host: "bastion"
      bastion_zone: "us-west1-a"
      project: "my-project"
      private_ip: "10.50.0.3"
      use_iap: true
    
    # Schema baseline configuration
    schema_baseline:
      expected_tables: 124
      expected_views: 2
      expected_sequences: 15
      expected_functions: 8
      expected_procedures: 3
      expected_roles: 12
      expected_extensions: 2
      expected_database_owner: "cloudsqlsuperuser"
      expected_table_owner: "postgres"
      
      # Ownership exceptions
      table_owner_exceptions:
        audit_log: "audit_user"
        system_config: "admin"
      
      # Required extensions
      required_extensions:
        - pgcrypto
        - uuid-ossp
```

## Commands

### Global Flags

All commands support these global flags:

```
--config string       Config file path (default: config.yaml)
--output string       Output format: text|json|yaml|tui (default: text)
--log-file string     Write logs to file (in addition to stderr)
--json-logs           Output logs in JSON format
```

### Cloud SQL Commands

```bash
# Drift analysis
drift-analysis-cli gcp sql [flags]

# Database schema inspection
drift-analysis-cli gcp sql db [flags]

# Direct database inspection (deprecated, use 'db' command)
drift-analysis-cli gcp sql inspect [flags]
```

### GKE Commands

```bash
# Drift analysis
drift-analysis-cli gcp gke [flags]
```

### GCE Commands

```bash
# Drift analysis
drift-analysis-cli gcp gce [flags]
```

## Database Schema Inspection

The `sql db` command connects directly to PostgreSQL databases to extract and compare schemas.

### Features

- **Complete Schema Extraction**: Tables, views, sequences, functions, procedures, roles, extensions
- **DDL Generation**: Export full CREATE statements for schema recreation
- **Drift Detection**: Compare current schema against cached baseline
- **Ownership Validation**: Track object ownership and detect changes
- **Connection Methods**: Cloud SQL Proxy, IAM Connector, SSH tunnels, or direct TCP

### Basic Usage

```bash
# List all database connections
./drift-analysis-cli gcp sql db --list

# Inspect database (creates/updates cache)
./drift-analysis-cli gcp sql db --connection mydb

# Compare with baseline
./drift-analysis-cli gcp sql db --connection mydb --compare

# Inspect all databases
./drift-analysis-cli gcp sql db --all
```

### Output Formats

```bash
# Summary view (default)
./drift-analysis-cli gcp sql db --connection mydb --format summary

# Full details
./drift-analysis-cli gcp sql db --connection mydb --format full

# DDL export
./drift-analysis-cli gcp sql db --connection mydb --format ddl

# JSON export
./drift-analysis-cli gcp sql db --connection mydb --format json

# YAML export
./drift-analysis-cli gcp sql db --connection mydb --format yaml
```

### Schema Comparison

The tool caches schemas in `.drift-cache/database-schemas/` for baseline comparisons:

```bash
# First run creates baseline
./drift-analysis-cli gcp sql db --connection prod-db

# Later runs compare against baseline
./drift-analysis-cli gcp sql db --connection prod-db --compare
```

**Detected Changes:**
- Table additions/deletions
- Column modifications
- Constraint changes
- Index changes
- Role changes
- Extension changes
- Ownership changes

## Drift Detection Details

### Cloud SQL Checks

**Core Configuration (MEDIUM)**
- PostgreSQL version
- Machine tier (CPU/Memory)
- Disk size, type, and autoresize

**Database Flags (LOW to MEDIUM)**
- All PostgreSQL configuration parameters
- Performance tuning (connections, work_mem, etc.)

**High Availability (HIGH to CRITICAL)**
- Availability type (ZONAL vs REGIONAL)
- Backup configuration and retention
- Point-in-time recovery (PITR)
- Transaction log retention

**Security (CRITICAL)**
- SSL/TLS requirements
- Public vs private IP
- Authorized networks
- IAM authentication

**Observability (LOW to MEDIUM)**
- Query Insights configuration
- Performance monitoring

### GKE Checks

**Networking (MEDIUM to HIGH) - 9 checks**
- Network/Subnetwork configuration
- Private cluster settings
- Master global access
- Master authorized networks
- Datapath provider (ADVANCED vs LEGACY)
- IP allocation policy (IPv4/IPv6)

**Security (CRITICAL to HIGH) - 6 checks**
- Shielded nodes
- Database encryption (ETCD at rest)
- Security posture (BASIC/ENTERPRISE)
- Workload identity
- Binary authorization
- Network policy

**Features & Observability (LOW to MEDIUM) - 10+ checks**
- System and workload logging
- Metrics collection (system, API server, controller)
- Kubernetes version and release channel
- Addon configurations (HTTP LB, HPA)
- Node pool configuration

### GCE Checks

**Security (CRITICAL)**
- Shielded VM (Secure Boot, vTPM, Integrity Monitoring)
- Service account (avoid default compute SA)
- Scope restrictions (avoid cloud-platform scope)
- External IP exposure
- Deletion protection

**Availability & Reliability (HIGH)**
- Preemptibility (avoid for production)
- Automatic restart
- Host maintenance behavior (MIGRATE vs TERMINATE)

**Configuration (MEDIUM)**
- Machine type
- Boot disk size and type
- Image family and project
- Network and subnetwork
- OS Login configuration

**Metadata & Labeling (LOW)**
- Startup scripts
- Custom metadata
- Resource labels
- Network tags

### Severity Levels

- **CRITICAL**: Security vulnerabilities, disabled encryption, public exposure
- **HIGH**: HA misconfigurations, disabled backups, major version drift
- **MEDIUM**: Performance settings, resource sizing, network configuration
- **LOW**: Optimization suggestions, monitoring configs, labeling

## Authentication

The CLI uses Application Default Credentials (ADC):

```bash
# Option 1: User credentials
gcloud auth application-default login

# Option 2: Service account
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
```

### Required IAM Permissions

**Cloud SQL**: `roles/cloudsql.viewer` or:
- `cloudsql.instances.get`
- `cloudsql.instances.list`
- `cloudsql.databases.list`

**GKE**: `roles/container.viewer` or:
- `container.clusters.get`
- `container.clusters.list`

**GCE**: `roles/compute.viewer` or:
- `compute.instances.get`
- `compute.instances.list`
- `compute.zones.list`

**Database Inspection**: Direct database credentials with:
- `CONNECT` privilege on database
- `SELECT` on `pg_catalog` tables
- `SELECT` on `information_schema` views
- Recommended: `pg_read_all_stats` role

## Advanced Usage

### Label-Based Filtering

Apply labels to resources for selective analysis:

**Cloud SQL:**
```bash
gcloud sql instances patch INSTANCE_NAME \
  --update-labels database-role=application
```

**GKE:**
```bash
gcloud container clusters update CLUSTER_NAME \
  --update-labels cluster-role=production \
  --location LOCATION
```

**GCE:**
```bash
gcloud compute instances add-labels INSTANCE_NAME \
  --labels instance-role=app-server,env=production \
  --zone ZONE
```

### Baseline Generation

Generate baselines from current infrastructure:

```bash
# Cloud SQL
./drift-analysis-cli gcp sql --generate-baseline --output sql-baseline.yaml

# GKE
./drift-analysis-cli gcp gke --generate-baseline --output gke-baseline.yaml

# GCE
./drift-analysis-cli gcp gce --generate-baseline --output gce-baseline.yaml
```

### CI/CD Integration

```bash
#!/bin/bash
set -e

# Run drift analysis
./drift-analysis-cli gcp gce --config config.yaml --output json > gce-drift.json

# Check for critical drifts
CRITICAL=$(jq '[.drifts[] | select(.severity=="CRITICAL")] | length' gce-drift.json)

if [ "$CRITICAL" -gt 0 ]; then
  echo "❌ Critical drifts detected!"
  jq '.drifts[] | select(.severity=="CRITICAL")' gce-drift.json
  exit 1
fi

echo "✅ No critical drifts detected"
```

### Daily Compliance Checks

```bash
#!/bin/bash
DATE=$(date +%Y%m%d)
REPORT_DIR="./reports"

mkdir -p "$REPORT_DIR"

# Run all analyses
./drift-analysis-cli gcp sql --config config.yaml --output json \
  --log-file "$REPORT_DIR/sql-$DATE.log" > "$REPORT_DIR/sql-drift-$DATE.json"

./drift-analysis-cli gcp gke --config config.yaml --output json \
  --log-file "$REPORT_DIR/gke-$DATE.log" > "$REPORT_DIR/gke-drift-$DATE.json"

./drift-analysis-cli gcp gce --config config.yaml --output json \
  --log-file "$REPORT_DIR/gce-$DATE.log" > "$REPORT_DIR/gce-drift-$DATE.json"

echo "✅ Compliance reports generated in $REPORT_DIR"
```

### Multi-Environment Analysis

```bash
# Development
./drift-analysis-cli gcp gce --config dev-config.yaml --output json

# Staging
./drift-analysis-cli gcp gce --config staging-config.yaml --output json

# Production
./drift-analysis-cli gcp gce --config prod-config.yaml --output json
```

## Development

```bash
# Run tests
go test ./...

# Run with race detection
go test -race ./...

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o drift-analysis-cli-linux
GOOS=darwin GOARCH=arm64 go build -o drift-analysis-cli-mac
GOOS=windows GOARCH=amd64 go build -o drift-analysis-cli.exe

# Run linters
golangci-lint run
```

### Project Structure

```
drift-analysis-cli/
├── main.go                    # CLI entry point
├── cmd/                       # Cobra commands
│   ├── root.go               # Root command with global flags
│   ├── gcp.go                # GCP parent command
│   ├── gcp_sql.go            # Cloud SQL drift command
│   ├── gcp_sql_db.go         # Database inspection command
│   ├── gcp_gke.go            # GKE drift command
│   └── gcp_gce.go            # GCE drift command
├── pkg/
│   ├── gcp/
│   │   ├── sql/              # Cloud SQL package
│   │   │   ├── analyzer.go  # Instance discovery & drift analysis
│   │   │   ├── comparators.go # Drift comparison logic
│   │   │   ├── report.go    # Report formatting
│   │   │   ├── inspector.go # Database inspection
│   │   │   ├── cache.go     # Schema caching
│   │   │   └── proxy.go     # Cloud SQL Proxy
│   │   ├── gke/              # GKE package
│   │   │   ├── analyzer.go  # Cluster discovery & drift analysis
│   │   │   ├── comparators.go # Drift comparison logic
│   │   │   └── report.go    # Report formatting
│   │   └── gce/              # GCE package
│   │       ├── analyzer.go  # Instance discovery & drift analysis
│   │       ├── comparators.go # Drift comparison logic with severity
│   │       ├── extractor.go # Config extraction from API
│   │       └── report.go    # Report formatting
│   ├── logger/               # Structured logging
│   │   └── logger.go        # JSON and text logging
│   └── tui/                  # Terminal UI
│       ├── model.go         # Bubble Tea model
│       └── converters.go    # Data converters
├── config.yaml               # Your configuration (gitignored)
├── config.yaml.example       # Example configuration
└── README.md                 # This file
```

## Error Handling

The CLI follows Go and Cobra best practices for error handling:

**Error Wrapping**: All errors are wrapped with context using `fmt.Errorf(..., %w, err)` to preserve error chains.

**Structured Logging**: Errors are logged with contextual metadata before being returned.

**Clean Error Messages**: Runtime errors show clear messages without usage spam (using `SilenceUsage: true`).

**Example Error Flow**:
```go
configData, err := os.ReadFile(cfgFile)
if err != nil {
    logger.Error("Failed to read config file", err, map[string]interface{}{
        "file": cfgFile,
    })
    return fmt.Errorf("failed to read config file: %w", err)
}
```

**Error Categories**:
- **Validation Errors**: Missing required flags, invalid values
- **Runtime Errors**: API failures, network issues, file I/O
- **Partial Failures**: Some resources succeed, others fail

For more details, see the inline comments in `cmd/` and `pkg/` packages.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Example Output

### Cloud SQL Drift Report

```
═══════════════════════════════════════════════════════════════════════════════
  GCP PostgreSQL Drift Analysis Report
═══════════════════════════════════════════════════════════════════════════════

Generated: 2026-01-12T16:00:00-08:00
Total Instances: 5
Instances with Drift: 3
Compliance Rate: 40.0%

Drift Summary
  [!] CRITICAL:   2
  [!] HIGH:       4
  [*] MEDIUM:     7
  [-] LOW:        3

───────────────────────────────────────────────────────────────────────────────
 Cloud SQL Instance: production-db-1 

Project:  my-project-123
Region:   us-central1
State:    RUNNABLE
Role:     application

Detected Drifts: 3

  [!] [CRITICAL] settings.ip_configuration.require_ssl
      Expected: true
      Actual:   false

  [!] [HIGH] settings.availability_type
      Expected: REGIONAL
      Actual:   ZONAL

  [*] [MEDIUM] database_flags.max_connections
      Expected: 200
      Actual:   100

Recommendations:
  - Enable SSL requirement to secure connections
  - Consider REGIONAL availability for production workloads
  - Review connection pool settings
```

### GCE Drift Report

```
═══════════════════════════════════════════════════════════════════════════════
  GCE Drift Analysis Report
═══════════════════════════════════════════════════════════════════════════════

Generated: 2026-01-12T16:00:00-08:00
Total Instances: 12
Instances with Drift: 3
Compliance Rate: 75.0%

Drift Summary
  [!] CRITICAL:   1
  [!] HIGH:       2
  [*] MEDIUM:     4
  [-] LOW:        2

───────────────────────────────────────────────────────────────────────────────
 Instance: app-server-1 

Project:  my-project
Zone:     us-west1-a
Status:   RUNNING

Detected Drifts: 2

  [!] [CRITICAL] network_config.external_ip
      Expected: false (no external IP)
      Actual:   35.203.XXX.XXX (external IP present)
      
      ⚠️  SECURITY ISSUE: Instance has unexpected external IP exposure

  [!] [HIGH] vm_config.preemptible
      Expected: false
      Actual:   true
      
      ⚠️  Production instance is preemptible - may be terminated at any time

Recommendations:
  - Remove external IP to improve security posture
  - Convert to non-preemptible instance for production reliability
```

### Database Schema Comparison

```
═══════════════════════════════════════════════════════════════════════════════
  Database Schema Drift Report
═══════════════════════════════════════════════════════════════════════════════

Connection: production-database
Database:   mydb
Timestamp:  2026-01-12T16:00:00-08:00

Schema Comparison Results
  Tables:     124 (expected: 124) ✓
  Views:      2   (expected: 2)   ✓
  Sequences:  16  (expected: 15) ⚠️ +1
  Functions:  8   (expected: 8)   ✓
  Procedures: 3   (expected: 3)   ✓
  Roles:      13  (expected: 12) ⚠️ +1
  Extensions: 2   (expected: 2)   ✓

Detected Changes: 3

  [!] New Sequence Added
      Name: user_activity_logs_id_seq
      Owner: postgres
      
  [!] New Role Added
      Name: reporting_user
      Superuser: false
      
  [*] Table Ownership Changed
      Table: audit_log
      Expected Owner: audit_user
      Actual Owner: postgres

Recommendations:
  - Review new sequence user_activity_logs_id_seq
  - Document purpose of new role reporting_user
  - Verify audit_log ownership change is intentional
```
