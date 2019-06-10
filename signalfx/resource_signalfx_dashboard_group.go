package signalfx

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

const DASHBOARD_GROUP_API_PATH = "/v2/dashboardgroup"

func dashboardGroupResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"synced": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the resource in the provider and SignalFx are identical or not. Used internally for syncing.",
			},
			"last_updated": &schema.Schema{
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Latest timestamp the resource was updated",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the dashboard group",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the dashboard group",
			},
			"teams": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Team IDs to associate the dashboard group to",
			},
		},

		Create: dashboardgroupCreate,
		Read:   dashboardgroupRead,
		Update: dashboardgroupUpdate,
		Delete: dashboardgroupDelete,
	}
}

/*
  Use Resource object to construct json payload in order to create a dasboard group
*/
func getPayloadDashboardGroup(d *schema.ResourceData) ([]byte, error) {
	payload := map[string]interface{}{
		"name":        d.Get("name").(string),
		"description": d.Get("description").(string),
		// We are not keeping track of this because it's already done in the dashboard resource.
		"dashboards": make([]string, 0),
	}

	if val, ok := d.GetOk("teams"); ok {
		payload["teams"] = val.([]interface{})
	}

	return json.Marshal(payload)
}

func dashboardgroupCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadDashboardGroup(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	url, err := buildURL(config.APIURL, DASHBOARD_GROUP_API_PATH, map[string]string{"empty": "true"})
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceCreate(url, config.AuthToken, payload, d)
}

func dashboardgroupRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	path := fmt.Sprintf("%s/%s", DASHBOARD_GROUP_API_PATH, d.Id())
	url, err := buildURL(config.APIURL, path, map[string]string{})
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceRead(url, config.AuthToken, d)
}

func dashboardgroupUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadDashboardGroup(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}
	path := fmt.Sprintf("%s/%s", DASHBOARD_GROUP_API_PATH, d.Id())
	url, err := buildURL(config.APIURL, path, map[string]string{})
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceUpdate(url, config.AuthToken, payload, d)
}

func dashboardgroupDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	path := fmt.Sprintf("%s/%s", DASHBOARD_GROUP_API_PATH, d.Id())
	url, err := buildURL(config.APIURL, path, map[string]string{})
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceDelete(url, config.AuthToken, d)
}
