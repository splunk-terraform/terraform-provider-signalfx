package pmeta

import (
	"context"

	"github.com/signalfx/signalfx-go"
	"github.com/signalfx/signalfx-go/alertmuting"
	"github.com/signalfx/signalfx-go/automated-archival"
	"github.com/signalfx/signalfx-go/chart"
	"github.com/signalfx/signalfx-go/dashboard"
	"github.com/signalfx/signalfx-go/dashboard_group"
	"github.com/signalfx/signalfx-go/datalink"
	"github.com/signalfx/signalfx-go/detector"
	"github.com/signalfx/signalfx-go/integration"
	"github.com/signalfx/signalfx-go/metric_ruleset"
	"github.com/signalfx/signalfx-go/metrics_metadata"
	"github.com/signalfx/signalfx-go/organization"
	"github.com/signalfx/signalfx-go/orgtoken"
	"github.com/signalfx/signalfx-go/sessiontoken"
	"github.com/signalfx/signalfx-go/slo"
	"github.com/signalfx/signalfx-go/team"
)

type Client interface {
	CreateSessionToken(ctx context.Context, tokenRequest *sessiontoken.CreateTokenRequest) (*sessiontoken.Token, error)
	DeleteSessionToken(ctx context.Context, token string) error
	CreateChart(ctx context.Context, chartRequest *chart.CreateUpdateChartRequest) (*chart.Chart, error)
	CreateSloChart(ctx context.Context, chartRequest *chart.CreateUpdateSloChartRequest) (*chart.Chart, error)
	DeleteChart(ctx context.Context, id string) error
	GetChart(ctx context.Context, id string) (*chart.Chart, error)
	UpdateChart(ctx context.Context, id string, chartRequest *chart.CreateUpdateChartRequest) (*chart.Chart, error)
	UpdateSloChart(ctx context.Context, id string, chartRequest *chart.CreateUpdateSloChartRequest) (*chart.Chart, error)
	ValidateChart(ctx context.Context, chartRequest *chart.CreateUpdateChartRequest) error
	CreateWebhookIntegration(ctx context.Context, oi *integration.WebhookIntegration) (*integration.WebhookIntegration, error)
	GetWebhookIntegration(ctx context.Context, id string) (*integration.WebhookIntegration, error)
	UpdateWebhookIntegration(ctx context.Context, id string, oi *integration.WebhookIntegration) (*integration.WebhookIntegration, error)
	DeleteWebhookIntegration(ctx context.Context, id string) error
	CreateDashboard(ctx context.Context, dashboardRequest *dashboard.CreateUpdateDashboardRequest) (*dashboard.Dashboard, error)
	DeleteDashboard(ctx context.Context, id string) error
	GetDashboard(ctx context.Context, id string) (*dashboard.Dashboard, error)
	UpdateDashboard(ctx context.Context, id string, dashboardRequest *dashboard.CreateUpdateDashboardRequest) (*dashboard.Dashboard, error)
	ValidateDashboard(ctx context.Context, dashboardRequest *dashboard.CreateUpdateDashboardRequest) error
	ValidateDashboardWithMode(ctx context.Context, dashboardRequest *dashboard.CreateUpdateDashboardRequest, validationMode signalfx.VisualizationObjectsValidation) error
	CreateGCPIntegration(ctx context.Context, gcpi *integration.GCPIntegration) (*integration.GCPIntegration, error)
	GetGCPIntegration(ctx context.Context, id string) (*integration.GCPIntegration, error)
	UpdateGCPIntegration(ctx context.Context, id string, gcpi *integration.GCPIntegration) (*integration.GCPIntegration, error)
	DeleteGCPIntegration(ctx context.Context, id string) error
	CreateDashboardGroup(ctx context.Context, dashboardGroupRequest *dashboard_group.CreateUpdateDashboardGroupRequest, skipImplicitDashboard bool) (*dashboard_group.DashboardGroup, error)
	DeleteDashboardGroup(ctx context.Context, id string) error
	GetDashboardGroup(ctx context.Context, id string) (*dashboard_group.DashboardGroup, error)
	UpdateDashboardGroup(ctx context.Context, id string, dashboardGroupRequest *dashboard_group.CreateUpdateDashboardGroupRequest) (*dashboard_group.DashboardGroup, error)
	ListBuiltInDashboardGroups(ctx context.Context, limit int, offset int) (*dashboard_group.SearchResult, error)
	CreateJiraIntegration(ctx context.Context, ji *integration.JiraIntegration) (*integration.JiraIntegration, error)
	GetJiraIntegration(ctx context.Context, id string) (*integration.JiraIntegration, error)
	UpdateJiraIntegration(ctx context.Context, id string, ji *integration.JiraIntegration) (*integration.JiraIntegration, error)
	DeleteJiraIntegration(ctx context.Context, id string) error
	GetSlo(ctx context.Context, id string) (*slo.SloObject, error)
	CreateSlo(ctx context.Context, sloRequest *slo.SloObject) (*slo.SloObject, error)
	ValidateSlo(ctx context.Context, sloRequest *slo.SloObject) error
	UpdateSlo(ctx context.Context, id string, sloRequest *slo.SloObject) (*slo.SloObject, error)
	DeleteSlo(ctx context.Context, id string) error
	CreateBigPandaIntegration(ctx context.Context, in *integration.BigPandaIntegration) (*integration.BigPandaIntegration, error)
	GetBigPandaIntegration(ctx context.Context, id string) (*integration.BigPandaIntegration, error)
	UpdateBigPandaIntegration(ctx context.Context, id string, in *integration.BigPandaIntegration) (*integration.BigPandaIntegration, error)
	DeleteBigPandaIntegration(ctx context.Context, id string) error
	CreateServiceNowIntegration(ctx context.Context, in *integration.ServiceNowIntegration) (*integration.ServiceNowIntegration, error)
	GetServiceNowIntegration(ctx context.Context, id string) (*integration.ServiceNowIntegration, error)
	UpdateServiceNowIntegration(ctx context.Context, id string, in *integration.ServiceNowIntegration) (*integration.ServiceNowIntegration, error)
	DeleteServiceNowIntegration(ctx context.Context, id string) error
	CreateOpsgenieIntegration(ctx context.Context, oi *integration.OpsgenieIntegration) (*integration.OpsgenieIntegration, error)
	GetOpsgenieIntegration(ctx context.Context, id string) (*integration.OpsgenieIntegration, error)
	UpdateOpsgenieIntegration(ctx context.Context, id string, oi *integration.OpsgenieIntegration) (*integration.OpsgenieIntegration, error)
	DeleteOpsgenieIntegration(ctx context.Context, id string) error
	CreateAWSCloudWatchIntegration(ctx context.Context, acwi *integration.AwsCloudWatchIntegration) (*integration.AwsCloudWatchIntegration, error)
	GetAWSCloudWatchIntegration(ctx context.Context, id string) (*integration.AwsCloudWatchIntegration, error)
	UpdateAWSCloudWatchIntegration(ctx context.Context, id string, acwi *integration.AwsCloudWatchIntegration) (*integration.AwsCloudWatchIntegration, error)
	DeleteAWSCloudWatchIntegration(ctx context.Context, id string) error
	CreateAlertMutingRule(ctx context.Context, muteRequest *alertmuting.CreateUpdateAlertMutingRuleRequest) (*alertmuting.AlertMutingRule, error)
	DeleteAlertMutingRule(ctx context.Context, name string) error
	GetAlertMutingRule(ctx context.Context, id string) (*alertmuting.AlertMutingRule, error)
	UpdateAlertMutingRule(ctx context.Context, id string, muteRequest *alertmuting.CreateUpdateAlertMutingRuleRequest) (*alertmuting.AlertMutingRule, error)
	CreatePagerDutyIntegration(ctx context.Context, pdi *integration.PagerDutyIntegration) (*integration.PagerDutyIntegration, error)
	GetPagerDutyIntegration(ctx context.Context, id string) (*integration.PagerDutyIntegration, error)
	GetPagerDutyIntegrationByName(ctx context.Context, name string) (*integration.PagerDutyIntegration, error)
	UpdatePagerDutyIntegration(ctx context.Context, id string, pdi *integration.PagerDutyIntegration) (*integration.PagerDutyIntegration, error)
	DeletePagerDutyIntegration(ctx context.Context, id string) error
	CreateDetector(ctx context.Context, detectorRequest *detector.CreateUpdateDetectorRequest) (*detector.Detector, error)
	DeleteDetector(ctx context.Context, id string) error
	GetDetector(ctx context.Context, id string) (*detector.Detector, error)
	GetDetectors(ctx context.Context, limit int, name string, offset int) ([]*detector.Detector, error)
	UpdateDetector(ctx context.Context, id string, detectorRequest *detector.CreateUpdateDetectorRequest) (*detector.Detector, error)
	SearchDetectors(ctx context.Context, limit int, name string, offset int, tags string) (*detector.SearchResults, error)
	GetDetectorEvents(ctx context.Context, id string, from int, to int, offset int, limit int) ([]*detector.Event, error)
	GetDetectorIncidents(ctx context.Context, id string, offset int, limit int) ([]*detector.Incident, error)
	ValidateDetector(ctx context.Context, detectorRequest *detector.ValidateDetectorRequestModel) error
	CreateTeam(ctx context.Context, t *team.CreateUpdateTeamRequest) (*team.Team, error)
	DeleteTeam(ctx context.Context, id string) error
	GetTeam(ctx context.Context, id string) (*team.Team, error)
	UpdateTeam(ctx context.Context, id string, t *team.CreateUpdateTeamRequest) (*team.Team, error)
	GetMetricRuleset(ctx context.Context, id string) (*metric_ruleset.GetMetricRulesetResponse, error)
	CreateMetricRuleset(ctx context.Context, metricRuleset *metric_ruleset.CreateMetricRulesetRequest) (*metric_ruleset.CreateMetricRulesetResponse, error)
	UpdateMetricRuleset(ctx context.Context, id string, metricRuleset *metric_ruleset.UpdateMetricRulesetRequest) (*metric_ruleset.UpdateMetricRulesetResponse, error)
	DeleteMetricRuleset(ctx context.Context, id string) error
	GenerateAggregationMetricName(ctx context.Context, generateAggregationNameRequest metric_ruleset.GenerateAggregationNameRequest) (string, error)
	CreateDataLink(ctx context.Context, dataLinkRequest *datalink.CreateUpdateDataLinkRequest) (*datalink.DataLink, error)
	DeleteDataLink(ctx context.Context, id string) error
	GetDataLink(ctx context.Context, id string) (*datalink.DataLink, error)
	UpdateDataLink(ctx context.Context, id string, dataLinkRequest *datalink.CreateUpdateDataLinkRequest) (*datalink.DataLink, error)
	SearchDataLinks(ctx context.Context, limit int, context string, offset int) (*datalink.SearchResults, error)
	GetIntegration(ctx context.Context, id string) (map[string]interface{}, error)
	DeleteIntegration(ctx context.Context, id string) error
	CreateVictorOpsIntegration(ctx context.Context, oi *integration.VictorOpsIntegration) (*integration.VictorOpsIntegration, error)
	GetVictorOpsIntegration(ctx context.Context, id string) (*integration.VictorOpsIntegration, error)
	UpdateVictorOpsIntegration(ctx context.Context, id string, oi *integration.VictorOpsIntegration) (*integration.VictorOpsIntegration, error)
	DeleteVictorOpsIntegration(ctx context.Context, id string) error
	GetSettings(ctx context.Context) (*automated_archival.AutomatedArchivalSettings, error)
	CreateSettings(ctx context.Context, settings *automated_archival.AutomatedArchivalSettings) (*automated_archival.AutomatedArchivalSettings, error)
	UpdateSettings(ctx context.Context, settings *automated_archival.AutomatedArchivalSettings) (*automated_archival.AutomatedArchivalSettings, error)
	DeleteSettings(ctx context.Context, deleteSettingsRequest *automated_archival.AutomatedArchivalSettingsDeleteRequest) error
	GetExemptMetrics(ctx context.Context) (*[]automated_archival.ExemptMetric, error)
	CreateExemptMetrics(ctx context.Context, exemptMetrics *[]automated_archival.ExemptMetric) (*[]automated_archival.ExemptMetric, error)
	DeleteExemptMetrics(ctx context.Context, deleteExemptMetricsRequest *automated_archival.ExemptMetricDeleteRequest) error
	CreateAzureIntegration(ctx context.Context, acwi *integration.AzureIntegration) (*integration.AzureIntegration, error)
	GetAzureIntegration(ctx context.Context, id string) (*integration.AzureIntegration, error)
	UpdateAzureIntegration(ctx context.Context, id string, acwi *integration.AzureIntegration) (*integration.AzureIntegration, error)
	DeleteAzureIntegration(ctx context.Context, id string) error
	CreateSlackIntegration(ctx context.Context, si *integration.SlackIntegration) (*integration.SlackIntegration, error)
	GetSlackIntegration(ctx context.Context, id string) (*integration.SlackIntegration, error)
	UpdateSlackIntegration(ctx context.Context, id string, si *integration.SlackIntegration) (*integration.SlackIntegration, error)
	DeleteSlackIntegration(ctx context.Context, id string) error
	CreateOrgToken(ctx context.Context, tokenRequest *orgtoken.CreateUpdateTokenRequest) (*orgtoken.Token, error)
	DeleteOrgToken(ctx context.Context, name string) error
	GetOrgToken(ctx context.Context, id string) (*orgtoken.Token, error)
	UpdateOrgToken(ctx context.Context, id string, tokenRequest *orgtoken.CreateUpdateTokenRequest) (*orgtoken.Token, error)
	SearchDimension(ctx context.Context, query string, orderBy string, limit int, offset int) (*metrics_metadata.DimensionQueryResponseModel, error)
	GetOrganizationMembers(ctx context.Context, limit int, query string, offset int, orderBy string) (*organization.MemberSearchResults, error)
}
