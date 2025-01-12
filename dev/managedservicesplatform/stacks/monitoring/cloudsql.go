package monitoring

import (
	"fmt"

	"github.com/hashicorp/terraform-cdk-go/cdktf"
	"github.com/sourcegraph/managed-services-platform-cdktf/gen/google/monitoringnotificationchannel"

	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/alertpolicy"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resourceid"
)

func createCloudSQLAlerts(
	stack cdktf.TerraformStack,
	id resourceid.ID,
	vars Variables,
	channels []monitoringnotificationchannel.MonitoringNotificationChannel,
) error {
	cloudSQLResourceName := fmt.Sprintf("%s:%s",
		vars.ProjectID, *vars.CloudSQLInstanceID)

	// CloudSQL instance alerts.
	// Iterate over a list of Cloud SQL alert configurations. Custom struct defines
	// the field we expect to vary between each.
	for _, config := range []struct {
		ID                   string
		Name                 string
		Description          string
		ThresholdAggregation *alertpolicy.ThresholdAggregation
	}{
		{
			ID:          "memory",
			Name:        "Cloud SQL - Memory Utilization",
			Description: "Cloud SQL instance memory utilization is above acceptable threshold.",
			ThresholdAggregation: &alertpolicy.ThresholdAggregation{
				Filters: map[string]string{
					"metric.type": "cloudsql.googleapis.com/database/memory/utilization",
				},
				Aligner:   alertpolicy.MonitoringAlignMean,
				Reducer:   alertpolicy.MonitoringReduceNone,
				Period:    "300s",
				Threshold: 0.8,
			},
		},
		{
			ID:          "cpu",
			Name:        "Cloud SQL - CPU Utilization",
			Description: "Cloud SQL instance CPU utilization is above acceptable threshold.",
			ThresholdAggregation: &alertpolicy.ThresholdAggregation{
				Filters: map[string]string{
					"metric.type": "cloudsql.googleapis.com/database/cpu/utilization",
				},
				Aligner:   alertpolicy.MonitoringAlignMean,
				Reducer:   alertpolicy.MonitoringReduceNone,
				Period:    "300s",
				Threshold: 0.9,
			},
		},
		{
			ID:          "server_up",
			Name:        "Cloud SQL - Server Availability",
			Description: "Cloud SQL instance is down.",
			ThresholdAggregation: &alertpolicy.ThresholdAggregation{
				Filters: map[string]string{
					"metric.type": "cloudsql.googleapis.com/database/up",
				},
				Aligner: alertpolicy.MonitoringAlignMin,
				Reducer: alertpolicy.MonitoringReduceNone,
				Period:  "60s",
				// 1 == up, 0 == down
				Comparison: alertpolicy.ComparisonLT,
				Threshold:  1,
			},
		},
		{
			ID:          "disk_utilization",
			Name:        "Cloud SQL - Disk Utilization",
			Description: "Cloud SQL instance disk utilization is above acceptable threshold.",
			ThresholdAggregation: &alertpolicy.ThresholdAggregation{
				Filters: map[string]string{
					"metric.type": "cloudsql.googleapis.com/database/disk/utilization",
				},
				Aligner:   alertpolicy.MonitoringAlignMean,
				Reducer:   alertpolicy.MonitoringReduceNone,
				Period:    "300s",
				Threshold: 0.95,
			},
		},
	} {
		if _, err := alertpolicy.New(stack, id, &alertpolicy.Config{
			// Resource we are targetting in this helper
			ResourceKind: alertpolicy.CloudSQL,
			ResourceName: cloudSQLResourceName,

			// Alert policy
			ID:                   config.ID,
			Name:                 config.Name,
			Description:          config.Description,
			ThresholdAggregation: config.ThresholdAggregation,

			// Shared configuration
			Service:              vars.Service,
			EnvironmentID:        vars.EnvironmentID,
			ProjectID:            vars.ProjectID,
			NotificationChannels: channels,
		}); err != nil {
			return err
		}
	}

	// CloudSQLDatabase alerts
	for _, config := range []struct {
		ID                   string
		Name                 string
		Description          string
		ThresholdAggregation *alertpolicy.ThresholdAggregation
	}{
		{
			ID:          "per_query_lock_time",
			Name:        "Cloud SQL - Per-Query Lock Time",
			Description: "Cloud SQL database queries are encountering lock times above acceptable thresholds.",
			ThresholdAggregation: &alertpolicy.ThresholdAggregation{
				Filters: map[string]string{
					"metric.type": "cloudsql.googleapis.com/database/postgresql/insights/perquery/lock_time",
				},
				GroupByFields: []string{
					"metric.label.querystring",
					"metric.label.user",
				},
				Aligner: alertpolicy.MonitoringAlignRate,
				Reducer: alertpolicy.MonitoringReduceMean,
				Period:  "120s",
				// Threshold of 0.2 seconds
				Threshold: 0.2 * 1_000_000, // metric is in microseconds (us)
			},
		},
	} {
		if _, err := alertpolicy.New(stack, id, &alertpolicy.Config{
			// Resource we are targetting in this helper
			ResourceKind: alertpolicy.CloudSQLDatabase,
			ResourceName: cloudSQLResourceName,

			// Alert policy
			ID:                   config.ID,
			Name:                 config.Name,
			Description:          config.Description,
			ThresholdAggregation: config.ThresholdAggregation,

			// Shared configuration
			Service:              vars.Service,
			EnvironmentID:        vars.EnvironmentID,
			ProjectID:            vars.ProjectID,
			NotificationChannels: channels,
		}); err != nil {
			return err
		}
	}

	return nil
}
