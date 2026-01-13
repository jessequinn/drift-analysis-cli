# Schema Drift Fixes - SQL Script

## Overview

Comprehensive SQL script to fix schema drift issues detected across all 23 configured databases, including ownership corrections and missing schema objects.

**Generated:** 2026-01-03
**File:** drift_fixes.sql
**Size:** 854 lines, 34KB

## Summary

### Ownership Issues
- **Total Databases Processed:** 23
- **Databases Needing Ownership Fixes:** 7
- **Total Ownership Changes:** 227
- **ALTER Statements:** 135 (tables/sequences) + 92 commented (functions)

### Missing Schema Objects
- **Dev Environment (security database):**
  - 2 missing tables (company_permissions_history, user_permissions_history)
  - 2 missing sequences (company_permissions_history_id_seq, user_permissions_history_id_seq)
  - 4 missing indexes
  - 1 missing constraint (sso_uuid_unique)

- **Staging Environment (security database):**
  - 1 missing constraint (sso_uuid_unique)

**TOTAL FIXES: 227 ownership changes + 10 missing objects**

## Databases with Drift

### Ownership Corrections

#### 1. Production EU - zpebackend
- Functions: 46 ownership changes
- Issue: citext extension functions owned by cloudsqlsuperuser

#### 2. QA - notification-k6
- Tables: 5 ownership changes
- Sequences: 4 ownership changes

#### 3. QA - security
- Tables: 29 ownership changes  
- Sequences: 25 ownership changes

#### 4. QA - security-k6
- Tables: 31 ownership changes
- Sequences: 27 ownership changes

#### 5. Staging - notification-k6
- Tables: 5 ownership changes
- Sequences: 4 ownership changes

#### 6. Staging - security
- Tables: 29 ownership changes
- Sequences: 25 ownership changes

#### 7. Staging - security-k6
- Tables: 31 ownership changes
- Sequences: 27 ownership changes

### Missing Schema Objects

#### Dev Environment - security database
**Missing History Tracking Tables:**

1. **company_permissions_history**
   - Tracks changes to company permissions
   - Columns: id, permission_id, changes_made, reason, created_date, created_by
   - Sequence: company_permissions_history_id_seq
   - Indexes: permission_id, created_date

2. **user_permissions_history**
   - Tracks changes to user permissions
   - Columns: id, user_permission_id, initial_permission_id, new_permission_id, reason, created_date, created_by
   - Sequence: user_permissions_history_id_seq
   - Indexes: user_permission_id, created_date

**Missing Constraint:**
- sso_uuid_unique: UNIQUE constraint on sso.uuid column

**Impact:** Cannot track permission change history in dev; potential for duplicate SSO UUIDs

#### Staging Environment - security database
**Missing Constraint:**
- sso_uuid_unique: UNIQUE constraint on sso.uuid column

**Impact:** Potential for duplicate SSO UUIDs

## What the Script Fixes

### 1. Ownership Corrections
Corrects ownership where database objects are owned by `cloudsqlsuperuser` but should be owned by `postgres` according to baseline expectations.

**Objects Fixed:**
- Tables
- Sequences
- Views
- Functions (commented out - require signature verification)

### 2. Missing Schema Objects
Adds tables, sequences, indexes, and constraints that exist in QA but are missing in Dev and/or Staging.

**Objects Added:**
- History tracking tables
- Auto-increment sequences
- Performance indexes
- Data integrity constraints

## Safety Features

1. **Transactional:** All changes wrapped in BEGIN/COMMIT blocks
2. **Idempotent:** Uses IF NOT EXISTS - safe to run multiple times
3. **Metadata Only:** Ownership changes don't modify data
4. **Reversible:** Can be easily reverted if needed
5. **Database-Specific:** Each database has its own transaction
6. **Non-Destructive:** Only adds or modifies, never drops

## Usage

### Prerequisites

Before applying fixes, verify no duplicate data exists:

```sql
-- For dev security database - check for duplicate SSO UUIDs
SELECT uuid, COUNT(*) 
FROM public.sso 
WHERE uuid IS NOT NULL 
GROUP BY uuid 
HAVING COUNT(*) > 1;
-- Should return 0 rows
```

### Option 1: Execute Entire Script
```bash
# Connect to Cloud SQL via SSH tunnel
psql "host=localhost port=<tunnel_port> user=postgres sslmode=require \
  sslcert=client-cert.pem sslkey=client-key.pem" \
  -f drift_fixes.sql
```

### Option 2: Execute Per Environment

**For Dev Environment:**
```bash
# Extract dev-specific fixes
grep -A 200 "DEV ENVIRONMENT" drift_fixes.sql > dev_security_fixes.sql

# Apply via drift CLI tunnel
./drift-analysis-cli gcp sql db \
  -c "zpe-cloud-test-environment:us-west1:test-microservices-kf48fjd7:security" \
  -f summary

# In another terminal, while tunnel is active:
psql -h localhost -p <port> -U postgres -d security -f dev_security_fixes.sql
```

