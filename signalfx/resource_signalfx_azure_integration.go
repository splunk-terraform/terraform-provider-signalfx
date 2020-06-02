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

func integrationAzureResource() *schema.Resource {
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
			"environment": &schema.Schema{
				Type:         schema.TypeString,
				Default:      "azure",
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: validation.StringInSlice([]string{string(integration.AZURE_DEFAULT), string(integration.AZURE_US_GOVERNMENT)}, true),
				Description:  "what type of Azure integration this is. The allowed values are `\"azure_us_government\"` and `\"azure\"`. Defaults to `\"azure\"`",
			},
			"app_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Azure application ID for the SignalFx app.",
			},
			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Azure secret key that associates the SignalFx app in Azure with the Azure tenant.",
			},
			"poll_rate": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "Azure poll rate (in seconds). One of `60` or `300`.",
				ValidateFunc: validation.IntInSlice([]int{60, 300}),
			},
			"services": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateAzureService,
				},
				Description: "List of Microsoft Azure service names for the Azure services you want SignalFx to monitor. SignalFx only supports certain services, and if you specify an unsupported one, you receive an API error.",
			},
			"subscriptions": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "List of Azure subscriptions that SignalFx should monitor.",
			},
			"tenant_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Azure ID of the Azure tenant.",
			},
			"named_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A named token to use for ingest",
				ForceNew:    true,
			},
		},

		Create: integrationAzureCreate,
		Read:   integrationAzureRead,
		Update: integrationAzureUpdate,
		Delete: integrationAzureDelete,
		Exists: integrationAzureExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func integrationAzureExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetAzureIntegration(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func integrationAzureRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	int, err := config.Client.GetAzureIntegration(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	return azureIntegrationAPIToTF(d, int)
}

func azureIntegrationAPIToTF(d *schema.ResourceData, azure *integration.AzureIntegration) error {
	debugOutput, _ := json.Marshal(azure)
	log.Printf("[DEBUG] SignalFx: Got Azure Integration to enState: %s", string(debugOutput))

	if err := d.Set("name", azure.Name); err != nil {
		return err
	}
	if err := d.Set("enabled", azure.Enabled); err != nil {
		return err
	}
	if err := d.Set("environment", strings.ToLower(string(azure.AzureEnvironment))); err != nil {
		return err
	}
	if err := d.Set("poll_rate", *azure.PollRate/1000); err != nil {
		return err
	}
	if err := d.Set("tenant_id", azure.TenantId); err != nil {
		return err
	}
	if err := d.Set("named_token", azure.NamedToken); err != nil {
		return err
	}
	if len(azure.Services) > 0 {
		services := make([]interface{}, len(azure.Services))
		for i, v := range azure.Services {
			services[i] = string(v)
		}
		if err := d.Set("services", schema.NewSet(schema.HashString, services)); err != nil {
			return err
		}
	}
	if len(azure.Subscriptions) > 0 {
		subs := make([]interface{}, len(azure.Subscriptions))
		for i, v := range azure.Subscriptions {
			subs[i] = v
		}
		if err := d.Set("subscriptions", schema.NewSet(schema.HashString, subs)); err != nil {
			return err
		}
	}

	return nil
}

func getPayloadAzureIntegration(d *schema.ResourceData) (*integration.AzureIntegration, error) {

	azure := &integration.AzureIntegration{
		Name:             d.Get("name").(string),
		Type:             "Azure",
		Enabled:          d.Get("enabled").(bool),
		AppId:            d.Get("app_id").(string),
		AzureEnvironment: integration.AzureEnvironment(strings.ToUpper(d.Get("environment").(string))),
		SecretKey:        d.Get("secret_key").(string),
		TenantId:         d.Get("tenant_id").(string),
	}

	if val, ok := d.GetOk("named_token"); ok {
		azure.NamedToken = val.(string)
	}

	if val, ok := d.GetOk("poll_rate"); ok {
		val := val.(int)
		if val != 0 {
			pollRate := integration.OneMinutely
			if val == 300 {
				pollRate = integration.FiveMinutely
			}
			azure.PollRate = &pollRate
		}
	}

	if val, ok := d.GetOk("services"); ok {
		tfServices := val.(*schema.Set).List()
		services := make([]integration.AzureService, len(tfServices))
		for i, s := range tfServices {
			s := s.(string)
			services[i] = integration.AzureServiceNames[s]
		}
		azure.Services = services
	}

	if val, ok := d.GetOk("subscriptions"); ok {
		tfSubs := val.(*schema.Set).List()
		subs := make([]string, len(tfSubs))
		for i, s := range tfSubs {
			s := s.(string)
			subs[i] = s
		}
		azure.Subscriptions = subs
	}

	return azure, nil
}

func integrationAzureCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadAzureIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Azure Integration Payload: %s", string(debugOutput))

	int, err := config.Client.CreateAzureIntegration(payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return azureIntegrationAPIToTF(d, int)
}

func integrationAzureUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadAzureIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Azure Integration Payload: %s", string(debugOutput))

	int, err := config.Client.UpdateAzureIntegration(d.Id(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return azureIntegrationAPIToTF(d, int)
}

func integrationAzureDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteAzureIntegration(d.Id())
}

func validateAzureService(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	for key, _ := range integration.AzureServiceNames {
		if key == value {
			return
		}
	}
	errors = append(errors, fmt.Errorf("%s not allowed; consult the documentation for a list of valid Azure service names", value))
	return
}
