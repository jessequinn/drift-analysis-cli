# Troubleshooting Cloud SQL Private IP Connections

## Issue: Connection Timeout or Reset

If you're seeing errors like:
- `failed to ping database: i/o timeout`
- `read: connection reset by peer`
- Connection hangs indefinitely

## Common Causes

### 1. VPC Connectivity
Your machine needs network access to the Cloud SQL VPC.

**Check:**
```bash
# Are you connected to the VPC via VPN?
# OR running from a GCE VM in the same VPC?
# OR have VPC peering configured?
```

**Solution**: Connect via VPN or run from a VM in the VPC.

### 2. Private Service Connection
Private IP requires a Private Service Connection between your VPC and Cloud SQL.

**Check:**
```bash
gcloud services vpc-peerings list --network=YOUR_VPC_NETWORK
```

**Solution**: If not configured, Cloud SQL private IP won't work from your machine. Use public IP instead.

### 3. Firewall Rules
Even with VPC access, firewall rules might block the connection.

**Check:**
```bash
gcloud compute firewall-rules list --filter="direction=INGRESS"
```

**Solution**: Ensure firewall rules allow your IP to connect.

### 4. Database User Permissions
The database user might not have permission to connect.

**Check:**
```bash
# Connect via gcloud to verify credentials work
gcloud sql connect test-c3ac43e6 --user=postgres --database=postgres --private-ip
```

**Solution**: If this works, the issue is with the proxy configuration. If this fails, fix the database user.

## Recommended Approach

### Option 1: Use Public IP (Easier for Development)

Update your config:
```yaml
database_connections:
  - name: "test-db"
    instance_connection_name: "project:region:instance"
    database: "postgres"
    username: "postgres"
    password: "..."
    use_private_ip: false  # Set to false
```

Then ensure the Cloud SQL instance has:
- Public IP enabled
- Your IP in authorized networks

### Option 2: Run from GCE VM

SSH into a GCE VM in the same VPC as Cloud SQL:
```bash
gcloud compute ssh YOUR_VM --zone YOUR_ZONE
# Then run the CLI from there
```

### Option 3: Use gcloud SQL Proxy Manually

Test if the proxy works standalone:
```bash
# Start proxy
cloud-sql-proxy --port=5432 --private-ip project:region:instance

# In another terminal, test connection
PGPASSWORD="password" psql -h localhost -p 5432 -U postgres -d postgres -c "SELECT 1;"
```

If this works, there may be an issue with the CLI's proxy integration.
If this doesn't work, the problem is with your VPC connectivity or Cloud SQL configuration.

## Quick Test

```bash
# Test if you can reach Cloud SQL's private IP at all
# (Replace with actual private IP from Cloud Console)
ping PRIVATE_IP_OF_CLOUDSQL

# If ping fails, you don't have VPC connectivity
```

## For This Specific Instance

```bash
# Get instance details
gcloud sql instances describe test-c3ac43e6 --project=zpe-cloud-test-environment

# Check if it has public IP
gcloud sql instances describe test-c3ac43e6 --project=zpe-cloud-test-environment --format="value(ipAddresses)"

# Try connecting via gcloud (this uses public IP by default)
gcloud sql connect test-c3ac43e6 --user=postgres --database=postgres
```

## Still Not Working?

The most likely issue is that your machine doesn't have VPC network access to the Cloud SQL instance's private IP. Consider:
1. Using public IP instead
2. Running from a GCE VM
3. Setting up VPN to the VPC
4. Using Cloud SQL Auth proxy in a different way

For immediate testing, add a public IP to the instance and set `use_private_ip: false` in the config.
