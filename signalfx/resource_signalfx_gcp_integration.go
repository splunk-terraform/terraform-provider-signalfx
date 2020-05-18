package signalfx

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
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
				Description:  "GCP poll rate",
				ValidateFunc: validation.IntInSlice([]int{60, 300}),
			},
			"services": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "GCP enabled services",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateGcpService,
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
	_, err := config.Client.GetGCPIntegration(d.Id())
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
	int, err := config.Client.GetGCPIntegration(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	return gcpIntegrationAPIToTF(d, int)
}

func getGCPPayloadIntegration(d *schema.ResourceData) *integration.GCPIntegration {
	gcp := &integration.GCPIntegration{
		Name:    d.Get("name").(string),
		Enabled: d.Get("enabled").(bool),
		Type:    "GCP",
	}

	if val, ok := d.GetOk("poll_rate"); ok {
		val := val.(int)
		if val != 0 {
			pollRate := integration.OneMinutely
			if val == 300 {
				pollRate = integration.FiveMinutely
			}
			gcp.PollRate = &pollRate
		}
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
	if err := d.Set("poll_rate", *gcp.PollRate/1000); err != nil {
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
	return nil
}

func integrationGCPCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getGCPPayloadIntegration(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create GCP Integration Payload: %s", string(debugOutput))

	int, err := config.Client.CreateGCPIntegration(payload)
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

	int, err := config.Client.UpdateGCPIntegration(d.Id(), payload)
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

	return config.Client.DeleteGCPIntegration(d.Id())
}

func validateGcpService(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	for key, _ := range integration.GcpServiceNames {
		if key == value {
			return
		}
	}
	errors = append(errors, fmt.Errorf("%s not allowed; consult the documentation for a list of valid GCP service names", value))
	return
}
