# Schema Baseline Configuration Guide

## Complete Schema Baseline Example

```yaml
database_connections:
  - name: "production-database"
    instance_connection_name: "project:region:instance"
    database: "mydb"
    username: "postgres"
    password: "secret"
    use_private_ip: true
    
    ssh_tunnel:
      enabled: true
      bastion_host: "bastion"
      bastion_zone: "us-west1-a"
      project: "my-project"
      private_ip: "10.50.0.3"
      use_iap: true
    
    # Complete schema baseline configuration
    schema_baseline:
      # ===================================================================
      # OBJECT COUNT VALIDATION
      # ===================================================================
      # Expected counts for each object type
      # Set to exact number you expect in the database
      # Drift is detected if actual count doesn't match
      
      expected_tables: 124        # Number of tables
      expected_views: 2           # Number of views
      expected_sequences: 15      # Number of sequences
      expected_functions: 8       # Number of functions
      expected_procedures: 3      # Number of stored procedures
      expected_roles: 12          # Number of database roles
      expected_extensions: 2      # Number of PostgreSQL extensions
      
      # ===================================================================
      # OWNERSHIP VALIDATION
      # ===================================================================
      # Ensures proper ownership of database objects
      # Critical for security and compliance
      
      # Database owner (typically cloudsqlsuperuser for Cloud SQL)
      expected_database_owner: "cloudsqlsuperuser"
      
      # Default owner for tables (if not in exceptions)
      expected_table_owner: "postgres"
      
      # Default owner for views (if not in exceptions)
      expected_view_owner: "postgres"
      
      # Whitelist: Only these owners are allowed
      allowed_owners:
        - "postgres"
        - "cloudsqlsuperuser"
        - "app_user"
      
      # Blacklist: These owners trigger violations
      forbidden_owners:
        - "root"           # Security risk
        - "admin"          # Insecure default
        - "test_user"      # Should not be in production
        - "developer"      # Development account
      
      # ===================================================================
      # OWNERSHIP EXCEPTIONS
      # ===================================================================
      # Specific tables/views that have different owners than default
      
      table_owner_exceptions:
        "public.audit_log": "cloudsqlsuperuser"     # Audit owned by superuser
        "public.system_config": "cloudsqlsuperuser" # System table
        "public.migrations": "cloudsqlsuperuser"    # Migration tracking
      
      view_owner_exceptions:
        "public.admin_dashboard": "cloudsqlsuperuser"  # Admin view
        "public.security_report": "cloudsqlsuperuser"  # Security view
      
      # ===================================================================
      # REQUIRED OBJECTS
      # ===================================================================
      # Objects that MUST exist - missing triggers violation
      
      required_tables:
        - "public.users"
        - "public.orders"
        - "public.products"
        - "public.audit_log"
        - "public.migrations"
      
      required_views:
        - "public.user_summary"
        - "public.order_stats"
      
      required_functions:
        - "public.calculate_total"
        - "public.validate_email"
      
      required_procedures:
        - "public.cleanup_old_data"
      
      required_extensions:
        - "uuid-ossp"      # UUID generation
        - "pg_trgm"        # Trigram matching for fuzzy search
        - "pgcrypto"       # Cryptographic functions
      
      # ===================================================================
      # FORBIDDEN OBJECTS
      # ===================================================================
      # Objects that MUST NOT exist - presence triggers violation
      # Useful for preventing test/debug objects in production
      
      forbidden_tables:
        - "public.temp_debug_table"
        - "public.test_data"
        - "public.old_users_backup"
        - "public.dev_testing"
```

## Validation Output

### Example 1: All Checks Pass

```
Inspection complete!
  Tables: 124
  Views: 2
  Sequences: 15
  Functions: 8
  Procedures: 3
  Roles: 12
  Extensions: 2

Validating against schema baseline...
[OK] Database matches baseline expectations
```

### Example 2: Count Mismatches

```
Inspection complete!
  Tables: 126
  Views: 2
  Sequences: 15
  Functions: 8
  Procedures: 3
  Roles: 12
  Extensions: 1

Validating against schema baseline...

[WARNING] Schema drift detected!

SCHEMA DRIFT DETECTED:

Count Mismatches:
  Tables: Expected 124, Found 126 (diff: +2)
  Extensions: Expected 2, Found 1 (diff: -1)
```

### Example 3: Ownership Violations

