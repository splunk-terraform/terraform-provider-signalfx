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
)

func integrationGCPResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the integration",
			},
			"enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether the integration is enabled or not",
			},
			"poll_rate": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      300,
				Description:  "GCP poll rate (in seconds). Between `60` and `600`.",
				ValidateFunc: validation.IntBetween(60, 600),
			},
			"services": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "GCP enabled services",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"custom_metric_type_domains": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "List of additional GCP service domain names that you want to monitor",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"project_service_keys": &schema.Schema{
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
			},
			"use_metric_source_project_for_quota": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "When this value is set to true Observability Cloud will force usage of a quota from the project where metrics are stored. For this to work the service account provided for the project needs to be provided with serviceusage.services.use permission or Service Usage Consumer role in this project. When set to false default quota settings are used.",
			},
			"include_list": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "List of custom metadata keys that you want Observability Cloud to collect for Compute Engine instances.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"named_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A named token to use for ingest",
				ForceNew:    true,
			},
			"import_gcp_metrics": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "If enabled, SignalFx will sync also Google Cloud Metrics data. If disabled, SignalFx will import only metadata. Defaults to true.",
			},
		},

		Create: integrationGCPCreate,
		Read:   integrationGCPRead,
		Update: integrationGCPUpdate,
		Delete: integrationGCPDelete,
		Exists: integrationGCPExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func integrationGCPExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetGCPIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func integrationGCPRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	int, err := config.Client.GetGCPIntegration(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
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

	if val, ok := d.GetOk("project_service_keys"); ok {
		keys := val.(*schema.Set).List()
		serviceKeys := make([]*integration.GCPProject, len(keys))
		for i, v := range keys {
			v := v.(map[string]interface{})
			serviceKeys[i] = &integration.GCPProject{
				ProjectId:  v["project_id"].(string),
				ProjectKey: v["project_key"].(string),
			}
		}
		gcp.ProjectServiceKeys = serviceKeys
	}

	if val, ok := d.GetOk("include_list"); ok {
		gcp.IncludeList = expandStringSetToSlice(val.(*schema.Set))
	}

	if val, ok := d.GetOk("custom_metric_type_domains"); ok {
		gcp.CustomMetricTypeDomains = expandStringSetToSlice(val.(*schema.Set))
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
		services := make([]interface{}, len(gcp.Services))
		for i, v := range gcp.Services {
			services[i] = string(v)
		}
		if err := d.Set("services", schema.NewSet(schema.HashString, services)); err != nil {
			return err
		}
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

func integrationGCPCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getGCPPayloadIntegration(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create GCP Integration Payload: %s", string(debugOutput))

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

func integrationGCPUpdate(d *schema.ResourceData, meta interface{}) error {
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

func integrationGCPDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteGCPIntegration(context.TODO(), d.Id())
}
