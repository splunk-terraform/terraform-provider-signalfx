package pmeta

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	"github.com/signalfx/signalfx-go"
	"github.com/signalfx/signalfx-go/alertmuting"
	automated_archival "github.com/signalfx/signalfx-go/automated-archival"
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

var _ Client = &FileClient{}

type FileClient struct {
	BaseDir string
}

func (f *FileClient) GetOrganizationMembers(ctx context.Context, limit int, query string, offset int, orderBy string) (*organization.MemberSearchResults, error) {
	panic("unsupported")
}

func (f *FileClient) SearchDimension(context.Context, string, string, int, int) (*metrics_metadata.DimensionQueryResponseModel, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateOrgToken(context.Context, string, *orgtoken.CreateUpdateTokenRequest) (*orgtoken.Token, error) {
	panic("unsupported")
}

func (f *FileClient) CreateSessionToken(_ context.Context, _ *sessiontoken.CreateTokenRequest) (*sessiontoken.Token, error) {
	return &sessiontoken.Token{}, nil
}

func (f *FileClient) DeleteSessionToken(context.Context, string) error {
	panic("unsupported")
}

func createID() string {
	var b []byte
	for i := range 12 {
		randomInt := rand.IntN(63)
		b[i] = byte(randomInt)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func (f *FileClient) CreateChart(ctx context.Context, chartRequest *chart.CreateUpdateChartRequest) (*chart.Chart, error) {
	id := createID()
	return f.UpdateChart(ctx, id, chartRequest)
}

func (f *FileClient) CreateSloChart(_ context.Context, _ *chart.CreateUpdateSloChartRequest) (*chart.Chart, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteChart(_ context.Context, id string) error {
	path := filepath.Join(f.BaseDir, "charts", fmt.Sprintf("%s.json", id))
	return os.Remove(path)
}

func (f *FileClient) GetChart(_ context.Context, id string) (*chart.Chart, error) {
	path := filepath.Join(f.BaseDir, "charts", fmt.Sprintf("%s.json", id))
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c chart.Chart
	err = json.Unmarshal(b, &c)
	return &c, err
}

func (f *FileClient) UpdateChart(_ context.Context, id string, chartRequest *chart.CreateUpdateChartRequest) (*chart.Chart, error) {
	path := filepath.Join(f.BaseDir, "charts", fmt.Sprintf("%s.json", id))
	c := &chart.Chart{
		Description:           chartRequest.Description,
		Id:                    id,
		LastUpdated:           time.Now().UnixMilli(),
		LastUpdatedBy:         "",
		Name:                  chartRequest.Name,
		Options:               chartRequest.Options,
		PackageSpecifications: chartRequest.PackageSpecifications,
		ProgramText:           chartRequest.ProgramText,
		Tags:                  chartRequest.Tags,
		SloId:                 "",
	}
	b, err := json.Marshal(chartRequest)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return nil, err
	}
	return c, nil
}

func (f *FileClient) UpdateSloChart(context.Context, string, *chart.CreateUpdateSloChartRequest) (*chart.Chart, error) {
	panic("unsupported")
}

func (f *FileClient) ValidateChart(context.Context, *chart.CreateUpdateChartRequest) error {
	return nil
}

func (f *FileClient) CreateWebhookIntegration(ctx context.Context, oi *integration.WebhookIntegration) (*integration.WebhookIntegration, error) {
	id := createID()
	return f.UpdateWebhookIntegration(ctx, id, oi)
}

func (f *FileClient) GetWebhookIntegration(_ context.Context, id string) (*integration.WebhookIntegration, error) {
	path := filepath.Join(f.BaseDir, "webhook_integrations", fmt.Sprintf("%s.json", id))
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var i integration.WebhookIntegration
	err = json.Unmarshal(b, &i)
	return &i, err
}

func (f *FileClient) UpdateWebhookIntegration(_ context.Context, id string, oi *integration.WebhookIntegration) (*integration.WebhookIntegration, error) {
	path := filepath.Join(f.BaseDir, "webhook_integrations", fmt.Sprintf("%s.json", id))
	oi.Id = id
	b, err := json.Marshal(oi)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return nil, err
	}
	return oi, nil
}

func (f *FileClient) DeleteWebhookIntegration(_ context.Context, id string) error {
	path := filepath.Join(f.BaseDir, "webhook_integrations", fmt.Sprintf("%s.json", id))
	return os.Remove(path)
}

func (f *FileClient) CreateDashboard(ctx context.Context, dashboardRequest *dashboard.CreateUpdateDashboardRequest) (*dashboard.Dashboard, error) {
	id := createID()
	return f.UpdateDashboard(ctx, id, dashboardRequest)
}

func (f *FileClient) DeleteDashboard(_ context.Context, id string) error {
	path := filepath.Join(f.BaseDir, "dashboards", fmt.Sprintf("%s.json", id))
	return os.Remove(path)
}

func (f *FileClient) GetDashboard(_ context.Context, id string) (*dashboard.Dashboard, error) {
	path := filepath.Join(f.BaseDir, "dashboards", fmt.Sprintf("%s.json", id))
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d dashboard.Dashboard
	err = json.Unmarshal(b, &d)
	return &d, err
}

func (f *FileClient) UpdateDashboard(_ context.Context, id string, dashboardRequest *dashboard.CreateUpdateDashboardRequest) (*dashboard.Dashboard, error) {
	path := filepath.Join(f.BaseDir, "dashboards", fmt.Sprintf("%s.json", id))
	d := &dashboard.Dashboard{
		AuthorizedWriters:     dashboardRequest.AuthorizedWriters,
		Permissions:           dashboardRequest.Permissions,
		ChartDensity:          &dashboardRequest.ChartDensity,
		Charts:                dashboardRequest.Charts,
		Created:               0,
		Creator:               "",
		CustomProperties:      nil,
		Description:           dashboardRequest.Description,
		DiscoveryOptions:      dashboardRequest.DiscoveryOptions,
		EventOverlays:         dashboardRequest.EventOverlays,
		Filters:               dashboardRequest.Filters,
		GroupId:               dashboardRequest.GroupId,
		Id:                    id,
		LastUpdated:           0,
		LastUpdatedBy:         "",
		Locked:                false,
		MaxDelayOverride:      nil,
		Name:                  dashboardRequest.Name,
		SelectedEventOverlays: dashboardRequest.SelectedEventOverlays,
		Tags:                  dashboardRequest.Tags,
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return nil, err
	}
	return d, nil
}

func (f *FileClient) ValidateDashboard(context.Context, *dashboard.CreateUpdateDashboardRequest) error {
	panic("unsupported")
}

func (f *FileClient) ValidateDashboardWithMode(context.Context, *dashboard.CreateUpdateDashboardRequest, signalfx.VisualizationObjectsValidation) error {
	panic("unsupported")
}

func (f *FileClient) CreateDashboardGroup(ctx context.Context, dashboardGroupRequest *dashboard_group.CreateUpdateDashboardGroupRequest, skipImplicitDashboard bool) (*dashboard_group.DashboardGroup, error) {
	id := createID()
	return f.UpdateDashboardGroup(ctx, id, dashboardGroupRequest)
}

func (f *FileClient) DeleteDashboardGroup(_ context.Context, id string) error {
	path := filepath.Join(f.BaseDir, "dashboard_groups", fmt.Sprintf("%s.json", id))
	return os.Remove(path)
}

func (f *FileClient) GetDashboardGroup(_ context.Context, id string) (*dashboard_group.DashboardGroup, error) {
	path := filepath.Join(f.BaseDir, "dashboard_groups", fmt.Sprintf("%s.json", id))
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d dashboard_group.DashboardGroup
	err = json.Unmarshal(b, &d)
	return &d, err
}

func (f *FileClient) UpdateDashboardGroup(_ context.Context, id string, dashboardGroupRequest *dashboard_group.CreateUpdateDashboardGroupRequest) (*dashboard_group.DashboardGroup, error) {
	path := filepath.Join(f.BaseDir, "dashboard_groups", fmt.Sprintf("%s.json", id))
	d := &dashboard_group.DashboardGroup{
		AuthorizedWriters: dashboardGroupRequest.AuthorizedWriters,
		Permissions:       dashboardGroupRequest.Permissions,
		Created:           0,
		Creator:           "",
		DashboardConfigs:  dashboardGroupRequest.DashboardConfigs,
		Dashboards:        dashboardGroupRequest.Dashboards,
		Description:       dashboardGroupRequest.Description,
		Id:                id,
		ImportQualifiers:  dashboardGroupRequest.ImportQualifiers,
		LastUpdated:       0,
		LastUpdatedBy:     "",
		Name:              dashboardGroupRequest.Name,
		Teams:             dashboardGroupRequest.Teams,
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return nil, err
	}
	return d, nil
}

func (f *FileClient) GetBigPandaIntegration(ctx context.Context, id string) (*integration.BigPandaIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateBigPandaIntegration(ctx context.Context, id string, in *integration.BigPandaIntegration) (*integration.BigPandaIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteBigPandaIntegration(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) CreateServiceNowIntegration(ctx context.Context, in *integration.ServiceNowIntegration) (*integration.ServiceNowIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) GetServiceNowIntegration(ctx context.Context, id string) (*integration.ServiceNowIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateServiceNowIntegration(ctx context.Context, id string, in *integration.ServiceNowIntegration) (*integration.ServiceNowIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteServiceNowIntegration(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) CreateOpsgenieIntegration(ctx context.Context, oi *integration.OpsgenieIntegration) (*integration.OpsgenieIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) GetOpsgenieIntegration(ctx context.Context, id string) (*integration.OpsgenieIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateOpsgenieIntegration(ctx context.Context, id string, oi *integration.OpsgenieIntegration) (*integration.OpsgenieIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteOpsgenieIntegration(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) CreateAWSCloudWatchIntegration(ctx context.Context, acwi *integration.AwsCloudWatchIntegration) (*integration.AwsCloudWatchIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) GetAWSCloudWatchIntegration(ctx context.Context, id string) (*integration.AwsCloudWatchIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateAWSCloudWatchIntegration(ctx context.Context, id string, acwi *integration.AwsCloudWatchIntegration) (*integration.AwsCloudWatchIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteAWSCloudWatchIntegration(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) CreateAlertMutingRule(ctx context.Context, muteRequest *alertmuting.CreateUpdateAlertMutingRuleRequest) (*alertmuting.AlertMutingRule, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteAlertMutingRule(ctx context.Context, name string) error {
	panic("unsupported")
}

func (f *FileClient) GetAlertMutingRule(ctx context.Context, id string) (*alertmuting.AlertMutingRule, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateAlertMutingRule(ctx context.Context, id string, muteRequest *alertmuting.CreateUpdateAlertMutingRuleRequest) (*alertmuting.AlertMutingRule, error) {
	panic("unsupported")
}

func (f *FileClient) CreatePagerDutyIntegration(ctx context.Context, pdi *integration.PagerDutyIntegration) (*integration.PagerDutyIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) GetPagerDutyIntegration(ctx context.Context, id string) (*integration.PagerDutyIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) GetPagerDutyIntegrationByName(ctx context.Context, name string) (*integration.PagerDutyIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) UpdatePagerDutyIntegration(ctx context.Context, id string, pdi *integration.PagerDutyIntegration) (*integration.PagerDutyIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) DeletePagerDutyIntegration(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) CreateDetector(ctx context.Context, detectorRequest *detector.CreateUpdateDetectorRequest) (*detector.Detector, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteDetector(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) GetDetector(ctx context.Context, id string) (*detector.Detector, error) {
	panic("unsupported")
}

func (f *FileClient) GetDetectors(ctx context.Context, limit int, name string, offset int) ([]*detector.Detector, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateDetector(ctx context.Context, id string, detectorRequest *detector.CreateUpdateDetectorRequest) (*detector.Detector, error) {
	panic("unsupported")
}

func (f *FileClient) SearchDetectors(ctx context.Context, limit int, name string, offset int, tags string) (*detector.SearchResults, error) {
	panic("unsupported")
}

func (f *FileClient) GetDetectorEvents(ctx context.Context, id string, from int, to int, offset int, limit int) ([]*detector.Event, error) {
	panic("unsupported")
}

func (f *FileClient) GetDetectorIncidents(ctx context.Context, id string, offset int, limit int) ([]*detector.Incident, error) {
	panic("unsupported")
}

func (f *FileClient) ValidateDetector(ctx context.Context, detectorRequest *detector.ValidateDetectorRequestModel) error {
	panic("unsupported")
}

func (f *FileClient) CreateTeam(ctx context.Context, t *team.CreateUpdateTeamRequest) (*team.Team, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteTeam(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) GetTeam(ctx context.Context, id string) (*team.Team, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateTeam(ctx context.Context, id string, t *team.CreateUpdateTeamRequest) (*team.Team, error) {
	panic("unsupported")
}

func (f *FileClient) GetMetricRuleset(ctx context.Context, id string) (*metric_ruleset.GetMetricRulesetResponse, error) {
	panic("unsupported")
}

func (f *FileClient) CreateMetricRuleset(ctx context.Context, metricRuleset *metric_ruleset.CreateMetricRulesetRequest) (*metric_ruleset.CreateMetricRulesetResponse, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateMetricRuleset(ctx context.Context, id string, metricRuleset *metric_ruleset.UpdateMetricRulesetRequest) (*metric_ruleset.UpdateMetricRulesetResponse, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteMetricRuleset(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) GenerateAggregationMetricName(ctx context.Context, generateAggregationNameRequest metric_ruleset.GenerateAggregationNameRequest) (string, error) {
	panic("unsupported")
}

func (f *FileClient) CreateDataLink(ctx context.Context, dataLinkRequest *datalink.CreateUpdateDataLinkRequest) (*datalink.DataLink, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteDataLink(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) GetDataLink(ctx context.Context, id string) (*datalink.DataLink, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateDataLink(ctx context.Context, id string, dataLinkRequest *datalink.CreateUpdateDataLinkRequest) (*datalink.DataLink, error) {
	panic("unsupported")
}

func (f *FileClient) SearchDataLinks(ctx context.Context, limit int, context string, offset int) (*datalink.SearchResults, error) {
	panic("unsupported")
}

func (f *FileClient) GetIntegration(ctx context.Context, id string) (map[string]interface{}, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteIntegration(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) CreateVictorOpsIntegration(ctx context.Context, oi *integration.VictorOpsIntegration) (*integration.VictorOpsIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) GetVictorOpsIntegration(ctx context.Context, id string) (*integration.VictorOpsIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateVictorOpsIntegration(ctx context.Context, id string, oi *integration.VictorOpsIntegration) (*integration.VictorOpsIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteVictorOpsIntegration(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) GetSettings(ctx context.Context) (*automated_archival.AutomatedArchivalSettings, error) {
	panic("unsupported")
}

func (f *FileClient) CreateSettings(ctx context.Context, settings *automated_archival.AutomatedArchivalSettings) (*automated_archival.AutomatedArchivalSettings, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateSettings(ctx context.Context, settings *automated_archival.AutomatedArchivalSettings) (*automated_archival.AutomatedArchivalSettings, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteSettings(ctx context.Context, deleteSettingsRequest *automated_archival.AutomatedArchivalSettingsDeleteRequest) error {
	panic("unsupported")
}

func (f *FileClient) GetExemptMetrics(ctx context.Context) (*[]automated_archival.ExemptMetric, error) {
	panic("unsupported")
}

func (f *FileClient) CreateExemptMetrics(ctx context.Context, exemptMetrics *[]automated_archival.ExemptMetric) (*[]automated_archival.ExemptMetric, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteExemptMetrics(ctx context.Context, deleteExemptMetricsRequest *automated_archival.ExemptMetricDeleteRequest) error {
	panic("unsupported")
}

func (f *FileClient) CreateAzureIntegration(context.Context, *integration.AzureIntegration) (*integration.AzureIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) GetAzureIntegration(context.Context, string) (*integration.AzureIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateAzureIntegration(context.Context, string, *integration.AzureIntegration) (*integration.AzureIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteAzureIntegration(context.Context, string) error {
	panic("unsupported")
}

func (f *FileClient) CreateSlackIntegration(context.Context, *integration.SlackIntegration) (*integration.SlackIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) GetSlackIntegration(context.Context, string) (*integration.SlackIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateSlackIntegration(context.Context, string, *integration.SlackIntegration) (*integration.SlackIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteSlackIntegration(context.Context, string) error {
	panic("unsupported")
}

func (f *FileClient) CreateOrgToken(context.Context, *orgtoken.CreateUpdateTokenRequest) (*orgtoken.Token, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteOrgToken(ctx context.Context, name string) error {
	panic("unsupported")
}

func (f *FileClient) GetOrgToken(ctx context.Context, id string) (*orgtoken.Token, error) {
	panic("unsupported")
}

func (f *FileClient) CreateGCPIntegration(ctx context.Context, gcpi *integration.GCPIntegration) (*integration.GCPIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) GetGCPIntegration(ctx context.Context, id string) (*integration.GCPIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateGCPIntegration(ctx context.Context, id string, gcpi *integration.GCPIntegration) (*integration.GCPIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteGCPIntegration(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) ListBuiltInDashboardGroups(ctx context.Context, limit int, offset int) (*dashboard_group.SearchResult, error) {
	panic("unsupported")
}

func (f *FileClient) CreateJiraIntegration(ctx context.Context, ji *integration.JiraIntegration) (*integration.JiraIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) GetJiraIntegration(ctx context.Context, id string) (*integration.JiraIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) UpdateJiraIntegration(ctx context.Context, id string, ji *integration.JiraIntegration) (*integration.JiraIntegration, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteJiraIntegration(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) GetSlo(ctx context.Context, id string) (*slo.SloObject, error) {
	panic("unsupported")
}

func (f *FileClient) CreateSlo(ctx context.Context, sloRequest *slo.SloObject) (*slo.SloObject, error) {
	panic("unsupported")
}

func (f *FileClient) ValidateSlo(ctx context.Context, sloRequest *slo.SloObject) error {
	panic("unsupported")
}

func (f *FileClient) UpdateSlo(ctx context.Context, id string, sloRequest *slo.SloObject) (*slo.SloObject, error) {
	panic("unsupported")
}

func (f *FileClient) DeleteSlo(ctx context.Context, id string) error {
	panic("unsupported")
}

func (f *FileClient) CreateBigPandaIntegration(ctx context.Context, in *integration.BigPandaIntegration) (*integration.BigPandaIntegration, error) {
	panic("unsupported")
}
