package sql

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/api/sqladmin/v1"
)

// DatabaseInstance represents a GCP Cloud SQL PostgreSQL instance with its configuration
type DatabaseInstance struct {
	Project           string
	Name              string
	State             string
	Region            string
	Config            *DatabaseConfig
	MaintenanceWindow *MaintenanceWindow
	Labels            map[string]string
	Databases         []string
}

// DatabaseConfig holds the configuration parameters for a PostgreSQL instance
type DatabaseConfig struct {
	DatabaseVersion   string            `yaml:"database_version" json:"database_version"`
	Tier              string            `yaml:"tier" json:"tier"`
	DatabaseFlags     map[string]string `yaml:"database_flags,omitempty" json:"database_flags,omitempty"`
	Settings          *Settings         `yaml:"settings,omitempty" json:"settings,omitempty"`
	DiskSize          int64             `yaml:"disk_size_gb" json:"disk_size_gb"`
	DiskType          string            `yaml:"disk_type" json:"disk_type"`
	DiskAutoresize    bool              `yaml:"disk_autoresize" json:"disk_autoresize"`
	MaintenanceDenied []string          `yaml:"maintenance_denied_periods,omitempty" json:"maintenance_denied_periods,omitempty"`
	RequiredDatabases []string          `yaml:"required_databases,omitempty" json:"required_databases,omitempty"`
}

// Settings contains the runtime and operational settings for a database instance
type Settings struct {
	AvailabilityType            string           `yaml:"availability_type" json:"availability_type"`
	BackupEnabled               bool             `yaml:"backup_enabled" json:"backup_enabled"`
	BackupStartTime             string           `yaml:"backup_start_time,omitempty" json:"backup_start_time,omitempty"`
	BackupRetentionDays         int64            `yaml:"backup_retention_days,omitempty" json:"backup_retention_days,omitempty"`
	PointInTimeRecovery         bool             `yaml:"point_in_time_recovery" json:"point_in_time_recovery"`
	TransactionLogRetentionDays int64            `yaml:"transaction_log_retention_days,omitempty" json:"transaction_log_retention_days,omitempty"`
	IPConfiguration             *IPConfiguration `yaml:"ip_configuration,omitempty" json:"ip_configuration,omitempty"`
	LocationPreference          string           `yaml:"location_preference,omitempty" json:"location_preference,omitempty"`
	DataDiskSizeGb              int64            `yaml:"data_disk_size_gb" json:"data_disk_size_gb"`
	PricingPlan                 string           `yaml:"pricing_plan" json:"pricing_plan"`
	ReplicationType             string           `yaml:"replication_type" json:"replication_type"`
	InsightsConfig              *InsightsConfig  `yaml:"insights_config,omitempty" json:"insights_config,omitempty"`
}

// IPConfiguration defines network and security settings for database access
type IPConfiguration struct {
	IPv4Enabled        bool     `yaml:"ipv4_enabled" json:"ipv4_enabled"`
	PrivateNetworkID   string   `yaml:"private_network,omitempty" json:"private_network,omitempty"`
	RequireSSL         bool     `yaml:"require_ssl" json:"require_ssl"`
	AuthorizedNetworks []string `yaml:"authorized_networks,omitempty" json:"authorized_networks,omitempty"`
}

// InsightsConfig configures Query Insights for performance monitoring
type InsightsConfig struct {
	QueryInsightsEnabled  bool  `yaml:"query_insights_enabled" json:"query_insights_enabled"`
	QueryPlansPerMinute   int64 `yaml:"query_plans_per_minute" json:"query_plans_per_minute"`
	QueryStringLength     int64 `yaml:"query_string_length" json:"query_string_length"`
	RecordApplicationTags bool  `yaml:"record_application_tags" json:"record_application_tags"`
}

// MaintenanceWindow defines when database maintenance can occur
type MaintenanceWindow struct {
	Day         int    `yaml:"day" json:"day"`
	Hour        int    `yaml:"hour" json:"hour"`
	UpdateTrack string `yaml:"update_track" json:"update_track"`
}