```
Inspection complete!
  Tables: 124
  Views: 2

Validating against schema baseline...

[WARNING] Schema drift detected!

SCHEMA DRIFT DETECTED:

Ownership Violations:
  [ERROR] Database: mydb - Owner: postgres, Expected: cloudsqlsuperuser
  [ERROR] Table: public.users - Forbidden owner: test_user
  [WARNING] Table: public.orders - Owner: app_user, Expected: postgres
  [WARNING] Table: public.products - Owner: developer, Expected: postgres
```

### Example 4: Missing and Forbidden Objects

```
Inspection complete!
  Tables: 125
  Views: 2
  Extensions: 1

Validating against schema baseline...

[WARNING] Schema drift detected!

SCHEMA DRIFT DETECTED:

Count Mismatches:
  Tables: Expected 124, Found 125 (diff: +1)

Missing Required Objects:
  [MISSING] Extension: pgcrypto
  [MISSING] Table: public.audit_log
  [MISSING] Function: public.validate_email

Forbidden Objects Found:
  [ERROR] Table: public.temp_debug_table (should not exist)
```

### Example 5: Multiple Violations

```
Inspection complete!
  Tables: 127
  Views: 3
  Extensions: 1

Validating against schema baseline...

[WARNING] Schema drift detected!

SCHEMA DRIFT DETECTED:

Count Mismatches:
  Tables: Expected 124, Found 127 (diff: +3)
  Views: Expected 2, Found 3 (diff: +1)
  Extensions: Expected 2, Found 1 (diff: -1)

Missing Required Objects:
  [MISSING] Extension: pgcrypto
  [MISSING] Table: public.migrations

Forbidden Objects Found:
  [ERROR] Table: public.temp_debug_table (should not exist)
  [ERROR] Table: public.test_data (should not exist)

Ownership Violations:
  [ERROR] Database: mydb - Owner: postgres, Expected: cloudsqlsuperuser
  [ERROR] Table: public.orders - Forbidden owner: test_user
  [WARNING] Table: public.products - Owner: app_user, Expected: postgres
  [WARNING] View: public.user_summary - Owner: readonly, Expected: postgres
```

## Use Cases

### 1. Production Monitoring
Monitor production database for unauthorized changes:
```bash
# Run daily via cron
./drift-analysis-cli gcp sql db --config config.yaml --connection prod-db
```

### 2. CI/CD Pipeline
Block deployments if schema doesn't match baseline:
```yaml
# .gitlab-ci.yml
schema-validation:
  script:
    - ./drift-analysis-cli gcp sql db --config config.yaml --connection prod-db
  only:
    - main
```

### 3. Compliance Auditing
Ensure databases meet security requirements:
- Database owned by cloudsqlsuperuser
- No forbidden owners (root, admin, test users)
- Required security extensions installed
- Audit tables properly configured

### 4. Migration Validation
After running migrations, verify:
- Correct number of new tables/views
- Proper ownership applied
- Required objects created
- No leftover temp tables

### 5. Multi-Environment Consistency
Ensure dev, staging, and prod have same structure:
```bash
# Compare all environments
./drift-analysis-cli gcp sql db --config config-dev.yaml --all
./drift-analysis-cli gcp sql db --config config-staging.yaml --all
./drift-analysis-cli gcp sql db --config config-prod.yaml --all
```

## Getting Actual Counts

To set up your baseline, first inspect without a baseline:

```bash
# Inspect to see current state
./drift-analysis-cli gcp sql db --config config.yaml --connection mydb

# Output shows actual counts:
#   Tables: 124
#   Views: 2
#   Sequences: 15
#   Functions: 8
#   Procedures: 3
#   Roles: 12
#   Extensions: 2

# Then add these values to your config as the baseline
```

## Best Practices

1. **Start with Counts**: Begin with just expected counts, add ownership later
2. **Use Allowed Owners**: Whitelist approach is more secure than blacklist
3. **Document Exceptions**: Comment why specific tables have different owners
4. **Version Control**: Commit baseline config to git for team sharing
5. **Regular Review**: Update baseline after approved schema changes
6. **Automate Checks**: Run in CI/CD to catch issues early
7. **Alert on Drift**: Integrate with monitoring for production alerts

## Troubleshooting

### Too Many Violations
If you see many violations on first run:
1. Run without baseline to see current state
2. Decide if current state is correct
3. Update baseline to match reality
4. Or fix schema to match desired baseline

### Exceptions Not Working
Ensure table/view names match exactly:
- Include schema prefix: `"public.users"` not `"users"`
- Case sensitive
- Check actual names in database

### Forbidden Owner False Positives
If legitimate owners are flagged:
- Add them to `allowed_owners` list
- Or remove from `forbidden_owners` list
- Or use `table_owner_exceptions` for specific cases
