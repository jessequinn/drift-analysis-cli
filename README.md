# GCP Drift Analysis CLI

[![CI](https://github.com/jessequinn/drift-analysis-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/jessequinn/drift-analysis-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jessequinn/drift-analysis-cli)](https://goreportcard.com/report/github.com/jessequinn/drift-analysis-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A comprehensive CLI tool for detecting configuration drift across Google Cloud Platform resources including Cloud SQL PostgreSQL instances and GKE clusters.

## Features

- Deep Drift Analysis: Compares resource configurations against defined baselines
- Multi-Project Support: Analyze resources across multiple GCP projects
- Multi-Resource Support: Cloud SQL and GKE cluster analysis
- Comprehensive Checks: Analyzes versions, configurations, security, networking, and more
- Security Recommendations: Identifies security gaps and misconfigurations
- Multiple Output Formats: Text, JSON, or YAML output
- Config Generation: Auto-generate baseline configs from existing resources
- Label-based Filtering: Target specific resource roles/types

## Installation

```bash
go mod download
go build -o drift-analysis-cli
```

## Quick Start

### Cloud SQL Analysis

```bash
# Analyze with baseline config
./drift-analysis-cli sql -config config.yaml

# Analyze specific projects
./drift-analysis-cli sql -projects "project-1,project-2"

# Filter by role
./drift-analysis-cli sql -projects "project-1" -filter-role application

# Generate baseline
./drift-analysis-cli sql -projects "project-1" -generate-config -output baseline.yaml

# Export as JSON
./drift-analysis-cli sql -config config.yaml -format json -output report.json
```

### GKE Analysis

```bash
# Analyze with baseline config
./drift-analysis-cli gke -config config.yaml

# Analyze specific projects
./drift-analysis-cli gke -projects "project-1,project-2"

# Filter by cluster role
./drift-analysis-cli gke -projects "project-1" -filter-role production

# Generate baseline
./drift-analysis-cli gke -projects "project-1" -generate-config -output baseline.yaml
```

## Configuration File Format

Create a unified `config.yaml` file for both SQL and GKE:

```yaml
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
```

## Cloud SQL Checks

### Core Configuration
- PostgreSQL version
- Machine tier (CPU/Memory)
- Disk size, type, and autoresize settings

### Database Flags
- All PostgreSQL configuration parameters
- Performance tuning settings
- Connection limits

### High Availability & Reliability
- Availability type (ZONAL vs REGIONAL)
- Backup configuration and retention
- Point-in-time recovery
- Transaction log retention

### Security
- SSL/TLS requirements
- Public vs private IP
- Authorized networks (Required/Extra detection)
- IAM authentication

### Observability
- Query Insights configuration
- Performance monitoring settings

### Database Validation
- Required databases present
- Extra databases detected

## GKE Checks

### Networking (9 checks)
- Network/Subnetwork configuration
- Private cluster settings
- Master global access
- Master authorized networks (Required/Extra detection)
- Datapath provider (ADVANCED vs LEGACY)
- IP allocation policy (IPv4/IPv6 stack)
- Cluster and services CIDR blocks

### Security (6 checks)
- Shielded nodes
- Database encryption (ETCD at rest)
- Security posture (BASIC/ENTERPRISE)
- Workload identity
- Binary authorization
- Network policy

### Features & Observability (10+ checks)
- System and workload logging
- System, API server, controller, and scheduler metrics
- Kubernetes version and release channel
- HTTP load balancing addon
- Horizontal pod autoscaling addon
- Node pool configuration (machine type, disk, auto-upgrade, auto-repair)

## Severity Levels

- CRITICAL: Security issues, disabled backups, encryption problems
- HIGH: HA configuration, PITR, major version drift, security features
- MEDIUM: Performance settings, resource tiers, network configuration
- LOW: Optimization suggestions, monitoring config

## Example Output

```
===============================================================================
  GCP PostgreSQL Drift Analysis Report
===============================================================================

Generated: 2025-12-29T13:55:00Z
Total Instances: 5
Instances with Drift: 3
Compliance Rate: 40.0%

Drift Summary:
  [!] CRITICAL: 2
  [!] HIGH:     4
  [*] MEDIUM:   7
  [-] LOW:      3

-------------------------------------------------------------------------------
Instance: production-db-1
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

## Authentication

The CLI uses Application Default Credentials (ADC). Set up authentication:

```bash
# Option 1: User credentials
gcloud auth application-default login

# Option 2: Service account
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
```

### Required IAM Permissions

**For Cloud SQL:**
- `cloudsql.instances.get`
- `cloudsql.instances.list`
- `cloudsql.databases.list`

Or the predefined role: `roles/cloudsql.viewer`

**For GKE:**
- `container.clusters.get`
- `container.clusters.list`

Or the predefined role: `roles/container.viewer`

## Command Line Options

### SQL Command
```
-projects string      Comma-separated list of GCP project IDs
-config string        Path to unified YAML config file
-output string        Output file path (default: stdout)
-format string        Output format: text, json, yaml (default: text)
-filter-role string   Filter instances by database-role label
-generate-config      Generate baseline config from current state
```

### GKE Command
```
-projects string      Comma-separated list of GCP project IDs
-config string        Path to unified YAML config file
-output string        Output file path (default: stdout)
-format string        Output format: text, json, yaml (default: text)
-filter-role string   Filter clusters by cluster-role label
-generate-config      Generate baseline config from current state
```

## Label-based Filtering

### Cloud SQL
Apply labels to your Cloud SQL instances:

```bash
gcloud sql instances patch INSTANCE_NAME \
  --update-labels database-role=application
```

Recommended labels:
- `application` - Main application databases
- `microservices` - Microservice-specific databases
- `vault` - HashiCorp Vault databases
- `monitoring` - Monitoring/observability databases

### GKE
Apply labels to your GKE clusters:

```bash
gcloud container clusters update CLUSTER_NAME \
  --update-labels cluster-role=production \
  --location LOCATION
```

Recommended labels:
- `production` - Production clusters
- `staging` - Staging clusters
- `development` - Development clusters

## Use Cases

### Daily Compliance Checks
```bash
./drift-analysis-cli sql -config config.yaml -format json -output reports/sql-drift-$(date +%Y%m%d).json
./drift-analysis-cli gke -config config.yaml -format json -output reports/gke-drift-$(date +%Y%m%d).json
```

### Multi-Environment Baseline Generation
```bash
./drift-analysis-cli sql -projects "dev-proj" -generate-config -output dev-sql-baseline.yaml
./drift-analysis-cli gke -projects "prod-proj" -generate-config -output prod-gke-baseline.yaml
```

### CI/CD Integration
```bash
#!/bin/bash
./drift-analysis-cli sql -config config.yaml -format json -output sql-drift.json
DRIFTED=$(jq '.drifted_instances' sql-drift.json)
if [ "$DRIFTED" -gt 0 ]; then
  echo "SQL drift detected! Review required."
  exit 1
fi
```

## CI/CD

This project includes comprehensive GitHub Actions workflows:

### Continuous Integration (CI)
Runs on every push and pull request to `main` and `develop` branches:

**Lint Job:**
- golangci-lint with 20+ linters enabled
- Code style and quality checks

**Security Job:**
- `govulncheck` - Scan for known vulnerabilities
- `gosec` - Security analysis for Go code

**Test Job:**
- Tests run on Go 1.23 and 1.24
- Race condition detection
- Code coverage reporting to Codecov

**Build Job:**
- Multi-platform builds (Linux, macOS, Windows)
- Multi-architecture (amd64, arm64)
- Build artifacts uploaded for 7 days

**Validate Job:**
- Code formatting check (`gofmt`)
- `go vet` static analysis
- Ineffectual assignment detection
- Error checking with `errcheck`
- Additional static analysis with `staticcheck`

### Release Workflow
Automatically creates releases when pushing version tags:

```bash
# Create and push a new release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

This will:
- Build binaries for all platforms
- Generate SHA256 checksums
- Create a GitHub release with assets
- Auto-generate release notes

### Local Development

```bash
# Format code
go fmt ./...

# Run linter locally
golangci-lint run

# Run vulnerability check
govulncheck ./...

# Run all tests
go test -v -race -coverprofile=coverage.txt ./...

# Build for current platform
go build -v -o drift-analysis-cli
```

## Development

```bash
# Run tests
go test ./...

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o drift-analysis-cli-linux
GOOS=darwin GOARCH=arm64 go build -o drift-analysis-cli-mac
GOOS=windows GOARCH=amd64 go build -o drift-analysis-cli.exe
```

## Project Structure

```
drift-analysis-cli/
├── main.go                    # CLI entry point with command routing
├── pkg/
│   ├── csql/                 # Cloud SQL package
│   │   ├── analyzer.go       # Discovery & drift analysis
│   │   ├── command.go        # Command handler
│   │   └── report.go         # Report formatting
│   └── gke/                  # GKE package
│       ├── analyzer.go       # Discovery & drift analysis
│       ├── command.go        # Command handler
│       └── report.go         # Report formatting
├── config.yaml               # Your configuration (gitignored)
├── config.yaml.example       # Example configuration
└── README.md                 # This file
```

## License

MIT