// Analyzer performs drift analysis on GCP Cloud SQL instances
type Analyzer struct {
	service    *sqladmin.Service
	lastReport *DriftReport
	projects   []string
}

// NewAnalyzer creates a new Analyzer instance with GCP API client
func NewAnalyzer(ctx context.Context) (*Analyzer, error) {
	service, err := sqladmin.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQL Admin client: %w", err)
	}

	return &Analyzer{service: service}, nil
}

// Close releases resources held by the Analyzer
func (a *Analyzer) Close() error {
	return nil
}

// Analyze performs drift analysis implementing analyzer.ResourceAnalyzer interface
func (a *Analyzer) Analyze(ctx context.Context, projects []string) error {
	a.projects = projects
	return nil
}

// GenerateReport generates a formatted report implementing analyzer.ResourceAnalyzer interface
func (a *Analyzer) GenerateReport() (string, error) {
	if a.lastReport == nil {
		return "", fmt.Errorf("no analysis has been performed yet")
	}
	return a.lastReport.FormatText(), nil
}

// GetDriftCount returns the number of drifts detected implementing analyzer.ResourceAnalyzer interface
func (a *Analyzer) GetDriftCount() int {
	if a.lastReport == nil {
		return 0
	}
	return a.lastReport.DriftedInstances
}

// DiscoverInstances finds all PostgreSQL instances across the specified GCP projects
func (a *Analyzer) DiscoverInstances(ctx context.Context, projects []string) ([]*DatabaseInstance, error) {
	var instances []*DatabaseInstance

	for _, project := range projects {
		projectInstances, err := a.discoverProjectInstances(ctx, project)
		if err != nil {
			return nil, fmt.Errorf("failed to discover instances in project %s: %w", project, err)
		}
		instances = append(instances, projectInstances...)
	}

	return instances, nil
}

// discoverProjectInstances lists all PostgreSQL instances in a single GCP project
func (a *Analyzer) discoverProjectInstances(ctx context.Context, project string) ([]*DatabaseInstance, error) {
	req := a.service.Instances.List(project)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	var instances []*DatabaseInstance
	for _, inst := range resp.Items {
		// Filter for PostgreSQL only
		if !isPostgreSQL(inst.DatabaseVersion) {
			continue
		}

		dbInstance := &DatabaseInstance{
			Project:           project,
			Name:              inst.Name,
			State:             inst.State,
			Region:            inst.Region,
			Config:            extractConfig(inst),
			MaintenanceWindow: extractMaintenanceWindow(inst),
			Labels:            inst.Settings.UserLabels,
		}

		// List databases in this instance
		databases, err := a.listDatabases(ctx, project, inst.Name)
		if err != nil {
			// Log error but continue - database listing is not critical
			fmt.Fprintf(os.Stderr, "Warning: Failed to list databases for %s: %v\n", inst.Name, err)
		} else {
			dbInstance.Databases = databases
		}

		instances = append(instances, dbInstance)
	}

	return instances, nil
}

// listDatabases retrieves the list of databases in a Cloud SQL instance
func (a *Analyzer) listDatabases(ctx context.Context, project, instance string) ([]string, error) {
	req := a.service.Databases.List(project, instance)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, db := range resp.Items {
		// Exclude template databases
		if db.Name != "template0" && db.Name != "template1" {
			databases = append(databases, db.Name)
		}
	}

	return databases, nil
}

// isPostgreSQL checks if the database version string represents a PostgreSQL instance
func isPostgreSQL(version string) bool {
	return len(version) >= 8 && version[:8] == "POSTGRES"
}

