# GKE Drift Analysis Enhancements

## Summary
Added 17 new comparison fields to GKE drift analysis, increasing coverage from 22 to 41 fields (86% increase).

## New Comparisons Added

### Cluster-Level (10 new fields)

#### Networking
1. `cluster.network` - VPC network validation (HIGH)
2. `cluster.subnetwork` - Subnet validation (HIGH)

#### IP Allocation Policy
3. `cluster.ip_allocation_policy.use_ip_aliases` - IP aliasing (CRITICAL)
4. `cluster.ip_allocation_policy.cluster_ipv4_cidr` - Pod IP range (HIGH)
5. `cluster.ip_allocation_policy.services_ipv4_cidr` - Service IP range (HIGH)

#### Monitoring
6. `cluster.monitoring_config.enable_controller_metrics` (LOW)
7. `cluster.monitoring_config.enable_scheduler_metrics` (LOW)

#### Addons
8. `cluster.addons.http_load_balancing` (LOW)
9. `cluster.addons.horizontal_pod_autoscaling` (LOW)
10. `cluster.addons.network_policy` (MEDIUM)

#### Maintenance Window
11. `cluster.maintenance_window.start_time` (LOW)
12. `cluster.maintenance_window.duration` (LOW)

### Node Pool-Level (7 new fields)

1. `nodepool.version` - K8s version for node pool (HIGH)
2. `nodepool.disk_type` - Storage type validation (MEDIUM)
3. `nodepool.service_account` - SA for workload permissions (HIGH)
4. `nodepool.initial_node_count` - Initial node count (LOW)
5. `nodepool.autoscaling.enabled` - Autoscaling status (HIGH)
6. `nodepool.autoscaling.min_node_count` - Min capacity (MEDIUM)
7. `nodepool.autoscaling.max_node_count` - Max capacity (MEDIUM)

## Complete Field List (41 fields)

### Cluster (29 fields)
- master_version, release_channel
- network, subnetwork, private_cluster, master_global_access, datapath_provider
- master_authorized_networks
- ip_allocation_policy (use_ip_aliases, cluster_ipv4_cidr, services_ipv4_cidr, stack_type)
- workload_identity, network_policy, binary_authorization, shielded_nodes
- database_encryption, security_posture
- logging_config (enable_system_logs, enable_workload_logs)
- monitoring_config (enable_system_metrics, enable_apiserver_metrics, enable_controller_metrics, enable_scheduler_metrics)
- addons (http_load_balancing, horizontal_pod_autoscaling, network_policy)
- maintenance_window (start_time, duration)

### Node Pool (12 fields)
- version, machine_type, disk_size_gb, disk_type, image_type
- service_account, initial_node_count
- autoscaling (enabled, min_node_count, max_node_count)
- auto_upgrade, auto_repair

## Files Modified

1. **pkg/gcp/gke/analyzer.go**
   - Enhanced `compareNetworking()` - added network/subnetwork
   - Enhanced `compareIPAllocation()` - added use_ip_aliases, CIDR ranges
   - Enhanced `compareMonitoringCluster()` - added controller/scheduler metrics
   - Added `compareAddons()` - new function for addon comparison
   - Added `compareMaintenanceWindow()` - new function for maintenance window
   - Enhanced `compareNodePools()` - added version, disk_type, service_account, autoscaling, initial_node_count

2. **config.yaml**
   - Updated GKE baseline with all new fields
   - Added comprehensive examples for networking, IP allocation, addons, maintenance window
   - Added node pool autoscaling configuration

3. **config.yaml.example**
   - Updated with same comprehensive GKE baseline
   - Serves as reference for all available configuration options

## Key Features

### Improved Coverage
- **Network Validation**: Ensures clusters use correct VPC/subnet
- **IP Range Management**: Validates pod and service IP allocations
- **Autoscaling Control**: Monitors capacity management configuration
- **Version Skew Detection**: Checks node pool K8s versions
- **Service Account Validation**: Ensures proper workload permissions

### Severity Levels
- **CRITICAL (1)**: IP aliasing configuration changes
- **HIGH (8)**: Network, subnet, IP ranges, version, service account, autoscaling
- **MEDIUM (6)**: Disk type, autoscaling limits, network policy addon
- **LOW (8)**: Monitoring metrics, HTTP LB, HPA, maintenance window, initial node count

## Testing Checklist
- [ ] Verify network/subnetwork comparison works
- [ ] Test IP allocation policy validation
- [ ] Confirm autoscaling configuration detection
- [ ] Validate node pool version comparison
- [ ] Check service account drift detection
- [ ] Test addon configuration comparison
- [ ] Verify maintenance window comparison
- [ ] Ensure severity levels are appropriate
