package sql

import "fmt"

// compareBackupSettings compares backup-related settings
func (a *Analyzer) compareBackupSettings(actual, baseline *Settings, drift *InstanceDrift) {
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

	if baseline.BackupRetentionDays > 0 && actual.BackupRetentionDays != baseline.BackupRetentionDays {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.backup_retention_days",
			Expected: fmt.Sprintf("%d", baseline.BackupRetentionDays),
			Actual:   fmt.Sprintf("%d", actual.BackupRetentionDays),
			Severity: "medium",
		})
	}

	if baseline.TransactionLogRetentionDays > 0 && actual.TransactionLogRetentionDays != baseline.TransactionLogRetentionDays {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.transaction_log_retention_days",
			Expected: fmt.Sprintf("%d", baseline.TransactionLogRetentionDays),
			Actual:   fmt.Sprintf("%d", actual.TransactionLogRetentionDays),
			Severity: "medium",
		})
	}

	if baseline.BackupStartTime != "" && actual.BackupStartTime != baseline.BackupStartTime {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.backup_start_time",
			Expected: baseline.BackupStartTime,
			Actual:   actual.BackupStartTime,
			Severity: "low",
		})
	}
}

// compareAvailabilitySettings compares availability-related settings
func (a *Analyzer) compareAvailabilitySettings(actual, baseline *Settings, drift *InstanceDrift) {
	if baseline.AvailabilityType != "" && actual.AvailabilityType != baseline.AvailabilityType {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.availability_type",
			Expected: baseline.AvailabilityType,
			Actual:   actual.AvailabilityType,
			Severity: "high",
		})
	}

	if baseline.PricingPlan != "" && actual.PricingPlan != baseline.PricingPlan {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.pricing_plan",
			Expected: baseline.PricingPlan,
			Actual:   actual.PricingPlan,
			Severity: "low",
		})
	}

	if baseline.ReplicationType != "" && actual.ReplicationType != baseline.ReplicationType {
		drift.Drifts = append(drift.Drifts, Drift{
			Field:    "settings.replication_type",
			Expected: baseline.ReplicationType,
			Actual:   actual.ReplicationType,
			Severity: "medium",
		})
	}
}

// compareIPConfig compares IP configuration settings
func (a *Analyzer) compareIPConfig(actual, baseline *Settings, drift *InstanceDrift) {
	if baseline.IPConfiguration == nil || actual.IPConfiguration == nil {
		return
	}

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

	if len(baseline.IPConfiguration.AuthorizedNetworks) > 0 {
		a.compareAuthorizedNetworks(baseline.IPConfiguration, actual.IPConfiguration, drift)
	}
}

// compareInsightsConfig compares insights configuration settings
func (a *Analyzer) compareInsightsConfig(actual, baseline *Settings, drift *InstanceDrift) {
	if baseline.InsightsConfig == nil || actual.InsightsConfig == nil {
		return
	}

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