// extractConfig extracts configuration parameters from a GCP database instance
func extractConfig(inst *sqladmin.DatabaseInstance) *DatabaseConfig {
	config := &DatabaseConfig{
		DatabaseVersion: inst.DatabaseVersion,
		Tier:            inst.Settings.Tier,
		DatabaseFlags:   make(map[string]string),
		DiskSize:        inst.Settings.DataDiskSizeGb,
		DiskType:        inst.Settings.DataDiskType,
	}

	if inst.Settings.StorageAutoResize != nil {
		config.DiskAutoresize = *inst.Settings.StorageAutoResize
	}

	// Extract database flags
	for _, flag := range inst.Settings.DatabaseFlags {
		config.DatabaseFlags[flag.Name] = flag.Value
	}

	// Extract settings
	settings := &Settings{
		AvailabilityType:    inst.Settings.AvailabilityType,
		BackupEnabled:       inst.Settings.BackupConfiguration != nil && inst.Settings.BackupConfiguration.Enabled,
		PointInTimeRecovery: inst.Settings.BackupConfiguration != nil && inst.Settings.BackupConfiguration.PointInTimeRecoveryEnabled,
		DataDiskSizeGb:      inst.Settings.DataDiskSizeGb,
		PricingPlan:         inst.Settings.PricingPlan,
		ReplicationType:     inst.Settings.ReplicationType,
	}

	if inst.Settings.BackupConfiguration != nil {
		settings.BackupStartTime = inst.Settings.BackupConfiguration.StartTime
		settings.TransactionLogRetentionDays = inst.Settings.BackupConfiguration.TransactionLogRetentionDays

		// Extract backup retention days
		if inst.Settings.BackupConfiguration.BackupRetentionSettings != nil {
			settings.BackupRetentionDays = inst.Settings.BackupConfiguration.BackupRetentionSettings.RetainedBackups
		}
	}

	if inst.Settings.LocationPreference != nil {
		settings.LocationPreference = inst.Settings.LocationPreference.Zone
	}

	// IP Configuration
	if inst.Settings.IpConfiguration != nil {
		ipConfig := &IPConfiguration{
			IPv4Enabled: inst.Settings.IpConfiguration.Ipv4Enabled,
			RequireSSL:  inst.Settings.IpConfiguration.RequireSsl,
		}

		if inst.Settings.IpConfiguration.PrivateNetwork != "" {
			ipConfig.PrivateNetworkID = inst.Settings.IpConfiguration.PrivateNetwork
		}

		for _, net := range inst.Settings.IpConfiguration.AuthorizedNetworks {
			ipConfig.AuthorizedNetworks = append(ipConfig.AuthorizedNetworks, net.Value)
		}

		settings.IPConfiguration = ipConfig
	}

	// Insights Config
	if inst.Settings.InsightsConfig != nil {
		settings.InsightsConfig = &InsightsConfig{
			QueryInsightsEnabled:  inst.Settings.InsightsConfig.QueryInsightsEnabled,
			QueryPlansPerMinute:   inst.Settings.InsightsConfig.QueryPlansPerMinute,
			QueryStringLength:     inst.Settings.InsightsConfig.QueryStringLength,
			RecordApplicationTags: inst.Settings.InsightsConfig.RecordApplicationTags,
		}
	}

	config.Settings = settings

	// Maintenance denial periods
	if inst.Settings.DenyMaintenancePeriods != nil {
		for _, period := range inst.Settings.DenyMaintenancePeriods {
			config.MaintenanceDenied = append(config.MaintenanceDenied,
				fmt.Sprintf("%s to %s", period.StartDate, period.EndDate))
		}
	}

	return config
}

// extractMaintenanceWindow extracts the maintenance window configuration from an instance
func extractMaintenanceWindow(inst *sqladmin.DatabaseInstance) *MaintenanceWindow {
	if inst.Settings.MaintenanceWindow == nil {
		return nil
	}

	return &MaintenanceWindow{
		Day:         int(inst.Settings.MaintenanceWindow.Day),
		Hour:        int(inst.Settings.MaintenanceWindow.Hour),
		UpdateTrack: inst.Settings.MaintenanceWindow.UpdateTrack,
	}
}

// AnalyzeDrift compares discovered instances against a baseline and generates a drift report
func (a *Analyzer) AnalyzeDrift(instances []*DatabaseInstance, baseline *DatabaseConfig) *DriftReport {
	report := &DriftReport{
		Timestamp:      time.Now(),
		TotalInstances: len(instances),
		Instances:      make([]*InstanceDrift, 0),
	}

	for _, inst := range instances {
		drift := a.analyzeInstance(inst, baseline)
		report.Instances = append(report.Instances, drift)

		if len(drift.Drifts) > 0 {
			report.DriftedInstances++
		}
	}

	a.lastReport = report
	return report
}

