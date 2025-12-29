# Cloud SQL Private IP Connection - Implementation Summary

## Problem
Cloud SQL instances with only private IP addresses (no public IP) cannot be connected to directly. They require either:
1. Running `gcloud sql connect` manually with `--private-ip` flag
2. Running `cloud-sql-proxy` with `--private-ip` flag manually
3. Being in the same VPC network

## Solution
The CLI now **automatically manages the Cloud SQL Auth Proxy** for private IP connections. When you configure a connection with `use_private_ip: true`, the proxy is:
- Started automatically in the background
- Managed throughout the database inspection/analysis
- Stopped automatically when done

## Architecture

```
User Config (config.yaml)
 ↓
ConnectionConfig (use_private_ip: true)
 ↓
NewInspectorFromConnectionConfig()
 ↓
NewInspectorWithProxy() 
 ↓
ProxyManager.Start()
 ├─ Spawns: cloud-sql-proxy --port=5432 --private-ip PROJECT:REGION:INSTANCE
 ├─ Waits for proxy to be ready (8 seconds)
 └─ Returns control to inspector
 ↓
DatabaseInspector.InspectDatabase()
 ├─ Connects to localhost:5432 (proxy endpoint)
 ├─ Proxy tunnels to Cloud SQL private IP
 ├─ Performs database inspection
 └─ Returns schema
 ↓
ProxyManager.Stop()
 └─ Kills proxy process
```

## Key Components

### 1. ProxyManager (`pkg/gcp/sql/proxy.go`)
Manages the Cloud SQL Proxy lifecycle:
- **Start()**: Spawns `cloud-sql-proxy` with appropriate flags
- **Stop()**: Terminates the proxy process
- **IsRunning()**: Checks if proxy is active

Supports two modes:
- `cloud-sql-proxy` binary (default, recommended)
- `gcloud beta sql connect` (alternative)

### 2. DatabaseInspector (`pkg/gcp/sql/inspector.go`)
Enhanced to support proxy management:
- **NewInspectorWithProxy()**: Creates inspector with proxy
- **InspectDatabase()**: Starts proxy before connecting, stops after

### 3. ConnectionConfig (`pkg/gcp/sql/command.go`)
Configuration structure:
```yaml
connection:
 instance_connection_name: "project:region:instance"
 database: "postgres"
 username: "postgres"
 password: "password"
 use_private_ip: true # Triggers automatic proxy management
```

## Usage Example

### Config File (config.yaml)
```yaml
projects:
 - zpe-cloud-test-environment

sql_baselines:
 - name: "application"
 connection:
 instance_connection_name: "zpe-cloud-test-environment:us-west1:test-c3ac43e6"
 database: "postgres"
 username: "postgres"
 password: "kXJPFKWpH0"
 use_private_ip: true
 project: "zpe-cloud-test-environment"

 config:
 database_version: POSTGRES_14
 # ... baseline config ...
```

### Running
```bash
./drift-analysis-cli sql -config config.yaml
```

Output:
```
Starting Cloud SQL Proxy for zpe-cloud-test-environment:us-west1:test-c3ac43e6...
Started cloud-sql-proxy (PID: 12345), waiting for it to be ready...
 Proxy process is running and ready
 Proxy started successfully
[... database inspection/analysis ...]
Stopping Cloud SQL Proxy...
```

## Prerequisites

1. **cloud-sql-proxy** binary installed:
 ```bash
 gcloud components install cloud-sql-proxy
 # or download from: https://github.com/GoogleCloudPlatform/cloud-sql-proxy
 ```

2. **VPC Connectivity**: Your machine must have network access to the Cloud SQL VPC:
 - Via VPN
 - Via Cloud VPN/Interconnect 
 - From a GCE VM in the same VPC

3. **Authentication**:
 ```bash
 gcloud auth application-default login
 ```

4. **IAM Permissions**:
 - `cloudsql.instances.connect`
 - Database user permissions

## Comparison: Before vs After

### Before (Manual Process)
```bash
# Terminal 1: Start proxy manually
cloud-sql-proxy --port=5432 --private-ip zpe-cloud-test-environment:us-west1:test-c3ac43e6

# Terminal 2: Run CLI (connects to localhost:5432)
./drift-analysis-cli sql -config config.yaml

# Terminal 1: Stop proxy manually (Ctrl+C)
```

### After (Automatic)
```bash
# Single command - proxy managed automatically
./drift-analysis-cli sql -config config.yaml
```

## Benefits

1. **Zero Manual Steps**: No need to run proxy separately
2. **Automatic Cleanup**: Proxy is always stopped, even on errors
3. **Consistent Experience**: Same command works for both public and private IP instances
4. **Error Handling**: Proxy failures are caught and reported clearly
5. **Resource Management**: No orphaned proxy processes

## Technical Details

### Proxy Command
```bash
cloud-sql-proxy \
 --port=5432 \
 --private-ip \
 zpe-cloud-test-environment:us-west1:test-c3ac43e6
```

### Connection String (Internal)
```
host=localhost port=5432 user=postgres password=XXX dbname=postgres sslmode=disable connect_timeout=30
```

### Process Management
- Proxy spawned with `exec.CommandContext()`
- PID tracked for lifecycle management
- Killed with `Process.Kill()` on cleanup
- 8-second startup wait for initialization

## Future Enhancements

Potential improvements:
1. **Health Checks**: Verify proxy is accepting connections before proceeding
2. **Port Auto-Selection**: Use random available port if 5432 is busy
3. **Proxy Reuse**: Share proxy across multiple connections
4. **Log Capture**: Capture and display proxy logs for debugging
5. **Fallback**: Auto-retry with different connection methods on failure

## Files Modified

1. `pkg/gcp/sql/proxy.go` - NEW: Proxy manager implementation
2. `pkg/gcp/sql/inspector.go` - Enhanced with proxy support
3. `pkg/gcp/sql/command.go` - Added ConnectionConfig
4. `config.yaml.example` - Updated with connection examples
5. `CLOUDSQL-CONNECTION-GUIDE.md` - NEW: User documentation

## Testing

Test the implementation:
```bash
# Create test script
cat > test-proxy.go << 'EOF'
package main
import (
 "context"
 "github.com/jessequinn/drift-analysis-cli/pkg/gcp/sql"
)
func main() {
 config := &sql.ConnectionConfig{
 InstanceConnectionName: "project:region:instance",
 Database: "postgres",
 Username: "postgres", 
 Password: "password",
 UsePrivateIP: true,
 }
 inspector, _ := sql.NewInspectorFromConnectionConfig(config)
 schema, _ := inspector.InspectDatabase(context.Background())
 println("Connected! Database:", schema.DatabaseName)
}
EOF

go run test-proxy.go
```

## Troubleshooting

### "Failed to start cloud-sql-proxy"
- Ensure `cloud-sql-proxy` is installed and in PATH
- Try: `gcloud components install cloud-sql-proxy`

### "Connection refused"
- Proxy may need more time to start (increase wait time)
- Check VPC connectivity
- Verify private IP is enabled on instance

### "Authentication failed"
- Run: `gcloud auth application-default login`
- Verify database user/password
- Check IAM permissions

### "Timeout: context deadline exceeded"
- Increase context timeout
- Check network connectivity to VPC
- Verify Private Service Connection is configured

## References

- [Cloud SQL Auth Proxy](https://cloud.google.com/sql/docs/postgres/sql-proxy)
- [Private IP for Cloud SQL](https://cloud.google.com/sql/docs/postgres/private-ip)
- [VPC Peering](https://cloud.google.com/vpc/docs/vpc-peering)