**For Staging Environment:**
```bash
# Extract staging-specific fixes
grep -A 50 "STAGING ENVIRONMENT.*sso_uuid_unique" drift_fixes.sql > staging_security_fixes.sql

# Apply via drift CLI tunnel
./drift-analysis-cli gcp sql db \
  -c "zpe-cloud-staging-environment:us-west1:staging-microservices-us-west1-iu9qfbl:security" \
  -f summary

# In another terminal:
psql -h localhost -p <port> -U postgres -d security -f staging_security_fixes.sql
```

### Option 3: Execute Per Database Section
Extract and run individual database sections from drift_fixes.sql.

## Verification

### After Ownership Fixes

```sql
-- Check tables owned by cloudsqlsuperuser (should be empty)
SELECT schemaname, tablename, tableowner 
FROM pg_tables 
WHERE tableowner = 'cloudsqlsuperuser';

-- Check sequences owned by cloudsqlsuperuser (should be empty)
SELECT schemaname, sequencename, sequenceowner 
FROM pg_sequences 
WHERE sequenceowner = 'cloudsqlsuperuser';
```

### After Adding Missing Objects (Dev)

```sql
\c security

-- Verify tables exist
SELECT COUNT(*) FROM pg_tables 
WHERE tablename IN ('company_permissions_history', 'user_permissions_history');
-- Should return: 2

-- Verify sequences exist
SELECT COUNT(*) FROM pg_sequences 
WHERE sequencename IN ('company_permissions_history_id_seq', 'user_permissions_history_id_seq');
-- Should return: 2

-- Verify constraint exists
SELECT COUNT(*) FROM pg_constraint WHERE conname = 'sso_uuid_unique';
-- Should return: 1

-- Verify ownership is correct
SELECT tablename, tableowner FROM pg_tables 
WHERE tablename LIKE '%permissions_history%';
-- Should show: postgres
```

### After Adding Missing Constraint (Staging)

```sql
\c security

-- Verify constraint exists
SELECT COUNT(*) FROM pg_constraint WHERE conname = 'sso_uuid_unique';
-- Should return: 1
```

## Re-run Drift Analysis

After applying fixes, update baselines and re-run drift analysis:

### 1. Update Config Baselines

For dev and staging security databases, update expected counts:
```yaml
schema_baseline:
  expected_tables: 31  # Changed from 29
  expected_sequences: 27  # Changed from 25
```

### 2. Test Drift Analysis

```bash
# Dev environment
./drift-analysis-cli gcp sql db \
  -c "zpe-cloud-test-environment:us-west1:test-microservices-kf48fjd7:security" \
  -f summary

# Staging environment
./drift-analysis-cli gcp sql db \
  -c "zpe-cloud-staging-environment:us-west1:staging-microservices-us-west1-iu9qfbl:security" \
  -f summary
```

Expected result: `[OK] Database matches baseline expectations`

## Function Ownership Notes

Function ownership fixes are **commented out** in the script because they require full function signatures. 

**To Fix Functions:**

1. Get full function signatures:
```sql
SELECT pronamespace::regnamespace || '.' || oid::regprocedure 
FROM pg_proc 
WHERE proowner = (SELECT oid FROM pg_roles WHERE rolname = 'cloudsqlsuperuser');
```

2. Update the commented ALTER FUNCTION statements with correct signatures

3. Execute the modified statements

## Rollback Procedures

### Rollback Ownership Changes
```sql
BEGIN;
ALTER TABLE public.<table_name> OWNER TO cloudsqlsuperuser;
ALTER SEQUENCE public.<sequence_name> OWNER TO cloudsqlsuperuser;
COMMIT;
```

### Rollback Added Objects (Dev)
```sql
BEGIN;
DROP TABLE IF EXISTS public.company_permissions_history CASCADE;
DROP TABLE IF EXISTS public.user_permissions_history CASCADE;
DROP SEQUENCE IF EXISTS public.company_permissions_history_id_seq CASCADE;
DROP SEQUENCE IF EXISTS public.user_permissions_history_id_seq CASCADE;
ALTER TABLE public.sso DROP CONSTRAINT IF EXISTS sso_uuid_unique;
COMMIT;
```

### Rollback Added Constraint (Staging)
```sql
BEGIN;
ALTER TABLE public.sso DROP CONSTRAINT IF EXISTS sso_uuid_unique;
COMMIT;
```

## Files Generated

1. **drift_fixes.sql** (854 lines, 34KB) - Complete SQL fix script
2. **generate_drift_fix.py** - Python script to generate ownership fixes from cached schemas
3. **DRIFT_FIXES_README.md** - This documentation

## Execution Order Recommendation

1. **Test on Dev first** (lowest risk)
2. **Then QA** (validate fixes work)
3. **Then Staging** (final validation)
4. **Finally Production** (after all testing)

For each environment:
1. Review relevant sections of drift_fixes.sql
2. Check for duplicate data (especially SSO UUIDs)
3. Apply fixes during maintenance window
4. Verify with SQL queries
5. Re-run drift analysis
6. Update config.yaml baselines if needed

## Next Steps

1. Review drift_fixes.sql sections for each environment
2. Schedule maintenance windows for each environment
3. Test on dev environment first
4. Update config.yaml baselines after fixes applied
5. Document any additional drift found during verification