// AnalyzeInstance compares a single instance against the baseline configuration (public method)
func (a *Analyzer) AnalyzeInstance(inst *DatabaseInstance, baseline *DatabaseConfig) *InstanceDrift {
	return a.analyzeInstance(inst, baseline)
}

// analyzeInstance compares a single instance against the baseline configuration
func (a *Analyzer) analyzeInstance(inst *DatabaseInstance, baseline *DatabaseConfig) *InstanceDrift {
	drift := &InstanceDrift{
		Project:           inst.Project,
		Name:              inst.Name,
		Region:            inst.Region,
		State:             inst.State,
		Labels:            inst.Labels,
		Databases:         inst.Databases,
		MaintenanceWindow: inst.MaintenanceWindow,
		Drifts:            make([]Drift, 0),
		Recommendations:   make([]string, 0),
	}

	if baseline == nil {
		// No baseline, provide recommendations based on best practices
		drift.Recommendations = a.getBestPracticeRecommendations(inst)
		return drift
	}

	// Compare with baseline - only check fields that are specified in baseline
	if baseline.DatabaseVersion != "" && inst.Config.DatabaseVersion != baseline.DatabaseVersion {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "database_version",
			Expected: baseline.DatabaseVersion,
			Actual:   inst.Config.DatabaseVersion,
			Severity: "medium",
		})
	}

	if baseline.Tier != "" && inst.Config.Tier != baseline.Tier {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "tier",
			Expected: baseline.Tier,
			Actual:   inst.Config.Tier,
			Severity: "high",
		})
	}

	if baseline.DiskType != "" && inst.Config.DiskType != baseline.DiskType {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "disk_type",
			Expected: baseline.DiskType,
			Actual:   inst.Config.DiskType,
			Severity: "medium",
		})
	}

	// Check disk size if specified in baseline
	if baseline.DiskSize > 0 && inst.Config.DiskSize != baseline.DiskSize {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "disk_size_gb",
			Expected: fmt.Sprintf("%d", baseline.DiskSize),
			Actual:   fmt.Sprintf("%d", inst.Config.DiskSize),
			Severity: "medium",
		})
	}

	// Only check disk autoresize if disk type is specified (indicating disk config matters)
	if baseline.DiskType != "" && inst.Config.DiskAutoresize != baseline.DiskAutoresize {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "disk_autoresize",
			Expected: fmt.Sprintf("%v", baseline.DiskAutoresize),
			Actual:   fmt.Sprintf("%v", inst.Config.DiskAutoresize),
			Severity: "low",
		})
	}

	// Compare database flags
	a.compareDatabaseFlags(inst.Config, baseline, drift)

	// Compare settings
	a.compareSettings(inst.Config.Settings, baseline.Settings, drift)

	// Check required databases
	a.checkRequiredDatabases(inst, baseline, drift)

	// Generate recommendations
	drift.Recommendations = a.getRecommendations(inst, baseline, drift)

	return drift
}

// checkRequiredDatabases validates that required databases exist on the instance
func (a *Analyzer) checkRequiredDatabases(inst *DatabaseInstance, baseline *DatabaseConfig, drift *InstanceDrift) {
	if len(baseline.RequiredDatabases) == 0 {
		return
	}

	// Create a set of existing databases for quick lookup
	existingDBs := make(map[string]bool)
	for _, db := range inst.Databases {
		existingDBs[db] = true
	}

	// Check each required database
	missingDatabases := make([]string, 0)
	for _, requiredDB := range baseline.RequiredDatabases {
		if !existingDBs[requiredDB] {
			missingDatabases = append(missingDatabases, requiredDB)
		}
	}

	// Create a set of required databases for checking extras
	requiredDBs := make(map[string]bool)
	for _, db := range baseline.RequiredDatabases {
		requiredDBs[db] = true
	}

	// Check for extra databases (in actual but not in required list)
	extraDatabases := make([]string, 0)
	for db := range existingDBs {
		if !requiredDBs[db] {
			extraDatabases = append(extraDatabases, db)
		}
	}

	// Report missing databases as drift
	if len(missingDatabases) > 0 {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "required_databases",
			Expected: fmt.Sprintf("%v", baseline.RequiredDatabases),
			Actual:   fmt.Sprintf("Missing: %v", missingDatabases),
			Severity: "high",
		})
	}

	// Report extra databases as drift (lower severity)
	if len(extraDatabases) > 0 {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "required_databases",
			Expected: fmt.Sprintf("%v", baseline.RequiredDatabases),
			Actual:   fmt.Sprintf("Extra: %v", extraDatabases),
			Severity: "medium",
		})
	}
}

