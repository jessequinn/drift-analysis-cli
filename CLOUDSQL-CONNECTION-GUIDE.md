# Cloud SQL Database Connection Configuration

This guide explains how to configure database connections for Cloud SQL instances with private IP addresses using the Cloud SQL Auth Proxy.

## Overview

For Cloud SQL instances with **only private IP** (no public IP), the CLI automatically manages the Cloud SQL Auth Proxy to create a secure tunnel. The proxy runs in the background during the connection and is automatically cleaned up afterward.

## How It Works

1. **Automatic Proxy Management**: When you configure a connection with `use_private_ip: true`, the CLI:
 - Starts `cloud-sql-proxy` in the background
 - Waits for the proxy to be ready
 - Connects through the proxy (localhost:5432)
 - Automatically stops the proxy when done

2. **No Manual Proxy Setup**: Unlike running `gcloud sql connect` or `cloud-sql-proxy` manually, the CLI handles everything for you.

## Connection Configuration

You can now specify database connection information in your `config.yaml` to enable database inspection and drift analysis using the Cloud SQL Proxy.

### Configuration Structure

```yaml
sql_baselines:
 - name: "your-baseline-name"
 filter_labels:
 database-role: "application"

 # Database connection information
 connection:
 # Option 1: Full connection name
 instance_connection_name: "project:region:instance"

 # Option 2: Separate components (will be combined automatically)
 # project: "my-project"
 # region: "us-central1"
 # instance_name: "my-instance"

 database: "postgres" # Database to connect to
 username: "postgres" # Database username
 password: "" # Password (leave empty for IAM auth)
 use_private_ip: true # Use private IP connection

 config:
 # ... baseline configuration ...
```

## Example: Connecting to Private IP Instance

Based on the command:
```bash
gcloud alpha sql connect test-c3ac43e6 \
 --database=postgres \
 --private-ip \
 --project=zpe-cloud-test-environment
```

Your config would be:

```yaml
projects:
 - zpe-cloud-test-environment

sql_baselines:
 - name: "test-instance"
 connection:
 instance_connection_name: "zpe-cloud-test-environment:us-central1:test-c3ac43e6"
 database: "postgres"
 username: "postgres"
 password: "" # Leave empty for IAM auth
 use_private_ip: true

 config:
 database_version: POSTGRES_15
 # ... other baseline settings ...
```

## Prerequisites

1. **VPC Connectivity**: Your machine must have network access to the VPC where the Cloud SQL instance is located
 - Via VPN
 - Via Cloud VPN
 - Via Interconnect
 - From a GCE VM in the same VPC

2. **Authentication**: Authenticate with gcloud:
 ```bash
 gcloud auth application-default login
 ```

3. **Permissions**: Your service account needs:
 - `cloudsql.instances.connect` permission
 - Database user permissions in PostgreSQL

4. **Private Service Connection**: Must be configured between your VPC and Cloud SQL

## Finding Your Instance Connection Name

To get the full connection name including region:

```bash
gcloud sql instances describe test-c3ac43e6 \
 --project=zpe-cloud-test-environment \
 --format="value(connectionName)"
```

This returns: `project:region:instance` (e.g., `zpe-cloud-test-environment:us-central1:test-c3ac43e6`)

## Authentication Options

### IAM Authentication (Recommended)
Leave `password` empty and ensure IAM authentication is enabled:
```yaml
connection:
 password: "" # Empty for IAM auth
```

### Password Authentication
Provide the password directly:
```yaml
connection:
 password: "your-secure-password"
```

**Security Note**: For production, use environment variables or secret managers instead of hardcoding passwords.

## Usage

Once configured, use the CLI to:

### 1. Analyze Drift
```bash
./drift-analysis-cli sql -config config.yaml
```

### 2. Inspect Database Schema
```bash
./drift-analysis-cli sql -config config.yaml -inspect
```

### 3. Generate Baseline Configuration
```bash
./drift-analysis-cli sql -config config.yaml -generate-config
```

## Troubleshooting

### Connection Timeout
- Verify VPC connectivity
- Check firewall rules allow egress to Cloud SQL
- Ensure Private Service Connection is active

### Authentication Failed
- For IAM auth: Run `gcloud auth application-default login`
- For password auth: Verify credentials
- Check database user exists: `gcloud sql users list --instance=INSTANCE`

### "Private IP not enabled"
The instance must have private IP configured:
```bash
gcloud sql instances patch INSTANCE \
 --network=projects/PROJECT/global/networks/NETWORK
```

## See Also

- [DATABASE-INSPECTION-GUIDE.md](DATABASE-INSPECTION-GUIDE.md) - Database inspection features
- [config-cloudsql-example.yaml](config-cloudsql-example.yaml) - Full example configuration
- [Google Cloud SQL Proxy Documentation](https://cloud.google.com/sql/docs/postgres/connect-instance-auth-proxy)
