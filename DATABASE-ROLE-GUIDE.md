# Database Role Labeling Guide

This guide explains the recommended label-based organization for Cloud SQL instances and GKE clusters.

## Overview

Labels enable you to:
- Filter resources by role or purpose
- Apply different baselines to different resource types
- Organize multi-tenant or multi-application environments
- Generate role-specific drift reports

## Cloud SQL Instance Labels

### Recommended Label: `database-role`

Apply labels using gcloud:
```bash
gcloud sql instances patch INSTANCE_NAME \
 --update-labels database-role=ROLE_VALUE \
 --project PROJECT_ID
```

### Recommended Roles

#### 1. Application Databases (`application`)
Primary databases for web applications or services.

**Characteristics:**
- Medium to high availability requirements
- Regular backups with 7-day retention
- Moderate to high performance needs
- SSL recommended
- Query insights enabled

**Example:**
```bash
gcloud sql instances patch main-app-db \
 --update-labels database-role=application
```

#### 2. Microservices Databases (`microservices`)
Databases for microservice architectures.

**Characteristics:**
- Service-specific databases
- Independent scaling
- May include time-series or specialized extensions
- Lower individual retention requirements
- High isolation needs

**Example:**
```bash
gcloud sql instances patch user-service-db \
 --update-labels database-role=microservices
```

#### 3. Vault Databases (`vault`)
HashiCorp Vault or secrets management databases.

**Characteristics:**
- High availability (REGIONAL)
- Extended backup retention (30+ days)
- Strict security requirements
- No public IP access
- Limited authorized networks

**Example:**
```bash
gcloud sql instances patch vault-backend \
 --update-labels database-role=vault
```

#### 4. Monitoring/Observability (`monitoring`)
Databases for monitoring tools, metrics, or test data.

**Characteristics:**
- Can be lower availability
- Shorter or no backup retention
- Test/development use
- Less strict security requirements

**Example:**
```bash
gcloud sql instances patch cloudprober-db \
 --update-labels database-role=monitoring
```

### Viewing Current Labels

```bash
# View labels for a specific instance
gcloud sql instances describe INSTANCE_NAME \
 --format="get(settings.userLabels)"

# List all instances with their labels
gcloud sql instances list \
 --format="table(name,settings.userLabels)"
```

## GKE Cluster Labels

### Recommended Label: `cluster-role`

Apply labels using gcloud:
```bash
gcloud container clusters update CLUSTER_NAME \
 --update-labels cluster-role=ROLE_VALUE \
 --location LOCATION \
 --project PROJECT_ID
```

### Recommended Roles

#### 1. Production Clusters (`production`)
Production workload clusters.

**Characteristics:**
- High availability and reliability
- Strict security settings (shielded nodes, workload identity)
- Advanced networking (GKE Dataplane V2)
- Full logging and monitoring
- Regular maintenance windows

**Example:**
```bash
gcloud container clusters update prod-us-central1 \
 --update-labels cluster-role=production \
 --location us-central1
```

#### 2. Staging Clusters (`staging`)
Pre-production testing environments.

**Characteristics:**
- Similar to production but lower resource allocations
- Full security features for testing
- May have relaxed SLAs
- Cost optimization enabled

**Example:**
```bash
gcloud container clusters update staging-cluster \
 --update-labels cluster-role=staging \
 --location us-west1
```

#### 3. Development Clusters (`development`)
Development and testing clusters.

**Characteristics:**
- Rapid release channel
- Lower resource allocations
- May have relaxed security for testing
- Frequent updates enabled

**Example:**
```bash
gcloud container clusters update dev-cluster \
 --update-labels cluster-role=development \
 --location us-east1
```

### Viewing Current Labels

```bash
# View labels for a specific cluster
gcloud container clusters describe CLUSTER_NAME \
 --location LOCATION \
 --format="get(resourceLabels)"

# List all clusters with their labels
gcloud container clusters list \
 --format="table(name,location,resourceLabels)"
```

## Configuration Example

### Multi-Baseline Config with Labels

```yaml
projects:
 - my-project-1
 - my-project-2

# Cloud SQL baselines by role
sql_baselines:
 - name: "application"
 filter_labels:
 database-role: "application"
 config:
 database_version: POSTGRES_15
 tier: db-custom-4-16384
 settings:
 availability_type: REGIONAL
 backup_retention_days: 7

 - name: "vault"
 filter_labels:
 database-role: "vault"
 config:
 database_version: POSTGRES_15
 settings:
 availability_type: REGIONAL
 backup_retention_days: 30
 ip_configuration:
 ipv4_enabled: false

 - name: "monitoring"
 filter_labels:
 database-role: "monitoring"
 config:
 settings:
 backup_enabled: false

# GKE baselines by role
gke_baselines:
 - name: "production"
 filter_labels:
 cluster-role: "production"
 cluster_config:
 release_channel: REGULAR
 shielded_nodes: true
 workload_identity: true
 security_posture: ENTERPRISE

 - name: "development"
 filter_labels:
 cluster-role: "development"
 cluster_config:
 release_channel: RAPID
 shielded_nodes: true
```

## Filtering by Role

### SQL Analysis
```bash
# Analyze only application databases
drift-analysis-cli sql -projects "my-project" -filter-role application

# Analyze only vault databases
drift-analysis-cli sql -projects "my-project" -filter-role vault
```

### GKE Analysis
```bash
# Analyze only production clusters
drift-analysis-cli gke -projects "my-project" -filter-role production

# Analyze only development clusters
drift-analysis-cli gke -projects "my-project" -filter-role development
```

## Best Practices

1. **Consistent Naming**: Use the same label values across projects
2. **Document Roles**: Maintain documentation of what each role means
3. **Apply on Creation**: Label resources when creating them
4. **Regular Audits**: Periodically check that all resources are labeled
5. **Automation**: Use Terraform or other IaC to enforce labeling
6. **Multiple Labels**: Combine with other labels (environment, team, cost-center)

## Custom Roles

You can define your own roles based on your organization's needs:

```yaml
sql_baselines:
 - name: "analytics"
 filter_labels:
 database-role: "analytics"
 config:
 # Analytics-specific configuration

 - name: "reporting"
 filter_labels:
 database-role: "reporting"
 config:
 # Reporting-specific configuration
```

## Troubleshooting

### No Instances Found
If filtering returns no instances:
1. Verify labels are correctly applied
2. Check label key and value match exactly (case-sensitive)
3. Ensure projects specified are correct

### Mixed Results
If some instances match and others don't:
1. Review which instances have which labels
2. Consider if multiple baselines are needed
3. Check for typos in label values

## Additional Labels

Consider combining with these standard GCP labels:
- `environment`: dev, staging, prod
- `team`: team-name
- `cost-center`: accounting code
- `application`: application name
- `managed-by`: terraform, manual, etc.

Example:
```bash
gcloud sql instances patch my-db \
 --update-labels database-role=application,environment=production,team=backend
```