// compareDatabaseFlags compares database flags between actual and baseline configurations
func (a *Analyzer) compareDatabaseFlags(config, baseline *DatabaseConfig, drift *InstanceDrift) {
	for key, baselineValue := range baseline.DatabaseFlags {
		actualValue, exists := config.DatabaseFlags[key]
		if !exists {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    fmt.Sprintf("database_flags.%s", key),
				Expected: baselineValue,
				Actual:   "not set",
				Severity: "medium",
			})
		} else if actualValue != baselineValue {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    fmt.Sprintf("database_flags.%s", key),
				Expected: baselineValue,
				Actual:   actualValue,
				Severity: "medium",
			})
		}
	}

	// Check for extra flags not in baseline
	for key, actualValue := range config.DatabaseFlags {
		if _, exists := baseline.DatabaseFlags[key]; !exists {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    fmt.Sprintf("database_flags.%s", key),
				Expected: "not set",
				Actual:   actualValue,
				Severity: "low",
			})
		}
	}
}

// compareSettings compares runtime settings between actual and baseline configurations
func (a *Analyzer) compareSettings(actual, baseline *Settings, drift *InstanceDrift) {
	if baseline == nil {
		return
	}

	if baseline.AvailabilityType != "" && actual.AvailabilityType != baseline.AvailabilityType {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.availability_type",
			Expected: baseline.AvailabilityType,
			Actual:   actual.AvailabilityType,
			Severity: "high",
		})
	}

	// Always check backup settings if baseline specifies them (critical for data safety)
	if actual.BackupEnabled != baseline.BackupEnabled {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.backup_enabled",
			Expected: fmt.Sprintf("%v", baseline.BackupEnabled),
			Actual:   fmt.Sprintf("%v", actual.BackupEnabled),
			Severity: "critical",
		})
	}

	if actual.PointInTimeRecovery != baseline.PointInTimeRecovery {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.point_in_time_recovery",
			Expected: fmt.Sprintf("%v", baseline.PointInTimeRecovery),
			Actual:   fmt.Sprintf("%v", actual.PointInTimeRecovery),
			Severity: "high",
		})
	}

	// Check backup retention days if specified in baseline
	if baseline.BackupRetentionDays > 0 && actual.BackupRetentionDays != baseline.BackupRetentionDays {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.backup_retention_days",
			Expected: fmt.Sprintf("%d", baseline.BackupRetentionDays),
			Actual:   fmt.Sprintf("%d", actual.BackupRetentionDays),
			Severity: "medium",
		})
	}

	// Check transaction log retention (PITR) days if specified in baseline
	if baseline.TransactionLogRetentionDays > 0 && actual.TransactionLogRetentionDays != baseline.TransactionLogRetentionDays {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.transaction_log_retention_days",
			Expected: fmt.Sprintf("%d", baseline.TransactionLogRetentionDays),
			Actual:   fmt.Sprintf("%d", actual.TransactionLogRetentionDays),
			Severity: "medium",
		})
	}

	// Check pricing plan if specified
	if baseline.PricingPlan != "" && actual.PricingPlan != baseline.PricingPlan {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.pricing_plan",
			Expected: baseline.PricingPlan,
			Actual:   actual.PricingPlan,
			Severity: "low",
		})
	}

	// Check replication type if specified
	if baseline.ReplicationType != "" && actual.ReplicationType != baseline.ReplicationType {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.replication_type",
			Expected: baseline.ReplicationType,
			Actual:   actual.ReplicationType,
			Severity: "medium",
		})
	}

	// Check backup start time if specified
	if baseline.BackupStartTime != "" && actual.BackupStartTime != baseline.BackupStartTime {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.backup_start_time",
			Expected: baseline.BackupStartTime,
			Actual:   actual.BackupStartTime,
			Severity: "low",
		})
	}

	// Compare IP configuration
	if baseline.IPConfiguration != nil && actual.IPConfiguration != nil {
		if actual.IPConfiguration.IPv4Enabled != baseline.IPConfiguration.IPv4Enabled {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "settings.ip_configuration.ipv4_enabled",
				Expected: fmt.Sprintf("%v", baseline.IPConfiguration.IPv4Enabled),
				Actual:   fmt.Sprintf("%v", actual.IPConfiguration.IPv4Enabled),
				Severity: "medium",
			})
		}

		if actual.IPConfiguration.RequireSSL != baseline.IPConfiguration.RequireSSL {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "settings.ip_configuration.require_ssl",
				Expected: fmt.Sprintf("%v", baseline.IPConfiguration.RequireSSL),
				Actual:   fmt.Sprintf("%v", actual.IPConfiguration.RequireSSL),
				Severity: "critical",
			})
		}

		// Compare authorized networks if specified in baseline
		if len(baseline.IPConfiguration.AuthorizedNetworks) > 0 {
			a.compareAuthorizedNetworks(baseline.IPConfiguration, actual.IPConfiguration, drift)
		}
	}

	// Compare insights config if specified in baseline
	if baseline.InsightsConfig != nil && actual.InsightsConfig != nil {
		if actual.InsightsConfig.QueryInsightsEnabled != baseline.InsightsConfig.QueryInsightsEnabled {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "settings.insights_config.query_insights_enabled",
				Expected: fmt.Sprintf("%v", baseline.InsightsConfig.QueryInsightsEnabled),
				Actual:   fmt.Sprintf("%v", actual.InsightsConfig.QueryInsightsEnabled),
				Severity: "low",
			})
		}

		if baseline.InsightsConfig.QueryPlansPerMinute > 0 &&
			actual.InsightsConfig.QueryPlansPerMinute != baseline.InsightsConfig.QueryPlansPerMinute {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "settings.insights_config.query_plans_per_minute",
				Expected: fmt.Sprintf("%d", baseline.InsightsConfig.QueryPlansPerMinute),
				Actual:   fmt.Sprintf("%d", actual.InsightsConfig.QueryPlansPerMinute),
				Severity: "low",
			})
		}

		if baseline.InsightsConfig.QueryStringLength > 0 &&
			actual.InsightsConfig.QueryStringLength != baseline.InsightsConfig.QueryStringLength {
			drift.Drifts = append(drift.Drifts, Drift{
				Field:    "settings.insights_config.query_string_length",
				Expected: fmt.Sprintf("%d", baseline.InsightsConfig.QueryStringLength),
				Actual:   fmt.Sprintf("%d", actual.InsightsConfig.QueryStringLength),
				Severity: "low",
			})
		}
	}
}

