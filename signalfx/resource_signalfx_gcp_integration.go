// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/signalfx/signalfx-go/integration"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/convert"
)

func integrationGCPResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the integration",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether the integration is enabled or not",
			},
			"poll_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      300,
				Description:  "GCP poll rate (in seconds). Between `60` and `600`.",
				ValidateFunc: validation.IntBetween(60, 600),
			},
			"services": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "GCP enabled services",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"custom_metric_type_domains": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "List of additional GCP service domain names that you want to monitor",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"auth_method": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{string(integration.SERVICE_ACCOUNT_KEY), string(integration.WORKLOAD_IDENTITY_FEDERATION)}, true),
				Description:  "Authentication method to use in this integration. If empty, Splunk Observability backend defaults to SERVICE_ACCOUNT_KEY",
			},
			"project_service_keys": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "GCP project service keys",
				Sensitive:   true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"project_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"project_key": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},
					},
				},
				ConflictsWith: []string{"project_wif_configs"},
			},
			"project_wif_configs": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "GCP WIF configs",
				Deprecated:  "Please use workload_identity_federation_config with projects instead",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"project_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"wif_config": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				ConflictsWith: []string{"project_service_keys"},
			},
			"workload_identity_federation_config": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "Workload Identity Federation configuration JSON",
				ConflictsWith: []string{"project_service_keys", "project_wif_configs"},
			},
			"projects": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "GCP projects configuration",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sync_mode": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateSyncMode,
							Default:      "SELECTED",
							Description:  "What mode of synchronizing projects should be used. Sync all tries to synchronize metrics and metadata from all discoverable projects.",
						},
						"selected_project_ids": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "List of project IDs to synchronize metrics and metadata from. Use only if you don't want to automatically synchronize all projects.",
						},
					},
				},
			},
			"wif_splunk_identity": {
				Type:        schema.TypeMap,
				Computed:    true,
				Optional:    true,
				Description: "The Splunk Observability GCP identity to include in GCP WIF provider definition.",
			},
			"use_metric_source_project_for_quota": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "When this value is set to true Observability Cloud will force usage of a quota from the project where metrics are stored. For this to work the service account provided for the project needs to be provided with serviceusage.services.use permission or Service Usage Consumer role in this project. When set to false default quota settings are used.",
			},
			"include_list": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "List of custom metadata keys that you want Observability Cloud to collect for Compute Engine instances.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"named_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A named token to use for ingest",
				ForceNew:    true,
			},
			"import_gcp_metrics": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "If enabled, Splunk Observability Cloud will sync also Google Cloud Metrics data. If disabled, Splunk Observability Cloud will import only metadata. Defaults to true.",
			},
		},

		Create: integrationGCPCreate,
		Read:   integrationGCPRead,
		Update: integrationGCPUpdate,
		Delete: integrationGCPDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func integrationGCPRead(d *schema.ResourceData, meta any) error {
	config := meta.(*signalfxConfig)

	int, err := config.Client.GetGCPIntegration(context.TODO(), d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	return gcpIntegrationAPIToTF(d, int)
}

func getGCPPayloadIntegration(d *schema.ResourceData) *integration.GCPIntegration {
	importGCPMetrics := d.Get("import_gcp_metrics").(bool)
	gcp := &integration.GCPIntegration{
		Name:                           d.Get("name").(string),
		Enabled:                        d.Get("enabled").(bool),
		UseMetricSourceProjectForQuota: d.Get("use_metric_source_project_for_quota").(bool),
		Type:                           "GCP",
		ImportGCPMetrics:               &importGCPMetrics,
	}

	if val, ok := d.GetOk("named_token"); ok {
		gcp.NamedToken = val.(string)
	}

	if val, ok := d.GetOk("poll_rate"); ok {
		gcp.PollRateMs = int64(val.(int)) * 1000
	}

	if val, ok := d.GetOk("services"); ok {
		servs := val.(*schema.Set).List()
		services := make([]integration.GcpService, len(servs))
		for i, v := range servs {
			v := integration.GcpService(v.(string))
			services[i] = v
		}
		gcp.Services = services
	}

	if val, ok := d.GetOk("auth_method"); ok {
		gcp.AuthMethod = integration.GCPAuthMethod(strings.ToUpper(val.(string)))
	}

	if val, ok := d.GetOk("project_service_keys"); ok {
		keys := val.(*schema.Set).List()
		serviceKeys := make([]*integration.GCPProject, len(keys))
		for i, v := range keys {
			v := v.(map[string]any)
			serviceKeys[i] = &integration.GCPProject{
				ProjectId:  v["project_id"].(string),
				ProjectKey: v["project_key"].(string),
			}
		}
		gcp.ProjectServiceKeys = serviceKeys
	}
	if val, ok := d.GetOk("project_wif_configs"); ok {
		keys := val.(*schema.Set).List()
		wifConfigs := make([]*integration.GCPProjectWIFConfig, len(keys))
		for i, v := range keys {
			v := v.(map[string]any)
			wifConfigs[i] = &integration.GCPProjectWIFConfig{
				ProjectId: v["project_id"].(string),
				WIFConfig: v["wif_config"].(string),
			}
		}
		gcp.WifConfigs = wifConfigs
	}

	if val, ok := d.GetOk("workload_identity_federation_config"); ok {
		gcp.WorkloadIdentityFederationConfig = val.(string)
	}

	if val, ok := d.GetOk("projects"); ok {
		projects := &integration.GCPProjects{}
		projectsList := val.([]any)
		if len(projectsList) > 0 {
			projectsConfig := projectsList[0].(map[string]any)

			if syncMode, exists := projectsConfig["sync_mode"]; exists && syncMode != nil {
				projects.SyncMode = integration.SyncMode(syncMode.(string))
			}

			if selectedProjects, exists := projectsConfig["selected_project_ids"]; exists && selectedProjects != nil {
				if projectSet, ok := selectedProjects.(*schema.Set); ok {
					projectList := projectSet.List()
					projectIds := make([]string, len(projectList))
					for i, project := range projectList {
						projectIds[i] = project.(string)
					}
					projects.SelectedProjectIds = projectIds
				}
			}
		}
		gcp.Projects = projects
	}

	if val, ok := d.GetOk("include_list"); ok {
		gcp.IncludeList = convert.SchemaListAll(val, convert.ToString)
	}

	if val, ok := d.GetOk("custom_metric_type_domains"); ok {
		gcp.CustomMetricTypeDomains = convert.SchemaListAll(val, convert.ToString)
	}

	return gcp
}

func gcpIntegrationAPIToTF(d *schema.ResourceData, gcp *integration.GCPIntegration) error {
	debugOutput, _ := json.Marshal(gcp)
	log.Printf("[DEBUG] SignalFx: Got GCP Integration to enState: %s", string(debugOutput))

	if err := d.Set("name", gcp.Name); err != nil {
		return err
	}
	if err := d.Set("enabled", gcp.Enabled); err != nil {
		return err
	}
	if err := d.Set("use_metric_source_project_for_quota", gcp.UseMetricSourceProjectForQuota); err != nil {
		return err
	}
	if err := d.Set("poll_rate", gcp.PollRateMs/1000); err != nil {
		return err
	}
	if err := d.Set("named_token", gcp.NamedToken); err != nil {
		return err
	}
	if err := d.Set("import_gcp_metrics", *gcp.ImportGCPMetrics); err != nil {
		return err
	}

	if len(gcp.Services) > 0 {
		services := make([]any, len(gcp.Services))
		for i, v := range gcp.Services {
			services[i] = string(v)
		}
		if err := d.Set("services", schema.NewSet(schema.HashString, services)); err != nil {
			return err
		}
	}

	if err := d.Set("auth_method", gcp.AuthMethod); err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Got GCP Integration wifConfig assign")

	if gcp.WifConfigs != nil {
		wifConfigs := convertWifConfigsToMap(gcp.WifConfigs)
		if err := d.Set("project_wif_configs", wifConfigs); err != nil {
			return fmt.Errorf("error setting project_wif_configs: %w", err)
		}
	} else {
		if err := d.Set("project_wif_configs", []any{}); err != nil {
			return fmt.Errorf("error unsetting project_wif_configs: %w", err)
		}
	}

	if err := d.Set("workload_identity_federation_config", gcp.WorkloadIdentityFederationConfig); err != nil {
		return fmt.Errorf("error setting workload_identity_federation_config: %w", err)
	}

	if gcp.Projects != nil {
		// Set projects field as a list with proper handling of selected_project_ids as a set
		projectsConfig := map[string]any{
			"sync_mode": string(gcp.Projects.SyncMode),
		}

		if len(gcp.Projects.SelectedProjectIds) > 0 {
			projectIds := make([]any, len(gcp.Projects.SelectedProjectIds))
			for i, id := range gcp.Projects.SelectedProjectIds {
				projectIds[i] = id
			}
			projectsConfig["selected_project_ids"] = schema.NewSet(schema.HashString, projectIds)
		} else {
			projectsConfig["selected_project_ids"] = schema.NewSet(schema.HashString, []any{})
		}

		projectsList := []any{projectsConfig}
		if err := d.Set("projects", projectsList); err != nil {
			return fmt.Errorf("error setting projects: %w", err)
		}
	}

	if err := d.Set("wif_splunk_identity", gcp.WifSplunkIdentity); err != nil {
		return err
	}
	// Note that the API doesn't return the project keys so we ignore them,
	// because there's not much reason to poke at just the project id.

	if err := d.Set("include_list", flattenStringSliceToSet(gcp.IncludeList)); err != nil {
		return err
	}

	if err := d.Set("custom_metric_type_domains", flattenStringSliceToSet(gcp.CustomMetricTypeDomains)); err != nil {
		return err
	}

	return nil
}

func integrationGCPCreate(d *schema.ResourceData, meta any) error {
	config := meta.(*signalfxConfig)
	payload := getGCPPayloadIntegration(d)

	// Convert payload to JSON to see what will be sent
	debugOutput, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling GCP integration payload to JSON for debug: %w", err)
	}

	// Print the JSON payload for debugging purposes
	log.Printf("[DEBUG] SignalFx: Create GCP Integration Payload: %s", string(debugOutput))

	// Make the actual API request to create the GCP Integration
	int, err := config.Client.CreateGCPIntegration(context.TODO(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return gcpIntegrationAPIToTF(d, int)
}
func integrationGCPUpdate(d *schema.ResourceData, meta any) error {
	config := meta.(*signalfxConfig)
	payload := getGCPPayloadIntegration(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update GCP Integration Payload: %s", string(debugOutput))

	int, err := config.Client.UpdateGCPIntegration(context.TODO(), d.Id(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return gcpIntegrationAPIToTF(d, int)
}

func integrationGCPDelete(d *schema.ResourceData, meta any) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteGCPIntegration(context.TODO(), d.Id())
}

func convertWifConfigsToMap(wifConfigs []*integration.GCPProjectWIFConfig) []map[string]any {

	result := make([]map[string]any, len(wifConfigs))
	for i, v := range wifConfigs {
		result[i] = map[string]any{
			"project_id": v.ProjectId,
			"wif_config": v.WIFConfig,
		}
	}
	return result
}

func validateSyncMode(v any, _ string) (we []string, errors []error) {
	value := v.(string)
	if value != string(integration.ALL_REACHABLE) && value != string(integration.SELECTED) {
		errors = append(errors, fmt.Errorf("%s not allowed; sync mode must be one of %s or %s", value, integration.ALL_REACHABLE, integration.SELECTED))
	}
	return
}
