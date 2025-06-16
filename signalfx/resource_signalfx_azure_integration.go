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
				Description: "Azure application ID for the Splunk Observability Cloud app.",
			},
			"custom_namespaces_per_service": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"service": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the service",
						},
						"namespaces": {
							Type:     schema.TypeSet,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "The namespaces to sync",
						},
					},
				},
				Description: "Allows for more fine-grained control of syncing of custom namespaces, should the boolean convenience parameter `sync_guest_os_namespaces` be not enough. The customer may specify a map of services to custom namespaces. If they do so, for each service which is a key in this map, we will attempt to sync metrics from namespaces in the value list in addition to the default namespaces.",
			},
			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Azure secret key that associates the Splunk Observability Cloud app in Azure with the Azure tenant.",
			},
			"poll_rate": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      300,
				Description:  "Azure poll rate (in seconds). Between `60` and `600`.",
				ValidateFunc: validation.IntBetween(60, 600),
			},
			"services": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "List of Microsoft Azure service names for the Azure services you want Splunk Observability Cloud to monitor. Splunk Observability Cloud only supports certain services, and if you specify an unsupported one, you receive an API error.",
			},
			"additional_services": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Additional Azure resource types that you want to sync with Observability Cloud.",
			},
			"resource_filter_rules": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of rules for filtering Azure resources by their tags. The source of each filter rule must be in the form filter('key', 'value'). You can join multiple filter statements using the and and or operators. Referenced keys are limited to tags and must start with the azure_tag_ prefix..",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"filter_source": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"subscriptions": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "List of Azure subscriptions that Splunk Observability Cloud should monitor.",
			},
			"sync_guest_os_namespaces": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If enabled, Splunk Observability Cloud will try to sync additional namespaces for VMs (including VMs in scale sets): telegraf/mem, telegraf/cpu, azure.vm.windows.guest (these are namespaces recommended by Azure when enabling their Diagnostic Extension). If there are no metrics there, no new datapoints will be ingested.",
			},
			"import_azure_monitor": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "If enabled, Splunk Observability Cloud will sync also Azure Monitor data. If disabled, Splunk Observability Cloud will import only metadata. Defaults to true.",
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
			"use_batch_api": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If enabled, Splunk Observability Cloud will collect datapoints using Azure Metrics Batch API. Consider this option if you are synchronizing high loads of data and you want to avoid throttling issues. Contrary to the default Metrics List API, Metrics Batch API is paid. Refer to Azure documentation for pricing info.",
			},
		},

		Create: integrationAzureCreate,
		Read:   integrationAzureRead,
		Update: integrationAzureUpdate,
		Delete: integrationAzureDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func integrationAzureRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	int, err := config.Client.GetAzureIntegration(context.TODO(), d.Id())
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
	if err := d.Set("poll_rate", azure.PollRateMs/1000); err != nil {
		return err
	}
	if err := d.Set("tenant_id", azure.TenantId); err != nil {
		return err
	}
	if err := d.Set("named_token", azure.NamedToken); err != nil {
		return err
	}
	if err := d.Set("sync_guest_os_namespaces", azure.SyncGuestOsNamespaces); err != nil {
		return err
	}
	if err := d.Set("import_azure_monitor", *azure.ImportAzureMonitor); err != nil {
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
	if len(azure.AdditionalServices) > 0 {
		additionalServices := make([]string, len(azure.AdditionalServices))
		for i, v := range azure.AdditionalServices {
			additionalServices[i] = v
		}
		if err := d.Set("additional_services", additionalServices); err != nil {
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
	if len(azure.CustomNamespacesPerService) > 0 {
		var customs []map[string]interface{}
		for k, v := range azure.CustomNamespacesPerService {
			namespaces := make([]interface{}, len(v))
			for i, ns := range v {
				namespaces[i] = ns
			}
			customs = append(customs, map[string]interface{}{
				"service":    k,
				"namespaces": schema.NewSet(schema.HashString, namespaces),
			})
		}
		if err := d.Set("custom_namespaces_per_service", customs); err != nil {
			return err
		}
	}
	if len(azure.ResourceFilterRules) > 0 {
		var rules []map[string]interface{}
		for _, v := range azure.ResourceFilterRules {
			filter_source := v.Filter.Source
			rules = append(rules, map[string]interface{}{
				"filter_source": filter_source,
			})
		}
		if err := d.Set("resource_filter_rules", rules); err != nil {
			return err
		}
	}
	if err := d.Set("use_batch_api", *azure.UseBatchApi); err != nil {
		return err
	}

	return nil
}

func getPayloadAzureIntegration(d *schema.ResourceData) (*integration.AzureIntegration, error) {
	importAzureMonitor := d.Get("import_azure_monitor").(bool)
	useBatchApi := d.Get("use_batch_api").(bool)
	azure := &integration.AzureIntegration{
		Name:                  d.Get("name").(string),
		Type:                  "Azure",
		Enabled:               d.Get("enabled").(bool),
		AppId:                 d.Get("app_id").(string),
		AzureEnvironment:      integration.AzureEnvironment(strings.ToUpper(d.Get("environment").(string))),
		SecretKey:             d.Get("secret_key").(string),
		TenantId:              d.Get("tenant_id").(string),
		SyncGuestOsNamespaces: d.Get("sync_guest_os_namespaces").(bool),
		ImportAzureMonitor:    &importAzureMonitor,
		UseBatchApi:           &useBatchApi,
	}

	if val, ok := d.GetOk("named_token"); ok {
		azure.NamedToken = val.(string)
	}

	if val, ok := d.GetOk("poll_rate"); ok {
		azure.PollRateMs = int64(val.(int)) * 1000
	}

	if val, ok := d.GetOk("services"); ok {
		tfServices := val.(*schema.Set).List()
		services := make([]integration.AzureService, len(tfServices))
		for i, v := range tfServices {
			v := integration.AzureService(v.(string))
			services[i] = v
		}
		azure.Services = services
	}

	if val, ok := d.GetOk("additional_services"); ok {
		tfAdditionalServices := val.([]interface{})
		additionalServices := make([]string, len(tfAdditionalServices))
		for i, s := range tfAdditionalServices {
			s := s.(string)
			additionalServices[i] = s
		}
		azure.AdditionalServices = additionalServices
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

	if val, ok := d.GetOk("custom_namespaces_per_service"); ok {
		customServiceNS := map[string][]string{}
		for _, csnsTF := range val.(*schema.Set).List() {
			csnsTF := csnsTF.(map[string]interface{})
			service := csnsTF["service"].(string)
			namespaces := csnsTF["namespaces"].(*schema.Set).List()
			customServiceNS[service] = make([]string, len(namespaces))
			for i, ns := range namespaces {
				customServiceNS[service][i] = ns.(string)
			}
		}
		azure.CustomNamespacesPerService = customServiceNS
	}

	if val, ok := d.GetOk("resource_filter_rules"); ok {
		resourceFilterRulesTF := val.([]interface{})
		resourceFilterRules := make([]integration.AzureFilterRule, len(resourceFilterRulesTF))
		for i, rfrTF := range resourceFilterRulesTF {
			ruleTF := rfrTF.(map[string]interface{})
			resourceFilterRules[i] = integration.AzureFilterRule{
				Filter: integration.AzureFilterExpression{
					Source: ruleTF["filter_source"].(string)}}
		}
		azure.ResourceFilterRules = resourceFilterRules
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

	int, err := config.Client.CreateAzureIntegration(context.TODO(), payload)
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

	int, err := config.Client.UpdateAzureIntegration(context.TODO(), d.Id(), payload)
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

	return config.Client.DeleteAzureIntegration(context.TODO(), d.Id())
}