// compareAuthorizedNetworks compares authorized network lists between baseline and actual
func (a *Analyzer) compareAuthorizedNetworks(baseline, actual *IPConfiguration, drift *InstanceDrift) {
	// Create sets for comparison
	baselineNets := make(map[string]bool)
	for _, net := range baseline.AuthorizedNetworks {
		baselineNets[net] = true
	}

	actualNets := make(map[string]bool)
	for _, net := range actual.AuthorizedNetworks {
		actualNets[net] = true
	}

	// Find required networks (in baseline but not in actual)
	requiredNets := make([]string, 0)
	for net := range baselineNets {
		if !actualNets[net] {
			requiredNets = append(requiredNets, net)
		}
	}

	// Find extra networks (in actual but not in baseline)
	extraNets := make([]string, 0)
	for net := range actualNets {
		if !baselineNets[net] {
			extraNets = append(extraNets, net)
		}
	}

	// Report required networks as high severity
	if len(requiredNets) > 0 {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.ip_configuration.authorized_networks",
			Expected: fmt.Sprintf("Required: %v", requiredNets),
			Actual:   fmt.Sprintf("%v", actual.AuthorizedNetworks),
			Severity: "high",
		})
	}

	// Report extra networks as medium severity
	if len(extraNets) > 0 {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.ip_configuration.authorized_networks",
			Expected: fmt.Sprintf("%v", baseline.AuthorizedNetworks),
			Actual:   fmt.Sprintf("Extra: %v", extraNets),
			Severity: "medium",
		})
	}
}

// getBestPracticeRecommendations generates recommendations based on PostgreSQL best practices
func (a *Analyzer) getBestPracticeRecommendations(inst *DatabaseInstance) []string {
	var recommendations []string

	// Backup recommendations
	if !inst.Config.Settings.BackupEnabled {
		recommendations = append(recommendations, "CRITICAL: Enable automated backups")
	}

	if !inst.Config.Settings.PointInTimeRecovery {
		recommendations = append(recommendations, "HIGH: Enable point-in-time recovery for better RPO")
	}

	// High availability
	if inst.Config.Settings.AvailabilityType != "REGIONAL" {
		recommendations = append(recommendations, "HIGH: Consider REGIONAL availability for production workloads")
	}

	// SSL
	if inst.Config.Settings.IPConfiguration != nil && !inst.Config.Settings.IPConfiguration.RequireSSL {
		recommendations = append(recommendations, "CRITICAL: Enable SSL requirement for all connections")
	}

	// Public IP
	if inst.Config.Settings.IPConfiguration != nil && inst.Config.Settings.IPConfiguration.IPv4Enabled {
		recommendations = append(recommendations, "MEDIUM: Consider using private IP instead of public IPv4")
	}

	// Disk autoresize
	if !inst.Config.DiskAutoresize {
		recommendations = append(recommendations, "MEDIUM: Enable disk autoresize to prevent storage issues")
	}

	// Query insights
	if inst.Config.Settings.InsightsConfig == nil || !inst.Config.Settings.InsightsConfig.QueryInsightsEnabled {
		recommendations = append(recommendations, "LOW: Enable Query Insights for better performance monitoring")
	}

	// Version check (simplified)
	if inst.Config.DatabaseVersion < "POSTGRES_14" {
		recommendations = append(recommendations, "MEDIUM: Consider upgrading to PostgreSQL 14+ for better performance and features")
	}

	// Maintenance window
	if inst.MaintenanceWindow == nil {
		recommendations = append(recommendations, "LOW: Set a maintenance window for predictable updates")
	}

	return recommendations
}

// getRecommendations generates actionable recommendations based on detected drift
func (a *Analyzer) getRecommendations(inst *DatabaseInstance, baseline *DatabaseConfig, drift *InstanceDrift) []string {
	var recommendations []string

	if len(drift.Drifts) == 0 {
		return []string{"No drift detected - instance matches baseline"}
	}

	// Check for critical drifts
	hasCritical := false
	for _, d := range drift.Drifts {
		if d.Severity == "critical" {
			hasCritical = true
			break
		}
	}

	if hasCritical {
		recommendations = append(recommendations, "CRITICAL drifts detected - immediate action required")
	}

	// Specific recommendations based on drift
	for _, d := range drift.Drifts {
		if d.Field == "settings.backup_enabled" && d.Actual == "false" {
			recommendations = append(recommendations, "Enable backups immediately to protect data")
		}
		if d.Field == "settings.ip_configuration.require_ssl" && d.Actual == "false" {
			recommendations = append(recommendations, "Enable SSL requirement to secure connections")
		}
		if d.Field == "tier" {
			recommendations = append(recommendations, "Tier mismatch may affect performance and cost")
		}
	}

	return recommendations
}

// GetTimestamp returns the current timestamp for report generation
func (a *Analyzer) GetTimestamp() time.Time {
	return time.Now()
}
